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

func TestS4B2RootSocketPeerCredentialsAndOnlyOriginalTuple(t *testing.T) {
	var invoked atomic.Int32
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()),
		func(_ context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
			invoked.Add(1)
			if req.OperationID != strings.Repeat("a", 32) {
				t.Fatal("root request original ID changed")
			}
			if req.PlanSHA256 == "" {
				return NativePeerRootResponse{Status: "REVIEW_ONLY", PlanSHA256: strings.Repeat("b", 64)}, nil
			}
			if req.PlanSHA256 != strings.Repeat("b", 64) {
				t.Fatal("unapproved digest reached root")
			}
			return NativePeerRootResponse{Status: "COMPLETE_VERIFIED",
				Receipt: &NativePeerConntrackJournalReceipt{Status: "COMPLETE_VERIFIED", PlanSHA256: req.PlanSHA256,
					ProbedScopes: 4, ReadyForPeer: false, MutationComplete: true}}, nil
		})
	ctx := t.Context()
	r, err := nativePeerRootCall(ctx, path, NativePeerRootRequest{OperationID: strings.Repeat("a", 32)})
	if err != nil || r.Status != "REVIEW_ONLY" || r.PlanSHA256 != strings.Repeat("b", 64) {
		t.Fatalf("root readonly request rejected: %+v %v", r, err)
	}
	r, err = nativePeerRootCall(ctx, path, NativePeerRootRequest{
		OperationID: strings.Repeat("a", 32), PlanSHA256: strings.Repeat("b", 64)})
	if err != nil || r.Receipt == nil || r.Receipt.ReadyForPeer || !r.Receipt.MutationComplete {
		t.Fatalf("root original request response not bounded: %+v %v", r, err)
	}
	if invoked.Load() != 2 {
		t.Fatal("unexpected number of root worker executions")
	}
	_, err = nativePeerRootCall(ctx, path, NativePeerRootRequest{OperationID: "../escape"})
	if err == nil || invoked.Load() != 2 {
		t.Fatal("unsafe original path traversed root socket")
	}
	_, err = nativePeerRootCall(ctx, path, NativePeerRootRequest{
		OperationID: strings.Repeat("a", 32), PlanSHA256: "wrong"})
	if err == nil || invoked.Load() != 2 {
		t.Fatal("invalid digest reached root")
	}
	t.Log("S4_B2_ROOT_SOCKET_UID_AND_EXACT_ORIGINAL_ONLY=PASS")
}

func TestS4B2RootSocketRejectsWrongUIDAndExtraFields(t *testing.T) {
	var invoked atomic.Int32
	h := func(_ context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
		invoked.Add(1)
		return NativePeerRootResponse{Status: "REVIEW_ONLY"}, nil
	}
	wrongPath, _ := s4b2RootSocketFixture(t, uint32(os.Getuid())+1, h)
	_, err := nativePeerRootCall(t.Context(), wrongPath, NativePeerRootRequest{
		OperationID: strings.Repeat("a", 32)})
	if err == nil || invoked.Load() != 0 {
		t.Fatalf("foreign peer UID admitted: err=%v calls=%d", err, invoked.Load())
	}
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()), h)
	for _, raw := range []string{
		`{"operation_id":"` + strings.Repeat("a", 32) + `","project":"foreign"}`,
		`{"operation_id":"` + strings.Repeat("a", 32) + `"}{"operation_id":"` + strings.Repeat("a", 32) + `"}`,
		`{"operation_id":"` + strings.Repeat("a", 32) + `","plan_sha256":"` + strings.Repeat("B", 64) + `"}`,
	} {
		conn, err := net.Dial("unix", path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Write([]byte(raw + "\n")); err != nil {
			conn.Close()
			t.Fatal(err)
		}
		if err := conn.(*net.UnixConn).CloseWrite(); err != nil {
			conn.Close()
			t.Fatal(err)
		}
		var got NativePeerRootResponse
		if err := json.NewDecoder(conn).Decode(&got); err != nil || got.Status != "DENIED" {
			conn.Close()
			t.Fatalf("unsafe wire payload accepted: %+v %v", got, err)
		}
		conn.Close()
	}
	if invoked.Load() != 0 {
		t.Fatalf("unexpected root action on invalid wire protocol: %d", invoked.Load())
	}
	t.Log("S4_B2_ROOT_SOCKET_FOREIGN_UID_AND_UNBOUNDED_WIRE_REFUSED=PASS")
}

