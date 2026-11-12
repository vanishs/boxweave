package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// This command is reachable only from the root-reviewed, restricted-UID
// recover-worker. It never asks Incus to create, update or delete anything.
func performUnknownCreateBinding(ctx context.Context, store *state.Store, a recoverArgs) error {
	original, err := store.CreateRecoveryInspect(ctx, a.Operation)
	if err != nil {
		return err
	}
	if original.Project != a.Project {
		return errors.New("foreign native Project: refuse")
	}
	reader := native.Reader{Socket: a.Socket, Project: a.Project}
	// Exact original admission marker is mandatory. Even a matching name and
	// UUID with the right ACL must not be adopted without this witness.
	box, err := reader.ObserveCreatedByOperation(ctx, original.Name, original.OperationID)
	if err != nil {
		return fmt.Errorf("original native create identity not proven: %w", err)
	}
	observed, err := reader.ObserveOwnedBox(ctx, original.Name, box.UUID)
	if err != nil {
		return fmt.Errorf("second native UUID/ETag readback failed: %w", err)
	}
	if observed.State != "PRESENT_UUID_CONFIRMED" {
		return errors.New("native state not currently verified present")
	}
	out := json.NewEncoder(os.Stdout)
	switch a.Action {
	case "create-inspect":
		return out.Encode(struct {
			Original    state.CreateRecoverySnapshot `json:"original"`
			Native      native.BoxObservation        `json:"native"`
			NativeState string                       `json:"native_state"`
			Warning     string                       `json:"warning"`
		}{original, observed, box.NativeState,
			"Binding does not prove original success; UNKNOWN remains locked and must be separately reviewed"})
	case "create-challenge":
		challenge, err := store.IssueCreateBindingChallenge(ctx, original, a.Reviewer, observed.Fingerprint)
		if err != nil {
			return err
		}
		return out.Encode(struct {
			Operation        string `json:"operation_id"`
			ExactUUID        string `json:"native_uuid"`
			Challenge        string `json:"challenge"`
			ExpiresInSeconds int    `json:"expires_in_seconds"`
		}{original.OperationID, box.UUID, challenge, 300})
	case "create-bind":
		if a.ConfirmUUID != box.UUID {
			return errors.New("original native UUID must be explicitly confirmed")
		}
		if err := store.BindUnknownCreatedBox(ctx, original, a.Reviewer,
			a.Challenge, box.UUID, observed.Fingerprint, a.Reason, a.Acknowledge); err != nil {
			return err
		}
		// The original remains UNKNOWN. Binding is not a release, retry,
		// automatic cleanup, forged native ACK, or claim of Create success.
		return out.Encode(struct {
			Operation     string `json:"operation_id"`
			NativeUUID    string `json:"native_uuid"`
			State         string `json:"original_state"`
			ClaimReleased bool   `json:"claim_released"`
			NativeRetried bool   `json:"native_retried"`
		}{original.OperationID, box.UUID, "UNKNOWN", false, false})
	}
	return errors.New("unsupported create recovery action")
}
