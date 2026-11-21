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


## B0 owner-directed preflight acceptance — 2026-11-20 PT

Owner requested the residual A1/A3/A4/A6 and physical Host/PRS checks first, without changing canonical A/B gates. **B0 PARTIAL**; B1–B6 not signed. Live installed RC9 source `17ab57959bee96ed7c4ee80655b4cc185c66d568`, binary SHA256 `734cd1290648bb4ec7f996ffd80d806d51d2773722ab22738d0126b17fa0ea23`, Schema5, Incus7.6. Host-status start tracking `27070998b70ff593fd26bde25075c5fb`, final `bef6bf9a86d234ad3988df44a7a74cac`. Only actual authorized owned-guest calls, isolated source clone and documentation; no host restart/config change, approval/circumvention, stable publication or existing-work deletion.

### Actual installed L0 quota/spool — scoped native PASS; root attestation pending

New disposable Box `s5b0-quota-1120`, UUID `382c4dc2-d82a-4477-a098-1dd2c5d0d640`, 2 GiB, 768 MiB, 1 CPU, no Internet/nesting; work_ref S5B0-quota, agent_ref s5b0-acceptance.

| Case | BW original | Native original | Observed result |
|---|---|---|---|
| Create | ecbeb00153fd95fe8fbaff77a2159c0f | 1de9d3b7-94f7-4e4c-b36a-fb91c3de5cd8 | SUCCEEDED |
| Readiness | 1430f040e84f12a95515c8014078e1df | 4f2954f4-ceb5-40a1-800f-421ad82bad74 | SUCCEEDED, guest exit 0 |
| Preflight df/stat | ab8731e2746faab8fe925a130a9990ac | b0d12b81-4144-4b2d-a2b6-0ae9cad03170 | Guest exit 0; Btrfs, df reports shared filesystem capacity, not quota cap |
| Bounded incompressible 2 GiB fixture write | 29a592b43b9d1b6a7409d16a1953d83e | a85b4c77-1006-4f03-a6f0-ea04f133f7ff | SUCCEEDED original, guest exit 1, empty recorded streams; do not infer actual stderr content |
| Fresh harmless post-fill Exec | 3ead46a6cd3289e38560168ffbe59185 | 9b6fe814-43c5-409e-afd3-d3e410701575 | FAILED / NATIVE_EXEC_SPOOL_QUOTA_EXHAUSTED_PRESTART, OUTPUT_SPOOL, guest outcome unproved, no stdout/stderr metadata; native pre-start failure, not headroom-pre-dispatch path |

Independent guest SFTP file list `ab8e4cbbbd61c52985b9644ae2da07cc` reports /var/tmp/s5b0-quota-fixture.bin size 1514610688 (< 2147483648 requested). Exact newly failed Exec is terminal, Activity tracking `a868db91cd954b5cdfc4604db214e3d7` classifies it TERMINAL_FAILED_NOT_ACTIVE, bounded sample no active claim. Actual operation_recover inspect returns NOT_REQUIRED / NOT_UNKNOWN, native_retried=false. This proves the installed typed pre-start spool-failure classification, not independent qgroup cap proof or the pre-dispatch headroom path. Physical qgroup attestation remains required; host_status still selected_storage_quota_verified=false. Preserve the filled Box for exact-host review (do not rerun fill or attempt Exec mutations). On trusted host, from reviewed RC9 checkout: `sudo python3 scripts/host_btrfs_quota_attest.py --project user-1001 --box-name s5b0-quota-1120`. Expected HOST_QGROUP_MAX_REACHED if actual raw qgroup referenced >= limit; if not, record real failure rather than forge witness. Recheck MCP status, then scoped Stop/Delete and retained output/issue verification; expiry/reboot separately B5. No Host operation performed by Agent.

### A1, A3/A4/A6 and PRS disposition

- A1 actual RC9 MCP fresh benign guest exit 127: BW `460b4cb736a142468182c466c05acb28`, native `7ed51b84-b0a3-4a9d-8b37-8e0baad7b7c1`; exit 126: BW `6df60a0f110d06884a67c6f9a734356b`, native `ebfa5ab4-7d2b-41d2-9ffd-562cdd91cbcb`. Both originals SUCCEEDED, exact stdout and stderr read back, separate observed nonzero guest exits and command_success=false. No shared controller restart; fresh installed smoke, not #151 post-restart independent witness. Issue #151 fetched OPEN; historical original a529cc5a3bb2f64714455d0612161a80 remains unclassified. Prior independent A1 native runners are supporting evidence only. A1 PARTIAL; no historical replay or waiver.
- A3/A4/A6 exact-source lab in preserved dev Box `11bbc73e-2bce-4cf9-857f-799f768305f8`, new isolated clone /tmp/boxweave-s5b0-fVVeun/repo, verified HEAD RC9. BW `9852fe675ff47b822f4050fcc5dbc4f8`, native `4474c099-fa65-44cb-9ede-9102efeba200`, guest exit 0, complete output: `python3 -m unittest discover -s scripts -p test_v1_stable_gate.py` ten tests OK; `go test -race -count=1 ./scripts/s5a3probe/...` both packages PASS. A3/A4 synthetic signing and compensation only; no standalone actual-client inert probe connection or supported human action-bound signature/issuer verification, production approval remains TRUSTED_CLIENT_APPROVAL_UNAVAILABLE. A4 actual authorized compensation BLOCKED; prior deterministic native passes retain their earlier exact identities. A6 offline gate regression PASS, feature freeze BLOCKED pending mandatory A resolution. No actual stable signing, protected-environment configuration, new product change or publication.
- Physical L0 PRS grant/revoke NOT_TESTED after RC9 upgrade: no host root endpoint/status or exact operator staging/approval available in current connector, and approved Root plan can stop all owned shared-ACL NIC consumers, including existing development work. Do not admit a live peer mutation or try to reach host root from a guest without separate exact approval. Need trusted host read-only service status/installed PRS binary/unit/endpoint identity, then exact authorized plan and disposable pair; maintain physical and isolated Native witnesses separately.
- Final box_list tracking `cb7f0165c30141563246df8be2d3e385`: three earlier Boxes unchanged in running/stopped disposition, plus retained quota fixture running. RC8 rollback backup preserved by non-interference, not independently verified. No remaining identified active UNKNOWN in bounded SQLite sample; not global native all-clear.

**Next:** complete root quota attestation while fixture exists, scoped cleanup, and read-only PRS service checks; continue independent A1/B cases while actual-client and prior incident scope decisions are pending. No B gate or A freeze signed by this B0 evidence.
