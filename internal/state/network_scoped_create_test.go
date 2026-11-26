package state

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

func TestPR178ScopedCreateAtomicACLExchange(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, token, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	project, bridge, name, uuid := "user-1001", "incusbr-1001", "alpha", "c34cab69-9fd4-4c28-b894-e06e4ea08b6a"
	intent := ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Start: true, Internet: true, BootAutostart: true, Options: json.RawMessage("{}")}
	raw, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	a := Admission{Layer: project, ActorID: token.ID, TokenID: token.ID, Key: "n2-scoped-create-atomic", RequestID: "create", Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw}
	rec, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	pin := VerifiedCreateBinding{bridge, peerscope.QuarantineACL, "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if ok, err := s.StageCreateBinding(ctx, rec.ID, project, name, pin); err != nil || !ok {
		t.Fatal(ok, err)
	}
	if err = s.NativeAck(ctx, rec.ID, "native-original"); err != nil {
		t.Fatal(err)
	}
	scoped := pin
	scoped.ACL, err = peerscope.ACLName(project, bridge, uuid)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeVerifiedCreateWithBinding(ctx, rec.ID, project, name, uuid, pin); err == nil {
		t.Fatal("legacy finalization accepted quarantine")
	}
	if err = s.RegisterCreatedBoxWithBinding(ctx, rec.ID, project, name, uuid, pin); err == nil {
		t.Fatal("legacy registration accepted quarantine")
	}
	for _, bad := range []VerifiedCreateBinding{pin, {bridge, "bw-peer-" + strings.Repeat("a", 48), pin.MAC, pin.IPv4, pin.IPv6}, {bridge, scoped.ACL, pin.MAC, "10.243.18.12", pin.IPv6}} {
		if err = s.FinalizeVerifiedScopedCreate(ctx, rec.ID, project, name, uuid, bad); err == nil {
			t.Fatal("invalid final binding accepted", bad)
		}
	}
	if _, err = s.db.Exec("CREATE TRIGGER scoped_abort BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'terminal fault'); END"); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeVerifiedScopedCreate(ctx, rec.ID, project, name, uuid, scoped); err == nil {
		t.Fatal("fault did not abort")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	proof, err := readScopedCreateOriginalTx(ctx, tx, rec.ID, project, bridge)
	tx.Rollback()
	if err != nil || proof.Quarantine != pin {
		t.Fatal("provisional claim changed after rollback", proof, err)
	}
	if _, err = s.db.Exec("DROP TRIGGER scoped_abort"); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeVerifiedScopedCreate(ctx, rec.ID, project, name, uuid, scoped); err != nil {
		t.Fatal(err)
	}
	ledger, err := s.ReadCompleteScopedNetworkAuditLedger(ctx, project, bridge)
	if err != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].ACL != scoped.ACL {
		t.Fatal(ledger, err)
	}
	got, err := s.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.BoxUUID != uuid {
		t.Fatal(got, err)
	}
}

// The transitional schema has legacy archive tables, but a Scoped controller
// cannot treat archived addresses as reusable or trust any mixed history.
func TestP1ScopedAddressHoldsPreserveUnknownClaims(t *testing.T) {
	ctx := context.Background()
	s, original, in, pin := p1CreateFixture(t, false)
	if e := s.Unknown(ctx, original.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); e != nil {
		t.Fatal(e)
	}
	h, e := s.ReadScopedCreateAddressHolds(ctx, in.Project, in.Bridge)
	if e != nil || len(h.IPs) != 2 || len(h.MACs) != 1 ||
		!h.IPs[netip.MustParseAddr(pin.IPv4)] || !h.IPs[netip.MustParseAddr(pin.IPv6)] || !h.MACs[pin.MAC] {
		t.Fatal("original prospective holds missing", h, e)
	}
	legacy, e := s.ReadCreateAddressHolds(ctx, in.Project, in.Bridge)
	if e != nil || len(legacy.IPs) != 2 || len(legacy.MACs) != 1 ||
		!legacy.IPs[netip.MustParseAddr(pin.IPv4)] || !legacy.IPs[netip.MustParseAddr(pin.IPv6)] || !legacy.MACs[pin.MAC] {
		t.Fatal("legacy unarchived holds changed", legacy, e)
	}
	other, e := s.ReadScopedCreateAddressHolds(ctx, in.Project, "incusbr-other")
	if e != nil || len(other.IPs) != 0 || len(other.MACs) != 0 {
		t.Fatal("foreign bridge holds leaked", other, e)
	}
	p1AssertOriginal(t, s, original.ID, "UNKNOWN", "")
	var n int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", original.ID).Scan(&n); e != nil || n != 4 {
		t.Fatal("original claims changed", n, e)
	}
}

