# Test strategy and experiment plan

Status: selected phase5 obligations executed through TASK-012/0130.9.0, with153 non-root Linux tests59.570s,21 distribution cases, actual systemd parser, built UI, actual installed-browser/package/failure-repair and Windows four-profile regression/guarded fixed commands. See [integrated report](TASK_012_013_RELEASE.md), [package evidence](task012-live-evidence.json), [client evidence](task013-profiles.json) and [exact source](task012013-source-manifest.json). Original strategy below is historical planning, not a substitute for the named actual results. Fixture/provider/packet/human/native-model evidence and unsupported deployment claims remain separate. No phase6 or inference ran.

## Layers and evidence

| Layer | Boundary and repeatable cases | Environment and task |
| --- | --- | --- |
| Unit/contract | Target normalization, revision/CAS, malformed result/cursor/protocol, plan confirmation/digest/expiry, model/profile selection, redaction | Deterministic fake provider/process/clock; TASK-001/003/007/008 |
| Storage/integration | Real SQLite migrations/foreign keys, binding conflicts, worker claim/crash/DB-full, receipt discovery, rollback and unknown dispatch | Non-root assigned LXC, isolated temporary state; TASK-002 |
| HTTP/browser | Real application+DB with fake provider; sessions, CSRF/Origin, login/reset, plans/receipts, graph persistence/copy failures, no arbitrary command proxy | Chromium fixture with controlled synthetic resources; TASK-004/011 |
| Provider contract | Strict API/capability/Guard/policy revisions, default fields, partial/unknown write, port conflict/IP reuse | In-memory/HTTP fixtures first; authenticated real read-only capability later; TASK-006/009/010 |
| Client compatibility | Native Windows and explicit WSL transport, actual Codex/ZCode registration/UI and independent diagnosis | Actual selected client; parsing/config fixtures are not real enrollment; TASK-003/005/007/008 |
| Real end-to-end | Project selection/binding -> trusted MCP identity/hello -> receipt/restart; later create/Tenant/ports/model completion | Reviewed dedicated resources and operator-selected client; TASK-005/013 |
| Install/repair | Staging/tamper/locking/saved config/inactive-successful oneshot/pending marker/Web failure independence | Non-root Linux fake-host sandbox, then explicitly reviewed isolated live install; TASK-012 |
| Difference/release | Map all ACs to actual evidence; explain deliberate legacy differences and remaining gaps | TASK-013/014; no universal/bare-metal/security claims from nested fixtures |

Proposed runners after test code exists: Python standard unittest discovery for backend/client contract suites, PowerShell AST parsing without setup execution, TypeScript typecheck + build, Node model tests and Playwright Chromium. Actual commands/paths and pinned runtime versions are recorded with the implementation; don't report these planned runners as executed. Run targeted tests first, then relevant regressions; no mirror tests for documents.

## Failure-path obligations

- Block trust/target mismatch before command dispatch. Similar names, IP reuse, copied UUID and unsupported config cannot cause fallback to another resource.
- Kill worker before intent, after durable dispatch marker, during delayed response and after remote success; query/restart must not submit again. Verify provider call counts. Test DB error before intent means zero dispatch; unknown after intent remains unknown.
- Persist one deadline per phase; monotonic clock for in-process work, consumed recorded budget across restarts, rollback detected as incomplete. Test excessive stdout/stderr, EOF, nonzero command exit and cleanup uncertainty. No claim that killing wsl.exe kills every remote descendant.
- Require exact stale-plan/revision conflict and policy all-or-none, preserved unrelated ingress/egress fields, account/shared key separation and malformed removal rejection. A stored revoke during degraded enforcement remains pending.
- Basic graph actions never call provider mutation; copy history records success only after clipboard succeeds; model settings remain client-local and distinct from MCP connection.
- Include source-payload tamper tests and synthetic-secret leakage cases; never test using real private keys/JWT contents in fixtures or output. Real credential use only through normal protected interfaces.

## Four launch profiles

