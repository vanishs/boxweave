package install

import (
	"context"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/state"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Test-only exact completed-original fault fixture. Only fixed owned test
// names in a disposable installation, never linked into production binary.
func TestPR177D4InstalledCompletedOriginal(t *testing.T) {
	if os.Getenv("BOXWEAVE_PR177_D4_LAB") != "disposable-installed-v1" {
		t.Skip("dedicated installed lab required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	project := os.Getenv("D4_TEST_PROJECT")
	mode := os.Getenv("D4_TEST_MODE")
	if !state.SafeDiagnosticProject(project) || mode != "LEGACY" && mode != "SCOPED_V2" && mode != "SCOPED_POWER" {
		t.Fatal("invalid fixture scope")
	}
	if mode == "SCOPED_POWER" && (os.Getenv("GITHUB_ACTIONS") != "true" ||
		os.Getenv("RUNNER_ENVIRONMENT") != "github-hosted" || os.Getenv("D4_TEST_PHASE") != "stage") {
		t.Fatal("dedicated hosted Scoped power fixture required")
	}
	if exec.Command("systemctl", "is-active", "--quiet", "boxweave.service").Run() == nil {
		t.Fatal("competing controller active")
	}
	switch os.Getenv("D4_TEST_PHASE") {
	case "stage":
		if os.Geteuid() == 0 {
			t.Fatal("restricted actor required")
		}
		db, e := state.OpenExistingRecovery(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		l, e := db.ReadNativeNetworkLedger(ctx, project)
		if e != nil {
			t.Fatal(e)
		}
		var a, b state.NativeNetworkBox
		for _, x := range l.Boxes {
			if x.Name == "d4-a" {
				a = x
			}
			if x.Name == "d4-b" {
				b = x
			}
		}
		if a.UUID == "" || !a.OriginalProof || mode == "SCOPED_V2" && (b.UUID == "" || !b.OriginalProof) || mode == "SCOPED_POWER" && len(l.Boxes) != 1 {
			t.Fatal("missing fixed owned fixtures")
		}
		_, tok, e := db.IssueToken(ctx, time.Hour)
		if e != nil {
			t.Fatal(e)
		}
		adm := state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "pr177-d4-completed-original-fixture", RequestID: "d4-completed-original", WorkRef: "pr177-d4-completed-original"}
		var id string
		if mode == "SCOPED_V2" {
			rec, e := db.AdmitScopedPeerSet(ctx, adm, a.UUID, b.UUID, true)
			if e != nil || rec.Duplicate {
				t.Fatal(rec, e)
			}
			id = rec.ID
			if e = db.StageAndClaimScopedPeerOriginal(ctx, id); e != nil {
				t.Fatal(e)
			}
		} else {
			adm.Action = "box_stop"
			adm.ResourceKey = "box:" + a.Name
			adm.BoxUUID = a.UUID
			adm.Incarnation = a.Incarnation
			adm.SafeIntent = []byte(`{"box_id":"` + a.UUID + `"}`)
			// The installed Scoped power witness records the actual native
			// ACK and deliberately omits only final business settlement.
			rec, e := db.AdmitScoped(ctx, adm)
			if e != nil || rec.Duplicate {
				t.Fatal(rec, e)
			}
			id = rec.ID
			if e = db.Claim(ctx, id); e != nil {
				t.Fatal(e)
			}
			n := native.Reader{Project: project, Socket: "/var/lib/incus/unix.socket.user"}
			ack, e := n.SetPower(ctx, a.Name, a.UUID, "stop")
			if e != nil {
				t.Fatal(e)
			}
			if e = db.NativeAck(ctx, id, ack); e != nil {
				t.Fatal(e)
			}
			for {
				op, e := n.NativeOperation(ctx, ack)
				if e != nil {
					t.Fatal(e)
				}
				if op.StatusCode.IsFinal() {
					if op.StatusCode != api.Success {
						t.Fatal("native stop not successful")
					}
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(100 * time.Millisecond):
				}
			}
		}
		t.Log("PR177_D4_COMPLETED_ORIGINAL=" + id)
	case "apply-native":
		if os.Geteuid() != 0 || mode != "SCOPED_V2" || exec.Command("systemctl", "is-active", "--quiet", "boxweave-scoped-lifecycle.service").Run() == nil {
			t.Fatal("isolated Root with stopped helper required")
		}
		id := os.Getenv("D4_TEST_ORIGINAL")
		db, e := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		proof, e := db.ReadRootScopedPeerOriginal(ctx, id, project, "incusbr-"+project[5:])
		if e != nil || !proof.Intent.Allow {
			t.Fatal("wrong original", e)
		}
		found := 0
		for _, b := range proof.Inventory.Boxes {
			if b.UUID == proof.Intent.SourceUUID && b.Name == "d4-a" {
				found++
			}
			if b.UUID == proof.Intent.TargetUUID && b.Name == "d4-b" {
				found++
			}
		}
		if found != 2 {
			t.Fatal("wrong dedicated pair")
		}
		// Actual native journal, no fabricated receipt; deliberately omit
		// SQLite finalization to simulate lost completion acknowledgement.
		rec, e := nativeScopedPeerApply(ctx, db, id)
		if e != nil || rec.Status != "COMPLETE_VERIFIED" || rec.RecoveryOnly {
			t.Fatal(rec, e)
		}
		t.Log("PR177_D4_NATIVE_COMPLETED_ACK_LOST=PASS")
	default:
		t.Fatal("invalid fixture phase")
	}
}
