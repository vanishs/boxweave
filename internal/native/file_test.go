package native

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	for name, want := range map[string]string{"normal": "file", "link": "symlink"} {
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

func TestS3SFTPReadBudgetCannotDecodeUnboundedDirectory(t *testing.T) {
	b := &sftpReadBudget{reader: strings.NewReader("123456789"), remaining: 4}
	buf := make([]byte, 12)
	n, err := b.Read(buf)
	if n != 4 || err != nil || string(buf[:n]) != "1234" || b.exhausted.Load() {
		t.Fatalf("first bounded SFTP packet read: %d %v", n, err)
	}
	n, err = b.Read(buf)
	if n != 0 || !errors.Is(err, ErrNativeUnsupported) || !b.exhausted.Load() {
		t.Fatalf("wire limit must fail closed without allocating remainder: %d %v", n, err)
	}
	if _, _, err := newBoundedSFTPClient(nil, MaxGuestFileListWireBytes); !errors.Is(err, ErrNativeUnsupported) {
		t.Fatalf("nil underlying connection must fail closed: %v", err)
	}
}
