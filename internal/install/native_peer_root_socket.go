package install

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/state"
	"golang.org/x/sys/unix"
)

const nativePeerRootSocketPath = "/run/boxweave-native-peer-root.sock"

// The IPC protocol deliberately has NO project, IP, ACL, shell or native
// parameters. An empty digest requests only a read-only original review.
// A nonempty digest asks the root helper to re-review the exact durable
// original and execute at most once in its own fsynced per-operation journal.
type NativePeerRootRequest struct {
	OperationID string `json:"operation_id"`
	PlanSHA256  string `json:"plan_sha256"`
	Reconcile   bool   `json:"reconcile,omitempty"`
}
type NativePeerRootResponse struct {
	Status     string                             `json:"status"`
	PlanSHA256 string                             `json:"plan_sha256,omitempty"`
	Receipt    *NativePeerConntrackJournalReceipt `json:"receipt,omitempty"`
	Error      string                             `json:"error,omitempty"`
}

type NativePeerRootScope struct {
	DBPath, Project, Bridge, ACL string
	OriginalID, ApprovedPlanSHA  string
	Approved                     bool
}

// A root operator approves ONE claimed original and ONE exact native plan;
// a future controller cannot use that authorization for another operation.
func nativePeerRootApprovedRequest(cfg NativePeerRootScope, req NativePeerRootRequest) bool {
	return cfg.Approved && nativePeerRootValidateRequest(req) == nil &&
		cfg.OriginalID != "" && cfg.ApprovedPlanSHA != "" &&
		req.OperationID == cfg.OriginalID &&
		(req.PlanSHA256 == "" || req.PlanSHA256 == cfg.ApprovedPlanSHA)
}

func nativePeerRootPeerUID(conn *net.UnixConn, expected uint32) error {
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
	if sockErr != nil || cred == nil || cred.Pid <= 0 || cred.Uid != expected {
		return errors.New("root socket request from unapproved service UID")
	}
	return nil
}