func TestP1ScopedAddressHoldsRefuseLegacyRetirement(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, b, pin, retired := retireFixture(t, s)
	// Before archive, both readers include the retained exact IPv4/IPv6/MAC.
	h, err := s.ReadScopedCreateAddressHolds(ctx, b.Project, pin.Bridge)
	if err != nil || !h.IPs[netip.MustParseAddr(pin.IPv4)] || !h.IPs[netip.MustParseAddr(pin.IPv6)] || !h.MACs[pin.MAC] {
		t.Fatal(h, err)
	}
	if err = retireWithProof(s, ctx, retired, strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	legacy, err := s.ReadCreateAddressHolds(ctx, b.Project, pin.Bridge)
	if err != nil || legacy.IPs[netip.MustParseAddr(pin.IPv4)] {
		t.Fatal("legacy archive witness changed", legacy, err)
	}
	for _, scope := range []struct{ project, bridge string }{{b.Project, pin.Bridge}, {"user-1001", "incusbr-1001"}} {
		if _, err = s.ReadScopedCreateAddressHolds(ctx, scope.project, scope.bridge); !errors.Is(err, ErrResourceLocked) {
			t.Fatal("mixed legacy history accepted", scope, err)
		}
	}
}

func TestP1ScopedReservationRefusesLegacyArchiveQueueAndCompaction(t *testing.T) {
	ctx := context.Background()
	for _, table := range []string{"network_ipv4_retirements", "network_ipv4_retirement_queue", "network_ipv4_compactions"} {
		t.Run(table, func(t *testing.T) {
			s := newStore(t)
			_, b, _, retired := retireFixture(t, s)
			var err error
			switch table {
			case "network_ipv4_retirements":
				err = retireWithProof(s, ctx, retired, strings.Repeat("a", 64))
			case "network_ipv4_retirement_queue":
				_, err = s.db.Exec("INSERT OR IGNORE INTO network_ipv4_retirement_queue(delete_operation_id,project,attempt_order) VALUES(?,?,1)", retired.DeleteOperationID, b.Project)
			case "network_ipv4_compactions":
				_, err = s.db.Exec("INSERT INTO network_ipv4_compactions(project,delete_operation_id,plan_sha256,prepared_at) VALUES(?,?,?,1)", b.Project, retired.DeleteOperationID, strings.Repeat("b", 64))
			}
			if err != nil {
				t.Fatal(err)
			}
			var n int
			if err = s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); err != nil || n != 1 {
				t.Fatal(n, err)
			}
			project, bridge, name := "user-1001", "incusbr-1001", "scoped-next"
			if _, err = s.ReadScopedCreateAddressHolds(ctx, project, bridge); !errors.Is(err, ErrResourceLocked) {
				t.Fatal("mixed legacy state accepted", err)
			}
			if _, err = s.ReadCompleteScopedNetworkAuditLedger(ctx, project, bridge); !errors.Is(err, ErrResourceLocked) {
				t.Fatal("mixed history audit accepted", err)
			}
			_, actor, err := s.IssueToken(ctx, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Options: json.RawMessage("{}")})
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.Admit(ctx, Admission{Layer: project, ActorID: actor.ID, TokenID: actor.ID, Key: "p1-scoped-mixed-legacy", RequestID: "create", Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw})
			if err != nil {
				t.Fatal(err)
			}
			if err = s.Claim(ctx, r.ID); err != nil {
				t.Fatal(err)
			}
			pin := VerifiedCreateBinding{bridge, peerscope.QuarantineACL, "02:16:3e:11:22:44", "10.243.18.44", "fd43:18::44"}
			if ok, e := s.StageCreateBinding(ctx, r.ID, project, name, pin); ok || !errors.Is(e, ErrResourceLocked) {
				t.Fatal("mixed history staged", ok, e)
			}
			if err = s.db.QueryRow("SELECT count(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", r.ID).Scan(&n); err != nil || n != 0 {
				t.Fatal("partial reservation", n, err)
			}
		})
	}
}

