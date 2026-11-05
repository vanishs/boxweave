# Full reacceptance run ledger and next-Agent handoff template

**Status:** workflow template only; not a claim that any test has run. For each invocation of [the full-project reacceptance runbook](../process/FULL_PROJECT_REACCEPTANCE.md), create a concise version-scoped report in `docs/verification/` (and sanitized test artifacts in existing approved locations). Keep prior reports immutable in meaning; do not overwrite historic PASS, UNKNOWN or incident evidence.

## 1. Run manifest

| Field | Required value |
| --- | --- |
| Run ID / start (UTC) / coordinator | Exact |
| Source revision / dirty state / intended candidate | Exact SHA, diff identification |
| Published tag + artifact SHA-256 / installed / running app-worker-Gateway | Record each **separately**; use U if unknown |
| Provider daemon/MCP and Incus versions | Observed or U |
| Outer project/backend and disposable Box UUID + createdAt | Immutable identity; no guessed ownership |
| Authorized operations / exclusions / tested policy mode | Named and scoped; R-02/R-04 opt-ins separately |
| Finder / Fixer / Verifier identities and independence | Actual Agents or **same-Agent sequential fallback** |
| Observation oracles / sanitized evidence location | Named source, command/receipt and stable reference |

### Prospective run-state and gate checklist (not proof of execution)

Record each gate as `PASS / FAIL / PARTIAL / BLOCKED / NOT_RUN`, with artifact/case evidence and an explicit reason. A past successful release does not satisfy a fresh installed gate.

| Gate | Actual evidence required | Outcome / case |
| --- | --- | --- |
| Runtime/source/help differential | Source early dispatch, installed executable help, SSH/stdio + HTTP MCP per authorized policy, Web/Provider route checks; hidden or unsupported actions listed | ... |
| Eligible tool triads | Number discovered, eligible, actually executed; missing normal/invalid/denial dimensions with why; zero catalog-only PASS | ... |
| Independent observer readiness | Separate outer Provider/guest/Incus/receipt reader, immutable target identity, verified ability to observe remote effects | ... |
| Stateful/concurrent attacks | Named two-actor interleavings and after-contention recovery or exact blocker; original dispatch/effect count | ... |
| Known-bad harness calibration | Deliberately false expected result detected; original evidence not overwritten | ... |
| Release stages | Source suite; exact candidate package; isolated installed original-repro; CI; GitHub publish; post-release manifest/official install, **each independently** | ... |
| Closure integrity | Fixed subfinding vs still unexplained original symptom; upstream attribution evidence; deferred and orphaned issues | ... |


### Upstream pinned contract and invocation manifest

For every BoxWeave reacceptance, read the [pinned Containarium evidence guide](UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) and [execution orchestration development plan](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md) and fill in:

| Field | Required observed record |
| --- | --- |
| Installed upstream daemon, CLI, MCP and REST identities | Per running process version and source/tag; if not available use U, not current `main` |
| Declared and runtime routes | Pinned proto RPC and HTTP annotations, native handler, forwarded MCP, actual HTTP response and BoxWeave adapter; unsupported distinguished from missing |
| Upstream changed paths | Compared installed version and previous upstream version, source paths, affected F/X features and exact test cases |
| Invocation channel and actor | Outer MCP vs nested installed stdio/SSH vs HTTP/OAuth vs Web vs native Provider host/CLI; access/permission gate |
| DOCMAP-style dependency | Document or claim, source paths/commit last checked, actual implementing interface, status, evidence and next recheck |
| Upstream-document confidence | Source-only, verified actual host command, isolated operation, or historical/possibly stale; pinned URL for each imported claim |

**No pass-through trust:** a native proto annotation or upstream design note can choose an adversarial test, but never independently proves the installed route. A BoxWeave MCP 404 proves a problem at that entrypoint, not automatic upstream fault ownership. The reference guide's root-level Proof Pack commands must not be invoked without the exact nested Provider host grant and a reviewed declaration.

## 2. Dynamic surface inventory (mandatory)

Produce a complete row per **MCP tool and CLI leaf in each actual installed policy/transport** from each available client context, or a detailed machine-generated manifest with a human-readable checked-in summary and immutable digest. A top-level help page or the outer ChatGPT MCP tool list is not a complete installed inventory. Record the discovered total and the unable-to-discover contexts. No hardcoded claim of a fixed MCP tool or Web route count.

