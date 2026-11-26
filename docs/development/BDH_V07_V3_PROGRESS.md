# BDH v0.7 V3 - actual selected capability implementation

Status: implemented, controlled RC14 validation and integrated Box regression
PASS. Episode revision 5 has review ready. Independent review, delivery
acceptance and later ordinary-task benefit remain pending. No promotion.

## Scoped source and integration

Episode BDH-E-343f1501118ad8d8c3dfe6615c718dc0 was selected from L014 rev13/14.
Claim d4e02e83a264a2b0399d5350755e86ba exit 0 on exact clean proposal source
53f96af0b107e081f487217d6f2582ab51770f36 in retained worktree
/root/pr193-bdh-v07-implementation. Five selected source files only:
dev/transfer_plan.py, dev/skills/box-development/SKILL.md, dev/INDEX.md,
dev/README.md, scripts/test_bwdev_transfer_plan.py. The planner prepares
bounded explicit exclusive-create requests, manifest, stable action keys,
and original intent; assess checks supplied receipts/readback pages and
refuses unknown/conflicting evidence; assemble verifies ordered parts and
whole digest, uses exclusive same-directory publication and retains parts.
No network dispatch, runtime Tool, permissions or target replacement.

Scoped commit a464e6d6501d3757e3fd7af4b936547f23a97a1b, exact Box/GitHub
tree 8aace529e6452c4a1d0f12c896e1b8033524d279, retained branch
checkpoint/pr193-v3-scoped-implementation-20261125 so exact source is
reachable without merging or rebasing the paused parent. This is a source
checkpoint, not final CI/independent acceptance. The child PR integration
retains V2c; four scoped files are byte-identical, README adds the same
new section while preserving V2c documentation. Integration-only feedback
test fixture adaptations retain actual digest-drift, immutable binding and
unchanged-asset refusal checks. No guard weakened.

## Actual controlled RC14 transfer

Owned Box 9d374e18-196a-493d-bf1f-2d582589f04d, existing RC14 controller.
Transfer a6da47c1d4d4d78a69108788805aeefd, 81920 bytes, two 40960-byte
parts plus 550-byte manifest. Full SHA256
531f14e08fffb77fc24449060434ee5f346d30bfa55a136765e1398fd6a551a8.

| Action | Original Operation | Independent read tracking |
| --- | --- | --- |
| Plan and initial intent | bdae6dbd04b359110032c1381176efc2 | guest exit 0 |
| Part 0 | b9074d0f63c684178db1556fa936894c | b896c9d23aed8cf7d074185315949f1d |
| Resume / simulated unknown | 75a19588add55026fd3dd957d025a170 | guest exit 0; VERIFIED part 0, READY next; UNKNOWN becomes RECONCILE |
| Part 1 | 01b60fc38f0103e9eeb8742495eda2b6 | 6a0eae2ff4b7e8731cbdf443b0c593b8 |
| Manifest | d8cf0aa63aca6ac7ec34a68ff153d537 | 415ed366539a974902dc09f42bee444f |
| Assemble | 4f6a9eac6d5394710ac08a65dcd150e0 | ba1e340f08dbac3a38bbdf2e639a67ae, 128543276a06b82b990b54fd74e085b1 |
| Final assess and cmp | 7a94e21c7336b9839dbbc2ab24b00782 | guest exit 0, VERIFIED_READBACK |
| Exact source recheck and collision | 41da67515388ccb2cd8ae5c19f70523a | guest exit 0; intentional collision exit 2, target unchanged |

All three create Operations terminal SUCCESS; assembly exit 0 with both
streams read. Two final readback pages were independently retrieved and
their actual bytes fed to assess. Retained source/target and original
intent, pre-dispatch snapshots, receipts/readback pages and completed
journal in /tmp/pr193-v3-real-bundle. No native mid-write, restart,
power-loss or hostile concurrent guest proof. The between-call unknown
case was a deliberate synthetic journal state, not a real lost receipt.

Controlled run accounting: 18 core MCP calls (3 prepared request reads,
3 creates, 3 terminal reads, 3 artifact readbacks, assembly Exec/terminal/
two streams, 2 final page reads), 34 additional setup/intent/evidence/
assessment calls, total 52. No manually authored source payloads or
bespoke assembly program for this transfer; five manually encoded
receipt/readback evidence payloads were needed on this remote surface.
V0 10 calls had less instrumentation and different readback mechanics;
not a matched independent comparison. Total-call benefit is not proven;
the actual residual journal-orchestration overhead is recorded in L014
rev19, including metric provenance and uncertainty. Do not promote or
force a benefit result; investigate in V4 and reopen if needed.

## Verification and learning history

- Helper and discovery edits 77538e63be3a4321f0cd273ece4dcbb7 exit 0.
- 9 targeted tests d2d48fc536dc46e99547f804e28ac7c9 exit 0, repeated on
  clean exact scoped source in 41da67515388ccb2cd8ae5c19f70523a exit 0.
- Scoped pre-attachment full test 44f3a9cbb3214f8c6fc86421bf951db1
  exit 1 (95/96; old L014 attachment digest drift); preserve failure.
- L014 rev17 actual observation, rev18 work/attempt-bound new attachment,
  implement recorded in episode rev3, exact source operation
  f08d8e2f13307a4df5638a87f5b62848 exit 0. Audit clean.
- Initial integration acab93c9cca389be3005d0501d5561ac exit 1 due to
  transcribed episode path; original partial copies checked in
  259045bb19091b5131433edd68fe5d3c exit 0, only missing files copied.
- Integrated full 3894cb095f729d874eccfd53c31eb56a exit 1, two historical
  feedback test fixture mismatches (clone baseline older than actual
  asset, forged historical binding rejected rather than drift).
  Corrected fixtures test real asset drift and current baseline; no
  production guard change. c44983982ba05be0318cfde3862a5183 exit 0:
  98 full tests, 14 Eval PASS, audit OBSERVED no findings.
- Exact clean scoped recheck 41da67515388ccb2cd8ae5c19f70523a exit 0,
  9 tests/14 Eval, audit against external learning history, actual
  readback and refusal. L014 rev19 plus episode revision 5 (implement
  and validate RECORDED, review READY) operation
  6f3e39dcec8c39d358d46c9e7ec5c0fa exit 0.
- [L014 revision 19](../../dev/learning/BDH-L014/00000019.json);
  [episode revision 5](BDH_V07_EPISODES/BDH-E-343f1501118ad8d8c3dfe6615c718dc0/00000005.json);
  [work boundary](BDH_V07_V3_WORK_HISTORY/00000002.json) records
  FEEDBACK_RECORDED, Evolve PENDING, no independent closure.
- V2c exact child source 6d31613074aca9264b8eaefccdb5d405bd520aaf
  workflows 45597014501 and 45597014491 SUCCESS (observed in
  3894cb095f729d874eccfd53c31eb56a). New integration CI separate.

Next: independent Agent review exact scoped a464e6d and integrated child
source; claim review on retained exact scoped worktree using external
history, persist review and authorized delivery only after evidence.
Then fresh independent ordinary task with normal UA locator, matched
comparison and later exact feedback. If no benefit, reopen with evidence.
Independent L4 and exact-head CI remain required. Parent #191 unchanged;
no merge, release, host/runtime change or promotion.
