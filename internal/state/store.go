// Package state is the durable, local ownership and admission boundary.
// No native effect may run before an operation has been journaled and claimed.
package state

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const SchemaVersion = 1

// Fresh-install S2 baseline: there is no retained deployed v1 database yet.
// Both tables and indexes are exact-fingerprinted; old S1 test DBs are refused,
// never silently migrated, adopted or deleted. Once deployed, schema changes
// will require reviewed migration and rollback.
var schemaDDL = []string{
	"CREATE TABLE schema_version (version INTEGER NOT NULL)",
	"CREATE TABLE credentials (id TEXT PRIMARY KEY, token_digest TEXT NOT NULL UNIQUE, role TEXT NOT NULL CHECK(role IN ('admin','viewer')), expires_at INTEGER NOT NULL, revoked INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL)",
	"CREATE TABLE owned_boxes (uuid TEXT PRIMARY KEY, project TEXT NOT NULL, name TEXT NOT NULL, incarnation TEXT NOT NULL, state TEXT NOT NULL, created_at INTEGER NOT NULL, UNIQUE(project,name,incarnation))",
	"CREATE TABLE operations (id TEXT PRIMARY KEY, tracking_id TEXT NOT NULL, request_id TEXT NOT NULL, layer TEXT NOT NULL, actor_id TEXT NOT NULL, token_id TEXT NOT NULL, key_digest TEXT NOT NULL, action TEXT NOT NULL, intent_digest TEXT NOT NULL, safe_intent BLOB NOT NULL, work_ref TEXT NOT NULL DEFAULT '', agent_ref TEXT NOT NULL DEFAULT '', resource_key TEXT NOT NULL DEFAULT '', box_uuid TEXT NOT NULL, incarnation TEXT NOT NULL, native_operation_id TEXT NOT NULL DEFAULT '', state TEXT NOT NULL, error_class TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, UNIQUE(layer,actor_id,key_digest))",
	"CREATE TABLE operation_events (seq INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL REFERENCES operations(id), at INTEGER NOT NULL, state TEXT NOT NULL, safe_detail TEXT NOT NULL DEFAULT '')",
	"CREATE TABLE recovery_audit (seq INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL REFERENCES operations(id), at INTEGER NOT NULL, reviewer TEXT NOT NULL, decision TEXT NOT NULL, evidence_digest TEXT NOT NULL)",
	"CREATE TABLE cursor_secrets (id INTEGER PRIMARY KEY CHECK(id=1), secret BLOB NOT NULL CHECK(length(secret)=32))",
	"CREATE TABLE exec_outputs (operation_id TEXT PRIMARY KEY REFERENCES operations(id), stdout_path TEXT NOT NULL DEFAULT '', stderr_path TEXT NOT NULL DEFAULT '', exit_code INTEGER, truncation INTEGER NOT NULL DEFAULT 0, observed_at INTEGER NOT NULL)",
	"CREATE TABLE exec_output_gc (operation_id TEXT NOT NULL REFERENCES operations(id), stream TEXT NOT NULL CHECK(stream IN ('stdout','stderr')), project TEXT NOT NULL, box_uuid TEXT NOT NULL, incarnation TEXT NOT NULL, name TEXT NOT NULL, native_operation_id TEXT NOT NULL, path TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN ('DELETE_SENT','EXPIRED','REVIEW_REQUIRED','REVIEWED_RELEASED')), claimed_at INTEGER NOT NULL, decided_at INTEGER NOT NULL DEFAULT 0, PRIMARY KEY(operation_id,stream))",
	"CREATE TABLE exec_output_gc_audit (seq INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL, stream TEXT NOT NULL, at INTEGER NOT NULL, event TEXT NOT NULL CHECK(event IN ('BEFORE_DELETE','EXPIRED','REVIEW_REQUIRED','RECOVERED_UNCERTAIN','CHALLENGE_ISSUED','REVIEWED_RELEASED')), reviewer TEXT NOT NULL DEFAULT '', evidence_digest TEXT NOT NULL DEFAULT '', FOREIGN KEY(operation_id,stream) REFERENCES exec_output_gc(operation_id,stream))",
}

