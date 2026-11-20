package native

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
)

// ForwardSpec is an instance-owned TCP/UDP proxy, never a network-scoped
// forward, arbitrary guest destination, host command or firewall edit.
type ForwardSpec struct {
	Device               string
	Protocol             string
	HostBind             string
	HostPort             int
	GuestPort            int
	ExternalAcknowledged bool
}

func (f ForwardSpec) Validate() error {
	if !strings.HasPrefix(f.Device, "bwpf-") || len(f.Device) != 21 {
		return ErrNativeUnsupported
	}
	for _, ch := range f.Device[5:] {
		if !strings.ContainsRune("0123456789abcdef", ch) {
			return ErrNativeUnsupported
		}
	}
	if (f.Protocol != "tcp" && f.Protocol != "udp") || f.HostPort < 1 || f.HostPort > 65535 ||
		f.GuestPort < 1 || f.GuestPort > 65535 {
		return ErrNativeUnsupported
	}
	ip := net.ParseIP(f.HostBind)
	if ip == nil || ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() {
		return ErrNativeUnsupported
	}
	// The public actor confirmation flag alone is NOT a trusted host-local
	// approval for binding an external interface. Keep B4 loopback-only until
	// a separate exact-address host approval mechanism is independently accepted.
	if !ip.IsLoopback() {
		return ErrNativeUnsupported
	}
	if f.HostBind != ip.String() {
		return ErrNativeUnsupported
	}
	// Reject all nonstandard loopbacks, so an unintended alias cannot silently
	// make a listener reachable from a different host network namespace.
	if ip.IsLoopback() && f.HostBind != "127.0.0.1" && f.HostBind != "::1" {
		return ErrNativeUnsupported
	}
	return nil
}

func (f ForwardSpec) nativeDevice() map[string]string {
	listenIP := f.HostBind
	if strings.Contains(listenIP, ":") {
		listenIP = "[" + listenIP + "]"
	}
	return map[string]string{
		"type":    "proxy",
		"bind":    "host",
		"listen":  fmt.Sprintf("%s:%s:%d", f.Protocol, listenIP, f.HostPort),
		"connect": f.Protocol + ":127.0.0.1:" + strconv.Itoa(f.GuestPort),
	}
}
func exactForwardDevice(d map[string]string, f ForwardSpec) bool {
	expected := f.nativeDevice()
	if len(d) != len(expected) {
		return false
	}
	for k, v := range expected {
		if d[k] != v {
			return false
		}
	}
	return true
}

func copyInstanceDevices(in api.DevicesMap) api.DevicesMap {
	result := api.DevicesMap{}
	for k, v := range in {
		out := map[string]string{}
		for key, value := range v {
			out[key] = value
		}
		result[k] = out
	}
	return result
}

func forwardDeviceChange(original api.DevicesMap, f ForwardSpec, create bool) (api.DevicesMap, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	result := copyInstanceDevices(original)
	prior, exists := result[f.Device]
	if create {
		if exists {
			return nil, ErrNativeDependents
		}
		// Only a uniquely named, self-owned local instance proxy is modified.
		result[f.Device] = f.nativeDevice()
	} else {
		if !exists || !exactForwardDevice(prior, f) {
			return nil, ErrNativeDependents
		}
		delete(result, f.Device)
	}
	return result, nil
}

// UpdateOwnedForward uses only the confined instance SDK handle. Conditional
// ETag update and an independent caller-side native terminal + postread are
// mandatory; any ambiguous Incus error remains UNKNOWN at the Worker boundary.
func (r Reader) UpdateOwnedForward(ctx context.Context, name, uuid string, f ForwardSpec, create bool) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	c, _, _, err := r.confinedExactBox(ctx, name, uuid)
	if err != nil {
		return "", err
	}
	project, _, err := c.GetProject(r.Project)
	if err != nil {
		return "", err
	}
	if project.Config["restricted"] == "true" && project.Config["restricted.devices.proxy"] != "allow" {
		return "", ErrNativeUnsupported
	}
	inst, etag, err := c.GetInstance(name)
	if err != nil {
		return "", err
	}
	if inst.Config["volatile.uuid"] != uuid || (inst.Project != "" && inst.Project != r.Project) || !nicPrivate(inst.ExpandedDevices) {
		return "", ErrNotOwned
	}
	if inst.Status != "Running" && inst.Status != "Ready" {
		return "", ErrNativeUnsupported
	}
	devices, err := forwardDeviceChange(inst.Devices, f, create)
	if err != nil {
		return "", err
	}
	// Fail if the reserved device name is occupied by a profile device.
	if _, occupied := inst.ExpandedDevices[f.Device]; occupied {
		if _, local := inst.Devices[f.Device]; !local {
			return "", ErrNativeDependents
		}
	}
	put := inst.InstancePut
	put.Devices = devices
	op, err := c.UpdateInstance(name, put, etag)
	if err != nil {
		return "", err
	}
	if op == nil || op.Get().ID == "" {
		return "", ErrUnverified
	}
	return op.Get().ID, nil
}

// ObserveOwnedForward is a read-only postcondition, not proof of listener
// reachability. Positive packet/connection witness is a separate B4 gate.
func (r Reader) ObserveOwnedForward(ctx context.Context, name, uuid string, f ForwardSpec, present bool) error {
	if err := f.Validate(); err != nil {
		return err
	}
	c, inst, _, err := r.confinedExactBox(ctx, name, uuid)
	if err != nil {
		return err
	}
	fresh, _, err := c.GetInstance(name)
	if err != nil {
		return err
	}
	if fresh.Config["volatile.uuid"] != uuid || (fresh.Project != "" && fresh.Project != r.Project) || !nicPrivate(fresh.ExpandedDevices) {
		return ErrNotOwned
	}
	actual, exists := fresh.Devices[f.Device]
	if present {
		if !exists || !exactForwardDevice(actual, f) {
			return ErrUnverified
		}
	} else if exists {
		return ErrUnverified
	}
	if _, occupied := inst.ExpandedDevices[f.Device]; occupied && !present {
		if _, local := inst.Devices[f.Device]; !local {
			return ErrNativeDependents
		}
	}
	return nil
}
