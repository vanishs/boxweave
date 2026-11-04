# BoxWeave Feature Roadmap — Master Index and Decision Register

**Status:** Planning inventory and cross-document navigation, **not** a release plan or implementation authorization.
**Code-first review:** 2026-11-01, `main` source v0.27.2, initial inspected HEAD `ce62d5a45f6b5de7bc6b34ee55251aee86c47fca`.
**Owner direction:** Keep Boxes clean, use separate Boxes for parallel projects, prefer external nonintrusive weaving, reuse the installed Provider/SSH/HTTP MCP architecture, and keep optional coordination nearly invisible for ordinary single-Agent use.
**Scope rule:** The roadmap tracks *ideas and decisions*. Product requirements and delivery authority belong to [SCOPE](../product/SCOPE.md), [REQUIREMENTS](../product/REQUIREMENTS.md), [ACCEPTANCE](../product/ACCEPTANCE.md), and [PHASE_STATUS](../process/PHASE_STATUS.md). Installed evidence belongs to the [feature matrix](../verification/FEATURE_MATRIX.md) and [verification queue](../verification/REMEDIATION_TEST_QUEUE.md). An owner-approved *design* is not permission to implement, publish, install, connect a public listener, create a Box or modify a host.

## Containarium v0.128.0 minimum-only upgrade contract (2026-11-03)

The owner selected **minimum versions without any upper-version allowlist** for dependent Containarium components, replacing any proposal to maintain an exact release compatibility table. The [complete upgrade source assessment](CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md), [repeatable Agent SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md) and [v0.43.2 evidence ledger](../verification/CONTAINARIUM_0128_COMPAT_P0_0432_2026-11-03.md) own the details. This P0 slice preserves coequal REST/MCP adapters and does not change accepted ADR-003, E2/E3/E4 scope, the host's running Provider, or default stable pin 0.43.1.

## Current source and stable-release checkpoint (2026-11-02)

The historical phase narratives below are kept for decision provenance;
their original incomplete statements are *not* the latest code status.
The current source, v0.42.2 candidate, selects v0.42.1 as the separately
reviewed default stable installer following fresh and actual v0.28.0
upgrade real-Box checks. [Stable selection and limited acceptance](../verification/STABLE_PROMOTION_2026-11-02.md)
records both real test targets, independent release SHA and remaining
risk. Protected R-04 coordination remains opt-in/not installed-accepted
and E2–E4 remain upstream-blocked; stable promotion is **not** a claim
of complete roadmap implementation.


## Long-term architecture alternative — API-first internal Provider (2026-11-03)

**DESIGN CANDIDATE, NOT APPROVED OR IMPLEMENTED.** The owner asked to evaluate a hypothetical full redesign without legacy data/interface compatibility obligations, targeting preservation of approximately 99% of existing *user-visible* functionality. The resulting preferred **candidate** is **BoxWeave MCP outward / Containarium REST API inward / independent Box SSH Executor / host CLI for maintenance**, with an explicit guarded upstream MCP compatibility lane during transition. This does **not** change the accepted [ADR-003](ADR/0003-transport-and-trust.md), the installed Provider pin or the current stable release. The 99% target is not measured and does not relax security/recovery invariants.

Read [API-first long-term architecture master](API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md) for boundaries and decision gates; [Provider migration and Agent acceptance plan](API_FIRST_PROVIDER_MIGRATION_PLAN.md) for the per-feature mapping, seven exceptions, implementation difficulty and phases; [unified trace/D1–D4 diagnostics evolution](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md) for logging and observability; and [candidate ADR-010](ADR/0010-api-first-control-plane-candidate.md) for rationale and explicit adoption conditions. These are **future studies only**, not new requirements or authorization to start migrations.

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
- R-01 *read-only* discovery/file reads can ship without R-02/R-04. Its **new mutating** operations must join the R-04 protection boundary when that boundary is enabled, and use the R-02 tracked path when offering reconnect recovery. Existing `connect(exec)` remains available but must not be falsely described as deduplicated.
- R-02 provides idempotent tracked requests and known-result lookup. R-04 provides **resource coordination**, not another task-runner or result store. A thin shared operation claim/dispatch-intent contract links their records, while preserving separate app and gateway security principals.
- R-03 is independent of all of them: first use proves a safe authenticated MCP read, not task persistence, file mutation, leasing or new public network infrastructure.
- Deliver and accept small independent slices; no release must bundle all four projects.

