package native

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
)

func TestS3RealIncusSFTPFileNegatives(t *testing.T) {
	if os.Getenv("S3_NATIVE_FILE_LAB") == "" {
		t.Skip("one-use signed Incus 7.6 runner only")
	}
	if os.Getenv("S3_NATIVE_FILE_LAB") != "approved-ephemeral-vm" ||
		os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Geteuid() != 0 {
		t.Fatal("refuse shared or unapproved Incus host")
	}
	const guest = "s3-file-native-ci-1115"
	const dir = "/tmp/s3-file-native"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	base, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := base.UseProject("default")
	inst, _, err := srv.GetInstance(guest)
	if err != nil || inst.Config["volatile.uuid"] == "" {
		t.Fatalf("disposable guest UUID unavailable: %v", err)
	}
	t.Logf("S3_NATIVE_GUEST_UUID=%s", inst.Config["volatile.uuid"])
	c, err := srv.GetInstanceFileSFTP(guest)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	original := []byte("S3_ORIGINAL_VICTIM_BYTES_KEEP")
	getBytes := func(p string) []byte {
		t.Helper()
		f, err := c.Open(p)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, MaxGuestFileBytes+1))
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	victim := dir + "/victim"
	if err := verifyGuestComponents(c, victim, "file", false); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(getBytes(victim), original) {
		t.Fatal("native original victim mismatch")
	}
	f, err := c.Open(victim)
	if err != nil {
		t.Fatal(err)
	}
	page := make([]byte, 9)
	n, rerr := f.ReadAt(page, 4)
	_ = f.Close()
	if rerr != nil && !errors.Is(rerr, io.EOF) || n != 9 ||
		!bytes.Equal(page[:n], original[4:13]) {
		t.Fatalf("SFTP bounded offset read differs from original: %d %v", n, rerr)
	}
	t.Log("S3_NATIVE_SFTP_BOUNDED_OFFSET_READ=PASS")
	infos, err := c.ReadDirContext(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]string{}
	for _, info := range infos {
		kinds[info.Name()] = guestFileType(info)
	}
	for name, kind := range map[string]string{
		"victim": "file", "symlink": "symlink",
		"hardlink": "file", "fifo": "special",
	} {
		if kinds[name] != kind {
			t.Fatalf("SFTP READDIR type %s=%s want %s", name, kinds[name], kind)
		}
	}
	t.Log("S3_NATIVE_SFTP_LINK_AND_SPECIAL_METADATA=PASS")
	for _, p := range []string{
		"/proc/self/fd/1", "/sys/kernel/security", "/dev/null",
		"/tmp/../etc/passwd", "/tmp//x", "/tmp/dir/",
	} {
		if SafeGuestFilePath(p) {
			t.Fatalf("accepted unsafe Guest path %q", p)
		}
	}
	for _, p := range []string{dir + "/symlink", dir + "/hardlink"} {
		if err := verifyGuestComponents(c, p, "file", true); !errors.Is(err, ErrGuestFileExists) {
			t.Fatalf("alias not recognized as existing %s: %v", p, err)
		}
		out, err := c.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
		if err == nil {
			_ = out.Close()
			t.Fatalf("native O_EXCL followed alias: %s", p)
		}
		if !bytes.Equal(getBytes(victim), original) {
			t.Fatalf("existing alias overwrote victim: %s", p)
		}
	}
	if err := verifyGuestComponents(c, "/tmp/s3-file-native-link/new", "file", true); !errors.Is(err, ErrNativeUnsupported) {
		t.Fatalf("symlink parent accepted: %v", err)
	}
	if err := verifyGuestComponents(c, dir+"/fifo", "file", false); !errors.Is(err, ErrNativeUnsupported) {
		t.Fatalf("special FIFO accepted: %v", err)
	}
	t.Log("S3_NATIVE_SFTP_SYMLINK_HARDLINK_FIFO_NO_EFFECT=PASS")
	const writers = 12
	race := path.Join(dir, "race-new")
	type outcome struct {
		id  int
		ok  bool
		err error
	}
	results := make(chan outcome, writers)
	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			other, err := srv.GetInstanceFileSFTP(guest)
			if err != nil {
				results <- outcome{id: id, err: err}
				return
			}
			defer other.Close()
			if err := verifyGuestComponents(other, race, "file", true); err != nil {
				if errors.Is(err, ErrGuestFileExists) {
					results <- outcome{id: id}
				} else {
					results <- outcome{id: id, err: err}
				}
				return
			}
			handle, err := other.OpenFile(race, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
			if err != nil {
				results <- outcome{id: id}
				return
			}
			payload := []byte(fmt.Sprintf("S3_RACE_WINNER_%02d", id))
			n, writeErr := handle.Write(payload)
			closeErr := handle.Close()
			if writeErr != nil || closeErr != nil || n != len(payload) {
				results <- outcome{id: id, err: fmt.Errorf("write/close %v %v %d", writeErr, closeErr, n)}
				return
			}
			results <- outcome{id: id, ok: true}
		}(i)
	}
	wg.Wait()
	close(results)
	successes, winner := 0, -1
	for v := range results {
		if v.err != nil {
			t.Fatalf("writer %d failed unexpectedly: %v", v.id, v.err)
		}
		if v.ok {
			successes++
			winner = v.id
		}
	}
	if successes != 1 {
		t.Fatalf("O_EXCL allowed %d successful writers", successes)
	}
	if !bytes.Equal(getBytes(race), []byte(fmt.Sprintf("S3_RACE_WINNER_%02d", winner))) {
		t.Fatal("SFTP concurrent loser overwrote winner bytes")
	}
	if !bytes.Equal(getBytes(victim), original) {
		t.Fatal("original victim changed during native file race")
	}
	t.Log("S3_NATIVE_SFTP_12_WRITERS_SINGLE_WINNER_NO_FOREIGN_EFFECT=PASS")
	if err := verifyGuestComponents(c, race, "file", true); !errors.Is(err, ErrGuestFileExists) {
		t.Fatalf("repeat native create must refuse existing winner: %v", err)
	}
	if _, err := c.Lstat(dir + "/missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing SFTP target must be recognizable as os.ErrNotExist: %v", err)
	}
	t.Log("S3_NATIVE_SFTP_EXISTING_AND_MISSING_TARGET_TYPES=PASS")
}
