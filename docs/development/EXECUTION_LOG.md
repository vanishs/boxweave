# v1 development execution log — actions, decisions, failures and handoffs

**This is the append-only development ledger, not a summary of all design docs.** New Agents read [Current checkpoint](#current-checkpoint) and newest matching log entry before the current [phase card](phases/README.md). Write each **meaningful development action, decision, native test result, failed attempt, owner block, docs correction and next-step hint**, even if a phase was PARTIAL. Do not log secrets, private TLS key material or sensitive guest content.

**Rules:** [Phase execution protocol](PHASE_EXECUTION_PROTOCOL.md). **One plan:** [Implementation entrypoint](../process/IMPLEMENTATION_PLAN.md). Older native SDK evidence remains under [verification/](../verification/ACCEPTANCE.md), not copied into every entry.

## Current checkpoint

| Field | Current recorded fact |
|---|---|
| Last updated | 2026-11-08; S1 started from verified v1 HEAD **e5c7ae26bdaa24f9243b7dec28dbd34ca561bcb2**; implementation branch `feat/v1-s1-foundation-20261108`. |
| Branch | **feat/v1-s1-foundation-20261108** from **v1** |
| Code state | **Docs-only design baseline; no v1 Go controller/installer exists yet.** Root VERSION=1.0.0 is not a release proof. |
| Completed work | SDK P0 historical only. **S1 IN_PROGRESS**; S2–S5 NOT_STARTED. Initial source is docs-only. No candidate binary yet. |
| Next intended phase | **S1 in progress** — [foundation](phases/01_FOUNDATION.md). Implement, test and attempt isolated acceptance. |
| Immutable phase boundary | Public catalog **20 tools**; no Containarium, new REST transport, Caddy, generic ACL service or Incus/Cowsql DB patch. |
| Known SDK limits | Native unsafe file conditional replace/delete → UNSUPPORTED until safe; detached Exec Operation.Cancel unsupported; one shared ACL/Scriptlet selected; partial NIC ACL update and actual v1 acceptance not yet tested. |
| Original UNKNOWN / operational cautions | **Current S1 isolated Box create UNKNOWN:** name `bw-v1-s1-1108`, task `7e1cbf043d1fd4ea3d9ec0093a04371e`, key `1d6007f5972bd21b01f89eacd42c3160`; upstream ACK absent. One read-only get returned NOT_FOUND and debug reported missing; **do not create/retry/delete/reclaim**, pending decisive original-receipt and native reconciliation. Older main-branch UNKNOWN and native Cowsql also remain independent. |
| Owner decisions needed right now | **None for planning.** Future shared-host upgrades/exposure, mandatory positive file CAS behavior or unresolved release-critical native enforcement may require explicit approval; log and continue other tasks when reached. |
| Next 1–3 actions | 1) Preserve/create reconciliation of original S1 Box UNKNOWN, no new risky request; 2) implement source/CI isolated from unavailable Box; 3) run source CI, record blocked native IDs and open PR against v1. |

**Update this small checkpoint before context compaction, agent handoff or a new native mutation.** The checkpoint is intentionally mutable; all evidence/decisions below remain append-only. Record actual HEAD/working branch and binary hash once a real implementation begins. If multiple Agents work in parallel, do not overwrite one another's original failure entries.

## Log entry format

New entries go **newest first after this template**, or append by monotonically increasing timestamp/run ID while maintaining the index. Keep each entry short and link details instead of pasting full outputs. Never silently edit a historical event; append a correction with a pointer to the earlier entry.

