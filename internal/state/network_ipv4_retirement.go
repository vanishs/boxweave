package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// IPv4RetirementCandidate binds one exact native Delete to its original
// Create and its old static tuple. It is a ledger assertion, NOT native
// proof. Peer history remains fenced until DROP compaction is verified. Revoked
// Forward history requires exact original terminal receipts and root socket absence.
type IPv4RetirementCandidate struct {
	Project, Bridge, ACL                 string
	Binding                              NetworkAuditBinding
	CreateOperationID, DeleteOperationID string
	OperationEpoch, EventEpoch           int64
	RequiresPeerCompaction               bool
}

func validRetirementProofSHA(s string) bool {
	if len(s) != 64 || s != strings.ToLower(s) {
		return false
	}
	_, e := hex.DecodeString(s)
	return e == nil
}

// Every historical proxy must have exact original identity and either a
// successful verified native removal or a proven no-dispatch original.
// REVOKED alone is not sufficient. The host must independently verify
// native device/listener absence before authorizing the archive.
const retirementForwardInvalid = `(
 f.incarnation!=b.incarnation OR f.state!='REVOKED' OR f.guest_ip!='127.0.0.1'
 OR length(f.forward_id)!=32 OR f.forward_id GLOB '*[^0-9a-f]*'
 OR f.native_device!='bwpf-'||substr(f.forward_id,1,16)
 OR NOT EXISTS (SELECT 1 FROM operations c WHERE c.id=f.forward_id AND c.layer=f.project AND c.box_uuid=f.box_uuid AND c.incarnation=f.incarnation
  AND c.action='port_forward_create' AND c.resource_key='box:'||b.name
  AND json_valid(c.safe_intent) AND json_extract(c.safe_intent,'$.box_id')=f.box_uuid
  AND json_extract(c.safe_intent,'$.protocol')=f.protocol AND json_extract(c.safe_intent,'$.host_bind')=f.host_bind
  AND json_extract(c.safe_intent,'$.host_port')=f.host_port AND json_extract(c.safe_intent,'$.guest_port')=f.guest_port
  AND ((c.state IN ('FAILED','CANCELLED') AND c.native_operation_id='' AND f.operation_id=c.id)
       OR (c.state='SUCCEEDED' AND c.native_operation_id!=''
          AND EXISTS (SELECT 1 FROM operations d WHERE d.id=f.operation_id AND d.layer=f.project AND d.box_uuid=f.box_uuid AND d.incarnation=f.incarnation
           AND d.action='port_forward_delete' AND d.state='SUCCEEDED' AND d.native_operation_id!=''
           AND d.resource_key='box:'||b.name AND json_valid(d.safe_intent) AND json_extract(d.safe_intent,'$.forward_id')=f.forward_id)))
 ))`

