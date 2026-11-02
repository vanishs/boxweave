# BoxWeave Feature Roadmap — Master Index and Decision Register

**Status:** Planning inventory and cross-document navigation, **not** a release plan or implementation authorization.
**Code-first review:** 2026-11-01, `main` source v0.27.2, initial inspected HEAD `ce62d5a45f6b5de7bc6b34ee55251aee86c47fca`.
**Owner direction:** Keep Boxes clean, use separate Boxes for parallel projects, prefer external nonintrusive weaving, reuse the installed Provider/SSH/HTTP MCP architecture, and keep optional coordination nearly invisible for ordinary single-Agent use.
**Scope rule:** The roadmap tracks *ideas and decisions*. Product requirements and delivery authority belong to [SCOPE](../product/SCOPE.md), [REQUIREMENTS](../product/REQUIREMENTS.md), [ACCEPTANCE](../product/ACCEPTANCE.md), and [PHASE_STATUS](../process/PHASE_STATUS.md). Installed evidence belongs to the [feature matrix](../verification/FEATURE_MATRIX.md) and [verification queue](../verification/REMEDIATION_TEST_QUEUE.md). An owner-approved *design* is not permission to implement, publish, install, connect a public listener, create a Box or modify a host.

## 1. How to read this roadmap

- **DESIGN — owner-selected:** behavior and scope discussed/selected; implementation is not yet complete or authorized here.
- **DESIGN — candidate:** existing documented proposal, still needs a product decision; not in current approved scope.
- **PARTIAL SOURCE:** a selected capability exists in code, but the comprehensive behavior under discussion is not implemented or proved.
- **SOURCE IMPLEMENTED / LIVE LIMITED:** source and some previously executed real tests exist; actual current host/client/install acceptance is *not* implied.
- **DEFERRED / OUT OF SCOPE:** do not initiate without a fresh user decision.

**Evidence discipline:** S = checked source, D = design/history, T = executed tests/fixtures, L = actual authorized installed/Box/client evidence, U = unverified or blocked. A proposal header dated at v0.21.4 does not supersede later M0–M5/D3/D4 code and live-check records. The version of `main`, the published release and the running machine can differ.

This master page owns the **overall relationship, provisional priority and navigation**. Each linked child owns detailed API, security, development batches and acceptance. Do not copy or fork a child specification here.

## 2. Four selected new feature designs

| Track / status | User outcome | Main dependency and important boundary | Design and tests |
| --- | --- | --- | --- |
| **R-01 Box operations** — DESIGN, owner-selected | An Agent can inspect project files, change code safely and execute tests in an exact managed Box without a mandatory in-Box service | Reuse Provider/SSH; distinct app-side `agent_box` is not assumed to be installed. Exact-target trust, workspace safety, conflict-aware changes | [Box operation interface v1](BOX_OPERATION_INTERFACE_V1.md) |
| **R-02 MCP task reliability** — DESIGN, owner-selected | After timeout/reconnect, recover the previously accepted task's state and *known* result; prevent accidental double dispatch | Durable request identity and receipt; `UNKNOWN` must never be silently retried. Does **not** promise guest process survival | [MCP task reliability R1](MCP_TASK_RELIABILITY_R1.md) |
| **R-03 Installation and first connection** — DESIGN, owner-selected | One short SSH/HTTPS-aware first-use path to an authenticated client-visible read | Build on `install.sh`, `boxweave client`, `boxweave http`, OAuth and existing doctor; do not invent a second installer or automatic public tunnel | [First-use onboarding v1](FIRST_USE_ONBOARDING_V1.md) |
| **R-04 Multi-Agent collision protection** — DESIGN, owner-selected | Multiple projects/Agents can share the controller; unrelated Boxes run in parallel; same-resource modifications fail fast | Optional 10-minute Box lease + heartbeat/renew, automatic expiry when safe, durable operation claim on uncertain dispatch; no queue or Agent runner | [Multi-Agent lease/conflict v1](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) |

