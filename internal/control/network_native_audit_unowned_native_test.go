package control

import (
	"context"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Reuse the prior one-use clone with a DELETED owner but intact live native
// fixture. Do not adopt it or disclose that unowned native object's identity.
func TestN3NativeMCPUnownedProjectInstanceRedaction(t *testing.T) {
	project, _ := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	s, e := state.Open(filepath.Join(home, "n3-orphan-restore-native-20261122", "state.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	call := n3AckMCP(t, ctx, s, r)
	args := map[string]any{"view": "native_network", "limit": 1}
	found := map[string]bool{}
	for i := 0; i < 8; i++ {
		p := call("box_activity_get", args)["activity"].(map[string]any)
		info := p["native_network"].(map[string]any)
		if info["owned"] != float64(0) || info["native"] != float64(1) || info["unowned_native"] != float64(1) ||
			info["orphan_bindings"] != float64(1) || info["orphan_fences"] != float64(2) ||
			info["entire_project_inventory_scanned"] != true || info["cross_source_atomic"] != false {
			t.Fatal("orphan scope unverified", p)
		}
		items, ok := p["issues"].([]any)
		if !ok || len(items) != 1 {
			t.Fatal("lossy pagination", p)
		}
		item := items[0].(map[string]any)
		if _, ok := item["box_id"]; ok {
			t.Fatal("unowned UUID disclosed", item)
		}
		if _, ok := item["operation_id"]; ok {
			t.Fatal("unknown native adopted", item)
		}
		if strings.Contains(item["resource_ref"].(string), n3AckName) {
			t.Fatal("unowned name disclosed", item)
		}
		found[item["reason"].(string)] = true
		next, _ := p["next_cursor"].(string)
		if next == "" {
			break
		}
		args["cursor"] = next
	}
	for _, r := range []string{"NETWORK_ORPHAN_BINDING_PRESENT", "NETWORK_ORPHAN_FENCE_PRESENT", "NATIVE_UNOWNED_PROJECT_INSTANCE"} {
		if !found[r] {
			t.Fatal("native/orphan evidence missing", r, found)
		}
	}
	if len(found) != 3 {
		t.Fatal("unexpected uncontrolled details", found)
	}
	t.Log("N3_NATIVE_MCP_UNOWNED_PROJECT_REDACTED=PASS")
}
