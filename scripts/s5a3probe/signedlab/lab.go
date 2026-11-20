// Package signedlab is a simulation of a possible trusted-client signature
// contract. It is not a ChatGPT protocol, a production verifier, or a native
// recovery path. It uses a separate disposable SQLite database.
package signedlab

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var ErrDenied = errors.New("lab approval refused")

const domain = "boxweave/s5a3/synthetic-approval/v1\x00"
const ttl = 5 * time.Minute

type Binding struct {
	Layer       string `json:"layer"`
	Actor       string `json:"actor"`
	Original    string `json:"original"`
	Action      string `json:"action"`
	Intent      string `json:"intent_digest"`
	UUID        string `json:"uuid"`
	Incarnation string `json:"incarnation"`
	Evidence    string `json:"evidence_digest"`
	RecoveryKey string `json:"recovery_key_digest"`
	Epoch       int64  `json:"actor_epoch"`
	Revision    int64  `json:"original_revision"`
}
type Challenge struct {
	Issuer   string  `json:"issuer"`
	Audience string  `json:"audience"`
	KeyID    string  `json:"key_id"`
	ID       string  `json:"approval_id"`
	Binding  Binding `json:"binding"`
	Issued   int64   `json:"issued"`
	Expires  int64   `json:"expires"`
}
type Assertion struct {
	Challenge Challenge `json:"challenge"`
	Decision  string    `json:"decision"`
	Signature string    `json:"signature"`
}
type Receipt struct {
	ID             string `json:"receipt_id"`
	ApprovalID     string `json:"approval_id"`
	Original       string `json:"original"`
	Decision       string `json:"decision"`
	Duplicate      bool   `json:"duplicate"`
	OriginalState  string `json:"original_state"`
	ClaimReleased  bool   `json:"synthetic_claim_released"`
	NativeEffect   bool   `json:"native_effect"`
	CompensationID string `json:"synthetic_compensation_id,omitempty"`
}
type verifier struct {
	issuer, audience, keyID string
	key                     ed25519.PublicKey
}
type Lab struct {
	db    *sql.DB
	trust verifier
	now   func() time.Time
}

