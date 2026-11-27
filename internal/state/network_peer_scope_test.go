package state

import (
	"context"
	"testing"
)

func TestS4B3ResolvePeerScopeOnlyPinnedCurrentOwnedPair(t *testing.T) {

	ctx := context.Background()
	s, _, token := n3ScopedOwnedFixture(t)
	a := Admission{Layer: purePeerProject, ActorID: token, TokenID: token, Key: "scope-admission-test-001", RequestID: "scope"}
	for _, x := range [][3]string{{"user-1002", purePeerUUID(1), purePeerUUID(2)}, {purePeerProject, purePeerUUID(1), purePeerUUID(1)}, {purePeerProject, purePeerUUID(1), "foreign-uuid"}} {
		bad := a
		bad.Layer = x[0]
		if _, e := s.AdmitScopedPeerSet(ctx, bad, x[1], x[2], true); e == nil {
			t.Fatal("untrusted pair", x)
		}
	}
	if _, e := s.db.Exec("UPDATE network_bindings SET state='OBSERVED' WHERE box_uuid=?", purePeerUUID(2)); e != nil {
		t.Fatal(e)
	}
	if _, e := s.AdmitScopedPeerSet(ctx, a, purePeerUUID(1), purePeerUUID(2), true); e == nil {
		t.Fatal("unpinned")
	}
	if _, e := s.db.Exec("UPDATE network_bindings SET state='PINNED' WHERE box_uuid=?", purePeerUUID(2)); e != nil {
		t.Fatal(e)
	}
	r, e := s.AdmitScopedPeerSet(ctx, a, purePeerUUID(1), purePeerUUID(2), true)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
		t.Fatal(e)
	}
	p, e := purePeerReadOnly(t, s).ReadRootScopedPeerOriginal(ctx, r.ID, purePeerProject, purePeerBridge)
	if e != nil || p.Intent.SourceUUID != purePeerUUID(1) || p.Intent.TargetUUID != purePeerUUID(2) || len(p.Inventory.Boxes) != 3 || p.Inventory.Boxes[0].ACL == p.Inventory.Boxes[1].ACL {
		t.Fatal(p, e)
	}

}
