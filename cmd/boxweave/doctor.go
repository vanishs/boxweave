package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

// Derive only the fixed installed restricted account. No project discovery
// from arbitrary DB/socket or use of root's own user-0 namespace.
func resolveDoctorScope(a doctorArgs) (doctorArgs, error) {
	if a.Project != "" {
		return a, nil
	}
	if a.DB != "/var/lib/boxweave/ledger.sqlite" || a.Socket != "/var/lib/incus/unix.socket.user" || a.ServiceUser != "boxweave" {
		return a, errors.New("explicit -project required for nonstandard diagnostic inputs")
	}
	u, e := user.Lookup("boxweave")
	if e != nil {
		return a, errors.New("installed service account unavailable; specify exact -project for partial inspection")
	}
	uid, e := strconv.Atoi(u.Uid)
	if e != nil || uid <= 0 || os.Geteuid() != 0 && os.Geteuid() != uid {
		return a, errors.New("exact restricted service account required for implicit project")
	}
	a.Project = "user-" + strconv.Itoa(uid)
	// This only selects a candidate. The existing manifest and runtime
	// observation independently verify actual installation identity.
	return a, nil
}

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

// Worker accepts no caller-supplied native scope, file path, shell or
// credential. It independently captures the installed restricted ledger.
func doctorNativeWorker() error {
	p := doctorInstallPlan(doctorArgs{Project: "user-" + strconv.Itoa(os.Geteuid()), ServiceUser: "boxweave", Socket: "/var/lib/incus/unix.socket.user"})
	uid, _, e := install.ServiceIdentity(p)
	if e != nil || uid != os.Geteuid() || uid == 0 {
		return errors.New("restricted installed diagnostic actor required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	r := native.Reader{Project: p.Project, Socket: p.Socket}
	st := doctorNativeActive(ctx, r)
	snap, e := state.OpenDiagnosticSnapshot(ctx, "/var/lib/boxweave/ledger.sqlite")
	var l *state.DiagnosticLedger
	if e == nil {
		defer snap.Close()
		if v, e := snap.ReadLedger(ctx, p.Project, "", "incusbr-"+strconv.Itoa(uid)); e == nil {
			l = &v
		}
	}
	return json.NewEncoder(os.Stdout).Encode(diagnostic.ReadNativeObservation(ctx, r, st, snap, l))
}
func doctorNative(ctx context.Context, a doctorArgs, identity install.DiagnosticInstallation, snap *state.DiagnosticSnapshot, l *state.DiagnosticLedger) diagnostic.NativeObservation {
	r := native.Reader{Socket: a.Socket, Project: a.Project}
	if os.Geteuid() != 0 {
		return diagnostic.ReadNativeObservation(ctx, r, doctorNativeActive(ctx, r), snap, l)
	}
	unavailable := func(code string) diagnostic.NativeObservation {
		return diagnostic.ReadNativeObservation(ctx, r, native.Status{Project: a.Project, ErrorClass: code}, nil, nil)
	}
	if identity.InstalledIdentity != "VERIFIED" || a.DB != "/var/lib/boxweave/ledger.sqlite" || a.ServiceUser != "boxweave" || a.Socket != "/var/lib/incus/unix.socket.user" {
		return unavailable("RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED")
	}
	p := doctorInstallPlan(a)
	uid, gid, e := install.ServiceIdentity(p)
	if e != nil || a.Project != "user-"+strconv.Itoa(uid) {
		return unavailable("RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED")
	}
	g, e := user.LookupGroup("incus")
	if e != nil {
		return unavailable("RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED")
	}
	ig, e := strconv.ParseUint(g.Gid, 10, 32)
	if e != nil || ig == 0 {
		return unavailable("RESTRICTED_DIAGNOSTIC_ACTOR_UNVERIFIED")
	}
	cmd := exec.CommandContext(ctx, "/proc/self/exe", "doctor-native-worker")
	cmd.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid), Groups: recoveryGroups(uint32(gid), uint32(ig))}}
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/home/boxweave", "XDG_CONFIG_HOME=/home/boxweave/.config", "LANG=C"}
	b, e := cmd.Output()
	var out diagnostic.NativeObservation
	if e != nil || len(b) > 262144 || json.Unmarshal(b, &out) != nil || out.Status.Project != a.Project {
		return unavailable("RESTRICTED_DIAGNOSTIC_READ_UNAVAILABLE")
	}
	return out
}

