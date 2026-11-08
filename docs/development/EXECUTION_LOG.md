# v1 development execution log — actions, decisions, failures and handoffs

**This is the append-only development ledger, not a summary of all design docs.** New Agents read [Current checkpoint](#current-checkpoint) and newest matching log entry before the current [phase card](phases/README.md). Write each **meaningful development action, decision, native test result, failed attempt, owner block, docs correction and next-step hint**, even if a phase was PARTIAL. Do not log secrets, private TLS key material or sensitive guest content.

**Rules:** [Phase execution protocol](PHASE_EXECUTION_PROTOCOL.md). **One plan:** [Implementation entrypoint](../process/IMPLEMENTATION_PLAN.md). Older native SDK evidence remains under [verification/](../verification/ACCEPTANCE.md), not copied into every entry.

## Current checkpoint

| Field | Current recorded fact |
|---|---|
| Last updated | 2026-11-08 |
| Branch | **v1** |
| Code state | **Docs-only design baseline; no v1 Go controller/installer exists yet.** Root VERSION=1.0.0 is not a release proof. |
| Completed work | SDK final P0 native/HTTP investigation **completed previously**, with verified limitations and cleanup. **Development phase S1–S5 NOT_STARTED**. 5-phase execution docs/links created as documentation only. |
| Next intended phase | **S1** — [foundation](phases/01_FOUNDATION.md). Read only [AGENTS](../../AGENTS.md), [implementation plan](../process/IMPLEMENTATION_PLAN.md), [phase protocol](PHASE_EXECUTION_PROTOCOL.md), S1 card and this checkpoint. |
| Immutable phase boundary | Public catalog **20 tools**; no Containarium, new REST transport, Caddy, generic ACL service or Incus/Cowsql DB patch. |
| Known SDK limits | Native unsafe file conditional replace/delete → UNSUPPORTED until safe; detached Exec Operation.Cancel unsupported; one shared ACL/Scriptlet selected; partial NIC ACL update and actual v1 acceptance not yet tested. |
| Original UNKNOWN / operational cautions | Prior outer L0 BoxWeave main uncertain create is **not part of this docs-only run**; do not retry or clean. Original native nested Cowsql events are environment records, not permission to mutate daemon. See [SDK final report](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) for exact prior cases. |
| Owner decisions needed right now | **None for planning.** Future shared-host upgrades/exposure, mandatory positive file CAS behavior or unresolved release-critical native enforcement may require explicit approval; log and continue other tasks when reached. |
| Next 1–3 actions | 1) Verify current v1 HEAD and no concurrent edits; 2) Execute S1 scope/read/implementation/installed acceptance in one pass; 3) append S1 evidence, cleanup, blockers and S2 handoff. |

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

## Future entries

A new Agent adds a separate entry whenever an S# phase starts, a risk decision changes, a native mutation becomes UNKNOWN, a fix is independently accepted or a phase hands off. Long machine logs and full acceptance proofs live under verification/ (or protected test-artifact paths) and are linked by exact ID.

When this file exceeds a reasonable one-session reading size, move **older immutable entries** into year/month archives under docs/development/logs/ with backlinks here and a searchable run-ID index. Preserve the **checkpoint + latest few entries** in this file; never erase evidence. For future Agents, read old archives **only by needed run ID**.
