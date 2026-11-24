package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// DiagnosticOriginal is private observation data. Never serialize Target,
// actor, token, or claim keys into the public plan; bind their canonical digest.
type DiagnosticOriginal struct {
	Target           RecoveryTarget `json:"-"`
	Digest           string         `json:"digest"`
	Claims           []string       `json:"-"`
	CredentialActive bool           `json:"credential_active"`
}

func (s *DiagnosticSnapshot) ReadOriginal(ctx context.Context, project, id string) (DiagnosticOriginal, error) {
	var out DiagnosticOriginal
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	t, e := readRecoveryTarget(ctx, tx, project, id)
	if e != nil {
		return out, e
	}
	var actor, token string
	if e = tx.QueryRowContext(ctx, "SELECT actor_id,token_id FROM operations WHERE id=? AND layer=?", id, project).Scan(&actor, &token); e != nil {
		return out, e
	}
	active, e := activeCredentialForActor(ctx, tx, actor, token)
	if e != nil {
		return out, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT layer,resource_key FROM operation_resource_claims WHERE operation_id=? ORDER BY layer,resource_key LIMIT 129", id)
	if e != nil {
		return out, e
	}
	var keys []string
	for rows.Next() {
		var layer, key string
		if e = rows.Scan(&layer, &key); e != nil {
			break
		}
		if layer != project || len(keys) >= 128 {
			e = ErrResourceLocked
			break
		}
		keys = append(keys, key)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return out, e
	}
	// Audit history and identity/claims are bound independently from the
	// observed native proof; no user-supplied review or actor is trusted.
	// Bind exact bounded event/audit content, not counts or raw output.
	history := sha256.New()
	for _, query := range []string{
		"SELECT seq,at,state,safe_detail,'' FROM operation_events WHERE operation_id=? ORDER BY seq LIMIT 1025",
		"SELECT seq,at,decision,evidence_digest,reviewer FROM recovery_audit WHERE operation_id=? ORDER BY seq LIMIT 1025",
	} {
		rows, e := tx.QueryContext(ctx, query, id)
		if e != nil {
			return out, e
		}
		count := 0
		for rows.Next() {
			var seq, at int64
			var a, b, c string
			if e = rows.Scan(&seq, &at, &a, &b, &c); e != nil {
				break
			}
			count++
			if count > 1024 {
				e = ErrResourceLocked
				break
			}
			raw, _ := json.Marshal([]any{seq, at, a, b, c})
			history.Write(raw)
			history.Write([]byte{'\n'})
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return out, e
		}
		history.Write([]byte{0})
	}
	b, e := json.Marshal(struct {
		Target           RecoveryTarget
		Actor, Token     string
		Claims           []string
		CredentialActive bool
		History          string
	}{t, actor, token, keys, active, hex.EncodeToString(history.Sum(nil))})
	if e != nil {
		return out, e
	}
	h := sha256.Sum256(append([]byte("boxweave/diagnostic-original/v1\n"), b...))
	if e = tx.Commit(); e != nil {
		return out, e
	}
	return DiagnosticOriginal{Target: t, Digest: hex.EncodeToString(h[:]), Claims: keys, CredentialActive: active}, nil
}

// These methods expose only query predicates on the private snapshot.
// The caller has no Store, transaction, admission or commit capability.
func (s *DiagnosticSnapshot) ReadRecoveryTarget(ctx context.Context, project, id string) (RecoveryTarget, error) {
	return s.store.ReadRecoveryTarget(ctx, project, id)
}
func (s *DiagnosticSnapshot) ExecGCReviewInspect(ctx context.Context, id, stream string) (ExecGCReview, error) {
	return s.store.ExecGCReviewInspect(ctx, id, stream)
}
func (s *DiagnosticSnapshot) CheckDependentDeleteRecovery(ctx context.Context, t RecoveryTarget) error {
	return s.store.CheckDependentDeleteRecovery(ctx, t)
}
func (s *DiagnosticSnapshot) CheckRecoveryCreateBinding(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	return s.store.CheckRecoveryCreateBinding(ctx, t, p)
}
func (s *DiagnosticSnapshot) CreatePinRepairStatus(ctx context.Context, t RecoveryTarget) (string, error) {
	return s.store.CreatePinRepairStatus(ctx, t)
}
func (s *DiagnosticSnapshot) ReadOriginalCreatePin(ctx context.Context, t RecoveryTarget) (VerifiedCreateBinding, error) {
	return s.store.ReadOriginalCreatePin(ctx, t)
}
func (s *DiagnosticSnapshot) CheckCreatePinRepair(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	return s.store.CheckCreatePinRepair(ctx, t, p)
}

// CheckPlanRecoveryProof reuses writer predicates without executing any
// finalizer, admission, challenge, or audit write, even on the copy.
func (s *DiagnosticSnapshot) CheckPlanRecoveryProof(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	current, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil || current != t {
		return ErrConflict
	}
	if e = requireNoIPv4Compaction(ctx, tx, t.Layer); e != nil {
		return e
	}
	if p.Fingerprint == "" {
		return ErrResourceLocked
	}
	switch {
	case p.Kind == RecoveryRepairPin:
		e = checkCreatePinRepairTx(ctx, tx, t, p)
	case p.Kind == RecoverySettle && t.Action == "box_peer_set":
		if t.ScopedPeer {
			r := p.ScopedPeer
			if t.State != "UNKNOWN" || t.Released || r.OriginalID != t.ID || r.Project != t.Layer || !r.RecoveryOnly || r.Status != "COMPLETE_VERIFIED" || r.VerifiedSteps != 4 || !scopedPeerSHA(r.ProofSHA256) || !scopedPeerSHA(r.PlanSHA256) || !scopedPeerSHA(r.FinalSHA256) {
				return ErrResourceLocked
			}
			proof, err := readScopedPeerOriginalTx(ctx, tx, t.ID, r.Project, r.Bridge, "UNKNOWN")
			if err != nil || ScopedPeerProofSHA256(proof) != r.ProofSHA256 {
				return ErrResourceLocked
			}
		} else {
			e = checkRecoveryPeerTx(ctx, tx, t, p)
		}
	case p.Kind == RecoverySettle || p.Kind == RecoveryBind:
		parent, err := checkGenericRecoveryScopeTx(ctx, tx, t, "")
		if err != nil {
			return err
		}
		if p.Kind == RecoveryBind {
			if t.Action != "box_create" || t.UUID != "" || t.Incarnation != "" || !validActivityBoxID(p.UUID) || p.Reason != "CREATE_MARKER_EXACT_IDENTITY" {
				return ErrResourceLocked
			}
		} else if t.Action == "box_create" && t.UUID == "" && t.Incarnation == "" {
			e = checkRecoveryCreateTx(ctx, tx, t, p)
		} else {
			if t.OwnedState != "OWNED" || t.UUID == "" || t.Incarnation == "" || p.UUID != t.UUID || t.NativeID == "" || p.NativeID != t.NativeID || p.Reason != "ORIGINAL_TERMINAL_POSTSTATE_VERIFIED" {
				return ErrResourceLocked
			}
			switch t.Action {
			case "port_forward_create", "port_forward_delete":
				e = checkRecoveryForwardTx(ctx, tx, t, p, parent)
			case "box_exec":
				if p.Exit == nil {
					return ErrResourceLocked
				}
			case "box_start", "box_stop", "box_update":
				if t.LifecyclePinStatus != "PINNED" && t.LifecyclePinStatus != "LEGACY_UNPINNED" || p.LifecycleBinding != t.LifecycleBinding {
					return ErrResourceLocked
				}
				if parent != "" {
					e = recoveryClaimKeys(ctx, tx, t, []string{t.Resource, deleteParentKey(parent)})
				}
			case "box_delete":
				if !strings.HasPrefix(p.Fingerprint, "DELETE_SUCCESS:") {
					return ErrResourceLocked
				}
			default:
				return ErrResourceLocked
			}
		}
	default:
		return ErrResourceLocked
	}
	if e != nil {
		return e
	}
	return tx.Commit()
}
