# S4-B6 integrated acceptance — 2026-11-20 PT

**Status: S4_B6_IN_PROGRESS / S4_B_PARTIAL.** Owner explicitly initiated B6. Sole Draft PR [#153](https://github.com/vanishs/boxweave/pull/153); no new PR, merge, release or shared L0 upgrade. Use [six-stage plan](S4_B_SIX_STAGE_PLAN_2026-11-16.md), [Box-first lifecycle](AGENT_WORKFLOW.md#development-environment-and-box-lifecycle) and the [B3 reboot decision](../decisions/S4_B3_REBOOT_SCOPE_AND_CLOSURE_2026-11-18.md).

## Recovery and exact baseline

- Start HEAD `d9e6239e5c29eb5451e72508437dc0d715f2896a`; comparison to final tested B5 source `4f775c4091de01a35c1afe3086252b1d7200e6e1` contains nine Markdown files only. Start HEAD quality/candidate workflow completed SUCCESS; docs-only jobs are not new executable/native evidence.
- B5 original raw CI audit: [S4-A](https://github.com/vanishs/boxweave/actions/runs/44697962427/job/136161774028), [B4/B5](https://github.com/vanishs/boxweave/actions/runs/44697962497/job/136159162312), [earlier linked PRS](https://github.com/vanishs/boxweave/actions/runs/44694417303/job/136147638041). Final B5 gate closed in [B5 report](S4_B5_DEPENDENT_DELETE_IMPLEMENTATION_2026-11-19.md). Runtime changes after earlier tests require appropriate B6 regression, not inferred new native PASS.
- Reused owned L1 `s4b5-nested-acceptance` UUID `be1af99d-b9b1-48f9-9548-524473eef163`, work_ref `s4b6-integration-20261120`, agent_ref `s4b6-dev`. Read-only original preflight `b6d115ac429e63d9b2634a652d5c6678` guest exit 0: inner Incus 7.6, no instances, outer AppArmor unconfined. Older dirty worktrees at /root/boxweave and /root/b5-exact-review are untouched. Separate /root/b6-integration detached worktree at start HEAD; /root/b5-final-exact preserved. No outer host or Project settings changed.
- Inner preflight `81d0ac59e863d077cedd97d3fbd95c42` exit 0: restricted nonroot service UID, group incus (not incus-admin), existing isolated bwbr0 and native ACL, exact installed B5 binary reports ready/Schema 5, no native instances; storage quota not verified. Do not reset the existing private B5 ledger. Use a separate test ledger.

## Initial B6 repair and verification

- Exact start HEAD `make verify` original `cdc6bd3ccf11f1457810c3a6808044e0` Operation SUCCEEDED but guest exit 2. Go negative fixtures for OAuth approval file permissions and Scriptlet root lock failed under umask 077: creation mode 0644 was masked to 0600. This is a fixture defect, not proof that runtime accepts unsafe permissions.
- Correct only test fixture construction with explicit chmod in private temporary paths, including the 0755 OAuth directory negative. Original fix attempt `85f7554188e9b9c62f5301dbb7ec6365` exit 1 was stopped by expected local hook executable-bit changes from make hooks; no source edits in that attempt. Successful fix/rerun `4325e9532599fcab54863b7955da3eda` native `e760600f-821f-4902-aaac-d2279fc86454`, guest exit 0: targeted race tests pass under both 077 and 022, full make verify PASS, all five Go package suites and Python 15+23+7 cases.
- Python test-kit known-mutation guard omitted the two new forward mutation tools. Explicitly deny both by name even when incorrectly advertised read-only; test mock deliberately misannotates both and verifies zero HTTP mutation effects. Original `81d0ac59e863d077cedd97d3fbd95c42` exit 0, 16 protocol tests PASS. This is test-tool safety hardening, not a runtime API change. No production Go source changed.

## Gates to close next

| Gate | Current evidence / next action |
|---|---|
| Exact final source quality and dual-arch | Local initial fix/rerun PASS; final head CI and full verification with Python guard pending |
| Installed restricted HTTP MCP four network schema/calls | 21-tool B4 CI on B5 final code and earlier official SDK B3/PRS PASS; repeat integrated candidate, actual native poststate |
| S4-A/S2 and S3 file regressions | S4-A/S2 original exact runtime success audited; S3 installed native reacceptance and current B6 source-equivalent checks pending |
| Scriptlet compatibility/rollback, peer isolation and dependent faults | Prior B2/B3/B4/B5 scoped evidence; B6 integrated independent evidence still required |
| Persistent whole-host reboot / incomplete original recovery | OPEN; B3 service-level restart is not physical reboot; use separately authorized dedicated persistent native host/VM, record kernel boot identity and original receipts. Do not reboot shared L0 or relabel container restart as host reboot. |
| Independent verifier and final B6 verdict | OPEN; review original raw native results, same source SHA/digest and residual risks before signing |

Retain both B5 development Boxes. No delete/stop without user-approved scope; no automatic deletion on PASS. At interruption preserve the worktree, source, Operation IDs and next safe step; user-confirmed deletion applies after successful acceptance and durable evidence handoff.
