package install

import (
	"strings"
	"testing"
)

func TestPR178N4DegradedControllerDependency(t *testing.T) {
	p := Plan{ScopedFreshReviewSHA: strings.Repeat("a", 64)}
	u := unit(p)
	if !strings.Contains(u, "Wants="+scopedLifecycleUnitName+"\nAfter="+scopedLifecycleUnitName+"\n") || strings.Contains(u, "Requires="+scopedLifecycleUnitName) || !strings.Contains(u, " -scoped-peer-fresh\n") {
		t.Fatal("controller must retain read/recovery API without weakening scoped mode", u)
	}
	if !strings.Contains(scopedIncusDropInText, "Requires="+scopedGuardUnitName+"\nAfter="+scopedGuardUnitName+"\n") || !strings.Contains(scopedLifecycleUnitText, "Requires=incus.service "+scopedGuardUnitName+"\n") {
		t.Fatal("native boot/helper must still require verified guard")
	}
}
