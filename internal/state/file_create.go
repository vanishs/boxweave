package state

import (
	"context"
	"errors"
	"time"
)

// FinalizeVerifiedGuestFileCreate commits a verified synchronous SFTP create
// directly from DISPATCHING to SUCCEEDED. SFTP has NO native Operation ID:
// never invent one or route the effect through NativeAck. The original intent
// already stores only a digest/size, not raw source bytes.
func (s *Store) FinalizeVerifiedGuestFileCreate(ctx context.Context, operationID string, box OwnedBox) error {
	if operationID == "" || box.UUID == "" || box.Project == "" ||
		box.Name == "" || box.Incarnation == "" || box.State != "OWNED" {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UnixNano()
	res, err := tx.ExecContext(ctx, `UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=?
		WHERE id=? AND action='box_file_apply' AND state='DISPATCHING'
		AND layer=? AND box_uuid=? AND incarnation=? AND resource_key=?
		AND native_operation_id=''
		AND EXISTS (SELECT 1 FROM owned_boxes WHERE uuid=? AND project=?
		AND name=? AND incarnation=? AND state='OWNED')`,
		now, operationID, box.Project, box.UUID, box.Incarnation, "box:"+box.Name,
		box.UUID, box.Project, box.Name, box.Incarnation)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO operation_events
		(operation_id,at,state,safe_detail) VALUES (?,?,?,?)`,
		operationID, now, "SUCCEEDED", "VERIFIED_SFTP_EXCLUSIVE_CREATE_NO_NATIVE_OPERATION"); err != nil {
		return err
	}
	return tx.Commit()
}

// SubmitGuestFileCreate uses the original durable Claim and resource lock.
// It cannot use the ordinary Worker because native SFTP has no asynchronous
// native operation receipt. The closure must independently verify the exact
// created bytes, and any uncertain error becomes UNKNOWN without redispatch.
func (w Worker) SubmitGuestFileCreate(id string, box OwnedBox, effect func(context.Context) error) error {
	if w.Store == nil || effect == nil {
		return ErrInvalid
	}
	rec, err := w.Store.Get(context.Background(), id)
	if err != nil {
		return err
	}
	if rec.State != "ADMITTED" || rec.Action != "box_file_apply" ||
		rec.BoxUUID != box.UUID || rec.Incarnation != box.Incarnation {
		return ErrState
	}
	go func() {
		budget := w.Budget
		if budget <= 0 {
			budget = 2 * time.Minute
		}
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		defer cancel()
		if err := w.Store.Claim(ctx, id); err != nil {
			return
		}
		if err := effect(ctx); err != nil {
			var denied PreDispatchError
			if errors.As(err, &denied) && denied.Class != "" {
				// Only proven exclusive-target-exists/no native write
				// is a terminal failure. Never infer this from a timeout.
				_ = w.Store.FailPreDispatch(context.Background(), id, denied.Class)
				return
			}
			// A lost-response/partial-write is an unknown native effect.
			reason := "SFTP_CREATE_OUTCOME_UNVERIFIED"
			_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", reason)
			return
		}
		if err := w.Store.FinalizeVerifiedGuestFileCreate(context.Background(), id, box); err != nil {
			_ = w.Store.Unknown(context.Background(), id, "DISPATCHING", "SFTP_FILE_FINAL_RECEIPT_UNVERIFIED")
		}
	}()
	return nil
}
