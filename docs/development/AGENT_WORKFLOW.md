# Agent development workflow

Use [AGENTS.md](../../AGENTS.md) for stable boundaries. This workflow is optional and proportional to the requested task.

## Start-of-task checklist
Confirm target/authority, current source and observable acceptance. Inspect the relevant code; do not restart a historical phase or open unrelated tasks.

## Engineering standards
Use existing mechanisms, preserve ownership and original receipts, make focused changes, and test failure cases without weakening safety. For uncertain effects, reconcile the original before any retry.

## Fast development, strict acceptance
Run applicable format/source checks, then independently authorized installed-native checks if required by the change. Record exact source, binary, environment and observed result; source PASS is not native or release PASS. See [quality gates](../process/QUALITY_GATES.md).

## Development environment and Box lifecycle
Reuse an authorized isolated workspace if appropriate; local checkout, VM and CI are valid alternatives. Do not provision or destroy resources merely because a workflow recommends it.

## Version, publication and handoff
Report changes, checks, limitations and any unresolved original IDs. No automatic release, merge, new phase, learning task or indefinite wait. Preserve historical evidence; use [current status](../STATUS.md) for release readiness.
