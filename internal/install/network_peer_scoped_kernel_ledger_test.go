package install

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Real private SQLite, nft and conntrack; SDK/sysfs/global witnesses are fixtures.
func TestScopedLedgerKernelBoundary(t *testing.T) {
	if os.Getenv("BOXWEAVE_PR178_DIRECTION_NETNS") != "isolated-v1" {
		t.Skip("isolated disposable netns only")
	}
	a, e := os.Readlink("/proc/self/ns/net")
	b, f := os.Readlink("/proc/1/ns/net")
	n, g := net.Interfaces()
	if os.Geteuid() != 0 || e != nil || f != nil || g != nil || a == b || len(n) != 1 || n[0].Name != "lo" {
		t.Fatal("unsafe fixture namespace")
	}
	mode := os.Getenv("BW_LEDGER_KERNEL_CASE")
	if mode != "stable" && mode != "drift" {
		t.Fatal("unknown case")
	}
	ctx := t.Context()
	p, d, sysfs, _ := pr178RootKernelFixture(t)
	ab := p.After[1]
	p.Intent.Allow = false
	p.Before = p.After
	p.After = p.Before[:1]
	p.RevokedAfter = []state.PeerGrantBinding{ab}
	_, root, db, id := scopedInstallStore(t, p)
	read := func(ctx context.Context) (state.ScopedPeerOriginalProof, error) {
		return root.ReadRootScopedPeerOriginal(ctx, id, p.Intent.Project, p.Intent.Bridge)
	}
	proof, e := read(ctx)
	if e != nil {
		t.Fatal(e)
	}
	base, e := nativePeerScopedRootReadSnapshot(ctx, d, proof)
	if e != nil {
		t.Fatal(e)
	}
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
	if e != nil {
		t.Fatal(e)
	}
	for _, acl := range plan.ExpectedBeforeByUUID {
		d.aclByName[acl.Name].NetworkACLPut = acl.NetworkACLPut
	}
	w := &pr178CASDriver{native: d}
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "add", p.Intent.Bridge, "type", "bridge")
	n3Cmd(t, ctx, "/usr/sbin/ip", "link", "set", p.Intent.Bridge, "up")
	m := pr178GuardMembers()
	ep := make([]*n3Endpoint, len(m))
	for i, x := range m {
		ep[i] = n3NewEndpoint(t, ctx, i, x, p.Intent.Bridge)
	}
	if e := nativePeerGuardFixedNftBatch(ctx, plan.GuardBefore.Install); e != nil {
		t.Fatal(e)
	}
	if e := NativePeerGuardKernelReadback(ctx, plan.GuardBefore); e != nil {
		t.Fatal(e)
	}
	for _, i := range []int{0, 2} {
		if !ep[i].call(t, map[string]any{"op": "open", "ip": m[i+1].IPv4, "udp": true, "id": "flow"}) {
			t.Fatal("baseline", i)
		}
	}
	ct, batches := 0, 0
	batch := func(ctx context.Context, s string) error {
		batches++
		if e := nativePeerGuardFixedNftBatch(ctx, s); e != nil {
			return e
		}
		if mode == "drift" {
			r, e := db.ExecContext(ctx, "UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=? AND family='ipv6'", proof.Intent.TargetUUID)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := r.RowsAffected(); e != nil || n != 1 {
				t.Fatal("fixture drift missing", n, e)
			}
		}
		return nil
	}
	dir := t.TempDir()
	rec, e := nativeScopedPeerExecute(ctx, dir, uint32(os.Getuid()), strings.Repeat("b", 64), read, d, w, sysfs, NativePeerGuardKernelReadback, batch, func(ctx context.Context, args ...string) ([]byte, error) {
		ct++
		return nativePeerConntrackOS(ctx, args...)
	}, func(context.Context, state.ScopedPeerOriginalProof, nativePeerScopedSDKSnapshot) (string, error) {
		return strings.Repeat("c", 64), nil
	})
	if mode == "stable" {
		if e != nil || rec.Status != "COMPLETE_VERIFIED" || rec.VerifiedSteps != 4 || ct < 2 || len(w.writes) != 2 {
			t.Fatal(rec, e, ct, w.writes)
		}
	} else {
		if e == nil || rec.Status == "COMPLETE_VERIFIED" || ct != 0 || len(w.writes) != 0 {
			t.Fatal("unsafe continuation", rec, e, ct, w.writes)
		}
		files, e := os.ReadDir(filepath.Join(dir, "peer-"+id))
		if e != nil || len(files) != 1 || files[0].Name() != scopedPeerEffectRecordName(0, false) {
			t.Fatal("partial journal", files, e)
		}
	}
	if batches != 1 {
		t.Fatal(batches)
	}
	if e := NativePeerGuardKernelReadback(ctx, plan.GuardAfter); e != nil {
		t.Fatal(e)
	}
	for _, i := range []int{0, 2} {
		if ep[i].call(t, map[string]any{"op": "probe", "id": "flow", "msg": "after"}) != (i == 2) {
			t.Fatal("unexpected flow", i)
		}
	}
	t.Logf("mixed ledger/kernel %s verified; conntrack=%d synthetic ACL=%d", mode, ct, len(w.writes))
}
