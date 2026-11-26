package install

import (
	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
	"testing"
)

type lifecycleLeaseReader struct {
	incus.InstanceServer
	project   string
	leases    []api.NetworkLease
	requested []string
	bridge    string
}

func (r *lifecycleLeaseReader) UseProject(p string) incus.InstanceServer {
	r.requested = append(r.requested, p)
	r.project = p
	return r
}
func (r *lifecycleLeaseReader) GetNetworkLeases(bridge string) ([]api.NetworkLease, error) {
	r.bridge = bridge
	if r.project != "user-1001" {
		return nil, nil
	}
	return r.leases, nil
}

func TestP1ScopedCompletedCreateLeaseProject(t *testing.T) {
	b := state.VerifiedCreateBinding{Bridge: "incusbr-1001", ACL: "bw-peer-example", MAC: "02:16:3e:11:22:33", IPv4: "10.43.18.11", IPv6: "fd43:18::11"}
	rec := scopedLifecycleRecord{Project: "user-1001", Name: "alpha", Start: true, Binding: b}
	v4 := api.NetworkLease{Hostname: rec.Name, Hwaddr: b.MAC, Address: b.IPv4, Type: "static"}
	v6 := api.NetworkLease{Hostname: rec.Name, Hwaddr: b.MAC, Address: b.IPv6, Type: "static"}
	dynamic := v6
	dynamic.Type = "dynamic"
	foreign := v4
	foreign.Hostname = "foreign"
	for _, tc := range []struct {
		name, project string
		leases        []api.NetworkLease
		ok            bool
	}{
		{"exact-project-dual-stack", "user-1001", []api.NetworkLease{v4, v6, dynamic}, true},
		{"missing-static-v6", "user-1001", []api.NetworkLease{v4, dynamic}, false},
		{"duplicate", "user-1001", []api.NetworkLease{v4, v6, v6}, false},
		{"foreign-lease", "user-1001", []api.NetworkLease{v4, v6, foreign}, false},
		{"wrong-project", "user-1002", []api.NetworkLease{v4, v6}, false},
		{"default-project", "default", []api.NetworkLease{v4, v6}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &lifecycleLeaseReader{project: "default", leases: tc.leases}
			e := nativeScopedCompletedCreateLeaseCheck(r, tc.project, rec)
			if (e == nil) != tc.ok {
				t.Fatal(e)
			}
			if tc.project == rec.Project {
				if len(r.requested) != 1 || r.requested[0] != rec.Project || r.bridge != b.Bridge {
					t.Fatal("wrong lease lookup scope", r.requested, r.bridge)
				}
			} else if len(r.requested) != 0 {
				t.Fatal("wrong project queried")
			}
		})
	}
}
