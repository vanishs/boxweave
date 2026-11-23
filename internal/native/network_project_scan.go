package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/netip"
	"reflect"
	"slices"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/networklimits"
)

// NetworkScanExpected is trusted same-Project SQLite data, never raw MCP input.
// No native configuration or credentials leave the confined Reader.
type NetworkScanExpected struct {
	Name, UUID, OriginalID, Bridge, ACL, MAC, IPv4, IPv6 string
	HasPin                                               bool
}
type NetworkScanIssue struct {
	BoxID, Reason string
}
type NetworkScanResult struct {
	Project                     string
	ObservedCount, UnownedCount int
	ObservationDigest           string
	Issues                      []NetworkScanIssue
}

type networkScanSource interface {
	GetInstanceNames(api.InstanceType) ([]string, error)
	GetInstances(api.InstanceType) ([]api.Instance, error)
	GetNetwork(string) (*api.Network, string, error)
	GetNetworkLeases(string) ([]api.NetworkLease, error)
	GetNetworkState(string) (*api.NetworkState, error)
}

// scanNativeProject compares an exact bounded inventory and cached static
// lease witnesses. It does not adopt unrecognized native names or disclose
// their config/identity. No native mutations are reachable.
func scanNativeProject(ctx context.Context, c networkScanSource, project string, expected []NetworkScanExpected) (NetworkScanResult, error) {
	out := NetworkScanResult{Project: project, Issues: []NetworkScanIssue{}}
	if e := ctx.Err(); e != nil {
		return out, e
	}
	names, e := c.GetInstanceNames(api.InstanceTypeAny)
	if e != nil {
		return out, e
	}
	instances, e := c.GetInstances(api.InstanceTypeAny)
	if e != nil {
		return out, e
	}
	if len(names) > networklimits.MaxProjectBoxes || len(instances) > networklimits.MaxProjectBoxes || len(names) != len(instances) {
		return out, ErrUnverified
	}
	actualNames := make([]string, 0, len(instances))
	found := map[string]*api.Instance{}
	for i := range instances {
		inst := &instances[i]
		if inst.Name == "" || (inst.Project != "" && inst.Project != project) || found[inst.Name] != nil {
			return out, ErrUnverified
		}
		actualNames = append(actualNames, inst.Name)
		found[inst.Name] = inst
	}
	slices.Sort(names)
	slices.Sort(actualNames)
	if !slices.Equal(names, actualNames) {
		return out, ErrUnverified
	}
	out.ObservedCount = len(instances)
	seen := map[string]bool{}
	type bridgeInfo struct {
		network *api.Network
		etag    string
		p4, p6  netip.Prefix
		leases  []api.NetworkLease
		err     error
	}
	bridges := map[string]bridgeInfo{}
	for _, want := range expected {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		if want.Name == "" || want.UUID == "" || seen[want.Name] {
			return out, ErrUnverified
		}
		seen[want.Name] = true
		add := func(reason string) {
			out.Issues = append(out.Issues, NetworkScanIssue{BoxID: want.UUID, Reason: reason})
		}
		inst := found[want.Name]
		if inst == nil {
			add("NATIVE_OWNED_BOX_MISSING")
			continue
		}
		if inst.Config["volatile.uuid"] != want.UUID {
			add("NATIVE_OWNED_UUID_DRIFT")
			continue
		}
		if !want.HasPin {
			add("NATIVE_DB_PIN_UNVERIFIED")
			continue
		}
		if want.OriginalID == "" || inst.Config[createOperationKey] != want.OriginalID {
			add("NATIVE_CREATE_MARKER_DRIFT")
			continue
		}
		if e = ValidatePeerACLBox(inst, project, want.Bridge, want.ACL, PeerACLBox{
			Name: want.Name, UUID: want.UUID, MAC: want.MAC, IPv4: want.IPv4, IPv6: want.IPv6}, false); e != nil {
			add("NATIVE_NIC_IDENTITY_DRIFT")
			continue
		}
		info, ok := bridges[want.Bridge]
		if !ok {
			var n *api.Network
			n, info.etag, e = c.GetNetwork(want.Bridge)
			info.network = n
			info.err = e
			if e == nil && n != nil {
				info.leases, info.err = c.GetNetworkLeases(want.Bridge)
				if info.err == nil && len(info.leases) > 131072 {
					info.err = ErrNativeUnsupported
				}
				if info.err == nil {
					var st *api.NetworkState
					st, info.err = c.GetNetworkState(want.Bridge)
					if info.err == nil {
						info.p4, info.p6, info.err = peerStatePrefixes(n, st)
					}
				}
			}
			if info.err == nil && n == nil {
				info.err = ErrUnverified
			}
			bridges[want.Bridge] = info
		}
		if info.err != nil {
			add("NATIVE_NETWORK_LEASE_READ_UNVERIFIED")
			continue
		}
		// Prefixes are separately verified inside the restricted SDK read.
		// This scan is an exact NIC + static lease witness, not a root/kernel audit.
		if !staticScanLeases(inst, info.network, info.p4, info.p6, info.leases, project, want) {
			add("NATIVE_STATIC_LEASE_DRIFT")
		}
	}
	for name := range found {
		if !seen[name] {
			out.UnownedCount++
		}
	}
	again, e := c.GetInstanceNames(api.InstanceTypeAny)
	if e != nil {
		return out, e
	}
	slices.Sort(again)
	if !slices.Equal(names, again) {
		return out, ErrUnverified
	}
	// Re-read the full native instance config. Even a harmless concurrent
	// change invalidates this diagnostic snapshot rather than producing a
	// false cross-source all-clear.
	latest, e := c.GetInstances(api.InstanceTypeAny)
	if e != nil {
		return out, e
	}
	slices.SortFunc(instances, func(a, b api.Instance) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	slices.SortFunc(latest, func(a, b api.Instance) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})
	if !reflect.DeepEqual(instances, latest) {
		return out, ErrUnverified
	}
	// An instance list can remain identical while the bridge leases or
	// gateway are changing. Re-read BOTH native witnesses and refuse
	// a torn or unstable snapshot instead of reporting a false all-clear.
	leaseWitnesses := map[string][]string{}
	bridgeWitnesses := map[string]string{}
	for bridge, first := range bridges {
		if first.err != nil {
			continue
		}
		if e = ctx.Err(); e != nil {
			return out, e
		}
		current, etag, readErr := c.GetNetwork(bridge)
		if readErr != nil {
			return out, readErr
		}
		if etag != first.etag || !reflect.DeepEqual(current, first.network) {
			return out, ErrUnverified
		}
		liveState, readErr := c.GetNetworkState(bridge)
		if readErr != nil {
			return out, readErr
		}
		p4, p6, readErr := peerStatePrefixes(current, liveState)
		if readErr != nil {
			return out, readErr
		}
		if p4 != first.p4 || p6 != first.p6 {
			return out, ErrUnverified
		}
		leases, readErr := c.GetNetworkLeases(bridge)
		if readErr != nil {
			return out, readErr
		}
		if len(leases) > 131072 {
			return out, ErrNativeUnsupported
		}
		baseline, readErr := canonicalNetworkLeaseSet(first.leases)
		if readErr != nil {
			return out, readErr
		}
		now, readErr := canonicalNetworkLeaseSet(leases)
		if readErr != nil {
			return out, readErr
		}
		if !slices.Equal(baseline, now) {
			return out, ErrUnverified
		}
		leaseWitnesses[bridge] = now
		bridgeRaw, je := json.Marshal(struct {
			Network *api.Network
			Etag    string
			P4, P6  netip.Prefix
		}{current, etag, p4, p6})
		if je != nil {
			return out, je
		}
		digest := sha256.Sum256(bridgeRaw)
		bridgeWitnesses[bridge] = hex.EncodeToString(digest[:])
	}
	leaseSets := map[string][]api.NetworkLease{}
	for k, v := range bridges {
		leaseSets[k] = v.leases
	}
	raw, e := json.Marshal(struct {
		Instances []api.Instance
		Leases    map[string][]string
		Network   map[string]string
	}{instances, leaseWitnesses, bridgeWitnesses})
	if e != nil {
		return out, e
	}
	h := sha256.Sum256(raw)
	out.ObservationDigest = hex.EncodeToString(h[:])
	if e := ctx.Err(); e != nil {
		return out, e
	}
	if out.UnownedCount > 0 {
		out.Issues = append(out.Issues, NetworkScanIssue{Reason: "NATIVE_UNOWNED_PROJECT_INSTANCE"})
	}
	return out, nil
}

