package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ReviewedRecoveryBinding is private trusted local-worker data. The
// public MCP API cannot set it. A digest is evidence, never authorization.
type ReviewedRecoveryBinding struct {
	ReviewSHA256, OriginalSHA256, LedgerSHA256, WitnessSHA256, Mode, Leaf string
	ExpiresNS                                                             int64
}

func (b ReviewedRecoveryBinding) valid() bool {
	return scopedPeerSHA(b.ReviewSHA256) && scopedPeerSHA(b.OriginalSHA256) && scopedPeerSHA(b.LedgerSHA256) && scopedPeerSHA(b.WitnessSHA256) && (b.Mode == "LEGACY" || b.Mode == "SCOPED_V2") && (b.Leaf == "MCP_SETTLE_ORIGINAL_NO_REPLAY" || b.Leaf == "MCP_BIND_ORIGINAL_IDENTITY" || b.Leaf == "MCP_REPAIR_ORIGINAL_PIN_LEDGER") && b.ExpiresNS > 0
}
func reviewDigestOf(r RecoveryRequest) string {
	if r.Review == nil {
		return ""
	}
	return r.Review.ReviewSHA256
}
func RecoveryWitnessDigest(t RecoveryTarget, p RecoveryProof) string {
	if p.Fingerprint == "" {
		return ""
	}
	b, e := json.Marshal(struct {
		Target RecoveryTarget
		Proof  RecoveryProof
	}{t, p})
	if e != nil {
		return ""
	}
	h := sha256.Sum256(append([]byte("boxweave/recover-plan-witness/v1\n"), b...))
	return hex.EncodeToString(h[:])
}

// Historical binding syntax remains readable for exact receipt lookup,
// but no historical leaf can authorize a new reviewed reconciliation.
func reviewedRecoveryLeaf(t RecoveryTarget, p RecoveryProof, mode string) string {
	if mode != "SCOPED_V2" || (t.Action == "box_peer_set" && !t.ScopedPeer) {
		return ""
	}
	if p.Kind == RecoverySettle {
		switch t.Action {
		case "box_start", "box_stop", "box_update", "box_exec", "box_peer_set", "port_forward_create", "port_forward_delete":
			return "MCP_SETTLE_ORIGINAL_NO_REPLAY"
		}
	}
	return ""
}
func checkReviewedRecoveryTx(ctx context.Context, tx *sql.Tx, r RecoveryRequest, t RecoveryTarget, p RecoveryProof) error {
	b := r.Review
	if b == nil || !b.valid() || r.LocalReviewer == "" || time.Now().UnixNano() >= b.ExpiresNS || reviewedRecoveryLeaf(t, p, b.Mode) != b.Leaf || RecoveryWitnessDigest(t, p) != b.WitnessSHA256 {
		return fmt.Errorf("reviewed recovery: transaction review or proof changed: %w", ErrConflict)
	}
	original, e := readDiagnosticOriginalTx(ctx, tx, r.Layer, r.OriginalID)
	if e != nil || original.Target != t || original.Digest != b.OriginalSHA256 || t.Released || !original.CredentialActive && t.Action == "box_peer_set" {
		return fmt.Errorf("reviewed recovery: transaction original changed: %w", ErrConflict)
	}
	l, e := readDiagnosticLedgerTx(ctx, tx, r.Layer, b.Mode, "incusbr-"+strings.TrimPrefix(r.Layer, "user-"))
	if e != nil || l.ForeignRows != 0 || l.ObservationDigest() != b.LedgerSHA256 {
		return fmt.Errorf("reviewed recovery: transaction ledger changed: %w", ErrConflict)
	}
	return ctx.Err()
}

// ReadReviewedRecoveryEvidence reads live source only on the restricted
// authorized execution path; never call from Doctor or the snapshot process.
func (s *Store) ReadReviewedRecoveryEvidence(ctx context.Context, project, id, mode string) (DiagnosticOriginal, DiagnosticLedger, error) {
	var o DiagnosticOriginal
	var l DiagnosticLedger
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return o, l, e
	}
	defer tx.Rollback()
	o, e = readDiagnosticOriginalTx(ctx, tx, project, id)
	if e != nil {
		return o, l, e
	}
	l, e = readDiagnosticLedgerTx(ctx, tx, project, mode, "incusbr-"+strings.TrimPrefix(project, "user-"))
	if e != nil {
		return o, l, e
	}
	return o, l, tx.Commit()
}

// ValidateReviewedRecoveryRequest is pure input validation for the local
// root handoff. It does not authorize a caller or touch SQLite.
func ValidateReviewedRecoveryRequest(r RecoveryRequest) error {
	if r.Review == nil || r.LocalReviewer == "" {
		return ErrInvalid
	}
	_, e := recoveryAdmission(r)
	return e
}
