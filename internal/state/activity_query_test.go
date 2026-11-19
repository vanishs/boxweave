package state

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func activityTestToken(t *testing.T, s *Store) Token {
	t.Helper()
	_, tok, err := s.IssueToken(context.Background(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func activityAdmit(t *testing.T, s *Store, tok Token, layer, key, action, box, resource string, claims ...string) Receipt {
	t.Helper()
	rec, err := s.Admit(context.Background(), Admission{
		Layer: layer, ActorID: tok.ID, TokenID: tok.ID,
		Key: key, RequestID: key, Action: action,
		SafeIntent: []byte("{}"), BoxUUID: box, Incarnation: "generation-1",
		ResourceKey: resource, ResourceKeys: claims,
		WorkRef: "activity-work", AgentRef: "activity-agent",
	})
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func TestActivityOverviewIncludesPeerTargetClaimWithoutForeignLayer(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	source := strings.Repeat("a", 32)
	target := strings.Repeat("b", 32)
	own := activityAdmit(t, s, tok, "user-1001", "activity-target-claim-001", "box_peer_set",
		source, "acl:default/bw-s4b2", "box:"+source, "box:"+target)
	foreign := activityAdmit(t, s, tok, "user-foreign", "activity-foreign-claim-002", "box_exec",
		target, "box:foreign")
	page, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
		View: "overview", BoxUUID: target})
	if err != nil || len(page.Operations) != 1 || page.Operations[0].ID != own.ID {
		t.Fatalf("target Box claim missing or foreign leaked: %+v %v", page, err)
	}
	if page.Scope != "user-1001" || page.Source != "SQLITE_SNAPSHOT" {
		t.Fatalf("untrusted Activity scope/source: %+v", page)
	}
	evidence, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
		View: "evidence", OperationID: foreign.ID})
	if !errors.Is(err, ErrNotFound) || evidence.Evidence != nil {
		t.Fatalf("foreign operation escaped Layer boundary: %+v %v", evidence, err)
	}
}

func TestActivityIssuesFindOldUnknownAndRootPeerDespiteLegacyRelease(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	ordinary := activityAdmit(t, s, tok, "user-1001", "activity-old-unknown-0001", "box_exec",
		strings.Repeat("c", 32), "box:old")
	if err := s.Claim(ctx, ordinary.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, ordinary.ID, "DISPATCHING", "LOST_RECEIPT"); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-40 * 24 * time.Hour).UnixNano()
	if _, err := s.db.ExecContext(ctx, "UPDATE operations SET created_at=? WHERE id=?", oldTime, ordinary.ID); err != nil {
		t.Fatal(err)
	}
	page, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
		View: "issues"})
	if err != nil || len(page.Issues) != 1 || page.Issues[0].OperationID != ordinary.ID {
		t.Fatalf("old UNKNOWN lost in recent-only history: %+v %v", page, err)
	}
	if page.AttentionComplete {
		t.Fatal("partial Activity implementation promised all native blockers")
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,1,'test','RELEASED','digest')", ordinary.ID); err != nil {
		t.Fatal(err)
	}
	peer := activityAdmit(t, s, tok, "user-1001", "activity-peer-unknown-002", "box_peer_set",
		strings.Repeat("d", 32), "acl:default/bw-s4b2", "box:"+strings.Repeat("d", 32))
	if err := s.Claim(ctx, peer.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, peer.ID, "DISPATCHING", "ROOT_UNKNOWN"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, "INSERT INTO recovery_audit(operation_id,at,reviewer,decision,evidence_digest) VALUES(?,2,'test','RELEASED','digest')", peer.ID); err != nil {
		t.Fatal(err)
	}
	page, err = s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
		View: "issues"})
	if err != nil || len(page.Issues) != 1 || page.Issues[0].OperationID != peer.ID ||
		page.Issues[0].Reason != "PROJECT_PEER_BARRIER" {
		t.Fatalf("generic RELEASED incorrectly cleared Root peer blocker: %+v %v", page, err)
	}
}

