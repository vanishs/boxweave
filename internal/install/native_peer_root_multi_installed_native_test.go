package install

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

// These fixtures run only on the disposable root-native CI runner. All
// admissions happen as the installed nonroot service UID and use a distinct
// original; the root service is approved afresh for each original.
func TestS4B3NativeNonrootAdmitFollowupPeer(t *testing.T) {
	s4b3InstalledOneUse(t, false)
	step := os.Getenv("BOXWEAVE_S4B3_STEP")
	if step != "grant_bc" && step != "revoke_ab" {
		t.Fatal("invalid native followup step")
	}
	raw, err := os.ReadFile(os.Getenv("BOXWEAVE_S4B3_SCOPE"))
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil || len(scope.Boxes) != 3 {
		t.Fatal("native scope", err)
	}
	u, err := user.Lookup("boxweave")
	if err != nil || fmt.Sprint(os.Geteuid()) != u.Uid || scope.Project != "user-"+u.Uid {
		t.Fatal("foreign nonroot followup identity")
	}
	s, err := state.Open(os.Getenv("BOXWEAVE_S4B3_DB"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, tok, err := s.IssueToken(context.Background(), "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a, b := scope.Boxes[0], scope.Boxes[1]
	allow := false
	if step == "grant_bc" {
		a, b = scope.Boxes[1], scope.Boxes[2]
		allow = true
	}
	intent := state.RootPeerRevokeSafeIntent{
		Project: scope.Project, Bridge: scope.Bridge, ACL: scope.ACL,
		SourceUUID: a.UUID, SourceIncarnation: a.UUID,
		TargetUUID: b.UUID, TargetIncarnation: b.UUID, Allow: allow,
	}
	safe, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	op, err := s.Admit(context.Background(), state.Admission{
		Layer: scope.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b3-disposable-" + step + "-unique", RequestID: "s4b3-multi-" + step,
		Action: "box_peer_set", BoxUUID: a.UUID, Incarnation: a.UUID,
		SafeIntent: safe, ResourceKey: "acl:default/" + scope.ACL,
		ResourceKeys: []string{"box:" + a.UUID, "box:" + b.UUID},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("S4_B3_NATIVE_FOLLOWUP_ORIGINAL=%s\n", op.ID)
	t.Log("S4_B3_NATIVE_NONROOT_FOLLOWUP_CLAIM_ADMISSION=PASS")
}

func TestS4B3NativeNonrootConfirmFollowupPeer(t *testing.T) {
	s4b3InstalledOneUse(t, false)
	step := os.Getenv("BOXWEAVE_S4B3_STEP")
	raw, err := os.ReadFile(os.Getenv("BOXWEAVE_S4B3_SCOPE"))
	if err != nil {
		t.Fatal(err)
	}
	var scope s4b2InstalledScope
	if err := json.Unmarshal(raw, &scope); err != nil || len(scope.Boxes) != 3 {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", os.Getenv("BOXWEAVE_S4B3_DB"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var stateName string
	err = db.QueryRow("SELECT state FROM operations WHERE id=?", os.Getenv("BOXWEAVE_S4B3_ORIGINAL")).Scan(&stateName)
	if err != nil || stateName != "SUCCEEDED" {
		t.Fatalf("followup controller original not completed: %s %v", stateName, err)
	}
	type peer struct {
		status  string
		desired int
	}
	readPeer := func(a, b int) peer {
		t.Helper()
		var p peer
		if err := db.QueryRow("SELECT status,desired FROM peer_grants WHERE project=? AND source_uuid=? AND target_uuid=?",
			scope.Project, scope.Boxes[a].UUID, scope.Boxes[b].UUID).Scan(&p.status, &p.desired); err != nil {
			t.Fatal(err)
		}
		return p
	}
	ab, bc := readPeer(0, 1), readPeer(1, 2)
	if bc.status != "CONFIRMED" || bc.desired != 1 {
		t.Fatalf("confirmed second grant lost: %+v", bc)
	}
	if step == "grant_bc" && (ab.status != "CONFIRMED" || ab.desired != 1) {
		t.Fatalf("first grant was overwritten: %+v", ab)
	}
	if step == "revoke_ab" && (ab.status != "REVOKED" || ab.desired != 0) {
		t.Fatalf("directional revoke did not commit: %+v", ab)
	}
	t.Logf("S4_B3_NATIVE_MULTI_GRANT_SINGLE_REVOKE_SQLITE_%s=PASS", step)
}
