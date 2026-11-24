package install

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vanishs/boxweave/internal/state"
)

// Kernel fixture uses a disposable sysfs-shaped directory. It does not
// spoof a real Host connection or imply that an nft preimage exists.
func pr178RootKernelFixture(t *testing.T) (
	state.ScopedPeerOriginalProof, *pr178RootNativeReadDriver, string, map[string]string) {
	t.Helper()
	proof, driver := pr178RootSDKFixture(t)
	root := t.TempDir()
	dir := filepath.Join(root, proof.Intent.Bridge, "brif")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	byName := map[string]string{}
	names := slices.Clone(driver.namesList)
	slices.Sort(names)
	for i, name := range names {
		host := fmt.Sprintf("vethpr178%02d", i)
		instance := driver.instances[name]
		if instance == nil {
			t.Fatal("missing tenant in sysfs fixture")
		}
		instance.Config["volatile.eth0.host_name"] = host
		byName[name] = host
		if err := os.Symlink("../../../owned", filepath.Join(dir, host)); err != nil {
			t.Fatal(err)
		}
	}
	return proof, driver, root, byName
}

func TestPR178RootSDKSysfsNFTDoublePreimageCompleteNoMutation(t *testing.T) {
	proof, driver, sysfs, _ := pr178RootKernelFixture(t)
	checks := 0
	out, err := nativePeerScopedRootKernelWithDriver(context.Background(),
		driver, proof, sysfs, func(ctx context.Context, guard NativePeerGuardPlan) error {
			checks++
			if guard.DigestSHA256 == "" || guard.DigestSHA256 != nativePeerGuardDigest(guard) {
				return errors.New("untrusted guard digest")
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		t.Fatal("complete tenant SDK+sysfs+nft read-only preflight failed", err)
	}
	if out.VerifiedBoxCount != 4 || out.KernelGuardChecks != 2 || checks != 2 ||
		out.Read.NativeACLBefore.ACLCount != 4 || out.Read.Plan.UnrelatedACLsVerifiedUnchanged != 2 ||
		out.GuardBeforeSHA256 != out.Read.Plan.GuardBefore.DigestSHA256 ||
		driver.networkReads != 4 || len(out.Read.InstanceETags) != 4 {
		t.Fatalf("incomplete native witness %+v, guardcalls=%d", out, checks)
	}
	for name, tag := range driver.instances {
		if driver.instReads[name] != 4 {
			t.Fatal("tenant instance not independently quadruple-read", name)
		}
		uuid := tag.Config["volatile.uuid"]
		acl := out.Read.Plan.ExpectedBeforeByUUID[uuid]
		if driver.aclReads[acl.Name] != 4 {
			t.Fatal("native root ACL not independently quadruple-read", acl.Name)
		}
	}
	if out.Read.Plan.OriginalBootAutostart[pr178B] != "false" ||
		out.Read.Plan.OriginalBootAutostart[pr178D] != "false" {
		t.Fatal("original tenant explicit false autostart was mutated")
	}
	t.Log("PR178_V2_SDK_SYSFS_NFT_READ_ONLY_COMPLETE=PASS")
}

func TestPR178RootSDKSysfsNFTRefusesPartialAndTransientNativeEvidence(t *testing.T) {
	cases := []struct {
		name   string
		change func(*testing.T, state.ScopedPeerOriginalProof, *pr178RootNativeReadDriver, string, map[string]string, *int)
	}{
		{"missing-sysfs-port", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			if err := os.Remove(filepath.Join(r, p.Intent.Bridge, "brif", n["pr178-b"])); err != nil {
				t.Fatal(err)
			}
		}},
		{"unlisted-foreign-veth", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			if err := os.Symlink("../../../foreign", filepath.Join(r, p.Intent.Bridge, "brif", "vethforeign001")); err != nil {
				t.Fatal(err)
			}
		}},
		{"swapped-host-veth", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			d.instances["pr178-b"].Config["volatile.eth0.host_name"] = "vethforeign11"
		}},
		{"missing-bridge", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			if err := os.RemoveAll(filepath.Join(r, p.Intent.Bridge)); err != nil {
				t.Fatal(err)
			}
		}},
		{"stopped-box-live-port", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			d.instances["pr178-c"].Status = "Stopped"
		}},
		{"no-SDK-network-ETag", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			d.bridgeETag = ""
		}},
		{"foreign-SDK-ACL", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			for _, acl := range d.aclByName {
				acl.Project = "foreign"
				break
			}
		}},
		{"invalid-protected-original", func(t *testing.T, p state.ScopedPeerOriginalProof, d *pr178RootNativeReadDriver, r string, n map[string]string, c *int) {
			d.bridge.Name = "foreign"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			proof, driver, sysfs, names := pr178RootKernelFixture(t)
			calls := 0
			tc.change(t, proof, driver, sysfs, names, &calls)
			got, err := nativePeerScopedRootKernelWithDriver(context.Background(),
				driver, proof, sysfs, func(context.Context, NativePeerGuardPlan) error { calls++; return nil })
			if err == nil || got.VerifiedBoxCount != 0 {
				t.Fatalf("incomplete native bridge/ACL/kernel accepted: %+v %v", got, err)
			}
		})
	}
}

