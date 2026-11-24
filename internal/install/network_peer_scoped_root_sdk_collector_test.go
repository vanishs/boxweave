package install

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

func pr178RootSDKClone[T any](p *T) *T {
	raw, e := json.Marshal(p)
	if e != nil {
		panic(e)
	}
	var out T
	if e = json.Unmarshal(raw, &out); e != nil {
		panic(e)
	}
	return &out
}

type pr178RootNativeReadDriver struct {
	bridge                                *api.Network
	bridgeETag                            string
	namesList                             []string
	instances                             map[string]*api.Instance
	instanceETags                         map[string]string
	aclByName                             map[string]*api.NetworkACL
	aclETags                              map[string]string
	networkReads                          int
	instReads                             map[string]int
	aclReads                              map[string]int
	driftNetwork, driftInstance, driftACL bool
}

func (d *pr178RootNativeReadDriver) network(name string) (*api.Network, string, error) {
	d.networkReads++
	if d.bridge == nil || name != d.bridge.Name {
		return nil, "", errors.New("unknown root network")
	}
	tag := d.bridgeETag
	if d.driftNetwork && d.networkReads == 2 {
		tag = "changed-native-bridge-etag"
	}
	return pr178RootSDKClone(d.bridge), tag, nil
}
func (d *pr178RootNativeReadDriver) names(project string) ([]string, error) {
	if project != "user-1001" {
		return nil, errors.New("foreign tenant project")
	}
	return slices.Clone(d.namesList), nil
}
func (d *pr178RootNativeReadDriver) instance(project, name string) (*api.Instance, string, error) {
	if project != "user-1001" {
		return nil, "", errors.New("foreign tenant project")
	}
	d.instReads[name]++
	source, ok := d.instances[name]
	if !ok {
		return nil, "", errors.New("missing native tenant instance")
	}
	tag := d.instanceETags[name]
	if d.driftInstance && name == "pr178-c" && d.instReads[name] == 2 {
		tag = "changed-native-incus-instance-etag"
	}
	return pr178RootSDKClone(source), tag, nil
}
func (d *pr178RootNativeReadDriver) acl(name string) (*api.NetworkACL, string, error) {
	d.aclReads[name]++
	source, ok := d.aclByName[name]
	if !ok {
		return nil, "", errors.New("missing root native ACL")
	}
	clone := pr178RootSDKClone(source)
	if d.driftACL && clone.Name != "" && strings.Contains(clone.UsedBy[0], "pr178-d") && d.aclReads[name] == 2 {
		clone.Egress = append(clone.Egress, api.NetworkACLRule{
			Action: "accept", Destination: "0.0.0.0/0", State: "enabled",
		})
	}
	return clone, d.aclETags[name], nil
}
func pr178RootSDKFixture(t *testing.T) (state.ScopedPeerOriginalProof, *pr178RootNativeReadDriver) {
	t.Helper()
	proof, network, evidence := pr178FourRootScopedFixture(t)
	network.Project = "default"
	network.Name = "incusbr-1001"
	network.UsedBy = slices.Clone(evidence.RootNetworkUsedBy)
	plan, err := NativePeerScopedRootTransitionFromEvidence(proof, network, evidence)
	if err != nil {
		t.Fatal(err)
	}
	driver := &pr178RootNativeReadDriver{
		bridge: pr178RootSDKClone(network), bridgeETag: "etag-root-network-preimage",
		namesList: slices.Clone(evidence.AllProjectNames),
		instances: map[string]*api.Instance{}, instanceETags: map[string]string{},
		aclByName: map[string]*api.NetworkACL{}, aclETags: map[string]string{},
		instReads: map[string]int{}, aclReads: map[string]int{},
	}
	for name, v := range evidence.Observed {
		clone := pr178RootSDKClone(v.ACL)
		expected := plan.ExpectedBeforeByUUID[v.Instance.Config["volatile.uuid"]]
		clone.NetworkACLPut = api.NetworkACLPut{
			Description: expected.Description, Config: maps.Clone(expected.Config),
			Ingress: slices.Clone(expected.Ingress),
			Egress:  slices.Clone(expected.Egress),
		}
		driver.instances[name] = pr178RootSDKClone(v.Instance)
		driver.instanceETags[name] = "etag-native-tenant-" + name
		driver.aclByName[clone.Name] = clone
		driver.aclETags[clone.Name] = v.ACLTag
	}
	return proof, driver
}
func TestPR178RootSDKReadOnlyCompleteTenantAndRootACLNoNativeMutation(t *testing.T) {
	proof, driver := pr178RootSDKFixture(t)
	got, err := nativePeerScopedRootReadOnlyWithDriver(context.Background(), driver, proof)
	if err != nil {
		t.Fatal("native Root SDK read-only capture rejected complete scope", err)
	}
	if got.Original.OriginalID != proof.OriginalID ||
		got.Plan.UnrelatedACLsVerifiedUnchanged != 2 ||
		got.NativeACLBefore.ACLCount != 4 ||
		len(got.NativeACLBefore.SHA256) != 64 ||
		got.NetworkETag != "etag-root-network-preimage" ||
		driver.networkReads != 2 || len(got.InstanceETags) != 4 {
		t.Fatalf("incomplete independent twice-observed root capture: %+v", got)
	}
	for _, name := range driver.namesList {
		if driver.instReads[name] != 2 {
			t.Fatal("native instance wasn't double-read", name)
		}
		uuid := driver.instances[name].Config["volatile.uuid"]
		acl := got.Plan.ExpectedBeforeByUUID[uuid]
		if driver.aclReads[acl.Name] != 2 {
			t.Fatal("native per-Box root ACL wasn't double-read", name)
		}
	}
	if got.Plan.OriginalBootAutostart[pr178B] != "false" ||
		got.Plan.OriginalBootAutostart[pr178D] != "false" {
		t.Fatal("untrusted native SDK lost explicit false autostart")
	}
	t.Log("PR178_V2_READONLY_ROOT_INCUS_CLIENT_COLLECTOR=PASS")
}
func TestPR178RootSDKReadOnlyRejectsUntrustedAndDriftingNativeScope(t *testing.T) {
	cases := []struct {
		name   string
		change func(*pr178RootNativeReadDriver, *state.ScopedPeerOriginalProof)
	}{
		{"wrong-bridge-default-project", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.bridge.Project = "user-1001" }},
		{"missing-bridge-etag", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.bridgeETag = "" }},
		{"missing-root-acl", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			delete(d.aclByName, p.Inventory.Boxes[0].ACL)
		}},
		{"missing-sibling-instance", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { delete(d.instances, "pr178-c") }},
		{"missing-sibling-name", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.namesList = d.namesList[:3] }},
		{"foreign-native-box", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			d.namesList = append(d.namesList, "alien")
		}},
		{"forged-v2-original", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { p.OriginalID = "untrusted" }},
		{"unknown-original", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { p.RecoveryOnly = true }},
		{"forged-original-project", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { p.Intent.Project = "foreign" }},
		{"incomplete-sqlite-Box-ledger", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			p.Inventory.Boxes = p.Inventory.Boxes[:2]
		}},
		{"missing-native-instance-etag", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.instanceETags["pr178-c"] = "" }},
		{"missing-native-acl-etag", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			for name := range d.aclETags {
				d.aclETags[name] = ""
				break
			}
		}},
		{"foreign-acl-usedby", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			for _, acl := range d.aclByName {
				acl.UsedBy = []string{"/1.0/instances/foreign?project=other"}
				break
			}
		}},
		{"root-ACL-foreign-accept", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			for _, acl := range d.aclByName {
				acl.Egress = append(acl.Egress, api.NetworkACLRule{Action: "accept", Destination: "0.0.0.0/0"})
				break
			}
		}},
		{"native-lease-drift", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			d.instances["pr178-b"].Devices["eth0"]["ipv4.address"] = "10.55.8.99"
		}},
		{"native-foreign-bridge-consumer", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) {
			d.bridge.UsedBy = append(d.bridge.UsedBy, "/1.0/instances/foreign?project=other")
		}},
		{"network-etag-changed-between-reads", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.driftNetwork = true }},
		{"instance-etag-changed-between-reads", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.driftInstance = true }},
		{"ACL-extra-accept-between-reads", func(d *pr178RootNativeReadDriver, p *state.ScopedPeerOriginalProof) { d.driftACL = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proof, d := pr178RootSDKFixture(t)
			tc.change(d, &proof)
			got, e := nativePeerScopedRootReadOnlyWithDriver(context.Background(), d, proof)
			if e == nil || got.Plan.OriginalID != "" {
				t.Fatalf("untrusted native Root SDK capture accepted: %+v", got)
			}
		})
	}
}
func TestPR178RootSDKReadOnlyRefusesCancelAndMissingProtectedStore(t *testing.T) {
	proof, d := pr178RootSDKFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := nativePeerScopedRootReadOnlyWithDriver(ctx, d, proof); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled root collector still performed a native read", e)
	}
	if d.networkReads != 0 {
		t.Fatal("root SDK network read occurred after cancel")
	}
	_, e := nativePeerScopedRootCollectReadOnly(context.Background(), nil, nil,
		strings.Repeat("a", 32), "user-1001", "incusbr-1001")
	if e == nil {
		t.Fatal("native root collector lacks protected SQLite capability")
	}
}
