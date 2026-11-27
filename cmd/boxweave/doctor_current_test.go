package main

import (
	"context"
	"github.com/vanishs/boxweave/internal/control"
	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"path/filepath"
	"testing"
)

func TestP3R2CurrentDoctorPreflight(t *testing.T) {
	i := install.DiagnosticInstallation{
		Candidate: "SCOPED_V2", Mode: "SCOPED_V2", Manifest: "VERIFIED", InstalledIdentity: "VERIFIED", Runtime: "VERIFIED",
		RootProof: "VERIFIED", RootJournal: "PRESENT", ScopedArtifacts: "PRESENT", Dependencies: "OBSERVED",
		ServiceUnit: "ACTIVE", GuardUnit: "ACTIVE", LifecycleUnit: "ACTIVE", StaticContract: "VERIFIED",
	}
	h := control.HostStatus{SchemaVersion: state.SchemaVersion, MCPReady: true, Native: native.Status{Ready: true}}
	r := diagnostic.NewReport("user-1001")
	r.Installation = i
	r.Database.State = "OBSERVED"
	r.MCP.State = "OBSERVED"
	r.Ledger = &state.DiagnosticLedger{}
	if !doctorPreflightReady(h, r) || doctorLedgerMode(i) != "SCOPED_V2" {
		t.Fatal("root current preflight refused")
	}
	restricted := i
	restricted.Mode = "UNVERIFIED"
	restricted.RootProof = "ROOT_AUTHORITY_REQUIRED"
	restricted.StaticContract = "UNVERIFIED"
	r.Installation = restricted
	if !doctorPreflightReady(h, r) || doctorLedgerMode(restricted) != "SCOPED_V2" {
		t.Fatal("restricted current preflight refused")
	}
	for _, tc := range []struct {
		name   string
		change func(*install.DiagnosticInstallation)
	}{
		{"legacy", func(i *install.DiagnosticInstallation) { i.Candidate = "LEGACY"; i.Mode = "UNSUPPORTED_LEGACY" }},
		{"mixed", func(i *install.DiagnosticInstallation) { i.Mode = "CONFLICT" }},
		{"stale", func(i *install.DiagnosticInstallation) { i.Runtime = "UNVERIFIED" }},
		{"foreign", func(i *install.DiagnosticInstallation) { i.InstalledIdentity = "UNVERIFIED" }},
		{"missing guard", func(i *install.DiagnosticInstallation) { i.GuardUnit = "ABSENT" }},
		{"missing proof", func(i *install.DiagnosticInstallation) { i.RootJournal = "UNVERIFIED" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := restricted
			tc.change(&bad)
			r.Installation = bad
			if doctorPreflightReady(h, r) {
				t.Fatal("unsupported scope accepted", bad)
			}
		})
	}
	r.Installation = i
	stopped := i
	stopped.Runtime = "UNVERIFIED"
	stopped.ServiceUnit = "INACTIVE"
	r.Installation = stopped
	if doctorPreflightReady(h, r) || doctorLedgerMode(stopped) != "SCOPED_V2" {
		t.Fatal("stopped installation should retain independent ledger observation, not readiness")
	}
	r.Installation = i
	h.SchemaVersion = state.SchemaVersion - 1
	if doctorPreflightReady(h, r) {
		t.Fatal("stale schema ready")
	}
	h.SchemaVersion = state.SchemaVersion
	r.Ledger = nil
	if doctorPreflightReady(h, r) {
		t.Fatal("missing ledger ready")
	}
	r.Ledger = &state.DiagnosticLedger{}
	r.Ledger.ForeignRows = 1
	if doctorPreflightReady(h, r) {
		t.Fatal("foreign ledger ready")
	}
	r.Ledger.ForeignRows = 0
	r.MCP.State = "UNVERIFIED"
	if doctorPreflightReady(h, r) {
		t.Fatal("missing endpoint ready")
	}
}

func TestP3R2MaintenanceHistoricalRefusal(t *testing.T) {
	l, source := captureMaintenanceLedger(context.Background(), filepath.Join(t.TempDir(), "absent.sqlite"), "user-1001", "LEGACY")
	if l != nil || source.Reason != "CURRENT_SCOPED_LEDGER_REQUIRED" {
		t.Fatal(l, source)
	}
	for _, mode := range []string{"LEGACY", "UNVERIFIED", "CONFLICT"} {
		r := maintenanceReport{Installation: install.MaintenanceInstallation{Adapter: "EXACT_RUNNING_OBSERVER_BINARY", Mode: mode}}
		result := summarizeMaintenance(r)
		found := false
		for _, b := range result.Blockers {
			if b.Code == "CURRENT_SCOPED_INSTALLATION_REQUIRED" && b.Severity == "HARD" {
				found = true
			}
		}
		if !found || result.Retirement != "BLOCKED" || result.ApplyAvailable || result.Complete {
			t.Fatal(result)
		}
	}
}