## 3. Historical Box environment plan: E0–E4 (all DESIGN, candidate)

The 2026-10-30 [environment master](BOX_ENVIRONMENT_PROPOSAL.md) and its five implementation handoffs are complete **as designs only**. They are **not** current BoxWeave snapshot, restore or clone implementations. A Containarium tenant **database** backup is not an Incus/managed Box snapshot.

| Batch | What it would add | Gate / dependency | Child specification |
| --- | --- | --- | --- |
| **E0** | Read-only authenticated Provider capability / authority / identity discovery for exact managed Box snapshots/clones | First prove the *deployed* Provider supports typed, authorized operations; no direct-Incus fallback | [E0 capability boundary](BOX_ENV_E0_CAPABILITIES.md) |
| **E1** | Bounded immutable Box environment fingerprint and before/after difference, useful for old/new Box comparison | Prefer external observations; do not require a functioning in-Box diagnostic agent; E0 capability checks first | [E1 fingerprints](BOX_ENV_E1_FINGERPRINTS.md) |
| **E2** | List, create and delete snapshots of stopped, managed Boxes, with exact source identity and quotas | E0 authenticated Provider snapshot contract and shared conflict guard; E1 capture is **optional** metadata, not a prerequisite | [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md) |
| **E3** | Explicitly planned and confirmed restore from an exact snapshot to the same stopped Box | Requires E2 and clear coverage of attached/custom volumes; never promise a whole-host rollback | [E3 restore](BOX_ENV_E3_RESTORE.md) |
| **E4** | Clone an exact stopped Box or observed snapshot as a *new* independently registered managed Box | E0 verified Provider clone/rekey contract; E2 is required **only for snapshot-source clone**, not direct stopped-instance clone; new identity, network/SSH/tenant registration and quarantine still mandatory | [E4 clone](BOX_ENV_E4_CLONE.md) |

**Owner decision still required:** whether to add E0–E4 to product scope. The sensible first step is a bounded E0 capability audit, *not* implementation of all five. E1 may be useful even when snapshot/clone is unsupported, provided only observed metadata is advertised. E2/E3/E4 must independently stop at `unsupported` when the upstream contract is missing.

Note that historical E0/E1 child prose may call D0–D2 "not implemented": those are frozen historical references. D0–D2 later shipped in source v0.22.0; D3/D4 have subsequent source implementation and separate live evidence. Preserve design provenance but rely on [current source/lived evidence](../verification/FEATURE_MATRIX.md).

## Phase 3 environment implementation — supported Provider surface only (2026-11-02)

The owner authorized development and real-Box acceptance. [E0/E1 implementation contract](BOX_ENV_E0_E1_IMPLEMENTATION.md) now supplies exact-target authenticated Provider read-only reports, conservative tool/schema capability classification, bounded stable/runtime fingerprint comparison, and immutable actor-scoped Web captures using app SQLite schema v13. New write actions E2/E3/E4 are **not implemented** where the connected Provider lacks verified snapshot/restore/clone/rekey and storage-volume coverage contracts. No direct-Incus fallback is authorized. Source CI and disposable real-Box read proof must be recorded separately from installed BoxWeave/Web acceptance. R-04 global coordinated mutations remain incomplete, so environment writes cannot be claimed safe yet.

### Pending upstream v0.126.0 re-evaluation and handoff (2026-11-02)

[Containarium v0.126.0 upgrade and E2/E3/E4 implementation handoff](CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md) records the immutable upstream tag identity, exact snapshot REST/gRPC versus database-backup and whole-box restore distinctions, ZFS storage requirements, future isolated upgrade/rollback gates, authorized Provider adapter decisions, and explicit Agent tests for auto-sleep/wake. This is **planning only**; the operator host and stable installer pin are unchanged. The older 58-tool Provider MCP catalog was not proof that snapshot APIs were absent from REST/CLI. E2/E3/E4 remain **not implemented** in BoxWeave until deployed, tested and independently accepted.

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

## Phase 1 implementation (authorized 2026-11-01)

