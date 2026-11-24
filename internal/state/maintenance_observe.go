package state

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
)

// MaintenanceLedger is a bounded observation of the original schema-6
// ledger, not an approval or live Root handle. All UNKNOWN originals,
// including historically released ones, remain visible.
type MaintenanceLedger struct {
	NonterminalRecordedClaims int                     `json:"nonterminal_recorded_claims"`
	PeerCapacity              *DiagnosticPeerCapacity `json:"peer_capacity,omitempty"`

	Project        string                `json:"project"`
	Boxes          []MaintenanceBox      `json:"boxes"`
	Retired        int                   `json:"retired"`
	Originals      []MaintenanceOriginal `json:"nonterminal_or_unknown_originals"`
	Claims         int                   `json:"recorded_claims"`
	Peers          []MaintenancePeer     `json:"peers"`
	Forwards       []MaintenanceForward  `json:"forwards"`
	ForeignRows    int                   `json:"foreign_scope_rows"`
	OrphanBindings int                   `json:"orphan_bindings"`
	OrphanFences   int                   `json:"orphan_fences"`
	Network        NativeNetworkLedger   `json:"-"`
}
type MaintenanceBox struct {
	UUID           string `json:"uuid"`
	Incarnation    string `json:"incarnation"`
	Name           string `json:"name"`
	CreateOriginal string `json:"create_original,omitempty"`
	CreateProof    bool   `json:"create_proof"`
	BindingState   string `json:"binding_state"`
	ACL            string `json:"acl,omitempty"`
}
type MaintenanceOriginal struct {
	ClaimProtection string `json:"claim_protection"`

	ID            string `json:"id"`
	Action        string `json:"action"`
	State         string `json:"state"`
	Claims        int    `json:"recorded_claims"`
	NativeReceipt bool   `json:"native_receipt"`
}
type MaintenancePeer struct {
	LastVerifiedOriginal string `json:"last_verified_original,omitempty"`

	Source            string `json:"source"`
	SourceIncarnation string `json:"source_incarnation"`
	Target            string `json:"target"`
	TargetIncarnation string `json:"target_incarnation"`
	State             string `json:"state"`
	Original          string `json:"original,omitempty"`
}
type MaintenanceForward struct {
	Protocol  string `json:"-"`
	Bind      string `json:"-"`
	HostPort  int    `json:"-"`
	GuestPort int    `json:"-"`

	ID          string `json:"id"`
	Box         string `json:"box"`
	Incarnation string `json:"incarnation"`
	Device      string `json:"device"`
	State       string `json:"state"`
	Original    string `json:"original,omitempty"`
}

const maintenanceRowLimit = 8192

