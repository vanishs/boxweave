package install

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
	"github.com/vanishs/boxweave/internal/native"
	"github.com/vanishs/boxweave/internal/peerscope"
	"github.com/vanishs/boxweave/internal/state"
)

type scopedCreateNativeProof struct {
	Original   state.ScopedCreateOriginalProof
	Binding    state.VerifiedCreateBinding
	UUID, ETag string
	Instance   api.InstancePut
	ACL        api.NetworkACLsPost
}

func nativeScopedCreateOriginalInactive(ctx context.Context, c incus.InstanceServer, id string) error {
	for i := 0; i < 2; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		op, _, err := c.GetOperation(id)
		if err == nil {
			if op == nil || op.ID != id || op.StatusCode != api.Success {
				return errors.New("original native Create not proven complete")
			}
		} else if !api.StatusErrorCheck(err, http.StatusNotFound) {
			return err
		}
		ops, err := c.GetOperations()
		if err != nil {
			return err
		}
		for _, o := range ops {
			if o.ID == id && !o.StatusCode.IsFinal() {
				return errors.New("original native Create still active")
			}
		}
		if i == 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
		}
	}
	return nil
}
func nativeScopedCreateBindingCheck(inst *api.Instance, p state.ScopedCreateOriginalProof, acl string, boot, egress bool) error {
	return nativeScopedCreateShapeCheck(inst, p, acl, boot, egress, false)
}

func nativeScopedCreateShapeCheck(inst *api.Instance, p state.ScopedCreateOriginalProof, acl string, boot, egress, running bool) error {
	if inst == nil || inst.Project != p.Intent.Project || inst.Name != p.Intent.Name || inst.Config["user.boxweave.create_operation"] != p.OriginalID ||
		inst.Config["volatile.uuid"] == "" || inst.Config["volatile.uuid.generation"] != inst.Config["volatile.uuid"] ||
		(!running && inst.Status != "Stopped" || running && inst.Status != "Running" && inst.Status != "Ready") || inst.Config["boot.autostart"] != strconv.FormatBool(boot) ||
		inst.Config["limits.cpu"] != strconv.Itoa(p.Intent.CPU) || inst.Config["limits.memory"] != strconv.FormatInt(p.Intent.MemoryMiB, 10)+"MiB" ||
		inst.ExpandedConfig["security.privileged"] == "true" || len(inst.ExpandedDevices) != 2 {
		return errors.New("scoped native Create identity or stopped/boot preimage changed")
	}
	b := p.Quarantine
	if err := native.ValidatePeerACLBox(inst, p.Intent.Project, b.Bridge, acl, native.PeerACLBox{Name: p.Intent.Name, UUID: inst.Config["volatile.uuid"], MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}, false); err != nil {
		return err
	}
	nic := inst.Devices["eth0"]
	want := "reject"
	if egress {
		want = "allow"
	}
	if nic["security.acls"] != acl || nic["security.acls.default.egress.action"] != want || nic["hwaddr"] != b.MAC ||
		inst.ExpandedDevices["root"]["type"] != "disk" || inst.ExpandedDevices["root"]["path"] != "/" || inst.ExpandedDevices["root"]["size"] != strconv.FormatInt(p.Intent.DiskGiB, 10)+"GiB" {
		return errors.New("scoped Create NIC or disk shape changed")
	}
	return nil
}
func nativeScopedCreateLeaseCheck(c incus.InstanceServer, bridge, name string, b state.VerifiedCreateBinding, preboot bool) error {
	leases, err := c.GetNetworkLeases(bridge)
	if err != nil {
		return err
	}
	v4, v6 := 0, 0
	for _, l := range leases {
		if l.Address != b.IPv4 && l.Address != b.IPv6 && l.Hwaddr != b.MAC && l.Hostname != name {
			continue
		}
		if l.Hostname != name || l.Hwaddr != b.MAC || l.Address != b.IPv4 && l.Address != b.IPv6 || l.Type != "static" && l.Type != "dynamic" {
			return errors.New("foreign or ambiguous native static lease")
		}
		if l.Type == "static" {
			if l.Address == b.IPv4 {
				v4++
			} else {
				v6++
			}
		}
	}
	// Before first start, Incus may not publish its static DHCP records.
	// Require exact reserved NIC on the sole matching native instance and
	// absence of any collision in all-project native device inventory.
	if preboot && v4 == 0 && v6 == 0 {
		all, err := c.GetInstancesAllProjects(api.InstanceTypeAny)
		if err != nil {
			return err
		}
		found := 0
		for _, inst := range all {
			for key, nic := range inst.ExpandedDevices {
				if nic["type"] != "nic" || nic["network"] != bridge {
					continue
				}
				if inst.Name == name && key == "eth0" && nic["hwaddr"] == b.MAC && nic["ipv4.address"] == b.IPv4 && nic["ipv6.address"] == b.IPv6 && inst.Status == "Stopped" && (inst.LastUsedAt.IsZero() || inst.LastUsedAt.Unix() == 0) {
					found++
					continue
				}
				if strings.EqualFold(nic["hwaddr"], b.MAC) || strings.EqualFold(inst.Config["volatile."+key+".hwaddr"], b.MAC) || nic["ipv4.address"] == b.IPv4 || nic["ipv6.address"] == b.IPv6 {
					return errors.New("native preboot address collision")
				}
			}
		}
		if found == 1 {
			return nil
		}
	}
	if v4 != 1 || v6 != 1 {
		return errors.New("missing native static leases")
	}
	return nil
}

