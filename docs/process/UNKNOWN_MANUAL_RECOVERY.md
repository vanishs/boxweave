# UNKNOWN manual recovery — approved minimal local-operator procedure

**Status:** owner-approved v1 design, **not implemented**. This is a host-local CLI/SQLite operation, **not** a new MCP tool, remote admin endpoint, replay daemon, or generalized lease management platform.

## Plain-language contract

An Agent asked BoxWeave to delete a Box. BoxWeave may have sent the request to Incus, but lost the reply. It is **not known** whether the Box was actually deleted. To prevent an overlapping create or delete from making the situation worse, BoxWeave retains a resource claim and the original `UNKNOWN` receipt.

The operator may **inspect and, only after a separate explicit confirmation, release that specific BoxWeave coordination claim**. Releasing it means “allow a separately authorized future request to be considered”; it **does not** mean “the original delete succeeded,” “Incus stopped,” “the original task was canceled,” or “please send the original delete again.” The original client idempotency key always returns its original receipt.

## Minimal protected procedure

1. **Identify** the original Tracking ID and Operation ID. From the already existing `operation_get`, `box_get` and `port_forward_list`, inspect sanitized phase evidence and the native Box/forward consequences. Never resolve an uncertain create/delete from a same-name or same-IP instance alone.
2. **Read only**: authorized host-local `boxweave recover inspect <operation_id>` (final CLI syntax may be adjusted before schema freeze) reads the original durable Operation, exact immutable native target UUID/incarnation, resource lock set, identity/receipt fingerprint, native observations and previous recovery audit. A mismatch, active native effect, foreign ownership or missing evidence remains explicitly reported.
3. The local tool issues a **short-lived, single-use challenge** bound to the exact original Operation + resource/identity fingerprint + reviewing operator. It reports a prominent warning: original native outcome may still be unknown. A stale/changed resource snapshot or reused challenge must be rejected.
4. **Independent operator decision**: after inspecting the exact native Box and related forward/peer state, the operator supplies a reason, challenge, literal exact-ID confirmation, and explicit acknowledgement that a previously dispatched action may still be running. Only this privileged local CLI path may release the exact local claim.
5. **Atomically append a protected audit record and release only that claim.** If the audit write fails, retain the lock. The audit includes time, reviewer, original Operation ID/state/receipt, exact claimed resource identities, reason and the release decision. Application-level writes never silently modify/delete prior audit events. Do not claim resistance to a fully privileged host root deleting the whole DB.
6. Recheck native state. The original Operation/Tracking IDs and original `UNKNOWN` state remain **unchanged** unless independently resolved through a separate evidence-based reconciliation. A **new** mutation uses a **new** client idempotency key and must still pass current ownership/identity/conflict checks. A release is never an automatic native retry, automatic cancellation, foreign resource adoption or rollback.

**Safe default:** If there is not enough independent evidence to justify the risk, **do not release**. A blocked recovery is preferable to possibly duplicating or interrupting a destructive native action. A confirmed native terminal outcome should be normally reconciled without forcing release.

## Evidence and test scope

Required negative tests: mismatched Box UUID/incarnation or same-name replacement; missing/expired/replayed challenge; claim changed after review; wrong reviewer/confirmation/reason; unverified native status; audit insertion failure; simultaneous release attempts; ordinary MCP token unable to invoke recovery; original idempotency key never re-dispatches after force release. Positive test: exact authorized local review releases only the chosen claim, persists one audit event, preserves original UNKNOWN, and a separate new authorized request must still pass new admission.

This generalizes the **principle**, not the legacy Python architecture, from [main `operation_coordination.py`](https://github.com/vanishs/boxweave/blob/main/bwmini/operation_coordination.py), [main real and simulated v0.45.0 R-04 recovery report](https://github.com/vanishs/boxweave/blob/main/docs/verification/R04_UNKNOWN_RECOVERY_0450_2026-11-05.md), and [main ADR-002 intent/reconciliation](https://github.com/vanishs/boxweave/blob/main/docs/design/ADR/0002-intent-and-reconciliation.md). Actual v1 installed acceptance is pending.

See [durable operations](../design/OPERATIONS.md), [security](../design/SECURITY.md) and [acceptance matrix](../verification/ACCEPTANCE.md).
