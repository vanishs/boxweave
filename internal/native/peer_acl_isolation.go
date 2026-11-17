package native

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
	"time"

	incus "github.com/lxc/incus/v7/client"
	"github.com/lxc/incus/v7/shared/api"
)

// PeerACLBox is an exact persisted native binding, never raw MCP input.
// Fixed IPv4/IPv6 addresses must remain claimed while the Box is stopped.
type PeerACLBox struct{ Name, UUID, MAC, IPv4, IPv6 string }

// PeerACLBridgeCheck must independently verify the root-visible bridge
// UsedBy, foreign NICs and native lease ownership. A project-only view is
// insufficient. There is no production implementation of this callback yet.
type PeerACLBridgeCheck func(context.Context) error

type peerACLDriver interface {
	names() ([]string, error)
	get(string) (*api.Instance, string, error)
	disableAutostart(string, api.InstancePut, string) (string, error)
	stop(string) (string, error)
	terminal(string) (bool, error)
}
type peerACLSDK struct{ c incus.InstanceServer }

func (p peerACLSDK) names() ([]string, error)                       { return p.c.GetInstanceNames(api.InstanceTypeAny) }
func (p peerACLSDK) get(name string) (*api.Instance, string, error) { return p.c.GetInstance(name) }
func (p peerACLSDK) disableAutostart(name string, put api.InstancePut, etag string) (string, error) {
	op, err := p.c.UpdateInstance(name, put, etag)
	if err != nil {
		return "", err
	}
	if op == nil || op.Get().ID == "" {
		return "", ErrUnverified
	}
	return op.Get().ID, nil
}
func (p peerACLSDK) stop(name string) (string, error) {
	op, err := p.c.UpdateInstanceState(name, api.InstanceStatePut{Action: "stop", Timeout: 30}, "")
	if err != nil {
		return "", err
	}
	if op == nil || op.Get().ID == "" {
		return "", ErrUnverified
	}
	return op.Get().ID, nil
}
func (p peerACLSDK) terminal(id string) (bool, error) {
	op, _, err := p.c.GetOperation(id)
	if err != nil {
		return false, err
	}
	if op == nil {
		return false, ErrUnverified
	}
	if !op.StatusCode.IsFinal() {
		return false, nil
	}
	if op.StatusCode != api.Success {
		return false, ErrUnverified
	}
	return true, nil
}