// Lifecycle effects preserve the already published peer policy; the
// terminal database snapshot alone does not authorize a native change.
func nativeScopedTerminalPolicy(ctx context.Context, db *state.Store, r nativePeerFreshJournalRecord) (state.ScopedTerminalPeerPolicy, error) {
	var zero state.ScopedTerminalPeerPolicy
	policy, err := db.ReadScopedTerminalPeerPolicy(ctx, r.Project, r.Bridge)
	if err != nil {
		return zero, err
	}
	original, initial, err := nativePeerFreshVerifiedPlan()
	if err != nil || original.Project != r.Project || original.Bridge != r.Bridge {
		return zero, errors.New("scoped lifecycle root identity mismatch")
	}
	_, last, err := scopedPeerBootCurrent(nativePeerFreshJournalDir, 0, original, initial, "")
	if err != nil || !slices.Equal(last.Active, policy.Active) {
		return zero, errors.New("scoped lifecycle policy differs from published root policy")
	}
	return policy, nil
}

// Root verifies the entire dedicated native inventory, including other
// independently owned Boxes. No write or power change to any of them.
func nativeScopedLifecycleInventory(ctx context.Context, c incus.InstanceServer, db *state.Store, r nativePeerFreshJournalRecord, nw *api.Network, pending string, pendingACL string, absentUUID string) error {
	policy, err := nativeScopedTerminalPolicy(ctx, db, r)
	if err != nil {
		return err
	}
	ledger, err := db.ReadNativeNetworkLedger(ctx, r.Project)
	if err != nil {
		return err
	}
	if ledger.OrphanBindings != 0 || ledger.OrphanFences != 0 {
		return errors.New("orphan scoped native ledger")
	}
	all, err := c.GetInstancesAllProjects(api.InstanceTypeAny)
	wantCount := len(ledger.Boxes)
	if absentUUID != "" {
		wantCount--
	}
	if pending != "" {
		wantCount++
	}
	if err != nil || len(all) != wantCount {
		return errors.New("scoped native inventory differs from ledger")
	}
	expected := map[string]bool{peerscope.QuarantineACL: true}
	if pendingACL != "" {
		expected[pendingACL] = true
	}
	own := c.UseProject(r.Project)
	for _, b := range ledger.Boxes {
		if b.UUID == absentUUID {
			continue
		}
		acl, err := peerscope.ACLName(r.Project, r.Bridge, b.UUID)
		if err != nil || b.Incarnation != b.UUID || b.OriginalResource != "box:"+b.Name || b.OriginalCount != 1 || !b.OriginalProof || b.BindingCount != 1 || b.BindingState != "PINNED" || b.Bridge != r.Bridge || b.ACL != acl || b.FenceCount != 2 || b.FenceIPv4 != 1 || b.FenceIPv6 != 1 || expected[acl] {
			return errors.New("scoped ledger lineage or ACL mismatch")
		}
		inst, _, err := own.GetInstance(b.Name)
		if err != nil {
			return err
		}
		if inst.Config["user.boxweave.create_operation"] != b.OriginalID || inst.Project != r.Project {
			return errors.New("scoped native original mismatch")
		}
		if err = native.ValidatePeerACLBox(inst, r.Project, r.Bridge, acl, native.PeerACLBox{Name: b.Name, UUID: b.UUID, MAC: b.MAC, IPv4: b.IPv4, IPv6: b.IPv6}, false); err != nil {
			return err
		}
		want, err := NativePeerScopedACLFromPolicySnapshot(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: acl}, nw, policy.Active, policy.Revoked, NativePeerACLSubject{UUID: b.UUID, IPv4: b.IPv4, IPv6: b.IPv6})
		if err != nil {
			return err
		}
		actual, tag, err := c.GetNetworkACL(acl)
		if err != nil || actual == nil || tag == "" || actual.Project != "default" || !reflect.DeepEqual(actual.Writable(), want.NetworkACLPut) || len(actual.UsedBy) != 1 || !nativePeerGuardOwnedURL(actual.UsedBy[0], "instances", b.Name, r.Project) {
			return errors.New("owned scoped ACL drift")
		}
		expected[acl] = true
	}
	names, err := c.GetNetworkACLNames()
	if err != nil {
		return err
	}
	if len(names) != len(expected) {
		return errors.New("unexpected root ACL inventory")
	}
	for _, n := range names {
		if !expected[n] {
			return errors.New("unowned root ACL")
		}
		delete(expected, n)
	}
	if len(expected) != 0 {
		return errors.New("missing root ACL")
	}
	for _, i := range all {
		if i.Project != r.Project || absentUUID != "" && i.Config["volatile.uuid"] == absentUUID {
			return errors.New("foreign native project consumer")
		}
	}
	return nil
}

