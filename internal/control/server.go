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
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type Service struct {
	Store        *state.Store
	Native       native.Reader
	BuildVersion string
	BuildCommit  string
	Listener     string
}

type HostStatus struct {
	TrackingID    string        `json:"tracking_id"`
	Version       string        `json:"version"`
	Commit        string        `json:"commit"`
	BinarySHA256  string        `json:"binary_sha256,omitempty"`
	GoVersion     string        `json:"go_version"`
	SchemaVersion int           `json:"schema_version"`
	MCPReady      bool          `json:"mcp_ready"`
	Native        native.Status `json:"native"`
	Ready         bool          `json:"ready"`
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
	out.Ready = out.MCPReady && out.Native.Ready
	return out
}
func (s *Service) TokenVerifier(ctx context.Context, raw string, _ *http.Request) (*auth.TokenInfo, error) {
	t, err := s.Store.VerifyToken(ctx, raw)
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{UserID: t.ID, Scopes: []string{t.Role}, Expiration: t.Expires, Extra: map[string]any{"token_id": t.ID}}, nil
}
func (s *Service) Handler() http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "boxweave", Version: s.BuildVersion},
		&mcp.ServerOptions{SetCacheable: func(_ context.Context, _ mcp.Request, c *mcp.Cacheable) {
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
	mcp.AddTool(server, &mcp.Tool{Name: "host_status", Description: "Read the authenticated BoxWeave build, schema, confined Incus project and readiness; no foreign inventory.", Annotations: readToolAnnotations()},
		func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, HostStatus, error) {
			if req == nil || req.Extra == nil || req.Extra.TokenInfo == nil {
				return nil, HostStatus{}, errors.New("UNAUTHENTICATED")
			}
			info := req.Extra.TokenInfo
			if info.UserID == "" || (!contains(info.Scopes, "admin") && !contains(info.Scopes, "viewer")) {
				return nil, HostStatus{}, errors.New("PERMISSION_DENIED")
			}
			// Request-specific actor comes only from req.Extra.TokenInfo.
			result := s.Status(ctx)
			id, err := newTrackingID()
			if err != nil {
				return nil, HostStatus{}, errors.New("TRACKING_UNAVAILABLE")
			}
			result.TrackingID = id
			result.BinarySHA256 = binaryDigest()
			return &mcp.CallToolResult{}, result, nil
		})
	s.registerOperationReadTools(server)
	s.registerBoxTools(server)
	s.registerOperationOutputCancel(server)
	sdk := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{Stateless: true, PropagateRequestCancellation: false})
	secured := auth.RequireBearerToken(s.TokenVerifier, nil)(sdk)
	origin := http.NewCrossOriginProtection()
	return origin.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, _, err := net.SplitHostPort(r.Host)
		if err != nil || (h != "127.0.0.1" && h != "localhost") {
			http.Error(w, "invalid Host", http.StatusForbidden)
			return
		}
		if r.Header.Get("Origin") != "" {
			// An operator-managed TLS proxy is a separately tested deployment.
			http.Error(w, "origin not permitted", http.StatusForbidden)
			return
		}
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		secured.ServeHTTP(w, r)
	}))
}
func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
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
	gcCtx,gcCancel:=context.WithCancel(context.Background())
	defer gcCancel()
	go s.execGCLoop(gcCtx)
	return server.ListenAndServe()
}