| ID | Surface / action | Source / installed seen? | Transport/policy | Read/write/risk | Test case ID or explicit defer reason | State / evidence |
| --- | --- | --- | --- | --- | --- | --- |
| MCP-S-001 | Actual tool name | Source S; running I confirmed/U | SSH stdio + auth/policy | Read / P1 | SC-... | PARTIAL / S,I |
| HTTP-... | Actual HTTP MCP tool | ... | Auth/client mode | ... | ... | ... |
| CLI-... | Actual installed CLI action | ... | Host/app permissions | ... | ... | ... |
| WEB-... | Actual route + method | ... | Cookie/CSRF/auth | ... | ... | ... |
| REST-... | Actual upstream interaction | ... | Authenticated Provider | ... | ... | ... |

Surface classification must cover **all** observed entries, even when not safe to execute. Entries exposed only upstream/directly should be marked **outside BoxWeave enforcement**; distinguish them from BoxWeave-owned routes. A missing catalog is a BLOCKED *discovery context*, not proof of an empty catalog.

### Inventory comparison and omission counters

The ledger must carry distinct dimensions **per installed transport/policy**, not just one undifferentiated tool name. Compute and display:

- `discovered / newly added / removed / renamed / hidden / advertised-but-unreachable` actions, with source-vs-installed-vs-help references;
- `eligible / attempted / positive proven / invalid checked / denial checked / no-effect oracle present / partial / forbidden-by-scope / unsupported / environment-missing / not attempted`, **not** a single aggregate "tested" number;
- action rows never invoked, rows with no raw evidence, and tests that exercised only missing configuration or parser refusal;
- unknown original mutations and retained exact resource identities, without retry or invented success;
- a pointer from every action row to a stable case ID and feature ID, and from every feature back to the relevant actions.

A missing SSH/HTTP client or credentials should create a named discovery-context blocker. Do not silently use the outer ChatGPT MCP catalog as the tested installed inner catalog.


## 2A. Required **per-action MCP/CLI adversarial execution ledger**

**Every discovered MCP tool and every installed CLI leaf command must appear here individually**, per real installed SSH/HTTP MCP transport, identity/policy and host/app CLI context when different. Enumerate recursively (including hidden `boxweave http`, `boxweave task`, `boxweave selftest` dispatch and `python -m bwmini` modes). The outer ChatGPT-exposed MCP catalog is not a valid replacement for inner installed catalog discovery. **No sampling exception** for low-risk/read-only/forwarded tools.

| ID / function | Actual runtime MCP tool or installed CLI leaf, version and policy | Positive real call/output | Invalid, missing, boundary-input result | Unauthorized/stale/wrong target denial or justified N/A | Independent postcondition, exact exit/receipt and cleanup | PASS/FAIL/PARTIAL/BLOCKED/SKIPPED/UNKNOWN, evidence and next safe step |
| --- | --- | --- | --- | --- | --- | --- |
| MCP-... | Exact tool, SSH/HTTP mode, current catalog | Sanitized request, actual reply, case ID | Named case + typed error | Case + observed zero forbidden side effects | Provider UUID/createdAt, guest, receipt, distinct observer | Scope-specific status and proof |
| CLI-... | Exact executable, subcommand and flags | Actual stdout/stderr, exit and behavior | Exit/error and named case | Auth/permission/target invariant where meaningful | Separate state observation or honest unavailable reason | Status and proof |

For every **safe/authorized/runnable** entry, deliberately attempt real installed positive, malformed/boundary and relevant denial cases, rather than only `tools/list`, `--help`, a single successful call, or fixtures. For read-only tools, positive + absent/invalid target; for mutators verify success and **no forbidden side effect** on denial. A dimension not meaningful for that command is `NOT_APPLICABLE` with a specific reason, not an invented denial. Any individual unsafe/unsupported/unavailable operation receives a **separate named** BLOCKED/SKIPPED/UNKNOWN row with exact authority/dependency blocker and next command. Low priority does not justify skipping. Discovery-only cannot be PASS. An unknown dispatched mutation is preserved, never retried merely to fill the table.

### Required machine-verifiable case record for every executed positive or rejection

Use the following fields in sanitized, immutable test-owned evidence even when collected manually. These specify a **future automation contract**, not an existing deployed schema or generator:

