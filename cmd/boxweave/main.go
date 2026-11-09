// BoxWeave v1 host-local CLI; no general host-shell or public REST surface.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

var (
	version = "0.0.0-s1-dev"
	commit  = "unverified"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: boxweave serve|status|doctor|token-create|token-revoke|build-info|install-stage|install-activate|install-status|upgrade|rollback|install-commit|recover|native-plan|native-bootstrap [flags]")
}
func run() error {
	if len(os.Args) < 2 {
		usage()
		return errors.New("subcommand required")
	}
	command := os.Args[1]
	if command == "recover-worker" {
		if len(os.Args) != 2 {
			return errors.New("private worker accepts no user arguments")
		}
		return recoverWorker()
	}
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	db := f.String("db", "/var/lib/boxweave/ledger.sqlite", "protected local SQLite path")
	socket := f.String("incus-socket", "/var/lib/incus/unix.socket.user", "restricted Incus Unix socket")
	project := f.String("project", "", "installer-verified confined project; mandatory for native reads")
	listener := f.String("listen", "127.0.0.1:18650", "loopback management HTTP listen address")
	role := f.String("role", "viewer", "token role")
	ttl := f.Duration("ttl", 30*24*time.Hour, "token validity")
	tokenID := f.String("id", "", "exact local credential ID")
	serviceUser := f.String("service-user", "boxweave", "dedicated nonroot incus-only service account")
	sourceBinary := f.String("source-binary", "", "source executable at approved exact digest")
	digest := f.String("expect-sha256", "", "locally reviewed binary sha256")
	approve := f.Bool("approve-stage", false, "explicitly authorize staging service files; no automatic start")
	approveLocal := f.Bool("approve-local", false, "host-local root authorization to activate, upgrade or rollback")
	nativePackageVersion := f.String("incus-package-version", "", "exact signed Ubuntu Incus package version (for fresh bootstrap)")
	nativePlanSHA := f.String("native-plan-sha256", "", "operator-reviewed native APT simulation digest")
	approveNative := f.Bool("approve-native-bootstrap", false, "approve exact signed APT plan on an empty local host")
	recoveryAction := f.String("action", "inspect", "manual recovery inspect|challenge|release")
	recoveryOperation := f.String("operation", "", "exact original UNKNOWN operation ID")
	recoveryReviewer := f.String("reviewer", "", "recorded host-local operator identity")
	recoveryChallenge := f.String("challenge", "", "single-use time-limited local recovery challenge")
	recoveryConfirm := f.String("confirm-uuid", "", "literal exact original native Box UUID")
	recoveryReason := f.String("reason", "", "documented review reason")
	recoveryAcknowledge := f.Bool("acknowledge-uncertain-native", false, "explicit acknowledgement that native effect may still be in flight")

	if err := f.Parse(os.Args[2:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	switch command {
	case "serve", "status", "doctor", "token-create", "token-revoke", "build-info", "install-stage", "install-activate", "install-status", "upgrade", "rollback", "install-commit", "recover", "native-plan", "native-bootstrap":
	default:
		usage()
		return errors.New("unsupported command")
	}
	// One fail-closed root-local mutex for every installer or native dependency
	// mutation, including a reviewed UNKNOWN claim release. Read-only status
	// and plan commands deliberately remain available during maintenance.
	switch command {
	case "install-stage", "install-activate", "upgrade", "rollback", "install-commit",
		"native-bootstrap", "recover":
		release, err := acquireMaintenanceLock()
		if err != nil { return err }
		defer release()
	}
	if command == "native-plan" || command == "native-bootstrap" {
		var report install.NativeBootstrapReport
		var err error
		if command == "native-plan" {
			report, err = install.NativeDependencyPlan(*nativePackageVersion)
		} else {
			report, err = install.NativeDependencyBootstrap(*nativePackageVersion, *nativePlanSHA, *approveNative)
		}
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(report)
	}
	if command == "recover" {
		return recoverLocal(recoverArgs{DB: *db, Socket: *socket, Project: *project, ServiceUser: *serviceUser,
			Action: *recoveryAction, Operation: *recoveryOperation, Reviewer: *recoveryReviewer,
			Challenge: *recoveryChallenge, ConfirmUUID: *recoveryConfirm, Reason: *recoveryReason,
			Approved: *approveLocal, Acknowledge: *recoveryAcknowledge})
	}
	if command == "build-info" {
		return json.NewEncoder(os.Stdout).Encode(struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
		}{Version: version, Commit: commit})
	}
	if command == "install-stage" || command == "install-activate" || command == "install-status" || command == "upgrade" || command == "rollback" || command == "install-commit" {
		p := install.Plan{User: *serviceUser, StateDir: "/var/lib/boxweave",
			BinaryPath: "/usr/local/bin/boxweave", UnitPath: "/etc/systemd/system/boxweave.service",
			SourceBinary: *sourceBinary, Project: *project, Socket: *socket, Version: version,
			ExpectedDigest: *digest}
		var m install.Manifest
		var err error
		switch command {
		case "install-stage":
			m, err = install.Stage(p, *approve)
		case "install-activate":
			m, err = install.Activate(p, *approveLocal)
		case "upgrade":
			m, err = install.Upgrade(p, *approveLocal)
		case "rollback":
			m, err = install.Rollback(p, *approveLocal)
		case "install-commit":
			m, err = install.CommitUpgrade(p, *approveLocal)
		case "install-status":
			m, err = install.Status(p)
		}
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(m)
	}
	s, err := state.Open(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Never blindly replay an operation possibly sent to Incus before a restart.
	if command == "serve" {
		if _, err := s.Recover(ctx); err != nil {
			return err
		}
	}
	svc := control.Service{Store: s, Native: native.Reader{Socket: *socket, Project: *project}, BuildVersion: version, BuildCommit: commit, Listener: *listener}
	switch command {
	case "serve":
		return svc.ListenAndServe()
	case "status", "doctor":
		status := svc.Status(ctx)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(status); err != nil {
			return err
		}
		if command == "doctor" && !status.Ready {
			return errors.New("preflight incomplete")
		}
		return nil
	case "token-create":
		if os.Geteuid() == 0 {
			return errors.New("token creation must use the dedicated local service identity")
		}
		raw, t, err := s.IssueToken(ctx, *role, *ttl)
		if err != nil {
			return err
		}
		// Printed once, only following an explicit host-local invocation.
		fmt.Fprintf(os.Stdout, "id=%s role=%s expires=%s\ntoken=%s\n", t.ID, t.Role, t.Expires.Format(time.RFC3339), raw)
		return nil
	case "token-revoke":
		if os.Geteuid() == 0 {
			return errors.New("token revocation must use the dedicated local service identity")
		}
		if *tokenID == "" {
			return errors.New("exact token ID required")
		}
		return s.RevokeToken(ctx, *tokenID)
	}
	return errors.New("unreachable")
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "boxweave:", err)
		os.Exit(1)
	}
}
