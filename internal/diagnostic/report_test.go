package diagnostic

import (
	"encoding/json"
	"github.com/vanishs/boxweave/internal/state"
	"strings"
	"testing"
)

func TestPR177D1IssueIdentityAndCoverage(t *testing.T) {
	r := NewReport("user-1001")
	r.AddIssue("database", "MISSING", "ledger", "UNVERIFIED")
	r.AddIssue("database", "MISSING", "ledger", "UNVERIFIED")
	if len(r.Issues) != 1 || r.CrossSourceAtomic || r.Coverage != "PARTIAL_D2_DIAGNOSIS" || r.Issues[0].Repairability != "NOT_EVALUATED" {
		t.Fatal(r)
	}
	same := NewReport("user-1001")
	same.AddIssue("database", "MISSING", "ledger", "UNVERIFIED")
	if same.Issues[0].ID != r.Issues[0].ID || IssueID("user-1002", "database", "MISSING", "ledger") == r.Issues[0].ID {
		t.Fatal("unstable or cross-scope ID")
	}
	if IssueID("a", "b:c", "d", "e") == IssueID("a:b", "c", "d", "e") {
		t.Fatal("ambiguous identity encoding")
	}
	b, e := json.Marshal(r)
	if e != nil || strings.Contains(string(b), "GoVersion") {
		t.Fatal(string(b), e)
	}
}

func TestPR177D2InventoryJSONContract(t *testing.T) {
	r := NewReport("user-1001")
	r.NativeInventoryInfo = &state.NativeNetworkAuditInfo{InventoryScanned: true}
	b, e := json.Marshal(r)
	if e != nil || !strings.Contains(string(b), `"entire_project_inventory_scanned":true`) || strings.Contains(string(b), `"inventory_scanned":`) {
		t.Fatal(string(b), e)
	}
}
