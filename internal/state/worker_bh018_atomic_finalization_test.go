package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests exercise SQLite transactions, not a mocked in-memory ledger.
// They do not dispatch a second native command after an ambiguous result.
func bh018AtomicFixture(t *testing.T, s *Store) (Admission, OwnedBox, string, string) {
	t.Helper()
	ctx := context.Background()
	_, token, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	const nativeID = "a3da911d-89e8-40b3-8b9f-cc9f08f08180"
	box := OwnedBox{UUID: "bh018-owned-uuid", Project: "test-project",
		Name: "bh018-owned", Incarnation: "bh018-owned-uuid", State: "OWNED"}
	if _, err = s.db.ExecContext(ctx, `INSERT INTO owned_boxes
		(uuid,project,name,incarnation,state,created_at)
		VALUES (?,?,?,?,'OWNED',?)`,
		box.UUID, box.Project, box.Name, box.Incarnation, time.Now().UnixNano()); err != nil {
		t.Fatal(err)
	}
	intent := Admission{Layer: box.Project, ActorID: token.ID, TokenID: token.ID,
		Key: "bh018-exec-original", RequestID: "original", Action: "box_exec",
		ResourceKey: "box:" + box.Name, BoxUUID: box.UUID,
		Incarnation: box.Incarnation, SafeIntent: []byte(`{"cmd":"fixture"}`)}
	rec, err := s.Admit(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Claim(ctx, rec.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.NativeAck(ctx, rec.ID, nativeID); err != nil {
		t.Fatal(err)
	}
	prefix := "/1.0/instances/" + box.Name + "/logs/exec-output/exec_" + nativeID
	return intent, box, rec.ID, prefix
}

func TestBH018AtomicFinalizationNoPartialReceiptOnSQLiteAbort(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	intent, box, opID, prefix := bh018AtomicFixture(t, s)
	const nativeID = "a3da911d-89e8-40b3-8b9f-cc9f08f08180"
	// Inject an SQLite failure *after* the transaction's UPDATE
	// but *before* its output INSERT and terminal event/COMMIT.
	if _, err := s.db.ExecContext(ctx, `CREATE TRIGGER bh018_inject_abort
		BEFORE INSERT ON exec_outputs
		BEGIN SELECT RAISE(ABORT, 'BH018_COMMIT_BOUNDARY_TEST'); END`); err != nil {
		t.Fatal(err)
	}
	err := s.FinalizeVerifiedExec(ctx, opID, box, nativeID,
		prefix+".stdout", prefix+".stderr", 127)
	if err == nil || !strings.Contains(err.Error(), "BH018_COMMIT_BOUNDARY_TEST") {
		t.Fatalf("injected SQLite boundary failed to abort: %v", err)
	}
	got, err := s.Get(ctx, opID)
	if err != nil || got.State != "VERIFYING" || got.NativeOperationID != nativeID {
		t.Fatalf("partial state leaked on rollback: %+v err=%v", got, err)
	}
	if _, err := s.ExecOutput(ctx, opID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("output row became independently durable: %v", err)
	}
	var partialEvents int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'",
		opID).Scan(&partialEvents); err != nil || partialEvents != 0 {
		t.Fatalf("terminal event leaked before COMMIT: count=%d err=%v", partialEvents, err)
	}
	if _, err := s.db.ExecContext(ctx, "DROP TRIGGER bh018_inject_abort"); err != nil {
		t.Fatal(err)
	}
	// A process restart at this point is fail-closed: the original remains
	// UNKNOWN and cannot be auto-finalized from a missing output row.
	if n, err := s.Recover(ctx); err != nil || n != 1 {
		t.Fatalf("recovery: n=%d err=%v", n, err)
	}
	got, err = s.Get(ctx, opID)
	if err != nil || got.State != "UNKNOWN" || got.NativeOperationID != nativeID {
		t.Fatalf("unknown original must remain locked: %+v err=%v", got, err)
	}
	intent.Key = "bh018-distinct-new-command"
	if _, err := s.Admit(ctx, intent); err == nil {
		t.Fatal("recovery improperly unlocked uncertain original native command")
	}
}

func TestBH018AtomicFinalizationSurvivesStoreCloseReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "bh018-ledger.sqlite")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	intent, box, opID, prefix := bh018AtomicFixture(t, s)
	const nativeID = "a3da911d-89e8-40b3-8b9f-cc9f08f08180"
	if err := s.FinalizeVerifiedExec(ctx, opID, box, nativeID,
		prefix+".stdout", prefix+".stderr", 126); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if n, err := s.Recover(ctx); err != nil || n != 0 {
		t.Fatalf("restart reopened a fully committed original Exec: n=%d err=%v", n, err)
	}
	r, err := s.Get(ctx, opID)
	if err != nil || r.State != "SUCCEEDED" || r.NativeOperationID != nativeID {
		t.Fatalf("atomic terminal status lost after restart: %+v err=%v", r, err)
	}
	out, err := s.ExecOutput(ctx, opID)
	if err != nil || out.ExitCode == nil || *out.ExitCode != 126 ||
		out.StdoutPath != prefix+".stdout" || out.StderrPath != prefix+".stderr" {
		t.Fatalf("terminal/output commit split after restart: %+v err=%v", out, err)
	}
	var events int
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM operation_events WHERE operation_id=? AND state='SUCCEEDED'",
		opID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("should be exactly one original terminal event: %d err=%v", events, err)
	}
	replay, err := s.Replay(ctx, intent)
	if err != nil || !replay.Duplicate || replay.ID != opID || replay.State != "SUCCEEDED" {
		t.Fatalf("original terminal after restart not deduplicated: %+v err=%v", replay, err)
	}
	// Two-phase finalization retries are forbidden, not silently idempotent
	// writes that could corrupt the first immutable Guest exit.
	if err := s.FinalizeVerifiedExec(ctx, opID, box, nativeID,
		prefix+".stdout", prefix+".stderr", 127); !errors.Is(err, ErrState) {
		t.Fatalf("second finalization changed immutable terminal: %v", err)
	}
	intent.Key = "bh018-different-new-command"
	if _, err := s.Admit(ctx, intent); err != nil {
		t.Fatalf("committed final falsely locked fresh command: %v", err)
	}
}

