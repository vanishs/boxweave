# Phase 4 feature/evidence matrix

All rows describe partial minimum-system progress, not first-release completion. Sources: SRC-030..032; details in [VERTICAL_SLICE](VERTICAL_SLICE.md). No deferred phase-5 feature is silently dropped.

| Requirement | Criterion | Tasks | Current evidence/status |
| --- | --- | --- | --- |
| REQ-001 | AC-001 | TASK-003 | Native connection probe only; full enrollment/replacement integrity path not implemented |
| REQ-002 | AC-002 | TASK-003 | Windows pin preflight and synthetic MCP mismatch; live trust chain unverified |
| REQ-003 | AC-003 | TASK-003 | Bounded POSIX client + native connection-only probe; real registration/UI incomplete |
| REQ-004 | AC-004 | TASK-001, TASK-002, TASK-005 | Typed scope/creation identity + zero-dispatch-on-change tests; direct Platform identity separate from application live path |
| REQ-005 | AC-005 | TASK-002, TASK-006 | Intent/one-use/no-replay groundwork; real create not in slice |
| REQ-006 | AC-006 | TASK-002, TASK-005 | Persisted deadline and unknown states, fixed result validation; live overall bound/cleanup not proven |
| REQ-007 | AC-007 | TASK-002, TASK-004 | SQLite receipt query/restart and synthetic loss recovery passed |
| REQ-008 | AC-008 | TASK-012 | Unstarted installer/recovery release work |
| REQ-009 | AC-009 | TASK-012, TASK-013 | Explicit fixture/live prerequisite failure only; full installed-health contract unstarted |
| REQ-010 | AC-010 | TASK-002, TASK-004 | Binding create/update/revisions/persistence verified; unbind/live binding continuation pending |
| REQ-011 | AC-011 | TASK-004, TASK-005 | Real authenticated loopback Web/browser fixture verified; production HTTPS/live services incomplete |
| REQ-012 | AC-012 | TASK-009 | Not started; requires phase-4 developer gate before feature work |
| REQ-013 | AC-013 | TASK-010 | Not started; no routes published |
| REQ-014 | AC-014 | TASK-011, TASK-013 | TASK-011 basic composition passed: real SQLite migration/CAS/API and Chromium card/comment/context/link/edit/copy/export/delete, zero graph provider calls. SRC-052; release TASK-013 remains pending |
| REQ-015 | AC-015 | TASK-008 | Not started; no model requests or credentials moved |
| REQ-016 | AC-016 | TASK-003, TASK-012 | Exact Linux wheel locks/source hashes; enrollment payload trust distribution incomplete |
| REQ-017 | AC-017 | TASK-005, TASK-013 | Linux/browser fixture + Windows preflight evidence; no real selected-profile acceptance |
| REQ-018 | AC-018 | TASK-004, TASK-005 | Target/action allowlist, auth/CSRF, actor-bound receipts; upstream broad privileges/production boundary unverified |

Latest SRC-041 disposition: shared-Tenant/nested/live application MCP verification is deferred and temporarily untested, not passed. Existing component/fixture evidence stands. Phase 5 is not running; TASK-011 is only a technical readiness candidate pending the remaining developer gate/explicit execution request.

SRC-045: native same-Tenant new-box creation, matching Guard attachment and app-to-backend TCP22 connectivity passed. Independent backend installation exit 124; managed launcher absent; live MCP and HTTP worker receipt remain NOT EXECUTED. Existing fixture demo remains active. Phase 5 not started.

SRC-047: supporting backend install exit0, managed timers/oneshots/marker checks, and actual application-hosted strict SSH MCP initialize/initialized/tools-list101/read-only tools-call passed. One inner create rejected and target reconciled absent; core PostgreSQL nested startup proc-mount permission failure observed. No real command receipt/HTTP live pass or whole phase4 acceptance.

## Current result — real chain verified (SRC-049/050)

Version 0.1.1 fixes a real queue-claim timing race. Thirty Linux tests pass, including a new regression that failed before the fix. An explicitly approved nesting-enabled backend now supports the real application -> independent managed MCP -> inner workload path. HTTP and Chromium binding/confirm/hello/receipt/restart/no-replay plus live trust/identity rejection passed. See [LIVE_MCP_CHAIN](../verification/LIVE_MCP_CHAIN.md) for exact identities, operation IDs, versions, commands and limits.

The previous missing-launcher/nested-startup blocker is resolved for this new lab only, not the old box or every LXC. TASK-005 selected Web path has live evidence; real Windows-native enrollment/Agent GUI acceptance remains incomplete. Developer required this real chain before phase 5 (SRC-048); no phase 5 feature was started and architecture/slice acceptance must still be recorded before TASK-011 or other new features. Original published demo remains fixture; isolated test servers are stopped, resources retained, no new public port or host-security change.

## Phase 5 iteration 1 complete — TASK-011 (SRC-051/052)

Developer acceptance of the minimum architecture/slice is explicitly recorded in SRC-051, after verified real Web/MCP evidence. TASK-011 basic Prompt Graph is implemented in 0.2.0: bounded cards, editable comments/context, undirected links, revision-checked persistence, conflict-preserved drafts, Markdown/clipboard handling and graph-only deletion. REQ-011/014/016 and AC-011/014/016 graph subset are evidenced; other selected-feature/client obligations remain incomplete.

Actual non-root project-LXC checks: 41 tests OK (11.173s), TypeScript/Vite build, final Chromium graph workflow including mobile fit and clipboard failure, original project hello/restart fixture regression; all exit0. Graph API forbids provider access in tests; browser has zero queued/executed operations. Thirteen changed source/lock files match remote SHA256. See PHASE_05_TASK_011.md for commands and precise evidence. No new box, infrastructure mutation, deployment, persistent listener, public route or push. Stop this iteration.

Next suggestion: finish TASK-003/005 minimum actual Windows-native enrollment/recognition evidence, then dependent TASK-006. Developer gate acceptance does not fabricate those missing client checks or waive per-task prerequisites. Phase 5 overall is not complete.

## Iteration 2 complete — TASK-003a, version 0.2.1 (SRC-053/054)

Scoped existing-identity Windows-native Codex registration now has trusted pre-execution payload verification, strict live connection gating, targeted config preservation, explicit replacement, byte-idempotence and backups. Actual isolated Windows/CLI/native MCP and failure-path tests passed; 41 project Linux tests and the three prior native preflight checks passed. No global/personal registration, new key, model request, mapping, box or deployment changed.

TASK-003a is complete in its stated existing-identity scope; parent TASK-003/005 remain partial. Actual GUI/TUI loading, bootstrap identity provisioning and other client/runtime profiles are unverified. Named-mutex/hash/backup recovery does not replace quiescing non-cooperating editors. See TASK_003A_NATIVE_REGISTRATION for commands, limitations and cooperative next steps. Stop this iteration; suggest TASK-003b, do not waive dependencies for TASK-006.
