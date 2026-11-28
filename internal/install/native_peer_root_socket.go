package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

const nativePeerRootSocketPath = "/run/boxweave-native-peer-root.sock"
const nativeRetirementSocketPath = "/run/boxweave-ipv4-retirement.sock"

// The Scoped lifecycle wire accepts only an original and an explicit
// read-only recovery selector. Legacy plan, ACL and retirement fields are
// unknown fields, not a compatibility dispatch path.
type NativePeerRootRequest struct {
	OperationID       string `json:"operation_id"`
	ReconcileApproved bool   `json:"reconcile_approved,omitempty"`
}

type NativePeerRootResponse struct {
	ScopedPeer              *state.ScopedPeerRootReceipt        `json:"scoped_peer,omitempty"`
	ScopedLifecycleRecovery *state.ScopedLifecycleRecoveryProof `json:"scoped_lifecycle_recovery,omitempty"`
	ScopedLifecycle         *ScopedLifecycleReceipt             `json:"scoped_lifecycle,omitempty"`
	Status                  string                              `json:"status"`
	Error                   string                              `json:"error,omitempty"`
}

func nativePeerRootPeerUID(conn *net.UnixConn, expected uint32) error {
	return nativePeerRootPeerIdentity(conn, expected, 0)
}
func nativePeerRootPeerIdentity(conn *net.UnixConn, expected uint32, pid int) error {
	if conn == nil {
		return errors.New("missing original local Unix peer")
	}
	raw, err := conn.SyscallConn()
	if err != nil {
		return err
	}
	var cred *unix.Ucred
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		cred, sockErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return err
	}
	if sockErr != nil || cred == nil || cred.Pid <= 0 || cred.Uid != expected || (pid > 0 && int(cred.Pid) != pid) {
		return errors.New("root socket request from unapproved service UID")
	}
	return nil
}

func nativePeerRootValidateRequest(req NativePeerRootRequest) error {
	if !scopedLifecycleID.MatchString(req.OperationID) {
		return errors.New("invalid scoped lifecycle original")
	}
	return nil
}

type nativePeerRootHandler func(context.Context, NativePeerRootRequest) (NativePeerRootResponse, error)

// Sequential, bounded, credential-gated local service. A single connection
// can never schedule an unbounded goroutine or a second concurrent effect.
func nativePeerRootServeListener(ctx context.Context, ln *net.UnixListener,
	uid uint32, handle nativePeerRootHandler) error {
	if ln == nil || handle == nil {
		return errors.New("missing local root listener or bounded action")
	}
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		func() {
			defer conn.Close()
			_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			if err := nativePeerRootPeerUID(conn, uid); err != nil {
				_ = json.NewEncoder(conn).Encode(NativePeerRootResponse{Status: "DENIED", Error: "unapproved local peer"})
				return
			}
			dec := json.NewDecoder(io.LimitReader(conn, 2048))
			dec.DisallowUnknownFields()
			var req NativePeerRootRequest
			if err := dec.Decode(&req); err != nil {
				_ = json.NewEncoder(conn).Encode(NativePeerRootResponse{Status: "DENIED", Error: "invalid bounded root request"})
				return
			}
			var extra any
			if err := dec.Decode(&extra); err != io.EOF || nativePeerRootValidateRequest(req) != nil {
				_ = json.NewEncoder(conn).Encode(NativePeerRootResponse{Status: "DENIED", Error: "ambiguous root request"})
				return
			}
			_ = conn.SetReadDeadline(time.Time{})
			callCtx, cancel := context.WithTimeout(ctx, 210*time.Second)
			defer cancel()
			result, err := handle(callCtx, req)
			if err != nil {
				log.Printf("Root original %s refused or uncertain: %v", req.OperationID, err)
				result.Status = "DENIED_OR_UNKNOWN"
				result.Error = err.Error()
			}
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			_ = json.NewEncoder(conn).Encode(result)
		}()
	}
}

// nativePeerRootReclaimStaleSocket handles only the exact old socket inode
// stranded by an interrupted helper. This does not clear root journals or
// authorize an operation that was previously dispatched.
func nativePeerRootReclaimStaleSocket(path string, uid, gid uint32) error {
	previous, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	stat, ok := previous.Sys().(*syscall.Stat_t)
	if !ok || previous.Mode()&os.ModeSocket == 0 ||
		previous.Mode().Perm() != 0660 || stat.Uid != uid || stat.Gid != gid {
		return errors.New("existing root peer socket type or ownership requires operator inspection")
	}
	conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		return errors.New("root peer listener already active")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		return fmt.Errorf("root peer socket state uncertain; operator inspection required: %w", err)
	}
	current, err := os.Lstat(path)
	if err != nil || !os.SameFile(previous, current) {
		return errors.New("root peer socket changed during stale-listener verification")
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
