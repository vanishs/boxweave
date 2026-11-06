# v0.49.1 source candidate: release-gate audit and administrator handoff

**State: PARTIAL, not released, not independently installed accepted.** This report records the current source changes against main `e3d745a19badc48e2df116de315807b51815348a`. Source v0.49.1 requires a new main CI candidate after integration; the existing v0.49.0 artifact must not be relabelled. Stable installation selection stays v0.43.1.

## Confirmed release defects and implemented source corrections

| Finding | Source correction | Evidence boundary |
| --- | --- | --- |
| Verifier run origin was unchecked | Authenticate repository, exact main SHA, workflow path, event, success, distinct run, artifact identity and both run attempts | Adversarial source tests; protected live workflow not exercised |
| Signed approval bound only the app ZIP | Bind exact release manifest and complete 26-asset index, including installer and checksum file; compare helpers with embedded app modules | Original main artifact bytes checked; no installation implied |
| Web-plan scenario silently disappeared | Correct multiword scenario ID parsing; account for all 20 canonical scenarios, with an explicit uncompiled-graph blocker | Source regression; no Web-plan behavior acceptance |
| Signed metadata-only cases could satisfy acceptance | Require original process evidence, native oracle, installed proof kind, named complete mandatory graphs; reject PARTIAL/FAIL/fixture evidence | Known-bad source calibration; cryptography does not prove a dishonest producer's observations |
| Caller could weaken required scenarios | Refuse any required-set override differing from mandatory policy | Source regression |
| Publication re-downloaded unchecked bytes | Recheck trusted workflow context, complete asset bytes and signature in the publishing job | Source workflow review; live publication blocked |
| Existing release assets could be modified before parity | Published releases get read-only parity checks; only drafts receive missing uploads; downloaded draft bytes must match before publishing, public bytes checked afterward | Independently discovered source defect; workflow syntax checked; live deny/success test outstanding |
| Workflow rerun could accidentally repeat installation | Independent worker has a fixed durable original-run path, never an attempt-specific reset path | No-replay source tests; no reset/unlock implementation |

The gate now accounts for all 20 canonical scenario IDs. Existing mandatory scenarios remain mandatory. Any applicable non-PASS denies approval; an unsupported optional capability needs hash-bound independent native capability evidence and remains explicitly excluded from full system acceptance. A missing scenario cannot silently disappear.

New modules remain under `tools/reacceptance`; ordinary customer ZIP and root helper allowlists are unchanged. No signing keys, fixture approvals, native mutators or background services were added to the product.

## Exact original main candidate audited without execution

Authenticated GitHub API observations confirmed successful main-push CI run `42170666520`, attempt 1, artifact `13480599237`. The new fetcher downloaded the original wrapper, verified its SHA-256 before allowlisted extraction, and verified all asset/checksum bytes and helper-to-ZIP equality.

| Identity | SHA-256 |
| --- | --- |
| Original artifact wrapper | `ec5843da89fad359e68b060169e954e045801b08905a43e94eefc0a580e5824b` |
| v0.49.0 app ZIP | `61bd09be590df0eff6f4e24d52b7cdefc1ca627ca6d2551c26e01a5764d23487` |
| Release manifest | `56c0851d35fc1db5cd22eb4b35c765ee9f7e1734816b1ef1e052a06230a14a74` |

Embedded provenance was exactly `{commit: e3d745a19badc48e2df116de315807b51815348a, state: clean}`, version 0.49.0. This is **artifact identity PASS**, not installed acceptance. Downloaded bytes remain in the developer `/tmp` audit workspace; no customer code was executed.

## Independent workflow: executable limited lane, intentionally unsuccessful acceptance

`.github/workflows/independent-verifier.yml` accepts only a successful original main-push candidate at its exact source. It downloads the original artifact wrapper through authenticated GitHub APIs and verifies its published digest before extraction. It requires a separately provisioned ephemeral self-hosted runner in an authorized disposable Box. The worker validates an administrator-owned identity/expiry grant, stages root-owned exact assets, durably records intent before installation, invokes the reviewed private installer once, and checks installed host/app provenance and authenticated doctor results.