| Field | Record |
|---|---|
| Run ID / UTC timestamp / Agent role | Unique stage/action identifier, who performed it |
| Phase / subtask / status | S1–S5; PASS/PARTIAL/FAIL/BLOCKED/UNKNOWN/SKIPPED; any DECISION_REQUIRED |
| Before state | Branch/source SHA, installed artifact hash/version, exact authorized environment, prior original native UUID and op/request key if applicable |
| Action | Sources read **on demand**, code/doc files changed, API/CLI calls, test/verify commands and why |
| Observations | Real native/HTTP output, denied effects, exact acceptance IDs and evidence tier/link, failures and root-cause confidence |
| Decision or correction | What changed and why, owner/authorization implications, doc↔code consistency and cross-phase impact |
| After state | New commit/PR, original residual UNKNOWN claims, task-owned cleanup, remaining blockers |
| Resume | Next safe action, relevant **smallest** docs to read, do-not-repeat warnings |

## Recorded entries

### LOG-20261108-S1-UNKNOWN-001 — isolated S1 Box create ambiguous (2026-11-08)

- **Phase/test:** S1 native development environment bootstrap / **UNKNOWN** (not a v1 Box acceptance case); original Box name `bw-v1-s1-1108`, backend `dev`, create task `7e1cbf043d1fd4ea3d9ec0093a04371e`, request key `1d6007f5972bd21b01f89eacd42c3160`.
- **Exact original request:** one `create_container` with cpu=4, memory=8GB, disk=50GB, nesting enabled; response `REST_GATE_PENDING`, `submitted`, upstream_acknowledged=false. Original task status read returned `UPSTREAM_CREATE_OUTCOME_UNKNOWN`, state=unknown, upstream_acknowledged=false.
- **Independent read-only observations:** one `get_container` returned NOT_FOUND; `debug_container` reported `containerState=missing`, daemon `0.128.0`, no host user. These are time-local negative reads, **not proof that no native dispatch can have occurred** and do not revoke the original UNKNOWN.
- **Safety decision:** no replay of `create_container`, no new Box with different name to evade original safety lock, no auto-delete/adopt and no Incus remediation. Save IDs and continue **independent Go source, CI, documentation, mock/unit tests**. Native installed S1 tests remain BLOCKED until an authorized lab is unquestionably available. Next probe: read *the same original task* and exact native ID once available.
- **Before/after:** implementation branch only; no confirmed new native resource or production configuration change. No S1 PASS inferred.

### LOG-20261108-S1-START — S1 opened, initial environment check (2026-11-08, implementation Agent)

- **Phase/status:** S1 / **IN_PROGRESS**; no application PASS yet. Verified remote v1 HEAD `e5c7ae26bdaa24f9243b7dec28dbd34ca561bcb2`; implementation branch `feat/v1-s1-foundation-20261108`.
- **Read:** AGENTS, current single plan, execution protocol, S1 card/checkpoint, scoped boundary and Go SDK/architecture/MCP security and Tracking contracts. No v1 Go sources existed at entry. Old installed BoxWeave self-report source version `0.50.4` is not the v1 candidate.
- **Environment observation:** container local Go version `go1.23.2`; local `git ls-remote` failed DNS resolution for github.com. Reviewed tagged Incus Go SDK requires Go 1.26.9+, so this local runtime is **not** a valid compile/acceptance environment. Existing remote Boxes are not owned by this run and must not be modified.
- **Action/authority:** prepare one **new, uniquely owned disposable development Box**, rather than touch the existing stable host, older Box or shared Incus configuration. S1 does not authorize global Scriptlet, shared host upgrade, daemon reboot or foreign-resource change.
- **Next:** record original new Box receipt/identity and worker code/Go pins, then run build, MCP native HTTP and negative tests. Any native uncertainty remains UNKNOWN with no retry.

### LOG-20261108-001 — new context-safe development orchestration (2026-11-08, documentation Agent)

