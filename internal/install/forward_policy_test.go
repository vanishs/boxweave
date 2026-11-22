package install

import (
	"strings"
	"testing"
)

func TestFreshInstallForwardPolicyAndManagementBind(t *testing.T) {
	p := Plan{User: "boxweave", BinaryPath: "/usr/local/bin/boxweave", StateDir: "/var/lib/boxweave", Socket: "/var/lib/incus/unix.socket.user", Project: "user-1001"}
	got := unit(p)
	if strings.Count(got, "\nEnvironment=BOXWEAVE_FORWARD_EXTERNAL_ALLOW=all\n") != 1 {
		t.Fatal("fresh unit must explicitly delegate only Forward address/port selection")
	}
	if !strings.Contains(got, "-listen 127.0.0.1:18650\n") || !strings.Contains(got, "\nUser=boxweave\n") || !strings.Contains(got, "\nNoNewPrivileges=true\n") {
		t.Fatal("fresh Forward policy must not change management listener or runtime privilege")
	}
}
