package native_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/state"
)

const nativeProbeConfigKey = "user.boxweave.s1_journal_probe"

// Opt-in ONLY on a test-owned disposable nested Incus Project/instance. This
// performs genuine native writes; CI and production never run it by default.
func s1NativeFixture(t *testing.T) (string, string, string) {
	t.Helper()
	if os.Getenv("BW_S1_NATIVE_INTEGRATION") != "approved-isolated-lab" {
		t.Skip("real native mutation needs explicit isolated lab and exact target")
	}
	project, name, uuid := os.Getenv("BW_S1_NATIVE_PROJECT"), os.Getenv("BW_S1_NATIVE_NAME"), os.Getenv("BW_S1_NATIVE_UUID")
	if os.Geteuid() == 0 || project == "" || name == "" || uuid == "" ||
		!strings.HasPrefix(project, "user-") || !strings.HasPrefix(name, "bw-s1-native-") {
		t.Fatal("refuse native effects outside a named restricted, nonroot lab")
	}
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket.user", nil)
	if err != nil {
		t.Fatal(err)
	}
	names, err := c.GetProjectNames()
	if err != nil || len(names) != 1 || names[0] != project {
		t.Fatalf("not a single confined Project: %v %v", names, err)
	}
	scoped := c.UseProject(project)
	p, _, err := scoped.GetProject(project)
	if err != nil || p.Config["restricted"] != "true" {
		t.Fatalf("not a restricted Project: %v", err)
	}
	i, _, err := scoped.GetInstance(name)
	if err != nil || i.Config["volatile.uuid"] != uuid {
		t.Fatalf("native target UUID changed or missing: %v", err)
	}
	return project, name, uuid
}

func s1NativeWrite(project, name, uuid, marker string) (string, error) {
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket.user", nil)
	if err != nil {
		return "", err
	}
	scoped := c.UseProject(project)
	i, etag, err := scoped.GetInstance(name)
	if err != nil {
		return "", err
	}
	if i.Config["volatile.uuid"] != uuid {
		return "", errors.New("native instance identity changed before effect")
	}
	next := i.Writable()
	if next.Config == nil {
		next.Config = make(map[string]string)
	}
	next.Config[nativeProbeConfigKey] = marker
	op, err := scoped.UpdateInstance(name, next, etag)
	if err != nil {
		return "", err
	}
	if err = op.Wait(); err != nil {
		return "", err
	}
	return op.Get().ID, nil
}

func s1NativeObserved(project, name, uuid, marker string) error {
	// A fresh native SDK client is the independent postcondition witness.
	c, err := incus.ConnectIncusUnix("/var/lib/incus/unix.socket.user", nil)
	if err != nil {
		return err
	}
	i, _, err := c.UseProject(project).GetInstance(name)
	if err != nil {
		return err
	}
	if i.Config["volatile.uuid"] != uuid || i.Config[nativeProbeConfigKey] != marker {
		return errors.New("independent native UUID/config postread disagrees")
	}
	return nil
}

