package state

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// ActivityQuery is an internal-only, authenticated-controller input. No MCP
// registration exists for it. Labels narrow a Layer; they never grant access.
type ActivityQuery struct {
	Layer       string
	Requester   string
	View        string
	OperationID string
	BoxUUID     string
	WorkRef     string
	AgentRef    string
	Limit       int
	Cursor      string
}

type ActivityIssue struct {
	Disposition string `json:"disposition"`
	OperationID string `json:"operation_id,omitempty"`
	BoxUUID     string `json:"box_id,omitempty"`
	ResourceRef string `json:"resource_ref,omitempty"`
	Reason      string `json:"reason"`
	State       string `json:"state"`
	AtNS        int64  `json:"at_ns"`
}

type ActivityEvent struct {
	Seq    int64  `json:"seq"`
	AtNS   int64  `json:"at_ns"`
	State  string `json:"state"`
	Detail string `json:"safe_detail,omitempty"`
}

type ActivityReview struct {
	Seq      int64  `json:"seq"`
	AtNS     int64  `json:"at_ns"`
	Decision string `json:"decision"`
}

type ActivityPeer struct {
	SourceUUID string `json:"source_uuid"`
	TargetUUID string `json:"target_uuid"`
	Status     string `json:"status"`
	Desired    bool   `json:"desired"`
	PendingID  string `json:"pending_operation_id,omitempty"`
	VerifiedID string `json:"last_verified_operation_id,omitempty"`
}

type ActivityGC struct {
	Stream string `json:"stream"`
	State  string `json:"state"`
	AtNS   int64  `json:"at_ns"`
}

type ActivityExec struct {
	ObservedAtNS int64  `json:"observed_at_ns"`
	ExitCode     *int64 `json:"exit_code,omitempty"`
	Truncated    bool   `json:"truncated"`
}

type ActivityEvidence struct {
	Original      Receipt          `json:"original"`
	Events        []ActivityEvent  `json:"events"`
	Reviews       []ActivityReview `json:"reviews"`
	ResourceKeys  []string         `json:"resource_keys"`
	Exec          *ActivityExec    `json:"exec,omitempty"`
	GC            []ActivityGC     `json:"gc"`
	Peers         []ActivityPeer   `json:"peers"`
	OwnedBoxState string           `json:"owned_box_state,omitempty"`
	Limitations   []string         `json:"limitations"`
}

type ActivityCoverageDetails struct {
	AuthorizedProjectOnly   bool `json:"authorized_project_only"`
	SQLiteSnapshotOnly      bool `json:"sqlite_snapshot_only"`
	NativeLiveVerified      bool `json:"native_live_verified"`
	CrossPageSnapshotStable bool `json:"cross_page_snapshot_stable"`
	AttentionComplete       bool `json:"attention_complete"`
	PageTruncated           bool `json:"page_truncated"`
	HistorySampleBounded    bool `json:"history_sample_bounded"`
}

type ActivityResult struct {
	CoverageDetails   *ActivityCoverageDetails `json:"coverage_details,omitempty"`
	Source            string                   `json:"source"`
	Scope             string                   `json:"scope"`
	AsOfNS            int64                    `json:"as_of_ns"`
	Coverage          string                   `json:"coverage"`
	Truncated         bool                     `json:"truncated"`
	AttentionComplete bool                     `json:"attention_complete"`
	NextCursor        string                   `json:"next_cursor,omitempty"`
	Operations        []Receipt                `json:"operations,omitempty"`
	Issues            []ActivityIssue          `json:"issues,omitempty"`
	History           []ActivityIssue          `json:"history,omitempty"`
	HistoryTruncated  bool                     `json:"history_truncated"`
	Evidence          *ActivityEvidence        `json:"evidence,omitempty"`
}