func TestActivityEvidenceReadsOrderedSanitizedEventsAndClaims(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	rec := activityAdmit(t, s, tok, "user-1001", "activity-evidence-order-001", "box_exec",
		strings.Repeat("a", 32), "box:sample", "box:"+strings.Repeat("b", 32))
	if err := s.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, rec.ID, "DISPATCHING", "UNKNOWN_RECEIPT"); err != nil {
		t.Fatal(err)
	}
	res, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001",
		Requester: tok.ID, View: "evidence", OperationID: rec.ID})
	if err != nil || res.Evidence == nil || res.Evidence.Original.ID != rec.ID ||
		len(res.Evidence.Events) < 3 || len(res.Evidence.ResourceKeys) != 2 ||
		res.Evidence.ResourceKeys[0] != "box:"+strings.Repeat("b", 32) {
		t.Fatalf("Activity evidence lost original/events/claims: %+v %v", res, err)
	}
	for i := 1; i < len(res.Evidence.Events); i++ {
		if res.Evidence.Events[i].Seq <= res.Evidence.Events[i-1].Seq {
			t.Fatalf("events out of sequence: %+v", res.Evidence.Events)
		}
	}
	if !strings.Contains(strings.Join(res.Evidence.Limitations, " "), "ROOT_JOURNAL_PRIVATE") {
		t.Fatal("private Root evidence not explicitly excluded")
	}
	if _, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001", Requester: tok.ID,
		View: "evidence", OperationID: rec.ID, WorkRef: "wrong-label"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("mismatched evidence filter was not narrowed: %v", err)
	}
}

func TestActivityPeerEvidenceVerifiedLinkNotNativePermission(t *testing.T) {
	ctx := context.Background()
	s, id := s4b3GrantFixture(t)
	if err := s.StagePeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.AckSynchronousACL(ctx, id); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeRootPeerGrant(ctx, id); err != nil {
		t.Fatal(err)
	}
	result, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001",
		Requester: id, View: "evidence", OperationID: id})
	if err != nil || result.Evidence == nil || len(result.Evidence.Peers) != 1 ||
		result.Evidence.Peers[0].VerifiedID != id ||
		result.Evidence.Peers[0].PendingID != "" {
		t.Fatalf("verified peer ledger original missing: %+v %v", result, err)
	}
	if !strings.Contains(strings.Join(result.Evidence.Limitations, " "), "NO_NATIVE_POSTSTATE") {
		t.Fatal("confirmed SQLite peer incorrectly implies current native ACL permission")
	}
}

func TestActivityQueryRejectsInvalidAndTruncates(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	tok := activityTestToken(t, s)
	for _, q := range []ActivityQuery{
		{Layer: "user-1001", Requester: "bad", View: "overview"},
		{Layer: "user-1001", Requester: tok.ID, View: "evidence"},
		{Layer: "user-1001", Requester: tok.ID, View: "issues", Limit: 10000},
		{Layer: "user-1001", Requester: tok.ID, View: "overview", BoxUUID: "x"},
	} {
		if _, err := s.ReadActivity(ctx, q); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid Activity query accepted: %+v %v", q, err)
		}
	}
	activityAdmit(t, s, tok, "user-1001", "activity-pagination-01", "box_exec",
		strings.Repeat("a", 32), "box:one")
	activityAdmit(t, s, tok, "user-1001", "activity-pagination-02", "box_exec",
		strings.Repeat("b", 32), "box:two")
	page, err := s.ReadActivity(ctx, ActivityQuery{Layer: "user-1001",
		Requester: tok.ID, View: "overview", Limit: 1})
	if err != nil || len(page.Operations) != 1 || !page.Truncated {
		t.Fatalf("bounded view claimed complete: %+v %v", page, err)
	}
}
