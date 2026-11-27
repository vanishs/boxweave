package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/peerscope"
	"net/netip"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func assertRemovedRetirementRefused(t *testing.T, s *Store, project, id string) {
	t.Helper()
	ctx := context.Background()
	if e := s.RetireVerifiedIPv4(ctx, project, id, func(context.Context, IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error) {
		t.Fatal("retired witness executed")
		return IPv4RetirementCandidate{}, "", ErrInvalid
	}); !errors.Is(e, ErrResourceLocked) {
		t.Fatal(e)
	}
	if _, e := s.ReadIPv4RetirementCandidate(ctx, project, id); !errors.Is(e, ErrResourceLocked) {
		t.Fatal(e)
	}
	if _, e := s.ReadIPv4RetirementPolicy(ctx, IPv4RetirementCandidate{Project: project, DeleteOperationID: id}); !errors.Is(e, ErrResourceLocked) {
		t.Fatal(e)
	}
}

func scopedReplacementOriginal(t *testing.T, s *Store, token, name string) Receipt {
	t.Helper()
	ctx := context.Background()
	raw, _ := json.Marshal(ScopedCreateIntent{Schema: 2, Project: purePeerProject, Bridge: purePeerBridge, Name: name, CPU: 1, MemoryMiB: 256, DiskGiB: 2, Options: json.RawMessage("{}")})
	r, e := s.AdmitScoped(ctx, Admission{Layer: purePeerProject, ActorID: token, TokenID: token, Key: "p1-replacement-" + name, RequestID: name, Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	return r
}

// Old archive/reuse is intentionally absent. Original records and all
// three held identities survive repeated refused legacy release attempts.
func TestIPv4RetirementAtomicArchiveReuse(t *testing.T) {
	ctx := context.Background()
	s, b, d := n3RetiredFixture(t)
	before, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
	if e != nil || len(before.Retired) != 1 {
		t.Fatal(before, e)
	}
	first, e := s.ReadScopedDrainWitness(ctx, b.Project)
	if e != nil {
		t.Fatal(e)
	}
	assertRemovedRetirementRefused(t, s, b.Project, d.ID)
	assertRemovedRetirementRefused(t, s, b.Project, d.ID)
	after, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal(after, e)
	}
	next, e := s.ReadScopedDrainWitness(ctx, b.Project)
	if e != nil || next.HistorySHA256 != first.HistorySHA256 {
		t.Fatal(next, e)
	}
	h, e := s.ReadScopedCreateAddressHolds(ctx, b.Project, purePeerBridge)
	p := after.Retired[0]
	if e != nil || !h.IPs[netip.MustParseAddr(p.IPv4)] || !h.IPs[netip.MustParseAddr(p.IPv6)] || !h.MACs[p.MAC] {
		t.Fatal(h, e)
	}
	var n int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", p.CreateID).Scan(&n); e != nil || n != 4 {
		t.Fatal(n, e)
	}
}

// Retained corruption checks live in TestNetworkHistoryRejectsOutstandingDependencies
// and TestPR178N3ScopedRetiredHistory.
func TestIPv4RetirementRefusesStaleOrUnsettledEvidence(t *testing.T) {
	ctx := context.Background()
	s, b, d := n3RetiredFixture(t)
	for _, id := range []string{d.ID, strings.Repeat("f", 32)} {
		assertRemovedRetirementRefused(t, s, b.Project, id)
	}
	if _, e := s.db.Exec("UPDATE operations SET state='UNKNOWN' WHERE id=?", d.ID); e != nil {
		t.Fatal(e)
	}
	assertRemovedRetirementRefused(t, s, b.Project, d.ID)
	got, e := s.Get(ctx, d.ID)
	if e != nil || got.State != "UNKNOWN" {
		t.Fatal(got, e)
	}
}