var schemaIndexes = []string{
	"CREATE INDEX operations_layer_time ON operations(layer,created_at,id)",
	"CREATE INDEX operations_actor_time ON operations(layer,actor_id,created_at,id)",
	"CREATE INDEX operations_work_time ON operations(layer,work_ref,created_at,id)",
	"CREATE INDEX operations_agent_time ON operations(layer,agent_ref,created_at,id)",
	"CREATE INDEX operations_state_time ON operations(layer,state,created_at,id)",
	"CREATE INDEX operations_resource_claim ON operations(layer,resource_key,state)",
	"CREATE UNIQUE INDEX owned_active_name ON owned_boxes(project,name) WHERE state='OWNED'",
	"CREATE INDEX exec_output_gc_box_status ON exec_output_gc(project,box_uuid,state)",
}

var (
	ErrConflict       = errors.New("idempotency key conflicts with original intent")
	ErrNotFound       = errors.New("operation not found")
	ErrInvalid        = errors.New("invalid admission")
	ErrRevoked        = errors.New("actor credential revoked before dispatch")
	ErrState          = errors.New("invalid operation transition")
	ErrResourceLocked = errors.New("original unresolved resource claim is still locked")
)

type Store struct{ db *sql.DB }

type Token struct {
	ID      string
	Role    string
	Expires time.Time
	Revoked bool
}

type Admission struct {
	Layer   string
	ActorID string
	TokenID string
	Key     string
	Action  string
	// SafeIntent must be a domain-approved, secret-free canonical JSON projection.
	// It is not an arbitrary raw MCP request or guest command.
	SafeIntent  []byte
	BoxUUID     string
	Incarnation string
	RequestID   string
	// ResourceKey is a typed, domain-constructed per-native-Box name lock, never
	// a caller-provided arbitrary SQL scope. It survives create before UUID exists.
	ResourceKey string
	WorkRef     string
	AgentRef    string
}

type Receipt struct {
	ID                string     `json:"operation_id"`
	TrackingID        string     `json:"tracking_id"`
	State             string     `json:"state"`
	NativeOperationID string     `json:"native_operation_id,omitempty"`
	Duplicate         bool       `json:"duplicate,omitempty"`
	WorkRef           string     `json:"work_ref,omitempty"`
	AgentRef          string     `json:"agent_ref,omitempty"`
	Action            string     `json:"action,omitempty"`
	BoxUUID           string     `json:"box_uuid,omitempty"`
	Incarnation       string     `json:"incarnation,omitempty"`
	ErrorClass        string     `json:"error_class,omitempty"`
	CreatedAt         int64      `json:"created_at,omitempty"`
	UpdatedAt         int64      `json:"updated_at,omitempty"`
	NextAction        NextAction `json:"next_action"`
}