func doctorMCP(ctx context.Context, identity install.DiagnosticInstallation) diagnostic.Source {
	if identity.Runtime != "VERIFIED" || identity.ServiceUnit != "ACTIVE" {
		return diagnostic.Source{State: "UNVERIFIED", Reason: "CONTROLLER_RUNTIME_UNVERIFIED"}
	}
	client := &http.Client{Timeout: time.Second, Transport: &http.Transport{Proxy: nil}, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:18650/mcp", strings.NewReader("{}"))
	if e != nil {
		return diagnostic.Source{State: "UNVERIFIED", Reason: "LOCAL_MCP_UNAVAILABLE"}
	}
	resp, e := client.Do(req)
	if e != nil {
		return diagnostic.Source{State: "UNVERIFIED", Reason: "LOCAL_MCP_UNAVAILABLE"}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	if resp.StatusCode != http.StatusUnauthorized {
		return diagnostic.Source{State: "UNVERIFIED", Reason: "LOCAL_MCP_UNEXPECTED_RESPONSE"}
	}
	return diagnostic.Source{State: "OBSERVED", Reason: "LOCAL_HTTP_401_ONLY_AUTHENTICATED_MCP_NOT_CHECKED", EpochNS: time.Now().UnixNano()}
}

func collectDoctor(ctx context.Context, a doctorArgs) (control.HostStatus, diagnostic.Report) {
	report := diagnostic.NewReport(a.Project)
	legacy := control.HostStatus{Version: version, Commit: commit, GoVersion: report.GoVersion}
	instctx, instcancel := context.WithTimeout(ctx, 5*time.Second)
	report.Installation = install.ObserveDiagnosticInstallation(instctx, doctorInstallPlan(a), a.DB)
	report.ClassifyInstallation()
	instcancel()
	if report.Installation.Mode != "LEGACY" && report.Installation.Mode != "SCOPED_V2" {
		reason := report.Installation.Reason
		if reason == "" {
			reason = "ARCHITECTURE_UNVERIFIED"
		}
		if os.Geteuid() != 0 && report.Installation.RootProof == "ROOT_AUTHORITY_REQUIRED" && report.Installation.InstalledIdentity == "VERIFIED" && report.Installation.Runtime == "VERIFIED" {
			report.AddRoutedIssue("installation", reason, "installation", "COVERAGE_LIMIT", "LOCAL_ROOT_OBSERVATION")
		} else {
			report.AddIssue("installation", reason, "installation", "UNVERIFIED")
		}
	}
	// Independent source budgets ensure a missing/busy database never hides
	// available native or installation observations.
	dbctx, dbcancel := context.WithTimeout(ctx, 5*time.Second)
	snap, e := state.OpenDiagnosticSnapshot(dbctx, a.DB)
	var ledger *state.DiagnosticLedger
	bridge := "incusbr-" + strings.TrimPrefix(a.Project, "user-")
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
			if l, err := snap.ReadLedger(dbctx, a.Project, report.Installation.Mode, bridge); err == nil {
				ledger = &l
				report.ClassifyLedger(l)
				report.ActivityCoverage.EpochNS = snap.EpochNS
				legacy.AddressRetirement = l.LegacyRetirement
				if l.LegacyRetirement != nil {
					legacy.AddressRetirementPolicy = "LEGACY_IPV4_RETIREMENT"
				}
				if l.PeerCapacity != nil {
					legacy.AddressRetirementPolicy = "SCOPED_RETAIN_IDENTITIES"
					legacy.ScopedPeerCapacity = l.PeerCapacity
					legacy.ScopedIdentityCapacity = l.ScopedIdentityCapacity
				}
			} else {
				report.AddRoutedIssue("database", "LEDGER_OBSERVATION_UNVERIFIED", "ledger", "UNVERIFIED", "INSPECT_EVIDENCE")
			}
			if report.Installation.Mode == "SCOPED_V2" {
				summary, e := snap.ScopedLedgerSummary(dbctx, a.Project, "incusbr-"+strings.TrimPrefix(a.Project, "user-"))
				if e == nil {
					report.ScopedLedger = &summary
					if ledger != nil && ledger.RetiredLimit > 0 && summary.Retired >= ledger.RetiredLimit {
						report.AddRoutedIssue("capacity", "SCOPED_RETIRED_IDENTITY_LIMIT", "ledger", "CONFIRMED", "NO_RESET_OR_COMPACTION")
					}
				} else {
					report.AddIssue("database", "SCOPED_LEDGER_OBSERVATION_UNVERIFIED", "ledger", "UNVERIFIED")
				}
			}

		} else {
			report.Database = diagnostic.Source{State: "UNVERIFIED", Reason: "DIAGNOSTIC_DB_SCHEMA_OR_INTEGRITY_UNVERIFIED"}
			report.AddIssue("database", report.Database.Reason, "ledger", "UNVERIFIED")
		}
	}
	dbcancel()
	// Native and ledger capture are distinct sources. Re-read the bounded
	// ledger after native I/O; never advertise mixed epochs as all-clear.
	nctx, ncancel := context.WithTimeout(ctx, 13*time.Second)
	obs := doctorNative(nctx, a, report.Installation, snap, ledger)
	ncancel()
	if snap != nil {
		if e = snap.Close(); e != nil {
			report.Database.State = "UNVERIFIED"
			report.Database.Reason = "PRIVATE_CAPTURE_CLEANUP_UNVERIFIED"
			legacy.MCPReady = false
			report.AddIssue("database", report.Database.Reason, "ledger", "UNVERIFIED")
		}
	}
	legacy.Native = obs.Status
	if ledger != nil && obs.LedgerDigest != "" {
		reCtx, reCancel := context.WithTimeout(ctx, 3*time.Second)
		after, err := state.OpenDiagnosticSnapshot(reCtx, a.DB)
		same := false
		if err == nil {
			latest, err := after.ReadLedger(reCtx, a.Project, report.Installation.Mode, bridge)
			same = err == nil && latest.ObservationDigest() == ledger.ObservationDigest() && obs.LedgerDigest == ledger.ObservationDigest()
			if after.Close() != nil {
				same = false
			}
		}
		reCancel()
		if !same {
			obs.Inventory = diagnostic.Source{State: "UNVERIFIED", Reason: "NETWORK_LEDGER_CHANGED_OR_RECHECK_UNAVAILABLE"}
			obs.InventoryInfo = nil
			obs.Issues = nil
			obs.Forward = diagnostic.Source{State: "UNVERIFIED", Reason: "LEDGER_EPOCH_CHANGED"}
			obs.ForwardVerified = 0
			obs.AddressCapacity = nil
			obs.AddressCapacitySource = diagnostic.Source{State: "UNVERIFIED", Reason: "LEDGER_EPOCH_CHANGED"}
		}
	}
	report.ClassifyNative(obs)
	if obs.AddressCapacity != nil {
		legacy.AddressCapacity = obs.AddressCapacity
		legacy.AddressCapacityStatus = "OBSERVED"
		if obs.AddressCapacity.RemainingIPv4UpperBound == 0 {
			legacy.AddressCapacityStatus = "EXHAUSTED"
		}
	} else {
		legacy.AddressCapacityStatus = "UNVERIFIED"
	}
	rootctx, rootcancel := context.WithTimeout(ctx, 4*time.Second)
	report.RootPolicy = install.ObserveDiagnosticRootPolicy(rootctx, a.Project, report.Installation)
	rootcancel()
	if ledger != nil && ledger.PeerCapacity != nil && report.RootPolicy.Sequence != nil && report.RootPolicy.ActiveDirections != nil && (*report.RootPolicy.Sequence != ledger.PeerCapacity.CompletedLedgerEvents || *report.RootPolicy.ActiveDirections != ledger.PeerCapacity.ActiveDirections) {
		report.AddRoutedIssue("root", "ROOT_SQLITE_POLICY_EPOCH_OR_OUTCOME_UNVERIFIED", "root-policy", "UNVERIFIED", "MCP_INSPECT_OR_ROOT_EVIDENCE_REQUIRED")
	}
	if report.RootPolicy.Conntrack == "MISSING" {
		if report.Installation.Mode == "SCOPED_V2" {
			report.AddRoutedIssue("installation", "SCOPED_CONNTRACK_MISSING", "conntrack", "CONFIRMED", "LOCAL_DEPENDENCY_REVIEW")
		} else {
			report.AddRoutedIssue("installation", "CURRENT_SCOPED_PREREQUISITE_NOT_PRESENT_MODE_UNVERIFIED", "conntrack", "UNVERIFIED", "INSPECT_INSTALLED_VERSION")
		}
	}
	if report.RootPolicy.State == "JOURNAL_INCOMPLETE_OR_UNVERIFIED" {
		report.AddRoutedIssue("root", "SCOPED_POLICY_JOURNAL_UNVERIFIED", "root-policy", "UNVERIFIED", "INSPECT_ROOT_EVIDENCE")
	}
	if report.RootPolicy.NewGrantHistoryAvailable != nil && !*report.RootPolicy.NewGrantHistoryAvailable {
		report.AddRoutedIssue("capacity", "ROOT_POLICY_HISTORY_RESERVE", "root-policy", "CONFIRMED", "NO_RESET_OR_COMPACTION")
	}
	if report.RootPolicy.State == "OBSERVED" && report.RootPolicy.Kernel != "OBSERVED" {
		report.AddRoutedIssue("root", "SCOPED_KERNEL_GUARD_UNVERIFIED", "root-guard", "UNVERIFIED", "INSPECT_ROOT_EVIDENCE")
	}
	report.MCP = doctorMCP(ctx, report.Installation)
	if report.MCP.State != "OBSERVED" {
		report.AddRoutedIssue("service", report.MCP.Reason, "mcp", "UNVERIFIED", "INSPECT_EVIDENCE")
	}
	policy, verified := install.ObserveDiagnosticForwardPolicy(ctx, doctorInstallPlan(a), report.Installation)
	report.ForwardPolicy = diagnostic.Source{State: "UNVERIFIED", Reason: "unverified"}
	if verified {
		report.ForwardPolicy = diagnostic.Source{State: "OBSERVED", Reason: policy, EpochNS: time.Now().UnixNano()}
	}
	legacy.Native.ExternalForwardPolicy = report.ForwardPolicy.Reason
	if verified && policy == "invalid" {
		report.AddRoutedIssue("forward", "EFFECTIVE_FORWARD_POLICY_INVALID", "controller", "CONFIRMED", "LOCAL_CONFIGURATION_REVIEW")
	}
	warnings := legacy.Native.Warnings[:0]
	for _, w := range legacy.Native.Warnings {
		if !strings.HasPrefix(w, "FORWARD_EXTERNAL_POLICY_DENIED:") {
			warnings = append(warnings, w)
		}
	}
	legacy.Native.Warnings = warnings

	if legacy.Native.Ready {
		report.Native = diagnostic.Source{State: "OBSERVED", Reason: "RESTRICTED_NATIVE_PREFLIGHT", EpochNS: time.Now().UnixNano()}
	} else {
		report.Native = diagnostic.Source{State: "UNVERIFIED", Reason: legacy.Native.ErrorClass}
		report.AddIssue("native", legacy.Native.ErrorClass, "project", "UNVERIFIED")
	}
	// Compatibility preflight only: partial evidence never certifies full health.
	legacy.Ready = legacy.MCPReady && legacy.Native.Ready && report.Installation.Mode != "CONFLICT"
	return legacy, report
}