// A historical peer may be removed from the compiled policy only when
// its exact root-verified revoke, pinned endpoint tuples and terminal
// event agree. A bare REVOKED bit is never sufficient.
const retirementPeerInvalid = `(
 p.status!='REVOKED' OR p.desired!=0 OR p.source_uuid=p.target_uuid
 OR p.source_uuid!=p.source_incarnation OR p.target_uuid!=p.target_incarnation
 OR p.last_verified_operation_id IS NULL OR p.operation_id!=p.last_verified_operation_id
 OR length(p.operation_id)!=32 OR p.operation_id GLOB '*[^0-9a-f]*'
 OR NOT EXISTS(SELECT 1 FROM operations v
 JOIN network_bindings x ON x.project=p.project AND x.box_uuid=p.source_uuid AND x.incarnation=p.source_incarnation AND x.nic='eth0'
 JOIN network_bindings y ON y.project=p.project AND y.box_uuid=p.target_uuid AND y.incarnation=p.target_incarnation AND y.nic='eth0'
 JOIN owned_boxes bx ON bx.project=x.project AND bx.uuid=x.box_uuid AND bx.incarnation=x.incarnation
 JOIN owned_boxes by ON by.project=y.project AND by.uuid=y.box_uuid AND by.incarnation=y.incarnation
 WHERE v.id=p.operation_id AND v.layer=p.project AND v.action='box_peer_set' AND v.state='SUCCEEDED'
 AND v.box_uuid=p.source_uuid AND v.incarnation=p.source_incarnation
 AND x.bridge=y.bridge AND x.acl=y.acl AND v.resource_key='acl:default/'||x.acl
 AND x.ipv4=p.source_ipv4 AND x.ipv6=p.source_ipv6 AND y.ipv4=p.target_ipv4 AND y.ipv6=p.target_ipv6
 AND ((bx.state='OWNED' AND x.state='PINNED') OR (bx.state='DELETED' AND x.state='RELEASED'))
 AND ((by.state='OWNED' AND y.state='PINNED') OR (by.state='DELETED' AND y.state='RELEASED'))
 AND json_valid(v.safe_intent)
 AND json_extract(v.safe_intent,'$.project')=p.project
 AND json_extract(v.safe_intent,'$.bridge')=x.bridge AND json_extract(v.safe_intent,'$.acl')=x.acl
 AND json_extract(v.safe_intent,'$.source_uuid')=p.source_uuid AND json_extract(v.safe_intent,'$.source_incarnation')=p.source_incarnation
 AND json_extract(v.safe_intent,'$.target_uuid')=p.target_uuid AND json_extract(v.safe_intent,'$.target_incarnation')=p.target_incarnation
 AND json_type(v.safe_intent,'$.allow')='false'
 AND (SELECT count(*) FROM json_each(v.safe_intent))=8
 AND EXISTS(SELECT 1 FROM operation_events e WHERE e.operation_id=v.id AND e.state='SUCCEEDED' AND e.safe_detail IN ('PEER_ROOT_SCOPED_REVOCATION_VERIFIED','ROOT_COMPLETED_UNKNOWN_RECONCILED_NO_REPLAY'))
 AND EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=v.id AND c.layer=p.project AND c.resource_key='acl:default/'||x.acl)
 AND EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=v.id AND c.layer=p.project AND c.resource_key='box:'||p.source_uuid)
 AND EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=v.id AND c.layer=p.project AND c.resource_key='box:'||p.target_uuid)
 ))`

// Both endpoints must still be unarchived for the DROP to be emitted.
// verifyIPv4RetirementHistory checks excluded rows before any policy read.
const retirementPeerUncompacted = `NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r
 WHERE r.project=p.project AND ((r.box_uuid=p.source_uuid AND r.incarnation=p.source_incarnation)
 OR (r.box_uuid=p.target_uuid AND r.incarnation=p.target_incarnation)))`