func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func digest(s string) string          { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func DigestToken(token string) string { return digest(token) }

func Open(path string) (*Store, error) {
	if path == "" || !filepath.IsAbs(path) || filepath.Clean(path) != path ||
		strings.ContainsAny(path, "?\x00") {
		return nil, fmt.Errorf("absolute canonical literal database path required")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return nil, err
	}
	// A private final parent alone is not enough: a symlink anywhere in the
	// ancestor path would permit two names for the same SQLite inode and two
	// independent .serve.lock files. Refuse aliases before opening the DB.
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, err
	}
	if physical != parent {
		return nil, fmt.Errorf("database parent path contains a symlink alias")
	}
	if st, err := os.Lstat(parent); err != nil {
		return nil, err
	} else if !st.IsDir() || st.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("database parent must be a private real directory")
	}
	// Never follow a preexisting link into an unmanaged host file.
	flags := os.O_RDWR
	if st, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		flags |= os.O_CREATE | os.O_EXCL
	} else if err != nil {
		return nil, err
	} else if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("database path must be a private regular file, not a symlink")
	}
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("database file permissions unsafe")
	}
	// Two hardlinked DB filenames have different adjacent serve-lock paths
	// but the same underlying SQLite inode; reject before init/recovery.
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok || stat.Nlink != 1 {
		return nil, fmt.Errorf("database hardlinks or unknown inode metadata refused")
	}
	db, err := sql.Open("sqlite3", "file:"+path+"?_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.init(flags&os.O_EXCL != 0); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) init(fresh bool) error {
	if !fresh {
		// Inspect existing files with read-only SQL BEFORE any journal mode,
		// DDL or migration: never modify a foreign/unversioned database.
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='schema_version'").Scan(&n); err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("existing database missing version: refuse foreign adoption")
		}
		var version int
		if err := s.db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version); err != nil {
			return err
		}
		if version != SchemaVersion {
			return fmt.Errorf("unsupported schema %d; refuse automatic migration", version)
		}
		// A LIMIT 1 version read alone can hide a second conflicting migration
		// record. Never choose one history or adopt unknown application tables.
		var versionRows int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&versionRows); err != nil {
			return err
		}
		if versionRows != 1 {
			return fmt.Errorf("ambiguous SQLite migration history: %d version rows", versionRows)
		}
		rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT GLOB 'sqlite_*'")
		if err != nil {
			return err
		}
		actualTables := make(map[string]bool)
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				_ = rows.Close()
				return err
			}
			actualTables[name] = true
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if len(actualTables) != len(schemaDDL) {
			return fmt.Errorf("unexpected SQLite schema/migration tables: refuse foreign adoption")
		}
		for _, expected := range schemaDDL {
			if !actualTables[strings.Fields(expected)[2]] {
				return fmt.Errorf("missing declared schema table: refuse foreign adoption")
			}
		}
		for _, expected := range schemaDDL {
			name := strings.Fields(expected)[2]
			var actual string
			if err := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&actual); err != nil {
				return fmt.Errorf("missing exact schema table %s: %w", name, err)
			}
			if actual != expected {
				return fmt.Errorf("schema version %d changed table %s: refuse undeclared migration", SchemaVersion, name)
			}
		}
		// Validate every index in the new initial schema as well as tables. No
		// silently changed/foreign index may influence a security-scoped query.
		indexRows, err := s.db.Query("SELECT name,sql FROM sqlite_master WHERE type='index' AND sql IS NOT NULL")
		if err != nil {
			return err
		}
		actualIndexes := map[string]string{}
		for indexRows.Next() {
			var name, sqlText string
			if err := indexRows.Scan(&name, &sqlText); err != nil {
				_ = indexRows.Close()
				return err
			}
			actualIndexes[name] = sqlText
		}
		if err := indexRows.Err(); err != nil {
			_ = indexRows.Close()
			return err
		}
		if err := indexRows.Close(); err != nil {
			return err
		}
		if len(actualIndexes) != len(schemaIndexes) {
			return errors.New("unexpected SQLite schema indexes: refuse foreign adoption")
		}
		for _, expected := range schemaIndexes {
			words := strings.Fields(expected)
			nameOffset := 2 // CREATE INDEX <name>
			if len(words) > 3 && words[1] == "UNIQUE" {
				nameOffset = 3 // CREATE UNIQUE INDEX <name>
			}
			name := words[nameOffset]
			if actualIndexes[name] != expected {
				return fmt.Errorf("schema index %s changed: refuse foreign adoption", name)
			}
		}
		// A foreign trigger/view can silently change mutation semantics even when
		// the six declared table names and schema_version are unchanged.
		if err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type IN ('trigger','view')").Scan(&n); err != nil {
			return err
		}
		if n != 0 {
			return errors.New("unexpected SQLite trigger/view: refuse foreign database")
		}
		var integrity string
		if err := s.db.QueryRow("PRAGMA quick_check").Scan(&integrity); err != nil {
			return err
		}
		if integrity != "ok" {
			return fmt.Errorf("integrity check failed")
		}
	}
	for _, q := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=FULL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	if !fresh {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return err
	}
	schema := append(append([]string{}, schemaDDL...), schemaIndexes...)
	schema = append(schema, "INSERT INTO schema_version(version) VALUES (1)")
	for _, q := range schema {
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("INSERT INTO cursor_secrets(id,secret) VALUES (1,?)", key); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Store) Schema(ctx context.Context) (int, error) {
	var v int
	err := s.db.QueryRowContext(ctx, "SELECT version FROM schema_version").Scan(&v)
	return v, err
}

