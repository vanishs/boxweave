# S2.5 RC6 L0–L5 authenticated MCP field matrix — 2026-11-15

> **Acceptance distinction:** functional recursive **L0 → L1 → L2 → L3 → L4 → L5 installed MCP path CONFIRMED** with the caveats below. This is NOT per-depth independent AppArmor security acceptance, NOT collision-free UID/GID reservation acceptance, NOT BH-018/019 final signoff, and NOT an authorization to merge PR #146.

## 1. Source, scope, and layer ownership

All six connections were queried **from their own independently authenticated MCP tools in this one session**, without reading other layers' databases. L0 is the physical host; each subsequent layer is its parent's independently running native Incus container. Each independently runs a daemon using a restricted **local** `user-1001` Project; the string `user-1001` is not a globally shared Project.

| Layer | Connected MCP | Build commit | Binary SHA256 | Incus | `ready`/`mcp_ready` | Own Box count at initial survey |
| --- | --- | --- | --- | --- | --- | ---: |
| L0 | `BoxWeave_L0_RC6` | `370e419c` official release | `0932ad43…d0610174` | 7.6 | true / true | 5 |
| L1 | `BoxWeave_L1_RC6` | `89b0f891` **locally built pre-release candidate** | `85788181…45a6509` | 7.6 | true / true | 10 |
| L2 | `BoxWeave_L2_RC6` | `370e419c` official release | `0932ad43…d0610174` | 7.6 | true / true | 2 (1 RUNNING + 1 STOPPED) |
| L3 | `BoxWeave_L3_RC6` | `370e419c` official release | `0932ad43…d0610174` | 7.6 | true / true | 1 RUNNING |
| L4 | `BoxWeave_L4__RC6` | `370e419c` official release | `0932ad43…d0610174` | 7.6 | true / true | 1 RUNNING |
| L5 | `BoxWeave_L5_RC6` | `370e419c` official release | `0932ad43…d0610174` | 7.6 | true / true | 0 |

The exact published RC6 amd64 release digest is `0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174`, commit `370e419c43f06b7488c7330ba61913ffec0e9f78`. **Do not substitute L1's locally built RC6 digest when asserting same-binary cross-layer equivalence.** L0 formerly had stale RC5 MCP tool-schema binding but this session's `BoxWeave_L0_RC6` is refreshed and read-only status/list operations succeeded.

All layers independently emitted the same two nonblocking but **not resolved** doctor warnings: `STORAGE_QUOTA_UNVERIFIED` (Incus quotas/Exec spool not verified) and `DEFAULT_IMAGE_ON_DEMAND`.

The exact known ancestry for the read-only child inspections was:

| Parent | Exact child Box name | Child Box UUID | Native state |
| --- | --- | --- | --- |
| L0 | `rc3-l1-nested-1113` | `cd1ea3d0-a24f-477b-b01c-08fb5acac53e` | Running |
| L1 | `rc6-l2-lab-1115` | `98ff4e4e-969c-4c77-803f-4af9f7b99463` | Running |
| L2 | `rc6-l3-enforced-1115` | `1762cebb-033b-4be2-a4f3-bc1af0caa8c6` | Running |
| L3 | `rc6-l4-enforced-1115` | `931dcf91-53cd-4e92-b87b-df7d21c8cbc2` | Running |
| L4 | `rc6-l5-enforced-1115` | `d5bb5b4f-21ab-4a1c-9460-1da94efc8b6f` | Running |

## 2. New L5 end-to-end smoke, real MCP + native Incus

Dedicated work_ref `RC6-L5-SMOKE-20261115`; only L5's **own empty restricted Project** was modified. Original `L5 host_status` verified exact official RC6 build, native Incus 7.6, ready/mcp_ready true, zero Boxes and zero historical operations.

Preflight before any privilege change:

- `lxc.apparmor.profile=unchanged` without lowlevel Project permission: original `81698c1015f7b32c038f6ee42f0d7930` **FAILED**, `SAFE_NATIVE_PREFLIGHT_DENIED`.
- `security.privileged=true`: `a2aaacdeb7185dcb03fb500f79d30c28` **FAILED**, `SAFE_NATIVE_PREFLIGHT_DENIED`.
- Parent L4 ran a read-only native inspection inside its exactly owned L5 Guest: original L4 Exec `c1e02980d7a34f3814ffef782db08f04`. It confirmed empty native instance inventory; Project restricted and nesting allowed; native AppArmor kernel Y, same inherited `(enforce)` label; original Project lowlevel unset.

