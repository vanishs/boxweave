package recovery

import (
	"context"
	"github.com/vanishs/boxweave/internal/state"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPR177D4ServiceReobservesAndRechecks(t *testing.T) {
	for _, fault := range []string{"complete", "native-drift", "authority-drift", "expired"} {
		t.Run(fault, func(t *testing.T) {
			ctx := context.Background()
			dir := t.TempDir()
			if e := os.Chmod(dir, 0700); e != nil {
				t.Fatal(e)
			}
			s, e := state.Open(filepath.Join(dir, "ledger.sqlite"))
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			_, tok, e := s.IssueToken(ctx, time.Hour)
			if e != nil {
				t.Fatal(e)
			}
			project := "user-1001"
			if e = s.RegisterOwnedBox(ctx, bu, project, "owned", bu); e != nil {
				t.Fatal(e)
			}
			rec, e := s.AdmitScoped(ctx, state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "d4-original-native-operation", Action: "box_start", RequestID: "d4-fixture", ResourceKey: "box:owned", BoxUUID: bu, Incarnation: bu, SafeIntent: []byte("{}")})
			if e != nil {
				t.Fatal(e)
			}
			if e = s.Claim(ctx, rec.ID); e != nil {
				t.Fatal(e)
			}
			if e = s.NativeAck(ctx, rec.ID, nu); e != nil {
				t.Fatal(e)
			}
			if e = s.Unknown(ctx, rec.ID, "VERIFYING", "TEST_ORIGINAL_UNKNOWN"); e != nil {
				t.Fatal(e)
			}
			o, l, e := s.ReadReviewedRecoveryEvidence(ctx, project, rec.ID, "SCOPED_V2")
			if e != nil {
				t.Fatal(e)
			}
			w := known()
			w.box.Project = project
			svc := Service{Store: s, Native: w, Layer: project}
			_, p := svc.observe(ctx, o.Target, "")
			if p.Kind != state.RecoverySettle {
				t.Fatal(p)
			}
			r := state.RecoveryRequest{Layer: project, LocalReviewer: "d4-operator", OriginalID: rec.ID, Key: "d4-reviewed-service-key", Review: &state.ReviewedRecoveryBinding{ReviewSHA256: strings.Repeat("a", 64), OriginalSHA256: o.Digest, LedgerSHA256: l.ObservationDigest(), WitnessSHA256: state.RecoveryWitnessDigest(o.Target, p), Mode: "SCOPED_V2", Leaf: "MCP_SETTLE_ORIGINAL_NO_REPLAY", ExpiresNS: time.Now().Add(time.Minute).UnixNano()}}
			if fault == "native-drift" {
				w.box.NativeState = "Stopped"
				svc.Native = w
			}
			if fault == "expired" {
				r.Review.ExpiresNS = time.Now().Add(-time.Second).UnixNano()
			}
			checks := 0
			check := func(context.Context) error {
				checks++
				if fault == "authority-drift" {
					return state.ErrConflict
				}
				return nil
			}
			result, e := svc.ReconcileReviewed(ctx, r, check)
			if fault != "complete" {
				if e == nil {
					t.Fatal("unsafe repair", result)
				}
				current, e := s.Get(ctx, rec.ID)
				if e != nil || current.State != "UNKNOWN" {
					t.Fatal(current, e)
				}
				return
			}
			if e != nil || checks != 1 || result.Disposition != "SETTLED" {
				t.Fatal(result, e, checks)
			}
			// Unavailable native observer after the commit cannot cause a replay.
			w.box.UUID = "wrong"
			svc.Native = w
			again, e := svc.ReconcileReviewed(ctx, r, func(context.Context) error { t.Fatal("duplicate rechecked native/authority"); return nil })
			if e != nil || !again.Receipt.Duplicate || again.Receipt.ID != result.Receipt.ID {
				t.Fatal(again, e)
			}
		})
	}
	t.Log("D4_REOBSERVED_NATIVE_AND_AUTHORITY_BEFORE_COMMIT_PASS")
}
