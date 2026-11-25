package main

import (
	"github.com/vanishs/boxweave/internal/diagnostic"
	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/state"
	"strings"
	"testing"
)

func TestH3BExactApproval(t *testing.T) {
	sha := strings.Repeat("a", 64)
	for _, x := range []string{"NO\n", "RECOVER " + strings.Repeat("b", 64) + "\n", "RECOVER " + sha, "RECOVER " + sha + " extra\n", strings.Repeat("x", 4096)} {
		if readGuidedApproval(strings.NewReader(x), sha) == nil {
			t.Fatal("unsafe approval")
		}
	}
	if e := readGuidedApproval(strings.NewReader("RECOVER "+sha+"\n"), sha); e != nil {
		t.Fatal(e)
	}
	a := doctorArgs{DB: "/tmp/custom", Socket: "/var/lib/incus/unix.socket.user", ServiceUser: "boxweave"}
	if _, e := resolveDoctorScope(a); e == nil {
		t.Fatal("implicit custom scope")
	}
	a.Project = "user-1001"
	if b, e := resolveDoctorScope(a); e != nil || b != a {
		t.Fatal(b, e)
	}
}
func TestH3BMaintenanceProjectionNoAuthority(t *testing.T) {
	r := maintenanceReport{Installation: install.MaintenanceInstallation{Adapter: "PINNED_RC13", Mode: "LEGACY", Identity: install.DiagnosticInstallation{Runtime: "UNVERIFIED"}, Preimage: install.MaintenanceEvidence{State: "OBSERVED"}}, Coverage: "BOUNDED_SOURCES_OBSERVED", Database: diagnostic.Source{State: "OBSERVED"}, LedgerSHA256: strings.Repeat("a", 64), Ledger: &state.MaintenanceLedger{Originals: []state.MaintenanceOriginal{{ID: strings.Repeat("a", 32), State: "UNKNOWN", Action: "box_stop"}}}}
	r.Native.Unowned = 1
	x := summarizeMaintenance(r)
	if x.Retirement != "BLOCKED" || x.Route != "OFFLINE_RETIREMENT_REVIEW" || x.ApplyAvailable || x.Complete {
		t.Fatal(x)
	}
	found := map[string]bool{}
	for _, b := range x.Blockers {
		found[b.Code] = true
	}
	for _, c := range []string{"ORIGINAL_UNKNOWN", "NATIVE_IDENTITY_CONFLICT", "CONTROLLER_RUNTIME_UNVERIFIED", "NATIVE_INVENTORY_UNVERIFIED"} {
		if !found[c] {
			t.Fatal(c, x)
		}
	}
}
