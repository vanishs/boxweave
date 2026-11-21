package control

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
)

func (s *Service) readDetailedBox(ctx context.Context, req *mcp.CallToolRequest, principal RequestPrincipal, uuid string) (*mcp.CallToolResult, getBoxResult, error) {
	box, err := s.readOwnedLedger(ctx, req, uuid)
	if err != nil {
		return nil, getBoxResult{}, err
	}
	if principal.Layer != s.Native.Project || box.Incarnation == "" {
		return nil, getBoxResult{}, native.ErrNotOwned
	}
	full, err := s.Native.GetExactBoxFull(ctx, box.Name, box.UUID)
	if err != nil {
		return nil, getBoxResult{}, err
	}
	current, err := s.readOwnedLedger(ctx, req, uuid)
	if err != nil {
		return nil, getBoxResult{}, err
	}
	if current.UUID != box.UUID || current.Name != box.Name || current.Incarnation != box.Incarnation {
		return nil, getBoxResult{}, native.ErrNotOwned
	}
	tracking, err := newTrackingID()
	if err != nil {
		return nil, getBoxResult{}, err
	}
	metadata, err := nativeFullJSON(full.InstanceFull)
	if err != nil {
		return nil, getBoxResult{}, err
	}
	// privateReadReply rechecks the actual token after native observation.
	return privateReadReply(ctx, s, principal.TokenID, getBoxResult{
		TrackingID: tracking, Box: full.Box, InstanceFull: metadata, ObservedAt: &full.ObservedAt, Attention: s.readAttention(ctx, principal, box.UUID),
	})
}

// Native Incus structs include legitimate null maps/slices and nil State.
// Keep the SDK's native JSON representation instead of generating a stricter
// incompatible MCP output schema or synthesizing non-null values.
func nativeFullJSON(full *api.InstanceFull) (map[string]any, error) {
	raw, err := json.Marshal(full)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