| Profile | Agent runtime | SSH/trust runtime | Required evidence |
| --- | --- | --- | --- |
| Codex/native | Windows | Windows OpenSSH | enrollment intent/preservation, actual MCP/tool/identity, registration/UI loading |
| Codex/WSL | Windows | selected WSL distro OpenSSH | same plus boundary paths/argv/EOF/cleanup; no distro fallback |
| ZCode/native | Windows | Windows OpenSSH | same plus separate native GLM setup completion |
| ZCode/WSL | Windows | selected WSL distro OpenSSH | same plus separate model ownership and transport process behavior |

Record actual OS, WSL distro, OpenSSH, Agent, application and provider versions plus negotiated MCP protocol. A profile with unavailable real runtime is unverified and blocks that profile's acceptance. Do not assume the old single protocol is the current universal choice; support only explicit versioned fixtures and proven peers.

## Low-cost experiments — none authorized to run by this plan

Each later live plan names exact host/project/logical target, identity, expected changes, a single deadline per phase (normally 600 seconds), success criteria and approved cleanup. Inspect/reuse suitable dedicated resources first. Current inventory does not authorize connecting/modifying them. No core or unrelated box, credential/SSH/firewall edits or public route for convenience.

| Experiment | Hypothesis and smallest test | Resource/change/deadline | Success and cleanup boundary |
| --- | --- | --- | --- |
| EXP-001 | Native transport/identity parser is sound: real initialize/initialized/list/read-only call plus identity/fixed hello | Reviewed client and preexisting dedicated workload; no create/policy/route; 600-second readiness/execution phase each | Actual identity/stdout/exit match; no insecure bypass. Leave workload untouched; remove only authorized temporary files/processes |
| EXP-002 | WSL selected-distro transport and cleanup preserve argv/deadline | Selected Windows client + explicit distro; synthetic peer first, no packages/distro install; 600 seconds per test batch | Mismatch/timeout/EOF yield correct status; descendants not falsely claimed cancelled; stop only owned fixture processes |
| EXP-003 | ZCode GLM basic wizard can preserve state and establish actual model selection | Version-pinned client config fixture, then native sign-in/settings handoff; no secret export; 600-second observation budget, user interaction time separate | MCP success and model readiness independent; tiny inference only under separate authorization; preserve real existing settings and native login |
| EXP-004 | Installed provider supports native Tenant atomic writes | Authenticated read-only capability/Guard first; later reviewed isolated synthetic Tenant/policy pair; 600 seconds per phase | Exact supported contract and revision all-or-none/readback; TCP only if separately scoped. No provider upgrade. Revoke/remove only test-owned state when cleanup explicitly approved |
| EXP-005 | Real route reaches intended client without invented TLS/source filtering | Reviewed dedicated box, temporary high port/listener/route after conflict check; 600 seconds | Actual client probe and correct disclosure; cleanup must explicitly cover exact test route/listener, never workload deletion |
| EXP-006 | Slice receipt/binding survives API/worker restart without replay | Assigned development LXC only, temporary real DB plus fake delayed provider; 600 seconds | Provider write count bounded, persisted unknown/deadline; stop fixture/remove temporary state after capture; no installed service restart |

EXP-001 and EXP-006 inform phase 4. EXP-002/003/004/005 inform later feature tasks unless needed earlier for feasibility; planning them is not permission to execute. No inference quota, policy mutation or public exposure has been used in phase 3.

## Gates and record format

Phase 4 passes only when TASK-005 demonstrates the selected real minimum slice and failure cases, with storage/restart evidence and explicit developer acceptance. Other profiles/basic features remain release obligations, not phase-4 completion claims. Phase 5 release candidate requires TASK-013 evidence for every selected AC; phase 6 compares approved behavior, not whole legacy parity.

Each evidence record: requirement/AC/task, exact revision/build hash, target identity (sanitized), runtime/provider versions, command or named action, start/end/deadline, actual stdout and exit (null if unknown), pass/fail/unverified, fixture vs live, cleanup outcome and remaining limits. Never paste full secret-bearing raw logs. Reboot/autostart, TCP isolation and effective confinement need distinct real evidence if claimed. Documents themselves get link/ID/coverage/DAG/whitespace checks; no product tests run until code exists.


