package control

import (
	"context"
	"errors"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type forwardCreateInput struct {
	BoxID                   string `json:"box_id"`
	GuestPort               int    `json:"guest_port"`
	Protocol                string `json:"protocol"`
	HostPort                int    `json:"host_port"`
	HostBind                string `json:"host_bind,omitempty"`
	ConfirmExternalExposure bool   `json:"confirm_external_exposure,omitempty"`
	IdempotencyKey          string `json:"idempotency_key"`
	WorkRef                 string `json:"work_ref,omitempty"`
	AgentRef                string `json:"agent_ref,omitempty"`
}
type forwardDeleteInput struct {
	ForwardID      string `json:"forward_id"`
	IdempotencyKey string `json:"idempotency_key"`
	WorkRef        string `json:"work_ref,omitempty"`
	AgentRef       string `json:"agent_ref,omitempty"`
}
type forwardListInput struct {
	BoxID  string `json:"box_id,omitempty"`
	Cursor string `json:"cursor,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}
type forwardView struct {
	state.ForwardClaim
	NativeVerified bool `json:"native_verified"`
}
type forwardListResult struct {
	TrackingID string        `json:"tracking_id"`
	Forwards   []forwardView `json:"forwards"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

func forwardListenerKey(proto, bind string, port int) string {
	return "listener:" + proto + ":" + bind + ":" + strconv.Itoa(port)
}
func specForClaim(f state.ForwardClaim) native.ForwardSpec {
	return native.ForwardSpec{
		Device: f.NativeDevice, Protocol: f.Protocol, HostBind: f.HostBind,
		HostPort: f.HostPort, GuestPort: f.GuestPort, ExternalAcknowledged: true,
	}
}
func (s *Service) registerForwardTools(server *mcp.Server) {
	addStrictTool(server, &mcp.Tool{
		Name:        "port_forward_list",
		Description: "List only authorized same-Project instance-owned TCP/UDP proxy claims. Native_verified is a separate read-only exact-device observation, not a live packet or external LAN reachability proof; incomplete native checks are false. Limit 1..50 and opaque forward_id keyset cursor.",
		Annotations: readToolAnnotations(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in forwardListInput) (*mcp.CallToolResult, forwardListResult, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, forwardListResult{}, err
		}
		limit := in.Limit
		if limit == 0 {
			limit = 20
		}
		claims, next, err := s.Store.ListForwards(ctx, principal.Layer, in.BoxID, in.Cursor, limit)
		if err != nil {
			return nil, forwardListResult{}, err
		}
		views := make([]forwardView, 0, len(claims))
		for _, claim := range claims {
			v := forwardView{ForwardClaim: claim}
			box, e := s.Store.OwnedBox(ctx, principal.Layer, claim.BoxID)
			if e == nil && box.Incarnation == claim.Incarnation {
				v.NativeVerified = claim.State == "CONFIRMED" && s.Native.ObserveOwnedForward(ctx, box.Name, box.UUID, specForClaim(claim), true) == nil
			}
			views = append(views, v)
		}
		tracking, err := newTrackingID()
		if err != nil {
			return nil, forwardListResult{}, errors.New("TRACKING_UNAVAILABLE")
		}
		return privateReadReply(ctx, s, principal.TokenID, forwardListResult{TrackingID: tracking, Forwards: views, NextCursor: next})
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "port_forward_create",
		Description: "Admit one exact owned Box TCP/UDP instance-native proxy to guest 127.0.0.1:guest_port. Host bind defaults to 127.0.0.1 loopback. All binds require native restricted.devices.proxy=allow; inspect host_status.native.project_permissions and external_forward_policy before creating. External binds require confirm_external_exposure AND trusted host-local policy. A fresh installer-created service defaults to all; unset or empty service policy denies external binds. Explicit all policy permits assigned Host IPs and 0.0.0.0/:: wildcard listeners, potentially including public interfaces. Legacy exact allowlists remain supported. Never expose without the operator's authorization; foreign targets are forbidden. An async receipt is NOT proof of listener reachability; use operation_get, port_forward_list and separate packet checks. Never blindly retry UNKNOWN.",
		Annotations: mutationAnnotations(true),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in forwardCreateInput) (*mcp.CallToolResult, state.Receipt, error) {
		principal, err := s.requestPrincipal(req)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		if in.HostBind == "" {
			in.HostBind = "127.0.0.1"
		}
		spec := native.ForwardSpec{Device: "bwpf-0123456789abcdef", Protocol: in.Protocol,
			HostBind: in.HostBind, HostPort: in.HostPort, GuestPort: in.GuestPort,
			ExternalAcknowledged: in.ConfirmExternalExposure}
		if in.IdempotencyKey == "" || in.BoxID == "" || spec.Validate() != nil {
			return nil, state.Receipt{}, state.ErrInvalid
		}
		work, agent, err := boundedWorkLabels(req, in.WorkRef, in.AgentRef)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		intent := struct {
			BoxID     string `json:"box_id"`
			Protocol  string `json:"protocol"`
			HostBind  string `json:"host_bind"`
			HostPort  int    `json:"host_port"`
			GuestPort int    `json:"guest_port"`
			External  bool   `json:"confirm_external_exposure"`
		}{in.BoxID, in.Protocol, in.HostBind, in.HostPort, in.GuestPort, in.ConfirmExternalExposure}
		safe, err := safeJSON(intent)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		box, err := s.Store.ForwardReplayBox(ctx, principal.Layer, in.BoxID)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		newRequest := func() state.Admission {
			return state.Admission{
				Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID,
				Key: in.IdempotencyKey, Action: "port_forward_create", BoxUUID: box.UUID, Incarnation: box.Incarnation,
				ResourceKey:  "box:" + box.Name,
				ResourceKeys: []string{"box:" + box.UUID, forwardListenerKey(in.Protocol, in.HostBind, in.HostPort)},
				SafeIntent:   safe, WorkRef: work, AgentRef: agent,
				ForwardCreate: &state.ForwardCreateAdmission{Protocol: in.Protocol, HostBind: in.HostBind, HostPort: in.HostPort, GuestPort: in.GuestPort},
			}
		}
		a := newRequest()
		previous, err := s.Store.Replay(ctx, a)
		if err == nil {
			return &mcp.CallToolResult{}, previous, nil
		}
		if !errors.Is(err, state.ErrNotFound) {
			return nil, state.Receipt{}, err
		}
		if err = s.Native.AuthorizeForwardCreate(spec); err != nil {
			return nil, state.Receipt{}, err
		}
		if _, err = s.readOwned(ctx, req, box.UUID); err != nil {
			return nil, state.Receipt{}, err
		}
		reqID, err := newTrackingID()
		if err != nil {
			return nil, state.Receipt{}, errors.New("TRACKING_UNAVAILABLE")
		}
		a.RequestID = reqID
		r, err := s.Store.Admit(ctx, a)
		if err != nil {
			return nil, state.Receipt{}, err
		}
		spec.Device = "bwpf-" + r.ID[:16]
		var nativeID string
		effect := func(ctx context.Context) (string, error) {
			id, e := s.Native.UpdateOwnedForward(ctx, box.Name, box.UUID, spec, true)
			nativeID = id
			return id, preflightNative(e)
		}
		verify := func(ctx context.Context) error {
			if err := nativeDone(ctx, s.Native, nativeID); err != nil {
				return err
			}
			if err := s.Native.ObserveOwnedForward(ctx, box.Name, box.UUID, spec, true); err != nil {
				return err
			}
			if err := s.Store.FinalizeVerifiedForward(ctx, r.ID, true); err != nil {
				return state.VerificationUncertain{Class: "FORWARD_CREATE_FINAL_DURABILITY_UNVERIFIED"}
			}
			return state.ErrVerifiedForwardCommitted
		}
		r, err = s.submitNative(r, effect, verify)
		return &mcp.CallToolResult{}, r, err
	})

	addStrictTool(server, &mcp.Tool{
		Name:        "port_forward_delete",
		Description: "Remove only a previously verified same-Project instance-owned proxy by forward_id. A durable original Operation, exact native device check, ETag mutation and post-read are required. Native ambiguity remains UNKNOWN; no arbitrary network/foreign device deletion.",
		Annotations: mutationAnnotations(true),
	}, s.operateForwardDelete)
}
func (s *Service) operateForwardDelete(ctx context.Context, req *mcp.CallToolRequest, in forwardDeleteInput) (*mcp.CallToolResult, state.Receipt, error) {
	principal, err := s.requestPrincipal(req)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	if in.IdempotencyKey == "" || len(in.ForwardID) != 32 {
		return nil, state.Receipt{}, state.ErrInvalid
	}
	work, agent, err := boundedWorkLabels(req, in.WorkRef, in.AgentRef)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	claim, err := s.Store.GetForward(ctx, principal.Layer, in.ForwardID)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	box, err := s.Store.ForwardReplayBox(ctx, principal.Layer, claim.BoxID)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	if box.Incarnation != claim.Incarnation {
		return nil, state.Receipt{}, state.ErrResourceLocked
	}
	safe, err := safeJSON(struct {
		ForwardID string `json:"forward_id"`
	}{in.ForwardID})
	if err != nil {
		return nil, state.Receipt{}, err
	}
	a := state.Admission{Layer: principal.Layer, ActorID: principal.ActorID, TokenID: principal.TokenID,
		Key: in.IdempotencyKey, Action: "port_forward_delete", BoxUUID: box.UUID, Incarnation: box.Incarnation,
		ResourceKey:  "box:" + box.Name,
		ResourceKeys: []string{"box:" + box.UUID, forwardListenerKey(claim.Protocol, claim.HostBind, claim.HostPort)},
		SafeIntent:   safe, WorkRef: work, AgentRef: agent, ForwardDeleteID: in.ForwardID}
	previous, err := s.Store.Replay(ctx, a)
	if err == nil {
		return &mcp.CallToolResult{}, previous, nil
	}
	if !errors.Is(err, state.ErrNotFound) {
		return nil, state.Receipt{}, err
	}
	if claim.State != "CONFIRMED" {
		return nil, state.Receipt{}, state.ErrResourceLocked
	}
	if _, err = s.readOwned(ctx, req, box.UUID); err != nil {
		return nil, state.Receipt{}, err
	}
	spec := specForClaim(claim)
	if err = s.Native.ObserveOwnedForward(ctx, box.Name, box.UUID, spec, true); err != nil {
		return nil, state.Receipt{}, err
	}
	reqID, err := newTrackingID()
	if err != nil {
		return nil, state.Receipt{}, errors.New("TRACKING_UNAVAILABLE")
	}
	a.RequestID = reqID
	r, err := s.Store.Admit(ctx, a)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	var nativeID string
	effect := func(ctx context.Context) (string, error) {
		id, e := s.Native.UpdateOwnedForward(ctx, box.Name, box.UUID, spec, false)
		nativeID = id
		return id, preflightNative(e)
	}
	verify := func(ctx context.Context) error {
		if err := nativeDone(ctx, s.Native, nativeID); err != nil {
			return err
		}
		if err := s.Native.ObserveOwnedForward(ctx, box.Name, box.UUID, spec, false); err != nil {
			return err
		}
		if err := s.Store.FinalizeVerifiedForward(ctx, r.ID, false); err != nil {
			return state.VerificationUncertain{Class: "FORWARD_DELETE_FINAL_DURABILITY_UNVERIFIED"}
		}
		return state.ErrVerifiedForwardCommitted
	}
	r, err = s.submitNative(r, effect, verify)
	return &mcp.CallToolResult{}, r, err
}
