# Source register

Reviewed legacy repository: `C:/Users/vanis/Downloads/boxweave`, ref `refs/heads/main`, commit `4318da6ea4e4e45c55cf43b64acbdd2ede65fc43` (H). Unless explicitly stated otherwise, tracked paths below refer to H and can be recovered with `git -C <legacy> show H:path`. Path groups are scoped evidence sets, not assertions that every line was audited. At phase 1 no remote website, upstream branch or PR was freshly verified; upstream statements in SRC-001 through SRC-024 are recorded claims. Phase 3 adds official documentation reads in SRC-027/028, not fresh upstream deployment/PR verification.

Confidence: A = direct source/static observation; B = historical report with environment/revision limits; C = proposal/inference. Neither A nor B implies fresh live acceptance.

| ID | Evidence paths/version | Kind/confidence and scope |
| --- | --- | --- |
| SRC-001 | README.md; providers/containarium/README.md at H | A: positioning, flows and declared support/non-goals; provider README inventoried, README read |
| SRC-002 | install.sh; docs/INSTALL-UPDATE.md; tests/test_installer_rerun.py at H | A: contract/docs read and test scenarios inspected; installer is not executed or fully audited |
| SRC-003 | docs/ARCHITECTURE.md; docs/CLIENTS.md; agents/README.md at H | A: architecture/adapter roles; architecture and adapter overview read, CLIENTS indexed as follow-up contract |
| SRC-004 | clients/posix/setup.py:probe/setup/toml_update; clients/windows/setup.ps1; server/clients.py; tests/test_diagnostics.py; tests/test_windows_config.py at H | A: generator and POSIX probe read; Windows source/test inventory, not complete Windows audit |
| SRC-005 | docs/CLI.md; server/cli.py:installation_state/inspect_box; tests/test_readiness_behavior.py at H | A: docs and function inventory; readiness contract, exact local name inspection |
| SRC-006 | server/webui/{app,store,graph,command,provider}.py; web/src; tests/test_web.py; web/browser at H | A: route/function inventory and selected implementation; frontend/test file inventory, not all UI exercised |
| SRC-007 | server/webui/{tenants,access}.py; tests/test_tenants.py; tests/test_tenant_network.py at H | A: capability/creation/access snapshot code and named regression scenarios inspected |
| SRC-008 | docs/SECURITY.md; server/hostkeys.py; server/tokens.py; tests/test_hostkeys.py at H | A: security doc and host-key selection/write code read; token/tests inventoried |
| SRC-009 | docs/TASKS-AND-SELFTEST.md; server/{jobs,smoke,mcp_client,provenance}.py; tests/{test_jobs,test_smoke}.py at H | A: contract and MCP client read; worker/smoke/test function inventory |
| SRC-010 | docs/WEB-CAPABILITIES.md; patches/containarium/README.md; docs/TENANT-UPSTREAM.md at H | A: capability and pinned-source profile read. Records upstream d4aad1a8a8cd748ab6211c67a3417f173316f9e1 and native-network-tenant/v1; no current upstream verification |
| SRC-011 | docs/TEST-BOXWEAVE-LXC.md at H | B: 2026-10-22 actual-run report, original source 9956e81 plus separately patched installer; not a current run |
| SRC-012 | Local git log at H, 32 commits 40a22d9..4318da6 | A: complete subject timeline reviewed, selected thematic histories; no claim every diff was read |
| SRC-013 | VERSION; web/package.json; docs/CLI.md; agents/README.md at H | A: 0.4.3 product vs 0.3.0 Web/adapter labels and older CLI prose |
| SRC-014 | docs/CLIENT-DIAGNOSTICS-READINESS.md; docs/DEVELOPMENT-READINESS-2026-10-22.md at H | B/C: historical diagnostics fixtures/proposals mixed with shipped updates; current POSIX probe cross-checked |
| SRC-015 | docs/WEB-TESTING.md; docs/RESEARCH-INTEGRATION.md at H | B: historical Web/source tests and explicit integration limits; research-to-product synthesis |
| SRC-016 | outputs/boxweave-next-stages/knowledge-base/{README,knowledge-gap-assessment,recursive-mcp-design}.md; outputs/boxweave-next-stages/current-state-reconciliation-v2.md | B/C: ignored, uncommitted 2026-10-21/22 knowledge/reconciliation; selected read, no full bundle audit |
| SRC-017 | outputs/boxweave-next-stages/7g1-L3-boundary/stage-report.md | B: ignored historical boundary and interruption recovery report; not current state |
| SRC-018 | outputs/pr3165-retest/report.md | B: ignored report at e26f5cf371aafbb952a83191b596deec1df0fc2a; reviewer-added regressions failed despite existing suite passing |
| SRC-019 | outputs/pr3165-fix/result.md; outputs/pr3165-main-sync/review-request.md; outputs/tenant-network-recovery.md | B: ignored later fix/main-sync and earlier downstream handoff; don't transfer any old authorization |
| SRC-020 | Uncommitted server/clients.py diff: 17 additions/5 deletions | A/C: reviewed current WIP, not H; potential replacement answer propagation defect; preserve unchanged |
| SRC-021 | BoxWeave MCP list_containers({}), 2026-10-28 | Fresh read-only observation: successful tool result, four Running resources. Proves inventory call only; no connect, installed versions or readiness claim |
| SRC-022 | User-supplied 00_MASTER_PLAN.md and 01_UNDERSTAND_AND_SEED_KNOWLEDGE.md | Authoritative for this phase because user explicitly requested execution. Master copied verbatim to process/MASTER_PLAN.md; phase prompt to process/PHASE_01_INSTRUCTIONS.md |
| SRC-023 | User request to execute phase 2; supplied 02_RETHINK_PRODUCT_AND_REQUIREMENTS.md, 2026-10-28 | Current process authority: documentation-only product analysis and explicit developer gate. Copied verbatim to process/PHASE_02_INSTRUCTIONS.md; SHA256 71ad3782f41892881060ab6abcea8779f325839686b042526e74fe149cec0857. Does not approve proposed product choices |
| SRC-024 | Phase 2 entry: local Git ref/status/worktree/hash checks and fresh BoxWeave MCP list_containers({}), 2026-10-28 | Fresh read-only evidence: rewrite/v2 clean at 58640ce939028ae37cfd6535bbf0702a84d4969a; main still H; same two legacy dirty entries and SRC-020 hash; inventory succeeded with four Running boxes. No execution/readiness/capability verification. Selected mutable research source hashes rechecked unchanged; no wholesale ignored-tree audit |
| SRC-025 | User request to execute phase 3; supplied 03_DESIGN_ARCHITECTURE_AND_PLAN.md | Process authority: architecture/ADR/task/test-strategy documents only, prerequisite product approval and stop before phase 4. Verbatim copy at process/PHASE_03_INSTRUCTIONS.md |
| SRC-026 | Developer five-point product approval in this chat, 2026-10-28 | Personal remote Agent development + binding; Windows native SSH and 'WSL SSL' interpreted contextually as WSL SSH; Codex and ZCode(GLM); basic Web, Tenant network writes, port publication, Prompt Graph and model wizard included; recursive deployment deferred; one trusted operator, trust failure stops, unknown not retried, bounded recovery. Supersedes phase 2 suggested deferrals; DEC-001..005 resolved. Basic operation details are phase 3 elaborations, not separately verified user research |
| SRC-027 | Official ZCode MCP and model documentation, read 2026-10-28: https://zcode.z.ai/en/docs/mcp-services and https://zcode.z.ai/en/docs/configuration | A: native MCP user config mcp.servers and separate native model/sign-in settings. Does not prove installed version compatibility or automation API; no model sign-in/inference tested |
| SRC-028 | Official Flask Waitress https://flask.palletsprojects.com/en/stable/deploying/waitress/ and SQLite https://sqlite.org/wal.html, read 2026-10-28 | A: Waitress buffered requests/multithreaded process and local bind guidance; SQLite single-writer/concurrency and WAL constraints. Used for architecture tradeoffs, not a performance/release/security certification |
| SRC-029 | Phase 3 Git/ref/status/client hash + BoxWeave MCP list_containers, 2026-10-28 | Fresh read-only: new worktree clean at 130e88429b3aed3e4a8dc8f37d5cd34ef5cae236; legacy main H and original dirty entries/client hash unchanged; successful inventory of four Running resources. No connect, readiness or provider capability claim |
| SRC-030 | User request for phase 4, supplied 04_BUILD_MINIMAL_VERTICAL_SLICE.md, and explicit approval of PHASE_04_EXECUTION_PLAN | Authorizes isolated development/tests in boxweave-tenant-dev, temporary owned processes and fixed identity/hello within stated limits; no new credentials, provider changes or phase 5 |
| SRC-031 | Phase 4 actual remote/local verification, 2026-10-28; see VERTICAL_SLICE and phase4-source-hashes.json | Non-root Ubuntu LXC implementation, 29 tests and clean locked reproduction, frontend build/Chromium fixture, actual Windows preflight and direct Platform MCP identity/hello. Distinct evidence classes; no app-to-live MCP acceptance. Remote legacy checkout stayed at 7818668 with its preexisting dirty state |
| SRC-032 | User reply: no independent application launcher currently available; preserve results | Confirmed external prerequisite remains unavailable. Preserve tested subset, do not mint/borrow credentials or declare the phase-4 gate satisfied |
| SRC-033 | User request to create a dedicated test container, deploy new BoxWeave and/or make it accessible for testing | Authorizes dedicated test deployment/persistent application services; exact unfiltered host-port mapping remains pending explicit scope confirmation. Does not authorize host SSH/firewall repairs or claiming fixture as real MCP |
| SRC-034 | Fresh create/get/connect/dependency logs/service states/TLS and application smoke tests; see TEST_DEPLOYMENT | New boxweave-v2-test at 10.206.30.6, verified identity, aa0857c source archive, actual installation exit files, three active/enabled services, verified TLS and fixture binding/receipt restart. Client endpoint not exposed/verified; app live MCP remains incomplete |
| SRC-035 | Official Caddy TLS/automatic HTTPS/global default_sni documentation read 2026-10-28 | Explains local CA trust and IP/SNI configuration. No global/client trust was installed; no production security claim |
| SRC-036 | User explicit approval of host TCP 18446 -> 10.206.30.6:18446 passthrough | Authorizes the reviewed unfiltered HTTPS test exposure, not firewall/SSH changes or a claim of LAN-only filtering |
| SRC-037 | Fresh container identity/service/conflict checks, route add/list, Windows native CA-verified HTTPS/login/resources/receipt test | Active exact route; actual client check exit 0, homepage 200, unauthenticated API 401, login/Secure cookie/demo resource/fixture receipt passed. No TLS bypass/global trust installation, no real application MCP acceptance |
| SRC-038 | User request and confirmation of isolated shared-Tenant application/MCP/nested hello lab | Authorizes new dedicated lab resources and isolated service setup, preserving existing demo/old resources; no host-wide upgrade/credential extraction or new public exposure |
| SRC-039 | Authenticated capability/Guard/inventory, one explicit shared create rejection and original-name reconciliation | Legacy boxweave.tenant-network/v1 advertised true flags, but bw-v2-lab-app creation rejected. No lab box observed; known exact missing-instance 500. No duplicate/fallback create. Live contract differs from approved source profile; exact running MCP binary unverified |
| SRC-040 | Read-only pinned official upstream client/network_tenant.go and pkg/core/incus/network_tenant.go, plus local historical downstream client source | Confirms exact upstream native-network-tenant/v1 requirement and matching error text versus old downstream contract. Supports likely mismatch diagnosis, not actual installed-binary attestation |
| SRC-041 | User asks to mark the current shared-Tenant/live MCP test temporarily untested, skip it, assess phase 5 readiness and not execute phase 5 | Explicit test deferral and stop instruction; preserve observed create failure, do not claim pass/whole-stage acceptance or start feature work |

