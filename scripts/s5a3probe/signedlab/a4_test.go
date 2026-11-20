package signedlab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func a4Seed(t *testing.T, f *fixture, kind string) a4Plan {
	t.Helper()
	p := a4Plan{Kind: kind, Original: original, UUID: "uuid-1", Incarnation: "incarnation-1", Resource: "exact-owned-fixture", Owned: true, ClaimPinned: true, NoConflict: true, Contained: true}
	switch kind {
	case a4Forward:
		p.Endpoint = "fixture-loopback-listener"
	case a4Peer:
		p.Peer = "peer-uuid"
		p.FreshRootPlan = true
	case a4Child:
		p.Parent = "unknown-delete-parent"
		p.ChildLinked = true
		p.ParentBarrier = true
	}
	a4Store(t, f, p)
	return p
}
func a4Store(t *testing.T, f *fixture, p a4Plan) {
	t.Helper()
	f.exec(t, "INSERT OR REPLACE INTO lab_a4_plan VALUES(?,?,?)", layer, original, encode(p))
	f.exec(t, "UPDATE lab_targets SET action=?,intent=?,evidence=? WHERE layer=? AND original=?", p.Kind, hexDigest(encode(struct{ Action, Resource string }{p.Kind, p.Resource})), hexDigest(encode(p)), layer, original)
}
func a4Post(p a4Plan, id string) a4Witness {
	return a4Witness{CompID: id, Original: p.Original, UUID: p.UUID, Incarnation: p.Incarnation, Kind: p.Kind, Resource: p.Resource, Endpoint: p.Endpoint, Peer: p.Peer, Parent: p.Parent, ExactIdentity: true, CompensationReceiptVerified: true, PostVerified: true, NoUnrelatedEffect: true, BarrierIntact: true, DeviceAbsent: true, ListenerAbsent: true, RootKernelDeny: true, ChildSafe: true}
}
func a4WitnessPut(t *testing.T, f *fixture, w a4Witness) {
	t.Helper()
	f.exec(t, "INSERT OR REPLACE INTO lab_a4_witness VALUES(?,?)", w.CompID, encode(w))
}
func a4Check(t *testing.T, f *fixture, id, state string, dispatches, released int) {
	t.Helper()
	var got string
	var count, rel int
	if e := f.l.db.QueryRow("SELECT state,dispatches FROM lab_a4_comp WHERE id=?", id).Scan(&got, &count); e != nil || got != state || count != dispatches {
		t.Fatalf("comp state %s/%d %v", got, count, e)
	}
	if e := f.l.db.QueryRow("SELECT state,released FROM lab_targets WHERE layer=? AND original=?", layer, original).Scan(&got, &rel); e != nil || got != "UNKNOWN" || rel != released {
		t.Fatalf("original state %s/%d %v", got, rel, e)
	}
}
func a4Admit(t *testing.T, f *fixture) (Assertion, Receipt) {
	t.Helper()
	a := f.approve(f.issue(t, key))
	r, e := f.consume(a)
	if e != nil || r.CompensationID == "" || r.CompensationID == r.ID || r.CompensationID == original || r.ClaimReleased || r.NativeEffect || r.OriginalState != "UNKNOWN" {
		t.Fatalf("unsafe approval %+v %v", r, e)
	}
	a4Check(t, f, r.CompensationID, "ADMITTED", 0, 0)
	return a, r
}
func TestS5A4SignedCompensationDomains(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{a4Forward, a4Peer, a4Child} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			p := a4Seed(t, f, kind)
			a, r := a4Admit(t, f)
			id := r.CompensationID
			// No evidence means no settlement even with a signed approval.
			if e := f.l.settleA4(ctx, id); !errors.Is(e, ErrDenied) {
				t.Fatal("premature settlement", e)
			}
			if e := f.l.dispatchA4(ctx, id); e != nil {
				t.Fatal(e)
			}
			a4Check(t, f, id, "UNKNOWN", 1, 0)
			if e := f.l.dispatchA4(ctx, id); !errors.Is(e, ErrDenied) {
				t.Fatal("redispatch", e)
			}
			if e := f.l.settleA4(ctx, id); !errors.Is(e, ErrDenied) {
				t.Fatal("missing witness", e)
			}
			// Reopen durable state after lost response.
			if e := f.l.db.Close(); e != nil {
				t.Fatal(e)
			}
			var e error
			f.l, e = openLab(f.path, f.v, func() time.Time { return f.now })
			if e != nil {
				t.Fatal(e)
			}
			replay, e := f.consume(a)
			if e != nil || !replay.Duplicate || replay.ID != r.ID || replay.CompensationID != id {
				t.Fatal("replay lost receipt", e)
			}
			if e = f.l.dispatchA4(ctx, id); !errors.Is(e, ErrDenied) {
				t.Fatal("redispatch after restart", e)
			}
			w := a4Post(p, id)
			w.PostVerified = false
			a4WitnessPut(t, f, w)
			if e = f.l.settleA4(ctx, id); !errors.Is(e, ErrDenied) {
				t.Fatal("unverified poststate", e)
			}
			a4Check(t, f, id, "UNKNOWN", 1, 0)
			w.PostVerified = true
			a4WitnessPut(t, f, w)
			if e = f.l.settleA4(ctx, id); e != nil {
				t.Fatal(e)
			}
			if e = f.l.settleA4(ctx, id); e != nil {
				t.Fatal(e)
			}
			released := 0
			if kind == a4Forward {
				released = 1
			}
			a4Check(t, f, id, "SUCCEEDED", 1, released)
			var n int
			if e = f.l.db.QueryRow("SELECT count(*) FROM lab_a4_events WHERE comp=?", id).Scan(&n); e != nil || n != 3 {
				t.Fatal("audit missing", e, n)
			}
			t.Logf("S5A4_SIMULATED_%s=PASS", kind)
		})
	}
}
func TestS5A4SignedCompensationRefusals(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{a4Forward, a4Peer, a4Child} {
		for _, fault := range []string{"unsigned", "forged", "expired", "denied", "stale-evidence", "revoked", "unowned", "unpinned", "conflict", "uncontained", "dispatch-drift"} {
			t.Run(kind+"/"+fault, func(t *testing.T) {
				f := newFixture(t)
				p := a4Seed(t, f, kind)
				switch fault {
				case "unowned":
					p.Owned = false
				case "unpinned":
					p.ClaimPinned = false
				case "conflict":
					p.NoConflict = false
				case "uncontained":
					p.Contained = false
				}
				a4Store(t, f, p)
				if fault == "unowned" || fault == "unpinned" || fault == "conflict" || fault == "uncontained" {
					if _, e := f.l.issue(ctx, layer, actor, original, key); !errors.Is(e, ErrDenied) {
						t.Fatal("unsafe challenge", e)
					}
					f.assertState(t, 0, 0, 0)
					return
				}
				c := f.issue(t, key)
				a := f.approve(c)
				switch fault {
				case "unsigned":
					a.Signature = ""
				case "forged":
					_, priv, e := ed25519.GenerateKey(rand.Reader)
					if e != nil {
						t.Fatal(e)
					}
					a = sign(priv, c, "approve")
				case "expired":
					f.now = f.now.Add(ttl)
				case "denied":
					a = sign(f.priv, c, "deny")
				case "stale-evidence":
					p.Resource = "changed-fixture"
					a4Store(t, f, p)
				case "revoked":
					f.exec(t, "UPDATE lab_actors SET active=0 WHERE layer=? AND actor=?", layer, actor)
				}
				r, e := f.consume(a)
				if fault == "denied" {
					if e != nil || r.ClaimReleased || r.CompensationID != "" {
						t.Fatal("denial admitted compensation", e)
					}
					f.assertState(t, 0, 1, 1)
				} else if fault == "dispatch-drift" {
					if e != nil || r.CompensationID == "" {
						t.Fatal(e)
					}
					p.NoConflict = false
					a4Store(t, f, p)
					if e = f.l.dispatchA4(ctx, r.CompensationID); !errors.Is(e, ErrDenied) {
						t.Fatal("stale dispatch", e)
					}
					a4Check(t, f, r.CompensationID, "ADMITTED", 0, 0)
					return
				} else {
					if !errors.Is(e, ErrDenied) || r.CompensationID != "" {
						t.Fatal("invalid signature accepted", e)
					}
					f.assertState(t, 0, 0, 0)
				}
				var n int
				if e = f.l.db.QueryRow("SELECT count(*) FROM lab_a4_comp").Scan(&n); e != nil || n != 0 {
					t.Fatal("unexpected comp", e, n)
				}
			})
		}
	}
	for _, tc := range []struct {
		kind   string
		mutate func(*a4Plan)
	}{
		{a4Peer, func(p *a4Plan) { p.FreshRootPlan = false }},
		{a4Child, func(p *a4Plan) { p.ParentBarrier = false }},
		{a4Child, func(p *a4Plan) { p.ChildLinked = false }},
		{a4Forward, func(p *a4Plan) { p.Endpoint = "" }},
	} {
		f := newFixture(t)
		p := a4Seed(t, f, tc.kind)
		tc.mutate(&p)
		a4Store(t, f, p)
		if _, e := f.l.issue(ctx, layer, actor, original, key); !errors.Is(e, ErrDenied) {
			t.Fatal("domain barrier bypass", tc.kind, e)
		}
	}
}
func TestS5A4SignedCompensationAtomicAndRace(t *testing.T) {
	ctx := context.Background()
	f := newFixture(t)
	p := a4Seed(t, f, a4Forward)
	a := f.approve(f.issue(t, key))
	a2 := f.approve(f.issue(t, key2))
	f.exec(t, "CREATE TRIGGER a4_receipt_cut BEFORE INSERT ON lab_receipts BEGIN SELECT RAISE(ABORT,'fixture cut'); END")
	if _, e := f.consume(a); e == nil {
		t.Fatal("admission escaped rollback")
	}
	f.assertState(t, 0, 0, 0)
	var n int
	if e := f.l.db.QueryRow("SELECT count(*) FROM lab_a4_comp").Scan(&n); e != nil || n != 0 {
		t.Fatal("comp escaped rollback", e)
	}
	f.exec(t, "DROP TRIGGER a4_receipt_cut")
	other, e := openLab(f.path, f.v, func() time.Time { return f.now })
	if e != nil {
		t.Fatal(e)
	}
	defer other.db.Close()
	type out struct {
		r Receipt
		e error
	}
	ch := make(chan out, 2)
	var wg sync.WaitGroup
	for _, x := range []struct {
		l *Lab
		k string
		a Assertion
	}{{f.l, key, a}, {other, key2, a2}} {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := x.l.consume(ctx, layer, actor, original, x.k, x.a); ch <- out{r, e} }()
	}
	wg.Wait()
	close(ch)
	id := ""
	denied := 0
	for x := range ch {
		if x.e == nil && id == "" {
			id = x.r.CompensationID
		} else if errors.Is(x.e, ErrDenied) {
			denied++
		} else {
			t.Fatal("duplicate admission", x.e)
		}
	}
	if id == "" || denied != 1 {
		t.Fatal("race failed")
	}
	if e = f.l.dispatchA4(ctx, id); e != nil {
		t.Fatal(e)
	}
	a4WitnessPut(t, f, a4Post(p, id))
	f.exec(t, "CREATE TRIGGER a4_verify_cut BEFORE INSERT ON lab_a4_events WHEN NEW.event='VERIFIED' BEGIN SELECT RAISE(ABORT,'fixture cut'); END")
	if e = f.l.settleA4(ctx, id); e == nil {
		t.Fatal("settlement escaped rollback")
	}
	a4Check(t, f, id, "UNKNOWN", 1, 0)
	f.exec(t, "DROP TRIGGER a4_verify_cut")
	if e = f.l.settleA4(ctx, id); e != nil {
		t.Fatal(e)
	}
	a4Check(t, f, id, "SUCCEEDED", 1, 1)
}
func TestS5A4SignedMCPFixture(t *testing.T) {
	f := newFixture(t)
	p := a4Seed(t, f, a4Peer)
	ctx := context.Background()
	server := mcp.NewServer(&mcp.Implementation{Name: "synthetic-a4", Version: "1"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "lab_issue"}, func(ctx context.Context, _ *mcp.CallToolRequest, in issueInput) (*mcp.CallToolResult, Challenge, error) {
		c, e := f.l.issue(ctx, layer, actor, in.Original, in.Key)
		return nil, c, e
	})
	mcp.AddTool(server, &mcp.Tool{Name: "lab_consume"}, func(ctx context.Context, _ *mcp.CallToolRequest, in consumeInput) (*mcp.CallToolResult, Receipt, error) {
		r, e := f.l.consume(ctx, layer, actor, in.Original, in.Key, in.Assertion)
		return nil, r, e
	})
	h := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer lab-only-token" {
			http.Error(w, "unauthorized", 401)
			return
		}
		h.ServeHTTP(w, r)
	}))
	defer ts.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "a4-lab-agent", Version: "1"}, nil)
	cs, e := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: ts.URL, DisableStandaloneSSE: true, MaxRetries: -1, HTTPClient: &http.Client{Transport: labTokenTransport{"lab-only-token"}}}, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer cs.Close()
	call := func(name string, args map[string]any, want bool) []byte {
		t.Helper()
		r, e := cs.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		ok := e == nil && r != nil && !r.IsError
		if ok != want {
			t.Fatalf("call %s want %v: %v", name, want, e)
		}
		if !ok {
			return nil
		}
		raw, e := json.Marshal(r.StructuredContent)
		if e != nil {
			t.Fatal(e)
		}
		return raw
	}
	var c Challenge
	if e = json.Unmarshal(call("lab_issue", map[string]any{"original": original, "key": key}, true), &c); e != nil {
		t.Fatal(e)
	}
	args := map[string]any{"original": original, "key": key, "assertion": Assertion{Challenge: c, Decision: "approve"}}
	call("lab_consume", args, false)
	args["assertion"] = f.approve(c)
	var r Receipt
	if e = json.Unmarshal(call("lab_consume", args, true), &r); e != nil || r.CompensationID == "" || r.ClaimReleased {
		t.Fatal("signed admission", e)
	}
	if e = f.l.dispatchA4(ctx, r.CompensationID); e != nil {
		t.Fatal(e)
	}
	a4WitnessPut(t, f, a4Post(p, r.CompensationID))
	if e = f.l.settleA4(ctx, r.CompensationID); e != nil {
		t.Fatal(e)
	}
	a4Check(t, f, r.CompensationID, "SUCCEEDED", 1, 0)
	t.Log("S5A4_SIMULATED_MCP_APPROVAL_COMPENSATION=PASS")
}

