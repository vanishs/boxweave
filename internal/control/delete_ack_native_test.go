// Disposable native acceptance only; no production fault switches.
package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/lxc/incus/v7/shared/api"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type b5AckCut struct{ Path, Project, Name, UUID, Original, Cut string }

func b5AckFixtureAllowed() bool {
	if os.Getuid() == 0 {
		return false
	}
	if os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") == "s4b4-disposable-only-20261119" {
		return true
	}
	// Explicit local disposable Box fixture. The marker is root-owned, and
	// binds the one exact restricted Project and fixture UUID; no CI impersonation.
	if os.Getenv("BOXWEAVE_B5_BOX_LAB") != "b5-disposable-box-v1" {
		return false
	}
	info, e := os.Lstat("/run/boxweave-b5-disposable-lab")
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0022 != 0 {
		return false
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok || st.Uid != 0 {
		return false
	}
	raw, e := os.ReadFile("/run/boxweave-b5-disposable-lab")
	if e != nil {
		return false
	}
	lines := strings.Fields(string(raw))
	return len(lines) == 2 && lines[0] == "user-"+strconv.Itoa(os.Getuid()) && len(lines[1]) == 36
}

func b5CheckLabUUID(t *testing.T, project, uuid string) {
	t.Helper()
	if os.Getenv("BOXWEAVE_B5_BOX_LAB") == "" {
		return
	}
	b, e := os.ReadFile("/run/boxweave-b5-disposable-lab")
	if e != nil || strings.TrimSpace(string(b)) != project+" "+uuid {
		t.Fatal("one-use lab UUID mismatch")
	}
}

func TestB5DeleteAckChild(t *testing.T) {
	wire := os.Getenv("BW_B5_DELETE_CUT")
	if wire == "" {
		t.Skip("subprocess fixture only")
	}
	if !b5AckFixtureAllowed() {
		t.Fatal("not one-use fixture")
	}
	var f b5AckCut
	if err := json.Unmarshal([]byte(wire), &f); err != nil {
		t.Fatal(err)
	}
	if f.Project != "user-"+strconv.Itoa(os.Getuid()) || f.Name != "s4b5ack" || (f.Cut != "before-ack" && f.Cut != "after-ack" && f.Cut != "receipt-unavailable" && f.Cut != "protected-partial") {
		t.Fatal("invalid fixture identity")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s, err := state.Open(f.Path)
	if err != nil {
		t.Fatal(err)
	}
	box, err := s.OwnedBox(ctx, f.Project, f.UUID)
	if err != nil || box.Name != f.Name {
		t.Fatal("wrong fixture ownership", err)
	}
	b5CheckLabUUID(t, f.Project, f.UUID)
	if f.Cut == "protected-partial" {
		svc := Service{Store: s, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: f.Project}}
		c, e := incus.ConnectIncusUnixWithContext(ctx, svc.Native.Socket, nil)
		if e != nil {
			t.Fatal(e)
		}
		scope := c.UseProject(f.Project)
		var finalNativeID string
		state.Worker{Store: s}.Execute(f.Original, func(ctx context.Context) (string, error) {
			n, e := svc.executeDependentDeleteWithWitness(ctx, f.Original, box, func(ctx context.Context) error {
				current, etag, e := scope.GetInstance(f.Name)
				if e != nil {
					return e
				}
				if current.Config["volatile.uuid"] != f.UUID {
					return native.ErrNotOwned
				}
				put := current.Writable()
				put.Config["security.protection.delete"] = "true"
				op, e := scope.UpdateInstance(f.Name, put, etag)
				if e != nil {
					return e
				}
				if e = nativeDone(ctx, svc.Native, op.Get().ID); e != nil {
					return e
				}
				fmt.Println("S4_B5_PARTIAL_PROTECTION_AFTER_CHILDREN=PASS")
				return nil
			})
			finalNativeID = n
			return n, e
		}, func(ctx context.Context) error {
			op, e := svc.Native.ObserveUntilFinal(ctx, finalNativeID)
			if e != nil || op.StatusCode == api.Success {
				t.Fatalf("expected actual native terminal refusal: %+v %v", op, e)
			}
			fmt.Println("S4_B5_ACTUAL_FINAL_NATIVE_DELETE_FAILED=PASS")
			return native.ErrUnverified
		})
		rec, e := s.Get(ctx, f.Original)
		if e != nil || rec.State != "UNKNOWN" {
			t.Fatalf("partial final refusal misclassified %+v %v", rec, e)
		}
		os.Exit(73)
	}
	if err = s.Claim(ctx, f.Original); err != nil {
		t.Fatal(err)
	}
	svc := Service{Store: s, Native: native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: f.Project}}
	id, err := svc.executeDependentDelete(ctx, f.Original, box)
	if err != nil {
		t.Fatal(err)
	}
	if err = nativeDone(ctx, svc.Native, id); err != nil {
		t.Fatal(err)
	}
	if err = svc.Native.ConfirmDeleteAbsence(ctx, f.Name); err != nil {
		t.Fatal(err)
	}
	if f.Cut != "before-ack" {
		if err = s.NativeAck(ctx, f.Original, id); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("S4_B5_FINAL_NATIVE_DELETE_OBSERVED original=%s native=%s cut=%s\n", f.Original, id, f.Cut)
	if f.Cut == "receipt-unavailable" {
		deadline := time.Now().Add(45 * time.Second)
		for {
			_, e := svc.Native.NativeOperation(ctx, id)
			if api.StatusErrorCheck(e, http.StatusNotFound) {
				break
			}
			if e != nil || time.Now().After(deadline) {
				t.Fatalf("native receipt did not become unavailable: %v", e)
			}
			time.Sleep(400 * time.Millisecond)
		}
	}
	// Abrupt process exit, without deferred SQLite close/final commit.
	os.Exit(73)
}

