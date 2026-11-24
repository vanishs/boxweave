// Package control exposes the exact S1 public MCP surface using the official SDK.
package control

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vanishs/boxweave/internal/install"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// privateMCPWriter applies the actor-private policy at the actual response
// commit boundary. The pinned MCP SDK writes "no-cache, no-transform" AFTER
// our handler sets headers; no-cache alone permits storage in shared caches.
// Preserve the SDK no-transform intent and keep flushing/ResponseController
// capabilities, without buffering guest output in the controller.
type privateMCPWriter struct {
	http.ResponseWriter
}

func (w *privateMCPWriter) applyPolicy() {
	w.Header().Set("Cache-Control", "private, no-store, no-cache, no-transform")
	// An SSE response may flush many times. Never append duplicate Vary
	// values on each chunk, and preserve any other SDK-supplied Vary keys.
	if !strings.Contains(strings.ToLower(strings.Join(w.Header().Values("Vary"), ",")),
		"authorization") {
		w.Header().Add("Vary", "Authorization")
	}
}

func (w *privateMCPWriter) WriteHeader(status int) {
	w.applyPolicy()
	w.ResponseWriter.WriteHeader(status)
}

func (w *privateMCPWriter) Write(p []byte) (int, error) {
	w.applyPolicy()
	return w.ResponseWriter.Write(p)
}

