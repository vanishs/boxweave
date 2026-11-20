package native

import (
	"context"
	"net/netip"
	"strings"
	"time"

	"github.com/lxc/incus/v7/shared/api"
)

type ReadyAddress struct{ Family, Address, Bridge string }
type ReadyObservation struct {
	Running         bool
	NetworkObserved bool
	Started         time.Time
	Addresses       []ReadyAddress
}

// ObserveReadiness is read-only and binds the snapshot to the exact instance.
// Do not interpret guest-reported addresses as public connectivity.
func (r Reader) ObserveReadiness(ctx context.Context, name, uuid string, network bool) (ReadyObservation, error) {
	if !network {
		_, inst, _, e := r.confinedExactBox(ctx, name, uuid)
		if e != nil {
			return ReadyObservation{}, e
		}
		return ReadyObservation{Running: inst.Status == "Running" || inst.Status == "Ready", Started: inst.LastUsedAt}, nil
	}
	obs, e := r.GetExactBoxFull(ctx, name, uuid)
	if e != nil {
		return ReadyObservation{}, e
	}
	return readinessObservation(obs.InstanceFull)
}
func readinessObservation(full *api.InstanceFull) (ReadyObservation, error) {
	if full == nil {
		return ReadyObservation{}, ErrUnverified
	}
	o := ReadyObservation{Running: full.Status == "Running" || full.Status == "Ready", Started: full.LastUsedAt}
	if !o.Running {
		return o, nil
	}
	if !nicPrivate(full.ExpandedDevices) {
		return o, ErrUnsafeNetwork
	}
	if full.State == nil || (full.State.Status != "Running" && full.State.Status != "Ready") {
		return o, nil
	}
	// Current owned configuration uses exactly one private eth0 NIC.
	d := full.ExpandedDevices["eth0"]
	if d["type"] != "nic" || d["network"] == "" {
		return o, ErrUnsafeNetwork
	}
	nic, ok := full.State.Network["eth0"]
	if !ok || nic.State != "up" {
		return o, nil
	}
	mac := full.Config["volatile.eth0.hwaddr"]
	if mac == "" {
		mac = d["hwaddr"]
	}
	if mac == "" || !strings.EqualFold(mac, nic.Hwaddr) {
		return o, ErrUnsafeNetwork
	}
	o.NetworkObserved = true
	for _, a := range nic.Addresses {
		if a.Scope != "global" {
			continue
		}
		ip, e := netip.ParseAddr(a.Address)
		if e != nil || !ip.IsGlobalUnicast() || ip.Is4In6() {
			continue
		}
		f := ""
		if a.Family == "inet" && ip.Is4() {
			f = "ipv4"
		}
		if a.Family == "inet6" && ip.Is6() {
			f = "ipv6"
		}
		if f == "" {
			continue
		}
		// Static address pins must match. Dynamic addresses are observations
		// from the exact private anti-spoofed NIC, not grants or routability.
		if want := d[f+".address"]; want != "" && want != ip.String() {
			continue
		}
		o.Addresses = append(o.Addresses, ReadyAddress{f, ip.String(), d["network"]})
	}
	return o, nil
}

// Fixed probe only. No caller argv or stdout/stderr spool. The guest command
// has a short deadline; absence of native receipt is always ambiguous.
func (r Reader) DispatchReadinessProbe(ctx context.Context, name, uuid string) (string, error) {
	c, inst, _, e := r.confinedExactBox(ctx, name, uuid)
	if e != nil {
		return "", e
	}
	if inst.Status != "Running" && inst.Status != "Ready" {
		return "", ErrNativeUnsupported
	}
	if e = ctx.Err(); e != nil {
		return "", e
	}
	op, e := c.ExecInstance(name, api.InstanceExecPost{
		Command:   []string{"/usr/bin/timeout", "--signal=TERM", "--kill-after=1s", "--", "5s", "/bin/true"},
		WaitForWS: false, RecordOutput: false,
	}, nil)
	if e != nil {
		return "", e
	}
	id := op.Get().ID
	if id == "" {
		return "", ErrUnverified
	}
	return id, nil
}

func ReadinessProbeExit(op *api.Operation) (int, error) {
	if op == nil || (op.StatusCode != api.Success && op.StatusCode != api.Failure) {
		return 0, ErrUnverified
	}
	var n int
	switch v := op.Metadata["return"].(type) {
	case int:
		n = v
	case float64:
		if v < 0 || v > 255 || float64(int(v)) != v {
			return 0, ErrUnverified
		}
		n = int(v)
	default:
		return 0, ErrUnverified
	}
	if n < 0 || n > 255 {
		return 0, ErrUnverified
	}
	if op.StatusCode == api.Failure && !(n == 126 && op.Err == "Command not executable" || n == 127 && op.Err == "Command not found") {
		return 0, ErrUnverified
	}
	return n, nil
}
