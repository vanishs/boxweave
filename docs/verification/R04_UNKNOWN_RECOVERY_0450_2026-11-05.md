# R-04 UNKNOWN safe recovery closure — v0.45.0 implementation and acceptance

**Source feature implementation; live release-installed result recorded separately below only after execution.** This is the sole P0 item that combines preventing repeated writes during an UNKNOWN original operation and restoring service through explicitly reviewed human resolution. The implementation is deliberately **manual**, never a task replay or a generic automatic recovery engine.

## Approved decision and command contract

Owner approved, 2026-11-04 Pacific: when the original execution result cannot be established, the local administrator **may force release the exact UNKNOWN operation claim** after reviewing the original task/Box evidence, accepting that a remote process may still run, performing an exact second confirmation, and durably recording the review. A normal lease expiry, new request key, Gateway restart, or client retry **must not** trigger forced release.

Use a trusted interactive local administrator shell on the **correct** installed BoxWeave host; do not enter the challenge in Agent chat, a remote MCP tool or a public log. Copy the exact original 32-hex operation ID from the original task/receipt. The commands execute as the dedicated `boxweave` account through a root-owned verified helper; the unprivileged application user, the local MCP Gateway and Web APIs do not gain a new recovery tool.

```sh
# 1. One-time post-upgrade migration of the private coordinator; idempotent.
sudo boxweave task provision

# 2. Inspect the original unknown; returns matched original metadata if available,
#    exact resource keys, immutable identity digest, and a short-lived challenge.
sudo boxweave task recover inspect ORIGINAL_32_HEX_OPERATION_ID

# 3. Independently inspect the original Box and task/process/guest state, and
#    decide whether uncertainty really warrants a manual override.
#    Do not rely on a Box with the same name but a different incarnation.
sudo boxweave box inspect ACTUAL_INSTANCE_NAME

# 4. Explicit irreversible claim release; substitute original ID, challenge,
#    reviewer-approved reason and the exact literal confirmation.
sudo boxweave task recover release ORIGINAL_32_HEX_OPERATION_ID \
  --challenge CHALLENGE_FROM_INSPECT \
  --reason 'Original remote result unprovable; independently reviewed Box state' \
  --confirm 'RELEASE ORIGINAL_32_HEX_OPERATION_ID' \
  --acknowledge-remote-may-still-run

# 5. Read the persisted operator/action/resource-key audit record.
sudo boxweave task recover audit ORIGINAL_32_HEX_OPERATION_ID
```

`inspect` issues a one-use challenge valid for **300 seconds**, linked to the exact original operation, its current UNKNOWN/dispatch state, identity digest, receipt reference, and active canonical resource keys. A newer inspection invalidates an older challenge. A changed claim, expired challenge, different reviewer, missing risk acknowledgement, wrong exact confirmation or inadequate audit reason fails without releasing anything. `SUDO_USER` (or direct root) identifies the operator; caller-supplied arbitrary actor IDs cannot be accepted by the root host entrypoint. The app-account helper has no Provider side effects.

**What `inspect` proves:** the original stored coordinator claim and resource keys. For a Gateway task, the original task database is read and the exact original task ID and identity digest are matched. The result says explicitly `exact_original_task` and `identity_matches_claim`; absent or different original evidence stays `unverified`. Other app receipts can be linked by exact ID but do not automatically prove Provider incarnation: their `identity_matches_claim` is false until independently established. The immutable identity digest cannot substitute for an independently observed original Provider UUID/createdAt and cannot prove the remote process has stopped. Verify that separately before deciding.

**What `release` does:** in one SQLite transaction, atomically insert an append-only (SQL UPDATE/DELETE denied) audit record containing original operation, reviewer, reason, exact original state, identity digest, original receipt, released resource keys and UTC-compatible epoch time, then delete just the original operation's resource claims and invalidate the challenge. Its original durable receipt and original UNKNOWN/dispatch state remain **unchanged**, so the original task remains visibly UNKNOWN and the request key remains idempotent. It does not run, cancel, roll back, restart or retry a remote command; it does not clear leases or other operation claims. Database audit triggers are a defense against ordinary UPDATE/DELETE, not a claim of tamper-proof WORM storage from host root.

**Safety boundary:** a forced release is an *operator-authorized exception* after reviewing the risk, not an assertion of execution success or a guarantee against a still-running guest. The helper returns `released_with_unknown_outcome`, never `completed`. The operator must not force release an unreviewed original. Ordinary Agents still receive `RECOVERY_REQUIRED` before this explicit act.

## Acceptance matrix

| Test | Expected result | Evidence state |
| --- | --- | --- |
| Existing real interrupted Gateway after dispatch | Original receipt UNKNOWN and two keys block same-name writes | **L PASS** v0.44.9, retained test-only scene; not force-released |
| Unknown 2-stage review and exact confirmation | New challenge then successful release; no automatic replay | **T PASS** fixture; **L U** until installed test |
| Invalid/expired/renewed challenge, changed state, wrong operator | Refuse; original two keys remain | **T PASS** fixture |
| Durable operator audit and transaction rollback on audit write failure | Never release without persisted audit | **T PASS** fixture |
| Two concurrent forced-release confirmations | Exactly one wins; exactly one audit row | **T PASS** fixture |
| Old v0.44.9 coordinator database upgraded in place | Add only recovery tables/triggers; retain old UNKNOWN/claims | **T PASS** fixture; **L U** until official upgrade |
| Standalone installed root host command → unprivileged app helper | No root app code, no recovery tool on MCP | **T PASS** mocked boundary; **L U** until installed test |
| Authoritative live post-release success and refused negative | Original task unchanged, evidence remains; next fresh legitimate write may proceed after approved forced release | **L U** until separately recorded |

## Regression commands (developer source)

```sh
.venv/bin/python -m unittest -v tests.test_operation_coordination tests.test_gateway_tasks tests.test_host_cli tests.test_operation_leases
.venv/bin/python scripts/validate.py
python scripts/check_knowledge.py
```

**Do not use either pre-existing retained UNKNOWN in a test:** old parent `bwr04-1104-matrix` operation `33e97ea5e16bb753e053644d535740fc`, and new same-name replacement parent `bwr04-1104-gates2` operation `344152969e5be069329bfcde312d93bf`. They are evidence of earlier incidents, not a cleanup target. For installed recovery acceptance, create an **additional separate test-owned interrupted task**, inspect, force release and observe real nested guest/Provider state before and after. Source, running/installed version, exact Provider incarnation and result must be recorded. Do not alter the owner-host stable pin or opt into production R-04 merely to exercise the CLI.
