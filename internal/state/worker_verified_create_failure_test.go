package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateNativeFailureRetainedOwnedStoppedBoxRequiresExplicitDelete(t *testing.T) {
	ctx := context.Background()
	store := newStore(t)
	_, token, err := store.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	original := Admission{
		Layer: "isolated-project", ActorID: token.ID, TokenID: token.ID,
		Key: "failed-create-retained-original-001", RequestID: "first",
		Action: "box_create", ResourceKey: "box:failed-guest",
		SafeIntent: []byte(`{"name":"failed-guest"}`),
	}
	r, err := store.Admit(ctx, original)
	if err != nil {
		t.Fatal(err)
	}
	Worker{Store: store}.Execute(r.ID,
		func(context.Context) (string, error) { return "native-terminal-failure", nil },
		func(context.Context) error {
			if err := store.RegisterCreatedBox(ctx, r.ID, "isolated-project", "failed-guest", "real-native-uuid"); err != nil {
				return err
			}
			return VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_OWNED_STOPPED"}
		})
	got, err := store.Get(ctx, r.ID)
	if err != nil || got.State != "FAILED" || got.BoxUUID != "real-native-uuid" ||
		got.NativeOperationID != "native-terminal-failure" ||
		got.ErrorClass != "NATIVE_CREATE_FAILED_OWNED_STOPPED" {
		t.Fatalf("failed create lost exact native receipt or UUID: %+v / %v", got, err)
	}
	original.Key = "failed-create-retained-other-key-002"
	if _, err = store.Admit(ctx, original); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("retained owned UUID did not lock its original name: %v", err)
	}
	deleteIntent := original
	deleteIntent.Action = "box_delete"
	deleteIntent.BoxUUID = "real-native-uuid"
	deleteIntent.Incarnation = "real-native-uuid"
	deleteIntent.Key = "explicit-new-delete-of-owned-guest"
	if _, err = store.Admit(ctx, deleteIntent); err != nil {
		t.Fatalf("explicit new delete wrongly blocked: %v", err)
	}
	unchanged, err := store.Replay(ctx, Admission{
		Layer: original.Layer, ActorID: token.ID, TokenID: token.ID,
		Key: "failed-create-retained-original-001", RequestID: "first",
		Action: "box_create", ResourceKey: "box:failed-guest", SafeIntent: original.SafeIntent,
	})
	if err != nil || !unchanged.Duplicate || unchanged.ID != r.ID || unchanged.State != "FAILED" {
		t.Fatalf("original Create was not immutable: %+v / %v", unchanged, err)
	}
}

func TestCreateVerifiedTerminalFailureVersusAmbiguousOutcome(t *testing.T) {
	for _, tc := range []struct {
		name      string
		verified  error
		wantState string
		wantClass string
	}{
		{"confirmed_no_instance", VerifiedEffectFailure{Class: "NATIVE_CREATE_FAILED_NO_INSTANCE"}, "FAILED", "NATIVE_CREATE_FAILED_NO_INSTANCE"},
		{"unknown_residue", errors.New("native failed but a guest remains"), "UNKNOWN", "POSTCONDITION_UNVERIFIED"},
		{"typed_uncertain", VerificationUncertain{Class: "NATIVE_CREATE_POSTREAD_UNVERIFIED"}, "UNKNOWN", "NATIVE_CREATE_POSTREAD_UNVERIFIED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			_, token, err := store.IssueToken(ctx, "admin", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			input := Admission{
				Layer: "test-project", ActorID: token.ID, TokenID: token.ID,
				Key: "verified-failure-original-create-001", RequestID: "first",
				Action: "box_create", ResourceKey: "box:guest",
				SafeIntent: []byte(`{"name":"guest"}`),
			}
			original, err := store.Admit(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			dispatches := 0
			Worker{Store: store, Budget: time.Second}.Execute(original.ID,
				func(context.Context) (string, error) {
					dispatches++
					return "native-op-failed-0001", nil
				},
				func(context.Context) error { return tc.verified })
			got, err := store.Get(ctx, original.ID)
			if err != nil || got.State != tc.wantState || got.ErrorClass != tc.wantClass ||
				got.NativeOperationID != "native-op-failed-0001" || dispatches != 1 {
				t.Fatalf("wrong verified outcome %+v, dispatch=%d, err=%v", got, dispatches, err)
			}
			again, err := store.Replay(ctx, input)
			if err != nil || !again.Duplicate || again.ID != original.ID || again.State != tc.wantState {
				t.Fatalf("original receipt not immutable: %+v, err=%v", again, err)
			}
			input.Key = "verified-failure-second-create-002"
			_, err = store.Admit(ctx, input)
			if tc.wantState == "UNKNOWN" && !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("UNKNOWN released original lock: %v", err)
			}
			if tc.wantState == "FAILED" && err != nil {
				t.Fatalf("proved empty name still locked: %v", err)
			}
		})
	}
}