// readIPv4RetirementCandidate is usable in both read-only root audit and
// the final writer transaction. It refuses even RELEASED UNKNOWN operations.
// The project-wide quiescence check is additional to exact UUID proof.
func readIPv4RetirementCandidate(ctx context.Context, tx *sql.Tx, project, original string) (IPv4RetirementCandidate, error) {
	var out IPv4RetirementCandidate
	if !auditName(project) || project == "default" || !validHexID(original) {
		return out, ErrInvalid
	}
	out.Project = project
	out.DeleteOperationID = original
	var busy int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations WHERE layer=? AND state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')", project).Scan(&busy); e != nil || busy != 0 {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	e := tx.QueryRowContext(ctx, `SELECT n.bridge,n.acl,n.mac,n.ipv4,n.ipv6,b.name,b.uuid,b.incarnation
 FROM owned_boxes b JOIN network_bindings n ON n.project=b.project AND n.box_uuid=b.uuid AND n.incarnation=b.incarnation
 JOIN operations d ON d.id=b.deleted_by_operation_id AND d.layer=b.project AND d.box_uuid=b.uuid AND d.incarnation=b.incarnation
 WHERE b.project=? AND b.state='DELETED' AND b.deleted_by_operation_id=? AND d.action='box_delete' AND d.state='SUCCEEDED'
 AND d.native_operation_id!='' AND d.resource_key='box:'||b.name AND n.nic='eth0' AND n.state='RELEASED'
 AND NOT EXISTS (SELECT 1 FROM network_ipv4_retirements r WHERE r.project=b.project AND r.box_uuid=b.uuid AND r.incarnation=b.incarnation)`, project, original).Scan(
		&out.Bridge, &out.ACL, &out.Binding.MAC, &out.Binding.IPv4, &out.Binding.IPv6, &out.Binding.Name, &out.Binding.UUID, &out.Binding.Incarnation)
	if errors.Is(e, sql.ErrNoRows) {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	if e != nil || !auditName(out.Bridge) || !auditName(out.ACL) || !auditName(out.Binding.Name) || out.Binding.UUID == "" || out.Binding.Incarnation != out.Binding.UUID || !auditMAC(out.Binding.MAC) || !auditIP(out.Binding.IPv4, false) || !auditIP(out.Binding.IPv6, true) {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	b := out.Binding
	var count int
	if e = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM operations c WHERE layer=? AND action='box_create' AND box_uuid=? AND incarnation=? AND state='SUCCEEDED' AND resource_key='box:'||?", project, b.UUID, b.Incarnation, b.Name).Scan(&count); e != nil || count != 1 {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT id FROM operations c WHERE layer=? AND action='box_create' AND box_uuid=? AND incarnation=? AND state='SUCCEEDED' AND resource_key='box:'||? AND "+successfulCreateLineageEvidence, project, b.UUID, b.Incarnation, b.Name).Scan(&out.CreateOperationID); e != nil || !validHexID(out.CreateOperationID) {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	pin := VerifiedCreateBinding{Bridge: out.Bridge, ACL: out.ACL, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}
	if e = requireCreateBindingReservation(ctx, tx, out.CreateOperationID, project, pin); e != nil {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=?`, project, b.UUID, b.Incarnation).Scan(&count); e != nil || count != 2 {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	for _, a := range []struct{ family, ip string }{{"ipv4", b.IPv4}, {"ipv6", b.IPv6}} {
		if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND bridge=? AND family=? AND address=? AND state='RELEASED' AND operation_id=?`, project, b.UUID, b.Incarnation, out.Bridge, a.family, a.ip, original).Scan(&count); e != nil || count != 1 {
			return IPv4RetirementCandidate{}, ErrResourceLocked
		}
	}
	// No old ACL relation or proxy may be silently discarded by this initial
	// implementation. Count even revoked/foreign-incarnation records.
	if e = tx.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM peer_grants p WHERE p.project=? AND (p.source_uuid=? OR p.target_uuid=?) AND `+retirementPeerInvalid+`)+
 (SELECT COUNT(*) FROM forward_claims f JOIN owned_boxes b ON b.project=f.project AND b.uuid=f.box_uuid WHERE f.project=? AND f.box_uuid=? AND `+retirementForwardInvalid+`)+
 (SELECT COUNT(*) FROM network_bindings WHERE project=? AND box_uuid=? AND (incarnation!=? OR state!='RELEASED' OR nic!='eth0'))+
 (SELECT COUNT(*) FROM network_address_fences WHERE bridge=? AND project!=? AND state!='RELEASED')+
 (SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND project!=? AND state!='RELEASED')`, project, b.UUID, b.UUID, project, b.UUID, project, b.UUID, b.Incarnation, out.Bridge, project, out.Bridge, project).Scan(&count); e != nil || count != 0 {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM peer_grants p WHERE p.project=? AND (p.source_uuid=? OR p.target_uuid=?) AND `+retirementPeerUncompacted, project, b.UUID, b.UUID).Scan(&count); e != nil {
		return IPv4RetirementCandidate{}, e
	}
	out.RequiresPeerCompaction = count != 0
	var pending string
	e = tx.QueryRowContext(ctx, "SELECT delete_operation_id FROM network_ipv4_compactions WHERE project=?", project).Scan(&pending)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return IPv4RetirementCandidate{}, e
	}
	if e == nil && (pending != original || !out.RequiresPeerCompaction) {
		return IPv4RetirementCandidate{}, ErrResourceLocked
	}
	if e = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid),0) FROM operations").Scan(&out.OperationEpoch); e != nil {
		return IPv4RetirementCandidate{}, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(seq),0) FROM operation_events").Scan(&out.EventEpoch); e != nil {
		return IPv4RetirementCandidate{}, e
	}
	return out, nil
}

func (s *Store) ReadIPv4RetirementCandidate(ctx context.Context, project, original string) (IPv4RetirementCandidate, error) {
	if s == nil || s.db == nil {
		return IPv4RetirementCandidate{}, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return IPv4RetirementCandidate{}, e
	}
	defer tx.Rollback()
	out, e := readIPv4RetirementCandidate(ctx, tx, project, original)
	if e != nil {
		return IPv4RetirementCandidate{}, e
	}
	if e = tx.Commit(); e != nil {
		return IPv4RetirementCandidate{}, e
	}
	return out, nil
}

// IPv4RetirementWitness must be supplied by the trusted controller, not
// a public request. It uses an independent root read-only connection.
type IPv4RetirementWitness func(context.Context, IPv4RetirementCandidate) (IPv4RetirementCandidate, string, error)

// RetireVerifiedIPv4 obtains the SQLite writer reservation BEFORE reading
// the candidate and holds it across the root witness and archive commit.
// WAL permits the independently opened root audit connection to read the
// committed ledger, but neither another controller nor a second SQLite writer
// can admit a native effect in this interval. Root checks all owned guests
// stopped and no bridge ports twice. Trusted host root remains outside the
// controller boundary. Any denial, timeout or crash leaves both fences held.
func (s *Store) RetireVerifiedIPv4(ctx context.Context, project, original string, witness IPv4RetirementWitness) error {
	if s == nil || s.db == nil || s.auditReadOnly || witness == nil {
		return ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Obtain the SQLite reserved writer lock before taking the read snapshot.
	// A zero-row UPDATE alone is not a sufficient documented lock guarantee.
	res, e := tx.ExecContext(ctx, "UPDATE schema_version SET version=version WHERE version=?", SchemaVersion)
	if e != nil {
		return e
	}
	if n, e := res.RowsAffected(); e != nil || n != 1 {
		return ErrResourceLocked
	}
	got, e := readIPv4RetirementCandidate(ctx, tx, project, original)
	if e != nil {
		return e
	}
	plan := ""
	if got.RequiresPeerCompaction {
		if e = tx.QueryRowContext(ctx, "SELECT plan_sha256 FROM network_ipv4_compactions WHERE project=? AND delete_operation_id=?", project, original).Scan(&plan); e != nil || !validRetirementProofSHA(plan) {
			return ErrResourceLocked
		}
	}
	verified, proof, e := witness(ctx, got)
	if e != nil {
		return e
	}
	if ctx.Err() != nil || verified != got || !validRetirementProofSHA(proof) {
		return ErrResourceLocked
	}
	after, e := readIPv4RetirementCandidate(ctx, tx, project, original)
	if e != nil || after != got {
		return ErrResourceLocked
	}
	if e = commitVerifiedIPv4RetirementTx(ctx, tx, got, proof, plan); e != nil {
		return e
	}
	return tx.Commit()
}

// commitVerifiedIPv4RetirementTx has no exported direct-digest entry point.
func commitVerifiedIPv4RetirementTx(ctx context.Context, tx *sql.Tx, got IPv4RetirementCandidate, nativeProofSHA, plan string) error {
	b := got.Binding
	if _, e := tx.ExecContext(ctx, `INSERT INTO network_ipv4_retirements(project,box_uuid,incarnation,bridge,acl,mac,ipv4,ipv6,create_operation_id,delete_operation_id,native_proof_sha256,peer_compaction_sha256,retired_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		got.Project, b.UUID, b.Incarnation, got.Bridge, got.ACL, b.MAC, b.IPv4, b.IPv6, got.CreateOperationID, got.DeleteOperationID, nativeProofSHA, plan, time.Now().UnixNano()); e != nil {
		return e
	}
	// Only the original Delete's exact fences are archived. Retain the
	// immutable binding and all original Create resource claims. IPv6/MAC
	// are never recycled; only the old IPv4 claim becomes reusable.
	r, e := tx.ExecContext(ctx, `DELETE FROM network_address_fences WHERE project=? AND box_uuid=? AND incarnation=? AND bridge=? AND state='RELEASED' AND operation_id=? AND ((family='ipv4' AND address=?) OR (family='ipv6' AND address=?))`, got.Project, b.UUID, b.Incarnation, got.Bridge, got.DeleteOperationID, b.IPv4, b.IPv6)
	if e != nil {
		return e
	}
	if n, e := r.RowsAffected(); e != nil || n != 2 {
		return ErrResourceLocked
	}
	if got.RequiresPeerCompaction {
		ds, e := tx.ExecContext(ctx, "DELETE FROM network_ipv4_compactions WHERE project=? AND delete_operation_id=? AND plan_sha256=?", got.Project, got.DeleteOperationID, plan)
		if e != nil {
			return e
		}
		if n, e := ds.RowsAffected(); e != nil || n != 1 {
			return ErrResourceLocked
		}
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM network_ipv4_retirement_queue WHERE delete_operation_id=? AND project=?", got.DeleteOperationID, got.Project); e != nil {
		return e
	}
	return nil
}

// verifyIPv4RetirementHistory validates every archived identity in a
// streaming read; no lifetime history-sized Go map and no truncation.
// Excluding these rows without exact evidence would admit stale privileges.
func verifyIPv4RetirementHistory(ctx context.Context, tx *sql.Tx, project, bridge, acl string) error {
	var total int
	if e := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM network_ipv4_retirements WHERE project=?", project).Scan(&total); e != nil {
		return e
	}
	if total == 0 {
		return nil
	}
	rows, e := tx.QueryContext(ctx, `SELECT r.box_uuid,r.incarnation,r.bridge,r.acl,r.mac,r.ipv4,r.ipv6,r.create_operation_id,r.delete_operation_id,r.native_proof_sha256,r.peer_compaction_sha256,r.retired_at,b.name
 FROM network_ipv4_retirements r
 JOIN owned_boxes b ON b.project=r.project AND b.uuid=r.box_uuid AND b.incarnation=r.incarnation AND b.state='DELETED' AND b.deleted_by_operation_id=r.delete_operation_id
 JOIN network_bindings n ON n.project=r.project AND n.box_uuid=r.box_uuid AND n.incarnation=r.incarnation AND n.nic='eth0' AND n.state='RELEASED' AND n.bridge=r.bridge AND n.acl=r.acl AND n.mac=r.mac AND n.ipv4=r.ipv4 AND n.ipv6=r.ipv6
 JOIN operations c ON c.id=r.create_operation_id AND c.layer=r.project AND c.box_uuid=r.box_uuid AND c.incarnation=r.incarnation AND c.action='box_create' AND c.state='SUCCEEDED' AND `+successfulCreateLineageEvidence+` AND c.resource_key='box:'||b.name
 JOIN operations d ON d.id=r.delete_operation_id AND d.layer=r.project AND d.box_uuid=r.box_uuid AND d.incarnation=r.incarnation AND d.action='box_delete' AND d.state='SUCCEEDED' AND d.native_operation_id!='' AND d.resource_key='box:'||b.name
 WHERE r.project=?
 AND NOT EXISTS (SELECT 1 FROM network_address_fences f WHERE f.project=r.project AND f.box_uuid=r.box_uuid AND f.incarnation=r.incarnation)
 AND NOT EXISTS (SELECT 1 FROM network_bindings x WHERE x.project=r.project AND x.box_uuid=r.box_uuid AND (x.incarnation!=r.incarnation OR x.nic!='eth0' OR x.state!='RELEASED'))
 AND NOT EXISTS (SELECT 1 FROM peer_grants p WHERE p.project=r.project AND (p.source_uuid=r.box_uuid OR p.target_uuid=r.box_uuid) AND (`+retirementPeerInvalid+` OR NOT EXISTS(SELECT 1 FROM network_ipv4_retirements cr WHERE cr.project=p.project AND length(cr.peer_compaction_sha256)=64 AND ((cr.box_uuid=p.source_uuid AND cr.incarnation=p.source_incarnation) OR (cr.box_uuid=p.target_uuid AND cr.incarnation=p.target_incarnation)))))
 AND NOT EXISTS (SELECT 1 FROM forward_claims f WHERE f.project=r.project AND f.box_uuid=r.box_uuid AND `+retirementForwardInvalid+`)
 AND (SELECT COUNT(*) FROM operation_resource_claims x WHERE x.operation_id=c.id AND x.resource_key LIKE 'create-pin:%')=4
 AND EXISTS (SELECT 1 FROM operation_resource_claims x WHERE x.operation_id=c.id AND x.layer=r.project AND x.resource_key='create-pin:'||r.bridge||':v4:'||r.ipv4)
 AND EXISTS (SELECT 1 FROM operation_resource_claims x WHERE x.operation_id=c.id AND x.layer=r.project AND x.resource_key='create-pin:'||r.bridge||':v6:'||r.ipv6)
 AND EXISTS (SELECT 1 FROM operation_resource_claims x WHERE x.operation_id=c.id AND x.layer=r.project AND x.resource_key='create-pin:'||r.bridge||':mac:'||r.mac)
 AND EXISTS (SELECT 1 FROM operation_resource_claims x WHERE x.operation_id=c.id AND x.layer=r.project AND x.resource_key='create-pin:'||r.bridge||':acl:'||r.acl)`, project)
	if e != nil {
		return e
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var uuid, inc, nw, ac, mac, v4, v6, create, del, proof, plan, name string
		var at int64
		if e = rows.Scan(&uuid, &inc, &nw, &ac, &mac, &v4, &v6, &create, &del, &proof, &plan, &at, &name); e != nil {
			return e
		}
		if uuid == "" || inc != uuid || nw != bridge || (acl != "" && ac != acl) || !auditName(ac) || !auditName(name) || !auditMAC(mac) || !auditIP(v4, false) || !auditIP(v6, true) || !validHexID(create) || !validHexID(del) || !validRetirementProofSHA(proof) || (plan != "" && !validRetirementProofSHA(plan)) || at <= 0 {
			return ErrResourceLocked
		}
		n++
	}
	if e = rows.Err(); e != nil {
		return e
	}
	if e = rows.Close(); e != nil {
		return e
	}
	if n != total {
		return ErrResourceLocked
	}
	return nil
}

