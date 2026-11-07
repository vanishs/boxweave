# Proposed v1 implementation slices

**Status:** plan only. The `v1` branch has no runtime implementation. Each slice is subject to resource ownership, recursive parity and durable operation acceptance.

| Order | Slice | Main proof before claiming complete |
|---|---|---|
| 0 | Lock compatible official Go SDK revisions, native Incus capability matrix, privilege-boundary threat model and typed MCP schemas | Pinned module graph, compiled SDK smoke test, real native Incus version/extension probe, reviewed ownership invariants |
| 1 | Build process/service, local install/status/recovery CLI, super-admin bootstrap, TLS-authenticated HTTP MCP and restricted local privileges | Authenticated external MCP read; foreign resources and raw host requests denied |
| 2 | SQLite identity/ownership model and durable async operation journal | Crash-after-dispatch, duplicate requests, conflict, UNKNOWN recovery without replay |
| 3 | Box lifecycle, safe update fields, Incus Exec and verified native status | Create/start/Exec/stop/delete on exact owned UUID; denial and independent post-read |
| 4 | Private network topology, directional ACL grants, default internet, optional block | Real same-bridge IPv4/IPv6 denial/allow; IP reuse; no foreign NIC mutation |
| 5 | Box-owned Caddy public routes and native TCP/UDP forwards | Domain/port collision, cross-Box public allowance, restart reconciliation, foreign upstream rejection |
| 6 | Nested installation/upgrade compatibility and full per-layer feature parity (no cross-layer controller) | L0/L1/L2 full P0 matrix, L3 key paths if supported, parent/sibling denial; ordinary parent Box lifecycle |
| 7 | Independent system reacceptance, safe installer upgrade/repair, exact-source release readiness | Finder/Fixer/Verifier original repro, fresh install, rollback/recovery evidence |

A slice changing installed surfaces cannot be marked fully accepted from source tests alone. Once the minimal P0 product becomes usable, complete acceptance is defined by [ACCEPTANCE.md](../verification/ACCEPTANCE.md), not by a count of passed unit tests.

P1, separately scoped later: guest file operations and storage-driver-supported snapshots. Backups are not in scope.

The current v1.0.0 design baseline is intentionally not a production deliverable.
