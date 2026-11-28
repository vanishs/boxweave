# Current development and acceptance plan

Use [current status](../STATUS.md) for exact source, published version and installed identity. S1–S5 and S2.5 milestone cards are historical; do not restart completed phases from old handoffs. Scoped Peer N1–N5, Doctor/Recover D0–D5, installation incident closure, scoped retirement and clean reinstall have merged. Prior results retain exact evidence scope.

## Active draft consolidation

[PR #191](https://github.com/vanishs/boxweave/pull/191) is the current
fresh-only schema-7 candidate. Follow the [P0-P5 plan](../development/CONSOLIDATION_PLAN.md),
[progress](../development/CONSOLIDATION_PROGRESS.md) and
[acceptance](../development/CONSOLIDATION_ACCEPTANCE.md). H remains frozen
pending Support and owner direction, C04 is deferred to P5, and final
release-specific evidence is separate. The published RC14 and merged v1
are not this candidate.

## Next authorized work after candidate acceptance

- Verify the exact authorized candidate and its immutable release assets; only then perform separately authorized Host recreation/deployment. Preserve unresolved native originals and dependencies.
- For defects or owner-approved features, open a bounded PR, inspect existing evidence, reproduce in authorized disposable Box/CI, implement/test/fix/retest, record exact receipts and limitations, and pass applicable CI before merge.
- Before stable v1 reconcile [acceptance](../verification/ACCEPTANCE.md), [risks](../verification/OPEN_RISKS.md), current installed/native evidence and [versioning](VERSIONING.md). Prior bounded substitutions remain scoped; do not infer physical acceptance.

See [phase protocol](../development/PHASE_EXECUTION_PROTOCOL.md), [workflow](../development/AGENT_WORKFLOW.md), [quality](QUALITY_GATES.md) and [historical phase cards](../history/development/phases/README.md).
