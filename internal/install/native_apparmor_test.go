package install

import (
	"strings"
	"testing"
)

func TestNativeUserNamespaceMap(t *testing.T) {
	cases := []struct {
		name, value string
		userNS      bool
		err         bool
	}{
		{"initial_host", "         0          0 4294967295\n", false, false},
		{"contiguous_l1", "0 1000000 1000000000\n", true, false},
		{"nested_l2", "0 1000000 999000000\n", true, false},
		{"split_service_user", "0 1000000 1001\n1001 1001 1\n1002 1001002 998998998\n", true, false},
		{"malformed", "oops\n", false, true},
		{"empty", "", false, true},
		{"zero", "0 0 0\n", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := nativeUserNSFromMap(tc.value)
			if (err != nil) != tc.err || (err == nil && got != tc.userNS) {
				t.Fatalf("userNS=%v err=%v, expected userNS=%v err=%v", got, err, tc.userNS, tc.err)
			}
		})
	}
}

func TestNativeAppArmorNestedHostPreflightClasses(t *testing.T) {
	cases := []struct {
		name  string
		state nativeAppArmorHostState
		code  string
	}{
		{"unconfined_host", nativeAppArmorHostState{Available: true, Profile: "unconfined"}, ""},
		{"supported_l1", nativeAppArmorHostState{Available: true, Profile: "unconfined", UserNS: true, Stacked: true}, ""},
		{"unsupported_l2_restricted", nativeAppArmorHostState{Available: true, Profile: "incus-user-1001_rc-install-lab-1113_</var/lib/incus> (enforce)", UserNS: true, Stacked: true}, "NATIVE_APPARMOR_CONFINED"},
		{"unsupported_l2_default", nativeAppArmorHostState{Available: true, Profile: "incus-bw-aa-inner-stack-1113_</var/lib/incus> (enforce)", UserNS: true, Stacked: true}, "NATIVE_APPARMOR_CONFINED"},
		{"unsupported_forced_unconfined", nativeAppArmorHostState{Available: true, Profile: "unconfined", UserNS: true}, "NATIVE_APPARMOR_NO_STACKING"},
		{"unknown_label", nativeAppArmorHostState{Available: true, UserNS: true, Stacked: true}, "NATIVE_APPARMOR_UNVERIFIED"},
		{"apparmor_absent", nativeAppArmorHostState{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyNativeAppArmorHost(tc.state)
			if tc.code == "" && err != nil || tc.code != "" && (err == nil || !strings.Contains(err.Error(), tc.code)) {
				t.Fatalf("got %v, expected code %q", err, tc.code)
			}
		})
	}
}
