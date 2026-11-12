package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestUnknownCreateReviewedNativeIdentityBindAndNoReplay(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	_, token, err := store.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	input := Admission{
		Layer: "isolated-project", ActorID: token.ID, TokenID: token.ID,
		Key: "original-unknown-create-recovery-001", RequestID: "original",
		Action: "box_create", ResourceKey: "box:unknown-a",
		SafeIntent: []byte(`{"name":"unknown-a","start":true}`),
	}
	original, err := store.Admit(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.Unknown(ctx, original.ID, "DISPATCHING", "NATIVE_OUTCOME_AMBIGUOUS"); err != nil {
		t.Fatal(err)
	}
	snap, err := store.CreateRecoveryInspect(ctx, original.ID)
	if err != nil || snap.Name != "unknown-a" || snap.Project != "isolated-project" ||
		snap.NativeOperationID != "" {
		t.Fatalf("original create snapshot: %+v / %v", snap, err)
	}
	if err := store.BindUnknownCreatedBox(ctx, snap, "operator-a", "not-a-challenge",
		"native-uuid-001", "native-observation-001", "safe manual binding checked", true); !errors.Is(err, ErrInvalid) {
		t.Fatalf("missing challenge allowed a bind: %v", err)
	}
	challenge, err := store.IssueCreateBindingChallenge(ctx, snap, "operator-a", "native-observation-001")
	if err != nil {
		t.Fatal(err)
	}
	negative := []struct {
		name, reviewer, chal, obs, reason string
		acknowledged                      bool
	}{
		{"wrong_reviewer", "operator-b", challenge, "native-observation-001", "independently checked binding", true},
		{"wrong_observation", "operator-a", challenge, "another-etag-observation", "independently checked binding", true},
		{"no_ack", "operator-a", challenge, "native-observation-001", "independently checked binding", false},
		{"short_reason", "operator-a", challenge, "native-observation-001", "short", true},
		{"wrong_challenge", "operator-a", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "native-observation-001", "independently checked binding", true},
	}
	for _, tc := range negative {
		t.Run(tc.name, func(t *testing.T) {
			if err := store.BindUnknownCreatedBox(ctx, snap, tc.reviewer, tc.chal, "native-uuid-001", tc.obs, tc.reason, tc.acknowledged); err == nil {
				t.Fatal("unsupported native binding accepted")
			}
			if _, err := store.OwnedBox(ctx, "isolated-project", "native-uuid-001"); !errors.Is(err, ErrNotFound) {
				t.Fatalf("negative test bound ownership: %v", err)
			}
		})
	}
	if err := store.BindUnknownCreatedBox(ctx, snap, "operator-a", challenge,
		"native-uuid-001", "native-observation-001", "native config identity independently verified", true); err != nil {
		t.Fatal(err)
	}
	if err := store.BindUnknownCreatedBox(ctx, snap, "operator-a", challenge,
		"native-uuid-002", "native-observation-001", "same challenge replay attempt", true); err == nil {
		t.Fatal("stale create binding replay accepted")
	}
	owned, err := store.OwnedBox(ctx, "isolated-project", "native-uuid-001")
	if err != nil || owned.Name != "unknown-a" {
		t.Fatalf("not registered: %+v / %v", owned, err)
	}
	receipt, err := store.Replay(ctx, input)
	if err != nil || receipt.State != "UNKNOWN" || receipt.NativeOperationID != "" ||
		receipt.BoxUUID != "native-uuid-001" || !receipt.Duplicate ||
		receipt.ID != original.ID {
		t.Fatalf("original result falsely resolved: %+v / %v", receipt, err)
	}
	input.Key = "second-unknown-create-recovery-002"
	if _, err := store.Admit(ctx, input); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("locked original became repeatable: %v", err)
	}
	boundReview, err := store.RecoveryInspect(ctx, original.ID)
	if err != nil || boundReview.BoxUUID != "native-uuid-001" || boundReview.NativeOperationID != "" {
		t.Fatalf("existing safe recovery flow did not become available: %+v / %v", boundReview, err)
	}
}

func TestUnknownCreateMalformedIntentFailsClosed(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	_, token, err := store.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	r, err := store.Admit(ctx, Admission{
		Layer: "project", ActorID: token.ID, TokenID: token.ID,
		Key: "invalid-intent-recovery-unknown-001", RequestID: "original",
		Action: "box_create", ResourceKey: "box:good",
		SafeIntent: []byte(`{"name":"other"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.Unknown(ctx, r.ID, "DISPATCHING", "LOST_NATIVE_ACK"); err != nil {
		t.Fatal(err)
	}
	if _, err = store.CreateRecoveryInspect(ctx, r.ID); !errors.Is(err, ErrState) {
		t.Fatalf("mismatched original native name accepted: %v", err)
	}
}