// staticScanLeases rejects dynamic-only, duplicate, conflicting hostname/MAC
// and IP leases. Exact network prefixes are a separate native preflight.
func staticScanLeases(inst *api.Instance, nw *api.Network, p4, p6 netip.Prefix, leases []api.NetworkLease, project string, want NetworkScanExpected) bool {
	if nw == nil || nw.Name != want.Bridge || nw.Type != "bridge" || !nw.Managed || nw.Status != "Created" || nw.Config["security.acls"] != "" || len(nw.Locations) > 1 {
		return false
	}
	// Reuse the same strict static lease rules as Create recovery; derive
	// prefixes only from native bridge configuration (never user input).

	b, e := staticPeerBindingSnapshot(inst, nw, p4, p6, leases, project, want.Name, want.UUID, want.OriginalID)
	return e == nil && b == (VerifiedStaticPeerBinding{Bridge: want.Bridge, ACL: want.ACL, MAC: want.MAC, IPv4: want.IPv4, IPv6: want.IPv6})
}
func (r Reader) ScanProjectNativeNetwork(ctx context.Context, expected []NetworkScanExpected) (NetworkScanResult, error) {
	if len(expected) > networklimits.MaxProjectBoxes {
		return NetworkScanResult{}, ErrNativeUnsupported
	}
	c, e := r.confined(ctx)
	if e != nil {
		return NetworkScanResult{}, e
	}
	return scanNativeProject(ctx, c, r.Project, expected)
}
