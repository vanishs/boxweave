package state

import (
	"context"
	"errors"
	"testing"
	"time"
)

func s4CreateDeleteFixture(t *testing.T, st *Store) (Admission, OwnedBox, Receipt) {
	t.Helper()
	ctx := context.Background()
	_, token, err := st.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	in := Admission{Layer: "s4-project", ActorID: token.ID, TokenID: token.ID,
		Key: "s4-lifecycle-original-create-key", RequestID: "original",
		Action: "box_create", ResourceKey: "box:s4-created", SafeIntent: []byte(`{"name":"s4-created"}`)}
	rec, err := st.Admit(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	box := OwnedBox{Project: "s4-project", Name: "s4-created", UUID: "s4-exact-native-uuid",
		Incarnation: "s4-exact-native-uuid", State: "OWNED"}
	return in, box, rec
}

func TestS4CreateUUIDOriginalTerminalAtomicRollbackAndSuccess(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	in, box, rec := s4CreateDeleteFixture(t, st)
	if err := st.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.NativeAck(ctx, rec.ID, "incus-create-native-original"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.ExecContext(ctx, `CREATE TRIGGER s4_create_abort
		BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED'
		BEGIN SELECT RAISE(ABORT, 'S4_CREATE_ATOMIC_FAULT'); END`); err != nil {
		t.Fatal(err)
	}
	if err := st.FinalizeVerifiedCreate(ctx, rec.ID, box.Project, box.Name, box.UUID); err == nil {
		t.Fatal("create transaction unexpectedly committed through injected fault")
	}
	if _, err := st.OwnedBox(ctx, box.Project, box.UUID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed create terminal leaked UUID registration: %v", err)
	}
	got, err := st.Get(ctx, rec.ID)
	if err != nil || got.State != "VERIFYING" || got.BoxUUID != "" {
		t.Fatalf("split original create registration and terminal: %+v %v", got, err)
	}
	if _, err := st.db.ExecContext(ctx, "DROP TRIGGER s4_create_abort"); err != nil {
		t.Fatal(err)
	}
	if err := st.FinalizeVerifiedCreate(ctx, rec.ID, box.Project, box.Name, box.UUID); err != nil {
		t.Fatal(err)
	}
	got, err = st.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.BoxUUID != box.UUID ||
		got.NativeOperationID != "incus-create-native-original" {
		t.Fatalf("wrong atomic original create: %+v %v", got, err)
	}
	if err := st.FinalizeVerifiedCreate(ctx, rec.ID, box.Project, box.Name, box.UUID); !errors.Is(err, ErrState) {
		t.Fatalf("second create terminal admitted: %v", err)
	}
	replay, err := st.Replay(ctx, in)
	if err != nil || !replay.Duplicate || replay.ID != rec.ID {
		t.Fatalf("original create replay: %+v %v", replay, err)
	}
}

func TestS4DeleteTombstoneOriginalTerminalAtomicAndUnknownCrash(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	in, box, create := s4CreateDeleteFixture(t, st)
	if err := st.Claim(ctx, create.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.NativeAck(ctx, create.ID, "incus-create-op"); err != nil {
		t.Fatal(err)
	}
	if err := st.FinalizeVerifiedCreate(ctx, create.ID, box.Project, box.Name, box.UUID); err != nil {
		t.Fatal(err)
	}
	del := in
	del.Key, del.RequestID, del.Action = "s4-delete-original-effect-key", "delete", "box_delete"
	del.BoxUUID, del.Incarnation = box.UUID, box.Incarnation
	del.SafeIntent = []byte(`{"box_id":"s4-exact-native-uuid"}`)
	r, err := st.Admit(ctx, del)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Claim(ctx, r.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.NativeAck(ctx, r.ID, "incus-delete-native-original"); err != nil {
		t.Fatal(err)
	}
	foreign := box
	foreign.Incarnation = "foreign"
	if err := st.FinalizeVerifiedDelete(ctx, r.ID, foreign); !errors.Is(err, ErrState) {
		t.Fatalf("foreign incarnation deleted claimed Box: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `CREATE TRIGGER s4_delete_abort
		BEFORE UPDATE ON operations WHEN NEW.state='SUCCEEDED' AND OLD.state='VERIFYING'
		BEGIN SELECT RAISE(ABORT, 'S4_DELETE_ATOMIC_FAULT'); END`); err != nil {
		t.Fatal(err)
	}
	if err := st.FinalizeVerifiedDelete(ctx, r.ID, box); err == nil {
		t.Fatal("delete transaction unexpectedly committed through injected fault")
	}
	if _, err := st.OwnedBox(ctx, box.Project, box.UUID); err != nil {
		t.Fatalf("delete rollback lost still owned identity: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, "DROP TRIGGER s4_delete_abort"); err != nil {
		t.Fatal(err)
	}
	if n, err := st.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("crash during original Delete not safely recovered: %d %v", n, err)
	}
	unknown, err := st.Get(ctx, r.ID)
	if err != nil || unknown.State != "UNKNOWN" || unknown.NativeOperationID != "incus-delete-native-original" {
		t.Fatalf("ambiguous Delete lost original native ID: %+v %v", unknown, err)
	}
	if _, err := st.OwnedBox(ctx, box.Project, box.UUID); err != nil {
		t.Fatalf("UNKNOWN Delete silently tombstoned Box: %v", err)
	}
	if _, err := st.Admit(ctx, Admission{Layer: del.Layer, ActorID: del.ActorID, TokenID: del.TokenID,
		Key: "s4-delete-distinct-retry-forbidden", Action: "box_delete", RequestID: "other",
		ResourceKey: del.ResourceKey, BoxUUID: del.BoxUUID, Incarnation: del.Incarnation,
		SafeIntent: del.SafeIntent}); !errors.Is(err, ErrResourceLocked) {
		t.Fatalf("UNKNOWN Delete permitted second native effect: %v", err)
	}
}

func TestS4DeleteAtomicTerminalTombstoneWorkerAndReplay(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	in, box, create := s4CreateDeleteFixture(t, st)
	Worker{Store: st}.Execute(create.ID,
		func(context.Context) (string, error) { return "incus-create-original", nil },
		func(context.Context) error {
			if err := st.FinalizeVerifiedCreate(ctx, create.ID, box.Project, box.Name, box.UUID); err != nil {
				return err
			}
			return ErrVerifiedCreateCommitted
		})
	c, err := st.Get(ctx, create.ID)
	if err != nil || c.State != "SUCCEEDED" || c.BoxUUID != box.UUID {
		t.Fatalf("worker create terminal: %+v %v", c, err)
	}
	del := in
	del.Action, del.Key, del.RequestID = "box_delete", "s4-original-delete-worker-001", "new"
	del.BoxUUID, del.Incarnation = box.UUID, box.Incarnation
	del.SafeIntent = []byte(`{"box_id":"s4-exact-native-uuid"}`)
	rec, err := st.Admit(ctx, del)
	if err != nil {
		t.Fatal(err)
	}
	effects := 0
	Worker{Store: st}.Execute(rec.ID,
		func(context.Context) (string, error) {
			effects++
			return "incus-delete-original", nil
		},
		func(context.Context) error {
			if err := st.FinalizeVerifiedDelete(ctx, rec.ID, box); err != nil {
				return err
			}
			return ErrVerifiedDeleteCommitted
		})
	got, err := st.Get(ctx, rec.ID)
	if err != nil || got.State != "SUCCEEDED" || got.NativeOperationID != "incus-delete-original" ||
		effects != 1 {
		t.Fatalf("worker failed atomic delete: %+v effects=%d err=%v", got, effects, err)
	}
	var by string
	var at int64
	if err := st.db.QueryRowContext(ctx, `SELECT deleted_by_operation_id,deleted_at FROM owned_boxes
		WHERE uuid=?`, box.UUID).Scan(&by, &at); err != nil || by != rec.ID || at <= 0 {
		t.Fatalf("tombstone lost original Delete reference: %q %d %v", by, at, err)
	}
	if _, err := st.OwnedBox(ctx, box.Project, box.UUID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("tombstoned Box still owned: %v", err)
	}
	again, err := st.Replay(ctx, del)
	if err != nil || !again.Duplicate || again.ID != rec.ID || again.NativeOperationID != "incus-delete-original" {
		t.Fatalf("delete replay changed native original: %+v %v", again, err)
	}
}
