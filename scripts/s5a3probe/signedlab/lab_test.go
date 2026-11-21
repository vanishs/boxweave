package signedlab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const layer = "lab-layer"
const actor = "lab-actor"
const original = "lab-original"
const key = "lab-recovery-key-001"
const key2 = "lab-recovery-key-002"

type fixture struct {
	l    *Lab
	priv ed25519.PrivateKey
	v    verifier
	now  time.Time
	path string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pub, priv, e := ed25519.GenerateKey(rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	f := &fixture{priv: priv, v: verifier{"synthetic-trusted-client", "synthetic-server", "synthetic-key-1", pub}, now: time.Unix(1800000000, 0), path: filepath.Join(t.TempDir(), "s5a3-synthetic.sqlite")}
	f.l, e = openLab(f.path, f.v, func() time.Time { return f.now })
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { f.l.db.Close() })
	f.exec(t, `INSERT INTO lab_actors VALUES('lab-layer','lab-actor',1,1),('lab-layer','other-actor',1,1),('other-layer','lab-actor',1,1);
 INSERT INTO lab_targets(layer,original,action,intent,uuid,incarnation,evidence,revision,state) VALUES
 ('lab-layer','lab-original','REVIEWED_RELEASE','intent-1','uuid-1','incarnation-1','evidence-1',1,'UNKNOWN'),
 ('lab-layer','other-original','REVIEWED_RELEASE','intent-2','uuid-2','incarnation-2','evidence-2',1,'UNKNOWN'),
 ('other-layer','lab-original','REVIEWED_RELEASE','intent-1','uuid-1','incarnation-1','evidence-1',1,'UNKNOWN');`)
	return f
}
func (f *fixture) exec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, e := f.l.db.Exec(q, args...); e != nil {
		t.Fatal(e)
	}
}
func (f *fixture) issue(t *testing.T, k string) Challenge {
	t.Helper()
	c, e := f.l.issue(context.Background(), layer, actor, original, k)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func sign(priv ed25519.PrivateKey, c Challenge, d string) Assertion {
	return Assertion{c, d, base64.RawURLEncoding.EncodeToString(ed25519.Sign(priv, message(c, d)))}
}
func (f *fixture) approve(c Challenge) Assertion { return sign(f.priv, c, "approve") }
func (f *fixture) consume(a Assertion) (Receipt, error) {
	return f.l.consume(context.Background(), layer, actor, original, key, a)
}
func (f *fixture) assertState(t *testing.T, wantReleased, wantReceipts, wantConsumed int) {
	t.Helper()
	var released, receipts, consumed, audits int
	var state string
	if e := f.l.db.QueryRow("SELECT state,released FROM lab_targets WHERE layer=? AND original=?", layer, original).Scan(&state, &released); e != nil {
		t.Fatal(e)
	}
	for _, q := range []struct {
		s string
		p *int
	}{{"SELECT count(*) FROM lab_receipts", &receipts}, {"SELECT count(*) FROM lab_approvals WHERE status='CONSUMED'", &consumed}, {"SELECT count(*) FROM lab_audit WHERE event LIKE 'CONSUMED_%'", &audits}} {
		if e := f.l.db.QueryRow(q.s).Scan(q.p); e != nil {
			t.Fatal(e)
		}
	}
	if state != "UNKNOWN" || released != wantReleased || receipts != wantReceipts || consumed != wantConsumed || audits != wantConsumed {
		t.Fatalf("inconsistent state %s %d %d %d %d", state, released, receipts, consumed, audits)
	}
}
func TestS5A3SignedPositiveRestartReplay(t *testing.T) {
	f := newFixture(t)
	c := f.issue(t, key)
	if again := f.issue(t, key); again != c {
		t.Fatal("issue retry changed challenge")
	}
	a := f.approve(c)
	r, e := f.consume(a)
	if e != nil || !r.ClaimReleased || r.NativeEffect || r.OriginalState != "UNKNOWN" || r.Duplicate || r.ID == "" {
		t.Fatalf("positive: %+v %v", r, e)
	}
	f.assertState(t, 1, 1, 1)
	// Reopen the actual SQLite file after the simulated lost response.
	if e = f.l.db.Close(); e != nil {
		t.Fatal(e)
	}
	f.l, e = openLab(f.path, f.v, func() time.Time { return f.now })
	if e != nil {
		t.Fatal(e)
	}
	f.now = f.now.Add(10 * time.Minute)
	again, e := f.consume(a)
	if e != nil || !again.Duplicate || again.ID != r.ID || again.NativeEffect {
		t.Fatalf("replay: %+v %v", again, e)
	}
	if _, e = f.consume(sign(f.priv, c, "deny")); !errors.Is(e, ErrDenied) {
		t.Fatal("different decision reused approval")
	}
	f.assertState(t, 1, 1, 1)
}
func TestS5A3SignedNegativeMatrix(t *testing.T) {
	for _, name := range []string{"unsigned", "forged", "changed-signature", "issuer", "audience", "key-id", "nonce", "layer", "actor", "original", "action", "intent", "uuid", "incarnation", "evidence", "key", "epoch", "revision", "decision", "expires", "expired", "future", "revoked", "epoch-changed", "stale-evidence", "stale-revision", "changed-uuid", "state-changed", "wrong-layer", "wrong-actor", "wrong-original", "wrong-key"} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			c := f.issue(t, key)
			a := f.approve(c)
			l, who, o, k := layer, actor, original, key
			switch name {
			case "unsigned":
				a.Signature = ""
			case "forged":
				_, p, e := ed25519.GenerateKey(rand.Reader)
				if e != nil {
					t.Fatal(e)
				}
				a = sign(p, c, "approve")
			case "changed-signature":
				sig, _ := base64.RawURLEncoding.DecodeString(a.Signature)
				sig[0] ^= 1
				a.Signature = base64.RawURLEncoding.EncodeToString(sig)
			// Re-sign altered fields with even the pinned test key: the server-
			// persisted exact challenge must still reject all changed bindings.
			case "issuer":
				c.Issuer = "wrong"
				a = f.approve(c)
			case "audience":
				c.Audience = "other"
				a = f.approve(c)
			case "key-id":
				c.KeyID = "other"
				a = f.approve(c)
			case "nonce":
				c.ID = "other"
				a = f.approve(c)
			case "layer":
				c.Binding.Layer = "other-layer"
				a = f.approve(c)
			case "actor":
				c.Binding.Actor = "other-actor"
				a = f.approve(c)
			case "original":
				c.Binding.Original = "other-original"
				a = f.approve(c)
			case "action":
				c.Binding.Action = "NATIVE_REPLAY"
				a = f.approve(c)
			case "intent":
				c.Binding.Intent = "changed"
				a = f.approve(c)
			case "uuid":
				c.Binding.UUID = "changed"
				a = f.approve(c)
			case "incarnation":
				c.Binding.Incarnation = "changed"
				a = f.approve(c)
			case "evidence":
				c.Binding.Evidence = "changed"
				a = f.approve(c)
			case "key":
				c.Binding.RecoveryKey = hexDigest([]byte(key2))
				a = f.approve(c)
			case "epoch":
				c.Binding.Epoch++
				a = f.approve(c)
			case "revision":
				c.Binding.Revision++
				a = f.approve(c)
			case "decision":
				a = sign(f.priv, c, "approved=true")
			case "expires":
				c.Expires++
				a = f.approve(c)
			case "expired":
				f.now = f.now.Add(ttl)
			case "future":
				f.now = f.now.Add(-time.Second)
			case "revoked":
				f.exec(t, "UPDATE lab_actors SET active=0 WHERE layer=? AND actor=?", layer, actor)
			case "epoch-changed":
				f.exec(t, "UPDATE lab_actors SET epoch=2 WHERE layer=? AND actor=?", layer, actor)
			case "stale-evidence":
				f.exec(t, "UPDATE lab_targets SET evidence='new' WHERE layer=? AND original=?", layer, original)
			case "stale-revision":
				f.exec(t, "UPDATE lab_targets SET revision=2 WHERE layer=? AND original=?", layer, original)
			case "changed-uuid":
				f.exec(t, "UPDATE lab_targets SET uuid='new' WHERE layer=? AND original=?", layer, original)
			case "state-changed":
				f.exec(t, "UPDATE lab_targets SET state='SUCCEEDED' WHERE layer=? AND original=?", layer, original)
			case "wrong-layer":
				l = "other-layer"
			case "wrong-actor":
				who = "other-actor"
			case "wrong-original":
				o = "other-original"
			case "wrong-key":
				k = key2
			}
			if r, e := f.l.consume(context.Background(), l, who, o, k, a); !errors.Is(e, ErrDenied) || r.ClaimReleased {
				t.Fatalf("accepted %s: %+v %v", name, r, e)
			}
			if name == "state-changed" {
				f.exec(t, "UPDATE lab_targets SET state='UNKNOWN' WHERE layer=? AND original=?", layer, original)
			}
			f.assertState(t, 0, 0, 0)
		})
	}
}
func TestS5A3SignedDeny(t *testing.T) {
	f := newFixture(t)
	c := f.issue(t, key)
	r, e := f.consume(sign(f.priv, c, "deny"))
	if e != nil || r.ClaimReleased || r.Decision != "deny" || r.NativeEffect {
		t.Fatalf("deny: %+v %v", r, e)
	}
	f.assertState(t, 0, 1, 1)
	if _, e = f.consume(f.approve(c)); !errors.Is(e, ErrDenied) {
		t.Fatal("denied approval reused")
	}
}
func TestS5A3SignedAtomicRollback(t *testing.T) {
	f := newFixture(t)
	a := f.approve(f.issue(t, key))
	f.exec(t, "CREATE TRIGGER fault BEFORE INSERT ON lab_receipts BEGIN SELECT RAISE(ABORT,'test fault'); END")
	if r, e := f.consume(a); e == nil || r.ClaimReleased {
		t.Fatalf("fault accepted %+v %v", r, e)
	}
	f.assertState(t, 0, 0, 0)
	f.exec(t, "DROP TRIGGER fault")
	if _, e := f.consume(a); e != nil {
		t.Fatal(e)
	}
	f.assertState(t, 1, 1, 1)
}
func TestS5A3SignedConcurrentConsumption(t *testing.T) {
	for _, same := range []bool{true, false} {
		t.Run(map[bool]string{true: "same-key", false: "different-approvals"}[same], func(t *testing.T) {
			f := newFixture(t)
			a := f.approve(f.issue(t, key))
			a2, k2 := a, key
			if !same {
				a2, k2 = f.approve(f.issue(t, key2)), key2
			}
			other, e := openLab(f.path, f.v, func() time.Time { return f.now })
			if e != nil {
				t.Fatal(e)
			}
			defer other.db.Close()
			type outcome struct {
				r Receipt
				e error
			}
			out := make(chan outcome, 2)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for _, x := range []struct {
				l *Lab
				k string
				a Assertion
			}{{f.l, key, a}, {other, k2, a2}} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					r, e := x.l.consume(context.Background(), layer, actor, original, x.k, x.a)
					out <- outcome{r, e}
				}()
			}
			close(start)
			wg.Wait()
			close(out)
			ok, dups, denied := 0, 0, 0
			var id string
			for v := range out {
				if v.e == nil {
					ok++
					if id != "" && id != v.r.ID {
						t.Fatal("two receipts")
					}
					id = v.r.ID
					if v.r.Duplicate {
						dups++
					}
				} else if errors.Is(v.e, ErrDenied) {
					denied++
				} else {
					t.Fatal(v.e)
				}
			}
			if same && (ok != 2 || dups != 1 || denied != 0) || !same && (ok != 1 || dups != 0 || denied != 1) {
				t.Fatalf("race %d %d %d", ok, dups, denied)
			}
			f.assertState(t, 1, 1, 1)
		})
	}
}

