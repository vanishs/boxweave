
# S2.5 BH-019 — Managed, safe recursive Incus: development and acceptance plan

**Date:** 2026-11-15 PT  
**PR:** [#146](https://github.com/vanishs/boxweave/pull/146) (Draft; base `v1`)  
**Decision:** Begin product code development immediately, but do not change PR readiness or mark BH-019 security accepted.


## 0. Owner decision (2026-11-15): complete implementation first, L0 smoke, RC6, then deep reacceptance

**This section supersedes the older per-work-package deploy-and-test rhythm in `4/`6 below.** The work packages remain the engineering scope and release-acceptance requirements, but the owner explicitly selected one *integrated implementation phase* before the first new L0 deployment:

1. **Implementation phase:** review latest PR HEAD and existing tests, then implement **all agreed 14 configuration surfaces (13 new; one existing)** plus the BH-019 safe recursion, ID map, network and nested-boot work in a cohesive development pass. Multiple internal commits, local unit/negative tests, code review and staged compilation are welcome; do not deploy or publish an incomplete feature as a successful RC6.
2. **L0 isolated deployment and *basic smoke*:** once integrated code compiles and passes unit/static checks, deploy to a **new owned disposable L0 lab Box or dedicated isolated L0 test instance**, without replacing an existing production/shared L0 service. Read fresh inventory and explicit service ownership before any install. Verify real `box_create` with dotted names, ordinary defaults, strict rejection of unauthorized modes, create→start→exec→get→update→stop→delete, owned network baseline, and selected safe recursion+UID/AppArmor readback. A **known P1 isolation escape, missing ownership check, corrupted terminal operation or insecure privilege fallback blocks the release**; fix then rerun smoke.
3. **Produce RC6 after basic smoke:** use repository's established RC version/tag/artifact conventions, release the **RC6 test candidate**, and record commit SHA, build versions, smoke evidence, untested scenarios and open gaps. RC6 availability **does not imply** BH-019 safe-recursion acceptance, security sign-off, HOST_GO, or `v1` merge.
4. **Repeated RC6 full acceptance and corrections:** test the *installed* RC6 via L0 BoxWeave, nested real Incus and actual HTTP MCP through L5, progressively adding concurrency, negative network tests, ID collision/exhaustion, restart/SIGKILL/ACK loss/UNKNOWN, snapshots, quotas, guest boot recovery, architecture builds and independent Verifier. Fix code and repeat tests; if fixes change the candidate SHA, build a **new traceable candidate** under existing release policy (do not silently claim old RC6 artifacts match new source). Finish same-SHA Actions after real lab gates; only then consider PR readiness/merge.
5. **Status reporting after each major stage:** explicit code SHA, deployment target and ownership, all executable checks (PASS/FAIL/BLOCKED/NOT TESTED), original BoxWeave and native Incus receipts, CI links, cleanup/post-inventory, RC6 artifact identity and next gate. Stop and document a genuinely unsafe blocker instead of widening restricted Project authority.

### 0.1 Exact MCP parameter inventory: 9 primary options + 5 nested advanced booleans

**Naming contract:** use the **original dotted names** as JSON Schema property names and MCP tool-call argument keys, not underscore aliases. Verify both Go MCP SDK schema parsing and **installed ChatGPT HTTP MCP** serialization of literal dotted keys. Names that begin `lxc.` refer to LXC settings, **not direct Incus instance configuration keys**: a trusted BoxWeave translation layer must map them into an Incus/LXC-compatible operation without raw user-controlled configuration or weakening a restricted Project. If the restricted Project or SDK makes an option impossible, fail closed with a clear reason; do not pretend that an accepted JSON field was effectively applied.

**Primary group (9 visible, of which `security.nesting` already exists):**

| MCP field (exact native spelling) | Type / accepted request | Meaning / implementation notes | Admission |
| --- | --- | --- | --- |
| `lxc.apparmor.profile` | `unset` / `unchanged` / `unconfined` / `generated` (string enum) | `unset` is a BoxWeave **sentinel meaning omit override**, never write literal `unset` into LXC. `unchanged` retains current policy; `generated` requests LXC generation; `unconfined` requests a policy change and may fail under ancestor enforce. | Dangerous `unconfined` requires separate high-trust authorization and must be forbidden as an automatic recursion fallback. Under restricted Projects, verify a server-owned narrow mapping rather than exposing `raw.lxc`. |
| `security.privileged` | optional boolean | Privileged container; explicitly **not** equivalent to AppArmor unconfined. | High-trust explicit authorization only, subject to `restricted.containers.privilege`; not permitted for safe managed-recursion acceptance. |
| `security.nesting` | optional boolean **already implemented** | Enable Incus container nesting, distinct from LXC generated-profile nesting support. | Keep existing behavior / compatibility. Do not treat it as automatic recursive-mode opt-in. |
| `security.idmap.isolated` | optional boolean | Request unique Incus-managed unprivileged ID allocation. | Coordinate with transactionally reserved BW UID/GID range; cannot combine with explicit `security.idmap.base`. |
| `security.idmap.size` | optional integer | Delegate bounded UID/GID range adequate for actual child and any descendants. | Checked limits, kernel maps, parent owned pool, exhaustion and lifecycle. |
| `security.idmap.base` | optional integer | Request an explicit host-relative mapping base; not guaranteed isolated by Incus. | Validate parent pool bounds, siblings/foreign occupants, disjointness, overflow; never trust stopped-instance admission alone. |
| `security.protection.delete` | optional boolean | Prevent accidental deletion. | Ownership stays mandatory; do not silently remove protection for cleanup, require explicit authorized change. |
| `boot.autostart` | optional boolean | Boot child automatically when its Incus daemon starts. | Verify graceful recovery and resource quota. |
| `security.guestapi` | optional boolean | Whether Guest API access is enabled. | Preserve existing restricted access model; test availability and no parent socket exposure. |

**Advanced nested group (5 new optional booleans):**

| MCP field (exact native spelling) | Type | Concise Agent-facing help and gate |
| --- | --- | --- |
| `lxc.apparmor.allow_nesting` | optional boolean | Adjust *LXC generated-profile* nesting allowances, **not** a replacement for `security.nesting`. Map boolean to LXC 1/0 only via trusted translation; do not infer generated profile can be loaded under deeply confined AppArmor. |
| `security.syscalls.intercept.mknod` | optional boolean | Intercept specifically supported device-node operations; only when workload requires and Project permits. |
| `security.syscalls.intercept.setxattr` | optional boolean | Intercept selected extended-attribute requests; opt-in and subject to host/Project capability checks. |
| `security.syscalls.intercept.bpf` | optional boolean | Enable supported BPF interception; potential new host attack surface, opt-in. |
| `security.syscalls.intercept.bpf.devices` | optional boolean | Device-cgroup BPF support; effective only with `security.syscalls.intercept.bpf=true` and required host/Project capability. |

**Excluded from this RC6 scope:** `security.syscalls.intercept.mount.fuse` (string-valued mapping, not a boolean), arbitrary `raw.lxc`, `lxc.apparmor.raw`, `lxc.apparmor.allow_incomplete`, arbitrary custom Profile names, unrestricted custom ID maps, general arbitrary daemon-environment variables, or automatic host policy adjustments. Adding these later requires separate security design.

**Optional fields mean three states:** absent = preserve Incus/default/unset semantics, explicit `false` = request false, explicit `true` = request true. Preserve omitted-vs-false in JSON decoding and in `box_update` patch semantics; inspect real Incus live-update support (many map, privileged and interception values require stop/start). Avoid implicit booleans or default conversions. For `lxc.apparmor.profile`, `unset` is distinct from actual empty string and only means no LXC override.

**Combination checks:** `security.privileged=true` is incompatible with the default safe-recursive security contract; explicit fixed `security.idmap.base` must not be combined with `security.idmap.isolated=true`; BPF device interception depends on BPF interception; `lxc.apparmor.allow_nesting` only has the documented generated-profile meaning; `unchanged` is secure *only if the actual parent enforces the right inherited policy*. Validate combinations **before** native create, repeat postcondition verification, and do not infer a valid configuration from a successful stopped `incus init`.

### 0.2 MCP Agent guidance: concise, relevant, not an installer

Use one or two practical sentences for each parameter's description, exposing default/omitted semantics, dependency and risk. Add one short cross-field paragraph to `box_create` / relevant capability or schema metadata, not an excessively long manual:

> `lxc.apparmor.profile`: Container LXC AppArmor choice. `unset` leaves Incus in control; `unchanged` inherits the current enforced profile; `generated` requests a new generated Profile; `unconfined` requires authorization and must not be an automatic workaround. For deep nested Incus, if inner AppArmor policy administration fails, inspect the ancestor `(enforce)` label first. **After installing incusd *inside* the Box via `box_exec`**, the deployment Agent can evaluate setting `INCUS_SECURITY_APPARMOR=false` **only on that inner daemon service**, alongside exact verified `unchanged` on its children. Never disable host AppArmor or bypass project restrictions.

> `security.privileged`: Privileged container; default false, use only with explicit high-trust authorization. Do not use to solve nesting or AppArmor startup errors.

> `lxc.apparmor.allow_nesting`: LXC generated-Profile helper, not equivalent to `security.nesting`; only meaningful where generated AppArmor is actually in use.

> `security.syscalls.intercept.bpf.devices`: Device-cgroup BPF interception. Requires `security.syscalls.intercept.bpf` and host/Project permission; opt-in only.

Other option help: one clear sentence about behavior and any important precondition. Agent should consult the **actual checked runtime capability response**, not assume a recommendation is permission. If beneficial, expose a compact **read-only capability check** using existing MCP patterns; do not create an entire installer/orchestrator.

**No new `incus_security_apparmor` Box parameter.** `INCUS_SECURITY_APPARMOR` is an independent **incusd process environment option**. BoxWeave MCP creates/operates Boxes; Agent installs and configures inner incusd with existing `box_exec`. It is related to, but not synonymous with, a per-container `lxc.apparmor.profile` choice. The documentation must explain both surfaces and avoid suggesting that a container creation request can directly alter a daemon shared by many Boxes.

**Authoritative upstream references:** [LXC AppArmor directives](https://linuxcontainers.org/lxc/manpages/man5/lxc.container.conf.5.html), [Incus instance keys and live updates](https://linuxcontainers.org/incus/docs/main/reference/instance_options/), [Incus ID-map semantics](https://linuxcontainers.org/incus/docs/main/userns-idmap/), [restricted Project privileges/interception/lowlevel](https://linuxcontainers.org/incus/docs/main/reference/projects/), [Incus syscall interception/BPF dependencies](https://linuxcontainers.org/incus/docs/main/syscall-interception/). Implement against pinned Incus 7.6 actual capabilities; reject unsupported differences instead of assuming the docs imply universal availability.

### 0.3 Development/testing division for RC6

| Stage | Required result | What the result **does not** prove |
| --- | --- | --- |
| Full integrated code patch | All 14 surfaces wired (13 new, existing nesting retained), typed validation, AppArmor/BH-019 idmap/network/boot fixes, MCP descriptions, local tests, Go quality/race | No native runtime PASS merely because code builds |
| L0 isolated smoke | Actual installed candidate, parameter round trips and subset lifecycle tests, no observed P1 unsafe fallback | No deep security/recovery/L5 full PASS |
| RC6 candidate | Tag/artifacts for **exact smoke-tested SHA** under repo conventions; preliminary known-limit notes | No formal S2.5/BH-019 acceptance or `v1` merge |
| Post-RC6 repeated acceptance | Full installed HTTP MCP and independent nested daemon L0→L5, network and identity negatives, quota/crash/UNKNOWN/recovery, Verifier and same-SHA CI | Only after all applicable gates can PR move toward release |

**Fail-closed RC6 creation condition:** A critical ownership/security regression, actual sibling access where isolation was promised, broad unrestricted low-level admission, privilege escalation via defaults, missing operation persistence or broken rollback cannot be described as “minor smoke issue”. If such a condition exists, fix before publishing RC6. Other explicitly tracked acceptance gaps may remain on a clearly designated *test candidate*, never disguised as production readiness.


## 1. Ground truth, goal and safety boundary

Real isolated native Incus 7.6 experiments have **functionally reached L5**: independent Incus daemons ran in L1–L4 and L4 created, started, and executed in L5. This worked with **nonprivileged containers**, an **ancestor AppArmor Profile remaining `(enforce)`**, real separate User and Mount namespaces, and no host policy changes. The method was to set official `INCUS_SECURITY_APPARMOR=false` on **each inner incusd service only**, configure its child LXC policy transition to the exact `lxc.apparmor.profile = unchanged`, and delegate sufficiently wide non-overlapping UID **and** GID mappings. No `complain`, privileged container or host AppArmor disable was required.

**Important:** The deep levels inherited **the same outer enforced AppArmor Profile**. They did NOT gain a new independent AppArmor policy namespace at every level. Functional recursion and high-assurance tenant security are separate gates; do not advertise independent per-depth AppArmor policy management.

Evidence, including full original native receipts:

- [Real L5 recursion (43 original BW/native operation pairs)](../verification/S25_BH019_INCUS76_INHERITED_ENFORCE_RECURSIVE_L5_2026-11-14.md)
- [Incus 7.6 comprehensive A/B, network, ID maps, restart, SIGKILL, snapshot (42 pairs)](../verification/S25_BH019_RECURSIVE_NATIVE_COMPREHENSIVE_MATRIX_2026-11-14.md)
- [Complain-mode A/B / policy-admin parser243](../verification/S25_BH019_INCUS76_COMPLAIN_RECURSION_LAB_2026-11-14.md)
- [Per-depth separate AppArmor namespace hypothesis](../research/S25_BH019_STACKING_NAMESPACE_HYPOTHESIS_2026-11-14.md)

**Goal:** BoxWeave, rather than a human running native incus commands, must safely create and manage independent recursive Incus/BoxWeave layers using typed APIs, existing ownership and operation receipts, strong network segregation, deterministic mapping, and crash recovery. Target installed BoxWeave CLI + HTTP MCP black-box L0→L5, plus deeper levels only when the machine's actual resource and kernel budgets permit. Keep independent-policy-namespace research separate: this implementation only claims ancestor policy inheritance until broader evidence is available.

**Out of scope / forbidden shortcuts:** SELinux, VM substitution, kernel patches, uncontrolled `raw.lxc`/`raw.apparmor`/`raw.idmap`, arbitrary host mounts, privileged containers, global/host AppArmor disable, changing existing shared Boxes/ACLs, opening a broad Incus admin socket, reusing old UNKNOWNs, or widening the Incus restricted Project to permit low-level arbitrary instance configuration.

## 2. Existing BoxWeave code: reuse instead of reimplementation

Read PR #146 **latest HEAD**, compare `v1`, and inspect live tests before coding. Confirmed on head `cf9ab37aa19d6b31d40b189851a0947f2acd3df4`:

- `internal/native/box.go`: `InstanceAction` currently has `Nesting` (ordinary `security.nesting=true`) and `Internet`, but no managed-recursion contract or ID-range allocator. Default nesting must **not** silently activate the new mode.
- `internal/native/box.go::nicPrivate` already requires **exactly one managed NIC**, a nonempty **per-NIC security ACL**, `security.acls.default.ingress.action=reject`, plus `security.ipv4_filtering=true`, `security.ipv6_filtering=true`, `security.mac_filtering=true`. `nicProfilePrivate` protects default Profile egress, `CheckBootstrap` rejects unsafe native Profile, `ownedPrivateNICEgress` changes only the verified owned NIC. **Keep and apply these guards at every nested BoxWeave depth.** Native default `incusbr0` being reachable between peers does not mean existing BoxWeave network guards failed.
- `Reader.confined`, `Strict`, Project/UUID matching, ETag, bounded resource options, operation identity and original native receipts are already security primitives. Retain them.
- `internal/native/profile_config.go::SafePerUserProfileConfig` intentionally rejects arbitrary Profile configuration and permits only the exact Incus service-user mapping. Do not replace this with a generic raw passthrough.
- Preserve BH-018 transactional operation-finality work and S1/S2 guards. No historic schema/API compatibility burden for unpublished v1 is needed, but all current tests and semantics must remain correct.

## 3. Nonnegotiable engineering invariants

1. **Typed opt-in and fail closed.** Add a deliberate server-authorized managed-recursion capability, separate from existing `nesting=true`, with a documented ownership/budget/safety contract. A missing kernel/AppArmor/Incus/Project/network feature yields an unsupported or blocked result, never an insecure fallback.
2. **AppArmor authority:** Only the *nested daemon's* runtime may receive `INCUS_SECURITY_APPARMOR=false`. The first container's AppArmor must still be enforced by the trusted parent. The exact `lxc.apparmor.profile = unchanged` transition must be produced only by audited trusted server code for an owned child in verified managed mode. Do not expose a general `raw.lxc` field or allow caller-supplied profile text. Check actual `/proc/self/attr/current` on every depth and after restarts. Relative `unconfined` at L1 alone is NOT a security proof.
3. **Hard Project restriction:** Determine how to implement exact unchanged semantics **without** weakening restricted Incus Project config. If low-level config is forbidden by the Project, do NOT set broad `restricted.containers.lowlevel` permissions or use unrestricted tenant credentials. Stop that implementation slice and present a reviewed trusted adapter or upstream-safe alternate design with negative tests.
4. **Mapping ownership:** Every actual child and sibling has a valid, non-overlapping, bounded, persistent UID **and** GID range. Allocation remains correct under concurrency, failed create, crash, ACK loss, restart, partial deletion, and ID budget exhaustion. A saved `volatile.idmap.next` is not proof the kernel accepted it.
5. **Network:** Every nested BoxWeave Box gets existing default-deny per-NIC ACL and IPv4/IPv6/MAC anti-spoof filters, including dynamic IPv6 and link-local concerns. Do not confuse disabling WAN egress with sibling isolation. No Internet, host, sibling or ancestor network paths open without documented permission.
6. **Ownership, lifecycle and receipts:** retain Project/UUID checks, no arbitrary adoption, original native Incus operation ID, idempotency keys, operation UNKNOWN safeguards, immutable Guest outputs, bounded resources, rollback/reconciliation and exact-test-fixture cleanup.
7. **No premature PASS:** inheritance of a single enforced Profile does not substitute for independent per-layer Profile namespaces, seccomp, cgroups, mount/FS and network safety. BH-019 production/security acceptance remains BLOCKED until full negative gates pass.

## 4. Development work packages — complete code as an integrated batch before L0 deployment

### WP0 — audit and failing tests before change

Read native Box create/update, bootstrap Project guards, CLI/MCP schemas, Incus v7 Go SDK and upstream 7.6 policy-loader paths, install/service logic, SQLite operation/state handling, current BH-018/BH-019 tests. Define the smallest opt-in public surface; avoid broad new flags. Add tests that fail safely for unapproved managed recursion, arbitrary raw config, privileged attempt, missing parent `(enforce)`, missing Project permission, wrong daemon scope, inadequate network and ID resources. Confirm normal `box_create(nesting=true)` remains unchanged.

**Exit:** Reviewed typed design, exact preflight/go-no-go checks, green existing tests and working new negative admissions.

### WP1 — safe managed-recursion plumbing (P1)

Implement verified inner `incusd`-only `INCUS_SECURITY_APPARMOR=false` setup and persistence on restarts. Solve exact LXC `unchanged` behavior through a trusted, least-privilege native layer **without** broad low-level Project permissions. Validate actual inherited `(enforce)` label after create/start/exec, and require a distinguishable, auditable owned managed-recursion marker. Keep all raw/text system configuration inaccessible to MCP callers. A child failing the label/Project/inheritance preflight must not be started.

**Exit:** Disposable Incus7.6 BoxWeave-managed L1→L2→L3 real create/exec/delete with security negatives; no production insecure fallback. If restricted Project makes this impossible, record **BLOCKED** and precise supported alternatives before changing its security baseline.

### WP2 — transactional UID/GID budget allocator (P1)

Design per-parent range allocation with checked interval arithmetic (overflow-safe, including both UID and GID), explicit minimum **65,536 usable guest IDs**, reserve for descendants, depth and sibling-count limits, exhaustion errors, and support for differing actual host subuid/subgid pools. The 4,194,304 → 4,128,768 → 4,063,232 → 3,997,696 lab is an **example**, not a default or a guarantee of 32 levels. Prevent sibling overlap and any range escaping the parent's **actual** permitted UID/GID maps. Atomically persist reservation before native creation; no reuse under pending/UNKNOWN, crashed or ACK-lost operations; release only after independently verified definitive teardown/rollback. Deterministically configure supported nested daemon ID budget and verify actual mappings after restart.

**Tests:** serial and parallel creates, overlap/out-of-bounds, values near limits, resource exhaustion, cancellation/rollback, SIGKILL in flight, unknown terminal, orphan reconciliation and no double-leases. Run real Ubuntu default `_apt`, `nogroup`, normal APT and dnsmasq installation; **do not permanently disable APT sandbox or rewrite OS identities**.

### WP3 — recursive private network, including link-local and spoofing (P1)

Reuse current `nicPrivate` / `nicProfilePrivate` / `CheckBootstrap` / `ownedPrivateNICEgress`. Require the same strict network bootstrap in **every independent nested BoxWeave daemon**, not just L0. Ensure policies are attached to the **instance NIC**, not only bridge-wide ACL. A native default bridge A↔B test was reachable by IPv4/global IPv6; address-specific drop blocked those but was bypassed through A's `fe80::` link-local IPv6. Adding `source=fe80::/10` blocked the tested link-local flow, and deleting the ACL restored reachability: **a partial causal mitigation, not full isolation**.

**Tests:** both directions, IPv4 and global IPv6, link-local IPv6, address rotation/secondary IPs, MAC/IPv4/IPv6 spoof negative attempts in an owned lab, DHCP, RA, NDP, DNS, multicast where relevant, outbound policy and explicit network opt-in, same-bridge direct traffic, parent/host access, ACL missing/misattached, restart and rollback. Include real A/B server probes with permitted positive controls. Default reject for unverified NIC/ACL state; no broad `allow`.

**Exit:** no unauthorized sibling or ancestor traffic, link-local covered and anti-spoof demonstrably active on every nested level, external egress remains opt-in.

### WP4 — nested Ubuntu/systemd/AppArmor service compatibility (P2)

Observed: with inner daemon `INCUS_SECURITY_APPARMOR=false`, nested Incus ran and restarted correctly, but Ubuntu `apparmor.service` independently attempted to reload system Profiles, failed `apparmor_parser` policy-admin, and left systemd `degraded` after a whole-parent boot. Implement a **guest-only, explicit, auditable** handling strategy (consider skipping precisely that profile-loader service under verified inherited-enforce mode), without touching physical-host AppArmor, changing the parent `(enforce)` policy or indiscriminately masking unrelated services. Verify initial boot, repeated stop/start, `systemctl` health, true inner daemon environment and actual kernel AA label.

**Exit:** no unexplained degraded systemd, no hidden AA loss, daemon/network/guest autostart still works after full parent restart.

### WP5 — true installed BoxWeave HTTP MCP + CLI acceptance, resilience and CI

On new owned disposable resources, install real current BoxWeave candidate at each required layer, using independent Incus and BoxWeave services, not only raw `incus`. Reproduce BoxWeave-managed L0→L1→L2→L3→L4→L5, with real create/start/exec/get/list/update/stop/delete and MCP client reconnection. Validate each layer's actual inherited enforced AA label, distinct User/Mount namespaces, ID budgets, private NIC ACL and anti-spoof state, CPU/memory/disk quota, IPv4/IPv6, WAN on/off, sibling security negatives and safe teardown. The historical native lab had L1–L4 independent Incus daemons and L5 as runnable container; product test must establish actual **BoxWeave daemon functionality** at every required management layer.

Do real repeated parent boots, inner daemon SIGKILL and independent recovery, bounded concurrent create/delete, snapshot→modify→restore, resource/disk-exhaustion, in-flight create/start/delete crash, ACK loss / UNKNOWN & idempotent reconcile, failed installer rollback, and multiple-depth ownership/Project isolation. Preserve original BoxWeave AND native operation IDs, Guest exit codes and postconditions; a successful outer exec wrapper does not mean an inner command passed. Do not touch/replay any historical UNKNOWN.

Run Go format/vet/tests/race, S1/S2/BH-018 and S2.5 security negative suites, real Incus 7.6 black-box, official MCP SDK verification, architecture builds, independent Verifier, and same-commit GitHub Actions **after** real labs. Never use a green workflow as a substitute for missing native/security evidence.

## 5. Explicit acceptance scoreboard

| Item | Baseline | Product release condition |
| --- | --- | --- |
| Native L5 recursion | **LAB PASS** | Real BoxWeave CLI + installed HTTP MCP at each layer |
| AppArmor | Shared ancestor `(enforce)` in native L2–L5 | Each managed child retains inherited enforced label through lifecycle; no general policy access |
| Native restricted Project / raw | Current BoxWeave forbids generic raw config | Server-controlled exact unchanged path without broad Project lowering; fail closed |
| UID/GID | Manual wide mapping worked; native stopped bad maps admitted | Persistent atomic checked allocator with concurrency + crash reconciliation |
| Network | Ordinary native bridge siblings reachable; NIC ACL partial success including `fe80::/10` | Every Box's default-deny per-NIC ACL, link-local+dynamic address+spoof tests |
| Boot | Inner incusd works; Ubuntu AppArmor unit caused `degraded` | Scoped guest-only handling, true parent enforcement preserved |
| Recovery | Selected native SIGKILL, snapshot and parent restart PASS | Full installed BoxWeave/MCP in-flight recoveries and resource negatives |
| Independent AppArmor per depth | **NOT PROVED** | Remains separately BLOCKED; do not claim as feature |
| Overall BH-019 | **SECURITY BLOCKED** | All P1, relevant P2, real black boxes, independent Verifier and Actions PASS |

**PR #146 stays Draft/unmerged until all release gates are satisfied.** No BH-018 retest waiver, no unverified success claim, no silent lowering of BoxWeave's threat model.

## 6. Work rhythm, safety and next-agent continuity

Follow **`0 owner-approved release order**: implement all scoped feature code in one integrated phase (local compile/unit/negative checks and multiple commits allowed), then deploy **once to a new isolated L0 fixture** for basic smoke, then build and identify RC6 test artifacts, then repeatedly run comprehensive installed multi-layer/HTTP MCP/security/fault acceptance. The WP0–WP5 exit criteria remain the **final** gates and are not an instruction to delay initial RC6 until every full L5 fault injection is complete. At every test stage, record original operation/native IDs, classify Guest results independently, clean exactly owned disposable fixtures and re-read inventory. Keep all changes on the PR branch; do not merge before sign-off. If a prerequisite would require bypassing a restricted Project or removing an enforced ancestor boundary, stop that feature and report the exact failure instead of granting privilege.

Never modify host-level AppArmor, shared L0/L1 instances, historical UNKNOWNs, or existing network/security profiles. Scope network and installation changes only to newly owned test fixtures. Prior tests observed 4 historical L0 and 9 L1 Boxes; **read fresh inventory instead of assuming counts have not changed**.

**Next agent:** start by reading **`0 (owner decision and exact 14-field contract)** and the two main L5/comprehensive lab reports; verify latest PR HEAD, then implement the entire agreed patch (WP0–WP4 plus typed MCP surfaces) before the first new L0 install. After integrated code and quality tests, L0 isolation-smoke → publish traceable **RC6 test candidate** if no critical safety failures → repeatedly perform WP5 full L5/HTTP MCP, negative and crash acceptance, fix and retest. Report each step's SHA/receipts/PASS/FAIL/BLOCKED/NOT TESTED; keep Draft until final independent sign-off.
