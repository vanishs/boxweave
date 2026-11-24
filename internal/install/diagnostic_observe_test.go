package install

import "testing"

func TestPR177D1ArchitectureClassification(t *testing.T) {
	base := DiagnosticInstallation{Manifest: "VERIFIED", InstalledIdentity: "VERIFIED", Runtime: "VERIFIED", RootJournal: "ABSENT", RootProof: "NOT_APPLICABLE", GuardUnit: "ABSENT", LifecycleUnit: "ABSENT", Candidate: "LEGACY", ScopedArtifacts: "ABSENT", Dependencies: "LEGACY_ABSENT"}
	if got := diagnosticClassify(base, false); got.Mode != "LEGACY" || got.Coverage != "MODE_PROVEN" {
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
	x.GuardUnit = "ACTIVE"
	x.LifecycleUnit = "TRANSITIONING"
	x.Dependencies = "OBSERVED"
	if got := diagnosticClassify(x, true); got.Mode != "SCOPED_V2" || got.Coverage != "MODE_PROVEN" {
		t.Fatal(got)
	}
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
		s, v := diagnosticRuntimeArgs([]byte(tc.argv), "user-1001", "/var/lib/incus/unix.socket.user", "/var/lib/boxweave/ledger.sqlite")
		if v != tc.valid || v && s != tc.scoped {
			t.Fatal(tc.argv, s, v)
		}
	}
}
