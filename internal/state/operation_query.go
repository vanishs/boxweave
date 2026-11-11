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
	"fmt"
	"strings"
)

// Client correlation is an untrusted, controller-local label. It never
// authenticates an Agent, grants read access, or locks a Box.
func validRef(s string) bool {
	if s == "" {
		return true
	}
	if len(s) > 96 || strings.Contains(s, "..") || strings.Contains(s, "//") {
		return false
	}
	for _, c := range []byte(s) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' ||
			c >= '0' && c <= '9' || c == '_' || c == '-' || c == '.' || c == '/' {
			continue
		}
		return false
	}
	return true
}

func ValidCorrelationRef(s string) bool { return validRef(s) }

type NextAction struct {
	Kind        string `json:"kind"`
	Tool        string `json:"tool,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
	WaitMS      int    `json:"wait_ms,omitempty"`
}

func nextAction(status, action, id string) NextAction {
	switch status {
	case "ADMITTED", "DISPATCHING", "RUNNING", "VERIFYING":
		return NextAction{Kind: "READ_OPERATION", Tool: "operation_get", OperationID: id}
	case "UNKNOWN":
		return NextAction{Kind: "HOST_LOCAL_REVIEW"}
	case "SUCCEEDED":
		if action == "box_exec" {
			return NextAction{Kind: "READ_OUTPUT", Tool: "operation_output_get", OperationID: id}
		}
	}
	return NextAction{Kind: "NONE"}
}

// OperationQuery is always authorized BEFORE optional label filters are
// applied. A caller-provided Agent/work label can never expand visibility.
type OperationQuery struct {
	Layer     string
	Requester string
	Role      string
	ActorID   string
	WorkRef   string
	AgentRef  string
	Action    string
	State     string
	SinceNS   int64
	UntilNS   int64
	Limit     int
	Cursor    string
}

type OperationPage struct {
	Operations []Receipt `json:"operations"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type operationCursor struct {
	CreatedAt int64  `json:"t"`
	ID        string `json:"i"`
	// SQLite operations are append-only. A first-page, authorized rowid
	// ceiling keeps later commits out of the original pagination walk.
	// This upper bound is HMAC-protected along with scope and position.
	MaxRowID  int64  `json:"r"`
	Scope     string `json:"s"`
	MAC       string `json:"m"`
}

func validHexID(id string) bool {
	if len(id) != 32 {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func validOperationState(s string) bool {
	switch s {
	case "", "ADMITTED", "DISPATCHING", "RUNNING", "VERIFYING", "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED":
		return true
	default:
		return false
	}
}

func (q OperationQuery) normalize() (OperationQuery, error) {
	if q.Layer == "" || len(q.Layer) > 64 || !validHexID(q.Requester) ||
		(q.Role != "admin" && q.Role != "viewer") ||
		!validRef(q.WorkRef) || !validRef(q.AgentRef) ||
		len(q.Action) > 64 || !validOperationState(q.State) ||
		q.SinceNS < 0 || q.UntilNS < 0 ||
		(q.UntilNS > 0 && q.SinceNS > q.UntilNS) ||
		len(q.Cursor) > 512 || q.Limit < 0 || q.Limit > 50 {
		return q, ErrInvalid
	}
	if q.ActorID != "" && !validHexID(q.ActorID) {
		return q, ErrInvalid
	}
	if q.Role == "viewer" {
		if q.ActorID != "" && q.ActorID != q.Requester {
			return q, ErrNotFound
		}
		q.ActorID = q.Requester
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	return q, nil
}

func cursorScope(q OperationQuery) string {
	q.Cursor = ""
	b, _ := json.Marshal(q)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func cursorMAC(secret []byte, c operationCursor) string {
	h := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(h, "%s\x00%d\x00%s\x00%d", c.Scope, c.CreatedAt, c.ID, c.MaxRowID)
	return hex.EncodeToString(h.Sum(nil))
}

// ListOperations never scans unbounded history and never accepts a cursor
// across actors, layers, roles or filters. A forged cursor cannot widen the
// parametrized SQL authorization even if its MAC implementation is altered.
func (s *Store) ListOperations(ctx context.Context, query OperationQuery) (OperationPage, error) {
	q, err := query.normalize()
	if err != nil {
		return OperationPage{}, err
	}
	var secret []byte
	if err := s.db.QueryRowContext(ctx, "SELECT secret FROM cursor_secrets WHERE id=1").Scan(&secret); err != nil {
		return OperationPage{}, err
	}
	if len(secret) != 32 {
		return OperationPage{}, errors.New("invalid private cursor key")
	}
	scope := cursorScope(q)
	var cursor operationCursor
	if q.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(q.Cursor)
		if e != nil || len(raw) > 256 || json.Unmarshal(raw, &cursor) != nil ||
			!validHexID(cursor.ID) || cursor.CreatedAt <= 0 || cursor.MaxRowID <= 0 || cursor.Scope != scope ||
			!hmac.Equal([]byte(cursor.MAC), []byte(cursorMAC(secret, cursor))) {
			return OperationPage{}, ErrInvalid
		}
	}
	var args []any
	parts := []string{"layer=?"}
	args = append(args, q.Layer)
	if q.ActorID != "" {
		parts = append(parts, "actor_id=?")
		args = append(args, q.ActorID)
	}
	if q.WorkRef != "" {
		parts = append(parts, "work_ref=?")
		args = append(args, q.WorkRef)
	}
	if q.AgentRef != "" {
		parts = append(parts, "agent_ref=?")
		args = append(args, q.AgentRef)
	}
	if q.Action != "" {
		parts = append(parts, "action=?")
		args = append(args, q.Action)
	}
	if q.State != "" {
		parts = append(parts, "state=?")
		args = append(args, q.State)
	}
	if q.SinceNS > 0 {
		parts = append(parts, "created_at>=?")
		args = append(args, q.SinceNS)
	}
	if q.UntilNS > 0 {
		parts = append(parts, "created_at<=?")
		args = append(args, q.UntilNS)
	}
	// A cursor without a stable admission watermark is not a snapshot:
	// later rows inserted while a client pages ASC would appear in the old
	// walk and silently change the last page. The highest authorized
	// SQLite rowid observed on page one cannot be exceeded by any later
	// insert committed to this append-only operations table. Bind it to
	// the cursor MAC, never trust a client-selected upper bound.
	var maxRowID int64
	if q.Cursor == "" {
		anchorSQL := "SELECT COALESCE(MAX(rowid),0) FROM operations WHERE " + strings.Join(parts, " AND ")
		if err := s.db.QueryRowContext(ctx, anchorSQL, args...).Scan(&maxRowID); err != nil {
			return OperationPage{}, err
		}
	} else {
		maxRowID = cursor.MaxRowID
	}
	parts = append(parts, "rowid<=?")
	args = append(args, maxRowID)
	if q.Cursor != "" {
		parts = append(parts, "(created_at>? OR (created_at=? AND id>?))")
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	stmt := `SELECT id,tracking_id,state,native_operation_id,work_ref,agent_ref,
		action,box_uuid,incarnation,error_class,created_at,updated_at
		FROM operations WHERE ` + strings.Join(parts, " AND ") +
		" ORDER BY created_at ASC,id ASC LIMIT ?"
	args = append(args, q.Limit+1)
	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return OperationPage{}, err
	}
	defer rows.Close()
	out := OperationPage{Operations: make([]Receipt, 0, q.Limit)}
	for rows.Next() {
		var r Receipt
		if err := rows.Scan(&r.ID, &r.TrackingID, &r.State, &r.NativeOperationID,
			&r.WorkRef, &r.AgentRef, &r.Action, &r.BoxUUID, &r.Incarnation,
			&r.ErrorClass, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return OperationPage{}, err
		}
		if len(out.Operations) == q.Limit {
			// A lookahead row proves that the next page actually exists.
			last := out.Operations[len(out.Operations)-1]
			c := operationCursor{CreatedAt: last.CreatedAt, ID: last.ID, MaxRowID: maxRowID, Scope: scope}
			c.MAC = cursorMAC(secret, c)
			b, err := json.Marshal(c)
			if err != nil {
				return OperationPage{}, err
			}
			out.NextCursor = base64.RawURLEncoding.EncodeToString(b)
			break
		}
		r.NextAction = nextAction(r.State, r.Action, r.ID)
		out.Operations = append(out.Operations, r)
	}
	if err := rows.Err(); err != nil {
		return OperationPage{}, err
	}
	return out, nil
}

// GetScopedOperation is the read boundary for operation_get/output/cancel:
// neither a guessed original ID nor a borrowed work_ref is authorization.
func (s *Store) GetScopedOperation(ctx context.Context, layer, requester, role, id string) (Receipt, error) {
	if layer == "" || !validHexID(requester) || !validHexID(id) ||
		(role != "admin" && role != "viewer") {
		return Receipt{}, ErrInvalid
	}
	var r Receipt
	var err error
	if role == "admin" {
		err = s.db.QueryRowContext(ctx, `SELECT id,tracking_id,state,native_operation_id,
			work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,updated_at
			FROM operations WHERE id=? AND layer=?`, id, layer).Scan(
			&r.ID, &r.TrackingID, &r.State, &r.NativeOperationID,
			&r.WorkRef, &r.AgentRef, &r.Action, &r.BoxUUID, &r.Incarnation,
			&r.ErrorClass, &r.CreatedAt, &r.UpdatedAt)
	} else {
		err = s.db.QueryRowContext(ctx, `SELECT id,tracking_id,state,native_operation_id,
			work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,updated_at
			FROM operations WHERE id=? AND layer=? AND actor_id=?`, id, layer, requester).Scan(
			&r.ID, &r.TrackingID, &r.State, &r.NativeOperationID,
			&r.WorkRef, &r.AgentRef, &r.Action, &r.BoxUUID, &r.Incarnation,
			&r.ErrorClass, &r.CreatedAt, &r.UpdatedAt)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, ErrNotFound
	}
	if err != nil {
		return Receipt{}, err
	}
	r.NextAction = nextAction(r.State, r.Action, r.ID)
	return r, nil
}