For this single positive smoke only, on that empty L5 Project set `restricted.containers.lowlevel=allow`, while verifying inherited enforced AppArmor, via scoped L4 `box_exec` original `dc64dc3b0576ec50ab7d877e223df917`. No ancestor Project or physical host config changed. BoxWeave still applies its own typed preflight and rejects privileged/unconfined requests; **native Project lowlevel=allow itself is not a generally safe security boundary**.

**Positive disposable Box** `rc6-l5-smoke-1115`, UUID `fa63c210-2876-441a-a2a9-0db0110df471`, CPU1/1024MiB/8GiB, internet=false, nesting=true, `lxc.apparmor.profile=unchanged`, `boot.autostart=false`, `security.guestapi=false`, `security.protection.delete=false`:

| Stage | Original operation ID | Observed result |
| --- | --- | --- |
| Create | `01d524a31c56b287744231ea01df6faf` | SUCCEEDED, native readback confirms requested options |
| Update stopped instance | `b14048e2dfa12d1b59187c2b8613f17f` | SUCCEEDED, CPU=2, memory=1280MiB, `security.guestapi=true` read back |
| Start | `a748fa3ce0160b02a0456a3183a334f5` | SUCCEEDED, native state Running |
| Root Exec | `2cd7be4c3fcd4cd3ef313b7befacb4bc` | SUCCEEDED, `command_success=true`, exit 0, UID0, inherited AppArmor enforcing, Seccomp2, private v4/v6 |
| Nonroot Exec | `bf6af8660643d4a6c97425bb69d347b0` | SUCCEEDED, guest_uid=65534, guest_gid=65534, exit 0, same enforced profile |
| Deliberate nonzero Guest Exec | `1242545d27a18b887be31f134e21c964` | **Native operation SUCCEEDED**, Guest exit 23, `command_success=false`, expected stderr retained |
| Stop | `e32ce4541bf153a3f0fedbd12c2265c3` | SUCCEEDED |
| Delete | `b52beeb48f84347fbda5f9bb2c57b2f1` | SUCCEEDED, final L5 `box_list` **empty** |

After deletion verified L5's *native* inventory truly empty, then reversed the smoke-only lowlevel permission to its original **unset** state, scoped original L4 Exec `efee5fb4d4f2e8fa9d3d988d4149cba6` **SUCCEEDED**, verified same `(enforce)` label. There is **no positive-test Box or widened Project lowlevel left on L5**.

**L5 functional verdict:** actual authenticated newly connected MCP read/admit/create/update/start/Exec/output/stop/delete **PASS**. These normal tests do not exercise asynchronous crash windows or prove security against malicious siblings.

## 3. Cross-layer native security witness (no mutation)

Read only from each *parent's* own authenticated MCP into its named owned child. L1→L2 original `77f7746e015b5a9af93906c17ee7be63`, L2→L3 `5972da94c97b5fb2e55d5ae7ae844a18`, L3→L4 `9e64c6845bcd5dd8a1fd662e1bafe060`, L4→L5 `5790018f7cffb001c0d3cf4be07dc807`; all SUCCEEDED, command exit 0.

| Guest | Observed PID1+Exec AppArmor policy label | Trailing UID/GID subordinate span from `/proc/self/*id_map` | Inner `incus.service` | Own Project lowlevel at time of read |
| --- | --- | --- | --- | --- |
| L2 | `incus-user-1001_rc6-l2-lab-1115_</var/lib/incus> (enforce)` | UID 998998998 / GID 998999009 | active; `INCUS_SECURITY_APPARMOR=false` | allow |
| L3 | **exact same inherited L2 label** | UID 997998998 / GID 997999009 | active; same scoped flag | allow |
| L4 | **exact same inherited L2 label** | UID 996998998 / GID 996999009 | active; same scoped flag | allow |
| L5 | **exact same inherited L2 label** | UID 995998998 / GID 995999009 | active; same scoped flag | allow at read time; restored to unset after smoke |

