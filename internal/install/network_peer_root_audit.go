package install

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

// NativePeerBridgeAuditTarget is an installer/root-local *read-only* evidence
// request. Names, UUIDs, MACs and fixed dual-stack IPs MUST originate from
// an independently protected, complete owned ledger, not public MCP input.
type NativePeerBridgeAuditTarget struct {
	Project string
	Bridge  string
	ACL     string
	Boxes   []native.PeerACLBox
}

// NativePeerBridgeAuditResult is a snapshot, NOT an exclusive lease, locked
// network transaction, privilege delegation or kernel/packet convergence.
type NativePeerBridgeAuditResult struct {
	Project            string `json:"project"`
	Bridge             string `json:"bridge"`
	ACL                string `json:"acl"`
	OwnedBoxes         int    `json:"owned_boxes"`
	NativeDigestSHA256 string `json:"native_digest_sha256"`
}

// NativePeerBridgeAudit is strictly a host-local root read, using the Incus
// admin socket ONLY in the installer context. The non-root MCP controller
// MUST NEVER receive this socket/credential or treat a stale audit as a
// production ACL grant. No network configuration is modified here.
func NativePeerBridgeAudit(ctx context.Context, target NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
	if os.Geteuid() != 0 {
		return NativePeerBridgeAuditResult{}, errors.New("native bridge audit requires host-local root")
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		return NativePeerBridgeAuditResult{}, err
	}
	return nativePeerBridgeAuditWithClient(ctx, root, target)
}

func nativePeerBridgeAuditWithClient(ctx context.Context, root incus.InstanceServer, t NativePeerBridgeAuditTarget) (NativePeerBridgeAuditResult, error) {
	var out NativePeerBridgeAuditResult
	if root == nil || !safePrereqName.MatchString(t.Project) || t.Project == "default" ||
		!safePrereqName.MatchString(t.Bridge) || !safePrereqName.MatchString(t.ACL) ||
		len(t.Boxes) == 0 || len(t.Boxes) > 64 {
		return out, errors.New("invalid root bridge-audit scope")
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	ownerNames := map[string]string{}
	names := make([]string, 0, len(t.Boxes))
	macs, ips := map[string]bool{}, map[string]bool{}
	for _, b := range t.Boxes {
		if !safePrereqName.MatchString(b.Name) || b.UUID == "" || ownerNames[b.Name] != "" ||
			b.IPv4 == "" || b.IPv6 == "" || b.MAC == "" ||
			ips[b.IPv4] || ips[b.IPv6] || macs[strings.ToLower(b.MAC)] {
			return out, errors.New("incomplete, duplicate or inconsistent native owner ledger")
		}
		ownerNames[b.Name] = b.UUID
		names = append(names, b.Name)
		ips[b.IPv4], ips[b.IPv6], macs[strings.ToLower(b.MAC)] = true, true, true
	}
	slices.Sort(names)
	network, ntag, err := root.GetNetwork(t.Bridge)
	if err != nil {
		return out, err
	}
	if network == nil || !network.Managed || network.Type != "bridge" || network.Status != "Created" ||
		network.Config["security.acls"] != "" {
		return out, errors.New("root network is not a dedicated unmodified bridge")
	}
	if err = nativePeerBridgeConsumersOwned(network.UsedBy, t.Project, ownerNames); err != nil {
		return out, err
	}
	// The dedicated Project cannot attach random physical NICs, additional
	// bridges or inherited foreign Profiles. Root reads remain authoritative.
	own := root.UseProject(t.Project)
	project, ptag, err := own.GetProject(t.Project)
	if err != nil {
		return out, err
	}
	if project.Config["restricted"] != "true" ||
		project.Config["restricted.devices.nic"] != "managed" ||
		project.Config["restricted.networks.access"] != t.Bridge {
		return out, errors.New("native Project may attach unmanaged or foreign network devices")
	}
	profile, pftag, err := own.GetProfile("default")
	if err != nil {
		return out, err
	}
	if profile == nil || len(profile.Devices) != 2 {
		return out, errors.New("unexpected native default Profile devices")
	}
	nic, ok := profile.Devices["eth0"]
	if !ok || nic["type"] != "nic" || nic["network"] != t.Bridge ||
		nic["security.acls"] != t.ACL || nic["security.acls.default.ingress.action"] != "reject" ||
		nic["security.ipv4_filtering"] != "true" || nic["security.ipv6_filtering"] != "true" ||
		nic["security.mac_filtering"] != "true" ||
		(nic["security.acls.default.egress.action"] != "" && nic["security.acls.default.egress.action"] != "reject") {
		return out, errors.New("native Profile does not provide default dual-stack anti-spoof isolation")
	}
	rootDisk, ok := profile.Devices["root"]
	if !ok || rootDisk["type"] != "disk" || rootDisk["path"] != "/" {
		return out, errors.New("unexpected native root device")
	}
	acl, atag, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return out, err
	}
	if acl == nil || acl.Name != t.ACL || len(acl.UsedBy) == 0 {
		return out, errors.New("dedicated ACL missing or unattached")
	}
	if err = nativePeerBridgeConsumersOwned(acl.UsedBy, t.Project, ownerNames); err != nil {
		return out, fmt.Errorf("ACL consumers: %w", err)
	}
	nativeNames, err := own.GetInstanceNames(api.InstanceTypeAny)
	if err != nil {
		return out, err
	}
	slices.Sort(nativeNames)
	if !slices.Equal(names, nativeNames) {
		return out, errors.New("unknown or missing native sibling instance")
	}
	for _, b := range t.Boxes {
		inst, _, err := own.GetInstance(b.Name)
		if err != nil {
			return out, err
		}
		if err = native.ValidatePeerACLBox(inst, t.Project, t.Bridge, t.ACL, b, false); err != nil {
			return out, fmt.Errorf("instance %q native identity drift: %w", b.Name, err)
		}
	}
	// Re-read both global resources and project/Profile before returning a
	// digest. This detects ordinary interleaving but is NOT a native global
	// transaction: a privileged external writer can still race immediately.
	network2, nt2, err := root.GetNetwork(t.Bridge)
	if err != nil {
		return out, err
	}
	acl2, at2, err := root.GetNetworkACL(t.ACL)
	if err != nil {
		return out, err
	}
	proj2, pt2, err := own.GetProject(t.Project)
	if err != nil {
		return out, err
	}
	prof2, pft2, err := own.GetProfile("default")
	if err != nil {
		return out, err
	}
	if reason := nativePeerAuditSnapshotReason(network, ntag, acl, atag, project, ptag, profile, pftag,
		network2, nt2, acl2, at2, proj2, pt2, prof2, pft2); reason != "" {
		return out, fmt.Errorf("native root snapshot changed during audit: %s", reason)
	}
	networkUsedBy, aclUsedBy := slices.Clone(network.UsedBy), slices.Clone(acl.UsedBy)
	slices.Sort(networkUsedBy)
	slices.Sort(aclUsedBy)
	data, _ := json.Marshal(struct {
		Target                                     NativePeerBridgeAuditTarget
		NativeNames                                []string
		NetworkTag, ACLTag, ProjectTag, ProfileTag string
		NetworkUsedBy, ACLUsedBy                   []string
	}{t, names, ntag, atag, ptag, pftag, networkUsedBy, aclUsedBy})
	sum := sha256.Sum256(append([]byte("s4b2-root-peer-audit-v1\n"), data...))
	out = NativePeerBridgeAuditResult{Project: t.Project, Bridge: t.Bridge, ACL: t.ACL,
		OwnedBoxes: len(t.Boxes), NativeDigestSHA256: hex.EncodeToString(sum[:])}
	return out, nil
}

