package control

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// privateReadReply checks the durable token just before yielding any actor-
// private read. HTTP ingress authorization can become stale during a slow
// SDK/native read. Revocation never cancels or retries a native operation.
func privateReadReply[Out any](ctx context.Context, svc *Service, tokenID string, out Out) (*mcp.CallToolResult, Out, error) {
	if err := svc.Store.RequireActiveTokenID(ctx, tokenID); err != nil {
		var zero Out
		return nil, zero, err
	}
	return &mcp.CallToolResult{}, out, nil
}
