---
name: verification
description: Select bounded source or native verification and report exact evidence.
---

# Verification and CI recovery

Use current code and actual CI configuration. Run applicable format and focused regressions, then any required broader acceptance. A source check cannot establish installed-native behavior. Do not weaken tests or bypass protection.

## Consolidated CI observation (BDH-L012 candidate refinement)
Check the exact repository, PR, head and expected workflow once; distinguish missing, pending, skipped, failed and successful checks. Poll only with a finite budget and new evidence.

## Bounded Operation observation (candidate)
Use the original receipt and bounded reads. A timeout is not cancellation or failure proof; do not retry uncertain mutations.

## Batch boundary
Verify the actual changed scope; do not reuse stale results for a different revision.

## Staged source batch (candidate)
Optional helper: `python3 dev/staged_batch.py --help`. Inspect the current command and select the actual intended paths.

## Go failure navigation (candidate)
Optional helper: `python3 dev/go_failure_index.py --help`. Inspect the first relevant failure and rerun the smallest appropriate regression after a fix.

### Committed local batch (candidate)
Verify a clean exact range if using the helper. Do not claim unrun or unobserved checks as PASS.