func (q ActivityQuery) normalize() (ActivityQuery, error) {
	if q.Layer == "" || len(q.Layer) > 64 || !validHexID(q.Requester) ||
		!validRef(q.WorkRef) || !validRef(q.AgentRef) ||
		(q.BoxUUID != "" && !validActivityBoxID(q.BoxUUID)) ||
		(q.OperationID != "" && !validHexID(q.OperationID)) ||
		q.Limit < 0 || q.Limit > 50 || len(q.Cursor) > 1024 {
		return q, ErrInvalid
	}
	if q.View != "overview" && q.View != "issues" && q.View != "evidence" {
		return q, ErrInvalid
	}
	if (q.View == "evidence") != (q.OperationID != "") {
		return q, ErrInvalid
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	return q, nil
}

func validActivityBoxID(id string) bool {
	if validHexID(id) {
		return true
	}
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
		} else if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

// The Activity cursor carries an authenticated scope, admission watermark and
// keyset. Its pages are bounded live snapshots, not an immutable historical
// database transaction spanning multiple requests.
type activityCursor struct {
	Scope      string `json:"s"`
	MaxRowID   int64  `json:"r"`
	AtNS       int64  `json:"t"`
	Reason     string `json:"q"`
	OriginalID string `json:"o"`
	Resource   string `json:"k"`
	MAC        string `json:"m"`
}

func activityScope(q ActivityQuery) string {
	q.Cursor = ""
	raw, _ := json.Marshal(q)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func activityCursorMAC(secret []byte, cursor activityCursor) string {
	cursor.MAC = ""
	raw, _ := json.Marshal(cursor)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(raw)
	return hex.EncodeToString(mac.Sum(nil))
}

func readActivityCursor(ctx context.Context, tx *sql.Tx, q ActivityQuery) ([]byte, activityCursor, error) {
	var secret []byte
	if err := tx.QueryRowContext(ctx, "SELECT secret FROM cursor_secrets WHERE id=1").Scan(&secret); err != nil {
		return nil, activityCursor{}, err
	}
	if len(secret) != 32 {
		return nil, activityCursor{}, ErrInvalid
	}
	cursor := activityCursor{Scope: activityScope(q)}
	if q.Cursor == "" {
		if err := tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(rowid),0) FROM operations WHERE layer=?", q.Layer).Scan(&cursor.MaxRowID); err != nil {
			return nil, activityCursor{}, err
		}
		return secret, cursor, nil
	}
	encoded, err := base64.RawURLEncoding.DecodeString(q.Cursor)
	if err != nil || len(encoded) > 512 || json.Unmarshal(encoded, &cursor) != nil ||
		cursor.Scope != activityScope(q) || cursor.MaxRowID < 0 ||
		!hmac.Equal([]byte(cursor.MAC), []byte(activityCursorMAC(secret, cursor))) {
		return nil, activityCursor{}, ErrInvalid
	}
	return secret, cursor, nil
}

func activityNextCursor(secret []byte, cursor activityCursor) string {
	cursor.MAC = activityCursorMAC(secret, cursor)
	data, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(data)
}

func (q ActivityQuery) where() (string, []any) {
	parts := []string{"o.layer=?"}
	args := []any{q.Layer}
	if q.WorkRef != "" {
		parts = append(parts, "o.work_ref=?")
		args = append(args, q.WorkRef)
	}
	if q.AgentRef != "" {
		parts = append(parts, "o.agent_ref=?")
		args = append(args, q.AgentRef)
	}
	if q.BoxUUID != "" {
		parts = append(parts, "(o.box_uuid=? OR EXISTS (SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.layer=? AND c.resource_key=?))")
		args = append(args, q.BoxUUID, q.Layer, "box:"+q.BoxUUID)
	}
	return strings.Join(parts, " AND "), args
}

const activityColumns = "o.id,o.tracking_id,o.state,o.native_operation_id,o.work_ref,o.agent_ref,o.action,o.box_uuid,o.incarnation,o.error_class,o.created_at,o.updated_at"

