package install

import (
	"context"
	"errors"
	"os"
	"os/user"

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

func nativePeerBridgeAuditFromStoreWithClient(ctx context.Context, s *state.Store, root incus.InstanceServer, project, bridge, acl string) (NativePeerBridgeAuditResult, error) {
	var zero NativePeerBridgeAuditResult
	if s == nil || root == nil {
		return zero, errors.New("missing protected state or native root witness")
	}
	ledger, err := s.ReadCompleteNetworkAuditLedger(ctx, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	return nativePeerBridgeAuditWithClient(ctx, root, nativePeerTargetFromLedger(ledger))
}
