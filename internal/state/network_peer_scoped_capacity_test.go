package state

import (
	"context"
	"errors"
	"testing"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// Seed synthetic terminal audit history only, not native proof or actual grants.
func pr178SeedPolicyHistory(t *testing.T, s *Store, id string, count int) {
	t.Helper()
	_, err := s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<?)
 INSERT INTO operations(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,box_uuid,incarnation,state,created_at,updated_at)
 SELECT printf('capacity-%d',n.i),o.tracking_id,o.request_id,o.layer,o.actor_id,o.token_id,printf('capacity-%d',n.i),o.action,o.intent_digest,o.safe_intent,o.box_uuid,o.incarnation,'SUCCEEDED',o.created_at,o.updated_at FROM n,operations o WHERE o.id=?`, count, id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.db.Exec(`INSERT INTO operation_events(operation_id,at,state,safe_detail)
 SELECT id,updated_at,'SUCCEEDED','SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED' FROM operations WHERE id LIKE 'capacity-%'`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPR178N5PolicyHistoryCapacity(t *testing.T) {
	l := networklimits.ScopedPeerHistoryLimit
	for _, tc := range []struct {
		name   string
		allow  bool
		used   int
		refuse bool
	}{
		{"last grant with reserved revoke", true, l - 3, false},
		{"grant would strand revoke", true, l - 2, true},
		{"reserved revoke remains possible", false, l - 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, id := pr178ScopedReadyToStage(t, tc.allow)
			pr178SeedPolicyHistory(t, s, id, tc.used)
			ctx := context.Background()
			if !tc.refuse {
				if err := s.StageAndClaimScopedPeerOriginal(ctx, id); err != nil {
					t.Fatal(err)
				}
				return
			}
			called := false
			err := (Worker{Store: s}).ExecuteScopedPeerOriginal(id, func(context.Context, string) (ScopedPeerRootReceipt, error) {
				called = true
				return ScopedPeerRootReceipt{}, nil
			})
			if !errors.Is(err, ErrScopedPeerHistoryFull) || called {
				t.Fatal(err, called)
			}
			var phase, detail string
			if err = s.db.QueryRow("SELECT state,error_class FROM operations WHERE id=?", id).Scan(&phase, &detail); err != nil || phase != "FAILED" || detail != "SCOPED_PEER_V2_HISTORY_CAPACITY" {
				t.Fatal(phase, detail, err)
			}
			var n int
			if err = s.db.QueryRow("SELECT COUNT(*) FROM peer_grants WHERE status='PENDING' AND operation_id=?", id).Scan(&n); err != nil || n != 0 {
				t.Fatal(n, err)
			}
			if err = s.db.QueryRow("SELECT COUNT(*) FROM operation_resource_claims c JOIN operations o ON o.id=c.operation_id WHERE c.operation_id=? AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')", id).Scan(&n); err != nil || n != 0 {
				t.Fatal("failed original still holds live claims", n, err)
			}
		})
	}
}
