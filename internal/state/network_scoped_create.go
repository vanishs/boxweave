package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/vanishs/boxweave/internal/peerscope"
)

// V2 is explicitly distinct from the legacy directly-starting Create.
// Options are controller-validated typed JSON, never Root native input.
type ScopedCreateIntent struct {
	Schema        int             `json:"schema"`
	Project       string          `json:"project"`
	Bridge        string          `json:"bridge"`
	Name          string          `json:"name"`
	Start         bool            `json:"start"`
	Internet      bool            `json:"internet"`
	BootAutostart bool            `json:"boot_autostart"`
	CPU           int             `json:"cpu"`
	MemoryMiB     int64           `json:"memory_mib"`
	DiskGiB       int64           `json:"disk_gib"`
	Image         string          `json:"image,omitempty"`
	Nesting       bool            `json:"nesting"`
	Options       json.RawMessage `json:"instance_options"`
}

type ScopedCreateOriginalProof struct {
	OriginalID, NativeID string
	Intent               ScopedCreateIntent
	Quarantine           VerifiedCreateBinding
}

// Read the native Create receipt and exact pre-reserved static tuple in one
// snapshot. No native UUID is supplied by the restricted actor.
func readScopedCreateOriginalTx(ctx context.Context, tx *sql.Tx, id, project, bridge string) (ScopedCreateOriginalProof, error) {
	var zero ScopedCreateOriginalProof
	if !validHexID(id) || !strings.HasPrefix(project, "user-") || !auditName(project) || !auditName(bridge) {
		return zero, ErrInvalid
	}
	var action, phase, layer, resource, uuid, inc, nativeID, actor, token string
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT action,state,layer,resource_key,box_uuid,incarnation,native_operation_id,actor_id,token_id,safe_intent FROM operations WHERE id=?`, id).Scan(&action, &phase, &layer, &resource, &uuid, &inc, &nativeID, &actor, &token, &raw)
	if err != nil {
		return zero, err
	}
	var intent ScopedCreateIntent
	if json.Unmarshal(raw, &intent) != nil {
		return zero, ErrResourceLocked
	}
	canonical, err := json.Marshal(intent)
	if err != nil || !bytes.Equal(canonical, raw) || intent.Schema != 2 || intent.Project != project || intent.Bridge != bridge ||
		!auditName(intent.Name) || resource != "box:"+intent.Name || action != "box_create" || phase != "VERIFYING" ||
		layer != project || uuid != "" || inc != "" || nativeID == "" || actor == "" || token == "" ||
		intent.CPU < 1 || intent.CPU > 64 || intent.MemoryMiB < 128 || intent.MemoryMiB > 262144 || intent.DiskGiB < 2 || intent.DiskGiB > 2048 ||
		len(intent.Options) == 0 || len(intent.Options) > 4096 || intent.Options[0] != '{' {
		return zero, ErrResourceLocked
	}
	active, err := activeCredentialForActor(ctx, tx, actor, token)
	if err != nil || !active {
		return zero, ErrRevoked
	}
	rows, err := tx.QueryContext(ctx, `SELECT resource_key,layer FROM operation_resource_claims WHERE operation_id=?`, id)
	if err != nil {
		return zero, err
	}
	defer rows.Close()
	pin := VerifiedCreateBinding{Bridge: bridge}
	prefix := "create-pin:" + bridge + ":"
	seen := map[string]bool{}
	for rows.Next() {
		var key, l string
		if err = rows.Scan(&key, &l); err != nil {
			return zero, err
		}
		if l != project || seen[key] {
			return zero, ErrResourceLocked
		}
		seen[key] = true
		if key == resource {
			continue
		}
		if !strings.HasPrefix(key, prefix) {
			return zero, ErrResourceLocked
		}
		kind, value, ok := strings.Cut(strings.TrimPrefix(key, prefix), ":")
		if !ok {
			return zero, ErrResourceLocked
		}
		switch kind {
		case "acl":
			if pin.ACL != "" {
				return zero, ErrResourceLocked
			}
			pin.ACL = value
		case "mac":
			if pin.MAC != "" {
				return zero, ErrResourceLocked
			}
			pin.MAC = value
		case "v4":
			if pin.IPv4 != "" {
				return zero, ErrResourceLocked
			}
			pin.IPv4 = value
		case "v6":
			if pin.IPv6 != "" {
				return zero, ErrResourceLocked
			}
			pin.IPv6 = value
		default:
			return zero, ErrResourceLocked
		}
	}
	if err = rows.Err(); err != nil {
		return zero, err
	}
	if len(seen) != 5 || !seen[resource] || !pin.valid() || pin.ACL != peerscope.QuarantineACL {
		return zero, ErrResourceLocked
	}
	if err = rows.Close(); err != nil {
		return zero, err
	}
	var busy int
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND id<>? AND action IN ('box_create','box_delete','box_peer_set','box_update') AND state IN ('DISPATCHING','RUNNING','VERIFYING','UNKNOWN')`, project, id).Scan(&busy); err != nil || busy != 0 {
		return zero, ErrResourceLocked
	}
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM peer_grants WHERE project=?`, project).Scan(&busy); err != nil || busy != 0 {
		return zero, ErrResourceLocked
	}
	return ScopedCreateOriginalProof{OriginalID: id, NativeID: nativeID, Intent: intent, Quarantine: pin}, nil
}

// ReadRootScopedCreateOriginal exposes only an exact original on a protected
// read-only ledger connection, never an arbitrary native mutation request.
func (s *Store) ReadRootScopedCreateOriginal(ctx context.Context, id, project, bridge string) (ScopedCreateOriginalProof, error) {
	var zero ScopedCreateOriginalProof
	if s == nil || s.db == nil || !s.IsNetworkAuditReadOnly() {
		return zero, ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback()
	p, err := readScopedCreateOriginalTx(ctx, tx, id, project, bridge)
	if err != nil {
		return zero, err
	}
	if err = tx.Commit(); err != nil {
		return zero, err
	}
	return p, nil
}

// Exchange only the provisional ACL claim after independently verifying
// the native UUID and derived ACL. Address and MAC holds never change.
// This must be within the same transaction as original Create finalization.
func exchangeScopedCreateACL(ctx context.Context, tx *sql.Tx, id, project, name, uuid string, b VerifiedCreateBinding) error {
	p, err := readScopedCreateOriginalTx(ctx, tx, id, project, b.Bridge)
	if err != nil {
		return err
	}
	acl, err := peerscope.ACLName(project, b.Bridge, uuid)
	if err != nil || b.ACL != acl || p.Intent.Name != name || p.Quarantine.Bridge != b.Bridge ||
		p.Quarantine.MAC != b.MAC || p.Quarantine.IPv4 != b.IPv4 || p.Quarantine.IPv6 != b.IPv6 ||
		p.Quarantine.ACL == b.ACL {
		return ErrResourceLocked
	}
	oldKey := p.Quarantine.reservationKeys()[3]
	newKey := b.reservationKeys()[3]
	var n int
	if err = tx.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM network_bindings WHERE bridge=? AND acl=?)+(SELECT COUNT(*) FROM operation_resource_claims WHERE resource_key=?)`, b.Bridge, b.ACL, newKey).Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return ErrResourceLocked
	}
	res, err := tx.ExecContext(ctx, `UPDATE operation_resource_claims SET resource_key=? WHERE operation_id=? AND layer=? AND resource_key=?`, newKey, id, project, oldKey)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err != nil || n != 1 {
		return ErrResourceLocked
	}
	return nil
}
func scopedCreateIntentSchema(ctx context.Context, tx *sql.Tx, id string) (int, error) {
	var raw []byte
	if err := tx.QueryRowContext(ctx, "SELECT safe_intent FROM operations WHERE id=?", id).Scan(&raw); err != nil {
		return 0, err
	}
	var hdr struct {
		Schema int `json:"schema"`
	}
	if json.Unmarshal(raw, &hdr) != nil {
		return 0, ErrInvalid
	}
	return hdr.Schema, nil
}

func scopedCreateAdmissionBarrier(ctx context.Context, tx *sql.Tx, a Admission) error {
	if a.Action != "box_create" && a.Action != "box_delete" && a.Action != "box_update" && a.Action != "box_peer_set" {
		return nil
	}
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND action='box_create' AND json_valid(safe_intent) AND json_extract(safe_intent,'$.schema')=2 AND state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')`, a.Layer).Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return ErrResourceLocked
	}
	if a.Action != "box_create" {
		return nil
	}
	var h struct {
		Schema int `json:"schema"`
	}
	if json.Unmarshal(a.SafeIntent, &h) != nil || h.Schema != 2 {
		return nil
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations WHERE layer=? AND action IN ('box_create','box_delete','box_update','box_peer_set') AND state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')`, a.Layer).Scan(&n); err != nil {
		return err
	}
	if n != 0 {
		return ErrResourceLocked
	}
	return nil
}
