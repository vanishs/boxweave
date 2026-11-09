package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type recoverArgs struct {
	DB, Socket, Project, ServiceUser, Action, Operation, Reviewer string
	Challenge, ConfirmUUID, Reason                                string
	Approved, Acknowledge                                         bool
}

// Only a physically local root operator with the exact installed identity may
// initiate a reviewed recovery. Drop root irreversibly BEFORE opening SQLite
// or touching the confined Incus socket: no root-owned WAL/SHM files.
func recoverLocal(a recoverArgs) error {
	if !a.Approved || os.Geteuid() != 0 {
		return errors.New("manual UNKNOWN review requires explicit local root authorization")
	}
	p := install.Plan{User: a.ServiceUser, StateDir: "/var/lib/boxweave",
		BinaryPath: "/usr/local/bin/boxweave", UnitPath: "/etc/systemd/system/boxweave.service",
		Project: a.Project, Socket: a.Socket, Version: version}
	if a.DB != filepath.Join(p.StateDir, "ledger.sqlite") || a.Operation == "" || a.Reviewer == "" {
		return errors.New("exact installed SQLite path, operation and reviewer required")
	}
	if a.Action != "inspect" && a.Action != "challenge" && a.Action != "release" {
		return errors.New("recovery action must be inspect/challenge/release")
	}
	if _, err := install.Status(p); err != nil {
		return fmt.Errorf("installed BoxWeave identity is not verified: %w", err)
	}
	uid, gid, err := install.ServiceIdentity(p)
	if err != nil {
		return err
	}
	if err = syscall.Setgroups([]int{gid}); err != nil {
		return err
	}
	if err = syscall.Setgid(gid); err != nil {
		return err
	}
	if err = syscall.Setuid(uid); err != nil {
		return err
	}
	if os.Geteuid() != uid {
		return errors.New("could not drop root privileges")
	}
	store, err := state.Open(a.DB)
	if err != nil {
		return err
	}
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	r, err := store.RecoveryInspect(ctx, a.Operation)
	if err != nil {
		return err
	}
	if r.Project != a.Project {
		return errors.New("UNKNOWN operation belongs to a foreign Project")
	}
	observed, err := (native.Reader{Socket: a.Socket, Project: a.Project}).ObserveOwnedBox(ctx, r.Name, r.BoxUUID)
	if err != nil {
		return fmt.Errorf("native re-observation refused: %w", err)
	}
	enc := json.NewEncoder(os.Stdout)
	switch a.Action {
	case "inspect":
		return enc.Encode(struct {
			Original state.RecoverySnapshot `json:"original"`
			Native   native.BoxObservation  `json:"native_observation"`
			Warning  string                 `json:"warning"`
		}{r, observed, "UNKNOWN is not a failure or success; do not retry the original request"})
	case "challenge":
		if r.Released {
			return errors.New("already reviewed and released")
		}
		challenge, err := store.IssueRecoveryChallenge(ctx, r, a.Reviewer, observed.Fingerprint)
		if err != nil {
			return err
		}
		// Printed once only to the protected local terminal; store holds digest.
		return enc.Encode(struct {
			Operation        string `json:"operation_id"`
			Challenge        string `json:"challenge"`
			ExpiresInSeconds int    `json:"expires_in_seconds"`
			Warning          string `json:"warning"`
		}{r.OperationID, challenge, 300, "Native outcome remains UNKNOWN; release never retries it"})
	case "release":
		if err := store.ReleaseRecoveryClaim(ctx, r, a.Reviewer, a.Challenge, a.ConfirmUUID,
			a.Reason, observed.Fingerprint, a.Acknowledge); err != nil {
			return err
		}
		return enc.Encode(struct {
			Operation     string `json:"operation_id"`
			State         string `json:"original_state"`
			ClaimReleased bool   `json:"claim_released"`
			AutoRetry     bool   `json:"auto_retry"`
		}{r.OperationID, "UNKNOWN", true, false})
	}
	return errors.New("invalid recovery mode")
}
