package state

import (
	"context"
	"encoding/json"
	"errors"
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

func TestH1BMaintenanceHistoryWitness(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	snap := DiagnosticSnapshot{store: s}
	a, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil || len(a.RetiredBoxes) != a.Retired || a.Retired != 1 || len(a.HistorySHA256) != 64 {
		t.Fatal(a, e)
	}
	again, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil || a.ObservationDigest() != again.ObservationDigest() {
		t.Fatal("stationary history", e)
	}
	// Only a synthetic test DB: change a historical row without changing counts.
	if _, e = s.db.Exec("UPDATE operations SET error_class='fixture_changed' WHERE layer=?", b.Project); e != nil {
		t.Fatal(e)
	}
	after, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil || after.Retired != a.Retired || after.Claims != a.Claims || after.HistorySHA256 == a.HistorySHA256 || after.ObservationDigest() == a.ObservationDigest() {
		t.Fatal("same-count history drift missed", e)
	}
}

// With no Legacy state, the new Scoped reader must preserve the exact
// schema-6 observation digest used by reviewed maintenance and drain.
func TestP1ScopedMaintenanceRetainsWitnessDigest(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	snap := DiagnosticSnapshot{store: s}
	legacy, e := snap.ReadMaintenanceLedger(ctx, b.Project)
	if e != nil {
		t.Fatal(e)
	}
	scoped, e := snap.ReadScopedMaintenanceLedger(ctx, b.Project)
	if e != nil || scoped.ObservationDigest() != legacy.ObservationDigest() || scoped.HistorySHA256 != legacy.HistorySHA256 {
		t.Fatal(scoped, e)
	}
	drain, e := s.ReadScopedDrainWitness(ctx, b.Project)
	if e != nil || drain.ObservationDigest() != scoped.ObservationDigest() {
		t.Fatal(drain, e)
	}
}

func TestP1ScopedMaintenanceRefusesMixedHistory(t *testing.T) {
	ctx := context.Background()
	for _, table := range []string{"network_ipv4_retirements", "network_ipv4_retirement_queue", "network_ipv4_compactions"} {
		t.Run(table, func(t *testing.T) {
			s, b, del := n3RetiredFixture(t)
			var e error
			// Synthetic mixed rows in the same Project; never native effects.
			switch table {
			case "network_ipv4_retirements":
				_, e = s.db.Exec(`INSERT INTO network_ipv4_retirements(project,box_uuid,incarnation,bridge,acl,mac,ipv4,ipv6,create_operation_id,delete_operation_id,native_proof_sha256,retired_at)
 SELECT n.project,n.box_uuid,n.incarnation,n.bridge,n.acl,n.mac,n.ipv4,n.ipv6,c.id,?,?,1 FROM network_bindings n JOIN operations c ON c.layer=n.project AND c.box_uuid=n.box_uuid AND c.incarnation=n.incarnation AND c.action='box_create' WHERE n.project=? AND n.box_uuid=?`, del.ID, strings.Repeat("a", 64), b.Project, b.UUID)
			case "network_ipv4_retirement_queue":
				_, e = s.db.Exec("INSERT INTO network_ipv4_retirement_queue(delete_operation_id,project,attempt_order) VALUES(?,?,1)", del.ID, b.Project)
			case "network_ipv4_compactions":
				_, e = s.db.Exec("INSERT INTO network_ipv4_compactions(project,delete_operation_id,plan_sha256,prepared_at) VALUES(?,?,?,1)", b.Project, del.ID, strings.Repeat("a", 64))
			}
			if e != nil {
				t.Fatal(e)
			}
			var n int
			if e = s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil || n != 1 {
				t.Fatal(n, e)
			}
			snap := DiagnosticSnapshot{store: s}
			legacy, e := snap.ReadMaintenanceLedger(ctx, b.Project)
			if e != nil || len(legacy.HistorySHA256) != 64 {
				t.Fatal(legacy, e)
			}
			if l, e := snap.ReadScopedMaintenanceLedger(ctx, b.Project); !errors.Is(e, ErrResourceLocked) || l.HistorySHA256 != "" {
				t.Fatal("mixed snapshot authorized", l, e)
			}
			if l, e := s.ReadScopedDrainWitness(ctx, b.Project); !errors.Is(e, ErrResourceLocked) || l.HistorySHA256 != "" {
				t.Fatal("mixed drain authorized", l, e)
			}
			after, e := snap.ReadMaintenanceLedger(ctx, b.Project)
			if e != nil || after.ObservationDigest() != legacy.ObservationDigest() {
				t.Fatal("observation changed", after, e)
			}
		})
	}
}