**Current checkpoint:** P1-A (claim core) and P1-B1 (Box identity/private DB adapter) were merged and source/fixture-accepted. [v0.29.0 main CI and source publication PASS](https://github.com/vanishs/boxweave/actions/runs/41404753003) at `160a09c4d2d2defff3adf3fe538861e28128b1a3`; unchanged stable v0.28.0 installer. P1-B2/P1-C (actual legacy mutation integration, provision/migration and cross SSH/HTTP/Web acceptance) remain **NOT IMPLEMENTED / LIVE U**. This is not global coordination or recovery readiness.


The owner authorized staged implementation, focused acceptance and commits for the unified operation foundation and related existing-path migration. The [Phase 1 implementation and acceptance ledger](../process/PHASE1_UNIFIED_OPERATION_BASE.md) is the active delivery handoff. P1-A source and fixture verification passed at `05a95aea` ([CI 41402675634](https://github.com/vanishs/boxweave/actions/runs/41402675634)). It introduces only a non-dispatching operation-claim core; it is **not** installed protection, a new MCP API, complete R-02 recovery or R-04 leasing. P1-B1 verified-identity/private-storage adapter passed source/fixture [CI 41403521130](https://github.com/vanishs/boxweave/actions/runs/41403521130) at `8ee12fd`, not installed provisioning. Subsequent adapters require verified source/transport/security gates before announcing coverage. Stable installation pins are unchanged.

## Intermediate architecture consolidation gate — ARCH-0 (2026-11-01)

**Inserted before P1-B2 and the next coordinated mutation integration stage.** The owner asked for a code-first audit of whether modules are excessively scattered. [Architecture consolidation source audit and staged acceptance](ARCHITECTURE_CONSOLIDATION_GATE.md) identifies six **logical ownership blocks** without requiring immediate package-wide renaming. The source audit is **S only**, not a completed refactor or live test.

ARCH-0A maps real executable roots, imports, sensitive authority, Provider mutations and deployment artifacts; ARCH-0B implements Provider-scoped identity and reviewed standalone/helper storage contracts; ARCH-0C extracts pure Provider/access/lifecycle interfaces behind compatibility exports. This remains separate from P1-B2 actual mutation integration. See [per-slice acceptance](ARCHITECTURE_CONSOLIDATION_GATE.md). ARCH-0A/B/C source release and fixture review PASSED at [main CI/publish 41410649605](https://github.com/vanishs/boxweave/actions/runs/41410649605), with immutable [v0.30.0](https://github.com/vanishs/boxweave/releases/tag/v0.30.0) targeting `06d1714021be9a0350982a02223ffefab9ba3305`, and at [CI 41409883731](https://github.com/vanishs/boxweave/actions/runs/41409883731); installed/host verification is still U. P1-B2 and Phase 2 must not claim integrated conflict protection while any gateway/app mutation bypass remains. Root/standalone installation helpers and generated client scripts are compatibility-sensitive; postpone cosmetic rearrangement. The prior roadmap's R-01/R-02/R-04 feature choices are unchanged.

## R-04 Web BoxAction opt-in source checkpoint

[Web BoxActions shared-claim adapter](PHASE2_APP_SHARED_ACTIONS.md) joins existing app receipt-backed BoxAction writes to the same resources as tracked SSH/HTTP Gateway exec, **only when app coordination.enabled=true**. Errors before upstream dispatch fail closed; observed successful terminal original receipts release claims; UNKNOWN/acknowledged operations retain them. Default app config remains unprotected for backwards compatibility pending migration, and legacy direct connect plus lifecycle/access remain outside guards. Thus neither global conflict coverage nor installed-host acceptance is complete.

## R-04 shared tracked-exec claim source checkpoint

[v0.37.0 partial integration](PHASE2_SHARED_DISPATCH_INTEGRATION.md) applies canonical Box+logical-name resources and durable dispatch intent to R-02 tracked SSH/HTTP Gateway exec requests. A root-published Provider/backend/project scope binds the Gateway identity to the app identity. Same-Box concurrency blocks; other Boxes continue. UNKNOWN retains the claim. **This is only one real dispatch class**: old direct connect, other MCP mutators, Web action workers and lifecycle/access remain outside enforcement; no global Box lease MCP is yet safe. Second stage and P1-B2 remain PARTIAL / live U.

## R-04 lease core incremental source checkpoint

The existing minimal coordinator now supports opt-in 10-minute hashed-token leases with renew/release/status and an expiry rule that **never** clears an unknown dispatched claim. [R-04 source design/verification](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md). Claim database provisioning is possible only through the authorized app-user task provisioning helper; `coordination_enabled=false` remains explicit. **No Gateway lease tool or actual app/SSH/HTTP mutator dispatch is enabled yet**, and no global concurrency protection or host acceptance is asserted. R-04 source core is a preparatory slice, not second-stage completion.

## Phase 2 incremental source delivery checkpoint

R-03 first-use guide and R-01 read-only Box file/dir operations passed exact-SHA CI and source release v0.31.0. R-02 gateway-private metadata-only task receipt adapter (design/implementation: [R-02 notes](PHASE2_GATEWAY_TASKS_IMPLEMENTATION.md)) passed [CI 41417633459](https://github.com/vanishs/boxweave/actions/runs/41417633459) store tests and [CI 41418231631](https://github.com/vanishs/boxweave/actions/runs/41418231631) gateway/packaging integration fixtures; release v0.32.0 and host acceptance are distinct pending gates. P1-B2/R-04 global cross-Web/SSH/HTTP coordinated mutations, optional leases, guarded BOI write and end-to-end live tests remain NOT IMPLEMENTED / U, not covered by the new task receipts.

## 5. Integration review: conflicts, reuse and corrected dependencies (2026-11-01)

This section resolves contradictions among the previously independent handoffs; it is **planning guidance, not a new approved implementation**. Implementing Agents must follow both the local child acceptance matrix and the shared gates below.

### 5.1 Integration decisions / conflict register

| ID | Overlap or conflict | Resolution (minimal shared contract) | Required child alignment |
| --- | --- | --- | --- |
| **INT-01 — R-02 vs R-04 persistence** | R-02 asks for gateway-owned task receipts and dispatch-intent; R-04 asks for cross-SSH/HTTP/Web shared operation claims/unknown recovery. Independent dispatch state machines could disagree after a crash. | **One authoritative, minimal resource-operation claim/dispatch-intent boundary** for mutations that are under conflict protection. R-02 retains its gateway-private request-key/result store; Web/app retains existing receipts; join using opaque operation ID and observed terminal evidence. Do not duplicate business receipts or share private outputs. | [R-02](MCP_TASK_RELIABILITY_R1.md), [R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) |
| **INT-02 — R-01 vs R-02 execution API** | R-01 proposes `boxweave_box_exec` via `connect(exec)`; R-02 proposes `boxweave_task_submit(operation=exec)`. Two independently dispatched exec engines would create ambiguous tracking and security semantics. | One internal approved `connect(exec)` execution adapter. Keep legacy `connect`; first ship R-01 safe reads. Prefer **one tracked execution entry** from R-02 for durable replies; R-01 `boxweave_box_exec` is a possible *alias/facade only*, not a second executor. Defer/omit it if it adds no user benefit. | [R-01](BOX_OPERATION_INTERFACE_V1.md), [R-02](MCP_TASK_RELIABILITY_R1.md) |
| **INT-03 — R-04 vs E0/E2/E3/E4 resource locks** | E0–E4 envisage separate source/destination locks, while R-04 already designs Box/name/Tenant/port claims. Parallel implementations could race or lock in opposite orders. | Use the same canonical Box-incarnation/name/shared-resource conflict vocabulary and one atomic claim-acquisition strategy **if R-04 is delivered**. E0 read-only discovery does not wait for it; E2/E3/E4 may not ship mutations without an equivalent cross-entry guard. Do not hold SQLite locks across remote calls. | [R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md), [E0](BOX_ENV_E0_CAPABILITIES.md), [E2](BOX_ENV_E2_SNAPSHOTS.md), [environment parent](BOX_ENVIRONMENT_PROPOSAL.md) |
| **INT-04 — E1/E2/E4 artificial sequencing** | Environment parent previously made E1 mandatory for E2 and E2 mandatory for all E4, although child designs make fingerprint optional and only snapshot-source clone requires E2. | E0 capability audit -> **E1 independent read-only capture** or **E2 stopped-instance snapshots**. E3 needs E2. E4 direct-instance clone needs E0 proven clone/rekey, whereas E4 snapshot-source clone also needs E2. All writes require shared claims or verified equivalent. | [E0–E4 parent](BOX_ENVIRONMENT_PROPOSAL.md), [E4](BOX_ENV_E4_CLONE.md) |
| **INT-05 — R-03 vs feature readiness** | First-use guide could accidentally require R-01/R-02/R-04, auto-create a Box, or label a local HTTP listener as an external connected Agent. | The standalone installer/onboarding gate is **authenticated `initialize` + `tools/list` + safe `list_containers` in the actual chosen client**. Expose advanced capabilities only as optional next steps. | [R-03](FIRST_USE_ONBOARDING_V1.md) |
| **INT-06 — P-05/P-03 overlap and M0–M5 history** | Source bundle/file transport and Agent test workflow overlap Box file tools and historical M3/M4, tempting duplicate transfer daemons and a new orchestration service. | R-01 owns small Box read/edit interfaces; P-05 requires a separately proven large/binary transfer gap. Keep P-03 developer workflow, not an automatic execution product. Reuse source-implemented app-side M3/M4 where actually authorized and installed. | [R-01](BOX_OPERATION_INTERFACE_V1.md), [Agent deployment](../process/AGENT_TEST_DEPLOYMENT.md), [M0–M5 status](../process/MCP_M0_M5_SOURCE_PROGRESS.md) |

### 5.2 Ownership, transaction and privilege boundaries

- **Cross-entry coordination is hard, not a new server:** R-04 needs a protected minimal claim authority reachable by **only** the already authorized app and gateway principals. R-02's gateway task-result storage must not become readable by the Web worker, and gateway must not be granted broad app SQLite access. A narrowly reviewed coordination SQLite store or an existing safe mediation boundary is acceptable; one mega-database/daemon is not the goal.
- **Single dispatch truth:** for protected mutations, acquire a short resource claim and durably mark dispatch intent **before sending any remote write**; link the resulting operation ID to the original R-02 or Web/app receipt. If writing either prerequisite fails, do not dispatch. Commit transactions before Provider/SSH calls. On a known result, persist its evidence then retire the claim; if the steps are interrupted, reconcile by the original receipt/target without replaying. Do not claim atomic distributed transactions between stores and Provider.
- **Unknown vs lease expiry:** a 10-minute lease may expire after Agent disappearance, but an unresolved dispatched operation claim stays until observed/reviewed. Expiry alone never proves a guest process ended. A lost standalone legacy/untracked `connect(exec)` has no invented R-02 receipt; R-04 must label its protection coverage exactly and retain conservative uncertainty when it can observe dispatch intent.
- **Security scope:** an Agent label or lease token is not independent authentication. Current single trusted operator and optional host-wide allowlist remain. Do not install an Agent in the Box, grant host shell authority, silently widen HTTP OAuth or assume client-specific RBAC.
- **Protect actual dispatch routes:** SSH direct MCP, isolated HTTP MCP and the Web/app paths covered by the feature must check the same claims. A subset may be developed/tested first, but **do not advertise global protection until every covered mutator and bypass path is audited**. Untyped forwarded mutators require an explicit block or a documented nonprotected compatibility boundary; they cannot silently evade an active lease.

### 5.3 Revised build order: independent slices, not one mega-release

```text
Standalone first use:       R-03 (SSH/HTTPS guidance + real safe read)
Standalone read-only:       R-01 BOI-0/BOI-1 read/capabilities
                           E0 (if environment features selected) -> E1 fingerprint
Shared mutation groundwork: INT-01 shared claim/dispatch contract + migration/security tests
                           -> R-02 tracked task receipts/known results
                           -> R-04 cross-entry conflict claims -> optional Box lease
                           -> R-01 guarded edits / any E2-E4 writes (if selected)
Environment branches:      E0 -> E2 snapshot -> E3 restore
                            E0 -> E4 direct clone (Provider clone/rekey proven)
                            E0 + E2 -> E4 snapshot-source clone
                           E1 fingerprint remains optional for E2/E3/E4
Later, only if needed:      P-01 sleep/wake, P-02 approvals, P-05 transfer, P-06 metrics,
                           P-03 orchestration and P-04 per-client RBAC remain deferred
```

The shared claim boundary can initially serve only a **tested, explicitly listed tool set**. R-02 may deliver standalone read-only task lookup sooner, but no mutating action may be declared safely coordinated until its particular dispatch path uses the shared boundary. R-04's default single-Agent behavior must remain unchanged except for real conflicts, and it does not require explicit Box leases for ordinary operations.

### 5.4 Joint acceptance tests (in addition to child tests)

| ID | Scenario | Done gate |
| --- | --- | --- |
| **INT-T01** | Same exact Box: direct SSH exec, HTTP tracked exec, Web/app action race | Conflicting dispatches cannot both proceed through a claimed protected route; different Boxes remain parallel |
| **INT-T02** | Gateway dies after dispatch intent; task receipt absent/partial, lease expires | Preserve conservative `UNKNOWN` claim, no automatic replay or unsafe takeover |
| **INT-T03** | Result persisted, gateway dies before resource claim release | Correlate original receipt and observed target; clean up only after evidence, without re-dispatch |
| **INT-T04** | E2 snapshot/restore or E4 clone races with Box stop/delete or same-name creation | Common canonical source/destination claims block collision; no separate incompatible lock namespace |
| **INT-T05** | First SSH/HTTPS client connecting before any new BOI/MTR/MAL tool installed | Existing `list_containers` succeeds; no lease/setup prerequisites or fake external ready status |
| **INT-T06** | Protected tool disabled/version mismatch, unknown forwarding, cross-principal DB access | Honest `unsupported/blocked/not_covered`, no policy bypass or broad app/receipt database access |

Evidence classes and installed/current revision must follow the [verification queue](../verification/REMEDIATION_TEST_QUEUE.md); successful fixture tests alone are not live cross-client acceptance.

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

## Phase 2 staged source acceptance — 2026-11-02

[Exact CI/Release evidence and outstanding gates](../verification/PHASE2_SOURCE_ACCEPTANCE_2026-11-02.md): v0.31.0 R-01(read-only)/R-03, v0.32.0 R-02 metadata-only task receipts, v0.33.0 R-04 **internal lease core only**. S/T/source release PASS for these individual slices; overall Phase 2 **PARTIAL**, since P1-B2 Web/SSH/HTTP shared dispatch, R-04 live tool/lease enforcement, guarded patch, and installed cross-entry recovery are NOT IMPLEMENTED or LIVE U. Stable PUBLISHED_RELEASE remains v0.28.0.

## Phase 2 source checkpoint: optional protected gateway mode (v0.39.0 candidate)

[Root-owned policy and forwarded-tool classification](PHASE2_PROTECTED_GATEWAY_POLICY.md) optionally denies direct connect, create/resize/delete, nesting apply and unclassified Provider writes before forwarding while exposing reviewed read tools and R-02 tracked exec. The marker is not provisioned/enabled by the installer; Web lifecycle/access writes are not covered. Source-only protection is not a global multi-Agent guarantee or installed acceptance.

## R-04 lifecycle app-worker source checkpoint (v0.40.0 candidate)

[Web lifecycle shared claims](PHASE2_LIFECYCLE_SHARED_CLAIMS.md) adds opt-in common Box/name/tenant resource claims for the app's existing create/start/stop/delete/resize worker. It records dispatch intent before Provider writes, retains the original app receipt, and releases a claim only after an observed desired-state terminal result. UNKNOWN, unverified resize and unverified nesting remain blocked. This joins the already opt-in Web BoxActions and tracked SSH/HTTP Gateway paths; Web Access/network/port changes, legacy external SSH and unclassified disabled-mode forwarded writes are still outside complete protection. The shared policy is **not enabled on the installed host**, and source CI is not a live Box proof. Overall P1-B2/R-04 still PARTIAL.


## Phase 2 cross-entry protected source slice (v0.41.0 candidate)

[Cross-entry source implementation and acceptance boundaries](PHASE2_CROSS_ENTRY_COMPLETION.md)
completes the source migration of the selected Web Access Tenant/port writes to
canonical shared claims, adds the **opt-in** protected Gateway lease endpoint
and a tracked 1024-byte optimistic file patch. All work uses original gateway or
Web receipts; UNKNOWN remains non-replayable. Root policy/DB/app opt-in must be
independently installed and dual-client tested. Outside-Gateway SSH/Provider
writes remain outside guarantee. Source release does not mean global installed
R-04 acceptance; detailed evidence remains separate.


## Phase 3 supported environment completion checkpoint (v0.42.0 candidate)

[Read-only E0/E1 development and real Box evidence](PHASE3_SUPPORTED_ENV_ACCEPTANCE.md)
documents strict observed-target fingerprint integrity, an authorized
two-Box Provider read/compare refusal, and real `v0.28.0` schema 12 to
v13 migration compatibility. E2/E3/E4 remain **BLOCKED by missing
authenticated upstream snapshot/restore/clone/rekey/storage contracts**;
nothing bypasses Provider with direct Incus. Source/fixtures are independent
of installed Web/SSH/HTTP and tested production upgrade/rollback. This
partial status must not be replaced with an unqualified Phase 3 PASS.


## ARCH-0 post-integration audit follow-up (v0.42.1 candidate)

[Source dependency and delivery findings](../verification/ARCH0_POST_INTEGRATION_AUDIT.md)
verify the original ARCH-0A/B/C source deliverables and close a
remaining CLI/deploy-check import cycle. Static tests enforce standalone
Gateway publication-parity and no app-private DB import into root
helpers. Six groups remain logical contracts. Installed shared-UID
coordination, independent old/new helper acceptance and cross-entry
runtime validation remain unverified, not silently accepted.


## Second-wave v0.42.6 execution checkpoint (2026-11-02)

[Exact source, installed multi-client, D4/E0/E1 and blocked feature ledger](../verification/BROAD_REACCEPTANCE_0426_2026-11-02.md). Two real Provider output-envelope bugs were closed in v0.42.5 (bare exit code) and v0.42.6 (stderr-only exit code). Scoped Gateway R-02 dedupe, R-01 CAS, R-04 protected lease + app primitive, D4 authorized diagnostics, and E0/E1 actual Provider/app test-client have live evidence; global Web-worker/cross-transport R-04 and external OAuth remain U, E2-E4 remain BLOCKED by missing Provider contracts. A new concurrent Provider create UNKNOWN race remains OPEN for follow-up; no broad new operator permissions were activated. Overall roadmap PARTIAL; stable default v0.42.1 unchanged.


### v0.28.0 direct installed upgrade checkpoint (2026-11-02)

[Second independent test Box v0.28.0 -> v0.42.6 acceptance](../verification/BROAD_REACCEPTANCE_0426_2026-11-02.md): pinned fresh older install and direct current upgrade both exit 0, authenticated doctor 10/10 both sides, real SQLite schema 12 -> 13 with old admin credential record unchanged, old v12 backups retained. This is live evidence, not a fixture. v0.26.x, fleet rollback/downgrade and unrelated Provider/Agent blockers remain U; successful task-owned outer Box cleaned up.


## 2026-11-02 v0.42.8 corrective acceptance and future Provider gate

The [isolated installed v0.42.8 D4 + upstream idle-stop acceptance](../verification/BLOCKED_REACCEPTANCE_0428_2026-11-02.md) supersedes the old *D4 positive blocked* and *upstream has no sleep implementation* assumptions: real exact-Box authorized user service active/inactive and bounded log reads passed; the actual Provider 0.120.1 idle-stop ticker stopped a guest once its last-start timestamp was recorded. Newly created cold-Box idle tracking and HTTP route wake remain unaccepted. E2/E3/E4 remain unimplemented at the BoxWeave layer. The [future v0.126.0 upgrade and API/storage handoff](CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md) owns the next gated work; the owner's Provider and default stable pin were not upgraded.


## 2026-11-02 P-01 upstream cold-start and HTTP wake candidate closure

The [fresh nested two-client HTTP route and cold-create acceptance](../verification/CONTAINARIUM_0126_AUTOSLEEP_WAKE_LIVE_2026-11-02.md) upgrades the **isolated *candidate* proof**, not the stable product: patched Containarium v0.126.0 achieved initial `LastStartedAt` stamping→real idle stop and Caddy wake proxy→first request HTTP 200→original guest body, with default-deny untrusted 403. Source code and focused tests were committed to the owner's [upstream fork branch](https://github.com/vanishs/Containarium/commit/40b9d650584ceef66ea196ec6ee6dd9d88263f2f); the upstream PR could not be created by the integration (403), so merge/release/official Provider upgrade are pending. This does not implement BoxWeave E2/E3/E4 or change BoxWeave v0.42.8/stable v0.42.1. The [Containarium 0.126 upgrade handoff](CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md) remains the delivery plan.


## 2026-11-03 upstream MCP follow-ups and host-hop packaging decision

**Owner-selected design, documentation only.** [Containarium upstream MCP follow-ups and host-hop SSH ownership](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md) records **P1 stable external MCP contract**, **P2 whole-Box backup create MCP parity**, **P2 trusted host-owned age restore keys**, and **Option A**: do not bundle vendored `containarium-shell` in future BoxWeave releases; automatically fetch/install the pinned, SHA-256-verified upstream wrapper **when missing on a BoxWeave-managed local Provider**, while externally managed Providers remain read-only without explicit repair authorization. The v0.42.9 wrapper-containing release is historical and immutable; installed host login shells must not be automatically removed. Upstream [#3500](https://github.com/FootprintAI/Containarium/issues/3500) makes no compatibility guarantee and excludes BoxWeave code changes. Current production Provider and stable pins are unchanged; all work remains separately gated. The child document links back here.


## 2026-11-03 UP-SSH-01 source implementation checkpoint (v0.42.10 candidate)

The selected **Option A** is source-implemented: future BoxWeave Release manifests exclude the vendored wrapper; a BoxWeave-managed local Provider fetches an immutable upstream commit with source and body hashes if missing, and external Providers require explicit repair. Errors in optional host-hop setup do not prevent independent MCP application delivery. [Exact source + Box verification](../verification/CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md) records the 661/661 Python tests and bounded real root-sandbox proof. Containarium binaries stay at pinned v0.120.1. CI, actual host-hop login, old-version installed upgrade, release publication and stable promotion are independent gates. [Child design and upstream P1/P2/P2](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md).

## 2026-11-03 shipped UP-SSH-01 v0.42.10 post-release checkpoint

[Owner-selected upstream dependency retrieval](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md) **SHIPPED** in [BoxWeave v0.42.10](https://github.com/vanishs/boxweave/releases/tag/v0.42.10); [PR #52](https://github.com/vanishs/boxweave/pull/52) and main CI passed. Real disposable Box published-v0.42.9 fresh install and published-v0.42.10 upgrade both reached `ready`; `boxweave doctor` 8/8 required PASS, existing wrapper unchanged, local Containarium remained **0.120.1**. [Post-release evidence](../verification/CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md). The actual host-hop SSH/wake/authorization lane is still **U** and default stable pin remains **0.42.1**. Upstream contract P1, backup parity P2 and host-owned age identity P2 remain future upstream decisions, not part of this shipped slice.


## 2026-11-03 UP-SSH-01 real SSH hop follow-up

[Live SSH acceptance in a new disposable Box](../verification/CONTAINARIUM_SHELL_REAL_SSH_HOP_2026-11-03.md) **PASS** for sshd→`containarium-shell`→nested guest command, host-IP SSH, STOPPED→RUNNING wake and invalid-key rejection with published BoxWeave v0.42.10 / pinned Containarium v0.120.1. **Remaining external dependencies:** remote sshpiper ingress and cross-target negative tests. **Security risk:** upstream host user has unrestricted passwordless `/usr/bin/incus` sudo; this is **not** tenant-isolation acceptance. Retain original P1 MCP contract / P2 backup parity / P2 host-held age key planning and do not mix this evidence with a new Containarium release upgrade. [Child owner decisions](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md).

## 2026-11-03 selected P1/P2 lightweight BoxWeave implementation (v0.42.11 candidate)

[P1 same-connection MCP subset diagnostic and P2 same-host restore-key guard](MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md) are **SOURCE IMPLEMENTED / TESTED** (release acceptance separate). Agent invokes `boxweave_diagnose(source=compat)` to compare critical tool schemas without polling; future stable upstream schema **contract** itself remains an independent Containarium P1 request. For restore, BoxWeave uses the existing upstream `restore_box_backup` when available and requires root-owned exact approval + local protected age key alias, no arbitrary Agent path and no new key service. Upstream `create_backup` whole-Box scope is still a distinct P2 gap. [Operator procedure](../process/MCP_COMPAT_RESTORE_OPERATOR_04211.md) · [source tests](../verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md). Containarium binaries stay pinned at 0.120.1; no stable pin promotion.