func TestS5A4SignedCompensationIncompletePostProof(t *testing.T) {
	ctx := context.Background()
	for _, kind := range []string{a4Forward, a4Peer, a4Child} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t)
			p := a4Seed(t, f, kind)
			_, r := a4Admit(t, f)
			id := r.CompensationID
			if e := f.l.dispatchA4(ctx, id); e != nil {
				t.Fatal(e)
			}
			cases := []func(*a4Witness){
				func(w *a4Witness) { w.ExactIdentity = false },
				func(w *a4Witness) { w.CompensationReceiptVerified = false },
				func(w *a4Witness) { w.NoUnrelatedEffect = false },
				func(w *a4Witness) { w.BarrierIntact = false },
				func(w *a4Witness) { w.UUID = "different" },
				func(w *a4Witness) { w.Incarnation = "different" },
				func(w *a4Witness) { w.Resource = "different" },
				func(w *a4Witness) { w.Original = "different" },
			}
			switch kind {
			case a4Forward:
				cases = append(cases, func(w *a4Witness) { w.Endpoint = "different" }, func(w *a4Witness) { w.DeviceAbsent = false }, func(w *a4Witness) { w.ListenerAbsent = false })
			case a4Peer:
				cases = append(cases, func(w *a4Witness) { w.Peer = "different" }, func(w *a4Witness) { w.RootKernelDeny = false })
			case a4Child:
				cases = append(cases, func(w *a4Witness) { w.Parent = "different" }, func(w *a4Witness) { w.ChildSafe = false })
			}
			for _, mutate := range cases {
				w := a4Post(p, id)
				mutate(&w)
				a4WitnessPut(t, f, w)
				if e := f.l.settleA4(ctx, id); !errors.Is(e, ErrDenied) {
					t.Fatal("invalid post witness", e)
				}
				a4Check(t, f, id, "UNKNOWN", 1, 0)
			}
			a4WitnessPut(t, f, a4Post(p, id))
			if e := f.l.settleA4(ctx, id); e != nil {
				t.Fatal(e)
			}
			released := 0
			if kind == a4Forward {
				released = 1
			}
			a4Check(t, f, id, "SUCCEEDED", 1, released)
		})
	}
}
func TestS5A4NoGenericReleaseEvenWithSignature(t *testing.T) {
	for _, kind := range []string{a4Forward, a4Peer, a4Child} {
		f := newFixture(t)
		a4Seed(t, f, kind)
		f.exec(t, "UPDATE lab_targets SET action='REVIEWED_RELEASE' WHERE layer=? AND original=?", layer, original)
		if _, e := f.l.issue(context.Background(), layer, actor, original, key); !errors.Is(e, ErrDenied) {
			t.Fatal("generic release bypass", kind, e)
		}
		f.assertState(t, 0, 0, 0)
	}
}

func TestS5A4PostDispatchIntentDrift(t *testing.T) {
	f := newFixture(t)
	p := a4Seed(t, f, a4Forward)
	_, r := a4Admit(t, f)
	ctx := context.Background()
	if e := f.l.dispatchA4(ctx, r.CompensationID); e != nil {
		t.Fatal(e)
	}
	a4WitnessPut(t, f, a4Post(p, r.CompensationID))
	f.exec(t, "UPDATE lab_targets SET intent='different' WHERE layer=? AND original=?", layer, original)
	if e := f.l.settleA4(ctx, r.CompensationID); !errors.Is(e, ErrDenied) {
		t.Fatal("changed intent accepted", e)
	}
	a4Check(t, f, r.CompensationID, "UNKNOWN", 1, 0)
}
