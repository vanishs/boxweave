package install

import (
	"context"
	"database/sql"
	"encoding/json"
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
	_, tok, err := s.IssueToken(ctx, "admin", time.Hour)
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
