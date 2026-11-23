package state

import (
	"context"
	"fmt"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Synthetic native receipts only: exercise the real admission, staged pin,
// finalization, deletion and retirement transaction with no Incus effects.
func retirementTestLifecycle(t *testing.T, s *Store, a Admission, i int, ip string) IPv4RetirementCandidate {
	t.Helper()
	ctx := context.Background()
	ck := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(i, e)
		}
	}
	name := fmt.Sprintf("retire-churn-%d", i)
	uuid := fmt.Sprintf("retire-churn-uuid-%d", i)
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", fmt.Sprintf("02:16:3e:33:%02x:%02x", i>>8, i&255), ip, fmt.Sprintf("fd43:18::%x", i+100)}
	a.Action = "box_create"
	a.ResourceKey = "box:" + name
	a.Key = "create-original-" + name
	a.RequestID = a.Key
	a.BoxUUID = ""
	a.Incarnation = ""
	a.CleanupDependencies = false
	a.SafeIntent = []byte(fmt.Sprintf("{\"name\":%q}", name))
	r, e := s.Admit(ctx, a)
	ck(e)
	ck(s.Claim(ctx, r.ID))
	ok, e := s.StageCreateBinding(ctx, r.ID, a.Layer, name, pin)
	ck(e)
	if !ok {
		t.Fatal("slot refused", i)
	}
	ck(s.NativeAck(ctx, r.ID, "native-create-"+name))
	ck(s.FinalizeVerifiedCreateWithBinding(ctx, r.ID, a.Layer, name, uuid, pin))
	b, e := s.OwnedBox(ctx, a.Layer, uuid)
	ck(e)
	a.Action = "box_delete"
	a.Key = "delete-original-" + name
	a.RequestID = a.Key
	a.BoxUUID = uuid
	a.Incarnation = uuid
	a.CleanupDependencies = true
	a.SafeIntent = []byte(fmt.Sprintf("{\"box_id\":%q}", uuid))
	r, e = s.Admit(ctx, a)
	ck(e)
	ck(s.Claim(ctx, r.ID))
	ck(s.AuthorizeCascadeFinal(ctx, r.ID))
	ck(s.NativeAck(ctx, r.ID, "native-delete-"+name))
	ck(s.FinalizeVerifiedDelete(ctx, r.ID, b))
	c, e := s.ReadIPv4RetirementCandidate(ctx, a.Layer, r.ID)
	ck(e)
	return c
}
func retirementTestAdmission(t *testing.T, s *Store, project string) Admission {
	t.Helper()
	_, token, e := s.IssueToken(context.Background(), time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	return Admission{Layer: project, ActorID: token.ID, TokenID: token.ID}
}
func TestIPv4RetirementQueuePersistentRoundRobin(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	path := strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive")
	a := retirementTestAdmission(t, s, "user-1001")
	var ids []string
	for i := 1; i <= 5; i++ {
		c := retirementTestLifecycle(t, s, a, i, fmt.Sprintf("10.24.18.%d", i+10))
		ids = append(ids, c.DeleteOperationID)
	}
	other := retirementTestAdmission(t, s, "user-1002")
	otherC := retirementTestLifecycle(t, s, other, 6, "10.24.18.90")
	// Exactly two tickets even if no proof succeeds. The next process must
	// advance, not repeatedly inspect the same first two forever.
	got, e := s.ClaimPendingIPv4Retirements(ctx, a.Layer, 2)
	if e != nil || !reflect.DeepEqual(got, ids[:2]) {
		t.Fatal(got, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got, e = s.ClaimPendingIPv4Retirements(ctx, a.Layer, 2)
	if e != nil || !reflect.DeepEqual(got, ids[2:4]) {
		t.Fatal(got, e)
	}
	got, e = s.ClaimPendingIPv4Retirements(ctx, a.Layer, 2)
	if e != nil || !reflect.DeepEqual(got, []string{ids[4], ids[0]}) {
		t.Fatal(got, e)
	}
	got, e = s.ClaimPendingIPv4Retirements(ctx, other.Layer, 2)
	if e != nil || !reflect.DeepEqual(got, []string{otherC.DeleteOperationID}) {
		t.Fatal(got, e)
	}
	// Another Project on the same bridge is not eligible for address proof;
	// the queue itself remains scoped independently. No ticket releases a pin.
	var n int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM network_address_fences WHERE project=? AND state='RELEASED'", a.Layer).Scan(&n); e != nil || n != 10 {
		t.Fatal(n, e)
	}
	if e = s.db.QueryRow("SELECT COUNT(*) FROM network_ipv4_retirements").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
func TestIPv4RetirementChurnBeyond256Originals(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a := retirementTestAdmission(t, s, "user-1001")
	const slot = "10.24.18.11"
	begin := time.Now()
	for i := 1; i <= 257; i++ {
		c := retirementTestLifecycle(t, s, a, i, slot)
		if e := retireWithProof(s, ctx, c, strings.Repeat("a", 64)); e != nil {
			t.Fatal(i, e)
		}
	}
	var n int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM network_ipv4_retirements").Scan(&n); e != nil || n != 257 {
		t.Fatal(n, e)
	}
	holds, e := s.ReadCreateAddressHolds(ctx, a.Layer, "bwbr0")
	if e != nil || holds.IPs[netip.MustParseAddr(slot)] {
		t.Fatal(e, holds)
	}
	c := retirementTestLifecycle(t, s, a, 258, slot)
	if c.Binding.IPv4 != slot {
		t.Fatal(c)
	}
	// Complete audit validates all archived identities even with empty live set.
	ledger, e := s.readCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-peer", true)
	if e != nil || len(ledger.Boxes) != 0 {
		t.Fatal(ledger, e)
	}
	t.Logf("N2_STATE_ONE_SLOT_257_REUSE_CYCLES=PASS elapsed=%s", time.Since(begin))
}

func TestIPv4RetirementStatusScopedDiagnostics(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	a := retirementTestAdmission(t, s, "user-1001")
	c := retirementTestLifecycle(t, s, a, 1, "10.24.18.11")
	before, err := s.ReadIPv4RetirementRollbackWitness(ctx, a.Layer)
	if err != nil || !validRetirementProofSHA(before) {
		t.Fatal(before, err)
	}
	ids, e := s.ClaimPendingIPv4Retirements(ctx, a.Layer, 2)
	if e != nil || len(ids) != 1 || ids[0] != c.DeleteOperationID {
		t.Fatal(ids, e)
	}
	if e = s.RecordIPv4RetirementDeferred(ctx, a.Layer, c.DeleteOperationID, "RETIREMENT_BRIDGE_NOT_QUIESCENT"); e != nil {
		t.Fatal(e)
	}
	got, e := s.ReadIPv4RetirementStatus(ctx, a.Layer)
	if e != nil || got.Pending != 1 || got.LastDeferredReason != "RETIREMENT_BRIDGE_NOT_QUIESCENT" {
		t.Fatal(got, e)
	}
	other, e := s.ReadIPv4RetirementStatus(ctx, "user-1002")
	if e != nil || other.Pending != 0 || other.LastDeferredReason != "" {
		t.Fatal(other, e)
	}
	if e = s.RecordIPv4RetirementDeferred(ctx, a.Layer, c.DeleteOperationID, "private root inventory"); e == nil {
		t.Fatal("raw error accepted")
	}
	if e = retireWithProof(s, ctx, c, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	after, err := s.ReadIPv4RetirementRollbackWitness(ctx, a.Layer)
	if err != nil || !validRetirementProofSHA(after) || after == before {
		t.Fatal(before, after, err)
	}
	got, e = s.ReadIPv4RetirementStatus(ctx, a.Layer)
	if e != nil || got.Pending != 0 || got.LastDeferredReason != "" {
		t.Fatal(got, e)
	}
}
