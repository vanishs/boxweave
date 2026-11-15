# BH-019 continuation — NSFS owner and ephemeral UserNS depth contrasts (2026-11-14 PT)

> Status: **new native observations, NOT policy-delegation PASS**. This experiment tests UserNS scope and permissions without writing or replacing AppArmor policy. Do **not** alter physical host, existing L0/L1 Boxes, historical UNKNOWNs, privilege mode, Incus security profile, or kernel sysctl based on these observations. SELinux intentionally out of scope.

## Isolation and exact environment

Two **new disposable, nonprivileged, nesting=true, internet=false** Ubuntu 24.04.5 Boxes, both with 1 CPU, 1024 MiB, 8 GiB. L0 uses authenticated BoxWeave RC5 `1224845c139177f3986d1ca3d51032849422ed1c`; L1 uses authenticated RC4 `c3023a72986085b77cd98f2fa5ac83ef40abc632`. Their native controllers report Incus 7.6. Both guests report Linux `7.0.0-43-generic`, `Seccomp: 2`, `NoNewPrivs: 0`, `/dev/kvm` absent, `apparmor_restrict_unprivileged_userns=1`, `apparmor_restrict_unprivileged_unconfined=0`. Neither test used privileged Incus, exposed parent/admin sockets, `raw.lxc`, AppArmor override/disable, nor a host policy or sysctl change. The only ephemeral namespace mutations were ordinary bounded `unshare(CLONE_NEWUSER)` subprocesses, which exited before cleanup.

- **L0 created Box:** `bh019-nsfs-l0-1114`; original UUID `c41b67be-66cc-44e5-b1f6-b33163839cce`; observed root UserNS `user:[4026532378]`; AppArmor `.ns_name=incus-user-1001_bh019-nsfs-l0-1114_<var-lib-incus>`; guest-visible process label `unconfined`.
- **L1 created Box:** `bh019-nsfs-l1-1114`; original UUID `0a063a00-054d-4b4b-883b-8a12264e6270`; observed root UserNS `user:[4026533859]`; AppArmor `.ns_name=incus-user-1001_rc3-l1-nested-1113_<var-lib-incus>` (**inherited outer name**); guest-visible label `incus-user-1001_bh019-nsfs-l1-1114_</var/lib/incus> (enforce)`.

## What was tested and actually returned

| Independent test | L0 RC5 fixture | L1 RC4 fixture |
| --- | --- | --- |
| `NS_GET_USERNS` on mount/PID/UTS/IPC/network/cgroup namespaces | All six return an owner UserNS matching own `/proc/self/ns/user` | All six match own `/proc/self/ns/user` |
| `NS_GET_PARENT` on own UserNS | **EPERM** | **EPERM** |
| Container root effective `CAP_MAC_ADMIN` bit 33 | Present (`CapEff=000001ffffffffff`) | Present (same) |
| Root, one ephemeral `unshare -Ur` + bounded child | **PASS**, child UserNS differs, AppArmor NS/name unchanged | **PASS**, child UserNS differs, AppArmor NS/name unchanged |
| Root, **two successive** nested `unshare -Ur --fork` | **PASS**, 3 distinct UserNS in that process tree: `4026532378 → 4026533857 → 4026533921`; AA namespace stays own Box name | **PASS**, `4026533859 → 4026533857 → 4026533921`; AA namespace stays *outer parent* name |
| UID1001/GID corresponding to local guest service group, one `unshare -Ur` | **Denied**: inner tool `CHILD_RC=1`, `Permission denied` | **PASS**: inner tool `CHILD_RC=0` and new UserNS |
| **Raw** `libc.unshare(CLONE_NEWUSER)` as same nonroot UID1001, independent from util-linux | **DENIED errno=13/EACCES**, `CapEff=0` | **PASS rc=0**, new userns `user:[4026533921]`, `CapEff=0` before call |

**Important distinctions.** The BoxWeave `box_exec` wrappers are `SUCCEEDED`, guest exit0 because controlled diagnostic scripts captured the negative inner subprocess/syscall rather than propagate a negative result to the wrapper. The **inner UID1001 L0 UserNS attempt FAILED** and must not be marked as PASS. The exact errno `EACCES` plus the simultaneous L1 allow demonstrate a genuine permission difference, but **not the precise rejecting subsystem**. Different visible AppArmor policy names, parent namespaces, identity mappings and security contexts require native audit/call-path comparison. A `seccomp=2` observation alone does not identify the denying filter, though `EACCES` is consistent with some security checks. No host audit access was requested.

`NS_GET_PARENT` returning EPERM is consistent with Linux `ioctl_ns(2)` scope rules; **no numeric `user_ns->level` or `view_ns->level` was obtained**. Neither `NS_GET_USERNS` nor transient UserNS success is a substitute for such a read. All successive ephemeral UserNS descendants retained their originally visible AppArmor namespace/label: this is direct evidence that **creating UserNS does not automatically create an independently delegated AppArmor policy namespace** on this host, even when nested process UserNS creation succeeds.

**Security implication:** An L1 container root seeing CAP_MAC_ADMIN **inside its UserNS** does not prove policy-admin access in the AppArmor policy namespace. Previous actual `apparmor_parser` exit243 and cross-depth inherited AppArmor namespace remain the decisive negative policy-admin observations. The current root/unprivileged UserNS outcome does not certify profile load/replace/remove, enforcement, sibling isolation or L2/L3 Incus recursion; do **not** misclassify it as BH-019 PASS or loosen the production security boundary.

## Original operation receipts — exact IDs, read back after admission

