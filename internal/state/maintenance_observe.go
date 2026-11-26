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
	LegacyPending             []string                `json:"legacy_pending_retirements"`
	LegacyCompactions         []string                `json:"legacy_compactions"`
	HistorySHA256             string                  `json:"history_sha256"`
	RetiredBoxes              []MaintenanceRetiredBox `json:"retired_boxes"`
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
type MaintenanceRetiredBox struct {
	UUID           string `json:"uuid"`
	Incarnation    string `json:"incarnation"`
	Name           string `json:"name"`
	DeleteOriginal string `json:"delete_original"`
}
type MaintenanceOriginal struct {
	RetirementScope string `json:"retirement_scope,omitempty"`
	BoxUUID         string `json:"box_uuid,omitempty"`
	Incarnation     string `json:"incarnation,omitempty"`
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
	return s.readMaintenanceLedger(ctx, project, false)
}

// ReadScopedMaintenanceLedger refuses mixed retirement history. It uses the
// same snapshot and never treats absence of legacy data as native authority.
func (s *DiagnosticSnapshot) ReadScopedMaintenanceLedger(ctx context.Context, project string) (MaintenanceLedger, error) {
	return s.readMaintenanceLedger(ctx, project, true)
}

func (s *DiagnosticSnapshot) readMaintenanceLedger(ctx context.Context, project string, scoped bool) (MaintenanceLedger, error) {
	var zero MaintenanceLedger
	if !SafeDiagnosticProject(project) {
		return zero, ErrInvalid
	}
	tx, e := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return zero, e
	}
	defer tx.Rollback()
	o, e := readMaintenanceLedger(ctx, tx, project, scoped)
	if e != nil {
		return zero, e
	}
	if e = tx.Commit(); e != nil {
		return zero, e
	}
	return o, nil
}
func readMaintenanceLedger(ctx context.Context, tx *sql.Tx, project string, scoped bool) (MaintenanceLedger, error) {
	var zero MaintenanceLedger
	if scoped {
		if e := requireScopedNoLegacyRetirementTx(ctx, tx); e != nil {
			return zero, e
		}
	}
	o := MaintenanceLedger{Project: project, LegacyPending: []string{}, LegacyCompactions: []string{}, RetiredBoxes: []MaintenanceRetiredBox{}, Boxes: []MaintenanceBox{}, Originals: []MaintenanceOriginal{}, Peers: []MaintenancePeer{}, Forwards: []MaintenanceForward{}}
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
	rows, e := tx.QueryContext(ctx, `SELECT uuid,incarnation,name,deleted_by_operation_id FROM owned_boxes WHERE project=? AND state='DELETED' ORDER BY uuid LIMIT ?`, project, maintenanceRowLimit+1)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var r MaintenanceRetiredBox
		if e = rows.Scan(&r.UUID, &r.Incarnation, &r.Name, &r.DeleteOriginal); e != nil {
			break
		}
		if len(o.RetiredBoxes) >= maintenanceRowLimit || !validActivityBoxID(r.UUID) || !validActivityBoxID(r.Incarnation) || !auditName(r.Name) || r.DeleteOriginal != "" && !validHexID(r.DeleteOriginal) {
			e = ErrDiagnosticInvalid
			break
		}
		o.RetiredBoxes = append(o.RetiredBoxes, r)
	}
	if e == nil {
		e = rows.Err()
	}
	if ce := rows.Close(); e == nil {
		e = ce
	}
	if e != nil || len(o.RetiredBoxes) != o.Retired {
		if e == nil {
			e = ErrDiagnosticInvalid
		}
		return zero, e
	}
	// Count only: never reveal another Project's identities or intent.
	foreignQueries := []string{
		"SELECT count(*) FROM owned_boxes WHERE project!=?",
		"SELECT count(*) FROM operations WHERE layer!=?",
		"SELECT count(*) FROM operation_resource_claims WHERE layer!=?",
		"SELECT count(*) FROM network_bindings WHERE project!=?",
		"SELECT count(*) FROM network_address_fences WHERE project!=?",
		"SELECT count(*) FROM peer_grants WHERE project!=?",
		"SELECT count(*) FROM forward_claims WHERE project!=?",
	}
	if !scoped {
		foreignQueries = append(foreignQueries, legacyMaintenanceForeignQueries()...)
	}
	for _, q := range foreignQueries {
		var n int
		if e = tx.QueryRowContext(ctx, q, project).Scan(&n); e != nil {
			return zero, e
		}
		o.ForeignRows += n
	}
	rows, e = tx.QueryContext(ctx, `SELECT o.id,o.action,o.state,o.box_uuid,o.incarnation,o.native_operation_id!='',(SELECT count(*) FROM operation_resource_claims c WHERE c.layer=o.layer AND c.operation_id=o.id)
 FROM operations o WHERE o.layer=? AND (o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING') ) ORDER BY o.id LIMIT ?`, project, maintenanceRowLimit+1)
	if e != nil {
		return zero, e
	}
	for rows.Next() {
		var v MaintenanceOriginal
		if e = rows.Scan(&v.ID, &v.Action, &v.State, &v.BoxUUID, &v.Incarnation, &v.NativeReceipt, &v.Claims); e != nil {
			break
		}
		if len(o.Originals) == maintenanceRowLimit || !validHexID(v.ID) {
			e = ErrDiagnosticBound
			break
		}
		v.Action = diagnosticAction(v.Action)
		// Report Box identity only when it belongs to this exact owned scope.
		// Unknown or incomplete Create targets are not adopted by observation.
		owned := false
		for _, b := range o.Boxes {
			if b.UUID == v.BoxUUID && b.Incarnation == v.Incarnation {
				owned = true
				break
			}
		}
		if !owned {
			v.BoxUUID, v.Incarnation = "", ""
		}
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
		if e = maintenanceRetirementScope(ctx, tx, project, o.Boxes, v); e != nil {
			return zero, e
		}
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
	if !scoped {
		if e := readLegacyMaintenancePending(ctx, tx, project, &o); e != nil {
			return zero, e
		}
	}
	o.HistorySHA256, e = maintenanceHistoryWitness(ctx, tx, project, scoped)
	if e != nil {
		return zero, e
	}
	return o, nil
}

