# Full-project black-box-first reacceptance and multi-Agent remediation

**Canonical trigger:** `Run full BoxWeave reacceptance.` (the same instruction without the trailing period is equivalent). **Status:** active owner-invoked Agent runbook, **not** evidence of any executed test. Keep the shortcut in [AGENTS.md](../../AGENTS.md); this revision does not itself invoke testing, change a host, bump a version, or publish a release.

## Mission and acceptance philosophy

Find real regressions in the **currently installed/exposed BoxWeave product**; reproduce them, repair them, and independently reverify the fixed behavior. Preserve uncertain findings for the next Agent. The objective is **defect discovery and independently observed behavior**, not a large count of familiar green tests.

Run this workflow after structural changes and on an explicit owner invocation. Start with runtime entrypoints and actual client behavior. Development-time Python unit/fixture suites remain useful as change-local regressions and **final release gates**, but do **not** count a routine rerun of those suites as broad reacceptance. `scripts/validate.py` is a fixture-only maintenance lane; seven Chromium fixtures with synthetic Provider data are not authenticated installed Web/Provider proof. A green `doctor --mcp` or MCP catalog is an inventory/health signal, not end-to-end acceptance.

Apply [AGENTS.md](../../AGENTS.md), [Agent development workflow](AGENT_TEST_WORKFLOW.md), [Agent test deployment](AGENT_TEST_DEPLOYMENT.md), [maintenance](MAINTENANCE.md), [test strategy](../verification/TEST_STRATEGY.md), [document authority](DOCUMENT_AUTHORITY.json), and current [feature matrix](../verification/FEATURE_MATRIX.md), [risk register](../verification/OPEN_RISKS.md), and [test queue](../verification/REMEDIATION_TEST_QUEUE.md). The [run ledger template](../verification/REACCEPTANCE_RUN_TEMPLATE.md) defines the evidence and handoff fields.

**Proceed end-to-end without asking for routine choices.** If an individual target is unsafe or unavailable, mark the exact gate BLOCKED / SKIPPED / UNKNOWN and continue independent work. Ask the owner only for a new target/permission, a privileged action outside the existing grant, an irreversible decision, or a genuinely indispensable external dependency. No Agent may silently expand authorization.

## Multi-Agent responsibilities and handoff contracts

Prefer **four cooperating roles**, with separate contexts and immutable handoffs. Do not confuse multiple substeps in one context with independent validation.

| Role | Owns | Must deliver | Must not do |
| --- | --- | --- | --- |
| **Coordinator / evidence steward** | Pin scope and versions, split risk domains, schedule nonconflicting work, maintain inventory, queue, evidence and release decision | Run manifest, work assignments, coverage/gaps, final decision | Declare PASS from another role's unsupported summary; run concurrent writes on the same target |
| **Finder / adversarial black-box tester** | Discover live surfaces, design user-facing scenarios, seek counterexamples, repeat small safe failure paths | Exact sanitized repro, independent expected/actual state, target identity, classification and priority | Repair source before preserving a failing baseline; treat preexisting fixtures as new discoveries |
| **Fixer / implementation Agent** | Investigate a confirmed repro, isolate BoxWeave vs upstream vs environment, write bounded fix and focused regression | Root-cause hypothesis, minimal diff, narrow regression results, exact candidate artifact | Mark own change independently accepted; erase failure evidence or weaken assertions |
| **Verifier / independent acceptance Agent** | Run the **original black-box scenario** against a pinned installed candidate without relying on Fixer's success assertion; inspect remote state independently | Independent PASS/FAIL/PARTIAL result, no-side-effect/side-effect oracle and clean-up evidence | Treat Fixer's unit tests or a health endpoint as proof; repeat an UNKNOWN write |

Optional specialists: **failure-injection/security Agent** (disposable environments only), **release/integration Agent** (manifest, install/upgrade, CI and published assets). Independent Verifier should ideally use a fresh Agent context and separate test Box/source of observations.

When true Agent delegation is available, Coordinator **must delegate discovery and verification independently of implementation**, and may parallelize disjoint feature families. When unavailable, perform the same staged roles sequentially, freeze the original Finder evidence before source changes, and explicitly label verification **same-Agent**; do not claim multi-Agent independence. Lack of delegation must not halt the authorized audit. Do not assume this trigger magically starts background Agents or new chat sessions.

