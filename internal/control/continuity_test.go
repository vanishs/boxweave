package control

import (
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

func TestS2SDKNativeCorrelationMetadataAndFallback(t *testing.T) {
	tests := []struct {
		name          string
		meta          mcp.Meta
		argumentWork  string
		argumentAgent string
		wantWork      string
		wantAgent     string
		wantErr       error
	}{
		{"legacy-no-metadata", nil, "work/alpha", "agent-1", "work/alpha", "agent-1", nil},
		{"metadata-only", mcp.Meta{"io.boxweave.work_ref": "work/alpha",
			"io.boxweave.agent_ref": "agent-2"}, "", "", "work/alpha", "agent-2", nil},
		{"agreed-metadata", mcp.Meta{"io.boxweave.work_ref": "work/alpha"},
			"work/alpha", "agent-1", "work/alpha", "agent-1", nil},
		{"contradictory", mcp.Meta{"io.boxweave.work_ref": "work/beta"},
			"work/alpha", "", "", "", state.ErrConflict},
		{"invalid-control", mcp.Meta{"io.boxweave.agent_ref": "agent\nsecret"},
			"", "", "", "", state.ErrInvalid},
		{"unsupported-type", mcp.Meta{"io.boxweave.agent_ref": 32},
			"", "", "", "", state.ErrInvalid},
		{"untrusted-URL", mcp.Meta{"io.boxweave.work_ref": "https://foreign.example"},
			"", "", "", "", state.ErrInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Meta: tc.meta}}
			w, a, err := boundedWorkLabels(req, tc.argumentWork, tc.argumentAgent)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("correlation error got %v want %v", err, tc.wantErr)
			}
			if err == nil && (w != tc.wantWork || a != tc.wantAgent) {
				t.Fatalf("correlation mismatch: %q/%q", w, a)
			}
		})
	}
}

func TestS2PrivateCorrelationDoesNotAuthenticate(t *testing.T) {
	st, _, _, _ := fixture(t)
	svc := &Service{Store: st, Native: native.Reader{Project: "bw-test"}}
	req := &mcp.CallToolRequest{
		Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID: ""}},
		Params: &mcp.CallToolParamsRaw{Meta: mcp.Meta{
			"io.boxweave.work_ref":  "spoofed/admin-work",
			"io.boxweave.agent_ref": "admin-agent",
		}},
	}
	if _, err := svc.requestPrincipal(req); err == nil {
		t.Fatal("unauthenticated request acquired management via self-declared metadata")
	}
}

func TestS2RequestPrincipalRequiresExplicitSDKCredential(t *testing.T) {
	st, _, _, _ := fixture(t)
	svc := &Service{Store: st, Native: native.Reader{Project: "bw-test"}}
	actor := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	credential := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	tests := []struct {
		name  string
		token any
		want  bool
	}{
		{"missing", nil, false},
		{"numeric", 7, false},
		{"empty", "", false},
		{"malformed", "not-an-id", false},
		{"valid distinct OAuth access ID", credential, true},
		{"valid manual credential", actor, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &mcp.CallToolRequest{Extra: &mcp.RequestExtra{
				TokenInfo: &auth.TokenInfo{UserID: actor, Extra: map[string]any{}},
			}}
			if tc.token != nil {
				req.Extra.TokenInfo.Extra["token_id"] = tc.token
			}
			principal, err := svc.requestPrincipal(req)
			if tc.want && (err != nil || principal.ActorID != actor || principal.Layer != "bw-test" ||
				principal.TokenID != tc.token) {
				t.Fatalf("valid SDK principal was not retained: %+v, %v", principal, err)
			}
			if !tc.want && (err == nil || principal != (RequestPrincipal{})) {
				t.Fatalf("missing/invalid independent token ID borrowed actor identity: %+v, %v", principal, err)
			}
		})
	}
}
