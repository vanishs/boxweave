package install

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

// Real schema-7 transitions; native witnesses are synthetic test inputs only.
func scopedInstallStore(t *testing.T, fixture state.ScopedPeerOriginalProof) (*state.Store, *state.Store, *sql.DB, string) {
	t.Helper()
	ctx := t.Context()
	dir := t.TempDir()
	if e := os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(dir, "audit.sqlite")
	s, e := state.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	db, e := sql.Open("sqlite3", path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	project, bridge := fixture.Intent.Project, fixture.Intent.Bridge
	for _, b := range fixture.Inventory.Boxes {
		raw, e := json.Marshal(state.ScopedCreateIntent{Schema: 2, Project: project, Bridge: bridge, Name: b.Name, CPU: 1, MemoryMiB: 256, DiskGiB: 2, Options: json.RawMessage("{}")})
		if e != nil {
			t.Fatal(e)
		}
		r, e := s.AdmitScoped(ctx, state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "install-fixture-create-" + b.Name, RequestID: b.Name, Action: "box_create", ResourceKey: "box:" + b.Name, SafeIntent: raw})
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Claim(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		pin := state.VerifiedCreateBinding{Bridge: bridge, ACL: peerscope.QuarantineACL, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}
		if ok, e := s.StageCreateBinding(ctx, r.ID, project, b.Name, pin); e != nil || !ok {
			t.Fatal(ok, e)
		}
		if e = s.NativeAck(ctx, r.ID, "fixture-create-"+b.Name); e != nil {
			t.Fatal(e)
		}
		pin.ACL = b.ACL
		if e = s.FinalizeVerifiedScopedCreate(ctx, r.ID, project, b.Name, b.UUID, pin); e != nil {
			t.Fatal(e)
		}
	}
	root, e := state.OpenNetworkAuditReadOnly(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { root.Close() })
	admit := func(key, src, dst string, allow bool) string {
		r, e := s.AdmitScopedPeerSet(ctx, state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: key, RequestID: key}, src, dst, allow)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		return r.ID
	}
	for _, g := range fixture.Before {
		id := admit("fixture-grant-"+g.SourceUUID+"-"+g.TargetUUID, g.SourceUUID, g.TargetUUID, true)
		p, e := root.ReadRootScopedPeerOriginal(ctx, id, project, bridge)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeScopedPeerOriginal(ctx, state.ScopedPeerRootReceipt{OriginalID: id, Project: project, Bridge: bridge, Status: "COMPLETE_VERIFIED", ProofSHA256: state.ScopedPeerProofSHA256(p), PlanSHA256: strings.Repeat("a", 64), FinalSHA256: strings.Repeat("b", 64), VerifiedSteps: 4}); e != nil {
			t.Fatal(e)
		}
	}
	i := fixture.Intent
	id := admit("fixture-current-original", i.SourceUUID, i.TargetUUID, i.Allow)
	return s, root, db, id
}
