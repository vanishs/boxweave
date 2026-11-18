package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"regexp"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// The root operator approves the pre-mutation SHA for exactly one claimed
// original. It is derived from host routing, all-Project inventory, exact ACL,
// native UUIDs, instance states/ETags and the protected SQLite identity.
// Unlike the post-isolation conntrack review, this is computable BEFORE the
// first native write, so the fsynced no-replay journal can precede isolation.
type nativePeerIntegratedPreflight struct {
	Plan                         NativePeerConntrackPlan
	Target                       NativePeerBridgeAuditTarget
	Binding                      nativePeerBinding
	GlobalSHA, RoutesSHA, ACLTag string
	Running                      []string
}

func nativePeerIntegratedReview(ctx context.Context, s *state.Store,
	project, bridge, acl, original string) (nativePeerIntegratedPreflight, error) {
	var zero nativePeerIntegratedPreflight
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("installed peer root review requires local root and protected read-only ledger")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || project != "user-"+u.Uid {
		return zero, errors.New("unapproved native root Project")
	}
	proof, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return zero, err
	}
	target, ledger, err := nativePeerConntrackTargetFromStore(ctx, s, project, bridge, acl,
		proof.Intent.SourceUUID, proof.Intent.TargetUUID)
	if err != nil {
		return zero, err
	}
	first, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl)
	if err != nil {
		return zero, err
	}
	global, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, target)
	if err != nil {
		return zero, err
	}
	routes, err := nativePeerHostRoutes(ctx, target)
	if err != nil {
		return zero, err
	}
	network, _, err := root.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	current, tag, err := root.GetNetworkACL(acl)
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
		return zero, errors.New("claimed peer absent from protected root inventory")
	}
	a4, e1 := netip.ParseAddr(a.IPv4)
	a6, e2 := netip.ParseAddr(a.IPv6)
	b4, e3 := netip.ParseAddr(b.IPv4)
	b6, e4 := netip.ParseAddr(b.IPv6)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		return zero, errors.New("unparseable pinned root addresses")
	}
	pair := nativePeerBinding{SourceUUID: a.UUID, TargetUUID: b.UUID,
		SourceIPv4: a4, SourceIPv6: a6, TargetIPv4: b4, TargetIPv6: b6}
	expected, err := nativeACLForVerifiedPeers(NativePrepareTarget{Project: project, ACL: acl},
		network, []nativePeerBinding{pair})
	if err != nil {
		return zero, err
	}
	// Never silently revoke, overwrite or re-order another live peer grant.
	if !reflect.DeepEqual(current.Ingress, expected.Ingress) ||
		!reflect.DeepEqual(current.Egress, expected.Egress) {
		return zero, errors.New("native ACL is not the exact single reviewed peer grant")
	}
	owned := root.UseProject(project)
	states := []string{}
	running := []string{}
	for _, box := range target.Boxes {
		inst, etag, err := owned.GetInstance(box.Name)
		if err != nil {
			return zero, err
		}
		if err := native.ValidatePeerACLBox(inst, project, bridge, acl, box, false); err != nil {
			return zero, err
		}
		if inst.Status == "Running" || inst.Status == "Ready" {
			running = append(running, box.Name)
		} else if inst.Status != "Stopped" {
			return zero, errors.New("unknown native peer Box state")
		}
		states = append(states, box.Name+"|"+box.UUID+"|"+inst.Status+"|"+inst.Config["boot.autostart"]+"|"+etag)
	}
	machine, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return zero, err
	}
	mid := strings.TrimSpace(string(machine))
	if len(mid) != 32 {
		return zero, errors.New("unverified host machine identity")
	}
	if _, err := hex.DecodeString(mid); err != nil {
		return zero, err
	}
	hostHash := sha256.Sum256([]byte("boxweave/root-peer-pre-isolation/v1\n" +
		mid + "\n" + original + "\n" + global + "\n" + routes + "\n" + tag + "\n" + strings.Join(states, "\n")))
	plan, err := nativePeerConntrackSealedPlan(ledger, hex.EncodeToString(hostHash[:]),
		first.NativeDigestSHA256, a.UUID, b.UUID)
	if err != nil {
		return zero, err
	}
	again, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(proof, again) {
		return zero, errors.New("claimed original changed during native preflight")
	}
	second, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl)
	if err != nil || second.NativeDigestSHA256 != first.NativeDigestSHA256 {
		return zero, errors.New("native owner scope changed during preflight")
	}
	return nativePeerIntegratedPreflight{Plan: plan, Target: target, Binding: pair,
		GlobalSHA: global, RoutesSHA: routes, ACLTag: tag, Running: running}, nil
}

// An installed controller must only request the exact root plan through the
// authenticated socket. This function is host-root-only and never executes.
func NativePeerRootIntegratedReviewOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original string) (NativePeerConntrackPlan, error) {
	p, e := nativePeerIntegratedReview(ctx, s, project, bridge, acl, original)
	return p.Plan, e
}