// Bounded digest of exact identity and historical effect records. No raw
// actor, credential, arbitrary intent, or log content is emitted.
func maintenanceHistoryWitness(ctx context.Context, tx *sql.Tx, project string, scoped bool) (string, error) {
	queries := []struct {
		sql    string
		legacy bool
	}{
		{`SELECT id,action,state,updated_at,resource_key,box_uuid,incarnation,native_operation_id,intent_digest,error_class FROM operations WHERE layer=? ORDER BY id`, false},
		{`SELECT operation_id,resource_key FROM operation_resource_claims WHERE layer=? ORDER BY operation_id,resource_key`, false},
		{`SELECT * FROM owned_boxes WHERE project=? ORDER BY uuid`, false},
		{`SELECT * FROM network_bindings WHERE project=? ORDER BY box_uuid,incarnation,nic`, false},
		{`SELECT * FROM network_address_fences WHERE project=? ORDER BY bridge,family,address`, false},
		{legacyMaintenanceArchiveHistory, true},
		{legacyMaintenanceQueueHistory, true},
		{legacyMaintenanceCompactionHistory, true},
		{`SELECT * FROM peer_grants WHERE project=? ORDER BY source_uuid,source_incarnation,target_uuid,target_incarnation`, false},
		{`SELECT * FROM forward_claims WHERE project=? ORDER BY forward_id`, false},
		{`SELECT e.seq,e.operation_id,e.at,e.state FROM operation_events e JOIN operations o ON o.id=e.operation_id WHERE o.layer=? ORDER BY e.seq`, false},
		{`SELECT a.seq,a.operation_id,a.at,a.decision,a.evidence_digest FROM recovery_audit a JOIN operations o ON o.id=a.operation_id WHERE o.layer=? ORDER BY a.seq`, false},
	}
	h := sha256.New()
	count, total := 0, 0
	for _, q := range queries {
		// Preserve schema-6 witness bytes for an empty legacy table. The
		// scoped reader has already proved all three empty in this transaction.
		h.Write([]byte(q.sql + "\n"))
		if scoped && q.legacy {
			continue
		}
		rows, e := tx.QueryContext(ctx, q.sql, project)
		if e != nil {
			return "", e
		}
		cols, e := rows.Columns()
		if e != nil {
			rows.Close()
			return "", e
		}
		for rows.Next() {
			values := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range values {
				ptrs[i] = &values[i]
			}
			if e = rows.Scan(ptrs...); e != nil {
				break
			}
			raw, err := json.Marshal(values)
			if err != nil {
				e = err
				break
			}
			count++
			total += len(raw)
			if count > 100000 || total > 64<<20 {
				e = ErrDiagnosticBound
				break
			}
			h.Write(raw)
			h.Write([]byte{'\n'})
		}
		if e == nil {
			e = rows.Err()
		}
		if ce := rows.Close(); e == nil {
			e = ce
		}
		if e != nil {
			return "", e
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
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