## Hashes for selected mutable local sources

SHA256 is a snapshot identity, not runtime validation. Ignored material remains outside commits; only conclusions and these references are migrated.

| Local source | SHA256 |
| --- | --- |
| SRC-020 server/clients.py | 6653b809eef737474b3804543c4ef7073deaffd5527219117a89679ea169e95d |
| SRC-016 knowledge-gap-assessment.md | dd83549b5e113df5c72445c3662d97b92dd1426b04567ccaf01d92d752cf7f05 |
| SRC-016 current-state-reconciliation-v2.md | b50cf91b1302a206a157adf9faf0e36be6ffe45218e95a1b15876459dcafd622 |
| SRC-017 stage-report.md | 68974c1d32c20fb2159ff86cab22c227a73370e3296f44adf07522193a75cfb4 |
| SRC-018 report.md | 5167f64dfc8385a595411a3632dfa8b7846d0f605dce9825d113b4b0e50b3989 |
| SRC-019 result.md | e7a7d1fcae9b74d92737b1e9cadcff45966a3a5668e09dace7b19439789f7b50 |
| SRC-019 tenant-network-recovery.md | 98df0a354d3281df814daeb004e4470d7a001b9913202a9fc4b0e595d2f6fbcc |

Excluded: untracked containarium-candidate.zip content; credentials/private keys/JWTs; raw experimental logs, data stores, copied upstream trees and nested source bundles. Their contents were not bulk-read or migrated. Filename-only discovery of ignored Markdown is not a full audit or privacy approval for other files.

