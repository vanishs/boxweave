// BoxWeave v1 host-local CLI; no general host-shell or public REST surface.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"golang.org/x/term"
	"os"
	"strings"
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
	fmt.Fprintln(os.Stderr, "usage: boxweave serve|status|doctor|token-create|token-revoke|oauth-setup|oauth-set-origin|oauth-disable|oauth-status|oauth-list-grants|oauth-revoke-grant|build-info|install-stage|install-activate|install-status|upgrade|rollback|install-commit|recover|native-plan|native-bootstrap|native-prereq-plan|native-prereq-verify|native-prepare-plan|native-prepare-apply|native-prepare-status|native-prepare-rollback|native-prepare-archive|native-migrate-plan|native-migrate-apply|native-migrate-status|native-migrate-reconcile|native-migrate-rollback|native-migrate-commit [flags]")
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
	tokenID := f.String("id", "", "exact local credential or OAuth grant ID")
	publicOrigin := f.String("public-origin", "", "exact operator-reviewed public HTTPS origin (without /mcp)")
	serviceUser := f.String("service-user", "boxweave", "dedicated nonroot incus-only service account")
	sourceBinary := f.String("source-binary", "", "source executable at approved exact digest")
	digest := f.String("expect-sha256", "", "locally reviewed binary sha256")
	approve := f.Bool("approve-stage", false, "explicitly authorize staging service files; no automatic start")
	approveLocal := f.Bool("approve-local", false, "host-local root authorization to activate, upgrade or rollback")
	nativePackageVersion := f.String("incus-package-version", "", "exact signed Ubuntu Incus package version (for fresh bootstrap)")
	nativePlanSHA := f.String("native-plan-sha256", "", "operator-reviewed native APT simulation digest")
	approveNative := f.Bool("approve-native-bootstrap", false, "approve exact signed APT plan on an empty local host")
	prereqSHA := f.String("native-prereq-sha256", "", "operator-reviewed exact host and native prerequisite snapshot")
	preparePool := f.String("pool", "", "existing reviewed native dir pool for pre-first-Box preparation")
	prepareNetwork := f.String("network", "", "existing reviewed managed native bridge")
	prepareACL := f.String("acl", "", "new exclusively owned bw-prefixed Incus network ACL")
	prepareSHA := f.String("native-prepare-sha256", "", "operator-reviewed exact native prepare plan/journal digest")
	oldIncusPackage := f.String("old-incus-package", "", "exact current signed Ubuntu Incus package version")
	confirmNativeHost := f.String("confirm-native-host-sha256", "", "operator-reviewed current local machine-ID digest")
	approveMigration := f.Bool("approve-native-migrate", false, "explicit local root approval for the exact old/new native daemon version and host")
	ackNativeDowntime := f.Bool("acknowledge-native-downtime", false, "accept exact native daemon stop/restart on a verified zero-consumer host")

	recoveryAction := f.String("action", "inspect", "manual recover inspect|challenge|release, create-inspect|create-challenge|create-bind, or gc-*")
	recoveryStream := f.String("stream", "", "original stdout|stderr only for locally reviewed Exec log GC")
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
	case "serve", "status", "doctor", "token-create", "token-revoke", "oauth-setup", "oauth-set-origin", "oauth-disable", "oauth-status", "oauth-list-grants", "oauth-revoke-grant", "build-info", "install-stage", "install-activate", "install-status", "upgrade", "rollback", "install-commit", "recover", "native-plan", "native-bootstrap", "native-prereq-plan", "native-prereq-verify", "native-prepare-plan", "native-prepare-apply", "native-prepare-status", "native-prepare-rollback", "native-prepare-archive", "native-migrate-plan", "native-migrate-apply", "native-migrate-status", "native-migrate-reconcile", "native-migrate-rollback", "native-migrate-commit":
	default:
		usage()
		return errors.New("unsupported command")
	}
	// One fail-closed root-local mutex for every installer or native dependency
	// mutation, including a reviewed UNKNOWN claim release. Read-only status
	// and plan commands deliberately remain available during maintenance.
	switch command {
	case "install-stage", "install-activate", "upgrade", "rollback", "install-commit",
		"native-bootstrap", "native-prepare-apply", "native-prepare-rollback", "native-prepare-archive",
		"native-migrate-apply", "native-migrate-reconcile", "native-migrate-rollback", "native-migrate-commit", "recover":
		release, err := acquireMaintenanceLock()
		if err != nil {
			return err
		}
		defer release()
	}
	if command == "native-prepare-plan" || command == "native-prepare-apply" ||
		command == "native-prepare-status" || command == "native-prepare-rollback" ||
		command == "native-prepare-archive" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		target := install.NativePrepareTarget{Project: *project, Pool: *preparePool, Network: *prepareNetwork, ACL: *prepareACL}
		enc := json.NewEncoder(os.Stdout)
		switch command {
		case "native-prepare-plan":
			report, err := install.NativePreparePlanReview(ctx, target)
			if err != nil {
				return err
			}
			return enc.Encode(report)
		case "native-prepare-apply":
			record, err := install.NativePrepareApply(ctx, target, *prepareSHA, *approveLocal)
			if e := enc.Encode(record); e != nil {
				return e
			}
			return err
		case "native-prepare-status":
			record, err := install.NativePrepareStatus()
			if err != nil {
				return err
			}
			return enc.Encode(record)
		case "native-prepare-rollback":
			record, err := install.NativePrepareRollback(ctx, *prepareSHA, *approveLocal)
			if e := enc.Encode(record); e != nil {
				return e
			}
			return err
		case "native-prepare-archive":
			path, err := install.NativePrepareArchive(ctx, *prepareSHA, *approveLocal)
			if err != nil {
				return err
			}
			return enc.Encode(struct {
				Path string `json:"archived_journal_path"`
			}{path})
		}
	}
	if command == "native-prereq-plan" || command == "native-prereq-verify" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var report install.NativePrereqReport
		var err error
		if command == "native-prereq-plan" {
			report, err = install.NativePrerequisites(ctx, *project)
		} else {
			report, err = install.NativePrerequisiteVerify(ctx, *project, *prereqSHA, *approveLocal)
		}
		if e := json.NewEncoder(os.Stdout).Encode(report); e != nil {
			return e
		}
		return err
	}
	if command == "native-migrate-plan" || command == "native-migrate-apply" ||
		command == "native-migrate-status" || command == "native-migrate-reconcile" || command == "native-migrate-rollback" ||
		command == "native-migrate-commit" {
		enc := json.NewEncoder(os.Stdout)
		switch command {
		case "native-migrate-plan":
			result, err := install.NativeMigrationReview(*oldIncusPackage, *nativePackageVersion)
			if err != nil {
				return err
			}
			return enc.Encode(result)
		case "native-migrate-apply":
			result, err := install.NativeMigrationApply(*oldIncusPackage, *nativePackageVersion,
				*nativePlanSHA, *confirmNativeHost, *approveMigration, *ackNativeDowntime)
			if err != nil {
				return err
			}
			return enc.Encode(result)
		case "native-migrate-status":
			result, err := install.NativeMigrationStatus()
			if err != nil {
				return err
			}
			return enc.Encode(result)
		case "native-migrate-reconcile":
			result, err := install.NativeMigrationReconcile(*confirmNativeHost, *nativePackageVersion, *approveMigration)
			if err != nil {
				return err
			}
			return enc.Encode(result)
		case "native-migrate-rollback":
			result, err := install.NativeMigrationRollback(*confirmNativeHost,
				*oldIncusPackage, *approveMigration, *ackNativeDowntime)
			if err != nil {
				return err
			}
			return enc.Encode(result)
		case "native-migrate-commit":
			archive, err := install.NativeMigrationFinalize(*confirmNativeHost,
				*nativePackageVersion, *approveMigration)
			if err != nil {
				return err
			}
			return enc.Encode(struct {
				Archive string `json:"audit_archive"`
			}{archive})
		}
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
			Action: *recoveryAction, Operation: *recoveryOperation, Reviewer: *recoveryReviewer, Stream: *recoveryStream,
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
	// Only the active server may run startup recovery. A concurrent serve
	// sharing SQLite could otherwise UNKNOWN the first server's live work.
	// The kernel releases this private flock automatically on SIGKILL.
	if command == "serve" {
		locked, err := acquireServeLock(*db)
		if err != nil {
			return err
		}
		defer locked.Close()
		if _, err := s.Recover(ctx); err != nil {
			return err
		}
		// A crash between native DELETE and its verification is NEVER
		// interpreted as safe to reissue. Audit and hold the exact Box.
		if _, err := s.RecoverExecGC(ctx); err != nil {
			return err
		}
	}
	// OAuth configuration is host-local only. A browser request may never
	// alter the approval secret, public issuer, grants or server identity.
	if strings.HasPrefix(command, "oauth-") {
		if os.Geteuid() == 0 {
			return errors.New("OAuth management must run under the dedicated local service account")
		}
		switch command {
		case "oauth-setup":
			fmt.Fprint(os.Stderr, "New OAuth approval password (min 16 characters): ")
			first, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil {
				return errors.New("OAuth setup requires a local interactive TTY")
			}
			fmt.Fprint(os.Stderr, "Repeat OAuth approval password: ")
			second, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(os.Stderr)
			if err != nil || string(first) != string(second) {
				return errors.New("approval password confirmation failed")
			}
			return s.SetupOAuthPassword(ctx, string(first))
		case "oauth-set-origin":
			if *publicOrigin == "" {
				return errors.New("-public-origin required")
			}
			return s.SetOAuthOrigin(ctx, *publicOrigin)
		case "oauth-disable":
			return s.SetOAuthOrigin(ctx, "")
		case "oauth-status":
			v, err := s.OAuthConfig(ctx)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(v)
		case "oauth-list-grants":
			grants, err := s.ListOAuthGrants(ctx)
			if err != nil {
				return err
			}
			return json.NewEncoder(os.Stdout).Encode(grants)
		case "oauth-revoke-grant":
			if *tokenID == "" {
				return errors.New("exact grant -id required")
			}
			return s.RevokeOAuthGrant(ctx, *tokenID)
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