**Cross-track design consistency:**
- R-01 code changes and `connect(exec)` must not bypass R-04 mutation conflict checks once conflict protection is activated.
- R-02's durable uncertainty contract should be reused by R-04 operation claims. Lease expiry is not proof that a remotely dispatched command ended; do not create two contradictory `UNKNOWN` systems.
- R-03 needs a safe *read* milestone first; no automatic Box creation, lifecycle writes, privileged diagnostics or new public endpoint merely for onboarding.
- Deliver and accept each slice independently; do not make all four features one large release gate.

## 3. Historical Box environment plan: E0–E4 (all DESIGN, candidate)

The 2026-10-30 [environment master](BOX_ENVIRONMENT_PROPOSAL.md) and its five implementation handoffs are complete **as designs only**. They are **not** current BoxWeave snapshot, restore or clone implementations. A Containarium tenant **database** backup is not an Incus/managed Box snapshot.

| Batch | What it would add | Gate / dependency | Child specification |
| --- | --- | --- | --- |
| **E0** | Read-only authenticated Provider capability / authority / identity discovery for exact managed Box snapshots/clones | First prove the *deployed* Provider supports typed, authorized operations; no direct-Incus fallback | [E0 capability boundary](BOX_ENV_E0_CAPABILITIES.md) |
| **E1** | Bounded immutable Box environment fingerprint and before/after difference, useful for old/new Box comparison | Prefer external observations; do not require a functioning in-Box diagnostic agent; E0 capability checks first | [E1 fingerprints](BOX_ENV_E1_FINGERPRINTS.md) |
| **E2** | List, create and delete snapshots of stopped, managed Boxes, with exact source identity and quotas | Only if E0 proves an authenticated Provider snapshot contract; pending/unknown lifecycle conflicts must block | [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md) |
| **E3** | Explicitly planned and confirmed restore from an exact snapshot to the same stopped Box | Requires E2 and clear coverage of attached/custom volumes; never promise a whole-host rollback | [E3 restore](BOX_ENV_E3_RESTORE.md) |
| **E4** | Clone an exact stopped Box or observed snapshot as a *new* independently registered managed Box | Requires verified Provider clone + new identity/rekey, network/SSH/tenant registration and safe quarantine of partial clones; not a raw `incus copy` shortcut | [E4 clone](BOX_ENV_E4_CLONE.md) |

**Owner decision still required:** whether to add E0–E4 to product scope. The sensible first step is a bounded E0 capability audit, *not* implementation of all five. E1 may be useful even when snapshot/clone is unsupported, provided only observed metadata is advertised. E2/E3/E4 must independently stop at `unsupported` when the upstream contract is missing.

Note that historical E0/E1 child prose may call D0–D2 "not implemented": those are frozen historical references. D0–D2 later shipped in source v0.22.0; D3/D4 have subsequent source implementation and separate live evidence. Preserve design provenance but rely on [current source/lived evidence](../verification/FEATURE_MATRIX.md).

## 4. Other previously proposed capabilities — current dispositions

These are **not** additional automatic work packages. They are candidates or partial features noted in previous source audits, MCP M0–M5 research, historical parity matrices and user workflow discussions.

