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

func TestB4ForwardQueuedCancellationAndRestartRelease(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	const project = "user-1001"
	const uuid = "33333333-3333-4333-8333-333333333333"
	b4OwnedFixture(t, s, project, "forward-recovery", uuid)
	a := Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Action: "port_forward_create", Key: "b4-cancel-before-dispatch-01", RequestID: "r-1", BoxUUID: uuid, Incarnation: uuid, ResourceKey: "box:forward-recovery", ResourceKeys: []string{"box:" + uuid, "listener:tcp:127.0.0.1:23051"}, SafeIntent: []byte("b4-recover"), ForwardCreate: &ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 23051, GuestPort: 8080}}
	makeClaim := func(key string) Receipt {
		t.Helper()
		a.Key = key
		a.RequestID = key
		r, e := s.Admit(ctx, a)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	verifyRelease := func(r Receipt, want string) {
		t.Helper()
		got, e := s.Get(ctx, r.ID)
		if e != nil || got.State != want {
			t.Fatalf("original %+v %v", got, e)
		}
		f, e := s.GetForward(ctx, project, r.ID)
		if e != nil || f.State != "REVOKED" {
			t.Fatalf("reservation not released atomically: %+v %v", f, e)
		}
	}
	first := makeClaim("b4-cancel-before-dispatch-01")
	if err = s.CancelQueued(ctx, project, first.ID); err != nil {
		t.Fatal(err)
	}
	verifyRelease(first, "CANCELLED")
	second := makeClaim("b4-recover-before-dispatch-02")
	if n, e := s.Recover(ctx); e != nil || n != 1 {
		t.Fatalf("recover %d %v", n, e)
	}
	verifyRelease(second, "FAILED")
	third := makeClaim("b4-preflight-before-dispatch-03")
	if err = s.Claim(ctx, third.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.FailPreDispatch(ctx, third.ID, "PROVED_NO_EFFECT"); err != nil {
		t.Fatal(err)
	}
	verifyRelease(third, "FAILED")
	fourth := makeClaim("b4-forward-for-delete-restart-04")
	if err = s.Claim(ctx, fourth.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.NativeAck(ctx, fourth.ID, "native-proxy-4"); err != nil {
		t.Fatal(err)
	}
	if err = s.FinalizeVerifiedForward(ctx, fourth.ID, true); err != nil {
		t.Fatal(err)
	}
	del := a
	del.Action = "port_forward_delete"
	del.Key = "b4-delete-queued-restart-05"
	del.RequestID = del.Key
	del.ForwardCreate = nil
	del.ForwardDeleteID = fourth.ID
	d, err := s.Admit(ctx, del)
	if err != nil {
		t.Fatal(err)
	}
	if n, e := s.Recover(ctx); e != nil || n != 1 {
		t.Fatalf("recover delete %d %v", n, e)
	}
	op, err := s.Get(ctx, d.ID)
	if err != nil || op.State != "FAILED" {
		t.Fatalf("delete %+v %v", op, err)
	}
	f, err := s.GetForward(ctx, project, fourth.ID)
	if err != nil || f.State != "CONFIRMED" {
		t.Fatalf("old forward not restored %+v %v", f, err)
	}
	del.Key = "b4-delete-retry-fresh-06"
	del.RequestID = del.Key
	if _, err = s.Admit(ctx, del); err != nil {
		t.Fatal(err)
	}
}

func TestForwardWildcardReservation(t *testing.T) {
	for _, pair := range [][2]string{{"0.0.0.0", "127.0.0.1"}, {"127.0.0.1", "0.0.0.0"}, {"::", "127.0.0.1"}, {"127.0.0.1", "::"}, {"::", "::1"}, {"::1", "::"}, {"0.0.0.0", "::"}} {
		t.Run(pair[0]+"-then-"+pair[1], func(t *testing.T) {
			ctx := context.Background()
			s := newStore(t)
			_, tok, err := s.IssueToken(ctx, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			project := "user-1001"
			a := "11111111-1111-4111-8111-111111111111"
			b := "22222222-2222-4222-8222-222222222222"
			b4OwnedFixture(t, s, project, "forward-a", a)
			b4OwnedFixture(t, s, project, "forward-b", b)
			makeAdmission := func(name, uuid, bind, proto, key string) Admission {
				return Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: key, Action: "port_forward_create", RequestID: key, BoxUUID: uuid, Incarnation: uuid,
					ResourceKey: "box:" + name, ResourceKeys: []string{"box:" + uuid, "listener:" + proto + ":" + bind + ":23051"}, SafeIntent: []byte(key),
					ForwardCreate: &ForwardCreateAdmission{Protocol: proto, HostBind: bind, HostPort: 23051, GuestPort: 8080}}
			}
			first := makeAdmission("forward-a", a, pair[0], "tcp", "wildcard-original-001")
			r, err := s.Admit(ctx, first)
			if err != nil {
				t.Fatal(err)
			}
			second := makeAdmission("forward-b", b, pair[1], "tcp", "wildcard-conflicting-002")
			if _, err = s.Admit(ctx, second); !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("overlapping pending reservation admitted: %v", err)
			}
			if err = s.Claim(ctx, r.ID); err != nil {
				t.Fatal(err)
			}
			if n, err := s.Recover(ctx); err != nil || n != 1 {
				t.Fatalf("recover: %d %v", n, err)
			}
			if _, err = s.Admit(ctx, second); !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("overlapping UNKNOWN reservation admitted: %v", err)
			}
			// Independent protocol is not reserved by TCP, even with wildcard.
			udp := makeAdmission("forward-b", b, pair[1], "udp", "wildcard-separate-udp-003")
			if _, err = s.Admit(ctx, udp); err != nil {
				t.Fatalf("unrelated UDP denied: %v", err)
			}
		})
	}
}
