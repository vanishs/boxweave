# BoxWeave v1 — Peer provenance and internal Activity implementation (2026-11-19)

Status: **development implementation / verification in progress** on the sole Draft [PR #153](https://github.com/vanishs/boxweave/pull/153). This is not an RC, public MCP Activity launch, production migration, or a re-signoff of S4-B3. Read the [source lineage investigation](V1_MCP_TO_DB_LOG_LINEAGE_SOURCE_AUDIT_AND_IMPLEMENTATION_DESIGN_2026-11-19.md) first.

## Delivered source boundaries

- **SQLite Schema v5 (fresh pilot database):** `peer_grants.last_verified_operation_id TEXT REFERENCES operations(id)`, nullable. Only terminal, verified `box_peer_set` grant/revoke and root-completed UNKNOWN read-only reconciliation write this pointer in the same SQLite transaction as the original `SUCCEEDED` event. `peer_grants.operation_id` remains the in-flight/staged pointer. Regrant preserves the preceding verified pointer while PENDING; new independently verified settlement replaces it. An initial/unverified/unknown peer has **no fabricated verified original**. Historical earlier completed originals remain discoverable through their own operations/events; `last_verified_operation_id` is only the latest verified snapshot, not an immutable per-cycle timeline.
- **Evidence indexes chosen after a disposable 100,000-operation / 100,000-event / 100,000-recovery-row EXPLAIN fixture:** `operation_events_op_seq(operation_id,seq)` and `recovery_audit_op_seq(operation_id,seq)`. Before: `SCAN operation_events` and `SCAN recovery_audit` for exact original ID. After: `SEARCH ... USING INDEX ... (operation_id=?)`. The active issue query uses existing `operations_layer_time(layer,...)`; its correlated recovery predicate also changes from audit scan to indexed lookup. No other speculative indexes have been added. This represents the specified SQL fixture and planner, not a native-load benchmark.
- **Private `Store.ReadActivity` projection** in `internal/state/activity_query.go`: `overview` (bounded newest originals), `issues` (old active/UNKNOWN Operations, non-releasable Root Peer barriers, Exec output GC blockers and unresolved Peer ledger), and `evidence` (authorized original + ordered operation events, limited recovery decisions, resource claims, Exec/GC metadata and Peer latest verified link). One SQLite read-only transaction per view, Layer-scoped parameters, 1–50 row caps, incomplete coverage flags. Current access-token ID is checked before and after the snapshot. Raw command/file contents, review evidence, Root journal, live native effects and secrets are **not** exposed.
- **No changes to the 17 existing public MCP Tool contracts**, no new input parameters, no new public `box_activity_get` registration, no Root/PRS protocol or native journal changes.

## Tests and known limits

The isolated disposable L0 Box ran the nine targeted Go state tests (Peer verified grant/revoke/regrant and UNKNOWN reconciliation; Activity scope, age, evidence, truncation) **9/9 PASS** on source including the initial Activity implementation at `c9c3bffea6a362032101bb00fbf4318b273b3d97`. Subsequent token recheck and evidence indexes require the full exact-head CI and Go test results to be recorded separately; do not transfer the earlier PASS automatically. Schema v5 fingerprint/version fail-closed tests cover older/newer versions. Go tests and disposable SQL EXPLAIN are not host-root / native packet acceptance.

**Known private Activity limitations (intentional, not hidden):**

1. `attention_complete=false` always: the current internal `issues` projection has not yet proved complete coverage of all live network fences, forward dependencies, release distinctions and current native state. An empty result **is not** an all-clear.
2. `overview` and `issues` are bounded samples, with honest `truncated`; no client-safe signed pagination cursor has been implemented. Optional labels never grant Layer access.
3. Only the latest verified peer backlink is in the current ledger; historical superseded grants may have no direct current-ledger pointer. No read API can prove current kernel ACL packets, a Git tree, a Guest command's intent, or unjournaled earlier MCP reads.
4. A separate Owner approval is required before any public `box_activity_get` Tool, MCP SDK schema, or production exposure. If approved, require current authenticated Layer, token recheck just before the MCP response, adversarial filter/cursor/snapshot tests, OAuth refresh/revoke negatives, and S4 native/Root regressions.

## Mandatory Schema v5 adoption warning — do not silently migrate

**Existing Schema v4 controller SQLite installations are incompatible and must fail closed. There is no automatic v4→v5 migration or sanctioned live-database conversion in this PR.** This pre-release compatibility decision does **not** authorize destroying or adopting real data.

A separately approved fresh environment requires:

1. Inspect and preserve all original `operation_id` receipts, ownership, in-flight/UNKNOWN originals, host-root/PRS journal evidence and native Box/ACL state; stop the *candidate* controller before making any replacement. Safeguard a coherent copy of the existing controller SQLite database including WAL/SHM as applicable. Do not reset or rewrite active production/shared resources.
2. Provision a **separate clean SQLite Schema v5 controller database** and explicitly reviewed isolated Incus Project/Box fixtures. Never point new v5 binary at v4 data expecting silent repair or replay of UNKNOWN originals. Old DB, historical operations, leases, tokens and grants do not magically transfer; retain their protected evidence separately.
3. Re-create controller credentials/manual bearer tokens and OAuth public origin, clients, grants/access/refresh credentials as appropriate; reauthorize and reconnect external ChatGPT/MCP connector configuration, and rebuild only disposable Boxes after scope-specific approval. Existing credentials/connections should not be represented as retained.
4. Independently check native ownership, ACL, guest and Root/PRS boundaries. No host restart, shared ACL update, Scriptlet change, live Project takeover or deletion is authorized by this development note.

Do **not** merge PR #153 or publish an RC from this checkpoint. Record exact tested HEAD, CI results, any outstanding blockers and the final staged implementation decision before further rollout.
