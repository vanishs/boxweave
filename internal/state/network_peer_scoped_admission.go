package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
)

func scopedPeerAdmission(i ScopedPeerOriginalIntent, a Admission) (Admission, error) {
	keys, e := scopedPeerResourceClaims(i)
	if e != nil {
		return Admission{}, e
	}
	raw, e := json.Marshal(i)
	if e != nil {
		return Admission{}, e
	}
	a.Action = "box_peer_set"
	a.BoxUUID = i.SourceUUID
	a.Incarnation = i.SourceIncarnation
	a.ResourceKey = "peer-guard:" + i.Bridge
	for _, k := range keys {
		if k != a.ResourceKey {
			a.ResourceKeys = append(a.ResourceKeys, k)
		}
	}
	a.SafeIntent = raw
	return a, nil
}

// Replay is checked before current ownership, without re-admitting an effect.
func (s *Store) ReplayScopedPeerSet(ctx context.Context, layer, actor, token, key, src, dst string, allow bool, work, agent string) (Receipt, bool, error) {
	if s == nil || s.db == nil || len(key) < 16 || len(key) > 128 || src == "" || dst == "" || src == dst {
		return Receipt{}, false, ErrInvalid
	}
	var action string
	var raw []byte
	e := s.db.QueryRowContext(ctx, "SELECT action,safe_intent FROM operations WHERE layer=? AND actor_id=? AND key_digest=?", layer, actor, digest(key)).Scan(&action, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return Receipt{}, false, nil
	}
	if e != nil {
		return Receipt{}, false, e
	}
	var i ScopedPeerOriginalIntent
	if action != "box_peer_set" || json.Unmarshal(raw, &i) != nil || i.Schema != 2 || i.Project != layer || i.SourceUUID != src || i.TargetUUID != dst || i.Allow != allow {
		return Receipt{}, false, ErrConflict
	}
	a, e := scopedPeerAdmission(i, Admission{Layer: layer, ActorID: actor, TokenID: token, Key: key, WorkRef: work, AgentRef: agent})
	if e != nil || !bytes.Equal(a.SafeIntent, raw) {
		return Receipt{}, false, ErrConflict
	}
	r, e := s.Replay(ctx, a)
	return r, e == nil, e
}

func (s *Store) AdmitScopedPeerSet(ctx context.Context, a Admission, src, dst string, allow bool) (Receipt, error) {
	if s == nil || s.db == nil || s.auditReadOnly || !strings.HasPrefix(a.Layer, "user-") || src == dst {
		return Receipt{}, ErrInvalid
	}
	bridge := "incusbr-" + strings.TrimPrefix(a.Layer, "user-")
	ledger, e := s.ReadCompleteScopedNetworkAuditLedger(ctx, a.Layer, bridge)
	if e != nil {
		return Receipt{}, ErrResourceLocked
	}
	i := ScopedPeerOriginalIntent{Schema: 2, Project: a.Layer, Bridge: bridge, SourceUUID: src, TargetUUID: dst, Allow: allow}
	for _, b := range ledger.Boxes {
		if b.UUID == src {
			i.SourceIncarnation = b.Incarnation
		}
		if b.UUID == dst {
			i.TargetIncarnation = b.Incarnation
		}
	}
	if i.SourceIncarnation == "" || i.TargetIncarnation == "" {
		return Receipt{}, ErrResourceLocked
	}
	a, e = scopedPeerAdmission(i, a)
	if e != nil {
		return Receipt{}, e
	}
	return s.Admit(ctx, a)
}