func scanActivityOperation(rows *sql.Rows) (Receipt, error) {
	var rec Receipt
	err := rows.Scan(&rec.ID, &rec.TrackingID, &rec.State, &rec.NativeOperationID,
		&rec.WorkRef, &rec.AgentRef, &rec.Action, &rec.BoxUUID, &rec.Incarnation,
		&rec.ErrorClass, &rec.CreatedAt, &rec.UpdatedAt)
	if err == nil {
		rec.NextAction = nextAction(rec.State, rec.Action, rec.ID)
	}
	return rec, err
}

// ReadActivity takes exactly one read transaction and never contacts Incus,
// Root, or a guest. Snapshot consistency applies only to SQLite. It cannot
// prove that unjournaled calls or native effects occurred.
func (s *Store) ReadActivity(ctx context.Context, input ActivityQuery) (ActivityResult, error) {
	q, err := input.normalize()
	if err != nil || s == nil || s.db == nil {
		return ActivityResult{}, ErrInvalid
	}
	if err := s.RequireActiveTokenID(ctx, q.Requester); err != nil {
		return ActivityResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return ActivityResult{}, err
	}
	defer tx.Rollback()
	result := ActivityResult{
		Source: "SQLITE_SNAPSHOT", Scope: q.Layer,
		AsOfNS: time.Now().UnixNano(), Coverage: "BOUNDED",
	}
	switch q.View {
	case "overview":
		err = readActivityOverview(ctx, tx, q, &result)
	case "issues":
		err = readActivityIssues(ctx, tx, q, &result)
	case "evidence":
		err = readActivityEvidence(ctx, tx, q, &result)
	}
	if err != nil {
		return ActivityResult{}, err
	}
	result.CoverageDetails = &ActivityCoverageDetails{
		AuthorizedProjectOnly: true, SQLiteSnapshotOnly: true,
		NativeLiveVerified: false, CrossPageSnapshotStable: false,
		AttentionComplete: result.AttentionComplete, PageTruncated: result.Truncated,
		HistorySampleBounded: q.View == "issues",
	}
	if err := tx.Commit(); err != nil {
		return ActivityResult{}, err
	}
	// An access token may be revoked during a slow historical scan. The
	// controller must still perform its own privateReadReply recheck if this
	// private query is ever exposed as a public MCP Tool.
	if err := s.RequireActiveTokenID(ctx, q.Requester); err != nil {
		return ActivityResult{}, err
	}
	return result, nil
}

