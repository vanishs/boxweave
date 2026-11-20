package install

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
)

// Synthetic entries are inserted ONLY on the ephemeral GitHub Actions Incus
// 7.6 root host. They do not imply any guest is running or a TCP socket is
// actually connected. This tests kernel conntrack deletion and foreign-flow
// preservation, distinct from installed live TCP/UDP revocation acceptance.
type s4b2NativeSeededConntrack struct {
	family, proto, source, target string
	sourcePort, targetPort        int
	foreign                       bool
}

func s4b2NativeConntrackSeedArgs(p s4b2NativeSeededConntrack) []string {
	args := []string{
		// Conntrack 1.4.8 permits --family on -L, but rejects it on
		// -I and -D; infer family from the exact source/dest IPs.
		"-I", "-p", p.proto,
		"-s", p.source, "-d", p.target,
		"-r", p.target, "-q", p.source,
		"--sport", strconv.Itoa(p.sourcePort),
		"--dport", strconv.Itoa(p.targetPort),
		"--reply-port-src", strconv.Itoa(p.targetPort),
		"--reply-port-dst", strconv.Itoa(p.sourcePort),
		"-t", "240", "-u", "SEEN_REPLY",
	}
	if p.proto == "tcp" {
		args = append(args, "--state", "ESTABLISHED")
	}
	return args
}

func s4b2NativeConntrackBoundedCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "/usr/sbin/conntrack", args...)
	cmd.Env = []string{"LC_ALL=C", "PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disposable native conntrack test failed: %w: %s", err, out)
	}
	return nil
}

func s4b2NativeSeedRealKernelConntrack(t *testing.T, ctx context.Context, boxes []native.PeerACLBox) []s4b2NativeSeededConntrack {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" ||
		os.Geteuid() != 0 || len(boxes) != 3 {
		t.Fatal("kernel conntrack synthetic seed forbidden outside single-use root CI")
	}
	a, b, c := boxes[0], boxes[1], boxes[2]
	entries := []s4b2NativeSeededConntrack{
		{"ipv4", "tcp", a.IPv4, b.IPv4, 45301, 45701, false},
		{"ipv4", "udp", b.IPv4, a.IPv4, 45302, 45702, false},
		{"ipv6", "tcp", a.IPv6, b.IPv6, 45303, 45703, false},
		{"ipv6", "udp", b.IPv6, a.IPv6, 45304, 45704, false},
		{"ipv4", "tcp", a.IPv4, c.IPv4, 45311, 45711, true},
		{"ipv6", "udp", c.IPv6, a.IPv6, 45312, 45712, true},
	}
	inserted := make([]s4b2NativeSeededConntrack, 0, len(entries))
	t.Cleanup(func() {
		// Even a partial insertion failure must remove only the fixed
		// source/dest/protocol/port tuples created by this native fixture.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		for _, p := range inserted {
			_ = s4b2NativeConntrackBoundedCommand(cleanupCtx,
				"-D", "-p", p.proto,
				"-s", p.source, "-d", p.target,
				"--sport", strconv.Itoa(p.sourcePort),
				"--dport", strconv.Itoa(p.targetPort))
		}
	})
	for _, p := range entries {
		if err := s4b2NativeConntrackBoundedCommand(ctx, s4b2NativeConntrackSeedArgs(p)...); err != nil {
			t.Fatalf("cannot seed exact disposable %s %s -> %s %s tuple: %v",
				p.family, p.source, p.target, p.proto, err)
		}
		inserted = append(inserted, p)
	}
	for _, p := range entries {
		present, err := nativePeerConntrackList(ctx, nativePeerConntrackOS, p.family, p.source, p.target)
		if err != nil || !present {
			t.Fatalf("native kernel did not confirm freshly seeded %s %s -> %s: found=%v error=%v",
				p.family, p.source, p.target, present, err)
		}
	}
	t.Log("S4_B2_NATIVE_KERNEL_SEEDED_TCP_UDP_DUALSTACK_AND_FOREIGN=PASS")
	return entries
}

func s4b2NativeVerifySeededKernelConntrackAfterPurge(t *testing.T, ctx context.Context, entries []s4b2NativeSeededConntrack) {
	t.Helper()
	foreignCount, targetCount := 0, 0
	for _, p := range entries {
		found, err := nativePeerConntrackList(ctx, nativePeerConntrackOS, p.family, p.source, p.target)
		if err != nil {
			t.Fatalf("post-purge kernel conntrack inventory uncertain for %s %s -> %s: %v", p.family, p.source, p.target, err)
		}
		if p.foreign {
			foreignCount++
			if !found {
				t.Fatalf("bounded root conntrack purge damaged unrelated foreign %s %s -> %s", p.family, p.source, p.target)
			}
			continue
		}
		targetCount++
		if found {
			t.Fatalf("bounded root conntrack purge left tracked pair %s %s -> %s", p.family, p.source, p.target)
		}
	}
	if targetCount != 4 || foreignCount != 2 {
		t.Fatalf("incomplete expected native kernel test coverage: pair=%d foreign=%d", targetCount, foreignCount)
	}
	t.Log("S4_B2_NATIVE_KERNEL_EXACT_PAIR_FOUR_SCOPES_DELETED=PASS")
	t.Log("S4_B2_NATIVE_KERNEL_FOREIGN_TCP_UDP_FLOW_UNTOUCHED=PASS")
}
