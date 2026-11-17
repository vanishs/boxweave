package install

import (
	"context"
	"errors"
	"fmt"
	"os"
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

// NativePeerConntrackPurgeFromStore is a host-root-only entrypoint for a
// *protected* SQLite-owned pair. Unlike the raw internal SDK test fixture,
// callers cannot assemble an ACL consumer/lease/MAC inventory themselves.
// It makes no MCP tool, no root socket delegation and no global conntrack
// flush. The native kernel bridge port, stop and dual-stack DROP guards are
// still mandatory within NativePeerConntrackPurge.
func NativePeerConntrackPurgeFromStore(ctx context.Context, s *state.Store,
	project, bridge, acl, srcUUID, dstUUID string) (NativePeerConntrackReport, error) {
	var zero NativePeerConntrackReport
	if os.Geteuid() != 0 {
		return zero, errors.New("protected conntrack invalidation requires host-local root")
	}
	target, before, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl, srcUUID, dstUUID)
	if err != nil {
		return zero, err
	}
	report, err := NativePeerConntrackPurge(ctx, target, srcUUID, dstUUID)
	if err != nil {
		return report, err
	}
	after, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(before, after) {
		return report, fmt.Errorf("protected peer lease/identity ledger changed during scoped conntrack purge: %v", err)
	}
	return report, nil
}
