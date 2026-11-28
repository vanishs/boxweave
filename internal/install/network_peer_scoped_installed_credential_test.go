package install

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/state"
)

type pr198ObservedACL struct {
	nativePeerScopedIncusACLDriver
	calls int
}

func (d *pr198ObservedACL) casACL(name string, put api.NetworkACLPut, tag string) error {
	d.calls++
	return d.nativePeerScopedIncusACLDriver.casACL(name, put, tag)
}

// Dedicated disposable installed lab: revoke only its own staged credential
// after a real deny-guard update, via the ordinary service-user command.
func TestScopedInstalledCredentialInterleaving(t *testing.T) {
	if os.Getenv("BW_INSTALLED_CREDENTIAL_LAB") != "pr198-dedicated" {
		t.Skip("dedicated installed lab only")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	const project = "user-1001"
	for _, u := range []string{"boxweave.service", "boxweave-scoped-lifecycle.service"} {
		if exec.Command("systemctl", "is-active", "--quiet", u).Run() == nil {
			t.Fatal("competing service", u)
		}
	}
	if os.Getenv("BW_INSTALLED_PHASE") == "stage" {
		if os.Geteuid() != 1001 {
			t.Fatal("restricted actor required")
		}
		s, e := state.Open(nativePeerRootInstalledDB)
		if e != nil {
			t.Fatal(e)
		}
		defer s.Close()
		l, e := s.ReadNativeNetworkLedger(ctx, project)
		if e != nil || len(l.Boxes) != 4 {
			t.Fatal("wrong fixture", e)
		}
		ids := map[string]string{}
		for _, b := range l.Boxes {
			ids[b.Name] = b.UUID
		}
		for _, n := range []string{"n4-a", "n4-b", "n4-c", "n4-d"} {
			if ids[n] == "" {
				t.Fatal("missing fixture", n)
			}
		}
		_, tok, e := s.IssueToken(ctx, time.Hour)
		if e != nil {
			t.Fatal(e)
		}
		r, e := s.AdmitScopedPeerSet(ctx, state.Admission{Layer: project, ActorID: tok.ID, TokenID: tok.ID, Key: "pr198-native-credential-drift", RequestID: "pr198-credential-drift"}, ids["n4-b"], ids["n4-a"], false)
		if e != nil || r.Duplicate {
			t.Fatal("admission failed", e)
		}
		if e = s.StageAndClaimScopedPeerOriginal(ctx, r.ID); e != nil {
			t.Fatal(e)
		}
		t.Log("PR198_ORIGINAL=" + r.ID)
		return
	}
	if os.Getenv("BW_INSTALLED_PHASE") != "apply" || os.Geteuid() != 0 {
		t.Fatal("wrong phase or actor")
	}
	id := os.Getenv("BW_INSTALLED_ORIGINAL")
	if !scopedLifecycleID.MatchString(id) {
		t.Fatal("invalid original")
	}
	db, e := state.OpenNetworkAuditReadOnly(nativePeerRootInstalledDB)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	r, initial, e := nativePeerFreshVerifiedPlan()
	if e != nil || r.Project != project {
		t.Fatal("wrong plan", e)
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
		return db.ReadRootScopedPeerOriginal(ctx, id, project, r.Bridge)
	}
	proof, e := read(ctx)
	if e != nil || proof.Intent.Allow || len(proof.Inventory.Boxes) != 4 {
		t.Fatal("wrong original", e)
	}
	names := map[string]string{}
	for _, b := range proof.Inventory.Boxes {
		names[b.Name] = b.UUID
	}
	if names["n4-b"] != proof.Intent.SourceUUID || names["n4-a"] != proof.Intent.TargetUUID || names["n4-c"] == "" || names["n4-d"] == "" {
		t.Fatal("wrong original pair")
	}
	base, e := nativePeerScopedRootReadSnapshot(ctx, nativePeerScopedIncusReader{root: c}, proof)
	if e != nil {
		t.Fatal(e)
	}
	plan, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
	if e != nil || plan.GuardBefore != current {
		t.Fatal("wrong preimage", e)
	}
	batches, ct := 0, 0
	driver := &pr198ObservedACL{nativePeerScopedIncusACLDriver: nativePeerScopedIncusACLDriver{root: c}}
	rec, e := nativeScopedPeerExecute(ctx, nativePeerFreshJournalDir, 0, r.HostSHA256, read, nativePeerScopedIncusReader{root: c}, driver, "/sys/class/net", NativePeerGuardKernelReadback, func(ctx context.Context, s string) error {
		batches++
		if s != plan.GuardAfter.Replace {
			t.Fatal("wrong batch")
		}
		if e := nativePeerGuardFixedNftBatch(ctx, s); e != nil {
			return e
		}
		if e := exec.CommandContext(ctx, "runuser", "-u", "boxweave", "--", "/usr/local/bin/boxweave", "token-revoke", "-project", project, "-id", proof.TokenID).Run(); e != nil {
			t.Fatal("credential revoke failed", e)
		}
		return nil
	}, func(ctx context.Context, args ...string) ([]byte, error) {
		ct++
		return nativePeerConntrackOS(ctx, args...)
	}, func(ctx context.Context, p state.ScopedPeerOriginalProof, s nativePeerScopedSDKSnapshot) (string, error) {
		return nativeScopedPeerGlobal(ctx, c, db, r, p, s)
	})
	if e == nil || rec.Status == "COMPLETE_VERIFIED" || batches != 1 || ct != 0 || driver.calls != 0 {
		t.Fatal("drift not fenced", rec, e, batches, ct, driver.calls)
	}
	if _, e = read(ctx); !errors.Is(e, state.ErrRevoked) {
		t.Fatal("credential not refused", e)
	}
	after, e := nativePeerScopedRootReadSnapshot(ctx, nativePeerScopedIncusReader{root: c}, proof)
	if e != nil || !reflect.DeepEqual(base, after) {
		t.Fatal("native ACL or inventory changed", e)
	}
	if e = NativePeerGuardKernelReadback(ctx, plan.GuardAfter); e != nil {
		t.Fatal(e)
	}
	files, e := os.ReadDir(filepath.Join(nativePeerFreshJournalDir, "peer-"+id))
	if e != nil || len(files) != 1 || files[0].Name() != scopedPeerEffectRecordName(0, false) {
		t.Fatal("journal mismatch", files, e)
	}
	original, e := db.Get(ctx, id)
	if e != nil || original.State != "DISPATCHING" {
		t.Fatal("original unexpectedly settled", e)
	}
	t.Log("PR198_INSTALLED_CREDENTIAL_DRIFT_FENCED=PASS")
}