- **Phase/status:** pre-S1 / **DOCS_COMPLETED**, not an implementation/SDK retest. This entry records the user's requested phase orchestration and document refactor; **no native Box, daemon or host config was accessed or changed** in this task.
- **Before:** branch v1 was docs-only; previous [IMPLEMENTATION_PLAN](../process/IMPLEMENTATION_PLAN.md) contained useful 0–7 implementation order but mixed historical research and active work. [ACCEPTANCE](../verification/ACCEPTANCE.md) had detailed named A-/SDK-G test cases; [MCP_TOOLS](../design/MCP_TOOLS.md) declared exactly 20 tools; [AGENT_WORKFLOW](AGENT_WORKFLOW.md) specified Finder/Fixer/Verifier and no false PASS. Latest verified SDK scope: [P0 last-gate results](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md).
- **Read and compared:** GitHub v1 tree and the existing implementation plan, docs index, project boundary, tool catalog, contract/operations, detailed acceptance, security, SDK research, release/installer and old agent workflow. No claim of source/runtime validation.
- **Decision D-DEV-001:** Keep **IMPLEMENTATION_PLAN.md as the sole top-level execution entry**; make [PHASE_EXECUTION_PROTOCOL](PHASE_EXECUTION_PROTOCOL.md) the single reusable procedure, [phase cards](phases/README.md) the five small self-contained work packages and this log the append-only continuity ledger. Other detailed design/verification docs remain authoritative for their original facts, **linked just in time** instead of copied.
- **Decision D-DEV-002:** Split work into **S1 foundation (1 tool), S2 Box lifecycle/operations (12 tools), S3 guest files/Agent workflow (3 tools), S4 network/nesting (4 tools), S5 full black-box/release (0 additional tools)**. S2 dependent Box deletion is **explicitly reaccepted in S4**, and all 20 tools get independent end-to-end S5 acceptance; no split permits silent loss of functionality.
- **Decision D-DEV-003:** Each S# must attempt scope/research/development/test/independent acceptance/fix/retest/commit **in one pass**. Repeated hard failures or required operator decisions become precise log/issues with evidence; **unblocked work continues**. A security-critical unknown blocks only unsafe dependent action and formal release, not all progress.
- **Decision D-DEV-004:** Document correction is allowed **at any checkpoint**, including shrinking context by adding links/archiving proven history. Every changed contract must include a cross-reference and acceptance-impact note; never overwrite original incident facts or call fake/SDK-only evidence installed PASS.
- **Decision D-DEV-005:** Explicitly retain the owner's **no Incus/Cowsql patch** non-goal. Native file CAS and detached Exec cancel limitations are contract truths; maintain safe refusals, no hidden replacement tool/service.
- **Files intended in this documentation-only change:** this execution log, [implementation entry](../process/IMPLEMENTATION_PLAN.md), [phase execution protocol](PHASE_EXECUTION_PROTOCOL.md), [five cards](phases/README.md), and backlinks from [AGENTS](../../AGENTS.md), [docs INDEX](../INDEX.md), [AGENT_WORKFLOW](AGENT_WORKFLOW.md), [acceptance](../verification/ACCEPTANCE.md) and README where appropriate. Exact final commit SHA belongs in the final verification/follow-up entry, not guessed in advance.
- **Tests:** link existence / phase scope inventory / GitHub branch file readback only; **no Go tests, native probes, release, or production Box changes requested or performed**. Never report these documentation checks as v1 product PASS.
- **Next:** after all links and stage mapping are written, verify current branch head and doc references; then a **new development Agent may start S1** directly with a bounded context window. Any later correction must be appended as LOG-20261108-002 or a new timestamped entry, not silently change this history.

### LOG-20261108-002 — published phase docs, cross-links and documentation checks (2026-11-08, documentation Agent)

