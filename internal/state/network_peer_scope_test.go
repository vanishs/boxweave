package state

import (
	"context"
	"testing"
)

func TestS4B3ResolvePeerScopeOnlyPinnedCurrentOwnedPair(t *testing.T) {
	s, _, _ := s4b2RootOperationFixture(t)
	nw, acl, err := s.ResolvePeerAuditScope(context.Background(), "user-1001", s4b2RootSrc, s4b2RootDst)
	if err != nil || nw != "bwbr0" || acl != "bw-s4b2" {
		t.Fatalf("scope not exact: %q %q %v", nw, acl, err)
	}
	for _, tc := range []struct{ project, src, dst string }{
		{"foreign", s4b2RootSrc, s4b2RootDst},
		{"user-1001", s4b2RootSrc, s4b2RootSrc},
		{"user-1001", s4b2RootSrc, "foreign-uuid"},
	} {
		if a, b, e := s.ResolvePeerAuditScope(context.Background(), tc.project, tc.src, tc.dst); e == nil {
			t.Fatalf("untrusted scope accepted %q %q", a, b)
		}
	}
	if _, err := s.db.Exec("UPDATE network_bindings SET state='OBSERVED' WHERE box_uuid=?", s4b2RootDst); err != nil {
		t.Fatal(err)
	}
	if a, b, e := s.ResolvePeerAuditScope(context.Background(), "user-1001", s4b2RootSrc, s4b2RootDst); e == nil {
		t.Fatalf("unpinned target accepted: %q %q", a, b)
	}
}
