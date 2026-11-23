package install

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/networklimits"
	"github.com/vanishs/boxweave/internal/state"
)

// Root-only, READ-ONLY SDK interface. No methods for ACL PUT, nft,
// instance state changes, conntrack, service restart or shell execution.
type nativePeerScopedRootReader interface {
	network(string) (*api.Network, string, error)
	names(string) ([]string, error)
	instance(string, string) (*api.Instance, string, error)
	acl(string) (*api.NetworkACL, string, error)
}

type nativePeerScopedIncusReader struct{ root incus.InstanceServer }

func (r nativePeerScopedIncusReader) network(name string) (*api.Network, string, error) {
	return r.root.UseProject("default").GetNetwork(name)
}
func (r nativePeerScopedIncusReader) names(project string) ([]string, error) {
	return r.root.UseProject(project).GetInstanceNames(api.InstanceTypeAny)
}
func (r nativePeerScopedIncusReader) instance(project, name string) (*api.Instance, string, error) {
	return r.root.UseProject(project).GetInstance(name)
}
func (r nativePeerScopedIncusReader) acl(name string) (*api.NetworkACL, string, error) {
	return r.root.UseProject("default").GetNetworkACL(name)
}

type nativePeerScopedSDKSnapshot struct {
	Network       *api.Network
	NetworkETag   string
	InstanceETags map[string]string
	Evidence      NativePeerScopedRootEvidence
}
type nativePeerScopedRootReadResult struct {
	Original        state.ScopedPeerOriginalProof
	Plan            NativePeerScopedRootTransition
	NativeACLBefore NativePeerScopedACLPreimageWitness
	NetworkETag     string
	InstanceETags   map[string]string
}

