package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type recoverArgs struct {
	DB, Socket, Project, ServiceUser, Action, Operation, Reviewer, Stream string
	Challenge, ConfirmUUID, Reason                                        string
	Approved, Acknowledge                                                 bool
}

func recoveryPlan(a recoverArgs) install.Plan {
	return install.Plan{User: a.ServiceUser, StateDir: "/var/lib/boxweave",
		BinaryPath: "/usr/local/bin/boxweave", UnitPath: "/etc/systemd/system/boxweave.service",
		Project: a.Project, Socket: a.Socket, Version: version}
}

// This is the only privileged local entry point. Credential selection is done
// by the kernel on a newly exec'd process, NOT syscall.Setuid inside Go's
// multithreaded process (which may leave other threads privileged/cached).
func recoverLocal(a recoverArgs) error {
	if !a.Approved || os.Geteuid() != 0 {
		return errors.New("manual UNKNOWN review requires explicit local root authorization")
	}
	p := recoveryPlan(a)
	if a.DB != filepath.Join(p.StateDir, "ledger.sqlite") || a.Operation == "" || a.Reviewer == "" {
		return errors.New("exact installed SQLite path, operation and reviewer required")
	}
	if a.Action != "inspect" && a.Action != "challenge" && a.Action != "release" &&
		a.Action != "gc-inspect" && a.Action != "gc-challenge" && a.Action != "gc-release" &&
		a.Action != "create-inspect" && a.Action != "create-challenge" && a.Action != "create-bind" {
		return errors.New("recovery action must be inspect/challenge/release, create-inspect/create-challenge/create-bind or gc-*")
	}
	if _, err := install.Status(p); err != nil {
		return fmt.Errorf("installed identity unverified: %w", err)
	}
	uid, gid, err := install.ServiceIdentity(p)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	descs, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	rootSide := os.NewFile(uintptr(descs[0]), "review-root-peer")
	childSide := os.NewFile(uintptr(descs[1]), "review-child-peer")
	defer rootSide.Close()
	defer childSide.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "recover-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{
		Uid: uint32(uid), Gid: uint32(gid), Groups: []uint32{uint32(gid)},
	}}
	cmd.ExtraFiles = []*os.File{childSide}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn restricted recovery process: %w", err)
	}
	_ = childSide.Close()
	writeErr := json.NewEncoder(rootSide).Encode(a)
	if writeErr != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("pass reviewed recovery request: %w", writeErr)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("restricted recovery worker failed: %w", err)
	}
	return nil
}

