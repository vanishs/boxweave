package install

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestPR178N3BootPolicyJournalChain(t *testing.T) {
	for _, fault := range []string{"none", "unpublished", "missing-stage", "wrong-receipt", "unknown-original", "symlink-policy", "missing-policy"} {
		t.Run(fault, func(t *testing.T) {
			ctx := context.Background()
			proof, nw, obs := pr178FourRootScopedFixture(t)
			proof.Before = nil
			proof.After = proof.After[1:]
			r := nativePeerFreshJournalRecord{Schema: 1, Mode: nativePeerFreshMode, HostSHA256: strings.Repeat("b", 64), Project: proof.Intent.Project, Bridge: proof.Intent.Bridge, IPv4CIDR: nw.Config["ipv4.address"], IPv6CIDR: nw.Config["ipv6.address"]}
			initial, e := nativePeerFreshInitialGuard(pr178GuardTarget(), nw)
			if e != nil {
				t.Fatal(e)
			}
			dir := t.TempDir()
			if e = os.Chmod(dir, 0700); e != nil {
				t.Fatal(e)
			}
			uid := uint32(os.Getuid())
			got, last, e := scopedPeerBootCurrent(dir, uid, r, initial, "")
			if e != nil || got != initial || last.Sequence != 0 {
				t.Fatal(got, last, e)
			}
			do := func(p state.ScopedPeerOriginalProof, allowPublish bool) (NativePeerGuardPlan, error) {
				plan, e := NativePeerScopedRootTransitionFromEvidence(p, nw, obs)
				if e != nil {
					t.Fatal(e)
				}
				scope := scopedPeerEffectJournalScope{HostSHA256: r.HostSHA256, LedgerSHA256: state.ScopedPeerProofSHA256(p), NativeSHA256: strings.Repeat("c", 64)}
				rec, e := nativeScopedPeerEffectsOnce(ctx, dir, uid, plan, scope, func(_ context.Context, i int) (string, error) {
					return nativePrepareStateDigest("fixture-prefix", i), nil
				}, func(context.Context, nativePeerScopedEffectStep) error { return nil })
				if e != nil {
					t.Fatal(e)
				}
				if _, _, e = scopedPeerBootCurrent(dir, uid, r, initial, ""); e == nil {
					t.Fatal("unpublished original restored")
				}
				if !allowPublish {
					return plan.GuardAfter, nil
				}
				receipt := state.ScopedPeerRootReceipt{OriginalID: p.OriginalID, Project: r.Project, Bridge: r.Bridge, Status: rec.Status, ProofSHA256: scope.LedgerSHA256, PlanSHA256: rec.PlanSHA256, FinalSHA256: rec.FinalSHA256, VerifiedSteps: rec.VerifiedSteps}
				if fault == "wrong-receipt" {
					receipt.FinalSHA256 = strings.Repeat("d", 64)
				}
				e = scopedPeerBootPublish(ctx, dir, uid, r, initial, p, receipt, func(_ context.Context, g NativePeerGuardPlan) error {
					if g != plan.GuardAfter {
						t.Fatal("wrong guard afterimage")
					}
					return nil
				})
				return plan.GuardAfter, e
			}
			want, e := do(proof, fault != "unpublished")
			if fault == "wrong-receipt" {
				if e == nil {
					t.Fatal("forged receipt published")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			switch fault {
			case "unpublished":
			case "missing-stage":
				if e = os.Remove(filepath.Join(dir, "peer-"+proof.OriginalID, "step-01-verified.json")); e != nil {
					t.Fatal(e)
				}
			case "unknown-original":
				if e = os.Mkdir(filepath.Join(dir, "peer-"+strings.Repeat("e", 32)), 0700); e != nil {
					t.Fatal(e)
				}
			case "symlink-policy":
				path := filepath.Join(dir, scopedPeerBootPolicyName(1))
				if e = os.Rename(path, path+".saved"); e != nil {
					t.Fatal(e)
				}
				if e = os.Symlink(path+".saved", path); e != nil {
					t.Fatal(e)
				}
			case "missing-policy":
				if e = os.Remove(filepath.Join(dir, scopedPeerBootPolicyName(1))); e != nil {
					t.Fatal(e)
				}
			}
			got, last, e = scopedPeerBootCurrent(dir, uid, r, initial, "")
			if fault != "none" {
				if e == nil {
					t.Fatal("uncertain policy restored", last)
				}
				return
			}
			if e != nil || got != want || last.Sequence != 1 {
				t.Fatal(got, last, e)
			}
			_, driver, sysfs, _ := pr178RootKernelFixture(t)
			plan, e := NativePeerScopedRootTransitionFromEvidence(proof, nw, obs)
			if e != nil {
				t.Fatal(e)
			}
			for _, a := range plan.ExpectedAfterByUUID {
				driver.aclByName[a.Name].NetworkACLPut = a.NetworkACLPut
			}
			unknown := proof
			unknown.RecoveryOnly = true
			read := func(context.Context) (state.ScopedPeerOriginalProof, error) { return unknown, nil }
			guard := func(_ context.Context, g NativePeerGuardPlan) error {
				if g != got {
					t.Fatal("wrong recovery guard")
				}
				return nil
			}
			global := func(context.Context, state.ScopedPeerOriginalProof, nativePeerScopedSDKSnapshot) (string, error) {
				return strings.Repeat("a", 64), nil
			}
			rec, e := nativeScopedPeerRecoverWithDriver(ctx, dir, uid, r, initial, read, driver, sysfs, guard, global)
			if e != nil || !rec.RecoveryOnly || rec.OriginalID != proof.OriginalID || rec.ProofSHA256 != state.ScopedPeerProofSHA256(proof) {
				t.Fatal("read-only recovery failed", rec, e)
			}
			driver.instances["pr178-c"].Config["boot.autostart"] = "false"
			if _, e = nativeScopedPeerRecoverWithDriver(ctx, dir, uid, r, initial, read, driver, sysfs, guard, global); e == nil {
				t.Fatal("recovery accepted changed native boot preference")
			}
			proof.OriginalID = strings.Repeat("f", 32)
			proof.Intent.Allow = false
			proof.Before = proof.After
			proof.After = nil
			proof.RevokedAfter = proof.Before
			want, e = do(proof, true)
			if e != nil {
				t.Fatal(e)
			}
			got, last, e = scopedPeerBootCurrent(dir, uid, r, initial, "")
			if e != nil || got != want || got != initial || last.Sequence != 2 {
				t.Fatal(got, last, e)
			}
		})
	}
}
