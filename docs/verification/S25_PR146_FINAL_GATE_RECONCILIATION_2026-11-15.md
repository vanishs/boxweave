# PR #146 — reconciliation and evidence-aware closeout checkpoint (2026-11-15)

**Branch:** `fix/v1-s25-bh018-bh019-20261114` (Draft PR #146). **Target:** `v1`. This is a branch reconciliation, **NOT an S2.5 signoff, stable release, or merge into `v1`**.

## Verified reconciliation of PR #147 before combining histories

The target `v1` advanced by exactly one commit `1224845c139177f3986d1ca3d51032849422ed1c`, the scoped BH-018-only pilot PR #147, after PR #146 forked at `c3023a72986085b77cd98f2fa5ac83ef40abc632`. GitHub reported `mergeable_state=dirty` and PR #146 was 175 commits ahead / one behind.

All 18 paths touched by #147 were inspected against PR #146 original HEAD `568724f2a7771e306d5ce9e39fab20c1776bbda6`:

- **14 identical blob SHAs:** BH-018 SQLite/worker/native tests, recovery logic, source probes and native acceptance script are byte-identical, not competing implementations.
- **2 Go files with additional RC6 typed-option code in #146:** `internal/control/box_tools.go`, `internal/native/box.go`. Compared both sides' unique lines; preserve the newer #146 option/readback additions and all byte-identical BH-018 code.
- **1 branch-specific workflow:** `.github/workflows/v1-s25-bh018-installed-candidate.yml` differs only by its allowed branch and matching job guard; retain the #146 branch variant.
- **1 #147-only note:** `docs/verification/V1_RC5_BH018_PILOT_SCOPE_2026-11-14.md`; preserve it unchanged in the combined tree.

Reconcile by a **non-force, two-parent merge commit** with the #146 source tree plus that one #147-only note. This does not modify the published immutable RC6 tag or binaries. Recheck GitHub PR mergeable state, quality/S1/dual-architecture and genuine native/install workflows on the NEW merge SHA; previous green runs are **not** the new SHA's checks. Keep Draft until all independent security and recovery gates pass.

## First-hand six-layer read-only check

2026-11-15: independently called authenticated `host_status` on L0, L1, L2, L3, L4, L5: **6/6** `ready=true`, `mcp_ready=true`, Incus **7.6**, Project `user-1001` per distinct daemon. L0 and L2–L5 report official immutable RC6 binary SHA256 `0932ad434d2a1b113e2254f6ca02a218c2dff473f70d7ed3b2f45e93d0610174`, code `370e419c`. L1 still reports distinct local candidate binary SHA `85788181e606300b441326527d853e04984f00b4cc2c4c7618a33f75845a6509`, commit `89b0f891`. **Do not call this six-layer same-binary acceptance.** Each level still emits `STORAGE_QUOTA_UNVERIFIED`; there is no actual disk/Exec spool hard quota proof.

Read-only `operation_list(state=UNKNOWN)` confirmed the **original, unchanged**:
- L0 `4a5130ee9025e7ee39c1866d16ad1b39`, native `fa2f54f1-bbe1-432d-88cc-52ad572e938c`, `POSTCONDITION_UNVERIFIED`;
- L0 `718b99438cfa8f1349895ade0e8dfa9a`, native `345841b8-18b9-46f9-a272-54da45d985f6`, `POSTCONDITION_UNVERIFIED`;
- L2 `c6938fe17342ad02567ae11cf5642590`, native `0f522d84-33b8-4535-bd0e-53fe0b093e26`, `POSTCONDITION_UNVERIFIED`.

No replay, cancellation, unlock, mutation, SIGKILL, teardown or same-resource attempt was performed on L0–L5.

## Exact-original-HEAD CI checkpoint (before this reconciliation merge)

- Native Incus 7.6 **3×9 original Exec** matrix and two-parent **BPF A/B**: [run 43767471059](https://github.com/vanishs/boxweave/actions/runs/43767471059), source `568724f2a7771e306d5ce9e39fab20c1776bbda6`, SUCCESS.
- Genuine installed nonroot HTTP MCP Incus 7.6 crash/WAL/ACK: [run 43767471113](https://github.com/vanishs/boxweave/actions/runs/43767471113), source `568724f2a7771e306d5ce9e39fab20c1776bbda6`, **IN PROGRESS when checked**; do not infer PASS.
- Same checks on predecessor `db2a5452`: [native](https://github.com/vanishs/boxweave/actions/runs/43767260249) and [installed](https://github.com/vanishs/boxweave/actions/runs/43767260237) succeeded; not sufficient to sign this report's NEW merge commit.
- Older focused green checks are not a substitute for the PR's quality-gate, S1 race, dual-architecture and independently commissioned Verifier on the final exact SHA.

## Formal disposition and next security work

| Scope | Verdict | Exact unresolved proof |
| --- | --- | --- |
| BH-018 code/atomic terminality and existing source regression | **IMPLEMENTED; targeted checks PASS at predecessor SHA** | New same-SHA installed HTTP MCP independent Verifier; original 0/23/126/127 and COMMIT-before/after SIGKILL, lost ACK, all-or-none SQLite, exactly one journal event; distinguish proven FAILED vs UNKNOWN without false inference from native 404 or stopped state |
| BH-019 functional L0–L5 recursion | **CONFIRMED SCOPED FUNCTIONAL**, NOT independent-security PASS | Per-depth inherited AppArmor label not equal to per-depth delegated policy namespace |
| BH-019 AppArmor namespace and policy admin | **BLOCKED** | Kernel/Incus 7.6 actually authorize policy load in a child-owned namespace under enforced ancestry; deny sibling/global policy writes, bounded namespace/profile budgets; no privileged/unconfined/host-AppArmor-disable fallback |
| BH-019 cgroup-device BPF | **PARTIAL** | Shallow A/B tests do not prove deep per-depth safe delegation and negative deny |
| BH-019 UID/GID reservation | **NOT IMPLEMENTED / BLOCKED** | Transactional durable UID **and** GID lease ranges, parent-pool and foreign occupancy, sibling disjointness, restart/UNKNOWN and exhaustion with fail-closed denial |
| BH-019 network | **PARTIAL** | Actual IPv4+IPv6 sibling and separate Project packet-level negative cases and positive controls at every required depth |
| BH-019 quota | **UNVERIFIED** | Native driver-backed storage and retained Exec spool hard bounds with actual negative exhaustion/limit tests |
| PR promotion | **BLOCKED / KEEP DRAFT** | Full S2.5 acceptance matrix, independent Verifier and all exact-new-SHA Actions |

**Do not change the original acceptance contract to call inheritance a policy namespace, or infer actual native start failure from a 126/127 return, vanished native Operation, or delayed Stopped readback.** If pinned Linux kernel/AppArmor/Incus cannot permit unprivileged safe per-depth policy admin, record a genuine platform blocker and require an explicit product-level design decision or upstream capability; do not silently degrade the requested security contract.

References: `docs/verification/S25_RC6_L0_L5_CROSS_LAYER_MCP_FIELD_MATRIX_2026-11-15.md`, `docs/development/S25_PR146_RC6_L0_L5_HANDOFF_AND_CLOSEOUT_2026-11-15.md`, `docs/research/S25_BH018_019_PINNED_ROOT_CAUSES_AND_HOST_GATE_2026-11-14.md`, and `docs/verification/BRIDGEHEAD_ACCEPTANCE.md`.
