package install

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

func TestS4B2ConntrackScopeMustComeFromFullSQLiteLedger(t *testing.T) {
	for _, tc := range []struct {
		name, quarantineUUID string
	}{
		{"complete-ledger", ""},
		{"quarantined-target", pr178B},
		{"quarantined-unrelated-owner", pr178C},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			fixture, d, sysfs, _ := pr178RootKernelFixture(t)
			ab := fixture.After[1]
			fixture.Intent.Allow = false
			fixture.Before = fixture.After
			fixture.After = fixture.Before[:1]
			fixture.RevokedAfter = []state.PeerGrantBinding{ab}

			s, root, db, id := scopedInstallStore(t, fixture)
			read := func(ctx context.Context) (state.ScopedPeerOriginalProof, error) {
				return root.ReadRootScopedPeerOriginal(ctx, id, fixture.Intent.Project, fixture.Intent.Bridge)
			}
			proof, err := read(ctx)
			if err != nil || len(proof.Inventory.Boxes) != 4 || proof.Intent.SourceUUID != pr178A || proof.Intent.TargetUUID != pr178B {
				t.Fatal("complete scoped pair identity was not derived from SQLite", proof, err)
			}
			for _, pair := range [][2]string{{pr178A, pr178A}, {pr178A, "foreign-uuid"}, {"", pr178B}} {
				if _, err := s.AdmitScopedPeerSet(ctx, state.Admission{
					Layer: proof.Intent.Project, ActorID: proof.ActorID, TokenID: proof.TokenID,
					Key: "invalid-conntrack-pair", RequestID: "invalid-pair",
				}, pair[0], pair[1], false); err == nil {
					t.Fatal("unbound pair UUID passed state gate", pair)
				}
			}

			base, err := nativePeerScopedRootReadSnapshot(ctx, d, proof)
			if err != nil {
				t.Fatal(err)
			}
			plan, err := NativePeerScopedRootTransitionFromEvidence(proof, base.Network, base.Evidence)
			if err != nil {
				t.Fatal(err)
			}
			for _, acl := range plan.ExpectedBeforeByUUID {
				d.aclByName[acl.Name].NetworkACLPut = acl.NetworkACLPut
			}
			w := &pr178CASDriver{native: d}
			guardNow := plan.GuardBefore
			batches, ct := 0, 0
			want := [][]string{
				{"-L", "-f", "ipv4", "-s", ab.SourceIPv4, "-d", ab.TargetIPv4},
				{"-L", "-f", "ipv6", "-s", ab.SourceIPv6, "-d", ab.TargetIPv6},
			}
			rec, err := nativeScopedPeerExecute(ctx, t.TempDir(), uint32(os.Getuid()), strings.Repeat("b", 64), read, d, w, sysfs,
				func(_ context.Context, g NativePeerGuardPlan) error {
					if g != guardNow {
						return errors.New("unexpected guard preimage")
					}
					return nil
				},
				func(_ context.Context, script string) error {
					batches++
					if script != plan.GuardAfter.Replace {
						t.Fatal("unexpected deny guard batch")
					}
					guardNow = plan.GuardAfter
					if tc.quarantineUUID != "" {
						r, err := db.ExecContext(ctx, "UPDATE network_address_fences SET state='QUARANTINED' WHERE box_uuid=?", tc.quarantineUUID)
						if err != nil {
							t.Fatal(err)
						}
						if n, err := r.RowsAffected(); err != nil || n != 2 {
							t.Fatal("missing dual-stack drift fixture", n, err)
						}
					}
					return nil
				},
				func(_ context.Context, args ...string) ([]byte, error) {
					if ct >= len(want) || !slices.Equal(args, want[ct]) {
						t.Fatal("conntrack scope differs from the exact SQLite pair", args)
					}
					ct++
					return []byte("conntrack v1.4.7: 0 flow entries have been shown"), nil
				},
				func(context.Context, state.ScopedPeerOriginalProof, nativePeerScopedSDKSnapshot) (string, error) {
					return strings.Repeat("c", 64), nil
				})
			if batches != 1 {
				t.Fatal("deny guard was not installed exactly once", batches)
			}
			if tc.quarantineUUID == "" {
				if err != nil || rec.Status != "COMPLETE_VERIFIED" || rec.ProofSHA256 != state.ScopedPeerProofSHA256(proof) || ct != len(want) || len(w.writes) != 2 {
					t.Fatal("complete scoped ledger did not authorize exact directional conntrack", rec, err, ct, w.writes)
				}
				return
			}
			if err == nil || rec.Status == "COMPLETE_VERIFIED" || ct != 0 || len(w.writes) != 0 {
				t.Fatal("incomplete SQLite inventory permitted native effects", rec, err, ct, w.writes)
			}
			if _, err := read(ctx); !errors.Is(err, state.ErrResourceLocked) {
				t.Fatal("quarantined ledger was not refused", err)
			}
		})
	}
}
