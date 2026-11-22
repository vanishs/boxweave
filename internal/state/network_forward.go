package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

// ForwardCreateAdmission is an internal typed ledger reservation; a matching
// original Operation and listener are committed in the same transaction.
type ForwardCreateAdmission struct {
	Protocol  string
	HostBind  string
	HostPort  int
	GuestPort int
}

type ForwardClaim struct {
	ForwardID    string `json:"forward_id"`
	BoxID        string `json:"box_id"`
	Incarnation  string `json:"incarnation"`
	NativeDevice string `json:"native_device"`
	Protocol     string `json:"protocol"`
	HostBind     string `json:"host_bind"`
	HostPort     int    `json:"host_port"`
	GuestPort    int    `json:"guest_port"`
	GuestIP      string `json:"guest_ip"`
	State        string `json:"state"`
	OperationID  string `json:"operation_id"`
}

func validForwardCreate(f ForwardCreateAdmission) bool {
	ip := net.ParseIP(f.HostBind)
	return (f.Protocol == "tcp" || f.Protocol == "udp") &&
		f.HostPort >= 1 && f.HostPort <= 65535 && f.GuestPort >= 1 && f.GuestPort <= 65535 &&
		ip != nil && f.HostBind == ip.String() && !ip.IsMulticast() &&
		!ip.IsLinkLocalUnicast() && !ip.Equal(net.IPv4bcast) && (!ip.IsLoopback() || f.HostBind == "127.0.0.1" || f.HostBind == "::1")
}

// stageForwardAdmission is invoked only inside Store.Admit's transaction.
func stageForwardAdmission(ctx context.Context, tx *sql.Tx, a Admission, operationID string) error {
	if a.Action != "port_forward_create" && a.Action != "port_forward_delete" {
		if a.ForwardCreate != nil || a.ForwardDeleteID != "" {
			return ErrInvalid
		}
		return nil
	}
	if a.BoxUUID == "" || a.Incarnation == "" {
		return ErrInvalid
	}
	var owned int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM owned_boxes WHERE project=? AND uuid=? AND incarnation=? AND state='OWNED'", a.Layer, a.BoxUUID, a.Incarnation).Scan(&owned); err != nil {
		return err
	}
	if owned != 1 {
		return ErrResourceLocked
	}
	if a.Action == "port_forward_create" {
		if a.ForwardCreate == nil || a.ForwardDeleteID != "" || !validForwardCreate(*a.ForwardCreate) {
			return ErrInvalid
		}
		f := a.ForwardCreate
		var busy int
		// Either wildcard conservatively reserves the protocol/port across
		// both address families (IPv6 may also listen for mapped IPv4).
		// All non-revoked claims, including uncertain originals, participate.
		err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM forward_claims WHERE protocol=? AND host_port=? AND state!='REVOKED' AND (host_bind=? OR host_bind IN ('0.0.0.0','::') OR ? IN ('0.0.0.0','::'))", f.Protocol, f.HostPort, f.HostBind, f.HostBind).Scan(&busy)
		if err != nil {
			return err
		}
		if busy != 0 {
			return ErrResourceLocked
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO forward_claims(forward_id,project,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state,operation_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)",
			operationID, a.Layer, a.BoxUUID, a.Incarnation, "bwpf-"+operationID[:16], f.Protocol, f.HostBind, f.HostPort, f.GuestPort, "127.0.0.1", "PENDING", operationID)
		if err != nil && strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrResourceLocked
		}
		return err
	}
	if a.ForwardCreate != nil || len(a.ForwardDeleteID) != 32 {
		return ErrInvalid
	}
	res, err := tx.ExecContext(ctx, "UPDATE forward_claims SET state='PENDING',operation_id=? WHERE forward_id=? AND project=? AND box_uuid=? AND incarnation=? AND state='CONFIRMED'",
		operationID, a.ForwardDeleteID, a.Layer, a.BoxUUID, a.Incarnation)
	if err != nil {
		return err
	}
	changed, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrResourceLocked
	}
	return nil
}

func scanForward(row interface{ Scan(...any) error }) (ForwardClaim, error) {
	var f ForwardClaim
	err := row.Scan(&f.ForwardID, &f.BoxID, &f.Incarnation, &f.NativeDevice, &f.Protocol, &f.HostBind, &f.HostPort, &f.GuestPort, &f.GuestIP, &f.State, &f.OperationID)
	return f, err
}

const forwardColumns = "forward_id,box_uuid,incarnation,native_device,protocol,host_bind,host_port,guest_port,guest_ip,state,operation_id"