func nativePeerRootKernelRules(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "/usr/sbin/nft", "list", "ruleset")
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	out, err := cmd.Output()
	if err != nil || len(out) == 0 || len(out) > 4<<20 {
		return "", errors.New("missing, failed or unbounded root nft ruleset readback")
	}
	return string(out), nil
}

// Verify the three distinct effective Incus chains on EACH resumed owned NIC.
// An ACL API GET alone is not proof that the host kernel accepted the DROP.
// The known Incus 7.6 per-NIC rule format is intentionally strict: unknown
// formats refuse verification instead of returning a false safe ACK.
func nativePeerRootVerifyNFT(rules, project string, boxes []string, p nativePeerBinding) error {
	if !safePrereqName.MatchString(project) || len(boxes) == 0 || len(rules) == 0 {
		return errors.New("missing exact native NIC kernel evidence")
	}
	pairs := [][2]string{
		{p.SourceIPv4.String(), p.TargetIPv4.String()},
		{p.TargetIPv4.String(), p.SourceIPv4.String()},
		{p.SourceIPv6.String(), p.TargetIPv6.String()},
		{p.TargetIPv6.String(), p.SourceIPv6.String()},
	}
	for _, name := range boxes {
		if !safePrereqName.MatchString(name) {
			return errors.New("invalid native kernel chain owner")
		}
		for _, prefix := range []string{"in", "fwd", "out"} {
			chain := prefix + "." + project + "_" + name + ".eth0"
			re := regexp.MustCompile("(?ms)^\\s*chain " + regexp.QuoteMeta(chain) + " \\{(.*?)^\\s*\\}")
			hits := re.FindAllStringSubmatch(rules, -1)
			if len(hits) != 1 {
				return fmt.Errorf("exact per-NIC nft chain %s missing or duplicated", chain)
			}
			body := hits[0][1]
			for i, pair := range pairs {
				family := "ip"
				if i >= 2 {
					family = "ip6"
				}
				match := family + " saddr " + pair[0] + " " + family + " daddr " + pair[1] + " "
				if strings.Contains(body, match+"accept") {
					return fmt.Errorf("kernel nft still has revoked peer ACCEPT in %s", chain)
				}
				if !strings.Contains(body, match+"drop") {
					return fmt.Errorf("kernel nft lacks exact bidirectional DROP in %s", chain)
				}
			}
		}
	}
	return nil
}

func nativePeerRootStartVerified(ctx context.Context, root incus.InstanceServer,
	project string, target NativePeerBridgeAuditTarget, running []string,
	pair nativePeerBinding) (int, error) {
	owned := root.UseProject(project)
	resumed := []string{}
	success := false
	defer func() {
		if success {
			return
		}
		// Native start may take effect even when its SDK response is missing.
		// Contain ALL previously authenticated bridge consumers, including
		// the member whose start failed before it could enter `resumed`.
		// This is best effort only: SQLite/root journals remain UNKNOWN and
		// no lease, autostart setting or claim is released on uncertainty.
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, box := range target.Boxes {
			op, e := owned.UpdateInstanceState(box.Name, api.InstanceStatePut{Action: "stop", Timeout: 20}, "")
			if e == nil && op != nil {
				_ = op.WaitContext(stopCtx)
			}
		}
	}()
	for _, name := range running {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		op, err := owned.UpdateInstanceState(name, api.InstanceStatePut{Action: "start", Timeout: 30}, "")
		if err != nil || op == nil {
			return 0, fmt.Errorf("native selective resume uncertain for %s: %w", name, err)
		}
		resumed = append(resumed, name)
		if err := op.WaitContext(ctx); err != nil {
			return 0, err
		}
		var selected native.PeerACLBox
		for _, box := range target.Boxes {
			if box.Name == name {
				selected = box
				break
			}
		}
		inst, _, err := owned.GetInstance(name)
		if err != nil {
			return 0, err
		}
		if err := native.ValidatePeerACLBox(inst, project, target.Bridge, target.ACL, selected, false); err != nil {
			return 0, err
		}
		if inst.Status != "Running" && inst.Status != "Ready" || inst.Config["boot.autostart"] != "false" {
			return 0, errors.New("native resume not running or autostart fence lost")
		}
		rules, err := nativePeerRootKernelRules(ctx)
		if err != nil {
			return 0, err
		}
		if err := nativePeerRootVerifyNFT(rules, project, resumed, pair); err != nil {
			return 0, err
		}
	}
	success = true
	return len(resumed), nil
}

