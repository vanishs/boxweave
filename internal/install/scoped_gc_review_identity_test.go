package install

import (
	"strings"
	"testing"
)

func TestScopedGCReviewManifestExact(t *testing.T) {
	p := Plan{User: "boxweave", StateDir: "/var/lib/boxweave", BinaryPath: "/usr/local/bin/boxweave", Project: "user-1001", Socket: "/var/lib/incus/unix.socket.user"}
	m := Manifest{Schema: manifestSchema, ScopedPeerMode: nativePeerFreshMode, ServiceUser: p.User, StateDir: p.StateDir, BinaryPath: p.BinaryPath, Project: p.Project, Socket: p.Socket, Version: "test", BuildCommit: strings.Repeat("a", 40), BinarySHA256: strings.Repeat("b", 64), UnitSHA256: strings.Repeat("c", 64)}
	if !matchScopedGCReviewManifest(p, m, m) {
		t.Fatal("exact identity refused")
	}
	for name, mutate := range map[string]func(*Manifest){
		"mode":       func(x *Manifest) { x.ScopedPeerMode = "" },
		"schema":     func(x *Manifest) { x.Schema++ },
		"project":    func(x *Manifest) { x.Project = "user-1002" },
		"socket":     func(x *Manifest) { x.Socket = "/wrong" },
		"state":      func(x *Manifest) { x.StateDir = "/wrong" },
		"user":       func(x *Manifest) { x.ServiceUser = "other" },
		"binary":     func(x *Manifest) { x.BinarySHA256 = strings.Repeat("d", 64) },
		"unit":       func(x *Manifest) { x.UnitSHA256 = strings.Repeat("d", 64) },
		"commit":     func(x *Manifest) { x.BuildCommit = strings.Repeat("d", 40) },
		"restart":    func(x *Manifest) { x.RestartRequired = true },
		"retirement": func(x *Manifest) { x.RetirementUnitSHA256 = strings.Repeat("d", 64) },
	} {
		t.Run(name, func(t *testing.T) {
			changed := m
			mutate(&changed)
			if matchScopedGCReviewManifest(p, m, changed) {
				t.Fatal("changed identity accepted")
			}
		})
	}
	changed := m
	changed.ScopedPeerMode = ""
	if matchScopedGCReviewManifest(p, changed, changed) {
		t.Fatal("legacy identity accepted")
	}
	changed = m
	changed.ServiceActive = true
	if !matchScopedGCReviewManifest(p, m, changed) {
		t.Fatal("observed service state incorrectly bound to static manifest")
	}
}