func (s *Store) GetForward(ctx context.Context, layer, forwardID string) (ForwardClaim, error) {
	if layer == "" || len(forwardID) != 32 {
		return ForwardClaim{}, ErrInvalid
	}
	f, err := scanForward(s.db.QueryRowContext(ctx, "SELECT "+forwardColumns+" FROM forward_claims WHERE project=? AND forward_id=?", layer, forwardID))
	if errors.Is(err, sql.ErrNoRows) {
		return ForwardClaim{}, ErrNotFound
	}
	return f, err
}

// ListForwards is a bounded same-layer durable projection; current native
// verification is separately required before claiming kernel reachability.
func (s *Store) ListForwards(ctx context.Context, layer, boxID, cursor string, limit int) ([]ForwardClaim, string, error) {
	if layer == "" || limit < 1 || limit > 50 || (cursor != "" && len(cursor) != 32) {
		return nil, "", ErrInvalid
	}
	q := "SELECT " + forwardColumns + " FROM forward_claims WHERE project=? AND state!='REVOKED' AND forward_id>?"
	args := []any{layer, cursor}
	if boxID != "" {
		q += " AND box_uuid=?"
		args = append(args, boxID)
	}
	q += " ORDER BY forward_id LIMIT ?"
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := make([]ForwardClaim, 0, limit)
	for rows.Next() {
		f, e := scanForward(rows)
		if e != nil {
			return nil, "", e
		}
		out = append(out, f)
	}
	if err = rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = out[len(out)-1].ForwardID
	}
	return out, next, nil
}

// FinalizeVerifiedForward is only called after an independently proved Incus
// terminal and exact device postcondition. Ledger, original and event commit
// together or not at all; A4 UNKNOWN settlement uses the same transaction helper.
func (s *Store) FinalizeVerifiedForward(ctx context.Context, operationID string, create bool) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = finishVerifiedForwardTx(ctx, tx, operationID, create, "VERIFYING", ""); err != nil {
		return err
	}
	return tx.Commit()
}

func finishVerifiedForwardTx(ctx context.Context, tx *sql.Tx, operationID string, create bool, expected, detail string) error {
	if expected != "VERIFYING" && expected != "UNKNOWN" {
		return ErrInvalid
	}
	var err error
	action := "port_forward_delete"
	state := "REVOKED"
	if create {
		action = "port_forward_create"
		state = "CONFIRMED"
	}
	var gotAction, gotState string
	err = tx.QueryRowContext(ctx, "SELECT action,state FROM operations WHERE id=?", operationID).Scan(&gotAction, &gotState)
	if err != nil {
		return err
	}
	if gotAction != action || gotState != expected {
		return ErrState
	}
	res, err := tx.ExecContext(ctx, "UPDATE forward_claims SET state=? WHERE operation_id=? AND state='PENDING'", state, operationID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrState
	}
	now := time.Now().UnixNano()
	res, err = tx.ExecContext(ctx, "UPDATE operations SET state='SUCCEEDED',error_class='',updated_at=? WHERE id=? AND state=?", now, operationID, expected)
	if err != nil {
		return err
	}
	affected, err = res.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)", operationID, now, "SUCCEEDED", detail); err != nil {
		return err
	}
	return nil
}

// ForwardReplayBox is used only to reconstruct authenticated original
// receipts. A subsequent new admission still requires a live OWNED row.
func (s *Store) ForwardReplayBox(ctx context.Context, layer, uuid string) (OwnedBox, error) {
	var b OwnedBox
	err := s.db.QueryRowContext(ctx, "SELECT uuid,project,name,incarnation,state FROM owned_boxes WHERE project=? AND uuid=?", layer, uuid).Scan(&b.UUID, &b.Project, &b.Name, &b.Incarnation, &b.State)
	if errors.Is(err, sql.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

// Atomically settle the reservation only on a proved no-effect transition.
// Never run this for an uncertain/dispatched original.
func settleForwardNoEffectTx(ctx context.Context, tx *sql.Tx, operationID string) error {
	var action, st string
	if err := tx.QueryRowContext(ctx, "SELECT action,state FROM operations WHERE id=?", operationID).Scan(&action, &st); err != nil {
		return err
	}
	if st != "FAILED" && st != "CANCELLED" {
		return ErrState
	}
	var res sql.Result
	var err error
	switch action {
	case "port_forward_create":
		res, err = tx.ExecContext(ctx, "UPDATE forward_claims SET state='REVOKED' WHERE operation_id=? AND state='PENDING'", operationID)
	case "port_forward_delete":
		res, err = tx.ExecContext(ctx, "UPDATE forward_claims SET state='CONFIRMED' WHERE operation_id=? AND state='PENDING'", operationID)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return fmt.Errorf("forward original %s not staged: %w", operationID, ErrState)
	}
	return nil
}
