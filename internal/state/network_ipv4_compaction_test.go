package state

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// Shared ACL DROP compaction is retired. Exact revoked directions, including
// a retired endpoint, survive; a forged old preview cannot compact history.
func TestIPv4PeerCompactionDurableFenceAndHistory(t *testing.T) {
	ctx := context.Background()
	s, one, token := n3ScopedOwnedFixture(t)
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "compact-grant", 1, 2, true))
	revoke := purePeerAdmit(t, s, token, "compact-revoke", 1, 2, false)
	purePeerComplete(t, s, revoke)
	purePeerComplete(t, s, purePeerAdmit(t, s, token, "keep-grant", 2, 3, true))
	keep := purePeerAdmit(t, s, token, "keep-revoke", 2, 3, false)
	purePeerComplete(t, s, keep)
	n3DeleteOwned(t, s, one, token)
	before, e := s.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge)
	if e != nil || len(before.Revoked) != 2 || len(before.Active) != 0 {
		t.Fatal(before, e)
	}
	r := purePeerReadOnly(t, s)
	after, e := r.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal(after, e)
	}
	var count int
	if e = s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE operation_id IN (?,?) AND status='REVOKED'", revoke.ID, keep.ID).Scan(&count); e != nil || count != 2 {
		t.Fatal(count, e)
	}
	if _, e = s.db.Exec("UPDATE peer_grants SET source_ipv4='10.200.0.99' WHERE operation_id=?", revoke.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = r.ReadScopedTerminalPeerPolicy(ctx, purePeerProject, purePeerBridge); !errors.Is(e, ErrResourceLocked) {
		t.Fatal("corrupt historical direction accepted", e)
	}
}
