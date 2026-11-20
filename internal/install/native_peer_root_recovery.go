package install

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerRootReconcileCompletedUnknown is a READ-ONLY host-root recovery
// witness for the single recoverable case: root journal says its ORIGINAL
// native effect completed but the controller lost the response and marked
// SQLite UNKNOWN. The original is never re-dispatched. An incomplete or
// partial journal NEVER passes this method and remains quarantined.
func NativePeerRootReconcileCompletedUnknown(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() || approvedSHA == "" {
		return zero, errors.New("root recovery requires protected read-only ledger and approval")
	}
	// The current durable original, never the socket caller, selects the mode.
	// Grant recovery is separately checked against the complete positive ACL
	// and actual per-NIC kernel policy; legacy B2 revoke stays unchanged.
	if _, grantErr := s.ReadRootPeerUnknownGrant(ctx, original, project, bridge, acl); grantErr == nil {
		return NativePeerRootReconcileCompletedUnknownGrant(ctx, s, project, bridge, acl, original, approvedSHA)
	}
	proof, err := s.ReadRootPeerUnknownRevocation(ctx, original, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	path, err := nativePeerConntrackOperationJournalPath(original)
	if err != nil {
		return zero, err
	}
	journal, err := nativePeerConntrackLoadJournal(path)
	if err != nil {
		return zero, err
	}
	if journal.Status != "COMPLETE_VERIFIED" || journal.Plan.PlanSHA256 != approvedSHA ||
		journal.Plan.Project != project || journal.Plan.Bridge != bridge ||
		journal.Plan.ACL != acl ||
		journal.Plan.SourceUUID != proof.Intent.SourceUUID ||
		journal.Plan.TargetUUID != proof.Intent.TargetUUID ||
		journal.Report.ProbedScopes != 4 || journal.Report.ReadyForPeer ||
		!journal.Report.KernelVerified || journal.Report.VerifiedNICs < 2 ||
		journal.Report.PostNativeDigestSHA256 == "" {
		return zero, errors.New("incomplete or mismatched root journal cannot be reconciled")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	target, _, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl,
		proof.Intent.SourceUUID, proof.Intent.TargetUUID)
	if err != nil {
		return zero, err
	}
	// Independent all-Project managed NIC and unmanaged host-route collision
	// checks run AGAIN after the original possible root crash.
	if _, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl); err != nil {
		return zero, err
	}
	if _, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, target); err != nil {
		return zero, err
	}
	if _, err := nativePeerHostRoutes(ctx, target); err != nil {
		return zero, err
	}
	nw, _, err := root.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	var a, b native.PeerACLBox
	for _, box := range target.Boxes {
		if box.UUID == proof.Intent.SourceUUID {
			a = box
		}
		if box.UUID == proof.Intent.TargetUUID {
			b = box
		}
	}
	if a.UUID == "" || b.UUID == "" {
		return zero, errors.New("lost pinned native peer identity")
	}
	a4, e1 := netip.ParseAddr(a.IPv4)
	b4, e2 := netip.ParseAddr(b.IPv4)
	a6, e3 := netip.ParseAddr(a.IPv6)
	b6, e4 := netip.ParseAddr(b.IPv6)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return zero, errors.New("invalid post-native peer address")
	}
	pair := nativePeerBinding{SourceUUID: a.UUID, TargetUUID: b.UUID,
		SourceIPv4: a4, TargetIPv4: b4, SourceIPv6: a6, TargetIPv6: b6}
	denied, err := nativeACLWithDeniedPeers(NativePrepareTarget{Project: project, ACL: acl},
		nw, nil, []nativePeerBinding{pair})
	if err != nil {
		return zero, err
	}
	got, _, err := root.GetNetworkACL(acl)
	if err != nil || got == nil || !nativePeerACLMatches(got, denied) {
		return zero, errors.New("current native ACL not the exact revoked dual-stack DROP")
	}
	if err := nativePeerACLDropCovered(got, a, b); err != nil {
		return zero, err
	}
	owned := root.UseProject(project)
	live := []string{}
	for _, box := range target.Boxes {
		inst, _, err := owned.GetInstance(box.Name)
		if err != nil {
			return zero, err
		}
		if err := native.ValidatePeerACLBox(inst, project, bridge, acl, box, false); err != nil {
			return zero, err
		}
		if (inst.Status != "Running" && inst.Status != "Ready") || inst.Config["boot.autostart"] != "false" {
			return zero, fmt.Errorf("native NIC %s not in expected fail-closed resumed state", box.Name)
		}
		live = append(live, box.Name)
	}
	if len(live) != journal.Report.VerifiedNICs {
		return zero, errors.New("post-restart running owned NIC count changed")
	}
	kernel, err := nativePeerRootKernelRules(ctx)
	if err != nil {
		return zero, err
	}
	if err := nativePeerRootVerifyNFT(kernel, project, live, pair); err != nil {
		return zero, err
	}
	// Cross-project IP-only deletion can affect unrelated holders; admission
	// above forbids collisions, and every original scoped tuple must now
	// be absent in the independent host kernel readback.
	for _, tuple := range []struct{ family, src, dst string }{
		{"ipv4", a.IPv4, b.IPv4}, {"ipv4", b.IPv4, a.IPv4},
		{"ipv6", a.IPv6, b.IPv6}, {"ipv6", b.IPv6, a.IPv6},
	} {
		found, err := nativePeerConntrackList(ctx, nativePeerConntrackOS, tuple.family, tuple.src, tuple.dst)
		if err != nil || found {
			return zero, errors.New("tracked root peer flow survived; no safe recovery")
		}
	}
	again, err := s.ReadRootPeerUnknownRevocation(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(again, proof) {
		return zero, errors.New("SQLite original changed during read-only root recovery")
	}
	final, err := nativePeerConntrackLoadJournal(path)
	if err != nil || !reflect.DeepEqual(final, journal) {
		return zero, errors.New("root original journal changed during recovery proof")
	}
	receipt := nativePeerConntrackReceipt(journal)
	if !receipt.KernelVerified || !receipt.MutationComplete || receipt.ReadyForPeer {
		return zero, errors.New("no read-only completed root recovery proof")
	}
	return receipt, nil
}