func TestPR178RootSDKSysfsNFTRefusesChangingKernelAndSDKState(t *testing.T) {
	cases := []string{
		"kernel-nft-preimage-fails-first", "kernel-nft-preimage-fails-second",
		"foreign-sysfs-veth-between-passes", "missing-sysfs-veth-between-passes",
		"native-bridge-etag-changed-between-passes", "sibling-autostart-changed-between-passes",
		"native-ACL-injected-accept-between-passes", "context-cancel-between-passes",
	}
	for _, tc := range cases {
		t.Run(tc, func(t *testing.T) {
			proof, driver, sysfs, names := pr178RootKernelFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			observed := func(ctx context.Context, plan NativePeerGuardPlan) error {
				calls++
				if plan.DigestSHA256 != nativePeerGuardDigest(plan) {
					return errors.New("corrupt nft digest")
				}
				switch tc {
				case "kernel-nft-preimage-fails-first":
					return errors.New("kernel nft preimage missing")
				case "kernel-nft-preimage-fails-second":
					if calls == 2 {
						return errors.New("kernel nft preimage missing after re-read")
					}
				case "foreign-sysfs-veth-between-passes":
					if calls == 1 {
						if e := os.Symlink("../../../foreign", filepath.Join(sysfs, proof.Intent.Bridge, "brif", "vethrawforeign")); e != nil {
							t.Fatal(e)
						}
					}
				case "missing-sysfs-veth-between-passes":
					if calls == 1 {
						if e := os.Remove(filepath.Join(sysfs, proof.Intent.Bridge, "brif", names["pr178-c"])); e != nil {
							t.Fatal(e)
						}
					}
				case "native-bridge-etag-changed-between-passes":
					if calls == 1 {
						driver.bridgeETag = "etag-changed-since-first-pass"
					}
				case "sibling-autostart-changed-between-passes":
					if calls == 1 {
						driver.instances["pr178-c"].Config["boot.autostart"] = "unexpected"
					}
				case "native-ACL-injected-accept-between-passes":
					if calls == 1 {
						for _, acl := range driver.aclByName {
							if len(acl.UsedBy) > 0 && acl.UsedBy[0] == "/1.0/instances/pr178-d?project=user-1001" {
								acl.Egress[0].Action = "accept"
							}
						}
					}
				case "context-cancel-between-passes":
					if calls == 1 {
						cancel()
					}
				}
				return nil
			}
			got, err := nativePeerScopedRootKernelWithDriver(ctx, driver, proof, sysfs, observed)
			if err == nil || got.KernelGuardChecks != 0 {
				t.Fatalf("transient native fault was accepted: %+v calls=%d", got, calls)
			}
		})
	}
}

func TestPR178RootSDKSysfsNFTRejectsMissingWitnessOrRecoveryOriginal(t *testing.T) {
	proof, driver, sysfs, _ := pr178RootKernelFixture(t)
	if _, err := nativePeerScopedRootKernelWithDriver(context.Background(), driver, proof, sysfs, nil); err == nil {
		t.Fatal("missing nft kernel reader was treated as authorization")
	}
	proof.RecoveryOnly = true
	if _, err := nativePeerScopedRootKernelWithDriver(context.Background(), driver, proof, sysfs,
		func(context.Context, NativePeerGuardPlan) error { return nil }); err == nil {
		t.Fatal("UNKNOWN original was accepted as a native root dispatch proof")
	}
}
