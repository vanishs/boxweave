package state

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
)

// A fully observed native Incus 7.6 LXC Exec ending with Guest 126/127 is
// terminal even though Incus labels it Failure. Its real side effects are
// UNKNOWN and are never replayed; a later distinct command may be admitted
// only after the exact original final+exit+log evidence is durably recorded.
// Contrast with worker_bh018_exec_boundary_test.go where an unverified
// *original* terminal result must remain UNKNOWN and retain its lock.
func TestBH018ExactTerminalSpecialExitPersistsAndReleasesClaimWithoutReplay(t *testing.T) {
	for _, tc := range []struct {
		name    string
		exit    int
		message string
	}{
		{"guest-real-exit-126", 126, "Command not executable"},
		{"guest-real-exit-127", 127, "Command not found"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store := newStore(t)
			_, token, err := store.IssueToken(ctx,time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			const id = "a3da911d-89e8-40b3-8b9f-cc9f08f08180"
			const guest = "bh018-owned"
			owned := OwnedBox{UUID: "bh018-uuid", Project: "test-project",
				Name: guest, Incarnation: "bh018-uuid", State: "OWNED"}
			if _, err := store.db.ExecContext(ctx, `INSERT INTO owned_boxes
				(uuid,project,name,incarnation,state,created_at)
				VALUES (?,?,?,?,'OWNED',?)`,
				owned.UUID, owned.Project, owned.Name, owned.Incarnation,
				time.Now().UnixNano()); err != nil {
				t.Fatal(err)
			}
			intent := Admission{Layer: "test-project", ActorID: token.ID, TokenID: token.ID,
				Key: "original-exec-special-exit-001", RequestID: "original",
				Action: "box_exec", ResourceKey: "box:" + guest,
				BoxUUID: "bh018-uuid", Incarnation: "bh018-uuid",
				SafeIntent: []byte(`{"argv_hash":"test-original"}`),
			}
			original, err := store.Admit(ctx, intent)
			if err != nil {
				t.Fatal(err)
			}
			prefix := fmt.Sprintf("/1.0/instances/%s/logs/exec-output/exec_%s", guest, id)
			final := &api.Operation{ID: id, StatusCode: api.Failure, Err: tc.message, Metadata: map[string]any{
				"return": float64(tc.exit),
				"output": map[string]any{"1": prefix + ".stdout", "2": prefix + ".stderr"},
			}}
			effectCalls := 0
			Worker{Store: store, Budget: time.Second}.Execute(original.ID,
				func(context.Context) (string, error) {
					effectCalls++
					return id, nil
				},
				func(ctx context.Context) error {
					ev, err := native.ExecEvidenceFromFinal(final)
					if err != nil {
						return err
					}
					if !native.ExecEvidenceMatchesOriginal(ev, guest, id) {
						return native.ErrUnverified
					}
					if ev.Exit == nil {
						return native.ErrUnverified
					}
					if err := store.FinalizeVerifiedExec(ctx, original.ID, owned,
						id, ev.StdoutPath, ev.StderrPath, *ev.Exit); err != nil {
						return err
					}
					return ErrVerifiedExecCommitted
				})
			stored, err := store.Get(ctx, original.ID)
			if err != nil || stored.State != "SUCCEEDED" || stored.NativeOperationID != id || effectCalls != 1 {
				t.Fatalf("known finished Exec wrongly UNKNOWN or replayed: receipt=%+v effects=%d err=%v", stored, effectCalls, err)
			}
			out, err := store.ExecOutput(ctx, original.ID)
			if err != nil || out.ExitCode == nil || *out.ExitCode != tc.exit ||
				out.StdoutPath != prefix+".stdout" || out.StderrPath != prefix+".stderr" {
				t.Fatalf("lost native final/guest nonzero exit receipt: %+v, %v", out, err)
			}
			replay, err := store.Replay(ctx, intent)
			if err != nil || !replay.Duplicate || replay.ID != original.ID ||
				replay.NativeOperationID != id || replay.State != "SUCCEEDED" || effectCalls != 1 {
				t.Fatalf("original replay issued a command or lost provenance: %+v %v", replay, err)
			}
			intent.Key = "different-new-command-allowed-after-exact-terminal-002"
			if _, err := store.Admit(ctx, intent); err != nil {
				t.Fatalf("verified finished Guest Exec permanently locked owned Box: %v", err)
			}
			if effectCalls != 1 {
				t.Fatal("terminal original effect was automatically executed again")
			}
		})
	}
}
