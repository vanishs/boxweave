# R-04 expired acknowledged operation claim synchronization — v0.44.8 candidate

**Scope and provenance:** 2026-11-04; previous release v0.44.7 source `4c1b88d06724ab93e4d997a99feefa54ced0bbe2`, project main at `26a127c25bf5c9be11deab51fdd59ba375a633ad` before this fix. Test development environment: task-owned isolated Ubuntu 24.04 parent `bwverify-0446-e2e-container` with exact installed v0.44.7, original application and shared coordinator retained. Local source work was performed in a separate feature branch; no installed component was patched for this regression reproduction.

## Reproduced defect

An already dispatched and acknowledged lifecycle create left `environment_operations.state=observing` and `coordination_operations.state=dispatch_intent`. When an observing operation's fixed deadline expired, `Lifecycle.work_one` directly updated its application receipt to `unknown/DEADLINE_EXCEEDED` and returned **without updating the shared coordination state**. The resource key remained claimed, so another Agent received `RESOURCE_BUSY` instead of the intended `RECOVERY_REQUIRED` for a known-uncertain terminal receipt.

The new test `test_observation_deadline_classifies_claim_as_unknown` reproduced the failure: expected coordination `unknown`, observed `dispatch_intent`; the first isolated run exited nonzero with exactly that assertion. No real Provider mutation or original UNKNOWN resource was modified during reproduction.

## Minimal repair and boundary

After the application receipt's deadline transition has been committed, update the shared claim from `dispatch_intent` to `unknown` **only for an operation that was already observing**. This happens outside the app's SQLite transaction. It does not release any resource claim, resend the original command, adopt any same-name Box, extend the deadline, relax identity or host-key validation, or touch the default stable selector. If the shared coordinator is inaccessible, the existing dispatch-intent claim remains retained and therefore still fails closed.

An independent regression test `test_expired_queued_action_never_claims_or_dispatches` asserts that expiry *before* dispatch produces a failed application receipt with neither Provider write nor shared claim. These tests complement the preexisting transient trust-recovery, persistent trust failure, cross-entry lease and same-key deduplication checks.

## Evidence and still-open gates

- **Reproduction:** 1 new shared-claim test failed against original v0.44.7 source with `dispatch_intent != unknown`.
- **Fix:** focused shared-claim suite 13/13 PASS, exit 0, in an isolated non-root Python 3.12 environment with copied non-secret, locked application dependencies.
- **Baseline:** 739/739 full Python tests passed before the second negative edge-case was added, using *test-process-only* legacy-policy masking for tests requiring default Gateway semantics; the five protected-policy-specific tests ran unmasked. Root-owned protected R-04 policy in the actual isolated installation was **never disabled**.
- **Pending for the exact v0.44.8 source:** complete 740-case repeat, knowledge check, GitHub source-bound CI + immutable release, independent official installer and real nested guest smoke. Do not relabel pending gates PASS until run.
- **Outside scope:** original `bwweb-0446`/`bwweb-0447` uncertain operations and their retained claims still require exact-incarnation reconciliation; this classification-only fix does **not** clear them. Real spontaneous first `TRUST_FAILED` to recovery and global concurrency combinations remain unverified. Outer host R-02 needs separate root provisioning. Containarium native create-after-error and disk shrink defects are record-only; no upstream edits.

New independent install target `bwfix-0448-e2e-container`, Ubuntu 24.04, 2 CPU / 4GB / 35GB, nested-capable after explicit stop/restart, is reserved for an immutable release-install test. The owner's outer host, Provider, network, SSH credentials and default `PUBLISHED_RELEASE=0.43.1` remain unchanged.
