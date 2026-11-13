package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

// Regression for a real interactive SSH session: the general command
// context may have expired while the operator was typing two passwords.
// Persistence must get a fresh deadline AFTER prompt completion.
func TestOAuthPasswordSetupSurvivesExpiredCLIContext(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := state.Open(filepath.Join(dir, "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	cliCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Minute))
	defer cancel()
	if !errors.Is(cliCtx.Err(), context.DeadlineExceeded) {
		t.Fatal("expected expired outer CLI context")
	}
	if err := setupOAuthPasswordAfterPrompt(store, "synthetic-test-password-123456"); err != nil {
		t.Fatalf("password setup wrongly inherited expired command deadline: %v", err)
	}
	yes, err := store.OAuthPasswordPresent(context.Background())
	if err != nil || !yes {
		t.Fatalf("new password hash not persisted: present=%t err=%v", yes, err)
	}
	cfg, err := store.OAuthConfig(context.Background())
	if err != nil || cfg.Enabled || cfg.Origin != "" {
		t.Fatalf("password setup incorrectly enabled public OAuth: %+v %v", cfg, err)
	}

	if err := setupOAuthPasswordAfterPrompt(store, "short"); !errors.Is(err, state.ErrOAuthInvalid) {
		t.Fatalf("weak password unexpectedly accepted: %v", err)
	}
	if err := setupOAuthPasswordAfterPrompt(store, "synthetic-rotated-password-123456"); err != nil {
		t.Fatalf("password rotation failed: %v", err)
	}
}
