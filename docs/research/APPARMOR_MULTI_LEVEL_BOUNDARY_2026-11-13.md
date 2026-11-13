# Incus 7.6 / AppArmor multi-level native Host boundary

Evidence date: 2026-11-13 (Pacific). Scope: disposable isolated nested Incus 7.6 labs.

## Observed topology and outcomes

- Physical dev -> old-main BoxWeave outer Box boxweave-test-ser2-container. The outer Box has a manually set lxc.apparmor.profile=unconfined plus unrelated PCI masking rules. This is not a v1 default and not a safe recommendation.
- In that outer Box, hg-v1-oauth-1113 and bw-aa-stack-test-1113 are sibling L1 containers. With security.nesting=true and default Incus AppArmor, their process profile is unconfined, namespace level=1, .ns_stacked=yes.
- bw-aa-stack-test-1113 installed Zabbly Incus 1:7.6-ubuntu24.04-202611072126. Root managed bridge/dnsmasq, restricted incus-user (UID 1001/GID 990) bridge/dnsmasq, and both ordinary and restricted L2 container creation/exec succeeded.
- Both ordinary L2 bw-aa-inner-stack-1113 and restricted L2 rc-install-lab-1113 have security.nesting=true and an enforced Incus AppArmor profile, with the same AppArmor namespace name/level as their parent L1. Both fail to initialize another Incus-managed bridge when apparmor_parser reports that policy admin privileges are denied.
- The ordinary L2 has no raw.idmap at all. Therefore the root cause cannot be attributed only to the restricted Project UID/GID mapping.
- A separate L1 with raw.lxc=lxc.apparmor.profile=unconfined, bw-aa-test, has namespace level=0, .ns_stacked=no. Root Incus's first dnsmasq ran, but incus-user failed because its dnsmasq AppArmor profile was missing, and the root daemon's nested image extraction failed because its incus_archive profile was missing. It is not a functional fix.

## Source-level explanation

Incus v7.6.0 internal/server/sys/apparmor.go detects an existing confined profile and disables per-container AppArmor profile support. It also detects the unsupported combination of an unprivileged user namespace without policy stacking. internal/server/instance/drivers/driver_lxc.go sets up a new AppArmor stacking namespace for an instance only when AppArmorStacking && !AppArmorStacked. It does not delegate another policy namespace in the already-stacked case. internal/server/apparmor/network.go still attempts to load a dnsmasq profile, which the kernel may reject. These source paths explain both actual failures without inventing an Incus defect or declaring unlimited nesting impossible.

Incus v7.6.0 cmd/incus-user/server.go generates the exact service-user raw.idmap map when allowed. BoxWeave v1 already accepts only this safe mapping or an empty profile configuration. Keep that rule.

Upstream references:

- https://github.com/lxc/incus/blob/v7.6.0/internal/server/sys/apparmor.go
- https://github.com/lxc/incus/blob/v7.6.0/internal/server/instance/drivers/driver_lxc.go
- https://github.com/lxc/incus/blob/v7.6.0/internal/server/apparmor/network.go
- https://github.com/lxc/incus/blob/v7.6.0/cmd/incus-user/server.go
- https://gitlab.com/apparmor/apparmor/-/blob/main/parser/rc.apparmor.functions
- https://linuxcontainers.org/incus/docs/main/reference/instance_options/
- https://linuxcontainers.org/incus/docs/main/reference/projects/

## v1 product change and scope

Before native bootstrap and fresh install-stage writes, the root-local binary now performs a read-only AppArmor/user-namespace preflight. It refuses a named AppArmor confined process (NATIVE_APPARMOR_CONFINED), a user-namespaced unconfined process without stacking (NATIVE_APPARMOR_NO_STACKING), or uninspectable nested confinement (NATIVE_APPARMOR_UNVERIFIED). native-prepare-plan and native-prereq-plan also use the check. It does not edit the kernel, AppArmor, parent Incus, raw.lxc, UID mappings, privileged settings, or public MCP tools.

Passing the check is not an Incus nesting certification, because the parent-side profile cannot be fully inferred from the guest. Physical-host validation, actual policy-loaded dnsmasq, root and restricted Incus, native image/instance lifecycle, network isolation and rollback remain separately required.

Decision: preserve the current v1 default nesting behavior and fail early on the two proven unsupported topologies. A true third-level Host requires a separately reviewed upstream Incus/AppArmor security mechanism and independent real safety testing; no safe supported third-level delegation method has been established for this pinned kernel/Incus combination.
