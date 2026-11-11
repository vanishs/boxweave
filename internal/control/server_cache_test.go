package control

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMCPHTTPNeverCachesPrivateOrDeniedResponses(t *testing.T) {
	svc := &Service{}
	for _, tc := range []struct {
		name string
		host string
		want int
	}{
		{name: "unauthenticated", host: "127.0.0.1:18640", want: http.StatusUnauthorized},
		{name: "foreign-host", host: "untrusted.invalid:18640", want: http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:18640/mcp",
				strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			request.Host = tc.host
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			response := httptest.NewRecorder()
			svc.Handler().ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("status %d, want %d", response.Code, tc.want)
			}
			if got := strings.ToLower(response.Header().Get("Cache-Control")); got != "private, no-store" {
				t.Fatalf("private MCP response has unsafe cache policy: %q", got)
			}
			if got := response.Header().Values("Vary"); !strings.Contains(strings.Join(got, ","), "Authorization") {
				t.Fatalf("actor-specific HTTP response missing Authorization Vary: %q", got)
			}
		})
	}
}

func TestPrivateMCPWriterOverridesLateSDKCacheControl(t *testing.T) {
	for _, tc := range []struct {
		name string
		write func(http.ResponseWriter)
	}{
		{name: "write-header", write: func(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) }},
		{name: "implicit-write", write: func(w http.ResponseWriter) { _, _ = w.Write([]byte("ok")) }},
		{name: "flush", write: func(w http.ResponseWriter) { w.(http.Flusher).Flush() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			protected := &privateMCPWriter{ResponseWriter: rec}
			protected.Header().Set("Cache-Control", "no-cache, no-transform")
			tc.write(protected)
			cache := strings.ToLower(rec.Header().Get("Cache-Control"))
			if !strings.Contains(cache, "private") || !strings.Contains(cache, "no-store") ||
				!strings.Contains(cache, "no-transform") {
				t.Fatalf("SDK cache header was not replaced at response commit: %q", cache)
			}
		})
	}
}

