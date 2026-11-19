package state

import (
	"context"
	"database/sql"
	"errors"
	"sort"
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
}

type ActivityIssue struct {
	OperationID string
	BoxUUID     string
	ResourceRef string
	Reason      string
	State       string
	AtNS        int64
}

type ActivityEvent struct {
	Seq    int64
	AtNS   int64
	State  string
	Detail string
}

type ActivityReview struct {
	Seq      int64
	AtNS     int64
	Decision string
}

type ActivityPeer struct {
	SourceUUID string
	TargetUUID string
	Status     string
	Desired    bool
	PendingID  string
	VerifiedID string
}

type ActivityGC struct {
	Stream string
	State  string
	AtNS   int64
}

type ActivityExec struct {
	ObservedAtNS int64
	ExitCode     *int64
	Truncated    bool
}

type ActivityEvidence struct {
	Original      Receipt
	Events        []ActivityEvent
	Reviews       []ActivityReview
	ResourceKeys  []string
	Exec          *ActivityExec
	GC            []ActivityGC
	Peers         []ActivityPeer
	OwnedBoxState string
	Limitations   []string
}

type ActivityResult struct {
	Source            string
	Scope             string
	AsOfNS            int64
	Coverage          string
	Truncated         bool
	AttentionComplete bool
	Operations        []Receipt
	Issues            []ActivityIssue
	Evidence          *ActivityEvidence
}

func (q ActivityQuery) normalize() (ActivityQuery, error) {
	if q.Layer == "" || len(q.Layer) > 64 || !validHexID(q.Requester) ||
		!validRef(q.WorkRef) || !validRef(q.AgentRef) ||
		(q.BoxUUID != "" && !validHexID(q.BoxUUID)) ||
		(q.OperationID != "" && !validHexID(q.OperationID)) ||
		q.Limit < 0 || q.Limit > 50 {
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
	where, args := q.where()
	rows, err := tx.QueryContext(ctx, "SELECT "+activityColumns+" FROM operations o WHERE "+where+
		" ORDER BY o.created_at DESC,o.id DESC LIMIT ?", append(args, q.Limit+1)...)
	if err != nil {
		return err
	}
	defer rows.Close()
	out.Operations = []Receipt{}
	for rows.Next() {
		rec, e := scanActivityOperation(rows)
		if e != nil {
			return e
		}
		if len(out.Operations) >= q.Limit {
			out.Truncated = true
			break
		}
		out.Operations = append(out.Operations, rec)
	}
	return rows.Err()
}

func readActivityIssues(ctx context.Context, tx *sql.Tx, q ActivityQuery, out *ActivityResult) error {
	where, args := q.where()
	stmt := "SELECT o.id,o.box_uuid,o.resource_key,o.state,o.created_at,o.action FROM operations o WHERE " + where +
		" AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')" +
		" AND (o.action='box_peer_set' OR NOT EXISTS (SELECT 1 FROM recovery_audit r WHERE r.operation_id=o.id AND r.decision='RELEASED'))" +
		" ORDER BY o.created_at ASC,o.id ASC LIMIT ?"
	rows, err := tx.QueryContext(ctx, stmt, append(args, q.Limit+1)...)
	if err != nil {
		return err
	}
	issues := []ActivityIssue{}
	for rows.Next() {
		var id, box, resource, state, action string
		var at int64
		if err = rows.Scan(&id, &box, &resource, &state, &at, &action); err != nil {
			break
		}
		reason := "UNRESOLVED_OPERATION"
		if action == "box_peer_set" {
			reason = "PROJECT_PEER_BARRIER"
		}
		issues = append(issues, ActivityIssue{OperationID: id, BoxUUID: box,
			ResourceRef: resource, State: state, Reason: reason, AtNS: at})
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
	// Terminal operations can still block native Box reuse while original
	// stdout/stderr deletion is in flight or awaits separate human review.
	gcStmt := "SELECT g.operation_id,g.box_uuid,g.name,g.state,g.claimed_at FROM exec_output_gc g " +
		" JOIN operations o ON o.id=g.operation_id WHERE " + where +
		" AND g.project=o.layer AND g.state IN ('DELETE_SENT','REVIEW_REQUIRED')" +
		" ORDER BY g.claimed_at ASC,g.operation_id ASC LIMIT ?"
	gcRows, err := tx.QueryContext(ctx, gcStmt, append(args, q.Limit+1)...)
	if err != nil {
		return err
	}
	for gcRows.Next() {
		var id, box, name, state string
		var at int64
		if err = gcRows.Scan(&id, &box, &name, &state, &at); err != nil {
			break
		}
		issues = append(issues, ActivityIssue{OperationID: id, BoxUUID: box,
			ResourceRef: "box:" + name, State: state, Reason: "EXEC_OUTPUT_GC_BLOCKER", AtNS: at})
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
	// Read the Peer ledger independently of operation timestamps; do not
	// guess an original ID from network names or a previous completed grant.
	peerSQL := "SELECT p.operation_id,p.source_uuid,p.target_uuid,p.status FROM peer_grants p " +
		" WHERE p.project=? AND p.status IN ('PENDING','UNKNOWN')"
	peerArgs := []any{q.Layer}
	if q.BoxUUID != "" {
		peerSQL += " AND (p.source_uuid=? OR p.target_uuid=?)"
		peerArgs = append(peerArgs, q.BoxUUID, q.BoxUUID)
	}
	peerSQL += " ORDER BY p.source_uuid,p.target_uuid LIMIT ?"
	peerRows, err := tx.QueryContext(ctx, peerSQL, append(peerArgs, q.Limit+1)...)
	if err != nil {
		return err
	}
	for peerRows.Next() {
		var id, source, target, state string
		if err = peerRows.Scan(&id, &source, &target, &state); err != nil {
			break
		}
		// Pending-ID is not a proof of completed kernel state; ledger
		// issues expose the direction without expanding actor visibility.
		issues = append(issues, ActivityIssue{BoxUUID: source,
			ResourceRef: "peer:" + source + "/" + target,
			Reason:      "PEER_LEDGER_UNRESOLVED", State: state})
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
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].AtNS == issues[j].AtNS {
			return issues[i].OperationID < issues[j].OperationID
		}
		return issues[i].AtNS < issues[j].AtNS
	})
	if len(issues) > q.Limit {
		out.Truncated = true
		issues = issues[:q.Limit]
	}
	out.Issues = issues
	// Not yet complete for all live fences/proxies and historical filter
	// combinations; never promise a negative proof of no blockers.
	out.AttentionComplete = false
	return nil
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
