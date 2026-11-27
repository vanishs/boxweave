package state

import (
	"context"
	"os"
	"testing"
)

func s4b3RootGrantFixture(t *testing.T) (*Store, *Store, string) {
	t.Helper()
	s, id := s4b3GrantFixture(t)
	if err := s.StagePeerGrant(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	var seq int
	var name, path string
	if err := s.db.QueryRow("PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	r, err := OpenNetworkAuditReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return s, r, id
}

func TestS4B3RootGrantOnlyExactClaimedOriginal(t *testing.T) {

	ctx := context.Background()
	s, r, id := pr178ScopedOriginalFixture(t, true)
	if _, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
		t.Fatal("unclaimed grant")
	}
	if e := s.Claim(ctx, id); e != nil {
		t.Fatal(e)
	}
	p, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge)
	if e != nil || p.OriginalID != id || !p.Intent.Allow || p.RecoveryOnly || len(p.After) != 2 {
		t.Fatal(p, e)
	}
	if _, e = r.ReadRootPeerRevocation(ctx, id, purePeerProject, purePeerBridge, "bw-s4b2"); e == nil {
		t.Fatal("legacy revoke accepted")
	}
	t.Log("S4_B3_ROOT_GRANT_CLAIMED_ORIGINAL_NOT_REVOKE=PASS")

}

func TestS4B3RootGrantRejectsPoststageDrift(t *testing.T) {

	ctx := context.Background()
	for _, sql := range []string{"UPDATE operations SET action='box_start' WHERE id=?", "UPDATE operations SET layer='foreign' WHERE id=?", "UPDATE credentials SET revoked=1 WHERE id=(SELECT token_id FROM operations WHERE id=?)", "UPDATE peer_grants SET target_uuid='foreign' WHERE operation_id=?", "UPDATE peer_grants SET desired=0 WHERE operation_id=?", "UPDATE peer_grants SET status='UNKNOWN' WHERE operation_id=?"} {
		t.Run(sql, func(t *testing.T) {
			s, r, id := pr178ScopedOriginalFixture(t, true)
			if e := s.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			if _, e := r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e != nil {
				t.Fatal(e)
			}
			res, e := s.db.Exec(sql, id)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := res.RowsAffected(); e != nil || n == 0 {
				t.Fatal(n, e)
			}
			if _, e = r.ReadRootScopedPeerOriginal(ctx, id, purePeerProject, purePeerBridge); e == nil {
				t.Fatal("drift accepted")
			}
		})
	}
	t.Log("S4_B3_ROOT_GRANT_UNKNOWN_CLAIMS_FOREIGN_FENCES_FAIL_CLOSED=PASS")

}
