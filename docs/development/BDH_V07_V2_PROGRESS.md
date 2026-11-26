# BDH v0.7 V1b / V2a / V2b - durable review and guarded improvement tasks

Status: Box-tested source slice; child CI pending. V1 review decisions and
V2 selection/materialization/guarded task recording implemented. V2 terminal
resolution and end-to-end independent evidence remain open. No promotion or
automatic dispatch is provided.

## Design and constraints

- V1b: explicit preserve/defer/reject/select decisions append to existing
  learning history, with expected revision and deterministic basis bound
  to exact prior evidence and asset digest. Historical coverage is not
  independent proof. Fresh significant evidence or an explicit distinct
  trigger can reopen a review; deferral also reconsiders new observations.
  No candidate lifecycle state changes automatically.
- V2a: selection requires a bounded proposal, exact clean Git source,
  candidate asset, explicit authority reference, risk, baseline, expected
  outcome, attempt budget and acceptance for implement/validate/review/
  deliver/benefit. Stable episode ID is derived from candidate and basis.
  Existing local immutable task revisions materialize one ready DAG.
  Repeated materialization validates existing history; no generic task
  init/replan/transition can bypass the selected episode lifecycle.
- V2b: explicit transitions bind candidate/task revisions, claim actor
  and attempt, checkpoint, evidence and authority reference. New claims
  require ready dependencies, unique attempts and budget. Source ancestry
  and changed paths must fit the selected scope. Implementation recording
  requires a new exact work/attempt-bound attachment and actual asset
  digest, not the historical Skill attachment. Later steps require the
  implemented source. Review and benefit require distinct actor references
  from implementers (benefit also distinct from reviewer).
- A blocked/unknown attempt is preserved; no direct reset/replay. Explicit
  reconciliation with new evidence is required, and an unknown outcome
  requires a separate actor reference. Read history reconstructs each
  immutable task revision from original events and learning snapshots.
- Actor/authority/evidence references are caller assertions; they do not
  authenticate people, authorize effects or establish real independence.
  Local exclusive publication is not a distributed lock or Git sync.
  No background Agent, product Tool, native privilege or merge/release.
- Final recorded task outcomes deliberately remain
  OUTCOMES_RECORDED_REVIEW_REQUIRED. V2c still needs explicit durable
  terminal resolution/reopen, integration of outstanding opportunity
  review into ordinary work boundaries, cross-process episode contention
  and persistence/reconciliation acceptance. V3/V4 actual implementation
  and independent utility evidence cannot be inferred from fixture tests.

## Box evidence

Exact base eb5ea275251a08219a0b5f5adff23b25e426a438 in
/root/pr193-bdh-v07, owned Box 9d374e18-196a-493d-bf1f-2d582589f04d,
RC14 controller unchanged. Initial clean checkout and actual L014
REVIEW_DUE: 61408c4a0799d8cdef3876b75d4be13e exit 0.

- V1b core 643225df52ecf93a082489de3733e1a4 exit 0, 8 tests;
  decision tests dad286b1fc9ec8188be50bbb7ed57c5c exit 0, 12 tests.
- V2a 13df9eebd8da5e501346f3b4360cf875 exit 0, 14 tests.
- V2b source 094164aca9ee43c413bbb8abb3168122 exit 0; found
  transition parameter shadowing on source inspection, corrected in
  0ff6e1784becb6bc38afaf035e1516c4 exit 0 before tests.
- 18 targeted tests a755b341d6907b420a3008b5868942c1 exit 0:
  decision history, stale/duplicate basis, defer/reopen, bounded scope and
  budget, selected DAG identity, unknown reconciliation, wrong-source/
  actor/attempt/dependency refusal, work-bound implementation, distinct
  review/benefit actors and no false closure. Actual temporary Git fixture
  verifies scoped vs out-of-scope commits. Other lifecycle cases use
  controlled fixture references; no independent Agent witness.
- Full regression c81bdbab5592723ce82431638e49ca24 exit 0: 87 tests,
  14 Eval PASS, audit OBSERVED with no findings, diff check PASS.
  Previous V1a push workflows 45591509198 and 45591509140 both SUCCESS.
  Tests exercised the edited source; exact staged Box tree must match
  GitHub tree at publication.
- Actual L014 rev13 review basis and evidence read in
  6dc46971384db06ab6e10e35bf01616a exit 0. New actual distinct six-payload
  observation appended in rev14, without claiming new capability benefit.
  [L014 rev14](../../dev/learning/BDH-L014/00000014.json);
  [protocol-1 work handoff](BDH_V07_V2_WORK_HISTORY/00000002.json).

## Continue

Reconcile exact child CI and clean checkout. Review real L014 evidence and
select a small bounded client-side manifest/planner if justified, without
assuming new product permissions. Keep actual proposal and ready task
history in a durable child PR checkpoint. Complete V2c terminal resolution
and integration safety; then V3 implement/test a real selected improvement,
V4 separate ordinary-task Agent and matched benefit, V5 independent L4
review and exact-head CI. Parent #191 remains paused at
3494ae8a35c50c386686c9d9d675c1ae37e1b1b7. No parent, runtime, merge or release.
