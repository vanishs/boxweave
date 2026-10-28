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
