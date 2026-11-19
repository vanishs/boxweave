package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestS4B3PeerIdempotencyRestoresOriginalWithoutNativeLease(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Open(filepath.Join(dir, "peer-replay.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	_, token, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	intent := RootPeerRevokeSafeIntent{
		Project: "user-1001", Bridge: "bwbr0", ACL: "bw-s4b2",
		SourceUUID: "source", SourceIncarnation: "source",
		TargetUUID: "target", TargetIncarnation: "target", Allow: true,
	}
	safe, err := json.Marshal(intent)
	if err != nil {
		t.Fatal(err)
	}
	key := "s4b3-unique-peer-original-2026"
	original, err := s.Admit(ctx, Admission{
		Layer: intent.Project, ActorID: token.ID, TokenID: token.ID, Key: key,
		Action: "box_peer_set", RequestID: "test-actor-original",
		BoxUUID: "source", Incarnation: "source",
		ResourceKey:  "acl:default/bw-s4b2",
		ResourceKeys: []string{"box:source", "box:target"}, SafeIntent: safe,
	})
	if err != nil {
		t.Fatal(err)
	}
	got, found, err := s.ReplayPeerSet(ctx, intent.Project, token.ID, token.ID, key, "source", "target", true, "", "")
	if err != nil || !found || !got.Duplicate || got.ID != original.ID {
		t.Fatalf("lost pinned native lease caused new original: %+v found=%v err=%v", got, found, err)
	}
	if _, found, err := s.ReplayPeerSet(ctx, intent.Project, token.ID, token.ID, key, "source", "target", false, "", ""); found || !errors.Is(err, ErrConflict) {
		t.Fatalf("same key opposite authorization was not rejected: found=%v err=%v", found, err)
	}
	if _, found, err := s.ReplayPeerSet(ctx, intent.Project, token.ID, token.ID, key, "source", "other", true, "", ""); found || !errors.Is(err, ErrConflict) {
		t.Fatalf("same key different peer was not rejected: found=%v err=%v", found, err)
	}
}
