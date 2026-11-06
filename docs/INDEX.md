**2026-11-05 controlling release approach:** [Agent-managed local reacceptance and direct publication](process/AGENT_MANAGED_ACCEPTANCE_RELEASE.md) requires real independent Agent/Box checks and immutable original evidence, but no release-specific GitHub workflow, special submission account, signer or protected Runner. Existing source CI remains ordinary checks; an Agent directly publishes through existing repository write access only after real full acceptance. The [full-project runbook](process/FULL_PROJECT_REACCEPTANCE.md) remains the detailed black-box test contract; [historical release trust proposal](knowledge/REACCEPTANCE_SIMPLIFICATION_REVIEW.md) is not adopted.

**v0.44.10 targeted correction:** [selftest private-plan handoff, typed diagnostics, test and release acceptance ledger](verification/SELFTEST_PLAN_HANDOFF_04410_2026-11-05.md); R-02 owner-host reboot test explicitly deferred, no global R-04 activation.

# BoxWeave v2 knowledge index

**Current system-first reacceptance guidance (2026-11-05):** [System-level scenario/invariant matrix](verification/REACCEPTANCE_SYSTEM_SCENARIOS.md) defines the primary **L2 workflow / L3 cross-entry fault** hunt and independent Finder→Fixer→Verifier evidence; [low-coupling automation and remaining developer handoff](design/REACCEPTANCE_SYSTEM_BUG_HUNT_IMPLEMENTATION.md) specifies **RACC-SYS-01..09**, a repo/CI/test-Box-only future runner, dynamically resolved upstream release-tag source materials and an enforced candidate-before-public-release gate. **v0.47.0 already includes reacceptance modules in the production app ZIP**; their removal is an **unimplemented packaging migration**. The canonical trigger and L1 all-action inventory remain unchanged, but **system invariants outrank tool-count completion**. No code, customer asset or stable pointer is changed by this guidance.

**Owner's single-command full reacceptance:** `Run full BoxWeave reacceptance` → [active exhaustive MCP/CLI black-box-first multi-Agent runbook](process/FULL_PROJECT_REACCEPTANCE.md) → [run inventory, every-command test ledger and owner-facing functional results template](verification/REACCEPTANCE_RUN_TEMPLATE.md) → [historical F01–F39 plus current feature acceptance baseline](verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md). Every installed MCP tool and CLI leaf must receive a bug-seeking real invocation where safe, or its own precise BLOCKED/UNKNOWN record. Combine prior call-chain/coordination/integration acceptance plans. Final results must name **every function**, tests and uncovered paths—not only a pass/fail count. Separate Finder/Fixer/Verifier when possible; independently observe actual effects, not merely already-green Python fixtures. Documentation changes do not themselves invoke this workflow, publish a release, or upgrade a host.

**Upstream Containarium reacceptance integration (2026-11-05):** [Pinned upstream evidence/reference guide](verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) explains which upstream DOCMAP, Proof Pack, CLAIMS, proto/CLI/MCP contract, failure drill, release rehearsal and upgrade runbook mechanisms can safely be used; [execution orchestration development plan](design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md) specifies Agent invocation paths, adapter capabilities, isolated proof producer, run artifacts, candidate gates and proposed RACC-AUTO-01..07 implementation milestones. **The v0.47.0 product source implements a scoped read-only-default execution/recording CLI and explicit-opt-in specialist adapters; these documents alone install nothing and do not authorize arbitrary upstream root execution.** The original owner trigger and [canonical runbook](process/FULL_PROJECT_REACCEPTANCE.md) remain authoritative.

**Current architecture decision (2026-11-05): MCP-first + REST-assisted hybrid.** Keep upstream MCP as the default for new/specialist Containarium operations and BoxWeave SSH/HTTP MCP for Agents; retain **seven shipped app-owned REST lifecycle operations**. The universal API-first redesign and bulk **8 P1 + 21 P2** REST migration are **ABANDONED**, not pending tasks. [Authoritative current architecture](design/ARCHITECTURE.md), [updated ADR-010](design/ADR/0010-api-first-control-plane-candidate.md), [archived migration study](design/API_FIRST_PROVIDER_MIGRATION_PLAN.md), [architecture history](design/API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md).

