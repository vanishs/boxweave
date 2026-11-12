package native

import "testing"

func TestS25OfficialIncusMultiUserProfileConfigOnly(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]string
		allow  bool
	}{
		{"empty_default", nil, true},
		{"official_user_idmap", map[string]string{"raw.idmap": "uid 1001 1001\ngid 990 990"}, true},
		{"wrong_uid", map[string]string{"raw.idmap": "uid 1002 1002\ngid 990 990"}, false},
		{"wrong_gid", map[string]string{"raw.idmap": "uid 1001 1001\ngid 989 989"}, false},
		{"additional_gid", map[string]string{"raw.idmap": "uid 1001 1001\ngid 990 990\ngid 0 0"}, false},
		{"raw_lxc", map[string]string{"raw.lxc": "lxc.apparmor.profile=unconfined"}, false},
		{"default_nesting", map[string]string{"security.nesting": "true"}, false},
		{"raw_idmap_and_raw_lxc", map[string]string{"raw.idmap": "uid 1001 1001\ngid 990 990", "raw.lxc": "foreign"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := SafePerUserProfileConfig(tc.config, "1001", "990"); got != tc.allow {
				t.Fatalf("profile config allowed=%v want %v", got, tc.allow)
			}
		})
	}
	if SafePerUserProfileConfig(nil, "0", "990") || SafePerUserProfileConfig(nil, "not-a-uid", "990") {
		t.Fatal("privileged or ambiguous service identities accepted")
	}
}
