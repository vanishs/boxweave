# ADR-002 — Transactional intent, observation after uncertainty

Status: selected for phase 4 validation. Inputs: REQ-004, REQ-005, REQ-006, REQ-007, REQ-010, REQ-012.

Options: JSON files + background threads; local SQLite + explicit worker; external queue/PostgreSQL. Select local SQLite with transactions/CAS, a single mutation worker and rollback journal/FULL synchronous initially. One host/one operator does not justify distributed infrastructure. SQLite has one writer; keep provider calls outside transactions ([official concurrency documentation](https://sqlite.org/wal.html), SRC-028). WAL could improve read concurrency, but adds checkpoint/shared-files concerns; revisit only after measured contention. DB stays on local storage.

The local database cannot atomically commit a remote provider write. Persist intent and dispatch marker before call. Crash between marker and send is unknown even if nothing was sent. Lease expiration never authorizes replay. Reconcile by exact target and provider evidence; absence alone cannot prove a delayed create will not complete. Provider idempotency may be used only if its actual contract is proven. No 'exactly once' guarantee is advertised.

Data revisions prevent lost local updates; provider revisions enforce atomic policy edits. No multiple-call substitute for native atomic Tenant access. Store evidence timestamps and nullable exits; resumed phase deadlines never reset. Synchronous DB-full/write failure prevents dispatch if intent wasn't durably stored. On busy timeout return retryable local conflict before send, not a duplicate remote operation.

Validation: TASK-002 crash points before intent, after intent, during response loss, and after remote commit; verify external call count and preserved unknown. TASK-006 delayed create and TASK-009 concurrent policy fixtures. Phase 4 proves receipt persistence on restart with a real local DB; no disk power-loss or production provider claim is made by fixtures.
