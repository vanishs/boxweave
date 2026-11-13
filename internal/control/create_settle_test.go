package control

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

func TestCreatePostreadBoundedAndNeverReDispatches(t *testing.T) {
	ctx := context.Background()
	base := native.BoxState{Project: "user-1002", Name: "test-owned", UUID: "native-uuid", NativeState: "Stopped", Internet: false}
	t.Run("stable_stopped_residue", func(t *testing.T) {
		calls := 0
		got, err := settleOriginalCreate(ctx, 45*time.Millisecond, func(context.Context) (native.BoxState, error) {
			calls++
			return base, nil
		}, true, false)
		if err != nil || got.UUID != base.UUID || got.NativeState != "Stopped" || calls < 2 {
			t.Fatalf("stable failed-start original not proven: %+v / %v / calls=%d", got, err, calls)
		}
	})
	t.Run("transient_postread_then_running", func(t *testing.T) {
		calls := 0
		got, err := settleOriginalCreate(ctx, time.Second, func(context.Context) (native.BoxState, error) {
			calls++
			if calls == 1 {
				return native.BoxState{}, errors.New("native lookup not yet published")
			}
			done := base
			done.NativeState = "Running"
			return done, nil
		}, true, false)
		if err != nil || got.NativeState != "Running" || calls != 2 {
			t.Fatalf("propagation not recovered: %+v / %v / calls=%d", got, err, calls)
		}
	})
	t.Run("same_name_foreign_marker_denied", func(t *testing.T) {
		calls := 0
		_, err := settleOriginalCreate(ctx, time.Second, func(context.Context) (native.BoxState, error) {
			calls++
			return native.BoxState{}, native.ErrNotOwned
		}, true, false)
		var uncertain state.VerificationUncertain
		if !errors.As(err, &uncertain) || uncertain.Class != "NATIVE_CREATE_IDENTITY_OR_NIC_UNVERIFIED" || calls != 1 {
			t.Fatalf("foreign marker was retried or adopted: %v / calls=%d", err, calls)
		}
	})
	t.Run("stopped_guest_uuid_changed_during_confirmation", func(t *testing.T) {
		calls := 0
		_, err := settleOriginalCreate(ctx, 45*time.Millisecond, func(context.Context) (native.BoxState, error) {
			calls++
			out := base
			if calls > 1 {
				out.UUID = "different-native-uuid"
			}
			return out, nil
		}, true, false)
		var uncertain state.VerificationUncertain
		if !errors.As(err, &uncertain) || uncertain.Class != "NATIVE_CREATE_POSTREAD_UNVERIFIED" {
			t.Fatalf("name reuse or changed UUID accepted: %v", err)
		}
	})
	t.Run("unexpected_external_egress_denied", func(t *testing.T) {
		_, err := settleOriginalCreate(ctx, time.Second, func(context.Context) (native.BoxState, error) {
			out := base
			out.Internet = true
			return out, nil
		}, true, false)
		var uncertain state.VerificationUncertain
		if !errors.As(err, &uncertain) || uncertain.Class != "NATIVE_CREATE_EGRESS_UNVERIFIED" {
			t.Fatalf("unexpected egress was accepted: %v", err)
		}
	})
	t.Run("permanent_transient_no_success_claim", func(t *testing.T) {
		_, err := settleOriginalCreate(ctx, 45*time.Millisecond, func(context.Context) (native.BoxState, error) {
			return native.BoxState{NativeState: "Starting"}, nil
		}, true, false)
		var uncertain state.VerificationUncertain
		if !errors.As(err, &uncertain) || uncertain.Class != "NATIVE_CREATE_POSTREAD_UNVERIFIED" {
			t.Fatalf("indeterminate state incorrectly treated as terminal: %v", err)
		}
	})
}

// Native Create can return a terminal failure while Incus still reports
// the requested guest momentarily Running before it settles to Stopped.
// The original must not turn UNKNOWN from that transient observation.
func TestTerminalFailedCreateRequiresStableStoppedReadback(t *testing.T) {
	base := native.BoxState{Project: "user-1002", Name: "failed-start", UUID: "native-verified-uuid",
		NativeState: "Running", Internet: false}
	t.Run("transient_running_then_stable_stopped", func(t *testing.T) {
		calls := 0
		got, err := settleOriginalCreateWithNativeFailure(context.Background(), 2*time.Second,
			func(context.Context) (native.BoxState, error) {
				calls++
				out := base
				if calls >= 2 {
					out.NativeState = "Stopped"
				}
				return out, nil
			}, true, false, true)
		if err != nil || got.NativeState != "Stopped" || got.UUID != base.UUID || calls < 3 {
			t.Fatalf("terminal failed-start not confirmed: %+v / %v / calls=%d", got, err, calls)
		}
	})
	t.Run("running_never_stopped_stays_unknown", func(t *testing.T) {
		_, err := settleOriginalCreateWithNativeFailure(context.Background(), 60*time.Millisecond,
			func(context.Context) (native.BoxState, error) { return base, nil }, true, false, true)
		var unknown state.VerificationUncertain
		if !errors.As(err, &unknown) || unknown.Class != "NATIVE_CREATE_POSTREAD_UNVERIFIED" {
			t.Fatalf("failed native op with running guest was falsely accepted: %v", err)
		}
	})
	t.Run("reused_name_uuid_between_stopped_reads_denied", func(t *testing.T) {
		calls := 0
		_, err := settleOriginalCreateWithNativeFailure(context.Background(), 750*time.Millisecond,
			func(context.Context) (native.BoxState, error) {
				calls++
				out := base
				out.NativeState = "Stopped"
				if calls > 1 {
					out.UUID = "other-uuid"
				}
				return out, nil
			}, true, false, true)
		var unknown state.VerificationUncertain
		if !errors.As(err, &unknown) || unknown.Class != "NATIVE_CREATE_IDENTITY_OR_NIC_UNVERIFIED" {
			t.Fatalf("changed uuid in terminal failure was accepted: %v", err)
		}
	})
}