// ReadMaintenanceLedger uses only the private snapshot. No source SQL
// connections, schema upgrade, credential content or arbitrary intent.
func (s *DiagnosticSnapshot) ReadMaintenanceLedger(ctx context.Context, project string) (MaintenanceLedger, error) {
	var zero MaintenanceLedger
	if !SafeDiagnosticProject(project) {
		return zero, ErrInvalid
	}
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	o, e := readMaintenanceLedger(ctx, tx, project)
	if e != nil {
		return zero, e
	}
	if e = tx.Commit(); e != nil {
		return zero, e
	}
	return o, nil
}
func readMaintenanceLedger(ctx context.Context, tx *sql.Tx, project string) (MaintenanceLedger, error) {
	var zero MaintenanceLedger
	o := MaintenanceLedger{Project: project, Boxes: []MaintenanceBox{}, Originals: []MaintenanceOriginal{}, Peers: []MaintenancePeer{}, Forwards: []MaintenanceForward{}}
	l, e := readNativeNetworkLedgerTx(ctx, tx, project)
	if e != nil {
		return zero, e
	}
	o.Network = l
	o.OrphanBindings, o.OrphanFences = l.OrphanBindings, l.OrphanFences
	for _, b := range l.Boxes {
		if !validActivityBoxID(b.UUID) || !validActivityBoxID(b.Incarnation) || !auditName(b.Name) || b.OriginalID != "" && !validHexID(b.OriginalID) || b.ACL != "" && !auditName(b.ACL) {
			return zero, ErrDiagnosticInvalid
		}
		o.Boxes = append(o.Boxes, MaintenanceBox{b.UUID, b.Incarnation, b.Name, b.OriginalID, b.OriginalProof, b.BindingState, b.ACL})
	}
	for _, q := range []struct {
		sql string
		dst *int
	}{
		{"SELECT count(*) FROM owned_boxes WHERE project=? AND state='DELETED'", &o.Retired},
		{"SELECT count(*) FROM operation_resource_claims WHERE layer=?", &o.Claims},
	} {
		if e = tx.QueryRowContext(ctx, q.sql, project).Scan(q.dst); e != nil {
			return zero, e
		}
	}
	// Count only: never reveal another Project's identities or intent.
	for _, q := range []string{
		"SELECT count(*) FROM owned_boxes WHERE project!=?",
		"SELECT count(*) FROM operations WHERE layer!=?",
		"SELECT count(*) FROM operation_resource_claims WHERE layer!=?",
		"SELECT count(*) FROM network_bindings WHERE project!=?",
		"SELECT count(*) FROM network_address_fences WHERE project!=?",
		"SELECT count(*) FROM peer_grants WHERE project!=?",
		"SELECT count(*) FROM forward_claims WHERE project!=?",
	} {
		var n int
		if e = tx.QueryRowContext(ctx, q, project).Scan(&n); e != nil {
			return zero, e
		}
		o.ForeignRows += n
	}
	rows, e := tx.QueryContext(ctx, `SELECT o.id,o.action,o.state,o.native_operation_id!='',(SELECT count(*) FROM operation_resource_claims c WHERE c.layer=o.layer AND c.operation_id=o.id)
 FROM operations o WHERE o.layer=? AND (o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING') ) ORDER BY o.id LIMIT ?`, project, maintenanceRowLimit+1)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var v MaintenanceOriginal
		if e = rows.Scan(&v.ID, &v.Action, &v.State, &v.NativeReceipt, &v.Claims); e != nil {
			break
		}
		if len(o.Originals) == maintenanceRowLimit || !validHexID(v.ID) {
			e = ErrDiagnosticBound
			break
		}
		v.Action = diagnosticAction(v.Action)
		o.Originals = append(o.Originals, v)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return zero, e
	}
	for n := range o.Originals {
		v := &o.Originals[n]
		nav, err := readRecoveryNavigation(ctx, tx, project, v.ID)
		if err != nil {
			return zero, err
		}
		v.ClaimProtection = nav.ClaimProtection
		o.NonterminalRecordedClaims += v.Claims
	}
	cap, err := readDiagnosticPeerCapacity(ctx, tx, project)
	if err != nil {
		return zero, err
	}
	o.PeerCapacity = &cap
	rows, e = tx.QueryContext(ctx, `SELECT source_uuid,source_incarnation,target_uuid,target_incarnation,status,operation_id,coalesce(last_verified_operation_id,'') FROM peer_grants WHERE project=? ORDER BY source_uuid,source_incarnation,target_uuid,target_incarnation LIMIT ?`, project, maintenanceRowLimit+1)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var v MaintenancePeer
		if e = rows.Scan(&v.Source, &v.SourceIncarnation, &v.Target, &v.TargetIncarnation, &v.State, &v.Original, &v.LastVerifiedOriginal); e != nil {
			break
		}
		if len(o.Peers) == maintenanceRowLimit || !validActivityBoxID(v.Source) || !validActivityBoxID(v.SourceIncarnation) || !validActivityBoxID(v.Target) || !validActivityBoxID(v.TargetIncarnation) || v.Original != "" && !validHexID(v.Original) || v.LastVerifiedOriginal != "" && !validHexID(v.LastVerifiedOriginal) {
			e = ErrDiagnosticInvalid
			break
		}
		o.Peers = append(o.Peers, v)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return zero, e
	}
	rows, e = tx.QueryContext(ctx, `SELECT forward_id,box_uuid,incarnation,native_device,state,operation_id,protocol,host_bind,host_port,guest_port FROM forward_claims WHERE project=? ORDER BY forward_id LIMIT ?`, project, maintenanceRowLimit+1)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var v MaintenanceForward
		if e = rows.Scan(&v.ID, &v.Box, &v.Incarnation, &v.Device, &v.State, &v.Original, &v.Protocol, &v.Bind, &v.HostPort, &v.GuestPort); e != nil {
			break
		}
		if len(o.Forwards) == maintenanceRowLimit || !validHexID(v.ID) || !validActivityBoxID(v.Box) || !validActivityBoxID(v.Incarnation) || !auditName(v.Device) || v.Original != "" && !validHexID(v.Original) {
			e = ErrDiagnosticInvalid
			break
		}
		o.Forwards = append(o.Forwards, v)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil {
		return zero, e
	}
	return o, nil
}

// ObservationDigest compares bounded inputs across independent captures.
// Never use this as a maintenance approval or replay key.
func (o MaintenanceLedger) ObservationDigest() string {
	endpointInputs := make([][]any, 0, len(o.Forwards))
	for _, f := range o.Forwards {
		endpointInputs = append(endpointInputs, []any{f.ID, f.Protocol, f.Bind, f.HostPort, f.GuestPort})
	}
	raw, _ := json.Marshal(struct {
		Ledger    MaintenanceLedger
		Network   NativeNetworkLedger
		Endpoints [][]any
	}{o, o.Network, endpointInputs})
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
