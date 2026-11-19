package install

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/user"
	"reflect"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// These opt-in test fixtures run ONLY on a fresh dedicated, privileged,
// throwaway GitHub Actions host. They are separate from the installed
// binary's real nonroot-stage -> root-plan -> SO_PEERCRED -> apply -> SQLite
// execution. Neither fixture is callable from an MCP request.
type s4b2InstalledScope struct {
	Project string
	Bridge  string
	ACL     string
	Boxes   []native.PeerACLBox
}

func s4b2InstalledOneUse(t *testing.T, root bool) {
	t.Helper()
	if os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("BOXWEAVE_S4B2_INSTALLED_ONE_USE") != "20261117-controller-root" {
		t.Skip("one-use privileged native installed acceptance only")
	}
	if (os.Geteuid() == 0) != root {
		t.Fatal("one-use fixture wrong operating-system authority")
	}
}

func TestS4B2NativeInstalledRootSeedReviewedGrant(t *testing.T) {
	s4b2InstalledOneUse(t, true)
	path := os.Getenv("BOXWEAVE_S4B2_INSTALLED_SCOPE")
	if path == "" {
		t.Fatal("missing exact disposable fixture output")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	u, err := user.Lookup("boxweave")
	if err != nil {
		t.Fatal(err)
	}
	project := "user-" + u.Uid
	scope := s4b2InstalledScope{Project: project, Bridge: "bwbr0", ACL: "bw-s4b2-native"}
	own := root.UseProject(project)
	for _, name := range []string{"s4b2a", "s4b2b", "s4b2c"} {
		inst, _, err := own.GetInstance(name)
		if err != nil {
			t.Fatal(err)
		}
		if inst.Status != "Running" && inst.Status != "Ready" {
			t.Fatalf("guest %s not running", name)
		}
		dev := inst.Devices["eth0"]
		if dev["ipv4.address"] == "" || dev["ipv6.address"] == "" {
			t.Fatal("missing native static NIC identity")
		}
		mac := strings.ToLower(inst.Config["volatile.eth0.hwaddr"])
		if mac == "" {
			mac = strings.ToLower(inst.ExpandedDevices["eth0"]["hwaddr"])
		}
		if mac == "" || inst.Config["volatile.uuid"] == "" {
			t.Fatal("missing native UUID or MAC")
		}
		scope.Boxes = append(scope.Boxes, native.PeerACLBox{Name: name, UUID: inst.Config["volatile.uuid"],
			MAC: mac, IPv4: dev["ipv4.address"], IPv6: dev["ipv6.address"]})
	}
	nw, _, err := root.GetNetwork(scope.Bridge)
	if err != nil {
		t.Fatal(err)
	}
	before, tag, err := root.GetNetworkACL(scope.ACL)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Ingress) != 0 {
		t.Fatal("unexpected preexisting nonempty ACL")
	}
	a, b := scope.Boxes[0], scope.Boxes[1]
	a4, e1 := netip.ParseAddr(a.IPv4)
	b4, e2 := netip.ParseAddr(b.IPv4)
	a6, e3 := netip.ParseAddr(a.IPv6)
	b6, e4 := netip.ParseAddr(b.IPv6)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		t.Fatal("invalid static addresses")
	}
	expected, err := nativeACLForVerifiedPeers(NativePrepareTarget{Project: project, ACL: scope.ACL},
		nw, []nativePeerBinding{{SourceUUID: a.UUID, TargetUUID: b.UUID, SourceIPv4: a4, TargetIPv4: b4, SourceIPv6: a6, TargetIPv6: b6}})
	if err != nil {
		t.Fatal(err)
	}
	if err := root.UpdateNetworkACL(scope.ACL, expected.NetworkACLPut, tag); err != nil {
		t.Fatal(err)
	}
	actual, _, err := root.GetNetworkACL(scope.ACL)
	if err != nil || !reflect.DeepEqual(actual.Ingress, expected.Ingress) ||
		!reflect.DeepEqual(actual.Egress, expected.Egress) {
		t.Fatalf("original native grant could not be verified: %v", err)
	}
	raw, err := json.Marshal(scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	t.Log("S4_B2_NATIVE_INSTALLED_SEED_EXACT_GRANT_AND_UUID=PASS")
}