func s1Admit(t *testing.T, s *state.Store, uuid, key string) state.Receipt {
	t.Helper()
	ctx := context.Background()
	_, token, err := s.IssueToken(ctx, "admin", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a := state.Admission{Layer: "L1", ActorID: token.ID, TokenID: token.ID, Key: key,
		Action: "lab-native-InstancePut-only", SafeIntent: []byte(`{"kind":"only-test-config"}`),
		BoxUUID: uuid, Incarnation: uuid, RequestID: "lab-" + key}
	r, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestS1IncusNativeJournalRecovery(t *testing.T) {
	project, name, uuid := s1NativeFixture(t)
	if os.Getenv("BW_S1_NATIVE_CRASH_CHILD") == "1" {
		db := os.Getenv("BW_S1_CHILD_DB")
		receiptPath := os.Getenv("BW_S1_CHILD_RECEIPT")
		marker := os.Getenv("BW_S1_CHILD_MARKER")
		s, err := state.Open(db)
		if err != nil {
			t.Fatal(err)
		}
		r := s1Admit(t, s, uuid, "s1-real-crash-key-00001")
		if err := s.Claim(context.Background(), r.ID); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(receiptPath, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = s1NativeWrite(project, name, uuid, marker); err != nil {
			t.Fatal(err)
		}
		// An abrupt exit AFTER an actual native mutation, BEFORE SQLite native ACK.
		// No deferred DB close; the parent runs in a completely new process.
		os.Exit(75)
	}
	t.Run("real-success", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
		s, err := state.Open(filepath.Join(dir, "journal.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		r := s1Admit(t, s, uuid, "s1-real-success-key-0001")
		marker := "success-" + r.ID
		effects := 0
		w := state.Worker{Store: s}
		w.Execute(r.ID, func(context.Context) (string, error) { effects++; return s1NativeWrite(project, name, uuid, marker) },
			func(context.Context) error { return s1NativeObserved(project, name, uuid, marker) })
		result, err := s.Get(context.Background(), r.ID)
		if err != nil || result.State != "SUCCEEDED" || result.NativeOperationID == "" || effects != 1 {
			t.Fatalf("native success journal invalid: %+v %v effects=%d", result, err, effects)
		}
		if err := s1NativeObserved(project, name, uuid, marker); err != nil {
			t.Fatal(err)
		}
		t.Logf("PASS actual verified Incus native operation %s", result.NativeOperationID)
	})
	t.Run("native-committed-but-reply-lost", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
		s, err := state.Open(filepath.Join(dir, "journal.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		r := s1Admit(t, s, uuid, "s1-real-lost-ack-key-0001")
		marker := "lost-ack-" + r.ID
		effects := 0
		w := state.Worker{Store: s}
		w.Execute(r.ID, func(context.Context) (string, error) {
			effects++
			if _, err := s1NativeWrite(project, name, uuid, marker); err != nil {
				return "", err
			}
			return "", errors.New("intentional loss of an otherwise committed native reply")
		}, func(context.Context) error { return s1NativeObserved(project, name, uuid, marker) })
		got, err := s.Get(context.Background(), r.ID)
		if err != nil || got.State != "UNKNOWN" || effects != 1 {
			t.Fatalf("false known result: %+v %v", got, err)
		}
		if err := s1NativeObserved(project, name, uuid, marker); err != nil {
			t.Fatal(err)
		}
		err = w.Submit(r.ID, func(context.Context) (string, error) { effects++; return "", nil }, func(context.Context) error { return nil })
		if !errors.Is(err, state.ErrState) || effects != 1 {
			t.Fatalf("blind replay after lost ACK: %v effects=%d", err, effects)
		}
		t.Log("PASS real committed Incus side effect + ambiguous reply stayed UNKNOWN without dispatch replay")
	})
	t.Run("crash-after-native-effect-before-durable-ack", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Chmod(dir, 0700); err != nil {
			t.Fatal(err)
		}
		db, receiptPath := filepath.Join(dir, "journal.sqlite"), filepath.Join(dir, "original.json")
		marker := "after-native-before-ledger-" + uuid
		cmd := exec.Command(os.Args[0], "-test.run=^TestS1IncusNativeJournalRecovery$")
		cmd.Env = append(os.Environ(), "BW_S1_NATIVE_CRASH_CHILD=1",
			"BW_S1_CHILD_DB="+db, "BW_S1_CHILD_RECEIPT="+receiptPath, "BW_S1_CHILD_MARKER="+marker)
		if output, err := cmd.CombinedOutput(); err == nil || !strings.Contains(err.Error(), "75") {
			t.Fatalf("expected explicit simulated process crash code75: %v output=%s", err, output)
		}
		b, err := os.ReadFile(receiptPath)
		if err != nil {
			t.Fatal(err)
		}
		var original state.Receipt
		if err = json.Unmarshal(b, &original); err != nil {
			t.Fatal(err)
		}
		s, err := state.Open(db)
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		n, err := s.Recover(context.Background())
		if err != nil || n != 1 {
			t.Fatalf("recover after native effect: %d %v", n, err)
		}
		got, err := s.Get(context.Background(), original.ID)
		if err != nil || got.State != "UNKNOWN" || got.TrackingID != original.TrackingID {
			t.Fatalf("loss of original receipt or false native result: %+v %v", got, err)
		}
		if err := s1NativeObserved(project, name, uuid, marker); err != nil {
			t.Fatal(err)
		}
		effects := 0
		w := state.Worker{Store: s}
		err = w.Submit(original.ID, func(context.Context) (string, error) { effects++; return "", nil }, func(context.Context) error { return nil })
		if !errors.Is(err, state.ErrState) || effects != 0 {
			t.Fatalf("native redispatch after restart: %v, %d", err, effects)
		}
		t.Log("PASS independent Incus side effect persisted across real OS exit; ledger preserves UNKNOWN")
	})
}
