package state

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// This fixture uses the actual legacy original admission and terminal
// SQLite finalizers. It SIMULATES successful Root verification and is NOT
// itself a proof of kernel effect; production preflight must verify native
// ACL/conntrack/nft independently.
func pr178LegacyVerifiedPeerFixture(t *testing.T, s *Store,
	source, target, src4, dst4, src6, dst6 string, allow bool) string {
	t.Helper()
	ctx := context.Background()
	_, tok, e := s.IssueToken(ctx, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	i := RootPeerRevokeSafeIntent{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		SourceUUID: source, SourceIncarnation: source,
		TargetUUID: target, TargetIncarnation: target, Allow: allow,
	}
	raw, e := json.Marshal(i)
	if e != nil {
		t.Fatal(e)
	}
	key := fmt.Sprintf("pr178-legacy-%s-%s-%t", source[len(source)-5:], target[len(target)-5:], allow)
	receipt, e := s.Admit(ctx, Admission{
		Layer: i.Project, ActorID: tok.ID, TokenID: tok.ID,
		Key: key, RequestID: key, Action: "box_peer_set", SafeIntent: raw,
		BoxUUID: source, Incarnation: source,
		ResourceKey:  "acl:default/bw-s4b2",
		ResourceKeys: []string{"box:" + source, "box:" + target},
	})
	if e != nil {
		t.Fatal("historical legacy original admission", e)
	}
	if allow {
		_, e = s.db.ExecContext(ctx, `INSERT INTO peer_grants
   (project,source_uuid,source_incarnation,target_uuid,target_incarnation,
    desired,status,source_ipv4,target_ipv4,source_ipv6,target_ipv6,operation_id)
   VALUES('user-1001',?,?,?,?,1,'PENDING',?,?,?,?,?)`,
			source, source, target, target, src4, dst4, src6, dst6, receipt.ID)
	} else {
		_, e = s.db.ExecContext(ctx, `UPDATE peer_grants SET desired=0,status='PENDING',
   operation_id=? WHERE project='user-1001' AND source_uuid=? AND
   target_uuid=? AND status='CONFIRMED' AND desired=1`, receipt.ID, source, target)
	}
	if e != nil {
		t.Fatal("fixture failed to stage historical native peer", e)
	}
	if e = s.Claim(ctx, receipt.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.AckSynchronousACL(ctx, receipt.ID); e != nil {
		t.Fatal(e)
	}
	if allow {
		e = s.FinalizeRootPeerGrant(ctx, receipt.ID)
	} else {
		e = s.FinalizeRootPeerRevocation(ctx, receipt.ID)
	}
	if e != nil {
		t.Fatal("fixture verified original not committed", e)
	}
	return receipt.ID
}
