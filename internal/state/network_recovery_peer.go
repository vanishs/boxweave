package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

type RecoveryPeerClaim struct {
	Intent       RootPeerRevokeSafeIntent
	Binding      PeerGrantBinding
	Actor, Token string
}

func readRecoveryPeer(ctx context.Context, q credentialRowQuery, t RecoveryTarget) (RecoveryPeerClaim, bool, error) {
	var p RecoveryPeerClaim
	var zero RecoveryPeerClaim
	if json.Unmarshal([]byte(t.SafeIntent), &p.Intent) != nil {
		return zero, false, nil
	}
	in := p.Intent
	raw, e := json.Marshal(in)
	if e != nil || string(raw) != t.SafeIntent || in.Project != t.Layer || in.SourceUUID != t.UUID || in.SourceIncarnation != t.Incarnation || t.Resource != "acl:default/"+in.ACL || in.SourceUUID == in.TargetUUID || !auditName(in.Bridge) || !auditName(in.ACL) || in.SourceUUID == "" || in.TargetUUID == "" || in.TargetIncarnation == "" || t.OwnedState != "OWNED" {
		return zero, false, nil
	}
	var n int
	if e = q.QueryRowContext(ctx, "SELECT count(*) FROM peer_grants WHERE operation_id=?", t.ID).Scan(&n); e != nil {
		return zero, false, e
	}
	if n != 1 {
		return zero, false, nil
	}
	var desired int
	var status string
	b := &p.Binding
	e = q.QueryRowContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,target_incarnation,source_ipv4,target_ipv4,source_ipv6,target_ipv6,desired,status FROM peer_grants WHERE project=? AND operation_id=?`, t.Layer, t.ID).Scan(&b.SourceUUID, &b.SourceIncarnation, &b.TargetUUID, &b.TargetIncarnation, &b.SourceIPv4, &b.TargetIPv4, &b.SourceIPv6, &b.TargetIPv6, &desired, &status)
	if errors.Is(e, sql.ErrNoRows) {
		return zero, false, nil
	}
	if e != nil {
		return zero, false, e
	}
	if status != "PENDING" || (desired != 0 && desired != 1) || in.Allow != (desired == 1) || b.SourceUUID != in.SourceUUID || b.TargetUUID != in.TargetUUID || b.SourceIncarnation != in.SourceIncarnation || b.TargetIncarnation != in.TargetIncarnation {
		return zero, false, nil
	}
	if e = q.QueryRowContext(ctx, "SELECT actor_id,token_id FROM operations WHERE id=? AND layer=?", t.ID, t.Layer).Scan(&p.Actor, &p.Token); e != nil {
		return zero, false, e
	}
	// Repeat the relevant current protected network identity before settlement.
	// The root witness has already checked the complete Project and kernel.
	for _, x := range []struct{ uuid, inc, v4, v6 string }{{b.SourceUUID, b.SourceIncarnation, b.SourceIPv4, b.SourceIPv6}, {b.TargetUUID, b.TargetIncarnation, b.TargetIPv4, b.TargetIPv6}} {
		e = q.QueryRowContext(ctx, `SELECT count(*) FROM network_bindings n JOIN owned_boxes o ON o.project=n.project AND o.uuid=n.box_uuid AND o.incarnation=n.incarnation
  WHERE n.project=? AND n.box_uuid=? AND n.incarnation=? AND o.state='OWNED' AND n.state='PINNED' AND n.nic='eth0' AND n.bridge=? AND n.acl=? AND n.ipv4=? AND n.ipv6=?`, t.Layer, x.uuid, x.inc, in.Bridge, in.ACL, x.v4, x.v6).Scan(&n)
		if e != nil {
			return zero, false, e
		}
		if n != 1 {
			return zero, false, nil
		}
	}
	return p, true, nil
}
func validProofSHA(s string) bool { return len(s) == 64 && strings.Trim(s, "0123456789abcdef") == "" }

// ValidCompletedPeerProof is a structural check on a private root response,
// not a public authorization assertion. The caller is the root-authenticated
// installed RPC (or a test witness); root independently verified the journal.
func ValidCompletedPeerProof(p RootPeerResult) bool {
	return p.Status == "RECONCILE_VERIFIED" && validProofSHA(p.PlanSHA256) && p.MutationComplete && !p.ReadyForPeer && p.ProbedScopes == 4 && p.DeletedScopes >= 0 && p.DeletedScopes <= 4 && p.KernelVerified && p.VerifiedNICs >= 2 && validProofSHA(p.PostNativeDigestSHA256)
}
func checkRecoveryPeerTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if t.Action != "box_peer_set" || t.State != "UNKNOWN" || t.Released || t.DependentDelete || !t.PeerReady || t.Name == "" || t.UUID == "" || t.Incarnation == "" || p.UUID != t.UUID || p.NativeID != t.NativeID || p.Reason != "ROOT_COMPLETED_POSTSTATE_VERIFIED" || p.Fingerprint == "" || !ValidCompletedPeerProof(p.Peer) {
		return ErrResourceLocked
	}
	peer := t.Peer
	in := peer.Intent
	active, e := activeCredentialForActor(ctx, tx, peer.Actor, peer.Token)
	if e != nil || !active {
		return ErrRevoked
	}
	parent, e := linkedDeletePeerParent(ctx, tx, t.ID, in, true)
	if e != nil {
		return e
	}
	if parent != "" {
		par, e := readCascadeParent(ctx, tx, parent)
		if e != nil || par.State != "UNKNOWN" {
			return ErrResourceLocked
		}
	}
	want := []string{t.Resource, "box:" + in.SourceUUID, "box:" + in.TargetUUID}
	if parent != "" {
		want = append(want, deleteParentKey(parent))
	}
	if e = recoveryClaimKeys(ctx, tx, t, want); e != nil {
		return e
	}
	var n int
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM operations o WHERE o.layer=? AND o.id<>? AND o.id<>?
 AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')
 AND (o.action='box_peer_set' OR NOT EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED'))`, t.Layer, t.ID, parent).Scan(&n)
	if e != nil || n != 0 {
		return ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM peer_grants WHERE project=? AND operation_id<>? AND status IN ('PENDING','UNKNOWN')", t.Layer, t.ID).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM exec_output_gc WHERE project=? AND state IN ('DELETE_SENT','REVIEW_REQUIRED')", t.Layer).Scan(&n); e != nil || n != 0 {
		return ErrResourceLocked
	}
	for _, b := range []struct{ uuid, inc string }{{in.SourceUUID, in.SourceIncarnation}, {in.TargetUUID, in.TargetIncarnation}} {
		pinned, e := verifyDeletePinned(ctx, tx, t.Layer, b.uuid, b.inc)
		if e != nil || !pinned {
			return ErrResourceLocked
		}
	}
	return nil
}

func finishRecoveryPeerTx(ctx context.Context, tx *sql.Tx, t RecoveryTarget, p RecoveryProof) error {
	if e := checkRecoveryPeerTx(ctx, tx, t, p); e != nil {
		return e
	}
	in := t.Peer.Intent
	if in.Allow {
		return finishCompletedUnknownRootPeerGrantTx(ctx, tx, t.ID)
	}
	return finishCompletedUnknownRootPeerRevocationTx(ctx, tx, t.ID)
}
