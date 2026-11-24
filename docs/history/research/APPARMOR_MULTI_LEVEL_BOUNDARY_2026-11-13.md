# Incus 7.6 / AppArmor multi-level native Host boundary — archived research

Evidence date: 2026-11-13 (Pacific). Scope: disposable isolated nested Incus 7.6 labs.

> **Historical evidence only.** The read-only AppArmor preflight code discussed below belonged to experimental **Draft PR #136**, was not merged into protected `v1`, and is **not** a requirement or an implemented feature of the released `v1.0.0-rc.1`. The owner decided against proceeding with this heuristic hard-reject change for now. This research is preserved in documentation-only PR #137 so later work can use the native evidence without inheriting the abandoned implementation. Do not mistake preflight results for actual safety certification.

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

## Historical PR #136 experimental code (not adopted in v1)

In **the unmerged PR #136 test binary only**, before native bootstrap and fresh install-stage writes, root-local code performed a read-only AppArmor/user-namespace preflight. It refuses a named AppArmor confined process (NATIVE_APPARMOR_CONFINED), a user-namespaced unconfined process without stacking (NATIVE_APPARMOR_NO_STACKING), or uninspectable nested confinement (NATIVE_APPARMOR_UNVERIFIED). native-prepare-plan and native-prereq-plan also use the check. It does not edit the kernel, AppArmor, parent Incus, raw.lxc, UID mappings, privileged settings, or public MCP tools.

Passing the check is not an Incus nesting certification, because the parent-side profile cannot be fully inferred from the guest. Physical-host validation, actual policy-loaded dnsmasq, root and restricted Incus, native image/instance lifecycle, network isolation and rollback remain separately required.

**Superseded proposal, not a current decision:** preserve the current v1 default nesting behavior and fail early on the two witnessed failure patterns. A true third-level Host requires a separately reviewed upstream Incus/AppArmor security mechanism and independent real safety testing; no safe supported third-level delegation method has been established for this pinned kernel/Incus combination.

## Addendum: direct physical-host v1 RC reproduction (2026-11-13 PT)

**New higher-fidelity test:** The owner created a *new* `bw-v1rc-l1-1113` directly under physical `dev` Incus **7.5.1 server/client**, without the legacy main outer Box or manual raw.lxc AppArmor override. The L1 was unprivileged with `security.nesting=true`; its inner Incus **7.6** initialized a managed bridge/dnsmasq successfully. The owner installed immutable v1.0.0-rc.1 (`a5e7b0e`) with restricted `user-1001`, and the actual ChatGPT `v1rc` OAuth connector created a Running nesting-enabled L2 and executed native guest commands successfully.

Inside the new L1, the process reported `unconfined`, AppArmor namespace level 1 and `.ns_stacked=yes`. In the actual restricted L2, the process reported an **enforced** `incus-user-1001_...` profile **in the same level-1 namespace**. A pinned Incus **7.6** inside L2 installed and started, but its first `incus admin init --auto --storage-backend=dir` failed with `apparmor_parser` exit 243: cannot replace `incus_dnsmasq-incusbr0`, **policy admin privileges denied**. The inner daemon reported that per-container AppArmor profiles are disabled because it is itself protected by AppArmor. No L3 exists. Full original MCP/native IDs, version readbacks, risk caveats and unchanged guest state: [v1 RC physical-host acceptance](../verification/V1_RC_PHYSICAL_HOST_NESTED_APPARMOR_2026-11-13.md).

**Interpretation corrections:**

- The old-main outer Box's unconfined override is **not necessary** for this failure to reproduce; it also occurs under a direct physical-host L1 without a manual AppArmor override. It still does **not** prove that the kernel or Incus universally forbids arbitrarily deep nesting.
- AppArmor `unconfined` observed *within* an L1 AppArmor namespace does not prove absence of outer policy. The difference between L1 and L2 profile labels is not a BoxWeave setting: the parent Incus/profile selection controls it.
- **Version compatibility is OPEN.** The actual physical L0 **server** (not just client) is 7.5.1; both nested daemons are 7.6. The upstream 7.5.1/7.6 code resemblance is suggestive, but no controlled `7.5.1 -> 7.5.1` guest comparison was performed. Do not label mismatch either root cause or excluded cause.
- The previous **hard-reject** preflight proposal above belonged to **unmerged PR #136**, not a validated universal installer requirement. A process-profile heuristic can refuse a potentially functioning host or allow a host whose real initialization fails. In the owner discussion, non-blocking diagnostic reporting plus authoritative actual Incus initialization failures was preferred for further design review; this addendum **does not alter code or claim a final approved policy change**.

Next: compare pinned 7.5.1 versus 7.6 with unchanged physical L0 and separately isolated native guests, then test AppArmor policy-namespace delegation only behind owner-approved, confined security boundaries. No physical L0 daemon upgrade and no AppArmor privilege relaxation is implied.
