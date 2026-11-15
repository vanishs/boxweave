
# S2.5 BH-019 — Managed, safe recursive Incus: development and acceptance plan

**Date:** 2026-11-15 PT  
**PR:** [#146](https://github.com/vanishs/boxweave/pull/146) (Draft; base `v1`)  
**Decision:** Begin product code development immediately, but do not change PR readiness or mark BH-019 security accepted.

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

## 4. Development work packages — implement in this order

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

Work in **small auditable steps**: WP0 design/tests → WP1 one live managed create/exec → WP2 allocation with two concurrent children → WP3 enforced network A/B → WP4 service boot → WP5 installed L5. After each step: unit/negative tests, exact disposable native Box tests, original result classification, bottom-up cleanup, latest inventory readback, then commit and document. Keep all changes on the PR branch; do not merge before sign-off. If a prerequisite would require bypassing a restricted Project or removing an enforced ancestor boundary, stop that feature and report the exact failure instead of granting privilege.

Never modify host-level AppArmor, shared L0/L1 instances, historical UNKNOWNs, or existing network/security profiles. Scope network and installation changes only to newly owned test fixtures. Prior tests observed 4 historical L0 and 9 L1 Boxes; **read fresh inventory instead of assuming counts have not changed**.

**Next agent:** start by reading this document and the two main L5/comprehensive lab reports; verify latest PR HEAD, then code WP0/WP1 first. Continue through the remaining WPs only with proven security invariants. Report test-by-test PASS/FAIL/BLOCKED and original receipts. No premature release.
