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

ARCH-0A maps real executable roots, imports, sensitive authority, Provider mutations and deployment artifacts; ARCH-0B implements Provider-scoped identity and reviewed standalone/helper storage contracts; ARCH-0C extracts pure Provider/access/lifecycle interfaces behind compatibility exports. This remains separate from P1-B2 actual mutation integration. See [per-slice acceptance](ARCHITECTURE_CONSOLIDATION_GATE.md). ARCH-0A/B/C source and fixture review PASSED at [CI 41409883731](https://github.com/vanishs/boxweave/actions/runs/41409883731); installed/host verification is still U. P1-B2 and Phase 2 must not claim integrated conflict protection while any gateway/app mutation bypass remains. Root/standalone installation helpers and generated client scripts are compatibility-sensitive; postpone cosmetic rearrangement. The prior roadmap's R-01/R-02/R-04 feature choices are unchanged.

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
