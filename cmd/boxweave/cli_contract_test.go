package main

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestP3R1CommandContract(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	// Help must not dispatch any retained command or private worker.
	if len(cliCommands) != 45 {
		t.Fatalf("active generic command inventory: %d", len(cliCommands))
	}
	for command := range cliCommands {
		t.Run(command, func(t *testing.T) {
			os.Args = []string{"boxweave", command, "--help"}
			if err := run(); !errors.Is(err, flag.ErrHelp) {
				t.Fatalf("help: %v", err)
			}
		})
	}
	os.Args = []string{"boxweave", "--help"}
	if err := run(); err != nil {
		t.Fatal(err)
	}
	db := filepath.Join(t.TempDir(), "untouched.sqlite")
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"not-a-boxweave-command", "--help"}, "unsupported command"},
		{[]string{"upgrade", "--help"}, "legacy installation mutation unsupported"},
		{[]string{"rollback", "-db", db}, "legacy installation mutation unsupported"},
		{[]string{"install-commit", "-approve-local"}, "legacy installation mutation unsupported"},
		{[]string{"native-peer-audit", "--db", db}, "unsupported command"},
		{[]string{"native-peer-purge-plan", "--db", db}, "unsupported command"},
		{[]string{"native-peer-root-unit-stage", "--help"}, "unsupported command"},
		{[]string{"status", "-db", db, "-approve-local"}, "flag provided but not defined"},
		{[]string{"serve", "-db", db, "-scoped-peer-fresh=false"}, "legacy controller mode unsupported"},
		{[]string{"build-info", "-db", db}, "flag provided but not defined"},
		{[]string{"native-scoped-offline-review", "-approve-local"}, "flag provided but not defined"},
		{[]string{"native-scoped-guard-boot", "-project", "user-1001"}, "flag provided but not defined"},
		{[]string{"doctor-native-worker", "--help"}, "private diagnostic worker accepts no user arguments"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			os.Args = append([]string{"boxweave"}, c.args...)
			if err := run(); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want %q, got %v", c.want, err)
			}
		})
	}
	if _, err := os.Stat(db); !os.IsNotExist(err) {
		t.Fatalf("refused command touched source: %v", err)
	}
}

func TestP3R1FlagSetPreservesDefaultsAndBoolean(t *testing.T) {
	all := flag.NewFlagSet("serve", flag.ContinueOnError)
	db := all.String("db", "fixed", "db")
	scoped := all.Bool("scoped-peer-fresh", true, "scoped")
	all.String("project", "", "project")
	all.String("incus-socket", "socket", "socket")
	all.String("listen", "loopback", "listener")
	all.Bool("approve-local", false, "unrelated approval")
	f := commandFlagSet(all, cliCommands["serve"])
	if f.Lookup("approve-local") != nil || f.Lookup("scoped-peer-fresh").DefValue != "true" {
		t.Fatal("wrong scoped flag set")
	}
	if err := f.Parse([]string{"-db", "new", "-scoped-peer-fresh"}); err != nil || *db != "new" || !*scoped {
		t.Fatalf("binding/default failure: %v", err)
	}
	if err := f.Parse([]string{"-scoped-peer-fresh=false"}); err != nil || *scoped {
		t.Fatalf("explicit false lost: %v", err)
	}
}
