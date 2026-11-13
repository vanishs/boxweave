// The single-controller OAuth ledger uses the existing protected SQLite DB.
// The go-oauth2 library validates public-client OAuth requests and PKCE.
// Its default Manager must NOT be used here: it consumes codes before PKCE
// validation and rotates refresh tokens in multiple non-atomic store calls.
package state

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"

	oauth2 "github.com/go-oauth2/oauth2/v4"
	"golang.org/x/crypto/argon2"
)

var ErrOAuthInvalid = errors.New("invalid OAuth credential or request")
var ErrOAuthRateLimited = errors.New("OAuth request rate limit")
var oauthPKCE = regexp.MustCompile("^[A-Za-z0-9_-]{43}$")
var oauthVerifier = regexp.MustCompile("^[A-Za-z0-9._~-]{43,128}$")

type OAuthSettings struct {
	Origin  string
	Enabled bool
}
type OAuthIntent struct {
	ClientID    string
	ClientName  string
	RedirectURI string
	Challenge   string
	Scope       string
	Resource    string
	State       string
}
type OAuthPair struct {
	Access    string
	Refresh   string
	ExpiresIn int
	Scope     string
	GrantID   string
}

func newOAuthSecret(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b), nil
}
func validOAuthOrigin(raw string) bool {
	if len(raw) < 12 || len(raw) > 255 || strings.TrimSpace(raw) != raw {
		return false
	}
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return false
	}
	if u.String() != raw || u.Port() != "" {
		return false
	}
	name := u.Hostname()
	if net.ParseIP(name) != nil || name == "localhost" || strings.HasSuffix(name, ".localhost") {
		return false
	}
	if !strings.Contains(name, ".") || strings.ToLower(name) != name {
		return false
	}
	for _, x := range strings.Split(name, ".") {
		if len(x) < 1 || len(x) > 63 || x[0] == '-' || x[len(x)-1] == '-' {
			return false
		}
		for _, c := range x {
			if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	return true
}
func validOAuthRedirect(raw string) bool {
	if len(raw) > 1024 || len(raw) < 12 || strings.TrimSpace(raw) != raw {
		return false
	}
	u, e := url.Parse(raw)
	if e != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" || u.String() != raw {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	return u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "::1" || u.Hostname() == "localhost")
}
func (s *Store) OAuthConfig(ctx context.Context) (OAuthSettings, error) {
	var o, p string
	err := s.db.QueryRowContext(ctx, "SELECT public_origin,password_hash FROM oauth_config WHERE id=1").Scan(&o, &p)
	return OAuthSettings{Origin: o, Enabled: o != "" && p != ""}, err
}
func (s *Store) OAuthPasswordPresent(ctx context.Context) (bool, error) {
	var v string
	err := s.db.QueryRowContext(ctx, "SELECT password_hash FROM oauth_config WHERE id=1").Scan(&v)
	return v != "", err
}
func hashOAuthPassword(password string) (string, error) {
	if len(password) < 16 || len(password) > 256 {
		return "", ErrOAuthInvalid
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, 2, 64*1024, 1, 32)
	return "argon2id$" + base64.RawURLEncoding.EncodeToString(salt) + "$" + base64.RawURLEncoding.EncodeToString(key), nil
}
func verifyOAuthPassword(encoded, password string) bool {
	x := strings.Split(encoded, "$")
	if len(x) != 3 || x[0] != "argon2id" || len(password) > 256 {
		return false
	}
	salt, es := base64.RawURLEncoding.DecodeString(x[1])
	expected, ee := base64.RawURLEncoding.DecodeString(x[2])
	if es != nil || ee != nil || len(salt) != 16 || len(expected) != 32 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 2, 64*1024, 1, 32)
	return subtle.ConstantTimeCompare(got, expected) == 1
}
func (s *Store) SetupOAuthPassword(ctx context.Context, password string) error {
	hashed, err := hashOAuthPassword(password)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE oauth_config SET password_hash=?,epoch=epoch+1 WHERE id=1", hashed); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_pending"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_codes"); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) SetOAuthOrigin(ctx context.Context, raw string) error {
	if raw != "" && !validOAuthOrigin(raw) {
		return ErrOAuthInvalid
	}
	// Make an issuer/resource change and old pending-flow invalidation one
	// transaction. Manual API tokens and native operation receipts are untouched.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var old string
	if err = tx.QueryRowContext(ctx, "SELECT public_origin FROM oauth_config WHERE id=1").Scan(&old); err != nil {
		return err
	}
	if old == raw {
		return nil
	}
	if _, err = tx.ExecContext(ctx, "UPDATE oauth_config SET public_origin=?,epoch=epoch+1 WHERE id=1", raw); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_pending"); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_codes"); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) rateOAuth(ctx context.Context, kind string, limit int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_attempts WHERE at<?", now-300); err != nil {
		return err
	}
	var n int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_attempts WHERE kind=? AND at>=?", kind, now-60).Scan(&n); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO oauth_attempts(at,kind) VALUES(?,?)", now, kind); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if n >= limit {
		return ErrOAuthRateLimited
	}
	return nil
}
func (s *Store) RegisterOAuthClient(ctx context.Context, name string, redirects []string) (string, error) {
	if err := s.rateOAuth(ctx, "dcr", 15); err != nil {
		return "", err
	}
	if len(name) < 1 || len(name) > 80 || len(redirects) < 1 || len(redirects) > 4 {
		return "", ErrOAuthInvalid
	}
	seen := map[string]bool{}
	for _, v := range redirects {
		if !validOAuthRedirect(v) || seen[v] {
			return "", ErrOAuthInvalid
		}
		seen[v] = true
	}
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_clients").Scan(&count); err != nil {
		return "", err
	}
	if count >= 200 {
		return "", ErrOAuthRateLimited
	}
	id, err := newOAuthSecret("bwc_")
	if err != nil {
		return "", err
	}
	data, _ := json.Marshal(redirects)
	_, err = s.db.ExecContext(ctx, "INSERT INTO oauth_clients(id,name,redirects,created_at) VALUES(?,?,?,?)", id, name, string(data), time.Now().Unix())
	return id, err
}
func (s *Store) oauthClient(ctx context.Context, id, redirect string) (string, error) {
	var name, raw string
	if err := s.db.QueryRowContext(ctx, "SELECT name,redirects FROM oauth_clients WHERE id=?", id).Scan(&name, &raw); err != nil {
		return "", ErrOAuthInvalid
	}
	var xs []string
	if json.Unmarshal([]byte(raw), &xs) != nil {
		return "", ErrOAuthInvalid
	}
	for _, x := range xs {
		if x == redirect {
			return name, nil
		}
	}
	return "", ErrOAuthInvalid
}
func parseOAuthScope(scope string) (string, error) {
	fields := strings.Fields(scope)
	if len(fields) == 0 {
		return "viewer", nil
	}
	if len(fields) > 2 {
		return "", ErrOAuthInvalid
	}
	hasAdmin, hasViewer := false, false
	for _, v := range fields {
		switch v {
		case "mcp:admin":
			if hasAdmin {
				return "", ErrOAuthInvalid
			}
			hasAdmin = true
		case "mcp:viewer":
			if hasViewer {
				return "", ErrOAuthInvalid
			}
			hasViewer = true
		default:
			return "", ErrOAuthInvalid
		}
	}
	if hasAdmin {
		return "admin", nil
	}
	return "viewer", nil
}
func (s *Store) BeginOAuth(ctx context.Context, client, redirect, scope, challenge, resource, state string) (string, error) {
	cfg, err := s.OAuthConfig(ctx)
	if err != nil || !cfg.Enabled {
		return "", ErrOAuthInvalid
	}
	if _, err = s.oauthClient(ctx, client, redirect); err != nil {
		return "", err
	}
	if _, err = parseOAuthScope(scope); err != nil {
		return "", err
	}
	if resource != cfg.Origin+"/mcp" || !oauthPKCE.MatchString(challenge) || len(state) < 8 || len(state) > 512 {
		return "", ErrOAuthInvalid
	}
	if err := s.rateOAuth(ctx, "authorize", 45); err != nil {
		return "", err
	}
	pending, err := newOAuthSecret("bwp_")
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_pending WHERE expires_at<?", time.Now().Unix()); err != nil {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM oauth_codes WHERE expires_at<?", time.Now().Unix()); err != nil {
		return "", err
	}
	var count int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM oauth_pending").Scan(&count); err != nil {
		return "", err
	}
	if count >= 512 {
		return "", ErrOAuthRateLimited
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO oauth_pending(digest,client_id,redirect_uri,challenge,scope,resource,state,expires_at) VALUES(?,?,?,?,?,?,?,?)",
		DigestToken(pending), client, redirect, challenge, scope, resource, state, time.Now().Add(5*time.Minute).Unix()); err != nil {
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}
	return pending, nil
}
func (s *Store) OAuthPending(ctx context.Context, pending string) (OAuthIntent, error) {
	var o OAuthIntent
	if len(pending) != 68 || !strings.HasPrefix(pending, "bwp_") {
		return o, ErrOAuthInvalid
	}
	var expires int64
	err := s.db.QueryRowContext(ctx, "SELECT p.client_id,c.name,p.redirect_uri,p.challenge,p.scope,p.resource,p.state,p.expires_at FROM oauth_pending p JOIN oauth_clients c ON c.id=p.client_id WHERE p.digest=?",
		DigestToken(pending)).Scan(&o.ClientID, &o.ClientName, &o.RedirectURI, &o.Challenge, &o.Scope, &o.Resource, &o.State, &expires)
	if err != nil || expires <= time.Now().Unix() {
		return OAuthIntent{}, ErrOAuthInvalid
	}
	return o, nil
}
func (s *Store) ApproveOAuth(ctx context.Context, pending, password string) (code, redirect, stateValue string, err error) {
	if err = s.rateOAuth(ctx, "consent", 5); err != nil {
		return
	}
	intent, e := s.OAuthPending(ctx, pending)
	if e != nil {
		err = e
		return
	}
	var hashed string
	if e = s.db.QueryRowContext(ctx, "SELECT password_hash FROM oauth_config WHERE id=1 AND public_origin||'/mcp'=?", intent.Resource).Scan(&hashed); e != nil {
		err = ErrOAuthInvalid
		return
	}
	if !verifyOAuthPassword(hashed, password) {
		err = ErrOAuthInvalid
		return
	}
	role, e := parseOAuthScope(intent.Scope)
	if e != nil {
		err = e
		return
	}
	code, e = newOAuthSecret("bwa_")
	if e != nil {
		err = e
		return
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		err = e
		return
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, "DELETE FROM oauth_pending WHERE digest=? AND expires_at>? AND resource=(SELECT public_origin||'/mcp' FROM oauth_config WHERE id=1)",
		DigestToken(pending), time.Now().Unix())
	if e != nil {
		err = e
		return
	}
	n, e := res.RowsAffected()
	if e != nil || n != 1 {
		err = ErrOAuthInvalid
		return
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO oauth_codes(digest,client_id,redirect_uri,challenge,role,resource,expires_at) VALUES(?,?,?,?,?,?,?)",
		DigestToken(code), intent.ClientID, intent.RedirectURI, intent.Challenge, role, intent.Resource, time.Now().Add(2*time.Minute).Unix())
	if e != nil {
		err = e
		return
	}
	if e = tx.Commit(); e != nil {
		err = e
		return
	}
	return code, intent.RedirectURI, intent.State, nil
}
func (s *Store) ExchangeOAuth(ctx context.Context, client, redirect, code, verifier, resource string) (OAuthPair, error) {
	if len(code) != 68 || !strings.HasPrefix(code, "bwa_") || !oauthVerifier.MatchString(verifier) {
		return OAuthPair{}, ErrOAuthInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OAuthPair{}, err
	}
	defer tx.Rollback()
	var rowClient, rowRedirect, challenge, role, rowResource string
	var expires int64
	if err = tx.QueryRowContext(ctx, "SELECT client_id,redirect_uri,challenge,role,resource,expires_at FROM oauth_codes WHERE digest=?", DigestToken(code)).Scan(&rowClient, &rowRedirect, &challenge, &role, &rowResource, &expires); err != nil {
		return OAuthPair{}, ErrOAuthInvalid
	}
	var origin string
	var epoch int
	if err = tx.QueryRowContext(ctx, "SELECT public_origin,epoch FROM oauth_config WHERE id=1").Scan(&origin, &epoch); err != nil {
		return OAuthPair{}, err
	}
	if expires <= time.Now().Unix() || rowClient != client || rowRedirect != redirect || resource != rowResource || resource != origin+"/mcp" || !oauth2.CodeChallengeS256.Validate(challenge, verifier) {
		return OAuthPair{}, ErrOAuthInvalid
	}
	pair, err := s.issueOAuthPairTx(ctx, tx, client, role, resource, epoch, "")
	if err != nil {
		return OAuthPair{}, err
	}
	r, err := tx.ExecContext(ctx, "DELETE FROM oauth_codes WHERE digest=?", DigestToken(code))
	if err != nil {
		return OAuthPair{}, err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return OAuthPair{}, ErrOAuthInvalid
	}
	if err = tx.Commit(); err != nil {
		return OAuthPair{}, err
	}
	return pair, nil
}
func (s *Store) issueOAuthPairTx(ctx context.Context, tx *sql.Tx, client, role, resource string, epoch int, existingGrant string) (OAuthPair, error) {
	grant := existingGrant
	var err error
	if grant == "" {
		grant, err = newID()
		if err != nil {
			return OAuthPair{}, err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO oauth_grants(id,client_id,role,resource,epoch,created_at) VALUES(?,?,?,?,?,?)", grant, client, role, resource, epoch, time.Now().Unix())
		if err != nil {
			return OAuthPair{}, err
		}
	}
	access, err := newOAuthSecret("bwo1_")
	if err != nil {
		return OAuthPair{}, err
	}
	refresh, err := newOAuthSecret("bwr1_")
	if err != nil {
		return OAuthPair{}, err
	}
	tokenID, err := newID()
	if err != nil {
		return OAuthPair{}, err
	}
	now := time.Now().Unix()
	_, err = tx.ExecContext(ctx, "INSERT INTO credentials(id,token_digest,role,expires_at,created_at) VALUES(?,?,?,?,?)", tokenID, DigestToken(access), role, now+3600, now)
	if err != nil {
		return OAuthPair{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO oauth_access(token_id,grant_id) VALUES(?,?)", tokenID, grant)
	if err != nil {
		return OAuthPair{}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO oauth_refresh(digest,grant_id,expires_at) VALUES(?,?,?)", DigestToken(refresh), grant, now+30*86400)
	if err != nil {
		return OAuthPair{}, err
	}
	return OAuthPair{Access: access, Refresh: refresh, ExpiresIn: 3600, Scope: "mcp:" + role, GrantID: grant}, nil
}
func (s *Store) RefreshOAuth(ctx context.Context, old, client, resource string) (OAuthPair, error) {
	if len(old) != 69 || !strings.HasPrefix(old, "bwr1_") {
		return OAuthPair{}, ErrOAuthInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return OAuthPair{}, err
	}
	defer tx.Rollback()
	var grant, rowClient, role, rowResource string
	var epoch, current int
	var expires int64
	var revoked int
	err = tx.QueryRowContext(ctx, "SELECT r.grant_id,r.expires_at,g.client_id,g.role,g.resource,g.epoch,g.revoked,c.epoch FROM oauth_refresh r JOIN oauth_grants g ON g.id=r.grant_id CROSS JOIN oauth_config c WHERE r.digest=?",
		DigestToken(old)).Scan(&grant, &expires, &rowClient, &role, &rowResource, &epoch, &revoked, &current)
	if err != nil || revoked != 0 || epoch != current || expires <= time.Now().Unix() || rowClient != client || rowResource != resource {
		return OAuthPair{}, ErrOAuthInvalid
	}
	var origin string
	if err = tx.QueryRowContext(ctx, "SELECT public_origin FROM oauth_config WHERE id=1").Scan(&origin); err != nil {
		return OAuthPair{}, err
	}
	if resource != origin+"/mcp" {
		return OAuthPair{}, ErrOAuthInvalid
	}
	r, err := tx.ExecContext(ctx, "DELETE FROM oauth_refresh WHERE digest=?", DigestToken(old))
	if err != nil {
		return OAuthPair{}, err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return OAuthPair{}, ErrOAuthInvalid
	}
	// Every successor is committed together with revocation of the prior access.
	if _, err = tx.ExecContext(ctx, "UPDATE credentials SET revoked=1 WHERE id IN (SELECT token_id FROM oauth_access WHERE grant_id=?)", grant); err != nil {
		return OAuthPair{}, err
	}
	pair, err := s.issueOAuthPairTx(ctx, tx, client, role, resource, current, grant)
	if err != nil {
		return OAuthPair{}, err
	}
	if err = tx.Commit(); err != nil {
		return OAuthPair{}, err
	}
	return pair, nil
}
func (s *Store) VerifyOAuthAccess(ctx context.Context, raw string) (Token, error) {
	if len(raw) != 69 || !strings.HasPrefix(raw, "bwo1_") {
		return Token{}, ErrRevoked
	}
	var out Token
	var expires int64
	var epoch, current, revoked, grantRevoked int
	var resource, origin string
	err := s.db.QueryRowContext(ctx, "SELECT c.id,c.role,c.expires_at,c.revoked,g.revoked,g.epoch,o.epoch,g.resource,o.public_origin FROM credentials c JOIN oauth_access a ON a.token_id=c.id JOIN oauth_grants g ON g.id=a.grant_id CROSS JOIN oauth_config o WHERE o.id=1 AND c.token_digest=?",
		DigestToken(raw)).Scan(&out.ID, &out.Role, &expires, &revoked, &grantRevoked, &epoch, &current, &resource, &origin)
	if err != nil || revoked != 0 || grantRevoked != 0 || epoch != current || resource != origin+"/mcp" || origin == "" || expires <= time.Now().Unix() {
		return Token{}, ErrRevoked
	}
	out.Expires = time.Unix(expires, 0).UTC()
	return out, nil
}
func (s *Store) RevokeOAuthGrant(ctx context.Context, id string) error {
	if len(id) != 32 {
		return ErrOAuthInvalid
	}
	r, err := s.db.ExecContext(ctx, "UPDATE oauth_grants SET revoked=1 WHERE id=?", id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrOAuthInvalid
	}
	return nil
}
func (s *Store) OAuthExplain(ctx context.Context) (string, error) {
	cfg, err := s.OAuthConfig(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("configured=%t origin=%s", cfg.Enabled, cfg.Origin), nil
}

// OAuthGrantSummary is intentionally secret-free and only returned by the
// dedicated host-local CLI, never by a public MCP tool.
type OAuthGrantSummary struct {
	ID        string
	Client    string
	Role      string
	CreatedAt int64
	Active    bool
}

func (s *Store) ListOAuthGrants(ctx context.Context) ([]OAuthGrantSummary, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT g.id,c.name,g.role,g.created_at,g.revoked,g.epoch,o.epoch FROM oauth_grants g JOIN oauth_clients c ON c.id=g.client_id CROSS JOIN oauth_config o WHERE o.id=1 ORDER BY g.created_at DESC,g.id DESC LIMIT 100")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []OAuthGrantSummary{}
	for rows.Next() {
		var g OAuthGrantSummary
		var revoked, issued, current int
		if err = rows.Scan(&g.ID, &g.Client, &g.Role, &g.CreatedAt, &revoked, &issued, &current); err != nil {
			return nil, err
		}
		g.Active = revoked == 0 && issued == current
		out = append(out, g)
	}
	return out, rows.Err()
}