All four show enabled AppArmor kernel, Seccomp=2, `/proc/self/cgroup` printed `0::/.lxc` within each container's namespace. **Critical:** these identical *label strings* do not prove that each depth has a separately enforced Policy Namespace, its own policy-admin capability, or cross-tenant denial. The current functional strategy **inherits an enforcing ancestor label** and disables only each *inner daemon's attempt to manage AppArmor profiles*. The control-plane's safe positive bootstrap is not the same as BH-019's original strong per-depth AppArmor isolation objective.

Observed trailing mappings shrink by exactly **1,000,000 IDs per additional depth** in this chain. The first UID-map segments include `0 1000000 1001` plus `1001 1001 1` on L2–L5. This is useful for budgeting and tests, **not** proof of durable sibling-exclusive UID/GID leases, exhaustion handling, crash recovery, or correctly isolated namespace stacks. Explicit `security.idmap.isolated=true` is still denied without a verified transactional reservation allocator.

Attempted L0→L1 read-only parent `box_exec` was **rejected before new operation admission** with `original unresolved resource claim is still locked`. This is a **meaningful expected fail-closed condition** under existing original UNKNOWN (below). We **did not** use the healthy L1 child MCP as a workaround to replay, cancel, unlock or alter the L0 claim. Because L0 parent Exec was locked, **there is no new L1 native AppArmor/ID-map witness from L0 in this comparison**. Do not backfill from guesses or the separate physical L0 unconfined smoke.

## 4. Cross-scope isolation and historical UNKNOWN

Even though every independent instance uses Project name `user-1001`, five deliberate **read-only foreign-UUID/operation-ID probes** were rejected with `operation not found`:

- L5 `box_get` of L4-owned UUID `d5bb5b4f-21ab-4a1c-9460-1da94efc8b6f`.
- L4 `box_get` of L3-owned UUID `931dcf91-53cd-4e92-b87b-df7d21c8cbc2`.
- L3 `box_get` of L2-owned UUID `1762cebb-033b-4be2-a4f3-bc1af0caa8c6`.
- L5 `operation_get` for L4 original Create `1f52d22739e7fb374f3e4f25d1e67c15`.
- L4 `operation_get` for L5 original Create `01d524a31c56b287744231ea01df6faf`.

This supports controller-local Box/Operation ownership visibility for the tested requests; it is **not** a penetration-test proof of all cross-layer leaks or exposed native sockets.

L0 `operation_list(state=UNKNOWN)` returned **two immutable original UNKNOWN receipts**, without reconciliation or replay:

1. `4a5130ee9025e7ee39c1866d16ad1b39`, `box_exec` on the actual parent-owned L1 UUID `cd1ea3d0-a24f-477b-b01c-08fb5acac53e`, `POSTCONDITION_UNVERIFIED`.
2. `718b99438cfa8f1349895ade0e8dfa9a`, `box_exec` on `s25-bh019-bpf-ab-d-1114` UUID `0d0ffd60-2fd6-4734-b40c-6eabdb561033`, same error class.

L2 independently still holds prior `box_start` UNKNOWN `c6938fe17342ad02567ae11cf5642590` on the **different stopped** experimental `rc6-l3-lab-1115`. It was originally followed by real Incus LXC startup AppArmor “Failed to change AppArmor profile to unconfined” (and separately reported cgroup-device BPF EPERM). None of these historical receipts were touched. An already-running nested child can have its own responsive MCP while its **parent's original resource claim remains locked**. **Do not automatically equate “child is running” with permission to mark the parent's original Exec SUCCEEDED/FAILED**, or assume an inability to dispatch further parent commands proves the child daemon unhealthy.

## 5. What to test next — staged matrix

**Stage A, read-only, independently repeatable, low risk:** On every L0–L5 authenticated MCP, record `host_status`, `box_list`, filtered `operation_list`, original `operation_get`/output, exact commit/binary SHA, Incus version/project/socket, AppArmor module+PID1/Exec labels, namespace IDs, subordinate UID/GID map size, restricted Project keys, ACL identity/ownership, and actual daemon active state. Freeze known live IDs. Repeat cross-layer foreign UUID/read-only non-disclosure. Clearly mark L0→L1 original-UNKNOWN lock rather than bypassing.