func readActivityOverview(ctx context.Context, tx *sql.Tx, q ActivityQuery, out *ActivityResult) error {
	secret, cursor, err := readActivityCursor(ctx, tx, q)
	if err != nil {
		return err
	}
	where, args := q.where()
	where += " AND o.rowid<=?"
	args = append(args, cursor.MaxRowID)
	if q.Cursor != "" {
		where += " AND (o.created_at<? OR (o.created_at=? AND o.id<?))"
		args = append(args, cursor.AtNS, cursor.AtNS, cursor.OriginalID)
	}
	rows, err := tx.QueryContext(ctx, "SELECT "+activityColumns+" FROM operations o WHERE "+where+
		" ORDER BY o.created_at DESC,o.id DESC LIMIT ?", append(args, q.Limit+1)...)
	if err != nil {
		return err
	}
	out.Operations = []Receipt{}
	for rows.Next() {
		rec, e := scanActivityOperation(rows)
		if e != nil {
			err = e
			break
		}
		if len(out.Operations) >= q.Limit {
			out.Truncated = true
			break
		}
		out.Operations = append(out.Operations, rec)
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if out.Truncated {
		last := out.Operations[len(out.Operations)-1]
		cursor.AtNS, cursor.OriginalID = last.CreatedAt, last.ID
		out.NextCursor = activityNextCursor(secret, cursor)
	}
	return nil
}

// The live issues projection includes all *SQLite-visible* effective admission
// blockers, not just recent Operations or primary Box claims. A current
// CONFIRMED Peer or RESERVED address fence is a dependency, not a failure.
// Unlike original events, network ledger rows are mutable: multiple cursor
// pages are independent bounded snapshots. Never promise an all-clear.
func readActivityIssues(ctx context.Context, tx *sql.Tx, q ActivityQuery, out *ActivityResult) error {
	secret, cursor, err := readActivityCursor(ctx, tx, q)
	if err != nil {
		return err
	}
	opWhere, opArgs := q.where()
	opWhere += " AND o.rowid<=?"
	opArgs = append(opArgs, cursor.MaxRowID)
	branches := []string{
		"SELECT CASE WHEN o.action='box_peer_set' THEN 'PROJECT_PEER_BARRIER' ELSE 'UNRESOLVED_OPERATION' END AS reason," +
			"o.id AS original,o.box_uuid AS box_uuid,o.resource_key AS resource,o.state AS state,o.created_at AS at_ns" +
			" FROM operations o WHERE " + opWhere +
			" AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN','OBSERVING')" +
			" AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))",
		"SELECT 'EXEC_OUTPUT_GC_BLOCKER',o.id,g.box_uuid,'gc:'||g.name||'/'||g.stream,g.state,g.claimed_at" +
			" FROM exec_output_gc g JOIN operations o ON o.id=g.operation_id WHERE " + opWhere +
			" AND g.project=o.layer AND g.state IN ('DELETE_SENT','REVIEW_REQUIRED')",
	}
	args := append([]any{}, opArgs...)
	args = append(args, opArgs...)

	// Exact network sources are constrained to the authenticated Project.
	// Labels only narrow rows with a genuine original; they never invent
	// attribution from identical names, IPs or timestamps.
	labelWhere := ""
	labelArgs := []any{}
	if q.WorkRef != "" {
		labelWhere += " AND o.work_ref=?"
		labelArgs = append(labelArgs, q.WorkRef)
	}
	if q.AgentRef != "" {
		labelWhere += " AND o.agent_ref=?"
		labelArgs = append(labelArgs, q.AgentRef)
	}
	peer := "SELECT CASE WHEN p.status='CONFIRMED' THEN 'PEER_ACTIVE_DEPENDENCY' ELSE 'PEER_LEDGER_UNRESOLVED' END," +
		"COALESCE(o.id,''),p.source_uuid," +
		"'peer:'||p.source_uuid||'/'||p.source_incarnation||'/'||p.target_uuid||'/'||p.target_incarnation," +
		"p.status,0 FROM peer_grants p LEFT JOIN operations o ON o.id=CASE WHEN p.operation_id!='' THEN p.operation_id ELSE p.last_verified_operation_id END" +
		" AND o.layer=p.project AND o.rowid<=? WHERE p.project=? AND p.status IN ('PENDING','UNKNOWN','CONFIRMED')"
	peerArgs := []any{cursor.MaxRowID, q.Layer}
	if q.BoxUUID != "" {
		peer += " AND (p.source_uuid=? OR p.target_uuid=?)"
		peerArgs = append(peerArgs, q.BoxUUID, q.BoxUUID)
	}
	branches = append(branches, peer+labelWhere)
	args = append(args, peerArgs...)
	args = append(args, labelArgs...)

	fences := "SELECT 'ADDRESS_FENCE_DEPENDENCY',COALESCE(o.id,''),f.box_uuid," +
		"'address:'||f.bridge||'/'||f.family||'/'||f.address,f.state,f.observed_at" +
		" FROM network_address_fences f LEFT JOIN operations o ON o.id=f.operation_id" +
		" AND o.layer=f.project AND o.rowid<=? WHERE f.project=? AND f.state!='RELEASED'"
	fenceArgs := []any{cursor.MaxRowID, q.Layer}
	if q.BoxUUID != "" {
		fences += " AND f.box_uuid=?"
		fenceArgs = append(fenceArgs, q.BoxUUID)
	}
	branches = append(branches, fences+labelWhere)
	args = append(args, fenceArgs...)
	args = append(args, labelArgs...)

	forward := "SELECT 'FORWARD_DEPENDENCY',COALESCE(o.id,''),f.box_uuid,'forward:'||f.forward_id," +
		"f.state,0 FROM forward_claims f LEFT JOIN operations o ON o.id=f.operation_id" +
		" AND o.layer=f.project AND o.rowid<=? WHERE f.project=? AND f.state!='REVOKED'"
	forwardArgs := []any{cursor.MaxRowID, q.Layer}
	if q.BoxUUID != "" {
		forward += " AND f.box_uuid=?"
		forwardArgs = append(forwardArgs, q.BoxUUID)
	}
	branches = append(branches, forward+labelWhere)
	args = append(args, forwardArgs...)
	args = append(args, labelArgs...)

	stmt := "WITH candidates(reason,original,box_uuid,resource,state,at_ns) AS (" +
		strings.Join(branches, " UNION ALL ") +
		") SELECT reason,original,box_uuid,resource,state,at_ns FROM candidates"
	if q.Cursor != "" {
		stmt += " WHERE (at_ns>? OR (at_ns=? AND (reason>? OR (reason=? AND (original>? OR (original=? AND resource>?))))))"
		args = append(args, cursor.AtNS, cursor.AtNS, cursor.Reason, cursor.Reason,
			cursor.OriginalID, cursor.OriginalID, cursor.Resource)
	}
	stmt += " ORDER BY at_ns ASC,reason ASC,original ASC,resource ASC LIMIT ?"
	args = append(args, q.Limit+1)
	rows, err := tx.QueryContext(ctx, stmt, args...)
	if err != nil {
		return err
	}
	out.Issues = []ActivityIssue{}
	for rows.Next() {
		var issue ActivityIssue
		if err = rows.Scan(&issue.Reason, &issue.OperationID, &issue.BoxUUID,
			&issue.ResourceRef, &issue.State, &issue.AtNS); err != nil {
			break
		}
		if len(out.Issues) >= q.Limit {
			out.Truncated = true
			break
		}
		issue.Disposition = activityIssueDisposition(issue.Reason, issue.State)
		out.Issues = append(out.Issues, issue)
	}
	if e := rows.Err(); err == nil {
		err = e
	}
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if out.Truncated {
		last := out.Issues[len(out.Issues)-1]
		cursor.AtNS, cursor.Reason, cursor.OriginalID, cursor.Resource =
			last.AtNS, last.Reason, last.OperationID, last.ResourceRef
		out.NextCursor = activityNextCursor(secret, cursor)
	}
	// A bounded scan cannot certify a physical network or the absence of an
	// unjournaled native effect. Also page transitions can change a ledger.
	out.AttentionComplete = false
	return readActivityHistory(ctx, tx, q, cursor.MaxRowID, out)
}

func activityIssueDisposition(reason, st string) string {
	switch reason {
	case "PEER_ACTIVE_DEPENDENCY", "ADDRESS_FENCE_DEPENDENCY":
		if st == "QUARANTINED" {
			return "HOST_REVIEW_REQUIRED"
		}
		return "ACTIVE_DEPENDENCY_NOT_ERROR"
	case "FORWARD_DEPENDENCY":
		if st == "UNKNOWN" {
			return "HOST_REVIEW_REQUIRED"
		}
		return "ACTIVE_DEPENDENCY_NOT_ERROR"
	case "EXEC_OUTPUT_GC_BLOCKER":
		return "HOST_REVIEW_REQUIRED"
	case "PEER_LEDGER_UNRESOLVED", "PROJECT_PEER_BARRIER", "UNRESOLVED_OPERATION":
		if st == "UNKNOWN" {
			return "HOST_REVIEW_REQUIRED"
		}
		return "IN_PROGRESS_OR_STALLED"
	default:
		return "INSPECT_EVIDENCE"
	}
}

// A small independent historical sample is NOT an active lock list and
// does not claim to summarize all historic failures. RELEASED means that
// an admission claim was released, NOT that a native UNKNOWN was resolved.
func readActivityHistory(ctx context.Context, tx *sql.Tx, q ActivityQuery, watermark int64, out *ActivityResult) error {
	where, args := q.where()
	args = append(args, watermark)
	limit := q.Limit
	if limit > 5 {
		limit = 5
	}
	args = append(args, limit+1)
	query := "SELECT o.id,o.box_uuid,o.resource_key,o.state,o.created_at FROM operations o WHERE " + where +
		" AND o.rowid<=? AND (o.state='FAILED' OR (o.state='UNKNOWN' AND o.action!='box_peer_set' " +
		"AND EXISTS (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))) " +
		"ORDER BY o.created_at DESC,o.id DESC LIMIT ?"
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	out.History = []ActivityIssue{}
	for rows.Next() {
		var issue ActivityIssue
		if err := rows.Scan(&issue.OperationID, &issue.BoxUUID, &issue.ResourceRef, &issue.State, &issue.AtNS); err != nil {
			return err
		}
		if len(out.History) >= limit {
			out.HistoryTruncated = true
			break
		}
		if issue.State == "UNKNOWN" {
			issue.Reason = "HISTORICAL_RELEASED_CLAIM"
			issue.Disposition = "CLAIM_RELEASED_NATIVE_OUTCOME_UNKNOWN"
		} else {
			issue.Reason = "HISTORICAL_FAILED_OPERATION"
			issue.Disposition = "TERMINAL_FAILED_NOT_ACTIVE"
		}
		out.History = append(out.History, issue)
	}
	return rows.Err()
}

func readActivityEvidence(ctx context.Context, tx *sql.Tx, q ActivityQuery, out *ActivityResult) error {
	where, args := q.where()
	rows, err := tx.QueryContext(ctx, "SELECT "+activityColumns+
		" FROM operations o WHERE "+where+" AND o.id=?", append(args, q.OperationID)...)
	if err != nil {
		return err
	}
	if !rows.Next() {
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return err
		}
		return ErrNotFound
	}
	rec, err := scanActivityOperation(rows)
	if e := rows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	ev := &ActivityEvidence{
		Original: rec, Events: []ActivityEvent{}, Reviews: []ActivityReview{},
		ResourceKeys: []string{}, GC: []ActivityGC{}, Peers: []ActivityPeer{},
		Limitations: []string{"NO_NATIVE_POSTSTATE", "ROOT_JOURNAL_PRIVATE", "READ_CALLS_NOT_JOURNALED"},
	}
	out.Evidence = ev
	events, err := tx.QueryContext(ctx, "SELECT seq,at,state,safe_detail FROM operation_events WHERE operation_id=? ORDER BY seq LIMIT ?", rec.ID, q.Limit+1)
	if err != nil {
		return err
	}
	for events.Next() {
		var e ActivityEvent
		if err = events.Scan(&e.Seq, &e.AtNS, &e.State, &e.Detail); err != nil {
			break
		}
		if len(ev.Events) >= q.Limit {
			out.Truncated = true
			break
		}
		ev.Events = append(ev.Events, e)
	}
	if e := events.Err(); err == nil {
		err = e
	}
	if e := events.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	reviews, err := tx.QueryContext(ctx, "SELECT seq,at,decision FROM recovery_audit WHERE operation_id=? ORDER BY seq LIMIT ?", rec.ID, q.Limit+1)
	if err != nil {
		return err
	}
	for reviews.Next() {
		var e ActivityReview
		if err = reviews.Scan(&e.Seq, &e.AtNS, &e.Decision); err != nil {
			break
		}
		if len(ev.Reviews) >= q.Limit {
			out.Truncated = true
			break
		}
		ev.Reviews = append(ev.Reviews, e)
	}
	if e := reviews.Err(); err == nil {
		err = e
	}
	if e := reviews.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	claims, err := tx.QueryContext(ctx, "SELECT resource_key FROM operation_resource_claims WHERE operation_id=? AND layer=? ORDER BY resource_key LIMIT ?", rec.ID, q.Layer, q.Limit+1)
	if err != nil {
		return err
	}
	for claims.Next() {
		var key string
		if err = claims.Scan(&key); err != nil {
			break
		}
		if len(ev.ResourceKeys) >= q.Limit {
			out.Truncated = true
			break
		}
		ev.ResourceKeys = append(ev.ResourceKeys, key)
	}
	if e := claims.Err(); err == nil {
		err = e
	}
	if e := claims.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	var observed int64
	var exit sql.NullInt64
	var truncated int
	err = tx.QueryRowContext(ctx, "SELECT observed_at,exit_code,truncation FROM exec_outputs WHERE operation_id=?", rec.ID).Scan(&observed, &exit, &truncated)
	if err == nil {
		ev.Exec = &ActivityExec{ObservedAtNS: observed, Truncated: truncated != 0}
		if exit.Valid {
			e := exit.Int64
			ev.Exec.ExitCode = &e
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	gcRows, err := tx.QueryContext(ctx, "SELECT stream,state,claimed_at FROM exec_output_gc WHERE operation_id=? ORDER BY stream LIMIT ?", rec.ID, q.Limit+1)
	if err != nil {
		return err
	}
	for gcRows.Next() {
		var item ActivityGC
		if err = gcRows.Scan(&item.Stream, &item.State, &item.AtNS); err != nil {
			break
		}
		if len(ev.GC) >= q.Limit {
			out.Truncated = true
			break
		}
		ev.GC = append(ev.GC, item)
	}
	if e := gcRows.Err(); err == nil {
		err = e
	}
	if e := gcRows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	peerRows, err := tx.QueryContext(ctx, "SELECT source_uuid,target_uuid,status,desired,operation_id,COALESCE(last_verified_operation_id,'') "+
		" FROM peer_grants WHERE project=? AND (operation_id=? OR last_verified_operation_id=?) ORDER BY source_uuid,target_uuid LIMIT ?",
		q.Layer, rec.ID, rec.ID, q.Limit+1)
	if err != nil {
		return err
	}
	for peerRows.Next() {
		var p ActivityPeer
		var desired int
		if err = peerRows.Scan(&p.SourceUUID, &p.TargetUUID, &p.Status, &desired, &p.PendingID, &p.VerifiedID); err != nil {
			break
		}
		if len(ev.Peers) >= q.Limit {
			out.Truncated = true
			break
		}
		p.Desired = desired == 1
		ev.Peers = append(ev.Peers, p)
	}
	if e := peerRows.Err(); err == nil {
		err = e
	}
	if e := peerRows.Close(); err == nil {
		err = e
	}
	if err != nil {
		return err
	}
	if rec.BoxUUID != "" {
		err = tx.QueryRowContext(ctx, "SELECT state FROM owned_boxes WHERE uuid=? AND project=? AND incarnation=?",
			rec.BoxUUID, q.Layer, rec.Incarnation).Scan(&ev.OwnedBoxState)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	if rec.Action == "box_peer_set" && len(ev.Peers) == 0 {
		ev.Limitations = append(ev.Limitations, "PEER_LINK_NOT_RECORDED_OR_SUPERSEDED")
	}
	if rec.State == "UNKNOWN" {
		ev.Limitations = append(ev.Limitations, "HOST_LOCAL_REVIEW_NO_REPLAY")
	}
	return nil
}
