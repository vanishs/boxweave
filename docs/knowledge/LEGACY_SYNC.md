# Legacy knowledge synchronization

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
