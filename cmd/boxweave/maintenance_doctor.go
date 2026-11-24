package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"time"

	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
)

// Maintenance is intentionally separate from the compatibility readiness
// status and Recover's issue selector. No approval, readiness or effect
// permit can be derived from this non-atomic observation.
type maintenanceReport struct {
	RootLedgerComparison string `json:"root_ledger_comparison"`

	Version           int                             `json:"version"`
	Scope             string                          `json:"scope"`
	ObserverVersion   string                          `json:"observer_version"`
	ObserverCommit    string                          `json:"observer_commit"`
	CrossSourceAtomic bool                            `json:"cross_source_atomic"`
	Coverage          string                          `json:"coverage"`
	Installation      install.MaintenanceInstallation `json:"installation"`
	Database          diagnostic.Source               `json:"database"`
	Ledger            *state.MaintenanceLedger        `json:"ledger,omitempty"`
	Native            install.MaintenanceNative       `json:"native"`
}

func maintenanceSelfSHA() string {
	f, e := os.Open("/proc/self/exe")
	if e != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, (128<<20)+1))
	if e != nil || n > 128<<20 || n == 0 {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
func captureMaintenanceLedger(ctx context.Context, db, project string) (*state.MaintenanceLedger, diagnostic.Source) {
	c, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	snap, e := state.OpenDiagnosticSnapshot(c, db)
	if e != nil {
		return nil, diagnostic.Source{State: "UNVERIFIED", Reason: diagnostic.DatabaseReason(e)}
	}
	l, e := snap.ReadMaintenanceLedger(c, project)
	ce := snap.Close()
	if e != nil {
		return nil, diagnostic.Source{State: "UNVERIFIED", Reason: diagnostic.DatabaseReason(e)}
	}
	if ce != nil {
		return nil, diagnostic.Source{State: "UNVERIFIED", Reason: "PRIVATE_CAPTURE_CLEANUP_UNVERIFIED"}
	}
	return &l, diagnostic.Source{State: "OBSERVED", Reason: snap.Capture, EpochNS: snap.EpochNS}
}
func collectMaintenance(ctx context.Context, a doctorArgs) maintenanceReport {
	o := maintenanceReport{Version: 1, Scope: a.Project, ObserverVersion: version, ObserverCommit: commit, Coverage: "PARTIAL", RootLedgerComparison: "UNVERIFIED", Database: diagnostic.Source{State: "NOT_CHECKED", Reason: "INSTALLED_ADAPTER_REQUIRED"}}
	o.Native.Evidence = install.MaintenanceEvidence{State: "NOT_CHECKED", Reason: "INSTALLED_ADAPTER_REQUIRED"}
	p := doctorInstallPlan(a)
	self := maintenanceSelfSHA()
	instctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	o.Installation = install.ObserveMaintenanceInstallation(instctx, p, a.DB, self, commit)
	cancel()
	if o.Installation.Adapter == "UNSUPPORTED" || o.Installation.Mode == "UNVERIFIED" {
		return o
	}
	var l *state.MaintenanceLedger
	d := diagnostic.Source{State: "UNVERIFIED", Reason: "LEDGER_ARTIFACT_UNSAFE"}
	for _, a0 := range o.Installation.ArtifactEvidence {
		if a0.ID == "ledger" && (a0.Evidence.State == "PRESENT" || a0.Evidence.State == "ABSENT") {
			l, d = captureMaintenanceLedger(ctx, a.DB, a.Project)
		}
	}

	o.Ledger, o.Database = l, d

	nc, stop := context.WithTimeout(ctx, 15*time.Second)
	o.Native = install.ObserveMaintenanceNative(nc, p, o.Installation, l)
	stop()
	// Bind separate captures; refuse to combine changing source epochs.
	if l == nil {
		return o
	}
	after, _ := captureMaintenanceLedger(ctx, a.DB, a.Project)
	if after == nil || after.ObservationDigest() != l.ObservationDigest() {
		o.Native = install.MaintenanceNative{Evidence: install.MaintenanceEvidence{State: "UNVERIFIED", Reason: "LEDGER_EPOCH_CHANGED_OR_RECHECK_UNAVAILABLE"}}
		o.Database.State = "PARTIAL"
		o.Database.Reason = "LEDGER_CHANGED_OR_RECHECK_UNAVAILABLE"
		return o
	}
	ic, stop := context.WithTimeout(ctx, 8*time.Second)
	again := install.ObserveMaintenanceInstallation(ic, p, a.DB, self, commit)
	stop()
	// Ignore dynamic runtime identity and service state? No: any such
	// change is useful evidence of drift and invalidates combined coverage.
	if !reflect.DeepEqual(o.Installation, again) {
		o.Native = install.MaintenanceNative{Evidence: install.MaintenanceEvidence{State: "UNVERIFIED", Reason: "INSTALLATION_EPOCH_CHANGED_OR_RECHECK_UNAVAILABLE"}}
		return o
	}
	rp := o.Installation.RootPolicy
	if o.Installation.Mode == "LEGACY" {
		o.RootLedgerComparison = "NOT_APPLICABLE"
	}
	if o.Installation.Mode == "SCOPED_V2" && rp.State == "OBSERVED" && rp.Sequence != nil && rp.ActiveDirections != nil && l.PeerCapacity != nil {
		o.RootLedgerComparison = "MISMATCH"
		if *rp.Sequence == l.PeerCapacity.CompletedLedgerEvents && *rp.ActiveDirections == l.PeerCapacity.ActiveDirections {
			o.RootLedgerComparison = "MATCH"
		}
	}
	if o.Native.Evidence.State == "OBSERVED" && o.Installation.Preimage.State == "OBSERVED" && l.ForeignRows == 0 {
		o.Coverage = "BOUNDED_SOURCES_OBSERVED"
	}
	return o
}

func runMaintenanceDoctor(a doctorArgs) error {
	if os.Geteuid() != 0 || !state.SafeDiagnosticProject(a.Project) || a.DB != "/var/lib/boxweave/ledger.sqlite" || a.Socket != "/var/lib/incus/unix.socket.user" || a.ServiceUser != "boxweave" {
		return errors.New("maintenance observation requires local root and exact installed scope")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	o := collectMaintenance(ctx, a)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if e := enc.Encode(o); e != nil {
		return e
	}
	if o.Coverage != "BOUNDED_SOURCES_OBSERVED" {
		return errors.New("maintenance observation incomplete; no effects authorized")
	}
	return nil
}
