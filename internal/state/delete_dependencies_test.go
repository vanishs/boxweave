package state

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func b5DeleteAdmission(t *testing.T, s *Store, uuid, name string) Admission {
	t.Helper()
	_, tok, e := s.IssueToken(context.Background(), time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	return Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "b5-exact-delete-parent-001", RequestID: "b5-parent", Action: "box_delete", BoxUUID: uuid, Incarnation: uuid, ResourceKey: "box:" + name, SafeIntent: []byte("delete-parent"), CleanupDependencies: true, WorkRef: "b5-parent-work"}
}
func b5ForwardFixture(t *testing.T, s *Store, a Admission, port int) Receipt {
	t.Helper()
	ctx := context.Background()
	a.Action = "port_forward_create"
	a.CleanupDependencies = false
	a.Key = "b5-fwd-" + a.BoxUUID
	a.RequestID = a.Key
	a.ForwardCreate = &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: port, GuestPort: 8080}
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-fwd-fixture"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedForward(ctx, r.ID, true); e != nil {
		t.Fatal(e)
	}
	return r
}
func b5CheckBlocked(t *testing.T, s *Store, a Admission) {
	t.Helper()
	a.Action = "box_update"
	a.BoxUUID = s4b2RootThird
	a.Incarnation = s4b2RootThird
	a.ResourceKey = "box:c"
	a.CleanupDependencies = false
	a.Key = "b5-sibling-update-locked"
	if _, e := s.Admit(context.Background(), a); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("project fence not enforced: %v", e)
	}
}
func TestB5DeleteForwardParentChildAtomicAndReplay(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	b4OwnedFixture(t, s, "user-1001", "a", s4b2RootSrc)
	a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
	original := b5ForwardFixture(t, s, a, 24105)
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	b5CheckBlocked(t, s, a)
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	cascade, claims, e := s.DeleteCascadePlan(ctx, r.ID)
	if e != nil || !cascade || len(claims) != 1 || claims[0].ForwardID != original.ID {
		t.Fatalf("plan %v %+v %v", cascade, claims, e)
	}
	child, e := s.NextDeleteChild(ctx, r.ID, true)
	if e != nil || child.Receipt.Action != "port_forward_delete" || child.Forward.ForwardID != original.ID {
		t.Fatalf("child %+v %v", child, e)
	}
	if _, e = s.NextDeleteChild(ctx, r.ID, true); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("parallel child admitted %v", e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("premature final", e)
	}
	b5CheckBlocked(t, s, a)
	if e = s.Claim(ctx, child.Receipt.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, child.Receipt.ID, "native-forward-delete"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedForward(ctx, child.Receipt.ID, false); e != nil {
		t.Fatal(e)
	}
	stop, e := s.NextDeleteChild(ctx, r.ID, true)
	if e != nil || stop.Receipt.Action != "box_stop" {
		t.Fatalf("stop %+v %v", stop, e)
	}
	if e = s.Claim(ctx, stop.Receipt.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, stop.Receipt.ID, "native-stop"); e != nil {
		t.Fatal(e)
	}
	if e = s.Verified(ctx, stop.Receipt.ID); e != nil {
		t.Fatal(e)
	}
	if done, e := s.NextDeleteChild(ctx, r.ID, false); e != nil || done.Receipt.ID != "" {
		t.Fatalf("final stage %+v %v", done, e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-box-delete"); e != nil {
		t.Fatal(e)
	}
	b, e := s.OwnedBox(ctx, a.Layer, a.BoxUUID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedDelete(ctx, r.ID, b); e != nil {
		t.Fatal(e)
	}
	got, e := s.Replay(ctx, a)
	if e != nil || got.ID != r.ID || !got.Duplicate || got.State != "SUCCEEDED" {
		t.Fatalf("replay %+v %v", got, e)
	}
	if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); !errors.Is(e, ErrNotFound) {
		t.Fatal("missing tombstone", e)
	}
	f, e := s.GetForward(ctx, a.Layer, original.ID)
	if e != nil || f.State != "REVOKED" || f.OperationID != child.Receipt.ID {
		t.Fatalf("forward %+v %v", f, e)
	}
}
func b5Peer(t *testing.T, s *Store, src, dst, sv4, dv4, sv6, dv6 string) {
	t.Helper()
	_, e := s.db.Exec("INSERT INTO peer_grants(project,source_uuid,source_incarnation,target_uuid,target_incarnation,desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6) VALUES('user-1001',?,?,?,?,1,'CONFIRMED',?,?,?,?)", src, src, dst, dst, sv4, dv4, sv6, dv6)
	if e != nil {
		t.Fatal(e)
	}
}
func b5RootReader(t *testing.T, s *Store) *Store {
	t.Helper()
	var n int
	var name, path string
	if e := s.db.QueryRow("PRAGMA database_list").Scan(&n, &name, &path); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(path, 0600); e != nil {
		t.Fatal(e)
	}
	reader, e := OpenNetworkAuditReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = reader.Close() })
	return reader
}
func TestB5DeletePeerRootLinkedProofAndRetainedDropHistory(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	b5Peer(t, s, s4b2RootSrc, s4b2RootDst, "10.243.18.11", "10.243.18.12", "fd43:18::11", "fd43:18::12")
	b5Peer(t, s, s4b2RootThird, s4b2RootSrc, "10.243.18.13", "10.243.18.11", "fd43:18::13", "fd43:18::11")
	s4b3RootAddOtherConfirmed(t, s)
	a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	root := b5RootReader(t, s)
	for i := 0; i < 2; i++ {
		child, e := s.NextDeleteChild(ctx, r.ID, true)
		if e != nil || child.Receipt.Action != "box_peer_set" {
			t.Fatalf("peer child %+v %v", child, e)
		}
		id := child.Receipt.ID
		if e = s.StageRootPeerRevocation(ctx, id); e != nil {
			t.Fatal(e)
		}
		op, e := s.Get(ctx, id)
		if e != nil || op.State != "DISPATCHING" {
			t.Fatalf("non-atomic child claim %+v %v", op, e)
		}
		proof, e := root.ReadRootPeerRevocation(ctx, id, a.Layer, "bwbr0", "bw-s4b2")
		if e != nil || proof.Intent.Allow {
			t.Fatalf("root proof %+v %v", proof, e)
		}
		// Independent read-only root proof must reject a forged parent link.
		if _, e = s.db.Exec("UPDATE operation_resource_claims SET resource_key=? WHERE operation_id=? AND resource_key=?", deleteParentKey("00000000000000000000000000000000"), id, deleteParentKey(r.ID)); e != nil {
			t.Fatal(e)
		}
		if _, e = root.ReadRootPeerRevocation(ctx, id, a.Layer, "bwbr0", "bw-s4b2"); e == nil {
			t.Fatal("forged parent link accepted")
		}
		if _, e = s.db.Exec("UPDATE operation_resource_claims SET resource_key=? WHERE operation_id=? AND resource_key=?", deleteParentKey(r.ID), id, deleteParentKey("00000000000000000000000000000000")); e != nil {
			t.Fatal(e)
		}
		before, after, e := root.ReadPeerRevocationSnapshot(ctx, a.Layer, "bwbr0", "bw-s4b2", id)
		if e != nil || len(before) != 3-i || len(after) != 2-i {
			t.Fatalf("before/after %v %v %v", before, after, e)
		}
		if e = s.AckSynchronousACL(ctx, id); e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeRootPeerRevocation(ctx, id); e != nil {
			t.Fatal(e)
		}
	}
	if done, e := s.NextDeleteChild(ctx, r.ID, false); e != nil || done.Receipt.ID != "" {
		t.Fatalf("final %+v %v", done, e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-delete-proof"); e != nil {
		t.Fatal(e)
	}
	box, e := s.OwnedBox(ctx, a.Layer, a.BoxUUID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedDelete(ctx, r.ID, box); e != nil {
		t.Fatal(e)
	}
	live, e := root.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-s4b2")
	if e != nil || len(live.Boxes) != 2 {
		t.Fatalf("remaining audit %+v %v", live, e)
	}
	drops, e := root.ReadPeerRevokedSnapshot(ctx, a.Layer, "bwbr0", "bw-s4b2")
	if e != nil || len(drops) != 2 {
		t.Fatalf("historic drops %+v %v", drops, e)
	}
	active, e := root.ReadPeerGrantSnapshot(ctx, a.Layer, "bwbr0", "bw-s4b2", "")
	if e != nil || len(active) != 1 || active[0].SourceUUID != s4b2RootDst || active[0].TargetUUID != s4b2RootThird {
		t.Fatalf("unrelated grant %+v %v", active, e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='forged' WHERE box_uuid=?", a.BoxUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = root.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-s4b2"); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("forged release history accepted %v", e)
	}
}
func TestB5DeleteRestartPartialCleanupUnknownFences(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	b4OwnedFixture(t, s, "user-1001", "a", s4b2RootSrc)
	a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
	original := b5ForwardFixture(t, s, a, 24106)
	r, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	child, e := s.NextDeleteChild(ctx, r.ID, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, child.Receipt.ID); e != nil {
		t.Fatal(e)
	}
	if n, e := s.Recover(ctx); e != nil || n != 2 {
		t.Fatalf("recover %d %v", n, e)
	}
	for _, id := range []string{r.ID, child.Receipt.ID} {
		rec, e := s.Get(ctx, id)
		if e != nil || rec.State != "UNKNOWN" {
			t.Fatalf("lost original %+v %v", rec, e)
		}
	}
	f, e := s.GetForward(ctx, a.Layer, original.ID)
	if e != nil || f.State != "PENDING" {
		t.Fatalf("unsafe release %+v %v", f, e)
	}
	b5CheckBlocked(t, s, a)
	if _, e = s.NextDeleteChild(ctx, r.ID, true); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("replayed original %v", e)
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = checkCascadeRelease(ctx, tx, r.ID); !errors.Is(e, ErrResourceLocked) {
		t.Fatalf("generic release unsafe %v", e)
	}
}

func TestB5ReviewKnownTerminalDeleteAtomicPinnedTombstone(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	s4b2InsertAuditFixture(t, s)
	a := b5DeleteAdmission(t, s, s4b2RootSrc, "a")
	parent, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, parent.ID, "known-terminal-native-id"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	orig, e := s.RecoveryInspect(ctx, parent.ID)
	if e != nil {
		t.Fatal(e)
	}
	observed := "DELETE_SUCCESS:unit-fixture"
	challenge, e := s.IssueRecoveryChallenge(ctx, orig, "b5-reviewer", observed)
	if e != nil {
		t.Fatal(e)
	}
	for _, test := range []struct {
		who, uuid, proof string
		ack              bool
	}{
		{"wrong-reviewer", a.BoxUUID, observed, true},
		{"b5-reviewer", "other-uuid", observed, true},
		{"b5-reviewer", a.BoxUUID, "DELETE_SUCCESS:changed", true},
		{"b5-reviewer", a.BoxUUID, observed, false},
	} {
		if e = s.FinalizeReviewedUnknownDelete(ctx, orig, test.who, challenge, test.uuid, "unit fixture independently verified", test.proof, test.ack); e == nil {
			t.Fatalf("unsafe review accepted %+v", test)
		}
	}
	if _, e = s.db.Exec("CREATE TRIGGER b5_inject_audit_failure BEFORE INSERT ON recovery_audit WHEN NEW.decision='DELETE_CONFIRMED' BEGIN SELECT RAISE(ABORT,'test audit failure'); END"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeReviewedUnknownDelete(ctx, orig, "b5-reviewer", challenge, a.BoxUUID, "unit fixture independently verified", observed, true); e == nil {
		t.Fatal("audit failure committed")
	}
	r, e := s.Get(ctx, parent.ID)
	if e != nil || r.State != "UNKNOWN" {
		t.Fatalf("non-atomic settlement %+v %v", r, e)
	}
	if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); e != nil {
		t.Fatal("tombstone not rolled back", e)
	}
	if _, e = s.db.Exec("DROP TRIGGER b5_inject_audit_failure"); e != nil {
		t.Fatal(e)
	}
	root := b5RootReader(t, s)
	if _, e = root.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-s4b2"); e != nil {
		t.Fatal("fences not rolled back", e)
	}
	if e = s.FinalizeReviewedUnknownDelete(ctx, orig, "b5-reviewer", challenge, a.BoxUUID, "unit fixture independently verified", observed, true); e != nil {
		t.Fatal(e)
	}
	if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); !errors.Is(e, ErrNotFound) {
		t.Fatal("no tombstone", e)
	}
	if ledger, e := root.ReadCompleteNetworkAuditLedger(ctx, a.Layer, "bwbr0", "bw-s4b2"); e != nil || len(ledger.Boxes) != 2 {
		t.Fatalf("postreview history %+v %v", ledger, e)
	}
	replay, e := s.Replay(ctx, a)
	if e != nil || !replay.Duplicate || replay.ID != parent.ID || replay.State != "SUCCEEDED" {
		t.Fatalf("replay %+v %v", replay, e)
	}
}