type labTokenTransport struct{ token string }

func (x labTokenTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	q := r.Clone(r.Context())
	q.Header.Set("Authorization", "Bearer "+x.token)
	return http.DefaultTransport.RoundTrip(q)
}

type issueInput struct {
	Original string `json:"original"`
	Key      string `json:"key"`
}
type consumeInput struct {
	Original  string    `json:"original"`
	Key       string    `json:"key"`
	Assertion Assertion `json:"assertion"`
}
type identityKey struct{}

func TestS5A3SignedActualSDKSimulation(t *testing.T) {
	f := newFixture(t)
	server := mcp.NewServer(&mcp.Implementation{Name: "synthetic-signed-lab", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "lab_issue"}, func(ctx context.Context, _ *mcp.CallToolRequest, in issueInput) (*mcp.CallToolResult, Challenge, error) {
		who, _ := ctx.Value(identityKey{}).(string)
		c, e := f.l.issue(ctx, layer, who, in.Original, in.Key)
		return nil, c, e
	})
	mcp.AddTool(server, &mcp.Tool{Name: "lab_consume"}, func(ctx context.Context, _ *mcp.CallToolRequest, in consumeInput) (*mcp.CallToolResult, Receipt, error) {
		who, _ := ctx.Value(identityKey{}).(string)
		r, e := f.l.consume(ctx, layer, who, in.Original, in.Key, in.Assertion)
		return nil, r, e
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true, PropagateRequestCancellation: false})
	// Fixtures only: real SDK on real loopback HTTP transport, no live
	// credentials or production routes, no signer reachable from the handler.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var who string
		switch r.Header.Get("Authorization") {
		case "Bearer lab-only-token":
			who = actor
		case "Bearer other-lab-token":
			who = "other-actor"
		default:
			http.Error(w, "unauthorized", 401)
			return
		}
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey{}, who)))
	}))
	defer ts.Close()
	connect := func(tok string) *mcp.ClientSession {
		cl := mcp.NewClient(&mcp.Implementation{Name: "lab-client", Version: "1"}, nil)
		cs, e := cl.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: ts.URL, DisableStandaloneSSE: true, MaxRetries: -1, HTTPClient: &http.Client{Transport: labTokenTransport{tok}}}, nil)
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { cs.Close() })
		return cs
	}
	cs := connect("lab-only-token")
	other := connect("other-lab-token")
	ctx := context.Background()
	call := func(s *mcp.ClientSession, tool string, args map[string]any, want bool) *mcp.CallToolResult {
		t.Helper()
		r, e := s.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: args})
		ok := e == nil && r != nil && !r.IsError
		if ok != want {
			t.Fatalf("%s expected %t: %v", tool, want, e)
		}
		return r
	}
	res := call(cs, "lab_issue", map[string]any{"original": original, "key": key}, true)
	raw, e := json.Marshal(res.StructuredContent)
	if e != nil {
		t.Fatal(e)
	}
	var c Challenge
	if e = json.Unmarshal(raw, &c); e != nil || c.ID == "" {
		t.Fatalf("challenge: %v", e)
	}
	// An agent who knows the full challenge cannot approve without the
	// separate signer's private key. Signing below is a test human surrogate.
	args := map[string]any{"original": original, "key": key, "assertion": Assertion{Challenge: c, Decision: "approve"}}
	call(cs, "lab_consume", args, false)
	a := f.approve(c)
	args["assertion"] = a
	call(other, "lab_consume", args, false)
	f.assertState(t, 0, 0, 0)
	res = call(cs, "lab_consume", args, true)
	raw, e = json.Marshal(res.StructuredContent)
	if e != nil {
		t.Fatal(e)
	}
	var r Receipt
	if e = json.Unmarshal(raw, &r); e != nil || !r.ClaimReleased || r.NativeEffect || r.OriginalState != "UNKNOWN" {
		t.Fatalf("result %s %v", raw, e)
	}
	res = call(cs, "lab_consume", args, true)
	raw, e = json.Marshal(res.StructuredContent)
	if e != nil {
		t.Fatal(e)
	}
	var replay Receipt
	if e = json.Unmarshal(raw, &replay); e != nil || !replay.Duplicate || replay.ID != r.ID {
		t.Fatalf("replay %s %v", raw, e)
	}
	f.assertState(t, 1, 1, 1)
	// No raw approval signature or private key material stored in the ledger.
	var stored string
	if e = f.l.db.QueryRow("SELECT assertion_digest FROM lab_receipts WHERE id=?", r.ID).Scan(&stored); e != nil || len(stored) != 64 || strings.Contains(stored, a.Signature) {
		t.Fatal("unsafe stored signature")
	}
}