func TestIPv4RetirementConcurrentReplacementOneSlot(t *testing.T) {
	ctx := context.Background()
	for _, fresh := range []bool{false, true} {
		t.Run(fmt.Sprint(fresh), func(t *testing.T) {
			s, b, _ := n3RetiredFixture(t)
			ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
			if e != nil {
				t.Fatal(e)
			}
			token := purePeerNewToken(t, s)
			var wg sync.WaitGroup
			type result struct {
				name string
				r    Receipt
				err  error
			}
			results := make(chan result, 2)
			for i := 1; i <= 2; i++ {
				name := fmt.Sprintf("next-%d", i)
				raw, _ := json.Marshal(ScopedCreateIntent{Schema: 2, Project: purePeerProject, Bridge: purePeerBridge, Name: name, CPU: 1, MemoryMiB: 256, DiskGiB: 2, Options: json.RawMessage("{}")})
				wg.Add(1)
				go func() {
					defer wg.Done()
					r, e := s.AdmitScoped(ctx, Admission{Layer: purePeerProject, ActorID: token, TokenID: token, Key: "p1-concurrent-create-" + name, RequestID: name, Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw})
					results <- result{name, r, e}
				}()
			}
			wg.Wait()
			close(results)
			winners, locked := 0, 0
			for x := range results {
				if errors.Is(x.err, ErrResourceLocked) {
					locked++
					continue
				}
				if x.err != nil {
					t.Fatal(x.err)
				}
				winners++
				if e := s.Claim(ctx, x.r.ID); e != nil {
					t.Fatal(e)
				}
				ip := ledger.Retired[0].IPv4
				if fresh {
					ip = "10.43.18.90"
				}
				pin := VerifiedCreateBinding{purePeerBridge, peerscope.QuarantineACL, "02:16:3e:44:11:99", ip, "fd43:18::99"}
				if ok, e := s.StageCreateBinding(ctx, x.r.ID, b.Project, x.name, pin); e != nil || ok != fresh {
					t.Fatal("wrong slot allocation", ok, e)
				}
			}
			if winners != 1 || locked != 1 {
				t.Fatal("wrong project admission", winners, locked)
			}
		})
	}
}

func TestIPv4RetirementHeldWriterReservation(t *testing.T) {
	ctx := context.Background()
	s, b, d := n3RetiredFixture(t)
	path := strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive")
	assertRemovedRetirementRefused(t, s, b.Project, d.ID)
	if e := s.Close(); e != nil {
		t.Fatal(e)
	}
	again, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer again.Close()
	assertRemovedRetirementRefused(t, again, b.Project, d.ID)
	ledger, e := again.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge)
	if e != nil || len(ledger.Retired) != 1 || ledger.Retired[0].DeleteID != d.ID {
		t.Fatal(ledger, e)
	}
	// A refused obsolete writer must not strand a new valid admission.
	scopedReplacementOriginal(t, again, purePeerNewToken(t, again), "restart-next")
}

