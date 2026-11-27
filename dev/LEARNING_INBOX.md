# Learning inbox

## Deferred opportunity review (current #193 plan)

See the [active BDH plan](../docs/development/BDH_DEVELOPMENT_PLAN.md#find-deferred-opportunities-again)
for the owner-authorized improvement policy. This is the existing navigation
entry for later review of observations, linked evidence and deferred work;
canonical candidate histories remain under [learning](learning/), with
original work and PR evidence linked from those records. Review current
revisions rather than relying on historical entries below. For example,
[L014 revision 39](learning/BDH-L014/00000039.json) retains the latest
recorded transfer residual; further transfer-specific development is
currently deferred, with REOPEN preserved.

During normal work, capture material facts and defer nonblocking BDH
improvements. On request, review the current evidence and propose reuse,
creation, optimization, adjustment, repair or continued deferral. An
opportunity or review is not authority to select or implement a new BDH
change. Use the existing commands and durable histories. On request, use:

    python3 dev/bwdev --json learn next dev/learning --all --brief

For a relevant ordinary task, use the existing filters rather than --all.
The brief view retains audit findings (including nonzero exit), candidate
identity, original evidence, latest matching observation, recorded cost
when present and review/reconsideration information. Read the full view
and learn opportunity before deciding; historical reconsideration is not
a current decision. Without a checkout, read the linked current records
through authorized GitHub tools. No separate queue or background scheduler.

Manual bootstrap queue and conventions. An active PR/issue may hold the canonical candidate instead of editing this file; add a link here only when useful for discovery after the task. Do not duplicate the full observation. At each material failure, correction, handoff or completion, check for reusable value. Record one disposition even if no asset is needed.

Candidate fields: stable ID, work ref and source/evidence link, bounded observation, cause (or hypothesis), affected versions/environment, proposed asset or existing asset to update, validation plan, risk/change class, status and next step. Dispositions: candidate, verified, active, deferred, stale, deprecated, rejected, no-new-asset. A verified historical fact is not a verified new procedure. Larger unrelated improvements are separate tasks; no need to block product delivery for them. Use [learning skill](skills/continuous-learning/SKILL.md).

## Initial seed from actual prior work

| ID | Observation / source | Disposition and next step |
| --- | --- | --- |
| BDH-L001 | [#188 H5](../docs/development/PR188_H5_DELIVERY.md): missing Git caused 13 Git-dependent errors; missing origin/v1 ref caused an earlier attempt to fail. | Candidate: environment preflight distinguishes Git/base-ref needs per test. Initial environment skill includes these checks; automation and fresh regression not yet verified. |
| BDH-L002 | Same H5 evidence: populated Box caused clean-state fixture failures; two external installer tests required a clean host. | Candidate: fixture capability tags and separate clean CI lane. Initial verification skill prevents treating fixture limits as PASS. Do not alter production admission. |
| BDH-L003 | Same H5 evidence: prior CI was bound to H4B parent; H5 required exact-head candidate and independent CI, with merge-result distinct. | Candidate: deterministic SHA/evidence binding check. Initial verification skill records the distinction; automation/eval pending. |

The original historical report owns actual commands, IDs and results. These observations are not claims about today's installed environment or final CI. Initial seed is intentionally small; mine additional history when a current task needs it.

## First bwdev implementation learning (PR #189)

| ID | Observation and evidence | Disposition |
| --- | --- | --- |
| BDH-L004 | Dedicated Box create dab2374c02b77d1d2e75deab58581a05 failed SAFE_NATIVE_PREFLIGHT_DENIED; the retained authorized lab supported a new isolated clone (a3859c60a98595358707d0d2ca517d3f). | Preserve refusal; recover/reuse a separate worktree or conventional fallback. No safety bypass. |
| BDH-L005 | A large argv source transfer was rejected at admission; exclusive per-file creation in the isolated clone worked. | Candidate: bound transfer size, read original receipt after uncertainty, prefer verified exclusive file transfer for new files. Do not infer an exact size limit from one refusal. |
| BDH-L006 | make invocation 5aaed2e8a6e98b6089779ebf1366bd9e exited 127. The follow-up 46197b71acc1db5a5e3b50b550ddcc49 observed absent Make, Go and origin/v1 while Python/Git were available. | Verified for that Box: optional capability-specific doctor blockers; use direct Python tests without installing tools. Base-ref regression included. |
| BDH-L007 | Earlier installed-lifecycle CI run 45547041460, Legacy job 138979037296: connection refused after test-induced service restart despite PID/UNKNOWN checks. | Candidate fixture timing diagnosis. Add bounded listener readiness observation, preserve original state/identity/authentication gates, rerun independent native CI. Offline regression is not installed acceptance. |

## v0.2 dogfooding

- BDH-L008: original Box staged tree matched the remote v0.1 tree exactly;
  verify both before fast-forwarding, preserve other worktree changes.
  Resume operation 07246fc614004a7a08384d2134c5b98e and sync operation
  2408c75359b5f2d28950f1f53c1a3479. Candidate for future resume helper.
- BDH-L009: task history uses an immutable local revision rather than
  overwriting shared state. Local concurrency regression is not distributed
  coordination proof. Use Git/PR for explicit cross-environment reconciliation.

## v0.3 dogfooding

- BDH-L010: staged and unstaged change sets must be considered separately;
  comparing only the working tree against HEAD can conceal staged changes
  that have been undone in the worktree. Regression in test_bwdev_verification.
- BDH-L011: actual verify run records PASS on the dirty development tree
  as UNBOUND, and missing gofmt as BLOCKED. Evidence operation
  efc3d3a720706260d9a48ee95f9e3957. Preserve this distinction in future
  acceptance automation; independent clean-source results remain separate.

## v0.4 structured learning

- [BDH-L012](learning/BDH-L012/00000001.json): first durable structured candidate from actual CI status; keep as candidate until reviewed. Captured and audited by operation 3e071e56e76d684500fa0c62cf553a62.

## v0.5 self-evolution

- [BDH-L012 revision 2](learning/BDH-L012/00000002.json): implementation and validation attached; candidate remains pending independent review. CI helper, Skill, regression and Eval now implement this learning. Start surfaces it on relevant tasks; audit checks asset drift. Do not implement it twice.

- [BDH-L013 revision 2](learning/BDH-L013/00000002.json): nonempty test discovery contract implemented and exercised; candidate pending review. This Box's builtin unittest already failed for zero tests, so no universal old-runner defect is asserted.

## Universal activation / dual-loop learning

- [BDH-L014 revision 5](learning/BDH-L014/00000005.json) structures historical BDH-L005 (do not duplicate), links real U2 friction, records deferral and rediscovery, and attaches the bounded Box source-transfer Skill improvement. [Revision 6](learning/BDH-L014/00000006.json) reconciles documented #191 reuse; [revision 7](learning/BDH-L014/00000007.json) records a distinct actual #192 transfer reuse. Candidate remains pending review; inspect implementation instead of duplicating it. Neither observation establishes independent Agent efficacy or promotion.

- [BDH-L014 revision 8](learning/BDH-L014/00000008.json): actual independent ordinary transfer investigation and manual pending handoff reconciled against the original Box evidence. [Feedback](../docs/development/BDH_V06_INDEPENDENT_TRANSFER_FEEDBACK.json) and [witness disposition](../docs/development/BDH_V06_RELIABLE_EVOLVE_PLAN.md#e4-independent-ordinary-task-witness-and-reconciliation-2026-11-25). Existing candidate remains open; assess a separate bounded manifest/resume helper after v0.6 acceptance, not an automatic promotion or duplicate capture. The prior independent Agent did not itself commit the structured record.

- [BDH-L014 revision 9](learning/BDH-L014/00000009.json): a second ordinary Agent autonomously began work, rediscovered and used the bounded transfer procedure, appended an exact observation in a separate inbox, and recorded a reconciled local HANDOFF. [Original report](https://github.com/vanishs/boxweave/pull/192#issuecomment-7157664834); [original work history](../docs/development/BDH_V06_RESUME_WORK_HISTORY/00000002.json) imported byte-for-byte. No promotion or complete helper. Actual transport failure and crash durability remain follow-up validation.

- [BDH-L012 revision 3](learning/BDH-L012/00000003.json): P0 closeout after #192 merge recorded actual exact-source CI and native gate separation; [work history](../docs/development/CONSOLIDATION_P0_CLOSE_WORK_HISTORY/00000002.json) reconciled by original local feedback operation. Reuse only, candidate remains open; no new implementation or promotion.

- [BDH-L014 revision 10](learning/BDH-L014/00000010.json): actual P1a edit admission refusal followed by verified exclusive script and short Exec; [work feedback](../docs/development/CONSOLIDATION_P1A_WORK_HISTORY/00000002.json) reconciled. Reuse, no new implementation or promotion.

- [BDH-L014 revision 11](learning/BDH-L014/00000011.json): P1b proactive exclusive transfer and exact readback; [append-only corrected work feedback](../docs/development/CONSOLIDATION_P1B_SOURCE_WORK_HISTORY/00000003.json) reconciled. Reuse, no promotion.

- [Bounded Operation observation candidate](../docs/development/BDH_OPERATION_POLLING_CHECKPOINT.md): actual P1 short-wait repetition and budget exhaustion; owner-authorized verification Skill refinement. Candidate, ordinary-task observation pending; no automatic promotion or proven savings.

- [BDH-L012 revision 5](learning/BDH-L012/00000005.json): [consolidated CI observation](../docs/development/BDH_CI_OBSERVATION_CHECKPOINT.md), candidate refinement; independent benefit pending.

- [Early checkpoint candidate](../docs/development/BDH_EARLY_CHECKPOINT.md): actual P1 interruption and reuse; candidate refinement of existing resume Skill and template. Observe overhead and actual recovery; no proven benefit yet.

- [Finish Gate and compact resume candidate](../docs/development/BDH_FINISH_GATE_CHECKPOINT.md): owner-authorized bounded workflow refinement; regression and Eval evidence, independent review and real-task benefit pending.

- [BDH-L015](learning/BDH-L015/00000001.json): [staged batch candidate](../docs/development/BDH_STAGED_BATCH_CHECKPOINT.md); actual D30 byte comparison, real-task reuse and independent review pending.
