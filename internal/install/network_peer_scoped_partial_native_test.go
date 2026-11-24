package install

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

// Test-only fault adapter: real exact ACL CAS then intentionally lose
// the acknowledgement. It is never linked into the installed executable.
type pr178N4LostACL struct {
	nativePeerScopedIncusACLDriver
	calls int
}

func (d *pr178N4LostACL) casACL(name string, put api.NetworkACLPut, tag string) error {
	d.calls++
	if e := d.nativePeerScopedIncusACLDriver.casACL(name, put, tag); e != nil {
		return e
	}
	return errors.New("fixture: applied native ACL, acknowledgement lost")
}

// Opt-in only on a disposable isolated installation, after normal N4 tests.
// Two processes preserve the ordinary service-user write and Root read-only
// ledger boundary. The fixed test pair is n4-a -> n4-b, no generic RPC.
func TestPR178N4InstalledPartialACL(t *testing.T) {
	if os.Getenv("BOXWEAVE_PR178_N4_PARTIAL_LAB") != "disposable-scoped-v2" {
		t.Skip("requires disposable installed lab")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	project := "user-1001"
	if os.Getenv("N4_TEST_PROJECT") != "" {
		project = os.Getenv("N4_TEST_PROJECT")
	}
	switch os.Getenv("N4_TEST_PHASE") {
	case "stage":
		if os.Geteuid() == 0 {
			t.Fatal("stage must run as restricted service actor")
		}
		db, e := state.Open(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		l, e := db.ReadNativeNetworkLedger(ctx, project)
		if e != nil {
			t.Fatal(e)
		}
		var a, b string
		for _, x := range l.Boxes {
			switch x.Name {
			case "n4-a":
				a = x.UUID
			case "n4-b":
				b = x.UUID
			}
		}
		if a == "" || b == "" {
			t.Fatal("missing dedicated fixtures")
		}
		_, tok, e := db.IssueToken(ctx, time.Hour)
		if e != nil {
			t.Fatal(e)
		}
		r, e := db.AdmitScopedPeerSet(ctx, state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "n4-fixture-partial-acl-ack-loss", RequestID: "n4-fixture-partial-acl", WorkRef: "pr178-n4-partial"}, a, b, true)
		if e != nil || r.Duplicate {
			t.Fatal("fixture admission refused", e)
		}
		if e = db.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		t.Log("PR178_N4_PARTIAL_ORIGINAL=" + r.ID)
	case "apply":
		if os.Geteuid() != 0 || exec.Command("systemctl", "is-active", "--quiet", "boxweave-scoped-lifecycle.service").Run() == nil {
			t.Fatal("requires isolated Root and stopped competing helper")
		}
		id := os.Getenv("N4_TEST_ORIGINAL")
		if !scopedLifecycleID.MatchString(id) {
			t.Fatal("invalid original")
		}
		db, e := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		r, initial, e := nativePeerFreshVerifiedPlan()
		if e != nil {
			t.Fatal(e)
		}
		if r.Project != project {
			t.Fatal("project mismatch")
		}
		if e = nativePeerFreshQuarantineReceipt(r); e != nil {
			t.Fatal(e)
		}
		current, _, e := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, r, initial, "")
		if e != nil {
			t.Fatal(e)
		}
		c, e := prepareClient(ctx)
		if e != nil {
			t.Fatal(e)
		}
		c = c.UseProject("default")
		read := func(ctx context.Context) (state.ScopedPeerOriginalProof, error) {
			return db.ReadRootScopedPeerOriginal(ctx, id, r.Project, r.Bridge)
		}
		proof, e := read(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if !proof.Intent.Allow {
			t.Fatal("fixture must be grant")
		}
		for _, b := range proof.Inventory.Boxes {
			if b.UUID == proof.Intent.SourceUUID && b.Name != "n4-a" || b.UUID == proof.Intent.TargetUUID && b.Name != "n4-b" {
				t.Fatal("wrong fixture pair")
			}
		}
		base, e := nativePeerScopedRootReadSnapshot(ctx, nativePeerScopedIncusReader{root: c}, proof)
		if e != nil {
			t.Fatal(e)
		}
		p, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
		if e != nil || p.GuardBefore != current {
			t.Fatal("wrong policy preimage", e)
		}
		d := &pr178N4LostACL{nativePeerScopedIncusACLDriver: nativePeerScopedIncusACLDriver{root: c}}
		rec, e := nativeScopedPeerExecute(ctx, nativePeerFreshJournalDir, 0, r.HostSHA256, read, nativePeerScopedIncusReader{root: c}, d, "/sys/class/net", NativePeerGuardKernelReadback, nativePeerGuardFixedNftBatch, nativePeerConntrackOS,
			func(ctx context.Context, p state.ScopedPeerOriginalProof, s nativePeerScopedSDKSnapshot) (string, error) {
				return nativeScopedPeerGlobal(ctx, c, db, r, p, s)
			})
		if e == nil || rec.Status == "COMPLETE_VERIFIED" || d.calls != 1 {
			t.Fatal("partial fault not observed", rec, e, d.calls)
		}
		for _, b := range proof.Inventory.Boxes {
			want := p.ExpectedBeforeByUUID[b.UUID]
			if b.UUID == proof.Intent.SourceUUID {
				want = p.ExpectedAfterByUUID[b.UUID]
			}
			got, _, e := d.getACL(b.ACL)
			if e != nil || !nativePeerScopedACLFullMatch(got, want, base.Evidence.Observed[b.Name].ACL.UsedBy[0]) {
				t.Fatal("unexpected ACL effect", b.Name, e)
			}
		}
		if e = NativePeerGuardKernelReadback(ctx, current); e != nil {
			t.Fatal("guard not retained", e)
		}
		dir := filepath.Join(nativePeerFreshJournalDir, "peer-"+id)
		files, e := os.ReadDir(dir)
		if e != nil || len(files) != 1 || files[0].Name() != scopedPeerEffectRecordName(0, false) {
			t.Fatal("partial journal mismatch", files, e)
		}
		if _, e = nativeScopedPeerApply(ctx, db, id); e == nil {
			t.Fatal("partial original was replayable")
		}
		if e = NativePeerGuardKernelReadback(ctx, current); e != nil {
			t.Fatal(e)
		}
		t.Log("PR178_N4_NATIVE_PARTIAL_ACL_ACK_LOSS=PASS")
	case "recover":
		if os.Geteuid() != 0 {
			t.Fatal("requires root read-only observer")
		}
		id := os.Getenv("N4_TEST_ORIGINAL")
		db, e := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer db.Close()
		original, e := db.Get(ctx, id)
		if e != nil || original.State != "UNKNOWN" {
			t.Fatal("original not UNKNOWN", original.State, e)
		}
		if _, e = nativeScopedPeerRecover(ctx, db, id); e == nil {
			t.Fatal("incomplete original recovered as success")
		}
		t.Log("PR178_N4_PARTIAL_NO_RECOVERY_REPLAY=PASS")
	default:
		t.Fatal("missing fixture phase")
	}
}
