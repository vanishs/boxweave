package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/recovery"
	"github.com/vanishs/boxweave/internal/state"
)

type recoverArgs struct {
	DB, Socket, Project, ServiceUser, Action, Operation, Reviewer, Stream string
	Challenge, ConfirmUUID, Reason, RecoveryKey                           string
	Approved, Acknowledge                                                 bool
	ScopedGCManifest                                                      *install.Manifest // private root-to-worker evidence; never a CLI argument
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
	if !gcReviewAction(a.Action) {
		return errors.New("scoped original requires the reviewed recovery route")
	}
	// main.run already holds the fixed maintenance lock across recoverLocal
	// and its worker. Do not reacquire the same non-reentrant flock here.
	// The local Root review still covers the protected fresh journal.
	a.ScopedGCManifest = nil
	if gcReviewAction(a.Action) {
		if err := gcRecoveryFences(); err != nil {
			return err
		}
	}
	m, err := install.Status(p)
	if err != nil {
		return fmt.Errorf("installed identity unverified: %w", err)
	}
	if err := install.VerifyScopedGCReviewIdentity(p, m); err != nil {
		return err
	}
	a.ScopedGCManifest = &m
	uid, gid, err := install.ServiceIdentity(p)
	if err != nil {
		return err
	}
	// Preserve only the already authorized restricted Incus group. The
	// dedicated account's primary group does not necessarily own the socket.
	incusGroup, err := user.LookupGroup("incus")
	if err != nil {
		return err
	}
	incusGID, err := strconv.ParseUint(incusGroup.Gid, 10, 32)
	if err != nil || incusGID == 0 {
		return errors.New("invalid restricted incus group")
	}
	groups := recoveryGroups(uint32(gid), uint32(incusGID))
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
		Uid: uint32(uid), Gid: uint32(gid), Groups: groups,
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
	if !gcReviewAction(a.Action) || a.ScopedGCManifest == nil || a.RecoveryKey != "" {
		return state.ErrInvalid
	}
	if err := gcRecoveryFences(); err != nil {
		return err
	}
	if err := install.VerifyScopedGCReviewIdentity(p, *a.ScopedGCManifest); err != nil {
		return err
	}
	store, err := state.OpenExistingRecovery(a.DB)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return performGCRecovery(ctx, store, a)
}

func gcReviewAction(action string) bool {
	return action == "gc-inspect" || action == "gc-challenge" || action == "gc-release"
}

func gcRecoveryFences() error {
	for _, check := range []func() error{install.CheckScopedDrainFence, install.CheckOfflineRetirementFence, install.CheckScopedRetirementFence} {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
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
	observed, err := recovery.ObserveGC(ctx, reader, r)
	if err != nil {
		return fmt.Errorf("original GC observation refused: %w", err)
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

// Do not inherit root or unrelated supplemental groups across the worker exec.
func recoveryGroups(primary, restricted uint32) []uint32 {
	if primary == restricted {
		return []uint32{primary}
	}
	return []uint32{primary, restricted}
}
