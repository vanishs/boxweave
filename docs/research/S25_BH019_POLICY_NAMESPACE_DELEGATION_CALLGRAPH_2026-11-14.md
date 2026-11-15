# BH-019 — AppArmor Policy Namespace: recursive delegation call graph and safe experiment gates
2026-11-14 PT. Scope: AppArmor, UserNS, Incus/LXC, BoxWeave's default safe nesting surface. **No distro/version matrix, patch tracking, BPF or VM route**.

> **Outcome:** The probable native mechanism is now more precise than `apparmor_parser rc243`: an already-stacked Incus/LXC host does not create or configure a new AppArmor child policy namespace, so the next UserNS observes an old namespace whose numeric AppArmor **level does not increase**, then the kernel's view/admin gate rejects operations. This is **source-grounded and lab-consistent**, not proof that lifting any guard is secure or will work. BH-019 remains BLOCKED.

## Primary native evidence already established

- The controlled [N×M v2](../verification/S25_BH019_NXM_V2_PLAN_RESULTS_2026-11-14.md) (L0 RC5/L1 RC4) recorded `/sys/kernel/security/apparmor/.ns_level=1` **in both** own-ns L0 Box and outer-ns-inheriting L1 Box, and still 1 after two additional transient UserNS creations. Non-root UID1001/1002/65534 `unshare(CLONE_NEWUSER)` L0 yields EACCES, L1 succeeds; that is a separately unresolved permission path, not proof of the AppArmor policy-admin root cause. In both Boxes local UID/GID ownership/DAC basics passed.
- The targeted [view-capable causal matrix](../verification/S25_BH019_POLICY_VIEW_CAPABLE_CAUSAL_MATRIX_2026-11-14.md) documented **L0 original UserNS** root with local `CAP_MAC_ADMIN` opening `/sys/kernel/security/apparmor/profiles`, while *one additional UserNS* retains the visible AA label/namespace and capabilities but gets **EACCES**. L1 Box, inheriting an older AA namespace, gets EACCES already at root. L0 UID1001/GID0/CapEff0 can open profiles but UID1001/GID1001 cannot: **view/root-in-UserNS != admin capability**. All tests were read-only, exact native receipts verified, disposable Boxes deleted.
- Prior genuine L2 Incus network-profile load failed with `apparmor_parser` exit **243**, "not policy admin". This does not establish that a secure descendant policy namespace *can* be created and delegated in the tested topology.

## End-to-end source graph (distinct controls)

**G0 — BoxWeave native creation, no privilege/profiles override.**
`internal/native/box.go` builds Incus create config, setting **only** `security.nesting=true` when requested. `internal/native/profile_config.go` allows an empty default Profile or exact documented Incus per-user raw.idmap; it rejects `raw.lxc`, arbitrary `raw.idmap` and unreviewed profile config. `profile_config_test.go` has an explicit rejection of `raw.lxc=lxc.apparmor.profile=unconfined`. Accordingly a BoxWeave-level "toggle" cannot legitimately create AppArmor policy delegation using the current API and safety contract.

**G1 — Incus detection/selection.**
The native Incus 7.6 `internal/server/instance/drivers/driver_lxc.go` branch:
- If `OS.AppArmorConfined || !OS.AppArmorAdmin`, use the **current process profile** for the next container (no newly selected per-Box namespace).
- Otherwise select generated `InstanceProfileName`; append `profile//&:InstanceNamespaceName:` only if **`AppArmorStacking && !AppArmorStacked`**. Thus already stacked means **no new namespace suffix**.
- Important false lead: `lxc.cap.drop` including `mac_admin mac_override` on stacked setups sits **inside `d.IsPrivileged()`** (not the path of the unprivileged lab Boxes). It cannot explain these nonprivileged Boxes' `CapEff=000001ffffffffff` observation.

Sources: [Incus 7.6 `driver_lxc.go` conditional](https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/instance/drivers/driver_lxc.go), including the `d.IsPrivileged()` `lxc.cap.drop` block.

