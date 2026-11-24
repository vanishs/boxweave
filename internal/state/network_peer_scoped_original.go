package state

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/vanishs/boxweave/internal/peerscope"
)

// ScopedPeerOriginalIntent is v2-only. It intentionally has no legacy
// shared ACL field, and must never be accepted as socket/MCP native input.
type ScopedPeerOriginalIntent struct {
	Schema            int    `json:"schema"`
	Project           string `json:"project"`
	Bridge            string `json:"bridge"`
	SourceUUID        string `json:"source_uuid"`
	SourceIncarnation string `json:"source_incarnation"`
	TargetUUID        string `json:"target_uuid"`
	TargetIncarnation string `json:"target_incarnation"`
	Allow             bool   `json:"allow"`
}

// Read-only authorization evidence, NOT a native mutation permission.
// An independent Root writer must redo SQLite/native/port/kernel proofs
// with a durable original and locks held.
type ScopedPeerOriginalProof struct {
	OriginalID                      string
	ActorID, TokenID, HistorySHA256 string
	Intent                          ScopedPeerOriginalIntent
	Inventory                       ScopedNetworkAuditLedger
	Before, After, Revoked          []PeerGrantBinding
	RevokedBefore, RevokedAfter     []PeerGrantBinding
	Regrant, RecoveryOnly           bool
}

func scopedPeerResourceClaims(i ScopedPeerOriginalIntent) ([]string, error) {
	a, e := peerscope.ACLName(i.Project, i.Bridge, i.SourceUUID)
	if e != nil {
		return nil, ErrInvalid
	}
	b, e := peerscope.ACLName(i.Project, i.Bridge, i.TargetUUID)
	if e != nil || a == b {
		return nil, ErrInvalid
	}
	claims := []string{"peer-guard:" + i.Bridge, "acl:default/" + a, "acl:default/" + b,
		"box:" + i.SourceUUID, "box:" + i.TargetUUID}
	slices.Sort(claims)
	return claims, nil
}
func (s *Store) ReadRootScopedPeerOriginal(ctx context.Context, id, project, bridge string) (ScopedPeerOriginalProof, error) {
	return s.readRootScopedPeerOriginal(ctx, id, project, bridge, "DISPATCHING")
}

