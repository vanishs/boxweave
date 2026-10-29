# Current branch identity

After the authorized branch promotion, the legacy review ref is refs/heads/legacy/pre-v2, pinned at d8a09dd180139159e035963502e7793a37e92bae. V2 is main. Earlier main/rewrite/v2 references below are historical identities, not current editing instructions. Review procedure and worktree boundaries are unchanged; see [promotion record](../process/BRANCH_PROMOTION.md).

# Legacy knowledge synchronization

Phase6/SRC-084..088: legacy d8a09dd180139159e035963502e7793a37e92bae is unchanged from phase5; export25 exact tracked public files only for isolated tests. Original main/worktree and untracked containarium-candidate.zip remain read-only/preserved; no archive inspection, code merge or push. Fresh committed replacement handling supersedes the initial SRC-020 WIP inference; a stale readiness-test extractor is newly recorded rather than edited. New0.9.0 base f602acf0996b46688ad2b1322769414fea894fe6 was audited and three acceptance gaps fixed independently in0.9.1. [Legacy hashes](../verification/phase6-legacy-hashes.json), [comparison](../verification/COMPARISON_REPORT.md).

Final phase5 check2026-10-29/SRC-079..083: legacy main d8a09dd180139159e035963502e7793a37e92bae unchanged. No new legacy implementation was merged; unrelated untracked containarium-candidate.zip remains preserved and unread. New distribution/client/release work was authored/tested in the assigned LXC and exact public bytes synchronized to rewrite/v2. Existing supporting bootstrap/provider versions are test context, not migrated product code. No push or legacy modification.

## Initial observation

- Reviewed: 2026-10-28 America/Los_Angeles.
- Legacy checkout: `C:/Users/vanis/Downloads/boxweave`.
- Legacy ref: `refs/heads/main`.
- First reviewed / last synchronized commit: `4318da6ea4e4e45c55cf43b64acbdd2ede65fc43`.
- Initial status: ` M server/clients.py` and `?? containarium-candidate.zip`.
- Working client source SHA256: `6653b809eef737474b3804543c4ef7073deaffd5527219117a89679ea169e95d`. Reviewed as WIP only; not included in H or copied to v2.
- Ignored outputs research was selectively reviewed, with mutable-source hashes in SOURCE_REGISTER. Untracked archive content was not read. Private credential stores were not read.
- Initial worktree inventory: only legacy checkout/main. Git 2.53.0.windows.3 supports direct orphan worktree creation; rewrite/v2 and chosen target did not exist.
- New worktree: `C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2`.

## Procedure at each phase boundary

1. Read current knowledge/status. Read-only inspect legacy status/ref/HEAD. Never stage or clean legacy WIP.
2. Compare last-reviewed SHA to the fresh ref. If advanced, check `git merge-base --is-ancestor <old> <new>` before describing a linear delta. Nonzero requires distinguishing divergence from command errors.
3. Review changed code/docs/tests/Agent notes for knowledge only. A rewritten ref requires divergence review, not assumed linear history. Recheck WIP/ignored evidence independently; unchanged HEAD does not mean those sources are unchanged.
4. Add/update source IDs, knowledge and risks; introduce REQ/AC/ADR only in their authorized phase. Product conflicts go to the developer; no automatic architecture or requirements change.
5. Update this ledger with old/new SHA, date, inspected scope, accepted knowledge, rejected migration and reason. Never merge or copy legacy implementation automatically.

## Ledger

| Date | From -> to | Scope / disposition |
| --- | --- | --- |
| 2026-10-28 | none -> 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43 | Initial SRC-001..022; curated legacy knowledge only. Implementation, private material, archive and raw outputs excluded. Final ref/status/hash checks recorded in HANDOFF |
| 2026-10-28 phase 2 | 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43 -> same | Main unchanged; no committed delta/ancestry comparison needed. Same client WIP hash and untracked archive; selected ignored narrative hashes unchanged, not a whole ignored-tree snapshot. SRC-023 authorizes analysis; SRC-024 records fresh discovery only. No legacy code/notes migrated or modified; no new upstream/live acceptance claims |
| 2026-10-28 phase 3 | 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43 -> same | No committed legacy delta; original client WIP hash and archive status preserved. SRC-025/026 authorize design after confirmed product scope; SRC-027/028 add official contract/tradeoff references; SRC-029 records discovery only. No code merge, old-note edit, connect, deployment or live experiment |
| 2026-10-28 phase 4 | 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43 -> same | Local main/status/client WIP hash unchanged. Remote existing legacy checkout 7818668 with 32 preexisting status entries was inspected, never modified; new source authored separately. Only curated knowledge/new source synchronized; no old code merged. SRC-030 authorizes execution; SRC-031 records separated evidence; SRC-032 preserves results with live gate unmet |

| 2026-10-28 next-iteration startup | 4318da6ea4e4e45c55cf43b64acbdd2ede65fc43 -> d8a09dd180139159e035963502e7793a37e92bae | Ancestry check exit0. Reviewed 21-file change inventory and selected client/Tenant/provider-upgrade docs/code: baked replacement intent, exact native contract identity, explicit separately authorized Provider upgrade, upstream version subcommand. No AGENTS.md committed delta. Accept knowledge only: exact native contract and configured identity remain necessary; MCP update does not update daemon; upgrades remain separate scope. No legacy implementation merged, no provider upgrade authorized. Current legacy status only untracked containarium-candidate.zip, left untouched; server/clients.py SHA256 now 43e218b05d90886ebed03015b6c02a4039dd1449abbbe147093213464f42b02b (external committed evolution, not our edit). Ignored notes were not broadly re-audited. Phase5 gate not accepted: developer requires real-chain completion first (SRC-048) |

