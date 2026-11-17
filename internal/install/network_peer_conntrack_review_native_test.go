package install

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/state"
)

// This helper runs ONLY from the existing 3-Box STOPPED native test on an
// independently created disposable root Incus 7.6 CI host. It writes *only*
// a temporary protected test ledger, never a shared Project/ACL.
func s4b2NativeRootReadOnlyPeerReviewFixture(t *testing.T, ctx context.Context,
	root incus.InstanceServer, target NativePeerBridgeAuditTarget) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("BOXWEAVE_DISPOSABLE_NATIVE_PROBE") != "s4b1-ci-20261116" ||
		os.Geteuid() != 0 || len(target.Boxes) != 3 {
		t.Fatal("native peer review must run only in exact disposable stopped fixture")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "root-peer-review.sqlite")
	writer, err := state.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, box := range target.Boxes {
		now := time.Now().UnixNano()
		if box.UUID == "" || box.MAC == "" || box.IPv4 == "" || box.IPv6 == "" {
			t.Fatal("incomplete actual native peer", box)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,'OWNED',?)",
			box.UUID, target.Project, box.Name, box.UUID, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, "INSERT INTO network_bindings(project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at) VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)",
			target.Project, box.UUID, box.UUID, "eth0", target.Bridge, target.ACL, strings.ToLower(box.MAC), box.IPv4, box.IPv6, now); err != nil {
			t.Fatal(err)
		}
		for _, p := range []struct{ family, ip string }{{"ipv4", box.IPv4}, {"ipv6", box.IPv6}} {
			if _, err := db.ExecContext(ctx, "INSERT INTO network_address_fences(bridge,family,address,project,box_uuid,incarnation,state,observed_at) VALUES(?,?,?,?,?,?,'RESERVED',?)",
				target.Bridge, p.family, p.ip, target.Project, box.UUID, box.UUID, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	reader, err := state.OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if !reader.IsNetworkAuditReadOnly() || writer.IsNetworkAuditReadOnly() {
		t.Fatal("root ledger audit reader and service writer privilege confused")
	}
	host := strings.Repeat("a", 64)
	src, dst := target.Boxes[0].UUID, target.Boxes[1].UUID
	if _, err := nativePeerConntrackReviewWithClient(ctx, writer, root, host, target.Project, target.Bridge, target.ACL, src, dst); err == nil {
		t.Fatal("writable controller DB used in root peer review")
	}
	t.Log("S4_B2_ROOT_PEER_REVIEW_WRITABLE_SQLITE_DENIED=PASS")
	plan, err := nativePeerConntrackReviewWithClient(ctx, reader, root, host, target.Project, target.Bridge, target.ACL, src, dst)
	if err != nil {
		t.Fatal("root native read-only conntrack review did not prove complete stopped dual-stack DROP", err)
	}
	if len(plan.PlanSHA256) != 64 || plan.MutationAuthorized || plan.SourceUUID != src || plan.TargetUUID != dst {
		t.Fatal("root review falsely authorizes peer mutation", plan)
	}
	t.Log("S4_B2_ROOT_PEER_REVIEW_PROTECTED_LEDGER_AND_STOPPED_NATIVE_DROP=PASS")
	if _, err := nativePeerConntrackReviewWithClient(ctx, reader, root, host, target.Project, target.Bridge, target.ACL, "foreign-uuid", dst); err == nil {
		t.Fatal("unowned source pair reached root review")
	}
	t.Log("S4_B2_ROOT_PEER_REVIEW_FOREIGN_PAIR_REFUSED=PASS")
	seeded := s4b2NativeSeedRealKernelConntrack(t, ctx, target.Boxes)
	// Real root SDK+kernel conntrack effect, bounded to this disposable
	// 3-Box stopped bridge and the verified protected SQLite ledger.
	journalPath := filepath.Join(dir, "native-peer-original.json")
	nativeDispatches := 0
	receipt, err := nativePeerConntrackDispatchOnce(ctx, journalPath, plan,
		plan.PlanSHA256, true, func(ctx context.Context) (NativePeerConntrackReport, error) {
			nativeDispatches++
			return nativePeerConntrackPurgeFromStoreWithClient(ctx, reader, root, nativePeerConntrackOS,
				target.Project, target.Bridge, target.ACL, src, dst)
		})
	if err != nil || receipt.Status != "COMPLETE_VERIFIED" ||
		!receipt.MutationComplete || receipt.ReadyForPeer ||
		receipt.ProbedScopes != 4 || receipt.DeletedScopes != 4 || nativeDispatches != 1 {
		t.Fatalf("real root-native one-shot incomplete: %+v %v dispatches=%d",
			receipt, err, nativeDispatches)
	}
	actual, err := nativePeerConntrackLoadJournal(journalPath)
	if err != nil || actual.Status != "COMPLETE_VERIFIED" ||
		actual.Plan.PlanSHA256 != plan.PlanSHA256 ||
		actual.Report.NativeDigestSHA256 != plan.NativeDigestSHA256 ||
		actual.Report.ProbedScopes != 4 {
		t.Fatalf("root-native journal not durably bound: %+v %v", actual, err)
	}
	s4b2NativeVerifySeededKernelConntrackAfterPurge(t, ctx, seeded)
	t.Log("S4_B2_ROOT_SQLITE_REAL_INCUS_CONNTRACK_JOURNALED_FOUR_SCOPES=PASS")
	if _, err := nativePeerConntrackDispatchOnce(ctx, journalPath, plan,
		plan.PlanSHA256, true, func(context.Context) (NativePeerConntrackReport, error) {
			nativeDispatches++
			t.Fatal("native effect retried after verified completion")
			return NativePeerConntrackReport{}, nil
		}); err == nil || nativeDispatches != 1 {
		t.Fatalf("root-native dispatch replay: %v count=%d", err, nativeDispatches)
	}
	t.Log("S4_B2_ROOT_SQLITE_REAL_INCUS_CONNTRACK_REPLAY_DENIED=PASS")
	if _, err := nativePeerConntrackPurgeFromStoreWithClient(ctx, writer, root, nativePeerConntrackOS,
		target.Project, target.Bridge, target.ACL, src, dst); err == nil {
		t.Fatal("writable SQLite owner ledger admitted to root-native purge")
	}
	t.Log("S4_B2_ROOT_SQLITE_REAL_INCUS_CONNTRACK_WRITER_DENIED=PASS")
	if _, err := db.ExecContext(ctx, "UPDATE network_bindings SET mac='02:16:3e:fe:fe:fe' WHERE box_uuid=?", dst); err != nil {
		t.Fatal(err)
	}
	if _, err := nativePeerConntrackReviewWithClient(ctx, reader, root, host, target.Project, target.Bridge, target.ACL, src, dst); err == nil {
		t.Fatal("root read-only peer review accepted poisoned persisted MAC")
	}
	t.Log("S4_B2_ROOT_PEER_REVIEW_MAC_DRIFT_REFUSED=PASS")
}