Use a shared, sanitized issue ledger, **not** a shared mutable scratchpad: every issue has a stable ID, exact source/installed/runtime versions, owner, source-of-truth observations and next action. Serialize repository merges, release operations and same-target Provider mutations; assign separate disposable Boxes for parallel writers. Commit/branch ownership must be explicit and changes integrated only after base-head review; no force-push or overwrite of other Agents' work.

## Scope and safety

The owner's reacceptance authorization permits necessary **test-owned** BoxWeave MCP operations in identified project test environments, including disposable Box create/start/stop/resize/connect/delete and supported nested Box experiments. Use a **genuinely clean new Box** for fresh-install proof and a separately seeded old release for upgrade proof. Existing older Boxes may be destructively exercised **only after confirming test ownership and immutable identity**. Prefer the outer controller as independent observer and recovery route if an inner stack fails.

This is **not** permission to modify the owner/production host, change Provider/Incus/storage/network/firewall/SSH/credentials/trust policy, restart `dev`, enable root protected R-04 mode, alter stable release selection, release historical UNKNOWN claims, or repeat a potentially committed mutation. R-02 owner-host cold-reboot acceptance is **explicitly deferred**. Obey original task/Box identity, no-secret logging and release controls from the linked runbooks.

Architecture matters: current design is **MCP-first + REST-assisted**, not API-first. Agent SSH/stdio MCP, HTTP MCP, application Web, application CLI and direct upstream Provider are **distinct ingress paths**. Basic app-owned lifecycle may use REST while specialist Provider operations and Gateway forwarding use MCP. R-02 task deduplication does not imply R-04 cross-entrypoint claim coverage; neither app claim opt-in nor protected Gateway policy is automatic. Direct upstream MCP/REST/Incus writes can bypass BoxWeave coordination. **Test only guarantees BoxWeave actually promises**, and identify bypass/unsupported paths rather than declaring universal parity. See [runtime call chains](../design/RUNTIME_CALL_CHAINS_0449.md) (historical source snapshot; refresh for the current revision).

## Execution pipeline — mandatory outputs and gates

### Phase 0 — Pin a run and allocate roles (Coordinator)

Record a unique run ID, timestamp, exact main/source SHA and dirty state if relevant, official release/tag and artifact digest where available, installed binary/manifest, **running** app/worker/Gateway version, upstream daemon/MCP version, target Box UUID/createdAt/backend/project, policy mode, and independent observation access. Source, downloaded Release, installed package and running process may all differ. Record every role and actual available execution tool; inspect current authorizations before any write.

Use past PASS as **historical test-selection input**, never retroactive current PASS. Define a time/risk budget and save original evidence before attempting repair. If a required version/source cannot be installed, continue source/inventory work but keep installed acceptance U/BLOCKED.

### Phase 1 — Dynamically inventory **what users can actually call** (Finder)

Build a current **surface ledger** from the deployed system **and** current source:

1. **SSH/stdio MCP and Streamable HTTP MCP**: from each available, authorized client/policy mode, perform a real protocol initialize and `tools/list` (and relevant contract/schema checks), then compare with Gateway exports, tool policy and upstream MCP catalog. Distinguish BoxWeave-native tools from forwarded Containarium tools; policies may expose different lists. An Inspector/independent client is preferred over calling only the same in-process Python helpers.
2. **Installed CLI**: enumerate real host `boxweave --help`, safe subcommand help, application CLI actions and any shipped helpers. Compare parser contract with the actual installed executable, version and permissions.
3. **Web**: enumerate application routes/HTTP methods, authentication/CSRF/Origin requirements and actual installed endpoints; compare actual login/plan/confirm/receipt flows with Flask route definitions. A source route map is S, an HTTP 401/403 alone is negative-only, and a browser fixture is T.
4. **Provider/guest/infrastructure**: inventory REST adapters, MCP forwarding, `connect(exec)`, independently observable target state, installer/upgrade/diagnostic entrypoints and known unsupported capabilities. Record which actions reach the same Provider and which bypass BoxWeave coordination.

For **every currently discovered callable endpoint/tool/action**, record name, transport, read/write/destructive class, required authority/policy, expected effect, runtime reachability, test scenario or reason for deferment, and acceptance evidence level. Newly added/removed/renamed surfaces require explicit reconciliation with docs and source. **Gate: 100% of discovered entries classified**, not 100% of dangerous operations executed. Missing catalogs/permissions are explicit BLOCKED inventory rows, never zero features or a PASS. Use one runtime-derived catalog per supported auth/policy mode; do not hardcode tool counts.

### Phase 2 — Independent installed black-box breadth scan (Finder)

