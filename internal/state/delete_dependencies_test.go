package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
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
	s, b, token := n3ScopedOwnedFixture(t)
	for i, p := range [][2]int{{1, 2}, {3, 1}, {2, 3}} {
		purePeerComplete(t, s, purePeerAdmit(t, s, token, fmt.Sprint("delete-grant-", i), p[0], p[1], true))
	}
	a := scopedDeleteAdmission(t, s, b)
	r, e := s.AdmitScoped(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	ro := purePeerReadOnly(t, s)
	for i := 0; i < 2; i++ {
		child, e := s.NextDeleteChild(ctx, r.ID, true)
		if e != nil || child.Receipt.Action != "box_peer_set" {
			t.Fatal(child, e)
		}
		id := child.Receipt.ID
		if e = s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
			t.Fatal(e)
		}
		p, e := ro.ReadRootScopedPeerOriginal(ctx, id, b.Project, purePeerBridge)
		if e != nil || p.ParentID != r.ID || p.Intent.Allow || len(p.Before) != 3-i || len(p.After) != 2-i {
			t.Fatal(p, e)
		}
		bad := deleteParentKey(strings.Repeat("0", 32))
		if _, e = s.db.Exec("UPDATE operation_resource_claims SET resource_key=? WHERE operation_id=? AND resource_key=?", bad, id, deleteParentKey(r.ID)); e != nil {
			t.Fatal(e)
		}
		if _, e = ro.ReadRootScopedPeerOriginal(ctx, id, b.Project, purePeerBridge); e == nil {
			t.Fatal("forged parent")
		}
		if _, e = s.db.Exec("UPDATE operation_resource_claims SET resource_key=? WHERE operation_id=? AND resource_key=?", deleteParentKey(r.ID), id, bad); e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeScopedPeerOriginal(ctx, scopedTestReceipt(t, ro, id, false)); e != nil {
			t.Fatal(e)
		}
	}
	if child, e := s.NextDeleteChild(ctx, r.ID, false); e != nil || child.Receipt.ID != "" {
		t.Fatal(child, e)
	}
	if e = s.AuthorizeCascadeFinal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, r.ID, "native-delete-proof"); e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeVerifiedScopedDelete(ctx, r.ID, b); e != nil {
		t.Fatal(e)
	}
	p, e := ro.ReadScopedTerminalPeerPolicy(ctx, b.Project, purePeerBridge)
	if e != nil || len(p.Revoked) != 2 || len(p.Active) != 1 || p.Active[0].SourceUUID != purePeerUUID(2) || p.Active[0].TargetUUID != purePeerUUID(3) {
		t.Fatal(p, e)
	}
	if _, e = s.db.Exec("UPDATE network_address_fences SET operation_id='forged' WHERE box_uuid=?", b.UUID); e != nil {
		t.Fatal(e)
	}
	if _, e = ro.ReadCompleteScopedNetworkAuditLedger(ctx, b.Project, purePeerBridge); !errors.Is(e, ErrResourceLocked) {
		t.Fatal(e)
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
	s, b, _ := n3ScopedOwnedFixture(t)
	a := scopedDeleteAdmission(t, s, b)
	parent := a4AdmitMode(t, s, a, true)
	if e := s.Claim(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.AuthorizeCascadeFinal(ctx, parent.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.NativeAck(ctx, parent.ID, "known-terminal-native-id"); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, parent.ID, "VERIFYING", "TEST_LOST_FINAL"); e != nil {
		t.Fatal(e)
	}
	old, e := s.RecoveryInspect(ctx, parent.ID)
	if e != nil {
		t.Fatal(e)
	}
	ch, e := s.IssueRecoveryChallenge(ctx, old, "b5-reviewer", "DELETE_SUCCESS:unit-fixture")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.FinalizeReviewedUnknownDelete(ctx, old, "b5-reviewer", ch, b.UUID, "unit fixture independently verified", "DELETE_SUCCESS:unit-fixture", true); e == nil {
		t.Fatal("legacy review bypass")
	}
	r, target := a4Request(t, s, parent.ID)
	root := purePeerReadOnly(t, s)
	orig, e := root.ReadRootScopedDeleteUnknownOriginal(ctx, parent.ID, b.Project, purePeerBridge)
	if e != nil {
		t.Fatal(e)
	}
	p := p1CompletedProof(t, target, "DELETE", orig.Binding, ScopedLifecycleOriginalSHA256(orig))
	p.UUID = b.UUID
	p.ScopedLifecycle.Record.UUID = b.UUID
	bad := p
	bad.ScopedLifecycle.OriginalSHA256 = strings.Repeat("d", 64)
	if _, e = s.CommitScopedRecovery(ctx, r, target, bad); e == nil {
		t.Fatal("wrong original proof")
	}
	if _, e = s.db.Exec("CREATE TRIGGER b5_inject_audit_failure BEFORE INSERT ON recovery_audit WHEN NEW.decision='RECOVERY_SETTLED' BEGIN SELECT RAISE(ABORT,'test audit failure'); END"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CommitScopedRecovery(ctx, r, target, p); e == nil {
		t.Fatal("audit failure committed")
	}
	p1AssertOriginal(t, s, parent.ID, "UNKNOWN", target.NativeID)
	if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DROP TRIGGER b5_inject_audit_failure"); e != nil {
		t.Fatal(e)
	}
	a4SettleMode(t, s, r, target, p, true)
	if _, e = s.OwnedBox(ctx, a.Layer, a.BoxUUID); !errors.Is(e, ErrNotFound) {
		t.Fatal("no tombstone", e)
	}
	replay, e := s.Replay(ctx, a)
	if e != nil || !replay.Duplicate || replay.ID != parent.ID || replay.State != "SUCCEEDED" {
		t.Fatal(replay, e)
	}

}

func scopedDeleteAdmission(t *testing.T, s *Store, b OwnedBox) Admission {
	a := b5DeleteAdmission(t, s, b.UUID, b.Name)
	a.Layer = b.Project
	a.SafeIntent, _ = json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{b.UUID})
	return a
}