// A change of a same-length Project config or a single Profile NIC property
// is just as security-critical as a changed object count. ETags and used_by
// alone cannot be substituted for the complete independently read snapshot.
// Stable authorization fields must match exactly, but Incus does not
// promise a particular ordering for used_by collections. Compare those as
// sets, NOT serialized list positions; never relax config/device/ACL diffs.
func nativePeerAuditSameUsedBy(a, b []string) bool {
	x, y := slices.Clone(a), slices.Clone(b)
	slices.Sort(x)
	slices.Sort(y)
	return slices.Equal(x, y)
}
func nativePeerAuditSnapshotReason(n *api.Network, ne string, a *api.NetworkACL, ae string,
	p *api.Project, pe string, f *api.Profile, fe string,
	n2 *api.Network, ne2 string, a2 *api.NetworkACL, ae2 string,
	p2 *api.Project, pe2 string, f2 *api.Profile, fe2 string) string {
	if n == nil || a == nil || p == nil || f == nil ||
		n2 == nil || a2 == nil || p2 == nil || f2 == nil {
		return "missing independent root read"
	}
	if ne != ne2 || ae != ae2 || pe != pe2 || fe != fe2 {
		return "native resource ETag changed"
	}
	if n.Name != n2.Name || n.Type != n2.Type || n.Managed != n2.Managed ||
		n.Status != n2.Status || n.Description != n2.Description ||
		!reflect.DeepEqual(n.Config, n2.Config) {
		return "native bridge configuration changed"
	}
	if !nativePeerAuditSameUsedBy(n.UsedBy, n2.UsedBy) {
		return "native bridge consumers changed"
	}
	if a.Name != a2.Name || !reflect.DeepEqual(a.NetworkACLPut, a2.NetworkACLPut) ||
		a.Project != a2.Project {
		return "native ACL rules or scope changed"
	}
	if !nativePeerAuditSameUsedBy(a.UsedBy, a2.UsedBy) {
		return "native ACL consumers changed"
	}
	if p.Name != p2.Name || !reflect.DeepEqual(p.ProjectPut, p2.ProjectPut) {
		return "native Project restriction changed"
	}
	if !nativePeerAuditSameUsedBy(p.UsedBy, p2.UsedBy) {
		return "native Project consumers changed"
	}
	if f.Name != f2.Name || !reflect.DeepEqual(f.ProfilePut, f2.ProfilePut) {
		return "native Profile NIC/device changed"
	}
	if !nativePeerAuditSameUsedBy(f.UsedBy, f2.UsedBy) {
		return "native Profile consumers changed"
	}
	return ""
}
func nativePeerAuditSnapshotEqual(n *api.Network, ne string, a *api.NetworkACL, ae string,
	p *api.Project, pe string, f *api.Profile, fe string,
	n2 *api.Network, ne2 string, a2 *api.NetworkACL, ae2 string,
	p2 *api.Project, pe2 string, f2 *api.Profile, fe2 string) bool {
	return nativePeerAuditSnapshotReason(n, ne, a, ae, p, pe, f, fe, n2, ne2, a2, ae2, p2, pe2, f2, fe2) == ""
}