func hexDigest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func encode(v any) []byte {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return b
}
func message(c Challenge, d string) []byte {
	return append([]byte(domain), encode(struct {
		Challenge Challenge `json:"challenge"`
		Decision  string    `json:"decision"`
	}{c, d})...)
}
func randomID() (string, error) {
	b := make([]byte, 16)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func (v verifier) verify(a Assertion) bool {
	c := a.Challenge
	if c.Issuer != v.issuer || c.Audience != v.audience || c.KeyID != v.keyID || len(v.key) != ed25519.PublicKeySize || (a.Decision != "approve" && a.Decision != "deny") {
		return false
	}
	if len(a.Signature) != base64.RawURLEncoding.EncodedLen(ed25519.SignatureSize) {
		return false
	}
	sig, e := base64.RawURLEncoding.Strict().DecodeString(a.Signature)
	return e == nil && base64.RawURLEncoding.EncodeToString(sig) == a.Signature && ed25519.Verify(v.key, message(c, a.Decision), sig)
}
func openLab(path string, v verifier, now func() time.Time) (*Lab, error) {
	if !filepath.IsAbs(path) || filepath.Base(path) != "s5a3-synthetic.sqlite" || v.issuer == "" || v.audience == "" || v.keyID == "" || len(v.key) != ed25519.PublicKeySize || now == nil {
		return nil, ErrDenied
	}
	// Only called from tests with t.TempDir(); never use the product ledger.
	u := url.URL{Scheme: "file", Path: path}
	db, e := sql.Open("sqlite3", u.String()+"?_busy_timeout=5000&_journal_mode=WAL&_foreign_keys=on&_txlock=immediate")
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`CREATE TABLE IF NOT EXISTS lab_actors(layer TEXT NOT NULL,actor TEXT NOT NULL,epoch INTEGER NOT NULL,active INTEGER NOT NULL,PRIMARY KEY(layer,actor));
 CREATE TABLE IF NOT EXISTS lab_targets(layer TEXT NOT NULL,original TEXT NOT NULL,action TEXT NOT NULL,intent TEXT NOT NULL,uuid TEXT NOT NULL,incarnation TEXT NOT NULL,evidence TEXT NOT NULL,revision INTEGER NOT NULL,state TEXT NOT NULL,released INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(layer,original));
 CREATE TABLE IF NOT EXISTS lab_approvals(id TEXT PRIMARY KEY, layer TEXT NOT NULL,actor TEXT NOT NULL,recovery_key TEXT NOT NULL,challenge BLOB NOT NULL,status TEXT NOT NULL, UNIQUE(layer,actor,recovery_key));
 CREATE TABLE IF NOT EXISTS lab_receipts(id TEXT PRIMARY KEY, approval TEXT NOT NULL UNIQUE REFERENCES lab_approvals(id), assertion_digest TEXT NOT NULL, result BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS lab_audit(approval TEXT NOT NULL REFERENCES lab_approvals(id), event TEXT NOT NULL, UNIQUE(approval,event));
 CREATE TABLE IF NOT EXISTS lab_a4_plan(layer TEXT NOT NULL,original TEXT NOT NULL,snapshot BLOB NOT NULL,PRIMARY KEY(layer,original));
 CREATE TABLE IF NOT EXISTS lab_a4_comp(id TEXT PRIMARY KEY,approval TEXT NOT NULL UNIQUE REFERENCES lab_approvals(id),layer TEXT NOT NULL,original TEXT NOT NULL UNIQUE,action TEXT NOT NULL,binding BLOB NOT NULL,state TEXT NOT NULL,dispatches INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE IF NOT EXISTS lab_a4_witness(comp TEXT PRIMARY KEY REFERENCES lab_a4_comp(id),observation BLOB NOT NULL);
 CREATE TABLE IF NOT EXISTS lab_a4_events(comp TEXT NOT NULL REFERENCES lab_a4_comp(id),event TEXT NOT NULL,UNIQUE(comp,event));`)
	if e != nil {
		db.Close()
		return nil, e
	}
	v.key = append(ed25519.PublicKey(nil), v.key...)
	return &Lab{db: db, trust: v, now: now}, nil
}
func readBinding(ctx context.Context, tx *sql.Tx, layer, actor, original, key string) (Binding, error) {
	b := Binding{Layer: layer, Actor: actor, Original: original, RecoveryKey: hexDigest([]byte(key))}
	var active, released int
	var state string
	e := tx.QueryRowContext(ctx, "SELECT epoch,active FROM lab_actors WHERE layer=? AND actor=?", layer, actor).Scan(&b.Epoch, &active)
	if e != nil || active != 1 {
		return b, ErrDenied
	}
	e = tx.QueryRowContext(ctx, "SELECT action,intent,uuid,incarnation,evidence,revision,state,released FROM lab_targets WHERE layer=? AND original=?", layer, original).Scan(&b.Action, &b.Intent, &b.UUID, &b.Incarnation, &b.Evidence, &b.Revision, &state, &released)
	if e != nil || state != "UNKNOWN" || released != 0 || b.UUID == "" || b.Incarnation == "" {
		return b, ErrDenied
	}
	if b.Action != "REVIEWED_RELEASE" {
		if _, e = readA4Plan(ctx, tx, b); e != nil {
			return b, e
		}
	} else {
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM lab_a4_plan WHERE layer=? AND original=?", layer, original).Scan(&n); e != nil || n != 0 {
			return b, ErrDenied
		}
	}
	return b, nil
}
func (l *Lab) issue(ctx context.Context, layer, actor, original, key string) (Challenge, error) {
	var c Challenge
	if len(key) < 16 || len(key) > 128 {
		return c, ErrDenied
	}
	tx, e := l.db.BeginTx(ctx, nil)
	if e != nil {
		return c, e
	}
	defer tx.Rollback()
	b, e := readBinding(ctx, tx, layer, actor, original, key)
	if e != nil {
		return c, e
	}
	var old []byte
	var status string
	e = tx.QueryRowContext(ctx, "SELECT challenge,status FROM lab_approvals WHERE layer=? AND actor=? AND recovery_key=?", layer, actor, b.RecoveryKey).Scan(&old, &status)
	if e == nil {
		if json.Unmarshal(old, &c) != nil || c.Binding != b || status != "PENDING" || l.now().Unix() >= c.Expires {
			return Challenge{}, ErrDenied
		}
		return c, tx.Commit()
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return c, e
	}
	id, e := randomID()
	if e != nil {
		return c, e
	}
	now := l.now()
	c = Challenge{Issuer: l.trust.issuer, Audience: l.trust.audience, KeyID: l.trust.keyID, ID: id, Binding: b, Issued: now.Unix(), Expires: now.Add(ttl).Unix()}
	_, e = tx.ExecContext(ctx, "INSERT INTO lab_approvals(id,layer,actor,recovery_key,challenge,status) VALUES(?,?,?,?,?,'PENDING')", id, layer, actor, b.RecoveryKey, encode(c))
	if e != nil {
		return Challenge{}, e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO lab_audit(approval,event) VALUES(?,'ISSUED')", id)
	if e != nil {
		return Challenge{}, e
	}
	return c, tx.Commit()
}

// Identity arguments are server-authenticated in the simulated transport,
// never taken from the signature or model claims. The original/key bind the
// exact desired request. Only a pinned test-client key can sign a decision.
func (l *Lab) consume(ctx context.Context, layer, actor, original, key string, a Assertion) (Receipt, error) {
	var r Receipt
	c := a.Challenge
	b := c.Binding
	if len(key) < 16 || len(key) > 128 || b.Layer != layer || b.Actor != actor || b.Original != original || b.RecoveryKey != hexDigest([]byte(key)) || !l.trust.verify(a) {
		return r, ErrDenied
	}
	tx, e := l.db.BeginTx(ctx, nil)
	if e != nil {
		return r, e
	}
	defer tx.Rollback()
	var active int
	var epoch int64
	if e = tx.QueryRowContext(ctx, "SELECT active,epoch FROM lab_actors WHERE layer=? AND actor=?", layer, actor).Scan(&active, &epoch); e != nil || active != 1 || epoch != b.Epoch {
		return r, ErrDenied
	}
	var payload []byte
	var status string
	e = tx.QueryRowContext(ctx, "SELECT challenge,status FROM lab_approvals WHERE id=? AND layer=? AND actor=? AND recovery_key=?", c.ID, layer, actor, b.RecoveryKey).Scan(&payload, &status)
	if e != nil || subtle.ConstantTimeCompare(payload, encode(c)) != 1 {
		return r, ErrDenied
	}
	assertionHash := hexDigest(encode(a))
	if status == "CONSUMED" {
		var saved []byte
		var hash string
		if e = tx.QueryRowContext(ctx, "SELECT assertion_digest,result FROM lab_receipts WHERE approval=?", c.ID).Scan(&hash, &saved); e != nil || hash != assertionHash || json.Unmarshal(saved, &r) != nil {
			return Receipt{}, ErrDenied
		}
		r.Duplicate = true
		return r, tx.Commit() // readback only, even after expiry; never another effect
	}
	now := l.now().Unix()
	if status != "PENDING" || now < c.Issued || now >= c.Expires || c.Expires-c.Issued != int64(ttl/time.Second) {
		return r, ErrDenied
	}
	current, e := readBinding(ctx, tx, layer, actor, original, key)
	if e != nil || current != b {
		return r, ErrDenied
	}
	rid, e := randomID()
	if e != nil {
		return r, e
	}
	r = Receipt{ID: rid, ApprovalID: c.ID, Original: original, Decision: a.Decision, OriginalState: "UNKNOWN", ClaimReleased: a.Decision == "approve" && b.Action == "REVIEWED_RELEASE"}
	if a.Decision == "approve" && b.Action != "REVIEWED_RELEASE" {
		r.CompensationID, e = admitA4(ctx, tx, c)
		if e != nil {
			return Receipt{}, e
		}
	}
	if r.ClaimReleased {
		res, e := tx.ExecContext(ctx, "UPDATE lab_targets SET released=1 WHERE layer=? AND original=? AND revision=? AND state='UNKNOWN' AND released=0", layer, original, b.Revision)
		if e != nil {
			return Receipt{}, e
		}
		n, e := res.RowsAffected()
		if e != nil || n != 1 {
			return Receipt{}, ErrDenied
		}
	}
	res, e := tx.ExecContext(ctx, "UPDATE lab_approvals SET status='CONSUMED' WHERE id=? AND status='PENDING'", c.ID)
	if e != nil {
		return Receipt{}, e
	}
	n, e := res.RowsAffected()
	if e != nil || n != 1 {
		return Receipt{}, ErrDenied
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO lab_receipts(id,approval,assertion_digest,result) VALUES(?,?,?,?)", rid, c.ID, assertionHash, encode(r))
	if e != nil {
		return Receipt{}, e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO lab_audit(approval,event) VALUES(?,?)", c.ID, fmt.Sprintf("CONSUMED_%s", a.Decision))
	if e != nil {
		return Receipt{}, e
	}
	if e = tx.Commit(); e != nil {
		return Receipt{}, e
	}
	return r, nil
}
