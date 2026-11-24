package recovery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

// Fake authenticated Root witness: checks orchestration only, not kernel.
func TestPR177D0ExtensionScopedWitnessDispatchBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, phase  string
		modify       func(*state.ScopedPeerRootReceipt)
		fail         bool
		calls        int
		kind, reason string
	}{
		{"admitted", "ADMITTED", nil, false, 0, state.RecoveryBlocked, "ORIGINAL_NOT_UNKNOWN"},
		{"dispatching", "DISPATCHING", nil, false, 0, state.RecoveryBlocked, "ORIGINAL_NOT_UNKNOWN"},
		{"completed", "SUCCEEDED", nil, false, 0, state.RecoveryNotNeeded, "ORIGINAL_TERMINAL"},
		{"unavailable", "UNKNOWN", nil, true, 1, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED"},
		{"not recovery", "UNKNOWN", func(r *state.ScopedPeerRootReceipt) { r.RecoveryOnly = false }, false, 1, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED"},
		{"foreign original", "UNKNOWN", func(r *state.ScopedPeerRootReceipt) { r.OriginalID = "other" }, false, 1, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED"},
		{"foreign project", "UNKNOWN", func(r *state.ScopedPeerRootReceipt) { r.Project = "other" }, false, 1, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED"},
		{"partial", "UNKNOWN", func(r *state.ScopedPeerRootReceipt) { r.VerifiedSteps = 3 }, false, 1, state.RecoveryBlocked, "SCOPED_PEER_ROOT_PROOF_REQUIRED"},
		{"verified", "UNKNOWN", nil, false, 1, state.RecoverySettle, "SCOPED_PEER_COMPLETED_NO_REPLAY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := original("box_peer_set")
			target.State = tc.phase
			target.ScopedPeer = true
			target.NativeID = ""
			calls := 0
			s := Service{Layer: target.Layer, ScopedPeerWitness: func(_ context.Context, id string) (state.ScopedPeerRootReceipt, error) {
				calls++
				if id != target.ID {
					t.Fatal("wrong original")
				}
				r := state.ScopedPeerRootReceipt{OriginalID: id, Project: target.Layer, Bridge: "bwbr0", Status: "COMPLETE_VERIFIED", ProofSHA256: strings.Repeat("a", 64), PlanSHA256: strings.Repeat("b", 64), FinalSHA256: strings.Repeat("c", 64), VerifiedSteps: 4, RecoveryOnly: true}
				if tc.modify != nil {
					tc.modify(&r)
				}
				if tc.fail {
					return r, errors.New("missing proof")
				}
				return r, nil
			}}
			v, p := s.observe(context.Background(), target, "")
			if calls != tc.calls || p.Kind != tc.kind || p.Reason != tc.reason {
				t.Fatal(calls, v, p)
			}
			if target.State == "UNKNOWN" && v.ClaimProtection != "RETAINED" {
				t.Fatal("observation released claims")
			}
			if p.Kind == state.RecoverySettle && p.Fingerprint == "" {
				t.Fatal("missing proof fingerprint")
			}
		})
	}
}