Work from **outside** the function under test using the shipped interface and a clean test-owned identity. For every **high-risk feature family** reachable within the grant, exercise at least: (a) an expected successful call with a verifiable postcondition, (b) an unauthorized/target-mismatch refusal with **no side effect**, and (c) invalid/missing input with a stable error contract. Lower-risk read-only and forwarded tools may use representative contract sampling, but all inventory rows must show why they were sampled or deferred.

Cover the **real installed** SSH MCP and HTTP MCP clients when available; installed host/app CLI actions; actual authenticated Web positive login and project/plan/confirmation/receipt; Provider-backed reads/writes; guest execution with stdout **and exit code**; and installer/upgrade/repair/doctor paths with provenance. For supported external client profiles, execute the **actual client**; otherwise mark X unverified, not PASS.

Prefer an independent postcondition oracle: exact Provider immutable UUID/createdAt/state, guest-observed result/cgroup, operation/task/claim receipts, separately read SQLite state (bounded/no secrets), and independently verified cleanup. A returned `success`, HTTP 2xx or `doctor 10/10` alone does not prove the requested effect. When the native Provider behaves incorrectly, label **UPSTREAM** separately from BoxWeave error handling.

**Gate:** produce a ranked findings list, a matrix of entrypoint families with positive/negative/invalid evidence, and a list of missed capabilities. If no defects are found, still report which exposed calls were not exercised; do not replace this phase with `scripts/validate.py`.

### Phase 3 — Stateful and cross-interface adversarial scenarios (Finder + specialist)

Prioritize risk-weighted **small sequences and pairwise interactions**, not a factorial explosion:

| Priority | Scenario / boundary | Independent oracle |
| --- | --- | --- |
| P0 | Same immutable Box: concurrent SSH MCP / HTTP MCP tracked-task / Web / app CLI lifecycle writes under **explicitly enabled isolated** R-04 policy | One allowed side effect, competing operation refusal/claim state, exact Box identity and receipts |
| P0 | Worker/Gateway interruption before dispatch, after dispatch intent, or after remote effect; timeout, duplicate request key, replay/reconnect | Original persisted receipt/claim; only verified original remote effect; no second execution |
| P0 | Same name reused, stale binding/IP, replacement UUID/createdAt, wrong tenant/actor | Rejection without touching new or unrelated Box |
| P1 | Create → get → resize CPU/memory → stop → start → `connect(exec)` → delete | Provider and guest readback after each step; confirmed absence after cleanup |
| P1 | Web authenticated plan → confirm → worker → receipt vs CLI/application actions | App DB receipt and actual Provider identity/state, not fixture status |
| P1 | MCP gateway protection, policy/allowlist, forwarded mutator bypass and REST/MCP path selection | Actual observed allowed/denied operations and **zero** unauthorized side effects |
| P1 | Clean install → client auth/MCP/diagnostics → nested child; exact older-version upgrade and retained state | Package/manifest/runtime agreement, child stdout + exit 0, data survival |
| P1 | Debug/diagnostic interfaces, token rotation, secrets projection, policy changes | No leaked secrets; expected authenticated read/error; independent trace linkage |
| P2 | Backend/provider unavailable, disk/port limits, recoverable errors, stop/start and resource cleanup | Stable error classifications, unmodified resources or correct audited partial state |

These are **test designs**, not proof that all paths are supported. Inspect current contracts before testing; skip unsupported or dangerous mutations and retain the exact reason. Do not claim that R-04 or pre-dispatch validation catches direct native Provider writes. Use matrix pairs centered on changed boundaries and prior incident classes rather than indiscriminate Cartesian combinations.

### Phase 4 — Fault injection **and test-harness calibration** (Finder + specialist)

In **disposable nested environments only**, inject bounded Provider timeout/HTTP 502/503/504, MCP transport loss, credential expiry/rotation, worker crash near dispatch intent, SQLite busy/transaction failure, exact-target trust change, and repeated same-key requests where the runner can observe cause and effect. Record the injection, expected protection, real receipt, and actual remote side-effect count. Never corrupt an original UNKNOWN or owner-host workload; never blindly replay uncertain writes.

Deliberately supply a **known-bad negative control** (for example, a fixture that falsely reports success while the independently read Box state does not change, or a deliberately invalid identity) to verify the **acceptance harness fails**. Use controlled test data, not a production fault or unreviewed code sabotage. A harness that accepts a known-bad case is itself **HARNESS_DEFECT**; related green results must be downgraded until corrected. Do not pretend the presence of a tool such as a property-based/stateful runner proves this phase was run.

