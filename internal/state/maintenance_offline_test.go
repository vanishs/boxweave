package state

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestH2BExactOriginalObservation(t *testing.T) {
	ctx := context.Background()
	s, b, _ := n3RetiredFixture(t)
	b, e := s.OwnedBox(ctx, b.Project, "11111111-1111-4111-8111-000000000002")
	if e != nil {
		t.Fatal(e)
	}
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(struct {
		BoxID string `json:"box_id"`
	}{b.UUID})
	rec, e := s.Admit(ctx, Admission{Layer: b.Project, ActorID: tok.ID, TokenID: tok.ID, Key: "h2b-power-original", RequestID: "h2b-power", Action: "box_stop", ResourceKey: "box:" + b.Name, BoxUUID: b.UUID, Incarnation: b.Incarnation, SafeIntent: raw})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Claim(ctx, rec.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.NativeAck(ctx, rec.ID, "native-power-original"); e != nil {
		t.Fatal(e)
	}
	if e = s.Unknown(ctx, rec.ID, "VERIFYING", "H2B_FIXTURE_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	snap := DiagnosticSnapshot{store: s}
	for i := 0; i < 2; i++ {
		l, e := snap.ReadMaintenanceLedger(ctx, b.Project)
		if e != nil {
			t.Fatal(e)
		}
		if len(l.Originals) != 1 {
			t.Fatal(l.Originals)
		}
		o := l.Originals[0]
		if o.ID != rec.ID || o.State != "UNKNOWN" || o.RetirementScope != "EXACT_OWNED_POWER_ORIGINAL" || o.ClaimProtection != "RETAINED" || o.Claims != 1 {
			t.Fatal(o)
		}
	}
}
