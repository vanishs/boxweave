package install

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func s4b2RootSocketFixture(t *testing.T, allowedUID uint32, handler nativePeerRootHandler) (string, context.CancelFunc) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "root-peer.sock")
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- nativePeerRootServeListener(ctx, ln, allowedUID, handler) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("root Unix listener stop: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("root listener did not terminate after cancel")
		}
	})
	return path, cancel
}

// Exercise the actual bounded listener without granting an unprivileged
// test fixture the root identity that the production client requires.
func scopedSocketTestCall(t *testing.T, path, raw string) NativePeerRootResponse {
	t.Helper()
	conn, err := net.DialTimeout("unix", path, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err = conn.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err = conn.(*net.UnixConn).CloseWrite(); err != nil {
		t.Fatal(err)
	}
	var out NativePeerRootResponse
	if err = json.NewDecoder(conn).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestS4B2RootSocketPeerCredentialsAndOnlyOriginalTuple(t *testing.T) {
	var invoked atomic.Int32
	id := strings.Repeat("a", 32)
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()),
		func(_ context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
			invoked.Add(1)
			if req.OperationID != id {
				t.Error("root request original ID changed")
			}
			status := "DISPATCH"
			if req.ReconcileApproved {
				status = "RECOVERY"
			}
			return NativePeerRootResponse{Status: status}, nil
		})
	for _, recovery := range []bool{false, true} {
		raw, err := json.Marshal(NativePeerRootRequest{OperationID: id, ReconcileApproved: recovery})
		if err != nil {
			t.Fatal(err)
		}
		want := "DISPATCH"
		if recovery {
			want = "RECOVERY"
		}
		if got := scopedSocketTestCall(t, path, string(raw)); got.Status != want {
			t.Fatal(got)
		}
	}
	if invoked.Load() != 2 {
		t.Fatal("unexpected number of root worker executions")
	}
	t.Log("S4_B2_ROOT_SOCKET_UID_AND_EXACT_ORIGINAL_ONLY=PASS")
}

func TestS4B2RootSocketRejectsWrongUIDAndExtraFields(t *testing.T) {
	var invoked atomic.Int32
	h := func(_ context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
		invoked.Add(1)
		return NativePeerRootResponse{Status: "VERIFIED"}, nil
	}
	id := strings.Repeat("a", 32)
	base := `{"operation_id":"` + id + `"}`
	wrongPath, _ := s4b2RootSocketFixture(t, uint32(os.Getuid())+1, h)
	// Credentials are checked before reading. Do not race a request write
	// against the listener closing a refused connection.
	wrong, err := net.DialTimeout("unix", wrongPath, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_ = wrong.SetReadDeadline(time.Now().Add(3 * time.Second))
	var denied NativePeerRootResponse
	err = json.NewDecoder(wrong).Decode(&denied)
	_ = wrong.Close()
	if err != nil || denied.Status != "DENIED" {
		t.Fatal("foreign peer UID admitted", denied, err)
	}
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()), h)
	for _, raw := range []string{
		`{"operation_id":"../escape"}`,
		base + base,
		`{"operation_id":"` + id + `","project":"foreign"}`,
		`{"operation_id":"` + id + `","plan_sha256":"` + strings.Repeat("b", 64) + `"}`,
		`{"operation_id":"` + id + `","reconcile":true}`,
		`{"operation_id":"` + id + `","retirement_review":true}`,
		`{"operation_id":"` + id + `","retirement_preview":true}`,
	} {
		if got := scopedSocketTestCall(t, path, raw); got.Status != "DENIED" {
			t.Fatalf("unsafe wire payload accepted: %s %+v", raw, got)
		}
	}
	if invoked.Load() != 0 {
		t.Fatal("unexpected root action on invalid wire protocol", invoked.Load())
	}
	t.Log("S4_B2_ROOT_SOCKET_FOREIGN_UID_AND_UNBOUNDED_WIRE_REFUSED=PASS")
}

// A killed root helper can strand its Unix socket. Restart is safe only
// when the exact socket is dead, has unchanged ownership/mode and was never
// replaced; the immutable UNKNOWN journal remains untouched.
func TestS4B2RootSocketCrashRestartOnlyExactStaleInode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper.sock")
	uid, gid := uint32(os.Getuid()), uint32(os.Getgid())
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err != nil {
		t.Fatalf("fresh approved helper refused: %v", err)
	}
	listen := func() *net.UnixListener {
		t.Helper()
		ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
		if err != nil {
			t.Fatal(err)
		}
		ln.SetUnlinkOnClose(false)
		if err := os.Chmod(path, 0660); err != nil {
			t.Fatal(err)
		}
		return ln
	}
	ln := listen()
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err == nil {
		t.Fatal("live listener was displaced")
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err != nil {
		t.Fatalf("stale listener not recoverable: %v", err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("stale inode not removed: %v", err)
	}
	ln = listen()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid+1); err == nil {
		t.Fatal("foreign group socket removed")
	}
	if _, err := os.Lstat(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err == nil {
		t.Fatal("wrong-mode socket removed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("foreign"), 0660); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err == nil {
		t.Fatal("foreign regular file removed")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("foreign", path); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerRootReclaimStaleSocket(path, uid, gid); err == nil {
		t.Fatal("symlink was replaced")
	}
	if dest, err := os.Readlink(path); err != nil || dest != "foreign" {
		t.Fatalf("foreign symlink altered: %q %v", dest, err)
	}
	t.Log("S4_B2_ROOT_SOCKET_CRASH_RESTART_STALE_ONLY_LIVE_FOREIGN_DENIED=PASS")
}

func TestScopedRootClientRequiresKernelRootPeer(t *testing.T) {
	var invoked atomic.Int32
	req := NativePeerRootRequest{OperationID: strings.Repeat("a", 32), ReconcileApproved: true}
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()), func(_ context.Context, got NativePeerRootRequest) (NativePeerRootResponse, error) {
		invoked.Add(1)
		if got != req {
			t.Error("request changed", got)
		}
		return NativePeerRootResponse{Status: "COMPLETE_VERIFIED"}, nil
	})
	res, err := nativeScopedRootCallAt(t.Context(), path, req)
	if os.Getuid() == 0 {
		if err != nil || res.Status != "COMPLETE_VERIFIED" || invoked.Load() != 1 {
			t.Fatalf("root readonly socket failed: %+v %v", res, err)
		}
	} else if err == nil || invoked.Load() != 0 {
		t.Fatal("nonroot socket was trusted as root witness")
	}
}