// UNKNOWN permits read-only recovery and NEVER another native dispatch.
func (s *Store) ReadRootScopedPeerUnknownOriginal(ctx context.Context, id, project, bridge string) (ScopedPeerOriginalProof, error) {
	return s.readRootScopedPeerOriginal(ctx, id, project, bridge, "UNKNOWN")
}
func (s *Store) readRootScopedPeerOriginal(ctx context.Context, id, project, bridge, expected string) (ScopedPeerOriginalProof, error) {
	var zero ScopedPeerOriginalProof
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if s == nil || s.db == nil || !s.IsNetworkAuditReadOnly() || !validHexID(id) ||
		expected != "DISPATCHING" && expected != "UNKNOWN" ||
		!strings.HasPrefix(project, "user-") || !auditName(project) ||
		!auditName(bridge) || project == "default" {
		return zero, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	first, e := readScopedPeerOriginalTx(ctx, tx, id, project, bridge, expected)
	if e != nil {
		return zero, e
	}
	if e = tx.Commit(); e != nil {
		return zero, e
	}
	again, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer again.Rollback()
	second, e := readScopedPeerOriginalTx(ctx, again, id, project, bridge, expected)
	if e != nil || !reflect.DeepEqual(first, second) {
		return zero, ErrResourceLocked
	}
	if e = again.Commit(); e != nil {
		return zero, e
	}
	return second, nil
}

func readScopedPeerOriginalTx(ctx context.Context, tx *sql.Tx, id, project, bridge, expected string) (ScopedPeerOriginalProof, error) {
	var zero ScopedPeerOriginalProof
	if tx == nil || !validHexID(id) || !strings.HasPrefix(project, "user-") || !auditName(project) || !auditName(bridge) || expected != "DISPATCHING" && expected != "UNKNOWN" && expected != "VERIFYING" {
		return zero, ErrInvalid
	}
	// One read transaction binds the complete Project NIC/MAC/IP/fence
	// snapshot to the same original/claims/actor and all grant directions.
	inventory, e := readCompleteScopedNetworkAuditLedgerTx(ctx, tx, project, bridge)
	if e != nil || len(inventory.Boxes) < 2 {
		return zero, ErrResourceLocked
	}
	owners := map[string]ScopedNetworkAuditBinding{}
	for _, b := range inventory.Boxes {
		owners[b.UUID] = b
	}
	retiredOwners := map[string]bool{}
	for _, b := range inventory.Retired {
		owners[b.UUID] = b.ScopedNetworkAuditBinding
		retiredOwners[b.UUID] = true
	}

	var action, status, actor, token, layer, source, incarnation, resource string
	var raw []byte
	e = tx.QueryRowContext(ctx, `SELECT action,state,actor_id,token_id,layer,box_uuid,
  incarnation,resource_key,safe_intent FROM operations WHERE id=?`, id).
		Scan(&action, &status, &actor, &token, &layer, &source, &incarnation, &resource, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return zero, ErrNotFound
	}
	if e != nil {
		return zero, e
	}
	var intent ScopedPeerOriginalIntent
	if json.Unmarshal(raw, &intent) != nil {
		return zero, ErrResourceLocked
	}
	canonical, e := json.Marshal(intent)
	if e != nil || !bytes.Equal(canonical, raw) || intent.Schema != 2 ||
		action != "box_peer_set" || status != expected || actor == "" || token == "" ||
		layer != project || intent.Project != project || intent.Bridge != bridge ||
		source != intent.SourceUUID || incarnation != intent.SourceIncarnation ||
		resource != "peer-guard:"+bridge ||
		intent.SourceUUID == intent.TargetUUID || intent.SourceUUID == "" ||
		intent.TargetUUID == "" || intent.SourceIncarnation == "" ||
		intent.TargetIncarnation == "" {
		return zero, ErrResourceLocked
	}
	a, okA := owners[intent.SourceUUID]
	b, okB := owners[intent.TargetUUID]
	if !okA || !okB || retiredOwners[intent.SourceUUID] || retiredOwners[intent.TargetUUID] || a.Incarnation != intent.SourceIncarnation ||
		b.Incarnation != intent.TargetIncarnation {
		return zero, ErrResourceLocked
	}
	expectedClaims, e := scopedPeerResourceClaims(intent)
	if e != nil {
		return zero, ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, `SELECT resource_key,layer FROM operation_resource_claims
  WHERE operation_id=? ORDER BY resource_key`, id)
	if e != nil {
		return zero, e
	}
	var claims []string
	for rows.Next() {
		var c, l string
		if e = rows.Scan(&c, &l); e != nil {
			break
		}
		if l != project {
			e = ErrResourceLocked
			break
		}
		claims = append(claims, c)
	}
	if x := rows.Err(); e == nil {
		e = x
	}
	if x := rows.Close(); e == nil {
		e = x
	}
	if e != nil {
		return zero, e
	}
	if !slices.Equal(expectedClaims, claims) {
		return zero, ErrResourceLocked
	}
	active, e := activeCredentialForActor(ctx, tx, actor, token)
	if e != nil || !active {
		return zero, ErrRevoked
	}
	var busy int
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o WHERE o.layer=? AND o.id<>?
  AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
  AND (o.action='box_peer_set' OR NOT EXISTS
  (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))`,
		project, id).Scan(&busy)
	if e != nil || busy != 0 {
		return zero, ErrResourceLocked
	}
	var regrant int
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events
	 WHERE operation_id=? AND safe_detail='SCOPED_PEER_V2_REGRANT_FROM_REVOKED'`,
		id).Scan(&regrant); e != nil || regrant > 1 ||
		(regrant == 1 && !intent.Allow) {
		return zero, ErrResourceLocked
	}
	rows, e = tx.QueryContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,
  target_incarnation,source_ipv4,target_ipv4,source_ipv6,target_ipv6,
  desired,status,operation_id,COALESCE(last_verified_operation_id,'')
  FROM peer_grants WHERE project=? ORDER BY source_uuid,target_uuid`, project)
	if e != nil {
		return zero, e
	}
	var before, after, revoked []PeerGrantBinding
	var pendingPair PeerGrantBinding
	pending, total := 0, 0
	history := sha256.New()
	for rows.Next() {
		var p PeerGrantBinding
		var desired int
		var st, original, verified string
		if e = rows.Scan(&p.SourceUUID, &p.SourceIncarnation, &p.TargetUUID,
			&p.TargetIncarnation, &p.SourceIPv4, &p.TargetIPv4, &p.SourceIPv6,
			&p.TargetIPv6, &desired, &st, &original, &verified); e != nil {
			break
		}
		total++
		row, e2 := json.Marshal(struct {
			Pair                      PeerGrantBinding
			Desired                   int
			State, Original, Verified string
		}{p, desired, st, original, verified})
		if e2 != nil {
			e = e2
			break
		}
		history.Write(row)
		history.Write([]byte{'\n'})
		src, ok1 := owners[p.SourceUUID]
		dst, ok2 := owners[p.TargetUUID]
		if !ok1 || !ok2 || p.SourceUUID == p.TargetUUID || (retiredOwners[p.SourceUUID] || retiredOwners[p.TargetUUID]) && (desired != 0 || st != "REVOKED") ||
			p.SourceIncarnation != src.Incarnation || p.TargetIncarnation != dst.Incarnation ||
			p.SourceIPv4 != src.IPv4 || p.TargetIPv4 != dst.IPv4 ||
			p.SourceIPv6 != src.IPv6 || p.TargetIPv6 != dst.IPv6 ||
			total > 32 {
			e = ErrResourceLocked
			break
		}
		switch {
		case desired == 1 && st == "CONFIRMED":
			e = scopedPeerVerifyTerminalLineageTx(ctx, tx, project, bridge,
				p, desired, st, original, verified)
			if e == nil {
				before = append(before, p)
				after = append(after, p)
			}
		case desired == 0 && st == "REVOKED":
			e = scopedPeerVerifyTerminalLineageTx(ctx, tx, project, bridge,
				p, desired, st, original, verified)
			if e == nil {
				revoked = append(revoked, p)
			}
		case st == "PENDING" && original == id && pending == 0 &&
			p.SourceUUID == intent.SourceUUID && p.TargetUUID == intent.TargetUUID &&
			p.SourceIncarnation == intent.SourceIncarnation &&
			p.TargetIncarnation == intent.TargetIncarnation &&
			(intent.Allow && desired == 1 || !intent.Allow && desired == 0):
			// A fresh grant has no prior verified original. Revoke and
			// regrant retain the exact previously successful A/B operation.
			if intent.Allow && regrant == 0 {
				if verified != "" {
					e = ErrResourceLocked
				}
			} else {
				e = scopedPeerVerifyHistoricalOperationTx(ctx, tx,
					project, bridge, p, verified, !intent.Allow)
			}
			if e != nil {
				break
			}
			pending++
			pendingPair = p
			if intent.Allow {
				after = append(after, p)
			} else {
				before = append(before, p)
			}
		default:
			e = ErrResourceLocked
		}
		if e != nil {
			break
		}
	}
	if x := rows.Err(); e == nil {
		e = x
	}
	if x := rows.Close(); e == nil {
		e = x
	}
	if e != nil {
		return zero, e
	}
	if pending != 1 || len(before) > 32 || len(after) > 32 {
		return zero, ErrResourceLocked
	}
	revokedBefore := slices.Clone(revoked)
	revokedAfter := slices.Clone(revoked)
	if regrant == 1 {
		revokedBefore = append(revokedBefore, pendingPair)
	} else if !intent.Allow {
		revokedAfter = append(revokedAfter, pendingPair)
	}
	sort := func(a []PeerGrantBinding) {
		slices.SortFunc(a, func(x, y PeerGrantBinding) int {
			return strings.Compare(x.SourceUUID+"/"+x.TargetUUID, y.SourceUUID+"/"+y.TargetUUID)
		})
	}
	sort(before)
	sort(after)
	sort(revoked)
	sort(revokedBefore)
	sort(revokedAfter)
	return ScopedPeerOriginalProof{OriginalID: id, ActorID: actor, TokenID: token, HistorySHA256: hex.EncodeToString(history.Sum(nil)), Intent: intent, Inventory: inventory,
		Before: before, After: after, Revoked: revoked,
		RevokedBefore: revokedBefore, RevokedAfter: revokedAfter,
		Regrant: regrant == 1, RecoveryOnly: expected == "UNKNOWN"}, nil
}

// ScopedPeerProofSHA256 binds the complete durable original, including actor,
// credential identity, all sibling directions and retained address history.
// RecoveryOnly is a phase marker, not a change to the immutable intent.
func ScopedPeerProofSHA256(p ScopedPeerOriginalProof) string {
	p.RecoveryOnly = false
	b, e := json.Marshal(p)
	if e != nil {
		return ""
	}
	h := sha256.Sum256(append([]byte("boxweave/scoped-peer-original/v2\n"), b...))
	return hex.EncodeToString(h[:])
}
