package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type lifecyclePinWitness struct {
	witness
	err error
}

func (w lifecyclePinWitness) VerifyPinnedDeleteNIC(_ context.Context, name, uuid, bridge, acl, mac, v4, v6 string) error {
	if name != "owned" || uuid != bu || bridge != "bwbr0" || acl != "bw-peer" || mac != "02:16:3e:11:22:33" || v4 != "10.24.18.11" || v6 != "fd43:18::11" {
		return errors.New("wrong pin passed to observer")
	}
	return w.err
}

func TestN3LifecycleNativePinMatrix(t *testing.T) {
	for _, tc := range []struct {
		name, ledger, reason string
		err                  error
	}{
		{"match", "PINNED", "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED", nil},
		{"missing", "MISSING", "LIFECYCLE_STATIC_LEDGER_MISSING", nil},
		{"conflict", "CONFLICT", "LIFECYCLE_STATIC_LEDGER_MISMATCH", nil},
		{"native drift", "PINNED", "NATIVE_PINNED_NIC_DRIFT", native.ErrUnsafeNetwork},
		{"native unavailable", "PINNED", "NATIVE_PINNED_NIC_UNVERIFIED", errors.New("read failure")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := original("box_start")
			target.LifecyclePinStatus = tc.ledger
			pin := state.VerifiedCreateBinding{Bridge: "bwbr0", ACL: "bw-peer", MAC: "02:16:3e:11:22:33", IPv4: "10.24.18.11", IPv6: "fd43:18::11"}
			target.LifecycleBinding = pin
			w := lifecyclePinWitness{witness: known(), err: tc.err}
			v, p := (Service{Native: w, Layer: "layer"}).observe(context.Background(), target, "")
			if p.Reason != tc.reason || v.ClaimProtection != "RETAINED" {
				t.Fatal(v, p)
			}
			if tc.name == "match" {
				if p.Kind != state.RecoverySettle || p.LifecycleBinding != pin {
					t.Fatal(p)
				}
			} else if p.Kind != state.RecoveryBlocked {
				t.Fatal(p)
			}
		})
	}
}