SRC-042 — 2026-10-28: User explicitly requested another shared-Tenant attempt. Fresh authenticated discovery, one original-name create retry and post-failure reconciliation reproduced the capability rejection; no lab resource observed. No phase 5 execution or host repair authorized by this retry.

SRC-043 — User authorized existing/new boxes. Existing box1/box2 native shared membership and actual identities checked through MCP; supporting backend install submitted once in box2 under timeout. Subsequent MCP transport timeouts leave installation result unknown; not a live integration pass.

SRC-044 — Requested three existing boxes in one Tenant then box1 testing. Fresh authenticated inventory/capability and connect checks recovered; exact existing membership unchanged; reassignment unsupported by exposed tools and source. Original box2 install exit reconciled to 124 with staging-directory error; incomplete.

SRC-045 — Explicit new-box/same-Tenant request; actual creation of bw-v2-mcp-test with boxweave-v2-test membership, identity exit 0, matching attached Guard ACL and app-to-new-box TCP22 banner verified. Supporting backend installation pending; real MCP acceptance pending.

SRC-045 outcome: installer total timeout recorded exit 124; APT/dpkg children still active at selected observation, launcher absent. Original demo three services active and loopback HTTP200. No live MCP/application pass or inner workload creation.

SRC-046 — Read-only follow-up confirms installer stopped: exit124, no installer/package processes in snapshot, launcher absent, missing staging directory error. Containarium daemon still running; no repair performed.