func (s *Store) IssueToken(ctx context.Context, role string, ttl time.Duration) (string, Token, error) {
	if role != "admin" && role != "viewer" || ttl <= 0 || ttl > 365*24*time.Hour {
		return "", Token{}, ErrInvalid
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", Token{}, err
	}
	raw := "bw1_" + hex.EncodeToString(b)
	id, err := newID()
	if err != nil {
		return "", Token{}, err
	}
	t := Token{ID: id, Role: role, Expires: time.Now().UTC().Add(ttl)}
	_, err = s.db.ExecContext(ctx, "INSERT INTO credentials (id,token_digest,role,expires_at,created_at) VALUES (?,?,?,?,?)", id, DigestToken(raw), role, t.Expires.Unix(), time.Now().Unix())
	if err != nil {
		return "", Token{}, err
	}
	return raw, t, nil
}
func (s *Store) VerifyToken(ctx context.Context, raw string) (Token, error) {
	if !strings.HasPrefix(raw, "bw1_") || len(raw) != 68 {
		return Token{}, ErrRevoked
	}
	var t Token
	var expires int64
	var revoked int
	err := s.db.QueryRowContext(ctx, "SELECT id,role,expires_at,revoked FROM credentials WHERE token_digest=?", DigestToken(raw)).Scan(&t.ID, &t.Role, &expires, &revoked)
	if err != nil {
		return Token{}, ErrRevoked
	}
	t.Expires = time.Unix(expires, 0).UTC()
	t.Revoked = revoked != 0
	if t.Revoked || !t.Expires.After(time.Now()) {
		return Token{}, ErrRevoked
	}
	return t, nil
}
func (s *Store) RevokeToken(ctx context.Context, id string) error {
	r, err := s.db.ExecContext(ctx, "UPDATE credentials SET revoked=1 WHERE id=?", id)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrNotFound
	}
	return nil
}
func (s *Store) tokenActive(ctx context.Context, id string) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials WHERE id=? AND revoked=0 AND expires_at>?", id, time.Now().Unix()).Scan(&n)
	return n == 1, err
}

// RequireActiveTokenID rechecks the durable credential immediately before a
// private long-poll result leaves the controller. Bearer auth at HTTP ingress
// is not sufficient when a token may expire or be revoked during wait_ms.
// This never retries, cancels, or alters the already-admitted native operation.
func (s *Store) RequireActiveTokenID(ctx context.Context, tokenID string) error {
	if !validHexID(tokenID) {
		return ErrRevoked
	}
	active, err := s.tokenActive(ctx, tokenID)
	if err != nil {
		return err
	}
	if !active {
		return ErrRevoked
	}
	return nil
}

