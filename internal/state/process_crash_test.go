package state

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// This verifies durability across a NEW OS PROCESS, rather than calling
// Recover on a still-live SQLite handle. It uses synthetic native ACKs only;
// it is not evidence of an actual Incus mutation or a native postcondition.
func TestProcessCrashUnknownNoReplay(t *testing.T) {
	if os.Getenv("BW_S1_CRASH_CHILD") == "1" {
		path := os.Getenv("BW_S1_CRASH_DB")
		marker := os.Getenv("BW_S1_CRASH_MARKER")
		stage := os.Getenv("BW_S1_CRASH_STAGE")
		ctx := context.Background()
		s, err := Open(path)
		if err != nil {
			panic(err)
		}
		_, token, err := s.IssueToken(ctx, "admin", time.Hour)
		if err != nil {
			panic(err)
		}
		a := Admission{
			Layer: "L1", ActorID: token.ID, TokenID: token.ID,
			Key:        "process-crash-test-fixed-key",
			Action:     "synthetic-operation-no-native-mutation",
			SafeIntent: []byte(`{"kind":"test"}`),
			RequestID:  "initial",
		}
		r, err := s.Admit(ctx, a)
		if err != nil {
			panic(err)
		}
		if err = s.Claim(ctx, r.ID); err != nil {
			panic(err)
		}
		if stage == "verifying" {
			if err = s.NativeAck(ctx, r.ID, "synthetic-native-operation-id"); err != nil {
				panic(err)
			}
		}
		data, err := json.Marshal(struct {
			Operation string
			Token     string
		}{r.ID, token.ID})
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile(marker, data, 0600); err != nil {
			panic(err)
		}
		// Abrupt process end: no Store.Close and no deferred cleanup.
		os.Exit(0)
	}
	for _, stage := range []string{"dispatching", "verifying"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path, marker := filepath.Join(dir, "ledger.sqlite"), filepath.Join(dir, "original-ids.json")
			cmd := exec.Command(os.Args[0], "-test.run=^TestProcessCrashUnknownNoReplay$")
			cmd.Env = append(os.Environ(), "BW_S1_CRASH_CHILD=1", "BW_S1_CRASH_DB="+path,
				"BW_S1_CRASH_MARKER="+marker, "BW_S1_CRASH_STAGE="+stage)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("child process failed: %v: %s", err, output)
			}
			var ids struct {
				Operation string
				Token     string
			}
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(raw, &ids); err != nil {
				t.Fatal(err)
			}
			s, err := Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			ctx := context.Background()
			n, err := s.Recover(ctx)
			if err != nil || n != 1 {
				t.Fatalf("recover = %d, %v", n, err)
			}
			r, err := s.Get(ctx, ids.Operation)
			if err != nil || r.ID != ids.Operation || r.State != "UNKNOWN" {
				t.Fatalf("original operation was lost: %+v %v", r, err)
			}
			if stage == "verifying" && r.NativeOperationID != "synthetic-native-operation-id" {
				t.Fatalf("lost original synthetic ACK identity: %+v", r)
			}
			a := Admission{Layer: "L1", ActorID: ids.Token, TokenID: ids.Token,
				Key: "process-crash-test-fixed-key", Action: "synthetic-operation-no-native-mutation",
				SafeIntent: []byte(`{"kind":"test"}`), RequestID: "retry-id"}
			replayed, err := s.Admit(ctx, a)
			if err != nil || !replayed.Duplicate || replayed.ID != r.ID || replayed.State != "UNKNOWN" {
				t.Fatalf("idem after process crash: %+v %v", replayed, err)
			}
			effects := 0
			w := Worker{Store: s}
			err = w.Submit(r.ID, func(context.Context) (string, error) {
				effects++
				return "should-never-run", nil
			}, func(context.Context) error { return nil })
			if !errors.Is(err, ErrState) || effects != 0 {
				t.Fatalf("native replay risk after process restart: effects=%d err=%v", effects, err)
			}
		})
	}
}
