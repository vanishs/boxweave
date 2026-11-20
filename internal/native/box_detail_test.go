package native

import (
	"errors"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
)

type fullReaderFixture struct {
	supported bool
	full      *api.InstanceFull
	err       error
	calls     int
}

func (f *fullReaderFixture) HasExtension(name string) bool {
	return name == "instance_get_full" && f.supported
}
func (f *fullReaderFixture) GetInstanceFull(name string) (*api.InstanceFull, string, error) {
	f.calls++
	return f.full, "etag", f.err
}

func TestS4CExactFullInstanceObservation(t *testing.T) {
	full := &api.InstanceFull{
		Instance: api.Instance{
			InstancePut: api.InstancePut{Config: api.ConfigMap{"volatile.uuid": "owned-uuid", "cloud-init.user-data": "synthetic-fixture"}},
			Name:        "bw-test", Project: "user-1001", Status: "Running",
			ExpandedConfig: api.ConfigMap{"limits.cpu": "2", "limits.memory": "1GiB"},
		},
		State:     &api.InstanceState{Status: "Running", Processes: 7},
		Snapshots: []api.InstanceSnapshot{{}},
		Backups:   []api.InstanceBackup{{}},
	}
	f := &fullReaderFixture{supported: true, full: full}
	start := time.Now()
	got, err := readExactBoxFull(f, "user-1001", "bw-test", "owned-uuid")
	if err != nil || f.calls != 1 || got.InstanceFull != full || got.ObservedAt.Before(start) ||
		got.Box.UUID != "owned-uuid" || got.Box.NativeState != "Running" || got.Box.GuestReady != nil || got.Box.CPU != "2" {
		t.Fatalf("wrong exact observation: calls=%d err=%v", f.calls, err)
	}
	full.Status = "Stopped"
	full.State = nil
	stopped, err := readExactBoxFull(f, "user-1001", "bw-test", "owned-uuid")
	if err != nil || stopped.InstanceFull.State != nil || stopped.Box.GuestReady == nil || *stopped.Box.GuestReady {
		t.Fatal("stopped metadata must not synthesize runtime state")
	}
	for _, tc := range []struct{ name, project, uuid string }{
		{"bw-other", "user-1001", "owned-uuid"},
		{"bw-test", "user-1002", "owned-uuid"},
		{"bw-test", "user-1001", "replaced-uuid"},
	} {
		denied, err := readExactBoxFull(f, tc.project, tc.name, tc.uuid)
		if !errors.Is(err, ErrNotOwned) || denied.InstanceFull != nil {
			t.Fatal("foreign observation escaped")
		}
	}
}

func TestS4CFullReadFailsClosedWithoutExtensionAndOnNativeError(t *testing.T) {
	f := &fullReaderFixture{}
	got, err := readExactBoxFull(f, "user-1001", "bw-test", "owned-uuid")
	if !errors.Is(err, ErrNativeUnsupported) || f.calls != 0 || got.InstanceFull != nil {
		t.Fatal("must not use non-atomic SDK fallback")
	}
	f.supported = true
	f.err = ErrUnverified
	got, err = readExactBoxFull(f, "user-1001", "bw-test", "owned-uuid")
	if !errors.Is(err, ErrUnverified) || f.calls != 1 || got.InstanceFull != nil {
		t.Fatal("native error must not become full view")
	}
	f.err = nil
	got, err = readExactBoxFull(f, "user-1001", "bw-test", "owned-uuid")
	if !errors.Is(err, ErrNotOwned) || got.InstanceFull != nil {
		t.Fatal("nil native response accepted")
	}
}