// Admit persists the original immutable intent hash and sanitized replay payload.
// Concurrent equal requests must return exactly the original IDs and cause no native effect.
func (s *Store) Admit(ctx context.Context, a Admission) (Receipt, error) {
	if a.Layer == "" || a.ActorID == "" || a.TokenID == "" || a.ActorID != a.TokenID || len(a.Key) < 16 || len(a.Key) > 128 || a.Action == "" ||
		len(a.SafeIntent) == 0 || len(a.SafeIntent) > 65536 || a.RequestID == "" ||
		!validRef(a.WorkRef) || !validRef(a.AgentRef) || len(a.ResourceKey) > 192 ||
		strings.ContainsAny(a.ResourceKey, "\x00\n\r") {
		return Receipt{}, ErrInvalid
	}
	// A non-secret digest avoids storing raw client key.
	keyHash := digest(a.Key)
	// Labels are stored immutably but are not part of the native intent hash.
	// A retry that omits optional attribution reuses the original labels.
	intentHash := digest(a.Action + "\x00" + string(a.SafeIntent) + "\x00" + a.BoxUUID + "\x00" + a.Incarnation)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Receipt{}, err
	}
	defer tx.Rollback()
	var found Receipt
	var priorHash, priorToken string
	err = tx.QueryRowContext(ctx, "SELECT id,tracking_id,state,intent_digest,token_id,native_operation_id,work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,updated_at FROM operations WHERE layer=? AND actor_id=? AND key_digest=?", a.Layer, a.ActorID, keyHash).Scan(&found.ID, &found.TrackingID, &found.State, &priorHash, &priorToken, &found.NativeOperationID, &found.WorkRef, &found.AgentRef, &found.Action, &found.BoxUUID, &found.Incarnation, &found.ErrorClass, &found.CreatedAt, &found.UpdatedAt)
	if err == nil {
		// Omitted optional labels do not erase the original attribution.
		// Explicitly contradicting a prior label always conflicts.
		if priorHash != intentHash || priorToken != a.TokenID ||
			(a.WorkRef != "" && a.WorkRef != found.WorkRef) ||
			(a.AgentRef != "" && a.AgentRef != found.AgentRef) {
			return Receipt{}, ErrConflict
		}
		found.Duplicate = true
		found.NextAction = nextAction(found.State, found.Action, found.ID)
		return found, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, err
	}
	// Reserve the native Box name across any potentially in-flight effect,
	// including a create that cannot know the native UUID until postread.
	// This is transactional with admission, not a process-local mutex.
	if a.ResourceKey != "" {
		var busy int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o
			WHERE o.layer=? AND o.resource_key=?
			AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
			AND NOT EXISTS (SELECT 1 FROM recovery_audit ra WHERE ra.operation_id=o.id AND ra.decision='RELEASED')`,
			a.Layer, a.ResourceKey).Scan(&busy)
		if err != nil {
			return Receipt{}, err
		}
		if busy != 0 {
			return Receipt{}, ErrResourceLocked
		}
	}
	// An original retained-output deletion currently in progress or requiring
	// independent review forbids admission of a new effect on the same native
	// Box/name. This participates in the SAME admission transaction as the
	// original mutation resource lock, preventing name-reuse TOCTOU.
	if a.BoxUUID != "" || a.ResourceKey != "" {
		var gcBusy int
		err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM exec_output_gc WHERE project=? "+
			"AND state IN ('DELETE_SENT','REVIEW_REQUIRED') AND "+
			"((?<>'' AND box_uuid=?) OR (?<>'' AND ('box:'||name)=?))",
			a.Layer, a.BoxUUID, a.BoxUUID, a.ResourceKey, a.ResourceKey).Scan(&gcBusy)
		if err != nil {
			return Receipt{}, err
		}
		if gcBusy > 0 {
			return Receipt{}, ErrResourceLocked
		}
	}
	if a.BoxUUID != "" && a.Incarnation != "" {
		// A different idempotency key cannot bypass an original UNKNOWN
		// native-resource claim. A local reviewed RELEASED audit may allow
		// a future NEW operation, but NEVER redispatches the original.
		var outstanding int
		err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM operations o
			WHERE o.box_uuid=? AND o.incarnation=?
			AND o.state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING','UNKNOWN')
			AND NOT EXISTS (SELECT 1 FROM recovery_audit a
			                WHERE a.operation_id=o.id AND a.decision='RELEASED')`,
			a.BoxUUID, a.Incarnation).Scan(&outstanding)
		if err != nil {
			return Receipt{}, err
		}
		if outstanding > 0 {
			return Receipt{}, ErrResourceLocked
		}
	}
	var active int
	err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials WHERE id=? AND revoked=0 AND expires_at>?", a.TokenID, time.Now().Unix()).Scan(&active)
	if err != nil {
		return Receipt{}, err
	}
	if active != 1 {
		return Receipt{}, ErrRevoked
	}
	id, err := newID()
	if err != nil {
		return Receipt{}, err
	}
	tracking, err := newID()
	if err != nil {
		return Receipt{}, err
	}
	now := time.Now().UnixNano()
	_, err = tx.ExecContext(ctx, "INSERT INTO operations (id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,work_ref,agent_ref,resource_key,box_uuid,incarnation,state,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		id, tracking, a.RequestID, a.Layer, a.ActorID, a.TokenID, keyHash, a.Action, intentHash, a.SafeIntent, a.WorkRef, a.AgentRef, a.ResourceKey, a.BoxUUID, a.Incarnation, "ADMITTED", now, now)
	if err != nil {
		return Receipt{}, err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state) VALUES (?,?,?)", id, now, "ADMITTED"); err != nil {
		return Receipt{}, err
	}
	if err = tx.Commit(); err != nil {
		return Receipt{}, err
	}
	return Receipt{ID: id, TrackingID: tracking, State: "ADMITTED", WorkRef: a.WorkRef, AgentRef: a.AgentRef, Action: a.Action, BoxUUID: a.BoxUUID, Incarnation: a.Incarnation, CreatedAt: now, UpdatedAt: now, NextAction: nextAction("ADMITTED", a.Action, id)}, nil
}
func (s *Store) Get(ctx context.Context, id string) (Receipt, error) {
	var r Receipt
	err := s.db.QueryRowContext(ctx, "SELECT id,tracking_id,state,native_operation_id,work_ref,agent_ref,action,box_uuid,incarnation,error_class,created_at,updated_at FROM operations WHERE id=?", id).Scan(&r.ID, &r.TrackingID, &r.State, &r.NativeOperationID, &r.WorkRef, &r.AgentRef, &r.Action, &r.BoxUUID, &r.Incarnation, &r.ErrorClass, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return r, ErrNotFound
	}
	if err == nil {
		r.NextAction = nextAction(r.State, r.Action, r.ID)
	}
	return r, err
}
func (s *Store) transition(ctx context.Context, id, expected, next, nativeID, errClass string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, "UPDATE operations SET state=?,native_operation_id=CASE WHEN ?!='' THEN ? ELSE native_operation_id END,error_class=?,updated_at=? WHERE id=? AND state=?",
		next, nativeID, nativeID, errClass, time.Now().UnixNano(), id, expected)
	if err != nil {
		return err
	}
	n, err := r.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrState
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)", id, time.Now().UnixNano(), next, errClass); err != nil {
		return err
	}
	return tx.Commit()
}