| Action | L0 BW / native Incus original | L1 BW / native Incus original |
| --- | --- | --- |
| New Box Create | `34d9168b78f7ac18886eb40ae29c4d35` / `302959d0-5fd0-4993-bf8e-9fc623af96d6` | `eb46a687e87224e4703d4743f10beb3d` / `f391204d-0c5e-4f0d-bd93-80ea801788bd` |
| Read-only `NS_GET_USERNS` and `NS_GET_PARENT` + CAP | `d00e3f356333304163d0f28dcb965e77` / `73e16f35-6e08-496c-9a0d-7db187c5e4c8` | `395e63935e9c6a43f8b4094a73c2018a` / `da0ed6d8-8425-4130-a95e-4640fc1bca93` |
| Root one ephemeral `unshare` | `9634d86d3c7b84ca227042ad572940c7` / `986d5ae9-6100-479f-84cb-73963c658a9f` | `be1cc3e975ae95d885676dbd2827fc29` / `3c2fad9c-a0ac-4d80-bae9-7bdf455e275a` |
| Root two successive `unshare` | `a4c546e9e2bec1a0c436ff4e6d806858` / `cde9fd2c-25d5-4dd7-aa77-13dc29682a44` | `c0a56ea87df43cbf8ec33bdb90275ef1` / `701cf4b8-26a0-4732-9605-c9402afa07ef` |
| UID1001 `unshare` with captured inner exit | `03ab7d8087a06f53bfbbd8c4cb2f205a` / `b226aaa4-d709-4777-9f1d-660a6d858464` | `e4ad048b957b3a506f3fc74dd4df9fd6` / `a9aa4f6a-824b-4970-a6a8-231f3b9685de` |
| UID1001 raw syscall and errno | `4c67060c7b7cbdb25d6fe38b2282fe08` / `f6ce0399-c85b-4844-a588-30549109e9a8` | `1b7e2901b27dbfc51ca5fb4ca623e755` / `3d8eaf37-1754-4b71-9a74-b8a6348a3d44` |
| Exact STOP | `bdf82fc63087793509f85599e1d0b19c` / `e26df404-91ac-487c-8f18-fc2acccaaaa4` | `eaa83b2ba575979f58c2a2a01f6d1707` / `d3b687ae-72a5-4a3b-a239-4425a18fe18d` |
| Exact DELETE | `f8f7c16809794c32a3bb5aec37a1ad00` / `d5fd06f4-a82c-443a-9caf-b76587370fb9` | `e37923db5500d75631513a251457ba2a` / `104a284a-6452-41c3-9053-232b9399e701` |

All original create, six individual Exec, stop, and delete operations **per fixture** were read back as `SUCCEEDED`, with true original native Incus IDs and original output. Both exact DELETE finalizers **SUCCEEDED**; subsequent separate `box_list` responses showed the original **4** L0 and **9** L1 pre-existing Boxes, with neither newly created name present. No new UNKNOWN. Do not misread the diagnostic inner unshare failure as a failed parent `box_exec` Operation.

## Upstream context and next discriminating experiments

- Linux [`ioctl_ns(2)` manual](https://man7.org/linux/man-pages/man2/ioctl_ns.2.html) explicitly warns that `NS_GET_PARENT` may give EPERM when the parent namespace is outside the caller's scope.
- The exact [Incus 7.6.0 `driver_lxc.go`](https://sources.debian.org/src/incus/7.6.0-1~exp1/internal/server/instance/drivers/driver_lxc.go) inherits current AppArmor profile when confined/not admin and avoids automatic deeper stacking when `AppArmorStacked`.
- The [Canonical LXD upstream issue #14770](https://github.com/canonical/lxd/issues/14770) remains open/waiting for an AppArmor upstream fix; its `L1→L2` and securityfs observations match the policy-delegation boundary, but do not certify Incus.
- [Incus release log](https://linuxcontainers.org/incus/news/) still lists 7.6 as the latest actual published monthly release as of this evidence window. Merely changing 7.6 patch-level binaries does not demonstrate a kernel AppArmor delegation fix.
- July 2026 AppArmor policy-namespace [resource limits](https://www.mail-archive.com/apparmor%40lists.ubuntu.com/msg13008.html) and [subtree follow-up](https://www.mail-archive.com/apparmor%40lists.ubuntu.com/msg13054.html) are proposals, not evidence of an installed safe delegation feature.

**Priority 1 — separated VM / new native CI kernel, NOT this shared L0/L1 chain:** reproduce default and isolated label candidate under verified enforced parent; obtain privileged kernel-side audit correlation for **both** the denied UID1001 `unshare(CLONE_NEWUSER)` and the policy-admin `apparmor_parser` rc243. Establish real numeric UserNS/policy-NS levels from trustworthy kernel instrumentation **without relaxing host policies**. Follow with per-depth policy load/replace/remove, enforced safe deny and cross-tenant/ancestor negative controls, then separate BPF devices load/attach/enforce and idmap checks. This is not presently runnable inside these fixtures because **both lack `/dev/kvm`**; using a VM would require an independently authorized KVM-capable lab.

**Priority 2 — controlled comparative image / AppArmor rule diagnosis:** pin versions of liblxc/parser, profile ABI and UID/GID mappings, then independently distinguish the new L0 nonroot UserNS `EACCES` from L1 allow, without inferring that broader AppArmor unconfined is secure. A green UserNS test alone never satisfies policy-admin or AppArmor delegation acceptance.

**Verdict:** BH-019 remains **BLOCKED**. No production AppArmor changes, no tested secure L2/L3 independent Incus policy admin and no PR merge.
