package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestS4B2HostKernelBridgePortCensus(t *testing.T) {
	base := t.TempDir()
	brif := filepath.Join(base, "bwbr0", "brif")
	if err := os.MkdirAll(brif, 0755); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgePortsEmptyAt(t.Context(), base, "bwbr0"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "unmanaged"), filepath.Join(brif, "foreign-veth")); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgePortsEmptyAt(t.Context(), base, "bwbr0"); err == nil || !strings.Contains(err.Error(), "live ports") {
		t.Fatalf("foreign raw NIC not rejected: %v", err)
	}
	if err := os.Remove(filepath.Join(brif, "foreign-veth")); err != nil {
		t.Fatal(err)
	}
	if err := nativePeerBridgePortsEmptyAt(t.Context(), base, "bwbr0"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", ".", "..", "../bwbr0", "bwbr0/other"} {
		if err := nativePeerBridgePortsEmptyAt(t.Context(), base, bad); err == nil {
			t.Fatalf("invalid bridge scope accepted %q", bad)
		}
	}
	if err := nativePeerBridgePortsEmptyAt(t.Context(), base, "missing"); err == nil {
		t.Fatal("missing sysfs bridge accepted")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := nativePeerBridgePortsEmptyAt(ctx, base, "bwbr0"); err != context.Canceled {
		t.Fatalf("cancel ignored: %v", err)
	}
}
