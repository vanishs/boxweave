package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// BH-018: the returned native ID, not an error label, determines whether
// a failure may safely be called pre-dispatch. These are source-level tests,
// not evidence that a particular Incus guest process did or did not start.
func TestBH018ExecFailureDispatchBoundaryPreservesOriginal(t *testing.T) {
	for _, tc := range []struct {
		name       string
		nativeID   string
		effectErr  error
		verifyErr  error
		wantState  string
		wantClass  string
		wantVerify bool
	}{
		{
			name: "proved-local-predispatch-startup-refusal",
			effectErr: PreDispatchError{Class: "EXEC_STARTUP_PRE_DISPATCH_REFUSED"},
			wantState: "FAILED", wantClass: "EXEC_STARTUP_PRE_DISPATCH_REFUSED",
		},
		{
			name: "mislabelled-predispatch-with-native-receipt",
			nativeID: "incus-original-exec-001",
			effectErr: PreDispatchError{Class: "EXEC_STARTUP_PRE_DISPATCH_REFUSED"},
			wantState: "UNKNOWN", wantClass: "NATIVE_RECEIPT_WITH_EFFECT_ERROR",
		},
		{
			name: "native-error-after-returned-receipt",
			nativeID: "incus-original-exec-002",
			effectErr: errors.New("native reported an error after accepting the request"),
			wantState: "UNKNOWN", wantClass: "NATIVE_RECEIPT_WITH_EFFECT_ERROR",
		},
		{
			name: "process-start-outcome-unverified-after-dispatch",
			nativeID: "incus-original-exec-003",
			verifyErr: errors.New("final Incus operation cannot prove guest startup"),
			wantState: "UNKNOWN", wantClass: "POSTCONDITION_UNVERIFIED",
			wantVerify: true,
		},
		{
			name: "missing-native-ack-is-ambiguous",
			effectErr: errors.New("transport timeout may have followed native dispatch"),
			wantState: "UNKNOWN", wantClass: "NATIVE_OUTCOME_AMBIGUOUS",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			_, token, err := store.IssueToken(ctx, "admin", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			input := Admission{
				Layer: "bh018-isolated", ActorID: token.ID, TokenID: token.ID,
				Key: "bh018-original-key", RequestID: "first",
				Action: "box_exec", ResourceKey: "box:bh018-owned",
				BoxUUID: "bh018-exact-uuid", Incarnation: "bh018-exact-uuid",
				SafeIntent: []byte(`{"box_id":"bh018-exact-uuid","command_digest":"test"}`),
			}
			original, err := store.Admit(ctx, input)
			if err != nil {
				t.Fatal(err)
			}
			effectCalls, verifyCalls := 0, 0
			Worker{Store: store, Budget: time.Second}.Execute(original.ID,
				func(context.Context) (string, error) {
					effectCalls++
					return tc.nativeID, tc.effectErr
				},
				func(context.Context) error {
					verifyCalls++
					return tc.verifyErr
				})
			got, err := store.Get(ctx, original.ID)
			if err != nil || got.State != tc.wantState || got.ErrorClass != tc.wantClass ||
				got.NativeOperationID != tc.nativeID || effectCalls != 1 {
				t.Fatalf("receipt or dispatch classification changed: %+v effects=%d err=%v", got, effectCalls, err)
			}
			if (verifyCalls != 0) != (!tc.wantVerify) {
				t.Fatalf("unexpected postread execution: %d", verifyCalls)
			}
			replay, err := store.Replay(ctx, input)
			if err != nil || !replay.Duplicate || replay.ID != original.ID ||
				replay.NativeOperationID != tc.nativeID || replay.State != tc.wantState {
				t.Fatalf("original idempotency receipt was not preserved: %+v %v", replay, err)
			}
			if tc.wantState == "UNKNOWN" {
				input.Key = "bh018-second-distinct-key"
				if _, err := store.Admit(ctx, input); !errors.Is(err, ErrResourceLocked) {
					t.Fatalf("UNKNOWN effect claim was unlocked: %v", err)
				}
			}
		})
	}
}
