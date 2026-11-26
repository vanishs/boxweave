package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/vanishs/boxweave/internal/peerscope"
)

// ScopedLifecycleRecord is the immutable protected Root journal record.
// It is never accepted as public tool input.
type ScopedLifecycleRecord struct {
	Schema                                                                                       int
	Kind, Status, OriginalID, NativeID, HostSHA256, GuardSHA256, Project, Name, UUID, PlanSHA256 string
	Binding                                                                                      VerifiedCreateBinding
	Start, Internet, BootAutostart                                                               bool
}

// A recovery witness is separately selected, read-only Root evidence bound
// to the original ledger snapshot. A completed effect reply is not this proof.
type ScopedLifecycleRecoveryProof struct {
	Record         ScopedLifecycleRecord
	OriginalSHA256 string
	RecoveryOnly   bool
}

func ScopedLifecycleOriginalSHA256(p any) string {
	raw, e := json.Marshal(p)
	if e != nil {
		return ""
	}
	return digest("boxweave/scoped-lifecycle-recovery/v2\n" + string(raw))
}

func checkScopedLifecycleRecoveryTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if !t.ScopedLifecycle || p.Kind != RecoverySettle || p.Reason != "SCOPED_LIFECYCLE_COMPLETED_NO_REPLAY" || t.State != "UNKNOWN" || t.Released || t.NativeID == "" || p.NativeID != t.NativeID || p.Fingerprint == "" {
		return ErrResourceLocked
	}
	if _, e := checkGenericRecoveryScopeTx(ctx, tx, t, ""); e != nil {
		return e
	}
	w := p.ScopedLifecycle
	r := w.Record
	if !w.RecoveryOnly || !scopedPeerSHA(w.OriginalSHA256) || !scopedPeerSHA(r.PlanSHA256) || !scopedPeerSHA(r.HostSHA256) || !scopedPeerSHA(r.GuardSHA256) || r.Schema != 2 || r.Status != "VERIFIED" || r.OriginalID != t.ID || r.NativeID != t.NativeID || r.Project != t.Layer || r.Name != t.Name || p.UUID != r.UUID || !validActivityBoxID(r.UUID) || !r.Binding.valid() {
		return ErrResourceLocked
	}
	acl, e := peerscope.ACLName(r.Project, r.Binding.Bridge, r.UUID)
	if e != nil || acl != r.Binding.ACL {
		return ErrResourceLocked
	}
	switch t.Action {
	case "box_create":
		if r.Kind != "CREATE" || t.UUID != "" || t.Incarnation != "" || !t.CreatePinStaged || p.CreateBinding != r.Binding {
			return ErrResourceLocked
		}
		orig, e := readScopedCreateOriginalTxPhase(ctx, tx, t.ID, t.Layer, r.Binding.Bridge, "UNKNOWN")
		if e != nil || ScopedLifecycleOriginalSHA256(orig) != w.OriginalSHA256 {
			return ErrResourceLocked
		}
		want := orig.Quarantine
		want.ACL = acl
		if want != r.Binding || orig.Intent.Name != r.Name || orig.Intent.Start != r.Start || orig.Intent.Internet != r.Internet || orig.Intent.BootAutostart != r.BootAutostart {
			return ErrResourceLocked
		}
	case "box_delete":
		if r.Kind != "DELETE" || !t.DependentDelete || t.UUID != r.UUID || t.Incarnation != r.UUID || t.OwnedState != "OWNED" || r.Start || r.Internet || r.BootAutostart {
			return ErrResourceLocked
		}
		orig, e := readScopedDeleteOriginalTxPhase(ctx, tx, t.ID, t.Layer, r.Binding.Bridge, "UNKNOWN")
		if e != nil || ScopedLifecycleOriginalSHA256(orig) != w.OriginalSHA256 || orig.Binding != r.Binding {
			return ErrResourceLocked
		}
	default:
		return ErrResourceLocked
	}
	return nil
}

func (s *Store) CheckScopedLifecycleRecovery(ctx context.Context, t RecoveryTarget, p RecoveryProof) error {
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now, e := readRecoveryTarget(ctx, tx, t.Layer, t.ID)
	if e != nil || now != t {
		return ErrConflict
	}
	if e = checkScopedLifecycleRecoveryTx(ctx, tx, t, p); e != nil {
		return e
	}
	return tx.Commit()
}

func finishScopedLifecycleRecoveryTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if e := checkScopedLifecycleRecoveryTx(ctx, tx, t, p); e != nil {
		return e
	}
	if t.Action == "box_create" {
		if e := exchangeScopedCreateACLPhase(ctx, tx, t.ID, t.Layer, t.Name, p.UUID, p.CreateBinding, "UNKNOWN"); e != nil {
			return e
		}
		return settleRecoveredCreateTx(ctx, tx, t, p)
	}
	return finishVerifiedDeleteTxMode(ctx, tx, t.ID, OwnedBox{UUID: t.UUID, Project: t.Layer, Name: t.Name, Incarnation: t.Incarnation, State: "OWNED"}, "UNKNOWN", "RECOVERY_SCOPED_DELETE_ACL_ABSENT_NO_REPLAY", true)
}