| Field | Record |
| --- | --- |
| `case_id` / `run_id` / attempt | Unique; preserve every attempt (never overwrite output in place) |
| Original target | Logical Box plus provider UUID/createdAt/backend/project and policy identity; explicitly absent for metadata-only commands |
| Source and runtime | Exact source SHA, candidate package SHA-256, installed/running app/Gateway/Provider version, transport and actor |
| Input and timing | Sanitized exact argv/JSON-RPC/HTTP method, start/end UTC, concurrency barrier/observed first state |
| Expected and actual | Typed result/status, complete exit code, bounded original stdout/stderr/response, immutable receipt IDs |
| Independent oracle | Which **different observer** checked durable state and side-effect count, value before/after, trust limitation |
| Clean-up and uncertainty | Resource ownership, original-operation status, post-cleanup readback; never automatic replay of UNKNOWN |
| Result provenance | PASS/FAIL/PARTIAL/BLOCKED/SKIPPED/UNKNOWN plus S/T/I/H/X and original evidence path/digest |

**Hard classifier:** A row with only `tools/list`, `--help`, a guessed response, `CONFIG_OR_RUNTIME_UNAVAILABLE` or another preflight refusal cannot claim functional positive PASS. An attempted mutation with no independent readback cannot claim side-effect PASS. A test that has no immutable original output/exit is **evidence incomplete** and must be downgraded until evidence exists.


## 2B. Required **owner-facing detailed functional acceptance table**

Use the [consolidated prior-scheme functional baseline](REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md), [historical 39-feature evidence depth](EVIDENCE_LAYERS_0449.md), [call-chain/R-02/R-04 plan](../design/RUNTIME_CALL_CHAINS_0449.md), [test strategy](TEST_STRATEGY.md), feature matrix and new installed surfaces. Reconcile **all historical F01–F39** plus current/new features; historical PASS never grants current PASS. Every tool/CLI leaf must map to one or more named functions and every function back to actual implementing commands/routes.

| Feature ID + understandable purpose | Actual MCP tools/CLI leaves/Web/REST entries | Concrete positive scenario tried/result | Concrete malformed, denied, boundary, concurrency or cross-interface scenario/result | Expected vs **independently observed** effect and receipt | Status, exact installed version and evidence S/T/I/H/X | Untested paths, bug/issue references and smallest next-agent test |
| --- | --- | --- | --- | --- | --- | --- |
| Fxx / human-readable feature name | Exact names (no counts-only) | Actual case, command and exit | Actual cases or detailed non-execution reason | Provider/guest/task/Incus independent read | Scoped PASS / PARTIAL / FAIL / BLOCKED / SKIPPED / UNKNOWN | Missing subpaths and next action |
| Xyy / newly discovered feature | Real entrypoints | Test or explicit not tested | Test or explicit not tested | Real oracle or unavailable | Same | Actual blocker |

**Acceptance of the report requires every named feature row, including passed, failed, blocked, skipped and untested features.** Each row explains **what the feature does, what was intentionally challenged, what occurred and what remains unverified**, so the owner can judge the intended coverage. A tally such as “15/68 tested” or a list of only failures is insufficient. If large, commit the full human-readable table plus the tool/action ledger, link them in the final response, and still display a readable grouped feature summary covering all areas (not only totals).

### Mandatory state-machine, race and cross-interface scenario matrix

| Scenario family | Two-actor / cross-interface setup | Required outcomes | Case / original evidence / status |
| --- | --- | --- | --- |
| `selftest` concurrent admission | Two unique plans/outputs, first run observably in-flight | Competing run typed refusal or documented dedup; only one child; original cleanup; fresh run succeeds | ... |
| R-02 submit and dedup | Same key/body and same key/different body, distinct clients | Exactly one dispatched effect; stable original receipt; conflict preserves original | ... |
| R-04 claims (isolated opt-in only) | SSH/HTTP/Web writers against exact same incarnation | One allowed claim/effect; conflicting request denied, unrelated Box forward progress | ... |
| Lifecycle/create/resize/delete | Concurrent mutation, stale UUID/name/revision, stop-start race | No write against replacement/other Box, independently reconciled final state | ... |
| Web/app planner and worker | Parallel confirmations, stale revision, worker crash after intent | No duplicate Provider operation, durable UNKNOWN on uncertain original | ... |
| Installer/upgrade lock | Separate test-owned install/update actors and state read | No partial second install, lock/restore consistency, installed version attested | ... |
| Entry-point/provider parity | Same **supported** action via live SSH MCP, HTTP MCP, Web, CLI, selected REST/native path | Normalize error meaning and effect count without claiming unsupported routes equivalent | ... |

Each row must include whether scheduling was **deterministic (observed barrier)** or **timing-only**. At least one test per changed stateful boundary must show in-flight contention plus subsequent recovery. Where protected host/policy credentials are not authorized, mark exact BLOCKED instead of faking a race. This matrix extends, never substitutes for, individual action triads.


