package native

import (
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestS2NetworkMustBeVerifiedBeforeFirstStart(t *testing.T) {
	device := api.DevicesMap{"eth0": map[string]string{
		"type": "nic", "network": "boxweave-bridge", "security.acls": "boxweave-owned",
		"security.ipv4_filtering": "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true",
	}}
	if !nicPrivate(device) {
		t.Fatal("safe exclusive NIC was not detected")
	}
	for _, key := range []string{"security.acls", "security.ipv4_filtering", "security.ipv6_filtering", "security.mac_filtering"} {
		changed := api.DevicesMap{"eth0": map[string]string{}}
		for k, v := range device["eth0"] {
			changed["eth0"][k] = v
		}
		delete(changed["eth0"], key)
		if nicPrivate(changed) {
			t.Fatalf("unverified private NIC accepted without %s", key)
		}
	}
	withForeign := api.DevicesMap{"eth0": device["eth0"], "other": map[string]string{"type": "nic", "network": "foreign"}}
	if nicPrivate(withForeign) {
		t.Fatal("second NIC bypassed private network restriction")
	}
}

func TestS2ExecOutputIsNotGuestExitZeroByDefault(t *testing.T) {
	success := &api.Operation{StatusCode: api.Success, Metadata: map[string]any{
		"return": float64(7),
		"output": map[string]any{"1": "/1.0/instances/alpha/logs/exec-output/a.stdout",
			"2": "/1.0/instances/alpha/logs/exec-output/a.stderr"},
	}}
	result, err := ExecEvidenceFromFinal(success)
	if err != nil || result.Exit == nil || *result.Exit != 7 || result.StdoutPath == "" || result.StderrPath == "" {
		t.Fatalf("guest exit 7 with durable output must be preserved: %+v %v", result, err)
	}
	missing := &api.Operation{StatusCode: api.Success, Metadata: map[string]any{
		"output": map[string]any{"1": "/1.0/instances/alpha/logs/exec-output/a.stdout"},
	}}
	if _, err := ExecEvidenceFromFinal(missing); err == nil {
		t.Fatal("missing guest exit was silently treated as success")
	}
	failure := &api.Operation{StatusCode: api.Failure, Metadata: success.Metadata}
	if _, err := ExecEvidenceFromFinal(failure); err == nil {
		t.Fatal("failed native operation treated as guest success")
	}
	for _, bad := range []string{"https://evil.test/1.0/instances/x/logs/exec-output/y",
		"/1.0/instances/a/../b/logs/exec-output/x", "/etc/shadow", "/1.0/instances/a/logs/exec-output/z?auth=1"} {
		if execPath(bad) != "" {
			t.Fatalf("untrusted output path accepted: %q", bad)
		}
	}
}
