package native

import (
	"encoding/json"
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

func TestS25DefaultProfileMustNotOpenEveryRawNativeGuest(t *testing.T) {
	safe := api.DevicesMap{"eth0": {
		"type": "nic", "network": "owned-bridge", "security.acls": "owned-acl",
		"security.acls.default.ingress.action": "reject",
		"security.ipv4_filtering":              "true", "security.ipv6_filtering": "true",
		"security.mac_filtering": "true",
	}}
	if !nicProfilePrivate(safe) {
		t.Fatal("implicit Incus reject not accepted")
	}
	safe["eth0"]["security.acls.default.egress.action"] = "reject"
	if !nicProfilePrivate(safe) {
		t.Fatal("explicit egress reject not accepted")
	}
	safe["eth0"]["security.acls.default.egress.action"] = "allow"
	if nicProfilePrivate(safe) {
		t.Fatal("open default Profile egress bypassed first-Box guard")
	}
	if !nicPrivate(safe) {
		t.Fatal("instance-owned internet allow was incorrectly disabled")
	}
	safe["eth0"]["security.acls.default.egress.action"] = "ignore"
	if nicProfilePrivate(safe) {
		t.Fatal("unrecognized egress action accepted")
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

func TestRC3IncusLocalAndCatalogImagesNotUbuntuLocked(t *testing.T) {
	for _, alias := range []string{"images:ubuntu/24.04", "images:debian/13", "images:alpine/3.21", "local:approved-debian"} {
		if !ValidImageChoice(alias) {
			t.Fatalf("valid Incus image alias unexpectedly restricted: %s", alias)
		}
	}
	if ValidImageChoice("https://untrusted.example/image") {
		t.Fatal("caller-specified remote URL accepted")
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

func TestS2OwnedNICInternetToggleNeverEditsSharedACL(t *testing.T) {
	base := api.DevicesMap{
		"eth0": map[string]string{
			"type": "nic", "name": "eth0", "network": "incusbr-owned",
			"security.acls":                        "boxweave-owned",
			"security.acls.default.ingress.action": "reject",
			"security.ipv4_filtering":              "true",
			"security.ipv6_filtering":              "true",
			"security.mac_filtering":               "true",
			"security.acls.default.egress.action":  "reject",
		},
		"root": map[string]string{"type": "disk", "path": "/", "pool": "owned-pool", "size": "2GiB"},
	}
	local := api.DevicesMap{"root": map[string]string{
		"type": "disk", "path": "/", "pool": "owned-pool", "size": "3GiB",
	}}
	allowed, err := ownedPrivateNICEgress(local, base, true)
	if err != nil {
		t.Fatal(err)
	}
	if !nicPrivate(allowed) ||
		allowed["eth0"]["security.acls.default.egress.action"] != "allow" ||
		allowed["eth0"]["security.acls.default.ingress.action"] != "reject" ||
		allowed["eth0"]["security.acls"] != "boxweave-owned" ||
		allowed["eth0"]["security.ipv6_filtering"] != "true" ||
		allowed["root"]["size"] != "3GiB" ||
		base["eth0"]["security.acls.default.egress.action"] != "reject" ||
		local["eth0"] != nil {
		t.Fatalf("internet allow modified shared profile, spoof defense or unrelated disk: %#v", allowed)
	}
	denied, err := ownedPrivateNICEgress(allowed, allowed, false)
	if err != nil || !nicPrivate(denied) ||
		denied["eth0"]["security.acls.default.egress.action"] != "reject" ||
		denied["root"]["size"] != "3GiB" {
		t.Fatalf("internet deny did not preserve root/ACL and block egress: %#v %v", denied, err)
	}
	missingACL := api.DevicesMap{"eth0": map[string]string{
		"type": "nic", "network": "incusbr-owned", "security.acls.default.ingress.action": "reject",
		"security.ipv4_filtering": "true", "security.ipv6_filtering": "true",
		"security.mac_filtering": "true",
	}}
	if _, err := ownedPrivateNICEgress(nil, missingACL, true); !errors.Is(err, ErrUnsafeNetwork) {
		t.Fatalf("unauthorized egress allowed without native ACL: %v", err)
	}
}

func TestS2EffectiveNICAndDiskAlwaysReadRegardlessOfMapIteration(t *testing.T) {
	devices := api.DevicesMap{
		"root": map[string]string{
			"type": "disk", "path": "/", "size": "3GiB",
		},
		"eth0": map[string]string{
			"type": "nic", "network": "incusbr-test",
			"security.acls.default.egress.action": "allow",
		},
	}
	// The old production code broke on whichever map key it visited first,
	// making native create success nondeterministically become UNKNOWN.
	for i := 0; i < 256; i++ {
		var state BoxState
		observeBoxEffectiveDevices(&state, devices)
		if !state.Internet || state.RootDiskSize != "3GiB" {
			t.Fatalf("iteration %d lost a native resource: %+v", i, state)
		}
	}
	devices["eth0"]["security.acls.default.egress.action"] = "reject"
	var disabled BoxState
	observeBoxEffectiveDevices(&disabled, devices)
	if disabled.Internet || disabled.RootDiskSize != "3GiB" {
		t.Fatalf("internet disabled did not preserve effective disk: %+v", disabled)
	}
}

func TestS2MalformedNativeExecReturnAndStreamMetadataFailClosed(t *testing.T) {
	goodPath := "/1.0/instances/alpha/logs/exec-output/a.stdout"
	samples := []struct {
		name     string
		metadata map[string]any
	}{
		{"string-exit", map[string]any{"return": "0", "output": map[string]any{"1": goodPath}}},
		{"fractional-exit", map[string]any{"return": float64(0.75), "output": map[string]any{"1": goodPath}}},
		{"overflow-exit", map[string]any{"return": 1e40, "output": map[string]any{"1": goodPath}}},
		{"boolean-exit", map[string]any{"return": true, "output": map[string]any{"1": goodPath}}},
		{"null-exit", map[string]any{"return": nil, "output": map[string]any{"1": goodPath}}},
		{"array-exit", map[string]any{"return": []any{float64(0)}, "output": map[string]any{"1": goodPath}}},
		{"missing-output", map[string]any{"return": float64(0)}},
		{"wrong-output-type", map[string]any{"return": float64(0), "output": "not-a-stream-map"}},
		{"empty-output", map[string]any{"return": float64(0), "output": map[string]any{}}},
		{"unsafe-output-path", map[string]any{"return": float64(0), "output": map[string]any{"1": "/etc/shadow"}}},
		{"foreign-output-url", map[string]any{"return": float64(0), "output": map[string]any{"1": "https://untrusted.example/log"}}},
	}
	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ExecEvidenceFromFinal(&api.Operation{
				StatusCode: api.Success, Metadata: tc.metadata,
			})
			if !errors.Is(err, ErrUnverified) {
				t.Fatalf("malformed native guest outcome treated as verified: %v", err)
			}
		})
	}
	// A successful native envelope with a distinct nonzero guest exit is
	// completed work, NOT command_success=true. A stderr-only native stream
	// does not imply a fabricated stdout log or a missing exit code.
	out, err := ExecEvidenceFromFinal(&api.Operation{
		StatusCode: api.Success,
		Metadata: map[string]any{
			"return": float64(9),
			"output": map[string]any{
				"2": "/1.0/instances/alpha/logs/exec-output/a.stderr",
			},
		},
	})
	if err != nil || out.Exit == nil || *out.Exit != 9 || out.StdoutPath != "" || out.StderrPath == "" {
		t.Fatalf("stderr-only nonzero native guest result lost distinction: %+v err=%v", out, err)
	}
}

// Guest readiness is a tri-state truth: native Stopped gives a verified false;
// native Running without an actual guest probe gives null, not a false fact.
func TestS2GuestReadyJSONTriStateDoesNotInferFromNativeRunning(t *testing.T) {
	running := BoxState{Name: "test", NativeState: "Running"}
	raw, err := json.Marshal(running)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	value, exists := m["guest_ready"]
	if !exists || value != nil {
		t.Fatalf("unprobed running guest readiness must be null: %s", raw)
	}
	notReady := false
	stopped := BoxState{Name: "test", NativeState: "Stopped", GuestReady: &notReady}
	raw, err = json.Marshal(stopped)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["guest_ready"] != false {
		t.Fatalf("stopped guest must report false: %s", raw)
	}
}

func TestS2NativeGuestReadinessOnlyGuestDeclaredAndCurrent(t *testing.T) {
	tests := []struct {
		name   string
		status string
		config map[string]string
		want   *bool
	}{
		{name: "guest has not explicitly marked ready", status: "Running"},
		{name: "user claim does not authenticate readiness", status: "Running",
			config: map[string]string{"user.guest_ready": "true"}},
		{name: "stale ready flag while stopped is false", status: "Stopped",
			config: map[string]string{"volatile.last_state.ready": "true"}, want: boolPtrForS2(false)},
		{name: "native guest ready flag", status: "Running",
			config: map[string]string{"volatile.last_state.ready": "true"}, want: boolPtrForS2(true)},
		{name: "native ready status", status: "Ready", want: boolPtrForS2(true)},
		{name: "paused is not ready", status: "Frozen",
			config: map[string]string{"volatile.last_state.ready": "true"}},
		{name: "stale guest off flag is still unknown", status: "Running",
			config: map[string]string{"volatile.last_state.ready": "false"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nativeGuestReadiness(tc.status, tc.config)
			if (got == nil) != (tc.want == nil) ||
				(got != nil && *got != *tc.want) {
				t.Fatalf("guest ready status=%q config=%v got=%v want=%v", tc.status, tc.config, got, tc.want)
			}
		})
	}
}

func boolPtrForS2(value bool) *bool { return &value }