### Phase 5 — Triage, repair and hand off without losing uncertainty (Coordinator → Fixer)

Classify each observation independently:

- **CONFIRMED_BUG**: exact call and independently evidenced discrepancy; severity P0/P1/P2, owned by BoxWeave when reproducibly attributable.
- **SUSPECT / NEEDS_REPRO**: credible contradictory evidence, incomplete oracle, timing-dependent signal, or unverified probable defect. **Do not mark PASS or close silently.**
- **UPSTREAM / ENVIRONMENT**: independent evidence points to native Containarium, transport, auth setup or dependencies; keep BoxWeave handling obligations separate.
- **EXPECTED_DENIAL / UNSUPPORTED**: documented, current policy, with no unauthorized effect.
- **HARNESS_DEFECT**: instrumentation, assertions or test setup made a false-green/false-red possible.
- **BLOCKED / SKIPPED / UNKNOWN**: state explicitly why; UNKNOWN dispatched mutations retain original identity and evidence with no automatic retry.

For a CONFIRMED_BUG, Fixer first preserves Finder's original **black-box** repro, then implements a narrowly scoped repair and targeted regression. Source-side Python/fixture tests are run **when relevant to the patch**, not a ritual full-suite rerun after each action. Record what changed, what did not, and any new interfaces the change affects. Fixer submits an exact candidate ref and installed artifact to Verifier; the issue cannot become independently PASS from Fixer's own tests.

For SUSPECT and untestable work, create an executable **next-Agent packet** (see template): immutable target/source identity, sanitized raw request/command, expected vs observed, exact response/exit/time, independent readback, trace/receipt links, competing explanations, next **smallest safe** reproduction attempt, authorization/resource blocker, priority and owner. Update [REMEDIATION_TEST_QUEUE](../verification/REMEDIATION_TEST_QUEUE.md) and [OPEN_RISKS](../verification/OPEN_RISKS.md); avoid vague "investigate later" notes.

### Phase 6 — Independent reacceptance, integration and release (Verifier + Coordinator)

Verifier executes the **original scenario first** against the **installed candidate**, including the negative/no-side-effect and remote-state oracle; then runs adjacent boundary regressions. A successful unit test, changed source, artifact creation, CI pass or original historic PASS does **not** promote it to installed/external PASS. If independent delegation is unavailable, state the limitation clearly.

Run **full Python/knowledge/frontend/browser fixtures once at the appropriate final integration/release gate**, and rerun them if later code changes invalidate that gate. Test exact packaging, installer provenance, clean install, retained-state upgrade and runtime identity in authorized isolated Boxes as appropriate. Confirm CI and immutable Release artifacts **separately**; publishing is not installation proof. A docs-only workflow-instruction edit requires no product version bump, suite rerun or release.

Release **one new immutable product version** only when actual applicable release gates pass and the project release process authorizes it; do **not** move the stable/default version pointer. A remaining P0 integrity/security failure blocks a claim of general acceptance or release unless the current approved release policy explicitly permits a scoped exception; document the decision and evidence. Unavailable independent checks can be SKIPPED/BLOCKED while other work proceeds, but must never be manufactured as PASS.

### Phase 7 — Report and stop (Coordinator)

Provide exact ref/commit, source/installed/published status, number of dynamically inventoried endpoints **by interface and reachability**, sampled positive/denial/invalid black-box paths, cross-interface/fault scenarios, discovered confirmed bugs (fixed/open), SUSPECT/UPSTREAM/HARNESS findings, independently reverified fixes, untouched deferred gates, original retained UNKNOWN identities, and cleaned/retained test-owned resources. Use **PASS, FAIL, PARTIAL, BLOCKED, SKIPPED, UNKNOWN** for scenario outcome; evidence **S** source, **D** historical, **T** fixture/test, **I/L** isolated installed/live, **H** operator host, **X** independent external client, and **U** unavailable. Always scope each status to exact version and interface.

If no important bugs were found, do **not** announce a blanket all-green outcome: provide the actual tested behavioral coverage, missing independent oracles, known-bad harness calibration result, and a prioritized next run. Update the authoritative matrix/risk/queue/status/handoff without rewriting older reports as present-tense evidence. Then stop and await instructions; do not automatically upgrade the owner host or begin another release cycle.

## One-sentence invocation

`Run full BoxWeave reacceptance.`

An Agent in this repository should read this file and run the above phases under the active authorization. No separate magic phrase or advance approval is necessary for routine steps already in scope. This document is an orchestration **instruction**, not an implemented multi-Agent scheduler or test runner.
