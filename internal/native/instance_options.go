package native

import (
 "strconv"

 "github.com/lxc/incus/v7/shared/api"
)

// InstanceOptions is an intentionally closed vocabulary of native Incus/LXC
// settings. Pointer fields preserve omitted versus explicitly false values.
// The MCP JSON property names are the upstream dotted names, never raw.lxc.
type InstanceOptions struct {
 AppArmorProfile *string `json:"lxc.apparmor.profile,omitempty" jsonschema:"LXC AppArmor mode. unset omits an override; unchanged or generated needs a reviewed trusted adapter. Never use unconfined to repair recursion; after installing inner incusd, configure INCUS_SECURITY_APPARMOR on that daemon only and verify ancestor enforce."`
 Privileged *bool `json:"security.privileged,omitempty" jsonschema:"Privileged container; not equivalent to AppArmor unconfined. Requires separate high-trust authorization and is forbidden for safe recursion."`
 IDMapIsolated *bool `json:"security.idmap.isolated,omitempty" jsonschema:"Use a distinct unprivileged UID/GID allocation. True requires verified parent budgets and transactional reservation; cannot combine with security.idmap.base."`
 IDMapSize *int64 `json:"security.idmap.size,omitempty" jsonschema:"Requested UID/GID range size. Must be at least 65536 and fit both verified parent pools and descendant budgets."`
 IDMapBase *int64 `json:"security.idmap.base,omitempty" jsonschema:"Explicit UID/GID map base. Requires verified parent bounds and collision-free reservation, including stopped and UNKNOWN boxes."`
 ProtectionDelete *bool `json:"security.protection.delete,omitempty" jsonschema:"Protect the instance from deletion. Turn off explicitly with authorization before a deliberate delete; ownership checks still apply."`
 BootAutostart *bool `json:"boot.autostart,omitempty" jsonschema:"Automatically start this instance after its Incus daemon restarts; check parent resource quotas and recovery behavior."`
 GuestAPI *bool `json:"security.guestapi,omitempty" jsonschema:"Enable or disable the Incus guest API. This does not grant access to the parent daemon socket."`
 AppArmorAllowNesting *bool `json:"lxc.apparmor.allow_nesting,omitempty" jsonschema:"LXC generated-profile nesting helper, not security.nesting. Requires a verified generated-profile translation in a permitted Project."`
 InterceptMknod *bool `json:"security.syscalls.intercept.mknod,omitempty" jsonschema:"Opt-in interception of supported mknod operations; requires host and restricted Project approval."`
 InterceptSetxattr *bool `json:"security.syscalls.intercept.setxattr,omitempty" jsonschema:"Opt-in interception of selected setxattr operations; requires host and restricted Project approval."`
 InterceptBPF *bool `json:"security.syscalls.intercept.bpf,omitempty" jsonschema:"Opt-in BPF syscall interception. Introduces host attack surface and requires explicit restricted Project permission."`
 InterceptBPFDevices *bool `json:"security.syscalls.intercept.bpf.devices,omitempty" jsonschema:"Device-cgroup BPF interception requires security.syscalls.intercept.bpf=true plus host/Project permission."`
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

// Apply writes only typed, supported keys. Unsupported low-level AppArmor and
// explicit ID maps fail CLOSED: the current restricted Incus Project denies
// raw.lxc and the transactional ID-budget allocator is not yet available.
// Never accept a field and silently leave it un-applied.
func (o InstanceOptions) Apply(config api.ConfigMap, project map[string]string) error {
 if o.Privileged != nil && *o.Privileged {
  return ErrNativeUnsupported // admin OAuth is not host high-trust authorization
 }
 if o.AppArmorProfile != nil {
  switch *o.AppArmorProfile {
  case "unset": // sentinel: omit the override, never pass literal "unset" to LXC
  case "unchanged", "generated", "unconfined":
   return ErrNativeUnsupported // no trusted restricted-Project low-level adapter
  default:
   return ErrNativeUnsupported
  }
 }
 if o.AppArmorAllowNesting != nil {
  return ErrNativeUnsupported // would require forbidden raw.lxc translation
 }
 if o.IDMapBase != nil || o.IDMapSize != nil ||
  (o.IDMapIsolated != nil && *o.IDMapIsolated) {
  return ErrNativeUnsupported // no persistent UID+GID pool reservation yet
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
  key string
  value *bool
 }{
  {"security.syscalls.intercept.mknod", o.InterceptMknod},
  {"security.syscalls.intercept.setxattr", o.InterceptSetxattr},
  {"security.syscalls.intercept.bpf", o.InterceptBPF},
  {"security.syscalls.intercept.bpf.devices", o.InterceptBPFDevices},
 } {
  if request.value == nil { continue }
  if *request.value && project["restricted.containers.interception"] != "allow" {
   return ErrNativeUnsupported
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

// observedNativeOptions is an allowlisted subset only; never expose raw host
// or Incus daemon configuration to a BoxWeave MCP client.
func observedNativeOptions(c api.ConfigMap) map[string]string {
 out := make(map[string]string)
 for _, key := range instanceOptionNativeKeys {
  if value, ok := c[key]; ok { out[key] = value }
 }
 return out
}

// Matches checks the actual native readback against all effective requested
// option values, including explicitly false booleans. Missing is not false.
func (o InstanceOptions) Matches(observed map[string]string) bool {
 expected := api.ConfigMap{}
 if err := o.Apply(expected, map[string]string{"restricted.containers.interception":"allow"}); err != nil {
  return false
 }
 for key, value := range expected {
  if observed[key] != value { return false }
 }
 return true
}
