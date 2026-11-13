package control

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/url"
	"strings"

	oauth2 "github.com/go-oauth2/oauth2/v4"
	oauthserver "github.com/go-oauth2/oauth2/v4/server"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/vanishs/boxweave/internal/state"
)

// Auth routes are a credential front-end, not a second MCP implementation.
// All successful /mcp calls still traverse the SDK's bearer middleware and
// the original request-scoped admin/viewer/owned-UUID checks.
func oauthValidator() *oauthserver.Server {
	cfg := oauthserver.NewConfig()
	cfg.AllowedResponseTypes = []oauth2.ResponseType{oauth2.Code}
	cfg.AllowedGrantTypes = []oauth2.GrantType{oauth2.AuthorizationCode, oauth2.Refreshing}
	cfg.AllowedCodeChallengeMethods = []oauth2.CodeChallengeMethod{oauth2.CodeChallengeS256}
	cfg.ForcePKCE = true
	srv := oauthserver.NewServer(cfg, nil)
	srv.SetClientInfoHandler(oauthserver.ClientFormHandler)
	return srv
}
func oauthJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func oauthError(w http.ResponseWriter, err error) {
	if errors.Is(err, state.ErrOAuthRateLimited) {
		w.Header().Set("Retry-After", "60")
		oauthJSON(w, http.StatusTooManyRequests, map[string]string{"error": "temporarily_unavailable"})
		return
	}
	oauthJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
}
func oauthMethod(w http.ResponseWriter, want string) {
	w.Header().Set("Allow", want)
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}
func (s *Service) oauthRoute(w http.ResponseWriter, r *http.Request, cfg state.OAuthSettings) {
	if !cfg.Enabled {
		http.NotFound(w, r)
		return
	}
	switch r.URL.Path {
	case "/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp":
		meta := &oauthex.ProtectedResourceMetadata{
			Resource: cfg.Origin + "/mcp", AuthorizationServers: []string{cfg.Origin},
			ScopesSupported:        []string{"mcp:viewer", "mcp:admin"},
			BearerMethodsSupported: []string{"header"},
			ResourceName:           "BoxWeave MCP",
		}
		auth.ProtectedResourceMetadataHandler(meta).ServeHTTP(w, r)
	case "/.well-known/oauth-authorization-server":
		if r.Method != http.MethodGet {
			oauthMethod(w, "GET")
			return
		}
		oauthJSON(w, 200, map[string]any{
			"issuer": cfg.Origin, "authorization_response_iss_parameter_supported": true, "authorization_endpoint": cfg.Origin + "/authorize",
			"token_endpoint": cfg.Origin + "/token", "registration_endpoint": cfg.Origin + "/register",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"scopes_supported":                      []string{"mcp:viewer", "mcp:admin"},
		})
	case "/register":
		s.oauthRegister(w, r)
	case "/authorize":
		s.oauthAuthorize(w, r, cfg)
	case "/token":
		s.oauthToken(w, r, cfg)
	default:
		http.NotFound(w, r)
	}
}
func (s *Service) oauthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		oauthMethod(w, "POST")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	var req struct {
		Name      string   `json:"client_name"`
		Secret    string   `json:"client_secret"`
		Redirects []string `json:"redirect_uris"`
		Method    string   `json:"token_endpoint_auth_method"`
		Grants    []string `json:"grant_types"`
		Response  []string `json:"response_types"`
	}
	// This JSON is public client metadata, never a credential.
	decoder := json.NewDecoder(r.Body)
	// RFC 7591 permits optional client display metadata. Ignore such fields
	// rather than rejecting ChatGPT or a future standards-compliant client.
	if err := decoder.Decode(&req); err != nil {
		oauthError(w, state.ErrOAuthInvalid)
		return
	}
	if req.Name == "" {
		req.Name = "MCP client"
	}
	if req.Secret != "" || (req.Method != "" && req.Method != "none") {
		oauthError(w, state.ErrOAuthInvalid)
		return
	}
	for _, v := range req.Grants {
		if v != "authorization_code" && v != "refresh_token" {
			oauthError(w, state.ErrOAuthInvalid)
			return
		}
	}
	for _, v := range req.Response {
		if v != "code" {
			oauthError(w, state.ErrOAuthInvalid)
			return
		}
	}
	id, err := s.Store.RegisterOAuthClient(r.Context(), req.Name, req.Redirects)
	if err != nil {
		oauthError(w, err)
		return
	}
	oauthJSON(w, http.StatusCreated, map[string]any{
		"client_id": id, "client_name": req.Name, "redirect_uris": req.Redirects,
		"token_endpoint_auth_method": "none",
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
	})
}

