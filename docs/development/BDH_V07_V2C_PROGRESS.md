# BDH v0.7 V2c - terminal resolution and durable pending handoff

Status: Box source and fixture acceptance PASS; CI pending. Real selected
episode is not resolved, implemented or independently validated yet.

- Terminal review appends a separate immutable learning revision, binding
  the exact terminal task revision and SHA256, distinct review evidence
  and a matching later ordinary-task observation (attempt and evidence).
  Benefit-confirmed requires helpful feedback without a recorded residual;
  reopen requires residual or non-helpful/uncertain evidence. No implicit
  lifecycle promotion. Later fresh residual observations are not erased.
  History audit replays the selected task history against original learning
  snapshots and rejects missing/tampered terminal evidence.
- A single canonical episode history root per inbox (repository
  docs/development/BDH_V07_EPISODES, or external sibling episodes) is used
  to avoid competing local paths. Real subprocess contention tests show
  one claim wins at one revision. This is not a distributed lock: separate
  checkouts require Git reconciliation, and caller identity/evidence
  authenticity still needs independent review.
- Material work boundaries persist a derived evolve_handoff snapshot
  separately from feedback status. Pending selected work or exact matched
  review-due evidence remains pending; unrelated backlog does not block
  ordinary delivery. Missing/invalid inputs are unresolved, not closure.
- An unresolved UNKNOWN/blocked attempt is not silently retired. Terminal
  resolution currently requires all task steps recorded; interrupted or
  exhausted episodes retain explicit pending/blocked handoff. There is no
  automatic cancellation, background scheduler or new product permission.

## Box evidence

Retained owned Box 9d374e18-196a-493d-bf1f-2d582589f04d,
base 6f79c7cd62a200b5861c09b814aca384fe69ab24. Resume
b2800901aaaf0d93296617691377c371 exit 0: prior current head CI
45593412415 and 45593412465 SUCCESS. Earlier 53f96af quality run
45593107520 CANCELLED (superseded), artifacts 45593107525 SUCCESS.

- Core source 516f122cfbdedeedd02b603da6cd058b exit 0, 18 tests.
- Terminal/reopen and cross-process contention 87d8581f6a23ec0f403cc04f885c4ae2
  exit 0, 20 evolution tests.
- Durable boundary snapshot 2d90d29f17837807dbb97d42423d1042 exit 0,
  18 feedback tests.
- Full 89 tests, 14 Eval, audit OBSERVED with no findings, diff check
  bdef17233c901802dacbc5ece9cb75a2 exit 0. Same operation creates retained
  isolated exact proposal source worktree /root/pr193-bdh-v07-implementation
  at 53f96af0b107e081f487217d6f2582ab51770f36, clean. No parent changes.
- Actual three-payload residual feedback appended to L014 rev16;
  no duplicate selection, no synthetic witness. Protocol-1 boundary
  [work revision 2](BDH_V07_V2C_WORK_HISTORY/00000002.json) records
  FEEDBACK_RECORDED with independent Evolve PENDING handoff.

No merge, release, parent/runtime changes or independent review claim.
V3 must implement the actual selected episode in its exact-source lane;
V4 independent ordinary-task and comparative witness, V5 L4/CI remain open.
