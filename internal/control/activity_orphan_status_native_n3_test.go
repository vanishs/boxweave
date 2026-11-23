package control

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// Read-only continuation of the exact one-use corrupted restore copy.
// Existing host_status MUST surface a bounded orphan sample without
// claiming native completeness, even though actual Incus guest is healthy.
func TestN3NativeMCPOrphanRestoreHostStatus(t *testing.T) {
	project, _ := n3AckLab(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	home, e := os.UserHomeDir()
	if e != nil {
		t.Fatal(e)
	}
	db := filepath.Join(home, "n3-orphan-restore-native-20261122", "state.db")
	s, e := state.Open(db)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	call := n3AckMCP(t, ctx, s, r)
	got := call("host_status", map[string]any{})
	at, ok := got["attention"].(map[string]any)
	if !ok || at["coverage"] != "BOUNDED_BLOCKER_SAMPLE" || at["scope"] != project {
		t.Fatal("status silently masked DB corruption", got)
	}
	rows, ok := at["items"].([]any)
	if !ok || len(rows) != 3 {
		t.Fatal("orphan summary omitted", at)
	}
	seen := map[string]int{}
	for _, item := range rows {
		issue := item.(map[string]any)
		seen[issue["reason"].(string)]++
		if issue["disposition"] != "INSPECT_EVIDENCE" {
			t.Fatal("masked conflict", issue)
		}
	}
	if seen["NETWORK_ORPHAN_BINDING"] != 1 || seen["NETWORK_ORPHAN_ADDRESS_FENCE"] != 2 {
		t.Fatal("not actionable host status", seen)
	}
	t.Log("N3_NATIVE_MCP_HOST_STATUS_ORPHAN_ATTENTION=PASS")
}