// IPv4RetirementPolicy provides the complete current ledger and every
// confirmed/revoked direction, including unrelated retained history.
// The exact retirement candidate is checked around all reads. Ordinary
// Peer APIs continue to refuse an empty Project. The caller still needs
// independent root proof and the project exclusion fence.
type IPv4RetirementPolicy struct {
	Candidate       IPv4RetirementCandidate
	Ledger          NetworkAuditLedger
	Active, Revoked []PeerGrantBinding
	AfterRevoked    []PeerGrantBinding
	Forwards        []ForwardClaim
}

func (s *Store) ReadIPv4RetirementPolicy(ctx context.Context, want IPv4RetirementCandidate) (IPv4RetirementPolicy, error) {
	var zero IPv4RetirementPolicy
	got, e := s.ReadIPv4RetirementCandidate(ctx, want.Project, want.DeleteOperationID)
	if e != nil || got != want {
		return zero, ErrResourceLocked
	}
	ledger, e := s.readCompleteNetworkAuditLedger(ctx, want.Project, want.Bridge, want.ACL, true)
	if e != nil {
		return zero, e
	}
	live, e := s.readPeerGrantSnapshotMode(ctx, want.Project, want.Bridge, want.ACL, "", false, true)
	if e != nil {
		return zero, e
	}
	old, e := s.readPeerRevokedSnapshot(ctx, want.Project, want.Bridge, want.ACL, true)
	if e != nil {
		return zero, e
	}
	forwards, e := s.readRetiredForwardHistory(ctx, want)
	if e != nil {
		return zero, e
	}
	after, e := s.ReadIPv4RetirementCandidate(ctx, want.Project, want.DeleteOperationID)
	if e != nil || after != want {
		return zero, ErrResourceLocked
	}
	var afterRevoked []PeerGrantBinding
	dropped := 0
	for _, p := range old {
		if p.SourceUUID == want.Binding.UUID || p.TargetUUID == want.Binding.UUID {
			dropped++
			continue
		}
		afterRevoked = append(afterRevoked, p)
	}
	if (dropped > 0) != want.RequiresPeerCompaction {
		return zero, ErrResourceLocked
	}
	return IPv4RetirementPolicy{Candidate: want, Ledger: ledger, Active: live, Revoked: old, AfterRevoked: afterRevoked, Forwards: forwards}, nil
}

