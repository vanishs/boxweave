package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func s4b2AdmitACL(t *testing.T, s *Store, token Token, key string) Receipt {
	t.Helper()
	r, err := s.Admit(context.Background(), Admission{
		Layer: "user-1001", ActorID: token.ID, TokenID: token.ID,
		Key: key, RequestID: "s4b2-native-acl",
		Action: "network_acl_reconcile", SafeIntent: []byte("bounded-exact-acl-etag-intent"),
		ResourceKey: "acl:default/bw-s4b2", ResourceKeys: []string{"box:source", "box:target"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestS4B2SynchronousACLTypedSuccessNoInventedNativeID(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := s4b2AdmitACL(t, s, tok, "s4b2-sync-success-000001")
	effectCount, verifyCount := 0, 0
	w := Worker{Store: s}
	w.ExecuteSynchronousACL(r.ID, func(context.Context) error { effectCount++; return nil },
		func(context.Context) error { verifyCount++; return nil })
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.State != "SUCCEEDED" || got.NativeOperationID != "" || effectCount != 1 || verifyCount != 1 {
		t.Fatalf("synchronous ACL receipt wrongly modeled: %+v %v effects=%d verify=%d", got, err, effectCount, verifyCount)
	}
	if err := w.SubmitSynchronousACL(r.ID, func(context.Context) error { t.Fatal("replayed"); return nil },
		func(context.Context) error { return nil }); !errors.Is(err, ErrState) {
		t.Fatalf("successful original redispatched: %v", err)
	}
}
func TestS4B2SynchronousACLUnknownPreservesLockAndNoRetry(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := s4b2AdmitACL(t, s, tok, "s4b2-sync-unknown-000001")
	effectCount := 0
	w := Worker{Store: s}
	w.ExecuteSynchronousACL(r.ID, func(context.Context) error { effectCount++; return errors.New("SDK HTTP timeout after possible PUT") },
		func(context.Context) error { t.Fatal("ambiguous SDK must not trust unverifiable postread"); return nil })
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.State != "UNKNOWN" || got.NativeOperationID != "" || effectCount != 1 {
		t.Fatalf("lost native sync ACK must be UNKNOWN: %+v %v", got, err)
	}
	_, err = s.Admit(ctx, Admission{
		Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-sync-bypass-attempt-00001", RequestID: "bypass",
		Action: "network_acl_reconcile", SafeIntent: []byte("malicious-retry"),
		ResourceKey: "acl:default/bw-s4b2",
	})
	if !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN ACL resource released for different request: %v", err)
	}
	if err := w.SubmitSynchronousACL(r.ID, func(context.Context) error { effectCount++; return nil },
		func(context.Context) error { return nil }); !errors.Is(err, ErrState) || effectCount != 1 {
		t.Fatalf("UNKNOWN redispatch: %v effects=%d", err, effectCount)
	}
}
func TestS4B2SynchronousACLFailedPostreadNeverClaimsSuccess(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r := s4b2AdmitACL(t, s, tok, "s4b2-sync-postread-00001")
	w := Worker{Store: s}
	w.ExecuteSynchronousACL(r.ID, func(context.Context) error { return nil },
		func(context.Context) error { return errors.New("one NIC failed kernel reload") })
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.State != "UNKNOWN" || got.NativeOperationID != "" {
		t.Fatalf("partial applied SDK ACK incorrectly successful: %+v %v", got, err)
	}
}
