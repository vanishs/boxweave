package control

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

// Seed a source-only v2 lifecycle history through the actual SQLite
// admission, reservation, ACK and finalization APIs. These synthetic native
// IDs are fixture data, never proof of Incus or Root execution.
func p1bSeedScopedCreate(t *testing.T, ctx context.Context, st *state.Store, tok state.Token, project, bridge, name, uuid, mac, v4, v6, key string) state.Receipt {
	t.Helper()
	intent := state.ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: name, Start: false, Internet: true, BootAutostart: true, CPU: 1, MemoryMiB: 1024, DiskGiB: 10, Options: json.RawMessage("{}")}
	raw, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	r, err := st.AdmitScoped(ctx, state.Admission{Layer: project, ActorID: tok.ActorID, TokenID: tok.ID, Key: key, RequestID: "source-fixture", Action: "box_create", ResourceKey: "box:" + name, SafeIntent: raw})
	if err != nil {
		t.Fatal(err)
	}
	if err = st.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	pin := state.VerifiedCreateBinding{Bridge: bridge, ACL: peerscope.QuarantineACL, MAC: mac, IPv4: v4, IPv6: v6}
	if ok, e := st.StageCreateBinding(ctx, r.ID, project, name, pin); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if err = st.NativeAck(ctx, r.ID, "fixture-native-"+name); err != nil {
		t.Fatal(err)
	}
	pin.ACL, err = peerscope.ACLName(project, bridge, uuid)
	if err != nil {
		t.Fatal(err)
	}
	if err = st.FinalizeVerifiedScopedCreate(ctx, r.ID, project, name, uuid, pin); err != nil {
		t.Fatal(err)
	}
	r, err = st.Get(ctx, r.ID)
	if err != nil || r.State != "SUCCEEDED" || r.BoxUUID != uuid {
		t.Fatal(r, err)
	}
	return r
}

