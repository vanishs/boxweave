package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/vanishs/boxweave/internal/networklimits"
)

// DiagnosticPeerCapacity is a SQLite ledger observation, not the Root
// chain length. Retained directions include revoked rows. Revocation
// reserve is computed by the same rule as scoped admission.
type DiagnosticPeerCapacity struct {
	RetainedDirections       int  `json:"retained_directions"`
	ActiveDirections         int  `json:"active_directions"`
	PendingOrUnknown         int  `json:"pending_or_unknown"`
	DirectionLimit           int  `json:"direction_limit"`
	CompletedLedgerEvents    int  `json:"completed_ledger_events"`
	HistoryLimit             int  `json:"history_limit"`
	NewGrantHistoryAvailable bool `json:"new_grant_history_available"`
	NewDirectionAvailable    bool `json:"new_direction_available"`
}

func readDiagnosticPeerCapacity(ctx context.Context, tx *sql.Tx, project string) (DiagnosticPeerCapacity, error) {
	out := DiagnosticPeerCapacity{DirectionLimit: 32, HistoryLimit: networklimits.ScopedPeerHistoryLimit}
	e := tx.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(status='CONFIRMED'),0),coalesce(sum(status IN ('PENDING','UNKNOWN')),0) FROM peer_grants WHERE project=?`, project).Scan(&out.RetainedDirections, &out.ActiveDirections, &out.PendingOrUnknown)
	if e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operation_events e JOIN operations o ON o.id=e.operation_id
 WHERE o.layer=? AND o.action='box_peer_set' AND o.state='SUCCEEDED' AND e.state='SUCCEEDED'
 AND e.safe_detail IN ('SCOPED_PEER_V2_GRANT_KERNEL_VERIFIED','SCOPED_PEER_V2_REVOCATION_KERNEL_VERIFIED')`, project).Scan(&out.CompletedLedgerEvents)
	if e != nil {
		return out, e
	}
	out.NewGrantHistoryAvailable = networklimits.ScopedPeerHistoryCanAppend(out.CompletedLedgerEvents, out.ActiveDirections, true)
	out.NewDirectionAvailable = out.RetainedDirections < out.DirectionLimit
	return out, nil
}
func (s *Store) ReadDiagnosticPeerCapacity(ctx context.Context, project string) (DiagnosticPeerCapacity, error) {
	if !auditName(project) || project == "default" {
		return DiagnosticPeerCapacity{}, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return DiagnosticPeerCapacity{}, e
	}
	defer tx.Rollback()
	out, e := readDiagnosticPeerCapacity(ctx, tx, project)
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}

type DiagnosticOperation struct {
	ID                 string `json:"original_id"`
	Action             string `json:"action"`
	State              string `json:"state"`
	Cause              string `json:"cause_domain"`
	ClaimProtection    string `json:"claim_protection"`
	LiveClaims         int    `json:"live_claims"`
	NativeReceipt      bool   `json:"native_receipt_recorded"`
	GuestOutcome       string `json:"guest_outcome"`
	GuestExitCode      *int   `json:"guest_exit_code,omitempty"`
	OutputMetadata     string `json:"output_metadata"`
	RecoveryOriginalID string `json:"recovery_original_id,omitempty"`

	RecoveryRoute string `json:"recovery_route"`
}
type DiagnosticOAuth struct {
	OriginConfigured   bool `json:"origin_configured"`
	PasswordConfigured bool `json:"password_configured"`
	Enabled            bool `json:"enabled"`
}
type DiagnosticForward struct {
	ID, UUID, Incarnation, Name, Device, Protocol, Bind string
	HostPort, GuestPort                                 int
}
type DiagnosticLedger struct {
	// Only redacted summary is serialized to the public report. Native
	// expectations are internal to the local restricted read worker.
	Active                     []DiagnosticOperation `json:"active_operations"`
	ActiveTruncated            bool                  `json:"active_truncated"`
	ActiveCount                int                   `json:"active_count"`
	HistoricalFailures         int                   `json:"historical_failures"`
	HistoricalRecoveryAttempts int                   `json:"historical_recovery_attempts"`
	GuestExecNonzeroCount      int                   `json:"guest_exec_nonzero_count"`
	GuestExecNonzeroSample     []DiagnosticOperation `json:"guest_exec_nonzero_sample"`

	HistorySample    []DiagnosticOperation `json:"history_sample"`
	HistoryTruncated bool                  `json:"history_truncated"`

	ReleasedUnknown   int                     `json:"released_unknown"`
	Recovered         int                     `json:"recovered"`
	Activity          ActivityResult          `json:"-"`
	OAuth             DiagnosticOAuth         `json:"oauth"`
	PeerCapacity      *DiagnosticPeerCapacity `json:"peer_capacity,omitempty"`
	LegacyRetirement  *IPv4RetirementStatus   `json:"legacy_retirement,omitempty"`
	RetiredIdentities int                     `json:"retired_identities"`
	RetiredLimit      int                     `json:"retired_limit,omitempty"`
	ForwardConfirmed  int                     `json:"forward_confirmed"`
	ForwardUnresolved int                     `json:"forward_unresolved"`
	Network           NativeNetworkLedger     `json:"-"`
	Forwards          []DiagnosticForward     `json:"-"`
	ForwardTruncated  bool                    `json:"forward_truncated"`
	NetworkObserved   bool                    `json:"network_observed"`
	HoldsObserved     bool                    `json:"holds_observed"`
	HeldIPv4          int                     `json:"held_ipv4"`
	HeldIPv6          int                     `json:"held_ipv6"`
	HeldMAC           int                     `json:"held_mac"`
	HoldsDigest       string                  `json:"-"`
}