### Independent proof and capability-claim annex

Each report must explicitly state **what produced the evidence**, not only who described it.

| Proof / claim dimension | Record and permitted outcome |
| --- | --- |
| Case recording mode | Same-Agent CLI capture, independently executed test runner, upstream generated pack, or separately sealed/verified pack (with exact producer Box) |
| Upstream Proof Pack availability | Not requested, host command missing, blocked (guest-root/declaration/ownership), generated, or sealed and verified; use exact command, Box UUID and manifest digest |
| Review of declared test commands | Exact pinned child repo commit, `.github/containarium.yml` or fallback `Makefile`, script/side-effect review, permission decision and sensitive-log control |
| Proof Pack contents when generated | Actual tool exit, per-step exit, duration, log hash/truncation, declaration SHA, source commit and artifact digest; verify seal separately if present |
| Additional black-box oracle | Actual MCP/CLI/Web request, original exit/protocol outcome, independently read Provider/guest state, denied side-effect count and cleanup; proof pack alone **never grants this** |
| Claim state | Implemented, opt-in, scoped, design-only, unsupported, unverified; separate this from PASS/FAIL/PARTIAL of a tested behavior |
| Harness calibration | Deliberately false proof/claim/output must be rejected; case ID, verdict and impact of a missed negative control |
| Source/claim drift | Whether an upstream claim is outdated vs pinned code; preserve evidence instead of silently adopting its status |

Do **not** confuse upstream `containariumd proof-pack generate` (host-side root-in-child command execution) with remote `containarium` CLI or a BoxWeave MCP tool. There is currently no automatic BoxWeave Proof Pack adapter. If unavailable, use a properly scoped BoxWeave-owned non-root recorder or mark the evidence boundary PARTIAL/BLOCKED.

## 3. Scenario evidence (black-box-first)

For **every safely runnable discovered MCP tool and CLI leaf**, the required adversarial dimensions are **positive + meaningful authorization/identity denial + malformed/boundary input**, or exact per-action unavailable/unsupported reasoning; combine the entries into risk-weighted scenarios from earlier acceptance schemes. Real authenticated Web positive tests are not replaced by fixtures or a 401. Test-owned mutation requires original identity and independent readback.

| Case | Boundary / workflow | Expected response and remote effect | Actual response/exit | Independent oracle | Source / installed / policy | Result + evidence class |
| --- | --- | --- | --- | --- | --- | --- |
| SC-... | Tool/API/CLI chain | Explicit success or failure; side effect count | Actual | UUID/createdAt, guest exit, Provider read, receipt | Exact | PASS / I, FAIL / I, or U |

Pairwise/stateful scenarios additionally name the original ordering and competing actor/transport. Fault scenarios name the bounded injected failure and what the harness should detect. A **known-bad negative control** must be marked DETECTED or MISSED; MISSED creates a HARNESS_DEFECT and downgrades related acceptance.

### Evidence contradictions and issue splitting

When an original incident has more than one observation or candidate explanation, create linked but distinct claims:

- **original symptom:** exact first return/exit/time, immutable original stdout/stderr, what the later read showed, whether output paths were shared, and the status still unexplained;
- **confirmed sub-defect:** reproducible pre-fix negative test, actual root-cause boundary, fix commit and installed-original-scenario evidence;
- **remaining hypothesis:** what was not reproduced, how to distinguish tooling/output collision from product behavior, smallest safe next test.

A confirmed concurrent-admission repair cannot automatically close an unexplained initial `unreconciled` versus later `passed` output inconsistency. Similarly, label an upstream root cause only with native Provider evidence; a forwarded Compose HTTP 404 alone proves user-visible failure, not which component owns it. Retain the `SUSPECT` or upstream-check task until its own closure predicate is satisfied.


## 4. Issue packet (one per finding, including SUSPECT)

```text
Issue ID: RACC-<run>-<sequence>
Class: CONFIRMED_BUG | SUSPECT/NEEDS_REPRO | UPSTREAM | ENVIRONMENT | HARNESS_DEFECT
Priority: P0 | P1 | P2
Boundary / call chain:
Original exact source SHA / installed release+digest / running versions:
Outer/backend/project / original Box UUID+createdAt / policy mode:
Actor/transport / timestamp / sanitised exact request or command:
Expected output, exit, receipt and independent remote state:
Observed output, exit, receipt and independent remote state:
Evidence location, trace/operation/task/claim ID (redacted):
Reproduction attempts and limits; why the observation may be ambiguous:
Threat to data/integrity and no-replay disposition:
Ownership: Finder -> Fixer -> independent Verifier
Next smallest safe investigation or recheck (exact command/condition):
Grant/resource/dependency needed, if blocked:
Fix commit / candidate installed artifact / original-repro verification result:
```

