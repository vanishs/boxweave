package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func n3AdmittedNoEffectCreate(t *testing.T) (*Store, Admission, Receipt, VerifiedCreateBinding) {
	t.Helper()
	ctx := context.Background()
	s := newStore(t)
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	a := Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID, Key: "n3-create-predispatch-original", RequestID: "create", Action: "box_create", ResourceKey: "box:alpha", SafeIntent: []byte(`{"name":"alpha","cpu":1,"memory_mib":1024,"disk_gib":10,"start":false,"nesting":false,"internet":true,"instance_options":{}}`)}
	rec, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	return s, a, rec, VerifiedCreateBinding{"bwbr0", "bw-peer", "02:16:3e:11:22:33", "10.24.18.11", "fd43:18::11"}
}
func n3CountOriginalStaticHolds(t *testing.T, s *Store, id string) int {
	t.Helper()
	var n int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", id).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}
func TestN3WorkerVerifiedNoDispatchCreateReturnsExactReservation(t *testing.T) {
	ctx := context.Background()
	s, a, r, pin := n3AdmittedNoEffectCreate(t)
	called, verified := 0, 0
	Worker{Store: s, Budget: time.Second}.Execute(r.ID, func(ctx context.Context) (string, error) {
		called++
		if ok, e := s.StageCreateBinding(ctx, r.ID, a.Layer, "alpha", pin); !ok || e != nil {
			t.Fatal(ok, e)
		}
		return "", PreDispatchError{Class: "VERIFIED_LOCAL_CREATE_PRE_DISPATCH"}
	}, func(context.Context) error { verified++; return nil })
	if called != 1 || verified != 0 {
		t.Fatal("unexpected native verifier", called, verified)
	}
	old, e := s.Get(ctx, r.ID)
	if e != nil || old.State != "FAILED" || old.NativeOperationID != "" || old.ErrorClass != "VERIFIED_LOCAL_CREATE_PRE_DISPATCH" {
		t.Fatal(old, e)
	}
	if n := n3CountOriginalStaticHolds(t, s, r.ID); n != 0 {
		t.Fatal("proved-zero-effect reservations leaked", n)
	}
	var ev int
	if e := s.db.QueryRow("SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='FAILED' AND safe_detail='PREDISPATCH_STATIC_RESERVATIONS_RELEASED_NO_NATIVE'", r.ID).Scan(&ev); e != nil || ev != 1 {
		t.Fatal(ev, e)
	}
	replay, e := s.Replay(ctx, a)
	if e != nil || !replay.Duplicate || replay.ID != r.ID || replay.State != "FAILED" {
		t.Fatal(replay, e)
	}
	a.Key = "n3-reuse-proved-zero-dispatch-address"
	a.RequestID = "recreate"
	next, e := s.Admit(ctx, a)
	if e != nil {
		t.Fatal("proved unused Box name still blocked", e)
	}
	if e = s.Claim(ctx, next.ID); e != nil {
		t.Fatal(e)
	}
	if ok, e := s.StageCreateBinding(ctx, next.ID, a.Layer, "alpha", pin); e != nil || !ok {
		t.Fatal("safe static address not reusable", ok, e)
	}
}
func TestN3CreateUncertainOrDispatchedFailureRetainsStaticHolds(t *testing.T) {
	for _, tc := range []struct {
		name, native string
		err          error
	}{
		{"ambiguous native effect", "", errors.New("unknown SDK transport outcome")},
		{"native receipt with effect error", "native-accepted-original", errors.New("accepted with SDK error")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, a, r, pin := n3AdmittedNoEffectCreate(t)
			Worker{Store: s, Budget: time.Second}.Execute(r.ID, func(ctx context.Context) (string, error) {
				if ok, e := s.StageCreateBinding(ctx, r.ID, a.Layer, "alpha", pin); e != nil || !ok {
					t.Fatal(ok, e)
				}
				return tc.native, tc.err
			}, nil)
			got, e := s.Get(context.Background(), r.ID)
			if e != nil || got.State != "UNKNOWN" || n3CountOriginalStaticHolds(t, s, r.ID) != 4 {
				t.Fatal("unknown released native reservations", got, e)
			}
		})
	}
}

func TestN3NoEffectCreateReservationReleaseAtomicFaultAndConflict(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []string{"delete journal fault", "missing pin claim", "native ack already recorded", "owned identity appeared"} {
		t.Run(tc, func(t *testing.T) {
			s, a, r, pin := n3AdmittedNoEffectCreate(t)
			if e := s.Claim(ctx, r.ID); e != nil {
				t.Fatal(e)
			}
			if ok, e := s.StageCreateBinding(ctx, r.ID, a.Layer, "alpha", pin); e != nil || !ok {
				t.Fatal(ok, e)
			}
			switch tc {
			case "delete journal fault":
				if _, e := s.db.Exec("CREATE TRIGGER n3_predispatch_rollback BEFORE DELETE ON operation_resource_claims WHEN OLD.resource_key LIKE 'create-pin:%' BEGIN SELECT RAISE(ABORT,'fixture pre-dispatch rollback'); END"); e != nil {
					t.Fatal(e)
				}
			case "missing pin claim":
				if _, e := s.db.Exec("DELETE FROM operation_resource_claims WHERE operation_id=? AND resource_key=?", r.ID, pin.reservationKeys()[0]); e != nil {
					t.Fatal(e)
				}
			case "native ack already recorded":
				if e := s.NativeAck(ctx, r.ID, "sent-to-incus"); e != nil {
					t.Fatal(e)
				}
			case "owned identity appeared":
				if e := s.RegisterOwnedBox(ctx, "22222222-2222-4222-8222-222222222222", a.Layer, "alpha", "22222222-2222-4222-8222-222222222222"); e != nil {
					t.Fatal(e)
				}
			}
			e := s.FailPreDispatch(ctx, r.ID, "LOCAL_NO_DISPATCH_CERTAIN")
			if e == nil {
				t.Fatal("tampered reservation prematurely released", tc)
			}
			got, gerr := s.Get(ctx, r.ID)
			wantState := "DISPATCHING"
			if tc == "native ack already recorded" {
				wantState = "VERIFYING"
			}
			if gerr != nil || got.State != wantState {
				t.Fatal("failure not atomic", tc, got, e, gerr)
			}
			want := 4
			if tc == "missing pin claim" {
				want = 3
			}
			if n := n3CountOriginalStaticHolds(t, s, r.ID); n != want {
				t.Fatal("partial release", n, want)
			}
			if tc == "delete journal fault" {
				if _, e := s.db.Exec("DROP TRIGGER n3_predispatch_rollback"); e != nil {
					t.Fatal(e)
				}
				if e := s.FailPreDispatch(ctx, r.ID, "LOCAL_NO_DISPATCH_CERTAIN"); e != nil {
					t.Fatal(e)
				}
				if n := n3CountOriginalStaticHolds(t, s, r.ID); n != 0 {
					t.Fatal("postfault safe release failed", n)
				}
				got, e := s.Get(ctx, r.ID)
				if e != nil || got.State != "FAILED" {
					t.Fatal(got, e)
				}
			}
		})
	}
}
