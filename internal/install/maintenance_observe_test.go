package install

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestH1AMaintenanceMetadata(t *testing.T) {
	d := t.TempDir()
	uid := uint32(os.Geteuid())
	p := filepath.Join(d, "regular")
	if err := os.WriteFile(p, []byte("private fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	if v := maintenanceArtifact(p, uid, unix.S_IFREG, 0600); v.State != "PRESENT" || v.Reason != "METADATA_ONLY_NOT_OWNERSHIP" {
		t.Fatal(v)
	}
	if v := maintenanceArtifact(filepath.Join(d, "absent"), uid, unix.S_IFREG, 0600); v.State != "ABSENT" {
		t.Fatal(v)
	}
	sym := filepath.Join(d, "symlink")
	if err := os.Symlink(p, sym); err != nil {
		t.Fatal(err)
	}
	if v := maintenanceArtifact(sym, uid, unix.S_IFREG, 0600); v.State == "PRESENT" {
		t.Fatal(v)
	}
	hard := filepath.Join(d, "hardlink")
	if err := os.Link(p, hard); err != nil {
		t.Fatal(err)
	}
	if v := maintenanceArtifact(p, uid, unix.S_IFREG, 0600); v.State == "PRESENT" {
		t.Fatal(v)
	}
	if err := os.Remove(hard); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(p, 0666); err != nil {
		t.Fatal(err)
	}
	if v := maintenanceArtifact(p, uid, unix.S_IFREG, 0600); v.State == "PRESENT" {
		t.Fatal(v)
	}
	parent := filepath.Join(d, "parent-link")
	if err := os.Symlink(d, parent); err != nil {
		t.Fatal(err)
	}
	if v := maintenanceArtifact(filepath.Join(parent, "regular"), uid, unix.S_IFREG, 0600); v.State == "PRESENT" {
		t.Fatal(v)
	}
}
func TestH1AMaintenanceAdapter(t *testing.T) {
	m := Manifest{Version: "1.0.0-rc.13", BuildCommit: "6c995687ae38b5cd0e23c464dee84f40a33a01ef", BinarySHA256: "38307cc8a6b9e39830e489873c31b1e0b99a67e4a3d8f06ee3be58038d6da68e"}
	if got := maintenanceAdapter(m, "", "other", "other"); got != "PINNED_RC13" {
		t.Fatal(got)
	}
	m.BuildCommit = "changed"
	if got := maintenanceAdapter(m, "", "other", "other"); got != "UNSUPPORTED" {
		t.Fatal(got)
	}
	if got := maintenanceAdapter(m, m.BinarySHA256, m.Version, m.BuildCommit); got != "EXACT_RUNNING_OBSERVER_BINARY" {
		t.Fatal(got)
	}
	if got := maintenanceAdapter(m, m.BinarySHA256, "different", m.BuildCommit); got != "UNSUPPORTED" {
		t.Fatal(got)
	}
}
func TestH1AMaintenanceConfigStates(t *testing.T) {
	if maintenanceConfig("after", "before", "after") != "AFTER" || maintenanceConfig("before", "before", "after") != "BEFORE" || maintenanceConfig("different", "before", "after") != "DRIFT" {
		t.Fatal("preimage classification")
	}
}
