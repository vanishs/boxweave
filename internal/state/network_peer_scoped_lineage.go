package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

// scopedPeerVerifyHistoricalOperationTx is a READ-ONLY immutable lineage
// validator for a completed grant/revoke that predates a v2 ACL migration.
// The old original's shared ACL can differ from the current per-Box ACLs,
// but it must have exact canonical intent, three old resource claims, a
// successful atomic peer completion and its exact kernel-verified event.
// V2 history is admitted only when its own future root completion event
// exists; no v2 terminal/root operation writer currently emits that event.
// Do NOT treat this as native kernel proof or permission to migrate ACLs.
func scopedPeerVerifyHistoricalOperationTx(ctx context.Context, tx *sql.Tx,
	project, bridge string, p PeerGrantBinding, verifiedID string, allow bool) error {
	if tx == nil || !validHexID(verifiedID) || p.SourceUUID == p.TargetUUID ||
		p.SourceUUID == "" || p.TargetUUID == "" {
		return ErrResourceLocked
	}
	var action, status, actor, token, layer, uuid, inc, resource string
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,
  box_uuid,incarnation,resource_key,safe_intent FROM operations WHERE id=?`, verifiedID).
		Scan(&action, &status, &actor, &token, &layer, &uuid, &inc, &resource, &raw)
	if err != nil || action != "box_peer_set" || status != "SUCCEEDED" ||
		actor == "" || token == "" || layer != project ||
		uuid != p.SourceUUID || inc != p.SourceIncarnation {
		return ErrResourceLocked
	}
	var expectedClaims []string
	var old RootPeerRevokeSafeIntent
	if json.Unmarshal(raw, &old) == nil {
		canonical, e := json.Marshal(old)
		if e == nil && bytes.Equal(canonical, raw) &&
			strings.HasPrefix(old.ACL, "bw-") && auditName(old.ACL) &&
			old.Project == project && old.Bridge == bridge && old.Allow == allow &&
			old.SourceUUID == p.SourceUUID && old.TargetUUID == p.TargetUUID &&
			old.SourceIncarnation == p.SourceIncarnation &&
			old.TargetIncarnation == p.TargetIncarnation &&
			resource == "acl:default/"+old.ACL {
			expectedClaims = []string{"acl:default/" + old.ACL,
				"box:" + p.SourceUUID, "box:" + p.TargetUUID}
		}
	}
	if len(expectedClaims) == 0 {
		var scoped ScopedPeerOriginalIntent
		if json.Unmarshal(raw, &scoped) != nil {
			return ErrResourceLocked
		}
		canonical, e := json.Marshal(scoped)
		if e != nil || !bytes.Equal(canonical, raw) ||
			scoped.Schema != 2 || scoped.Allow != allow ||
			scoped.Project != project || scoped.Bridge != bridge ||
			scoped.SourceUUID != p.SourceUUID || scoped.TargetUUID != p.TargetUUID ||
			scoped.SourceIncarnation != p.SourceIncarnation ||
			scoped.TargetIncarnation != p.TargetIncarnation ||
			resource != "peer-guard:"+bridge {
			return ErrResourceLocked
		}
		expectedClaims, e = scopedPeerResourceClaims(scoped)
		if e != nil {
			return ErrResourceLocked
		}
	}
	slices.Sort(expectedClaims)
	rows, err := tx.QueryContext(ctx, `SELECT resource_key FROM operation_resource_claims
  WHERE operation_id=? ORDER BY resource_key`, verifiedID)
	if err != nil {
		return err
	}
	var claims []string
	for rows.Next() {
		var claim string
		if err = rows.Scan(&claim); err != nil {
			break
		}
		claims = append(claims, claim)
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if !slices.Equal(claims, expectedClaims) {
		return ErrResourceLocked
	}

	// v1 completion and UNKNOWN recovery use distinct immutable event labels.
	// New v2 labels are RESERVATIONS only: no production dispatcher may claim
	// either until kernel/Incus/conntrack durability is independently proven.
	var event string
	if allow {
		event = "PEER_ROOT_SCOPED_GRANT_KERNEL_VERIFIED"
	} else {
		event = "PEER_ROOT_SCOPED_REVOCATION_VERIFIED"
	}
	alt := "PEER_ROOT_COMPLETED_UNKNOWN_GRANT_READONLY_NO_REPLAY"
	if !allow {
		alt = "ROOT_COMPLETED_UNKNOWN_RECONCILED_NO_REPLAY"
	}
	proposed := "SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED"
	if !allow {
		proposed = "SCOPED_PEER_V2_REVOCATION_KERNEL_VERIFIED"
	}
	var count int
	if len(expectedClaims) == 3 {
		// The legacy shared-ACL original MUST have a legacy native
		// completion/reconciliation event; no v2 event alias is allowed.
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
  WHERE operation_id=? AND state='SUCCEEDED'
  AND safe_detail IN (?,?)`, verifiedID, event, alt).Scan(&count)
	} else {
		// This future v2 event is not yet emitted by any installed
		// worker; a legacy success can never complete a scoped original.
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
  WHERE operation_id=? AND state='SUCCEEDED'
  AND safe_detail=?`, verifiedID, proposed).Scan(&count)
	}
	if err != nil || count != 1 {
		return ErrResourceLocked
	}
	return nil
}

// The active/durable historical row must have precisely the shape emitted
// by old Root finalization; staging never changes last_verified until native
// kernel reconciliation is independently complete.
func scopedPeerVerifyTerminalLineageTx(ctx context.Context, tx *sql.Tx,
	project, bridge string, p PeerGrantBinding, desired int, status, operationID, verifiedID string) error {
	if !validHexID(verifiedID) {
		return ErrResourceLocked
	}
	if desired == 1 && status == "CONFIRMED" {
		if operationID != "" {
			return ErrResourceLocked
		}
		return scopedPeerVerifyHistoricalOperationTx(ctx, tx, project, bridge, p, verifiedID, true)
	}
	if desired == 0 && status == "REVOKED" {
		if operationID != verifiedID {
			return ErrResourceLocked
		}
		return scopedPeerVerifyHistoricalOperationTx(ctx, tx, project, bridge, p, verifiedID, false)
	}
	return errors.New("unrecognized final Peer lineage")
}