func TestB5NativeDeleteAckLoss(t *testing.T) {
	if !b5AckFixtureAllowed() {
		t.Skip("one-use fixture only")
	}
	cut := os.Getenv("BW_B5_ACK_CUT")
	if cut != "before-ack" && cut != "after-ack" && cut != "receipt-unavailable" && cut != "protected-partial" {
		t.Fatal("exact cut required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	project := "user-" + strconv.Itoa(os.Getuid())
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, err := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if err != nil {
		t.Fatal(err)
	}
	inst, _, err := c.UseProject(project).GetInstance("s4b5ack")
	if err != nil {
		t.Fatal(err)
	}
	uuid := inst.Config["volatile.uuid"]
	b5CheckLabUUID(t, project, uuid)
	if uuid == "" || inst.Status != "Running" {
		t.Fatal("incorrect native fixture")
	}
	private := filepath.Join(t.TempDir(), "private")
	if err = os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(private, "b5.sqlite")
	s, err := state.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RegisterOwnedBox(ctx, uuid, project, "s4b5ack", uuid); err != nil {
		t.Fatal(err)
	}
	_, tok, err := s.IssueToken(ctx, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	a := state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "b5-ack-fixture-forward-001", Action: "port_forward_create", RequestID: "b5-forward", BoxUUID: uuid, Incarnation: uuid, ResourceKey: "box:s4b5ack", ResourceKeys: []string{"box:" + uuid, forwardListenerKey("tcp", "127.0.0.1", 24118)}, SafeIntent: []byte("b5-ack-fwd"), ForwardCreate: &state.ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: 24118, GuestPort: 19091}}
	f, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := s.GetForward(ctx, project, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	spec := specForClaim(claim)
	var nativeID string
	state.Worker{Store: s}.Execute(f.ID, func(c context.Context) (string, error) {
		var e error
		nativeID, e = r.UpdateOwnedForward(c, "s4b5ack", uuid, spec, true)
		return nativeID, e
	}, func(c context.Context) error {
		if e := nativeDone(c, r, nativeID); e != nil {
			return e
		}
		if e := r.ObserveOwnedForward(c, "s4b5ack", uuid, spec, true); e != nil {
			return e
		}
		if e := s.FinalizeVerifiedForward(c, f.ID, true); e != nil {
			return e
		}
		return state.ErrVerifiedForwardCommitted
	})
	initial, err := s.Get(ctx, f.ID)
	if err != nil || initial.State != "SUCCEEDED" {
		t.Fatalf("forward not confirmed %+v %v", initial, err)
	}
	b4MCPHostEcho(t, "tcp", "127.0.0.1", 24118, true)
	a.Action = "box_delete"
	a.Key = "b5-ack-fixture-parent-002"
	a.RequestID = "b5-parent"
	a.ResourceKeys = nil
	a.SafeIntent = []byte("b5-ack-parent")
	a.ForwardCreate = nil
	a.CleanupDependencies = true
	parent, err := s.Admit(ctx, a)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(b5AckCut{Path: p, Project: project, Name: "s4b5ack", UUID: uuid, Original: parent.ID, Cut: cut})
	if err != nil {
		t.Fatal(err)
	}
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestB5DeleteAckChild$", "-test.v")
	child.Env = append(os.Environ(), "BW_B5_DELETE_CUT="+string(wire))
	output, err := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 73 {
		t.Fatalf("fault cut not reached: %v %s", err, output)
	}
	t.Logf("%s", output)
	s, err = state.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	wantRecovered := int64(1)
	if cut == "protected-partial" {
		wantRecovered = 0
	}
	if n, e := s.Recover(ctx); e != nil || n != wantRecovered {
		t.Fatalf("recovery %d %v", n, e)
	}
	original, err := s.Get(ctx, parent.ID)
	if err != nil || original.State != "UNKNOWN" || (original.NativeOperationID == "") != (cut == "before-ack") {
		t.Fatalf("original %+v %v", original, err)
	}
	if cut == "protected-partial" {
		current, _, e := c.UseProject(project).GetInstance("s4b5ack")
		if e != nil || current.Config["volatile.uuid"] != uuid || current.Status != "Stopped" || current.Config["security.protection.delete"] != "true" {
			t.Fatalf("partial native poststate wrong %v", e)
		}
	} else if err = r.ConfirmDeleteAbsence(ctx, "s4b5ack"); err != nil {
		t.Fatal(err)
	}
	b4MCPHostEcho(t, "tcp", "127.0.0.1", 24118, false)
	claim, err = s.GetForward(ctx, project, f.ID)
	if err != nil || claim.State != "REVOKED" {
		t.Fatalf("child not retained %+v %v", claim, err)
	}
	if _, err = s.OwnedBox(ctx, project, uuid); err != nil {
		t.Fatal("uncertain original must not manufacture tombstone", err)
	}
	replay, err := s.Replay(ctx, a)
	if err != nil || replay.ID != parent.ID || !replay.Duplicate || replay.State != "UNKNOWN" {
		t.Fatalf("lost replay %+v %v", replay, err)
	}
	a.Key = "b5-ack-fixture-fresh-003"
	a.RequestID = "b5-fresh"
	if _, err = s.Admit(ctx, a); !errors.Is(err, state.ErrResourceLocked) {
		t.Fatal("fresh mutation not fenced", err)
	}
	if cut == "protected-partial" {
		if !strings.Contains(string(output), "S4_B5_ACTUAL_FINAL_NATIVE_DELETE_FAILED=PASS") {
			t.Fatal("missing native terminal refusal witness")
		}
		fmt.Println("S4_B5_REAL_PARTIAL_CLEANUP_FINAL_DELETE_REFUSED_UNKNOWN=PASS")
		return
	}
	fmt.Printf("S4_B5_FINAL_DELETE_ACK_RECOVER_UNKNOWN_NO_REPLAY=PASS cut=%s\n", cut)
	snapshot, e := s.RecoveryInspect(ctx, parent.ID)
	if e != nil {
		t.Fatal(e)
	}
	obs, e := r.ObserveUnknownDelete(ctx, snapshot.Name, snapshot.BoxUUID, snapshot.NativeOperationID)
	if e != nil {
		t.Fatal(e)
	}
	switch cut {
	case "before-ack":
		if obs.Class != "NATIVE_ACK_MISSING" {
			t.Fatal(obs)
		}
		fmt.Println("S4_B5_MISSING_ACK_NOT_INFERRED_SUCCESS=PASS")
	case "receipt-unavailable":
		if obs.Class != "NATIVE_RECEIPT_UNAVAILABLE" {
			t.Fatal(obs)
		}
		fmt.Println("S4_B5_EXPIRED_OR_MISSING_NATIVE_RECEIPT_UNKNOWN=PASS")
	case "after-ack":
		if obs.Class != "ORIGINAL_TERMINAL_DELETE_VERIFIED" {
			t.Fatal(obs)
		}
		challenge, e := s.IssueRecoveryChallenge(ctx, snapshot, "b5-native-review", obs.Fingerprint)
		if e != nil {
			t.Fatal(e)
		}
		if e = s.FinalizeReviewedUnknownDelete(ctx, snapshot, "wrong-review", challenge, uuid, "native receipt and exact absence reviewed", obs.Fingerprint, true); e == nil {
			t.Fatal("wrong reviewer settled")
		}
		second, e := r.ObserveUnknownDelete(ctx, snapshot.Name, snapshot.BoxUUID, snapshot.NativeOperationID)
		if e != nil || second.Class != "ORIGINAL_TERMINAL_DELETE_VERIFIED" || second.Fingerprint != obs.Fingerprint {
			t.Fatalf("second exact proof %+v %v", second, e)
		}
		if e = s.FinalizeReviewedUnknownDelete(ctx, snapshot, "b5-native-review", challenge, uuid, "native receipt and exact absence reviewed", second.Fingerprint, true); e != nil {
			t.Fatal(e)
		}
		settled, e := s.Get(ctx, parent.ID)
		if e != nil || settled.State != "SUCCEEDED" {
			t.Fatalf("settled %+v %v", settled, e)
		}
		if _, e = s.OwnedBox(ctx, project, uuid); !errors.Is(e, state.ErrNotFound) {
			t.Fatal("missing settled tombstone", e)
		}
		if e = s.FinalizeReviewedUnknownDelete(ctx, snapshot, "b5-native-review", challenge, uuid, "native receipt and exact absence reviewed", second.Fingerprint, true); e == nil {
			t.Fatal("reused challenge settled")
		}
		fmt.Println("S4_B5_KNOWN_TERMINAL_DELETE_REVIEWED_ATOMIC_SETTLEMENT=PASS")
	}
}