func diagnosticAction(a string) string {
	switch a {
	case "box_create", "box_delete", "box_start", "box_stop", "box_update", "box_exec", "box_file_apply", "box_peer_set", "port_forward_create", "port_forward_delete", "operation_recover":
		return a
	}
	return "OTHER"
}
func (s *DiagnosticSnapshot) ReadLedger(ctx context.Context, project, mode, bridge string) (DiagnosticLedger, error) {
	out := DiagnosticLedger{Active: []DiagnosticOperation{}, HistorySample: []DiagnosticOperation{}, GuestExecNonzeroSample: []DiagnosticOperation{}}
	if !auditName(project) || project == "default" {
		return out, ErrInvalid
	}
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	out, e = readDiagnosticLedgerTx(ctx, tx, project, mode, bridge)
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}
func readDiagnosticLedgerTx(ctx context.Context, tx *sql.Tx, project, mode, bridge string) (DiagnosticLedger, error) {
	out := DiagnosticLedger{Active: []DiagnosticOperation{}, HistorySample: []DiagnosticOperation{}, GuestExecNonzeroSample: []DiagnosticOperation{}}
	var e error
	// Reuse the Activity SQL without cursor secrets or fake actor identity.
	q := ActivityQuery{Layer: project, View: "issues", Limit: 50}
	var cursor activityCursor
	if e = tx.QueryRowContext(ctx, "SELECT coalesce(max(rowid),0) FROM operations WHERE layer=?", project).Scan(&cursor.MaxRowID); e != nil {
		return out, e
	}
	if e = readActivityIssuesAt(ctx, tx, q, &out.Activity, cursor, nil); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(o.state='FAILED'),0),
 coalesce(sum(o.state='UNKNOWN' AND o.action NOT IN ('box_peer_set','port_forward_create','port_forward_delete') AND NOT EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.layer=o.layer AND c.resource_key='delete-project:'||o.layer) AND EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED')),0),
 coalesce(sum(o.state='SUCCEEDED' AND EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RECOVERY_SETTLED')),0)
 FROM operations o WHERE layer=?`, project).Scan(new(int), &out.HistoricalFailures, &out.ReleasedUnknown, &out.Recovered); e != nil {
		return out, e
	}
	const live = "('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')"
	const retained = " AND (state!='UNKNOWN' OR action IN ('box_peer_set','port_forward_create','port_forward_delete') OR EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=operations.id AND c.layer=operations.layer AND c.resource_key='delete-project:'||operations.layer) OR NOT EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=operations.id AND a.decision='RELEASED'))"

	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE layer=? AND state IN "+live+retained, project).Scan(&out.ActiveCount); e != nil {
		return out, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT id,action,state,error_class,native_operation_id FROM operations WHERE layer=? AND state IN "+live+retained+" ORDER BY created_at,id LIMIT 51", project)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var d DiagnosticOperation
		var rawAction, rawError, native string
		if e = rows.Scan(&d.ID, &rawAction, &d.State, &rawError, &native); e != nil {
			break
		}
		if len(out.Active) == 50 {
			out.ActiveTruncated = true
			break
		}
		if !validHexID(d.ID) {
			e = ErrResourceLocked
			break
		}
		d.Action = diagnosticAction(rawAction)
		d.Cause = OperationCause(Receipt{Action: d.Action, State: d.State, ErrorClass: rawError})
		d.NativeReceipt = native != ""
		d.GuestOutcome = "NOT_APPLICABLE"
		d.OutputMetadata = "NOT_APPLICABLE"
		if d.Action == "box_exec" {
			d.GuestOutcome = "UNPROVED"
			d.OutputMetadata = "UNPROVED"
		}
		out.Active = append(out.Active, d)
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
	for i := range out.Active {
		d := &out.Active[i]
		nav, e := readRecoveryNavigation(ctx, tx, project, d.ID)
		if e != nil {
			return out, e
		}
		d.ClaimProtection = nav.ClaimProtection
		d.RecoveryRoute = nav.Disposition
		if d.State == "UNKNOWN" && d.Action == "box_peer_set" {
			target, e := readRecoveryTarget(ctx, tx, project, d.ID)
			if e != nil {
				return out, e
			}
			if target.ScopedPeer {
				d.RecoveryRoute = "MCP_RECOVERY_INSPECTION_REQUIRED"
			}
		}
		if d.ClaimProtection != "REVIEWED_RELEASED" {
			if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operation_resource_claims WHERE layer=? AND operation_id=?", project, d.ID).Scan(&d.LiveClaims); e != nil {
				return out, e
			}
		}
	}
	// History samples include terminal Exec metadata, not guest output or
	// arbitrary request intent. A failed recovery is distinct from its original.
	out.HistoryTruncated = out.Activity.HistoryTruncated
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE layer=? AND action='operation_recover' AND state='FAILED'", project).Scan(&out.HistoricalRecoveryAttempts); e != nil {
		return out, e
	}
	for _, h := range out.Activity.History {
		if !validHexID(h.OperationID) {
			return out, ErrResourceLocked
		}
		var d DiagnosticOperation
		var action, rawError, native string
		d.ID = h.OperationID
		if e = tx.QueryRowContext(ctx, "SELECT action,state,error_class,native_operation_id FROM operations WHERE layer=? AND id=?", project, h.OperationID).Scan(&action, &d.State, &rawError, &native); e != nil {
			return out, e
		}
		d.Action = diagnosticAction(action)
		d.Cause = OperationCause(Receipt{Action: d.Action, State: d.State, ErrorClass: rawError})
		d.NativeReceipt = native != ""
		d.GuestOutcome = "NOT_APPLICABLE"
		d.OutputMetadata = "NOT_APPLICABLE"
		nav, e := readRecoveryNavigation(ctx, tx, project, d.ID)
		if e != nil {
			return out, e
		}
		d.ClaimProtection = nav.ClaimProtection
		d.RecoveryRoute = nav.Disposition
		if nav.Original != nil {
			d.RecoveryOriginalID = nav.Original.OriginalID
		}
		if d.Action == "box_exec" {
			d.GuestOutcome = "UNPROVED"
			d.OutputMetadata = "UNPROVED"
			if d.State == "SUCCEEDED" {
				var exit sql.NullInt64
				var metadata bool
				if e = tx.QueryRowContext(ctx, "SELECT exit_code,stdout_path!='' OR stderr_path!='' FROM exec_outputs WHERE operation_id=?", d.ID).Scan(&exit, &metadata); e == nil {
					d.OutputMetadata = "RECORDED_NOT_LIVE_VERIFIED"
					if exit.Valid {
						v := int(exit.Int64)
						d.GuestExitCode = &v
						d.GuestOutcome = "EXIT_NONZERO"
						if v == 0 {
							d.GuestOutcome = "EXIT_ZERO"
						}
					}
				} else if e == sql.ErrNoRows {
					d.OutputMetadata = "NOT_RECORDED"
				} else {
					return out, e
				}
			}
		}
		out.HistorySample = append(out.HistorySample, d)
	}
	// A SUCCEEDED Exec records native completion, not command success.
	const nonzero = " FROM operations o JOIN exec_outputs x ON x.operation_id=o.id WHERE o.layer=? AND o.action='box_exec' AND o.state='SUCCEEDED' AND x.exit_code!=0"
	if e = tx.QueryRowContext(ctx, "SELECT count(*)"+nonzero, project).Scan(&out.GuestExecNonzeroCount); e != nil {
		return out, e
	}
	rows, e = tx.QueryContext(ctx, "SELECT o.id,o.native_operation_id!='',x.exit_code"+nonzero+" ORDER BY o.updated_at DESC,o.id DESC LIMIT 5", project)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		d := DiagnosticOperation{Action: "box_exec", State: "SUCCEEDED", Cause: "NONE_RECORDED", ClaimProtection: "NOT_UNKNOWN", GuestOutcome: "EXIT_NONZERO", OutputMetadata: "RECORDED_NOT_LIVE_VERIFIED", RecoveryRoute: "NO_RECOVERY_REQUIRED"}
		var exit int
		if e = rows.Scan(&d.ID, &d.NativeReceipt, &exit); e != nil {
			break
		}
		if !validHexID(d.ID) {
			e = ErrResourceLocked
			break
		}
		d.GuestExitCode = &exit
		out.GuestExecNonzeroSample = append(out.GuestExecNonzeroSample, d)
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
	if e = tx.QueryRowContext(ctx, "SELECT public_origin!='',password_hash!='' FROM oauth_config WHERE id=1").Scan(&out.OAuth.OriginConfigured, &out.OAuth.PasswordConfigured); e != nil {
		return out, e
	}
	out.OAuth.Enabled = out.OAuth.OriginConfigured && out.OAuth.PasswordConfigured
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM owned_boxes WHERE project=? AND state='DELETED'", project).Scan(&out.RetiredIdentities); e != nil {
		return out, e
	}
	if mode == "SCOPED_V2" {
		cap, e := readDiagnosticPeerCapacity(ctx, tx, project)
		if e != nil {
			return out, e
		}
		out.PeerCapacity = &cap
		out.RetiredLimit = scopedRetiredHistoryLimit
	}
	if e = tx.QueryRowContext(ctx, "SELECT coalesce(sum(state='CONFIRMED'),0),coalesce(sum(state IN ('PENDING','UNKNOWN')),0) FROM forward_claims WHERE project=?", project).Scan(&out.ForwardConfirmed, &out.ForwardUnresolved); e != nil {
		return out, e
	}
	rows, e = tx.QueryContext(ctx, `SELECT f.forward_id,f.box_uuid,f.incarnation,coalesce(b.name,''),f.native_device,f.protocol,f.host_bind,f.host_port,f.guest_port
 FROM forward_claims f LEFT JOIN owned_boxes b ON b.project=f.project AND b.uuid=f.box_uuid AND b.incarnation=f.incarnation AND b.state='OWNED'
 WHERE f.project=? AND f.state='CONFIRMED' ORDER BY f.forward_id LIMIT 51`, project)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var f DiagnosticForward
		if e = rows.Scan(&f.ID, &f.UUID, &f.Incarnation, &f.Name, &f.Device, &f.Protocol, &f.Bind, &f.HostPort, &f.GuestPort); e != nil {
			break
		}
		if len(out.Forwards) == 50 {
			out.ForwardTruncated = true
			break
		}
		if !validHexID(f.ID) {
			e = ErrResourceLocked
			break
		}
		out.Forwards = append(out.Forwards, f)
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
	// Observe every predicate in the same transaction, including live commit.
	if mode == "LEGACY" {
		r, e := readIPv4RetirementStatusTx(ctx, tx, project)
		if e != nil {
			return out, e
		}
		out.LegacyRetirement = &r
	}
	if l, e := readNativeNetworkLedgerTx(ctx, tx, project); e == nil {
		out.Network = l
		out.NetworkObserved = true
	}
	if bridge != "" {
		if h, e := readCreateAddressHoldsTx(ctx, tx, project, bridge); e == nil {
			out.HoldsObserved = true
			out.HeldMAC = len(h.MACs)
			raw, _ := json.Marshal(h)
			digest := sha256.Sum256(raw)
			out.HoldsDigest = hex.EncodeToString(digest[:])
			for a := range h.IPs {
				if a.Is4() {
					out.HeldIPv4++
				} else {
					out.HeldIPv6++
				}
			}
		}
	}
	return out, nil
}

// ObservationDigest binds internal cross-source inputs; not an approval
// digest. Exclude volatile capture time and secrets, compare after scan.
func (d DiagnosticLedger) ObservationDigest() string {
	raw, _ := json.Marshal(struct {
		N                                         NativeNetworkLedger
		F                                         []DiagnosticForward
		A                                         []DiagnosticOperation
		NOK                                       bool
		Truncated                                 bool
		Holds                                     string
		HoldsOK                                   bool
		ForwardCount, ForwardUnknown, ActiveCount int
	}{d.Network, d.Forwards, d.Active, d.NetworkObserved, d.ForwardTruncated, d.HoldsDigest, d.HoldsObserved, d.ForwardConfirmed, d.ForwardUnresolved, d.ActiveCount})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func (s *DiagnosticSnapshot) ReadAddressHolds(ctx context.Context, project, bridge string) (networklimits.AddressHolds, error) {
	return s.store.ReadCreateAddressHolds(ctx, project, bridge)
}

// SafeDiagnosticRef never emits untrusted DB text or arbitrary addresses.
func SafeDiagnosticRef(raw string) string {
	if validHexID(raw) || validActivityBoxID(raw) {
		return raw
	}
	h := sha256.Sum256([]byte(raw))
	return "ref-" + hex.EncodeToString(h[:])
}
func SafeDiagnosticProject(project string) bool {
	return strings.HasPrefix(project, "user-") && auditName(project) && len(project) <= 64
}
