package control

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

func (s *Service) admitScopedCreate(ctx context.Context, req *mcp.CallToolRequest, in createBoxInput, start, internet bool) (*mcp.CallToolResult, state.Receipt, error) {
	options, err := json.Marshal(in.InstanceOptions)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	in.InstanceOptions = in.InstanceOptions.ForCreate()
	intent := state.ScopedCreateIntent{Schema: 2, Project: s.Native.Project, Bridge: "incusbr-" + strings.TrimPrefix(s.Native.Project, "user-"),
		Name: in.Name, Start: start, Internet: internet, BootAutostart: *in.BootAutostart, CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB, Image: in.Image, Nesting: in.Nesting, Options: options}
	r, err := s.admitNative(ctx, req, "box_create", in.IdempotencyKey, "", "", "box:"+in.Name, in.WorkRef, in.AgentRef, intent)
	if err != nil {
		return nil, state.Receipt{}, err
	}
	var nativeID string
	var reserved state.VerifiedCreateBinding
	effect := func(ctx context.Context) (string, error) {
		id, e := s.Native.CreateScopedQuarantined(ctx, native.InstanceAction{OperationID: r.ID, Name: in.Name, CPU: in.CPU, MemoryMiB: in.MemoryMiB, DiskGiB: in.DiskGiB, Image: in.Image, Start: start, Internet: internet, Nesting: in.Nesting, Options: in.InstanceOptions},
			func(b native.VerifiedStaticPeerBinding) (bool, error) {
				ok, e := s.Store.StageCreateBinding(ctx, r.ID, intent.Project, intent.Name, state.VerifiedCreateBinding(b))
				if ok && e == nil {
					reserved = state.VerifiedCreateBinding(b)
				}
				return ok, e
			}, s.Store.ReadScopedCreateAddressHolds)
		nativeID = id
		return id, preflightNative(e)
	}
	verify := func(ctx context.Context) error {
		unknown := state.VerificationUncertain{Class: "SCOPED_CREATE_NATIVE_OR_ROOT_UNVERIFIED"}
		op, e := s.Native.ObserveUntilFinal(ctx, nativeID)
		if api.StatusErrorCheck(e, http.StatusNotFound) {
			if e = s.Native.OriginalOperationInactive(ctx, nativeID); e != nil {
				return unknown
			}
		} else if e != nil || op == nil || op.StatusCode != api.Success {
			return unknown
		}
		receipt, e := install.NativeScopedLifecycleCall(ctx, r.ID, "CREATE")
		if e != nil {
			return unknown
		}
		acl, e := peerscope.ACLName(intent.Project, intent.Bridge, receipt.UUID)
		if e != nil {
			return unknown
		}
		reserved.ACL = acl
		if receipt.Project != intent.Project || receipt.Name != in.Name || receipt.NativeID != nativeID || receipt.Binding != reserved || receipt.Start != start || receipt.Internet != internet || receipt.BootAutostart != intent.BootAutostart {
			return unknown
		}
		if start {
			id, e := s.Native.SetPower(ctx, in.Name, receipt.UUID, "start")
			if e != nil {
				return unknown
			}
			op, e := s.Native.ObserveUntilFinal(ctx, id)
			if api.StatusErrorCheck(e, http.StatusNotFound) {
				if e = s.Native.OriginalOperationInactive(ctx, id); e != nil {
					return unknown
				}
			} else if e != nil || op == nil || op.StatusCode != api.Success {
				return unknown
			}
			deadline := time.Now().Add(10 * time.Second)
			verified := false
			for time.Now().Before(deadline) && ctx.Err() == nil {
				b, e := s.Native.ObserveOriginalCreateStaticPeerBinding(ctx, in.Name, receipt.UUID, r.ID)
				if e == nil && state.VerifiedCreateBinding(b) == receipt.Binding {
					verified = true
					break
				}
				select {
				case <-ctx.Done():
					return unknown
				case <-time.After(350 * time.Millisecond):
				}
			}
			if !verified {
				return unknown
			}
		}
		for i := 0; i < 2; i++ {
			got, e := s.Native.ObserveCreatedByOperation(ctx, in.Name, r.ID)
			if e != nil || got.UUID != receipt.UUID || got.Internet != internet || !exactCreateNativeShape(got, in) || got.NativeOptions["boot.autostart"] != strconv.FormatBool(intent.BootAutostart) || start && got.NativeState != "Running" && got.NativeState != "Ready" || !start && got.NativeState != "Stopped" {
				return unknown
			}
			b := receipt.Binding
			if e = s.Native.VerifyPinnedDeleteNIC(ctx, in.Name, receipt.UUID, b.Bridge, b.ACL, b.MAC, b.IPv4, b.IPv6); e != nil {
				return unknown
			}
		}
		if e = s.Store.FinalizeVerifiedScopedCreate(ctx, r.ID, intent.Project, in.Name, receipt.UUID, receipt.Binding); e != nil {
			return unknown
		}
		return state.ErrVerifiedCreateCommitted
	}
	r, err = s.submitNative(r, effect, verify)
	return &mcp.CallToolResult{}, r, err
}
