# Test strategy and experiment plan

Status: planned, not executed. Authority: approved REQ/AC (SRC-026). Legacy B/F observations inform regression selection but do not override new acceptance. Test/task names below describe future obligations; there is no implementation test suite in this document-only branch yet.

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
