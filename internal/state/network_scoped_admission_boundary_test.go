package state

import (
	"errors"
	"testing"
)

// Admission must reject malformed or unowned endpoints without recording
// a new peer operation, grant, or resource claim. Fixtures are private SQLite.
func TestScopedPeerAdmissionBoundaryNoEffect(t *testing.T) {
	for _, tc := range []struct {
		name, src, dst, fence string
		want                  error
	}{
		{"same", purePeerUUID(1), purePeerUUID(1), "", ErrInvalid},
		{"missing-target", purePeerUUID(1), purePeerUUID(99), "", ErrResourceLocked},
		{"missing-source", purePeerUUID(99), purePeerUUID(2), "", ErrResourceLocked},
		{"empty-source", "", purePeerUUID(2), "", ErrResourceLocked},
		{"empty-target", purePeerUUID(1), "", "", ErrResourceLocked},
		{"quarantined-target-ipv4", purePeerUUID(1), purePeerUUID(2), "ipv4", ErrResourceLocked},
		{"quarantined-target-ipv6", purePeerUUID(1), purePeerUUID(2), "ipv6", ErrResourceLocked},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _ := n3ScopedOwnedFixture(t)
			ctx := t.Context()
			token := purePeerNewToken(t, s)
			ledger, err := s.ReadCompleteScopedNetworkAuditLedger(ctx, purePeerProject, purePeerBridge)
			if err != nil || len(ledger.Boxes) != 3 {
				t.Fatalf("invalid fixture: boxes=%d err=%v", len(ledger.Boxes), err)
			}
			if tc.fence != "" {
				result, err := s.db.ExecContext(ctx, "UPDATE network_address_fences SET state='QUARANTINED' WHERE project=? AND box_uuid=? AND family=?", purePeerProject, tc.dst, tc.fence)
				if err != nil {
					t.Fatal(err)
				}
				if n, err := result.RowsAffected(); err != nil || n != 1 {
					t.Fatalf("fixture fence not changed: %d %v", n, err)
				}
			}
			counts := func() (out [3]int) {
				t.Helper()
				for i, table := range []string{"operations", "peer_grants", "operation_resource_claims"} {
					if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&out[i]); err != nil {
						t.Fatal(err)
					}
				}
				return out
			}
			before := counts()
			r, err := s.AdmitScopedPeerSet(ctx, Admission{
				Layer: purePeerProject, ActorID: token, TokenID: token,
				Key: "scoped-boundary-" + tc.name, RequestID: tc.name,
			}, tc.src, tc.dst, true)
			if !errors.Is(err, tc.want) || r.ID != "" {
				t.Fatalf("invalid pair admitted: id=%q err=%v want=%v", r.ID, err, tc.want)
			}
			if after := counts(); after != before {
				t.Fatalf("rejected pair wrote durable rows: before=%v after=%v", before, after)
			}
		})
	}
}
