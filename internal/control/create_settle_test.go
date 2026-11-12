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
