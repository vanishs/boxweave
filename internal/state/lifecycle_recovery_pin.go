package state

import (
	"context"
	"database/sql"
	"errors"
)

// readLifecycleRecoveryPin attaches the same durable static tuple required
// by ordinary lifecycle verification. Missing modern pins and inconsistent
// fences are not permission to fall back to a generic private NIC check.
func readLifecycleRecoveryPin(ctx context.Context, q credentialRowQuery, t RecoveryTarget) (VerifiedCreateBinding, string, error) {
	var pin VerifiedCreateBinding
	var bindings, fences, originals int
	e := q.QueryRowContext(ctx, `SELECT
	 (SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=?),
	 (SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=?),
	 (SELECT COUNT(*) FROM operations WHERE layer=? AND action='box_create' AND box_uuid=? AND incarnation=?)`, t.Layer, t.UUID, t.Layer, t.UUID, t.Layer, t.UUID, t.Incarnation).Scan(&bindings, &fences, &originals)
	if e != nil {
		return pin, "", e
	}
	if originals > 1 {
		return pin, "CONFLICT", nil
	}
	modern := false
	var originalID, originalState, originalResource string
	if originals == 1 {
		var intent string
		var staged bool
		e = q.QueryRowContext(ctx, `SELECT o.id,o.state,o.resource_key,o.safe_intent,EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.resource_key LIKE 'create-pin:%') FROM operations o WHERE o.layer=? AND o.action='box_create' AND o.box_uuid=? AND o.incarnation=?`, t.Layer, t.UUID, t.Incarnation).Scan(&originalID, &originalState, &originalResource, &intent, &staged)
		if e != nil {
			return pin, "", e
		}
		if originalResource != t.Resource {
			return pin, "CONFLICT", nil
		}
		modern = staged || requiresVerifiedCreateBinding(intent)
	}
	if bindings == 0 && fences == 0 {
		if modern {
			return pin, "MISSING", nil
		}
		return pin, "LEGACY_UNPINNED", nil
	}
	if bindings == 0 || fences < 2 {
		return pin, "MISSING", nil
	}
	if bindings != 1 || fences != 2 {
		return pin, "CONFLICT", nil
	}
	e = q.QueryRowContext(ctx, `SELECT bridge,acl,mac,ipv4,ipv6 FROM network_bindings WHERE project=? AND box_uuid=? AND incarnation=? AND nic='eth0' AND state='PINNED'`, t.Layer, t.UUID, t.Incarnation).Scan(&pin.Bridge, &pin.ACL, &pin.MAC, &pin.IPv4, &pin.IPv6)
	if errors.Is(e, sql.ErrNoRows) {
		return VerifiedCreateBinding{}, "CONFLICT", nil
	}
	if e != nil {
		return VerifiedCreateBinding{}, "", e
	}
	if !pin.valid() {
		return VerifiedCreateBinding{}, "CONFLICT", nil
	}
	var exact int
	e = q.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND bridge=? AND state='RESERVED' AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?))`, t.Layer, t.UUID, t.Incarnation, pin.Bridge, pin.IPv4, pin.IPv6).Scan(&exact)
	if e != nil {
		return VerifiedCreateBinding{}, "", e
	}
	if exact != 2 {
		return VerifiedCreateBinding{}, "CONFLICT", nil
	}
	if modern {
		if originalState != "SUCCEEDED" {
			return VerifiedCreateBinding{}, "CONFLICT", nil
		}
		// Native ACK lost Create is valid only with its exact linked
		// recovery receipt and audit; an edited state bit alone is not
		// provenance sufficient for native lifecycle or Delete.
		var lineage int
		e = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations c WHERE c.id=? AND c.state='SUCCEEDED' AND EXISTS(SELECT 1 FROM operation_events e WHERE e.operation_id=c.id AND e.state='SUCCEEDED') AND "+successfulCreateLineageEvidence, originalID).Scan(&lineage)
		if e != nil {
			return VerifiedCreateBinding{}, "", e
		}
		if lineage != 1 {
			return VerifiedCreateBinding{}, "CONFLICT", nil
		}
		var count int
		if e = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND resource_key LIKE 'create-pin:%'", originalID).Scan(&count); e != nil {
			return VerifiedCreateBinding{}, "", e
		}
		if count != 4 {
			return VerifiedCreateBinding{}, "CONFLICT", nil
		}
		for _, key := range pin.reservationKeys() {
			if e = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM operation_resource_claims WHERE operation_id=? AND layer=? AND resource_key=?", originalID, t.Layer, key).Scan(&count); e != nil {
				return VerifiedCreateBinding{}, "", e
			}
			if count != 1 {
				return VerifiedCreateBinding{}, "CONFLICT", nil
			}
		}
		if e = q.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND operation_id=?", t.Layer, t.UUID, t.Incarnation, originalID).Scan(&count); e != nil {
			return VerifiedCreateBinding{}, "", e
		}
		if count != 2 {
			return VerifiedCreateBinding{}, "CONFLICT", nil
		}
	}
	return pin, "PINNED", nil
}
