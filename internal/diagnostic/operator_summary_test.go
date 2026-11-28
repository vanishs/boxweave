package diagnostic

import (
	"github.com/vanishs/boxweave/internal/install"
	"testing"
	"time"
)

func TestH3BOperatorSummaryAuthorityAndHealth(t *testing.T) {
	r := NewReport("user-1001")
	r.Installation = install.DiagnosticInstallation{Mode: "SCOPED_V2", Runtime: "VERIFIED", ServiceUnit: "ACTIVE"}
	r.Database.State = "OBSERVED"
	r.Native.State = "OBSERVED"
	r.MCP.State = "OBSERVED"
	r.ForwardDevices.State = "OBSERVED"
	r.ForwardPolicy.State = "OBSERVED"
	r.RootPolicy.State = "OBSERVED"
	if got := Summarize(r, 0); got.Status != "HEALTHY_OBSERVED" || got.Maintenance != "SEPARATE_REVIEW_REQUIRED" {
		t.Fatal(got)
	}
	r.AddRoutedIssue("network", "LEGACY_IPV4_RETIREMENT_PENDING", "ledger", "IN_PROGRESS", "INSPECT_LEGACY_RETIREMENT")
	if got := Summarize(r, 0); got.Status != "HEALTHY_OBSERVED" || got.PendingIssues != 1 {
		t.Fatal(got)
	}
	r.AddRoutedIssue("installation", "MODE_EVIDENCE_INCOMPLETE", "installation", "COVERAGE_LIMIT", "LOCAL_ROOT_OBSERVATION")
	if got := Summarize(r, 1001); got.Status != "UNVERIFIED" || got.ConfirmedIssues != 0 || got.UnverifiedIssues != 0 || got.CoverageLimits != 1 {
		t.Fatal(got)
	}
	p := NewRecoverPlan(r, IssueID(r.Scope, "installation", "MODE_EVIDENCE_INCOMPLETE", "installation"), 1001, time.Now())
	if p.CanRepair || p.ApplyAvailable || p.Reason != "ROOT_COVERAGE_REQUIRED_NOT_A_CONFIRMED_FAULT" {
		t.Fatal(p)
	}
	r.Installation.ServiceUnit = "FAILED"
	if got := Summarize(r, 0); got.Status != "BLOCKED" {
		t.Fatal(got)
	}
	r.AddRoutedIssue("root", "SCOPED_LIFECYCLE_FAILED", "root-helper", "CONFIRMED", "INSPECT_ROOT_EVIDENCE")
	p = NewRecoverPlan(r, IssueID(r.Scope, "root", "SCOPED_LIFECYCLE_FAILED", "root-helper"), 0, time.Now())
	if p.ApplyAvailable || p.CanRepair || p.Reason != "SERVICE_RESTORATION_NOT_PROVEN_SAFE" {
		t.Fatal(p)
	}
}

func TestP3R2SummaryNeverCertifiesUnsupportedOrMissingRoot(t *testing.T) {
	r := NewReport("user-1001")
	r.Installation = install.DiagnosticInstallation{Mode: "SCOPED_V2", Runtime: "VERIFIED", ServiceUnit: "ACTIVE"}
	for _, s := range []*Source{&r.Database, &r.Native, &r.MCP, &r.ForwardDevices, &r.ForwardPolicy} {
		s.State = "OBSERVED"
	}
	if got := Summarize(r, 0); got.Status != "UNVERIFIED" {
		t.Fatal(got)
	}
	r.RootPolicy.State = "OBSERVED"
	r.Installation.Mode = "UNSUPPORTED_LEGACY"
	r.ClassifyInstallation()
	if got := Summarize(r, 0); got.Status != "BLOCKED" || got.ConfirmedIssues != 1 {
		t.Fatal(got)
	}
}