func (w *privateMCPWriter) Flush() {
	w.applyPolicy()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *privateMCPWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type Service struct {
	ScopedPeerFresh bool
	Store           *state.Store
	Native          native.Reader
	BuildVersion    string
	BuildCommit     string
	Listener        string
}

type HostStatus struct {
	TrackingID            string                      `json:"tracking_id"`
	Version               string                      `json:"version"`
	Commit                string                      `json:"commit"`
	BinarySHA256          string                      `json:"binary_sha256,omitempty"`
	GoVersion             string                      `json:"go_version"`
	SchemaVersion         int                         `json:"schema_version"`
	MCPReady              bool                        `json:"mcp_ready"`
	Native                native.Status               `json:"native"`
	Ready                 bool                        `json:"ready"`
	Attention             *attentionSnapshot          `json:"attention,omitempty"`
	ProjectCapacity       *state.ProjectBoxCapacity   `json:"project_capacity,omitempty"`
	AddressCapacity       *native.AddressCapacity     `json:"address_capacity,omitempty"`
	AddressRetirement     *state.IPv4RetirementStatus `json:"address_retirement,omitempty"`
	AddressCapacityStatus string                      `json:"address_capacity_status,omitempty"`
}

func newTrackingID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func binaryDigest() string {
	name, err := os.Executable()
	if err != nil {
		return ""
	}
	f, err := os.Open(name)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}
func (s *Service) Status(ctx context.Context) HostStatus {
	v, err := s.Store.Schema(ctx)
	out := HostStatus{Version: s.BuildVersion, Commit: s.BuildCommit, GoVersion: runtime.Version(), MCPReady: err == nil && v == state.SchemaVersion, SchemaVersion: v}
	out.Native = s.Native.Read(ctx)
	if cap, e := s.Store.ReadProjectBoxCapacity(ctx, s.Native.Project); e == nil {
		out.ProjectCapacity = &cap
	}
	out.Ready = out.MCPReady && out.Native.Ready
	if out.MCPReady {
		if retirement, e := s.Store.ReadIPv4RetirementStatus(ctx, s.Native.Project); e == nil {
			if retirement.Pending > 0 && !install.NativeRetirementSocketPresent() {
				retirement.LastDeferredReason = "RETIREMENT_HELPER_NOT_CONFIGURED"
			}
			out.AddressRetirement = &retirement
		}
	}
	if out.Ready {
		if capacity, e := s.Native.ReadCreateAddressCapacity(ctx, s.Store.ReadCreateAddressHolds); e == nil {
			out.AddressCapacity = &capacity
			out.AddressCapacityStatus = "OBSERVED"
			if capacity.RemainingIPv4UpperBound == 0 {
				out.AddressCapacityStatus = "EXHAUSTED"
			}
		} else {
			out.AddressCapacityStatus = "UNVERIFIED"
		}
	}
	return out
}
func (s *Service) TokenVerifier(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
	t, err := s.Store.VerifyToken(ctx, raw)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{UserID: t.ActorID, Expiration: t.Expires, Extra: map[string]any{"token_id": t.ID}}, nil
}
func (s *Service) Handler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "boxweave", Version: s.BuildVersion},
		&mcp.ServerOptions{Instructions: serverContinuityInstructions, SetCacheable: func(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
			// Every resource and Operation is project/actor-private and
			// status changes while work is in flight. Never cache publicly.
			c.CacheScope = "private"
			c.TTLMs = 0
		}})
	server.AddReceivingMiddleware(func(next mcp.MethodHandler) mcp.MethodHandler {
		return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
			start := time.Now()
			result, err := next(ctx, method, req)
			actor := "unauthenticated"
			if extra := req.GetExtra(); extra != nil && extra.TokenInfo != nil {
				// Redacted immutable credential reference, NOT token, params,
				// argv, env, guest output, prompt or private native message.
				h := sha256.Sum256([]byte(extra.TokenInfo.UserID))
				actor = hex.EncodeToString(h[:4])
			}
			outcome := "ok"
			if err != nil {
				outcome = "error"
			}
			slog.InfoContext(ctx, "mcp_request", "method", method,
				"actor", actor, "duration_ms", time.Since(start).Milliseconds(),
				"result", outcome)
			return result, err
		}
	})
	addStrictTool(server, &mcp.Tool{Name: "host_status", Description: "Read the authenticated BoxWeave build, schema, confined Incus project and readiness; attention shows bounded same-Project SQLite blockers/dependencies; project_capacity counts owned and unresolved Create slots, not native address/storage availability or permission to dispatch; address_capacity reports scoped IPv4 snapshot upper bound, never an allocation guarantee; no foreign inventory.", Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, HostStatus, error) {
			principal, err := s.requestPrincipal(req)
			if err != nil {
				return nil, HostStatus{}, err
			}
			// Request-specific actor comes only from req.Extra.TokenInfo.
			result := s.Status(ctx)
			id, err := newTrackingID()
			if err != nil {
				return nil, HostStatus{}, errors.New("TRACKING_UNAVAILABLE")
			}
			result.TrackingID = id
			result.BinarySHA256 = binaryDigest()
			attention := s.readAttention(ctx, principal, "")
			result.Attention = &attention
			return privateReadReply(ctx, s, principal.TokenID, result)
		})
	s.registerOperationReadTools(server)
	s.registerRecoveryTool(server)
	s.registerActivityTools(server)
	s.registerForwardTools(server)
	s.registerBoxTools(server)
	s.registerReadinessTool(server)
	s.registerPeerTools(server)
	s.registerFileTools(server)
	s.registerOperationOutputCancel(server)
	sdk := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: false})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w = &privateMCPWriter{ResponseWriter: w}
		cfg := state.OAuthSettings{}
		if s.Store != nil {
			var err error
			cfg, err = s.Store.OAuthConfig(r.Context())
			if err != nil {
				http.Error(w, "authorization state unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		// The caller controls neither issuer nor resource identity. Only an
		// exact host configured by a host-local actor can traverse the public
		// route; arbitrary forwarded-proxy headers are deliberately ignored.
		h, _, splitErr := net.SplitHostPort(r.Host)
		local := splitErr == nil && (h == "127.0.0.1" || h == "localhost")
		public := cfg.Enabled && r.Host == strings.TrimPrefix(cfg.Origin, "https://")
		if !local && !public {
			http.Error(w, "invalid Host", http.StatusForbidden)
			return
		}
		if orig := r.Header.Get("Origin"); orig != "" && (orig == "null" || !cfg.Enabled || orig != cfg.Origin) {
			http.Error(w, "invalid Origin", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/mcp" {
			s.oauthRoute(w, r, cfg)
			return
		}
		// Fail closed even for an unconfigured test/local handler; an empty
		// store must never reach typed tools or silently allow anonymous MCP.
		if s.Store == nil {
			http.Error(w, "no bearer token", http.StatusUnauthorized)
			return
		}
		var opts *auth.RequireBearerTokenOptions
		if cfg.Enabled {
			opts = &auth.RequireBearerTokenOptions{ResourceMetadataURL: oauthMetadataURL(cfg.Origin)}
		}
		// The official SDK performs a second localhost Host safety check.
		// After *independently* checking exact public authority and origin,
		// give the SDK its private upstream Host, not any Forwarded header.
		upstream := r.Clone(r.Context())
		upstream.Host = s.Listener
		auth.RequireBearerToken(s.TokenVerifier, opts)(sdk).ServeHTTP(w, upstream)
	})
}
func (s *Service) ListenAndServe() error {
	if s.Store == nil || s.Listener == "" {
		return errors.New("service unconfigured")
	}
	h, p, err := net.SplitHostPort(s.Listener)
	if err != nil || h != "127.0.0.1" || p == "" {
		return errors.New("MCP listener must bind 127.0.0.1")
	}
	if os.Geteuid() == 0 {
		return errors.New("refusing to run MCP service as root")
	}
	server := &http.Server{Addr: s.Listener, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 16}
	gcCtx, gcCancel := context.WithCancel(context.Background())
	defer gcCancel()
	go s.execGCLoop(gcCtx)
	if !s.ScopedPeerFresh {
		go s.ipv4RetirementLoop(gcCtx)
	}
	return server.ListenAndServe()
}
