# BDH v0.7 V3c repair: rejected review and durable transfer recovery

Status: both P1 findings corrected and originating-Agent Box validation PASS.
Independent rereview and exact-head CI pending. Not independent acceptance,
delivery, benefit confirmation or promotion.

Original independent review:
https://github.com/vanishs/boxweave/pull/193#pullrequestreview-6056418511
integrated 0cdc49769c7a1fa5140ca5479817eb4d57454e45, scoped
a464e6d6501d3757e3fd7af4b936547f23a97a1b; logical reviewer
pr193-v3c-reviewer-20261125. Preserve FAIL, two P1 findings, original
probes fce5f680f7e89915a680183be361b24d (fixture error later in combined
probe) and 336dd26981846931e863ae97f1300d5b exit 0. No independent PASS
was imported or inferred.

## V3c-02 (first): rejection, rework, abandonment

- Explicit review/BLOCKED/rejected import permits actual external review
  from ready review, or matching current claim; exact implemented clean
  source, distinct reviewer/attempt and budget required. Not identity
  authentication. Normal blocked retry cannot erase known rejection.
- Explicit review/PENDING/rework on same rejected source, unique
  reconciliation reference, no downstream delivery, enough original
  budget; reset current DAG, preserve every immutable event and attempt.
  New implementation attachment and different descendant Git SHA,
  selected scope and fresh validation required before separate review.
- Independent pre-delivery abandon can bind known blocker/rejection or
  exhausted attempts with known failure to exact task revision/digest.
  No benefit field or invented witness. Reconsider/reselect becomes
  possible. Active, unknown and delivered effects cannot be abandoned.
  Original authority, budget and independent review constraints remain.
- Separate clean scoped checkout can be selected explicitly with
  --source-root for external inbox episode/attach. This avoids changing
  selected assets merely to update harness code.
- Actual Git fixture first/repaired commits and review/rework/validation/
  rereview, abandon/reselection and UNKNOWN refusal: 22 targeted tests
  5937586744a8a37f73ad5e070b0bbe6d exit 0, earlier existing 20 tests
  4699d67f31df835a91314e87106fdb81 exit 0.
- Actual episode BDH-E-343f1501118ad8d8c3dfe6615c718dc0: original rev5
  READY review; L014 rev20 links independent failure; rev6 REWORK_REQUIRED,
  rev7 READY implement, rev8 ACTIVE corrected claim:
  7895f2d662dc78abc9f70de5a5154b84 exit 0. Old attempts retained.
  Repaired source and new bound attachment L014 rev21, task rev9 READY
  validate (229e4bc879e7776daec05a8433bd145c exit 0). Task rev10 validation
  claim, rev11 validation recorded and review READY, 5 attempts used.
  [Latest episode](BDH_V07_EPISODES/BDH-E-343f1501118ad8d8c3dfe6615c718dc0/00000011.json)
  is not a successful review.

## V3c-01: canonical durable pre-dispatch journal

- plan creates exact canonical bundle and fsynced initial immutable event.
  assess takes the bundle only; old MANIFEST JOURNAL invocation and copied
  bundle path refuse. begin uses expected revision, exact request digest,
  exclusive link publication and fsync before returning a new intent.
  No dispatch by the helper. Original in-flight/uncertain state stops
  any new begin. Stale status cannot authorize a second dispatch.
- record appends actual receipt/readback with fixed original key, immutable
  original Operation identity, terminal state and known guest exit code;
  cannot reset to NOT_SUBMITTED or alter verified evidence. Unknown with
  no receipt stays reconciliation; no inference from existing file alone.
- 13 targeted tests: process exit after durable claim, concurrent claim
  conflict, local publication/fsync acknowledgement failure, stale/copy
  refusal, receipt updates and original identity, assembly after effect
  before receipt, corruption/missing/reorder/collision tests. First
  4a2a357f9d60bb591569765a5f424ee7 exit 0 (9 prior tests), expanded
  aa8b0c08345a94fc09f681fdddbc8a3a exit 0 (13).
