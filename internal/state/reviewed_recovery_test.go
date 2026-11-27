package state

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func d4Review(t *testing.T, s *Store, layer, id, mode, leaf string, p RecoveryProof) (RecoveryRequest, RecoveryTarget) {
	t.Helper()
	o, l, e := s.ReadReviewedRecoveryEvidence(t.Context(), layer, id, mode)
	if e != nil {
		t.Fatal(e)
	}
	b := &ReviewedRecoveryBinding{ReviewSHA256: strings.Repeat("a", 64), OriginalSHA256: o.Digest, LedgerSHA256: l.ObservationDigest(), WitnessSHA256: RecoveryWitnessDigest(o.Target, p), Mode: mode, Leaf: leaf, ExpiresNS: time.Now().Add(time.Minute).UnixNano()}
	return RecoveryRequest{Layer: layer, LocalReviewer: "d4-operator", OriginalID: id, Key: "d4-review-exact-key-0001", Review: b}, o.Target
}
func TestPR177D4ReviewedCommitBoundary(t *testing.T) {
	for _, fault := range []string{"complete", "expired", "wrong-proof", "wrong-leaf", "wrong-mode", "history", "claim", "ledger", "revoked", "partial", "commit-fault", "mcp-injection"} {
		t.Run(fault, func(t *testing.T) {
			ctx := context.Background()
			s, ro, id := pr178ScopedReadyToStage(t, true)
			if e := s.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
				t.Fatal(e)
			}
			if e := s.Unknown(ctx, id, "DISPATCHING", "SCOPED_PEER_V2_ROOT_OUTCOME_UNVERIFIED"); e != nil {
				t.Fatal(e)
			}
			orig, e := s.ReadRecoveryTarget(ctx, "user-1001", id)
			if e != nil {
				t.Fatal(e)
			}
			p := RecoveryProof{Kind: RecoverySettle, Reason: "SCOPED_PEER_COMPLETED_NO_REPLAY", UUID: orig.UUID, Fingerprint: "fixture-read-only-witness", ScopedPeer: scopedTestReceipt(t, ro, id, true)}
			r, target := d4Review(t, s, orig.Layer, id, "SCOPED_V2", "MCP_SETTLE_ORIGINAL_NO_REPLAY", p)
			switch fault {
			case "expired":
				r.Review.ExpiresNS = time.Now().Add(-time.Second).UnixNano()
			case "wrong-proof":
				p.Fingerprint = "changed"
			case "wrong-leaf":
				r.Review.Leaf = "MCP_BIND_ORIGINAL_IDENTITY"
			case "wrong-mode":
				r.Review.Mode = "LEGACY"
			case "history":
				if _, e = s.db.Exec("UPDATE operation_events SET safe_detail=safe_detail||'-drift' WHERE operation_id=? AND seq=(SELECT min(seq) FROM operation_events WHERE operation_id=?)", id, id); e != nil {
					t.Fatal(e)
				}
			case "claim":
				if _, e = s.db.Exec("INSERT INTO operation_resource_claims(operation_id,layer,resource_key) VALUES(?,?,'unreviewed-resource')", id, orig.Layer); e != nil {
					t.Fatal(e)
				}
			case "ledger":
				if _, e = s.db.Exec("UPDATE owned_boxes SET name=name||'-changed' WHERE project=? AND uuid!=?", orig.Layer, orig.UUID); e != nil {
					t.Fatal(e)
				}
			case "revoked":
				var token string
				if e = s.db.QueryRow("SELECT token_id FROM operations WHERE id=?", id).Scan(&token); e != nil {
					t.Fatal(e)
				}
				if e = s.RevokeToken(ctx, token); e != nil {
					t.Fatal(e)
				}
			case "partial":
				p.ScopedPeer.Status = "PARTIAL"
				p.ScopedPeer.VerifiedSteps = 1
				r.Review.WitnessSHA256 = RecoveryWitnessDigest(target, p)
			case "commit-fault":
				if _, e = s.db.Exec("CREATE TRIGGER d4_abort BEFORE INSERT ON operation_events WHEN NEW.safe_detail LIKE 'SETTLED:%' BEGIN SELECT RAISE(ABORT,'fixture commit interruption'); END"); e != nil {
					t.Fatal(e)
				}
			case "mcp-injection":
				r.LocalReviewer = ""
				r.ActorID = strings.Repeat("1", 32)
				r.TokenID = r.ActorID
			}
			rec, e := s.CommitRecovery(ctx, r, target, p)
			if fault != "complete" {
				if e == nil {
					t.Fatal("unsafe reviewed commit", rec)
				}
				var count int
				if e = s.db.QueryRow("SELECT count(*) FROM operations WHERE action='operation_recover'").Scan(&count); e != nil || count != 0 {
					t.Fatal("refusal created receipt", count, e)
				}
				current, e := s.Get(ctx, id)
				if e != nil || current.State != "UNKNOWN" {
					t.Fatal("refusal changed original", current, e)
				}
				return
			}
			if e != nil || rec.Disposition != "SETTLED" || rec.Receipt.NativeOperationID != "" || rec.Receipt.ID == id {
				t.Fatal(rec, e)
			}
			post, e := s.ReadReviewedRecoveryPostcheck(ctx, r, rec)
			if e != nil || post.State != "DURABLE_ORIGINAL_POSTSTATE_VERIFIED" || post.OriginalState != "SUCCEEDED" {
				t.Fatal(post, e)
			}
			// Same exact review/key returns one receipt even after expiry; never
			// authorize a new key by refreshing the original plan's expiry.
			r.Review.ExpiresNS = time.Now().Add(-time.Second).UnixNano()
			again, e := s.CommitRecovery(ctx, r, target, p)
			if e != nil || !again.Receipt.Duplicate || again.Receipt.ID != rec.Receipt.ID {
				t.Fatal(again, e)
			}
			changed := r
			copy := *r.Review
			copy.ReviewSHA256 = strings.Repeat("b", 64)
			changed.Review = &copy
			if _, e = s.ReplayRecovery(ctx, changed); !errors.Is(e, ErrConflict) {
				t.Fatal("same key different review", e)
			}
			changed = r
			changed.Key = "d4-different-recovery-key"
			if _, e = s.CommitRecovery(ctx, changed, target, p); e == nil {
				t.Fatal("stale new key")
			}
			var n int
			if e = s.db.QueryRow("SELECT count(*) FROM recovery_audit WHERE operation_id=? AND decision='RECOVERY_SETTLED'", id).Scan(&n); e != nil || n != 1 {
				t.Fatal(n, e)
			}
		})
	}
	t.Log("D4_EXACT_REVIEW_ATOMIC_REFUSAL_AND_LOST_ACK_PASS")
}
func TestPR177D4ReviewedConcurrencyAndIdentityBinding(t *testing.T) {
	ctx := context.Background()
	for _, same := range []bool{true, false} {
		t.Run(fmt.Sprint(same), func(t *testing.T) {
			s := newStore(t)
			r0, target := recoveryOriginalMode(t, s, "box_start", true)
			p := recoverProof(target)
			p.LifecycleBinding = target.LifecycleBinding
			r, _ := d4Review(t, s, r0.Layer, r0.OriginalID, "SCOPED_V2", "MCP_SETTLE_ORIGINAL_NO_REPLAY", p)
			type outcome struct {
				rec RecoveryRecord
				e   error
			}
			out := make(chan outcome, 2)
			var wg sync.WaitGroup
			for i := 0; i < 2; i++ {
				req := r
				if !same {
					req.Key = fmt.Sprintf("different-reviewed-key-%d", i)
				}
				wg.Add(1)
				go func() { defer wg.Done(); rec, e := s.CommitRecovery(ctx, req, target, p); out <- outcome{rec, e} }()
			}
			wg.Wait()
			close(out)
			var ids []string
			conflicts := 0
			for x := range out {
				if errors.Is(x.e, ErrConflict) {
					conflicts++
					continue
				}
				if x.e != nil {
					t.Fatal(x.e)
				}
				ids = append(ids, x.rec.Receipt.ID)
			}
			if same && (len(ids) != 2 || ids[0] != ids[1] || conflicts != 0) || !same && (len(ids) != 1 || conflicts != 1) {
				t.Fatal(ids, conflicts)
			}
		})
	}
	s, original, _, _ := p1CreateFixture(t, false)
	if e := s.Unknown(ctx, original.ID, "DISPATCHING", "ACK_LOST"); e != nil {
		t.Fatal(e)
	}
	r0, target := a4Request(t, s, original.ID)
	p := RecoveryProof{Kind: RecoveryBind, Reason: "CREATE_MARKER_EXACT_IDENTITY", UUID: p1UUID, Fingerprint: "reviewed-marker"}
	r, _ := d4Review(t, s, r0.Layer, r0.OriginalID, "SCOPED_V2", "MCP_BIND_ORIGINAL_IDENTITY", p)
	if _, e := s.CommitScopedRecovery(ctx, r, target, p); e == nil {
		t.Fatal("retired reviewed bind accepted")
	}
	current, e := s.ReadRecoveryTarget(ctx, r.Layer, target.ID)
	if e != nil || current != target || current.State != "UNKNOWN" || current.Released {
		t.Fatal(current, e)
	}
	if _, e = s.ReplayRecovery(ctx, r); !errors.Is(e, ErrNotFound) {
		t.Fatal("refusal created receipt", e)
	}
	t.Log("D4_CONCURRENT_REVIEW_AND_IDENTITY_PROTECTION_PASS")
}

func TestPR177D4ExistingOnlyWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", "ledger.sqlite")
	if s, e := OpenExistingRecovery(path); e == nil {
		s.Close()
		t.Fatal("created absent database")
	}
	if _, e := os.Lstat(filepath.Dir(path)); !os.IsNotExist(e) {
		t.Fatal("created source parent", e)
	}
}
