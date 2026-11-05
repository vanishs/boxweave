## v0.44.10 selftest operator plan handoff — 2026-11-05 UTC

[Root-bounded plan read, app-user stdin handoff, typed diagnostics and exact-version test plan](SELFTEST_PLAN_HANDOFF_04410_2026-11-05.md): **selftest permission issue repaired in source**; host private 0700 home and 0600 plan no longer require moving or changing permissions, while strict confirmation, journal and fail-closed controls remain. Targeted selftest tests and exact-source 748 Python + build + 7 Chromium fixture workflows PASS. Full v0.44.10 source/CI/installed acceptance must be separately marked from earlier v0.44.9 evidence. **R-02 owner-host cold-reboot receipt test explicitly DEFERRED by owner**; no restart, source/host storage or global R-04 activation. Default stable v0.43.1 unchanged.

## P1-05 / P1-06 issue disposition — scoped v0.128.0 native MCP checks, 2026-11-05 UTC

[Issue #76](https://github.com/vanishs/boxweave/issues/76) **closed NOT_REPRODUCIBLE (not verified fixed)**: two independent fresh native MCP creates returned success and independently materialized distinct immutable instances; a known-existing-name duplicate returned expected PROVIDER_REJECTED with no changed identity. The original v0.43.1 response/side-effect correlation was not recovered, and P1-03 historical UNKNOWN remains blocked; no change to source fail-closed create policy. [Issue #75](https://github.com/vanishs/boxweave/issues/75) remains **UPSTREAM OPEN / DEFERRED**: sampled BoxWeave no-shrink guards passed, but native 0.128.0 metadata shrink was confirmed; no explicit fix is mentioned in upstream through v0.130.0 release notes. Both newly created #76 isolated children were cleaned after immutable identity verification. Existing earlier version-scoped outcomes below remain historical, not retroactively changed.

## v0.44.9 evidence-depth crosswalk — 2026-11-05 UTC

[39 feature groups, source/test/isolated installed/owner-host/external-client evidence levels, version caveats and P0–P2 gaps](EVIDENCE_LAYERS_0449.md) are a documentation-only evidence index. A source or fixture PASS is not owner-host or external acceptance. P0 PARTIAL: [official v0.44.9 sampled three-entrance Web/SSH/HTTP conflict and true Gateway-crash UNKNOWN L PASS](REMEDIATION_TEST_QUEUE.md), while Web-first in-flight, natural expiry, reincarnation, historical UNKNOWN reconciliation and owner-host global enforcement remain U. P1 open: outer R-02 reboot persistence, genuine trust-readiness recovery, external OAuth and real Windows/WSL clients, native Provider create-after-error and shrink. The existing REQ/AC mapping and historical acceptance below remain authoritative within their tested scope.

## v0.44.9 source-grounded architecture and call-chain view — 2026-11-05 UTC

[Detailed entrypoint graph, REST/MCP Provider selection, shared SQLite, protected-mode policy, mutation coverage and P0/P1 test matrix](../design/RUNTIME_CALL_CHAINS_0449.md) — **S (source review)**; it does **not** change recorded installed outcomes. App-owned basic lifecycle REST and specialist MCP coexist; SSH and Streamable HTTP MCP share the standalone Gateway; Web/CLI use independent app services. Gateway tracked tasks participate in shared claims, but without separate root protected mode other forwarded writes are not globally coordinated; app claims are independently opt-in. **P0/PARTIAL** full authenticated Web/SSH/HTTP conflict, crash/lease expiry and bypass audit; **P1/PARTIAL** outer R-02 post-reboot durability, external HTTP OAuth/TLS, native shrink/create-after-error; see [current v0.44.9 real evidence](REACCEPTANCE_0449_2026-11-05.md). No host settings, source behavior, or stable installer pin changed for this documentation review.

## v0.44.9 release + independently installed upgrade — 2026-11-05 UTC

[Exact source, CI, official v0.44.8→v0.44.9 isolated installation, doctor 10/10 and live create/hello/stop/delete with cleanup](REACCEPTANCE_0449_2026-11-05.md) **PASS**. Follow-up: operator-home selftest plan file is unreadable by app user, yielding typed `SELFTEST_PLAN_INVALID`; app-private owned 0600 plan succeeds. P1-01 real outer tracked task PASS / post-reboot U; P1-02 partial, P1-03 blocked, P1-04 full matrix partial, P1-05/06 upstream open. Stable pin v0.43.1 unchanged; owner outer host not upgraded.

## v0.44.9 P1 and broad implemented-surface reacceptance — 2026-11-05 UTC

[Authoritative new candidate test and owner-host task ledger](REACCEPTANCE_0449_2026-11-05.md): R-02 outer initialized, live tracked receipt/get/list/dedup L PASS (post-provision cold reboot U); R-04 dispatched-claim competing writer now returns RECOVERY_REQUIRED without releasing/replaying (fail-before-fix + targeted T PASS), full cross-entrypoint matrix still PARTIAL; 742 Python tests, knowledge, frontend and seven browser fixtures T PASS on the patched source under an actually enabled isolated protected policy. P1-02 PARTIAL, P1-03 BLOCKED, P1-05/P1-06 UPSTREAM OPEN. Historic 0.44.8 source/installed evidence remains scoped below. v0.44.9 release/CI and exact installation must be checked separately.


## 2026-11-04 PT / 2026-11-05 UTC — v0.44.8 P1 acceptance crosswalk

[Authoritative new six-item evidence, exact environments and remaining gaps](P1_REMEDIATION_ACCEPTANCE_0448_2026-11-04.md). P1-01 **PARTIAL / host provisioning blocked** (new installed tracked receipts, dedup, cold restart PASS); P1-02 **PARTIAL** (successful new app creates, real transient TRUST_FAILED not observed); P1-03 **BLOCKED** (original UNKNOWN immutable-incarnation correlation missing); P1-04 **PARTIAL** (real app-vs-protected Gateway lease conflict, independent Box, killed in-flight Gateway UNKNOWN with one actual guest side effect; full Web/HTTP/SSH concurrency still U); P1-05 **UPSTREAM OPEN** (BoxWeave source-side failed-create UNKNOWN regression PASS); P1-06 **installed sampled BoxWeave pre-dispatch refusal PASS / UPSTREAM OPEN** (real native 0.128.0 disk shrink confirmed and reversed in isolated child). Official v0.44.8 SHA `3ff0a6c7`, 740/740 tests and new installed doctor 10/10; no owner-host, original incidents or stable release pin mutated. Historical candidate checkpoints remain date-scoped.

## 2026-11-04 v0.44.8 candidate: expired observing R-04 claim classification

[Reproduced fail-before-fix and pending release gates](R04_DEADLINE_CLAIM_SYNC_0448_2026-11-04.md). An already acknowledged create could reach application `unknown/DEADLINE_EXCEEDED` via the worker's direct deadline branch while leaving the shared resource coordinator at `dispatch_intent`; another Agent saw RESOURCE_BUSY instead of the more accurate RECOVERY_REQUIRED. New fixture demonstrated that failure, minimal classification-only code change keeps original resource claimed and no command replay. Source-focused 13 PASS; complete source and independent official installed v0.44.8 gates still pending at this checkpoint. Original two UNKNOWN incidents retained; outer host unaffected; upstream Containarium defects only recorded.

## 2026-11-04 official v0.44.7 installed R-04 follow-up

[Current installed-release evidence](RELEASE_0447_POSTINSTALL_R04_2026-11-04.md): release/provenance/doctor 10/10 L PASS; official v0.44.7 new Box create and connection-ready L PASS; R-04 same-target UNKNOWN rejection, direct write rejection, app-vs-lease contention and identical-key parallel dedup L PASS on sampled pathways. New test child safely removed, originals preserved. Actual transient TRUST_FAILED recovery on **official installed v0.44.7 remains U** (source fixtures T PASS); owner-host R-02 U/BLOCKED; upstream native create-after-error and shrink atomicity OPEN; real HTTP admin, cross-client matrix, external transports U. Historical v0.44.6 patched-candidate claims below are not upgraded by implication.

## v0.44.7 protected R-04 and post-create trust-readiness scoped crosswalk

[Exact failure/retest evidence and boundaries](TRUST_READINESS_R04_0447_2026-11-04.md): **S/T PASS** bounded post-create `TRUST_FAILED` observation/retry vs stable-target refusal, without replay; 36 focused cases and 738 full Python tests PASS. **L PASS on isolated manually patched v0.44.6 candidate**: real Web lifecycle creates/identity, shared-claim blocked/terminal semantics with separate protected Gateway, duplicate task side effect once, lease holder/busy semantics and exact live child identity. **U** source-bound v0.44.7 CI/release and installed v0.44.7 upgrade, production host R-02 provisioning, real authenticated installed Web HTTP, global SSH/REST/Web R-04 matrix, upstream create-after-commit ambiguity/native atomic shrink. No current-owner host or stable pin mutation. Distinguish legacy default mode source fixtures from the deliberately enabled protected test guest.

## v0.44.6 current isolated installed crosswalk (2026-11-04)

[Evidence and exact boundary](FOUR_GATE_ISOLATED_ACCEPTANCE_0446_2026-11-04.md): **L PASS** Nesting configuration + restarted parent + real nested MCP child create/Python hello-world/stop/delete; fresh 0.44.5 install, 0.44.5→0.44.6 upgrade and installed 10/10 MCP doctor; R-02 after explicit local private store provisioning, task receipts/dedup/restart/parallel side effect once; MCP and authenticated REST no-shrink on live children and zero REST PUT; installed app's Web-shared lifecycle plan no-shrink on a managed live child; installed v0.44.6 physical `-container` alias positive and wrong-IP negative. **T PASS** Web route authenticated Flask fixture 422/zero dispatch plus wrong-Origin 403. **L BLOCKED/U** actual authenticated HTTP POST to the installed Web listener (only unauthenticated 401 observed); owner-host installer/upgrade and R-02 provisioning; global R-04 cross-entry claims and upstream-native atomic disk shrink enforcement. New parent stopped and child resources deleted. `PUBLISHED_RELEASE` still 0.43.1. Older reports below remain historical for their exact source/installation revisions.

## v0.44.5 current-round scoped crosswalk

[Feature and combination results](REACCEPTANCE_0445_2026-11-04.md). L PASS for isolated Box lifecycle, CPU/memory writes and readbacks, native Tenant/guard posture; connect/file reads BLOCKED by gateway exact trust, task tracking unprovisioned. Disk-shrink safeguard S+historical T only, installed 0.44.5 U. Fresh install, Web/auth/REST, R-04 global, external clients/OAuth, monitoring positive, E2–E4 and external Provider upgrade/rollback remain U/SKIPPED; source CI/release PASS at exact immutable tag `156afa40` ([run 41820398547](https://github.com/vanishs/boxweave/actions/runs/41820398547)): 731 unit, seven browser, 205 docs, 71 Python 3.14 focused tests and 25 assets; installed tests still BLOCKED.

# Feature/evidence matrix — historical phase6 scope plus 0.13.0 remediation obligations\n\n**Latest source/live supersession:** [v0.44.3 whole-project acceptance and remaining gates](REACCEPTANCE_0443_2026-11-04.md). Older phase6 and v0.26.x judgments below are preserved by exact version; they do not supersede the v0.44.3 isolated installation, authenticated MCP and provider 0.128.0 checks.\n

Current authority SRC-084..088: [comparison and run inventory](COMPARISON_REPORT.md), [risk dispositions](OPEN_RISKS.md), [fresh phase6 evidence](phase6-runtime-evidence.json), [actual generated Windows enrollment](phase6-windows-web-profiles.json). Three missing/false-positive paths in0.9.0 are now fixed; earlier completion prose does not override these findings. The following REQ→AC→TASK/implementation→normal/error/authority/boundary/state matrix is authoritative. Fixture, current live, previous live, human and static evidence stay distinct.

| REQ / AC | TASK and implementation | Executed normal / abnormal / authority / boundary / state checks | Evidence and remaining boundary |
| --- | --- | --- | --- |
| REQ-001 / AC-001 |003/007/014; profile_client, profiles.generate, ClientSetup |4 native/WSL Codex/ZCode descriptors + hash-gated scripts, explicit replacement/preservation; wrong runtime/pin/version/source/fields, exact WSL distro; unchanged config after repeat | Fresh browser downloads + actual Windows scripts/registration; prior human UI separately retained; no automatic Web execution |
| REQ-002 / AC-002 |003/005/007/014; SSH pin and identity validators | Separate server/workload trust, malformed public pin/private-content rejection, identity-before-command; changed trust stops without key repair | tests/test_profiles.py, test_enrollment.py, public profiles; prior actual trust/identity negatives. No bypass or key export |
| REQ-003 / AC-003 |003/007/014; read_provider + CLI diagnose | Empty inventory valid only after real protocol/discovery/read; dead launcher fails; no database dependency/migration; registration/GUI remain separate | New empty-target regression and actual0.9.0/0.9.1 same-input counterexample; current109-tool read preserves DB6 bytes |
| REQ-004 / AC-004 |005/006/014; Target/Service/Provider | Exact backend/project/logical/creation identity, workspace, stale target/revision refusal before hello | Prior actual installed fixed hello and4 transports; new source/browser binding detail; fixture identity versus remote execution labeled |
| REQ-005 / AC-005 |002/006/014; persisted intents/worker | Delayed/lost create, original name, transient absence, crash and no duplicate/force adoption; scope/identity mismatch | test_lifecycle/test_recovery + prior real scoped create/cleanup; no new remote provisioning in phase6 |
| REQ-006 / AC-006 |002/005/006; deadline/readiness/exit | Running vs SSH, one persisted deadline, clock rollback, actual exit7, lost worker/null exit and output bounds | Current163 suite and local actual same-input exits0/7; prior real readiness. No remote cancellation SLA |
| REQ-007 / AC-007 |002/004/014; history/reconcile/unbind | Actor-scoped paging/restart/unknown, stale queued work fails after unbind, receipts retained without replay | test_history/test_recovery/new binding tests and browser; prior real restart evidence |
| REQ-008 / AC-008 |012/014; distribution/Store7 | Hash/path/ownership/concurrent install gates, consistent backup, failure marker/repair, settings/data preservation;6→7 migration and old binary rejection |21 distribution cases + actual independent-copy0.9.1 install, backup6/database7/original bytes/PIDs unchanged. Primary0.9.0 deliberately retained |
| REQ-009 / AC-009 |012/014; health/deploy_check/diagnose |401 reachability not auth; state categories, oneshot/timer distinctions; empty-target false-ready fixed | Current actual109-tool read and copied install, prior fresh supporting timer observations. No boot/confinement claim |
| REQ-010 / AC-010 |004/014; binding tombstones, revisions, Web | Create/reselect/detail/edit/unbind/rebind; stale plan/CAS/auth/Origin/CSRF; queued work stops; project/Graph/receipt remain; offline unbind | Seven new binding/diagnostic tests, source browser and actual installed-copy unbind. No cancel of already dispatched work |
| REQ-011 / AC-011 |004/009/010/014; authenticated API/shared services | Login/logout/reset/expiry/CSRF/Origin/exact bodies, one-use/stale plans, receipt outcomes; Web generator performs no infrastructure operation | Existing and new Web tests + seven actual browser workflows with fixture infra; prior actual Web/worker and current isolated installed browser |
| REQ-012 / AC-012 |009; Wire/Access | Account/membership separation, shared new create, exact capability/Guard/revisions, atomic conflict/no partial policy and degraded revoke | Current test_access; previous actual0.120.1 native/TCP evidence for unchanged modules. Full old live comparison unverified |
| REQ-013 / AC-013 |010; owned protocol-specific routes | Target/IP reuse/collision/owner nonce/ambiguous writes; exact approved audience, configuration vs actual packets, owned removal | Current tests and fixture browser; previous actual TCP/UDP positive/negative/cleanup and convergence delay. No new public route |
| REQ-014 / AC-014 |011; Graphs/XYFlow | Card/context/comment/link persistence, CAS drafts, clipboard rejection, escaped/redacted export, graph-only delete, zero execution | Current Graph tests/full browser and project unbind retains Graph; no graph scheduler or complete secret detector |
| REQ-015 / AC-015 |008; Models/native handoff | Exact public provider/model/version, stale revision/invalid choices/secret fields, native-existing preserves mode/settings; no inference | Current model tests/browser; separate existing user confirmation Z.ai/GLM-5.3/access SRC074. No fresh inference/model SLA |
| REQ-016 / AC-016 |003/012/014; digest/public artifact gates | Tamper before setup; generated descriptor+helper digests before local execution, chosen replacement survives, unknown credential fields rejected, recognized-secret redaction | Fresh real downloads/PowerShell hashes/Windows scripts, distribution tests and exact source manifests. Operator trust, not publisher signature |
| REQ-017 / AC-017 |007/013/014; evidence/matrix | Actual Agent/runtime/protocol/source revisions, declared unsupported states, separate old/new static/fixture/live and human evidence | Current generated4-profile registration, previous actual hello/human UI, fresh suite/install/comparison. No fabricated performance/coverage percentages |
| REQ-018 / AC-018 |003/006/009/010/014; allowlists/pinned targets | Wrong owner/backend/host mutation fields denied; exact destructive/public scope; unbind is local and generator never contacts Provider | All denial/actor/scope tests, actual original state preserved and private test services cleaned up. Broad upstream authority remains single-operator trust boundary |

The historical selected phase6 P0/P1 comparison was closed within its reported scope. **REQ-019 remains open; REQ-020 has the selected CLI evidence recorded in MAIN_014_REVIEW**; the owner-approved integrated first-host experience remains incomplete. Source-level phase7 entry is eligible; phase7 maintenance is recorded separately; production deployment is not implied. See OPEN_RISKS for explicit operational/unverifiable bounds.


Historical matrices: [preserved snapshot](FEATURE_MATRIX_HISTORY.md).

## New 0.10+ requirements — separate from historical phase6 acceptance

| REQ / AC | TASK and implementation | Executed normal / abnormal / authority / boundary / state checks | Evidence and remaining boundary |
| --- | --- | --- | --- |
| REQ-019 / AC-019 | TASK-015; `install.sh`, `bootstrap.py`, `distribution.py`, ADR-007 | Dedicated app-account/bus/launcher/token-rotation bootstrap, actual MCP read and app failure/repair passed; complete empty-host/public release/reboot remains unverified | **Open**: host bootstrap incomplete. [queue](REMEDIATION_TEST_QUEUE.md), RISK-610 |
| REQ-020 / AC-020 | TASK-016; `boxweave.sh`, `manage.py`, `deploy_check.py` | Current unit regressions plus installed version/status/doctor, actual exact local Incus reads, invalid/missing targets, scoped authenticated MCP and failure/repair. [Current evidence](MAIN_014_REVIEW.md) | **Accepted within selected CLI scope**; wider host/network claims remain separate. [queue](REMEDIATION_TEST_QUEUE.md), RISK-611 |

REQ-003/008/009/011/018/019 nesting extension: TASK-017/ADR-008 -> nesting.py, mcp_gate.py, online.py, manage.py, provider.py, lifecycle.py, Environments.tsx -> test_nesting.py contract checks; [same-child actual comparison and U new-helper boundary](NESTING_VERIFICATION.md). Prior0.15.1 execution is T for that exact version only.

## Remote MCP HTTP(S) / OAuth (implemented; fixture plus earlier loopback acceptance)

| REQ / AC | TASK and implementation | Executed normal / abnormal / authority / boundary / state checks | Evidence and remaining boundary |
| --- | --- | --- | --- |
| REQ-021 / AC-021 | REMOTE-R0/R1; remote_mcp.py, existing mcp_gate.py | Fixture suite plus 0.24.0 local HTTP initialize/list/call, Origin/protocol negatives and SSH parity | 0.24.0 loopback acceptance; 0.25.0 installed/external U |
| REQ-022 / AC-022 | REMOTE-R2; remote_auth.py, Store v12 | Fixture suite plus 0.24.0 DCR/PKCE/Bearer, replay/wrong-verifier denial and revocation | 0.24.0 loopback accepted; v0.25.0 public client U |
| REQ-023 / AC-023 | REMOTE-R2; opaque path capability | Fixture suite plus 0.24.0 opaque-link authorization, guessed/revoked link denial | Older loopback accepted; each live client/current version U |
| REQ-024 / AC-024 | REMOTE-R3/R4; CLI, remote_tls.py, Web RemoteMCP | Fixture suite plus 0.24.0 loopback TLS proxy OAuth and Caddy negative | Real public TLS/current external clients U |


## D3/D4 agent diagnostics (source implemented; installed test gate open)

| REQ / AC | Source implementation | T fixture coverage | U / required acceptance |
| --- | --- | --- | --- |
| REQ-025 / AC-025 | D3: `bwmini/gate_diagnostics.py`, `mcp_gate.py`, D2 app reader, `online.py`, installer helper manifest; [master](../design/DIAGNOSTICS_D3_D4_MASTER.md) | `tests/test_gate_diagnostics.py`, upstream passthrough, bounded journal/redaction, actor/trace selectors, retention and catalog; record exact CI run in [queue](REMEDIATION_TEST_QUEUE.md) | Authenticated live SSH/HTTP, installed app-user report, split-host and true Provider fault recovery not yet confirmed |
| REQ-026 / AC-026 | D4: exact Box incarnation root-owned allowlist, fixed guest probes, summarized output and privacy contract | Default DENY before Provider, policy/identity rejection, no raw stdout, synthetic Provider read | A specifically approved real Box with correct trust, negative unauthorized case and actual service/log read required; no deployment or mutation authority implied |

## Phase 1 unified operation base (owner authorized; source partial only)

v0.29.0: [source CI + publish PASS 41404753003](https://github.com/vanishs/boxweave/actions/runs/41404753003) on `160a09c4d2d2defff3adf3fe538861e28128b1a3`; immutable `v0.29.0` tag points to the same source. P1-A minimal resource claims and P1-B1 identity/private DB adapter are source/fixture **PASS**. Shared coordinator is **not yet provisioned/enabled**; no protected Web/SSH/HTTP path or installation acceptance. Stable installer remains pinned to v0.28.0.


[Delivery ledger](../process/PHASE1_UNIFIED_OPERATION_BASE.md) tracks P1-A source-only operation-claim primitives. The core adds no exposed MCP/Web mutation, no production SQLite migration, no task-result store and no deployed multi-Agent protection. Targeted source/fixture CI passed at `05a95aea` ([run 41402675634](https://github.com/vanishs/boxweave/actions/runs/41402675634)); the final merged SHA and installed/live behavior require separate evidence. P1-B/P1-C adapter coverage, host permissions, historical receipt migration and live cross-transport tests remain pending.

## Code-first source coverage and acceptance state (2026-11-01)

[The current code-derived implementation, endpoint, module, and test inventory](CODE_FIRST_ACCEPTANCE_2026-11-01.md) enumerates source surfaces even when missing from older feature prose, maps them to CI and known real evidence, and preserves PARTIAL/BLOCKED/UNVERIFIED for current runtime checks. An implemented endpoint is not itself accepted.

## Current 2026-11-01 code-first live acceptance supersession

The historical REQ/AC matrix above is not a blanket acceptance of current installed services. The [whole-project A01–A30 ledger](CODE_FIRST_LIVE_ACCEPTANCE_2026-11-01.md) crosswalks all source-backed features, includes implementation not described in earlier feature narratives (`boxweave job`, developer source bundling, MCP self-info, CLI report kind/instance failure semantics), records actual new-Box and historical-Box execution, and preserves **PASS/PARTIAL/FAIL/BLOCKED/UNKNOWN/UNVERIFIED** per exact layer. Real v0.26.1 install and upgrade evidence does not by itself certify an installed v0.26.2. Neither upstream Containerium Provider tools nor future E0–E4 proposal-only capabilities should be represented as BoxWeave product acceptance.

## Post-host-v0.26.2 installed-feature reacceptance (2026-11-01)

The [A01–A33 live reacceptance](REACCEPTANCE_0262_TO_0264_2026-11-01.md) covers the original source matrix and the newly discovered (A31) Incus `query` project-selector failure, (A32) Web HTTP 404/405 incorrectly reclassified as 500, and (A33) partial success of nesting-enabled Box creation with opaque client error. Source fixes for A31/A32 were published in v0.26.3/v0.26.4; A33 and outer-host nesting execution remain OPEN and must not be promoted to PASS from fixture or from guest-local helper evidence.

## v0.26.6 reverse-call-chain audit supplement (2026-11-01)

See [DEEP_AUDIT_0266_2026-11-01.md](DEEP_AUDIT_0266_2026-11-01.md). **A23 previously unverified selftest** now has a real new nesting-enabled Box `selftest plan → run → create → hello → stop → delete` **L PASS**: application receipt `state=passed`, four successful per-step receipts, independently zero inner instances after cleanup. **A34 newly identified implementation/test gap:** OAuth admin authorization `_rate` wrote a 5/minute attempt in the same `Store.db` transaction that rolled back on denied credentials, nullifying the intended rate limit for repeated denials. v0.26.6 commits rate attempt separately and adds deterministic 429/ledger/no-code/recovery fixture. **A35 observed diagnostic gap:** Invalid selftest confirmation exits 2 and creates zero guests, but host CLI returns generic `SELFTEST_FAILED_INSPECT_STEPS` rather than exact typed app refusal. Preserve as partial/P2, not a selftest unauthorized-mutation failure.

## v0.26.7 runtime selftest trust race and typed failures (A36)

[New scoped trust-readiness acceptance](SELFTEST_TRUST_RACE_0267_2026-11-01.md): **L FAIL v0.26.6** on two confirmed plans. Created inner instances but `hello:TRUST_FAILED`; stop/delete succeeded and independent list was empty. v0.26.7 retries **only** the fixed non-mutating hello on classified temporary trust-readiness errors within a bounded share of task budget; exact host keys remain strictly verified. The host CLI no longer discards typed app failure/refusal output just because app exits 2; nonallowlisted errors stay redacted. **T CI and L installed retest required** for PASS, alongside A35 typed-error usability closure.

## v0.26.7 scoped installed closure — A23/A35/A36, 2026-11-01

The prior v0.26.6 failed trust-readiness runs remain actual negative evidence. The published v0.26.7 positive test in [SELFTEST_TRUST_RACE_0267](SELFTEST_TRUST_RACE_0267_2026-11-01.md) now gives **L PASS** for an application-owned selftest including full created-child identity/hello/cleanup and a typed no-mutation invalid-confirmation refusal. A35 CLI diagnostic gap is closed **for the tested allowlisted confirmation error and failed-step response contract**; unknown external responses still require separate validation. A36 retry is complete for **one installed true selftest**; no universal claim for all networks/SSH transport timings. A34 OAuth rate limiter remains T/CI PASS with installed package, and public client authorization/TLS L/U. The task-owned parent Box and its nested child are deleted and absent in the final MCP inventory.

## ARCH-0 architecture consolidation — source/fixture checkpoint

[Gate and source audit](../design/ARCHITECTURE_CONSOLIDATION_GATE.md) establish six logical ownership blocks and [executed-route inventory](ARCH0_BOUNDARY_INVENTORY.md). ARCH-0A source map CI 41408681895 PASS; ARCH-0B Provider-scoped exact incarnation / standalone gateway helper packaging / private SQLite safety CI 41409106116 PASS; ARCH-0C pure Provider/access/lifecycle extraction with original public aliases CI 41409883731 PASS. **S/T=PASS** for source/fixtures only. **L/U** for installed SSH/HTTP/Web, cross-UID real storage permissions, mixed-version upgrade/rollback and mutation coordination. New source is v0.30.0 and default stable install remains v0.28.0 until separately promoted.

### ARCH-0 v0.30.0 source publication closure

**Source release PASS:** [main CI + immutable publish 41410649605](https://github.com/vanishs/boxweave/actions/runs/41410649605) at `06d1714021be9a0350982a02223ffefab9ba3305`; [v0.30.0](https://github.com/vanishs/boxweave/releases/tag/v0.30.0) tag targets the exact merged source commit and releases 16 assets, including `operation_coordination.py`, `coordination_runtime.py`, and the SHA256-verified `boxweave.zip`. Default `PUBLISHED_RELEASE` still v0.28.0; no automatic host promotion. **Installed L=U**.

## Phase 2 staged source acceptance — 2026-11-02

[Exact CI/Release evidence and outstanding gates](../verification/PHASE2_SOURCE_ACCEPTANCE_2026-11-02.md): v0.31.0 R-01(read-only)/R-03, v0.32.0 R-02 metadata-only task receipts, v0.33.0 R-04 **internal lease core only**. S/T/source release PASS for these individual slices; overall Phase 2 **PARTIAL**, since P1-B2 Web/SSH/HTTP shared dispatch, R-04 live tool/lease enforcement, guarded patch, and installed cross-entry recovery are NOT IMPLEMENTED or LIVE U. Stable PUBLISHED_RELEASE remains v0.28.0.


## Fresh v0.42.2 source and Box reacceptance (2026-11-02)

[Exact new-source/source-CI, real-Box, old-host comparisons and status ledger](DEEP_REACCEPTANCE_0422_2026-11-02.md) supersedes broad inferences from older version-specific entries. Python 640/640 and exact-head CI PASS; fresh v0.42.2 nested install + authenticated Provider read + real create/hello/stop/delete and post-restart doctor PASS. Actual cross-entry protected R-04 races, provisioned R-02 task recovery, D4 allowlisted positive, E2-E4, external OAuth/desktop, and newly installed old-to-new upgrade remain U/BLOCKED. This report records only observed evidence, not global acceptance.

## v0.42.3 R-01 validation correction (2026-11-02)

[Source-level R-01 correction and evidence](REACCEPTANCE_0423_R01_2026-11-02.md): read-only tools explicitly document home-relative paths and publicly return bounded, stable argument/path/guest-unverified codes instead of the misleading generic trust gate. New negative Gateway fixture proves no Provider dispatch for malformed selectors; 642 Python tests PASS. Installed 0.42.3 live classification is U pending exact tag/deployment, and prior R-04/R-02/D4/E2–E4 limitations remain unchanged.


### v0.42.3 installed R-01 diagnostic acceptance (2026-11-02)

[Installed and exact-release evidence](REACCEPTANCE_0423_R01_2026-11-02.md) supersedes the pre-publication L=U note above: pinned fresh install exit 0, 10/10 authenticated doctor PASS, configured standalone Gateway under app user returns BOX_PATH_INVALID for absolute/traversal paths and BOX_READ_INVALID for bad offset, with correct tool descriptions. R-01 P3 diagnostic behavior **T+L PASS in this exact scope**; prior no-symlink/guest bounded access guarantees remain guarded by source fixtures and earlier 0.42.2 live checks. Optional multi-Agent and Provider-unavailable roadmap outcomes remain PARTIAL/BLOCKED.


## Real host update to v0.42.3: Gateway R-01 reacceptance (2026-11-02)

[Exact host MCP read and denial evidence](HOST_0423_GATEWAY_REACCEPTANCE_2026-11-02.md) independently proves the upgraded host's `BOX_PATH_INVALID` classification, full binary window/EOF/SHA-256, symlink and intermediate symlink refusal, pagination, exact SSH host override denial and compatibility reads on three old Boxes. L/PASS scoped; test-owned files removed. This is not a host-local CLI/Web version probe or a complete authorization of D4/R-02/R-04. The D3 Gateway journal still classifies local known refusals as generic unknown (P3).


## D3 deterministic classification source repair — v0.42.4 (2026-11-02)

[Exact source and source-only acceptance](D3_CLASSIFICATION_FIX_0424_2026-11-02.md): known fixed local R-01 read/SSH target refusals may be journalled as failed with bounded reason code. Upstream JSON-RPC errors and potentially dispatched writes remain unknown. 644 Python, 4 architecture tests PASS in prepared Box; full installed upgrade/Gateway acceptance remains L=U until separately observed.


### D3 v0.42.4 actual installed upgrade acceptance (2026-11-02)

[Source and live differential acceptance](D3_CLASSIFICATION_LIVE_0424_2026-11-02.md): immutable v0.42.4 CI verify/publish PASS, real pinned 0.42.3 -> 0.42.4 disposable-Box upgrade exit 0, actual Gateway D3 fixed-code refusal **L PASS** and conservative `unknown` **L PASS**, authenticated Provider + doctor 10/10 PASS. Earlier source-only U for this scoped D3 P3 defect is closed, not unrelated R-02/R-04/D4 positive or external OAuth.


## Broad v0.42.6 exact-source and installed reacceptance (2026-11-02)

[Complete per-feature live/fixture/blocked ledger](BROAD_REACCEPTANCE_0426_2026-11-02.md) supersedes prior incomplete source-only claims: final v0.42.6 exact-release CI/PUBLISH PASS, 645 Python and 4 architecture tests PASS, in-place v0.42.4->v0.42.5->v0.42.6 real Box upgrade PASS, Provider/doctor 10/10, R-02 empty-output and stderr-only result repairs live PASS, stale CAS recorded completed/1 without overwrite, independent app/Gateway R-04 claim conflict, D4 allowed process/port, E0/E1 actual Provider plus authenticated test-client routes. Global cross-entry R-04, externally authenticated HTTP/SSH/OAuth clients, in-flight kill recovery, old-to-current fleet upgrade, real model inference and missing E2-E4 Provider contracts remain U/BLOCKED. A concurrent-create Provider error despite actual successful creations is retained as an OPEN uncertain-write issue. Original seven outer Boxes preserved; new incident Box STOPPED/retained; stable pin v0.42.1 unchanged.


### Direct v0.28.0 -> v0.42.6 installed migration closure (2026-11-02)

[Addendum to complete installed reacceptance](BROAD_REACCEPTANCE_0426_2026-11-02.md): an independently seeded v0.28.0 Box installed and upgraded directly to v0.42.6 with **both installer exit 0, both doctor --mcp 10/10 PASS**, exact source commits, actual live app SQLite **schema 12 -> 13** and original admin credential hash preserved without value disclosure. Two schema-12 backup files remain intact; they are not the live DB. Task-owned Box deleted after PASS; seven historical Boxes and separate stopped incident Box unchanged. v0.26.x direct migration and production downgrade/rollback remain U.


## Independent multi-client and D4 parser reacceptance — v0.42.7 candidate (2026-11-02)

[All remaining items and exact evidence](REACCEPTANCE_FULL_RETEST_0427_2026-11-02.md): 647 Python source regressions PASS; independent installed v0.42.6 Box test had R-02 two-client one-dispatch and reconnect PASS, R-04 protected lease and same-Box denial/different-Box success PASS, CAS successful/stale PASS. D4 real process/ports PASS; empty-journal bare Provider receipt and stderr-only service exit triggered a new parser ISSUE. Source v0.42.7 fixes bounded empty logs and classifies known numeric failure without raw stderr. Installed v0.42.7 L/U until a pinned upgrade. External OAuth/Web worker races, E2–E4, live model and intermittent upstream create errors stay partial/U/BLOCKED.


### v0.42.7 D4 real installed parser closure (2026-11-02)

[Full residual ledger and exact live v0.42.7 upgrade](REACCEPTANCE_FULL_RETEST_0427_2026-11-02.md): immutable v0.42.7 CI verify/publish PASS; fresh v0.42.6 -> v0.42.7 installed upgrade exit 0, doctor --mcp 10/10 PASS, real D4 zero-line logs now observed/0 and stderr-only nonzero is not_verified/PROJECT_COMMAND_EXIT_NONZERO without raw stderr. Root allowlist negative, process/ports and D3 invalid-path still pass; 3 nested test Boxes and outer test Box deleted, protected-policy marker restored absent. Global Web/SSH/HTTP cross-transport R-04, E2–E4, external OAuth, positive guest user service and model inference remain PARTIAL/U/BLOCKED.


## v0.42.8 D4 positive real service and upstream idle-stop — 2026-11-02

[Exact installed Gateway, two task-owned nested Boxes, upstream storage/API evidence and cleanup](BLOCKED_REACCEPTANCE_0428_2026-11-02.md): **D4 normal running systemd user service T+L PASS on v0.42.8** (`observed/active/0`, log line counts, correct inactive and unavailable classifications, default deny). Previous v0.42.7 missing-XDG user-bus behavior was reproduced before the fix. **Actual v0.120.1 auto-idle-stop L PASS** after a test-only stop/start established `LastStartedAt`; manual restart and opt-out PASS. Initial newly created Box has no activity/start stamp and was not auto-stopped, so universal cold-create sleep remains U; HTTP request wake remains BLOCKED by absent configured route/proxy. E2/E3/E4 are **not implemented**; upstream REST/CLI snapshots exist but the test Incus pool uses `dir` and current 58-tool MCP catalog has no snapshot/restore/clone. [Future 0.126.0 handoff](../design/CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md). v0.42.8 CI verify/publish and installed doctor 10/10 PASS; all task-owned resources removed and nine original Boxes preserved. Full product remains PARTIAL.


## Patched upstream v0.126.0 cold auto-sleep + first-request HTTP wake (2026-11-02)

[Exact isolated live evidence](CONTAINARIUM_0126_AUTOSLEEP_WAKE_LIVE_2026-11-02.md): fixed six-file [fork commit](https://github.com/vanishs/Containarium/commit/40b9d650584ceef66ea196ec6ee6dd9d88263f2f) compiled and installed **only inside an isolated nested test Box**; focused Go tests PASS, BoxWeave v0.42.8 installed doctor 10/10 PASS. Newly created Box with a 1-minute auto-sleep toggle **stamped LastStartedAt and actually auto-stopped without manual restart**, and a second client Box requested a sleeping HTTP target through real Caddy wake routing, received **first-request HTTP 200 + exact app response in 1.65s**, and target transitioned STOPPED→RUNNING. Untrusted direct wake request returned HTTP 403. Baseline 0.120.1 had both defects. Upstream PR creation was **403 forbidden**; this code is **NOT yet upstream merged/published, nor a BoxWeave runtime release**. No owner host upgrade; all new Boxes deleted, original nine preserved. E2/E3/E4 and public HTTPS remain U.

## Independent current main v0.43.0 reacceptance (2026-11-03)

[Source-/test-/live-graded A01–A14 matrix](REACCEPTANCE_0430_2026-11-03.md) supersedes an assumption that prior old-Box or fixture evidence automatically applies to current installed software. Fresh 699 tests, 7 real Chromium fixture flows, knowledge, newly installed v0.43.0 REST+MCP and DC-01 reads are PASS in their exact scopes; unresolved Provider create-after-commit, unprovisioned R-02 and untested R-04/global/external/E2–E4/upgrade gates remain separately classified U/BLOCKED.

## v0.43.1 post-commit lifecycle and Gateway error hardening (2026-11-03)

[One-row-per-gate source/live ledger](CREATE_OUTCOME_0431_2026-11-03.md). New tests reproduced a real source bug: post-dispatch MCP error falsely terminalized a created Box as failed. Repair makes it UNKNOWN, holds coordination claims, prevents replay and exposes a bounded Agent create-error follow-up. Pending main CI and isolated installed source-version verification must not be inferred from focused fixture PASS; upstream create-after-commit 500 cause remains open.

## v0.43.1 immutable release and isolated host gate (2026-11-03)

[Published and live acceptance ledger](STABLE_PROMOTION_0431_2026-11-03.md) records CI/tag and real 0.43.0→0.43.1 upgrade, installed doctor 10/10 and REST/MCP coordinator smoke. Default installer pin is an independent promotion; no production-host or upstream HTTP 500 repair is implied.

## Updated host v0.43.1 real reacceptance (2026-11-03)

[Per-feature current-grade matrix](POST_HOST_UPGRADE_REACCEPTANCE_0431_2026-11-03.md): live create/SSH/bounded reads/stop-start/persistence/resize/network ACL and historical 0.26.1 read/doctor PASS, source 703/703 + seven browsers PASS. Deployment-disabled task tracking, protected R-04, D4 positive, missing current client DC-01 catalog entry and unsupported untested storage/external client/nested child remain U/BLOCKED.

## v0.44.4 current-host independent source/live crosswalk (2026-11-04)

[Exact feature-level statuses and new upstream disk-shrink mismatch](REACCEPTANCE_0444_2026-11-04.md). **L FAIL** on upstream 0.128.0 shrink (25GB→24GB returned success contrary to grow-only contract). New Agent Gateway / REST / Web plan disk-size guard **S+T PASS**, installed v0.44.4 L pending. Existing current main 728 units/7 browsers PASS; task Box v0.44.3 install+doctor 10/10 and nested child execute+cleanup L PASS. R-02 unprovisioned; real global R-04, external OAuth/native clients and E2–E4 remain U/BLOCKED. Original operator host untouched.
