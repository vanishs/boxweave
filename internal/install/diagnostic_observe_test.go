package install

import "testing"

func TestPR177D1ArchitectureClassification(t *testing.T) {
	base := DiagnosticInstallation{Manifest: "VERIFIED", InstalledIdentity: "VERIFIED", Runtime: "VERIFIED", RootJournal: "ABSENT", RootProof: "NOT_APPLICABLE", GuardUnit: "ABSENT", LifecycleUnit: "ABSENT", Candidate: "LEGACY", ScopedArtifacts: "ABSENT", Dependencies: "LEGACY_ABSENT"}
	if got := diagnosticClassify(base, false); got.Mode != "UNSUPPORTED_LEGACY" || got.Coverage != "PARTIAL" || got.Reason != "LEGACY_INSTALLATION_UNSUPPORTED" {
		t.Fatal(got)
	}
	if got := diagnosticClassify(base, true); got.Mode != "CONFLICT" {
		t.Fatal(got)
	}
	x := base
	x.ScopedArtifacts = "PARTIAL"
	if got := diagnosticClassify(x, false); got.Mode != "CONFLICT" {
		t.Fatal(got)
	}
	x = base
	x.RootJournal = "PRESENT"
	if got := diagnosticClassify(x, false); got.Mode != "CONFLICT" {
		t.Fatal(got)
	}
	x = base
	x.Runtime = "UNVERIFIED"
	if got := diagnosticClassify(x, false); got.Mode != "UNVERIFIED" {
		t.Fatal(got)
	}
	x = base
	x.Candidate = "SCOPED_V2"
	x.ScopedArtifacts = "PRESENT"
	x.RootJournal = "PRESENT"
	x.RootProof = "VERIFIED"
	x.StaticContract = "VERIFIED"
	x.GuardUnit = "ACTIVE"
	x.LifecycleUnit = "TRANSITIONING"
	x.Dependencies = "OBSERVED"
	if got := diagnosticClassify(x, true); got.Mode != "SCOPED_V2" || got.Coverage != "MODE_PROVEN" {
		t.Fatal(got)
	}
	x.StaticContract = "UNVERIFIED"
	if got := diagnosticClassify(x, true); got.Mode != "UNVERIFIED" {
		t.Fatal(got)
	}
	x.StaticContract = "VERIFIED"
	x.RootProof = "ROOT_AUTHORITY_REQUIRED"
	if got := diagnosticClassify(x, true); got.Mode != "UNVERIFIED" {
		t.Fatal(got)
	}
	x.RootProof = "VERIFIED"
	x.RootJournal = "ABSENT"
	if got := diagnosticClassify(x, true); got.Mode != "CONFLICT" {
		t.Fatal(got)
	}
	x.RootJournal = "PRESENT"
	x.Manifest = "UNVERIFIED"
	if got := diagnosticClassify(x, true); got.Mode != "UNVERIFIED" {
		t.Fatal(got)
	}
}
func TestPR177D1RuntimeScopeParsing(t *testing.T) {
	for _, tc := range []struct {
		argv          string
		valid, scoped bool
	}{
		{"/usr/local/bin/boxweave\x00serve\x00-project\x00user-1001\x00", true, false},
		{"/usr/local/bin/boxweave\x00serve\x00-project=user-1001\x00-scoped-peer-fresh\x00", true, true},
		{"/usr/local/bin/boxweave\x00serve\x00-project=user-1001\x00-scoped-peer-fresh=false\x00", true, false},
		{"/usr/local/bin/boxweave\x00serve\x00-project=user-1002\x00", false, false},
		{"/usr/local/bin/boxweave\x00serve\x00-project=user-1001\x00-db=/elsewhere\x00", false, false},
		{"/usr/local/bin/boxweave\x00serve\x00-project=user-1001\x00-scoped-peer-fresh\x00-scoped-peer-fresh=false\x00", false, false},
		{"/usr/local/bin/boxweave\x00recover\x00-project=user-1001\x00", false, false},
	} {
		s, v := diagnosticRuntimeArgs([]byte(tc.argv), "user-1001", "/var/lib/incus/unix.socket.user", "/var/lib/boxweave/ledger.sqlite", false)
		if v != tc.valid || v && s != tc.scoped {
			t.Fatal(tc.argv, s, v)
		}
	}
}

func TestP3R2ScopedRuntimeDefault(t *testing.T) {
	for _, tc := range []struct {
		flags         string
		valid, scoped bool
	}{
		{"", true, true},
		{"-scoped-peer-fresh\x00", true, true},
		{"-scoped-peer-fresh=false\x00", true, false},
		{"-listen=127.0.0.1:18650\x00", true, true},
		{"-listen=127.0.0.1:9999\x00", false, false},
		{"-service-user\x00boxweave\x00", false, false},
	} {
		raw := []byte("/usr/local/bin/boxweave\x00serve\x00-project=user-1001\x00" + tc.flags)
		scoped, valid := diagnosticRuntimeArgs(raw, "user-1001", "/var/lib/incus/unix.socket.user", "/var/lib/boxweave/ledger.sqlite", true)
		if valid != tc.valid || valid && scoped != tc.scoped {
			t.Fatal(tc, scoped, valid)
		}
	}
}