## Actual phase 4 commands/evidence

29 tests run under both the development and a clean hash-locked Ubuntu/Python 3.12 environment; frontend TypeScript/Vite build; real Chromium with explicit infrastructure fixtures; Windows PowerShell AST and three no-network preflight cases. Commands and actual exits are in [VERTICAL_SLICE](VERTICAL_SLICE.md), with [FEATURE_MATRIX](FEATURE_MATRIX.md). No real independent application MCP session, native enrollment/UI acceptance, create/policy/route/model action or production deployment test ran. The direct Platform MCP identity/hello result is a separate observation, not completion of EXP-001 through the application. EXP-006 restart/no-replay is verified with fixture infrastructure; arbitrary remote cancellation remains unverified.

## Phase 5 iteration 1 complete — TASK-011 (SRC-051/052)

Developer acceptance of the minimum architecture/slice is explicitly recorded in SRC-051, after verified real Web/MCP evidence. TASK-011 basic Prompt Graph is implemented in 0.2.0: bounded cards, editable comments/context, undirected links, revision-checked persistence, conflict-preserved drafts, Markdown/clipboard handling and graph-only deletion. REQ-011/014/016 and AC-011/014/016 graph subset are evidenced; other selected-feature/client obligations remain incomplete.

Actual non-root project-LXC checks: 41 tests OK (11.173s), TypeScript/Vite build, final Chromium graph workflow including mobile fit and clipboard failure, original project hello/restart fixture regression; all exit0. Graph API forbids provider access in tests; browser has zero queued/executed operations. Thirteen changed source/lock files match remote SHA256. See PHASE_05_TASK_011.md for commands and precise evidence. No new box, infrastructure mutation, deployment, persistent listener, public route or push. Stop this iteration.

Next suggestion: finish TASK-003/005 minimum actual Windows-native enrollment/recognition evidence, then dependent TASK-006. Developer gate acceptance does not fabricate those missing client checks or waive per-task prerequisites. Phase 5 overall is not complete.

## Iteration 2 complete — TASK-003a, version 0.2.1 (SRC-053/054)

Scoped existing-identity Windows-native Codex registration now has trusted pre-execution payload verification, strict live connection gating, targeted config preservation, explicit replacement, byte-idempotence and backups. Actual isolated Windows/CLI/native MCP and failure-path tests passed; 41 project Linux tests and the three prior native preflight checks passed. No global/personal registration, new key, model request, mapping, box or deployment changed.

TASK-003a is complete in its stated existing-identity scope; parent TASK-003/005 remain partial. Actual GUI/TUI loading, bootstrap identity provisioning and other client/runtime profiles are unverified. Named-mutex/hash/backup recovery does not replace quiescing non-cooperating editors. See TASK_003A_NATIVE_REGISTRATION for commands, limitations and cooperative next steps. Stop this iteration; suggest TASK-003b, do not waive dependencies for TASK-006.

## Current iteration 3 — TASK-003b1, version 0.2.2 (SRC-055..058)

TASK-003b is split into TASK-003b1 (explicit Windows-native executable selection plus recorded cooperative TUI/read-only acceptance, complete) and TASK-003b2 (product bootstrap identity provisioning, incomplete). Parent TASK-003 and full TASK-005 remain partial; do not waive TASK-006 dependencies. Developer architecture gate SRC-051 remains accepted.

The native bootstrap/registration accepts an operator-selected absolute -CodexExecutable and retains exact-version/payload/trust checks. Real Windows run-06 passes with Codex absent from PATH, plus missing/relative runtime rejection and existing preservation/integrity regressions; personal config digest unchanged. Project-LXC 41 tests pass (11.586s); native AST/3 rejection checks pass. SRC-057 records the user's actual TUI connected(2 tools) and list_containers result v2-live-hello-container against the dedicated backend. This is separate from desktop GUI, instruction/model readiness, WSL/ZCode and release evidence. See [TASK-003b1 evidence](TASK_003B1_NATIVE_CLIENT.md).

