package control

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
)

// A separate worker process is killed after a real native terminal outcome,
// either before the SQLite native ACK or after it but before post-read/finalize.
// All fault fixtures are explicitly limited to the one-use CI host.
type b4CrashFixture struct {
	Path, Project, UUID, Original, Cut string
	Spec                               native.ForwardSpec
	Create                             bool
}

func TestB4ForwardCrashChild(t *testing.T) {
	raw := os.Getenv("BW_B4_CRASH_CHILD")
	if raw == "" {
		t.Skip("worker subprocess only")
	}
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") != "s4b4-disposable-only-20261119" {
		t.Fatal("not disposable")
	}
	var f b4CrashFixture
	if e := json.Unmarshal([]byte(raw), &f); e != nil {
		t.Fatal(e)
	}
	if f.Project != "user-"+strconv.Itoa(os.Getuid()) {
		t.Fatal("incorrect project")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, e := state.Open(f.Path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Claim(ctx, f.Original); e != nil {
		t.Fatal(e)
	}
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: f.Project}
	nativeID, e := r.UpdateOwnedForward(ctx, "s4b4native", f.UUID, f.Spec, f.Create)
	if e != nil {
		t.Fatal(e)
	}
	if e = nativeDone(ctx, r, nativeID); e != nil {
		t.Fatal(e)
	}
	if f.Cut == "after-ack" {
		if e = s.NativeAck(ctx, f.Original, nativeID); e != nil {
			t.Fatal(e)
		}
	}
	// Independent observation validates that the native effect really happened;
	// never report a fake ACK, and do not commit post-read to the ledger.
	if e = r.ObserveOwnedForward(ctx, "s4b4native", f.UUID, f.Spec, f.Create); e != nil {
		t.Fatal(e)
	}
	fmt.Printf("S4_B4_CRASH_CHILD_NATIVE_TERMINAL id=%s cut=%s create=%v\n", nativeID, f.Cut, f.Create)
	if e = syscall.Kill(os.Getpid(), syscall.SIGKILL); e != nil {
		t.Fatal(e)
	}
	t.Fatal("SIGKILL did not terminate worker")
}

