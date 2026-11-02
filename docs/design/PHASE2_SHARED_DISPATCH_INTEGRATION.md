# R-04 cross-transport shared claims — tracked-exec source slice

**State:** Stage 2 partial. This slice connects the existing R-02 SSH/HTTP gateway **tracked exec** to the single P1 operation coordinator. It does NOT authorize the root operator to enable global mutation coordination, nor is it a full multi-Agent guarantee.

## Reviewed source identity

The root host-key reconciler publishes an additional nonsecret `scope={provider:"containarium",backend:actual_host_name,project:"default"}` in its existing 45-second registry. This is consistent with the installed default app's Provider scope; custom app Provider/backend/project configurations that differ fail the shared identity check. `coordination_bridge.app_box` and `gateway_box` compute the **same exact incarnation key** using scope, logical, name, created timestamp and SSH user. They also reserve a scope-qualified logical-name resource so recreation cannot bypass an unresolved claim. No caller supplies the root-published scope or a new host credential.

## Gateway tracked-exec transaction

A trusted `boxweave_task_submit` acquires two shared SQLite resources atomically before the original `connect(exec)` dispatch and commits dispatch intent in both the original metadata receipt and shared coordination DB. Absent/corrupt/unprovisioned coordination DB, mismatched scope, active lease, another accepted operation or unresolved UNKNOWN all fail closed before dispatch. The gateway runs the original Provider adapter, persists its known exit code (no stdout/stderr body), then releases the claim only after the original receipt is committed. Transport loss produces UNKNOWN and keeps the claim; a failed post-result claim release is conservative until verified reconciliation. Old request keys submitted under earlier identity hash may be refused as conflicts, never replayed.

The shared claims DB remains separately provisioned using the existing `sudo boxweave task provision` entry; no implicit host permission changes, no elevated root process reading app SQLite, no new RPC daemon. The Box lease core remains internal and is not advertised as globally safe.

## Coverage boundary

**Protected by this slice:** `boxweave_task_submit` from both SSH MCP and authenticated HTTP MCP (which execute the same gateway code), when running the new root-published helper and explicit private provisioner.

**NOT protected:** legacy direct `connect(exec)`, `create_container`, `resize_container`, `boxweave_nesting apply`, other forwarded mutators, Web/app BoxActions and lifecycle/access writes. A fixture Web actor can acquire the same shared resource, but actual Web worker integration and root activation are pending. Those bypasses mean global R-04 protection is NOT ACCEPTED. Do not enable an operator-visible lease tool yet.

## Validation

Source fixtures must prove app-vs-gateway canonical identity, two concurrent SQLite connections on the same Box (one accepted, one RESOURCE_BUSY), different Boxes accepted, UNKNOWN preserving resource, expiry not releasing unknown, missing DB failing before dispatch, conflicting historical incarnation/name, wrong scope refusing, duplicate request keys not redispatched, and standalone installed helper imports. Linux self-tests and full CI are distinct from real installed SSH/HTTP/Web acceptance, version compatibility and crash injection. Stable installer remains v0.28.0 until separately promoted.
