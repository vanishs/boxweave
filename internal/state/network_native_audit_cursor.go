package state

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/vanishs/boxweave/internal/networklimits"
	"time"
)

type NativeNetworkAuditInfo struct {
	Owned             int  `json:"owned"`
	Native            int  `json:"native"`
	Unowned           int  `json:"unowned_native"`
	OrphanBindings    int  `json:"orphan_bindings"`
	OrphanFences      int  `json:"orphan_fences"`
	Unsettled         int  `json:"unsettled_operations"`
	InventoryScanned  bool `json:"entire_project_inventory_scanned"`
	CrossSourceAtomic bool `json:"cross_source_atomic"`
}
type nativeAuditCursor struct {
	Layer, Requester, BoxID, Digest string
	Offset                          int
	MAC                             string
}

// PageNativeNetworkIssues issues a signed, actor+project-scoped cursor.
// Every continuation redoes the full native read and refuses a changed
// diagnostic digest, not a silently truncated or mixed-page "all clear".
func (s *Store) PageNativeNetworkIssues(ctx context.Context, layer, requester, boxID string, limit int, cursor, digest string, issues []ActivityIssue, info NativeNetworkAuditInfo) (ActivityResult, error) {
	var zero ActivityResult
	if !auditName(layer) || !validHexID(requester) || (boxID != "" && !validActivityBoxID(boxID)) || limit < 0 || limit > 50 || len(cursor) > 2048 || len(digest) != 64 || len(issues) > 8*networklimits.MaxProjectBoxes+8 {
		return zero, ErrInvalid
	}
	if limit == 0 {
		limit = 20
	}
	if e := s.RequireActiveTokenID(ctx, requester); e != nil {
		return zero, e
	}
	var secret []byte
	if e := s.db.QueryRowContext(ctx, "SELECT secret FROM cursor_secrets WHERE id=1").Scan(&secret); e != nil {
		return zero, e
	}
	sign := func(v nativeAuditCursor) string {
		v.MAC = ""
		raw, _ := json.Marshal(v)
		h := hmac.New(sha256.New, secret)
		h.Write(raw)
		return hex.EncodeToString(h.Sum(nil))
	}
	v := nativeAuditCursor{Layer: layer, Requester: requester, BoxID: boxID, Digest: digest}
	if cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(cursor)
		if e != nil {
			return zero, ErrInvalid
		}
		if e = json.Unmarshal(raw, &v); e != nil {
			return zero, ErrInvalid
		}
		actual, ae := hex.DecodeString(v.MAC)
		expected, ee := hex.DecodeString(sign(v))
		if ae != nil || ee != nil || !hmac.Equal(actual, expected) || v.Layer != layer || v.Requester != requester || v.BoxID != boxID || v.Digest != digest || v.Offset <= 0 || v.Offset >= len(issues) {
			return zero, ErrInvalid
		}
	}
	if v.Offset > len(issues) {
		return zero, ErrInvalid
	}
	end := v.Offset + limit
	if end > len(issues) {
		end = len(issues)
	}
	out := ActivityResult{Source: "SQLITE_AND_NATIVE_READONLY", Scope: layer, AsOfNS: time.Now().UnixNano(), Coverage: "PROJECT_NATIVE_SNAPSHOT_NONATOMIC", Issues: append([]ActivityIssue{}, issues[v.Offset:end]...), Truncated: end < len(issues), NativeNetwork: &info}
	out.AttentionComplete = false
	out.CoverageDetails = &ActivityCoverageDetails{AuthorizedProjectOnly: true, SQLiteSnapshotOnly: false, NativeLiveVerified: info.InventoryScanned, CrossPageSnapshotStable: false, AttentionComplete: false, PageTruncated: out.Truncated}
	if !info.InventoryScanned {
		out.Source = "SQLITE_ONLY_NATIVE_UNAVAILABLE"
		out.Coverage = "NATIVE_PROJECT_INVENTORY_UNVERIFIED"
	}
	if out.Truncated {
		next := nativeAuditCursor{Layer: layer, Requester: requester, BoxID: boxID, Digest: digest, Offset: end}
		next.MAC = sign(next)
		raw, e := json.Marshal(next)
		if e != nil {
			return zero, e
		}
		out.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	if e := s.RequireActiveTokenID(ctx, requester); e != nil {
		return zero, e
	}
	return out, nil
}
