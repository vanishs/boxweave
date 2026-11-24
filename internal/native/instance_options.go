package native

import (
	"os"
	"strconv"
	"strings"

	"github.com/lxc/incus/v7/shared/api"
)

// InstanceOptions is an intentionally closed vocabulary of native Incus/LXC
// settings. Pointer fields preserve omitted versus explicitly false values.
// The MCP JSON property names are the upstream dotted names, never raw.lxc.
type InstanceOptions struct {
	AppArmorProfile      *string `json:"lxc.apparmor.profile,omitempty" jsonschema:"LXC AppArmor mode. unset omits an override; unchanged requires lowlevel=allow and an enforcing parent; generated needs lowlevel=allow. Never use unconfined to repair recursion; after installing inner incusd, configure INCUS_SECURITY_APPARMOR on that daemon only and verify ancestor enforce."`
	Privileged           *bool   `json:"security.privileged,omitempty" jsonschema:"Privileged container; not equivalent to AppArmor unconfined. Requires separate high-trust authorization and is forbidden for safe recursion."`
	IDMapIsolated        *bool   `json:"security.idmap.isolated,omitempty" jsonschema:"Use a distinct unprivileged UID/GID allocation. True requires verified parent budgets and transactional reservation; cannot combine with security.idmap.base."`
	IDMapSize            *int64  `json:"security.idmap.size,omitempty" jsonschema:"Requested UID/GID range size. Must be at least 65536 and fit both verified parent pools and descendant budgets."`
	IDMapBase            *int64  `json:"security.idmap.base,omitempty" jsonschema:"Explicit UID/GID map base. Requires verified parent bounds and collision-free reservation, including stopped and UNKNOWN boxes."`
	ProtectionDelete     *bool   `json:"security.protection.delete,omitempty" jsonschema:"Protect the instance from deletion. Turn off explicitly with authorization before a deliberate delete; ownership checks still apply."`
	BootAutostart        *bool   `json:"boot.autostart,omitempty" jsonschema:"Automatically start this instance after its Incus daemon restarts; check parent resource quotas and recovery behavior."`
	GuestAPI             *bool   `json:"security.guestapi,omitempty" jsonschema:"Enable or disable the Incus guest API. This does not grant access to the parent daemon socket."`
	AppArmorAllowNesting *bool   `json:"lxc.apparmor.allow_nesting,omitempty" jsonschema:"LXC generated-profile nesting helper, not security.nesting. Requires a verified generated-profile translation in a permitted Project."`
	InterceptMknod       *bool   `json:"security.syscalls.intercept.mknod,omitempty" jsonschema:"Opt-in interception of supported mknod operations; requires host and restricted Project approval."`
	InterceptSetxattr    *bool   `json:"security.syscalls.intercept.setxattr,omitempty" jsonschema:"Opt-in interception of selected setxattr operations; requires host and restricted Project approval."`
	InterceptBPF         *bool   `json:"security.syscalls.intercept.bpf,omitempty" jsonschema:"Opt-in BPF syscall interception. Introduces host attack surface and requires explicit restricted Project permission."`
	InterceptBPFDevices  *bool   `json:"security.syscalls.intercept.bpf.devices,omitempty" jsonschema:"Device-cgroup BPF interception requires security.syscalls.intercept.bpf=true plus host/Project permission."`
}

// ForCreate materializes BoxWeave-owned defaults only for a NEW instance.
// Keep the original omitted-field create intent for historical idempotency replay.
// Never use this for box_update; explicitly disabled autostart stays disabled.
func (o InstanceOptions) ForCreate() InstanceOptions {
	if o.BootAutostart == nil {
		enabled := true
		o.BootAutostart = &enabled
	}
	return o
}

// IsZero determines whether box_update contains a requested new field.
func (o InstanceOptions) IsZero() bool {
	return o.AppArmorProfile == nil && o.Privileged == nil &&
		o.IDMapIsolated == nil && o.IDMapSize == nil && o.IDMapBase == nil &&
		o.ProtectionDelete == nil && o.BootAutostart == nil && o.GuestAPI == nil &&
		o.AppArmorAllowNesting == nil && o.InterceptMknod == nil &&
		o.InterceptSetxattr == nil && o.InterceptBPF == nil && o.InterceptBPFDevices == nil
}

// RequiresStop is intentionally conservative. Incus cannot live-update
// namespace mappings or all interception settings; leave a stopped-instance
// operation as the explicit safe operator path.
func (o InstanceOptions) RequiresStop() bool {
	return o.AppArmorProfile != nil || o.Privileged != nil || o.IDMapIsolated != nil ||
		o.IDMapSize != nil || o.IDMapBase != nil || o.AppArmorAllowNesting != nil ||
		o.InterceptMknod != nil || o.InterceptSetxattr != nil || o.InterceptBPF != nil ||
		o.InterceptBPFDevices != nil
}