- Scoped repair fcceaf11ec271a6ed7662580f65c59e8c8f164dd, tree
  bcf7cb36f318efabc72bc5149557c73128f080eb, direct descendant of a464e6d,
  retained checkpoint branch. Four changed paths inside original five
  selected assets; format 0107ccaacd1c2a1e4c6eab7d8b6f2ec0 exit 0.
  Child integration preserves V2c and adds lifecycle fix; exact planner,
  Skill and transfer tests bytes copied.

### Real RC14 controlled recovery, 4096 bytes

Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, original RC14 controller.
Transfer 3614572b3b17f43d34e74e8cb2624c53, one part, 439-byte manifest.
Full SHA256 e14845ea882b3ed869cc02fe8e3136194cb870eb060bf6f5be0fdfc45755bd14.

| Action | Original operation / read evidence |
| --- | --- |
| Clean scoped checkout, plan, begin part, before-dispatch RECONCILE | d3d9f0410dac4b923280ee3f2caae99a exit 0 |
| Actual part create | abe5ce549c61b920f3ca24cbc4e8af79 SUCCEEDED |
| After part effect before receipt, duplicate begin refused | c431da3b0ead3d5583bd20b9948c093c exit 0 (intentional refusal exit 2) |
| Actual part readback | 7e67d3cab5f6af98e49400b33913b008 |
| Append part receipt, begin manifest | 9de677280e442e6e6c9e83b462af58e3 exit 0 |
| Manifest create/readback | a8ede67b05b719873592007a6dfbe857 SUCCEEDED / 86013da17c3f044128ca829f9e9330be |
| Append manifest receipt, begin assembly, pre-effect RECONCILE | 92670349ef9605ad6bb30e109ec99e9b exit 0 |
| Actual assembly | 444c3eb51320112f8ad75140bf6a30db exit 0 |
| After assembly effect before receipt, duplicate begin refused | f1ca6bd7ad8002cf18ca32ed7457cdaa exit 0 (intentional refusal exit 2) |
| Independent final readback | b3ad676481dd6d6063740d42ff170b71 |
| Final canonical journal rev7 VERIFIED_READBACK, cmp, scoped 13 tests | ab54a131f90fcfba715a11813f56e966 exit 0 |

No native operation replayed. Original journal /tmp/pr193-v3c-real-bundle
and source/target retained. The receipt-save interruption was deliberately
tested between actual calls; no actual lost controller response, restart
or power-loss durability claimed. Process exit and fsync failure are
local fixture tests. Trusted canonical local storage only; no protection
against deliberate rollback/deletion of final history or cross-host
copies, no distributed locking/authentication. The offline helper does
not prevent a caller bypassing begin or replaying a connector call outside
the documented protocol. No change to native admission or idempotency.

## Final originating-Agent evidence and next action

Integrated full 104 tests, Eval 14, audit OBSERVED []:
229e4bc879e7776daec05a8433bd145c exit 0. Exact-source post-claim recheck
5a1daa18d52ab66085d30532b2d0e3ae exit 0: canonical journal rev7 verified,
cmp and 13 scoped tests. L014 rev22 actual repair feedback includes
three manual receipt payloads, no comparison with V0 or independent
utility. L014 rev23 binds the exact originating implementation observation. First boundary
revision 2 rejected missing originating-attempt association (d942ea239c90f768cb07cd02796203d2 exit 1); preserved, not rewritten.
[Corrected work handoff](BDH_V07_V3C_REWORK_HISTORY/00000003.json) records
FEEDBACK_RECORDED and Evolve PENDING. No benefit or promotion.
Next: independent rereview both P1 fixes on exact integrated child
and corrected scoped source; only then delivery and V4 ordinary-task
matched benefit measurement. V5 remains open. Parent #191 unchanged.
