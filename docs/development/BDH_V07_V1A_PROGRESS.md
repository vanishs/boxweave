# BDH v0.7 V1a  residual evidence and review readiness

Status: first V1 source slice Box-verified; V1b durable opportunity decisions
and review-range reconciliation remain open. Do not claim V1-V5 closure.

## Implemented

- Backward-compatible optional residual metadata on new observations:
  explicit signature, impact, environment, cost with integer unit and
  provenance, and uncertainty. Unknown cost/environment/uncertainty
  remain null; old observations have no residual metadata and are not
  synthetically backfilled.
- History-derived advisory coverage at an implementation attachment or
  VERIFIED/ACTIVE review event. Coverage is a historical observation
  boundary, not proof of resolution or independent review. New typed
  residual evidence and negative outcomes can surface an opportunity
  beyond that boundary; positive reuse alone does not imply friction.
- Relevant DEFERRED new evidence prompts reconsideration even when
  implementation exists; STALE/asset drift prompts review. Relevant
  VERIFIED and ACTIVE can reappear with actual new review signals.
  No status transition, execution, selection, promotion or permission.
- learn next, work plan, start and related material feedback display the
  advisory opportunity separately from feedback status. Exact association
  still compares all five v0.6 observation facts; optional metadata does
  not break exact matching or weaken immutable observation history.

V1a intentionally does not persist a no-change/reject/select decision or
an improvement episode. V1b must add an explicit durable evidence review
cursor (with decision, reason, expected revision and reconsideration
condition) and test repeated review/no-change and new evidence. V2 must
bind a selected episode to existing task/checkpoint records and verify
scope and continuation. No automatic candidate selection or new product
Tool is introduced. Historical observations without typed residual cost
stay unknown; no semantic cost inference or independence score.

## Box-first validation

Environment: retained /root/pr193-bdh-v07 in Box
9d374e18-196a-493d-bf1f-2d582589f04d (RC14 controller unchanged).
V0 published c57c9900a00cfdf1ccd3214ca58b90a0ad8cea69 tree
e0a0f05e2e5fe75efd1ddaa4076534a27ff01b9b fetched and checkout verified;
original sync 8b2aa63e166791a9804955f4fff0efdd ended guest exit 2
only because sh did not expand grep brace syntax, after exact checkout.
Source read corrected; no retry of original effects.

- Core source edit 31e4a569ed8a5dcc72e390f9970bdfef exit 0.
- Targeted 8 tests d078a1f000172cb51b67c72d86411216 exit 0:
  coverage versus fresh residual, useful no-gap reuse, reviewed asset
  rediscovery, deferred/stale, relevance, invalid metadata and exact
  deduplication, unknown metrics and exact feedback matching.
- Full 77 tests, 14 Eval cases, audit and git diff --check
  3a2e7ab7314ac3a9fc679a4bc1bc1141 exit 0.
  Tested source tree dirty relative to V0; commit tree will be compared
  byte-for-byte to the staged Box tree, not claimed as installed proof.

## Actual typed feedback, not an independent witness

During V1a actual source edits, the existing L014 Skill was used twice
with distinct 10648- and 7957-byte script transfers, terminal receipts,
whole-file reads and short Exec. This is explicit harness work rather
than a fresh independent ordinary task. New L014 revision 13 was created
with expected rev12 in 27ea38aee31821199462102a4a636bc6 guest exit 0.
It records two separately constructed Base64 payloads as the limited
observed manual-actions metric, not total time or all orchestration.
No historical revision rewritten; candidate state CANDIDATE unchanged.

Actual work plan and learn next both return REVIEW_DUE, reason
explicit-residual, coverage_event 5, covered_observation_count 1,
observation_count 9, exact new evidence reference and next_action
review residual improvement opportunity. The old rev11 selector only
returned review linked implementation. This validates new advisory
routing on real evidence, not a new implementation or later utility.

See [L014 rev13](../../dev/learning/BDH-L014/00000013.json) and
[protocol-1 handoff](BDH_V07_V1A_WORK_HISTORY/00000002.json). Feedback
can still be RECORDED while opportunity review is due; V1b/V2 own
explicit review and durable assignment. No independent Agent witness,
automatic promotion or product/native acceptance claimed.

Exact-head child CI remains separate. Parent #191 paused unchanged at
3494ae8a35c50c386686c9d9d675c1ae37e1b1b7; no merge or release.

Initial boundary recording 52790340fc35e6f7f6f51958cc998afc exit 1
was refused for duplicate envelope evidence references (diagnosis
ce5a648d48c623407e3b80c67432db54 exit 2); revision 1 retained.
Corrected evidence list deduplicated references without changing facts;
protocol-1 FEEDBACK_RECORDED and REVIEW_DUE both verified.