func TestS4B2NativeInstalledNonrootSeedOriginal(t *testing.T) {
	s4b2InstalledOneUse(t, false)
	path := os.Getenv("BOXWEAVE_S4B2_INSTALLED_SCOPE")
	dbPath := os.Getenv("BOXWEAVE_S4B2_INSTALLED_DB")
	if path == "" || dbPath == "" {
		t.Fatal("missing preapproved fixture paths")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil {
		t.Fatal(err)
	}
	u, err := user.Lookup("boxweave")
	if err != nil || u.Uid != fmt.Sprint(os.Geteuid()) || scope.Project != "user-"+u.Uid ||
		scope.Bridge != "bwbr0" || scope.ACL != "bw-s4b2-native" || len(scope.Boxes) != 3 {
		t.Fatal("nonroot fixture OS identity or native scope uncertain")
	}
	s, err := state.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	now := time.Now().UnixNano()
	for _, b := range scope.Boxes {
		if _, err := db.ExecContext(ctx, `INSERT INTO owned_boxes
  (uuid,project,name,incarnation,state,created_at)
  VALUES(?,?,?,?,'OWNED',?)`, b.UUID, scope.Project, b.Name, b.UUID, now); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO network_bindings
  (project,box_uuid,incarnation,nic,bridge,acl,mac,ipv4,ipv6,state,observed_at)
  VALUES(?,?,?,?,?,?,?,?,?,'PINNED',?)`,
			scope.Project, b.UUID, b.UUID, "eth0", scope.Bridge, scope.ACL, b.MAC, b.IPv4, b.IPv6, now); err != nil {
			t.Fatal(err)
		}
		for _, p := range []struct{ family, addr string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
			if _, err := db.ExecContext(ctx, `INSERT INTO network_address_fences
   (bridge,family,address,project,box_uuid,incarnation,state,observed_at)
   VALUES(?,?,?,?,?,?,'RESERVED',?)`,
				scope.Bridge, p.family, p.addr, scope.Project, b.UUID, b.UUID, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	_, tok, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a, b := scope.Boxes[0], scope.Boxes[1]
	intent := state.RootPeerRevokeSafeIntent{Project: scope.Project, Bridge: scope.Bridge, ACL: scope.ACL,
		SourceUUID: a.UUID, SourceIncarnation: a.UUID, TargetUUID: b.UUID, TargetIncarnation: b.UUID, Allow: false}
	safe, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	op, err := s.Admit(ctx, state.Admission{Layer: scope.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-installed-root-one-use-20261117-v1", RequestID: "s4b2-installed-root-one-use",
		Action: "box_peer_set", BoxUUID: a.UUID, Incarnation: a.UUID,
		SafeIntent: safe, ResourceKey: "acl:default/" + scope.ACL,
		ResourceKeys: []string{"box:" + a.UUID, "box:" + b.UUID}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO peer_grants
 (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
 desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
 VALUES(?,?,?,?,?,1,'CONFIRMED',?,?,?,?,?)`,
		scope.Project, a.UUID, a.UUID, b.UUID, b.UUID, a.IPv4, b.IPv4, a.IPv6, b.IPv6, ""); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("S4_B2_INSTALLED_ORIGINAL_ID=%s\n", op.ID)
	t.Log("S4_B2_INSTALLED_NONROOT_SQLITE_ORIGINAL_AND_GRANT=PASS")
}

// This root-only disposable regression forces an ambiguous SDK Start on a
// nonexistent member AFTER a successful root revocation. Even if a native
// Start succeeds before losing its SDK ACK, containment must attempt to stop
// every verified sibling, not only members with acknowledged starts.
func TestS4B2NativeInstalledAmbiguousResumeStopsAllSiblings(t *testing.T) {
	s4b2InstalledOneUse(t, true)
	raw, err := os.ReadFile(os.Getenv("BOXWEAVE_S4B2_INSTALLED_SCOPE"))
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil || len(scope.Boxes) != 3 {
		t.Fatalf("missing exact native sibling inventory: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, box := range scope.Boxes {
		inst, _, err := root.UseProject(scope.Project).GetInstance(box.Name)
		if err != nil || inst.Config["volatile.uuid"] != box.UUID ||
			(inst.Status != "Running" && inst.Status != "Ready") || inst.Config["boot.autostart"] != "false" {
			t.Fatalf("unverified pre-test native member %s: %v", box.Name, err)
		}
	}
	a, b := scope.Boxes[0], scope.Boxes[1]
	a4, e1 := netip.ParseAddr(a.IPv4)
	b4, e2 := netip.ParseAddr(b.IPv4)
	a6, e3 := netip.ParseAddr(a.IPv6)
	b6, e4 := netip.ParseAddr(b.IPv6)
	if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
		t.Fatal("invalid protected peer identity")
	}
	pair := nativePeerBinding{SourceUUID: a.UUID, TargetUUID: b.UUID,
		SourceIPv4: a4, TargetIPv4: b4, SourceIPv6: a6, TargetIPv6: b6}
	target := NativePeerBridgeAuditTarget{Project: scope.Project, Bridge: scope.Bridge,
		ACL: scope.ACL, Boxes: scope.Boxes}
	if _, err := nativePeerRootStartVerified(ctx, root, scope.Project, target,
		[]string{"s4b2-not-an-owned-member"}, pair); err == nil {
		t.Fatal("ambiguous unowned resume incorrectly succeeded")
	}
	for _, box := range scope.Boxes {
		inst, _, err := root.UseProject(scope.Project).GetInstance(box.Name)
		if err != nil || inst.Config["volatile.uuid"] != box.UUID ||
			inst.Status != "Stopped" || inst.Config["boot.autostart"] != "false" {
			t.Fatalf("ambiguous native Start left sibling %s running/rearmed: state=%v err=%v", box.Name, inst, err)
		}
	}
	t.Log("S4_B2_INSTALLED_ROOT_AMBIGUOUS_RESUME_ALL_SIBLINGS_STOPPED_NO_REARM=PASS")
}

// Disposable native root fixture: real Incus ETag DROP occurs, then a
// package-private uncertain result stops the single original before conntrack.
func TestS4B2NativeInstalledPostACLRealEffectUnknown(t *testing.T) {
	s4b2InstalledOneUse(t, true)
	path, dbPath := os.Getenv("BOXWEAVE_S4B2_INSTALLED_SCOPE"), os.Getenv("BOXWEAVE_S4B2_INSTALLED_DB")
	original, sha := os.Getenv("BOXWEAVE_S4B2_INSTALLED_ORIGINAL"), os.Getenv("BOXWEAVE_S4B2_INSTALLED_SHA")
	if path == "" || dbPath == "" || len(original) != 32 || len(sha) != 64 {
		t.Fatal("missing exact one-use native original and approved digest")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil || len(scope.Boxes) != 3 {
		t.Fatalf("invalid native scope: %v", err)
	}
	ledger, err := state.OpenNetworkAuditReadOnly(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	before, err := ledger.Get(ctx, original)
	if err != nil || before.State != "DISPATCHING" {
		t.Fatalf("not a claimed controller original: %+v %v", before, err)
	}
	faults := 0
	receipt, err := nativePeerRootIntegratedApplyWithPostACLTestFault(ctx, ledger,
		scope.Project, scope.Bridge, scope.ACL, original, sha, true,
		func() error { faults++; return fmt.Errorf("one-use after real ACL PUT") })
	if err == nil || faults != 1 || receipt.Status != "DISPATCH_UNKNOWN" || receipt.MutationComplete {
		t.Fatalf("real ACL effect was not frozen: %+v %v fault_calls=%d", receipt, err, faults)
	}
	journalPath, err := nativePeerConntrackOperationJournalPath(original)
	if err != nil {
		t.Fatal(err)
	}
	journal, err := nativePeerConntrackLoadJournal(journalPath)
	if err != nil || journal.Status != "DISPATCH_UNKNOWN" || journal.Plan.PlanSHA256 != sha {
		t.Fatalf("root original unexpectedly completed: %+v %v", journal, err)
	}
	root, err := incus.ConnectIncusUnixWithContext(ctx, "/var/lib/incus/unix.socket", nil)
	if err != nil {
		t.Fatal(err)
	}
	acl, _, err := root.GetNetworkACL(scope.ACL)
	if err != nil || acl == nil {
		t.Fatalf("lost native ACL: %v", err)
	}
	if err := nativePeerACLDropCovered(acl, scope.Boxes[0], scope.Boxes[1]); err != nil {
		t.Fatalf("real ACL readback does not prove exact dual-stack DROP: %v", err)
	}
	for _, b := range scope.Boxes {
		inst, _, err := root.UseProject(scope.Project).GetInstance(b.Name)
		if err != nil || inst.Config["volatile.uuid"] != b.UUID || inst.Status != "Stopped" || inst.Config["boot.autostart"] != "false" {
			t.Fatalf("partial native mutation rearmed sibling %s: %v", b.Name, err)
		}
	}
	if err := nativePeerBridgeKernelNoLivePorts(ctx, scope.Bridge); err != nil {
		t.Fatalf("partial native mutation left active bridge NICs: %v", err)
	}
	t.Log("S4_B2_INSTALLED_ROOT_REAL_ACL_PUT_THEN_UNKNOWN_JOURNAL_ALL_NICS_STOPPED=PASS")
}

// The same one-use original must freeze its sibling Project in SQLite, even
// after a malicious/obsolete generic RELEASED audit row is inserted.
func TestS4B2NativeInstalledPartialUnknownProjectLock(t *testing.T) {
	s4b2InstalledOneUse(t, false)
	path, dbPath, original := os.Getenv("BOXWEAVE_S4B2_INSTALLED_SCOPE"),
		os.Getenv("BOXWEAVE_S4B2_INSTALLED_DB"), os.Getenv("BOXWEAVE_S4B2_INSTALLED_ORIGINAL")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil || len(scope.Boxes) != 3 {
		t.Fatal("missing original scope", err)
	}
	s, err := state.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	rec, err := s.Get(ctx, original)
	if err != nil || rec.State != "UNKNOWN" {
		t.Fatalf("no original UNKNOWN: %+v %v", rec, err)
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var status string
	var desired int
	if err := db.QueryRowContext(ctx, "SELECT status,desired FROM peer_grants WHERE operation_id=?", original).Scan(&status, &desired); err != nil || status != "PENDING" || desired != 0 {
		t.Fatalf("partial root released or lost grant: %s %d %v", status, desired, err)
	}
	_, token, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	sibling := scope.Boxes[2]
	admit := func(key string) error {
		_, e := s.Admit(ctx, state.Admission{Layer: scope.Project, ActorID: token.ID, TokenID: token.ID,
			Key: key, RequestID: key, Action: "box_exec", SafeIntent: []byte("{}"),
			ResourceKey: "box:" + sibling.UUID, BoxUUID: sibling.UUID, Incarnation: sibling.UUID})
		return e
	}
	if err := admit("partial-root-sibling-before-release"); !errors.Is(err, state.ErrResourceLocked) {
		t.Fatalf("partial root allowed sibling admission: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest)
		VALUES(?,?,?,?,?)`, original, time.Now().UnixNano(), "forged-legacy", "RELEASED", "untrusted"); err != nil {
		t.Fatal(err)
	}
	if err := admit("partial-root-sibling-after-forged-release"); !errors.Is(err, state.ErrResourceLocked) {
		t.Fatalf("generic RELEASED bypassed root project lock: %v", err)
	}
	t.Log("S4_B2_INSTALLED_ROOT_PARTIAL_UNKNOWN_SQLITE_GRANT_PROJECT_FENCE_FORGED_RELEASE_DENIED=PASS")
}
