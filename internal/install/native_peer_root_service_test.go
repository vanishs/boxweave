package install

import (
	"strings"
	"testing"
)

func TestS4B2RootSystemdPinnedOneOperationAndNoShellInjection(t *testing.T) {
	cfg := NativePeerRootScope{
		DBPath: nativePeerRootInstalledDB, Project: "user-1002",
		Bridge: "bwbr0", ACL: "bw-s4b2-native",
		OriginalID:      strings.Repeat("a", 32),
		ApprovedPlanSHA: strings.Repeat("b", 64), Approved: true,
	}
	text, err := NativePeerRootUnitText(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"User=root", "Group=root", "Restart=on-failure",
		"NoNewPrivileges=true", "ProtectHome=true", "ProtectSystem=full",
		"ExecStart=/usr/local/bin/boxweave native-peer-root-serve",
		"-operation-id " + strings.Repeat("a", 32),
		"-peer-plan-sha256 " + strings.Repeat("b", 64),
		" -approve-local",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing tightly scoped unit witness %q", want)
		}
	}
	for _, forbidden := range []string{"ExecStartPre=", "/bin/sh", "EnvironmentFile=", "DynamicUser="} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("unsafe unit expansion %s", forbidden)
		}
	}
	bad := []NativePeerRootScope{}
	x := cfg
	x.Approved = false
	bad = append(bad, x)
	x = cfg
	x.DBPath = "/tmp/root-ledger.sqlite"
	bad = append(bad, x)
	x = cfg
	x.Project = "user-1002; touch /tmp/pwn"
	bad = append(bad, x)
	x = cfg
	x.Bridge = "bridge\nExecStart=/bin/sh"
	bad = append(bad, x)
	x = cfg
	x.ACL = "other/acl"
	bad = append(bad, x)
	x = cfg
	x.OriginalID = "forged"
	bad = append(bad, x)
	x = cfg
	x.ApprovedPlanSHA = strings.Repeat("x", 64)
	bad = append(bad, x)
	for i, x := range bad {
		if _, err := NativePeerRootUnitText(x); err == nil {
			t.Fatalf("unsafe systemd scope %d accepted", i)
		}
	}
	t.Log("S4_B2_ROOT_SYSTEMD_PINNED_SCOPE_AND_INJECTION_DENIAL=PASS")
}
