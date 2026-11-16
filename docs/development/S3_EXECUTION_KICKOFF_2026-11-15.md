# S3 minimal execution kickoff — after scoped S2.5 (2026-11-15)

**Phase-1 code checkpoint (2026-11-15):** The 3 typed File Tools, restricted native Incus SFTP list/read/O_EXCL create, synchronous SFTP SQLite finalization (no invented native Op) and basic tests are now committed to Draft PR #152. [Source and exact unresolved native phase-2 obligations](S3_FILE_TOOLS_PHASE1_CODE_CHECKPOINT_2026-11-15.md). Old step 1 below is an archival entry, NOT an instruction to reimplement tools. **Phase-2 actual native File negatives and phase-3 single-Agent loop have NOT passed.**


**State:** S3 IN_PROGRESS, not accepted. Existing [PR #152](https://github.com/vanishs/boxweave/pull/152) is the only change vehicle; target protected v1 based on merged S2.5 PR #146 commit e4cab7b692faf490da5786a8f6975dff79330341. No new PR, no host daemon mutation.

**Current authoritative scope:** [S3/S4/S5 minimal work rebalance](S3_S4_S5_MINIMAL_REBALANCE_2026-11-15.md) · [S3 phase card](phases/03_FILES_AGENT_LOOP.md). The older initial S3 start-order of #148 → Create/Delete → #151 → three Files → full Agent handoff is superseded. These issues are not closed or waived.

## Smallest S3 runnable sequence

1. Build the three Guest file tool schemas and exact owned-UUID/role/restricted SFTP native handlers; exclude host path, foreign ownership, symlink/hardlink and special files. Bounded list/read and exclusive O_EXCL create with original SQLite receipt. Keep replace/delete modes with typed UNSUPPORTED if point-of-use atomic CAS cannot be proved.
2. Test one native Box admin/Viewer and file safety negatives, concurrent exclusive-create single winner, original-key replay and bounded/read partial hash truthfulness.
3. Run **one supervised Agent** installed MCP + real native Guest Git → deliberate failing test → original output read → fix inside Guest via existing box_exec/Git → passing test and exact guest file/native readback. No network Tool, no large checkpoint orchestration, no physical L0 service restart.
4. Capture a tiny reusable case record: exact source/install SHA, Incus/Box UUID, Tool+role+mode, original operation/native ID, observed file/guest exit and native negative poststate. Run current quality and affected S1/S2/S2.5 independent test on the exact candidate, then stage signoff/normal merge.

## Work carried to the next phases (tracked, not forgotten)

- **Early S4 before risky network-changing mutations:** [#148](https://github.com/vanishs/boxweave/issues/148) stable OAuth grant actor vs active token, cross-grant and viewer continuity, plus native **X-02/X-03** Create/Delete coherent SQLite/native commit/crash/UNKNOWN handling. Then four Network Tools, ACL/Scriptlet, dependent delete, #150 AppArmor/BPF/UID/GID and safe L1/L2 support.
- **S5 release gate:** [#151](https://github.com/vanishs/boxweave/issues/151) intermittent postrestart original fresh readback attribution and native fate, repeated installed failure tests; complete two-Agent TC-015/016 handoff/race, TC-019/020, all 20 Tools and CLI, real WAN IPv6/LAN/security/independent Verifier, original receipts and exact signed release provenance.

**Critical S3 limitation:** only attended one-trusted-operator use, no unattended automatic guest writes across OAuth rotation, no original-key retry after refresh, no cross-UNKNOWN release or blind native retry. If basic single-session/ordinary restart native behavior proves unsafe, S3 is BLOCKED until repaired even if the extended #151 campaign belongs to S5.

**S2.5 accepted input:** official L0 RC6 binary/manifest, native 8/8, scoped trusted operator, separate L1 candidate and 5+10 running Boxes; original UNKNOWN locked, unverified quotas and stronger AA/BPF limitations preserved. [Signed host report](../verification/S25_PHYSICAL_HOST_DEV_BASE_POSTFLIGHT_2026-11-15.md).

**Next code task:** safe native Guest file List/Read O_EXCL Create implementation and tests, *not* OAuth actor/schema overhaul. [Full Issue inventory](GITHUB_ISSUE_TRIAGE_2026-11-15.md).
