# Feature/evidence matrix — historical phase6 scope plus 0.13.0 remediation obligations

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
