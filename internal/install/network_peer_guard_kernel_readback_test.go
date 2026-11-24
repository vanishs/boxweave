package install

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func pr178SyntheticNFTReadback(t *testing.T, plan NativePeerGuardPlan) []map[string]any {
	t.Helper()
	var exprs [][]any
	if err := json.Unmarshal([]byte(plan.ExpectedKernelJSON), &exprs); err != nil {
		t.Fatal(err)
	}
	content := []map[string]any{
		{"metainfo": map[string]any{"version": "1.0.9", "release_name": "test-nft", "json_schema_version": float64(1)}},
		{"table": map[string]any{"family": "bridge", "name": plan.Table, "handle": float64(777)}},
		{"chain": map[string]any{
			"family": "bridge", "table": plan.Table, "name": "protect", "handle": float64(1),
			"type": "filter", "hook": "forward", "prio": float64(-250), "policy": "accept",
		}},
	}
	for i, expr := range exprs {
		content = append(content, map[string]any{
			"rule": map[string]any{
				"family": "bridge", "table": plan.Table, "chain": "protect",
				"handle": float64(i + 2), "expr": expr,
			},
		})
	}
	return content
}
func pr178NativeEnvelope(t *testing.T, items []map[string]any) []byte {
	t.Helper()
	b, err := json.Marshal(map[string]any{"nftables": items})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func pr178ReadbackPlan(t *testing.T) NativePeerGuardPlan {
	t.Helper()
	cd := pr178Grant(pr178C, pr178D, "10.55.8.13", "10.55.8.14", "fd55:8::13", "fd55:8::14")
	ab := pr178Grant(pr178A, pr178B, "10.55.8.11", "10.55.8.12", "fd55:8::11", "fd55:8::12")
	return pr178Guard(t, pr178GuardMembers(), []state.PeerGrantBinding{ab, cd}, nil)
}
func TestPR178StrictNFTJSONKernelReadbackExactDualStackAndCounters(t *testing.T) {
	p := pr178ReadbackPlan(t)
	items := pr178SyntheticNFTReadback(t, p)
	raw := pr178NativeEnvelope(t, items)
	if err := NativePeerGuardKernelVerifyJSON(p, raw); err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		rule, ok := item["rule"].(map[string]any)
		if !ok {
			continue
		}
		expr := rule["expr"].([]any)
		for _, entry := range expr {
			e := entry.(map[string]any)
			if counter, yes := e["counter"]; yes {
				c := counter.(map[string]any)
				c["packets"] = float64(235)
				c["bytes"] = float64(152480)
			}
		}
	}
	if err := NativePeerGuardKernelVerifyJSON(p, pr178NativeEnvelope(t, items)); err != nil {
		t.Fatal("in-flight counter changes alone must not invalidate policy:", err)
	}
	if strings.Contains(p.Install, "flush ruleset") || p.DigestSHA256 != nativePeerGuardDigest(p) {
		t.Fatal("kernel policy compiler lost immutable digest")
	}
}
func TestPR178StrictNFTJSONRejectsShadowAclAndNativeDrift(t *testing.T) {
	p := pr178ReadbackPlan(t)
	cases := []struct {
		name   string
		change func([]map[string]any) []map[string]any
	}{
		{"missing-rule", func(a []map[string]any) []map[string]any { return a[:len(a)-1] }},
		{"extra-shadow-allow", func(a []map[string]any) []map[string]any {
			a = append(a, map[string]any{"rule": map[string]any{
				"family": "bridge", "table": p.Table, "chain": "protect", "handle": float64(600),
				"expr": []any{map[string]any{"accept": nil}},
			}})
			return a
		}},
		{"wrong-base-priority", func(a []map[string]any) []map[string]any {
			a[2]["chain"].(map[string]any)["prio"] = float64(-200)
			return a
		}},
		{"wrong-hook", func(a []map[string]any) []map[string]any {
			a[2]["chain"].(map[string]any)["hook"] = "input"
			return a
		}},
		{"wrong-policy", func(a []map[string]any) []map[string]any {
			a[2]["chain"].(map[string]any)["policy"] = "drop"
			return a
		}},
		{"wrong-bridge", func(a []map[string]any) []map[string]any {
			e := a[3]["rule"].(map[string]any)["expr"].([]any)
			e[0].(map[string]any)["match"].(map[string]any)["right"] = "incusbr-FOREIGN"
			return a
		}},
		{"wrong-source-mac", func(a []map[string]any) []map[string]any {
			e := a[3]["rule"].(map[string]any)["expr"].([]any)
			e[2].(map[string]any)["match"].(map[string]any)["right"] = "10:66:6a:00:00:00"
			return a
		}},
		{"wrong-dualstack-destination", func(a []map[string]any) []map[string]any {
			e := a[5]["rule"].(map[string]any)["expr"].([]any)
			e[4].(map[string]any)["match"].(map[string]any)["right"] = "fd55:8::99"
			return a
		}},
		{"reverse-unconditional-allow", func(a []map[string]any) []map[string]any {
			e := a[4]["rule"].(map[string]any)["expr"].([]any)
			a[4]["rule"].(map[string]any)["expr"] = append(e[:5], e[len(e)-1])
			return a
		}},
		{"drop-replaced-allow", func(a []map[string]any) []map[string]any {
			rule := a[len(a)-1]["rule"].(map[string]any)
			e := rule["expr"].([]any)
			e[len(e)-1] = map[string]any{"accept": nil}
			return a
		}},
		{"unknown-counter-key", func(a []map[string]any) []map[string]any {
			rule := a[len(a)-1]["rule"].(map[string]any)
			e := rule["expr"].([]any)
			e[len(e)-2].(map[string]any)["counter"].(map[string]any)["unexpected"] = float64(1)
			return a
		}},
		{"negative-counter", func(a []map[string]any) []map[string]any {
			rule := a[len(a)-1]["rule"].(map[string]any)
			e := rule["expr"].([]any)
			e[len(e)-2].(map[string]any)["counter"].(map[string]any)["packets"] = float64(-1)
			return a
		}},
		{"duplicated-rule-handle", func(a []map[string]any) []map[string]any {
			a[4]["rule"].(map[string]any)["handle"] = float64(2)
			return a
		}},
		{"foreign-rule-chain", func(a []map[string]any) []map[string]any {
			a[3]["rule"].(map[string]any)["chain"] = "evil"
			return a
		}},
		{"foreign-table", func(a []map[string]any) []map[string]any {
			a[1]["table"].(map[string]any)["name"] = "bw_pg_deadbeefdeadbeef"
			return a
		}},
		{"unknown-object", func(a []map[string]any) []map[string]any {
			return append(a, map[string]any{"set": map[string]any{"name": "bad"}})
		}},
		{"wrong-json-schema", func(a []map[string]any) []map[string]any {
			a[0]["metainfo"].(map[string]any)["json_schema_version"] = float64(2)
			return a
		}},
		{"unknown-extra-attribute", func(a []map[string]any) []map[string]any {
			a[3]["rule"].(map[string]any)["comment"] = "spoof"
			return a
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := pr178SyntheticNFTReadback(t, p)
			items = tc.change(items)
			if err := NativePeerGuardKernelVerifyJSON(p, pr178NativeEnvelope(t, items)); err == nil {
				t.Fatal("unsafe native nft readback accepted")
			}
		})
	}
	corrupted := p
	corrupted.DigestSHA256 = strings.Repeat("0", 64)
	if err := NativePeerGuardKernelVerifyJSON(corrupted, pr178NativeEnvelope(t, pr178SyntheticNFTReadback(t, p))); err == nil {
		t.Fatal("forged digest accepted")
	}
	corrupted = p
	corrupted.ExpectedKernelJSON = "[]"
	if err := NativePeerGuardKernelVerifyJSON(corrupted, pr178NativeEnvelope(t, pr178SyntheticNFTReadback(t, p))); err == nil {
		t.Fatal("forged expectation accepted")
	}
	for _, bad := range [][]byte{nil, []byte("not-json"), []byte("{}"), []byte("[]"), []byte(strings.Repeat("x", 1<<20+1))} {
		if err := NativePeerGuardKernelVerifyJSON(p, bad); err == nil {
			t.Fatal("untrusted or unbounded kernel output accepted")
		}
	}
	if _, err := nativePeerGuardNFTRead(context.Background(), "../../shell"); err == nil {
		t.Fatal("nft reader accepted arbitrary path/table injection")
	}
}
