package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"os"
)

// Only reachable from the approved local root credential-drop path.
// No native mutation or automatic redispatch is possible.
func performDeleteRecovery(ctx context.Context, store *state.Store, a recoverArgs) error {
	r, e := store.RecoveryInspect(ctx, a.Operation)
	if e != nil {
		return e
	}
	if r.Action != "box_delete" || r.Project != a.Project || r.Released {
		return errors.New("unreviewable original Delete")
	}
	obs, e := (native.Reader{Socket: a.Socket, Project: a.Project}).ObserveUnknownDelete(ctx, r.Name, r.BoxUUID, r.NativeOperationID)
	if e != nil {
		return e
	}
	out := json.NewEncoder(os.Stdout)
	if a.Action == "delete-inspect" {
		return out.Encode(struct {
			Original state.RecoverySnapshot           `json:"original"`
			Native   native.DeleteRecoveryObservation `json:"native"`
		}{r, obs})
	}
	if obs.Class != "ORIGINAL_TERMINAL_DELETE_VERIFIED" {
		return errors.New("original successful native Delete receipt and exact absence required; UNKNOWN retained")
	}
	if a.Action == "delete-challenge" {
		c, e := store.IssueRecoveryChallenge(ctx, r, a.Reviewer, obs.Fingerprint)
		if e != nil {
			return e
		}
		return out.Encode(map[string]any{"operation_id": r.OperationID, "challenge": c, "expires_in_seconds": 300})
	}
	if a.Action == "delete-confirm" {
		if e = store.FinalizeReviewedUnknownDelete(ctx, r, a.Reviewer, a.Challenge, a.ConfirmUUID, a.Reason, obs.Fingerprint, a.Acknowledge); e != nil {
			return e
		}
		return out.Encode(map[string]any{"operation_id": r.OperationID, "state": "SUCCEEDED", "native_retried": false})
	}
	return errors.New("invalid Delete recovery action")
}