func TestS4B2RootSocketOperatorPinsOneOriginalAndPlan(t *testing.T) {
	original := strings.Repeat("a", 32)
	digest := strings.Repeat("b", 64)
	cfg := NativePeerRootScope{
		Approved: true, OriginalID: original, ApprovedPlanSHA: digest,
	}
	if !nativePeerRootApprovedRequest(cfg, NativePeerRootRequest{OperationID: original}) ||
		!nativePeerRootApprovedRequest(cfg, NativePeerRootRequest{OperationID: original, PlanSHA256: digest}) {
		t.Fatal("root refused exact approved read and original effect")
	}
	for _, req := range []NativePeerRootRequest{
		{OperationID: strings.Repeat("c", 32)},
		{OperationID: strings.Repeat("c", 32), PlanSHA256: digest},
		{OperationID: original, PlanSHA256: strings.Repeat("d", 64)},
	} {
		if nativePeerRootApprovedRequest(cfg, req) {
			t.Fatalf("one root approval reused for unrelated original: %+v", req)
		}
	}
	cfg.Approved = false
	if nativePeerRootApprovedRequest(cfg, NativePeerRootRequest{OperationID: original}) {
		t.Fatal("root operation admitted without local host approval")
	}
	t.Log("S4_B2_ROOT_SOCKET_OPERATOR_EXACT_OPERATION_AND_DIGEST_ONLY=PASS")
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

func TestS5A4RootReadOnlyApprovalScope(t *testing.T) {
	original, digest := strings.Repeat("a", 32), strings.Repeat("b", 64)
	cfg := NativePeerRootScope{Approved: true, OriginalID: original, ApprovedPlanSHA: digest}
	req := NativePeerRootRequest{OperationID: original, ReconcileApproved: true}
	if !nativePeerRootApprovedRequest(cfg, req) {
		t.Fatal("exact approved readonly original refused")
	}
	for _, bad := range []NativePeerRootRequest{
		{OperationID: original, ReconcileApproved: true, PlanSHA256: digest},
		{OperationID: original, ReconcileApproved: true, Reconcile: true},
		{OperationID: strings.Repeat("c", 32), ReconcileApproved: true},
	} {
		if nativePeerRootApprovedRequest(cfg, bad) {
			t.Fatalf("ambiguous or foreign approval accepted: %+v", bad)
		}
	}
	cfg.Approved = false
	if nativePeerRootApprovedRequest(cfg, req) {
		t.Fatal("unapproved recovery accepted")
	}
	var invoked atomic.Int32
	path, _ := s4b2RootSocketFixture(t, uint32(os.Getuid()), func(_ context.Context, r NativePeerRootRequest) (NativePeerRootResponse, error) {
		invoked.Add(1)
		if r != req {
			t.Fatalf("request changed: %+v", r)
		}
		return NativePeerRootResponse{Status: "RECONCILE_VERIFIED"}, nil
	})
	res, err := nativePeerRootCall(t.Context(), path, req)
	if os.Getuid() == 0 {
		if err != nil || res.Status != "RECONCILE_VERIFIED" || invoked.Load() != 1 {
			t.Fatalf("root readonly socket failed: %+v %v", res, err)
		}
	} else if err == nil || invoked.Load() != 0 {
		t.Fatal("nonroot socket was trusted as root witness")
	}
}
