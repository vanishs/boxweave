package state

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

// ReplayPeerSet checks an existing exact Actor/key original BEFORE current
// native/pin lookups. A deleted Box, revoked lease, or disconnected PRS cannot
// induce a second network mutation for the same key. All authorization still
// flows through Replay's live credential and signed-intent digest checks.
func (s *Store) ReplayPeerSet(ctx context.Context, layer, actor, token, key,
	source, target string, allow bool, work, agent string) (Receipt, bool, error) {
	if s == nil || s.db == nil || layer == "" || actor == "" || token == "" ||
		len(key) < 16 || len(key) > 128 || source == "" || target == "" || source == target {
		return Receipt{}, false, ErrInvalid
	}
	var action string
	var raw []byte
	err := s.db.QueryRowContext(ctx,
		"SELECT action,safe_intent FROM operations WHERE layer=? AND actor_id=? AND key_digest=?",
		layer, actor, digest(key)).Scan(&action, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, false, nil
	}
	if err != nil {
		return Receipt{}, false, err
	}
	var i RootPeerRevokeSafeIntent
	if action != "box_peer_set" || json.Unmarshal(raw, &i) != nil ||
		i.Project != layer || !auditName(i.Bridge) || !auditName(i.ACL) ||
		i.SourceUUID != source || i.TargetUUID != target || i.Allow != allow ||
		i.SourceIncarnation == "" || i.TargetIncarnation == "" {
		return Receipt{}, false, ErrConflict
	}
	canonical, err := json.Marshal(i)
	if err != nil || !bytes.Equal(canonical, raw) {
		return Receipt{}, false, ErrConflict
	}
	r, err := s.Replay(ctx, Admission{
		Layer: layer, ActorID: actor, TokenID: token, Key: key, Action: action,
		BoxUUID: i.SourceUUID, Incarnation: i.SourceIncarnation,
		ResourceKey:  "acl:default/" + i.ACL,
		ResourceKeys: []string{"box:" + i.SourceUUID, "box:" + i.TargetUUID},
		SafeIntent:   raw, WorkRef: work, AgentRef: agent,
	})
	return r, err == nil, err
}