func TestB4NativeOriginalCrashRecovery(t *testing.T) {
	if os.Getenv("GITHUB_ACTIONS") != "true" || os.Getenv("BOXWEAVE_B4_MCP_ONE_USE") != "s4b4-disposable-only-20261119" {
		t.Skip("disposable native only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	project := "user-" + strconv.Itoa(os.Getuid())
	r := native.Reader{Socket: "/var/lib/incus/unix.socket.user", Project: project}
	c, e := incus.ConnectIncusUnixWithContext(ctx, r.Socket, nil)
	if e != nil {
		t.Fatal(e)
	}
	inst, _, e := c.UseProject(project).GetInstance("s4b4native")
	if e != nil {
		t.Fatal(e)
	}
	uuid := inst.Config["volatile.uuid"]
	if uuid == "" {
		t.Fatal("missing uuid")
	}
	for _, tc := range []struct {
		name, cut string
		create    bool
		port      int
	}{
		{"CREATE_ACK_LOST", "before-ack", true, 23101},
		{"DELETE_ACK_LOST", "before-ack", false, 23102},
		{"CREATE_POSTREAD_INTERRUPTED", "after-ack", true, 23103},
		{"DELETE_POSTREAD_INTERRUPTED", "after-ack", false, 23104},
	} {
		t.Run(tc.name, func(t *testing.T) {
			private := filepath.Join(t.TempDir(), "private")
			if e := os.Mkdir(private, 0700); e != nil {
				t.Fatal(e)
			}
			if e := os.Chmod(private, 0700); e != nil {
				t.Fatal(e)
			}
			path := filepath.Join(private, "b4.sqlite")
			s, e := state.Open(path)
			if e != nil {
				t.Fatal(e)
			}
			defer func() { _ = s.Close() }()
			fixture, e := sql.Open("sqlite3", path)
			if e != nil {
				t.Fatal(e)
			}
			_, e = fixture.ExecContext(ctx, "INSERT INTO owned_boxes(uuid,project,name,incarnation,state,created_at) VALUES(?,?,?,?,?,?)", uuid, project, "s4b4native", uuid, "OWNED", time.Now().UnixNano())
			_ = fixture.Close()
			if e != nil {
				t.Fatal(e)
			}
			_, tok, e := s.IssueToken(ctx, time.Hour)
			if e != nil {
				t.Fatal(e)
			}
			a := state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "b4-fault-create-" + tc.name, Action: "port_forward_create", RequestID: "create-" + tc.name, BoxUUID: uuid, Incarnation: uuid, ResourceKey: "box:s4b4native", ResourceKeys: []string{"box:" + uuid, forwardListenerKey("tcp", "127.0.0.1", tc.port)}, SafeIntent: []byte("native-fault-create-" + tc.name), ForwardCreate: &state.ForwardCreateAdmission{Protocol: "tcp", HostBind: "127.0.0.1", HostPort: tc.port, GuestPort: 19091}}
			creation, e := s.Admit(ctx, a)
			if e != nil {
				t.Fatal(e)
			}
			claim, e := s.GetForward(ctx, project, creation.ID)
			if e != nil {
				t.Fatal(e)
			}
			spec := specForClaim(claim)
			original := creation
			if !tc.create {
				// Initial forward is fully verified through exactly the real Worker.
				var nativeID string
				state.Worker{Store: s}.Execute(creation.ID, func(ctx context.Context) (string, error) {
					var e error
					nativeID, e = r.UpdateOwnedForward(ctx, "s4b4native", uuid, spec, true)
					return nativeID, e
				}, func(ctx context.Context) error {
					if e := nativeDone(ctx, r, nativeID); e != nil {
						return e
					}
					if e := r.ObserveOwnedForward(ctx, "s4b4native", uuid, spec, true); e != nil {
						return e
					}
					if e := s.FinalizeVerifiedForward(ctx, creation.ID, true); e != nil {
						return e
					}
					return state.ErrVerifiedForwardCommitted
				})
				init, e := s.Get(ctx, creation.ID)
				if e != nil || init.State != "SUCCEEDED" {
					t.Fatalf("initial native creation %+v %v", init, e)
				}
				b4MCPHostEcho(t, "tcp", "127.0.0.1", tc.port, true)
				a.Action = "port_forward_delete"
				a.Key = "b4-fault-delete-" + tc.name
				a.RequestID = "delete-" + tc.name
				a.ForwardCreate = nil
				a.ForwardDeleteID = creation.ID
				a.SafeIntent = []byte("native-fault-delete-" + tc.name)
				original, e = s.Admit(ctx, a)
				if e != nil {
					t.Fatal(e)
				}
			}
			if e = s.Close(); e != nil {
				t.Fatal(e)
			}
			wire, e := json.Marshal(b4CrashFixture{Path: path, Project: project, UUID: uuid, Original: original.ID, Cut: tc.cut, Spec: spec, Create: tc.create})
			if e != nil {
				t.Fatal(e)
			}
			process := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestB4ForwardCrashChild$", "-test.v")
			process.Env = append(os.Environ(), "BW_B4_CRASH_CHILD="+string(wire))
			output, e := process.CombinedOutput()
			var fatal *exec.ExitError
			if !errors.As(e, &fatal) {
				t.Fatalf("worker not killed: %v %s", e, output)
			}
			status, ok := fatal.Sys().(syscall.WaitStatus)
			if !ok || !status.Signaled() || status.Signal() != syscall.SIGKILL {
				t.Fatalf("expected real SIGKILL; %v %s", e, output)
			}
			t.Logf("%s", output)
			// Same production startup recovery; reopen private DB after process death.
			s, e = state.Open(path)
			if e != nil {
				t.Fatal(e)
			}
			if n, e := s.Recover(ctx); e != nil || n != 1 {
				t.Fatalf("startup recovery %d %v", n, e)
			}
			got, e := s.Get(ctx, original.ID)
			if e != nil || got.State != "UNKNOWN" || (tc.cut == "before-ack" && got.NativeOperationID != "") || (tc.cut == "after-ack" && got.NativeOperationID == "") {
				t.Fatalf("misclassified native uncertainty: %+v %v", got, e)
			}
			pending, e := s.GetForward(ctx, project, creation.ID)
			if e != nil || pending.State != "PENDING" || pending.OperationID != original.ID {
				t.Fatalf("lost pending native claim %+v %v", pending, e)
			}
			replay, e := s.Replay(ctx, a)
			if e != nil || !replay.Duplicate || replay.ID != original.ID || replay.State != "UNKNOWN" {
				t.Fatalf("lost original replay %+v %v", replay, e)
			}
			a.Key = "b4-fault-fresh-" + tc.name
			a.RequestID = "new-" + tc.name
			if _, e = s.Admit(ctx, a); !errors.Is(e, state.ErrResourceLocked) {
				t.Fatalf("unsafe fresh dispatch after restart: %v", e)
			}
			if e = r.ObserveOwnedForward(ctx, "s4b4native", uuid, spec, tc.create); e != nil {
				t.Fatal(e)
			}
			b4MCPHostEcho(t, "tcp", "127.0.0.1", tc.port, tc.create)
			// Trusted one-use fixture teardown only. NEVER a production UNKNOWN retry.
			if tc.create {
				id, e := r.UpdateOwnedForward(ctx, "s4b4native", uuid, spec, false)
				if e != nil {
					t.Fatal(e)
				}
				if e = nativeDone(ctx, r, id); e != nil {
					t.Fatal(e)
				}
			}
			if e = r.ObserveOwnedForward(ctx, "s4b4native", uuid, spec, false); e != nil {
				t.Fatal(e)
			}
			b4MCPHostEcho(t, "tcp", "127.0.0.1", tc.port, false)
			t.Logf("S4_B4_NATIVE_%s_RESTART_UNKNOWN_NO_REPLAY_PACKET=PASS", tc.name)
		})
	}
	fmt.Println("S4_B4_NATIVE_REAL_PROCESS_KILL_ACK_AND_POSTREAD_MATRIX=PASS")
}
