package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestS4B2RootAllSiblingProjectAdmissionBarrier(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	adm := func(key, project, action, uuid string) Admission {
		return Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID,
			Key: key, RequestID: key, Action: action, SafeIntent: []byte("{}"),
			ResourceKey: "box:member-name", BoxUUID: uuid, Incarnation: uuid}
	}
	old, err := s.Admit(ctx, adm("old-active-sibling", "user-1001", "box_exec", "sibling-c"))
	if err != nil {
		t.Fatal(err)
	}
	peer := adm("root-project-barrier", "user-1001", "box_peer_set", "source-a")
	peer.ResourceKey = "acl:default/bw-s4b2"
	if _, err := s.Admit(ctx, peer); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("root overlapped active sibling operation: %v", err)
	}
	if _, err := s.db.ExecContext(ctx, "UPDATE operations SET state='SUCCEEDED' WHERE id=?", old.ID); err != nil {
		t.Fatal(err)
	}
	root, err := s.Admit(ctx, peer)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ key, action, uuid string }{
		{"during-root-exec", "box_exec", "sibling-c"},
		{"during-root-start", "box_start", "sibling-c"},
		{"during-root-stop", "box_stop", "sibling-c"},
		{"during-root-create", "box_create", ""},
	} {
		a := adm(tc.key, "user-1001", tc.action, tc.uuid)
		if tc.action == "box_create" {
			a.ResourceKey = "box:brand-new-name"
		}
		if _, err := s.Admit(ctx, a); !errors.Is(err, ErrResourceLocked) {
			t.Fatalf("%s admitted across Project-wide all-NIC root fence: %v", tc.action, err)
		}
	}
	if _, err := s.Admit(ctx, adm("foreign-can-continue", "user-1002", "box_exec", "foreign-uuid")); err != nil {
		t.Fatalf("foreign Project blocked: %v", err)
	}
	if err := s.Claim(ctx, root.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, root.ID, "DISPATCHING", "ROOT_EFFECT_ACK_UNCERTAIN"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Admit(ctx, adm("unknown-rearm-bypass", "user-1001", "box_exec", "sibling-c")); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN Project-wide root fence bypassed: %v", err)
	}
	// Even a forged legacy recovery audit RELEASED decision cannot rearm
	// the privileged root journal, whether the new request is a sibling
	// operation or a second root Peer effect.
	if _, err := s.db.ExecContext(ctx,
		"INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,?,?,?,?)",
		root.ID, time.Now().UnixNano(), "untrusted-review", "RELEASED", "forged"); err != nil {
		t.Fatal(err)
	}
	nextPeer := peer
	nextPeer.Key = "generic-audit-must-not-rearm-root"
	nextPeer.RequestID = nextPeer.Key
	if _, err := s.Admit(ctx, nextPeer); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("forged generic release bypassed privileged Root UNKNOWN: %v", err)
	}
	if _, err := s.Admit(ctx, adm("generic-audit-sibling-bypass", "user-1001", "box_exec", "sibling-c")); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("forged generic release allowed sibling mutation: %v", err)
	}
	t.Log("S4_B2_ALL_SIBLING_PROJECT_ADMISSION_AND_UNKNOWN_FREEZE=PASS")
}

func s4b2InsertForgedActiveSibling(t *testing.T, s *Store, id string) {
	t.Helper()
	_, err := s.db.Exec(`INSERT INTO operations
		(id,tracking_id,request_id,layer,actor_id,token_id,key_digest,action,intent_digest,
		 safe_intent,box_uuid,incarnation,state,created_at,updated_at)
		SELECT '11111111111111111111111111111111','c','c',layer,actor_id,token_id,
			   'different-sibling-key','box_exec','other','{}',?,?,'DISPATCHING',created_at,updated_at
		FROM operations WHERE id=?`, s4b2RootThird, s4b2RootThird, id)
	if err != nil {
		t.Fatal(err)
	}
}

func TestS4B2RootStageAndReadRecheckOldSiblingOperations(t *testing.T) {
	ctx := context.Background()
	t.Run("controller-stage", func(t *testing.T) {
		s, _, id := s4b2RootOperationFixture(t)
		s4b2PendingExistingGrantToConfirmed(t, s)
		s4b2InsertForgedActiveSibling(t, s, id)
		if err := s.StageRootPeerRevocation(ctx, id); !errors.Is(err, ErrResourceLocked) {
			t.Fatalf("unsafe stage admitted existing sibling: %v", err)
		}
	})
	t.Run("host-readonly", func(t *testing.T) {
		s, reader, id := s4b2RootOperationFixture(t)
		s4b2PendingExistingGrantToConfirmed(t, s)
		if err := s.StageRootPeerRevocation(ctx, id); err != nil {
			t.Fatal(err)
		}
		if err := s.Claim(ctx, id); err != nil {
			t.Fatal(err)
		}
		if _, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); err != nil {
			t.Fatal("valid baseline root proof: ", err)
		}
		s4b2InsertForgedActiveSibling(t, s, id)
		if _, err := reader.ReadRootPeerRevocation(ctx, id, "user-1001", "bwbr0", "bw-s4b2"); !errors.Is(err, ErrResourceLocked) {
			t.Fatalf("root failed to reject concurrent sibling: %v", err)
		}
	})
	t.Log("S4_B2_ALL_SIBLING_STAGE_AND_ROOT_READONLY_RECHECK=PASS")
}