// This is ONE exact root-side integrated effect, journaled UNKNOWN before
// ANY guest stop, ACL PUT, conntrack delete or selective restart. No public
// root call accepts arbitrary IP ranges, ACL rules, host paths or commands.
func NativePeerRootIntegratedApplyOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original, approvedSHA string, approved bool) (NativePeerConntrackJournalReceipt, error) {
	var zero NativePeerConntrackJournalReceipt
	if !approved || os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("root integrated peer revoke requires explicit scope approval")
	}
	path, err := nativePeerConntrackOperationJournalPath(original)
	if err != nil {
		return zero, err
	}
	release, err := nativeScriptletAcquireLock(nativePeerConntrackLockPath)
	if err != nil {
		return zero, err
	}
	defer release()
	pre, err := nativePeerIntegratedReview(ctx, s, project, bridge, acl, original)
	if err != nil {
		return zero, err
	}
	if approvedSHA == "" || approvedSHA != pre.Plan.PlanSHA256 {
		return zero, errors.New("root integrated peer approval pre-isolation SHA mismatch")
	}
	if err := nativePeerConntrackPrepareOriginalJournalDirectory(); err != nil {
		return zero, err
	}
	return nativePeerConntrackDispatchOnce(ctx, path, pre.Plan, approvedSHA, approved,
		func(ctx context.Context) (NativePeerConntrackReport, error) {
			root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
			if err != nil {
				return NativePeerConntrackReport{}, err
			}
			check := func(ctx context.Context) error {
				if _, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl); err != nil {
					return err
				}
				g, err := nativePeerGlobalNetworkScopeSnapshot(ctx, root, pre.Target)
				if err != nil || g != pre.GlobalSHA {
					return errors.New("global managed network scope drift")
				}
				h, err := nativePeerHostRoutes(ctx, pre.Target)
				if err != nil || h != pre.RoutesSHA {
					return errors.New("host kernel routes drift")
				}
				proof, err := s.ReadRootPeerRevocation(ctx, original, project, bridge, acl)
				if err != nil || proof.OperationID != original {
					return errors.New("root original lost ownership during quiescence")
				}
				return nil
			}
			if err := native.IsolateSharedPeerACLWithClient(ctx, root, project, bridge, acl, pre.Target.Boxes, check); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if err := nativePeerBridgeKernelNoLivePorts(ctx, bridge); err != nil {
				return NativePeerConntrackReport{}, err
			}
			current, tag, err := root.GetNetworkACL(acl)
			if err != nil || current == nil || tag != pre.ACLTag {
				return NativePeerConntrackReport{}, errors.New("native ACL changed across full-NIC stop barrier")
			}
			nw, _, err := root.GetNetwork(bridge)
			if err != nil {
				return NativePeerConntrackReport{}, err
			}
			denied, err := nativeACLWithDeniedPeers(NativePrepareTarget{Project: project, ACL: acl},
				nw, nil, []nativePeerBinding{pre.Binding})
			if err != nil {
				return NativePeerConntrackReport{}, err
			}
			if err := root.UpdateNetworkACL(acl, denied.NetworkACLPut, tag); err != nil {
				return NativePeerConntrackReport{}, fmt.Errorf("native ACL ETag DROP update uncertain: %w", err)
			}
			got, _, err := root.GetNetworkACL(acl)
			if err != nil || !reflect.DeepEqual(got.Ingress, denied.Ingress) ||
				!reflect.DeepEqual(got.Egress, denied.Egress) {
				return NativePeerConntrackReport{}, errors.New("native dual-stack DROP ACL readback incomplete")
			}
			var a, b native.PeerACLBox
			for _, box := range pre.Target.Boxes {
				if box.UUID == pre.Binding.SourceUUID {
					a = box
				}
				if box.UUID == pre.Binding.TargetUUID {
					b = box
				}
			}
			if err := nativePeerACLDropCovered(got, a, b); err != nil {
				return NativePeerConntrackReport{}, err
			}
			if err := check(ctx); err != nil {
				return NativePeerConntrackReport{}, err
			}
			report, err := NativePeerConntrackPurgeFromStore(ctx, s, project, bridge, acl, a.UUID, b.UUID)
			if err != nil {
				return report, err
			}
			postDigest := report.NativeDigestSHA256
			// The journal is bound to the PRE-MUTATION digest. The post ACL
			// digest is separate, never misreported as an unchanged ACL.
			report.PostNativeDigestSHA256 = postDigest
			report.NativeDigestSHA256 = pre.Plan.NativeDigestSHA256
			if err := check(ctx); err != nil {
				return report, err
			}
			count, err := nativePeerRootStartVerified(ctx, root, project, pre.Target, pre.Running, pre.Binding)
			if err != nil {
				return report, err
			}
			report.VerifiedNICs = count
			report.KernelVerified = count >= 2
			if !report.KernelVerified {
				return report, errors.New("no live per-NIC dual-stack kernel evidence")
			}
			if err := check(ctx); err != nil {
				return report, err
			}
			return report, nil
		})
}
