package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/user"
	"reflect"
	"slices"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// A root-side B3 preview is scoped to one already-claimed, admin-owned
// grant original. It is NOT an SDK ACL update or a public root capability.
type nativePeerGrantPreflight struct {
	Plan                         NativePeerConntrackPlan
	Target                       NativePeerBridgeAuditTarget
	Proof                        state.RootPeerGrant
	Before, After                api.NetworkACLsPost
	Source, Destination          native.PeerACLBox
	ACLTag, GlobalSHA, RoutesSHA string
	Running                      []string
}

func nativePeerRootGrantReview(ctx context.Context, s *state.Store,
	project, bridge, acl, original string) (nativePeerGrantPreflight, error) {
	var zero nativePeerGrantPreflight
	if os.Geteuid() != 0 || s == nil || !s.IsNetworkAuditReadOnly() {
		return zero, errors.New("root grant requires protected read-only SQLite")
	}
	u, err := user.Lookup("boxweave")
	if err != nil || project != "user-"+u.Uid {
		return zero, errors.New("unapproved host root Project")
	}
	proof, err := s.ReadRootPeerGrant(ctx, original, project, bridge, acl)
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
	nw, _, err := root.GetNetwork(bridge)
	if err != nil {
		return zero, err
	}
	current, tag, err := root.GetNetworkACL(acl)
	if err != nil || current == nil || tag == "" {
		return zero, errors.New("unverifiable pre-grant ACL ETag")
	}
	scope := NativePrepareTarget{Project: project, ACL: acl}
	beforeRevoked := slices.Clone(proof.Revoked)
	if proof.Regrant {
		var found bool
		for _, row := range proof.Target {
			if row.SourceUUID == proof.Intent.SourceUUID && row.TargetUUID == proof.Intent.TargetUUID {
				beforeRevoked = append(beforeRevoked, row)
				found = true
				break
			}
		}
		if !found {
			return zero, errors.New("missing exact pending regrant tombstone")
		}
	}
	before, err := NativePeerACLFromPolicySnapshot(scope, nw, proof.Confirmed, beforeRevoked)
	if err != nil {
		return zero, err
	}
	after, err := NativePeerACLFromPolicySnapshot(scope, nw, proof.Target, proof.Revoked)
	if err != nil {
		return zero, err
	}
	if !nativePeerACLMatches(current, before) || !nativePeerACLPolicyChanges(before, after) {
		return zero, errors.New("pre-grant native ACL must match exact confirmed-only allowlist")
	}
	owned := root.UseProject(project)
	var a, b native.PeerACLBox
	var running, states []string
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
			return zero, errors.New("unverified grant instance lifecycle state")
		}
		states = append(states, box.Name+"|"+box.UUID+"|"+inst.Status+"|"+inst.Config["boot.autostart"]+"|"+etag)
		if box.UUID == proof.Intent.SourceUUID {
			a = box
		}
		if box.UUID == proof.Intent.TargetUUID {
			b = box
		}
	}
	if a.UUID == "" || b.UUID == "" {
		return zero, errors.New("native grant pair not in owned NIC inventory")
	}
	if !nativePeerGrantEndpointsRunning(target, running, a.UUID, b.UUID) {
		return zero, errors.New("grant source and target must both be running for actual per-NIC verification")
	}
	mid, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return zero, err
	}
	machine := strings.TrimSpace(string(mid))
	if len(machine) != 32 {
		return zero, errors.New("unverified host identity")
	}
	if _, err := hex.DecodeString(machine); err != nil {
		return zero, err
	}
	intentBytes, err := json.Marshal(struct {
		Original      string
		Before, After api.NetworkACLsPost
		Proof         state.RootPeerGrant
	}{original, before, after, proof})
	if err != nil {
		return zero, err
	}
	hash := sha256.Sum256([]byte("boxweave/b3/root-grant/v1\n" +
		machine + "\n" + global + "\n" + routes + "\n" + tag + "\n" +
		strings.Join(states, "\n") + "\n" + string(intentBytes)))
	plan, err := nativePeerConntrackSealedPlan(ledger, hex.EncodeToString(hash[:]),
		first.NativeDigestSHA256, a.UUID, b.UUID)
	if err != nil {
		return zero, err
	}
	recheck, err := s.ReadRootPeerGrant(ctx, original, project, bridge, acl)
	if err != nil || !reflect.DeepEqual(proof, recheck) {
		return zero, errors.New("SQLite grant original drifted during preflight")
	}
	second, err := nativePeerBridgeAuditFromStoreWithClient(ctx, s, root, project, bridge, acl)
	if err != nil || first.NativeDigestSHA256 != second.NativeDigestSHA256 {
		return zero, errors.New("native grant owner scope drifted during preflight")
	}
	return nativePeerGrantPreflight{Plan: plan, Target: target, Proof: proof,
		Before: before, After: after, Source: a, Destination: b, ACLTag: tag,
		GlobalSHA: global, RoutesSHA: routes, Running: running}, nil
}

// NativePeerRootGrantReviewOperation is root-only and read-only. The installed
// socket will select by the durable original, never caller-supplied ACL data.
func NativePeerRootGrantReviewOperation(ctx context.Context, s *state.Store,
	project, bridge, acl, original string) (NativePeerConntrackPlan, error) {
	p, err := nativePeerRootGrantReview(ctx, s, project, bridge, acl, original)
	return p.Plan, err
}

// Require both live endpoints before a grant can be acknowledged. A prior
// pair of unrelated running siblings cannot substitute for native evidence.
func nativePeerGrantEndpointsRunning(target NativePeerBridgeAuditTarget, running []string, srcUUID, dstUUID string) bool {
	var src, dst bool
	for _, b := range target.Boxes {
		for _, name := range running {
			if b.Name == name && b.UUID == srcUUID {
				src = true
			}
			if b.Name == name && b.UUID == dstUUID {
				dst = true
			}
		}
	}
	return src && dst
}
