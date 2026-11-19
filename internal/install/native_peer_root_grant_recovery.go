package install

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"reflect"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// NativePeerRootReconcileCompletedUnknownGrant is strictly READ ONLY.
// A journal that was not already COMPLETE_VERIFIED never authorizes a replay,
// restart, ACL write, or release of the original network claims.
func NativePeerRootReconcileCompletedUnknownGrant(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() || approvedSHA == "" {
		return zero, errors.New("grant recovery needs approved protected root read-only ledger")
	}
	proof, err := s.ReadRootPeerUnknownGrant(ctx, original, project, bridge, acl)
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
		journal.Plan.Project != project || journal.Plan.Bridge != bridge || journal.Plan.ACL != acl ||
		journal.Plan.SourceUUID != proof.Intent.SourceUUID || journal.Plan.TargetUUID != proof.Intent.TargetUUID ||
		journal.Report.ProbedScopes != 4 || journal.Report.ReadyForPeer ||
		!journal.Report.KernelVerified || journal.Report.VerifiedNICs < 2 ||
		journal.Report.PostNativeDigestSHA256 == "" {
		return zero, errors.New("grant recovery refuses incomplete or mismatched root journal")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	target, _, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl, proof.Intent.SourceUUID, proof.Intent.TargetUUID)
	if err != nil {
		return zero, err
	}
	if _, err = nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl); err != nil {
		return zero, err
	}
	if _, err = nativePeerGlobalNetworkScopeSnapshot(ctx, root, target); err != nil {
		return zero, err
	}
	if _, err = nativePeerHostRoutes(ctx, target); err != nil {
		return zero, err
	}
	nw, _, err := root.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	expected, err := NativePeerACLFromPolicySnapshot(NativePrepareTarget{Project: project, ACL: acl}, nw, proof.Target, proof.Revoked)
	if err != nil {
		return zero, err
	}
	current, _, err := root.GetNetworkACL(acl)
	if err != nil || current == nil || !nativePeerACLMatches(current, expected) {
		return zero, errors.New("grant recovery current whole native ACL differs from reviewed durable policy")
	}
	owned := root.UseProject(project)
	var running []string
	for _, b := range target.Boxes {
		inst, _, e := owned.GetInstance(b.Name)
		if e != nil {
			return zero, e
		}
		if e = native.ValidatePeerACLBox(inst, project, bridge, acl, b, false); e != nil {
			return zero, e
		}
		if inst.Config["boot.autostart"] != "false" {
			return zero, errors.New("grant recovery autostart fence absent")
		}
		switch inst.Status {
		case "Running", "Ready":
			running = append(running, b.Name)
		case "Stopped":
		default:
			return zero, errors.New("grant recovery unknown native NIC state")
		}
	}
	if len(running) != journal.Report.VerifiedNICs {
		return zero, errors.New("grant recovery resumed NIC count differs from original journal")
	}
	rules, err := nativePeerRootKernelRules(ctx)
	if err != nil {
		return zero, err
	}
	if err = nativePeerRootVerifyGrantNFT(rules, project, target.Boxes, running, proof.Target); err != nil {
		return zero, err
	}
	if err = nativePeerRootVerifyRevokedNFT(rules, project, running, proof.Revoked); err != nil {
		return zero, err
	}
	var source, dest native.PeerACLBox
	for _, b := range target.Boxes {
		if b.UUID == proof.Intent.SourceUUID {
			source = b
		}
		if b.UUID == proof.Intent.TargetUUID {
			dest = b
		}
	}
	if source.UUID == "" || dest.UUID == "" {
		return zero, errors.New("grant recovery source/destination native UUID absent")
	}
	for _, raw := range []string{source.IPv4, source.IPv6, dest.IPv4, dest.IPv6} {
		ip, e := netip.ParseAddr(raw)
		if e != nil || ip.String() != raw || !ip.IsGlobalUnicast() || ip.Is4In6() {
			return zero, errors.New("grant recovery noncanonical native address")
		}
	}
	// The original root grant deliberately purged these four directed scopes.
	// A surviving tracked original flow makes read-only settlement unsafe.
	for _, tuple := range []struct{ family, src, dst string }{
		{"ipv4", source.IPv4, dest.IPv4}, {"ipv4", dest.IPv4, source.IPv4},
		{"ipv6", source.IPv6, dest.IPv6}, {"ipv6", dest.IPv6, source.IPv6},
	} {
		found, e := nativePeerConntrackList(ctx, nativePeerConntrackOS, tuple.family, tuple.src, tuple.dst)
		if e != nil || found {
			return zero, errors.New("grant recovery old tracked peer flow or conntrack uncertainty")
		}
	}
	again, err := s.ReadRootPeerUnknownGrant(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(again, proof) {
		return zero, errors.New("grant recovery SQLite original changed during verification")
	}
	final, err := nativePeerConntrackLoadJournal(path)
	if err != nil || !reflect.DeepEqual(final, journal) {
		return zero, errors.New("grant recovery original journal changed during verification")
	}
	receipt := nativePeerConntrackReceipt(journal)
	if !receipt.KernelVerified || !receipt.MutationComplete || receipt.ReadyForPeer {
		return zero, errors.New("grant recovery incomplete read-only root witness")
	}
	return receipt, nil
}