**The worker currently exits 2 with PARTIAL after those checks.** It does not execute all required stateful scenarios, sign, emit release approval or upload the accepted release-evidence artifact name. A passing doctor is not release acceptance. No workflow signer has been wired: doing so before genuine complete scenario production would create an unsafe appearance of completion. The unsigned `signing_preflight` CLI is implemented and tested; it rejects incomplete evidence and cannot access a key or publish.

## Minimal administrator setup (owner will arrange provisioning)

Authenticated repository API inspection returned no environments and only the existing CI workflow. No environment, trust key, runner registration, host policy or credential was created by this change.

1. Provision a **fresh disposable Box**, distinct from Finder and all original UNKNOWN hosts. Record outer Provider/backend, UUID, createdAt, OS and non-root runner account. Prepare the documented BoxWeave nested-host prerequisites separately; an Actions runner label does not create Incus or authorize an upstream installation. Never assign this label to the owner host.
2. Register an **ephemeral**, repository-scoped Linux x64 self-hosted runner labelled `boxweave-independent-verifier`, with no signer key or production credential. Restrict access to reviewed main-only workflows and the protected Verifier environment. The administrator must review the root execution of repository worker code in this exact disposable Box; no general production sudo grant is implied.
3. Create root-owned, non-group/world-writable `/var/lib/boxweave-reacceptance` and a private root-owned `/etc/boxweave-reacceptance/install-grant.json`. The grant uses scope `test_owned_private_candidate_install_only`, `approved: true`, exact `hostname`, `/etc/machine-id`, `outer_uuid`, numeric-string `outer_created_at`, and `expires_epoch` less than 24 hours away. Supply values from independent target identity observation. Do not include tokens, SSH keys or signing material. The fixed `original-worker` evidence directory must initially be absent; once created it is retained. Never delete it to rerun UNKNOWN installation.
4. Configure `boxweave-independent-verifier` and `boxweave-reacceptance-release` environments with main-only deployment rules, an independent required reviewer, self-review prevented, and administrator bypass disabled where the GitHub plan permits. Missing plan support is a blocker requiring an equivalent reviewed trust boundary, not permission to weaken checks.
5. Select an independent signing custodian and protected signing environment **separate from the test runner**. Keep its private key out of chat, repository, test Box and general CI. The existing gate expects an OpenSSL SHA-256-verifiable public key; the release environment public variable is `REACCEPTANCE_VERIFIER_PUBLIC_KEY`. Configure it only after the signing custodian, complete real evidence producer and protected signing workflow have been reviewed. A generated fixture key is not a release identity.

The owner has stated that administrator provisioning will be arranged. This does not establish that any of the above is configured or accepted. Do not dispatch the new install workflow until target setup and exact authorization are independently checked.

## Requirement status after source audit

| Requirement | State | Actual capability and remaining acceptance |
| --- | --- | --- |
| RACC-SYS-01 off-host packaging | PARTIAL | Production exclusion exists and source guards cover it; final new built-product install/upgrade unverified |
| RACC-SYS-02 observed upstream resolver | PARTIAL | Pinned resolver/cache exists; incident independently pinned running daemon and MCP asset identity; full authenticated route compatibility incomplete |
| RACC-SYS-03 scenario planning | PARTIAL | Risk/graph planner exists; it is not execution and not all graph nodes represent complete user journeys |
| RACC-SYS-04 every discovered action | PARTIAL | Installed read-only positive/invalid calls and accounting exist; mutation/denial/SSH/HTTP/all CLI leaves incomplete |
| RACC-SYS-05 graph/race/failure execution | PARTIAL | Durable once-only primitive and four-step lifecycle exist; full graphs, deterministic cross-interface actors and recovery drivers incomplete |
| RACC-SYS-06 real role orchestration | PARTIAL | Distinct Agents performed discovery, implementation review and source verification in this task; product runner still emits manual handoffs; no automatic clean installed Verifier orchestration claim |
| RACC-SYS-07 exact candidate release | PARTIAL | Stronger source/signature/asset/context gates, fetcher and limited separate install workflow implemented; protected infrastructure, signer, full real proofs and live gate acceptance BLOCKED |
| RACC-SYS-08 report/issue synchronization | PARTIAL | Per-requirement and scenario blockers recorded; Issue #88 receives a sanitized update; full discovered feature/action report still incomplete |
| RACC-SYS-09 calibration/security limits | PARTIAL | Forged/partial proofs, altered bytes, wrong attempts, unsafe ZIP members and no-replay source controls tested; live fault campaign and performance bounds incomplete |