func TestBH018AtomicFinalizationRejectsForeignIdentityWithoutAnyRow(t *testing.T) {
	const nativeID = "a3da911d-89e8-40b3-8b9f-cc9f08f08180"
	for _, tc := range []struct {
		name  string
		alter func(*OwnedBox, *string, *string, *string)
	}{
		{"foreign-project", func(b *OwnedBox, id, out, stderr *string) { b.Project = "other-project" }},
		{"foreign-box-uuid", func(b *OwnedBox, id, out, stderr *string) { b.UUID = "foreign" }},
		{"foreign-incarnation", func(b *OwnedBox, id, out, stderr *string) { b.Incarnation = "foreign" }},
		{"foreign-box-name", func(b *OwnedBox, id, out, stderr *string) { b.Name = "foreign" }},
		{"tombstoned-claim", func(b *OwnedBox, id, out, stderr *string) { b.State = "DELETED" }},
		{"wrong-native-op", func(b *OwnedBox, id, out, stderr *string) { *id = "00000000-0000-0000-0000-000000000000" }},
		{"foreign-stdout-path", func(b *OwnedBox, id, out, stderr *string) {
			*out = "/1.0/instances/foreign/logs/exec-output/exec_" + *id + ".stdout"
		}},
		{"mismatched-stderr-stream", func(b *OwnedBox, id, out, stderr *string) { *stderr = *out }},
		{"missing-output", func(b *OwnedBox, id, out, stderr *string) { *out, *stderr = "", "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newStore(t)
			_, box, opID, prefix := bh018AtomicFixture(t, s)
			id, stdout, stderr := nativeID, prefix+".stdout", prefix+".stderr"
			tc.alter(&box, &id, &stdout, &stderr)
			if err := s.FinalizeVerifiedExec(ctx, opID, box, id, stdout, stderr, 127); err == nil {
				t.Fatal("forged original identity was allowed to release resource")
			}
			got, err := s.Get(ctx, opID)
			if err != nil || got.State != "VERIFYING" || got.NativeOperationID != nativeID {
				t.Fatalf("unsafe state on rejected original evidence: %+v err=%v", got, err)
			}
			if _, err := s.ExecOutput(ctx, opID); !errors.Is(err, ErrNotFound) {
				t.Fatalf("partial/fake output persisted: %v", err)
			}
		})
	}
}

func TestBH018WorkerRejectsForgedAtomicExecCompletionSignal(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)
	_, _, opID, _ := bh018AtomicFixture(t, s)
	// Explicitly return the verifier's private control signal without
	// having committed a real final receipt. Worker must read back the
	// original journal rather than trust this signal.
	if err := s.Unknown(ctx, opID, "VERIFYING", "test-reset-for-worker"); err != nil {
		t.Fatal(err)
	}
	// Create a separate admitted original because only ADMITTED may Claim.
	_, token, err := s.IssueToken(ctx,time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	b := Admission{Layer: "separate", ActorID: token.ID, TokenID: token.ID,
		Key: "bh018-forged-committed-control", RequestID: "original",
		Action: "box_exec", ResourceKey: "box:independent", BoxUUID: "second",
		Incarnation: "second", SafeIntent: []byte(`{"fixture":1}`)}
	original, err := s.Admit(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	Worker{Store: s, Budget: time.Second}.Execute(original.ID,
		func(context.Context) (string, error) { return "a3da911d-89e8-40b3-8b9f-cc9f08f08180", nil },
		func(context.Context) error { return ErrVerifiedExecCommitted })
	got, err := s.Get(ctx, original.ID)
	if err != nil || got.State != "UNKNOWN" {
		t.Fatalf("forged completion sentinel incorrectly unlocked original: %+v %v", got, err)
	}
}

// Verified terminal path/exit must be readable after COMMIT. Reject a
// forged native exit that cannot round-trip through the public int32 reader.
func TestBH018AtomicFinalizationRejectsOutOfRangeExitBeforeWrite(t *testing.T) {
	ctx := context.Background()
	for _, v := range []int64{int64(1) << 40, -(int64(1) << 40)} {
		value := int(v)
		if int64(value) != v {
			continue // 32-bit architecture: no representable invalid int.
		}
		s := newStore(t)
		_, box, opID, prefix := bh018AtomicFixture(t, s)
		if err := s.FinalizeVerifiedExec(ctx, opID, box,
			"a3da911d-89e8-40b3-8b9f-cc9f08f08180", prefix+".stdout", "", value); !errors.Is(err, ErrInvalid) {
			t.Fatalf("out-of-range terminal exit %d accepted: %v", v, err)
		}
		got, err := s.Get(ctx, opID)
		if err != nil || got.State != "VERIFYING" {
			t.Fatalf("out-of-range exit altered source claim: %+v %v", got, err)
		}
		if _, err := s.ExecOutput(ctx, opID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("invalid exit produced output receipt: %v", err)
		}
	}
}
