package install

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
)

// A disposable-native RISK REPRODUCTION only. The conntrack kernel has no
// BoxWeave Project identity; same-IP but different-port tuples are ambiguous.
// This helper runs AFTER the original stopped A/B/C journaled root proof.
func s4b2NativeWitnessIPOnlyTupleCollision(t *testing.T, ctx context.Context, src, dst native.PeerACLBox) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" ||
		os.Geteuid() != 0 {
		t.Fatal("IP-only conntrack collision experiment requires disposable root CI")
	}
	found, err := nativePeerConntrackList(ctx, nativePeerConntrackOS, "ipv4", src.IPv4, dst.IPv4)
	if err != nil || found {
		t.Fatalf("collision fixture not empty before seed: found=%v err=%v", found, err)
	}
	entries := []s4b2NativeSeededConntrack{
		{family: "ipv4", proto: "tcp", source: src.IPv4, target: dst.IPv4, sourcePort: 47301, targetPort: 47701},
		{family: "ipv4", proto: "tcp", source: src.IPv4, target: dst.IPv4, sourcePort: 47302, targetPort: 47702},
	}
	inserted := make([]s4b2NativeSeededConntrack, 0, len(entries))
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, p := range inserted {
			_ = s4b2NativeConntrackBoundedCommand(cleanupCtx, "-D", "-p", p.proto,
				"-s", p.source, "-d", p.target,
				"--sport", strconv.Itoa(p.sourcePort), "--dport", strconv.Itoa(p.targetPort))
		}
	})
	for _, e := range entries {
		if err := s4b2NativeConntrackBoundedCommand(ctx, s4b2NativeConntrackSeedArgs(e)...); err != nil {
			t.Fatalf("cannot insert separate fixed-port IPv4 TCP tuple: %v", err)
		}
		inserted = append(inserted, e)
	}
	before, err := nativePeerConntrackOS(ctx, "-L", "-f", "ipv4", "-s", src.IPv4, "-d", dst.IPv4)
	if err != nil || !strings.Contains(string(before), "sport=47301") ||
		!strings.Contains(string(before), "sport=47302") {
		t.Fatalf("kernel did not prove two original IP-identical distinct-port entries: %v, %s", err, before)
	}
	output, err := nativePeerConntrackOS(ctx, "-D", "-s", src.IPv4, "-d", dst.IPv4)
	if err != nil {
		t.Fatalf("IP-only conntrack deletion failed: %v, %s", err, output)
	}
	after, err := nativePeerConntrackList(ctx, nativePeerConntrackOS, "ipv4", src.IPv4, dst.IPv4)
	if err != nil || after {
		t.Fatalf("IP-only deletion did not remove both ports: found=%v err=%v", after, err)
	}
	t.Log("S4_B2_NATIVE_ROOT_IP_ONLY_DELETE_COLLATERAL_SAME_IP_DISTINCT_PORTS=RISK_CONFIRMED")
}
