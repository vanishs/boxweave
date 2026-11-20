package native

import (
	"errors"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestRC8ExecSpoolQuotaHeadroom(t *testing.T) {
	const gib = int64(1 << 30)
	cases := []struct {
		name  string
		used  uint64
		total int64
		want  bool
	}{
		{"full-quota", 2 << 30, 2 * gib, true},
		{"within-16mib-margin", (2 << 30) - (8 << 20), 2 * gib, true},
		{"exactly-16mib-free", (2 << 30) - (16 << 20), 2 * gib, false},
		{"enough-free", (2 << 30) - (48 << 20), 2 * gib, false},
		{"reported-over-limit", (2 << 30) + 1, 2 * gib, true},
		{"unknown-total", 2 << 30, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := &api.StorageVolumeStateUsage{Used: tc.used, Total: tc.total}
			if got := execSpoolHeadroomLow(u); got != tc.want {
				t.Fatalf("got %t want %t", got, tc.want)
			}
		})
	}
	if execSpoolHeadroomLow(nil) {
		t.Fatal("missing native usage is not proof of exhaustion")
	}
	if !errors.Is(ErrNativeExecQuotaLow, ErrNativeExecQuotaLow) {
		t.Fatal("lost typed pre-dispatch quota error")
	}
}

func TestRC8ExactIncusRecordedExecOpenQuotaFailure(t *testing.T) {
	const id = "6f0e3a7a-7ff7-4dc3-b46f-7d7965533ba9"
	base := "exec_" + id + ".stdout"
	for _, errText := range []string{
		"open /var/lib/incus/storage-pools/default/containers/user-1001_box/exec-output/" + base + ": disk quota exceeded",
		"open " + base + ": no space left on device",
	} {
		op := &api.Operation{ID: id, StatusCode: api.Failure, Err: errText}
		if !ExecRecordingQuotaPreStartFailure(op, "box", id) {
			t.Fatalf("genuine pre-Guest recorded output OpenFile quota denial missing: %s", errText)
		}
	}
	ambiguous := []api.Operation{
		{ID: id, StatusCode: api.Failure, Err: "no space left on device"},
		{ID: id, StatusCode: api.Failure, Err: "open unrelated.stdout: disk quota exceeded"},
		{ID: id, StatusCode: api.Success, Err: "open " + base + ": disk quota exceeded"},
		{ID: "other-id", StatusCode: api.Failure, Err: "open " + base + ": disk quota exceeded"},
		{ID: id, StatusCode: api.Failure, Err: "open " + base + ": disk quota exceeded", Metadata: map[string]any{"return": float64(1)}},
		{ID: id, StatusCode: api.Failure, Err: "open " + base + ": disk quota exceeded", Metadata: map[string]any{"output": map[string]any{}}},
		{ID: id, StatusCode: api.Failure, Err: "open " + base + ": disk quota exceeded\nmore"},
		{ID: id, StatusCode: api.Failure, Err: "close " + base + ": disk quota exceeded"},
	}
	for i := range ambiguous {
		if ExecRecordingQuotaPreStartFailure(&ambiguous[i], "box", id) {
			t.Fatalf("ambiguous recorded Exec failure misclassified as pre-start: %+v", ambiguous[i])
		}
	}
}
