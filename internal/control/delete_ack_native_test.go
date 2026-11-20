// Disposable native acceptance only; no production fault switches.
package control

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

type b5AckCut struct{ Path, Project, Name, UUID, Original, Cut string }

func b5AckFixtureAllowed() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true" && os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") == "s4b4-disposable-only-20261119" && os.Getuid() != 0
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
	if f.Project != "user-"+strconv.Itoa(os.Getuid()) || f.Name != "s4b5ack" || (f.Cut != "before-ack" && f.Cut != "after-ack") {
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
	if f.Cut == "after-ack" {
		if err = s.NativeAck(ctx, f.Original, id); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("S4_B5_FINAL_NATIVE_DELETE_OBSERVED original=%s native=%s cut=%s\n", f.Original, id, f.Cut)
	// Abrupt process exit, without deferred SQLite close/final commit.
	os.Exit(73)
}

func TestB5NativeDeleteAckLoss(t *testing.T) {
	if !b5AckFixtureAllowed() {
		t.Skip("one-use fixture only")
	}
	cut := os.Getenv("BW_B5_ACK_CUT")
	if cut != "before-ack" && cut != "after-ack" {
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
	if n, e := s.Recover(ctx); e != nil || n != 1 {
		t.Fatalf("recovery %d %v", n, e)
	}
	original, err := s.Get(ctx, parent.ID)
	if err != nil || original.State != "UNKNOWN" || (original.NativeOperationID == "") != (cut == "before-ack") {
		t.Fatalf("original %+v %v", original, err)
	}
	if err = r.ConfirmDeleteAbsence(ctx, "s4b5ack"); err != nil {
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
	fmt.Printf("S4_B5_FINAL_DELETE_ACK_RECOVER_UNKNOWN_NO_REPLAY=PASS cut=%s\n", cut)
}