func nativeScopedCreateReview(ctx context.Context, c incus.InstanceServer, db *state.Store, id string, r nativePeerFreshJournalRecord) (scopedCreateNativeProof, error) {
	var zero scopedCreateNativeProof
	p, err := db.ReadRootScopedCreateOriginal(ctx, id, r.Project, r.Bridge)
	if err != nil {
		return zero, err
	}
	_, _, nw, err := nativePeerFreshQuarantineRead(ctx, c, r, p.Intent.Name)
	if err != nil {
		return zero, err
	}
	if err = nativeScopedLifecycleInventory(ctx, c, db, r, nw, p.Intent.Name, "", ""); err != nil {
		return zero, err
	}
	own := c.UseProject(r.Project)
	if err = nativeScopedCreateOriginalInactive(ctx, own, p.NativeID); err != nil {
		return zero, err
	}
	inst, tag, err := own.GetInstance(p.Intent.Name)
	if err != nil || tag == "" {
		return zero, errors.New("native Create preimage unavailable")
	}
	if err = nativeScopedCreateBindingCheck(inst, p, peerscope.QuarantineACL, false, false); err != nil {
		return zero, err
	}
	if err = nativeScopedCreateLeaseCheck(c, r.Bridge, p.Intent.Name, p.Quarantine, true); err != nil {
		return zero, err
	}
	uuid := inst.Config["volatile.uuid"]
	b := p.Quarantine
	b.ACL, err = peerscope.ACLName(r.Project, r.Bridge, uuid)
	if err != nil {
		return zero, err
	}
	if _, _, err = c.GetNetworkACL(b.ACL); !api.StatusErrorCheck(err, http.StatusNotFound) {
		return zero, errors.New("scoped ACL collision or absence uncertain")
	}
	acl, err := NativePeerScopedACLFromPolicySnapshot(NativePrepareTarget{Project: r.Project, Network: r.Bridge, ACL: b.ACL}, nw, nil, nil, NativePeerACLSubject{UUID: uuid, IPv4: b.IPv4, IPv6: b.IPv6})
	if err != nil {
		return zero, err
	}
	return scopedCreateNativeProof{p, b, uuid, tag, inst.Writable(), acl}, nil
}

func nativeScopedCreateAttached(ctx context.Context, c incus.InstanceServer, r nativePeerFreshJournalRecord, p scopedCreateNativeProof, boot, egress bool) (*api.Instance, string, error) {
	_, _, nw, err := nativePeerFreshQuarantineRead(ctx, c, r, "")
	if err != nil {
		return nil, "", err
	}
	if nw.Name != p.Binding.Bridge {
		return nil, "", errors.New("scoped native bridge changed")
	}
	inst, tag, err := c.UseProject(r.Project).GetInstance(p.Original.Intent.Name)
	if err != nil || tag == "" || inst.Config["volatile.uuid"] != p.UUID {
		return nil, "", errors.New("scoped native UUID changed")
	}
	if err = nativeScopedCreateBindingCheck(inst, p.Original, p.Binding.ACL, boot, egress); err != nil {
		return nil, "", err
	}
	a, at, err := c.GetNetworkACL(p.Binding.ACL)
	if err != nil || a == nil || at == "" || a.Project != "default" || !reflect.DeepEqual(a.Writable(), p.ACL.NetworkACLPut) || len(a.UsedBy) != 1 || !nativePeerGuardOwnedURL(a.UsedBy[0], "instances", inst.Name, r.Project) {
		return nil, "", errors.New("scoped ACL not exclusively attached")
	}
	if err = nativeScopedCreateLeaseCheck(c, r.Bridge, inst.Name, p.Binding, true); err != nil {
		return nil, "", err
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return nil, "", err
	}
	return inst, tag, nil
}

