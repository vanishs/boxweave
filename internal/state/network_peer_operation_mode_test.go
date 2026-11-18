package state

import (
	"context"
	"errors"
	"testing"
)

func TestS4B3PeerOriginalModeSelectorNeverAuthorizesWrongBranch(t *testing.T) {
	ctx := context.Background()
	grant, grantID := s4b3GrantFixture(t)
	if allow, err := grant.PeerOperationDirection(ctx, grantID, "ADMITTED"); err != nil || !allow {
		t.Fatalf("exact original grant direction denied: %v %v", allow, err)
	}
	if _, err := grant.PeerOperationDirection(ctx, grantID, "DISPATCHING"); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("grant mode accepted wrong state: %v", err)
	}
	if err := grant.StagePeerGrant(ctx, grantID); err != nil {
		t.Fatal(err)
	}
	if err := grant.Claim(ctx, grantID); err != nil {
		t.Fatal(err)
	}
	if allow, err := grant.PeerOperationDirection(ctx, grantID, "DISPATCHING"); err != nil || !allow {
		t.Fatalf("claimed grant direction changed: %v %v", allow, err)
	}
	if _, err := grant.db.Exec("UPDATE operations SET safe_intent='{}' WHERE id=?", grantID); err != nil {
		t.Fatal(err)
	}
	if _, err := grant.PeerOperationDirection(ctx, grantID, "DISPATCHING"); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("tampered original accepted for root mode: %v", err)
	}
	revoke, _, revokeID := s4b2RootOperationFixture(t)
	if allow, err := revoke.PeerOperationDirection(ctx, revokeID, "ADMITTED"); err != nil || allow {
		t.Fatalf("exact revoke direction denied: %v %v", allow, err)
	}
	if _, err := revoke.PeerOperationDirection(ctx, revokeID, "UNKNOWN"); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("unconfirmed revoke recovered: %v", err)
	}
	if _, err := revoke.PeerOperationDirection(ctx, revokeID, "invalid"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid direction phase accepted: %v", err)
	}
	t.Log("S4_B3_CONTROLLER_CLI_ORIGINAL_DIRECTION_STRICT_NONREPLAY=PASS")
}
