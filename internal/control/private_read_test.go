package control

import (
	"context"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS2AllPrivateReadReturnTypesRejectRevokedCredential(t *testing.T) {
	st, _, admin, _ := fixture(t)
	ctx := context.Background()
	original, err := st.VerifyToken(ctx, admin)
	if err != nil { t.Fatal(err) }
	svc := &Service{Store: st}
	if _, _, err := privateReadReply(ctx, svc, original.ID, HostStatus{}); err != nil {
		t.Fatalf("active credential refused: %v", err)
	}
	if err := st.RevokeToken(ctx, original.ID); err != nil { t.Fatal(err) }
	for name, err := range map[string]error{
		"host_status": privateReadError(ctx, svc, original.ID, HostStatus{}),
		"box_list": privateReadError(ctx, svc, original.ID, listBoxesResult{}),
		"box_get": privateReadError(ctx, svc, original.ID, getBoxResult{}),
		"operation_get": privateReadError(ctx, svc, original.ID, operationGetResult{}),
		"operation_list": privateReadError(ctx, svc, original.ID, operationListResult{}),
		"operation_output_get": privateReadError(ctx, svc, original.ID, operationOutputResult{}),
	} {
		if !errors.Is(err, state.ErrRevoked) {
			t.Fatalf("%s returned actor-private data after local revocation: %v", name, err)
		}
	}
}

// A small typed wrapper keeps the test checking every public response type
// without creating fake MCP Tool results or native side effects.
func privateReadError[T any](ctx context.Context, svc *Service, token string, value T) error {
	_, _, err := privateReadReply(ctx, svc, token, value)
	return err
}