**Current v0.44.9 architecture:** [runtime call chains, ingress guard coverage and P0/P1 acceptance map](design/RUNTIME_CALL_CHAINS_0449.md). This is a source-backed view, **not** global R-04 production acceptance. The authoritative current implementation map is [ARCHITECTURE](design/ARCHITECTURE.md).

**Five-level evidence depth:** [v0.44.9 feature-by-feature evidence](verification/EVIDENCE_LAYERS_0449.md) compares source review, automated tests, isolated installed, owner host, and independent external-client proof. Test levels are independent, not a product coverage percentage.

**Latest exact-version evidence:** [v0.44.9 immutable release, independent installed v0.44.8→v0.44.9 upgrade, live nested selftest and outer R-02](verification/REACCEPTANCE_0449_2026-11-05.md). Source 742 Python tests and seven browser fixtures passed; owner host global R-04 is still disabled. See [PHASE_STATUS](process/PHASE_STATUS.md) and [OPEN_RISKS](verification/OPEN_RISKS.md).

Start here, not in a chronological transcript. [PHASE_STATUS](process/PHASE_STATUS.md) owns current gates; [VERSION](../VERSION) owns the product version. This index owns navigation only.

| Containarium minimum-only upgrade and reusable Agent SOP | [v0.128.0 compatibility review](design/CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md), [standard upgrade SOP](process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md), [v0.43.2 P0 acceptance ledger](verification/CONTAINARIUM_0128_COMPAT_P0_0432_2026-11-03.md): only component version floors, no upper bound; stable pointer unchanged |
| Find | Authoritative route |
| Containarium running-daemon minimum follow-up | [v0.43.3 runtime gate](verification/CONTAINARIUM_0128_RUNTIME_GATE_0433_2026-11-03.md): independently checks authenticated daemon process version, not only on-disk daemon/MCP binaries; no upper version cap |
| --- | --- |
| Upstream MCP priorities and future host-hop packaging | [2026-11-03 owner decisions](design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md): stable MCP subset P1, whole-Box backup MCP P2, host-held age identity P2, Option A for `containarium-shell`: unbundle from Release, auto-fetch verified upstream dependency on managed local Providers, opt-in only on external Providers (design only) |
| Real SSH forwarding/wake via containarium-shell | [2026-11-03 real SSH evidence](verification/CONTAINARIUM_SHELL_REAL_SSH_HOP_2026-11-03.md): command hop, stopped wake, unapproved-key denial **PASS**; host Incus sudo scope and sshpiper/cross-tenant checks **OPEN** |
| Upstream host-hop pinned fetch source verification | [v0.42.10 isolated tests and open installed gates](verification/CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md): 661 Python tests, root-sandbox network/download checks, no new Containarium binary |
| v0.42.11 MCP compatibility probe and local restore-key guard | [Lightweight design](design/MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md), [operator procedure](process/MCP_COMPAT_RESTORE_OPERATOR_04211.md), [source/test evidence](verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md); upstream Provider version unchanged |
| Agent-safe host diagnostic coordinator v1 | [DC-01 design](design/DIAGNOSTIC_COORDINATOR_V1.md), [v0.43.0 exact installed-live acceptance and stable pin](verification/STABLE_PROMOTION_DC01_0430_2026-11-03.md): read-only preflight, authenticated REST/MCP and actual D2 Trace PASS in isolated nested host; original dev host unchanged |
| Current and candidate feature roadmap | [Global feature roadmap](design/FEATURE_ROADMAP_MASTER.md): selected designs, historic E0-E4 proposals, deferred candidates, priority/dependencies and child backlinks |
| Future Agent upstream integration handoff | [Containarium v0.126.0 upgrade and E2/E3/E4 gates](design/CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md): pin, isolated upgrade, authenticated API mapping, snapshot/backup/clone boundaries, sleep/wake verification |
| Upstream candidate P-01 live acceptance | [Isolated Containarium v0.126.0 cold-create and HTTP wake test](verification/CONTAINARIUM_0126_AUTOSLEEP_WAKE_LIVE_2026-11-02.md): candidate source fix, fork commit, real two-Box HTTP result, cleanup and upstream PR gate |
| New Agent / developer startup | [FINAL_HANDOFF](process/FINAL_HANDOFF.md) |
| Current work / authorization | [PHASE_STATUS](process/PHASE_STATUS.md), [current handoff](process/HANDOFF.md) |
| Product goal and selected boundaries | [PRODUCT](product/PRODUCT.md), [SCOPE](product/SCOPE.md) |
| Required outcomes and acceptance | [REQUIREMENTS](product/REQUIREMENTS.md), [ACCEPTANCE](product/ACCEPTANCE.md) |
| Actual runtime / module design | [ARCHITECTURE](design/ARCHITECTURE.md), [ADR index and status](design/ADR/INDEX.md) |
| Transport-to-Provider call graph, R-04 entry coverage and follow-up tests | [Current v0.44.9 runtime call chains](design/RUNTIME_CALL_CHAINS_0449.md) (S; see separate T/L evidence) |
| Task dependencies | [IMPLEMENTATION_PLAN](process/IMPLEMENTATION_PLAN.md) |
| Evidence depth and next acceptance by feature | [v0.44.9 five-level evidence matrix](verification/EVIDENCE_LAYERS_0449.md) |
| REQ/AC → code/tests/evidence | [FEATURE_MATRIX](verification/FEATURE_MATRIX.md), [code-first acceptance baseline](verification/CODE_FIRST_ACCEPTANCE_2026-11-01.md), [latest selftest trust-readiness remediation](verification/SELFTEST_TRUST_RACE_0267_2026-11-01.md), [OAuth transaction deep audit](verification/DEEP_AUDIT_0266_2026-11-01.md), [nesting and partial-create acceptance](verification/NESTING_ACCEPTANCE_0265_2026-11-01.md), [post-v0.26.2 full-project reacceptance](verification/REACCEPTANCE_0262_TO_0264_2026-11-01.md), [previous live ledger](verification/CODE_FIRST_LIVE_ACCEPTANCE_2026-11-01.md) |
| Host-hop wrapper release and bounded remediation | [v0.42.9 Containarium shell source/test record](verification/CONTAINARIUM_SHELL_PACKAGE_0429.md): managed Provider install, external opt-in, host-hop acceptance still open |
| Containarium host-hop SSH missing wrapper | [Read-only diagnosis and operator-only recovery](process/CONTAINARIUM_SHELL_RECOVERY.md): distinguish direct Box MCP from missing backend shell; never run broad sudoers setup automatically |
| Agent-driven rapid development/test workflow | [AGENT_TEST_WORKFLOW](process/AGENT_TEST_WORKFLOW.md), [Agent deployment v1.0](process/AGENT_TEST_DEPLOYMENT.md) and [local source bundle helper](../scripts/agent_source_bundle.py) (developer practice) |
| How to run and verify | [INSTALL](INSTALL.md), [test strategy](verification/TEST_STRATEGY.md), [validation runner](../scripts/validate.py) |
| Current unresolved risks | [OPEN_RISKS](verification/OPEN_RISKS.md) |
| New questions/proposals | [OPEN_QUESTIONS](knowledge/OPEN_QUESTIONS.md) |
| Evidence/source identities | [SOURCE_REGISTER](knowledge/SOURCE_REGISTER.md) |
| Legacy observations / review baseline | [BEHAVIOR_BASELINE](knowledge/BEHAVIOR_BASELINE.md), [LEGACY_SYNC](knowledge/LEGACY_SYNC.md) |
| Daily change and knowledge workflow | [MAINTENANCE](process/MAINTENANCE.md), [machine-readable authority map](process/DOCUMENT_AUTHORITY.json) |
| Latest maintenance/cold-start proof | [PHASE_07_REVIEW](verification/PHASE_07_REVIEW.md) |

