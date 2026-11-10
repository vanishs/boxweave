package native

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestS1CustomDirPoolOnHostRootCannotProveBacking(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "pool")
	if err := os.Mkdir(custom, 0700); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(custom)
	if err != nil {
		t.Fatal(err)
	}
	_, separate, err := inspectDirPoolFilesystem("user-storage", custom, "/")
	if !errors.Is(err, errNativeBackingUnverified) || separate {
		t.Fatalf("custom storage on host root must be reported unverified: separate=%v err=%v", separate, err)
	}
	after, err := os.Stat(custom)
	if err != nil || !after.IsDir() || !after.ModTime().Equal(before.ModTime()) {
		t.Fatal("storage preflight modified the target directory")
	}
}

func TestS1DefaultRootDirPoolSpaceIsReadFromSelectedFilesystem(t *testing.T) {
	free, separate, err := inspectDirPoolFilesystem("default", t.TempDir(), "/")
	if err != nil && !errors.Is(err, errNativeSpaceLow) {
		t.Fatal(err)
	}
	if separate {
		t.Fatal("ordinary root-backed default dir must not claim separate mount")
	}
	if free == 0 {
		t.Fatal("missing actual local statfs available bytes")
	}
}

func TestS1DirPoolMissingSourceAndSymlinkAreRejected(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	_, _, err := inspectDirPoolFilesystem("another", missing, "/")
	if err == nil || !strings.Contains(err.Error(), "directory unavailable") {
		t.Fatalf("missing selected storage source must be rejected: %v", err)
	}
	link := filepath.Join(root, "alias")
	if err := os.Symlink("/", link); err != nil {
		t.Fatal(err)
	}
	_, _, err = inspectDirPoolFilesystem("another", link, "/")
	if err == nil || !strings.Contains(err.Error(), "directory unavailable") {
		t.Fatalf("symlinked pool path must be rejected: %v", err)
	}
}

func TestS1PoolNameMayNotEscapeCanonicalPath(t *testing.T) {
	for _, name := range []string{"", ".", "..", "../default", "a/b", "a\\b"} {
		if _, _, err := localDirPoolSpace(name); err == nil {
			t.Fatalf("unsafe pool name accepted: %q", name)
		}
	}
}


// The longest effective mount wins. A project-quota mount hidden by a
// non-quota bind mount is NOT proof that the selected native pool is safe.
func TestS2QuotaMountSourceMustBeRealEnabledExt4OrXFS(t *testing.T) {
	path := "/var/lib/incus/storage-pools/bw-eligible"
	cases := []struct {
		name string
		info string
		want bool
	}{
		{"ext4-enabled", "44 1 7:1 / /var/lib/incus/storage-pools/bw-eligible rw,prjquota - ext4 /dev/loop2 rw,prjquota", true},
		{"xfs-enabled", "44 1 7:1 / /var/lib/incus/storage-pools/bw-eligible rw,pquota - xfs /dev/loop2 rw", true},
		{"no-project-quota", "44 1 7:1 / /var/lib/incus/storage-pools/bw-eligible rw,relatime - ext4 /dev/loop2 rw", false},
		{"unsupported-driver", "44 1 7:1 / /var/lib/incus/storage-pools/bw-eligible rw,prjquota - btrfs /dev/loop2 rw,prjquota", false},
		{"parent-quota-hidden", "43 1 7:1 / /var/lib/incus/storage-pools rw,prjquota - ext4 /dev/loop2 rw,prjquota\n44 43 8:1 / /var/lib/incus/storage-pools/bw-eligible rw - ext4 /dev/loop3 rw", false},
		{"foreign-mount", "44 1 7:1 / /var/lib/incus/storage-pools/other rw,prjquota - ext4 /dev/loop2 rw,prjquota", false},
		{"invalid-mountinfo", "garbage", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dirProjectQuotaMountReady(path, tc.info); got != tc.want {
				t.Fatalf("quota mount proof got %v want %v", got, tc.want)
			}
		})
	}
	if dirProjectQuotaMountReady("/var/lib/incus/storage-pools/bad name", cases[0].info) {
		t.Fatal("ambiguous mount-point encoding accepted")
	}
}
