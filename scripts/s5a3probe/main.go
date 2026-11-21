// Standalone, inert client-capability lab. Not linked into BoxWeave.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxRequest = 64 << 10
const confirmTool = "s5a3_probe_confirm"

type observation struct {
	MetadataFields          int  `json:"metadata_fields"`
	CandidateMetadataFields int  `json:"candidate_metadata_fields"`
	CandidateHeaderFields   int  `json:"candidate_header_fields"`
	InputResponsePresent    bool `json:"input_response_present"`
	RequestStatePresent     bool `json:"request_state_present"`
	TLSVerifiedChainPresent bool `json:"tls_verified_chain_present"`
}
type result struct {
	Observation       observation `json:"observation"`
	ChallengeMatches  bool        `json:"challenge_matches"`
	SignatureVerified bool        `json:"signature_verified"`
	Authorized        bool        `json:"authorized"`
	NativeEffect      bool        `json:"native_effect"`
	Disposition       string      `json:"disposition"`
}
type observationKey struct{}

func candidate(k string) bool {
	k = strings.ToLower(k)
	for _, v := range []string{"approval", "approve", "signature", "attestation", "confirmation", "consent", "signed", "assertion"} {
		if strings.Contains(k, v) {
			return true
		}
	}
	return false
}

// Only category counts and presence bits leave this process. No request
// headers, metadata keys/values, arguments, certificates or raw JSON logged.
// A candidate field is NOT a signature and may be entirely caller-generated.
func observe(r *http.Request, body []byte) observation {
	var o observation
	o.TLSVerifiedChainPresent = r.TLS != nil && len(r.TLS.VerifiedChains) > 0
	for k := range r.Header {
		if candidate(k) {
			o.CandidateHeaderFields++
		}
	}
	var root struct {
		Method string `json:"method"`
		Params struct {
			Name  string                     `json:"name"`
			Meta  map[string]json.RawMessage `json:"_meta"`
			Input json.RawMessage            `json:"inputResponses"`
			State json.RawMessage            `json:"requestState"`
		} `json:"params"`
	}
	if json.Unmarshal(body, &root) != nil || root.Method != "tools/call" || root.Params.Name != confirmTool {
		return o
	}
	o.MetadataFields = len(root.Params.Meta)
	for k := range root.Params.Meta {
		if candidate(k) {
			o.CandidateMetadataFields++
		}
	}
	o.InputResponsePresent = len(root.Params.Input) > 0 && !bytes.Equal(root.Params.Input, []byte("null"))
	o.RequestStatePresent = len(root.Params.State) > 0 && !bytes.Equal(root.Params.State, []byte("null"))
	return o
}

type challengeOutput struct {
	Challenge string `json:"challenge"`
	Notice    string `json:"notice"`
}
type confirmInput struct {
	Challenge string `json:"challenge"`
}

func newProbe(challenge string, output io.Writer) http.Handler {
	server := mcp.NewServer(&mcp.Implementation{Name: "boxweave-s5a3-inert", Version: "0.1.0"}, nil)
	yes, no := true, false
	mcp.AddTool(server, &mcp.Tool{Name: "s5a3_probe_challenge", Description: "Read a harmless non-secret lab challenge. No Box, database or authorization is involved.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, challengeOutput, error) {
		return nil, challengeOutput{challenge, "Test only; the challenge is not a credential."}, nil
	})
	// Test write-confirmation UX only: deliberately no side effect. Never
	// register in the production tool catalog; never accept any proof here.
	mcp.AddTool(server, &mcp.Tool{Name: confirmTool, Description: "Inert test of MCP client approval metadata after a confirmation prompt. No real recovery, resource modification or authorization. Use the current lab challenge.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &yes, IdempotentHint: true, OpenWorldHint: &no}}, func(ctx context.Context, _ *mcp.CallToolRequest, in confirmInput) (*mcp.CallToolResult, result, error) {
		o, _ := ctx.Value(observationKey{}).(observation)
		res := result{Observation: o, ChallengeMatches: in.Challenge == challenge, Disposition: "UNVERIFIED_NO_AUTHORIZATION"}
		// Encoder writes only a fixed typed structure; no client-controlled strings.
		raw, _ := json.Marshal(res)
		_, _ = fmt.Fprintln(output, string(raw))
		return nil, res, nil
	})
	sdk := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: false})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mcp" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost {
			sdk.ServeHTTP(w, r)
			return
		}
		body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequest))
		if e != nil {
			http.Error(w, "bounded request refused", http.StatusRequestEntityTooLarge)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r = r.WithContext(context.WithValue(r.Context(), observationKey{}, observe(r, body)))
		sdk.ServeHTTP(w, r)
	})
}
func main() {
	// Only local lab. Deployment via an explicitly authorized private
	// connector is separate; never binds to an external guest interface.
	var seed [16]byte
	if _, e := rand.Read(seed[:]); e != nil {
		log.Fatal("challenge generation failed")
	}
	s := &http.Server{Addr: "127.0.0.1:18653", Handler: newProbe(hex.EncodeToString(seed[:]), os.Stdout), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 << 10}
	fmt.Println("S5A3_INERT_PROBE_READY loopback=127.0.0.1:18653 path=/mcp")
	if e := s.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal("probe listen failed")
	}
}