| Candidate | Source/plan state at v0.27.2 | Recommended disposition and why | Where its child detail lives |
| --- | --- | --- | --- |
| **P-01 Full Box auto-sleep + observed wake/revival** | **PARTIAL SOURCE**: `bwmini/auto_sleep.py` validates and forwards an explicit upstream toggle; Web/receipts expose acknowledged action. Actual sleep, idleness detection and restart correctness are not a proven complete product | Conditional **P2**. Consider only after real usage shows resource pressure; verify actual Provider state transitions and running-work interaction | [M2 Platform reuse](mcp-integration/M2_PLATFORM_REUSE.md); [M0–M5 implementation/evidence](../process/MCP_M0_M5_SOURCE_PROGRESS.md) |
| **P-02 Coding-run approval integration** | **CANDIDATE**, upstream `code_runs/logs/pending/answer` concept researched; no general BoxWeave run-approval bridge | Conditional **P3**. Do not expose run-internal signed approval MCP as a universal approval server | [M5 Web/Agent workflows](mcp-integration/M5_WEB_AND_WORKFLOWS.md); [agent-box matrix](MCP_AGENT_BOX_CAPABILITY_MATRIX.md) |
| **P-03 Agent-directed automatic test orchestration** | **PARTIAL DEVELOPER PRACTICE**, test rules, selftests and local source bundle exist; no general autonomous testing service | **DEFER** as a product. Continue ordinary Agent-initiated test plans via separate Boxes; revisit only with a measured repeated workflow | [Agent test deployment](../process/AGENT_TEST_DEPLOYMENT.md); [M5](mcp-integration/M5_WEB_AND_WORKFLOWS.md) |
| **P-04 Per-client / per-Box MCP authorization** | **NOT IMPLEMENTED**; optional host-wide allowlist exists but does not identify or isolate individual trusted-operator Agents | **DEFER** broad RBAC. Reassess if independent untrusted users/accounts need different privileges; R-04 leases are coordination, not security identity | [MCP security](MCP_INTEGRATION_SECURITY.md); [MCP capability master](MCP_CAPABILITY_INTEGRATION_MASTER.md); [M0–M5 status](../process/MCP_M0_M5_SOURCE_PROGRESS.md) |
| **P-05 File upload/download and cross-machine source sync** | **PARTIAL tooling**: app-side configured `agent_box`, existing SSH exec and developer `agent_source_bundle.py`; no verified general direct MCP file transfer product | Conditional **P2** after R-01. First confirm exact trust, binary/large-file bounds, transfer channel, OS permissions and scope. Do not mistake bundle generation for remote delivery | [Box operations](BOX_OPERATION_INTERFACE_V1.md); [Agent deployment](../process/AGENT_TEST_DEPLOYMENT.md) |
| **P-06 Real-time resource metrics and monitoring** | **PARTIAL SOURCE**: status, diagnostics and upstream candidate metrics are distinct from proven full runtime telemetry | Conditional **P3** only if specific needed metrics and authority are confirmed; avoid a broad dashboard/monitoring daemon | [MCP capability master](MCP_CAPABILITY_INTEGRATION_MASTER.md); [M2](mcp-integration/M2_PLATFORM_REUSE.md); [legacy inventory](../knowledge/FEATURE_INVENTORY.md) |
| **P-07 Simplified upstream-only fresh-host dependency installation** | **RESEARCH ONLY / NOT APPROVED**: prior installer-boundary audit and subsequent source remediation; installed-host parity remains a separate gate | **DEFER architecture replacement**. R-03 improves first-use UX without silently handing existing Incus/Provider installation to an unreviewed upstream script | [Installer/Provider boundary audit](../verification/CONTAINARIUM_INSTALLER_BOUNDARY.md); [First-use R-03](FIRST_USE_ONBOARDING_V1.md) |
| **P-08 Advanced HTTP proxy/domain routing, bulk Box actions and full legacy monitoring/UI parity** | Historical legacy or postponed scope, not current selected product promise | **DEFER** until concrete demand; do not automatically resurrect v0.5.1 breadth | [Approved scope](../product/SCOPE.md); [archive parity](../verification/ARCHIVE_051_TO_MAIN_AUDIT.md) |

### Already implemented source — do not re-add as a new feature by mistake

- Selected MCP M0–M5 was merged into `main` in PR #9; app-side agent-box reads, bounded actions, optional gateway allowlist and auto-sleep toggles already have source implementations, with separate live acceptance limits. [Source + real-test progression](../process/MCP_M0_M5_SOURCE_PROGRESS.md).
- D0–D2 and the selected D3/D4 implementation are not hypothetical proposals. Current exact-runtime D4 positive and full external transports remain separate acceptance questions. [Diagnostics master](DIAGNOSTICS_D3_D4_MASTER.md), [feature matrix](../verification/FEATURE_MATRIX.md).
- Remote HTTP MCP, OAuth, bearer and credential-URL modes exist in source; user-facing guided first-use is the new R-03 design, and actual externally reachable client integration still needs version-specific evidence. [Remote master](REMOTE_MCP_MASTER.md), [R-03](FIRST_USE_ONBOARDING_V1.md).

