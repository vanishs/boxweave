package state

import (
	"context"
	"errors"
	"strings"
	"testing"
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

func TestPeerVerifiedGrantLinkIsAtomicAndNeverBackfilledOnUnknown(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3ScopedOwnedFixture(t)
	tok := purePeerNewToken(t, s)
	orig := purePeerAdmit(t, s, tok, "lineage-pending", 1, 2, true)
	if e := s.StageAndClaimScopedPeerOriginal(ctx, orig.ID); e != nil {
		t.Fatal(e)
	}
	if pending, verified, _ := verifiedPeerLineage(t, s); pending != orig.ID || verified != "" {
		t.Fatal(pending, verified)
	}
	proof := purePeerReceipt(t, s, orig)
	bad := proof
	bad.ProofSHA256 = strings.Repeat("f", 64)
	if e := s.FinalizeScopedPeerOriginal(ctx, bad); e == nil {
		t.Fatal("forged receipt accepted")
	}
	if e := s.Unknown(ctx, orig.ID, "DISPATCHING", "SIMULATED_ROOT_UNKNOWN"); e != nil {
		t.Fatal(e)
	}
	if e := s.FinalizeScopedPeerOriginal(ctx, proof); e == nil {
		t.Fatal("dispatch receipt settled UNKNOWN")
	}
	if pending, verified, _ := verifiedPeerLineage(t, s); pending != orig.ID || verified != "" {
		t.Fatal(pending, verified)
	}
}
func TestPeerGrantRevokeRegrantVerifiedOriginalNeverAliasesPreviousOriginal(t *testing.T) {
	ctx := context.Background()
	s, _, _ := n3ScopedOwnedFixture(t)
	tok := purePeerNewToken(t, s)
	prev := ""
	for _, tc := range []struct {
		key     string
		allow   bool
		desired int
	}{{"grant", true, 1}, {"revoke", false, 0}, {"regrant", true, 1}} {
		orig := purePeerAdmit(t, s, tok, "lineage-"+tc.key, 1, 2, tc.allow)
		if orig.ID == prev {
			t.Fatal("reused original")
		}
		if e := s.StageAndClaimScopedPeerOriginal(ctx, orig.ID); e != nil {
			t.Fatal(e)
		}
		if pending, verified, desired := verifiedPeerLineage(t, s); pending != orig.ID || verified != prev || desired != tc.desired {
			t.Fatal(tc.key, pending, verified, desired, prev)
		}
		if e := s.FinalizeScopedPeerOriginal(ctx, purePeerReceipt(t, s, orig)); e != nil {
			t.Fatal(e)
		}
		wantPending := ""
		if !tc.allow {
			wantPending = orig.ID
		}
		if pending, verified, desired := verifiedPeerLineage(t, s); pending != wantPending || verified != orig.ID || desired != tc.desired {
			t.Fatal(tc.key, pending, verified, desired)
		}
		prev = orig.ID
	}
}
func TestPeerCompletedUnknownGrantReconciliationRetainsOriginalWithoutReplay(t *testing.T) {
	scopedUnknownPeerLineage(t, true)
}
func TestPeerCompletedUnknownRevocationReconciliationRetainsOriginal(t *testing.T) {
	scopedUnknownPeerLineage(t, false)
}
func scopedUnknownPeerLineage(t *testing.T, allow bool) {
	t.Helper()
	ctx := context.Background()
	s, _, _ := n3ScopedOwnedFixture(t)
	tok := purePeerNewToken(t, s)
	prev := ""
	if !allow {
		grant := purePeerAdmit(t, s, tok, "lineage-before-revoke", 1, 2, true)
		purePeerComplete(t, s, grant)
		prev = grant.ID
	}
	orig := purePeerAdmit(t, s, tok, "lineage-lost-ack", 1, 2, allow)
	r := purePeerReadOnly(t, s)
	calls := 0
	var first ScopedPeerRootReceipt
	w := Worker{Store: s}
	callback := func(_ context.Context, id string) (ScopedPeerRootReceipt, error) {
		calls++
		if id != orig.ID {
			t.Fatal("wrong original", id)
		}
		first = scopedTestReceipt(t, r, id, false)
		return first, errors.New("synthetic lost ACK")
	}
	if e := w.ExecuteScopedPeerOriginal(orig.ID, callback); e == nil || calls != 1 {
		t.Fatal(e, calls)
	}
	if pending, verified, _ := verifiedPeerLineage(t, s); pending != orig.ID || verified != prev {
		t.Fatal(pending, verified, prev)
	}
	proof := scopedTestReceipt(t, r, orig.ID, true)
	if proof.ProofSHA256 != first.ProofSHA256 {
		t.Fatal("recovery changed staged digest")
	}
	if e := s.FinalizeScopedPeerOriginal(ctx, proof); e != nil {
		t.Fatal(e)
	}
	wantPending, wantDesired := "", 1
	if !allow {
		wantPending, wantDesired = orig.ID, 0
	}
	if pending, verified, desired := verifiedPeerLineage(t, s); pending != wantPending || verified != orig.ID || desired != wantDesired {
		t.Fatal(pending, verified, desired)
	}
	if rec, e := s.Get(ctx, orig.ID); e != nil || rec.State != "SUCCEEDED" || rec.NativeOperationID != "" {
		t.Fatal(rec, e)
	}
	if e := w.ExecuteScopedPeerOriginal(orig.ID, callback); e == nil || calls != 1 {
		t.Fatal("replayed", e, calls)
	}
}
