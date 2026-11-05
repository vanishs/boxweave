# Reacceptance execution orchestration — upstream-aware implementation plan

**Status:** **initial scoped v0.47.0 execution engine implemented in source**, with installed/live acceptance separately recorded in [v0.47.0 verification](../verification/REACCEPTANCE_ENGINE_0470_2026-11-05.md). This remains the design for future **full** automation; no general background scheduler, all-transport fully autonomous test dispatcher, new public BoxWeave MCP endpoint or elevated privilege is supplied. The optional upstream Proof Pack adapter is test-owned, reviewed and explicit opt-in; documentation itself never executes it or releases a version. The established owner command remains **`Run full BoxWeave reacceptance`**. Existing [canonical runbook](../process/FULL_PROJECT_REACCEPTANCE.md), [verification template](../verification/REACCEPTANCE_RUN_TEMPLATE.md) and [safety restrictions](../../AGENTS.md) still take precedence.

**Reference set:** [pinned Containarium upstream reading guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md). Treat native upstream `v0.128.0` and future installed versions as different contracts; reading upstream `main` never verifies a running daemon. Current BoxWeave architecture is **MCP-first + REST-assisted**, not Containarium's CLI-first design.

## 1. Problem, non-goals and required outcome

The v0.46.1 reacceptance enumerated 68 installed local-stdio MCP names, 26 host/helper CLI leaves and 36 app modes, but numerous individual results remained PARTIAL/BLOCKED; many calls came through the **outer** Gateway and did not prove the installed **inner** transport. The run found hidden callable host commands (#80), real Compose 404 (#79), inconsistent missing-target errors (#81), and an ambiguous selftest observation (#83). Subsequent v0.46.2 work found a reproducible concurrency admission race that repeated serial selftests could not reveal. The existing 130-row ledger was valuable for accountability, but it was **not 130 PASS**.

The new architecture must make a missing or unreliable test **machine-visible**, give Agents guided high-risk counterexamples, support a separately provisioned Verifier, and preserve exact evidence of side effects and uncertain original operations. Its value is measured by newly discovered/verified failure boundaries, not a high green percentage.

**Non-goals:** replacing Containarium, making all native Provider writes subject to BoxWeave R-04 claims, cloning upstream's entire Go test suite, configuring operator-host credentials/network, running unreviewed repo commands as root, retroactively making earlier PASS exhaustive, or treating a proof artifact as a functional oracle.

## 2. Execution architecture and component ownership

`Run full BoxWeave reacceptance` currently instructs an **Agent**, not a resident scheduler. Preserve this path while incrementally implementing a deterministic runner and capability adapters. A future invocation may call the runner through an existing **authorized test-Box execution path**; do not require the Agent to guess host shell or direct Incus credentials.

```text
Owner: "Run full BoxWeave reacceptance"
  |
  +-- Coordinator (Agent)                 pins scope, permissions, versions and roles
       |
       +-- Upstream reference resolver    v0.128.0/installed tag, proto/MCP/CLI/routes, docs sources
       +-- Surface & impact inventory     source dispatch + installed CLI/help + SSH/HTTP MCP + Web
       +-- Scenario selector              features F01-F39/X*, delta analysis, incident playbooks
       +-- Test capability broker         read-only / disposable mutation / protected host (default deny)
       +-- Environment allocator          exact owned UUID/createdAt, isolated clean/seeded/auth lanes
       |
       +-- Finder (independent actor)     original positive/invalid/denied + race/fault cases
       |    +-- installed surface adapters
       |    +-- upstream contract comparator (read-only by default)
       |    +-- independent Provider/guest/receipt observation
       |
       +-- Fixer                          reproduce baseline, narrowly fix and produce candidate
       +-- Proof producer / case recorder capture original execution bytes, not coder assertions
       +-- Verifier (separate Agent/Box)  rerun original installed candidate failure + recovery
       |
       +-- Evidence validator             unalterable attempts, no missing-oracle PASS, known-bad control
       +-- Release coordinator            private candidate -> installed gates -> immutable publication
       +-- Report writer                  named feature table, action ledger, issues, handoff
```

**Role integrity:** a source/fixture run by Fixer does not count as independently installed verification. Proof producer executes test commands, Verifier decides whether tested user-visible behavior and side effects match. A separate pack-producer Box is desirable, but not always available; when the same Agent does all roles, mark `same-Agent` and avoid independently certified language. Multiple writable actors must use distinct Box identities unless a specific concurrency test explicitly targets the same immutable child.

## 3. Concrete adapters and invocations

| Adapter | Existing callable mechanism today | Implemented / proposed responsibility | Required boundary |
| --- | --- | --- | --- |
| `boxweave_outer_mcp` | Authorized `boxweave_info`, `list_containers`, `get_container`, `debug_container`, `connect(box, exec)`, other actually exposed tools | Read current external surface, provision/observe only owned Boxes, run a *guest* command through exact-target SSH. | Must re-read real tool schemas; outer MCP catalog is **not** proof of installed inner gateway method coverage. |
| `installed_host_cli` | In an **authorized disposable Box**: `sudo -n boxweave version`, `provenance`, `doctor --mcp`, `selftest plan/run`, `--help` | Enumerate special pre-parser dispatch and CLI leaves; test actual exit and installed package. | `--help` and bad-config only are not positive acceptance. Avoid host service mutations outside a dedicated lane. |
| `installed_stdio_mcp` | Real installed `mcp_gate.py` JSON-RPC `initialize` → `notifications/initialized` → `tools/list` → `tools/call`, SSH transport where supported | Build **runtime-derived** tool/schema and per-method triads against owned targets. | Must distinguish local stdio from remote SSH client and each effective auth/policy mode. |
| `installed_http_mcp` | Authorized `/mcp` client using a real OAuth/test-scoped token when configured | Compare catalog/tool behavior and revoked/forbidden/Origin boundaries. | Anonymous HTTP401 is denial-only; never mint or rotate credentials to satisfy a count. |
| `web_app` | Actual installed authorized Web session, planned/confirmed operations and worker receipts | Plan/confirm/worker/origin/csrf and observable Provider effects. | Never substitute fake-browser fixture for authenticated installed positive. |
| `upstream_contract` | Read pinned upstream `*.proto` annotations, `internal/mcp` tools, `internal/cmd` dispatch, `internal/client` and actual installed daemon/REST responses | Detect schema/route/error drift, capabilities and unsupported native behavior. | Native direct writes can bypass coordination; default **read-only** and test-only permission for mutation. |
| `upstream_proof_pack` | On dedicated nested **Provider host** with v0.128.0 capability: `containariumd proof-pack generate ...`; sealed `verify` separately | Optional source/unit/E2E **execution provenance** and immutable digest reference. | **No BoxWeave 68-tool MCP generator exists**; normal `containarium` client lacks command. Requires reviewed declaration, safe guest-root test, exact host-side permission. |
| `own_proof_recorder` | Proposed local runner/observer + immutable JSON/log digest | Mandatory independent, per-action **black-box** evidence if upstream proof pack is unsuitable/unavailable. | Cannot inherit unauthorized root access or trust caller's success text. |
| `github_source_release` | Existing GitHub repo/CI/release actions | Pin source, inspect diffs, open issues/PRs, reconcile candidate and published assets. | Publishing and changing default stable installer pin are separate actions; never publish during a documentation update. |

**Important invocation distinction:** an outer `connect(box="test-lab", exec="...")` enters the **test Box guest**. If that guest deliberately runs its own nested Containarium daemon, it is a nested **Provider host for its children**; only then can an approved `containariumd proof-pack generate <owned-child> ...` use its local Incus. It cannot reach the owner host's daemon by magic, and a guest command is not authority for arbitrary root actions on the owner host.

### 3A. Agent immediate-use decision sequence (before code exists)

1. Read `AGENTS.md`, `FULL_PROJECT_REACCEPTANCE.md`, the [upstream guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md), this design, the baseline, historical issue queue and active permissions. Pin exact BoxWeave installed process/release/Provider versions and source SHA. Do not infer a runtime version from documentation.
2. Use read-only BoxWeave MCP/catalog, installed CLI help/dispatch and genuine MCP `initialize/tools/list` at every authorized transport/policy. Compare source registration, installed actual callable leaves, advertised help and native upstream proto/runtime paths. Identify missing contexts explicitly.
3. Make a feature-by-feature [action ledger](../verification/REACCEPTANCE_RUN_TEMPLATE.md): actual normal, invalid/boundary, denied/stale if semantically relevant, independent readback. Map to F01-F39/X*; log capability levels separately (`implemented` / `enabled` / `proven` / `design-only` / `unsupported` / `blocked`).
4. Allocate **unique owned** disposable target identities, assign disjoint mutation writers, stage two-actor test only where relevant, set immutable stdout/stderr paths and capture errors on the first run. On a dispatched UNKNOWN, stop all future mutation attempts for that exact target and produce a read-only handoff, never auto-replay.
5. On a changed stateful boundary, schedule observable in-flight first action, competing second actor, zero-second-effect check, first action completion, and post-contention fresh success. Preserve pre-fix failure as a negative control.
6. For source-test evidence, prefer the BoxWeave-owned non-root proof recorder. Consider **upstream Proof Pack only if** its `generate` capability and guest-root commands are reviewed on a dedicated nested Provider host; an ordinary auth-only MCP test does not justify invoking it.
7. Fix source and prepare an *installable exact candidate* without public Release; have a separate Verifier rerun the original black-box failure on installed candidate first. Use publish-free CI rehearsal where implemented. Only then follow the project's approved immutable release pipeline; separately confirm official-release bytes, runtime and stable pointer.
8. Produce human function matrix **and** complete per-action ledger from the same raw case IDs. Preserve unresolved suspected symptoms, upstream cause unknowns, partial/unavailable oracles, blocked profiles, cleanup identities and exact next-Agent probes.

This procedure is executable manually through available channels now; it is **not** a claim that a one-command test engine is already wired.

### 3B. Upstream Proof Pack optional pathway and fallbacks

```text
Discover: installed containariumd version and proof-pack --help
    |
  Is this a dedicated authorized Provider host with a child owned by this run?
     NO -> skip upstream integration, use BoxWeave recorder (BLOCKED upstream path)
    YES
    |
  Inspect the child's pinned checkout and .github/containarium.yml test:
  (or implicit Makefile fallback), nested scripts, root side effects, network/deps
    |
  Is each command and proof export explicitly safe and authorized?
     NO -> use BoxWeave recorder; preserve exact denied capability
    YES
    |
  Generate pack from host through approved broker:
  containariumd proof-pack generate <exact-owned-child-container>
     --dir <reviewed-checkout> --out <private-host-path>
    |
  Capture exit, source/declaration/log hashes, original pack, verify guest cleanup
    |
  Is it sealed (not just generated) with an accessible audit chain?
     YES -> independently verify via the *supported* host/REST path and record
            provenance/anchor scope; NO -> generated-only, no sealing claim
    |
  Link pack to source-test case IDs. Run separate BoxWeave black-box and
  Provider/guest oracles. Pack PASSED != all user capabilities PASSED.
```

**Do not silently create `.github/containarium.yml` or fall back to `make test` on an unreviewed repository.** Upstream `generate` runs inside the child as **root** and can execute the declared shell commands. The default runner should execute BoxWeave-owned test cases without root; host-side Proof Pack is an **opt-in specialized adapter**, not a mandatory dependency or new MCP facade. The `verify` route is for sealed packs; it cannot safely be used as an arbitrary test runner.

## 4. State, schema and PASS classification contracts

### 4A. Proposed test-owned evidence layout

```text
reacceptance/<RUN_ID>/
  run-manifest.json          # creator roles, exact source/release/runtime, grant, upstream SHA
  upstream-map.json          # tag/SHA, source files, proto RPC/HTTP mappings, status of each route
  surfaces.json              # source vs actual installed callable/advertised by transport/policy
  dependencies.json          # upstream + BoxWeave path globs -> actions -> F/X features -> cases
  cases/<CASE_ID>/
    attempt-001.json         # immutable raw sanitized argv/protocol, timestamp, exit/error
    stdout.bin.sha256        # actual bytes/log hash, size, truncation indicator (private bytes elsewhere)
    stderr.bin.sha256
    observer-001.json        # independent provider/guest/db state, before/after, effect count
    cleanup-001.json        # exact target and deletion/retention readback
  proofs/<PROOF_ID>.json     # optional generated/sealed upstream pack with clear trust level
  negative-controls.json    # seeded false-oracle / false-claim DETECTED or MISSED
  coverage.json             # complete triad/gate states from actual case links
  report-features.md        # all named capabilities; no count-only PASS
  report-actions.md         # every MCP/CLI leaf and relevant Web/REST path
  issues.json               # original symptom separate from confirmed sub-defect, next-agent packet
```

Store raw logs privately in an approved test-owned evidence directory; sanitize any shared markdown/PR attachments and enforce bounded capture. Log digests **do not make a test correct**; they make the observed attempt detectable and reproducible. A timestamp or source SHA cannot substitute for an immutable original Box identity.

**Minimum case fields** (exact serialization will be frozen when implemented):

```json
{
  "schema_version": 1,
  "run_id": "RACC-EXAMPLE",
  "case_id": "SC-SELFTEST-CONCURRENT-EXAMPLE",
  "source_sha": "<exact-source-sha>",
  "release_digest": "<candidate-artifact-sha256-or-null>",
  "installed": {"boxweave": "<observed>", "containarium": "<observed>"},
  "target": {"logical": "<owned-test-name>", "provider_uuid": "<observed>", "created_at": "<observed>"},
  "invocation": {"interface": "installed_host_cli", "actor": "first-test-actor", "policy": "<observed>"},
  "requirements": ["positive", "denied", "no-forbidden-effect", "after-contention-recovery"],
  "attempts": ["<immutable-sanitized-case-artifact-id>"],
  "oracle": {"observer": "independent-incus-list-and-journal", "effects_before": 0, "effects_after": 1},
  "proof": {"kind": "none", "status": "NOT_REQUESTED"},
  "outcome": "PARTIAL",
  "not_tested": ["example case is not an executed result"]
}
```

### 4B. Orthogonal statuses (never collapse)

- **Capability state:** `implemented` / `opt_in` / `design_only` / `unsupported_here` / `unknown`. This is not a test result.
- **Execution/evidence level:** `S` source, `T` fixture/regression, `I` isolated installed, `H` owner host, `X` independent external client, `U` unavailable. These dimensions are distinct and do not monotonically imply each other.
- **Case outcome:** `PASS`, `FAIL`, `PARTIAL`, `BLOCKED`, `SKIPPED`, `UNKNOWN`; separate classifier `CONFIRMED_BUG`, `SUSPECT`, `UPSTREAM`, `ENVIRONMENT`, `HARNESS_DEFECT` for incident ownership.
- **Proof mode:** `NO_PROOF`, `RUNNER_RECORDED`, `UPSTREAM_GENERATED`, `UPSTREAM_SEALED_VERIFIED`. `UPSTREAM_GENERATED` is weaker than sealed/independently verified; both are separate from a tested postcondition.
- **Blocker reason:** actual missing auth, missing child/collector/backend, owner authorization excluded, installed interface unavailable, dependency missing, unimplemented automation; **not** low priority.

A user-facing PASS requires an actual **eligible positive** invocation, applicable malformed/denied boundary or explicit justified N/A, **original exit/response**, independent oracle for effects/no-effects, exact version/interface, and tested cleanup. A `proof-pack` success, `doctor 10/10`, `tools/list`, `--help`, schema compile, API 401, or missing-config exit alone can only support narrower claims. The required dimensions must be checked by the verifier, not inferred from a narrative. A seeded **known-bad** output must fail the same evidence validator; if it passes, related green claims downgrade to `HARNESS_DEFECT`.

A claim of **independent Verifier** requires a distinct Agent or equivalent independent run context and source of observation; a detached process launched by the Fixer isn't itself independent verification. Do not erase earlier original ambiguous output when closing a reproducible concurrent bug.

## 5. Change-impact and upstream version watch

Define a bipartite/many-to-many dependency registry **in addition to**, not replacing, `DOCUMENT_AUTHORITY.json` or `FEATURE_MATRIX.md`:

- `doc_section_id`, `source_globs`, `version_last_checked_sha`, `runtime_adapters`, `surface_ids`, `feature_ids`, `case_ids`, `evidence_grade`, `status`, `source_of_truth_url` and optional `issue`.
- On every new BoxWeave source run, diff source paths and map changes onto current entrypoints and cases; on every pinned upstream version change, diff `proto`, `internal/mcp`, `internal/cmd`, `internal/server`, `internal/client`, Incus/SSH, release and Proof Pack packages. Run new/changed cases first, then all eligible leaves to honor the owner's **full reacceptance** contract.
- Require that every new/renamed/removed/hidden actual callable action be assigned a stable feature and negative/positive test plan. A `--help` or parsed decorator count is only discovery, never completion.
- For an upstream route 404 (#79), pin **declared proto REST annotation**, **actual installed HTTP route**, **direct native server result**, **forwarded platform MCP** and **guest/systemd state** separately; do not upgrade/patch native daemon just to make the ledger green.
- For the error mapping incident (#81), compare missing/duplicate/invalid actions through native RPC/REST, forwarded MCP, BoxWeave app CLI and Web, capturing typed error and proving no accidental create.
- A code-change impact map prioritizes cases; it **never** retroactively grants installed PASS to unchanged leaves.

## 6. Environment allocator, permissions and safety

**Default mode: read-only metadata and inventory.** Disposable mutation is allowed only when the current owner grant, test-Box ownership/UUID/createdAt and backend are verified. The broker must treat a logical Box name as untrusted until bound to the exact incarnation; refuse reassignment/stale identity and never delete or release an UNKNOWN Box merely to satisfy cleanup.

Required lanes:

1. **Inventory lane:** installed tools/CLI/Provider read-only, no credentials or changes.
2. **Clean nested lane:** fresh exact-version install and nested child for user-visible writes; outer independent observer/cleanup.
3. **Seeded-upgrade lane:** install pinned older release then candidate, preserve known marker and receipt; separate from clean proof.
4. **Scoped-auth lane:** explicit test-only Web/admin/OAuth/SSH identities and *isolated opt-in* R-04, if granted. Absent permissions are BLOCKED, not reason for new token provisioning on owner host.
5. **Fault-injection lane:** separately created contained Provider/worker/DB to kill/restart without touching outer `dev` or operator host.
6. **External-client lane:** real native Windows/WSL/SSH/HTTPS only with actual approved client and network, else X unverified.

Define `OWNED_FOR_RUN`, `PRESERVE_FAILURE_SCENE`, `UNKNOWN_NO_REPLAY` and `CLEANED_WITH_ORACLE` resource states; persist the exact original ID and cleanup proof. Parallel tasks cannot mutate the same Box except during a deliberately coordinated race. Max runtime, Box quotas and budget are explicit per phase. Loss of the coordinator/worker does not authorize an automatic compensating mutation.

**Forbidden without a new owner authorization:** operator-host reboot (R-02), protected R-04 enablement on an owner host, original UNKNOWN release/replay, direct native shrink, storage/Incus/SSH/network/tenant/credentials change on owner hosts, external public exposure, or arbitrary `containariumd` execution. A user request for reacceptance is not blanket root permission.

## 7. Test case families and proof requirements

| Family / priority | Original failure or risk | Minimum black-box and independent evidence |
| --- | --- | --- |
| P0 source/runtime/help parity | CLI #80: callable hidden families | Discover installed dispatch and top help; safe invocation across all leaves, compare source/installed help; create a known-hidden false input control |
| P0 stateful parallel admission | Selftest #83 and R-02/R-04 | First actor observable in-flight, second typed denied/idempotent, zero second remote effects, original cleanup, fresh subsequent action PASS |
| P0 API/gateway behavior | Compose #79 and error #81 | Native/forwarded/REST route/CLI error and effect parity on test-owned exact target, upstream version pinned; attribute only with native evidence |
| P0 release candidate | Previously public release precedes installed verification | Candidate SHA/digest, isolated installed original-repro and recoverability before publish, same payload in immutable release/asset |
| P1 Web/HTTP auth | 401-only evidence does not prove positive | Real isolated authenticated login/handshake/tools call, CSRF/Origin/revoke deny, provider observer |
| P1 document/claim drift | DOCMAP or CLAIMS stale | Source dependency/last-checked version, status claim tied to tests/runtime, intentional misleading-claim fixture must fail |
| P1 failure recovery | Native SSH/DB degradation, worker timeout | Before/fault/degraded/recovery, typed errors, zero forbidden state, no blind replay, original receipt retained |
| P2 ongoing upstream expansion | New MCP tool/Incus ability | Exact new upstream MCP/proto/CLI and BoxWeave surface reconcile, per-action triads and supported/unsupported classification |

For native parallel/fault behavior, collect precise external call counts or an equivalent durable Provider/guest marker, not solely client retries. Source-only and installed black-box failures must be reported as separate boundaries.

## 8. Implementation work packages and ownership

The following **code tasks are proposed, NOT implemented**. Carry forward runbook's `RACC-AUTO-01`–`RACC-AUTO-07` identifiers; do not create a second competing priority registry.

| Task | Recommended owner | Build artifacts/interfaces | Exact acceptance criterion |
| --- | --- | --- | --- |
| `RACC-AUTO-01` surface + upstream drift | Inventory/Finder | `scripts/reacceptance/inventory`, source/help/MCP adapter probes, upstream pinned mapper, `surfaces.json` and dependency registry | Reproduce hidden-command issue with deliberately hidden callable command; pinpoint changed proto route and all affected feature IDs, no catalog-only PASS |
| `RACC-AUTO-02` safe case dispatcher | Coordinator/Executor | Versioned case schema, per-transport task implementations, preflight safety broker, case output store | Execute eligible per-action normal/invalid/denial and independent readback; case with missing config only cannot PASS; UNKNOWN halts same-target mutators |
| `RACC-AUTO-03` deterministic races | Finder/concurrency specialist | Two-process barrier, verified in-flight observer, short state-machine sequence library, crash/failure injection | Original pre-fix parallel selftest fails known control; fixed installed candidate passes exactly one side effect and a fresh after-contention run |
| `RACC-AUTO-04` disposable lanes | Environment owner | Owned identity registry and clean/seeded/scoped-auth/fault lanes, cleanup state machine | Only exact owned UUID/createdAt can be mutated/deleted; fault scene retained; owner host is inaccessible to hazardous cases |
| `RACC-AUTO-05` candidate-first gate | Release/Verifier | Publish-free build rehearsal, candidate digest/installer, prepublication installed verification, final tag bytes comparator | Final released bytes identical to installed-verified candidate (or rerun on change); no tag created when original repro fails; stable pointer unchanged |
| `RACC-AUTO-06` contract comparator | Finder/upstream specialist | Pinned proto/HTTP/MCP/native/CLI mapping, typed error normalization, disabled/unsupported paths | For #79/#81-like cases distinguish upstream route absent from BoxWeave translation; zero unauthorized writes; source does not prove runtime |
| `RACC-AUTO-07` proof & report | Evidence producer/Verifier | Private non-root recorder; optional gated upstream Proof Pack adapter; immutable output; claims/doc dependency table; issue/report sync | Tampered log/false PASS rejected by known-bad control, features+actions generated from same cases, unresolved original symptom remains open |

**Implementation sequencing:**

- **Milestone M0 (docs/reference — this change):** link upstream guide and execution plan into active trigger, report template, test strategy and index. No runner/CI/release invented.
- **M1 (P0):** schema + read-only discovery, dependency mapping, **non-root** evidence recorder and classifier. Must demonstrate test-harness failure on planted missing evidence/hidden action.
- **M2 (P0):** isolated profile and installed direct MCP/CLI actions; concurrent-state engine with barriers and independent effects. Real baseline-vs-candidate selftest case.
- **M3 (P0):** publish-free candidate builder and installed Verifier gate, then official tag/asset validation. Integrate with current release workflow without changing stable pin.
- **M4 (P1 optional):** upstream `proof-pack` host integration and sealing verification under explicit authorized nested Provider host; multi-Agent role automation; Web/HTTP client and cross-ingress comparator; auto-generated reports.
- **M5 (P1):** upstream-diff-triggered test selection and documentation claims CI; full owner-facing audit with per-feature gap and next-Agent packets.

**Definition of done per work package:** source tests, intentionally failing negative control, installed isolation test as relevant, immutable evidence, exact permissions, independent-or-same-Agent distinction, docs and verification status. No result is called `automated` until its executable exists and a fresh installed run proves it.

## 9. Failure gates and non-claims

- **Gate G0 inventory:** all discovered actual MCP/CLI leaves and relevant Web/REST paths mapped or precisely BLOCKED; real auth catalogs distinguished from outer aliases.
- **G1 case coverage:** every eligible action attempted with necessary positive/invalid/denial triad or justified N/A, original exit/log preserved and independent mutation oracle. Summaries alone fail.
- **G2 adversarial:** changed stateful boundaries have controlled concurrency/fault cases; any uncontained dispatched UNKNOWN stops that target; prior symptoms tracked separately.
- **G3 proof:** known-bad assertion *must* fail; upstream generated/verified pack has separate provenance grade; tester/coder independence stated.
- **G4 candidate:** applicable original black-box reproduction corrected in *installed* pinned candidate **before public release**, CI and build gates passed.
- **G5 publish:** immutable tag/assets/manifest SHA verified, official-release smoke and stable pointer observation separate; no premature blanket PASS.
- **G6 report:** fully named function table plus every action status, upstream ownership unknowns, residual SUSPECT, cleanup and next-agent packets.

**When the platform lacks a tool:** execute a safe scoped manual alternative (using existing authorized MCP/CLI) or report `BLOCKED/NOT_IMPLEMENTED`. Do **not** create generic host-exec, hidden elevated REST endpoint or public token just to make the engine appear complete.

## 10. Completion handoff for future coding Agent

First read this file, then the [upstream reference guide](../verification/UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md), current [runbook](../process/FULL_PROJECT_REACCEPTANCE.md), [action template](../verification/REACCEPTANCE_RUN_TEMPLATE.md), `DOCUMENT_AUTHORITY.json`, current implementation/code and exact installed versions. Implement one approved milestone at a time while preserving the canonical trigger. Run all new code and negative controls in disposable source/test Boxes; preserve previously accepted real user workflows. **Do not activate any new privileged host adapter by default**.

The owner should continue to invoke `Run full BoxWeave reacceptance` without changing the phrase. Documentation here makes future orchestration implementable and current manual Agent operation more informed, but **does not itself schedule, delegate, run or certify any tests**.
