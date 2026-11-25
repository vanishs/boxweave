# V4e runtime adaptation and feedback regression evidence

2026-11-25; scope PR #193 only. Canonical v1 9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d; starting child 8a054524a9c2253c2577d26ea5df5de2a0c7d11d; parent 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7. Prior [V4e investigation](https://github.com/vanishs/boxweave/pull/193#issuecomment-7176668144), independent REOPEN, L014 rev38 and episode rev17 are known; this is not a blind benefit witness.

## Feedback regression repair

- Original CI quality run 45714371589, job 139556539547: the two feedback tests expected PENDING from a selected episode, but live history moved through the second REOPEN. The current production selector correctly excludes an unrelated helped observation without residual evidence. The old tests had a moving lifecycle fixture.
- Owned Running Box 9d374e18-196a-493d-bf1f-2d582589f04d, project user-1001; RC14 3fb71a6ddf47d13d165e5c039144aa66fad0eb05. Preflight 47222868604057737d10169b4ef596e4, clean original checkout verification 675c426eaced426b59f5dc35573c236d (both exit 0; both complete streams). New isolated clone /root/pr193-v4e-ci-fix, exact starting head, Python 3.12.3 and Git 2.43.0. No parent checkout touched.
- Baseline 2c53062fefccb25edbcb2f3c04d448db, original tracking 1af2a01d84b7ccafbabfdd347ccfa574, guest exit 1, both original streams complete: reproduced both failures. Patch  cd23fc94639e9c228d7e62e8c0f1e9a9 exit 0. Pin only the lifecycle-sensitive tests to immutable learning revision 33 in private test copies. Add two explicit revision 38 cases: unrelated helped feedback remains recorded without matched pending; matching residual makes REVIEW_DUE pending. Keep exact-link and false-completion checks. No production logic changed.
- Feedback suite 442fe468bd22a8197a3947d437cb2f5a, original tracking 61b0a06b7ce07587ce19c7fe6c009b40, both streams complete, 20/20 tests, exit 0; git diff --check PASS. File SHA256 b5f5ec54961a0897952935569ef5f75d466171490b5b879108710fe5b5d5d22f. Verified whole-file MCP read 474987330d2e12767ff71268b9c3e256 (25097 bytes, same SHA256) before GitHub contents commit 52cd4c2511bff8f7432e3be37cdc1fc8ee05cad0. Only scripts/test_bwdev_feedback.py changed.
- Repository-wide format attempt 557f19207da034c13efdf62e2d064fc4 exit 1 because gofmt absent (no Go changes); changed-file format and full Python run 461488ed6a5a26eb4b98eaa2eaf8bce8, original tracking 8d93fb3f533f0d4ef705f92fd063d21b, FORMAT PASS: 1 selected tracked file, 318 tests, one unrelated failure and four skipped, exit 1. test_external_installer_plan_and_wrong_digest expects a clean fixture; populated Box returns existing installation. No change to installer or fixture acceptance. A concurrent read-only formatting Exec admission was explicitly refused due to unresolved resource claim; preserved original running test and observed terminal result before subsequent work. Exact-head CI must run full gates; no overall PASS asserted.

## Adaptation experiment: existing authorized connector to immutable evidence

Trial A uses a fixed, narrow read-only adapter in the current Agent call, no downloaded code execution, separate credentials or new MCP Tool. It verifies an actual original operation against its previously retained admission identity and records only sanitized evidence through the already authorized GitHub connector. No native mutation or unknown replay. The following checkpoint is revision 1 of the read-only experiment, not a trusted hot-path dispatch journal. Pin the created commit and reread it in a separate invocation, then independently query the original operation again. Do not infer executable reuse from Git storage.

```json
{
  "schema": 1,
  "revision": 1,
  "purpose": "read-only fixture, not dispatch authority",
  "source": "52cd4c2511bff8f7432e3be37cdc1fc8ee05cad0",
  "operation_id": "442fe468bd22a8197a3947d437cb2f5a",
  "original_tracking_id": "61b0a06b7ce07587ce19c7fe6c009b40",
  "lookup_tracking_id": "e2f332a28462b09d15bd62a2bab75c0a",
  "box_uuid": "9d374e18-196a-493d-bf1f-2d582589f04d",
  "incarnation": "9d374e18-196a-493d-bf1f-2d582589f04d",
  "action": "box_exec",
  "work_ref": "pr193-v4e-ci-fix",
  "state": "SUCCEEDED",
  "guest_exit_code": 0
}
```

Pending trial B: re-read the exact immutable Git commit, validate identity and issue a fresh read-only lookup. Evaluate callable loading and private caller-state limitations separately. Do not select a third improvement until genuine reusable callable plus trusted state bridge is proven.