SRC-047 — Developer requested retry without installation timeout. Actual retry exit0 and managed runtime health observed; strict SSH MCP handshake/discovery101/read-only tools-call passed from v2 app. One inner create rejected, original name absent. Inner core PostgreSQL fatal proc-mount permission failure observed read-only. No core/host fix or phase5 work.

SRC-048 — Phase5 request with boxes/30-minute prep allowance; explicit follow-up chooses real-chain completion before phase5, no live-test waiver. Read-only provider source associates Podman with nesting and policy-dependent privileged/unconfined settings; specific new-box approval pending, no create submitted.

SRC-049 — Explicit approval for new bw-v2-mcp-nested (2 CPU / 4 GB / 30 GB, same network Tenant), Podman/nesting with possible privileged/unconfined effects disclosed; prepare up to 30 minutes, no existing-box/host setting changes or new public port. Original-name reconciliation after create timeout confirmed one new instance, identity exit0.

SRC-050 — Actual new-backend install exit0, managed timers/oneshots/marker checks; real independent MCP 0.114.2 initialize/initialized/tools-list101/read-only call, one inner create and identity/Python exit0. Initial HTTP run exposed queue-time sampling race; regression failed before remote-authored 0.1.1 fix then all30 tests passed. Isolated fixed-candidate HTTP and Chromium live_mcp receipt/restart/no-replay and trust/identity failure gates passed. No phase5 feature/whole-stage acceptance, original demo unchanged. Sanitized evidence and exact hashes: verification/LIVE_MCP_CHAIN.md and nested-live-results.json.

SRC-051 — Developer explicitly confirms acceptance of the verified minimum architecture/slice and requests TASK-011. Stage5 starts for this one Prompt Graph slice only. Real Windows/WSL registration and release evidence remain future requirements.

SRC-052 — TASK-011 remote-authored 0.2.0 implementation and actual non-root project-LXC evidence: 41 tests, TypeScript/Vite, final Chromium graph and original fixture-slice regression all passed. SQLite schema2 migration preserves old admin/project/session/receipt; conflict/clipboard/redaction/non-execution/deletion verified. Exact13-file source hashes and sanitized browser evidence under verification. No deployment, new box, infrastructure action or push.
