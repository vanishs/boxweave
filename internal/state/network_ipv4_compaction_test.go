package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func compactionTestCreate(t *testing.T, s *Store, base Admission, i int, ip string) (OwnedBox, Admission) {
	t.Helper()
	ctx := context.Background()
	name := fmt.Sprintf("compact-box-%d", i)
	uuid := fmt.Sprintf("compact-exact-uuid-%d", i)
	a := base
	a.Action = "box_create"
	a.ResourceKey = "box:" + name
	a.Key = "compact-create-" + name
	a.RequestID = a.Key
	a.SafeIntent = []byte(fmt.Sprintf("{\"name\":%q}", name))
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	pin := VerifiedCreateBinding{"bwbr0", "bw-peer", fmt.Sprintf("02:16:3e:44:11:%02x", i), ip, fmt.Sprintf("fd43:18::%x", i+200)}
	ok, e := s.StageCreateBinding(ctx, r.ID, a.Layer, name, pin)
	if e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-create-"+name); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedCreateWithBinding(ctx, r.ID, a.Layer, name, uuid, pin); e != nil {
		t.Fatal(e)
	}
	b, e := s.OwnedBox(ctx, a.Layer, uuid)
	if e != nil {
		t.Fatal(e)
	}
	return b, a
}
func compactionTestPeer(t *testing.T, s *Store, base Admission, src, dst OwnedBox, allow bool) string {
	t.Helper()
	ctx := context.Background()
	a := base
	a.Action = "box_peer_set"
	a.ResourceKey = "acl:default/bw-peer"
	a.ResourceKeys = []string{"box:" + src.UUID, "box:" + dst.UUID}
	a.Key = fmt.Sprintf("compact-peer-%s-%s-%t", src.UUID, dst.UUID, allow)
	a.RequestID = a.Key
	a.BoxUUID = src.UUID
	a.Incarnation = src.Incarnation
	raw, e := json.Marshal(RootPeerRevokeSafeIntent{Project: a.Layer, Bridge: "bwbr0", ACL: "bw-peer", SourceUUID: src.UUID, SourceIncarnation: src.Incarnation, TargetUUID: dst.UUID, TargetIncarnation: dst.Incarnation, Allow: allow})
	if e != nil {
		t.Fatal(e)
	}
	a.SafeIntent = raw
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if allow {
		e = s.StagePeerGrant(ctx, r.ID)
	} else {
		e = s.StageRootPeerRevocation(ctx, r.ID)
	}
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AckSynchronousACL(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if allow {
		e = s.FinalizeRootPeerGrant(ctx, r.ID)
	} else {
		e = s.FinalizeRootPeerRevocation(ctx, r.ID)
	}
	if e != nil {
		t.Fatal(e)
	}
	return r.ID
}
func compactionTestDelete(t *testing.T, s *Store, base Admission, b OwnedBox) IPv4RetirementCandidate {
	t.Helper()
	ctx := context.Background()
	a := base
	a.Action = "box_delete"
	a.ResourceKey = "box:" + b.Name
	a.Key = "compact-delete-" + b.Name
	a.RequestID = a.Key
	a.BoxUUID = b.UUID
	a.Incarnation = b.Incarnation
	a.CleanupDependencies = true
	a.SafeIntent = []byte(fmt.Sprintf("{\"box_id\":%q}", b.UUID))
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-delete-"+b.Name); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedDelete(ctx, r.ID, b); e != nil {
		t.Fatal(e)
	}
	c, e := s.ReadIPv4RetirementCandidate(ctx, a.Layer, r.ID)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestIPv4PeerCompactionDurableFenceAndHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	path := strings.TrimSuffix(s.ExecArchiveDir(), ".exec-archive")
	a := retirementTestAdmission(t, s, "user-1001")
	one, original := compactionTestCreate(t, s, a, 1, "10.24.18.11")
	two, _ := compactionTestCreate(t, s, a, 2, "10.24.18.12")
	three, _ := compactionTestCreate(t, s, a, 3, "10.24.18.13")
	compactionTestPeer(t, s, a, one, two, true)
	revoke := compactionTestPeer(t, s, a, one, two, false)
	compactionTestPeer(t, s, a, two, three, true)
	keep := compactionTestPeer(t, s, a, two, three, false)
	c := compactionTestDelete(t, s, a, one)
	if !c.RequiresPeerCompaction {
		t.Fatal("old DROP not identified")
	}
	pol, e := s.ReadIPv4RetirementPolicy(ctx, c)
	if e != nil || len(pol.Revoked) != 2 || len(pol.AfterRevoked) != 1 || pol.AfterRevoked[0].SourceUUID != two.UUID {
		t.Fatal(pol, e)
	}
	if e = retireWithProof(s, ctx, c, strings.Repeat("a", 64)); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("missing compaction fence accepted", e)
	}
	// Invalid preview must leave admission usable.
	if ok, e := s.PrepareIPv4Compaction(ctx, a.Layer, c.DeleteOperationID, func(context.Context, IPv4RetirementCandidate) (string, error) { return "", ErrResourceLocked }); ok || e == nil {
		t.Fatal(ok, e)
	}
	if _, e = s.ReadIPv4CompactionPlan(ctx, a.Layer, c.DeleteOperationID); !errors.Is(e, ErrNotFound) {
		t.Fatal("failed preview left fence", e)
	}
	// A corrupt original revoke must not be silently compacted.
	if _, e = s.db.Exec("UPDATE peer_grants SET source_ipv4='10.24.18.99' WHERE operation_id=?", revoke); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadIPv4RetirementCandidate(ctx, a.Layer, c.DeleteOperationID); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("corrupt peer accepted", e)
	}
	if _, e = s.db.Exec("UPDATE peer_grants SET source_ipv4=? WHERE operation_id=?", c.Binding.IPv4, revoke); e != nil {
		t.Fatal(e)
	}
	plan := strings.Repeat("b", 64)
	if ok, e := s.PrepareIPv4Compaction(ctx, a.Layer, c.DeleteOperationID, func(_ context.Context, got IPv4RetirementCandidate) (string, error) {
		if got != c {
			t.Fatal(got, c)
		}
		return plan, nil
	}); !ok || e != nil {
		t.Fatal(ok, e)
	}
	// Exact original replay remains available; unrelated effects fail closed.
	if r, e := s.Admit(ctx, original); e != nil || !r.Duplicate {
		t.Fatal(r, e)
	}
	fresh := original
	fresh.Key = "compact-new-write-refused"
	fresh.ResourceKey = "box:replacement"
	if _, e = s.Admit(ctx, fresh); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("admission bypass", e)
	}
	if e = s.RetireVerifiedIPv4(ctx, a.Layer, c.DeleteOperationID, func(context.Context, IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error) {
		return IPv4RetirementCandidate{}, "", context.DeadlineExceeded
	}); e == nil {
		t.Fatal("lost root ACK accepted")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if got, e := s.ReadIPv4CompactionPlan(ctx, a.Layer, c.DeleteOperationID); e != nil || got != plan {
		t.Fatal(got, e)
	}
	if _, e = s.Admit(ctx, fresh); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("restart reopened fence", e)
	}
	// Resuming a prepared original must not re-preview or erase the plan.
	if ok, e := s.PrepareIPv4Compaction(ctx, a.Layer, c.DeleteOperationID, func(context.Context, IPv4RetirementCandidate) (string, error) {
		t.Fatal("prepared plan was replaced")
		return "", ErrInvalid
	}); !ok || e != nil {
		t.Fatal(ok, e)
	}
	// Inject final commit fault: the original Project and address fences
	// survive even after a trusted root proof was received.
	if _, e = s.db.Exec("CREATE TRIGGER compact_abort BEFORE DELETE ON network_ipv4_compactions BEGIN SELECT RAISE(ABORT,'test fault'); END"); e != nil {
		t.Fatal(e)
	}
	if e = retireWithProof(s, ctx, c, strings.Repeat("a", 64)); e == nil {
		t.Fatal("split compaction accepted")
	}
	if _, e = s.ReadIPv4CompactionPlan(ctx, a.Layer, c.DeleteOperationID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DROP TRIGGER compact_abort"); e != nil {
		t.Fatal(e)
	}
	if e = retireWithProof(s, ctx, c, strings.Repeat("a", 64)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadIPv4CompactionPlan(ctx, a.Layer, c.DeleteOperationID); !errors.Is(e, ErrNotFound) {
		t.Fatal(e)
	}
	rev, e := s.ReadPeerRevokedSnapshot(ctx, a.Layer, "bwbr0", "bw-peer")
	if e != nil || len(rev) != 1 || rev[0].SourceUUID != two.UUID {
		t.Fatal(rev, e)
	}
	var count int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id IN (?,?) AND status='REVOKED'", revoke, keep).Scan(&count); e != nil || count != 2 {
		t.Fatal(count, e)
	}
	newBox, _ := compactionTestCreate(t, s, a, 4, c.Binding.IPv4)
	if newBox.UUID == one.UUID {
		t.Fatal("identity reused")
	}
	live, e := s.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-peer")
	if e != nil || len(live.Boxes) != 3 {
		t.Fatal(live, e)
	}
	// Corrupting an archived revoke or deleting its receipt invalidates
	// all future address/policy evidence, not only the next grant.
	if _, e = s.db.Exec("UPDATE peer_grants SET status='CONFIRMED',desired=1 WHERE operation_id=?", revoke); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-peer"); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("archived peer corruption not caught", e)
	}
}