// Forward state and both original receipts survive Scoped Delete. Archive
// proof itself is retired, but exact original lineage is still required
// before final Scoped retirement.
func TestIPv4RetirementVerifiedForwardHistory(t *testing.T) {
	ctx := context.Background()
	s, b, token := n3ScopedOwnedFixture(t)
	a := Admission{Layer: b.Project, ActorID: token, TokenID: token, Key: "p1-forward-history-create", RequestID: "create", Action: "port_forward_create", BoxUUID: b.UUID, Incarnation: b.Incarnation, ResourceKey: "box:" + b.Name, ResourceKeys: []string{"box:" + b.UUID, "listener:tcp:127.0.0.1:23001"}, ForwardCreate: &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 23001, GuestPort: 8080}, SafeIntent: []byte(fmt.Sprintf(`{"box_id":%q,"protocol":"tcp","host_bind":"127.0.0.1","host_port":23001,"guest_port":8080,"confirm_external_exposure":false}`, b.UUID))}
	ck := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	f, e := s.AdmitScoped(ctx, a)
	ck(e)
	ck(s.Claim(ctx, f.ID))
	ck(s.NativeAck(ctx, f.ID, "native-forward-create"))
	ck(s.FinalizeVerifiedForward(ctx, f.ID, true))
	a.Action = "port_forward_delete"
	a.Key = "p1-forward-history-delete"
	a.ForwardCreate = nil
	a.ForwardDeleteID = f.ID
	a.SafeIntent = []byte(fmt.Sprintf(`{"forward_id":%q}`, f.ID))
	d, e := s.AdmitScoped(ctx, a)
	ck(e)
	ck(s.Claim(ctx, d.ID))
	ck(s.NativeAck(ctx, d.ID, "native-forward-delete"))
	ck(s.FinalizeVerifiedForward(ctx, d.ID, false))
	boxDel := n3DeleteOwned(t, s, b, token)
	assertRemovedRetirementRefused(t, s, b.Project, boxDel.ID)
	claim, e := s.GetForward(ctx, b.Project, f.ID)
	if e != nil || claim.State != "REVOKED" || claim.OperationID != d.ID {
		t.Fatal(claim, e)
	}
	for _, id := range []string{f.ID, d.ID, boxDel.ID} {
		r, e := s.Get(ctx, id)
		if e != nil || r.State != "SUCCEEDED" {
			t.Fatal(r, e)
		}
	}
	// A proved no-dispatch Create is a valid retained revoked row as well.
	a.Action = "port_forward_create"
	a.Key = "p1-forward-history-cancel"
	a.ForwardDeleteID = ""
	a.ForwardCreate = &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 23001, GuestPort: 8080}
	other, e := s.OwnedBox(ctx, b.Project, "11111111-1111-4111-8111-000000000002")
	ck(e)
	a.BoxUUID, a.Incarnation, a.ResourceKey = other.UUID, other.Incarnation, "box:"+other.Name
	a.ResourceKeys = []string{"box:" + other.UUID, "listener:tcp:127.0.0.1:23001"}
	a.SafeIntent = []byte(fmt.Sprintf(`{"box_id":%q,"protocol":"tcp","host_bind":"127.0.0.1","host_port":23001,"guest_port":8080,"confirm_external_exposure":false}`, other.UUID))
	cancelled, e := s.AdmitScoped(ctx, a)
	ck(e)
	ck(s.CancelQueued(ctx, b.Project, cancelled.ID))
	n3DeleteOwned(t, s, other, token)
	third, e := s.OwnedBox(ctx, b.Project, "11111111-1111-4111-8111-000000000003")
	ck(e)
	n3DeleteOwned(t, s, third, token)
	s.auditReadOnly = true
	read := func() error {
		_, e := s.ReadScopedRetirementLedger(ctx, b.Project, "incusbr-1001")
		return e
	}
	ck(read())
	before, e := s.ReadScopedDrainWitness(ctx, b.Project)
	ck(e)
	if _, e = s.db.Exec("UPDATE forward_claims SET host_port=23002 WHERE forward_id=?", f.ID); e != nil {
		t.Fatal(e)
	}
	after, e := s.ReadScopedDrainWitness(ctx, b.Project)
	ck(e)
	if after.HistorySHA256 == before.HistorySHA256 {
		t.Fatal("changed forward history hidden")
	}
	if e = read(); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("changed forward history admitted to retirement: %v", e)
	}
	_, e = s.db.Exec("UPDATE forward_claims SET host_port=23001 WHERE forward_id=?", f.ID)
	ck(e)
	ck(read())
	for _, tc := range []struct {
		name, change, restore, id string
	}{
		{"wrong-device", "UPDATE forward_claims SET native_device='other' WHERE forward_id=?", "UPDATE forward_claims SET native_device='bwpf-'||substr(forward_id,1,16) WHERE forward_id=?", f.ID},
		{"wrong-owner", "UPDATE forward_claims SET incarnation='other' WHERE forward_id=?", "UPDATE forward_claims SET incarnation=box_uuid WHERE forward_id=?", f.ID},
		{"missing-create-receipt", "UPDATE operations SET native_operation_id='' WHERE id=?", "UPDATE operations SET native_operation_id='native-forward-create' WHERE id=?", f.ID},
		{"missing-delete-receipt", "UPDATE operations SET native_operation_id='' WHERE id=?", "UPDATE operations SET native_operation_id='native-forward-delete' WHERE id=?", d.ID},
		{"cancelled-with-native-receipt", "UPDATE operations SET native_operation_id='unexpected' WHERE id=?", "UPDATE operations SET native_operation_id='' WHERE id=?", cancelled.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.db.Exec(tc.change, tc.id); err != nil {
				t.Fatal(err)
			}
			err := read()
			if _, e := s.db.Exec(tc.restore, tc.id); e != nil {
				t.Fatal(e)
			}
			if !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("invalid history admitted: %v", err)
			}
			if e := read(); e != nil {
				t.Fatal("restored valid history refused", e)
			}
		})
	}

}
