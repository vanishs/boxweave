package recovery

import (
	"context"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

type planReadFixture struct {
	ObservationStore
	target        state.RecoveryTarget
	drift         bool
	failCheck     bool
	reads, checks int
}

func (f *planReadFixture) ReadRecoveryTarget(context.Context, string, string) (state.RecoveryTarget, error) {
	f.reads++
	t := f.target
	if f.drift && f.reads > 1 {
		t.UpdatedAt++
	}
	return t, nil
}
func (f *planReadFixture) CheckPlanRecoveryProof(context.Context, state.RecoveryTarget, state.RecoveryProof) error {
	f.checks++
	if f.failCheck {
		return state.ErrResourceLocked
	}
	return nil
}
func TestPR177D3PlanObservationUsesReadOnlyPredicates(t *testing.T) {
	ctx := context.Background()
	target := original("box_start")
	target.LifecyclePinStatus = "LEGACY_UNPINNED"
	for _, tc := range []struct {
		name                                     string
		drift, failCheck, badUUID, missingNative bool
		kind                                     string
		err                                      bool
	}{
		{"verified", false, false, false, false, state.RecoverySettle, false},
		{"drift", true, false, false, false, "", true},
		{"writer precondition", false, true, false, false, state.RecoveryBlocked, false},
		{"foreign uuid", false, false, true, false, state.RecoveryBlocked, false},
		{"missing native", false, false, false, true, state.RecoveryBlocked, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &planReadFixture{target: target, drift: tc.drift, failCheck: tc.failCheck}
			w := known()
			if tc.badUUID {
				w.box.UUID = "other"
			}
			if tc.missingNative {
				w.opErr = errors.New("unavailable")
			}
			s := Service{ReadStore: f, Native: w, Layer: target.Layer}
			out, e := s.InspectPlan(ctx, target)
			if tc.err {
				if !errors.Is(e, state.ErrConflict) {
					t.Fatal(out, e)
				}
				return
			}
			if e != nil || out.Kind != tc.kind {
				t.Fatal(out, e)
			}
			if (out.ProofSHA256 != "") != (tc.kind == state.RecoverySettle) {
				t.Fatal(out)
			}
			if tc.kind == state.RecoverySettle && f.checks != 1 {
				t.Fatal("missing writer preflight")
			}
			if _, e = s.Reconcile(ctx, state.RecoveryRequest{Layer: target.Layer, OriginalID: target.ID}); !errors.Is(e, state.ErrInvalid) {
				t.Fatal("observation adapter entered writer")
			}
		})
	}
	t.Log("D3_SHARED_OBSERVER_STALE_AND_PRECONDITION_PASS")
}
