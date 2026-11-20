package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// nativePeerTargetFromLedger converts a complete, durable SQLite claim into
// a native verification request; neither the ledger nor this mapping is proof
// of ownership. Incus root must independently read all effective NICs.
func nativePeerTargetFromLedger(ledger state.NetworkAuditLedger) NativePeerBridgeAuditTarget {
	t := NativePeerBridgeAuditTarget{Project: ledger.Project, Bridge: ledger.Bridge, ACL: ledger.ACL}
	for _, b := range ledger.Boxes {
		t.Boxes = append(t.Boxes, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6})
	}
	return t
}

// NativePeerBridgeAuditFromStore is a host-root, read-only rendezvous between
// the protected service SQLite owner ledger and the independently observed
// Incus global bridge/Project/ACL/instance inventory. It grants no ACL write,
// no guest restart, no kernel access and no MCP root delegation.
func NativePeerBridgeAuditFromStore(ctx context.Context, s *state.Store, project, bridge, acl string) (NativePeerBridgeAuditResult, error) {
	var zero NativePeerBridgeAuditResult
	if os.Geteuid() != 0 {
		return zero, errors.New("native peer audit requires host root")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || project != "user-"+u.Uid {
		return zero, errors.New("native peer audit requires exact confined service Project")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	return nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl)
}

// nativePeerBridgeAuditFromStoreWithVerifier sandwiches two independent
// native root audits between three protected SQLite owner/lease snapshots.
// A single stale SQLite claim or a changed native digest never authorizes
// a positive audit receipt. This is NOT a cross-resource transaction: trusted
// out-of-band host root writers can still race after the last read.
func nativePeerBridgeAuditFromStoreWithVerifier(ctx context.Context, s *state.Store, project, bridge, acl string,
	readNative func(context.Context, NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error)) (NativePeerBridgeAuditResult, error) {
	var zero NativePeerBridgeAuditResult
	if s == nil || readNative == nil {
		return zero, errors.New("missing protected ledger or native root witness")
	}
	before, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	target := nativePeerTargetFromLedger(before)
	first, err := readNative(ctx, target)
	if err != nil {
		return zero, err
	}
	if first.Project != project || first.Bridge != bridge || first.ACL != acl ||
		first.OwnedBoxes != len(before.Boxes) || len(first.NativeDigestSHA256) != 64 {
		return zero, errors.New("native audit proof is incomplete or mismatched")
	}
	middle, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(before, middle) {
		return zero, fmt.Errorf("protected network identity/lease changed during root audit: %w", errors.Join(err, state.ErrResourceLocked))
	}
	second, err := readNative(ctx, target)
	if err != nil {
		return zero, err
	}
	if !reflect.DeepEqual(first, second) {
		return zero, errors.New("independent native root audit changed between reads")
	}
	after, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(before, after) {
		return zero, fmt.Errorf("protected network identity/lease changed after root audit: %w", errors.Join(err, state.ErrResourceLocked))
	}
	return second, nil
}

func nativePeerBridgeAuditFromStoreWithClient(ctx context.Context, s *state.Store, root incus.InstanceServer, project, bridge, acl string) (NativePeerBridgeAuditResult, error) {
	if root == nil {
		return NativePeerBridgeAuditResult{}, errors.New("missing native root witness")
	}
	return nativePeerBridgeAuditFromStoreWithVerifier(ctx, s, project, bridge, acl,
		func(ctx context.Context, target NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
			return nativePeerBridgeAuditWithClient(ctx, root, target)
		})
}
