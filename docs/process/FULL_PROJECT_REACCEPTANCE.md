## 2026-11-06 controlling publication policy

The owner restored the **simple automatic GitHub main-CI release** used before
the signed verifier gate. On a successful `main` push, ordinary CI publishes
the source-bound release with GitHub's built-in write-scoped token. It does not
require acceptance evidence, a signer, a special account, protected environment,
or a dedicated Runner. The independent multi-Agent black-box reacceptance
defined below remains required before asserting **full installed acceptance**,
but is no longer a GitHub source-release publication gate. Preserve genuine
FAILED/PARTIAL/UNKNOWN statuses and original evidence. See
[the current owner decision](AGENT_MANAGED_ACCEPTANCE_RELEASE.md).
All previous "no publication workflow / direct Agent" guidance is superseded.

# Full-project black-box-first reacceptance and multi-Agent remediation

**Canonical trigger:** `Run full BoxWeave reacceptance.` (the same instruction without the trailing period is equivalent). **Status:** active owner-invoked Agent runbook, **not** evidence of any executed test. Keep the shortcut in [AGENTS.md](../../AGENTS.md); this revision does not itself invoke testing, change a host, bump a version, or publish a release.

## Source candidate v0.48.0 developer runner and current enforcement boundary

**Status: PARTIAL source implementation, no release or whole-system acceptance.** See
[the RACC-SYS-01..09 code/evidence checkpoint](../verification/RACC_SYSTEM_0480_CANDIDATE_2026-11-05.md).
Production app ZIP builds from this candidate exclude developer-only reacceptance
modules/data. The runner remains in a checked-out repository or ephemeral CI/Box
through `python scripts/reacceptance.py` or `python -m tools.reacceptance.cli`.
The legacy `python -m bwmini.reacceptance` is compatible **only in the source tree**;
do not call it from ordinary customer installation. Historical released v0.47.0
still has the older bundled behavior.

Use the one-sentence trigger unchanged. In an authorized prepared checkout, begin
with code/source and installed runtime identity; run developer-only `inventory`
then `plan` to schedule L2/L3 scenarios **before** L1 breadth is exhausted.
`breadth-safe` records only reviewed read-only probes and per-action blockers.
`upstream observe/resolve` records daemon, CLI and MCP individually and pins only
the matching upstream tagged commit (never baseline or floating main).
`report-system` yields a scenario-first report and complete named appendix,
**not** independent acceptance. For example:

```sh
python scripts/reacceptance.py inventory --root . --out .runtime/racc-inventory.json
python scripts/reacceptance.py plan --inventory .runtime/racc-inventory.json --out .runtime/racc-plan.json
python scripts/reacceptance.py breadth-safe --root . --out .runtime/racc-breadth.json
python scripts/reacceptance.py report-system --inventory .runtime/racc-inventory.json --plan .runtime/racc-plan.json --out .runtime/racc-report.md
python scripts/reacceptance.py upstream observe --out .runtime/racc-installed-versions.json
```

None of these commands may turn a catalogue, a runtime preflight or a synthetic
known-bad case into a functional PASS. The once-only ledger is an execution
safety primitive, **not** a complete race/fault scheduler; the role handoff does
**not** spawn Agents. Use genuinely distinct workers/Boxes when available; otherwise
label same-Agent/manual verification. Maintain original immutable target UUID,
createdAt, no-replay UNKNOWN, actual post-state oracle, and non-disruptive cleanup.

**Current CI/publication (2026-11-06 owner choice):** A successful ordinary
`main` push `verify` automatically runs `publish` using the exact candidate,
built-in `GITHUB_TOKEN` and checked immutable tag/assets. There is **no**
signing/reviewer/evidence `workflow_dispatch` acceptance gate and no special
Runner or second GitHub account. PR #90 was closed unmerged. This source-CI
Release cannot itself establish a full independently installed PASS; all
incomplete/UNKNOWN/blocked scenarios retain their real status. Do not
implicitly change the stable installer pointer. See
[the controlling release policy](AGENT_MANAGED_ACCEPTANCE_RELEASE.md).

