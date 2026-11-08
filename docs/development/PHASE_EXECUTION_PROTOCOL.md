# Development phase execution protocol — context-safe, continuous and evidence-bound

**Role:** Rules for running the five development phases in [the single implementation entrypoint](../process/IMPLEMENTATION_PLAN.md). Read this once when beginning or resuming any phase. This is a **workflow document, not another implementation plan**. The immutable v1 product boundary remains [PRODUCT_BOUNDARY](../decisions/PRODUCT_BOUNDARY.md); the runtime is **not built as of 2026-11-08**.

## 1. Entry, context budget and just-in-time reading

A new Agent starts with **only** [AGENTS.md](../../AGENTS.md), [implementation entrypoint](../process/IMPLEMENTATION_PLAN.md), this protocol, [current phase card](phases/README.md), and the **recent current-phase tail** of [EXECUTION_LOG](EXECUTION_LOG.md). Fetch **the relevant subsection** of an authoritative design/verification document **when** the phase card reaches that subtask. Do not read all research, historical commits, old source, large acceptance and all five phase cards in advance.

Before an actual task: record branch/current HEAD, existing local/remote edits, current installed/version/digest, original external Box/operation identity if any, authoritative scope and relevant acceptance IDs. Treat old reports as upstream facts for their **exact tested versions**, never evidence that current v1 has passed. Use current code and native readback as reality, and record conflict with a design promise; do not silently redefine the promise.

**Document precedence on conflicts:** latest explicit owner decision and permanent [scope](../decisions/PRODUCT_BOUNDARY.md) / [operational policy](../decisions/P0_OPERATIONAL_POLICIES.md) → [public tool catalog](../design/MCP_TOOLS.md) and [tool contracts](../design/MCP_TOOL_CONTRACTS.md) → domain design docs → frozen SDK evidence → earlier historical research. Current implementation and real evidence are **observations**, not permission to weaken a higher-priority product contract. Check owner approvals before privileged/shared-host actions.

For phase-specific context, each phase card has two tiers: **required on entry** (few files) and **read only at named subtask**. Open the relevant path/section, not the whole repo. A referenced acceptance ID may be fetched just before its test; no need to preload all of [ACCEPTANCE](../verification/ACCEPTANCE.md).

## 2. One phase = one continuous end-to-end execution

Each major phase is meant to be completed **in one autonomous development pass** by the next executing Agent, not split into "only investigate", "only write plan" or "only test" deliveries. Subtasks may be delegated or developed in isolated branches/Boxes, but the phase owner remains responsible for integrating and verifying the **same source SHA and artifact**.

Perform these **internal checkpoints in order**, looping locally on failures:

1. **Scope / fetch:** read just-in-time sources, check previous phase handoff and stable immutable IDs, enumerate the actual tool/CLI leaves and named acceptance cases for this subtask.
2. **Focused research:** verify any still-unknown Incus/MCP API semantics against tagged SDK code and existing real proofs; avoid reopening already proven P0 research without new contradicting evidence.
3. **Implement:** small typed Go changes, official SDKs, bounded SQLite journal, own resources only; no speculative service, host modification or accidental reuse of unstable v0.x components.
4. **Developer tests:** source build, unit/race/negative checks, protocol client and narrowly authorized disposable native tests; record exact command, test inventory, output and candidate SHA.
5. **Acceptance:** end-to-end against exact current candidate using independent native poststate, prohibited-effect witnesses, and independent verifier when available; never translate fake/native-mock PASS into real-installed PASS.
6. **Fix and rerun:** reproduce, explain, update code **and/or docs**, rerun the original failing case plus related regressions. For recurring unresolved problems, collect evidence and continue **other unaffected subtasks**; do not spin indefinitely or silently mark done.
7. **Close / handoff:** recheck every phase-specific deliverable, preserve evidence and IDs, clean **only owned confirmed** disposable objects, update the status table, append the action log, commit coherent code/docs, and write exact next resume instruction.

**Docs may be corrected at any checkpoint**, not just completion: fix source↔docs inconsistency, remove duplicate or superseded instructions, add small cross-links, or split material to save future context. Record the reason and any acceptance impact in the log. Do not rewrite a historical test as though it never failed.

## 3. Never lose a failure or block the entire campaign unnecessarily

Use **PASS / PARTIAL / FAIL / BLOCKED / UNKNOWN / SKIPPED** with exact version, layer, test type and artifact SHA. A partial phase can have completed artifacts and outstanding named gates. A phase cannot be declared PASS while a mandatory phase-local deliverable is unverified; a real release must not pass if mandatory product-wide gates remain open.