- **Phase/status:** pre-S1 / **DOCS_COMPLETED**. No Go runtime, installer, L0/L1 Box, native Incus, live MCP, release asset or production setting changed by this **documentation-only** task.
- **Created new focused files:** [common phase protocol](PHASE_EXECUTION_PROTOCOL.md), [five-phase index](phases/README.md), [S1 foundation](phases/01_FOUNDATION.md), [S2 lifecycle/operations](phases/02_BOX_LIFECYCLE_OPERATIONS.md), [S3 guest files](phases/03_FILES_AGENT_LOOP.md), [S4 network/nesting](phases/04_NETWORK_NESTING.md), [S5 exact-release acceptance](phases/05_RELEASE_ACCEPTANCE.md), plus this immutable [execution log](EXECUTION_LOG.md). Eight new files total including the log.
- **Reorganized existing entrypoints and corrected cross-references:** [IMPLEMENTATION_PLAN](../process/IMPLEMENTATION_PLAN.md) is the **single active phase summary** replacing the mixed 0–7 slice/historical-note format. [AGENTS](../../AGENTS.md) was compressed from ~10.2k to ~8.0k characters, retaining essential safety/SDK invariants and just-in-time source links. [README](../../README.md) now points to the one-shot phases and fixes a broken unordered feature list while retaining the global Scriptlet warning. [INDEX](../INDEX.md) points to the source-of-truth map. [AGENT_WORKFLOW](AGENT_WORKFLOW.md) focuses on stage-time testing instead of requiring every design file up front. [ACCEPTANCE](../verification/ACCEPTANCE.md) specifies cross-phase delete/file/network gate closure; [OPEN_RISKS](../verification/OPEN_RISKS.md) now instructs per-ID loading and distinguishes historical evidence.
- **Stage mapping checked:** **S1=1**, **S2=12**, **S3=3**, **S4=4**, **S5=0 newly introduced**, collectively **20 declared public MCP tools**; S2 box_delete with network dependencies is **reaccepted in S4**, while every installed Tool and CLI leaf gets final S5 black-box coverage. Permanent scope and previous SDK results were not amended as though new runtime testing occurred.
- **Validation actually performed:** GitHub v1 branch tree readback showed **56 files** and all 8 new files, up from original **48**. All **14 reviewed new/changed entry/phase/acceptance documents** fetched successfully; **283 relative local Markdown links** in that set were checked against actual v1 Git tree with **zero unresolved path targets**. This verifies paths, **not** external URLs/heading anchors/Go compilation/native behavior.
- **Specific decisions restated:** five small self-contained phase cards, one shared procedure, an append-only log plus short mutable current checkpoint; avoid repeated historical research; safe fixing and evidence preservation without stalling unrelated tasks; no automatic native UNKNOWN replay or assumed release PASS; no Incus Cowsql repair component.
- **Commit evidence:** new docs began at **f3696a4bd88c96bc583eb17524134ba391d62728** (phase protocol), stage cards through **f276b4d3848d6eb9a7701c83ac3587863e025d1b**, current implementation entry **380df361f2b4496073e6a5b92eec177408237a56**, compact Agent entry **4fbb05e8c23c68cde6820ca5971c3c679db7816a**, final crosslink/risk note **b42be46ca2355506e5285db06f97c07f29af8419**. **The actual final HEAD will differ after this log update**; use Git branch readback, not this stale pre-log SHA, as the current source.
- **Outstanding:** **all S1–S5 remain NOT_STARTED**; this was documentation orchestration only. Native SDK-known positive/file-CAS limitations and product release gates remain unchanged. No release candidate or installed v1 Tool PASS can be inferred.
- **Next safe entry:** verify current v1 HEAD, read the **one plan, one protocol, S1 card and current checkpoint**, then **execute S1 all the way through its own code/build/negative/native acceptance/fix and documented handoff**. New context should *not* load S2–S5 or long historical research unless S1 reaches that exact dependency.

## Future entries

A new Agent adds a separate entry whenever an S# phase starts, a risk decision changes, a native mutation becomes UNKNOWN, a fix is independently accepted or a phase hands off. Long machine logs and full acceptance proofs live under verification/ (or protected test-artifact paths) and are linked by exact ID.

When this file exceeds a reasonable one-session reading size, move **older immutable entries** into year/month archives under docs/development/logs/ with backlinks here and a searchable run-ID index. Preserve the **checkpoint + latest few entries** in this file; never erase evidence. For future Agents, read old archives **only by needed run ID**.
