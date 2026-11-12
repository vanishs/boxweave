# HOST_GO three-round live checkpoint — do not infer PASS from this file

This is the **small mutable checkpoint** for [the user-triggered three-round runbook](../development/HOST_GO_THREE_STAGE_RUNBOOK.md). Detailed factual incidents append to [EXECUTION_LOG](../development/EXECUTION_LOG.md) and the per-ID [BH matrix](BRIDGEHEAD_ACCEPTANCE.md); never overwrite past failures. Only the currently invoked round can change to IN_PROGRESS, and evidence after the round must be committed to PR #133 before reporting it. A future Agent must read current GitHub and native state rather than trusting a stale checkpoint.

## Planning snapshot (2026-11-12, documentation only)

- Existing **tested source**: `7a01000a12114eba0fe4937af0e955bd6e46ed87` with expanded nested MCP LAB 5/5 at that revision. Latest **PR #133 documentation checkpoint**: `31d91584f07ff19135e96f086cd348bd4075f45d`, five same-head Actions SUCCESS, Draft against `v1`. On current head the candidate v1 binary digest was `ee64e247e6054ec62d3e93db4c2a25c51f369816d9b80a9faab07dde2658d3e4`. Reconfirm exact current head and artifacts on invocation; these are historical baselines, not authorization.
- [Exact-head formal matrix](S25_EXACT_HEAD_FINAL_LAB_2026-11-12.md): BH-014 individually signed; remaining BH-001–016 cases PARTIAL/BLOCKED, BH-017 NOT_AUTHORIZED, TC-011–014 not fully signed. Expanded LAB PASS is **not formal LAB_PASS**.
- `LAB_PASS=NO`; `HOST_GO=NOT_AUTHORIZED` until accepted real-host proof. Physical host exact identity has not been provided; it **must** be separately and explicitly approved in Round 3.
- Quota stress waiver remains; `DIR_QUOTA_UNVERIFIED` must be disclosed. No old `main` co-installation is needed or permitted. No production host or shared Incus mutation by the author of this planning document.
- Historical UNKNOWN: `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`, `a2fdf885b87a0b66fb3a17f913316c55` — no write retry, deletion or adoption.
- `boxweave-test-ser2` is a previously used isolated development Box; `boxweave-test-cli` contains an existing native-only L1 `bw-s25-l1a` in Project `user-1002`. Before using either, read-only inventory + native consent/ownership; never presume current availability or an empty Project.

## Round scoreboard

| Trigger | Status | Cases / evidence | Exact code and native binary | Remaining blockers | User told? |
|---|---|---|---|---|---|
| `完成HOST_GO 第一阶段` | **NOT_STARTED** | BH-001/002(lab)/003/005/009/010/011/015, TC-011: to verify with Box-first packet negatives | unset | baseline BH-009 direct-WAN/foreign/spoof exact-head | NO |
| `完成HOST_GO 第二阶段` | **NOT_STARTED** | BH-004/006/007/008/012/013/014/015/016, TC-011–014: to verify via fault + real Agent | unset | baseline fault/UNKNOWN/fallback, TC-013 actual Agent | NO |
| `完成HOST_GO 第三阶段` | **NOT_STARTED** | BH-017 exact physical host + all mandatory prior cases, actual Agent, full final CI | unset | exact physical host authority not granted; Round 1/2 gates | NO |

## Mandatory round-end entry (overwrite the active row only; preserve prior checkpoint elsewhere)

For the round that the user actually invoked, record `started_at`, `completed_at`, `stage_verdict` (PASS/PARTIAL/BLOCKED), exact git branch/commit and binary SHA, current candidate signed Incus runtime, the actual host/Box inventory and native original IDs, BH/TC statuses with I/N/V, verifier identity, original UNKNOWN and failed operation receipts, cleanup and retained Boxes, commit/PR comment/Actions URLs, and `next_ordinal_requires_new_user_turn=true`. Add an append-only incident in `EXECUTION_LOG.md` and show the owner the result. Do not self-invoke another ordinal.