If a local item cannot be repaired after a concrete reproduction and bounded reasonable fix/retest attempts:
- log original request/key, Tracking ID, Box UUID/native operation ID, environment, actual observed result, attempted corrections and remaining uncertainty;
- add/update a narrowly scoped issue or [OPEN_RISKS](../verification/OPEN_RISKS.md), with severity, evidence, next safe probe and the **affected acceptance IDs**;
- mark only the dependent item BLOCKED/UNKNOWN; **proceed with remaining independent phase tasks and, if possible, independent next-stage tasks**, recording any conditions;
- keep safe ownership locks for UNKNOWN effects; **never blindly replay a possibly committed mutation**;
- if an explicit operator choice is indispensable, record **DECISION_REQUIRED** with options, impact and a recommendation in the [log](EXECUTION_LOG.md); continue other permitted tasks. Do not silently choose a new authority, external exposure or feature reduction.

**Publication and unsafe mutations are exceptions to “continue”:** lack of required evidence, new high-risk host permission or an unresolved security-critical blocker prevents only the dependent action/release; don't bypass it for schedule. No automatic host reboot, root MCP daemon, foreign-instance mutation or global policy overwrite without appropriate local authorization.

**Environmental Cowsql issue:** nested Incus missing leader is recorded as **environment-only** until evidence proves otherwise. Never create a BoxWeave Incus/Cowsql database fixer/daemon; only ordinary exact-identity UNKNOWN reconciliation is within scope. See [final SDK evidence](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md).

## 4. Continuity and context-compaction safety

The execution log is an **append-only decision and operations trail**, not an infinitely growing prompt. Its [current checkpoint](EXECUTION_LOG.md#current-checkpoint) and newest entries have priority; read older entries by link/run ID only for a specific failure. If it grows too large, archive an older date range under docs/development/logs/ and preserve an index/link; do not discard incident IDs, decisions or test proofs.

**Update the checkpoint before context compaction, handoff, major retry, or committing to a new Box.** Write, in at most one short screen:
- phase/subtask, branch/HEAD, working tree/PR, installed artifact digest/environment, actor and timestamp;
- **exact completed** acceptance IDs and evidence links, next 1–3 concrete actions;
- original pending/UNKNOWN resources and **do-not-repeat operations** (request keys, native UUID/op IDs);
- unresolved blocks/owner decisions, fallback permitted without changing scope;
- file/doc links to read *next*, not a transcription of their contents.

A new Agent **must verify actual Git/Incus/CI state first**, not assume the last Agent's narrative is the current truth. Resume the existing phase from original evidence; no automatic restart or replacement Box. If a prior Agent failed between writing and committing, reconcile the diff and read-only native poststate before acting.

## 5. Evidence and status bookkeeping

**One source of each kind:** design contract in design/, actual native/HTTP proof in verification/, scope decisions in decisions/, work execution in this log, phase work breakdown in one card per phase. Link rather than copying long paragraphs across files.

Each acceptance entry must identify **case ID**, test mode (unit/fake, built HTTP, native L0/L1/L2, independent external witness), artifact/source hash, attempted API/CLI path, observed actual state including negative effects, and exact evidence location. **Only a native/installed test of the actual v1 candidate can close product acceptance**; isolated upstream SDK P0 results are not that. [Acceptance matrix](../verification/ACCEPTANCE.md) is the authoritative catalog; phase cards provide only the relevant ID subsets.

Log meaningful commands, file changes, PR/commit or issue IDs, chosen alternatives, failures/unknowns, bounded fix attempts, changes to phase order/contracts, evidence location, and cleanup. Do **not** log bearer tokens, TLS private keys, raw secret output or private user data. Host-specific identifiers belong in protected local logs where necessary; the public log can use safe hashes, redacted IDs and links.

## 6. Cross-phase contracts

- **Phase 1 → 2:** runnable Go service, typed MCP/auth, exact identity/Project confinement, SQLite schema and operation admission, host-local install/rollback and native version ledger.
- **Phase 2 → 3:** owned Box lifecycle, Exec, reconnectable outputs, operation receipt/idempotency/UNKNOWN. Network-dependent delete cleanup is declared an explicit **Phase 4 reacceptance dependency**, not silently stamped fully verified in Phase 2.
- **Phase 3 → 4:** guest files and full minimal Agent development loop with honestly unsupported conditional replace/delete when atomicity cannot be proven. Do not implement a new patch engine.
- **Phase 4 → 5:** complete **20/20** tools, one shared installer-created ACL, native forwarding, peer grants, normal Box deletion of owned dependents, nested install/loop, per-NIC/IPv6/foreign negative gates and public LAN witness.
- **Phase 5 → publication:** exact candidate binary/source/provenance, independent installed verification, adversarial and negative tests, CI gate and documented residual issues. **Unverified mandatory release gates block publication**, not unrelated bug fixing.

**Do not label Phase 5/release PASS by aggregating earlier SDK unit probes or only counting tests.** The five [phase cards](phases/README.md) and [implementation entry](../process/IMPLEMENTATION_PLAN.md) together describe the full work; historical research is loaded just in time.
