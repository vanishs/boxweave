package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func s3OwnedFileFixture(t *testing.T) (*Store, Token, OwnedBox) {
	t.Helper()
	s := newStore(t)
	ctx := context.Background()
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	box := OwnedBox{
		UUID:    "c0000000-0000-4000-8000-000000000812",
		Project: "s3", Name: "s3-file-box",
		Incarnation: "c0000000-0000-4000-8000-000000000812",
		State:       "OWNED",
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO owned_boxes
		(uuid,project,name,incarnation,state,created_at) VALUES (?,?,?,?,?,?)`,
		box.UUID, box.Project, box.Name, box.Incarnation, box.State, time.Now().UnixNano())
	if err != nil {
		t.Fatal(err)
	}
	return s, token, box
}

func TestS3FileCreateAtomicTerminalAndOriginalIdempotency(t *testing.T) {
	ctx := context.Background()
	s, token, box := s3OwnedFileFixture(t)
	a := Admission{
		Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "s3-file-create-atomic-0001", RequestID: "s3-first",
		Action: "box_file_apply", SafeIntent: []byte(`{"path":"/tmp/create","sha256":"example","mode":"create"}`),
		ResourceKey: "box:" + box.Name, BoxUUID: box.UUID,
		Incarnation: box.Incarnation,
	}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.FinalizeVerifiedGuestFileCreate(ctx, r.ID, box); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, r.ID)
	if err != nil || got.State != "SUCCEEDED" || got.NativeOperationID != "" {
		t.Fatalf("SFTP invents native operation id or lost original terminal: %+v %v", got, err)
	}
	if err := s.FinalizeVerifiedGuestFileCreate(ctx, r.ID, box); !errors.Is(err, ErrState) {
		t.Fatalf("re-finalized original file create: %v", err)
	}
	replay, err := s.Replay(ctx, a)
	if err != nil || !replay.Duplicate || replay.ID != r.ID || replay.TrackingID != r.TrackingID {
		t.Fatalf("original SFTP receipt cannot replay after terminal: %+v %v", replay, err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED' AND safe_detail='VERIFIED_SFTP_EXCLUSIVE_CREATE_NO_NATIVE_OPERATION'",
		r.ID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("atomic SFTP terminal event %d: %v", events, err)
	}
}

func TestS3FileCreateUncertainEffectLocksNativeBoxAndNeverReplays(t *testing.T) {
	ctx := context.Background()
	s, token, box := s3OwnedFileFixture(t)
	a := Admission{
		Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "s3-file-create-unknown-0001", RequestID: "s3-first",
		Action: "box_file_apply", SafeIntent: []byte(`{"path":"/tmp/unknown","sha256":"example","mode":"create"}`),
		ResourceKey: "box:" + box.Name, BoxUUID: box.UUID,
		Incarnation: box.Incarnation,
	}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Unknown(ctx, r.ID, "DISPATCHING", "SFTP_CREATE_OUTCOME_UNVERIFIED"); err != nil {
		t.Fatal(err)
	}
	a.Key = "s3-file-create-unknown-0002"
	a.RequestID = "s3-other"
	if _, err := s.Admit(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("new idempotency key bypassed original UNKNOWN native Box: %v", err)
	}
	a.Key = "s3-file-create-unknown-0001"
	replay, err := s.Replay(ctx, a)
	if err != nil || !replay.Duplicate || replay.ID != r.ID || replay.State != "UNKNOWN" {
		t.Fatalf("uncertain original was not preserved: %+v %v", replay, err)
	}
}
