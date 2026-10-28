# Source register

Reviewed legacy repository: `C:/Users/vanis/Downloads/boxweave`, ref `refs/heads/main`, commit `4318da6ea4e4e45c55cf43b64acbdd2ede65fc43` (H). Unless explicitly stated otherwise, tracked paths below refer to H and can be recovered with `git -C <legacy> show H:path`. Path groups are scoped evidence sets, not assertions that every line was audited. No remote website, upstream branch or PR was freshly verified; upstream statements below are repository-recorded claims.

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
