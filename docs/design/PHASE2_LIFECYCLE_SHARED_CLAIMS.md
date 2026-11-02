# P1-B2/R-04 — opt-in app lifecycle shared resource claims

**Stage:** source candidate, NOT globally activated. It joins the existing app lifecycle `create/start/stop/delete/resize` worker to the same separately provisioned SQLite claim authority used by R-02 tracked SSH/HTTP Gateway requests and opt-in Web BoxActions. Original environment operation receipts and observation-only recovery are unchanged. No Provider commands or results enter the coordinator.

## Authority and conflict keys

When trusted app configuration sets `coordination.enabled=true`, the lifecycle worker verifies the root-published 45-second Provider/backend/project scope against the app's configured Provider, then reserves atomic scope-qualified resources before any remote mutation. Existing Boxes use the exact pinned incarnation `box:` plus `name:` keys; a not-yet-created Box uses a deterministic pre-creation identity with the logical `name:` key, avoiding a new-box bypass. A requested shared tenant adds `tenant:` where relevant. Independent other Boxes are not serialized by a global lock.

The original app operation becomes `dispatching` before remote dispatch; the shared claim's dispatch intent is committed before the first Provider write. Unavailable/corrupt claim DB, conflicting live or UNKNOWN operation, stale root scope, mismatched immutable target and invalid resources fail before the Provider call. No operation holds a SQLite transaction over the remote call, and the worker never retries a dispatched write on transport loss.

## Terminal verification

The original app operation must first persist a succeeded receipt backed by an observed desired state. Only then does the shared claim release. An in-progress observation retains the claim across polls/restarts. UNKNOWN, lost upstream response, unverified resize resource values, or nested-execution capability not actually verified remain held for explicit reconciliation. Acknowledgement alone is not completion. This intentionally favors a visible blocked state over unsafe automatic release.

## Source and rollout limits

This is a source-only opt-in adapter: default installations still behave as before until reviewed configuration and shared DB provisioning. Protected SSH/HTTP Gateway mode is also separately opt-in; their activation must be coordinated with the same user/DB/Provider identity. Other app access/network/port mutations, original unmanaged forwarded MCP writes outside protected policy, and direct external SSH outside the Gateway are not globally protected. Neither all-entrypoint R-04 acceptance nor a real installed host upgrade is claimed.

Test `tests/test_lifecycle_shared_claims.py` in a disposable Linux Box and whole CI for independent Boxes, same-name create collision, exact managed stop collision, pre-dispatch claim failure, lost create response, pending observation, unverified resize and stale root scope. Follow with installed Web/SSH/HTTP concurrency, old-to-new upgrades, original Provider ACK/recovery and rollback before live signoff.
