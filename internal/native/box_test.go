package native

import (
	"errors"
	"reflect"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

func TestS2NetworkMustBeVerifiedBeforeFirstStart(t *testing.T) {
	device := api.DevicesMap{"eth0": map[string]string{
		"type": "nic", "network": "boxweave-bridge", "security.acls": "boxweave-owned",
		"security.acls.default.ingress.action": "reject",
		"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true", "security.mac_filtering": "true",
	}}
	if !nicPrivate(device) {
		t.Fatal("safe exclusive NIC was not detected")
	}
	for _, key := range []string{"security.acls", "security.acls.default.ingress.action", "security.ipv4_filtering", "security.ipv6_filtering", "security.mac_filtering"} {
		changed := api.DevicesMap{"eth0": map[string]string{}}
		for k, v := range device["eth0"] {
			changed["eth0"][k] = v
		}
		delete(changed["eth0"], key)
		if nicPrivate(changed) {
			t.Fatalf("unverified private NIC accepted without %s", key)
		}
	}
	// A foreign ACL or policy that defaults to allowing inbound traffic
	// is not proof of isolation, even when MAC/IP filtering is enabled.
	for _, changedACL := range []string{"boxweave-owned,other-acl", "boxweave-owned,", ","} {
		changed := api.DevicesMap{"eth0": map[string]string{}}
		for k, v := range device["eth0"] {
			changed["eth0"][k] = v
		}
		changed["eth0"]["security.acls"] = changedACL
		if nicPrivate(changed) {
			t.Fatalf("multiple/empty ACL attachment accepted: %q", changedACL)
		}
	}
	for _, unsafeAction := range []string{"allow", "log", "drop"} {
		changed := api.DevicesMap{"eth0": map[string]string{}}
		for k, v := range device["eth0"] {
			changed["eth0"][k] = v
		}
		changed["eth0"]["security.acls.default.ingress.action"] = unsafeAction
		if nicPrivate(changed) {
			t.Fatalf("unexpected/non-reject default ingress accepted: %q", unsafeAction)
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

func TestS2RecordedOutputDoHTTPUsesExactSDKURLAndRestrictedProject(t *testing.T) {
	want := "http://unix.socket/1.0/instances/alpha/logs/exec-output/out?project=user-1002"
	got, err := verifiedRecordedOutputURL("http://unix.socket",
		"/1.0/instances/alpha/logs/exec-output/out", "user-1002")
	if err != nil || got != want {
		t.Fatalf("raw relative DoHTTP URL cannot reach native output: got=%q err=%v", got, err)
	}
	for _, tc := range []struct{ base, path, project string }{
		{"https://evil.example", "/etc/shadow", "owned"},
		{"http://unix.socket", "/1.0/instances/a/../b/logs/exec-output/out", "owned"},
		{"http://unix.socket", "/1.0/instances/a/logs/exec-output/out", ""},
		{"http://unix.socket?project=default", "/1.0/instances/a/logs/exec-output/out", "owned"},
		{"file:///var/lib/incus/unix.socket", "/1.0/instances/a/logs/exec-output/out", "owned"},
		{"http://unix.socket", "https://foreign/1.0/instances/a/logs/exec-output/out", "owned"},
	} {
		if target, err := verifiedRecordedOutputURL(tc.base, tc.path, tc.project); err == nil {
			t.Fatalf("unsafe native output read URL accepted: %q", target)
		}
	}
}

func TestS2ApprovedUbuntuLTSImageMetadata(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fields map[string]string
		ok     bool
	}{
		{"native simplestreams codename", map[string]string{"os": "Ubuntu", "release": "noble"}, true},
		{"numeric release", map[string]string{"os": "Ubuntu", "release": "24.04"}, true},
		{"different OS", map[string]string{"os": "Debian", "release": "noble"}, false},
		{"older LTS", map[string]string{"os": "Ubuntu", "release": "jammy"}, false},
		{"missing metadata", nil, false},
	} {
		if got := approvedUbuntu2404(tc.fields); got != tc.ok {
			t.Fatalf("%s approved=%v want=%v", tc.name, got, tc.ok)
		}
	}
}

func TestS2GuestTimeoutArgvHasOptionsSeparatorBeforeDuration(t *testing.T) {
	argv, err := boundedGuestCommand(ExecSpec{Argv: []string{"/bin/sh", "-c", "printf safe"},
		TimeoutMS: 5000})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/usr/bin/timeout", "--signal=TERM", "--kill-after=5s",
		"--", "5.000s", "/bin/sh", "-c", "printf safe"}
	if !reflect.DeepEqual(argv, want) {
		t.Fatalf("GNU timeout argv prefix incorrectly placed: got=%v want=%v", argv, want)
	}
	direct, err := boundedGuestCommand(ExecSpec{Argv: []string{"/bin/true"}})
	if err != nil || !reflect.DeepEqual(direct, []string{"/bin/true"}) {
		t.Fatalf("direct recorded Exec accidentally wrapped: %v %v", direct, err)
	}
	for _, ms := range []int{-1, 999, 900001} {
		if _, err := boundedGuestCommand(ExecSpec{TimeoutMS: ms}); !errors.Is(err, ErrNativeUnsupported) {
			t.Fatalf("invalid guest time budget %d accepted: %v", ms, err)
		}
	}
}
