package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/diagnostic"
)

func TestPR177D3PlanCLIChild(t *testing.T) {
	raw := os.Getenv("BOXWEAVE_D3_CLI_ARGS")
	if raw == "" {
		return
	}
	var args []string
	if json.Unmarshal([]byte(raw), &args) != nil {
		os.Exit(3)
	}
	os.Args = append([]string{"boxweave"}, args...)
	if run() != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
func TestPR177D3PlanCLINoSourceCreation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	dir := t.TempDir()
	path := filepath.Join(dir, "missing", "ledger.sqlite")
	id := diagnostic.IssueID("user-1001", "database", "DIAGNOSTIC_DB_MISSING", "ledger")
	run := func(args ...string) []byte {
		t.Helper()
		raw, _ := json.Marshal(args)
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPR177D3PlanCLIChild$")
		cmd.Env = append(os.Environ(), "BOXWEAVE_D3_CLI_ARGS="+string(raw))
		b, e := cmd.Output()
		if x, ok := e.(*exec.ExitError); !ok || x.ExitCode() != 1 {
			t.Fatal("expected refusal", e)
		}
		return b
	}
	b := run("recover", "--plan", "--issue", id, "-project", "user-1001", "-db", path, "-incus-socket", filepath.Join(dir, "absent.socket"))
	var p diagnostic.RecoverPlan
	if e := json.Unmarshal(b, &p); e != nil || p.Issue == nil || p.Issue.ID != id || p.Status != "BLOCKED_UNPROVEN" || p.CanRepair || p.ApplyAvailable || p.ReviewSHA256 != "" {
		t.Fatal(string(b), e)
	}
	if _, e := os.Lstat(filepath.Dir(path)); !os.IsNotExist(e) {
		t.Fatal("source created", e)
	}
	for _, extra := range [][]string{{"--apply"}, {"--approve-local"}, {"--action", "release"}, {"--confirm-review-sha256", strings.Repeat("a", 64)}} {
		args := append([]string{"recover", "--plan", "--issue", id, "-project", "user-1001", "-db", path}, extra...)
		if b := run(args...); len(b) != 0 {
			t.Fatal("mixed plan accepted", string(b))
		}
	}
	if b := run("recover", "--apply", "--issue", id, "-project", "user-1001"); len(b) != 0 {
		t.Fatal("D4 apply accepted")
	}
	if b := run("recover-plan-worker"); len(b) != 0 {
		t.Fatal("untrusted private worker accepted")
	}
	t.Log("D3_CLI_NO_SOURCE_CREATION_NO_APPLY_PASS")
}
