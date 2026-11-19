# S4-B3 — focused native closure audit and remaining formal gate (2026-11-18 PT)

> **2026-11-18 PT owner scope correction:** physical whole-host reboot evidence is **explicitly deferred to S4-B6/S5**; B3 accepts *real Incus and PRS service-level outage* with original journal/SQLite/ACL/native packet readback if the focused independent run passes. The precise Root ETag GET→PUT native contender and independent final evidence audit remain **open until actually verified**. See [approval and acceptance boundaries](../decisions/S4_B3_REBOOT_SCOPE_AND_CLOSURE_2026-11-18.md). This does not retroactively turn a SIGKILL test into a full host reboot or assert `S4_B3_PASS`.

**Decision: `S4_B3_PARTIAL`. Fourteen evidence-backed scoped PASS verdicts are now retained. Do NOT issue `S4_B3_PASS` until the last mandatory evidence boundary is resolved.** This audit extends the [eight original scoped verdicts](S4_B3_SCOPED_ACCEPTANCE_2026-11-18.md), not a re-run or a change to those original receipts.

## Proven unchanged executable source

- Baseline tested B3 executable SHA: `cd3b2b95b857fe3dcc055090b2d8c7dc1a635a40`, original [9/9 real Incus 7.6 native matrix](https://github.com/vanishs/boxweave/actions/runs/44418327692) SUCCESS.
- The compare from that SHA to the latest reviewed development HEAD `4fb6a9022fd8495daca349301b0d9586716e9ae7` contains **zero** changed shipped Go source or native test Go files; only focused Actions YAML and documentation/archive. Historical B3 worktrees remain untouched and [classified](S4_B3_RECOVERED_WORKTREE_LEDGER_2026-11-18.md).
- Latest same-source focused Actions completed SUCCESS without rerunning the initial nine jobs. [Quality](https://github.com/vanishs/boxweave/actions/runs/44434454378) and [amd64/arm64](https://github.com/vanishs/boxweave/actions/runs/44434454396) also SUCCESS on `4931304416a1ac451f6bac288f75db1b46505890` (later changes are docs only).

## Added native receipts and bounded acceptance

| Scoped sub-gate | Source-exact original native witness | Result |
|---|---|---|
| Foreign Project live IPv4/v6 TCP/UDP during B3 grant | [run 44432962230 / job 135254661846](https://github.com/vanishs/boxweave/actions/runs/44432962230/job/135254661846), original `80a7fa920bb8e4683d1b5ec15990332b`, four established counters rise without a client failure | PASS |
| Installed B3 PRS SIGKILL restart without original replay | Same job; `COMPLETE_VERIFIED` root journal SHA unchanged, original SQLite SUCCEEDED and retry refused | PASS |
| B3 external post-plan ACL/ETag contention fail-closed | [run 44433198146 / job 135255465032](https://github.com/vanishs/boxweave/actions/runs/44433198146/job/135255465032); original `b9d26b9d027672c8f6f09a8f03af6f02`, became UNKNOWN, zero CONFIRMED grant | PASS |
| B3 same-Project distinct Peer admission while prior original DISPATCHING | [run 44434157023 / job 135258671956](https://github.com/vanishs/boxweave/actions/runs/44434157023/job/135258671956); original `b917987b2ba585b87c8b6e5cbb820e93`, second admission locked; original then completes | PASS |
| B3 service binary upgrade/rollback without completed-original replay | Same job; different compiled version, restart twice, journal/ACL digests unchanged | PASS |
| Foreign Project four established flows during B3 selective directional revoke | [run 44434454430 / job 135259667143](https://github.com/vanishs/boxweave/actions/runs/44434454430/job/135259667143); each foreign TCP4/TCP6/UDP4/UDP6 count rises 8→300; B→C retained, A→B revoked | PASS |

For detailed original marker lines, artifact IDs and cleanup limitations, see the appended [scoped acceptance ledger](S4_B3_SCOPED_ACCEPTANCE_2026-11-18.md). The first nine matrix jobs are **not** counted as newly executed here. Read-only review of original logs after each completed run confirmed actual non-echoed result markers and that all targeted native jobs finished SUCCESS. Disposable foreign Project/bridge deletion was separately logged; L0 shared Project, ACL and prior dirty worktrees were never mutated.

## Honest remaining mandatory gap

1. **Actual host reboot of an in-flight/persisted B3 Root grant original and post-boot original/journal/SQLite/network readback** is **not independently witnessed** in these jobs. Tested SIGKILL/systemd restart and binary upgrade/rollback are narrower and must not be relabeled as physical host reboot. A GitHub-hosted ephemeral runner cannot be presumed to resume the same preserved host state after `reboot`. Reuse completed B3 ACK-loss read-only recovery and B2 restart evidence only for their proved scopes.
2. **A competing ACL mutation inside the live Root stop→ETag PUT fanout window** was not injected. The source's separate ETag check and sealed-plan binding are visible, stale *pre-effect* ETag conflict was natively refused, and existing B3 post-ACL/partial-fault isolation remained PASS; these combined observations do not prove every kernel fanout interleaving. Decide whether that narrower internal race is itself a mandatory independent gate, without inventing an experiment.
3. **Formal independent stage-wide reviewer / final exact-head acceptance** remains required by the previous scoped signoff. This audit was read-only relative to B3 product code, but the same agent authored the focused workflows; it does not misrepresent its own evidence review as a separate unaffiliated verifier.

**Next smallest path:** obtain the missing reboot-bound original evidence on an authorized disposable persistent native host (or an explicit owner-approved revision of the B3 mandatory scope), then have an independent reviewer compare the existing original receipts against the immutable final executable SHA and issue the formal stage verdict. Do not repeat the already accepted 9/9 matrix, do not modify L0 shared host policy, and do not force a final PASS by inference. PR #153 remains Draft; no merge/RC/release.
