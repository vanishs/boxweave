package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestS4B2NetworkMultiResourceAdmissionAndUnknownFence(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	first := Admission{
		Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-network-key-first-001", RequestID: "request-first",
		Action: "network-test-only", SafeIntent: []byte("first"),
		ResourceKey:  "box:alpha",
		ResourceKeys: []string{"box:beta", "acl:default/bw-s4-test"},
	}
	a, err := s.Admit(ctx, first)
	if err != nil {
		t.Fatal(err)
	}
	// The normalized set order is canonical, not an alternative key.
	first.ResourceKeys = []string{"acl:default/bw-s4-test", "box:beta"}
	same, err := s.Admit(ctx, first)
	if err != nil || !same.Duplicate || same.ID != a.ID {
		t.Fatalf("unstable replay %+v %v", same, err)
	}
	altered := first
	altered.ResourceKeys = []string{"box:beta"}
	if _, err := s.Admit(ctx, altered); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed immutable lock-set was accepted: %v", err)
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=?", a.ID).Scan(&count); err != nil || count != 3 {
		t.Fatalf("incomplete atomic resource reservation: %d %v", count, err)
	}
	conflicting := Admission{
		Layer: first.Layer, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-network-key-second-001", RequestID: "request-second",
		Action: "network-test-only", SafeIntent: []byte("second"),
		ResourceKey: "box:gamma", ResourceKeys: []string{"acl:default/bw-s4-test"},
	}
	if _, err := s.Admit(ctx, conflicting); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("shared ACL claim not held: %v", err)
	}
	if err := s.Claim(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := s.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("original claimed effect not kept as UNKNOWN: n=%d err=%v", n, err)
	}
	if _, err := s.Admit(ctx, conflicting); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN released shared ACL: %v", err)
	}
	independent := conflicting
	independent.Key = "s4b2-independent-key-0001"
	independent.ResourceKeys = []string{"acl:default/unrelated"}
	if _, err := s.Admit(ctx, independent); err != nil {
		t.Fatalf("independent unrelated resource was blocked: %v", err)
	}
	invalid := independent
	invalid.Key = "s4b2-invalid-key-0000001"
	invalid.ResourceKeys = []string{"host:unrestricted"}
	if _, err := s.Admit(ctx, invalid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("untyped host mutation claim accepted: %v", err)
	}
}

func TestS4B2NetworkClaimGuardRejectsUnboundedAndBadKeys(t *testing.T) {
	keys := []string{"acl:default/bw", "box:alpha", "box:beta"}
	a := Admission{ResourceKey: "box:alpha", ResourceKeys: keys}
	normalized, err := normalizeAdmissionResourceKeys(a)
	if err != nil || len(normalized) != 3 {
		t.Fatalf("dedup/canonical: %v %v", normalized, err)
	}
	for _, bad := range []string{"host:all", "acl:", "acl:space forbidden", "acl:\nunsafe", "listener:â"} {
		if validInternalClaimKey(bad) {
			t.Fatalf("invalid claim key accepted: %q", bad)
		}
	}
	a.ResourceKeys = make([]string, 9)
	for i := range a.ResourceKeys {
		a.ResourceKeys[i] = "box:alpha"
	}
	if _, err := normalizeAdmissionResourceKeys(a); !errors.Is(err, ErrInvalid) {
		t.Fatal("unbounded resource fanout was accepted")
	}
}