// IsolateSharedPeerACL is a native pre-PUT safety barrier, NOT an ACL write.
// The caller must own durable locks for ALL project Boxes and the shared ACL.
// No public Peer handler exists yet. A failed call must retain UNKNOWN and
// never restart guests or restore autostart without independent verification.
func (r Reader) IsolateSharedPeerACL(ctx context.Context, bridge, acl string, boxes []PeerACLBox, check PeerACLBridgeCheck) error {
	if check == nil {
		return ErrUnverified
	}
	c, err := r.confined(ctx)
	if err != nil {
		return err
	}
	return isolatePeerACL(ctx, peerACLSDK{c: c}, r.Project, bridge, acl, boxes, check)
}
func peerACLWait(ctx context.Context, d peerACLDriver, id string) error {
	if id == "" {
		return ErrUnverified
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done, err := d.terminal(id)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		t := time.NewTimer(400 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}
func peerACLMAC(s string) (string, error) {
	m, err := net.ParseMAC(s)
	if err != nil || len(m) != 6 || m[0]&1 != 0 {
		return "", ErrUnsafeNetwork
	}
	return strings.ToLower(m.String()), nil
}
func peerACLCheckBox(inst *api.Instance, project, bridge, acl string, b PeerACLBox, stopped bool) error {
	if inst == nil || inst.Name != b.Name || inst.Config["volatile.uuid"] != b.UUID ||
		(inst.Project != "" && inst.Project != project) || !nicPrivate(inst.ExpandedDevices) ||
		len(inst.Profiles) != 1 || inst.Profiles[0] != "default" {
		return ErrNotOwned
	}
	nic, ok := inst.ExpandedDevices["eth0"]
	if !ok || nic["type"] != "nic" || nic["network"] != bridge ||
		nic["security.acls"] != acl || nic["security.acls.default.ingress.action"] != "reject" {
		return ErrUnsafeNetwork
	}
	// An observed DHCP address is NOT a native static lease reservation.
	local, ok := inst.Devices["eth0"]
	if !ok || local["type"] != "nic" || local["ipv4.address"] != b.IPv4 || local["ipv6.address"] != b.IPv6 ||
		nic["ipv4.address"] != b.IPv4 || nic["ipv6.address"] != b.IPv6 {
		return ErrUnsafeNetwork
	}
	expected, err := peerACLMAC(b.MAC)
	if err != nil {
		return err
	}
	observed := false
	for _, raw := range []string{nic["hwaddr"], inst.Config["volatile.eth0.hwaddr"]} {
		if raw == "" {
			continue
		}
		got, e := peerACLMAC(raw)
		if e != nil || got != expected {
			return ErrUnsafeNetwork
		}
		observed = true
	}
	if !observed {
		return ErrUnsafeNetwork
	}
	if stopped && (inst.Status != "Stopped" || inst.Config["boot.autostart"] != "false") {
		return ErrUnverified
	}
	if !stopped && inst.Status != "Running" && inst.Status != "Ready" && inst.Status != "Stopped" {
		return ErrUnverified
	}
	return nil
}
func isolatePeerACL(ctx context.Context, d peerACLDriver, project, bridge, acl string, boxes []PeerACLBox, check PeerACLBridgeCheck) error {
	if check == nil || project == "" || bridge == "" || acl == "" || len(boxes) == 0 || len(boxes) > 64 {
		return ErrUnverified
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	names, err := d.names()
	if err != nil {
		return err
	}
	if len(names) != len(boxes) {
		return fmt.Errorf("project instance inventory drift: %w", ErrNotOwned)
	}
	owned := make(map[string]PeerACLBox, len(boxes))
	ips, macs := map[string]bool{}, map[string]bool{}
	for _, b := range boxes {
		if !safeName.MatchString(b.Name) || b.UUID == "" || owned[b.Name].UUID != "" {
			return ErrNotOwned
		}
		v4, e4 := netip.ParseAddr(b.IPv4)
		v6, e6 := netip.ParseAddr(b.IPv6)
		mac, em := peerACLMAC(b.MAC)
		if e4 != nil || e6 != nil || em != nil || !v4.Is4() || !v6.Is6() ||
			!v4.IsGlobalUnicast() || !v6.IsGlobalUnicast() || ips[b.IPv4] || ips[b.IPv6] || macs[mac] {
			return ErrUnsafeNetwork
		}
		ips[b.IPv4], ips[b.IPv6], macs[mac] = true, true, true
		owned[b.Name] = b
	}
	slices.Sort(names)
	for _, name := range names {
		b, ok := owned[name]
		if !ok {
			return ErrNotOwned
		}
		inst, _, err := d.get(name)
		if err != nil {
			return err
		}
		if err = peerACLCheckBox(inst, project, bridge, acl, b, false); err != nil {
			return err
		}
	}
	if err := check(ctx); err != nil {
		return err
	}
	// Disable daemon-restart autostart on EVERY consumer before stopping any.
	// On uncertain native write, Worker holds UNKNOWN; never retry or PUT ACL.
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		inst, tag, err := d.get(name)
		if err != nil {
			return err
		}
		if err = peerACLCheckBox(inst, project, bridge, acl, owned[name], false); err != nil {
			return err
		}
		if inst.Config["boot.autostart"] != "false" {
			put := inst.InstancePut
			put.Config = make(api.ConfigMap, len(inst.Config)+1)
			for k, v := range inst.Config {
				put.Config[k] = v
			}
			put.Config["boot.autostart"] = "false"
			id, err := d.disableAutostart(name, put, tag)
			if err != nil {
				return fmt.Errorf("autostart native write uncertain for %s: %w", name, err)
			}
			if err = peerACLWait(ctx, d, id); err != nil {
				return err
			}
		}
		after, _, err := d.get(name)
		if err != nil {
			return err
		}
		if err = peerACLCheckBox(after, project, bridge, acl, owned[name], false); err != nil {
			return err
		}
		if after.Config["boot.autostart"] != "false" {
			return ErrUnverified
		}
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return err
		}
		inst, _, err := d.get(name)
		if err != nil {
			return err
		}
		if err = peerACLCheckBox(inst, project, bridge, acl, owned[name], false); err != nil {
			return err
		}
		if inst.Config["boot.autostart"] != "false" {
			return ErrUnverified
		}
		if inst.Status != "Stopped" {
			id, err := d.stop(name)
			if err != nil {
				return fmt.Errorf("native stop uncertain for %s: %w", name, err)
			}
			if err = peerACLWait(ctx, d, id); err != nil {
				return err
			}
		}
	}
	for _, name := range names {
		inst, _, err := d.get(name)
		if err != nil {
			return err
		}
		if err = peerACLCheckBox(inst, project, bridge, acl, owned[name], true); err != nil {
			return err
		}
	}
	if err := check(ctx); err != nil {
		return err
	}
	return nil
}