// ClaimPendingIPv4Retirements rotates the bounded maintenance queue across
// process restarts. A ticket only records an attempted review, never a native
// authorization or a successful retirement. Old permanently deferred entries
// cannot starve later eligible originals. Claim at most the requested batch
// in the same transaction; crash after claim merely delays that read-only try.
func (s *Store) ClaimPendingIPv4Retirements(ctx context.Context, project string, limit int) ([]string, error) {
	if s == nil || s.db == nil || s.auditReadOnly || !auditName(project) || project == "default" || limit < 1 || limit > 8 {
		return nil, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	// Take the reserved writer lock before selecting any queue positions.
	res, e := tx.ExecContext(ctx, "UPDATE schema_version SET version=version WHERE version=?", SchemaVersion)
	if e != nil {
		return nil, e
	}
	if n, e := res.RowsAffected(); e != nil || n != 1 {
		return nil, ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, `SELECT b.deleted_by_operation_id
 FROM owned_boxes b
 JOIN network_bindings n ON n.project=b.project AND n.box_uuid=b.uuid AND n.incarnation=b.incarnation AND n.nic='eth0'
 JOIN operations d ON d.id=b.deleted_by_operation_id AND d.layer=b.project AND d.box_uuid=b.uuid AND d.incarnation=b.incarnation AND d.state='SUCCEEDED' AND d.action='box_delete'
 LEFT JOIN network_ipv4_retirement_queue q ON q.delete_operation_id=d.id AND q.project=b.project
 WHERE b.project=? AND b.state='DELETED' AND n.state='RELEASED'
 AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=b.project AND r.box_uuid=b.uuid AND r.incarnation=b.incarnation)
 AND NOT EXISTS(SELECT 1 FROM peer_grants p WHERE p.project=b.project AND (p.source_uuid=b.uuid OR p.target_uuid=b.uuid) AND `+retirementPeerInvalid+`)
 AND NOT EXISTS(SELECT 1 FROM network_ipv4_compactions cp WHERE cp.project=b.project AND cp.delete_operation_id!=d.id)
 AND NOT EXISTS(SELECT 1 FROM forward_claims f WHERE f.project=b.project AND f.box_uuid=b.uuid AND `+retirementForwardInvalid+`)
 ORDER BY COALESCE(q.attempt_order,0),b.deleted_at,b.uuid LIMIT ?`, project, limit)
	if e != nil {
		return nil, e
	}
	var out []string
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return nil, e
		}
		if !validHexID(id) {
			rows.Close()
			return nil, ErrResourceLocked
		}
		out = append(out, id)
	}
	if e = rows.Err(); e != nil {
		rows.Close()
		return nil, e
	}
	if e = rows.Close(); e != nil {
		return nil, e
	}
	var last int64
	if e = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(attempt_order),0) FROM network_ipv4_retirement_queue WHERE project=?", project).Scan(&last); e != nil {
		return nil, e
	}
	if last < 0 || last > (1<<63)-1-int64(limit) {
		return nil, ErrResourceLocked
	}
	for _, id := range out {
		last++
		r, e := tx.ExecContext(ctx, `INSERT INTO network_ipv4_retirement_queue(delete_operation_id,project,attempt_order) VALUES(?,?,?)
 ON CONFLICT(delete_operation_id) DO UPDATE SET attempt_order=excluded.attempt_order WHERE project=excluded.project`, id, project, last)
		if e != nil {
			return nil, e
		}
		if n, e := r.RowsAffected(); e != nil || n != 1 {
			return nil, ErrResourceLocked
		}
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return out, nil
}

