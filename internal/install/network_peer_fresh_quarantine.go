package install

import (
	"context"
	"errors"
	"github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
	"net/http"
	"os"
	"os/user"
	"reflect"
)

func nativePeerFreshQuarantinePolicy(r nativePeerFreshJournalRecord, nw *api.Network) (api.NetworkACLsPost, error) {
	if nw == nil || nw.Name != r.Bridge || nw.Config["ipv4.address"] != r.IPv4CIDR || nw.Config["ipv6.address"] != r.IPv6CIDR || nw.Config["security.acls"] != "" {
		return api.NetworkACLsPost{}, errors.New("scoped bridge changed")
	}
	// Explicit catch-all deny, not merely the profile's default action:
	// an accidentally overridden Internet action never enables a pending NIC.
	acl, err := nativePrepareACL(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: peerscope.QuarantineACL}, nw)
	if err != nil {
		return acl, err
	}
	acl.Description = "BoxWeave scoped v2 quarantine only"
	acl.Ingress = []api.NetworkACLRule{{Action: "reject", State: "enabled", Source: "0.0.0.0/0,::/0"}}
	acl.Egress = []api.NetworkACLRule{{Action: "reject", State: "enabled", Destination: "0.0.0.0/0,::/0"}}
	return acl, nil
}

func nativePeerFreshQuarantineRead(ctx context.Context, c incus.InstanceServer, r nativePeerFreshJournalRecord, pendingName string) (*api.Profile, *api.Project, *api.Network, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, nil, err
	}
	own := c.UseProject(r.Project)
	nw, _, err := c.GetNetwork(r.Bridge)
	if err != nil {
		return nil, nil, nil, err
	}
	desired, err := nativePeerFreshQuarantinePolicy(r, nw)
	if err != nil {
		return nil, nil, nil, err
	}
	pr, _, err := own.GetProject(r.Project)
	if err != nil || pr == nil || pr.Config["restricted"] != "true" || pr.Config["features.networks"] != "false" || pr.Config["restricted.networks.access"] != r.Bridge || !reflect.DeepEqual(pr.Writable(), nativePrepareProject(pr.Writable())) {
		return nil, nil, nil, errors.New("scoped restricted project changed")
	}
	pf, _, err := own.GetProfile("default")
	u, ue := user.Lookup("boxweave")
	g, ge := user.LookupGroup("incus")
	if err != nil || pf == nil || ue != nil || ge != nil || !native.SafePerUserProfileConfig(pf.Config, u.Uid, g.Gid) || r.Project != "user-"+u.Uid {
		return nil, nil, nil, errors.New("scoped profile identity changed")
	}
	target := NativePrepareTarget{Project: r.Project, Pool: pf.Devices["root"]["pool"], Network: r.Bridge, ACL: peerscope.QuarantineACL}
	want := nativePrepareProfile(target, pf.Writable())
	want.Devices["eth0"]["security.acls.default.egress.action"] = "reject"
	if !safePrereqName.MatchString(target.Pool) || !reflect.DeepEqual(pf.Writable(), want) {
		return nil, nil, nil, errors.New("quarantine profile drift")
	}
	acl, tag, err := c.GetNetworkACL(peerscope.QuarantineACL)
	if err != nil || acl == nil || tag == "" || acl.Project != "default" || !reflect.DeepEqual(acl.Writable(), desired.NetworkACLPut) {
		return nil, nil, nil, errors.New("quarantine ACL drift")
	}
	profileSeen, pendingSeen := false, false
	for _, ref := range acl.UsedBy {
		if nativePeerGuardOwnedURL(ref, "profiles", "default", r.Project) && !profileSeen {
			profileSeen = true
			continue
		}
		if pendingName != "" && !pendingSeen && nativePeerGuardOwnedURL(ref, "instances", pendingName, r.Project) {
			pendingSeen = true
			continue
		}
		return nil, nil, nil, errors.New("quarantine has unexpected consumer")
	}
	if !profileSeen || (pendingName != "") != pendingSeen {
		return nil, nil, nil, errors.New("missing quarantine profile")
	}
	return pf, pr, nw, nil
}