func nativePeerRootValidateRequest(req NativePeerRootRequest) error {
	if _, err := nativePeerConntrackOperationJournalPath(req.OperationID); err != nil {
		return err
	}
	if req.Reconcile && req.PlanSHA256 == "" {
		return errors.New("root reconciliation requires the original reviewed digest")
	}
	if req.PlanSHA256 != "" {
		if len(req.PlanSHA256) != 64 {
			return errors.New("root request digest must be canonical SHA256")
		}
		for _, c := range req.PlanSHA256 {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				return errors.New("untrusted root request digest")
			}
		}
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

// NativePeerRootServe must be explicitly started as host root by a local
// operator for a fixed Project/bridge/ACL. Root checks the restricted
// service UID on EACH connection with SO_PEERCRED, and independently reads
// all SQLite/native facts. This is never an MCP/root socket proxy.
func NativePeerRootServe(ctx context.Context, cfg NativePeerRootScope) error {
	if os.Geteuid() != 0 || !cfg.Approved {
		return errors.New("root peer helper requires explicit host-local approval")
	}
	u, err := user.Lookup("boxweave")
	if err != nil {
		return err
	}
	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil || gid < 1 {
		return errors.New("unsafe installed service group")
	}
	if cfg.Project != "user-"+u.Uid || cfg.Bridge == "" || cfg.ACL == "" ||
		nativePeerRootValidateRequest(NativePeerRootRequest{
			OperationID: cfg.OriginalID, PlanSHA256: cfg.ApprovedPlanSHA}) != nil ||
		cfg.ApprovedPlanSHA == "" {
		return errors.New("root helper must bind one exact original and operator-approved plan digest")
	}
	db, err := state.OpenNetworkAuditReadOnly(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()
	// Recover only a confirmed dead exact root-owned endpoint after crash;
	// no root intent or previously dispatched effect may be replayed.
	if err := nativePeerRootReclaimStaleSocket(nativePeerRootSocketPath, 0, uint32(gid)); err != nil {
		return err
	}
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: nativePeerRootSocketPath, Net: "unix"})
	if err != nil {
		return err
	}
	ln.SetUnlinkOnClose(false)
	original, err := os.Lstat(nativePeerRootSocketPath)
	if err != nil {
		_ = ln.Close()
		return err
	}
	defer func() {
		_ = ln.Close()
		if current, err := os.Lstat(nativePeerRootSocketPath); err == nil && os.SameFile(original, current) {
			_ = os.Remove(nativePeerRootSocketPath)
		}
	}()
	if err := os.Chown(nativePeerRootSocketPath, 0, gid); err != nil {
		return err
	}
	if err := os.Chmod(nativePeerRootSocketPath, 0660); err != nil {
		return err
	}
	return nativePeerRootServeListener(ctx, ln, uint32(uid),
		func(callCtx context.Context, req NativePeerRootRequest) (NativePeerRootResponse, error) {
			if !nativePeerRootApprovedRequest(cfg, req) {
				return NativePeerRootResponse{}, errors.New("root helper operator approval does not match original and plan")
			}
			if req.Reconcile {
				receipt, err := NativePeerRootReconcileCompletedUnknown(callCtx, db,
					cfg.Project, cfg.Bridge, cfg.ACL, req.OperationID, req.PlanSHA256)
				if err != nil {
					return NativePeerRootResponse{Status: "RECONCILE_REFUSED"}, err
				}
				return NativePeerRootResponse{Status: "RECONCILE_VERIFIED", Receipt: &receipt}, nil
			}
			if req.PlanSHA256 == "" {
				plan, err := NativePeerRootIntegratedReviewOperation(callCtx, db,
					cfg.Project, cfg.Bridge, cfg.ACL, req.OperationID)
				if err != nil {
					return NativePeerRootResponse{}, err
				}
				if plan.PlanSHA256 != cfg.ApprovedPlanSHA {
					return NativePeerRootResponse{}, errors.New("root preview differs from operator-approved pre-isolation SHA")
				}
				return NativePeerRootResponse{Status: "REVIEW_ONLY", PlanSHA256: plan.PlanSHA256}, nil
			}
			receipt, err := NativePeerRootIntegratedApplyOperation(callCtx, db,
				cfg.Project, cfg.Bridge, cfg.ACL, req.OperationID, req.PlanSHA256, true)
			if err != nil {
				return NativePeerRootResponse{Status: "DENIED_OR_UNKNOWN", Receipt: &receipt}, err
			}
			if !receipt.KernelVerified || receipt.VerifiedNICs < 2 || receipt.PostNativeDigestSHA256 == "" {
				return NativePeerRootResponse{Status: "DENIED_OR_UNKNOWN", Receipt: &receipt},
					errors.New("root effect lacks per-NIC nft kernel evidence")
			}
			return NativePeerRootResponse{Status: receipt.Status, Receipt: &receipt}, nil
		})
}

func nativePeerRootCall(ctx context.Context, path string, req NativePeerRootRequest) (NativePeerRootResponse, error) {
	var out NativePeerRootResponse
	if err := nativePeerRootValidateRequest(req); err != nil {
		return out, err
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		return out, err
	}
	defer conn.Close()
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return out, errors.New("root handoff did not connect by Unix credentials")
	}
	if err := json.NewEncoder(uc).Encode(req); err != nil {
		return out, err
	}
	if err := uc.CloseWrite(); err != nil {
		return out, err
	}
	_ = uc.SetReadDeadline(time.Now().Add(210 * time.Second))
	if err := json.NewDecoder(io.LimitReader(uc, 4096)).Decode(&out); err != nil {
		return out, err
	}
	if out.Error != "" {
		return out, fmt.Errorf("native root rejected original: %s", out.Error)
	}
	if out.Status == "" || out.Status == "DENIED" {
		return out, errors.New("native root returned no confirmed original status")
	}
	return out, nil
}

// NativePeerRootCall is for the installed nonroot controller. It cannot
// select a Project, bridge, ACL, host command, arbitrary IP or root path.
func NativePeerRootCall(ctx context.Context, original, reviewedSHA string) (NativePeerRootResponse, error) {
	return nativePeerRootCall(ctx, nativePeerRootSocketPath, NativePeerRootRequest{
		OperationID: original, PlanSHA256: reviewedSHA,
	})
}

// NativePeerRootCallReconcile is ONLY a read-only post-effect root witness.
// It cannot dispatch the original again, clear UNKNOWN or edit host policy.
func NativePeerRootCallReconcile(ctx context.Context, original, reviewedSHA string) (NativePeerRootResponse, error) {
	return nativePeerRootCall(ctx, nativePeerRootSocketPath, NativePeerRootRequest{
		OperationID: original, PlanSHA256: reviewedSHA, Reconcile: true,
	})
}
