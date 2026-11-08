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
    "time"

    _ "github.com/mattn/go-sqlite3"
)

const SchemaVersion = 1

var (
    ErrConflict = errors.New("idempotency key conflicts with original intent")
    ErrNotFound = errors.New("operation not found")
    ErrInvalid = errors.New("invalid admission")
    ErrRevoked = errors.New("actor credential revoked before dispatch")
    ErrState = errors.New("invalid operation transition")
)

type Store struct { db *sql.DB }

type Token struct {
    ID string
    Role string
    Expires time.Time
    Revoked bool
}

type Admission struct {
    Layer string
    ActorID string
    TokenID string
    Key string
    Action string
    // SafeIntent must be a domain-approved, secret-free canonical JSON projection.
    // It is not an arbitrary raw MCP request or guest command.
    SafeIntent []byte
    BoxUUID string
    Incarnation string
    RequestID string
}

type Receipt struct {
    ID string `json:"operation_id"`
    TrackingID string `json:"tracking_id"`
    State string `json:"state"`
    NativeOperationID string `json:"native_operation_id,omitempty"`
    Duplicate bool `json:"duplicate,omitempty"`
}

func newID() (string,error) {
    b:=make([]byte,16)
    if _,err:=rand.Read(b);err!=nil {return "",err}
    return hex.EncodeToString(b),nil
}
func digest(s string) string {h:=sha256.Sum256([]byte(s));return hex.EncodeToString(h[:])}
func DigestToken(token string) string {return digest(token)}

func Open(path string) (*Store,error) {
    if path=="" || !filepath.IsAbs(path) || strings.ContainsAny(path,"?\x00") {return nil,fmt.Errorf("absolute literal database path required")}
    if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return nil,err}
    if st,err:=os.Stat(filepath.Dir(path));err!=nil{return nil,err}else if st.Mode().Perm()&0077!=0 {
        return nil,fmt.Errorf("database parent must not be group/world accessible")
    }
    f,err:=os.OpenFile(path,os.O_CREATE|os.O_RDWR,0600)
    if err!=nil{return nil,err}
    st,err:=f.Stat();_ = f.Close()
    if err!=nil{return nil,err}
    if !st.Mode().IsRegular() || st.Mode().Perm()&0077!=0{return nil,fmt.Errorf("database file permissions unsafe")}
    db,err:=sql.Open("sqlite3","file:"+path+"?_busy_timeout=5000&_foreign_keys=on")
    if err!=nil{return nil,err}
    db.SetMaxOpenConns(1)
    s:=&Store{db:db}
    if err:=s.init();err!=nil{_ = db.Close();return nil,err}
    return s,nil
}
func (s *Store) Close() error {return s.db.Close()}

func (s *Store) init() error {
    for _,q:=range []string{
        "PRAGMA journal_mode=WAL",
        "PRAGMA synchronous=FULL",
        "PRAGMA foreign_keys=ON",
        "CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)",
    } {if _,err:=s.db.Exec(q);err!=nil{return err}}
    var v int
    err:=s.db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&v)
    if err!=nil && !errors.Is(err,sql.ErrNoRows){return err}
    if err==nil && v!=SchemaVersion{return fmt.Errorf("unsupported schema %d; refuse mutation/automatic migration",v)}
    if errors.Is(err,sql.ErrNoRows){
        // Only initialize an empty, new database. Existing but unversioned files fail closed.
        var n int
        if e:=s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name!='schema_version'").Scan(&n);e!=nil{return e}
        if n!=0{return fmt.Errorf("existing unversioned database, refusing adoption")}
        tx,e:=s.db.Begin();if e!=nil{return e}
        defer tx.Rollback()
        schema:=[]string{
            "CREATE TABLE credentials (id TEXT PRIMARY KEY, token_digest TEXT NOT NULL UNIQUE, role TEXT NOT NULL CHECK(role IN ('admin','viewer')), expires_at INTEGER NOT NULL, revoked INTEGER NOT NULL DEFAULT 0, created_at INTEGER NOT NULL)",
            "CREATE TABLE owned_boxes (uuid TEXT PRIMARY KEY, project TEXT NOT NULL, name TEXT NOT NULL, incarnation TEXT NOT NULL, state TEXT NOT NULL, created_at INTEGER NOT NULL, UNIQUE(project,name,incarnation))",
            "CREATE TABLE operations (id TEXT PRIMARY KEY, tracking_id TEXT NOT NULL, request_id TEXT NOT NULL, layer TEXT NOT NULL, actor_id TEXT NOT NULL, token_id TEXT NOT NULL, key_digest TEXT NOT NULL, action TEXT NOT NULL, intent_digest TEXT NOT NULL, safe_intent BLOB NOT NULL, box_uuid TEXT NOT NULL, incarnation TEXT NOT NULL, native_operation_id TEXT NOT NULL DEFAULT '', state TEXT NOT NULL, error_class TEXT NOT NULL DEFAULT '', created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL, UNIQUE(layer,actor_id,key_digest))",
            "CREATE TABLE operation_events (seq INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL REFERENCES operations(id), at INTEGER NOT NULL, state TEXT NOT NULL, safe_detail TEXT NOT NULL DEFAULT '')",
            "CREATE TABLE recovery_audit (seq INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL REFERENCES operations(id), at INTEGER NOT NULL, reviewer TEXT NOT NULL, decision TEXT NOT NULL, evidence_digest TEXT NOT NULL)",
            "INSERT INTO schema_version(version) VALUES (1)",
        }
        for _,q:=range schema{if _,e=tx.Exec(q);e!=nil{return e}}
        return tx.Commit()
    }
    // Do not replace or silently repair an existing inconsistent ledger.
    for _,name:=range []string{"credentials","owned_boxes","operations","operation_events","recovery_audit"} {
        var n int
        if err:=s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",name).Scan(&n);err!=nil{return err}
        if n!=1{return fmt.Errorf("missing required table %s",name)}
    }
    var integrity string
    if err:=s.db.QueryRow("PRAGMA quick_check").Scan(&integrity);err!=nil{return err}
    if integrity!="ok" {return fmt.Errorf("integrity check failed")}
    return nil
}
func (s *Store) Schema(ctx context.Context)(int,error) {
    var v int;err:=s.db.QueryRowContext(ctx,"SELECT version FROM schema_version").Scan(&v);return v,err
}