func nativeScopedCreateApply(ctx context.Context, db *state.Store, id string) (scopedLifecycleRecord, error) {
	var zero scopedLifecycleRecord
	r, _, err := nativePeerFreshVerifiedPlan()
	if err != nil {
		return zero, err
	}
	if err = nativePeerFreshQuarantineReceipt(r); err != nil {
		return zero, err
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return zero, err
	}
	c, err := prepareClient(ctx)
	if err != nil {
		return zero, err
	}
	c = c.UseProject("default")
	p, err := nativeScopedCreateReview(ctx, c, db, id, r)
	if err != nil {
		return zero, err
	}
	path, err := scopedLifecyclePath(id)
	if err != nil {
		return zero, err
	}
	d, err := nativePeerFreshJournalCreateAt(path, 0)
	if err != nil {
		return zero, err
	}
	defer d.Close()
	rec := scopedLifecycleRecord{Schema: 2, Kind: "CREATE", Status: "DISPATCH_UNKNOWN", OriginalID: id, NativeID: p.Original.NativeID,
		HostSHA256: r.HostSHA256, GuardSHA256: r.GuardSHA256, Project: r.Project, Name: p.Original.Intent.Name, UUID: p.UUID, Binding: p.Binding,
		PlanSHA256: nativePrepareStateDigest("scoped-create-v2", p), Start: p.Original.Intent.Start, Internet: p.Original.Intent.Internet, BootAutostart: p.Original.Intent.BootAutostart}
	if err = scopedLifecycleWrite(d, rec); err != nil {
		return zero, err
	}
	again, err := nativeScopedCreateReview(ctx, c, db, id, r)
	if err != nil || !reflect.DeepEqual(again, p) {
		return rec, errors.New("scoped Create preimage changed; preserve UNKNOWN")
	}
	if _, err = nativePeerFreshGuardCurrent(ctx, r.Project, r.Bridge); err != nil {
		return rec, err
	}
	if err = c.CreateNetworkACL(p.ACL); err != nil {
		return rec, err
	}
	a, at, err := c.GetNetworkACL(p.Binding.ACL)
	if err != nil || a == nil || at == "" || a.Project != "default" || len(a.UsedBy) != 0 || !reflect.DeepEqual(a.Writable(), p.ACL.NetworkACLPut) {
		return rec, errors.New("new scoped ACL not exact detached policy")
	}
	own := c.UseProject(r.Project)
	inst, tag, err := own.GetInstance(rec.Name)
	if err != nil || tag != p.ETag || !reflect.DeepEqual(inst.Writable(), p.Instance) || inst.Status != "Stopped" {
		return rec, errors.New("scoped Create preimage changed after ACL create")
	}
	w := inst.Writable()
	w.Devices["eth0"]["security.acls"] = p.Binding.ACL
	op, err := own.UpdateInstance(rec.Name, w, tag)
	if err != nil {
		return rec, err
	}
	if err = op.Wait(); err != nil {
		return rec, err
	}
	for i := 0; i < 2; i++ {
		if _, _, err = nativeScopedCreateAttached(ctx, c, r, p, false, false); err != nil {
			return rec, err
		}
	}
	rec.Status = "ACL_VERIFIED"
	if err = scopedLifecycleWrite(d, rec); err != nil {
		return rec, err
	}
	// Only after the durable exact ACL witness may we restore caller boot
	// intent and egress. No start has been issued in the privileged phase.
	inst, tag, err = nativeScopedCreateAttached(ctx, c, r, p, false, false)
	if err != nil {
		return rec, err
	}
	original, err := db.ReadRootScopedCreateOriginal(ctx, id, r.Project, r.Bridge)
	if err != nil || !reflect.DeepEqual(original, p.Original) {
		return rec, errors.New("original changed before restoring boot intent")
	}
	w = inst.Writable()
	w.Config["boot.autostart"] = strconv.FormatBool(rec.BootAutostart)
	if rec.Internet {
		w.Devices["eth0"]["security.acls.default.egress.action"] = "allow"
	}
	op, err = own.UpdateInstance(rec.Name, w, tag)
	if err != nil {
		return rec, err
	}
	if err = op.Wait(); err != nil {
		return rec, err
	}
	for i := 0; i < 2; i++ {
		if _, _, err = nativeScopedCreateAttached(ctx, c, r, p, rec.BootAutostart, rec.Internet); err != nil {
			return rec, err
		}
		_, _, nw, err := nativePeerFreshQuarantineRead(ctx, c, r, "")
		if err != nil {
			return rec, err
		}
		if err = nativeScopedLifecycleInventory(ctx, c, db, r, nw, rec.Name, p.Binding.ACL, ""); err != nil {
			return rec, err
		}
	}
	rec.Status = "VERIFIED"
	if err = scopedLifecycleWrite(d, rec); err != nil {
		return rec, err
	}
	return rec, nil
}