## Mandatory scenario results for this changed candidate

| Scenario | State | Reason |
| --- | --- | --- |
| SYS-LIFE-01 / SYS-LIFE-02 | BLOCKED | v0.49.1 not installed; prior v0.49.0 four-step lifecycle is historical PARTIAL, not full graph |
| SYS-JOB-01 / SYS-COORD-01 | BLOCKED | No new exact-candidate two-actor task/coordination native effect test |
| SYS-UNKNOWN-01 | UNKNOWN | Original incident remains frozen; new candidate controlled response-loss scenario not run |
| SYS-SELFTEST-01 | BLOCKED | No new candidate two-actor in-flight-barrier installed test |
| SYS-UPGRADE-01 | BLOCKED | No new main artifact or independent baseline-to-candidate upgrade/rollback |
| SYS-UPSTREAM-01 | PARTIAL | Actual incident runtime and pinned source observed; complete installed cross-route compatibility not tested |
| SYS-COMPOSE-01 / SYS-AUTH-01 / SYS-SSH-01 / SYS-REST-01 | BLOCKED | No new candidate functional positive/invalid/denied cross-interface tests |
| SYS-WEB-PLAN-01 / SYS-NEST-01 / SYS-NET-01 / SYS-SEC-01 / SYS-BACKUP-01 / SYS-DIFF-01 | BLOCKED | Required independent candidate capability, grants and behavioral evidence absent |
| SYS-REPORT-01 | PARTIAL | Known-bad source calibration passed; full independent installed reporter test incomplete |
| SYS-RELEASE-01 | PARTIAL | Original candidate identity verified and source deny controls tested; real independent signature, gate deny/success rehearsal and public parity not exercised |

See [Issue #88 latency evidence](RACC_0491_CREATE_LATENCY_2026-11-05.md). No original uncertain target was replayed or modified. Owner-host reboot/R-04 activation, Provider upgrades, SSH trust changes and stable pin changes remain outside this work.

## Verification and next executable checks

Local `.runtime/reacceptance-0491/action-ledger.md` accounts for 68 outer ChatGPT tools and 167 source action entries, all explicitly unaccepted for this changed candidate. The outer catalog is not the installed SSH/HTTP catalog; source inventory is not an invocation. The planner now reports all 20 named scenarios. No new-candidate behavioral action was executed.

Independent source verification ran 44 focused tests and reproduced the published-release immutability case with mocked GitHub: an existing published release with missing assets now exits 1 with zero uploads, edits, creates or API writes. This is independent source calibration, not a live public-release test.

Focused source tests, architecture checks and knowledge checks are recorded in [the verification queue](REMEDIATION_TEST_QUEUE.md). Initial bare Python 3.12 execution lacked Werkzeug; that environment failure is retained rather than reported as a product regression. An isolated `/tmp/boxweave-0491-venv` was prepared from hash-pinned `requirements-dev.lock` for the complete source regression lane. These remain developer source tests on WSL1, not prepared-LXC deployment acceptance.

Next: integrate reviewed source as a new commit, obtain its exact successful main candidate, finish missing scenario drivers, complete administrator setup, run independent install and original repro without touching preserved UNKNOWN targets, then validate a protected signer and both denied/approved official workflow paths. Publish only after all mandatory gates pass; independently download and compare public assets to the signed candidate. Keep the stable selection unchanged.