// The completed original still owns the exact pin. Missing ledger records
// can be repaired, but neither a different ACL nor mixed retirement history
// can authorize a repair. Fixtures use real SQLite, not native evidence.
func TestP1ScopedPinRepairRetainsOriginal(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	pin.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	if e := s.FinalizeVerifiedScopedCreate(ctx, orig.ID, in.Project, in.Name, p1UUID, pin); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	if got, e := s.ReadOriginalCreatePin(ctx, target); e != nil || got != pin {
		t.Fatal(got, e)
	}
	if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", p1UUID); e != nil {
		t.Fatal(e)
	}
	p := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: p1UUID, Fingerprint: "synthetic-exact-pin", CreateBinding: pin}
	bad := p
	bad.CreateBinding.ACL = "bw-peer"
	if e := s.CheckCreatePinRepair(ctx, target, bad); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("wrong ACL", e)
	}
	if e := s.CheckCreatePinRepair(ctx, target, p); e != nil {
		t.Fatal(e)
	}
	rec, e := s.CommitRecovery(ctx, req, target, p)
	if e != nil || rec.Disposition != "SETTLED" || rec.Receipt.ID == orig.ID {
		t.Fatal(rec, e)
	}
	p1AssertOriginal(t, s, orig.ID, "SUCCEEDED", target.NativeID)
	if got, e := s.ReadOriginalCreatePin(ctx, target); e != nil || got != pin {
		t.Fatal(got, e)
	}
	if _, e = s.ReadCompleteScopedNetworkAuditLedger(ctx, in.Project, in.Bridge); e != nil {
		t.Fatal(e)
	}
}

func TestP1ScopedPinRepairRefusesMixedHistory(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	pin.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	if e := s.FinalizeVerifiedScopedCreate(ctx, orig.ID, in.Project, in.Name, p1UUID, pin); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	_, _, _, retired := retireFixture(t, s)
	if e := retireWithProof(s, ctx, retired, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ReadOriginalCreatePin(ctx, target); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("mixed history read", e)
	}
	if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", p1UUID); e != nil {
		t.Fatal(e)
	}
	p := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: p1UUID, Fingerprint: "synthetic-exact-pin", CreateBinding: pin}
	if e := s.CheckCreatePinRepair(ctx, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("mixed history preflight", e)
	}
	if _, e := s.CommitRecovery(ctx, req, target, p); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("mixed history repair", e)
	}
	var n int
	if e := s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE box_uuid=?", p1UUID).Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if e := s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=?", orig.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	p1AssertOriginal(t, s, orig.ID, "SUCCEEDED", target.NativeID)
}

func TestP1ScopedPinRepairRefusesDuplicateTerminalClaim(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	pin.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	if e := s.FinalizeVerifiedScopedCreate(ctx, orig.ID, in.Project, in.Name, p1UUID, pin); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	if _, e := s.db.Exec("DELETE FROM network_address_fences WHERE box_uuid=? AND family='ipv4'", p1UUID); e != nil {
		t.Fatal(e)
	}
	_, actor, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	other, e := s.AdmitScoped(ctx, Admission{Layer: in.Project, ActorID: actor.ID, TokenID: actor.ID, Key: "p1-scoped-pin-repair-duplicate", RequestID: "fixture", Action: "box_create", ResourceKey: "box:other", SafeIntent: []byte("{}")})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CancelQueued(ctx, in.Project, other.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,?)", other.ID, in.Project, pin.reservationKeys()[0]); e != nil {
		t.Fatal(e)
	}
	proof := RecoveryProof{Kind: RecoveryRepairPin, Reason: "CREATE_PIN_LEDGER_REPAIRED", NativeID: target.NativeID, UUID: p1UUID, Fingerprint: "synthetic-exact-pin", CreateBinding: pin}
	if e = s.CheckCreatePinRepair(ctx, target, proof); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("duplicate retained claim preflight accepted", e)
	}
	if _, e = s.CommitRecovery(ctx, req, target, proof); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("duplicate retained claim repair accepted", e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE box_uuid=?", p1UUID).Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=?", orig.ID).Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
	p1AssertOriginal(t, s, orig.ID, "SUCCEEDED", target.NativeID)
}
