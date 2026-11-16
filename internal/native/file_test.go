package native

import (
	"os"
	"path/filepath"
	"testing"
)

func TestS3CanonicalGuestFilePathDenyMagicAndTraversal(t *testing.T) {
	allowed := []string{
		"/home/ubuntu/main.go", "/tmp/result.log", "/root/work/a.txt",
		"/opt/repository/src/thing", "/var/tmp/build.output",
	}
	for _, p := range allowed {
		if !SafeGuestFilePath(p) {
			t.Errorf("refused canonical Guest path %q", p)
		}
	}
	denied := []string{
		"", "/", "../host", "/tmp/../etc/shadow", "/tmp/./thing",
		"/tmp//thing", "/tmp/", "//tmp/a", "/proc/self/fd/3",
		"/sys/kernel/security", "/dev/fd/1", "/run/host",
		"/var/run/sock", "/tmp/a\x00b", "/tmp/a\nb", "/tmp/a\\b",
	}
	for _, p := range denied {
		if SafeGuestFilePath(p) {
			t.Errorf("accepted unsafe Guest path %q", p)
		}
	}
}

func TestS3NativeFileTypeClassificationDoesNotFollowLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "normal"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("normal", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"normal":"file", "link":"symlink"} {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if got := guestFileType(info); got != want {
			t.Fatalf("lstat %s: %s != %s", name, got, want)
		}
	}
	if info, err := os.Lstat(dir); err != nil || guestFileType(info) != "directory" {
		t.Fatalf("directory changed: %+v %v", info, err)
	}
}
