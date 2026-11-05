# R-04 UNKNOWN safe recovery closure — v0.45.0 implementation and acceptance

**Implementation complete; independent isolated installed acceptance PASS for a real SSH Gateway UNKNOWN recovery.** This is the sole P0 item that combines preventing repeated writes during an UNKNOWN original operation and restoring service through explicitly reviewed human resolution. The implementation is deliberately **manual**, never a task replay or a generic automatic recovery engine.

## Installed v0.45.0 isolated live acceptance — 2026-11-05 UTC

**Scoped L PASS, not whole-system production certification.** Source-bound immutable [v0.45.0 Release](https://github.com/vanishs/boxweave/releases/tag/v0.45.0) has **25 published assets**, exact tag/source SHA `901662a243028a02a46062dcdb4e582530c32ae0`, and [GitHub Actions verify/publish success](https://github.com/vanishs/boxweave/actions/runs/41999464328). Local rebased source checks passed **755 Python tests**, knowledge validation and frontend build. **Stable default pin remained v0.43.1; the production owner host was not changed.**

- **Upgrade environment:** existing *test-owned* outer Box `bwr04-1104-gates2-container` installed official v0.44.9 → official v0.45.0 using pinned published `install.sh --version 0.45.0`; final installer **exit 0**, `boxweave version` `0.45.0`, published provenance SHA `901662a...` and `doctor --mcp` **10/10 PASS**. Provider baseline Containarium v0.128.0 unchanged. First installer refusal **`OPERATOR_REMOTE_SETTINGS_MOVE_TO_CONFIG_JSON`** was caused by earlier disposable-lab overrides in generated `runtime.json`; the operator-backed-up exact test-only HTTPS, lifecycle and coordination settings were moved to persistent `config.json`. An intervening retry encountered transient GitHub Release CDN HTTP 504 for `release.json`; verified published asset availability recovered and the same official installer succeeded on the next run. No code change, manifest bypass or relaxation of trust was used.
- **Existing history survived migration:** `sudo boxweave task provision` returned `ready`, added reviewed private `coordination_recovery_reviews` and `coordination_recovery_audit` tables plus deny-update/delete SQL triggers. The older independent `r04-replace` operation `344152969e5be069329bfcde312d93bf` remained **`dispatch_intent` with 2 claimed resources and zero recovery audits** before and after. Original separate lab `bwr04-1104-matrix` incident `33e97ea5e16bb753e053644d535740fc` was **not touched**.
- **Original real crash target:** independent pinned-host-key, enrolled test SSH MCP client targeted `r04-webfirst-container`, Provider UUID `c9fe2536-fc3f-45ae-9782-2d67ee1d8ace`, created `2026-11-05T07:12:30.44654013Z`; it stayed Running. Protected SSH catalog provided 15 tools, including deduplicated submission and lease, but **no agent-facing recovery tool**. A fresh tracked guest command was dispatched under original ID `86ed3534d1ba4cf820513dd1702164e1`; the coordinator had already committed `dispatch_intent` and **2 resource keys** when only that exact test Gateway process (PID 77348) was SIGKILLed. The real guest later wrote **one byte** despite the Gateway's lost completion response. Gateway task read surfaced **`UNKNOWN`**, original coordinator row remained `dispatch_intent`, and independent same-Box SSH write refused **`RECOVERY_REQUIRED`**; its denied marker was absent.
- **Real installed privileged operator review:** `sudo boxweave task recover inspect 86ed3534d1ba4cf820513dd1702164e1` returned a private 300-second challenge, original task ID, original `UNKNOWN` receipt, **`exact_original_task=true`**, **`identity_matches_claim=true`**, exact resource keys and the uncertainty warning. Its challenge was stored only in a private test file, never printed to public logs. `audit` was initially empty. Three deliberate negative confirmations were refused: missing risk acknowledgement and wrong exact confirmation each **`RECOVERY_CONFIRMATION_REQUIRED`** (exit 2); wrong one-use challenge **`RECOVERY_REVIEW_EXPIRED_OR_CHANGED`** (exit 2). The original guest write and immutable target identity had been independently inspected before approved override.
- **Real controlled release:** with original ID, valid challenge, operator/reason, exact `RELEASE <operation-id>` confirmation and explicit `--acknowledge-remote-may-still-run`, the installed CLI returned **`released_with_unknown_outcome`**, released **exactly 2 resource keys**, did **not cancel the guest** or modify the original receipt, and atomically stored audit event `f4a806cd4ad856238dbac7383cf2b476` identifying operator, reason, original state, identity digest, receipt ref and two resource keys. Independent SQLite and installed `task recover audit` readback both reported **1 audit row**. Original Gateway receipt remained `dispatch_intent` in the private raw row and **UNKNOWN** through `TaskStore`; original coordinator state still `dispatch_intent` with zero keys. The older `r04-replace` historical UNKNOWN still held its **2** keys, with **0** audit rows.
- **No replay and real resumed work:** resubmitting the *original* request key with identical payload returned **the identical original task ID** and cached `unknown` in **0.5 seconds**, not a new execution. A separately generated new request key successfully completed a real authenticated Gateway write under ID `9642148af5fa71206b7b9d8de1c407bc` (exit 0). Independent Incus guest inspection found original **1-byte** marker, new **7-byte** marker, no denied-write marker and CPU still 1. The test SSH authorized-key enrollment was revoked without affecting other keys; its private key and the private recovery challenge file were deleted. Private preupgrade SQLite backups and audited original task evidence remain in the disposable parent.

**Scope and remaining U:** The native recovery core, exact original Gateway linkage, fail-closed behavior and manual force-release **passed installed SSH Gateway crash/recovery testing**. Native application lifecycle/access-origin UNKNOWN recovery has targeted fixture coverage but **not an equivalent independently run installed failure/release case**; public OAuth/TLS, full Web-first three-way contention and production/global R-04 protection are separate unverified gates. An operator-authorized force release does **not** prove a remote process has stopped. Release `v0.45.0` is immutable; later documentation acceptance does not change its code or its stable installer pin.

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
| Unknown 2-stage review and exact confirmation | New challenge then successful release; no automatic replay | **T PASS; L PASS** official v0.45.0 SSH crash case |
| Invalid/expired/renewed challenge, changed state, wrong operator | Refuse; original two keys remain | **T PASS** fixture |
| Durable operator audit and transaction rollback on audit write failure | Never release without persisted audit | **T PASS** fixture |
| Two concurrent forced-release confirmations | Exactly one wins; exactly one audit row | **T PASS** fixture |
| Old v0.44.9 coordinator database upgraded in place | Add only recovery tables/triggers; retain old UNKNOWN/claims | **T PASS; L PASS** installed v0.44.9→v0.45.0 upgrade |
| Standalone installed root host command → unprivileged app helper | No root app code, no recovery tool on MCP | **T PASS; L PASS** installed sudo CLI plus protected MCP catalog |
| Authoritative live post-release success and refused negative | Original task unchanged, evidence remains; next fresh legitimate write may proceed after approved forced release | **L PASS** genuine crash, original guest readback, negative gate and resumed real write |

## Regression commands (developer source)

```sh
.venv/bin/python -m unittest -v tests.test_operation_coordination tests.test_gateway_tasks tests.test_host_cli tests.test_operation_leases
.venv/bin/python scripts/validate.py
python scripts/check_knowledge.py
```

**Do not use either pre-existing retained UNKNOWN in a test:** old parent `bwr04-1104-matrix` operation `33e97ea5e16bb753e053644d535740fc`, and new same-name replacement parent `bwr04-1104-gates2` operation `344152969e5be069329bfcde312d93bf`. They are evidence of earlier incidents, not a cleanup target. Installed recovery acceptance **completed using a separate new test-owned interrupted task**, with exact original claim, guest/Provider evidence and audit retained; do not use the two pre-existing incidents. Source, running/installed version, exact Provider incarnation and result must be recorded. Do not alter the owner-host stable pin or opt into production R-04 merely to exercise the CLI.