func nativePeerFreshQuarantineSetup(ctx context.Context) error {
	r, plan, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return err
	}
	if err = NativePeerGuardKernelReadback(ctx, plan); err != nil {
		return err
	}
	before, err := nativePeerFreshRootSnapshot(ctx)
	if err != nil || before.NativeSHA256 != r.NativeSHA256 || before.Guard != plan {
		return errors.New("fresh quarantine preimage changed")
	}
	if err = nativePeerFreshPathsAbsentExceptJournal(os.Lstat, true); err != nil {
		return err
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return err
	}
	c = c.UseProject("default")
	own := c.UseProject(r.Project)
	pf, ft, err := own.GetProfile("default")
	if err != nil || pf == nil || ft == "" {
		return errors.New("missing quarantine profile preimage")
	}
	pr, pt, err := own.GetProject(r.Project)
	if err != nil || pr == nil || pt == "" {
		return errors.New("missing quarantine project preimage")
	}
	nw, _, err := c.GetNetwork(r.Bridge)
	if err != nil {
		return err
	}
	acl, err := nativePeerFreshQuarantinePolicy(r, nw)
	if err != nil {
		return err
	}
	if _, _, err = c.GetNetworkACL(peerscope.QuarantineACL); !api.StatusErrorCheck(err, http.StatusNotFound) {
		return errors.New("quarantine ACL already exists or absence uncertain")
	}
	target := NativePrepareTarget{Project: r.Project, Pool: pf.Devices["root"]["pool"], Network: r.Bridge, ACL: peerscope.QuarantineACL}
	desired := nativePrepareProfile(target, pf.Writable())
	desired.Devices["eth0"]["security.acls.default.egress.action"] = "reject"
	d, err := nativePeerFreshJournalOpenAt(nativePeerFreshJournalDir, 0)
	if err != nil {
		return err
	}
	defer d.Close()
	r.Status = "QUARANTINE_DISPATCH_UNKNOWN"
	if err = nativePeerFreshJournalWrite(d, r); err != nil {
		return err
	}
	again, err := nativePeerFreshRootSnapshot(ctx)
	if err != nil || again != before {
		return errors.New("quarantine original drifted; preserve UNKNOWN")
	}
	preimage := scopedRetirementPreimage{
		Schema: 1, Origin: r,
		BeforeProject: pr.Writable(), BeforeProfile: pf.Writable(),
		AfterProject: nativePrepareProject(pr.Writable()), AfterProfile: desired,
		Bridge: nw.Writable(), Quarantine: acl,
	}
	if err = scopedRetirementPreimageWrite(d, preimage); err != nil {
		return err
	}
	if err = c.CreateNetworkACL(acl); err != nil {
		return err
	}
	if err = own.UpdateProject(r.Project, nativePrepareProject(pr.Writable()), pt); err != nil {
		return err
	}
	if err = own.UpdateProfile("default", desired, ft); err != nil {
		return err
	}
	for i := 0; i < 2; i++ {
		if _, _, _, err = nativePeerFreshQuarantineRead(ctx, c, r, ""); err != nil {
			return err
		}
		all, err := c.GetInstancesAllProjects(api.InstanceTypeAny)
		if err != nil || len(all) != 0 {
			return errors.New("quarantine setup observed unexpected native consumer")
		}
		if err = NativePeerGuardKernelReadback(ctx, plan); err != nil {
			return err
		}
	}
	r.Status = "QUARANTINE_VERIFIED"
	return nativePeerFreshJournalWrite(d, r)
}

func nativePeerFreshQuarantineReceipt(r nativePeerFreshJournalRecord) error {
	d, err := nativePeerFreshJournalOpenAt(nativePeerFreshJournalDir, 0)
	if err != nil {
		return err
	}
	defer d.Close()
	for _, status := range []string{"QUARANTINE_DISPATCH_UNKNOWN", "QUARANTINE_VERIFIED"} {
		got, err := nativePeerFreshJournalRead(d, status)
		r.Status = status
		if err != nil || got != r {
			return errors.New("quarantine setup not exactly verified")
		}
	}
	return nil
}