func (s *Store) IssueToken(ctx context.Context,role string,ttl time.Duration)(string,Token,error){
    if role!="admin"&&role!="viewer" || ttl<=0 || ttl>365*24*time.Hour {return "",Token{},ErrInvalid}
    b:=make([]byte,32)
    if _,err:=rand.Read(b);err!=nil{return "",Token{},err}
    raw:="bw1_"+hex.EncodeToString(b)
    id,err:=newID();if err!=nil{return "",Token{},err}
    t:=Token{ID:id,Role:role,Expires:time.Now().UTC().Add(ttl)}
    _,err=s.db.ExecContext(ctx,"INSERT INTO credentials (id,token_digest,role,expires_at,created_at) VALUES (?,?,?,?,?)",id,DigestToken(raw),role,t.Expires.Unix(),time.Now().Unix())
    if err!=nil{return "",Token{},err}
    return raw,t,nil
}
func (s *Store) VerifyToken(ctx context.Context,raw string)(Token,error) {
    if !strings.HasPrefix(raw,"bw1_")||len(raw)!=68{return Token{},ErrRevoked}
    var t Token;var expires int64;var revoked int
    err:=s.db.QueryRowContext(ctx,"SELECT id,role,expires_at,revoked FROM credentials WHERE token_digest=?",DigestToken(raw)).Scan(&t.ID,&t.Role,&expires,&revoked)
    if err!=nil{return Token{},ErrRevoked}
    t.Expires=time.Unix(expires,0).UTC();t.Revoked=revoked!=0
    if t.Revoked || !t.Expires.After(time.Now()) {return Token{},ErrRevoked}
    return t,nil
}
func(s *Store) RevokeToken(ctx context.Context,id string) error {
    r,err:=s.db.ExecContext(ctx,"UPDATE credentials SET revoked=1 WHERE id=?",id)
    if err!=nil{return err}
    n,_:=r.RowsAffected();if n!=1{return ErrNotFound}
    return nil
}
func (s *Store) tokenActive(ctx context.Context,id string)(bool,error){
    var n int
    err:=s.db.QueryRowContext(ctx,"SELECT COUNT(*) FROM credentials WHERE id=? AND revoked=0 AND expires_at>?",id,time.Now().Unix()).Scan(&n)
    return n==1,err
}

