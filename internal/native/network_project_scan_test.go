package native

import (
	"context"
	"errors"
	"fmt"
	"github.com/vanishs/boxweave/internal/networklimits"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

type n3ScanFake struct {
	names      []string
	instances  []api.Instance
	otherNames []string
	reads      int
}

func (f *n3ScanFake) GetInstanceNames(_ api.InstanceType) ([]string, error) {
	f.reads++
	if f.reads > 1 && f.otherNames != nil {
		return append([]string{}, f.otherNames...), nil
	}
	return append([]string{}, f.names...), nil
}
func (f *n3ScanFake) GetInstances(_ api.InstanceType) ([]api.Instance, error) {
	return append([]api.Instance{}, f.instances...), nil
}
func (f *n3ScanFake) GetNetwork(string) (*api.Network, string, error)     { return nil, "", ErrUnverified }
func (f *n3ScanFake) GetNetworkLeases(string) ([]api.NetworkLease, error) { return nil, ErrUnverified }
func (f *n3ScanFake) GetNetworkState(string) (*api.NetworkState, error)   { return nil, ErrUnverified }

func n3FakeInst(name, uuid, original, project string) api.Instance {
	cfg := map[string]string{"volatile.uuid": uuid}
	if original != "" {
		cfg["user.boxweave.create_operation"] = original
	}
	return api.Instance{Name: name, Project: project, InstancePut: api.InstancePut{Config: cfg}}
}
func TestN3NativeProjectInventoryScopeAndIncompleteCases(t *testing.T) {
	ctx := context.Background()
	project := "user-1001"
	expected := []NetworkScanExpected{{Name: "alpha", UUID: "uuid-alpha", HasPin: true, OriginalID: "original"}}
	cases := []struct {
		name         string
		native       []api.Instance
		names, after []string
		want         string
		unowned      int
		error        bool
	}{

		{"missing", nil, nil, nil, "NATIVE_OWNED_BOX_MISSING", 0, false},
		{"uuid drift", []api.Instance{n3FakeInst("alpha", "foreign-uuid", "original", "")}, []string{"alpha"}, nil, "NATIVE_OWNED_UUID_DRIFT", 0, false},
		{"marker drift", []api.Instance{n3FakeInst("alpha", "uuid-alpha", "", "")}, []string{"alpha"}, nil, "NATIVE_CREATE_MARKER_DRIFT", 0, false},
		{"unowned extra name redacted", []api.Instance{n3FakeInst("alpha", "uuid-alpha", "original", ""), n3FakeInst("foreign-private", "foreign-uuid", "", "")}, []string{"alpha", "foreign-private"}, nil, "NATIVE_UNOWNED_PROJECT_INSTANCE", 1, false},
		{"inventory name mismatch", nil, []string{"unknown"}, nil, "", 0, true},
		{"inventory changes across read", []api.Instance{n3FakeInst("alpha", "uuid-alpha", "", "")}, []string{"alpha"}, []string{"beta"}, "", 0, true},
		{"wrong project", []api.Instance{n3FakeInst("alpha", "uuid-alpha", "", "foreign")}, []string{"alpha"}, nil, "", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &n3ScanFake{instances: tc.native, names: tc.names, otherNames: tc.after}
			out, e := scanNativeProject(ctx, d, project, expected)
			if tc.error {
				if !errors.Is(e, ErrUnverified) {
					t.Fatal("invalid inventory accepted", out, e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if out.UnownedCount != tc.unowned {
				t.Fatal(out)
			}
			found := false
			for _, issue := range out.Issues {
				if issue.Reason == tc.want {
					found = true
				}
				if issue.BoxID == "foreign-private" || issue.BoxID == "foreign-uuid" {
					t.Fatal("native foreign details leaked", out)
				}
			}
			if !found {
				t.Fatal("missed expected reason", out, tc.want)
			}
		})
	}
}

func TestN3NativeProject1024ObjectsAnd1025FailureNoPartial(t *testing.T) {
	const project = "user-1001"
	f := &n3ScanFake{}
	expected := make([]NetworkScanExpected, 0, networklimits.MaxProjectBoxes)
	for i := 0; i < networklimits.MaxProjectBoxes; i++ {
		name := fmt.Sprintf("box-%04d", i)
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i+1)
		f.names = append(f.names, name)
		f.instances = append(f.instances, n3FakeInst(name, uuid, "", project))
		expected = append(expected, NetworkScanExpected{Name: name, UUID: uuid, HasPin: false})
	}
	result, e := scanNativeProject(context.Background(), f, project, expected)
	if e != nil || result.ObservedCount != networklimits.MaxProjectBoxes ||
		result.UnownedCount != 0 || len(result.Issues) != networklimits.MaxProjectBoxes || result.ObservationDigest == "" {
		t.Fatal("bounded complete native 1024 scan missing", result.ObservedCount, len(result.Issues), e)
	}
	for _, issue := range result.Issues {
		if issue.BoxID == "" || issue.Reason != "NATIVE_DB_PIN_UNVERIFIED" {
			t.Fatal("synthetic owner incorrectly adopted", issue)
		}
	}
	name := "box-overflow"
	f.names = append(f.names, name)
	f.instances = append(f.instances, n3FakeInst(name, "overflow-uuid", "", project))
	f.reads = 0
	extra, e := scanNativeProject(context.Background(), f, project, expected)
	if !errors.Is(e, ErrUnverified) || extra.ObservationDigest != "" || len(extra.Issues) != 0 {
		t.Fatal("1025 native objects were silently truncated", extra, e)
	}
}
