package install

import (
	"strings"
	"testing"
)

// Historical unit bytes can still be recognized, but Scoped never installs
// or depends on the obsolete retirement service.
func TestIPv4RetirementInstallerUnitIsolation(t *testing.T) {
	p := Plan{Project: "user-1001", RetirementTarget: &NativePrepareTarget{Project: "user-1001", Network: "bwbr0", ACL: "bw-peer"}}
	if strings.Contains(unit(p), nativeRetirementUnitName) || !strings.Contains(unit(p), "Wants="+scopedLifecycleUnitName) {
		t.Fatal("legacy retirement dependency must not be installed")
	}
	p.RetirementTarget = nil
	if strings.Contains(unit(p), nativeRetirementUnitName) {
		t.Fatal("legacy manual install gained root scope")
	}
	cfg := NativePeerRootScope{Approved: true, RetirementReview: true, RetirementCompact: true, DBPath: nativePeerRootInstalledDB, Project: "user-1001", Bridge: "bwbr0", ACL: "bw-peer"}
	body, err := NativePeerRootUnitText(cfg)
	if err != nil || !strings.Contains(body, "PartOf=boxweave.service") || !strings.Contains(body, "BindsTo=boxweave.service") {
		t.Fatal("historical unit recognition failed", err)
	}
}