**Create-after-timeout/guest-readiness incident (#92):** The
[scoped original and v0.49.3 host retest](../verification/HTTP_MCP_CREATE_0493_REAL_2026-11-06.md)
shows a prompt `submitted` HTTP receipt, same-ID duplicate request, an
initially STOPPED/guest-user-missing but eventually successful Box and SSH
without operator repair. For new high-risk HTTP MCP create tests, separately
record (a) the original task ID; (b) exact duplicate-no-replay; (c) read-only
status polling, including `create_container(status_only=true)` when the
ChatGPT connector hides the standalone status tool; (d) original native
incarnation; (e) terminal acknowledgement followed by independently verified
guest/SSH readiness; and (f) every untested cross-entry/restart/nesting
scenario. Do not mark a transient absence during `submitted` as a terminal
failure or turn `RUNNING` into independent acceptance.


## v0.49.0 source-only MVP increment (not official release acceptance)

Use `python scripts/reacceptance.py installed-readonly --root . --out
.runtime/racc-readonly --host-argv-json HOST.json --mcp-argv-json MCP.json` in
an authorized prepared checkout and available installed runtime. This probes
reviewed nonmutating tools and records real response bytes; positive results
remain PARTIAL without an independent oracle. Every other discovered entry
is marked BLOCKED, not silently excluded.

For a separate **explicitly authorized test-owned nested-Provider host** only,
`python scripts/reacceptance.py lifecycle --grant PRIVATE_GRANT.json --confirm
I_AUTHORIZE_TEST_OWNED_LIFECYCLE --mcp-argv-json MCP.json --out PRIVATE_RUN`
attempts the bounded SYS-LIFE-01 create/stop/start/delete path with native
Incus oracle and durable no-replay intent. Do not run on the owner host or
reuse a non-owned guest. This limited path does not test resize, guest exec,
cross-entry concurrency or the other 18 scenario families.

Run the additional independent verification on a separately authorized Box, using the *same source-bound CI candidate artifact* and `python scripts/reacceptance.py installed-verifier --candidate EXACT_CI_BOXWEAVE.zip --manifest EXACT_CI_RELEASE.json --source EXACT_SOURCE_COMMIT --out PRIVATE_REPORT [--lifecycle PRIVATE_ACTUAL_VERIFIER_LIFECYCLE --selftest PRIVATE_ACTUAL_VERIFIER_SELFTEST]`. This produces immutable read-only evidence, matched host/app version and checksum findings, and a PARTIAL per-scenario matrix. **It deliberately cannot mint a signed release approval.**

Read-only UNKNOWN follow-up is available from the developer checkout via `python scripts/reacceptance.py unknown-inspect --original EXACT_ORIGINAL_EVIDENCE_DIR --grant NEW_PRIVATE_TEST_OWNED_IDENTITY_GRANT --confirm I_AUTHORIZE_TEST_OWNED_LIFECYCLE --out FRESH_INSPECTION_DIR`. It checks a new scoped test-owned grant, the original immutable dispatch intent, UNKNOWN result, raw attempt digest, original native incarnation and current native Incus readback. A found child remains UNKNOWN even when UUID matches; a missing child does not prove the original never existed; identity mismatch is a hard blocker. It never replays, unlocks or deletes. This is not full R-04 operator recovery.

**Historical signed-release proposal only:** Earlier v0.48/0.49.0
candidate notes required `candidate_run_id`, `verifier_run_id`, protected
environment and signing. Those were NOT shipped as the current publish
contract. Retain them only as historical experiments; a current simple
main-CI source Release uses its same-run verified candidate bytes and never
claims full independent installed acceptance.


## Current controlling interpretation — system bug hunting, not tool testing

**Owner clarification (2026-11-05): the primary deliverable is *BoxWeave system-level bug discovery → root-cause repair → separately executed reacceptance*.** Tool enumeration, CLI help, isolated invalid-input calls and proof-engine self-tests remain **necessary L1 coverage accounting** but must not monopolize the campaign or be presented as the goal. Plan and schedule **L2 complete user workflows** and **L3 cross-entry, concurrent, fault and recovery scenarios** early, using actual BoxWeave architecture, high-risk invariants, recent structural diffs and prior incidents. Do not abandon full MCP/CLI discovery; distinguish breadth evidence from system-level assurance.

**The preferred order is:** identify invariants and dangerous state transitions → form causal hypotheses and [scenario graphs](../verification/REACCEPTANCE_SYSTEM_SCENARIOS.md) → pin version/permission and allocate isolated actors → start high-value L2/L3 probes in parallel with L1 breadth → preserve original failure → Fixer makes narrow correction → a genuinely independent Verifier repeats the original installed scenario → require exact private candidate before release → report workflow and defects **before** the per-action appendix.

Read the [dedicated low-coupling automation/developer handoff](../design/REACCEPTANCE_SYSTEM_BUG_HUNT_IMPLEMENTATION.md). Released v0.47.0 hosts ship the original reacceptance Python modules inside `bwmini`; the unpublished v0.48.0 candidate extracts them to test-only sources, pending exact package and installed verification. Do not make normal customer installation depend on the new automation, on GitHub access, or on an upstream source checkout. The upstream reference resolver must observe the **running** Containarium version and pull matching **pinned tag/commit source material on demand** into the test workspace; `bwmini/upstream_version.py`'s baseline is not the observed runtime.

## Mission and acceptance philosophy

Find real regressions in the **currently installed/exposed BoxWeave product**; reproduce them, repair them, and independently reverify the fixed behavior. Preserve uncertain findings for the next Agent. The objective is **defect discovery and independently observed behavior**, not a large count of familiar green tests. **All current MCP tools and installed CLI leaf commands must receive individual adversarial black-box test attempts or exact per-command safety/dependency blockers, followed by a detailed owner-reviewable feature list.** Catalog discovery or classification alone is never acceptance.

Run this workflow after structural changes and on an explicit owner invocation. Start with runtime entrypoints and actual client behavior. Development-time Python unit/fixture suites remain useful as change-local regressions and **final release gates**, but do **not** count a routine rerun of those suites as broad reacceptance. `scripts/validate.py` is a fixture-only maintenance lane; seven Chromium fixtures with synthetic Provider data are not authenticated installed Web/Provider proof. A green `doctor --mcp` or MCP catalog is an inventory/health signal, not end-to-end acceptance.

Apply the [pinned Containarium upstream evidence guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) and [upstream-aware execution design](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md) to each new run. **The guide's documents are reference evidence, not runtime proof or permission to call arbitrary upstream host commands.** The v0.47.0 source additionally ships a read-only-default runtime inventory/evidence CLI, an explicit test-owned selftest concurrency probe and a guarded optional host Proof Pack adapter ([implementation guide](../verification/REACCEPTANCE_ENGINE_0470_2026-11-05.md)). **The docs alone install nothing; read the actual running release before using these tools.** A general background or fully automatic multi-Agent scheduler and all-surface destructive dispatcher are **not shipped**.

Apply [AGENTS.md](../../AGENTS.md), [Agent development workflow](AGENT_TEST_WORKFLOW.md), [Agent test deployment](AGENT_TEST_DEPLOYMENT.md), [maintenance](MAINTENANCE.md), [test strategy](../verification/TEST_STRATEGY.md), [document authority](DOCUMENT_AUTHORITY.json), and current [feature matrix](../verification/FEATURE_MATRIX.md), [risk register](../verification/OPEN_RISKS.md), and [test queue](../verification/REMEDIATION_TEST_QUEUE.md). The [run ledger template](../verification/REACCEPTANCE_RUN_TEMPLATE.md) defines the evidence and handoff fields. Reconcile the new run against the [previous 39-feature evidence matrix and newer acceptance schemes consolidated into the functional coverage baseline](../verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md); do not inherit old PASS labels.

**Proceed end-to-end without asking for routine choices.** If an individual target is unsafe or unavailable, mark the exact gate BLOCKED / SKIPPED / UNKNOWN and continue independent work. Ask the owner only for a new target/permission, a privileged action outside the existing grant, an irreversible decision, or a genuinely indispensable external dependency. No Agent may silently expand authorization.

## 2026-11-05 operational refinement: acceptance must be executable, not just documented

The following are **mandatory Agent procedures now**, not claims that an automated harness, scheduler or CI acceptance gate has been implemented. Execute manually with available tools when necessary and mark unimplemented automation explicitly. The canonical command remains unchanged. The detailed tool-development backlog is at the end of this document.

- **Catalog reconciliation before testing:** compare current source registration/early dispatch, *installed executable and effective help*, and actual SSH/stdio/HTTP MCP `tools/list` plus Web route/Provider reachability. Report differences as candidate defects even if a callable command is absent from help; inspecting `argparse.add_parser` alone cannot discover special dispatch (`#80`). Missing authentication is an inventory blocker, not an empty catalog.
- **Reproducible behavior over green counts:** prioritize causal counterexamples: original failure, actual remote state, race interleavings, typed errors and later recovery. Healthy `doctor`, 100 repeated serial PASS, or all-green fixture CI cannot establish simultaneous-actor safety (`#83`), functional Compose support (`#79`) or release parity.
- **No fabricated evidence:** each executable result must have an immutable test-run/case ID and independently observable exit/response and postcondition. A synthetic `CONFIG_OR_RUNTIME_UNAVAILABLE` response exercises preflight only: it cannot be a positive functional acceptance. Track missing evidence separately from unsafe permissions and unsupported capabilities.
- **Separate evidence from source publication:** ordinary main CI may publish source-bound assets after its normal `verify`; no signed independent verifier gate is configured. To claim **full installed acceptance**, require the applicable isolated black-box tests and independent Verifier evidence. CI and release parity never substitute for installed behavior. Keep original UNKNOWN and stable pin unchanged.
- **Do not collapse distinct findings:** closing a confirmed root cause does not close an earlier ambiguous symptom unless the evidence establishes causation or independently verifies the symptom is resolved. A `SUSPECT` can remain open with a precise next reproduction after an associated confirmed defect has shipped.

These refinements apply prospectively to new runs. Historic reports keep their original observations and version-scoped classifications.


## Mandatory MCP/CLI full-coverage and owner-visible acceptance contract

**Two independent coverage axes are mandatory, not interchangeable:**

1. **Action-by-action black-box coverage:** enumerate **every** discovered callable tool in each real installed MCP catalog (SSH/stdio, Streamable HTTP, policy/identity variants; native BoxWeave and forwarded Provider tools) and **every** installed CLI leaf/action (host `boxweave`, special dispatch including `http`, `task`, `selftest`, app `python -m bwmini`, shipped helpers). Recursively inspect `--help`, binary/parser/source registration and actual dispatch; a top-level help page or this chat's outer MCP catalog is not proof of a nested instance's exposed commands. Each action must have a named test row and an actual call when safe and authorized; **representative sampling of low-risk or forwarded tools is no longer sufficient**.
2. **Function-by-function behavior and integration:** use the [functional baseline](../verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md) to map all discovered MCP/CLI leaves to user-facing features, requirements, actual Provider routes and prior high-value acceptance scenarios. Refresh old [five-layer evidence depth](../verification/EVIDENCE_LAYERS_0449.md), [call chains](../design/RUNTIME_CALL_CHAINS_0449.md), [test strategy](../verification/TEST_STRATEGY.md) and R-02/R-04/UNKNOWN, installer/repair, native/WSL client, Web and network interactions to the current version. Historic evidence selects probes but does not certify the new installed implementation.

For each **safely executable MCP tool or CLI leaf**, deliberately seek counterexamples through (a) a real positive/normal result, (b) malformed, missing, extreme or otherwise invalid input, and (c) an unauthorized, wrong-target/stale-revision or equivalent guaranteed-denial case **where semantically meaningful and possible within existing authority**. For read-only actions, use both an actual positive observation and absent/bad target; for mutations verify the intended remote change or zero forbidden side effects using a separate Provider/guest/receipt/Incus oracle. Include CLI argument parsing, exit codes, stdout/stderr boundaries, command discovery, configuration permission and transactional behavior. Mark a dimension `NOT_APPLICABLE` **with its concrete rationale** rather than inventing an unsafe denial or treating a normal read as a security test.

An action unavailable on this environment, unsafe without additional permission, unsupported by a known platform, or ambiguous after a dispatched write gets its **own** `BLOCKED` / `SKIPPED` / `PARTIAL` / `UNKNOWN` row, exact reason, proof already gathered, original identity and smallest safe next-Agent action. Ordinary low priority is **not** a reason to skip testing. Never auto-approve host reboot, credential rotation, external public exposure, direct destructive upstream tests, R-04 enablement or historical UNKNOWN reconciliation merely to fill a cell. Continue other independent cases rather than interrupting the run. **A discovery-only row can never be PASS.** Mark a claimed per-action PASS only for the tested scope with meaningful positive, negative/error and independent-oracle evidence; missing coverage remains explicit PARTIAL.

**Mandatory gates:** reconcile 100% of discovered MCP tools and CLI leaves into the per-action ledger, attempt 100% of *safe/authorized/runnable* entries, assign explicit statuses and next steps to 100% including blocked entries, map all entries to a full named feature coverage table, and execute risk-based cross-entrypoint combinations from previous schemes. Counts are only audit checks. If any mandatory entry remains unattempted, lack of inner installed catalog, or unsupported positive path, the overall result remains scoped **PARTIAL/BLOCKED**—never “everything passed”.

**Owner-facing deliverable is not a scorecard:** publish the complete, readable **named feature acceptance table** including what the feature does, which exact MCP/CLI/Web/REST routes implement it, concrete successful/invalid/denied tests attempted, expected versus observed side effects, scope/version/evidence grade, current status, uncovered scenarios and precise defect links. Provide a separate **one-row-per-tool/CLI-leaf action ledger** with exact invocation/test IDs and evidence. In the final conversation show a readable function-grouped list/table (including unverified/blocked features), and link the full feature and action ledgers; do not end with only “15/68 passed” or totals and a generic disclaimer.

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

Use past PASS as **historical test-selection input**, never retroactive current PASS. At planning time produce a feature-by-feature proposed scope from the [consolidated historical baseline](../verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md), and reconcile it with current tool/command catalogs before testing. The expected scope includes old F01–F39 plus current-version additions; these IDs are a seed, not a cap. Define a time/risk budget and save original evidence before attempting repair. If a required version/source cannot be installed, continue source/inventory work but keep installed acceptance U/BLOCKED.

### Phase 0A — Pin upstream contracts and test invocation capabilities

Before invoking any mutating test, record **upstream installed daemon version and source SHA** as independently as possible. Compare to the exact pinned upstream tag, not floating `main`. Follow the [upstream reference guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) to map `proto/containarium/v1/*` RPC/HTTP annotations, `internal/mcp/*` forwarded tools, `internal/cmd/*` host/client distinction and `internal/server/*` runtime handlers. Reconcile declared route versus actual installed response and relevant BoxWeave MCP-first/REST-assisted adapter; a source route alone never proves runtime support.

Discover **available invocation channels** and whether they are authorized: (a) outer BoxWeave MCP reads and explicitly owned Box operations, (b) actual installed host/app CLI in a disposable Box, (c) installed local/SSH stdio MCP, (d) properly authenticated HTTP MCP/Web where test-scoped credentials exist, (e) native upstream RPC/REST/CLI **read only by default**, and (f) optional upstream Proof Pack host generator **only after review**. Use the [execution development plan's adapter table](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md#3-concrete-adapters-and-invocations) to avoid confusing outer `connect(exec)` guest SSH with a root shell on the owner Provider host.

Diff the exact upstream `proto`/MCP/server/CLI/Incus/SSH and release paths against the previously observed upstream version, and use DOCMAP-style `source → interface → feature → test case` dependency mapping to **prioritize** impacted tests. Continue exhaustive per-leaf attempts required by this runbook; diff selection never substitutes for breadth. Mark missing version/route/credential or actual install as BLOCKED/PARTIAL, not a blank catalog or PASS.


For source revisions at or after v0.47.0, have the Coordinator use the **already packaged, non-root-by-default execution engine** from an authorized test-owned Box: `python -m bwmini.reacceptance bootstrap-readonly --root <exact-source-or-installed-tree> --out <new-private-run-path>`, optionally with **reviewed** installed host/stdio MCP argv JSON vectors and a pinned upstream tree. This produces real discovery and bounded safe reads, leaves the run open for Finder's other cases and never certifies a functional PASS from a catalogue. Agents must then import executed cases with `record`, independently inspect write effects, `seal` and `verify` after all eligible cases, and use `report` for the full 49-feature/action ledger. If an installed older version lacks these modules, follow the manual documented process; do not claim the v0.47.0 runner exists there.

### Phase 0B — Invariant-driven test selection and independent scenario allocation (mandatory, before breadth)

The Coordinator must build an **initial system-level hypothesis board** *before* mechanically testing every tool:

1. Read the current BoxWeave call chains, mutable-state/journal boundaries, MCP↔REST↔CLI/Web adapters, current architecture/policy gates, source delta, historically unresolved bug reports and live Provider capabilities. Map **at least one expected invariant** to every proposed workflow and **at least one independent oracle** to each mutation.
2. Select high-risk scenarios from [SYS-LIFE / SYS-JOB / SYS-COORD / SYS-UNKNOWN / SYS-SSH / SYS-WEB / SYS-UPGRADE / SYS-COMPOSE and other seeded scenario families](../verification/REACCEPTANCE_SYSTEM_SCENARIOS.md). Write test-owned target identities, actor boundaries, deterministic schedule/barrier, failure window, expected allowed/forbidden effect, cleanup and scope of permission. Generate new hypotheses from **current code changes**, not only known issues.
3. Allocate independent test Boxes/actors to L2 and L3 **immediately**, alongside the general L1 catalog/breadth lane. When a high-confidence counterexample appears, preserve it and begin Fixer investigation without waiting to finish the entire catalog. The Coordinator continues the independent remaining test inventory or explicitly hands it to another Agent.
4. Mark the plan as `PLANNED`, not PASS. Do not claim a separate Finder/Verifier Agent exists merely because a role was named in a Markdown file. No separate actor available → `SAME_AGENT` downgrade or exact BLOCKED independent verification.

**Risk allocation guidance:** prioritize substantive work at roughly **L1 20% / L2 40% / L3 40%** (adjustable for current risk). Preserve **100% inventory accountability** and actual authorized L1 attempts, but measure success primarily by new reproducible system counterexamples and completed independently verified workflows—not tool-call count.



### Phase 1 — Dynamically inventory **what users can actually call** (Finder)

Build a current **surface ledger** from the deployed system **and** current source:

1. **SSH/stdio MCP and Streamable HTTP MCP**: from each available, authorized client/policy mode, perform a real protocol initialize and `tools/list` (and relevant contract/schema checks), then compare with Gateway exports, tool policy and upstream MCP catalog. Distinguish BoxWeave-native tools from forwarded Containarium tools; policies may expose different lists. An Inspector/independent client is preferred over calling only the same in-process Python helpers.
2. **Installed CLI**: enumerate real host `boxweave --help`, **every recursively reachable leaf command and option**, special/pre-parser dispatch not shown by global help (notably HTTP MCP, task coordination and selftest), application CLI actions and shipped helpers. Cross-check actual dispatch/parser/source registration to discover unadvertised commands and verify the installed executable, version, permissions and real command paths. Each leaf becomes an independently accountable test target.
3. **Web**: enumerate application routes/HTTP methods, authentication/CSRF/Origin requirements and actual installed endpoints; compare actual login/plan/confirm/receipt flows with Flask route definitions. A source route map is S, an HTTP 401/403 alone is negative-only, and a browser fixture is T.
4. **Provider/guest/infrastructure**: inventory REST adapters, MCP forwarding, `connect(exec)`, independently observable target state, installer/upgrade/diagnostic entrypoints and known unsupported capabilities. Record which actions reach the same Provider and which bypass BoxWeave coordination.

For **every currently discovered callable endpoint/tool/action**, record name, transport, read/write/destructive class, required authority/policy, expected effect, runtime reachability, function ID(s), test IDs, and acceptance evidence level. Distinguish the outer Agent's tool catalog from actual installed inner SSH and HTTP MCP catalogs and the direct upstream catalog. Newly added/removed/renamed surfaces require explicit source/runtime reconciliation. **Gate: all current MCP and CLI leaf entries discovered and classified, all discovery-blocked contexts separately declared, every discovered action assigned a test plan—not yet a PASS.** Missing catalogs/permissions are explicit BLOCKED inventory rows, never zero features. Use one runtime-derived catalog per supported auth/policy mode; do not hardcode tool counts.

### Phase 1A — detect undisclosed interfaces and coverage drift (mandatory)

Create one comparison table keyed by `transport + effective identity/policy + installed release digest + exact callable action`. For each host CLI family, recurse both human help and runtime/source dispatcher branches, including early interceptors; attempt safe `--help` or deliberately invalid invocation on each discovered leaf, then verify a real positive wherever authorized. A hidden callable command is a **discoverability failure**, not simply an extra row. For each native/forwarded MCP method compare actual schema, route support and error translation, not only matching names. Compare Web's actual registered methods and authenticated route behavior with source decorators; HTTP 401 alone proves denial but not route success.

The inventory gate must flag: (1) source-only, (2) installed-only, (3) installed-but-undocumented, (4) advertised-but-not-callable, (5) catalog differences by transport/policy, (6) Provider capability/REST-route mismatch, and (7) surface unverified due to authority or credentials. Demand an owner-facing reason and a linked scenario for each difference. Do not hardcode the historical 68/26/36 counts or auto-convert a tool catalog to PASS.


**L1 lane design:** Execute a *bounded*, parallelizable inventory and per-entry black-box queue. This is the **breadth safety net**, not the entire reacceptance campaign. Every current eligible MCP/CLI leaf must still receive real positive, malformed/invalid and meaningful denial dimensions when safe/applicable, or have an exact owned-target/authorization/unsupported blocker. A `tools/list`/`--help` outcome proves discoverability only. Do not postpone L2/L3 scenario execution until the L1 queue drains.

### Phase 2 — Independent installed black-box breadth scan (Finder)

Work from **outside** the function under test using the **installed shipped interface**, a clean test-owned identity and a reproducible immutable target. **Execute a separate adversarial black-box case for every current, safely callable MCP tool and installed CLI leaf**, regardless of whether it is low-risk, read-only, forwarded or previously tested by fixtures. For each, seek (a) a real successful call with independently checked postcondition, (b) an unauthorized/target-mismatch or meaningful denial with **no forbidden side effect**, and (c) malformed/missing/boundary input with a stable typed error and CLI exit contract. Mark a dimension NOT_APPLICABLE only when specifically justified. A representative call to one forwarded tool never covers sibling tools. An unexecuted action is BLOCKED/SKIPPED/PARTIAL with concrete reason, not PASS.

Cover the **real installed** SSH MCP and HTTP MCP clients when available; installed host/app CLI actions; actual authenticated Web positive login and project/plan/confirmation/receipt; Provider-backed reads/writes; guest execution with stdout **and exit code**; and installer/upgrade/repair/doctor paths with provenance. For supported external client profiles, execute the **actual client**; otherwise mark X unverified, not PASS.

Prefer an independent postcondition oracle: exact Provider immutable UUID/createdAt/state, guest-observed result/cgroup, operation/task/claim receipts, separately read SQLite state (bounded/no secrets), and independently verified cleanup. A returned `success`, HTTP 2xx or `doctor 10/10` alone does not prove the requested effect. When the native Provider behaves incorrectly, label **UPSTREAM** separately from BoxWeave error handling.

**Gate:** deliver the one-row-per-MCP/CLI-leaf black-box ledger, **with concrete per-action** positive/denial/invalid outcomes, native/forwarded/transport provenance, independent oracle, test case and limits; **also deliver the owner-facing detailed function list**. Confirm every safe/authorized/runnable tool and CLI leaf was actually invoked, every missing case or catalog is explicitly BLOCKED/PARTIAL, and no history/fixture green was promoted. Produce ranked failures and unresolved questions even when no defects are confirmed; do not replace this phase with `scripts/validate.py`.

**L2/L3 is the primary hunt:** Follow the [system scenario matrix](../verification/REACCEPTANCE_SYSTEM_SCENARIOS.md), actual branch/Provider capabilities and change-impact graph. Test an entire **operation sequence** with independently checked intermediate states, not five unrelated green calls; then inject a **specific failure boundary or second actor**. The original failure and all Provider effects/receipts must remain traceable through Fixer and separately running Verifier.

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

These are **test designs**, not proof that all paths are supported. Preserve and adapt earlier [five-level F01–F39 acceptance proposals](../verification/EVIDENCE_LAYERS_0449.md), [R-02/R-04 protected and Web-first/SSH-first/HTTP-first call-chain matrix](../design/RUNTIME_CALL_CHAINS_0449.md), [failure-path/client-profile strategy](../verification/TEST_STRATEGY.md), [R-04 UNKNOWN manual recovery](../verification/R04_UNKNOWN_RECOVERY_0450_2026-11-05.md), and [current v0.46.1 incidents](../verification/REACCEPTANCE_0461_2026-11-05_BLACKBOX_RUN2.md) into the fresh installed scenarios. The [consolidated functional baseline](../verification/REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md) lists all historical and current themes, including CLI help hidden entries, debug REST parity and selftest. Inspect current contracts before testing; block unsupported or dangerous mutations by exact entry and retain the reason. Do not claim that R-04 or pre-dispatch validation catches direct native Provider writes. Use matrix pairs centered on changed boundaries and prior incident classes rather than indiscriminate Cartesian combinations.

### Phase 3A — deterministic concurrent-actor and state-transition attacks

Every *mutable state machine* with a read/check-then-write decision, durable intent, lock, lease, deduplication or recovery path gets a concurrency case, even if serial smoke succeeds. Prioritize `selftest plan/run`, R-02 submit/dedup, R-04 claims, lifecycle create/delete/resize, installed Web plan/confirm/worker, model or Tenant revision updates and installer/update locks. Create **two distinct actors/requests with different identities and unique immutable output paths**. Prove that the first original operation has entered a specific state (journal row / Provider child UUID / dispatch-intent receipt) **before** issuing a second competing request; do not infer overlap merely from timestamps.

Test the bounded outcomes: exactly one allowed dispatch/effect, typed rejection or documented idempotent receipt for the competing caller, independent proof of **zero forbidden second effects**, original completion/cleanup, and a **new independent operation succeeding after contention ends**. Add one early-vs-late scheduling interleaving and crash/timeout at a known boundary where test-owned isolation and grant permit. Use both source-level cross-process/barrier tests and a real installed two-client probe for claims about process/Agent safety. Five repetitions of the same serial path alone do not establish concurrency behavior. Where precise scheduling cannot be controlled, record a probabilistic observation, not a deterministic proof.

Each invocation gets its own original stdout/stderr, exit, start/end timestamps, request key, journal/task ID and exact Box UUID/createdAt; never share a mutable result-file path between actors. On timeout or `UNKNOWN`, **stop dependent cases**, inspect the original Box/receipt through the outer read-only observer, retain the scene and do not redispatch. Never auto-resolve an original historical `UNKNOWN` to unblock a test.


### Phase 4 — Fault injection **and test-harness calibration** (Finder + specialist)

In **disposable nested environments only**, inject bounded Provider timeout/HTTP 502/503/504, MCP transport loss, credential expiry/rotation, worker crash near dispatch intent, SQLite busy/transaction failure, exact-target trust change, and repeated same-key requests where the runner can observe cause and effect. Record the injection, expected protection, real receipt, and actual remote side-effect count. Never corrupt an original UNKNOWN or owner-host workload; never blindly replay uncertain writes.

Deliberately supply a **known-bad negative control** (for example, a fixture that falsely reports success while the independently read Box state does not change, or a deliberately invalid identity) to verify the **acceptance harness fails**. Use controlled test data, not a production fault or unreviewed code sabotage. A harness that accepts a known-bad case is itself **HARNESS_DEFECT**; related green results must be downgraded until corrected. Do not pretend the presence of a tool such as a property-based/stateful runner proves this phase was run.

### Phase 4A — evidence integrity and oracle calibration

Before accepting a scenario, require a test-owned immutable record of: source/installed/runtime versions, actor/policy/transport, sanitized request, exact original exit and stdout/stderr or protocol response, wall-clock timestamps, independent remote-effect count/readback, error mapping, cleanup and original resource identity. Compare response and receipt against a **separate observation path** (guest files/cgroups, Provider immutable state, Incus list, database journal under approved read permissions). A read through the same potentially faulty wrapper is supporting, not independent, evidence.

The harness must visibly **FAIL a deliberately known-bad assertion** (wrong child identity, falsely expected single side effect, or mismatched guest bytes). Report `DETECTED`/`MISSED`, the test ID, and which other claims lose validity when missed. Calibrate at least one positive and one negative oracle on the current install. Never overwrite the first contradictory output with a later result: preserve both as separate observations, compare artifact inode/path, producer PID and timestamps, and label `SUSPECT` until explained.


### Phase 4B — Optional upstream Proof Pack, independent evidence and claims audit

For reviewable source/contract tests, use the [Containarium Proof Pack reference](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md#4-proof-pack-integration-is-optional-host-side-and-safety-gated) **only when it is actually available in the installed test host and explicitly safe to invoke**. On installed upstream v0.128.0, `containariumd proof-pack generate` belongs to the **host/server binary**, not the ordinary `containarium` CLI and not BoxWeave's currently exposed 68 MCP methods. Its `generate` runs repository-declared commands **as root in the target owned child Box**. The Coordinator must verify a disposable nested Provider host, exact owned child incarnation, trusted checkout SHA, reviewed `.github/containarium.yml` commands or `Makefile` fallback, log-safety and permissions before issuing any command. Unreviewed fallback or absent host access means skip this optional adapter, not bypass safety.

A generated pack has command exit/log/duration/hash evidence; a **sealed/verified** pack needs a separate authorized verification path and relevant audit store. Neither proves per-tool black-box positive, invalid, denial, side-effect count or cleanup. Continue to capture BoxWeave-owned **per-attempt immutable** stdout/stderr/exit, Provider/guest/Incus observations, cleanup and known-bad harness controls even if Proof Pack is PASS. The pack can be one evidence item in the [run ledger](../verification/REACCEPTANCE_RUN_TEMPLATE.md), never a replacement for the action ledger or independent Verifier.

Adopt upstream `CLAIMS.md`'s **implemented / opt-in / scoped / design-only / not-verified** claim review, but verify each claim against current installed behavior. Docs can be stale: upstream `DOCMAP`'s mention of `/writer-docs audit` is not confirmed as an installed executable, and upstream Proof Pack text contains older statements that may differ from current source. The [upstream reference guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md) records examples; new runs must pin their own sources.


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

### Phase 5A — split incident root cause from unresolved symptom

Record the exact original observation, the smallest reproducible sub-finding, and the specific behavior actually repaired as **separate acceptance claims**. For example, a confirmed `selftest` concurrent-start journal race can be closed after installed contention verification, while an earlier `unreconciled`-then-`passed` observation remains a separate `SUSPECT` if no direct causal link or same-symptom reproduction exists. Link but never silently merge independent questions. For upstream-labeled incidents such as Compose route `#79`, record *which native Provider route/version was actually probed*; a Gateway HTTP 404 alone proves the user-visible failure, **not** upstream root cause. Record a concrete upstream-version-change retest trigger without requesting privileged changes now.

Classify suspected defects as BoxWeave, upstream, environment or test-harness only to the supported attribution depth. Preserve original evidence and issue ID; each issue requires a closure predicate and named regression that reproduces the failed condition, not merely an adjacent happy path.


### Phase 6 — Independent reacceptance, integration and release (Verifier + Coordinator)

Verifier executes the **original scenario first** against the **installed candidate**, including the negative/no-side-effect and remote-state oracle; then runs adjacent boundary regressions. A successful unit test, changed source, artifact creation, CI pass or original historic PASS does **not** promote it to installed/external PASS. If independent delegation is unavailable, state the limitation clearly.

Run **full Python/knowledge/frontend/browser fixtures once at the appropriate final integration/release gate**, and rerun them if later code changes invalidate that gate. Test exact packaging, installer provenance, clean install, retained-state upgrade and runtime identity in authorized isolated Boxes as appropriate. Confirm CI and immutable Release artifacts **separately**; publishing is not installation proof. A docs-only workflow-instruction edit requires no product version bump, suite rerun or release.

Release **one new immutable product version** only when actual applicable release gates pass and the project release process authorizes it; do **not** move the stable/default version pointer. A remaining P0 integrity/security failure blocks a claim of general acceptance or release unless the current approved release policy explicitly permits a scoped exception; document the decision and evidence. Unavailable independent checks can be SKIPPED/BLOCKED while other work proceeds, but must never be manufactured as PASS.

### Phase 6A — exact candidate-first publication gate

For an independently accepted product change, the preferred order is:
(1) freeze the source and candidate digest; (2) run focused source/contract
tests; (3) build an installable candidate; (4) install it on an explicitly
authorized separate Box and verify its runtime identity; (5) have a different
Verifier execute the original failing black-box scenario and neighboring
safety cases; (6) reconcile the full named scenario ledger; (7) merge to
`main` and independently verify the simple source-CI Release and asset
parity; (8) check the official upgrade when authorized. **These are gates
for an installed independent acceptance claim**, not an extra GitHub
workflow approval before ordinary source publication.

If tooling cannot deliver the exact candidate privately/securely to the test Box, do **not** push unpublished unreviewed source to public GitHub as a workaround: record `BLOCKED` and the missing candidate-build/install capability. If post-merge source SHA necessarily changes, verify the final artifact's source and payload equivalence before publication or repeat affected gates. Any missing installed evidence must remain explicitly **scope-limited BLOCKED/PARTIAL**, never a blanket PASS. A known P0 data-integrity/security risk requires an owner decision before deploying or declaring an installed release accepted. Ordinary source-CI publication is not that deployment or installed acceptance decision. The operator's production host, R-02 reboot, R-04 activation and stable pin remain excluded.


### Phase 6B — Upstream release rehearsal patterns are inspiration, not currently wired BoxWeave CI

Upstream [`RELEASE-PROCESS.md`](https://github.com/FootprintAI/Containarium/blob/v0.128.0/docs/RELEASE-PROCESS.md) documents build-only workflow dispatch before publishing. **Do not invoke its repository's workflows for a BoxWeave release** or assume BoxWeave already has the same publication-free capability. Use the approved BoxWeave release pipeline and candidate-first gate above. The [execution development plan](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md) defines the future publish-free adapter and acceptance conditions; until built, mark missing private candidate gate explicitly and avoid inventing a PASS. Upstream `distros-py-release.yml` workflow-dispatch is a **publishing path**, not a rehearsal.


**Reporting order:** (1) system-invariant and **full workflow** matrix, with original counterexamples, risk, fix commit and independent verifier; (2) unresolved SUSPECT / UNKNOWN / upstream ownership; (3) candidate release gate and whether tests ran on the exact installed candidate; (4) full named feature coverage; **then** (5) the exhaustive MCP/CLI leaf/action appendix. An action returned 200/PASS is **not** a complete workflow PASS; a 235-row inventory is not 235 tests. State how many genuinely novel system-level bug hypotheses were executed and what remains untested.

### Phase 7 — Report and stop (Coordinator)

Provide exact ref/commit, source/installed/published status, number of dynamically inventoried endpoints **by interface and reachability**, **per-entry** positive/denial/invalid black-box paths for all safe MCP tools and CLI leaves, cross-interface/fault scenarios, discovered confirmed bugs (fixed/open), SUSPECT/UPSTREAM/HARNESS findings, independently reverified fixes, untouched deferred gates, original retained UNKNOWN identities, and cleaned/retained test-owned resources. The actual **full named functional acceptance table** must be present, covering each current user capability (including the historical F01–F39 seed), its actual MCP and CLI commands, what was tested and what was not, observed results, and linked raw case IDs. A machine manifest without a human-readable feature table is insufficient. Use **PASS, FAIL, PARTIAL, BLOCKED, SKIPPED, UNKNOWN** for scenario outcome; evidence **S** source, **D** historical, **T** fixture/test, **I/L** isolated installed/live, **H** operator host, **X** independent external client, and **U** unavailable. Always scope each status to exact version and interface.

If no important bugs were found, do **not** announce a blanket all-green outcome: provide **the full detailed named feature list and per-action evidence ledger**, actual behavioral coverage and per-feature missing cases, missing independent oracles, known-bad harness calibration result, and a prioritized next run. Counts alone or a grouped green summary fail the reporting gate. Update the authoritative matrix/risk/queue/status/handoff without rewriting older reports as present-tense evidence. Then stop and await instructions; do not automatically upgrade the owner host or begin another release cycle.

### New system-level and low-coupling automation handoff (supersedes an interface-count-only roadmap)

The [implementation work packages `RACC-SYS-01..09`](../design/REACCEPTANCE_SYSTEM_BUG_HUNT_IMPLEMENTATION.md) now own the unfinished engineering: remove test-only runner/data from normal shipping **after characterization**, on-demand exact installed-version upstream source resolution, invariant/scenario planner, all eligible MCP+CLI black-box dispatcher, deterministic stateful/failed-operation executor, real multi-Agent Finder/Fixer/Verifier coordinator and **enforced pre-publication installed-candidate release gate**, with scenario-first report/evidence persistence. Earlier `RACC-AUTO-01..07` are historical partial components; **none** of these full new capabilities is automatically implemented by editing instructions. Do not ship generic root host execution or update the stable installer selector.



## Automation implementation backlog — **not yet implemented by this document**

These tasks describe future tools. They must **not** be reported as installed, automatically run or already enforcing CI until code, deployment and acceptance evidence exist. The same owner trigger uses available Agent/manual procedures in the interim.

| Work item | Intended deliverable | Minimum measurable completion gate |
| --- | --- | --- |
| `RACC-AUTO-01` P0 | Runtime-driven MCP/CLI/Web/REST discovery and differential inventory | Fresh installed inventories per transport/policy; detect a deliberately hidden callable command and newly removed/renamed tool, match every entry to feature/test ID |
| `RACC-AUTO-02` P0 | Safe test dispatcher + machine evidence schema | For every eligible leaf, run positive/invalid/denial-or-N/A with exact exits and immutable receipts; reject catalog-only and bad-config-only PASS; honor auth and UNKNOWN no-retry |
| `RACC-AUTO-03` P0 | Bounded concurrency/state-machine harness with scheduling barriers | Two actual isolated processes/Agents; one admission and zero forbidden effects, followed by successful new request, including a reproducible pre-fix failure |
| `RACC-AUTO-04` P0 | Reusable disposable isolation profiles | Clean install lane; nested Provider lane; scoped auth/Web/R-04 lane, with exact Box identity, ownership, independent observer and safe cleanup; unsupported auth/network gates remain blocked |
| `RACC-AUTO-05` P0 | Candidate-first build/install/upgrade acceptance gate | Exact artifact digest/source proof; installed original-repro rerun before public release; publish asset/official installer verification afterward, no default pin movement |
| `RACC-AUTO-06` P1 | Cross-ingress behavior and Provider contract comparator | Same exact target through available MCP/REST/CLI/Web routes, compare error types, receipts and side effects; unsupported operations and bypass paths reported, not assumed equivalent |
| `RACC-AUTO-07` P1 | Report generator, known-bad oracle gate and issue queue sync | Deterministic human feature table + per-action ledger from raw evidence; enforce evidence/cleanup fields; keep distinct open SUSPECT/UPSTREAM/UNKNOWN findings; generate safe next-agent packets |

Keep development tasks in the canonical project task/risk records when implementation is authorized; this backlog is a specification, **not authorization to execute new host policies, public exposure, credentials or destructive native Provider operations**. During a docs-only run, record that no code/tests/release were created.


## One-sentence invocation

`Run full BoxWeave reacceptance.`

An Agent in this repository should read this file and run the above phases under the active authorization. No separate magic phrase or advance approval is necessary for routine steps already in scope. This document is an orchestration **instruction**, not an implemented multi-Agent scheduler or test runner.
