# S1 — Runnable foundation, safe local install, identities and durable admission

**State: NOT_STARTED (2026-11-08).** **Execute S1 in one pass**: focused read → any required tagged SDK verification → Go implementation → tests → native/HTTP acceptance → repair/retest → log and commit. **This card does not authorize host upgrades or production installation.**

**Why first:** Every later tool must share the same ownership, authentication, Tracking ID, idempotency and durable UNKNOWN contract. Building those separately inside each later stage would recreate v0.x complexity.

## Read only when needed

**On entry (small set):** [Permanent boundary](../../decisions/PRODUCT_BOUNDARY.md), [technology stack](../../design/TECH_STACK.md), [architecture](../../design/ARCHITECTURE.md), [phase protocol](../PHASE_EXECUTION_PROTOCOL.md), and current [log checkpoint](../EXECUTION_LOG.md#current-checkpoint).

**At MCP/auth implementation:** [MCP binding](../../design/MCP_BINDING.md), [security](../../design/SECURITY.md), [Tracking/context](../../design/TRACKING_CONTEXT.md), [MCP P0 native HTTP evidence](../../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md). Fetch only relevant sections.

**At install/SQLite/Incus bootstrap:** [Dependency lifecycle](../../process/DEPENDENCY_LIFECYCLE.md), [operational defaults](../../decisions/DEFAULTS.md), [Incus integration](../../design/INCUS_INTEGRATION.md), [operations](../../design/OPERATIONS.md), and [manual UNKNOWN recovery](../../process/UNKNOWN_MANUAL_RECOVERY.md). For an actual unresolved API fact open [native 20/20 mapping](../../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) **only at that call**.

**At testing:** Acceptance IDs **A-001–004, A-016–019, A-022, A-032–034, A-071–072, A-075–077, A-080** from [ACCEPTANCE](../../verification/ACCEPTANCE.md). Phase 5 will re-run these on the final artifact.

## Deliverables — all belong in this phase

1. **Go skeleton that actually builds and runs**, with only the official Go MCP SDK and Incus Go SDK; clear typed MCP transport ↔ authenticated domain ↔ SQLite journal ↔ native Incus client boundaries. No custom JSON-RPC/MCP server, second BoxWeave REST control surface, Containarium dependency, tracing platform or unneeded abstraction.
2. **Authenticated Streamable HTTP MCP** on loopback only; new/legacy protocol interoperability when SDK supports it. Multiple revocable tokens with just **admin/viewer**, role and native resource ownership authorization for every handler via the actual SDK request actor; public errors contain safe Tracking ID. Unknown JSON inputs/invalid schemas rejected; Origin/Host/body bounds enforced.
3. **Durable SQLite** with explicit checked schema version, migration/backup and recovery behavior; exact owned-resource identity ledger and Box UUID/Project namespace constraints; appendable/redacted Operation transitions, distinct Tracking ID, MCP operation ID, native Incus operation ID, caller idempotency scope and original immutable intent. Insert admission before any side effect; repeated same key/actor/intent returns same operation, conflicting use errors. No native call from an unaudited handler.
4. **Worker boundary** independent of HTTP request lifetime: admitted mutation execution survives caller disconnect and reconciles UNKNOWN. Crash after admission, crash around dispatch, missing native reply and revoked token during queue must never cause blind redispatch. Implement the smallest correct queue/journal; no general distributed scheduler.
5. **Host-local installer/bootstrap/status/upgrade foundation**: compatible existing daemon reuse, preflight for restricted service user/Project, effective network/storage/root profile, API extension and dependency version ledger, actual binary/config checksum; protected token/SQLite permissions, no auto host reboot, no automatic foreign-Project adoption. Shared Incus upgrade, new root privilege and disruptive changes need separate operator authorization. The **one shared ACL + daemon-global Scriptlet installation** is reserved for S4, though S1 installer should expose an additive, safe local config/rollback hook.
6. **First public tool:** **host_status** with exact version/build/native capability/readiness data; no foreign host inventory. A narrowly named host-local CLI for install/status/upgrade/doctor is not a new public MCP tool.

## Prove, don't assume

- **Direct HTTP and negative auth:** modern discover/legacy initialize, list/call, missing/malformed/bad/viewer/revoked token, host/Origin/body cap, actor concurrency and typed error/Tracking ID. Reuse SDK proof as guidance; **run against the built v1 candidate** to count installed PASS.
- **Durability:** restart/kill the v1 worker after durable admission, after dispatch evidence and after native ACK but before SQLite commit; verify original IDs, replay protection, native postread or UNKNOWN and safe operator recovery. No fake-native tests alone counted as installed/native PASS.
- **Incus isolation:** the actual non-root service account can access only the prepared owned Project and cannot read foreign boxes, mount host resources, use incus-admin or mutate shared daemon policy. Check selected storage pool/root profile and effective values.
- **Installer:** clean host candidate, existing compatible daemon, incompatible and already-in-use host, interrupted upgrade/config rollback, no reboot, exact installed artifact. Stop at unapproved destructive host maintenance and record decision, continue other tests.
- **Read-only host_status:** exact installed build/version and API extension status distinguish healthy daemon from full app readiness.

## Phase exit and handoff

S1 is **PASS** only with a runnable installed development candidate, secured MCP, actual native access, SQLite admission and restart/UNKNOWN evidence, meaningful failure tests, and its documented local installer/status behavior. If shared-host installer acceptance needs new authority, leave that exact subcase BLOCKED and continue the rest; do **not** call overall S1 PASS or production-ready.

**Handoff to S2:** Git HEAD/candidate SHA, Go module pins, install artifact, SQLite schema/version, actor/role semantics, native Project and resource namespace, operation state machine and idempotency keys, all tests/evidence, original UNKNOWN claims and CLI leaves. Put this in the [append-only log](../EXECUTION_LOG.md) **before** changing phase. S2 must not reinterpret or duplicate S1's identity/journal contract.

**Not S1 work:** all Box lifecycle, guest files, peer ACL changes, native forwarding, nested controller orchestration, release publication and custom Incus/Cowsql repairs.
