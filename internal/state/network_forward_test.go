package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func b4OwnedFixture(t *testing.T, s *Store, project, name, uuid string) {
	t.Helper()
	_, err := s.db.Exec("INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES (?,?,?,?,?,?)", uuid, project, name, uuid, "OWNED", time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
}
func TestB4ForwardAtomicAdmissionConfirmDeleteAndUnknownFence(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	project := "user-1001"
	aID := "11111111-1111-4111-8111-111111111111"
	bID := "22222222-2222-4222-8222-222222222222"
	b4OwnedFixture(t, s, project, "forward-a", aID)
	b4OwnedFixture(t, s, project, "forward-b", bID)
	f := &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 23001, GuestPort: 8080}
	a := Admission{
		Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "s4b4-forward-create-original-001", Action: "port_forward_create",
		RequestID: "request-1", BoxUUID: aID, Incarnation: aID,
		ResourceKey: "box:forward-a", ResourceKeys: []string{"box:" + aID, "listener:tcp:127.0.0.1:23001"},
		SafeIntent: []byte("forward-create-001"), ForwardCreate: f,
	}
	created, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := s.GetForward(ctx, project, created.ID)
	if err != nil || claim.State != "PENDING" || claim.NativeDevice != "bwpf-"+created.ID[:16] || claim.OperationID != created.ID {
		t.Fatalf("not atomic: %+v %v", claim, err)
	}
	same, err := s.Admit(ctx, a)
	if err != nil || !same.Duplicate || same.ID != created.ID {
		t.Fatalf("replay failed: %+v %v", same, err)
	}
	collide := a
	collide.Key = "s4b4-forward-collision-original-002"
	collide.BoxUUID = bID
	collide.Incarnation = bID
	collide.ResourceKey = "box:forward-b"
	collide.ResourceKeys = []string{"box:" + bID, "listener:tcp:127.0.0.1:23001"}
	if _, err := s.Admit(ctx, collide); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("listener collision admitted %v", err)
	}
	if err := s.Claim(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.NativeAck(ctx, created.ID, "native-fwd-create-001"); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeVerifiedForward(ctx, created.ID, true); err != nil {
		t.Fatal(err)
	}
	stateNow, err := s.Get(ctx, created.ID)
	if err != nil || stateNow.State != "SUCCEEDED" {
		t.Fatalf("not terminal %+v %v", stateNow, err)
	}
	claim, err = s.GetForward(ctx, project, created.ID)
	if err != nil || claim.State != "CONFIRMED" {
		t.Fatalf("not confirmed: %+v %v", claim, err)
	}
	if _, err = s.GetForward(ctx, "foreign-project", created.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross project exposed %v", err)
	}
	all, next, err := s.ListForwards(ctx, project, "", "", 50)
	if err != nil || next != "" || len(all) != 1 {
		t.Fatalf("list %+v %q %v", all, next, err)
	}
	del := Admission{
		Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "s4b4-forward-delete-original-003", Action: "port_forward_delete",
		RequestID: "request-3", BoxUUID: aID, Incarnation: aID, ResourceKey: "box:forward-a",
		ResourceKeys: []string{"box:" + aID, "listener:tcp:127.0.0.1:23001"},
		SafeIntent:   []byte("forward-delete-003"), ForwardDeleteID: created.ID,
	}
	removed, err := s.Admit(ctx, del)
	if err != nil {
		t.Fatal(err)
	}
	claim, _ = s.GetForward(ctx, project, created.ID)
	if claim.State != "PENDING" || claim.OperationID != removed.ID {
		t.Fatalf("delete not staged %+v", claim)
	}
	if err = s.Claim(ctx, removed.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.NativeAck(ctx, removed.ID, "native-fwd-delete-003"); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeVerifiedForward(ctx, removed.ID, false); err != nil {
		t.Fatal(err)
	}
	claim, _ = s.GetForward(ctx, project, created.ID)
	if claim.State != "REVOKED" {
		t.Fatalf("delete not settled %+v", claim)
	}
	all, _, err = s.ListForwards(ctx, project, "", "", 50)
	if err != nil || len(all) != 0 {
		t.Fatalf("revoked visible %+v %v", all, err)
	}
	unknown := a
	unknown.Key = "s4b4-forward-unknown-original-004"
	unknown.RequestID = "request-4"
	unknown.SafeIntent = []byte("forward-create-004")
	unknown.ForwardCreate = &ForwardCreateAdmission{Protocol: "udp", HostBind: "127.0.0.1", HostPort: 23002, GuestPort: 9000}
	unknown.ResourceKeys = []string{"box:" + aID, "listener:udp:127.0.0.1:23002"}
	uncertain, err := s.Admit(ctx, unknown)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, uncertain.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := s.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("recover %+v %v", n, err)
	}
	claim, _ = s.GetForward(ctx, project, uncertain.ID)
	if claim.State != "PENDING" {
		t.Fatalf("unknown released active forward %+v", claim)
	}
	retry := unknown
	retry.Key = "s4b4-forward-new-key-after-unknown-005"
	if _, err = s.Admit(ctx, retry); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("unknown was blindly released %v", err)
	}
}
