package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"

	incus "github.com/lxc/incus/v7/client"
	"reflect"

	"github.com/vanishs/boxweave/internal/state"
)

// nativePeerConntrackTargetFromStore requires a complete durable owner,
// dual-stack NIC and lease snapshot. UUID arguments select only a pair from
// that snapshot: no caller-provided MAC, IP or arbitrary CIDR is trusted.
func nativePeerConntrackTargetFromStore(ctx context.Context, s *state.Store,
	project, bridge, acl, srcUUID, dstUUID string) (NativePeerBridgeAuditTarget, state.NetworkAuditLedger, error) {
	var zero NativePeerBridgeAuditTarget
	var empty state.NetworkAuditLedger
	if s == nil || srcUUID == "" || dstUUID == "" || srcUUID == dstUUID {
		return zero, empty, errors.New("missing protected pair ledger or distinct UUID scope")
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return zero, empty, err
	}
	source, target := false, false
	for _, b := range ledger.Boxes {
		if b.UUID == srcUUID {
			source = true
		}
		if b.UUID == dstUUID {
			target = true
		}
	}
	if !source || !target {
		return zero, empty, errors.New("pair UUID not in complete reserved native ledger")
	}
	return nativePeerTargetFromLedger(ledger), ledger, nil
}

// nativePeerConntrackPurgeFromStoreWithClient is the exact ledger + kernel
// bridge + root Incus + scoped conntrack implementation. The public root
// entrypoint establishes OS privilege and service-project ownership before
// calling it. The injected client/executor are for isolated native tests,
// never parameters on the MCP or local operator CLI.
func nativePeerConntrackPurgeFromStoreWithClient(ctx context.Context, s *state.Store,
	root incus.InstanceServer, run nativePeerConntrackRun,
	project, bridge, acl, srcUUID, dstUUID string) (NativePeerConntrackReport, error) {
	var zero NativePeerConntrackReport
	if s == nil || !s.IsNetworkAuditReadOnly() || root == nil || run == nil {
		return zero, errors.New("protected read-only ledger, root SDK and fixed conntrack executor required")
	}
	target, before, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl, srcUUID, dstUUID)
	if err != nil {
		return zero, err
	}
	// Native UsedBy alone cannot detect a host-created raw veth link. Keep
	// the separate kernel inventory fence before AND after exact invalidation.
	if err := nativePeerBridgeKernelNoLivePorts(ctx, target.Bridge); err != nil {
		return zero, err
	}
	report, err := nativePeerConntrackWithClient(ctx, root, target, srcUUID, dstUUID, run)
	if err != nil {
		return report, err
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, target.Bridge); err != nil {
		return report, fmt.Errorf("host bridge acquired a port during exact conntrack purge: %w", err)
	}
	after, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(before, after) {
		return report, fmt.Errorf("protected peer lease/identity ledger changed during scoped conntrack purge: %v", err)
	}
	return report, nil
}

// NativePeerConntrackPurgeFromStore is a host-root-only entrypoint for a
// complete protected SQLite-owned pair. Caller-supplied IP/MAC/CIDRs are
// never accepted. No general shell, MCP root socket or global conntrack -F.
func NativePeerConntrackPurgeFromStore(ctx context.Context, s *state.Store,
	project, bridge, acl, srcUUID, dstUUID string) (NativePeerConntrackReport, error) {
	var zero NativePeerConntrackReport
	if os.Geteuid() != 0 {
		return zero, errors.New("protected conntrack invalidation requires host-local root")
	}
	if !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("root conntrack invalidation requires validated read-only SQLite owner ledger")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || project != "user-"+u.Uid {
		return zero, errors.New("conntrack scope is not the confined service Project")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	return nativePeerConntrackPurgeFromStoreWithClient(ctx, s, root, nativePeerConntrackOS,
		project, bridge, acl, srcUUID, dstUUID)
}
