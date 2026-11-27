package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
)

const p1Project = "user-1001"
const p1Bridge = "incusbr-1001"
const p1UUID = "c34cab69-9fd4-4c28-b894-e06e4ea08b6a"

// Real SQLite transitions with synthetic Root/native evidence: these tests
// prove state predicates and atomicity, not actual Incus effects.
func p1CreateFixture(t *testing.T, ack bool) (*Store, Receipt, ScopedCreateIntent, VerifiedCreateBinding) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	in := ScopedCreateIntent{Schema: 2, Project: p1Project, Bridge: p1Bridge, Name: "alpha", CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Start: true, Internet: true, BootAutostart: true, Options: json.RawMessage("{}")}
	raw, e := json.Marshal(in)
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.AdmitScoped(ctx, Admission{Layer: in.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "p1b-create-recovery-fixture", RequestID: "fixture", Action: "box_create", ResourceKey: "box:" + in.Name, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{p1Bridge, peerscope.QuarantineACL, "02:16:3e:11:22:33", "10.243.18.11", "fd43:18::11"}
	if ok, e := s.StageCreateBinding(ctx, r.ID, in.Project, in.Name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if ack {
		if e = s.NativeAck(ctx, r.ID, "fixture-native-create"); e != nil {
			t.Fatal(e)
		}
	}
	return s, r, in, pin
}

func p1CompletedProof(t *testing.T, target RecoveryTarget, kind string, b VerifiedCreateBinding, sha string) RecoveryProof {
	t.Helper()
	r := ScopedLifecycleRecord{Schema: 2, Kind: kind, Status: "VERIFIED", OriginalID: target.ID, NativeID: target.NativeID, Project: target.Layer, Name: target.Name, UUID: p1UUID, Binding: b, HostSHA256: strings.Repeat("a", 64), GuardSHA256: strings.Repeat("b", 64), PlanSHA256: strings.Repeat("c", 64)}
	if kind == "CREATE" {
		r.Start, r.Internet, r.BootAutostart = true, true, true
	}
	p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY", NativeID: target.NativeID, UUID: p1UUID, Fingerprint: "synthetic-complete-poststate", ScopedLifecycle: ScopedLifecycleRecoveryProof{Record: r, OriginalSHA256: sha, RecoveryOnly: true}}
	if kind == "CREATE" {
		p.CreateBinding = b
	}
	return p
}

func p1AssertOriginal(t *testing.T, s *Store, id, state, nativeID string) {
	t.Helper()
	r, e := s.Get(context.Background(), id)
	if e != nil || r.State != state || r.NativeOperationID != nativeID {
		t.Fatal(r, e)
	}
}

func TestP1ScopedCreateCompletedRecoveryAtomic(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	if e := s.Unknown(ctx, orig.ID, "VERIFYING", "FINAL_COMMIT_LOST"); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	if !target.ScopedLifecycle {
		t.Fatal(target)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = readScopedCreateOriginalTx(ctx, tx, orig.ID, in.Project, in.Bridge); e == nil {
		t.Fatal("UNKNOWN gained dispatch predicate")
	}
	proof, e := readScopedCreateOriginalTxPhase(ctx, tx, orig.ID, in.Project, in.Bridge, "UNKNOWN")
	tx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	final := pin
	final.ACL, e = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	if e != nil {
		t.Fatal(e)
	}
	good := p1CompletedProof(t, target, "CREATE", final, ScopedLifecycleOriginalSHA256(proof))
	if e = s.CheckScopedLifecycleRecovery(ctx, target, good); e != nil {
		t.Fatal(e)
	}
	for name, change := range map[string]func(*RecoveryProof){
		"effect-reply":    func(p *RecoveryProof) { p.ScopedLifecycle.RecoveryOnly = false },
		"partial-journal": func(p *RecoveryProof) { p.ScopedLifecycle.Record.Status = "ACL_VERIFIED" },
		"wrong-original":  func(p *RecoveryProof) { p.ScopedLifecycle.Record.OriginalID = strings.Repeat("f", 32) },
		"changed-digest":  func(p *RecoveryProof) { p.ScopedLifecycle.OriginalSHA256 = strings.Repeat("e", 64) },
		"wrong-native":    func(p *RecoveryProof) { p.ScopedLifecycle.Record.NativeID = "other" },
		"changed-boot":    func(p *RecoveryProof) { p.ScopedLifecycle.Record.BootAutostart = false },
		"changed-address": func(p *RecoveryProof) {
			p.CreateBinding.IPv4 = "10.243.18.12"
			p.ScopedLifecycle.Record.Binding = p.CreateBinding
		},
		"legacy-bind": func(p *RecoveryProof) { p.Kind = RecoveryBind; p.Reason = "CREATE_MARKER_EXACT_IDENTITY" },
	} {
		t.Run(name, func(t *testing.T) {
			bad := good
			change(&bad)
			if _, e := s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
				t.Fatal("forged settlement accepted")
			}
			p1AssertOriginal(t, s, orig.ID, "UNKNOWN", target.NativeID)
			if _, e := s.OwnedBox(ctx, in.Project, p1UUID); !errors.Is(e, ErrNotFound) {
				t.Fatal(e)
			}
		})
	}
	if _, e = s.db.Exec("CREATE TRIGGER p1_abort BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'test final fault'); END"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitScopedRecovery(ctx, req, target, good); e == nil {
		t.Fatal("fault did not roll back")
	}
	p1AssertOriginal(t, s, orig.ID, "UNKNOWN", target.NativeID)
	if _, e = s.OwnedBox(ctx, in.Project, p1UUID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	tx, e = s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	after, e := readScopedCreateOriginalTxPhase(ctx, tx, orig.ID, in.Project, in.Bridge, "UNKNOWN")
	tx.Rollback()
	if e != nil || after.Quarantine != pin {
		t.Fatal(after, e)
	}
	if _, e = s.db.Exec("DROP TRIGGER p1_abort"); e != nil {
		t.Fatal(e)
	}
	a4SettleMode(t, s, req, target, good, true)
	p1AssertOriginal(t, s, orig.ID, "SUCCEEDED", target.NativeID)
	ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, in.Project, in.Bridge)
	if e != nil || len(ledger.Boxes) != 1 || ledger.Boxes[0].ACL != final.ACL {
		t.Fatal(ledger, e)
	}
}

func TestP1ScopedCreateMissingACKStaysUnknown(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, false)
	if e := s.Unknown(ctx, orig.ID, "DISPATCHING", "ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, orig.ID)
	final := pin
	final.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	bad := p1CompletedProof(t, target, "CREATE", final, strings.Repeat("a", 64))
	if _, e := s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
		t.Fatal("missing ACK settled")
	}
	bad.Kind = RecoveryBind
	bad.Reason = "CREATE_MARKER_EXACT_IDENTITY"
	if _, e := s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
		t.Fatal("legacy bind accepted")
	}
	p1AssertOriginal(t, s, orig.ID, "UNKNOWN", "")
}

func TestP1ScopedDeleteCompletedRecoveryAtomic(t *testing.T) {
	ctx := context.Background()
	s, orig, in, pin := p1CreateFixture(t, true)
	final := pin
	final.ACL, _ = peerscope.ACLName(in.Project, in.Bridge, p1UUID)
	// Recover Create first, so Delete and the final retirement audit exercise
	// the complete recovered lineage, not two isolated settlement paths.
	if e := s.Unknown(ctx, orig.ID, "VERIFYING", "FINAL_COMMIT_LOST"); e != nil {
		t.Fatal(e)
	}
	createReq, createTarget := a4Request(t, s, orig.ID)
	createTx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	createProof, e := readScopedCreateOriginalTxPhase(ctx, createTx, orig.ID, in.Project, in.Bridge, "UNKNOWN")
	createTx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	a4SettleMode(t, s, createReq, createTarget, p1CompletedProof(t, createTarget, "CREATE", final, ScopedLifecycleOriginalSHA256(createProof)), true)
	a := b5DeleteAdmission(t, s, p1UUID, in.Name)
	a.Layer = in.Project
	a.SafeIntent, _ = json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{p1UUID})
	parent := a4AdmitMode(t, s, a, true)
	if e := s.Claim(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.AuthorizeCascadeFinal(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.NativeAck(ctx, parent.ID, "fixture-native-delete"); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, parent.ID, "VERIFYING", "FINAL_COMMIT_LOST"); e != nil {
		t.Fatal(e)
	}
	req, target := a4Request(t, s, parent.ID)
	if !target.ScopedLifecycle {
		t.Fatal(target)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = readScopedDeleteOriginalTx(ctx, tx, parent.ID, in.Project, in.Bridge); e == nil {
		t.Fatal("UNKNOWN gained dispatch predicate")
	}
	proof, e := readScopedDeleteOriginalTxPhase(ctx, tx, parent.ID, in.Project, in.Bridge, "UNKNOWN")
	tx.Rollback()
	if e != nil {
		t.Fatal(e)
	}
	good := p1CompletedProof(t, target, "DELETE", final, ScopedLifecycleOriginalSHA256(proof))
	if e = s.CheckScopedLifecycleRecovery(ctx, target, good); e != nil {
		t.Fatal(e)
	}
	bad := recoverProof(target)
	bad.Fingerprint = "DELETE_SUCCESS:legacy"
	if _, e = s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
		t.Fatal("instance absence without ACL proof settled")
	}
	bad = good
	bad.ScopedLifecycle.Record.Binding.ACL = peerscope.QuarantineACL
	if _, e = s.CommitScopedRecovery(ctx, req, target, bad); e == nil {
		t.Fatal("wrong ACL settled")
	}
	if _, e = s.db.Exec("CREATE TRIGGER p1_abort BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' BEGIN SELECT RAISE(ABORT,'test final fault'); END"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitScopedRecovery(ctx, req, target, good); e == nil {
		t.Fatal("fault did not roll back")
	}
	p1AssertOriginal(t, s, parent.ID, "UNKNOWN", target.NativeID)
	if _, e = s.OwnedBox(ctx, in.Project, p1UUID); e != nil {
		t.Fatal(e)
	}
	var n int
	if e = s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE box_uuid=? AND state='RESERVED'", p1UUID).Scan(&n); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if _, e = s.db.Exec("DROP TRIGGER p1_abort"); e != nil {
		t.Fatal(e)
	}
	a4SettleMode(t, s, req, target, good, true)
	p1AssertOriginal(t, s, parent.ID, "SUCCEEDED", target.NativeID)
	if _, e = s.OwnedBox(ctx, in.Project, p1UUID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	if e = s.db.QueryRow("SELECT count(*) FROM network_address_fences WHERE box_uuid=? AND state='RELEASED' AND operation_id=?", p1UUID, parent.ID).Scan(&n); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	// Empty active inventory is valid for the retirement-specific reader.
	check := func() error {
		tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if e != nil {
			return e
		}
		defer tx.Rollback()
		ledger, e := readCompleteScopedNetworkAuditLedgerModeTx(ctx, tx, in.Project, in.Bridge, true)
		if e != nil {
			return e
		}
		if len(ledger.Boxes) != 0 || len(ledger.Retired) != 1 || ledger.Retired[0].CreateID != orig.ID || ledger.Retired[0].DeleteID != parent.ID {
			t.Fatal(ledger)
		}
		return nil
	}
	if e = check(); e != nil {
		t.Fatal("recovered retirement history refused", e)
	}
	if _, e = s.db.Exec("DELETE FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'", parent.ID); e != nil {
		t.Fatal(e)
	}
	if e = check(); e == nil {
		t.Fatal("recovery event without linked audit accepted")
	}
}
