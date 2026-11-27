package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScopedRetirementLedgerEmptyAndLiveRefused(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "ledger.sqlite")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	st.Close()
	read := func() (ScopedNetworkAuditLedger, error) {
		r, err := OpenNetworkAuditReadOnly(path)
		if err != nil {
			return ScopedNetworkAuditLedger{}, err
		}
		defer r.Close()
		return r.ReadScopedRetirementLedger(ctx, "user-1001", "bwbr0")
	}
	if got, err := read(); err != nil || len(got.Boxes) != 0 || len(got.Retired) != 0 {
		t.Fatalf("empty retirement ledger refused: %+v %v", got, err)
	}
	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s4b2InsertAuditFixture(t, st)
	st.Close()
	if _, err := read(); err == nil {
		t.Fatal("live or foreign original admitted to retirement")
	}
}

func TestScopedRetirementLedgerRetainsTerminalClaims(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3RetiredFixture(t)
	const project = "user-1001"
	const bridge = "incusbr-1001"
	_, token, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	for i := 2; i <= 3; i++ {
		uuid := fmt.Sprintf("11111111-1111-4111-8111-%012d", i)
		box, err := s.OwnedBox(ctx, project, uuid)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(struct {
			BoxID string `json:"box_id"`
		}{uuid})
		rec, err := s.AdmitScoped(ctx, Admission{Layer: project, ActorID: token.ID, TokenID: token.ID, Key: "retire-" + box.Name, RequestID: box.Name, Action: "box_delete", CleanupDependencies: true, ResourceKey: "box:" + box.Name, BoxUUID: uuid, Incarnation: box.Incarnation, SafeIntent: raw})
		if err != nil {
			t.Fatal(err)
		}
		if err = s.Claim(ctx, rec.ID); err != nil {
			t.Fatal(err)
		}
		if err = s.NativeAck(ctx, rec.ID, "native-retire-"+box.Name); err != nil {
			t.Fatal(err)
		}
		if err = s.FinalizeVerifiedScopedDelete(ctx, rec.ID, box); err != nil {
			t.Fatal(err)
		}
	}
	// The fixture uses an in-memory store. Exercise the read-only admission
	// gate as well as the SQL snapshot; production opens a real read-only DB.
	s.auditReadOnly = true
	ledger, err := s.ReadScopedRetirementLedger(ctx, project, bridge)
	if err != nil || len(ledger.Boxes) != 0 || len(ledger.Retired) != 3 {
		t.Fatalf("terminal historical claims refused: %+v %v", ledger, err)
	}
	if _, err = s.db.Exec("UPDATE operation_resource_claims SET layer='foreign' WHERE operation_id=?", ledger.Retired[0].CreateID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReadScopedRetirementLedger(ctx, project, bridge); err == nil {
		t.Fatal("foreign claim admitted")
	}
}
