package state

import (
	"context"
	"database/sql"
	"encoding/json"
)

// ReviewedRecoveryPostcheck is an independent durable read, not a fresh
// native/packet witness. A later original change must not undo a receipt.
type ReviewedRecoveryPostcheck struct {
	State              string `json:"state"`
	OriginalState      string `json:"original_state,omitempty"`
	OriginalBox        string `json:"original_box,omitempty"`
	OriginalProtection string `json:"original_protection,omitempty"`
}

func (s *Store) ReadReviewedRecoveryPostcheck(ctx context.Context, r RecoveryRequest, rec RecoveryRecord) (ReviewedRecoveryPostcheck, error) {
	out := ReviewedRecoveryPostcheck{State: "UNVERIFIED"}
	a, e := recoveryAdmission(r)
	if e != nil || r.Review == nil {
		return out, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if e = recoveryAuthorized(ctx, tx, r, a); e != nil {
		return out, e
	}
	replay, e := readRecoveryReplay(ctx, tx, r, a)
	if e != nil || replay.Receipt.ID != rec.Receipt.ID || replay.OriginalID != r.OriginalID || replay.Disposition != rec.Disposition || replay.Receipt.State != "SUCCEEDED" {
		return out, ErrConflict
	}
	t, e := readRecoveryTarget(ctx, tx, r.Layer, r.OriginalID)
	if e != nil {
		return out, e
	}
	out.OriginalState = t.State
	out.OriginalBox = SafeDiagnosticRef(t.UUID)
	out.OriginalProtection = "NOT_UNKNOWN"
	if t.State == "UNKNOWN" {
		out.OriginalProtection = "RETAINED"
		if t.Released {
			out.OriginalProtection = "REVIEWED_RELEASED"
		}
	}
	out.State = "DURABLE_ORIGINAL_POSTSTATE_VERIFIED"
	if rec.Disposition == "IDENTITY_BOUND" {
		if t.State != "UNKNOWN" || t.Released || t.UUID != rec.Receipt.BoxUUID || t.Incarnation != rec.Receipt.Incarnation {
			out.State = "CURRENT_ORIGINAL_CHANGED"
		}
	} else if rec.Disposition != "SETTLED" || t.State != "SUCCEEDED" {
		out.State = "CURRENT_ORIGINAL_CHANGED"
	}
	if r.Review.Leaf == "MCP_REPAIR_ORIGINAL_PIN_LEDGER" {
		status, err := createPinRepairStatusTx(ctx, tx, t)
		if err != nil || status != "COMPLETE" {
			out.State = "CURRENT_ORIGINAL_CHANGED"
		}
	}
	// Validate the leaf's durable postcondition, not merely a terminal bit.
	var audits int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM recovery_audit WHERE operation_id=? AND reviewer=? AND decision=?", t.ID, a.ActorID, "RECOVERY_"+rec.Disposition).Scan(&audits); e != nil || audits < 1 {
		return out, ErrConflict
	}
	if t.Action == "box_peer_set" && out.State == "DURABLE_ORIGINAL_POSTSTATE_VERIFIED" {
		var in ScopedPeerOriginalIntent
		if json.Unmarshal([]byte(t.SafeIntent), &in) != nil || in.Project != t.Layer || in.SourceUUID != t.UUID || in.SourceIncarnation != t.Incarnation {
			out.State = "CURRENT_ORIGINAL_CHANGED"
		} else {
			status, op, desired := "REVOKED", t.ID, 0
			if in.Allow {
				status, op, desired = "CONFIRMED", "", 1
			}
			var n int
			e = tx.QueryRowContext(ctx, "SELECT count(*) FROM peer_grants WHERE project=? AND source_uuid=? AND source_incarnation=? AND target_uuid=? AND target_incarnation=? AND desired=? AND status=? AND operation_id=? AND last_verified_operation_id=?", t.Layer, in.SourceUUID, in.SourceIncarnation, in.TargetUUID, in.TargetIncarnation, desired, status, op, t.ID).Scan(&n)
			if e != nil || n != 1 {
				out.State = "CURRENT_ORIGINAL_CHANGED"
			}
		}
	}
	if t.Action == "box_delete" && t.OwnedState != "DELETED" {
		out.State = "CURRENT_ORIGINAL_CHANGED"
	}
	if t.Action == "box_create" && t.State == "SUCCEEDED" && t.CreatePinRequired && t.CreatePinStaged {
		st, e := createPinRepairStatusTx(ctx, tx, t)
		if e != nil || st != "COMPLETE" {
			out.State = "CURRENT_ORIGINAL_CHANGED"
		}
	}
	return out, tx.Commit()
}
func (s *DiagnosticSnapshot) ReplayReviewedRecovery(ctx context.Context, r RecoveryRequest) (RecoveryRecord, error) {
	if r.Review == nil || r.LocalReviewer == "" {
		return RecoveryRecord{}, ErrInvalid
	}
	return s.store.ReplayRecovery(ctx, r)
}
func (s *DiagnosticSnapshot) ReadReviewedRecoveryPostcheck(ctx context.Context, r RecoveryRequest, rec RecoveryRecord) (ReviewedRecoveryPostcheck, error) {
	return s.store.ReadReviewedRecoveryPostcheck(ctx, r, rec)
}