Earlier evidence remains available: [phase6 comparison](verification/COMPARISON_REPORT.md), [phase5 release](verification/TASK_012_013_RELEASE.md), [native network/ports](verification/TASK_009_010_ACCESS.md), [historical index](INDEX_HISTORY.md). These are dated/revisioned records, not current gate owners. Archived status/handoff/matrix snapshots are linked from their current owners.

Use local Markdown links and stable REQ-/AC-/TASK-/ADR-/SRC-/RISK-/DEC-/EXP- IDs. Reports record evidence; the code, schema and installed manifest remain the truth for actual runtime state. Documentation does not deploy an application.

## Feature planning (global parent; no implementation authority)

- **[Feature roadmap master](design/FEATURE_ROADMAP_MASTER.md)** — one status-aware view of selected Box operations, MCP task reliability, first-use onboarding and multi-Agent collision prevention; historical E0–E4 environment proposals; and conditional/deferred MCP/workflow candidates. Individual design documents link back to this master. Source implementation, real installed acceptance and product approval remain separate.

## 0.10+ operator remediation routes

- [Management CLI](CLI.md)
- [Host bootstrap and upstream ownership](HOST_BOOTSTRAP.md)
- [0.5.1 source parity audit](verification/ARCHIVE_051_TO_MAIN_AUDIT.md)
- [New change validation queue](verification/REMEDIATION_TEST_QUEUE.md)

