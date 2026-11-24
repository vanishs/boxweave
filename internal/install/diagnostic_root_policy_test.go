package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPR177D2PrerequisiteExecutableClassification(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test-executable")
	uid := uint32(os.Geteuid())
	if got := diagnosticExecutableState(p, uid); got != "MISSING" {
		t.Fatal(got)
	}
	if e := os.WriteFile(p, []byte("fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	if got := diagnosticExecutableState(p, uid); got != "UNVERIFIED" {
		t.Fatal(got)
	}
	if e := os.Chmod(p, 0700); e != nil {
		t.Fatal(e)
	}
	if got := diagnosticExecutableState(p, uid); got != "PRESENT" {
		t.Fatal(got)
	}
	if e := os.Chmod(p, 0777); e != nil {
		t.Fatal(e)
	}
	if got := diagnosticExecutableState(p, uid); got != "UNVERIFIED" {
		t.Fatal(got)
	}
}