**G2 — Incus namespace creation and profile-template privileges.**
- `internal/server/apparmor/apparmor.go createNamespace()`: **returns without `mkdir` if `!AppArmorStacking || AppArmorStacked`**. Otherwise creates its child under `/sys/kernel/security/apparmor/policy/namespaces/<instance-name>`.
- `internal/server/apparmor/instance.go`: generated `lxcProfileTpl` gets `feature_stacking = AppArmorStacking && !AppArmorStacked`. Thus even if a developer forces just one namespace creation guard to the other branch, the rendered AppArmor profile's **per-namespace transition rule controls remain gated**.
- The usual generated template's `feature_stacking` branch contains scoped `change_profile` patterns into its own designated namespace; the `security.nesting` flag is **not an authorization to inject arbitrary AppArmor transitions**.
Sources: [Incus `apparmor.go`](https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/apparmor/apparmor.go), [`instance.go`](https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/apparmor/instance.go).

**G3 — independent LXC implementation, same safety guard.**
The liblxc `src/lxc/lsm/apparmor.c` generated-profile path includes `if (aa_can_stack && !aa_is_stacked)` before including namespace-scoped `change_profile` rules; `make_apparmor_namespace` returns without creating a namespace if `!aa_can_stack || aa_is_stacked`. This is not BoxWeave-specific. However, BoxWeave uses Incus-selected native profiles rather than directly opting into LXC's `generated` path; do **not** conflate the two call sites.
Source: [LXC `apparmor.c`](https://github.com/lxc/lxc/blob/main/src/lxc/lsm/apparmor.c).

**G4 — kernel namespace allocation and parent permission.**
- In `security/apparmor/apparmorfs.c`, the special `policy/namespaces` `ns_mkdir_op` **calls `aa_may_manage_policy(... AA_MAY_LOAD_POLICY)` before creating a child**. This is *not* an ordinary unguarded filesystem mkdir.
- In `security/apparmor/policy_ns.c`, `__aa_create_ns` explicitly sets `child->level = parent->level + 1`. The target child level can therefore be prepared by a still-authorized **parent-layer actor** before an inner UserNS operates at that level, in principle.
- In `security/apparmor/policy.c`, `aa_policy_view_capable` for a noninit UserNS requires `root_in_user_ns`, target visibility, `unprivileged_userns_apparmor_policy != 0`, **and `user_ns->level == labels_view(label)->level`**. `aa_policy_admin_capable` additionally demands valid `CAP_MAC_ADMIN` through both kernel and AA mediation, plus `!aa_g_lock_policy`. `aa_may_manage_policy` denies with "not policy admin" if admin gate false. The source explains the read-only EACCES and earlier parser243, while the **actual UserNS numeric level is not directly instrumented** in L0/L1 tests.
Sources: [apparmorfs.c `ns_mkdir_op`](https://codebrowser.dev/linux/linux/security/apparmor/apparmorfs.c.html), [policy_ns.c `__aa_create_ns`](https://codebrowser.dev/linux/linux/security/apparmor/policy_ns.c.html), [policy.c `aa_policy_admin_capable`](https://linux.googlesource.com/linux/kernel/git/torvalds/linux/+/54ac9ff8f1196afc49d644a1625e0af1c9fcf7f5/security/apparmor/policy.c).

## Decision matrix and what it does NOT establish

| Parent execution context | Actual upstream route | Empirical outcome / decision |
| --- | --- | --- |
| AppArmor available, userns-capable, `Stacking=true`, `Stacked=false`, daemon has policy admin and is not confined | Incus creates child AA namespace; generated profile uses namespace stacking suffix and includes target-scoped change_profile rules | Matches single-level L0→L1 success. **Not** proof of full ancestor/sibling separation. |
| Already `Stacked=true`, daemon has parent-namespace admin | Incus **skips** child AA namespace creation; generated profile disables feature_stacking and namespace suffix | Matches L1→L2 containing Box inheriting level1 AA policy namespace. |
| Daemon itself confined or lacks AppArmorAdmin | Incus reuses daemon current profile as LXC profile; createNamespace additionally has independent stack-state guard | Matches observed enforced deep Box and inability to administer next policy; no basis for assuming further recursion. |
| New UserNS created inside current Box without new AA policy namespace | Kernel sees unchanged AA label/view, same `.ns_level`, root/caps may be fresh, but policy view gate fails at new UserNS depth | **Directly witnessed:** L0 profile-list open OK→EACCES after UserNS creation, despite full local CapEff. |
| Force-enable one `Stacked` guard only | G1 / G2 / G3 may disagree; LSM policy view, transitions, and parent profile enforcement still apply | **Unverified, unsafe as a general fix**. Do not suggest removing one guard or relaxing ancestor. |

### Hypothesis: pre-child, least-privilege policy delegation — NOT a tested solution

A secure solution may require **coordinated**:
1. A parent that still has **correct same-level policy-admin privileges** to allocate **exactly one named child AppArmor policy namespace** (the kernel increments the level by 1), without permitting uncontrolled sibling/ancestor policy administration.
2. A parent-controlled **bounded AppArmor profile transition** into that child namespace, retaining all enforced parent restrictions; no `unconfined` fallback that breaks ancestor enforcement.
3. Correct matching **child UserNS level and child AppArmor label/view namespace level**, so the child may view and, only where authorized, load its **own** profiles. Distinguish UserNS local CapEff from AppArmor policy administration.
4. Verified native policy lifecycle (owned load/replace/remove) and **explicit** failed attempts at ancestor/sibling policy modification, plus full independent nested incusd first bridge and next Box.
5. Explicit failure classification if kernel/LXC/Incus cannot supply this contract; never silently mutate a parent's AppArmor namespace.

**Major feasibility caveat:** an AppArmor maintainer documented that the LSM's current user-namespace distinction only properly handles the root namespace and its first child in this class of deep-nesting problem, independently of abstract policy stacking support. Therefore **same-level namespace allocation alone may not overcome all LSM limitations**, even if the above proposal fits a source-level call graph. Do not claim arbitrary-depth support based on level fields, or that the integration is a minor BoxWeave config change. Reference: [Canonical LXD #14770 maintainer dialogue](https://github.com/canonical/lxd/issues/14770).

## Why no new policy-mutation native probe was run in this continuation

The last independent native L0/L1 tests already resolved the read-only predicates and validated direct `.ns_level`, `profiles_open`, UID/GID/CAP contrasts with exact original receipts and cleanup. The **next discriminating experiment** is an actual `mkdir` of a child AA namespace and an authorized, constrained profile/label transition. These mutate **kernel AppArmor policy state**, and on an inherited level1 namespace could affect shared ancestor policy. Existing user/session guardrail explicitly prohibits writing into inherited L1 AppArmor policy or changing shared host. This continuation therefore performs only **source audit and a new falsifiable acceptance plan**, rather than blindly executing a stateful policy trial in L0/L1. No fixture creation or policy writes here.

### Exact conditions for a future controlled native go/no-go

Only if an owner-authorized, **disposable parent-controlled policy namespace** can be proven distinct from any existing Box/policy/ancestor:
- Prove parent policy remains enforceable and cannot be disabled by child, sibling policies inaccessible.
- Pin exact original native create/exec/policy test IDs; reject if namespace owner/label/view ambiguous or securityfs mount points into old namespace.
- Create one uniquely named child namespace and **limit transitions** to that name; inspect the child AA namespace's `.ns_level` and own new UserNS; require root `profiles_open` positive in child, while nonroot should not get policy-admin.
- Only then consider benign local profile load/remove in newly owned child AA namespace, followed by explicit ancestor/sibling denial controls and rollback.
- End with actual L2 incusd network policy lifecycle and L3 Box only after all security negative checks pass. On failure, stop and preserve original receipts, not broad fallback to privileged/unconfined/host-global overrides.

## Verdict

**BH-019: BLOCKED.** The source path provides a *coherent* dual-blocker explanation (no child namespace creation/transition in stacked Incus+LXC; kernel level-aligned view/admin privileges required), with a source-derived feasible experiment design, **not** evidence of a safe implementation. No product/source change should be merged from this research; PR #146 remains Draft. No Actions acceptance claimed for doc-only changes.
