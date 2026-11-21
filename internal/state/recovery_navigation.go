package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

const recoveryLinkLimit = 5
const recoveryLegacyWindow = 128

// A5 is a durable navigation snapshot, not a live diagnosis or a grant of
// authority. Recovery receipt links do not imply the original succeeded.
type RecoveryLink struct {
	OperationID string `json:"operation_id"`
	OriginalID  string `json:"original_operation_id"`
	State       string `json:"state"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
	Stream      string `json:"stream,omitempty"`
	AtNS        int64  `json:"at_ns"`
}
type RecoveryNavigation struct {
	Source              string         `json:"source"`
	AsOfNS              int64          `json:"as_of_ns"`
	OriginalState       string         `json:"operation_state"`
	UpdatedAt           int64          `json:"operation_updated_at_ns"`
	ClaimProtection     string         `json:"claim_protection"`
	Disposition         string         `json:"disposition"`
	NextAction          NextAction     `json:"next_action"`
	Original            *RecoveryLink  `json:"original_link,omitempty"`
	Recent              []RecoveryLink `json:"recent_recoveries"`
	LinksTruncated      bool           `json:"links_truncated"`
	LegacyWindowBounded bool           `json:"legacy_window_bounded"`
	NativeLiveVerified  bool           `json:"native_live_verified"`
	ApprovalCapability  string         `json:"approval_capability"`
	OutputAvailability  string         `json:"output_availability"`
}

const recoveryLinkColumns = "o.id,o.state,o.created_at,CAST(o.safe_intent AS TEXT),o.resource_key,COALESCE((SELECT safe_detail FROM operation_events e WHERE e.operation_id=o.id ORDER BY seq DESC LIMIT 1),'')"

func scanRecoveryLink(row interface{ Scan(...any) error }) (RecoveryLink, bool, error) {
	var v RecoveryLink
	var intent, resource, detail string
	if e := row.Scan(&v.OperationID, &v.State, &v.AtNS, &intent, &resource, &detail); e != nil {
		return v, false, e
	}
	var in struct {
		Original string `json:"original_operation_id"`
		Stream   string `json:"stream,omitempty"`
	}
	if json.Unmarshal([]byte(intent), &in) != nil || !validHexID(in.Original) || in.Original == v.OperationID || (in.Stream != "" && in.Stream != "stdout" && in.Stream != "stderr") {
		return v, false, nil
	}
	canonical, _ := json.Marshal(in)
	if intent != string(canonical) || (resource != "" && resource != "recovery:"+in.Original) {
		return v, false, nil
	}
	d, r, ok := strings.Cut(detail, ":")
	if !ok || !validRecoveryDetail(d, r) || (v.State == "SUCCEEDED") != (d == "SETTLED" || d == "IDENTITY_BOUND") || (v.State != "SUCCEEDED" && v.State != "FAILED") {
		return v, false, nil
	}
	v.OriginalID, v.Stream, v.Disposition, v.Reason = in.Original, in.Stream, d, r
	return v, true, nil
}
func readRecoveryNavigation(ctx context.Context, tx *sql.Tx, layer, id string) (RecoveryNavigation, error) {
	v := RecoveryNavigation{Source: "SQLITE_SNAPSHOT", AsOfNS: time.Now().UnixNano(), Recent: []RecoveryLink{}, ApprovalCapability: "TRUSTED_CLIENT_APPROVAL_UNAVAILABLE", OutputAvailability: "NOT_APPLICABLE"}
	var action string
	var created int64
	var released, dependent, settled bool
	e := tx.QueryRowContext(ctx, `SELECT action,state,created_at,updated_at,
 EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RELEASED'),
 EXISTS(SELECT 1 FROM recovery_audit a WHERE a.operation_id=o.id AND a.decision='RECOVERY_SETTLED'),
 EXISTS(SELECT 1 FROM operation_resource_claims c WHERE c.operation_id=o.id AND c.layer=o.layer AND c.resource_key='delete-project:'||o.layer)
 FROM operations o WHERE o.layer=? AND o.id=?`, layer, id).Scan(&action, &v.OriginalState, &created, &v.UpdatedAt, &released, &settled, &dependent)
	if errors.Is(e, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	if e != nil {
		return v, e
	}
	v.NextAction = nextAction(v.OriginalState, action, id)
	v.ClaimProtection = "NOT_UNKNOWN"
	v.Disposition = "NO_RECOVERY_REQUIRED"
	if v.OriginalState == "SUCCEEDED" && settled {
		v.Disposition = "RECOVERY_SETTLED"
	}
	if v.OriginalState == "UNKNOWN" {
		v.ClaimProtection = "RETAINED"
		v.Disposition = "INSPECT_REQUIRED"
		v.NextAction = NextAction{Kind: "INSPECT_RECOVERY", Tool: "operation_recover", OperationID: id, Mode: "inspect"}
		if released {
			if dependent || action == "box_peer_set" || action == "port_forward_create" || action == "port_forward_delete" {
				v.Disposition = "NETWORK_LEGACY_RELEASE_REVIEW_REQUIRED"
			} else {
				v.ClaimProtection = "REVIEWED_RELEASED"
				v.Disposition = "CLAIM_RELEASED_OUTCOME_UNKNOWN"
				v.NextAction = NextAction{Kind: "READ_EVIDENCE", Tool: "box_activity_get", OperationID: id, View: "evidence"}
			}
		}
	} else if !operationStateTerminal(v.OriginalState) {
		v.Disposition = "IN_PROGRESS"
	}
	if action == "box_exec" {
		v.OutputAvailability = "NOT_CHECKED_USE_OPERATION_OUTPUT_GET"
	}
	if action == "operation_recover" {
		link, ok, e := scanRecoveryLink(tx.QueryRowContext(ctx, "SELECT "+recoveryLinkColumns+" FROM operations o WHERE o.layer=? AND o.id=? AND o.action='operation_recover'", layer, id))
		if e != nil {
			return v, e
		}
		if !ok {
			v.Disposition = "RECOVERY_LINK_UNVERIFIED"
			return v, nil
		}
		var exists int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM operations WHERE layer=? AND id=?", layer, link.OriginalID).Scan(&exists); e != nil {
			return v, e
		}
		if exists != 1 {
			v.Disposition = "RECOVERY_LINK_UNVERIFIED"
			return v, nil
		}
		v.Original = &link
		v.Disposition = link.Disposition
		v.NextAction = NextAction{Kind: "READ_ORIGINAL", Tool: "operation_get", OperationID: link.OriginalID}
		return v, nil
	}
	// Current receipts use an existing indexed diagnostic resource reference.
	// Historical schema-5 receipts have no indexable original reference. Scan
	// at most 128 recent same-layer rows; explicitly report partial coverage.
	rows, e := tx.QueryContext(ctx, "SELECT "+recoveryLinkColumns+" FROM operations o WHERE o.layer=? AND o.resource_key=? AND o.action='operation_recover' ORDER BY o.created_at DESC,o.id DESC LIMIT ?", layer, "recovery:"+id, recoveryLinkLimit+1)
	if e != nil {
		return v, e
	}
	for rows.Next() {
		link, ok, e := scanRecoveryLink(rows)
		if e != nil {
			rows.Close()
			return v, e
		}
		if ok && link.OriginalID == id {
			v.Recent = append(v.Recent, link)
		} else {
			v.LinksTruncated = true
		}
	}
	e = rows.Err()
	ce := rows.Close()
	if e != nil {
		return v, e
	}
	if ce != nil {
		return v, ce
	}
	// Use a materialized bounded candidate window before matching legacy links.
	rows, e = tx.QueryContext(ctx, `WITH recent AS MATERIALIZED (SELECT id,action,resource_key,created_at FROM operations WHERE layer=? AND created_at>=? ORDER BY created_at DESC,id DESC LIMIT ?) SELECT id,action,resource_key FROM recent ORDER BY created_at DESC,id DESC`, layer, created, recoveryLegacyWindow+1)
	if e != nil {
		return v, e
	}
	var legacy []string
	count := 0
	for rows.Next() {
		var rid, a, key string
		if e = rows.Scan(&rid, &a, &key); e != nil {
			rows.Close()
			return v, e
		}
		count++
		if count > recoveryLegacyWindow {
			v.LegacyWindowBounded = true
			break
		}
		if a == "operation_recover" && key == "" {
			legacy = append(legacy, rid)
		}
	}
	e = rows.Err()
	ce = rows.Close()
	if e != nil {
		return v, e
	}
	if ce != nil {
		return v, ce
	}
	// Batch legacy candidates, avoiding one SQL roundtrip per candidate.
	if len(legacy) > 0 {
		args := []any{layer}
		for _, rid := range legacy {
			args = append(args, rid)
		}
		rows, e = tx.QueryContext(ctx, "SELECT "+recoveryLinkColumns+" FROM operations o WHERE o.layer=? AND o.id IN ("+strings.TrimSuffix(strings.Repeat("?,", len(legacy)), ",")+")", args...)
		if e != nil {
			return v, e
		}
		for rows.Next() {
			link, ok, e := scanRecoveryLink(rows)
			if e != nil {
				rows.Close()
				return v, e
			}
			if ok && link.OriginalID == id {
				v.Recent = append(v.Recent, link)
			}
		}
		e = rows.Err()
		ce = rows.Close()
		if e != nil {
			return v, e
		}
		if ce != nil {
			return v, ce
		}
	}
	sort.Slice(v.Recent, func(i, j int) bool {
		if v.Recent[i].AtNS == v.Recent[j].AtNS {
			return v.Recent[i].OperationID > v.Recent[j].OperationID
		}
		return v.Recent[i].AtNS > v.Recent[j].AtNS
	})
	if len(v.Recent) > recoveryLinkLimit {
		v.Recent = v.Recent[:recoveryLinkLimit]
		v.LinksTruncated = true
	}
	if v.OriginalState == "SUCCEEDED" {
		for _, link := range v.Recent {
			if link.Disposition == "SETTLED" && link.Stream == "" {
				v.Disposition = "RECOVERY_SETTLED"
				break
			}
		}
	}
	return v, nil
}
func operationStateTerminal(s string) bool {
	switch s {
	case "SUCCEEDED", "FAILED", "UNKNOWN", "CANCELLED", "TIMED_OUT":
		return true
	}
	return false
}

func (s *Store) ReadRecoveryNavigation(ctx context.Context, layer, id string) (RecoveryNavigation, error) {
	if layer == "" || !validHexID(id) {
		return RecoveryNavigation{}, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return RecoveryNavigation{}, e
	}
	defer tx.Rollback()
	v, e := readRecoveryNavigation(ctx, tx, layer, id)
	if e != nil {
		return v, e
	}
	return v, tx.Commit()
}