## 5. Proposed dependency/priority map (nonbinding)

```text
LOW-FRICTION AGENT EXPERIENCE
  R-03 first connection ───────────────> real authenticated read
  R-01 Box operation contracts ─────────┐
  R-02 durable MCP receipts/UNKNOWN ────┼──> R-04 Box/host conflict coordination
                                       └──> optional P-05 transfer
ENVIRONMENT RELIABILITY
  E0 authorized Provider capabilities ──> E1 environment fingerprints
                                      └──> E2 snapshots ──> E3 restore
                                                      └──> E4 clone + identity/rekey
LATER ONLY IF NEEDED
  P-01 actual sleep/wake; P-02 approvals; P-06 monitoring; P-04 client RBAC
```

**Suggested direction, not an approved release sequence:**

1. Prioritize R-03, R-02 and R-01 as independent slices that reduce everyday Agent friction and uncertainty; implement R-04 minimally where conflict risk appears. Do not block the ordinary single-Agent path on an unnecessary lease UI.
2. If the owner selects environment features, start with **E0 read-only Provider capability check**. E1 is an independently useful follow-on; commit to E2/E3/E4 only after provider support and safe identity handling are demonstrated.
3. Do not build a general scheduler, agent execution service, DB-backed mega-framework or complete monitoring Web console merely because the old proposal named them.

## 6. Explicitly declined or excluded paths

- **Git Worktree manager:** declined. Use separate Boxes for independent development projects; do not build a Worktree "first-class resource" or background Worktree GC.
- **Master Agent launching sub-Agents / Agent Runner:** not the user's multi-Agent concern. The selected problem is resource collisions between already-running independent projects/Agents.
- **Automatic queue on conflict:** declined. Conflicting mutations return `RESOURCE_BUSY` immediately; a crashed Agent's lease expires unless a genuinely uncertain operation claim remains.
- **Mandatory resident Agent, Debug MCP, service or filesystem watcher inside each Box:** excluded by the clean Box preference.
- **Per-Agent RBAC, global team workflow scheduler, automatic prompt-graph execution:** not selected as a product goal.
- **Host/provider migration or automatic broad host changes:** no host action is authorized by any roadmap entry.

## 7. Roadmap governance and link maintenance

1. Update this **master row and child document together** when a product decision, feature stage or blocked dependency changes; child docs link back here via `[Feature roadmap](FEATURE_ROADMAP_MASTER.md)`.
2. For a selected implementation: first reconcile [SCOPE](../product/SCOPE.md), [REQUIREMENTS](../product/REQUIREMENTS.md), [ACCEPTANCE](../product/ACCEPTANCE.md), current Provider/tool catalog and latest [PHASE_STATUS](../process/PHASE_STATUS.md). Then add a small implementation batch with tests.
3. Record SOURCE/T/CI/LIVE/U/BLOCKED separately in [FEATURE_MATRIX](../verification/FEATURE_MATRIX.md) and [REMEDIATION_TEST_QUEUE](../verification/REMEDIATION_TEST_QUEUE.md); never convert a design or tested historical release into current-host PASS.
4. Maintain **backlinks on feature-owning designs only**, not historical immutable acceptance/incident reports. The latter remain evidence at their original revision.
5. This roadmap is intentionally compact and additive: no code changes, no active Box actions, no release/version bump and no new permission claims are implied.

**One-sentence product direction:** *Keep BoxWeave's Box/Provider/MCP foundation, make ordinary Agent usage reliable, coordinate rare conflicts without queues, and add environment snapshots/clones only when the authenticated deployed Provider can prove it supports them.*