**Stage B, temporary normal lifecycle, moderate risk:** Per layer, use an independently named disposable Box and durable unique idempotency keys; enforce restricted typed options, verify native Create→Update→Start→Exec root+nonroot→stdout+stderr+nonzero exit→Stop→Delete, original receipts after MCP reconnect, same-key exact replay and changed-intent rejection. Record cleanup. Test internet=false and private v4/v6 without silently treating bridge-default values as packet witnesses. Restore any only-for-test lowlevel config to its original value on an empty Project.

**Stage C, dedicated isolated security negative matrix, higher risk:** Test same Project plus real separate sibling and cross-Project tenants, IPv4/IPv6 cross-tenant packet denial (including ULA/link-local/gateway, egress, spoofing), ACL deny precedence, guestapi false, no host root/socket exposure; native AppArmor **policy admin positive and negative** per exact depth and parent policy stacking/namespace semantics; cgroup-device BPF actual safe load/attach/detach and denial as separate from AppArmor; collisions of **durably reserved** UID/GID ranges across sibling creates/stops/crash/UNKNOWN and budget exhaustion; real native storage quota including `dir` limitations / Ext4-XFS project quotas, Exec spool disk pressure. Run only in fresh exclusive native labs with an independently reviewed host/kernel capability matrix. **Do not claim PASS by turning off the ancestor host AppArmor, using privileged/unconfined Guests or treating shared ancestor Profile as an independent descendant Profile.**

**Stage D, BH-018 lost ACK/SIGKILL, controlled one-use test lab only:** Independent installed HTTP MCP 126/127 and one/nonzero effects, native original receipts, actual SQL WAL atomicity at live service process-kill before/after COMMIT, durable exact stdout/exit/event, replay dedup and new-key admission vs provable UNKNOWN+lock, normal and abrupt restart. Independent SQLite read-only oracle, separate verifier, one clear pass/fail per original operation. **Never induce crashes on shared L0/L1–L5 or replay the three historical UNKNOWNs.**

**Stage E, external MCP and version/release discipline:** each level independently validate OAuth metadata 200, anonymous /mcp 401, authenticated official SDK ListTools and limited tool invocation after Quick Tunnel restart/hostname-change; note per-layer control-plane endpoint registration can have stale schemas, and public 403 `invalid Host` while OAuth disabled is NOT public-MCP acceptance. Replace the L1 locally built RC6 candidate with the exact release only on explicit operator approval and after safely handling its pending backup and native test consequences. Run exact code+doc SHA source tests, native 7.6 dual architecture, installed HTTP MCP and independent verifier before any formal PR/release promotion.

## 6. Consolidated verdict and PR #146 gates

- **L5 basic MCP: PASS**, including actual Box lifecycle/Exec/output and cleanup.
- **L0–L5 independently contacted: 6/6 PASS**, same build identity on 5/6 (L1 local candidate exception), Incus 7.6 across all; **not** a depth security acceptance.
- **Inherited enforcing AppArmor observed L2–L5: PASS**, **per-level independent policy administration: NOT PROVEN / previously denied**, BH-019 remains BLOCKED.
- **Original ID/user namespace budgets: observed**, transactional isolation allocation, collisions/exhaustion: NOT PROVEN.
- **Scoped cross-controller foreign Box/Operation visibility: 5 expected denials**, not comprehensive privilege escape testing.
- **L0 original UNKNOWN locks: preserved**, BH-018 terminal classifier and independent installed verifier still require case-specific signoff, do not relabel automatic.
- **Storage quotas and Exec spool: unresolved**, despite all layers ready.

For code/CI truth, see `docs/verification/BRIDGEHEAD_ACCEPTANCE.md`, `docs/research/S25_BH018_019_PINNED_ROOT_CAUSES_AND_HOST_GATE_2026-11-14.md`, and the four immediately preceding L2→L3, L3→L4 and L4→L5 reports, together with this report. PR #146 remains Draft and not safely mergeable; any RC6 lab prerelease is clearly **pre-acceptance**, not a stable/HOST_GO security signoff.
