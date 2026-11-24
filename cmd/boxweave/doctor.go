package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type doctorArgs struct{ DB, Socket, Project, ServiceUser string }

func doctorInstallPlan(a doctorArgs) install.Plan {
	return install.Plan{User: a.ServiceUser, StateDir: "/var/lib/boxweave", BinaryPath: "/usr/local/bin/boxweave", UnitPath: "/etc/systemd/system/boxweave.service", Project: a.Project, Socket: a.Socket, Version: version}
}

// Private nonroot read-only worker; no privilege escalation, credentials,
// arbitrary paths, Root socket dispatch or business mutation.
func doctorNativeActive(ctx context.Context, r native.Reader) native.Status {
	// Connecting to a systemd socket can start an inactive service. Refuse
	// that observation unless the known daemon is already active.
	cmd := exec.CommandContext(ctx, "/usr/bin/systemctl", "show", "-p", "ActiveState", "--value", "incus.service")
	cmd.Env = []string{"PATH=/usr/bin:/bin", "LANG=C"}
	b, e := cmd.Output()
	if e != nil || strings.TrimSpace(string(b)) != "active" {
		return native.Status{Project: r.Project, ErrorClass: "INCUS_SERVICE_ACTIVE_UNVERIFIED"}
	}
	return r.Read(ctx)
}

func doctorNativeWorker() error {
	p := doctorInstallPlan(doctorArgs{Project: "user-" + strconv.Itoa(os.Geteuid()), ServiceUser: "boxweave", Socket: "/var/lib/incus/unix.socket.user"})
	uid, _, e := install.ServiceIdentity(p)
	if e != nil || uid != os.Geteuid() || uid == 0 {
		return errors.New("restricted installed diagnostic actor required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return json.NewEncoder(os.Stdout).Encode(doctorNativeActive(ctx, native.Reader{Project: p.Project, Socket: p.Socket}))
}

func doctorNative(ctx context.Context, a doctorArgs, identity install.DiagnosticInstallation) native.Status {
	r := native.Reader{Socket: a.Socket, Project: a.Project}
	if os.Geteuid() != 0 {
		return doctorNativeActive(ctx, r)
	}
	if identity.InstalledIdentity != "VERIFIED" || a.DB != "/var/lib/boxweave/ledger.sqlite" || a.ServiceUser != "boxweave" || a.Socket != "/var/lib/incus/unix.socket.user" {
		return native.Status{Project: a.Project, ErrorClass: "RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED"}
	}
	p := doctorInstallPlan(a)
	uid, gid, e := install.ServiceIdentity(p)
	if e != nil || a.Project != "user-"+strconv.Itoa(uid) {
		return native.Status{Project: a.Project, ErrorClass: "RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED"}
	}
	g, e := user.LookupGroup("incus")
	if e != nil {
		return native.Status{Project: a.Project, ErrorClass: "RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED"}
	}
	ig, e := strconv.ParseUint(g.Gid, 10, 32)
	if e != nil || ig == 0 {
		return native.Status{Project: a.Project, ErrorClass: "RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED"}
	}
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "doctor-native-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: recoveryGroups(uint32(gid), uint32(ig))}}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config", "LANG=C"}
	b, e := cmd.Output()
	var out native.Status
	if e != nil || len(b) > 65536 || json.Unmarshal(b, &out) != nil || out.Project != a.Project {
		return native.Status{Project: a.Project, ErrorClass: "RESTRICTED_DIAGNOSTIC_READ_UNAVAILABLE"}
	}
	return out
}

func runDoctor(a doctorArgs) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	report := diagnostic.NewReport(a.Project)
	legacy := control.HostStatus{Version: version, Commit: commit, GoVersion: report.GoVersion}
	instctx, instcancel := context.WithTimeout(ctx, 5*time.Second)
	report.Installation = install.ObserveDiagnosticInstallation(instctx, doctorInstallPlan(a), a.DB)
	instcancel()
	if report.Installation.Mode != "LEGACY" && report.Installation.Mode != "SCOPED_V2" {
		reason := report.Installation.Reason
		if reason == "" {
			reason = "ARCHITECTURE_UNVERIFIED"
		}
		report.AddIssue("installation", reason, "installation", "UNVERIFIED")
	}
	// Independent source budgets ensure a missing/busy database never hides
	// available native or installation observations.
	dbctx, dbcancel := context.WithTimeout(ctx, 5*time.Second)
	snap, e := state.OpenDiagnosticSnapshot(dbctx, a.DB)
	if e != nil {
		report.Database = diagnostic.Source{State: "UNVERIFIED", Reason: diagnostic.DatabaseReason(e)}
		report.AddIssue("database", report.Database.Reason, "ledger", "UNVERIFIED")
	} else {
		legacy.SchemaVersion, e = snap.Schema(dbctx)
		if e == nil && legacy.SchemaVersion == state.SchemaVersion {
			legacy.MCPReady = true // compatibility field only, never endpoint proof
			report.Database = diagnostic.Source{State: "OBSERVED", Reason: snap.Capture, EpochNS: snap.EpochNS}
			if cap, e := snap.ProjectCapacity(dbctx, a.Project); e == nil {
				legacy.ProjectCapacity = &cap
			} else {
				report.AddIssue("database", "PROJECT_CAPACITY_UNVERIFIED", "ledger", "UNVERIFIED")
			}
			if report.Installation.Mode == "SCOPED_V2" {
				summary, e := snap.ScopedLedgerSummary(dbctx, a.Project, "incusbr-"+strings.TrimPrefix(a.Project, "user-"))
				if e == nil {
					report.ScopedLedger = &summary
				} else {
					report.AddIssue("database", "SCOPED_LEDGER_OBSERVATION_UNVERIFIED", "ledger", "UNVERIFIED")
				}
			}

		} else {
			report.Database = diagnostic.Source{State: "UNVERIFIED", Reason: "DIAGNOSTIC_DB_SCHEMA_OR_INTEGRITY_UNVERIFIED"}
			report.AddIssue("database", report.Database.Reason, "ledger", "UNVERIFIED")
		}
		if e = snap.Close(); e != nil {
			report.Database.State = "UNVERIFIED"
			report.Database.Reason = "PRIVATE_CAPTURE_CLEANUP_UNVERIFIED"
			legacy.MCPReady = false
			report.AddIssue("database", report.Database.Reason, "ledger", "UNVERIFIED")
		}
	}
	dbcancel()
	nctx, ncancel := context.WithTimeout(ctx, 5*time.Second)
	legacy.Native = doctorNative(nctx, a, report.Installation)
	ncancel()
	if legacy.Native.Ready {
		report.Native = diagnostic.Source{State: "OBSERVED", Reason: "RESTRICTED_NATIVE_PREFLIGHT", EpochNS: time.Now().UnixNano()}
	} else {
		report.Native = diagnostic.Source{State: "UNVERIFIED", Reason: legacy.Native.ErrorClass}
		report.AddIssue("native", legacy.Native.ErrorClass, "project", "UNVERIFIED")
	}
	// Only compatibility preflight; D1 does not certify full health. Existing
	// status remains unchanged; D2 expands shared classification and capacity.
	legacy.Ready = legacy.MCPReady && legacy.Native.Ready && report.Installation.Mode != "CONFLICT"
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e = enc.Encode(struct {
		control.HostStatus
		Diagnostic diagnostic.Report `json:"diagnostic"`
	}{legacy, report}); e != nil {
		return e
	}
	if !legacy.Ready {
		return errors.New("preflight incomplete")
	}
	return nil
}
