// Synthetic A4 compensation model. No production imports, no native calls,
// no recovery of real resource claims. A3 signature admits a NEW operation;
// post-effect verification is independent of the approval.
package signedlab

import (
	"context"
	"database/sql"
	"encoding/json"
)

const (
	a4Forward = "A4_WITHDRAW_EXACT_FORWARD"
	a4Peer    = "A4_CONTAIN_PEER_DENY"
	a4Child   = "A4_CONTAIN_DELETE_CHILD"
)

type a4Plan struct {
	Kind, Original, UUID, Incarnation, Resource, Endpoint, Peer, Parent                  string
	Owned, ClaimPinned, NoConflict, Contained, FreshRootPlan, ParentBarrier, ChildLinked bool
}

func (p a4Plan) valid(b Binding) bool {
	if p.Kind != b.Action || p.Original != b.Original || p.UUID != b.UUID || p.Incarnation != b.Incarnation || p.Resource == "" || !p.Owned || !p.ClaimPinned || !p.NoConflict || !p.Contained {
		return false
	}
	switch p.Kind {
	case a4Forward:
		return p.Endpoint != "" && p.Peer == "" && p.Parent == ""
	case a4Peer:
		return p.Peer != "" && p.Peer != p.UUID && p.FreshRootPlan && p.Parent == ""
	case a4Child:
		return p.Parent != "" && p.Parent != p.Original && p.ChildLinked && p.ParentBarrier
	}
	return false
}
func readA4Plan(ctx context.Context, tx *sql.Tx, b Binding) (a4Plan, error) {
	var raw []byte
	var p a4Plan
	e := tx.QueryRowContext(ctx, "SELECT snapshot FROM lab_a4_plan WHERE layer=? AND original=?", b.Layer, b.Original).Scan(&raw)
	if e != nil || json.Unmarshal(raw, &p) != nil || hexDigest(encode(p)) != b.Evidence || b.Intent != hexDigest(encode(struct{ Action, Resource string }{p.Kind, p.Resource})) || !p.valid(b) {
		return p, ErrDenied
	}
	return p, nil
}
func admitA4(ctx context.Context, tx *sql.Tx, c Challenge) (string, error) {
	b := c.Binding
	if _, e := readA4Plan(ctx, tx, b); e != nil {
		return "", e
	}
	id, e := randomID()
	if e != nil {
		return "", e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO lab_a4_comp(id,approval,layer,original,action,binding,state) VALUES(?,?,?,?,?,?,'ADMITTED')", id, c.ID, b.Layer, b.Original, b.Action, encode(b)); e != nil {
		return "", ErrDenied
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO lab_a4_events VALUES(?,'ADMITTED')", id)
	return id, e
}

type compRecord struct {
	id, approval, state string
	b                   Binding
	dispatches          int
}

func readComp(ctx context.Context, tx *sql.Tx, id string) (compRecord, error) {
	var c compRecord
	var raw []byte
	c.id = id
	e := tx.QueryRowContext(ctx, "SELECT approval,state,binding,dispatches FROM lab_a4_comp WHERE id=?", id).Scan(&c.approval, &c.state, &raw, &c.dispatches)
	if e != nil || json.Unmarshal(raw, &c.b) != nil {
		return c, ErrDenied
	}
	return c, nil
}

// Recheck fresh simulated safe facts at dispatch, not just the signature.
// The original stays UNKNOWN. No repeat dispatch even when ACK is lost.
func (l *Lab) dispatchA4(ctx context.Context, id string) error {
	tx, e := l.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	c, e := readComp(ctx, tx, id)
	if e != nil || c.state != "ADMITTED" || c.dispatches != 0 {
		return ErrDenied
	}
	b := c.b
	var epoch, revision int64
	var active, released int
	var originalState, intent, action, uuid, inc, evidence string
	e = tx.QueryRowContext(ctx, "SELECT epoch,active FROM lab_actors WHERE layer=? AND actor=?", b.Layer, b.Actor).Scan(&epoch, &active)
	if e != nil || active != 1 || epoch != b.Epoch {
		return ErrDenied
	}
	e = tx.QueryRowContext(ctx, "SELECT revision,state,released,intent,action,uuid,incarnation,evidence FROM lab_targets WHERE layer=? AND original=?", b.Layer, b.Original).Scan(&revision, &originalState, &released, &intent, &action, &uuid, &inc, &evidence)
	if e != nil || revision != b.Revision || originalState != "UNKNOWN" || released != 0 || intent != b.Intent || action != b.Action || uuid != b.UUID || inc != b.Incarnation || evidence != b.Evidence {
		return ErrDenied
	}
	if _, e = readA4Plan(ctx, tx, b); e != nil {
		return e
	}
	var raw []byte
	var status string
	var challenge Challenge
	e = tx.QueryRowContext(ctx, "SELECT challenge,status FROM lab_approvals WHERE id=?", c.approval).Scan(&raw, &status)
	if e != nil || status != "CONSUMED" || json.Unmarshal(raw, &challenge) != nil || challenge.Binding != b || l.now().Unix() >= challenge.Expires || l.now().Unix() < challenge.Issued {
		return ErrDenied
	}
	// UNKNOWN is deliberate: fixture dispatch could have escaped even if
	// the response vanished. It can only be reconciled, never repeated.
	r, e := tx.ExecContext(ctx, "UPDATE lab_a4_comp SET state='UNKNOWN',dispatches=1 WHERE id=? AND state='ADMITTED' AND dispatches=0", id)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil || n != 1 {
		return ErrDenied
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO lab_a4_events VALUES(?,'DISPATCH_UNKNOWN')", id); e != nil {
		return e
	}
	return tx.Commit()
}

type a4Witness struct {
	CompID, Original, UUID, Incarnation, Kind, Resource, Endpoint, Peer, Parent                                                                         string
	ExactIdentity, CompensationReceiptVerified, PostVerified, NoUnrelatedEffect, BarrierIntact, DeviceAbsent, ListenerAbsent, RootKernelDeny, ChildSafe bool
}

func (l *Lab) settleA4(ctx context.Context, id string) error {
	tx, e := l.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	c, e := readComp(ctx, tx, id)
	if e != nil {
		return e
	}
	if c.state == "SUCCEEDED" {
		return tx.Commit()
	} // no duplicate effect
	if c.state != "UNKNOWN" || c.dispatches != 1 {
		return ErrDenied
	}
	b := c.b
	p, e := readA4Plan(ctx, tx, b)
	if e != nil {
		return e
	}
	var raw []byte
	var w a4Witness
	e = tx.QueryRowContext(ctx, "SELECT observation FROM lab_a4_witness WHERE comp=?", id).Scan(&raw)
	if e != nil || json.Unmarshal(raw, &w) != nil || w.CompID != id || w.Original != b.Original || w.UUID != b.UUID || w.Incarnation != b.Incarnation || w.Kind != b.Action || w.Resource != p.Resource || w.Endpoint != p.Endpoint || w.Peer != p.Peer || w.Parent != p.Parent || !w.ExactIdentity || !w.CompensationReceiptVerified || !w.PostVerified || !w.NoUnrelatedEffect || !w.BarrierIntact {
		return ErrDenied
	}
	switch b.Action {
	case a4Forward:
		if !w.DeviceAbsent || !w.ListenerAbsent {
			return ErrDenied
		}
	case a4Peer:
		if !w.RootKernelDeny {
			return ErrDenied
		}
	case a4Child:
		if !w.ChildSafe {
			return ErrDenied
		}
	default:
		return ErrDenied
	}
	var state string
	var released int
	var rev int64
	var uuid, inc, evidence, action, intent string
	e = tx.QueryRowContext(ctx, "SELECT state,released,revision,uuid,incarnation,evidence,action,intent FROM lab_targets WHERE layer=? AND original=?", b.Layer, b.Original).Scan(&state, &released, &rev, &uuid, &inc, &evidence, &action, &intent)
	if e != nil || state != "UNKNOWN" || released != 0 || rev != b.Revision || uuid != b.UUID || inc != b.Incarnation || evidence != b.Evidence || action != b.Action || intent != b.Intent {
		return ErrDenied
	}
	// Only the modeled exact Forward reservation can retire. Peer Project
	// and dependent Delete barriers remain until separate domain reconciliation.
	if b.Action == a4Forward {
		r, e := tx.ExecContext(ctx, "UPDATE lab_targets SET released=1 WHERE layer=? AND original=? AND released=0 AND state='UNKNOWN'", b.Layer, b.Original)
		if e != nil {
			return e
		}
		n, e := r.RowsAffected()
		if e != nil || n != 1 {
			return ErrDenied
		}
	}
	if _, e = tx.ExecContext(ctx, "UPDATE lab_a4_comp SET state='SUCCEEDED' WHERE id=? AND state='UNKNOWN'", id); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO lab_a4_events VALUES(?,'VERIFIED')", id); e != nil {
		return e
	}
	return tx.Commit()
}