| 2026-10-28 phase5 TASK-011 | d8a09dd180139159e035963502e7793a37e92bae -> same | Fresh local main/status unchanged: only untracked containarium-candidate.zip. No new committed delta, no archive/ignored-tree audit or implementation merge. SRC-051 accepts gate and selects graph; all new source authored in assigned project LXC. Preserve remaining real-client/provider risks independently of graph evidence |

| 2026-10-28 phase5 TASK-003a | d8a09dd180139159e035963502e7793a37e92bae -> same at startup | No committed delta or archive/ignored-tree review. New client code authored through authenticated existing MCP in assigned LXC; Windows runtime checks are actual client-platform verification. Upstream Codex CLI replacement preservation issue is a new observed runtime fact, not legacy code migration. No old implementation merge or global registration/host mutation |

| 2026-10-28 phase5 TASK-003b1 | d8a09dd180139159e035963502e7793a37e92bae -> same | Fresh main/status remains unchanged, only unrelated untracked archive. No legacy implementation/ignored-note/archive migration. Dedicated native TUI evidence and PATH mismatch are new operational observations, not legacy behavior imports. Product source authored in assigned LXC; native tests in isolated Windows home; legacy checkout untouched |

| Phase5 entry audit SRC-059/060 | d8a09dd180139159e035963502e7793a37e92bae -> same | Fresh legacy HEAD/status unchanged, only unrelated untracked archive. No committed knowledge delta, archive/private-data review, code migration or old-project edit. Existing governance was re-read to distinguish phase5 entry from later feature/release obligations |

| Phase5 TASK-002a SRC-061/062 | d8a09dd180139159e035963502e7793a37e92bae -> same | Fresh legacy HEAD/status unchanged; only unrelated archive, not inspected. No knowledge delta or old-code migration. New receipt-history source authored in assigned LXC and synchronized with hashes; legacy checkout untouched |

| TASK-002/003 continuous run SRC-063 | d8a09dd -> same | Legacy ref/status unchanged at startup, only unrelated untracked archive. No legacy implementation migration or private/archive inspection. Assigned dev box identity verified; new source tested there |

| TASK-003 completion SRC-065 | d8a09dd -> same | No legacy implementation migration, archive/private-data read or old-project edit. Existing client observations used as versioned evidence only. New keys generated locally, public-only bootstrap via existing MCP to dedicated box; normal transport identity never used as host shell |

| TASK-005/006/007 startup SRC-066 | d8a09dd -> same | Legacy HEAD/status unchanged, only unrelated untracked archive. No code merge/archive/private-data review. Existing remote resources inspected; fresh current-source runtime packaged/tested separately, original services untouched |


TASK-006 / SRC-069 / 0.5.0: scoped lifecycle plans, single dispatch, pinned managed environments, schema4 recovery, Web/CLI receipts and config-scope guards implemented. 82 non-root Linux tests and Chromium fixture passed; first actual five-action Web/MCP cycle passed. Final repeat correctly stops TRUST_FAILED after reused-IP mismatch; trusted reconciler reports failure and was not repaired or bypassed. Approved exact test resource cleanup read back absent; original hello preserved. See docs/verification/TASK_006_LIFECYCLE.md for separate success/failure evidence and limits. Continue TASK-007 under SRC-066; no full release claim.


TASK-007 / SRC-070..072 / 0.6.0: selected four-profile connection/registration/UI scope complete; see [profile evidence](../verification/TASK_007_PROFILES.md) for actual Windows matrix,95 Linux tests/build, separate human ZCode/WSL confirmation and limits. Python3.11+ setup runtime is explicit; existing PowerShell/native path retained. Model and release acceptance remain pending. Legacy main d8a09dd unchanged, no automatic implementation migration or push. Continuous remaining phase5 work is authorized; no implicit public/destructive/inference action.


TASK-008 / SRC-073..075 / 0.7.0: native-guided GLM wizard complete in selected scope; schema5 revisioned public intent, shared CLI/Web, native credentials only and operator report distinct from inference.106 tests/build/Chromium pass; user confirms Z.ai/GLM-5.3/access. See TASK_008_MODEL_WIZARD.md for independent native and fixture evidence. No model request, provider repair or public deployment. Continue SRC-072 network/ports and dependent packaging/release.
TASK-009/010 / SRC-076..078 / 0.8.0: selected basic Tenant/port scope implemented and evidenced.132 Linux tests/build/Chromium fixture passed; fresh approved0.120.1 nested backend supplied actual shared membership, baseline TCP isolation, atomic bidirectional grants/revokes, stale-CAS rejection without partial write and TCP/UDP publish/unpublish packet observations. Preserve separate failed TRUST_FAILED creation and immediate convergence probes; corrected only the fresh installation's erroneous trusted CIDR through its installer. No replay/adoption or manual trust bypass. Final actual hello exit0, worker restart no replay, all three dedicated workloads/routes absent; outer lab and two empty policies retained. See docs/verification/TASK_009_010_ACCESS.md for exact attempts, commands, source hashes and limits. TASK-012 packaging/install/update/repair and TASK-013 release acceptance remain incomplete; phase6/model inference not authorized.

## Phase7 maintenance review

Legacy HEAD remains d8a09dd180139159e035963502e7793a37e92bae, read-only; no merge or product synchronization. Future manual review follows [maintenance](../process/MAINTENANCE.md).
