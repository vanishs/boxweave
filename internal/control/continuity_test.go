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
		Extra: &mcp.RequestExtra{TokenInfo: &auth.TokenInfo{UserID:""}},
		Params: &mcp.CallToolParamsRaw{Meta: mcp.Meta{
			"io.boxweave.work_ref":"spoofed/admin-work",
			"io.boxweave.agent_ref":"admin-agent",
		}},
	}
	if _, _, err := requireAuthenticated(req, svc); err == nil {
		t.Fatal("unauthenticated request acquired management via self-declared metadata")
	}
}
