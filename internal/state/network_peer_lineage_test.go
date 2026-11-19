package state

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func verifiedPeerLineage(t *testing.T, s *Store) (string, string, int) {
	t.Helper()
	var pending, verified string
	var desired int
	err := s.db.QueryRow("SELECT operation_id,COALESCE(last_verified_operation_id,''),desired FROM peer_grants").
		Scan(&pending, &verified, &desired)
	if err != nil {
		t.Fatal(err)
	}
	return pending, verified, desired
}

func admitPeerLineageOriginal(t *testing.T, s *Store, allow bool, key string) string {
	t.Helper()
	ctx := context.Background()
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(RootPeerRevokeSafeIntent{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		SourceUUID: s4b2RootSrc, SourceIncarnation: s4b2RootSrc,
		TargetUUID: s4b2RootDst, TargetIncarnation: s4b2RootDst, Allow: allow,
	})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := s.Admit(ctx, Admission{
		Layer: "user-1001", ActorID: tok.ID, TokenID: tok.ID,
		Key: key, RequestID: key, Action: "box_peer_set",
		SafeIntent: raw, BoxUUID: s4b2RootSrc, Incarnation: s4b2RootSrc,
		ResourceKey: "acl:default/bw-s4b2",
		ResourceKeys: []string{"box:" + s4b2RootSrc, "box:" + s4b2RootDst},
	})
	if err != nil {
		t.Fatal(err)
	}
	return rec.ID
}

func TestPeerVerifiedGrantLinkIsAtomicAndNeverBackfilledOnUnknown(t *testing.T) {
	ctx := context.Background()
	s, id := s4b3GrantFixture(t)
	if err := s.StagePeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	if pending, verified, _ := verifiedPeerLineage(t, s); pending != id || verified != "" {
		t.Fatalf("staging fabricated a verified grant: %s %s", pending, verified)
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); !errors.Is(err, ErrState) {
		t.Fatalf("pre-kernel confirmation was accepted: %v", err)
	}
	if _, verified, _ := verifiedPeerLineage(t, s); verified != "" {
		t.Fatal("unverified grant wrote a permanent original")
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, id, "DISPATCHING", "SIMULATED_ROOT_UNKNOWN"); err != nil {
		t.Fatal(err)
	}
	if _, verified, _ := verifiedPeerLineage(t, s); verified != "" {
		t.Fatal("Root UNKNOWN was reported as verified")
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); !errors.Is(err, ErrState) {
		t.Fatalf("Root UNKNOWN bypassed verified settlement: %v", err)
	}
	if pending, verified, _ := verifiedPeerLineage(t, s); pending != id || verified != "" {
		t.Fatalf("Root UNKNOWN lost pending original or invented proof: %s %s", pending, verified)
	}
}

func TestPeerGrantRevokeRegrantVerifiedOriginalNeverAliasesPreviousOriginal(t *testing.T) {
	ctx := context.Background()
	s, first := s4b3GrantFixture(t)
	if err := s.StagePeerGrant(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := s.AckSynchronousACL(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, first); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != "" || verified != first || desired != 1 {
		t.Fatalf("confirmed grant lost verified original: pending=%q verified=%q desired=%d", pending, verified, desired)
	}
	revoke := admitPeerLineageOriginal(t, s, false, "lineage-revoke")
	if err := s.StageRootPeerRevocation(ctx, revoke); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != revoke || verified != first || desired != 0 {
		t.Fatalf("staged revoke overwrote last verified grant: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
	if err := s.Claim(ctx, revoke); err != nil {
		t.Fatal(err)
	}
	if err := s.AckSynchronousACL(ctx, revoke); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerRevocation(ctx, revoke); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != revoke || verified != revoke || desired != 0 {
		t.Fatalf("revoke did not link its own verified original: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
	regrant := admitPeerLineageOriginal(t, s, true, "lineage-regrant")
	if err := s.StagePeerGrant(ctx, regrant); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != regrant || verified != revoke || desired != 1 {
		t.Fatalf("staged regrant incorrectly changed verified provenance: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
	if err := s.Claim(ctx, regrant); err != nil {
		t.Fatal(err)
	}
	if err := s.AckSynchronousACL(ctx, regrant); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, regrant); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != "" || verified != regrant || desired != 1 {
		t.Fatalf("regrant did not persist exact terminal original: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
}

func TestPeerCompletedUnknownGrantReconciliationRetainsOriginalWithoutReplay(t *testing.T) {
	s, _, id := s4b3UnknownGrantRecoveryFixture(t)
	ctx := context.Background()
	sha := strings.Repeat("c", 64)
	proof := RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha,
		ProbedScopes: 4, MutationComplete: true, KernelVerified: true,
		VerifiedNICs: 3, PostNativeDigestSHA256: strings.Repeat("d", 64)}
	if _, verified, _ := verifiedPeerLineage(t, s); verified != "" {
		t.Fatal("Root UNKNOWN already had verified provenance")
	}
	if err := (Worker{Store: s}).ReconcileCompletedUnknownRootPeer(id, sha,
		func(context.Context, string, string) (RootPeerResult, error) { return proof, nil }); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != "" || verified != id || desired != 1 {
		t.Fatalf("read-only reconciliation lost exact original: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
	if rec, err := s.Get(ctx, id); err != nil || rec.State != "SUCCEEDED" {
		t.Fatalf("settlement failed: %+v %v", rec, err)
	}
}

func TestPeerCompletedUnknownRevocationReconciliationRetainsOriginal(t *testing.T) {
	ctx := context.Background()
	s, _, id := s4b2RootOperationFixture(t)
	s4b2PendingExistingGrantToConfirmed(t, s)
	if err := s.StageRootPeerRevocation(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, id, "DISPATCHING", "SIMULATED_ACK_LOSS"); err != nil {
		t.Fatal(err)
	}
	if _, verified, _ := verifiedPeerLineage(t, s); verified != "" {
		t.Fatal("Root UNKNOWN revoke had false verified original")
	}
	sha := strings.Repeat("a", 64)
	proof := RootPeerResult{Status: "RECONCILE_VERIFIED", PlanSHA256: sha,
		ProbedScopes: 4, DeletedScopes: 2, MutationComplete: true,
		KernelVerified: true, VerifiedNICs: 3, PostNativeDigestSHA256: strings.Repeat("b", 64)}
	if err := (Worker{Store: s}).ReconcileCompletedUnknownRootPeer(id, sha,
		func(context.Context, string, string) (RootPeerResult, error) { return proof, nil }); err != nil {
		t.Fatal(err)
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != id || verified != id || desired != 0 {
		t.Fatalf("recovered revoke misattributed provenance: pending=%s verified=%s desired=%d", pending, verified, desired)
	}
}