// Claim atomically verifies current token revocation and writes dispatch intent.
// Returning ErrRevoked means no native effect can be sent by this worker.
func (s *Store) Claim(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var token, state string
	if err = tx.QueryRowContext(ctx, "SELECT token_id,state FROM operations WHERE id=?", id).Scan(&token, &state); err != nil {
		return err
	}
	if state != "ADMITTED" {
		return ErrState
	}
	var active int
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM credentials WHERE id=? AND revoked=0 AND expires_at>?", token, time.Now().Unix()).Scan(&active); err != nil {
		return err
	}
	next, reason := "DISPATCHING", ""
	if active != 1 {
		next, reason = "FAILED", "AUTH_REVOKED_BEFORE_DISPATCH"
	}
	now := time.Now().UnixNano()
	if _, err = tx.ExecContext(ctx, "UPDATE operations SET state=?,error_class=?,updated_at=? WHERE id=? AND state='ADMITTED'", next, reason, now, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)", id, now, next, reason); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	if active != 1 {
		return ErrRevoked
	}
	return nil
}
func (s *Store) NativeAck(ctx context.Context, id, nativeID string) error {
	if nativeID == "" {
		return ErrInvalid
	}
	return s.transition(ctx, id, "DISPATCHING", "VERIFYING", nativeID, "")
}

// NativeAckUnknown preserves a trusted native Operation ID when a previously
// accepted native effect could not be durably ACKed in the normal transaction.
// The original remains UNKNOWN and retains its resource lock; no redispatch.
func (s *Store) NativeAckUnknown(ctx context.Context, id, nativeID string) error {
	if nativeID == "" {
		return ErrInvalid
	}
	return s.transition(ctx, id, "DISPATCHING", "UNKNOWN", nativeID, "NATIVE_ACK_DURABILITY_UNVERIFIED")
}
func (s *Store) Verified(ctx context.Context, id string) error {
	return s.transition(ctx, id, "VERIFYING", "SUCCEEDED", "", "")
}
func (s *Store) Unknown(ctx context.Context, id, from, reason string) error {
	if len(reason) > 120 {
		reason = reason[:120]
	}
	return s.transition(ctx, id, from, "UNKNOWN", "", reason)
}

// Recover never replays a possibly dispatched effect on process restart.
func (s *Store) Recover(ctx context.Context) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, "SELECT id,state FROM operations WHERE state IN ('ADMITTED','DISPATCHING','RUNNING','VERIFYING')")
	if err != nil {
		return 0, err
	}
	type original struct {
		id    string
		state string
	}
	var ids []original
	for rows.Next() {
		var record original
		if err = rows.Scan(&record.id, &record.state); err != nil {
			_ = rows.Close()
			return 0, err
		}
		ids = append(ids, record)
	}
	if err = rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	if err = rows.Close(); err != nil {
		return 0, err
	}
	for _, item := range ids {
		now := time.Now().UnixNano()
		next, reason := "UNKNOWN", "PROCESS_RESTART_OR_LOST_ACK"
		if item.state == "ADMITTED" {
			// No Claim was committed and therefore the process was forbidden
			// from dispatching native work. The in-memory handler is gone:
			// do not silently replay an arbitrary old guest command.
			next, reason = "FAILED", "PROCESS_RESTART_BEFORE_DISPATCH"
		}
		if _, err = tx.ExecContext(ctx,
			"UPDATE operations SET state=?,error_class=?,updated_at=? WHERE id=? AND state=?",
			next, reason, now, item.id, item.state); err != nil {
			return 0, err
		}
		if _, err = tx.ExecContext(ctx,
			"INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)",
			item.id, now, next, reason); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return int64(len(ids)), nil
}
