package install

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/lxc/incus/v7/shared/api"
)

type pr178CASDriver struct {
	native       *pr178RootNativeReadDriver
	writes       []string
	applyLostACK bool
	spoofAfter   bool
}

func (d *pr178CASDriver) getACL(name string) (*api.NetworkACL, string, error) {
	p := d.native.aclByName[name]
	if p == nil {
		return nil, "", errors.New("missing native ACL")
	}
	v := pr178RootSDKClone(p)
	if d.spoofAfter && len(d.writes) > 0 {
		v.Description = "forged post-readback"
	}
	return v, d.native.aclETags[name], nil
}
func (d *pr178CASDriver) casACL(name string, put api.NetworkACLPut, etag string) error {
	if d.native.aclETags[name] != etag {
		return errors.New("native ETag precondition failed")
	}
	d.writes = append(d.writes, name)
	d.native.aclByName[name].NetworkACLPut = put
	d.native.aclETags[name] = fmt.Sprintf("new-native-etag-%s-%d", name, len(d.writes))
	if d.applyLostACK {
		return errors.New("native applied but SDK ACK lost")
	}
	return nil
}
func pr178CASPlan(t *testing.T) (NativePeerScopedRootTransition, *pr178RootNativeReadDriver, *pr178CASDriver, map[string]NativePeerGuardObservedOwner) {
	t.Helper()
	proof, driver := pr178RootSDKFixture(t)
	got, e := nativePeerScopedRootReadOnlyWithDriver(context.Background(), driver, proof)
	if e != nil {
		t.Fatal(e)
	}
	if !got.Plan.IntentAllow {
		t.Fatal("grant fixture wrong direction")
	}
	observed := map[string]NativePeerGuardObservedOwner{}
	for name, inst := range driver.instances {
		uuid := inst.Config["volatile.uuid"]
		acl := got.Plan.ExpectedBeforeByUUID[uuid]
		observed[uuid] = NativePeerGuardObservedOwner{
			Instance: pr178RootSDKClone(inst), ACL: pr178RootSDKClone(driver.aclByName[acl.Name]),
			ACLTag: driver.aclETags[acl.Name],
		}
		if name == "" {
			t.Fatal("empty native Project name")
		}
	}
	return got.Plan, driver, &pr178CASDriver{native: driver}, observed
}
func pr178CASReverse(plan NativePeerScopedRootTransition) NativePeerScopedRootTransition {
	r := plan
	r.IntentAllow = false
	r.GuardBefore, r.GuardAfter = plan.GuardAfter, plan.GuardBefore
	r.ExpectedBeforeByUUID = plan.ExpectedAfterByUUID
	r.ExpectedAfterByUUID = plan.ExpectedBeforeByUUID
	r.SourceACLBefore, r.SourceACLAfter = plan.SourceACLAfter, plan.SourceACLBefore
	r.TargetACLBefore, r.TargetACLAfter = plan.TargetACLAfter, plan.TargetACLBefore
	return r
}
func pr178CASWitness(t *testing.T, plan NativePeerScopedRootTransition,
	wantAfter bool) *int {
	t.Helper()
	_ = plan
	_ = wantAfter
	v := 0
	return &v
}
func TestPR178ScopedNativeACLETagCASGrantRevokeNeverTouchesCD(t *testing.T) {
	plan, native, writer, observed := pr178CASPlan(t)
	c := pr178RootSDKClone(native.aclByName[plan.ExpectedBeforeByUUID[pr178C].Name])
	d := pr178RootSDKClone(native.aclByName[plan.ExpectedBeforeByUUID[pr178D].Name])
	beforeBootC := native.instances["pr178-c"].Config["boot.autostart"]
	beforeBootD := native.instances["pr178-d"].Config["boot.autostart"]
	count := 0
	grantGuard := func(ctx context.Context, p NativePeerGuardPlan) error {
		count++
		if p.DigestSHA256 != plan.GuardBefore.DigestSHA256 {
			return errors.New("grant moved guard early")
		}
		return ctx.Err()
	}
	for _, id := range []string{plan.SourceUUID, plan.TargetUUID} {
		r, e := nativePeerScopedACLReplaceOnce(context.Background(), plan, id, observed[id], grantGuard, writer)
		if e != nil || !r.Attempted || !r.Verified || r.NewETag == r.OldETag {
			t.Fatalf("native CAS grant failed: %+v %v", r, e)
		}
	}
	if len(writer.writes) != 2 || writer.writes[0] != plan.SourceACLName ||
		writer.writes[1] != plan.TargetACLName || count != 6 {
		t.Fatal("grant must update exactly A and B under unchanged deny guard", writer.writes, count)
	}
	if !reflect.DeepEqual(c, native.aclByName[c.Name]) ||
		!reflect.DeepEqual(d, native.aclByName[d.Name]) ||
		native.instances["pr178-c"].Config["boot.autostart"] != beforeBootC ||
		native.instances["pr178-d"].Config["boot.autostart"] != beforeBootD {
		t.Fatal("A/B policy write changed C/D ACL or boot.autostart")
	}
	revoke := pr178CASReverse(plan)
	revokeWitness := func(ctx context.Context, p NativePeerGuardPlan) error {
		if p.DigestSHA256 != revoke.GuardAfter.DigestSHA256 {
			return errors.New("revoke ACL before kernel guard DROP")
		}
		return ctx.Err()
	}
	for _, id := range []string{revoke.SourceUUID, revoke.TargetUUID} {
		acl := revoke.ExpectedBeforeByUUID[id]
		box := observed[id]
		box.ACL = pr178RootSDKClone(native.aclByName[acl.Name])
		box.ACLTag = native.aclETags[acl.Name]
		rec, e := nativePeerScopedACLReplaceOnce(context.Background(), revoke, id, box, revokeWitness, writer)
		if e != nil || !rec.Verified || !rec.Attempted {
			t.Fatal("revoke ACL update was not observed", rec, e)
		}
	}
	if len(writer.writes) != 4 || !reflect.DeepEqual(c, native.aclByName[c.Name]) ||
		!reflect.DeepEqual(d, native.aclByName[d.Name]) {
		t.Fatal("revoke touched unrelated C/D ACL", writer.writes)
	}
}
func TestPR178ScopedNativeACLETagCASRefusesUntrustedPreimage(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*NativePeerScopedRootTransition, *pr178RootNativeReadDriver, *NativePeerGuardObservedOwner, *pr178CASDriver)
	}{
		{"intent-polarity-mismatch", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			p.IntentAllow = false
		}},
		{"boot-map-key-substitution", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			delete(p.OriginalBootAutostart, pr178C)
			p.OriginalBootAutostart["foreign"] = "false"
		}},
		{"foreign-target-UUID", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			p.SourceUUID = pr178C
		}},
		{"spurious-native-accept", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.ACL.Egress = append(o.ACL.Egress, api.NetworkACLRule{Action: "accept", Destination: "0.0.0.0/0"})
		}},
		{"foreign-native-project", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.Instance.Project = "foreign"
		}},
		{"stale-A-UUID", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.Instance.Config["volatile.uuid"] = pr178D
		}},
		{"autostart-changed", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.Instance.Config["boot.autostart"] = "false"
		}},
		{"foreign-UsedBy", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.ACL.UsedBy = []string{"/1.0/instances/foreign?project=other"}
		}},
		{"foreign-ACL-etag", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			o.ACLTag = "foreign"
		}},
		{"changed-live-Etag", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			n.aclETags[p.SourceACLName] = "native-competing-etag"
		}},
		{"changed-live-ACL", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			n.aclByName[p.SourceACLName].Description = "foreign"
		}},
		{"tampered-other-C-ACL", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			b := p.ExpectedAfterByUUID[pr178C]
			b.Description = "changed C"
			p.ExpectedAfterByUUID[pr178C] = b
		}},
		{"missing-other-D-ACL", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			delete(p.ExpectedBeforeByUUID, pr178D)
		}},
		{"forged-guard-digest", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			p.GuardAfter.DigestSHA256 = "bad"
		}},
		{"same-target-policy", func(p *NativePeerScopedRootTransition, n *pr178RootNativeReadDriver, o *NativePeerGuardObservedOwner, d *pr178CASDriver) {
			p.ExpectedAfterByUUID[p.SourceUUID] = p.ExpectedBeforeByUUID[p.SourceUUID]
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan, driver, writer, observed := pr178CASPlan(t)
			o := observed[plan.SourceUUID]
			tc.mutate(&plan, driver, &o, writer)
			r, e := nativePeerScopedACLReplaceOnce(context.Background(), plan, pr178A, o,
				func(context.Context, NativePeerGuardPlan) error { return nil }, writer)
			if e == nil || r.Attempted || r.Verified || len(writer.writes) != 0 {
				t.Fatalf("unsafe native CAS applied: %+v %v writes=%v", r, e, writer.writes)
			}
		})
	}
}
func TestPR178ScopedNativeACLETagCASACKLossAndFaultsNeverRetry(t *testing.T) {
	for _, tc := range []string{"applied-ACK-lost", "postimage-fault", "guard-before-fault", "guard-at-PUT-fault", "guard-post-ACK-fault", "cancelled"} {
		t.Run(tc, func(t *testing.T) {
			plan, driver, writer, obs := pr178CASPlan(t)
			o := obs[plan.SourceUUID]
			if tc == "applied-ACK-lost" {
				writer.applyLostACK = true
			}
			if tc == "postimage-fault" {
				writer.spoofAfter = true
			}
			calls := 0
			guard := func(ctx context.Context, p NativePeerGuardPlan) error {
				calls++
				if tc == "guard-before-fault" ||
					tc == "guard-at-PUT-fault" && calls == 2 ||
					tc == "guard-post-ACK-fault" && calls == 3 {
					return errors.New("strict native nft readback missing")
				}
				return ctx.Err()
			}
			ctx := context.Background()
			if tc == "cancelled" {
				c, stop := context.WithCancel(ctx)
				stop()
				ctx = c
			}
			receipt, e := nativePeerScopedACLReplaceOnce(ctx, plan, plan.SourceUUID, o, guard, writer)
			if e == nil || receipt.Verified {
				t.Fatalf("uncertain or invalid native ACL step falsely verified: %+v %v", receipt, e)
			}
			attempted := tc == "applied-ACK-lost" || tc == "postimage-fault" || tc == "guard-post-ACK-fault"
			if receipt.Attempted != attempted || len(writer.writes) != map[bool]int{true: 1, false: 0}[attempted] {
				t.Fatal("unexpected number of native writes", tc, receipt, writer.writes)
			}
			if attempted {
				// Fake kernel/SDK did apply the ACL; the lost reply remains UNKNOWN.
				if !nativePeerScopedACLFullMatch(driver.aclByName[plan.SourceACLName],
					plan.SourceACLAfter, o.ACL.UsedBy[0]) {
					t.Fatal("failed ACK injection did not apply native ACL")
				}
			}
		})
	}
}
