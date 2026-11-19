# S4-B3 recovered L0 worktree inventory and preservation ledger (2026-11-18 PT)

**Status: RECOVERY_CHECKPOINT_SAVED; NOT a code merge or S4_B3_PASS.** Original dirty worktrees remain untouched. Baseline: sole [Draft PR #153](https://github.com/vanishs/boxweave/pull/153) code at `cd3b2b95b857fe3dcc055090b2d8c7dc1a635a40`; subsequent `97a528347db69585759f5ba3f3185c4bf4eec6c1` modifies development documentation only. Recheck HEAD before using this historical snapshot.

## What was independently inspected

Through installed BoxWeave L0 RC7 `box_list`, `box_exec`, `operation_get`, `operation_output_get`, and read-only `box_file_read`, inspected **6 relevant running Boxes / 11 local Git worktrees**. For all uncommitted/untracked regular source files outside generated `.pyc` and build binaries, compared the exact `git hash-object` SHA-1 blob to GitHub's recursive HEAD tree blob SHA, not just filenames or source commit ages.

**Result: 71 content candidates = 45 bit-for-bit already on remote + 25 different from the latest remote version + 1 not present on remote.** A different blob by itself is NOT evidence that its historical changes are missing; most worktrees are older than remote and the 9-scenario native matrix is on the more developed remote code.

| L0 Box | Paths and source checkout HEAD | Recovery |
|---|---|---|
| `33fd7fd3-8712-4ca7-92bf-91bee3bf4a82` `s4b3-final-isolated-1118` | `/root/boxweave` `2407568` | One older workflow variant; preserved in place |
| `3e77c5d6-4156-4953-9b45-4101233f029d` `s4b3-p01-isolated-1118-a` | `/root/boxweave` `f8c629a` | Only generated Python bytecode untracked |
| `a3a6836d-b526-4d10-b1e5-ca0554f1c622` `s4b3-p0-root-1118-x` | `/root/boxweave` `f8c629a` | One older workflow variant |
| `bd977f73-b830-48ef-b635-5d34b4a79095` `s4b3-p01-audit-1118-0731` | `/root/work` `07967d7`; `/root/work-b3-completed-ack` `f0e495f`; `/root/work-b3-mcp-group-fix` `b44b9a1`; `/root/work-b3-revoke-fault` `ac183a4`; `/root/work-v1-mcp-description` `3c663d7` | 27 exact remote matches, six differences; completed ACK and MCP-description worktree contents matched source-current where checked |
| `c31fff74-c817-4cf2-86a2-fcbab672d124` `s4b3-secure-fix-1118-02` | `/root/boxweave` `7acd929` | **One truly untracked test absent from remote**; entire source preserved below |
| `745d1550-97e8-4a02-9f19-4ef356a0274e` `s4b2-scriptlet-dev-1117` | `/root/boxweave` `74e4d4e`; `/root/boxweave-b2-current` `851bcbd`; `/root/boxweave-b2-foreign-probe` `fdf643a`; `/root/boxweave-b2-latest` `5f6ee4b`; `/root/boxweave-b2-scriptlet-ack` `5cbbbec` | 18 exact remote matches, 12 older divergences |

## Singular code candidate not found in remote

- Original: L0 `c31fff74-c817-4cf2-86a2-fcbab672d124:/root/boxweave/internal/install/native_peer_root_multirevoke_native_test.go`.
- Original local `git hash-object` SHA: `735a4e22365f2890c624a9578868a580c1e8de69`; exact 6,760-byte source.
- **Archived full bytes** at [S4_B3_UNMERGED_MULTIREVOKE_TEST_DRAFT_2026-11-18.go.txt](archive/S4_B3_UNMERGED_MULTIREVOKE_TEST_DRAFT_2026-11-18.go.txt).
- This older native test seeds three directional ACL grants and tests single-direction revocation while keeping other grants. **DO NOT simply move it into `internal/install`**: it uses direct test-fixture SQL/ACL setup, depends on environment fixtures and is not hooked into the current workflow. The current HEAD already has tested `multi_revoke` / `multi_regrant` real-network scenarios and `native_peer_root_multi_installed_native_test.go`. Preserve for a future targeted gap review, not as an unverified product change.

## Divergent historical worktree files — NOT YET accepted for merging

Each entry is present in the latest remote tree but has **different** local bytes. A complete remote-aware semantic hunk review was **not** established for each; remote Git Blob mismatch is not sufficient to mark any as an uncovered fix. Leave these paths untouched until a separate targeted review.

- `s4b3-final-isolated-1118:/root/boxweave`: `.github/workflows/v1-s4-b3-grant-native.yml` (`eff3fa7c`).
- `s4b3-p0-root-1118-x:/root/boxweave`: same workflow (`107754ec`).
- `s4b3-p01-audit-1118-0731:/root/work`: `internal/control/peer_tools.go` (`09c77f56`), `scripts/s4_b3_native_mcp_client.go` (`5281ac95`), B3 workflow (`2a98fad3`), `internal/control/strict_tool_test.go` (`3f6e3dca`); separate older `work-b3-mcp-group-fix` workflow (`6e328b1f`) and `work-b3-revoke-fault` workflow (`70cd4178`).
- `s4b3-secure-fix-1118-02:/root/boxweave`: `internal/state/network_peer_revoke_snapshot.go` (`f6156e77`), `internal/state/network_peer_revoke_snapshot_test.go` (`5e7fcae4`), `internal/install/native_peer_root_integrated.go` (`ad4ad740`), `internal/state/network_peer_root_handoff.go` (`ffa62a32`), B3 workflow (`17916291`).
- `s4b2-scriptlet-dev-1117:/root/boxweave`: `internal/install/native_scriptlet_apply.go` (`0619ad3a`), `internal/install/network_peer_revocation.go` (`a0eb3188`), B2 quiescent workflow (`d9875f41`), B2 parity workflow (`32ea1f05`), `cmd/boxweave/main.go` (`7c6b8360`); `/root/boxweave-b2-foreign-probe` B2 quiescent workflow (`606f5395`); `/root/boxweave-b2-latest`: `native_scriptlet_atomic_journal_test.go` (`b209e372`), `network_peer_conntrack.go` (`6903e0bb`), `network_peer_conntrack_native_test.go` (`de8a213a`), `network_peer_conntrack_test.go` (`3e5224e7`), B2 workflow (`186a232d`), `native_scriptlet_apply.go` (`bb99a593`).

## Recovery query evidence, immutable provenance

All six read-only inventory and working-tree hash checks used `work_ref=s4-b3-recovery-audit-20261118`. The exact `box_exec` Operation IDs for **hash comparison** were:

- final `200dfadf4a5b37f7e4e3e764116664bf`
- p01 `480eadd38c9e93fea5772fe96fde2525`
- p0root `716853023a763dac154a6d7735f44aa7`
- audit `1032ed7aa53334e6a4f967ffdc062935`
- secure `a5c494551a44b1618995a09d8b7d6baa`
- b2script `b264ababbd6bbd35e973de1e63aeff43`

All six original Operations `SUCCEEDED`, `operation_output_get` returned `exit_code=0`. This is a source inventory observation, not native B3 acceptance and not proof that old divergent hunks are safe to merge.

## Safe next actions

1. **Already in GitHub**: 45 byte-identical local candidates need NO new commit.
2. **Unique draft safely archived**: if its test scenario is missing from the current B3 matrix, review/port from the archive onto a *fresh* latest-HEAD worktree, run `gofmt`, `make verify`, and independent native CI before implementing.
3. **Older divergent candidates**: evaluate smallest hunk against new HEAD (never wholesale `git add` / force push). Do not touch historical L0 worktrees as a shortcut.
4. **B3 acceptance**: use the separate source-exact GitHub Actions evidence; B3 remains partial until independently accepted remaining requirements are resolved or strictly scoped.

No Box Stop/Delete or host-root/ACL changes were made during this recovery inspection.