func TestP1BScopedMCPReplayWithoutNativeDispatch(t *testing.T) {
	st, ts, raw, other := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	tok, err := st.VerifyToken(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	const bridge = "incusbr-1001"
	const src = "00000000-0000-4000-8000-000000000001"
	const dst = "00000000-0000-4000-8000-000000000002"
	const createKey = "p1b-scoped-create-alpha"
	const peerKey = "p1b-scoped-peer-original"
	created := p1bSeedScopedCreate(t, ctx, st, tok, testScopedProject, bridge, "alpha", src, "02:16:3e:00:00:11", "10.243.18.11", "fd43:18::11", createKey)
	p1bSeedScopedCreate(t, ctx, st, tok, testScopedProject, bridge, "beta", dst, "02:16:3e:00:00:12", "10.243.18.12", "fd43:18::12", "p1b-scoped-create-beta")
	ledger, e := st.ReadCompleteScopedNetworkAuditLedger(ctx, testScopedProject, bridge)
	if e != nil || len(ledger.Boxes) != 2 {
		t.Fatal(ledger, e)
	}
	peer, e := st.AdmitScopedPeerSet(ctx, state.Admission{Layer: testScopedProject, ActorID: tok.ActorID, TokenID: tok.ID, Key: peerKey, RequestID: "source-fixture"}, src, dst, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = st.Claim(ctx, peer.ID); e != nil {
		t.Fatal(e)
	}
	if e = st.Unknown(ctx, peer.ID, "DISPATCHING", "FIXTURE_ACK_UNPROVED"); e != nil {
		t.Fatal(e)
	}
	connect := func(token string) *mcp.ClientSession {
		t.Helper()
		c, e := activityMCPClient(ctx, ts.URL, &http.Client{Transport: bearerTransport{base: http.DefaultTransport, token: token}})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { c.Close() })
		return c
	}
	c := connect(raw)
	call := func(c *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
		t.Helper()
		r, e := c.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if e != nil || r == nil {
			t.Fatal(name, e)
		}
		return r
	}
	status := call(c, "host_status", map[string]any{})
	if status.IsError {
		t.Fatal(status)
	}
	var hs HostStatus
	b, e := json.Marshal(status.StructuredContent)
	if e != nil || json.Unmarshal(b, &hs) != nil || hs.PeerMode != "SCOPED_V2" || !hs.MCPReady || hs.Ready || hs.Native.ServiceReady || hs.AddressRetirementPolicy != "SCOPED_RETAIN_IDENTITIES" || hs.AddressRetirement != nil {
		t.Fatal(string(b), e)
	}
	assertReplay := func(name string, args map[string]any, original state.Receipt, want string) {
		t.Helper()
		r := call(c, name, args)
		if r.IsError {
			t.Fatal(name, r.Content)
		}
		b, e := json.Marshal(r.StructuredContent)
		var got state.Receipt
		if e != nil || json.Unmarshal(b, &got) != nil || got.ID != original.ID || got.TrackingID != original.TrackingID || !got.Duplicate || got.State != want || got.NativeOperationID != original.NativeOperationID {
			t.Fatal(name, string(b), e)
		}
	}
	ca := map[string]any{"name": "alpha", "start": false, "idempotency_key": createKey}
	pa := map[string]any{"source_box_id": src, "target_box_id": dst, "allow": true, "idempotency_key": peerKey}
	assertReplay("box_create", ca, created, "SUCCEEDED")
	assertReplay("box_peer_set", pa, peer, "UNKNOWN")
	ca["boot.autostart"] = false
	if f := h4aFailure(t, call(c, "box_create", ca), nil); f.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(f)
	}
	pa["allow"] = false
	if f := h4aFailure(t, call(c, "box_peer_set", pa), nil); f.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatal(f)
	}
	pa["allow"] = true
	if r := call(connect(other), "box_peer_set", pa); !r.IsError {
		t.Fatal("other actor replayed original", r)
	}
	pa["idempotency_key"] = "p1b-scoped-peer-new-key"
	if r := call(c, "box_peer_set", pa); !r.IsError {
		t.Fatal("new key bypassed absent native witness", r)
	}
	got, e := st.Get(ctx, peer.ID)
	if e != nil || got.State != "UNKNOWN" || got.NativeOperationID != "" || got.ErrorClass != "FIXTURE_ACK_UNPROVED" {
		t.Fatal(got, e)
	}
	page, e := st.ListOperations(ctx, state.OperationQuery{Layer: testScopedProject, Requester: tok.ActorID})
	if e != nil || len(page.Operations) != 3 {
		t.Fatal(page, e)
	}
	// A foreign UUID and a missing UUID yield the same scoped Peer refusal,
	// without disclosing ownership or admitting an operation.
	const foreign = "00000000-0000-4000-8000-000000000003"
	p1bSeedScopedCreate(t, ctx, st, tok, "user-1002", "incusbr-1002", "foreign", foreign, "02:16:3e:00:00:13", "10.244.18.13", "fd44:18::13", "p1b-foreign-create-fixture")
	pa["source_box_id"] = foreign
	f1 := h4aFailure(t, call(c, "box_peer_set", pa), nil)
	pa["source_box_id"] = "00000000-0000-4000-8000-000000000004"
	f2 := h4aFailure(t, call(c, "box_peer_set", pa), nil)
	if f1 != f2 || f1.Code != "PEER_SCOPE_UNVERIFIED_OR_LOCKED" || f1.Next.OperationID != "" {
		t.Fatal(f1, f2)
	}
	page, e = st.ListOperations(ctx, state.OperationQuery{Layer: testScopedProject, Requester: tok.ActorID})
	if e != nil || len(page.Operations) != 3 {
		t.Fatal(page, e)
	}
	t.Log("P1B_SCOPED_MCP_REPLAY_AND_REFUSAL=PASS (source/SQLite only)")
}
