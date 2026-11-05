# ADR-010 — MCP-first + REST-assisted hybrid Provider policy (historical API-first candidate)

> **2026-11-05 DECISION UPDATE: MCP-first + REST-assisted hybrid SELECTED; universal API-first and 29-tool bulk REST migration ABANDONED.** The owner cancelled the proposed 8+21 interface migration; [PR #77](https://github.com/vanishs/boxweave/pull/77) is closed, its merge commit `020ebfe8` having been reverted by `e19e60bb` on `main`. The existing app REST lifecycle subset stays implemented; specialist upstream MCP stays default. The previously chosen long-term **hybrid coexistence** and equal standing of ADR-003/ADR-010 for their *already supported* scopes remain in force. Any new targeted REST adapter needs its own justification and acceptance, not authority from this candidate ADR.

**Status: OWNER-SELECTED ARCHITECTURAL POLICY (2026-11-05); implementation and installed acceptance remain feature-specific.** Historical proposal date: 2026-11-03. The owner chose to keep [ADR-003](0003-transport-and-trust.md) and ADR-010 at the **same architectural level**, with **different coexisting responsibilities** for the long term: ADR-003 preserves restricted upstream MCP/SSH transport and trust, while ADR-010 describes REST-backed app management and a separate SSH Box Executor. **Neither supersedes nor outranks the other.** The prior implementation of ADR-003 remains valid; existing production transports, release gates and [ADR-009](0009-remote-mcp-transports-and-auth.md) remain unchanged. This direction is not permission to migrate unverified operations.

**Current owners:** [authoritative architecture](../ARCHITECTURE.md), [trust and transport ADR-003](0003-transport-and-trust.md). **Archived research:** [former API-first master](../API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md), [abandoned bulk REST migration](../API_FIRST_PROVIDER_MIGRATION_PLAN.md). **Independent diagnostics:** [unified trace evolution](../UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md).

## Selected decision — MCP-first + REST-assisted (2026-11-05)

- Keep **upstream Containarium MCP as the normal outbound interface for new/specialist tools** and retain Agent-facing BoxWeave MCP over SSH/stdio and authenticated HTTP. MCP is the preferred default **integration policy**, not a claim that the MCP protocol alone supplies least privilege or safer mutations.
- Keep the seven **already implemented** app-owned direct REST lifecycle operations (`list/get/create/start/stop/delete/resize`) unchanged when the reviewed localhost API binding is configured. REST remains available **as an auxiliary per-feature adapter** when MCP demonstrably lacks a required capability or a typed REST integration has independently justified observability, performance or contract benefits. Document scope, capability, permissions, degradation and negative tests before any additional REST route is approved.
- **Abandon** conversion of the remaining **8 P1 + 21 P2 (29) MCP tools** as a batch; no default cutover, end-state REST-only target, mandatory new SSH Executor or removal of supported upstream MCP tools. Preserve the prior findings as archived research, not work orders. [PR #77](https://github.com/vanishs/boxweave/pull/77) was merged then reverted on `main` (`020ebfe8` / `e19e60bb`).
- Keep **one reviewed intent/identity/claim/UNKNOWN safety contract** across any BoxWeave-controlled writes regardless of selected protocol. R-04 is **not globally accepted just because MCP is preferred**; raw upstream Provider/Incus access remains outside BoxWeave interception. After a write possibly dispatched by REST or MCP, **never blindly retry or switch transport**; reconcile the original operation.
- ADR-003 remains **coequal in decision authority**, and continues to own existing MCP/SSH transport and trust. This ADR states **which outbound integration to prefer**; “MCP-first” is a default implementation choice, **not an ADR precedence or automatic downgrade rule**.

## Historical API-first comparison (2026-11-03; superseded by decision above)

## Context

BoxWeave currently has two different upstream Containarium MCP consumers: its application Provider (typed service workflow through stdio tool calls) and the Agent-facing gateway (upstream MCP tool relay plus BoxWeave-specific protection and tools). Containarium's Protobuf/gRPC REST gateway exposes core management operations, some of which are missing as individually registered MCP tools. Some MCP tools, particularly `connect(exec)`, `push/sync` and `code_*`, also use SSH/local transfer and cannot be replaced by one lifecycle REST request.

The owner asked to consider a hypothetical radical rewrite: no obligation to preserve legacy database/interface compatibility, but preserve ~99% of user-visible capabilities. The previous in-place recommendation to continue upstream MCP-first was reevaluated under that assumption. Actual published releases, live hosts, credentials and third-party clients do not inherit those hypothetical freedoms without a separately approved deployment plan.

## Considered options

1. **Continue platform MCP for all internal calls.** Easiest immediate migration and upstream catalogue reuse; keeps tool schema/text translation and two different operation/diagnostic paths.
2. **Use Containarium CLI as internal execution API.** Familiar operational surface but subprocess output/privilege/mode ambiguity and some host-local direct Incus actions; not preferred as a management core.
3. **REST-first management, SSH-first Box executor, BoxWeave-owned MCP outward (historical rejected candidate).** Structured machine contract and one operation/trace boundary; requires separate SSH implementation and careful gateway feature parity.
4. **Direct gRPC-first management.** Protobuf-generated types directly, but additional deployed mTLS/server configuration and runtime exposure; evaluate later with evidence.
5. **Direct Incus/host shell.** Rejected: bypasses upstream ownership, authorization and safety boundaries.

## Historical proposed responsibility split (superseded; DO NOT IMPLEMENT as written)

- Outbound provider lifecycle, Tenant/Guard, routes, metrics and backup/snapshot operations use a typed Containarium REST adapter verified against the installed Protobuf/grpc-gateway contract.
- The Agent continues using BoxWeave's own authenticated MCP (SSH stdio / Streamable HTTP OAuth), which becomes a facade over the common Operation Core.
- Guest execution, files and detached coding runs use an exact Box-scoped SSH Executor; optional `agent-box` MCP remains permitted where necessary and independently authorized.
- A minimal host CLI repairs/diagnoses the installation independently; it cannot be an unrestricted Agent escape hatch.
- An explicit, allowlisted upstream MCP compatibility lane is a **long-lived, complementary channel**, not merely a temporary deprecated bridge. **Owner-selected Option A** preserves upstream MCP writes while progressively mediating each mutating tool through the shared Operation Core before dispatch; no untracked mutation bypass may be claimed as protected. For tools not yet safely mediated, coverage must be stated honestly and protected-mode writes must fail closed rather than bypass the guard. No blanket retirement of MCP writes was selected.
- A single common operation/identity/claim/receipt/trace contract spans Web, Agent MCP and CLI. D3 continues to diagnose inbound MCP and adds Provider API spans; D4 remains exact-Box preauthorized and bounded.
- The hypothetical 99% parity goal needs a baseline feature ledger; security invariants are non-negotiable even if a percentage threshold is met.

## Historical candidate consequences and costs (not current migration authorization)

Advantages: fewer outbound protocol translations, direct typed status/error classification, simpler REST capability checking and improved coherent D1–D4 trace design. Upstream MCP tool renames no longer break the main application path. Management and guest execution failure planes become explicit.

Costs: substantial Agent gateway rewrite, SSH identity and host-key management, optional `agent-box` routing, OAuth/MCP feature-parity tests, credential installation changes, API version drift, and possible host API availability/secret-locality issues. REST status does not equal actual business completion; an API response alone is not a daemon-internal trace or a retry grant.

## Historical full-REST acceptance gates (not an active work plan)

1. A0 complete inventory of current user-reachable app and forwarded upstream MCP capabilities, with immutable version reference and feature-level parity statuses.
2. A1 read-only authenticated REST proof on the **actual intended Containarium runtime**, including auth/TLS, empty inventory, identity and error/timeout handling.
3. A validated safe SSH/Box operation plan for `connect(exec)`, files, detached runs, missing helpers and Box replacement.
4. A unified operation-claim model covering **all** Agent/Web mutators with crash/unknown semantics and no blind replay.
5. D1–D4 correlation/secret-non-disclosure tests, plus operator-approved key custody for backup restore.
6. Independent isolated-host install/upgrade/rollback plan, no automatic Provider upgrade, and an explicit decision on intentionally unsupported features.
7. Keep ADR-003 and ADR-010 as **coequal, complementary records**: do **not** supersede ADR-003. Record the dispatch owner for each overlapping operation and verify coexistence during a long migration period; treat any future retirement or compatibility reduction as a *new* explicit owner decision.

## Current disposition

**MCP-first + REST-assisted is the selected policy; the universal REST-first and 29-interface bulk migration plans are abandoned.** The seven existing app REST lifecycle methods are retained; specialist upstream MCP is preferred by default. This ADR is not authority to change source, host policy, release, API tokens, or deployed Provider configuration. R-04 protected writes and UNKNOWN reconciliation remain independent safety gates. Historical alternatives above remain reference material only.
