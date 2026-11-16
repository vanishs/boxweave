# S3 execution kickoff — BoxWeave v1 (2026-11-15)

**Issue review checkpoint:** [full Open Issue reconciliation](GITHUB_ISSUE_TRIAGE_2026-11-15.md) after merged #146: #131 closed as implemented S2 GC; **#148 and #151 are active S3 P1**, #150 S4 with S5 verification; legacy #121/#123 deliberately remain open independently after v1 functional merges.


**Status:** S3 IN_PROGRESS, S4/S5 NOT_STARTED. Protected v1 base merge [PR #146](https://github.com/vanishs/boxweave/pull/146), exact commit e4cab7b692faf490da5786a8f6975dff79330341.

**Accepted predecessor:** S2.5 SCOPED_PASS_WITH_DECLARED_LIMITS for a supervised, trusted single-developer non-production early development base. Host-local RC6 manifest, native 8/8, and resource headroom were independently observed. Full BH019/AppArmor/BPF/UID/GID/network/disk quota, untrusted sibling security and release are explicitly NOT accepted. [Signed scope](../verification/S25_PHYSICAL_HOST_DEV_BASE_POSTFLIGHT_2026-11-15.md) · [cross-stage ownership](S25_EARLY_DEV_BASE_PHASE_REALLOCATION_2026-11-15.md).

## S3 work order — begin with safety regression

1. **#148 OAuth refresh and original idempotency (first).** Reproduce same grant access A → committed Exec key K → refresh B → same intent/key, with exact original Operation and exactly one native Guest side effect; prove independent grant isolation, viewer private history continuity, revoked/expired token refusal, manual token actor separation and in-flight Claim. Fix stable OAuth grant actor separately from ephemeral token credential ID. No unattended mutating Agent loop until proved.
2. **X-02/X-03 Create/Delete SQLite cut boundaries.** Disposable source/installed/native SIGKILL and SQLite fault tests; fix only observed unsafe split, preserve original UNKNOWN. Never rewrite already working atomic BH018 Exec finalization.
3. **#151 installed MCP restart fresh readback.** Preserve original admitted ID and classify native/SQLite result; bounded fresh-runner repeats, no uncertain native replay. Latest original #146 installed test attempt 1 failed after a fresh exit126 readback was admitted; repeat does not erase evidence.
4. **Exactly three public File tools:** box_file_list, box_file_read, box_file_apply (create, replace, delete modes). Use confined official Incus SFTP, bounded paths/output, reject traversal, symlink/hardlink/special targets and foreign UUID. Unprovable atomic conditional replace/delete must return typed UNSUPPORTED, never pretend CAS.
5. **Full installed Agent loop:** owned Box Git checkout → failing test → captured original Exec output → source fix → passing test → independent native postread; TC-015/16 two-Agent interrupted handoff, viewer and security negatives. Test full S3 and affected S1/S2/S2.5 behavior on final exact head, sign off only evidence-backed supported cases.

## S4 and S5 queued — not canceled or prematurely started

- **S4:** four networking Tools, peer/forward/LAN/nesting parity, BH019 AppArmor policy namespace/admin kernel decision, cgroup BPF actual attach/deny, collision-free UID+GID lease allocator, deep sibling/foreign packets and proven supported quota/Exec spool bounds.
- **S5:** complete 20/20 MCP Tool + all CLI black box, authenticated nested installed/native/independent Verifier security and original failure recovery, host-capable public IPv6 as required, exact-binary release provenance and controlled publication.

**Execution rules:** S3 work stays on separate PR, not direct v1 writes; no mutation to historical UNKNOWN or running shared L0–L5; no privileged/unconfined or disabling host AppArmor, no host admin socket delegation. Destructive fault cases on new disposable CI/native labs only. The published RC6 tag/asset stays immutable.

**Current next executable action:** #148 deterministic failing regression, then actor/token security fix in disposable tests. No S3 PASS claimed by creating this document.
