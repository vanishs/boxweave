package state

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestH1AMaintenanceRetainedAndUnknown(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil || l.Retired != 1 || len(l.Boxes) != 2 || l.ForeignRows != 0 || l.OrphanBindings != 0 || l.OrphanFences != 0 {
		t.Fatal(l, e)
	}
	again, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil || l.ObservationDigest() != again.ObservationDigest() {
		t.Fatal("stationary digest", e)
	}
	s2, _, id := pr178ScopedReadyToStage(t, true)
	if e = s2.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e = s2.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); e != nil {
		t.Fatal(e)
	}
	snap2 := DiagnosticSnapshot{store: s2}
	u, e := snap2.ReadMaintenanceLedger(ctx, "user-1001")
	if e != nil || u.Claims != 8 || u.NonterminalRecordedClaims != 5 || len(u.Originals) != 1 {
		t.Fatal(u, e)
	}
	found := false
	for _, o := range u.Originals {
		if o.ID == id && o.State == "UNKNOWN" && o.Claims == 5 {
			found = true
		}
	}
	if !found {
		t.Fatal("original protected claims not found", u.Originals)
	}

}
func TestH1AMaintenanceForeignAndRedaction(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	foreign := activityAdmit(t, s, tok, "user-9999", "h1a-foreign-000000", "box_exec", strings.Repeat("a", 32), "box:foreign-name")
	own := activityAdmit(t, s, tok, "user-1001", "h1a-own-000000000", "box_exec", strings.Repeat("b", 32), "box:own")
	if e := s.Claim(ctx, own.ID); e != nil {
		t.Fatal(e)
	}
	if e := s.Unknown(ctx, own.ID, "DISPATCHING", "UNCERTAIN_NATIVE"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,1,'test','RELEASED','test')", own.ID); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("UPDATE oauth_config SET public_origin='https://private.example',password_hash='never-print-secret' WHERE id=1"); e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	l, e := snap.ReadMaintenanceLedger(ctx, "user-1001")
	if e != nil || l.ForeignRows == 0 || len(l.Originals) != 1 || l.Originals[0].ID != own.ID || l.Originals[0].State != "UNKNOWN" {
		t.Fatal(l, e)
	}
	raw, e := json.Marshal(l)
	if e != nil || strings.Contains(string(raw), foreign.ID) || strings.Contains(string(raw), "foreign-name") || strings.Contains(string(raw), "never-print-secret") || strings.Contains(string(raw), "private.example") {
		t.Fatal("redaction", e)
	}
	if _, e = snap.ReadMaintenanceLedger(ctx, "default"); e == nil {
		t.Fatal("default accepted")
	}
}
