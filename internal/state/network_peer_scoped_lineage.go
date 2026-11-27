package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
)

// scopedPeerVerifyHistoricalOperationTx verifies a completed grant/revoke
// against its exact immutable original, resource claims and terminal event.
// Only canonical Scoped originals are accepted. A database proof is not a native kernel proof.
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
	var scoped ScopedPeerOriginalIntent
	if json.Unmarshal(raw, &scoped) != nil {
		return ErrResourceLocked
	}
	canonical, e := json.Marshal(scoped)
	if e != nil || !bytes.Equal(canonical, raw) || scoped.Schema != 2 || scoped.Allow != allow || scoped.Project != project || scoped.Bridge != bridge || scoped.SourceUUID != p.SourceUUID || scoped.TargetUUID != p.TargetUUID || scoped.SourceIncarnation != p.SourceIncarnation || scoped.TargetIncarnation != p.TargetIncarnation || resource != "peer-guard:"+bridge {
		return ErrResourceLocked
	}
	expectedClaims, _, e := scopedPeerParentClaims(ctx, tx, verifiedID, scoped, "SUCCEEDED")
	if e != nil {
		return ErrResourceLocked
	}
	slices.Sort(expectedClaims)
	rows, err := tx.QueryContext(ctx, `SELECT resource_key,layer FROM operation_resource_claims
  WHERE operation_id=? ORDER BY resource_key`, verifiedID)
	if err != nil {
		return err
	}
	var claims []string
	for rows.Next() {
		var claim, layer string
		if err = rows.Scan(&claim, &layer); err != nil || layer != project {
			err = ErrResourceLocked
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

	proposed := "SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED"
	if !allow {
		proposed = "SCOPED_PEER_V2_REVOCATION_KERNEL_VERIFIED"
	}
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail=?`, verifiedID, proposed).Scan(&count)
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
