package install

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestP3R4ScopedUnitDirectoryUmask(t *testing.T) {
	if os.Getenv("BOXWEAVE_R4_UNIT_CHILD") == "1" {
		dir := os.Getenv("BOXWEAVE_R4_UNIT_DIR")
		syscall.Umask(0077)
		if err := scopedWriteExclusive(filepath.Join(dir, "new", "unit.conf"), "[Unit]\\n"); err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(filepath.Join(dir, "new"))
		if err != nil || st.Mode().Perm() != 0755 {
			t.Fatal(st, err)
		}
		st, err = os.Stat(filepath.Join(dir, "new", "unit.conf"))
		if err != nil || st.Mode().Perm() != 0600 {
			t.Fatal(st, err)
		}
		if err = os.Mkdir(filepath.Join(dir, "existing"), 0700); err != nil {
			t.Fatal(err)
		}
		if err = scopedWriteExclusive(filepath.Join(dir, "existing", "unit.conf"), "[Unit]\\n"); err != nil {
			t.Fatal(err)
		}
		st, err = os.Stat(filepath.Join(dir, "existing"))
		if err != nil || st.Mode().Perm() != 0700 {
			t.Fatal("existing directory changed", st, err)
		}
		if err = scopedWriteExclusive(filepath.Join(dir, "new", "unit.conf"), "changed"); err == nil {
			t.Fatal("exclusive write accepted")
		}
		b, err := os.ReadFile(filepath.Join(dir, "new", "unit.conf"))
		if err != nil || string(b) != "[Unit]\\n" {
			t.Fatal("original modified", err)
		}
		return
	}
	if os.Geteuid() != 0 {
		t.Skip("root-owned fixture only; no host unit writes")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestP3R4ScopedUnitDirectoryUmask$")
	cmd.Env = append(os.Environ(), "BOXWEAVE_R4_UNIT_CHILD=1", "BOXWEAVE_R4_UNIT_DIR="+t.TempDir())
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v", b, err)
	}
}
