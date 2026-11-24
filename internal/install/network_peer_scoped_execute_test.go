package install

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestPR178N3ComposedScopedEffects(t *testing.T) {
	for _, allow := range []bool{true, false} {
		for _, fault := range []string{"none", "acl-ack-lost", "guard-ack-lost", "ct-ack-lost", "sibling-boot-drift", "original-drift", "global-drift"} {
			t.Run(map[bool]string{true: "grant-", false: "revoke-"}[allow]+fault, func(t *testing.T) {
				ctx := context.Background()
				proof, d, sysfs, _ := pr178RootKernelFixture(t)
				proof.ActorID = "fixture-actor"
				proof.TokenID = "fixture-token"
				if !allow {
					ab := proof.After[1]
					proof.Intent.Allow = false
					proof.Before = proof.After
					proof.After = proof.Before[:1]
					proof.RevokedAfter = []state.PeerGrantBinding{ab}
				}
				base, e := nativePeerScopedRootReadSnapshot(ctx, d, proof)
				if e != nil {
					t.Fatal(e)
				}
				p, e := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
				if e != nil {
					t.Fatal(e)
				}
				for _, a := range p.ExpectedBeforeByUUID {
					d.aclByName[a.Name].NetworkACLPut = a.NetworkACLPut
				}
				c := pr178RootSDKClone(d.aclByName[p.ExpectedBeforeByUUID[pr178C].Name])
				sibling := pr178RootSDKClone(d.instances["pr178-d"])
				w := &pr178CASDriver{native: d, applyLostACK: fault == "acl-ack-lost"}
				guardNow := p.GuardBefore
				batchCalls, ctCalls := 0, 0
				read := func(context.Context) (state.ScopedPeerOriginalProof, error) {
					q := proof
					if fault == "original-drift" && (len(w.writes) > 0 || batchCalls > 0) {
						q.ActorID = "changed"
					}
					return q, nil
				}
				guard := func(_ context.Context, g NativePeerGuardPlan) error {
					if g != guardNow {
						return errors.New("wrong guard prefix")
					}
					return nil
				}
				batch := func(_ context.Context, script string) error {
					batchCalls++
					if script != p.GuardAfter.Replace {
						return errors.New("untrusted batch")
					}
					guardNow = p.GuardAfter
					if fault == "sibling-boot-drift" {
						d.instances["pr178-d"].Config["boot.autostart"] = "true"
					}
					if fault == "guard-ack-lost" {
						return errors.New("applied guard ACK lost")
					}
					return nil
				}
				ct := func(_ context.Context, args ...string) ([]byte, error) {
					ctCalls++
					if args[0] == "-L" {
						if ctCalls <= 1 && fault == "ct-ack-lost" {
							return []byte("tcp src=" + args[4] + " dst=" + args[6] + "\nconntrack v1.4.7: 1 flow entries have been shown"), nil
						}
						return []byte("conntrack v1.4.7: 0 flow entries have been shown"), nil
					}
					return nil, errors.New("applied conntrack ACK lost")
				}
				global := func(context.Context, state.ScopedPeerOriginalProof, nativePeerScopedSDKSnapshot) (string, error) {
					if fault == "global-drift" && (len(w.writes) > 0 || batchCalls > 0) {
						return strings.Repeat("d", 64), nil
					}
					return strings.Repeat("c", 64), nil
				}
				dir := t.TempDir()
				run := func() (state.ScopedPeerRootReceipt, error) {
					return nativeScopedPeerExecute(ctx, dir, uint32(os.Getuid()), strings.Repeat("b", 64), read, d, w, sysfs, guard, batch, ct, global)
				}
				rec, e := run()
				if fault == "none" {
					if e != nil || rec.Status != "COMPLETE_VERIFIED" || rec.ProofSHA256 != state.ScopedPeerProofSHA256(proof) || len(w.writes) != 2 || batchCalls != 1 || ctCalls != 2 || guardNow != p.GuardAfter {
						t.Fatal(rec, e, w.writes, batchCalls, ctCalls)
					}
				} else if e == nil || rec.Status == "COMPLETE_VERIFIED" {
					t.Fatal("uncertain original completed", rec, e)
				}
				if !reflect.DeepEqual(c, d.aclByName[c.Name]) || fault != "sibling-boot-drift" && !reflect.DeepEqual(sibling, d.instances["pr178-d"]) {
					t.Fatal("unrelated native state modified")
				}
				old := len(w.writes) + batchCalls + ctCalls
				if _, e = run(); e == nil || len(w.writes)+batchCalls+ctCalls != old {
					t.Fatal("original replayed", e)
				}
			})
		}
	}
}
