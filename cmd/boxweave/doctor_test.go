package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

func TestPR177D1CLIChild(t *testing.T) {
	p := os.Getenv("BOXWEAVE_D1_CLI_FIXTURE")
	if p == "" {
		return
	}
	os.Args = []string{"boxweave", "doctor", "-db", p, "-project", "user-1001", "-incus-socket", filepath.Join(filepath.Dir(p), "absent.socket")}
	if run() != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
func TestPR177D1CLINoSourceCreation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	runChild := func(path string) map[string]any {
		t.Helper()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPR177D1CLIChild$")
		cmd.Env = append(os.Environ(), "BOXWEAVE_D1_CLI_FIXTURE="+path)
		b, e := cmd.Output()
		if x, ok := e.(*exec.ExitError); !ok || x.ExitCode() != 1 {
			t.Fatal("expected incomplete preflight", e)
		}
		var v map[string]any
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		if v["ready"] != false {
			t.Fatal("false preflight ready")
		}
		return v
	}
	missing := filepath.Join(dir, "absent", "ledger.sqlite")
	v := runChild(missing)
	d := v["diagnostic"].(map[string]any)
	if d["database"].(map[string]any)["reason"] != "DIAGNOSTIC_DB_MISSING" {
		t.Fatal(d)
	}
	if _, e := os.Lstat(filepath.Dir(missing)); !os.IsNotExist(e) {
		t.Fatal("missing parent created", e)
	}
	p := filepath.Join(dir, "ledger.sqlite")
	s, e := state.Open(p)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	v = runChild(p)
	d = v["diagnostic"].(map[string]any)
	if d["database"].(map[string]any)["state"] != "OBSERVED" || v["schema_version"] != float64(state.SchemaVersion) {
		t.Fatal(d)
	}
	after, e := os.ReadFile(p)
	if e != nil || string(before) != string(after) {
		t.Fatal("source changed", e)
	}
	for _, ext := range []string{"-wal", "-shm", "-journal"} {
		if _, e = os.Lstat(p + ext); !os.IsNotExist(e) {
			t.Fatal("source sidecar created", ext, e)
		}
	}
}
