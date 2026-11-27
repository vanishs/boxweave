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
	_, token, err := s.IssueToken(ctx, time.Hour)
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
	r, err := s.AdmitScoped(ctx, a)
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
	r, err := s.AdmitScoped(ctx, a)
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
	if _, err := s.AdmitScoped(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("new idempotency key bypassed original UNKNOWN native Box: %v", err)
	}
	a.Key = "s3-file-create-unknown-0001"
	replay, err := s.Replay(ctx, a)
	if err != nil || !replay.Duplicate || replay.ID != r.ID || replay.State != "UNKNOWN" {
		t.Fatalf("uncertain original was not preserved: %+v %v", replay, err)
	}
}

func TestS3FileCreateVerifiedNoEffectDoesNotLockBox(t *testing.T) {
	ctx := context.Background()
	s, token, box := s3OwnedFileFixture(t)
	a := Admission{
		Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "s3-file-existing-original-0001", RequestID: "s3-first",
		Action: "box_file_apply", SafeIntent: []byte("{}"),
		ResourceKey: "box:" + box.Name, BoxUUID: box.UUID,
		Incarnation: box.Incarnation,
	}
	r, err := s.AdmitScoped(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	worker := Worker{Store: s, Budget: time.Second}
	if err := worker.SubmitGuestFileCreate(r.ID, box, func(context.Context) error {
		return PreDispatchError{Class: "SFTP_EXCLUSIVE_FILE_ALREADY_EXISTS"}
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		original, err := s.Get(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		if original.State == "FAILED" {
			if original.ErrorClass != "SFTP_EXCLUSIVE_FILE_ALREADY_EXISTS" {
				t.Fatalf("unexpected proven no-effect classification: %+v", original)
			}
			break
		}
		if original.State == "UNKNOWN" || time.Now().After(deadline) {
			t.Fatalf("proven no-effect must not lock Box as UNKNOWN: %+v", original)
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.Key = "s3-file-existing-original-0002"
	a.RequestID = "s3-new"
	if _, err := s.AdmitScoped(ctx, a); err != nil {
		t.Fatalf("proven no-effect leaked original Box lock: %v", err)
	}
}

func TestS3FileCreateAmbiguousWorkerEffectRemainsLocked(t *testing.T) {
	ctx := context.Background()
	s, token, box := s3OwnedFileFixture(t)
	a := Admission{
		Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "s3-file-transport-original-0001", RequestID: "s3-first",
		Action: "box_file_apply", SafeIntent: []byte("{}"),
		ResourceKey: "box:" + box.Name, BoxUUID: box.UUID,
		Incarnation: box.Incarnation,
	}
	r, err := s.AdmitScoped(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Worker{Store: s, Budget: time.Second}).SubmitGuestFileCreate(r.ID, box, func(context.Context) error {
		return errors.New("simulated lost SFTP reply")
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		original, err := s.Get(ctx, r.ID)
		if err != nil {
			t.Fatal(err)
		}
		if original.State == "UNKNOWN" {
			break
		}
		if original.State == "FAILED" || original.State == "SUCCEEDED" || time.Now().After(deadline) {
			t.Fatalf("lost SFTP reply cannot be signed completed: %+v", original)
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.Key = "s3-file-transport-original-0002"
	a.RequestID = "s3-new"
	if _, err := s.AdmitScoped(ctx, a); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("ambiguous native original lost its lock: %v", err)
	}
}