// readRetiredForwardHistory is private evidence for the root verifier. It
// supplies only historical typed claims, never a caller-controlled port.
func (s *Store) readRetiredForwardHistory(ctx context.Context, c IPv4RetirementCandidate) ([]ForwardClaim, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT f.forward_id,f.box_uuid,f.incarnation,f.native_device,f.protocol,f.host_bind,f.host_port,f.guest_port,f.guest_ip,f.state,f.operation_id FROM forward_claims f WHERE f.project=? AND f.box_uuid=? ORDER BY f.forward_id`, c.Project, c.Binding.UUID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	var out []ForwardClaim
	for rows.Next() {
		f, e := scanForward(rows)
		if e != nil {
			return nil, e
		}
		if len(out) >= 256 || f.State != "REVOKED" || f.Incarnation != c.Binding.Incarnation || !validHexID(f.ForwardID) || !validHexID(f.OperationID) || f.NativeDevice != "bwpf-"+f.ForwardID[:16] || !validForwardCreate(ForwardCreateAdmission{f.Protocol, f.HostBind, f.HostPort, f.GuestPort}) || f.GuestIP != "127.0.0.1" {
			return nil, ErrResourceLocked
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// IPv4RetirementStatus exposes bounded same-Project counts and a safe
// failure class, never another actor's original operation or root inventory.
type IPv4RetirementStatus struct {
	Pending              int    `json:"pending"`
	ProtectedCompactions int    `json:"protected_compactions"`
	LastDeferredReason   string `json:"last_deferred_reason,omitempty"`
}

func validRetirementFailureClass(reason string) bool {
	switch reason {
	case "RETIREMENT_DEADLINE", "RETIREMENT_LEDGER_NOT_QUIESCENT_OR_INELIGIBLE", "RETIREMENT_NATIVE_PROOF_UNAVAILABLE", "RETIREMENT_BRIDGE_NOT_QUIESCENT", "RETIREMENT_NATIVE_IDENTITY_UNVERIFIED", "RETIREMENT_PROXY_NOT_CLEAR", "RETIREMENT_LEASE_NOT_CLEAR", "RETIREMENT_KERNEL_REFERENCE_UNVERIFIED", "RETIREMENT_CONNTRACK_NOT_CLEAR", "RETIREMENT_POLICY_UNVERIFIED", "RETIREMENT_COMPACTION_PROTECTED", "RETIREMENT_HELPER_NOT_CONFIGURED":
		return true
	}
	return false
}
func (s *Store) RecordIPv4RetirementDeferred(ctx context.Context, project, original, reason string) error {
	if s == nil || s.db == nil || s.auditReadOnly || !auditName(project) || !validHexID(original) || !validRetirementFailureClass(reason) {
		return ErrInvalid
	}
	_, e := s.db.ExecContext(ctx, "UPDATE network_ipv4_retirement_queue SET last_reason=?,last_attempt_at=? WHERE project=? AND delete_operation_id=?", reason, time.Now().UnixNano(), project, original)
	return e
}
func (s *Store) ReadIPv4RetirementStatus(ctx context.Context, project string) (IPv4RetirementStatus, error) {
	var out IPv4RetirementStatus
	if s == nil || s.db == nil || !auditName(project) || project == "default" {
		return out, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	out, e = readIPv4RetirementStatusTx(ctx, tx, project)
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}
func readIPv4RetirementStatusTx(ctx context.Context, tx *sql.Tx, project string) (IPv4RetirementStatus, error) {
	var out IPv4RetirementStatus
	var e error
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM owned_boxes b JOIN network_bindings n ON n.project=b.project AND n.box_uuid=b.uuid AND n.incarnation=b.incarnation AND n.nic='eth0' WHERE b.project=? AND b.state='DELETED' AND n.state='RELEASED' AND NOT EXISTS(SELECT 1 FROM network_ipv4_retirements r WHERE r.project=b.project AND r.box_uuid=b.uuid AND r.incarnation=b.incarnation)`, project).Scan(&out.Pending); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM network_ipv4_compactions WHERE project=?", project).Scan(&out.ProtectedCompactions); e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, "SELECT last_reason FROM network_ipv4_retirement_queue WHERE project=? AND last_reason!='' ORDER BY last_attempt_at DESC,attempt_order DESC LIMIT 1", project).Scan(&out.LastDeferredReason)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	if out.LastDeferredReason != "" && !validRetirementFailureClass(out.LastDeferredReason) {
		return IPv4RetirementStatus{}, ErrResourceLocked
	}
	return out, nil
}

