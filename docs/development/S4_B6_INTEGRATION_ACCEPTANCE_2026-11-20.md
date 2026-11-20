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

## Exact built candidate and integrated native smoke — scoped PASS

- Tested code `1f1d1d8011f053a44faa3d4bf7ad7ab931763336`, clean /root/b6-candidate, build Go1.26.9 CGO1, binary SHA256 `85695931a2f01196b0830986c83b0d44e951636cd8e8ffbbc30f7e584282ff73`. Outer build Exec `51be7ff672a9d238d817f0ebefb888bf`, guest exit 0. Separate nonroot service account with only restricted Incus group, localhost 18673, fresh private SQLite Schema5, existing isolated inner Incus 7.6. No replacement of installed B5 binary or ledger.
- [Exact witness script (historical text)](../verification/evidence/s4b6-installed-smoke-2026-11-20.py.txt), SHA256 `ceabc76485d0571ee70c68dacfe043f7622efcfa84f45039d9e95f75cbb781e1`, pinned to this exact disposable lab; not a generic production runner. First smoke Exec `33ca87d150cc415af3c09b918a04444d` exit 1: short idempotency key rejected before admission. Separate read-only inventory/SQLite diagnosis `d6a7dfee292af91e1be2ae1715b1baa7` exit 0 proved zero operations and zero native instances; corrected test keys to minimum 16+ characters, no original mutation replay.
- Actual smoke Exec `62b83f03e41a8bcbf7d4b1d9f7ab4a60`, native `7fb29aa0-fa63-4d7d-ab0d-2c90f1cef974`, guest exit 0; 21-tool catalog and four network schema annotations, real authenticated Python MCP Create, all three native guest file tools, Exec exit 0 and byte readback, foreign peer and wildcard forward refusal, real localhost TCP+UDP echoes, native device verification, actual clean controller process stop/restart with fresh client and original SQL operations and packets retained, TCP forward Delete, surviving UDP traffic, dependent Box Delete, UDP denial, original same-key replay and native absence. Echo setup used native file push/Exec strictly within the disposable child; do not treat that helper setup as MCP file transfer. This is same-Agent inspected real native functional evidence, not independently run official SDK/PRS or recursive security certification.

| Original action | BW operation | Native operation |
|---|---|---|
| Create | eeea5e052ac679a6ab1dd0c52b2453b0 | d914ad83-6b47-4b8d-a379-ca246b9f5e1a |
| File create | 4faf1a44c2421c34e30ce5943cb006ea | none (SFTP) |
| Exec | 4cded606bb455e12b974578b7a88c915 | a0e852af-926a-4b6a-b383-6a3664a3aa0e |
| TCP forward | d01c5f91e30de8eae163e5e8543b9443 | 3c74557f-68d9-4ffc-a84a-153a72cce903 |
| UDP forward | 9c9720818fc38736857a2c0592dd3a68 | b826c628-47b7-41c9-8390-daf86ac74da3 |
| TCP forward Delete | be5db9c70cb37b913221f5a5a539b695 | e5b7a713-22fd-40b2-abe6-1535815d784c |
| Parent dependent Delete | 436505f41a04785a40512a0a1f19dfc0 | 8b19cb97-29a4-450f-ade6-d7e91c426780 |
| UDP Delete child | 181a94dac2694483a9d7b1abfc31e06a | 33d6971f-84e2-4b38-87bd-49add514c42c |
| Stop child | 961a5a868113348c9d1356da6bf7d9ea | fcb22add-1d5f-4160-a23e-11dfba7941fe |

Separate read-only SQLite/native postread Exec `b4b3326a57b0e27edbd77a59f27e988b` exit 0: exact child UUID `649de269-4ba5-4cc6-9ff7-e61a32a075df` DELETED by original parent, nine total originals all SUCCEEDED, two REVOKED forward claims, zero inner native instances and no candidate or forwarding listeners. First postread `1a08a8dc662a8086040820193d873370` had all original rows/tombstone but typo in table name; corrected read-only query, no replay. The explicitly scoped disposable child was deleted as part of its acceptance case; L1 development Box and the earlier B5 development Box remain intact. Keep /var/tmp/s4b6-integration private DB/checkpoint and exact binary; no live candidate listener. No secret copied to GitHub.

**CI at checkpoint:** exact code [quality](https://github.com/vanishs/boxweave/actions/runs/44702385817) and [dual-arch](https://github.com/vanishs/boxweave/actions/runs/44702385657) SUCCESS; [S4-A exact-code native](https://github.com/vanishs/boxweave/actions/runs/44702385615) still in progress at read. Do not claim a completed final B6 gate. Remaining: same-source official SDK/PRS positive peer/revoke and anti-spoof/Scriptlet/rollback, full affected fault regressions and independent verifier; actual persistent-host reboot/power cycle is still OPEN, not this controller restart. Use a separately authorized persistent isolated host/VM, not shared L0. **S4_B6_IN_PROGRESS / S4_B_PARTIAL**; no merge/release. Resume from this existing Box and final source, not a new lab except where required.
