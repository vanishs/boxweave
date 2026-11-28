package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/native"
)

type createRecoveryObserver interface {
	ObserveOriginalCreateStaticPeerBinding(context.Context, string, string, string) (native.VerifiedStaticPeerBinding, error)
	OriginalOperationInactive(context.Context, string) error
}