**Do not** put credentials, JWT, private keys, raw secret config, private DBs or unsanitized stderr in reports. An UNKNOWN operation is never converted to FAIL or retried merely to complete the table. Recovered information is appended without erasing the original uncertain observation.

### Candidate-first release attestation (fill all applicable rows)

| Sequence | Immutable proof / result | Status |
| --- | --- | --- |
| 1. Frozen candidate | Source SHA, build artifact SHA-256, package manifest and proposed version | ... |
| 2. Prepublication installed gate | Fresh/seeded Box identity, installed candidate digest, running processes; original black-box failure rerun by Verifier | ... |
| 3. Negative and stateful gate | Invalid/denied/no-forbidden-effect, deterministic concurrent test and post-contention recovery | ... |
| 4. Source CI and packaging | Unit, knowledge, frontend/browser where relevant, exact package hash; candidate equals eventual release payload | ... |
| 5. Immutable GitHub release | Workflow result, final tag SHA, release.json source/digest, asset total and checksums | ... |
| 6. Official-release smoke | Official installer install/upgrade and independent runtime identity readback | ... |
| 7. Policy and exclusions | Stable pointer unchanged; owner-host/UNKNOWN/network exemptions and release exception, if any | ... |

If a final merge commit modifies the source identity, do not silently treat a previously verified candidate as identical: verify payload equivalence or rerun the affected installed gates. A missing prepublication installed candidate route is a declared gate BLOCKER, not a reason to publish first and call postpublication testing prepublication evidence.


### Upstream-difference and execution-mechanism gate summary

| Area | Required explicit statement |
| --- | --- |
| Upstream pinned version and delta | Exact last/current tag+SHA; paths changed and cases newly selected |
| Proof Pack | NOT_USED/BLOCKED/GENERATED/SEALED_VERIFIED and why; cannot substitute for per-action external oracle |
| Full entrypoints | Enumerate installed native/forwarded MCP, CLI, Web/REST by actual auth/policy; distinguish only-discovered |
| Runtime automation | Document-only/manual / installed runner with tested version / unavailable — never imply implementation |
| Attested release | Privately verified candidate before publish if supported; postpublish asset signatures/checksums and installed official smoke separately |
| Documentation confidence | Claim-state and source dependency references; outdated upstream docs noted as unresolved, not authoritative |
| Remaining handoff | Orphaned SUSPECT, native/upstream attribution unknown, exact next non-destructive test and permissions |


## 5. Final decision record

| Metric | Required report |
| --- | --- |
| Discovered surfaces | Totals by SSH MCP, HTTP MCP, host/app CLI, Web, REST/Provider; available and blocked contexts |
| Evaluated capabilities | **Full named owner-facing functional table** and separate **every MCP/CLI leaf** black-box ledger; positive / denial / invalid / unsupported / not-executed details per action and feature, with counts only supplemental |
| Real-world depth | T fixture vs I/L installed vs H operator host vs X independent client, scoped to exact version |
| State-machine + fault coverage | Named sequences, independent oracle and injected failures, not just test counts |
| Known-bad harness check | DETECTED / MISSED / NOT RUN and implication |
| Outcomes | CONFIRMED_BUG fixed/open; SUSPECT queued; UPSTREAM; HARNESS_DEFECT; UNKNOWN preserved |
| Fix provenance | Finder original repro -> Fixer commit -> Verifier independent installed result |
| Shipping gate | Source/CI, release asset, clean install, upgrade, runtime acceptance **separately** |
| Operator exclusions | Deferred R-02 reboot, owner-host R-04 policy, original UNKNOWN, stable selector |
| Ownership and cleanup | Exact surviving test Box identities, approved cleanup evidence |
| Next-Agent packet(s) | Executable next action, evidence and blocker, no vague TODO |

Update [FEATURE_MATRIX](FEATURE_MATRIX.md), [REMEDIATION_TEST_QUEUE](REMEDIATION_TEST_QUEUE.md), [OPEN_RISKS](OPEN_RISKS.md), [PHASE_STATUS](../process/PHASE_STATUS.md), and [FINAL_HANDOFF](../process/FINAL_HANDOFF.md) **only for new evidence or work state actually changed**. This template itself does not change acceptance state, request a release or authorize host mutation.