func runDoctor(a doctorArgs, summary bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	legacy, report := collectDoctor(ctx, a)
	report.Operator = diagnostic.Summarize(report, os.Geteuid())
	if legacy.CapacityNeedsReview() {
		report.Operator.Next = append(report.Operator.Next, "Review capacity budgets before new allocations; do not clear history or recycle addresses manually.")
	}
	if summary {
		fmt.Fprintf(os.Stderr, "Doctor %s: %s; installation=%s; runtime=%s; evidence=%s\n", report.Scope, report.Operator.Status, report.Installation.Mode, report.Installation.Runtime, report.Coverage)
		if report.ScopedLedger != nil {
			fmt.Fprintf(os.Stderr, "Owned Boxes=%d; retained identities=%d\n", report.ScopedLedger.Owned, report.ScopedLedger.Retired)
		}
		if legacy.AddressCapacity != nil {
			fmt.Fprintf(os.Stderr, "IPv4 %s: held=%d; remaining upper bound=%d; %s\n", legacy.AddressCapacity.IPv4Prefix, legacy.AddressCapacity.BlockedIPv4, legacy.AddressCapacity.RemainingIPv4UpperBound, legacy.AddressCapacity.Level)
			if legacy.AddressCapacity.DurableHeldIPv4 != nil && legacy.AddressCapacity.NativeOnlyBlockedIPv4 != nil {
				fmt.Fprintf(os.Stderr, "IPv4 durable held=%d; native-only blocked=%d\n", *legacy.AddressCapacity.DurableHeldIPv4, *legacy.AddressCapacity.NativeOnlyBlockedIPv4)
			}
		}
		if p := legacy.ProjectCapacity; p != nil {
			fmt.Fprintf(os.Stderr, "Live Box slots: owned=%d pending=%d remaining=%d; %s\n", p.Owned, p.PendingCreates, p.Remaining, p.Level)
		}
		if p := legacy.ScopedIdentityCapacity; p != nil {
			fmt.Fprintf(os.Stderr, "Scoped identities: retired=%d owned=%d pending=%d new remaining=%d; %s\n", p.Retired, p.Owned, p.PendingCreates, p.RemainingNewIdentities, p.Level)
		}
		if p := legacy.ScopedPeerCapacity; p != nil {
			fmt.Fprintf(os.Stderr, "Peer directions: active=%d revoked=%d pending/unknown=%d remaining=%d; %s; grant-history upper bound=%d reserved revokes=%d; %s\n", p.ActiveDirections, p.RevokedDirections, p.PendingOrUnknown, p.RemainingDirections, p.DirectionLevel, p.RemainingGrantEventsUpperBound, p.ReservedRevokeEvents, p.GrantHistoryLevel)
		}
		for _, next := range report.Operator.Next {
			fmt.Fprintln(os.Stderr, next)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(struct {
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