// Admit persists the original immutable intent hash and sanitized replay payload.
// Concurrent equal requests must return exactly the original IDs and cause no native effect.
func (s *Store) Admit(ctx context.Context,a Admission)(Receipt,error) {
    if a.Layer==""||a.ActorID==""||a.TokenID==""||len(a.Key)<16||len(a.Key)>128||a.Action==""||
        len(a.SafeIntent)==0||len(a.SafeIntent)>65536||a.RequestID=="" {return Receipt{},ErrInvalid}
    // A non-secret digest avoids storing raw client key.
    keyHash:=digest(a.Key);intentHash:=digest(a.Action+"\x00"+string(a.SafeIntent)+"\x00"+a.BoxUUID+"\x00"+a.Incarnation)
    tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return Receipt{},err}
    defer tx.Rollback()
    var found Receipt;var priorHash,priorToken string
    err=tx.QueryRowContext(ctx,"SELECT id,tracking_id,state,intent_digest,token_id,native_operation_id FROM operations WHERE layer=? AND actor_id=? AND key_digest=?",a.Layer,a.ActorID,keyHash).Scan(&found.ID,&found.TrackingID,&found.State,&priorHash,&priorToken,&found.NativeOperationID)
    if err==nil {
        if priorHash!=intentHash||priorToken!=a.TokenID{return Receipt{},ErrConflict}
        found.Duplicate=true
        return found,tx.Commit()
    }
    if !errors.Is(err,sql.ErrNoRows){return Receipt{},err}
    var active int
    err=tx.QueryRowContext(ctx,"SELECT COUNT(*) FROM credentials WHERE id=? AND revoked=0 AND expires_at>?",a.TokenID,time.Now().Unix()).Scan(&active)
    if err!=nil{return Receipt{},err}
    if active!=1{return Receipt{},ErrRevoked}
    id,err:=newID();if err!=nil{return Receipt{},err}
    tracking,err:=newID();if err!=nil{return Receipt{},err}
    now:=time.Now().UnixNano()
    _,err=tx.ExecContext(ctx,"INSERT INTO operations (id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,safe_intent,box_uuid,incarnation,state,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
        id,tracking,a.RequestID,a.Layer,a.ActorID,a.TokenID,keyHash,a.Action,intentHash,a.SafeIntent,a.BoxUUID,a.Incarnation,"ADMITTED",now,now)
    if err!=nil{return Receipt{},err}
    if _,err=tx.ExecContext(ctx,"INSERT INTO operation_events(operation_id,at,state) VALUES (?,?,?)",id,now,"ADMITTED");err!=nil{return Receipt{},err}
    if err=tx.Commit();err!=nil{return Receipt{},err}
    return Receipt{ID:id,TrackingID:tracking,State:"ADMITTED"},nil
}
func (s *Store) Get(ctx context.Context,id string)(Receipt,error){
    var r Receipt
    err:=s.db.QueryRowContext(ctx,"SELECT id,tracking_id,state,native_operation_id FROM operations WHERE id=?",id).Scan(&r.ID,&r.TrackingID,&r.State,&r.NativeOperationID)
    if errors.Is(err,sql.ErrNoRows){return r,ErrNotFound}
    return r,err
}
func (s *Store) transition(ctx context.Context,id,expected,next,nativeID,errClass string) error {
    tx,err:=s.db.BeginTx(ctx,nil);if err!=nil{return err}
    defer tx.Rollback()
    r,err:=tx.ExecContext(ctx,"UPDATE operations SET state=?,native_operation_id=CASE WHEN ?!='' THEN ? ELSE native_operation_id END,error_class=?,updated_at=? WHERE id=? AND state=?",
        next,nativeID,nativeID,errClass,time.Now().UnixNano(),id,expected)
    if err!=nil{return err}
    n,err:=r.RowsAffected();if err!=nil{return err};if n!=1{return ErrState}
    if _,err=tx.ExecContext(ctx,"INSERT INTO operation_events(operation_id,at,state,safe_detail) VALUES (?,?,?,?)",id,time.Now().UnixNano(),next,errClass);err!=nil{return err}
    return tx.Commit()
}
func(s *Store) Claim(ctx context.Context,id string)error{
    active,err:=s.operationActorActive(ctx,id);if err!=nil{return err}
    if !active{return s.transition(ctx,id,"ADMITTED","FAILED","", "AUTH_REVOKED_BEFORE_DISPATCH")}
    // A crash after this durable transition is UNKNOWN even if the native call wasn't sent.
    return s.transition(ctx,id,"ADMITTED","DISPATCHING","","")
}
func(s *Store) operationActorActive(ctx context.Context,id string)(bool,error){
    var token string
    err:=s.db.QueryRowContext(ctx,"SELECT token_id FROM operations WHERE id=?",id).Scan(&token)
    if err!=nil{return false,err}
    return s.tokenActive(ctx,token)
}
func(s *Store) NativeAck(ctx context.Context,id,nativeID string)error {
    if nativeID==""{return ErrInvalid}
    return s.transition(ctx,id,"DISPATCHING","VERIFYING",nativeID,"")
}
func(s *Store) Verified(ctx context.Context,id string)error{return s.transition(ctx,id,"VERIFYING","SUCCEEDED","","")}
func(s *Store) Unknown(ctx context.Context,id,from,reason string)error{
    if len(reason)>120{reason=reason[:120]}
    return s.transition(ctx,id,from,"UNKNOWN","",reason)
}
// Recover never replays a possibly dispatched effect on process restart.
func(s *Store) Recover(ctx context.Context)(int64,error){
    r,err:=s.db.ExecContext(ctx,"UPDATE operations SET state='UNKNOWN',error_class='PROCESS_RESTART_OR_LOST_ACK',updated_at=? WHERE state IN ('DISPATCHING','RUNNING','VERIFYING')",time.Now().UnixNano())
    if err!=nil{return 0,err}
    return r.RowsAffected()
}