// nativePeerScopedRootReadSnapshot obtains data ONLY through a Root SDK
// reader with exact default-owned Network+ACL and tenant-owned Instances.
// This is NOT host sysfs/nft readback and has no accepted native effect.
func nativePeerScopedRootReadSnapshot(ctx context.Context, r nativePeerScopedRootReader,
	proof state.ScopedPeerOriginalProof) (nativePeerScopedSDKSnapshot, error) {
	var zero nativePeerScopedSDKSnapshot
	if ctx == nil || r == nil {
		return zero, errors.New("missing root SDK read context")
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	i := proof.Intent
	if proof.OriginalID == "" || proof.RecoveryOnly || i.Schema != 2 ||
		!strings.HasPrefix(i.Project, "user-") || i.Bridge == "" ||
		proof.Inventory.Project != i.Project || proof.Inventory.Bridge != i.Bridge ||
		len(proof.Inventory.Boxes) < 2 ||
		len(proof.Inventory.Boxes) > networklimits.MaxProjectBoxes {
		return zero, errors.New("Root SDK read lacks verified original and full tenant")
	}
	bridge, etag, e := r.network(i.Bridge)
	if e != nil || bridge == nil || etag == "" ||
		bridge.Project != "default" || bridge.Name != i.Bridge {
		return zero, errors.New("root default-Project bridge or ETag not independently observed")
	}
	if e = ctx.Err(); e != nil {
		return zero, e
	}
	names, e := r.names(i.Project)
	if e != nil || len(names) != len(proof.Inventory.Boxes) {
		return zero, errors.New("native tenant instances do not match complete protected ledger")
	}
	expectedNames := make([]string, 0, len(proof.Inventory.Boxes))
	byName := map[string]state.ScopedNetworkAuditBinding{}
	for _, b := range proof.Inventory.Boxes {
		if b.Name == "" || byName[b.Name].Name != "" {
			return zero, errors.New("native tenant SQLite names duplicated")
		}
		byName[b.Name] = b
		expectedNames = append(expectedNames, b.Name)
	}
	slices.Sort(expectedNames)
	actualNames := slices.Clone(names)
	slices.Sort(actualNames)
	if !slices.Equal(expectedNames, actualNames) {
		return zero, errors.New("foreign/missing native Box in tenant Project")
	}
	out := nativePeerScopedSDKSnapshot{
		Network: bridge, NetworkETag: etag,
		InstanceETags: map[string]string{},
		Evidence: NativePeerScopedRootEvidence{
			AllProjectNames:   actualNames,
			RootNetworkUsedBy: slices.Clone(bridge.UsedBy),
			Observed:          map[string]NativePeerGuardObservedOwner{},
		},
	}
	for _, name := range actualNames {
		if e = ctx.Err(); e != nil {
			return zero, e
		}
		b := byName[name]
		inst, instanceTag, e := r.instance(i.Project, name)
		if e != nil || inst == nil || instanceTag == "" ||
			inst.Name != name || inst.Project != i.Project ||
			inst.Config["volatile.uuid"] != b.UUID {
			return zero, fmt.Errorf("tenant Box native instance identity/ETag missing: %q", name)
		}
		aclName, e := NativePeerScopedACLName(i.Project, i.Bridge, b.UUID)
		if e != nil || aclName != b.ACL {
			return zero, errors.New("native ACL identity differs from scoped SQLite")
		}
		acl, aclTag, e := r.acl(aclName)
		if e != nil || acl == nil || aclTag == "" ||
			acl.Project != "default" || acl.Name != aclName {
			return zero, fmt.Errorf("root ACL or exact native ETag missing: %q", name)
		}
		out.InstanceETags[b.UUID] = instanceTag
		out.Evidence.Observed[name] = NativePeerGuardObservedOwner{
			Instance: inst, ACL: acl, ACLTag: aclTag,
		}
	}
	if e = ctx.Err(); e != nil {
		return zero, e
	}
	return out, nil
}

// Captures the complete Incus native state TWICE and compares every full
// object, ETag and observed UsedBy. A concurrent ACL or Box change cannot
// silently pass an inconsistent read. This cannot replace global durable
// locks or kernel nft/sysfs independent pre/post validation.
func nativePeerScopedRootReadOnlyWithDriver(ctx context.Context, r nativePeerScopedRootReader,
	proof state.ScopedPeerOriginalProof) (nativePeerScopedRootReadResult, error) {
	var zero nativePeerScopedRootReadResult
	first, e := nativePeerScopedRootReadSnapshot(ctx, r, proof)
	if e != nil {
		return zero, e
	}
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, first.Network, first.Evidence)
	if e != nil {
		return zero, e
	}
	preimage, e := NativePeerScopedVerifyObservedACLPreimage(plan, first.Evidence)
	if e != nil {
		return zero, e
	}
	second, e := nativePeerScopedRootReadSnapshot(ctx, r, proof)
	if e != nil {
		return zero, e
	}
	if !reflect.DeepEqual(first, second) {
		return zero, errors.New("untrusted native SDK Root state drifted during multi-ACL read")
	}
	return nativePeerScopedRootReadResult{
		Original: proof, Plan: plan, NativeACLBefore: preimage,
		NetworkETag: first.NetworkETag, InstanceETags: first.InstanceETags,
	}, nil
}

// root-only read-only join of the protected SQLite v2 claimed original and
// actual SDK root + tenant clients; no production Root/MCP caller exists yet.
// In particular it NEVER authorizes nft/ACL effects based on this result:
// the physical Host must additionally prove sysfs/conntrack/nft kernel
// preimage, original journal and live resource locks.
func nativePeerScopedRootCollectReadOnly(ctx context.Context, store *state.Store,
	root incus.InstanceServer, original, project, bridge string) (nativePeerScopedRootReadResult, error) {
	var zero nativePeerScopedRootReadResult
	if ctx == nil || store == nil || root == nil {
		return zero, errors.New("missing protected root SQLite/Incus read-only capability")
	}
	if e := ctx.Err(); e != nil {
		return zero, e
	}
	proof, e := store.ReadRootScopedPeerOriginal(ctx, original, project, bridge)
	if e != nil {
		return zero, e
	}
	out, e := nativePeerScopedRootReadOnlyWithDriver(ctx, nativePeerScopedIncusReader{root: root}, proof)
	if e != nil {
		return zero, e
	}
	// A second full original read rejects any changed actor, leases, sibling
	// policy or original state between SQLite/SDK snapshots. Physical native
	// writers are still barred until a durable one-original Host journal.
	again, e := store.ReadRootScopedPeerOriginal(ctx, original, project, bridge)
	if e != nil || !reflect.DeepEqual(proof, again) {
		return zero, errors.New("protected SQLite peer original changed during root native read")
	}
	return out, nil
}