- [New host/CLI architecture decision](design/ADR/0007-host-install-boundary.md)

- [F-001–F-015 semantic parity](verification/FEATURE_PARITY_ARCHIVE_051.md)
- [Old/new Web workflow navigation and browser validation](verification/WEB_WORKFLOW_PARITY.md)

- [Operator troubleshooting](TROUBLESHOOTING.md)
- [Manual migration from archived 0.5.1](MIGRATING_FROM_051.md)
- [Upstream Provider/Incus contract](UPSTREAM_PROVIDER.md)

- [Development and isolated fixture](DEVELOPMENT.md)

- [Historical installation-guide snapshot](INSTALL_HISTORY.md)

- [0.14.0 main completion and current acceptance evidence](verification/MAIN_014_REVIEW.md)

- [Streamed installer and two-box MCP acceptance](verification/STREAMED_INSTALL_TEST.md)

- [Container nesting](NESTING.md) and [measured verification](verification/NESTING_VERIFICATION.md)

## Diagnostics implementation (D0–D2 in 0.22.0; D3/D4 source in 0.25.0)

- [Current source/behavior audit](verification/DIAGNOSTICS_SOURCE_AUDIT.md)
- [Evidence-first CLI and existing-MCP implementation proposal](design/DIAGNOSTICS_PROPOSAL.md)
- [D0 implementation specification — diagnostic correctness](design/DIAGNOSTICS_D0_IMPLEMENTATION.md)
- [D1 implementation specification — trace and evidence](design/DIAGNOSTICS_D1_IMPLEMENTATION.md)
- [D2 implementation specification — reports and nested diagnosis](design/DIAGNOSTICS_D2_IMPLEMENTATION.md)
- [D3+D4 owner-approved master](design/DIAGNOSTICS_D3_D4_MASTER.md)
- [D3+D4 implementation and tests](design/DIAGNOSTICS_D3_D4_IMPLEMENTATION.md)
- [D3/D4 operator guide](DIAGNOSTICS.md)

## Box environment management research (proposal; NOT IMPLEMENTED)

- [Architecture — Box checkpoints, cloning and environment fingerprints](design/BOX_ENVIRONMENT_PROPOSAL.md)
- [E0 — Provider capability / security boundary](design/BOX_ENV_E0_CAPABILITIES.md)
- [E1 — Environment fingerprint capture and difference](design/BOX_ENV_E1_FINGERPRINTS.md)
- [E2 — Snapshot create, list and delete](design/BOX_ENV_E2_SNAPSHOTS.md)
- [E3 — Snapshot restore](design/BOX_ENV_E3_RESTORE.md)
- [E4 — Independent Box cloning](design/BOX_ENV_E4_CLONE.md)

These environment features are **design-only**; D0–D2 shipped in 0.22.0, while D3/D4 are source-implemented in 0.25.0 with live acceptance separate.


## MCP task reliability R1 (historical design; R-02 implemented with scoped acceptance)

- [Reconnect-recoverable MCP task records, deduplication, bounded result retrieval and UNKNOWN safety](design/MCP_TASK_RELIABILITY_R1.md). Historical design objective; the later R-02 metadata-only tracked-task implementation now has real owner-host submit/get/list/dedup evidence. It does **not** store guest output or guarantee a terminated process continues; see the current call-chain map and R-02 post-reboot acceptance gap.

## Multi-Agent resource conflict protection (historical design; R-04 partial, not globally active)