func TestS5A3SignedPendingRestartAndAdmission(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	c := f.issue(t, key)
	if _, e := f.l.issue(ctx, layer, actor, "other-original", key); !errors.Is(e, ErrDenied) {
		t.Fatal("key reused across original")
	}
	if _, e := f.l.issue(ctx, layer, "unknown-actor", original, key2); !errors.Is(e, ErrDenied) {
		t.Fatal("unknown actor issued")
	}
	if _, e := f.l.issue(ctx, layer, actor, "missing-original", key2); !errors.Is(e, ErrDenied) {
		t.Fatal("unknown target issued")
	}
	f.exec(t, "UPDATE lab_targets SET action='NETWORK_REPAIR' WHERE layer=? AND original=?", layer, "other-original")
	if _, e := f.l.issue(ctx, layer, actor, "other-original", key2); !errors.Is(e, ErrDenied) {
		t.Fatal("unsupported action issued")
	}
	f.assertState(t, 0, 0, 0)
	if e := f.l.db.Close(); e != nil {
		t.Fatal(e)
	}
	var e error
	f.l, e = openLab(f.path, f.v, func() time.Time { return f.now })
	if e != nil {
		t.Fatal(e)
	}
	if again := f.issue(t, key); again != c {
		t.Fatal("pending challenge lost on reopen")
	}
	a := f.approve(c)
	if _, e = f.consume(a); e != nil {
		t.Fatal(e)
	}
	f.exec(t, "UPDATE lab_actors SET active=0 WHERE layer=? AND actor=?", layer, actor)
	if _, e = f.consume(a); !errors.Is(e, ErrDenied) {
		t.Fatal("revoked requester read replay")
	}
	f.assertState(t, 1, 1, 1)
}