// Not exposed as an MCP or ordinary CLI operation. A nonroot process may only
// enter via fd3 paired with an actual root-owned Unix socket peer. Opening any
// self-created pipe/socket as the service user is insufficient.
func recoverWorker() error {
	if os.Geteuid() == 0 {
		return errors.New("recovery worker must be nonroot")
	}
	fd := 3
	peer, err := syscall.GetsockoptUcred(fd, syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	if err != nil || peer.Uid != 0 {
		return errors.New("root-reviewed Unix socket peer required")
	}
	pipes := os.NewFile(uintptr(fd), "trusted-recovery-request")
	if pipes == nil {
		return errors.New("missing trusted recovery request")
	}
	defer pipes.Close()
	var a recoverArgs
	if err := json.NewDecoder(io.LimitReader(pipes, 8192)).Decode(&a); err != nil {
		return fmt.Errorf("invalid reviewed request: %w", err)
	}
	if !a.Approved {
		return errors.New("root authorization missing")
	}
	p := recoveryPlan(a)
	uid, _, err := install.ServiceIdentity(p)
	if err != nil {
		return err
	}
	if os.Geteuid() != uid {
		return errors.New("incorrect restricted operating identity")
	}
	return performRecovery(a)
}

func performRecovery(a recoverArgs) error {
	p := recoveryPlan(a)
	if a.DB != filepath.Join(p.StateDir, "ledger.sqlite") || a.Operation == "" || a.Reviewer == "" {
		return errors.New("invalid reviewed recovery target")
	}
	if _, err := install.Status(p); err != nil {
		return err
	}
	store, err := state.Open(a.DB)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if strings.HasPrefix(a.Action, "gc-") {
		return performGCRecovery(ctx, store, a)
	}
	if strings.HasPrefix(a.Action, "create-") {
		return performUnknownCreateBinding(ctx, store, a)
	}
	r, err := store.RecoveryInspect(ctx, a.Operation)
	if err != nil {
		return err
	}
	if r.Project != a.Project {
		return errors.New("UNKNOWN belongs to a foreign Project")
	}
	observed, err := (native.Reader{Socket: a.Socket, Project: a.Project}).ObserveOwnedBox(ctx, r.Name, r.BoxUUID)
	if err != nil {
		return fmt.Errorf("native re-observation refused: %w", err)
	}
	out := json.NewEncoder(os.Stdout)
	switch a.Action {
	case "inspect":
		return out.Encode(struct {
			Original state.RecoverySnapshot `json:"original"`
			Native   native.BoxObservation  `json:"native_observation"`
			Warning  string                 `json:"warning"`
		}{r, observed, "UNKNOWN is not success or failure; never replay the original request"})
	case "challenge":
		if r.Released {
			return errors.New("already reviewed and released")
		}
		challenge, err := store.IssueRecoveryChallenge(ctx, r, a.Reviewer, observed.Fingerprint)
		if err != nil {
			return err
		}
		return out.Encode(struct {
			Operation        string `json:"operation_id"`
			Challenge        string `json:"challenge"`
			ExpiresInSeconds int    `json:"expires_in_seconds"`
			Warning          string `json:"warning"`
		}{r.OperationID, challenge, 300, "Native outcome may still be UNKNOWN; release never retries it"})
	case "release":
		if err := store.ReleaseRecoveryClaim(ctx, r, a.Reviewer, a.Challenge, a.ConfirmUUID,
			a.Reason, observed.Fingerprint, a.Acknowledge); err != nil {
			return err
		}
		return out.Encode(struct {
			Operation     string `json:"operation_id"`
			State         string `json:"original_state"`
			ClaimReleased bool   `json:"claim_released"`
			AutoRetry     bool   `json:"auto_retry"`
		}{r.OperationID, "UNKNOWN", true, false})
	}
	return errors.New("unsupported recovery mode")
}

// performGCRecovery is reachable ONLY through the existing root-reviewed
// Unix peer/credential-drop path. It NEVER dispatches a native DELETE;
// the outcome of a prior lost ACK remains permanently recorded for review.
func performGCRecovery(ctx context.Context, store *state.Store, a recoverArgs) error {
	if a.Stream != "stdout" && a.Stream != "stderr" {
		return errors.New("exact original stdout/stderr review target required")
	}
	r, err := store.ExecGCReviewInspect(ctx, a.Operation, a.Stream)
	if err != nil {
		return err
	}
	if r.Project != a.Project {
		return errors.New("recorded native log belongs to another Project")
	}
	reader := native.Reader{Socket: a.Socket, Project: a.Project}
	if _, _, err := reader.GetExactBox(ctx, r.Name, r.BoxUUID); err != nil {
		return fmt.Errorf("exact native GC UUID not re-observed: %w", err)
	}
	// Observe only existence, never disclose one guest byte in diagnostics.
	_, _, err = reader.ReadRecordedOutput(ctx, r.Path, 0, 1)
	observed := "NATIVE_PRESENT"
	if errors.Is(err, native.ErrRecordedOutputUnavailable) {
		observed = "NATIVE_MISSING"
	} else if err != nil {
		return fmt.Errorf("original native log existence uncertain: %w", err)
	}
	out := json.NewEncoder(os.Stdout)
	switch a.Action {
	case "gc-inspect":
		return out.Encode(struct {
			Original          state.ExecGCReview
			NativeObservation string
			Warning           string
		}{r, observed, "The earlier native DELETE outcome may be uncertain; no retry and no invented expiry"})
	case "gc-challenge":
		challenge, err := store.IssueExecGCReviewChallenge(ctx, r, a.Reviewer, observed)
		if err != nil {
			return err
		}
		return out.Encode(struct {
			Operation        string
			Stream           string
			Challenge        string
			ExpiresInSeconds int
			Warning          string
		}{r.OperationID, r.Stream, challenge, 300, "Release lifts only this claim; no native DELETE is retried"})
	case "gc-release":
		if err := store.ReleaseExecGCReview(ctx, r, a.Reviewer, a.Challenge,
			a.ConfirmUUID, a.Reason, observed, a.Acknowledge); err != nil {
			return err
		}
		return out.Encode(struct {
			Operation      string
			Stream         string
			State          string
			ClaimReleased  bool
			NativeRetried  bool
			ExpiryInferred bool
		}{r.OperationID, r.Stream, "REVIEWED_RELEASED", true, false, false})
	}
	return errors.New("invalid native GC review operation")
}
