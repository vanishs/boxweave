package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

// S4-A #148: a refreshed access token must be a NEW credential of the SAME
// authorized grant actor, not a new native idempotency scope.
func TestS4OAuthGrantActorReplayAfterTerminalAndCredentialIsolation(t *testing.T) {
	ctx := context.Background()
	s, client, pending, verifier, _, redirect, password := oauthFixture(t)
	code, _, _, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil {
		t.Fatal(err)
	}
	pair, err := s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.VerifyToken(ctx, pair.Access)
	if err != nil || before.ActorID != pair.GrantID || before.ID == before.ActorID {
		t.Fatalf("oauth actor/credential conflated: %+v %v", before, err)
	}
	input := Admission{Layer: "test-s4", ActorID: before.ActorID, TokenID: before.ID,
		Key: "s4-refresh-original-key-000001", Action: "box_exec",
		SafeIntent: []byte(`{"argv":["true"]}`), RequestID: "first"}
	original, err := s.Admit(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.NativeAck(ctx, original.ID, "native-original-once-only"); err != nil {
		t.Fatal(err)
	}
	if err := s.Verified(ctx, original.ID); err != nil {
		t.Fatal(err)
	}
	next, err := s.RefreshOAuth(ctx, pair.Refresh, client, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	after, err := s.VerifyToken(ctx, next.Access)
	if err != nil || after.ActorID != before.ActorID || after.ID == before.ID {
		t.Fatalf("refresh changed stable actor or reused credential: %+v %v", after, err)
	}
	input.TokenID, input.RequestID = after.ID, "refreshed"
	for _, replay := range []func(context.Context, Admission) (Receipt, error){s.Replay, s.Admit} {
		got, err := replay(ctx, input)
		if err != nil || !got.Duplicate || got.ID != original.ID ||
			got.TrackingID != original.TrackingID || got.NativeOperationID != "native-original-once-only" ||
			got.State != "SUCCEEDED" {
			t.Fatalf("replay not original: %+v %v", got, err)
		}
	}
	if err := s.RequireActiveTokenID(ctx, before.ID); !errors.Is(err, ErrRevoked) {
		t.Fatalf("old access remained live: %v", err)
	}
	input.TokenID = before.ID
	if _, err := s.Replay(ctx, input); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked access replayed: %v", err)
	}
	input.TokenID = after.ID
	if err := s.RevokeOAuthGrant(ctx, pair.GrantID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Admit(ctx, input); !errors.Is(err, ErrRevoked) {
		t.Fatalf("revoked grant admitted/replayed: %v", err)
	}
}

// Refresh cannot authorize the prior credential's queued Claim, and an
// independent manual token or grant cannot borrow another grant's receipt.
func TestS4OAuthRefreshQueuedClaimAndForeignActor(t *testing.T) {
	ctx := context.Background()
	s, client, pending, verifier, challenge, redirect, password := oauthFixture(t)
	code, _, _, err := s.ApproveOAuth(ctx, pending, password)
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.ExchangeOAuth(ctx, client, redirect, code, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	old, err := s.VerifyToken(ctx, first.Access)
	if err != nil {
		t.Fatal(err)
	}
	queued := Admission{Layer: "test-s4", ActorID: old.ActorID, TokenID: old.ID,
		Key: "s4-refresh-queued-original-00001", Action: "box_exec",
		SafeIntent: []byte(`{"argv":["true"]}`), RequestID: "original"}
	rec, err := s.Admit(ctx, queued)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.RefreshOAuth(ctx, first.Refresh, client, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	newAccess, err := s.VerifyToken(ctx, second.Access)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, rec.ID); !errors.Is(err, ErrRevoked) {
		t.Fatalf("refresh allowed revoked queued dispatch: %v", err)
	}
	queued.TokenID = newAccess.ID
	again, err := s.Replay(ctx, queued)
	if err != nil || again.State != "FAILED" || again.ID != rec.ID {
		t.Fatalf("fresh credential must see non-dispatched original: %+v %v", again, err)
	}
	// A second grant issued to the SAME OAuth client is still a different actor.
	otherPending, err := s.BeginOAuth(ctx, client, redirect, "", challenge, testOAuthOrigin+"/mcp", "other-state-12345678")
	if err != nil {
		t.Fatal(err)
	}
	otherCode, _, _, err := s.ApproveOAuth(ctx, otherPending, password)
	if err != nil {
		t.Fatal(err)
	}
	otherPair, err := s.ExchangeOAuth(ctx, client, redirect, otherCode, verifier, testOAuthOrigin+"/mcp")
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.VerifyToken(ctx, otherPair.Access)
	if err != nil {
		t.Fatal(err)
	}
	queued.ActorID, queued.TokenID = other.ActorID, other.ID
	if _, err := s.Replay(ctx, queued); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other grant borrowed operation: %v", err)
	}
	manualRaw, manual, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if verified, err := s.VerifyToken(ctx, manualRaw); err != nil || verified.ActorID != manual.ID {
		t.Fatalf("manual actor changed: %+v %v", verified, err)
	}
	queued.ActorID, queued.TokenID = old.ActorID, manual.ID
	if _, err := s.Replay(ctx, queued); !errors.Is(err, ErrRevoked) {
		t.Fatalf("manual token impersonated grant: %v", err)
	}
}
