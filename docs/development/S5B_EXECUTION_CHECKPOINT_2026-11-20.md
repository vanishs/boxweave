# S5B — RC9 acceptance checkpoint (2026-11-20 PT)

## Scope and authority

Owner directed RC9 publication, installed the upgrade on L0, and requested a dedicated S5B acceptance PR. Use this branch as the sole S5B evidence/fix branch. The canonical [S5 phase card](phases/05_RELEASE_ACCEPTANCE.md) and [A checkpoint](S5A_EXECUTION_CHECKPOINT_2026-11-20.md) govern required gates. RC9 is a development prerelease and early B evidence baseline, NOT a formally feature-frozen or release-qualified candidate. A1 #151 historical witness, real A3 trusted-client approval and approval-dependent A4 production compensation remain open; A6 tooling has Box evidence, but full A6 freeze is blocked. Continue independent B work without pretending these exits are signed. Do not expand product scope, enable unverified approval, bypass claims, publish stable or change the stable pointer. No automatic merge, release, host upgrade or rollback.

## Exact initial baseline

- Protected v1 / RC9 source: `17ab57959bee96ed7c4ee80655b4cc185c66d568`; PR #163 merged. [Release v1.0.0-rc.9](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.9); [release CI](https://github.com/vanishs/boxweave/actions/runs/44845819683) SUCCESS.
- Installed L0: version 1.0.0-rc.9, amd64 binary SHA256 `734cd1290648bb4ec7f996ffd80d806d51d2773722ab22738d0126b17fa0ea23`, Schema5, Go1.26.9, Incus7.6, restricted Project user-1001. Owner's installer reports active=true, restart_required=false. Actual authenticated host_status tracking `1410aa765fe1d7cea2640e2418d70112` confirms exact identity, ready=true and mcp_ready=true. A5 attention present. Current connector exposes all 23 tools including operation_recover. No reconnect or OAuth replacement required in this observed session.
- Existing inventory after upgrade: `11bbc73e-2bce-4cf9-857f-799f768305f8` development Box running, `645deac6-4e96-45fc-9bbb-2723c897ebce` nested lab stopped, `e1706830-aa0f-4cb0-a772-93d13e130790` install lab stopped. Preserve existing work and RC8 rollback backup; no install-commit. Use separately named disposable test resources for new cases. Shared existing dev Box may be used for ordinary source tests without deleting earlier work.
- Activity issue snapshot empty but bounded, not a full native all-clear. Host quota attestation unverified; PRS runtime/endpoint not independently checked after upgrade. Host-specific actions require separate owner authorization. No secret values in chat/evidence.

## Work queue

| Gate | Status | Next evidence |
|---|---|---|
| B1 installed interfaces, CLI, install compatibility and provenance | NOT_STARTED (initial read-only preflight above) | enumerate 23 actual MCP tools, installed exact identity, private/read/negative contracts, full Tool/CLI crosswalk; isolated install/upgrade/uninstall/rollback; inspect PRS separately |
| B2 integrated S1–S4 features | NOT_STARTED | real lifecycle, File/Git, archives, readiness/detail, Peer/Forward, exact dependencies, flat network and bounded stress |
| B3 fault/concurrency/restart | NOT_STARTED | #151 repeat and classify, native ACK/SQLite/PRS faults, retained claims, no duplicate effects, archive pressure |
| B4 actual ChatGPT/OAuth/Agent handoff | NOT_STARTED | #148 grant identity, refresh/revoke, external Git handoff; inert actual-client A3 signature probe; never treat UI consent or synthetic signing as proof |
| B5 independent host/physical | NOT_STARTED | quota, physical LAN/public IPv6, host/power assurance, installed rollback/uninstall, supported nesting (not #150 hostile deep recursion) |
| B6 independent signoff | NOT_STARTED | negative controls, TC-019 crosswalk, all mandatory evidence, exact final candidate and signed release gate |

## Execution rules

Start by reading AGENTS.md, canonical phase card, latest A checkpoint and this file; recheck remote HEAD, installed live identity, existing work and current CI. Wait for the owner to request the next gate. For requested tests use rapid Box-first iteration, actual installed RC9 MCP where appropriate, uniquely labeled work_ref/agent_ref and idempotency keys, check terminal receipt and guest exit; preserve UNKNOWN instead of replay. Use authorized disposable owned resources, exact native/readback evidence, safe cleanup only. If blocked, classify and continue independent tests. Do not weaken security to enable nesting; record simulated versus actual native versus physical versus actual ChatGPT evidence accurately. Host commands and destructive maintenance, power/network disruptions, sensitive configuration or public endpoint changes require explicit owner approval; prepare instructions if needed. Do not use simulated results to sign a missing native/physical/client witness.

Record each case with exact candidate source/binary, Box/Operation/native IDs, scope, original receipts, result/exit, evidence link, residual risk and next safe action. Add tests and actual defect fixes on this single branch when the owner requests work. A fix changing runtime requires a new candidate and affected regressions; RC9 evidence is not automatically final. Keep Draft until explicit merge authorization. Do not claim S5B_PASS or final release qualification before mandatory A closure and B evidence on the exact accepted frozen candidate.
