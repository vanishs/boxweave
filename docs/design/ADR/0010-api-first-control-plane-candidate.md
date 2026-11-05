# ADR-010 (candidate) — selective REST + MCP hybrid Provider architecture

**Status: OWNER-SELECTED COMPLEMENTARY LONG-TERM DIRECTION / FORMAL IMPLEMENTATION ACCEPTANCE PENDING.** Date: 2026-11-03. The owner chose to keep [ADR-003](0003-transport-and-trust.md) and ADR-010 at the **same architectural level**, with **different coexisting responsibilities** for the long term: ADR-003 preserves restricted upstream MCP/SSH transport and trust, while ADR-010 describes REST-backed app management and a separate SSH Box Executor. **Neither supersedes nor outranks the other.** The prior implementation of ADR-003 remains valid; existing production transports, release gates and [ADR-009](0009-remote-mcp-transports-and-auth.md) remain unchanged. This direction is not permission to migrate unverified operations.

**Parent:** [long-term architecture master](../API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md). **Implementation plan:** [Provider migration and parity](../API_FIRST_PROVIDER_MIGRATION_PLAN.md). **Observability:** [unified trace evolution](../UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md).

## Context

BoxWeave currently has two different upstream Containarium MCP consumers: its application Provider (typed service workflow through stdio tool calls) and the Agent-facing gateway (upstream MCP tool relay plus BoxWeave-specific protection and tools). Containarium's Protobuf/gRPC REST gateway exposes core management operations, some of which are missing as individually registered MCP tools. Some MCP tools, particularly `connect(exec)`, `push/sync` and `code_*`, also use SSH/local transfer and cannot be replaced by one lifecycle REST request.

The owner asked to consider a hypothetical radical rewrite: no obligation to preserve legacy database/interface compatibility, but preserve ~99% of user-visible capabilities. The previous in-place recommendation to continue upstream MCP-first was reevaluated under that assumption. Actual published releases, live hosts, credentials and third-party clients do not inherit those hypothetical freedoms without a separately approved deployment plan.

## Considered options

1. **Continue platform MCP for all internal calls.** Easiest immediate migration and upstream catalogue reuse; keeps tool schema/text translation and two different operation/diagnostic paths.
2. **Use Containarium CLI as internal execution API.** Familiar operational surface but subprocess output/privilege/mode ambiguity and some host-local direct Incus actions; not preferred as a management core.
3. **REST-first management with independent SSH Box executor (historical full-replacement candidate).** Structured machine contracts, but substantial migration, trust and tool parity costs; not the selected universal end state.
4. **Direct gRPC-first management.** Protobuf-generated types directly, but additional deployed mTLS/server configuration and runtime exposure; evaluate later with evidence.
5. **Selective REST + retained upstream MCP/SSH and inbound BoxWeave MCP (owner-selected hybrid).** Keep typed app lifecycle REST and add more adapters only with evidence; preserve specialist and advanced upstream MCP capabilities, with a shared claim/trace contract. No mandatory protocol conversion.
6. **Direct Incus/host shell.** Rejected: bypasses upstream ownership, authorization and safety boundaries.

## Owner-selected target responsibility split (implementation acceptance pending)

- Outbound basic **app-owned lifecycle already uses opt-in typed REST**. Tenant/Guard, routes, metrics, backup/snapshots are **selective REST candidates**, not mandatory replacements: keep upstream MCP if it is the supported/safer capability path, and require deployed route/auth/identity/postcondition evidence for any REST adapter.
- Agents continue using BoxWeave's own authenticated MCP (SSH stdio / Streamable HTTP OAuth). Where promised, mutating tools must be protected by a common operation/claim/UNKNOWN boundary; this is **not yet globally accepted**.
- Guest execution, files and detached coding runs **retain bounded existing MCP/SSH paths**. An independent exact Box-scoped SSH Executor is optional and requires separate trust, parity and negative tests; optional `agent-box` MCP remains independently authorized.
- A minimal host CLI repairs/diagnoses the installation independently; it cannot be an unrestricted Agent escape hatch.
- An explicit, allowlisted upstream MCP compatibility lane is a **long-lived, complementary channel**, not merely a temporary deprecated bridge. **Owner-selected Option A** preserves upstream MCP writes while progressively mediating each mutating tool through the shared Operation Core before dispatch; no untracked mutation bypass may be claimed as protected. For tools not yet safely mediated, coverage must be stated honestly and protected-mode writes must fail closed rather than bypass the guard. No blanket retirement of MCP writes was selected.
- A single common operation/identity/claim/receipt/trace contract spans Web, Agent MCP and CLI. D3 continues to diagnose inbound MCP and adds Provider API spans; D4 remains exact-Box preauthorized and bounded.
- The hypothetical 99% parity goal needs a baseline feature ledger; security invariants are non-negotiable even if a percentage threshold is met.

## Consequences and costs

Potential advantages **for selected migrated functions**: fewer protocol translations, typed status/error classification, direct REST capability checks and improved D1–D4 trace contracts. Upstream MCP tool renames no longer break the main application path. Management and guest execution failure planes become explicit.

Costs of the **historical full-rewrite alternative**: substantial Agent Gateway rewrite, independent SSH identity and host-key management, optional agent-box routing, OAuth/MCP parity testing, credential changes and API/runtime drift. Hybrid incremental changes avoid making these universal prerequisites, but retain their own security and diagnostics acceptance gates. REST status does not equal actual business completion; an API response alone is not a daemon-internal trace or a retry grant.

## Acceptance gates for full implementation and deployment

1. A0 complete inventory of current user-reachable app and forwarded upstream MCP capabilities, with immutable version reference and feature-level parity statuses.
2. A1 read-only authenticated REST proof on the **actual intended Containarium runtime**, including auth/TLS, empty inventory, identity and error/timeout handling.
3. A validated safe SSH/Box operation plan for `connect(exec)`, files, detached runs, missing helpers and Box replacement.
4. A unified operation-claim model covering **all** Agent/Web mutators with crash/unknown semantics and no blind replay.
5. D1–D4 correlation/secret-non-disclosure tests, plus operator-approved key custody for backup restore.
6. Independent isolated-host install/upgrade/rollback plan, no automatic Provider upgrade, and an explicit decision on intentionally unsupported features.
7. Keep ADR-003 and ADR-010 as **coequal, complementary records**: do **not** supersede ADR-003. Record the dispatch owner for each overlapping operation and verify coexistence during a long migration period; treat any future retirement or compatibility reduction as a *new* explicit owner decision.

## Current disposition

**Long-term REST + MCP hybrid architecture and Option A selected; no code/host change authorized by this record.** The existing MCP/SSH path and app-owned REST slice coexist by design; future REST extensions are per-operation decisions, not a plan to eliminate MCP. The [grouped backlog and priorities](../API_FIRST_PROVIDER_MIGRATION_PLAN.md#21-v0449-remaining-rest-opportunities-by-functional-group) identify current candidates and retained MCP work. Full protected mutation coverage requires separate evidence. The architecture has no global REST-over-MCP or MCP-over-REST ordering; the selected operation contract and security boundary determine the path. Accepting this design direction is not claiming completed API parity, universal protected mutations or a deployed migration.
