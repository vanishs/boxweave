//go:build ignore

// Only a one-use hosted RC10 fixture. Exercise actual installed CLI auth
// leaves; mint controlled test-only OAuth grants through the existing state
// issuance path (not a real ChatGPT client witness). Never print secrets.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/vanishs/boxweave/internal/state"
)

func must(b bool, label string) {
	if !b {
		panic(label)
	}
}
func main() {
	project := flag.String("project", "", "exact disposable Project")
	flag.Parse()
	must(os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("RUNNER_ENVIRONMENT") == "github-hosted" &&
		os.Getenv("GITHUB_REPOSITORY") == "vanishs/boxweave" &&
		os.Getenv("GITHUB_REF") == "refs/heads/docs/s5b-rc10-closeout-20261121" &&
		os.Getuid() != 0 && *project == "user-"+strconv.Itoa(os.Getuid()), "unsafe CLI test scope")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cli := func(leaf string, args ...string) string {
		cmd := exec.CommandContext(ctx, "/usr/local/bin/boxweave", append([]string{leaf, "-project", *project}, args...)...)
		output, err := cmd.Output()
		must(err == nil, "installed CLI failed: "+leaf)
		return string(output)
	}
	store, err := state.Open("/var/lib/boxweave/ledger.sqlite")
	must(err == nil, "ledger")
	defer store.Close()
	cfg, err := store.OAuthConfig(ctx)
	must(err == nil && !cfg.Enabled && cfg.Origin == "", "existing OAuth identity")
	dir, err := os.MkdirTemp("/home/boxweave", "rc10-cli-auth-")
	must(err == nil, "private temp directory")
	defer os.RemoveAll(dir)
	secret := dir + "/approval"
	cli("oauth-setup", "-generate", "-output-file", secret)
	info, err := os.Stat(secret)
	must(err == nil && info.Mode().Perm() == 0600, "secret mode")
	password, err := os.ReadFile(secret)
	must(err == nil && len(password) >= 16, "private secret")
	cli("oauth-setup", "-password-file", secret)
	const origin = "https://s5b-ephemeral.example.com"
	cli("oauth-set-origin", "-public-origin", origin)
	must(json.Unmarshal([]byte(cli("oauth-status")), &cfg) == nil && cfg.Enabled && cfg.Origin == origin, "CLI status")
	const redirect = "https://chatgpt.com/connector_platform_oauth_redirect"
	client, err := store.RegisterOAuthClient(ctx, "RC10 one-use CLI test", []string{redirect})
	must(err == nil, "test client")
	verifier := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ._~"
	d := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(d[:])
	issue := func(label string) state.OAuthPair {
		pending, e := store.BeginOAuth(ctx, client, redirect, "", challenge, origin+"/mcp", label)
		must(e == nil, "pending")
		code, _, _, e := store.ApproveOAuth(ctx, pending, strings.TrimSuffix(string(password), "\n"))
		must(e == nil, "approve")
		pair, e := store.ExchangeOAuth(ctx, client, redirect, code, verifier, origin+"/mcp")
		must(e == nil && pair.Access != "" && pair.GrantID != "", "exchange")
		return pair
	}
	pair := issue("cli-revoke")
	other := issue("cli-disable")
	list := cli("oauth-list-grants")
	must(strings.Contains(list, pair.GrantID) && strings.Contains(list, other.GrantID), "grant listing")
	denied := func(access string) {
		req, e := http.NewRequestWithContext(ctx, http.MethodPost, "http://127.0.0.1:18650/mcp", strings.NewReader("{}"))
		must(e == nil, "request")
		req.Header.Set("Authorization", "Bearer "+access)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		resp, e := http.DefaultClient.Do(req)
		must(e == nil, "HTTP request")
		resp.Body.Close()
		must(resp.StatusCode == http.StatusUnauthorized, "revoked credential accepted")
	}
	_, err = store.VerifyToken(ctx, pair.Access)
	must(err == nil, "before revoke")
	cli("oauth-revoke-grant", "-id", pair.GrantID)
	_, err = store.VerifyToken(ctx, pair.Access)
	must(err != nil, "revoke not durable")
	denied(pair.Access)
	_, err = store.VerifyToken(ctx, other.Access)
	must(err == nil, "other grant changed")
	cli("oauth-disable")
	must(json.Unmarshal([]byte(cli("oauth-status")), &cfg) == nil && !cfg.Enabled && cfg.Origin == "", "disable status")
	denied(other.Access)
	receipt := strings.Split(strings.TrimSpace(cli("token-create", "-ttl", "10m")), "\n")
	must(len(receipt) == 2 && strings.HasPrefix(receipt[0], "id=") && strings.HasPrefix(receipt[1], "token="), "token receipt")
	id := strings.TrimPrefix(strings.Fields(receipt[0])[0], "id=")
	access := strings.TrimPrefix(receipt[1], "token=")
	_, err = store.VerifyToken(ctx, access)
	must(err == nil, "issued token")
	cli("token-revoke", "-id", id)
	denied(access)
	fmt.Println("S5B_RC10_CLI_AUTH=PASS")
}