var oauthConsentPage = template.Must(template.New("consent").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><title>BoxWeave MCP authorization</title><body>
<h1>Authorize MCP client</h1><p>This client will be allowed to use your BoxWeave MCP tools.</p>
<p>Client: <strong>{{.ClientName}}</strong></p><p>Callback: <strong>{{.RedirectURI}}</strong></p>
<p>Requested access: <strong>{{.Role}}</strong> (admin permits Box mutations and Exec; viewer is read-only).</p>
<form action="/authorize" method="post">
<input type="hidden" name="pending" value="{{.Pending}}">
<label>Local BoxWeave OAuth approval password <input type="password" name="password" required autocomplete="current-password"></label>
<label><input type="checkbox" name="approve" value="yes" required> I approve this client and its requested role.</label>
<button type="submit">Authorize</button></form></body></html>`))

func (s *Service) oauthAuthorize(w http.ResponseWriter, r *http.Request, cfg state.OAuthSettings) {
	switch r.Method {
	case http.MethodGet:
		// Use the upstream OAuth library for request/PKCE S256/response_type
		// validation; the durable store checks registered exact redirects,
		// resource, scopes and session. Do not use the library's non-atomic Manager.
		req, err := oauthValidator().ValidationAuthorizeRequest(r)
		if err != nil {
			oauthError(w, err)
			return
		}
		pending, err := s.Store.BeginOAuth(r.Context(), req.ClientID, req.RedirectURI, req.Scope,
			req.CodeChallenge, r.URL.Query().Get("resource"), req.State)
		if err != nil {
			oauthError(w, err)
			return
		}
		intent, err := s.Store.OAuthPending(r.Context(), pending)
		if err != nil {
			oauthError(w, err)
			return
		}
		role := "viewer"
		if strings.Contains(" "+intent.Scope+" ", " mcp:admin ") {
			role = "admin"
		}
		callback, err := url.Parse(intent.RedirectURI)
		if err != nil {
			oauthError(w, err)
			return
		}
		origin := callback.Scheme + "://" + callback.Host
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self' "+origin+"; frame-ancestors 'none'")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = oauthConsentPage.Execute(w, struct{ ClientName, RedirectURI, Role, Pending string }{intent.ClientName, intent.RedirectURI, role, pending})
	case http.MethodPost:
		if r.Host != strings.TrimPrefix(cfg.Origin, "https://") || r.Header.Get("Origin") != cfg.Origin {
			http.Error(w, "invalid origin", http.StatusForbidden)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		if err := r.ParseForm(); err != nil {
			oauthError(w, err)
			return
		}
		if r.PostForm.Get("approve") != "yes" {
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}
		code, callback, stateValue, err := s.Store.ApproveOAuth(r.Context(), r.PostForm.Get("pending"), r.PostForm.Get("password"))
		if err != nil {
			if errors.Is(err, state.ErrOAuthRateLimited) {
				oauthError(w, err)
			} else {
				http.Error(w, "access denied", http.StatusForbidden)
			}
			return
		}
		target, err := url.Parse(callback)
		if err != nil {
			http.Error(w, "callback invalid", http.StatusInternalServerError)
			return
		}
		q := target.Query()
		q.Set("code", code)
		q.Set("state", stateValue)
		// RFC 9207 issuer identification binds the callback to the
		// authorization server the client selected during discovery.
		q.Set("iss", cfg.Origin)
		target.RawQuery = q.Encode()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		http.Redirect(w, r, target.String(), http.StatusFound)
	default:
		oauthMethod(w, "GET, POST")
	}
}
func (s *Service) oauthToken(w http.ResponseWriter, r *http.Request, cfg state.OAuthSettings) {
	if r.Method != http.MethodPost {
		oauthMethod(w, "POST")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		oauthError(w, state.ErrOAuthInvalid)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	if err := r.ParseForm(); err != nil {
		oauthError(w, err)
		return
	}
	grant, req, err := oauthValidator().ValidationTokenRequest(r)
	if err != nil || req == nil {
		oauthJSON(w, 400, map[string]string{"error": "invalid_request"})
		return
	}
	if req.ClientSecret != "" {
		oauthJSON(w, 401, map[string]string{"error": "invalid_client"})
		return
	}
	client, resource := req.ClientID, r.Form.Get("resource")
	if resource != cfg.Origin+"/mcp" {
		oauthJSON(w, 400, map[string]string{"error": "invalid_target"})
		return
	}
	var pair state.OAuthPair
	switch grant {
	case oauth2.AuthorizationCode:
		pair, err = s.Store.ExchangeOAuth(r.Context(), client, req.RedirectURI, req.Code, req.CodeVerifier, resource)
	case oauth2.Refreshing:
		pair, err = s.Store.RefreshOAuth(r.Context(), req.Refresh, client, resource)
	default:
		err = state.ErrOAuthInvalid
	}
	if err != nil {
		oauthJSON(w, 400, map[string]string{"error": "invalid_grant"})
		return
	}
	oauthJSON(w, 200, map[string]any{"access_token": pair.Access, "token_type": "Bearer", "expires_in": pair.ExpiresIn,
		"refresh_token": pair.Refresh, "scope": pair.Scope})
}
func oauthMetadataURL(origin string) string {
	return origin + "/.well-known/oauth-protected-resource/mcp"
}
