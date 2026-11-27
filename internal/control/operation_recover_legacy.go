package control

import (
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

func (s *Service) legacyRecoveryObserver(layer string) recovery.Service {
	return recovery.Service{Store: s.Store, Native: s.Native, Layer: layer, PeerWitness: recovery.ObserveCompletedPeer}
}

func observeLegacyPeerNavigation(result state.Receipt, matches bool, diagnostic *operationDiagnostic, navigation *state.RecoveryNavigation) {
	if result.Action == "box_peer_set" && result.State == "ADMITTED" && matches {
		diagnostic.StatusClass = "LOCAL_APPROVAL_REQUIRED"
		diagnostic.NextAction = state.NextAction{Kind: "LOCAL_REVIEW_REQUIRED"}
		navigation.NextAction = diagnostic.NextAction
		navigation.Disposition = "LEGACY_PEER_LOCAL_APPROVAL_REQUIRED"
	}
}
