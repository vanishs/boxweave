package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func oauthAutomationStore(t *testing.T) *state.Store {
	t.Helper()
	dbDir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dbDir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := state.Open(filepath.Join(dbDir, "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOAuthSetupGenerateAndPasswordFile(t *testing.T) {
	s := oauthAutomationStore(t)
	dir := t.TempDir()
	secret := filepath.Join(dir, "oauth-approval")
	if err := setupOAuthNoninteractive(s, true, secret, ""); err != nil {
		t.Fatalf("generate: %v", err)
	}
	data, err := os.ReadFile(secret)
	if err != nil {
		t.Fatal(err)
	}
	// 32 bytes of crypto/rand become 43 URL-safe characters, plus newline.
	if len(data) != 44 || data[len(data)-1] != '\n' {
		t.Fatalf("unexpected generated secret shape: length %d", len(data))
	}
	if strings.Contains(string(data), "=") {
		t.Fatal("generated password must be URL-safe and unpadded")
	}
	info, err := os.Stat(secret)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("secret permissions: %v", info.Mode().Perm())
	}
	present, err := s.OAuthPasswordPresent(context.Background())
	if err != nil || !present {
		t.Fatalf("password not stored: %t %v", present, err)
	}
	if err := setupOAuthNoninteractive(s, false, "", secret); err != nil {
		t.Fatalf("read generated password file: %v", err)
	}
	if err := setupOAuthNoninteractive(s, true, secret, ""); err == nil {
		t.Fatal("existing secret file was overwritten")
	}
	after, err := os.ReadFile(secret)
	if err != nil || string(after) != string(data) {
		t.Fatal("failed generate attempt modified existing secret")
	}
}

func TestOAuthSetupRejectsUnsafeFiles(t *testing.T) {
	s := oauthAutomationStore(t)
	dir := t.TempDir()
	secret := filepath.Join(dir, "approval")
	const good = "generated-test-password-1234567\n"
	if err := os.WriteFile(secret, []byte(good), 0644); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, false, "", secret); err == nil {
		t.Fatal("group/world-readable secret accepted")
	}
	if err := os.Chmod(secret, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "symlink")
	if err := os.Symlink(secret, link); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, false, "", link); err == nil {
		t.Fatal("symlink secret accepted")
	}
	hardlink := filepath.Join(dir, "hardlink")
	if err := os.Link(secret, hardlink); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, false, "", secret); err == nil {
		t.Fatal("hardlinked secret accepted")
	}
	if err := os.Remove(hardlink); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, false, "", secret); err != nil {
		t.Fatalf("valid owner-only secret rejected: %v", err)
	}
	publicDir := filepath.Join(t.TempDir(), "public")
	if err := os.Mkdir(publicDir, 0755); err != nil {
		t.Fatal(err)
	}
	secretPublic := filepath.Join(publicDir, "password")
	if err := os.WriteFile(secretPublic, []byte(good), 0600); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, false, "", secretPublic); err == nil {
		t.Fatal("non-private directory accepted")
	}
	if err := setupOAuthNoninteractive(s, true, filepath.Join(publicDir, "generated"), ""); err == nil {
		t.Fatal("generated secret allowed in public directory")
	}
}

func TestOAuthSetupRejectsInvalidOptionsAndContents(t *testing.T) {
	s := oauthAutomationStore(t)
	dir := t.TempDir()
	secret := filepath.Join(dir, "approval")
	for _, tc := range []struct {
		generate bool
		output   string
		input    string
	}{
		{true, "", ""},
		{true, secret, secret},
		{false, secret, ""},
		{false, "", ""},
		{false, secret, secret},
		{true, "relative", ""},
	} {
		if err := setupOAuthNoninteractive(s, tc.generate, tc.output, tc.input); err == nil {
			t.Fatalf("invalid options accepted: %+v", tc)
		}
	}
	for _, bad := range []string{
		"short\n",
		"line-one-password\nline-two-password\n",
		strings.Repeat("x", 257),
		"long-enough-password\r\n",
		"long-enough-password\x00",
	} {
		if err := os.WriteFile(secret, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if err := setupOAuthNoninteractive(s, false, "", secret); err == nil {
			t.Fatalf("unsafe content accepted (length %d)", len(bad))
		}
	}
}

func TestOAuthSetupPersistenceFailureCleansNewFile(t *testing.T) {
	s := oauthAutomationStore(t)
	dir := t.TempDir()
	secret := filepath.Join(dir, "new-password")
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := setupOAuthNoninteractive(s, true, secret, ""); err == nil {
		t.Fatal("closed DB allowed password setup")
	}
	if _, err := os.Lstat(secret); !os.IsNotExist(err) {
		t.Fatalf("secret file retained after database failure: %v", err)
	}
}