// ReadIPv4RetirementRollbackWitness is a stable ordered digest of all
// retired native originals. Require no protected mutation; a rollback may
// never forget a root ACL compaction or release a fresh address fence.
func (s *Store) ReadIPv4RetirementRollbackWitness(ctx context.Context, project string) (string, error) {
	if s == nil || s.db == nil || !auditName(project) || project == "default" {
		return "", ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	var pending int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM network_ipv4_compactions").Scan(&pending); e != nil {
		return "", e
	}
	if pending != 0 {
		return "", ErrResourceLocked
	}
	var foreign int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM network_ipv4_retirements WHERE project!=?", project).Scan(&foreign); e != nil {
		return "", e
	}
	if foreign != 0 {
		return "", ErrResourceLocked
	}
	rows, e := tx.QueryContext(ctx, `SELECT project,box_uuid,incarnation,bridge,acl,mac,ipv4,ipv6,create_operation_id,delete_operation_id,native_proof_sha256,peer_compaction_sha256,retired_at FROM network_ipv4_retirements WHERE project=? ORDER BY box_uuid,incarnation`, project)
	if e != nil {
		return "", e
	}
	defer rows.Close()
	h := sha256.New()
	fmt.Fprintln(h, "BoxWeave native retirement rollback witness v1", project)
	for rows.Next() {
		var fields [12]string
		var when int64
		args := make([]any, 0, 13)
		for i := range fields {
			args = append(args, &fields[i])
		}
		args = append(args, &when)
		if e = rows.Scan(args...); e != nil {
			return "", e
		}
		b, e := json.Marshal(fields)
		if e != nil {
			return "", e
		}
		fmt.Fprintf(h, "%s %d\n", b, when)
	}
	if e = rows.Err(); e != nil {
		return "", e
	}
	if e = rows.Close(); e != nil {
		return "", e
	}
	if e = tx.Commit(); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
