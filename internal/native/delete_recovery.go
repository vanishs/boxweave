package native

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/lxc/incus/v7/shared/api"
	"net/http"
)

type DeleteRecoveryObservation struct {
	Class       string         `json:"class"`
	Box         BoxObservation `json:"box"`
	NativeID    string         `json:"native_operation_id"`
	Fingerprint string         `json:"observation_digest,omitempty"`
}

// ObserveUnknownDelete is a read-only classifier. Inactivity/absence alone
// do not establish the causality of a missing or expired native receipt.
func (r Reader) ObserveUnknownDelete(ctx context.Context, name, uuid, id string) (DeleteRecoveryObservation, error) {
	out := DeleteRecoveryObservation{NativeID: id}
	b, e := r.ObserveOwnedBox(ctx, name, uuid)
	if e != nil {
		return out, e
	}
	out.Box = b
	if id == "" {
		out.Class = "NATIVE_ACK_MISSING"
		return out, nil
	}
	op, e := r.NativeOperation(ctx, id)
	if api.StatusErrorCheck(e, http.StatusNotFound) {
		if e = r.OriginalOperationInactive(ctx, id); e != nil {
			return out, e
		}
		out.Class = "NATIVE_RECEIPT_UNAVAILABLE"
		return out, nil
	}
	if e != nil {
		return out, e
	}
	if op.ID != id {
		return out, ErrUnverified
	}
	if !op.StatusCode.IsFinal() {
		out.Class = "NATIVE_ACTIVE"
		return out, nil
	}
	if op.StatusCode != api.Success {
		out.Class = "NATIVE_TERMINAL_FAILURE"
		return out, nil
	}
	if b.State != "ABSENT_UNCERTAIN" {
		out.Class = "NATIVE_SUCCESS_POSTSTATE_UNVERIFIED"
		return out, nil
	}
	if e = r.ConfirmDeleteAbsence(ctx, name); e != nil {
		return out, e
	}
	// The durable original ID was returned by the exact DeleteInstance
	// request. Require current native SUCCESS and independent absence.
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%d", r.Project, name, uuid, id, op.StatusCode)))
	out.Class = "ORIGINAL_TERMINAL_DELETE_VERIFIED"
	out.Fingerprint = "DELETE_SUCCESS:" + hex.EncodeToString(sum[:])
	return out, nil
}
