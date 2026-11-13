package state

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const testOAuthOrigin = "https://lab.trycloudflare.com"
const testOAuthRedirect = "https://chatgpt.com/connector_platform_oauth_redirect"

func oauthFixture(t *testing.T) (*Store, string, string, string, string, string, string) {
	t.Helper()
	s := newStore(t)
	ctx := context.Background()
	if err := s.SetupOAuthPassword(ctx, "hostgo-fixture-approval-password-123"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetOAuthOrigin(ctx, testOAuthOrigin); err != nil {
		t.Fatal(err)
	}
	client, err := s.RegisterOAuthClient(ctx, "ChatGPT test", []string{testOAuthRedirect})
	if err != nil {
		t.Fatal(err)
	}
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ._~"
	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])
	pending, err := s.BeginOAuth(ctx, client, testOAuthRedirect, "mcp:viewer mcp:admin", challenge, testOAuthOrigin+"/mcp", "test-state-12345678")
	if err != nil {
		t.Fatal(err)
	}
	return s, client, pending, verifier, challenge, testOAuthRedirect, "hostgo-fixture-approval-password-123"
}
func TestOAuthAtomicPKCERefreshAndOriginRevocation(t *testing.T) {
	ctx := context.Background()
	s, client, pending, verifier, _, redirect, password := oauthFixture(t)
	manual, manualInfo, err := s.IssueToken(ctx, "viewer", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err = s.ApproveOAuth(ctx, pending, "invalid-approval-password"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("bad password: %v", err)
	}
	intent, err := s.OAuthPending(ctx, pending)
	if err != nil || intent.Challenge == "" {
		t.Fatalf("pending changed after failed consent: %v", err)
	}
	code, callback, st, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil || callback != redirect || st != "test-state-12345678" {
		t.Fatalf("approve: %v %q %q", err, callback, st)
	}
	if _, _, _, err = s.ApproveOAuth(ctx, pending, password); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("replayed pending: %v", err)
	}
	if _, err = s.ExchangeOAuth(ctx, client, redirect, code, "wrong-verifier", testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("wrong PKCE: %v", err)
	}
	if _, err = s.ExchangeOAuth(ctx, client, redirect, code, verifier, "https://evil.example/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("wrong audience: %v", err)
	}
	pair, err := s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if pair.Scope != "mcp:admin" || pair.Access == pair.Refresh || pair.GrantID == "" {
		t.Fatalf("bad pair: %+v", pair)
	}
	verified, err := s.VerifyToken(ctx, pair.Access)
	if err != nil || verified.Role != "admin" {
		t.Fatalf("OAuth admin not mapped: %+v %v", verified, err)
	}
	if err = s.RequireActiveTokenID(ctx, verified.ID); err != nil {
		t.Fatalf("active OAuth dispatch actor rejected: %v", err)
	}
	if _, err = s.VerifyToken(ctx, pair.Refresh); !errors.Is(err, ErrRevoked) {
		t.Fatalf("refresh used as bearer: %v", err)
	}
	if _, err = s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("authorization replay: %v", err)
	}
	if _, err = s.RefreshOAuth(ctx, pair.Refresh, "forged-client", testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("refresh client mismatch: %v", err)
	}
	next, err := s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.VerifyToken(ctx, pair.Access); !errors.Is(err, ErrRevoked) {
		t.Fatalf("rotated old access accepted: %v", err)
	}
	if err = s.RequireActiveTokenID(ctx, verified.ID); !errors.Is(err, ErrRevoked) {
		t.Fatalf("rotated old access dispatch accepted: %v", err)
	}
	if _, err = s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("old refresh replay: %v", err)
	}
	if _, err = s.VerifyToken(ctx, next.Access); err != nil {
		t.Fatalf("rotated access invalid: %v", err)
	}
	if err = s.SetOAuthOrigin(ctx, "https://changed.trycloudflare.com"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.VerifyToken(ctx, next.Access); !errors.Is(err, ErrRevoked) {
		t.Fatalf("old origin accepted: %v", err)
	}
	if _, err = s.RefreshOAuth(ctx, next.Refresh, client, testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("old refresh accepted: %v", err)
	}
	if got, err := s.VerifyToken(ctx, manual); err != nil || got.ID != manualInfo.ID {
		t.Fatalf("local API token affected: %+v %v", got, err)
	}
}
func TestOAuthConcurrentRefreshSingleSuccess(t *testing.T) {
	ctx := context.Background()
	s, client, pending, verifier, _, redirect, password := oauthFixture(t)
	code, _, _, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp")
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	var ok, invalid int
	for e := range results {
		if e == nil {
			ok++
		} else if errors.Is(e, ErrOAuthInvalid) {
			invalid++
		} else {
			t.Fatalf("unexpected concurrent error: %v", e)
		}
	}
	if ok != 1 || invalid != 11 {
		t.Fatalf("refresh concurrency: success=%d invalid=%d", ok, invalid)
	}
}
func TestOAuthFailedSQLiteTransactionDoesNotConsumeCodeOrRefresh(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Join(dir, "ledger.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	password := "fixture-password-very-long-for-testing"
	if err = s.SetupOAuthPassword(ctx, password); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOAuthOrigin(ctx, testOAuthOrigin); err != nil {
		t.Fatal(err)
	}
	client, err := s.RegisterOAuthClient(ctx, "fixture", []string{testOAuthRedirect})
	if err != nil {
		t.Fatal(err)
	}
	verifier := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ._~"
	h := sha256.Sum256([]byte(verifier))
	pending, err := s.BeginOAuth(ctx, client, testOAuthRedirect, "mcp:viewer", base64.RawURLEncoding.EncodeToString(h[:]), testOAuthOrigin+"/mcp", "my-state-12345678")
	if err != nil {
		t.Fatal(err)
	}
	code, _, _, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ExchangeOAuth(ctx, client, testOAuthRedirect, code, verifier, testOAuthOrigin+"/mcp"); err == nil {
		t.Fatal("read-only store unexpectedly issued tokens")
	}
	if _, err = s.db.Exec("PRAGMA query_only=OFF"); err != nil {
		t.Fatal(err)
	}
	pair, err := s.ExchangeOAuth(ctx, client, testOAuthRedirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatalf("code consumed under failed transaction: %v", err)
	}
	if _, err = s.db.Exec("PRAGMA query_only=ON"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp"); err == nil {
		t.Fatal("read-only refresh unexpectedly succeeded")
	}
	if _, err = s.db.Exec("PRAGMA query_only=OFF"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp"); err != nil {
		t.Fatalf("refresh consumed on failure: %v", err)
	}
}
func TestOAuthRejectUntrustedOriginRedirectAndPKCE(t *testing.T) {
	ctx := context.Background()
	s, client, _, _, challenge, redirect, _ := oauthFixture(t)
	for _, origin := range []string{"http://example.com", "https://127.0.0.1", "https://example.com/mcp", "https://example.com/", "https://EXAMPLE.COM", "https://example.com:443", "https://user@example.com", "https://internal"} {
		if err := s.SetOAuthOrigin(ctx, origin); !errors.Is(err, ErrOAuthInvalid) {
			t.Errorf("accepted noncanonical public origin %q: %v", origin, err)
		}
	}
	for _, bad := range []string{"https://evil.example/callback", "http://evil.example/callback", ""} {
		if _, err := s.BeginOAuth(ctx, client, bad, "mcp:viewer", challenge, testOAuthOrigin+"/mcp", "my-state-12345678"); !errors.Is(err, ErrOAuthInvalid) {
			t.Errorf("accepted redirect %q: %v", bad, err)
		}
	}
	for _, bad := range []string{"bad", "mcp:other", "mcp:admin mcp:admin"} {
		if _, err := s.BeginOAuth(ctx, client, redirect, bad, challenge, testOAuthOrigin+"/mcp", "my-state-12345678"); !errors.Is(err, ErrOAuthInvalid) {
			t.Errorf("accepted scope %q: %v", bad, err)
		}
	}
	if _, err := s.BeginOAuth(ctx, client, redirect, "mcp:admin", "plain-verifier", testOAuthOrigin+"/mcp", "my-state-12345678"); !errors.Is(err, ErrOAuthInvalid) {
		t.Errorf("accepted plain challenge: %v", err)
	}
}

func TestOAuthCredentialAndTunnelOriginResetInvalidatesPendingAndGrants(t *testing.T) {
	ctx := context.Background()
	s, client, pending, verifier, challenge, redirect, password := oauthFixture(t)
	code, _, _, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	next, err := s.BeginOAuth(ctx, client, redirect, "mcp:viewer", challenge, testOAuthOrigin+"/mcp", "state-for-next-12345")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetupOAuthPassword(ctx, "new-approval-password-long-12345"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.OAuthPending(ctx, next); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("old consent survives password rotation: %v", err)
	}
	if _, err = s.VerifyToken(ctx, pair.Access); !errors.Is(err, ErrRevoked) {
		t.Fatalf("access survives password rotation: %v", err)
	}
	if _, err = s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp"); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("refresh survives password rotation: %v", err)
	}
	newPending, err := s.BeginOAuth(ctx, client, redirect, "mcp:viewer", challenge, testOAuthOrigin+"/mcp", "state-before-oscillation-123")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetOAuthOrigin(ctx, "https://new.trycloudflare.com"); err != nil {
		t.Fatal(err)
	}
	if err = s.SetOAuthOrigin(ctx, testOAuthOrigin); err != nil {
		t.Fatal(err)
	}
	if _, err = s.OAuthPending(ctx, newPending); !errors.Is(err, ErrOAuthInvalid) {
		t.Fatalf("stale session survived A-B-A hostname: %v", err)
	}
}