- [Minimal Box lease and operation conflict protection — shared resources, fail-fast conflicts, 10-minute expiry, UNKNOWN safety, implementation batches and acceptance](design/MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md). The later R-04 claim/lease code and protected isolated tests exist, but owner-host global enforcement and the authenticated Web/SSH/HTTP concurrency matrix remain **PARTIAL/P0**. Multiple projects keep separate Agents/Boxes; no automatic runner or whole-fleet coordination should be inferred.

## Installation and first-use onboarding (research/design only)

- [First-use onboarding v1 — source audit, SSH/HTTPS guided paths, first authenticated MCP read, readiness states, phased development and acceptance](design/FIRST_USE_ONBOARDING_V1.md). This document proposes UI/CLI guidance only; it does not claim a shipped `boxweave setup` command or an installed v0.27.1 host test.

## Box operation interface v1 (design only; not implemented)

- [External-first Box operation interface — capabilities, file operations, command results, safety, implementation batches and acceptance](design/BOX_OPERATION_INTERFACE_V1.md). Preserves clean Boxes and existing Provider/SSH architecture; this design does not authorize code or deployment changes.

## MCP capability integration (source inventory and staged implementation)

- [Master design and updated main status](design/MCP_CAPABILITY_INTEGRATION_MASTER.md)
- [Existing feature gap audit](verification/MCP_CAPABILITY_SOURCE_AUDIT.md)
- [Platform tool matrix](design/MCP_PLATFORM_CAPABILITY_MATRIX.md)
- [agent-box tool/resource matrix](design/MCP_AGENT_BOX_CAPABILITY_MATRIX.md)
- [Trust boundaries](design/MCP_INTEGRATION_SECURITY.md)
- [M0–M5 delivery handoff](design/MCP_INTEGRATION_DELIVERY_PLAN.md) and [acceptance](verification/MCP_INTEGRATION_TEST_PLAN.md)

MCP integration is distinct from D0–D2 diagnostics and E0–E4 environment proposals. PR #6 is research/design and must not be reported as runtime feature completion.

## MCP integration implementation checkpoint

- [Actual source completion, CI evidence, open decisions and next developer handoff](process/MCP_REUSE_IMPLEMENTATION_HANDOFF.md) (historical checkpoint; see later 0.24.0 live M0–M5 evidence and current code-first register)

## M0–M5 MCP source implementation (live Box acceptance deferred)

- [M0–M5 source state, operator configuration, Web API and remaining live acceptance](process/MCP_M0_M5_SOURCE_PROGRESS.md). Direct-gateway tool policy is optional host-wide enforcement, **not per-user RBAC**; agent-box requires a separately reviewed fixed launcher and live Box testing.

## Remote MCP HTTP(S) and OAuth (implemented; public deployment acceptance is separate)

- [Master design, eight owner decisions and phase map](design/REMOTE_MCP_MASTER.md)
- [ADR-009: keep SSH, add remote Streamable HTTP with no vendor tunnel dependency](design/ADR/0009-remote-mcp-transports-and-auth.md)
- Phased implementation: [R0 shared dispatcher](design/remote-mcp/R0_PROTOCOL_AND_DISPATCH.md), [R1 HTTP](design/remote-mcp/R1_STREAMABLE_HTTP.md), [R2 OAuth/API Token/credential URL](design/remote-mcp/R2_AUTH_AND_CREDENTIALS.md), [R3 HTTPS/proxies](design/remote-mcp/R3_HTTPS_AND_PROXY.md), [R4 Web onboarding](design/remote-mcp/R4_WEB_CLIENT_EXPERIENCE.md)
- [Negative and positive acceptance matrix](verification/REMOTE_MCP_TEST_PLAN.md), [Agent implementation handoff](process/REMOTE_MCP_HANDOFF.md)

**Code implemented:** v0.24.0 introduced remote HTTP MCP/OAuth and the capability remains in v0.25.0. Historical loopback HTTP/OAuth and local TLS-proxy tests are recorded; real public TLS, all external clients and current installed v0.25.0 require distinct acceptance. SSH/stdio remains supported, Web cookies are not OAuth MCP credentials, and the host-wide allowlist is not per-Agent RBAC.

- [Remote HTTP MCP source setup, CLI and safety limits](REMOTE_MCP.md) (v0.24.0 source only; real public/TLS acceptance pending)
