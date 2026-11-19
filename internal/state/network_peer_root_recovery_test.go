package state

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestS4B2RootUnknownCompletedReceiptReadOnlyRecoveryAndNewAdmission(t *testing.T) {
	ctx := context.Background()
	s, _, id := s4b2RootOperationFixture(t)
	s4b2PendingExistingGrantToConfirmed(t, s)
	if err := s.StageRootPeerRevocation(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, id, "DISPATCHING", "SIMULATED_ACK_LOST_AFTER_ROOT_COMPLETED"); err != nil {
		t.Fatal(err)
	}
	sha := strings.Repeat("a", 64)
	calls := 0
	w := Worker{Store: s}
	proof := func(_ context.Context, original, digest string) (RootPeerResult, error) {
		calls++
		if original != id || digest != sha {
			t.Fatal("root recovery received unapproved original")
		}
		return RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha,
			ProbedScopes: 4, DeletedScopes: 2, MutationComplete: true, KernelVerified: true,
			VerifiedNICs: 3, PostNativeDigestSHA256: strings.Repeat("b", 64)}, nil
	}
	if err := w.ReconcileCompletedUnknownRootPeer(id, sha, proof); err != nil {
		t.Fatal(err)
	}
	rec, err := s.Get(ctx, id)
	if err != nil || rec.State != "SUCCEEDED" || calls != 1 {
		t.Fatalf("recovery failed to preserve old original: %+v %v %d", rec, err, calls)
	}
	if err := w.ReconcileCompletedUnknownRootPeer(id, sha, proof); err == nil || calls != 1 {
		t.Fatal("already reconciled original replayed")
	}
	_, tok, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := s.Admit(ctx, Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key:       "s4b2-reviewed-new-original-after-completed-unknown",
		RequestID: "fresh-reviewed", Action: "network_acl_reconcile", SafeIntent: []byte("new separately reviewed ACL intent"),
		ResourceKey: "acl:default/bw-s4b2"})
	if err != nil || fresh.ID == id {
		t.Fatalf("fresh ACL authorization did not use a new original: %+v %v", fresh, err)
	}
	t.Log("S4_B2_ROOT_UNKNOWN_COMPLETED_JOURNAL_READONLY_RECOVERY_NEW_ORIGINAL_AUTHORIZATION=PASS")
}

func TestS4B2RootUnknownIncompleteOrForgedReceiptPreservesFence(t *testing.T) {
	for _, bad := range []string{"lost-root", "no-kernel", "no-nics", "wrong-sha", "partial", "incomplete-digest"} {
		t.Run(bad, func(t *testing.T) {
			ctx := context.Background()
			s, _, id := s4b2RootOperationFixture(t)
			s4b2PendingExistingGrantToConfirmed(t, s)
			if err := s.StageRootPeerRevocation(ctx, id); err != nil {
				t.Fatal(err)
			}
			if err := s.Claim(ctx, id); err != nil {
				t.Fatal(err)
			}
			if err := s.Unknown(ctx, id, "DISPATCHING", "SIMULATED_PARTIAL_ROOT_DISPATCH"); err != nil {
				t.Fatal(err)
			}
			sha := strings.Repeat("a", 64)
			result := RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha,
				ProbedScopes: 4, DeletedScopes: 2, MutationComplete: true, KernelVerified: true,
				VerifiedNICs: 3, PostNativeDigestSHA256: strings.Repeat("b", 64)}
			if bad == "no-kernel" {
				result.KernelVerified = false
			}
			if bad == "no-nics" {
				result.VerifiedNICs = 0
			}
			if bad == "wrong-sha" {
				result.PlanSHA256 = strings.Repeat("c", 64)
			}
			if bad == "partial" {
				result.ProbedScopes = 3
			}
			if bad == "incomplete-digest" {
				result.PostNativeDigestSHA256 = ""
			}
			if err := (Worker{Store: s}).ReconcileCompletedUnknownRootPeer(id, sha,
				func(context.Context, string, string) (RootPeerResult, error) {
					if bad == "lost-root" {
						return RootPeerResult{}, errors.New("root unreachable")
					}
					return result, nil
				}); err == nil {
				t.Fatal("unsafe partial/forged root recovery accepted")
			}
			r, err := s.Get(ctx, id)
			if err != nil || r.State != "UNKNOWN" {
				t.Fatalf("unknown recovered without proof: %+v %v", r, err)
			}
			_, tok, err := s.IssueToken(ctx,time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Admit(ctx, Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
				Key: "s4b2-unsafe-unknown-conflict", RequestID: "unsafe",
				Action: "network_acl_reconcile", SafeIntent: []byte("forbidden"),
				ResourceKey: "acl:default/bw-s4b2"}); !errors.Is(err, ErrResourceLocked) {
				t.Fatalf("unknown ACL claim released before verified recovery: %v", err)
			}
		})
	}
	t.Log("S4_B2_ROOT_UNKNOWN_BAD_RECOVERY_FROZEN_AND_NO_NEW_ADMISSION=PASS")
}

// The older local recovery challenge is insufficient to release a
// kernel-global root ACL/conntrack UNKNOWN. Even explicit acknowledgment and
// an exact UUID cannot bypass a partially unknown host effect.
func TestS4B2RootUnknownGenericLegacyRecoveryCannotReleasePeerClaims(t *testing.T) {
	ctx := context.Background()
	s, _, id := s4b2RootOperationFixture(t)
	s4b2PendingExistingGrantToConfirmed(t, s)
	if err := s.StageRootPeerRevocation(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, id, "DISPATCHING", "ROOT_INTENT_ACK_UNKNOWN"); err != nil {
		t.Fatal(err)
	}
	snap, err := s.RecoveryInspect(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Action != "box_peer_set" || snap.State != "UNKNOWN" {
		t.Fatal("not the root peer original")
	}
	observation := "operator observed no finished root journal"
	challenge, err := s.IssueRecoveryChallenge(ctx, snap, "rootreview", observation)
	if err != nil {
		t.Fatal(err)
	}
	err = s.ReleaseRecoveryClaim(ctx, snap, "rootreview", challenge, snap.BoxUUID,
		"Root peer UNKNOWN must not be generically unlocked", observation, true)
	if !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("general recovery bypassed root journal: %v", err)
	}
	record, err := s.Get(ctx, id)
	if err != nil || record.State != "UNKNOWN" {
		t.Fatal("root original moved after unsafe release")
	}
	_, tok, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Admit(ctx, Admission{Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key: "s4b2-generic-root-unlock-forbidden", RequestID: "old-root-bypass",
		Action: "network_acl_reconcile", SafeIntent: []byte("forbidden native ACL release"),
		ResourceKey: "acl:default/bw-s4b2"}); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("generic recovery released the root peer ACL despite UNKNOWN: %v", err)
	}
	t.Log("S4_B2_PEER_UNKNOWN_LEGACY_GENERIC_RELEASE_SECURITY_BYPASS_DENIED=PASS")
}