// Apply writes only exact audited LXC options when a restricted Project has
// explicitly opted into lowlevel=allow. It never permits arbitrary raw.lxc,
// unconfined, privileged guests, or unleased UID/GID range requests.
func (o InstanceOptions) Apply(config api.ConfigMap, project map[string]string) error {
	if o.Privileged != nil && *o.Privileged {
		return ErrPrivilegedBoxDenied // admin OAuth is not host high-trust authorization
	}
	if !approvedRawLXC(config["raw.lxc"]) {
		return ErrNativeUnsupported
	}
	if o.AppArmorProfile != nil {
		switch *o.AppArmorProfile {
		case "unset": // omit: do not mutate any existing override
		case "unchanged", "generated":
			if project["restricted.containers.lowlevel"] != "allow" {
				return ErrProjectLowlevelDenied
			}
			config["raw.lxc"] = "lxc.apparmor.profile = " + *o.AppArmorProfile
		default: // unconfined has no BoxWeave high-trust grant
			return ErrNativeUnsupported
		}
	}
	if o.AppArmorAllowNesting != nil {
		if o.AppArmorProfile == nil || *o.AppArmorProfile != "generated" {
			return ErrNativeUnsupported
		}
		if project["restricted.containers.lowlevel"] != "allow" {
			return ErrProjectLowlevelDenied
		}
		value := "0"
		if *o.AppArmorAllowNesting {
			value = "1"
		}
		config["raw.lxc"] += "\nlxc.apparmor.allow_nesting = " + value
	}
	if o.IDMapBase != nil || o.IDMapSize != nil ||
		(o.IDMapIsolated != nil && *o.IDMapIsolated) {
		return ErrIDMapReservationUnavailable // no persistent UID+GID pool reservation yet
	}
	if o.IDMapIsolated != nil && !*o.IDMapIsolated {
		config["security.idmap.isolated"] = "false"
	}
	if o.Privileged != nil {
		config["security.privileged"] = "false"
	}
	if o.ProtectionDelete != nil {
		config["security.protection.delete"] = strconv.FormatBool(*o.ProtectionDelete)
	}
	if o.BootAutostart != nil {
		config["boot.autostart"] = strconv.FormatBool(*o.BootAutostart)
	}
	if o.GuestAPI != nil {
		config["security.guestapi"] = strconv.FormatBool(*o.GuestAPI)
	}
	for _, request := range []struct {
		key   string
		value *bool
	}{
		{"security.syscalls.intercept.mknod", o.InterceptMknod},
		{"security.syscalls.intercept.setxattr", o.InterceptSetxattr},
		{"security.syscalls.intercept.bpf", o.InterceptBPF},
		{"security.syscalls.intercept.bpf.devices", o.InterceptBPFDevices},
	} {
		if request.value == nil {
			continue
		}
		if *request.value && project["restricted.containers.interception"] != "allow" {
			return ErrProjectInterceptionDenied
		}
		config[request.key] = strconv.FormatBool(*request.value)
	}
	if config["security.syscalls.intercept.bpf.devices"] == "true" &&
		config["security.syscalls.intercept.bpf"] != "true" {
		return ErrNativeUnsupported
	}
	return nil
}

var instanceOptionNativeKeys = [...]string{
	"security.privileged", "security.idmap.isolated", "security.idmap.size",
	"security.idmap.base", "security.protection.delete", "boot.autostart",
	"security.guestapi", "security.syscalls.intercept.mknod",
	"security.syscalls.intercept.setxattr", "security.syscalls.intercept.bpf",
	"security.syscalls.intercept.bpf.devices",
}

// observedNativeOptions contains only audited option values. Never expose
// unknown user-controlled raw.lxc text to an MCP caller.
func observedNativeOptions(c api.ConfigMap) map[string]string {
	out := make(map[string]string)
	for _, key := range instanceOptionNativeKeys {
		if value, ok := c[key]; ok {
			out[key] = value
		}
	}
	raw := strings.TrimSpace(c["raw.lxc"])
	if approvedRawLXC(raw) && raw != "" {
		parts := strings.Split(raw, "\n")
		out["lxc.apparmor.profile"] = strings.TrimPrefix(parts[0], "lxc.apparmor.profile = ")
		if len(parts) == 2 {
			out["lxc.apparmor.allow_nesting"] = strings.TrimPrefix(parts[1], "lxc.apparmor.allow_nesting = ")
		}
	}
	return out
}

// Matches checks the actual native readback against all effective requested
// option values, including explicitly false booleans. Missing is not false.
func (o InstanceOptions) Matches(observed map[string]string) bool {
	expected := api.ConfigMap{}
	if err := o.Apply(expected, map[string]string{"restricted.containers.interception": "allow", "restricted.containers.lowlevel": "allow"}); err != nil {
		return false
	}
	for key, value := range observedNativeOptions(expected) {
		if observed[key] != value {
			return false
		}
	}
	return true
}

// approvedRawLXC is a strict byte-for-byte server-generated allowlist.
// This is deliberately NOT a general "raw.lxc" parser or passthrough.
func approvedRawLXC(raw string) bool {
	switch raw {
	case "", "lxc.apparmor.profile = unchanged", "lxc.apparmor.profile = generated",
		"lxc.apparmor.profile = generated\nlxc.apparmor.allow_nesting = 0",
		"lxc.apparmor.profile = generated\nlxc.apparmor.allow_nesting = 1":
		return true
	}
	return false
}

func appArmorLabelEnforced(label string) bool {
	return strings.HasSuffix(strings.TrimSpace(label), "(enforce)")
}

// CheckInheritedAppArmor refuses to dispatch an "unchanged" transition from
// an unconfined (or unreadable) parent: inheritance cannot create enforcement.
func (o InstanceOptions) CheckInheritedAppArmor() error {
	if o.AppArmorProfile == nil || *o.AppArmorProfile != "unchanged" {
		return nil
	}
	b, err := os.ReadFile("/proc/self/attr/current")
	if err != nil || !appArmorLabelEnforced(string(b)) {
		return ErrNativeUnsupported
	}
	return nil
}