The dedicated test key/18447 route were separately approved in SRC-055/056, and retained; no new route/resource/policy/model operation is part of this implementation iteration. Existing public Web demo remains fixture. No automatic key provisioning or whole-phase completion claim. Next task is TASK-003b2; stop this one slice after its code/test/docs commit, no push.

TASK-002a SRC-062 actual: tests/test_history.py adds5 cases covering tied keyset pages beyond100 records, actor/detail isolation, empty/end pages, new inserts, malformed cursor/limit/query, independent authenticated HTTP/expiry, real CLI contract and database/provider no-change. Full46 tests passed. scripts/history_browser_check.py uses owned loopback18644, real Chromium/HTTP/SQLite and fixture infrastructure;20+6 paging/unknown/refresh failure/restart/no replay/mobile passed. Original hello and Graph Chromium regressions passed. [Commands and limits](TASK_002A_RECEIPT_HISTORY.md). Source-only delivery, not a live infrastructure or deployed-UI claim.

TASK-002 completion (SRC-063/064), version0.3.0: actual SQLite FULL/claim/result-audit failure and real process death before send/after effect, concurrent claims and persisted deadlines tested. Command exit7 is failed with actual exit retained and unexpected output withheld. Explicit actor-bound target reconciliation records matching/changed/unavailable without resolving unknown or retrying. Schema3 additive observations preserve old data; older versions refuse3, use a consistent private backup for downgrade. Full58 tests19.511s, frontend build and history/original/Graph Chromium passed. Same-Tenant app-hosted real MCP reconcile matched v2-live-hello with0 hello calls; synthetic unknown receipt stayed unchanged. See [completion evidence](TASK_002_COMPLETION.md). TASK-006 create/lifecycle/reconciliation remains separate; arbitrary command execution is not added.

TASK-003 completion0.4.0 (SRC-065): selected native Codex path now includes fresh local private-ACL key preparation, exact non-root public-key-only restricted launcher bootstrap and independent verified-probe/registration diagnosis. Real fresh-v3 enrollment/CLI/MCP and Windows positive/negative/preservation tests pass; full64 Linux tests18.550s pass, including6 authorizer cases. Existing native TUI evidence SRC-057 remains separate; machine overall stays incomplete when GUI unverified. No general host shell, private/sign-in export, new route, model invocation or silent repair. Other profiles/packaging/release remain TASK-007/012/013. See [completion report](TASK_003_COMPLETION.md).


TASK-006 / SRC-069 / 0.5.0: scoped lifecycle plans, single dispatch, pinned managed environments, schema4 recovery, Web/CLI receipts and config-scope guards implemented. 82 non-root Linux tests and Chromium fixture passed; first actual five-action Web/MCP cycle passed. Final repeat correctly stops TRUST_FAILED after reused-IP mismatch; trusted reconciler reports failure and was not repaired or bypassed. Approved exact test resource cleanup read back absent; original hello preserved. See docs/verification/TASK_006_LIFECYCLE.md for separate success/failure evidence and limits. Continue TASK-007 under SRC-066; no full release claim.


TASK-007 / SRC-070..072 / 0.6.0: selected four-profile connection/registration/UI scope complete; see [profile evidence](TASK_007_PROFILES.md) for actual Windows matrix,95 Linux tests/build, separate human ZCode/WSL confirmation and limits. Python3.11+ setup runtime is explicit; existing PowerShell/native path retained. Model and release acceptance remain pending. Legacy main d8a09dd unchanged, no automatic implementation migration or push. Continuous remaining phase5 work is authorized; no implicit public/destructive/inference action.


TASK-008 / SRC-073..075 / 0.7.0: native-guided GLM wizard complete in selected scope; schema5 revisioned public intent, shared CLI/Web, native credentials only and operator report distinct from inference.106 tests/build/Chromium pass; user confirms Z.ai/GLM-5.3/access. See TASK_008_MODEL_WIZARD.md for independent native and fixture evidence. No model request, provider repair or public deployment. Continue SRC-072 network/ports and dependent packaging/release.
