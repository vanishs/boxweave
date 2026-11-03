# API-first control-plane evolution — long-term architecture master

> **Status: DESIGN CANDIDATE / DISCUSSION RECORD, not an implementation approval, migration decision, installed result, release gate, or compatibility promise.**
> Recorded: 2026-11-03. BoxWeave source baseline: \`main\` at \`4f5ffe7fae586c79e6d75ed90560ee196e618ec3\` (\`VERSION=0.42.11\`). Containarium source inspected: \`main\` at \`e5deb167c973033d9a847abdcb2420e917095546\`. The BoxWeave-managed installer is pinned separately to Containarium \`v0.120.1\`; do not assume current upstream-main API is installed.
>
> **Core direction under study:** outward-facing **MCP-first**, internal **Containarium REST API-first**, Box execution **SSH-first**, local maintenance **CLI-only**. This does *not* mean eliminating MCP from BoxWeave: BoxWeave owns its Agent-facing MCP, and an upstream MCP compatibility lane remains until its actual functional coverage is mapped and accepted.

## Purpose and navigation

This is the parent of three future-looking design records:

1. [Migration and feature-parity plan](API_FIRST_PROVIDER_MIGRATION_PLAN.md) — code inventory, REST coverage, difficulty, phases, parity measurement and test gates.
2. [Unified operation trace and diagnostics](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md) — D1–D4 evolution, request/operation correlation, uncertainty, privacy and live acceptance.
3. [Candidate ADR-010](ADR/0010-api-first-control-plane-candidate.md) — decision alternatives, reasoning and explicit open gates.

Related binding current records: [current architecture](ARCHITECTURE.md), [ADR-003](ADR/0003-transport-and-trust.md), [ADR-009](ADR/0009-remote-mcp-transports-and-auth.md), [roadmap master](FEATURE_ROADMAP_MASTER.md), [D3/D4 approved contract](DIAGNOSTICS_D3_D4_MASTER.md), [R-01 Box operations](BOX_OPERATION_INTERFACE_V1.md), [R-02 durable tasks](MCP_TASK_RELIABILITY_R1.md), [R-04 lease/conflicts](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md), [upstream #3500 follow-ups](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md), [actual acceptance ledger](../verification/FEATURE_MATRIX.md).

**Precedence and guardrail:** this candidate does not supersede existing accepted ADRs, current product scope, identity/security contracts or the published installer. No production code or host operation should be changed merely because this document exists. Revisit formally by a future owner decision after the listed experiments.

## 1. Why revisit the architecture?

BoxWeave is not only an MCP forwarding proxy. It provides a durable environment-management application: project-to-Box bindings, lifecycle, Tenant/Guard/network changes, TCP/UDP publication, monitoring and diagnostics, Agent coding operations, job receipts, multi-Agent conflict control, Web UI, public client enrollment, prompt graph, model guidance and installer/repair.

Current call paths:

\`\`\`text
Agent MCP (SSH stdio or authenticated Streamable HTTP)
  -> BoxWeave mcp_gate.py (catalogue relay, auth/trust, tools and rewrites)
  -> Containarium platform MCP stdio process
  -> upstream REST/gRPC gateway / SSH helpers
  -> Containarium daemon / Incus / target Box

Browser / host CLI
  -> BoxWeave service + SQLite worker/receipts
  -> bwmini/provider.py, access_provider.py, agent_box.py
  -> separately configured upstream platform MCP or per-Box agent-box MCP
  -> daemon / SSH / Box
\`\`\`

This has worked for incremental delivery; the costs appear when the product needs one stable operation, error, ownership and trace model across *both* Agent and Web mutations. MCP tool-discovery schema and some text output are poor long-term substitutes for a typed machine-to-machine API. REST/gRPC are not inherently more secure or reliable; the benefit comes from **a smaller, explicit contract** and the opportunity to centralize policy, dispatch, receipts and tracing.

Do not confuse upstream's **CLI-first delivery policy** with an instruction for BoxWeave to shell out to \`containarium\` for every action. Upstream CLI, REST and MCP are client entry points to common underlying operations, except for host-local CLI paths. The upstream MCP server itself normally uses REST; its \`connect(exec)\`, \`code_*\` and \`push/sync\` have SSH or local transfer portions not representable as plain lifecycle REST calls.

## 2. Scope assumptions: radical redesign versus in-place migration

**Thought experiment:** all old data, CLI spellings and external interface names could be replaced, provided the user-visible functional outcome is retained to a target of **99%**. That percentage is a **design objective, not a measured claim**. It does not authorize ignoring the existing operator's active host, credentials, data, configured clients or stable release when any real deployment is attempted.

Define an acceptance denominator before committing to a rewrite:

- Count *existing user-reachable behaviors* from source and installed evidence, not aspirational roadmap rows or an unreviewed number of MCP tools.
- Distinguish **built-in BoxWeave features**, **forwarded upstream MCP tool availability**, **optional per-Box agent-box features**, **environment/host-specific capabilities** and **design-only E2–E4 behaviors**.
- Mark each feature \`preserved\`, \`equivalent-with-observable-difference\`, \`intentional-retirement\`, \`not-yet-proven\`, \`unsupported-by-runtime\`. Unknown is not preserved.
- Do not justify removal of a rare **security-critical invariant** (no duplicate create, no unauthorised Box execution, target-incarnation pinning, secret non-disclosure) by a numeric 99% threshold. Such invariants have 100% mandatory acceptance.
- An upstream MCP tool that BoxWeave transparently advertises must be accounted for *separately* from app-owned typed operations. Its **presence in a catalogue** is not proof of authorization, execution or user reliance; actual feature reachability must be measured.

The **in-place migration** route can preserve current Web/SQLite and change the app Provider first. The **new-core rewrite** route can consolidate more deeply but has far larger acceptance and host-cutover cost. These are separate planning options and must not be mixed in an estimate.

## 3. Proposed target boundaries

\`\`\`text
                 Codex / Claude / other Agent clients          Browser         Operator
                 SSH-stdio MCP | HTTP(S)-OAuth MCP             Web/API         CLI
                                   \            |                  |             /
                                    \-----------+------------------+------------/
                                                |
                             BoxWeave Transport / Auth facades
                                                |
                              BoxWeave Operation Core (one)
                     intent / policy / Box incarnation / claims / receipts
                       state observation / UNKNOWN / diagnostic trace
                                                |
                       +------------------------+-----------------------+
                       |                        |                       |
              Containarium API adapter    Box Executor             Native app
                REST + proto contract     SSH/optional             graphs/models/
              lifecycle, network, etc.    agent-box MCP            profiles/setup
                       |                        |
                daemon REST gateway       exact managed Box
                       |
                Containarium daemon
                 / Incus / storage

            Host operator CLI: separate, narrow privileged recovery path
            Upstream MCP compatibility lane: explicit/allowlisted and audited
\`\`\`

### 3.1 BoxWeave owns

- One typed set of operations and schemas serving MCP, Web and local CLI, with protocol translation at *boundaries*, not duplicated business rules.
- Actor authorization and operator policy, exact managed Box identity (provider/backend/project/logical/immutable incarnation), per-resource conflict/lease/dispatch-intent, durable receipts, reconciliation, independent Agent-facing tool contracts and diagnostics.
- Bounded outbound API transport, TLS/loopback and credential-policy validation, explicit capabilities and error normalization.
- Guest execution transport and its SSH identity, stdout/stderr limits, file-root restrictions, trust pinning and task-result semantics.
- Private prompt graphs, public client setup, model handoff and its own release/host ownership boundaries.

### 3.2 Containarium still owns

- Authorization and the *actual* lifecycle, Tenant, network, route, backup, snapshot, storage and guest-source-of-truth operations.
- Its daemon, Incus integration and service-side validation. BoxWeave does not bypass a missing API with raw Incus or host-root shell.
- Daemon-side tracing **only when upstream implements and exposes it**; downstream \`trace_id\` is not proof that the daemon consumed a matching identifier.

### 3.3 Three internal blocks, multiple adapters

1. **Provider API:** typed JSON→domain translation, request authentication, timeout, response-shape validation, feature probing, bounded retries of *safe reads only*, upstream response IDs where available. REST maps to a Protobuf-defined contract through grpc-gateway; direct gRPC remains a possible later benchmark/ADR, not an implied requirement.
2. **Box Executor:** separate SSH execution and file/copy/sync adapters. Prefer service-owned key/certificate, inspect upstream-stamped \`ssh_host\` and verified incarnation, never accept caller-supplied arbitrary host, username, key file or remote root. Optional per-Box \`agent-box\` MCP may remain an implementation detail for its unique capabilities.
3. **Operation Core:** single intent/claims/dispatch/observed state machine; all exposed mutators (including any compatibility path) must pass it **before** a side effect. Use one task identity and trace correlation from both entry points.

The host repair CLI remains independently functional when MCP, API or Web is down, but host privilege must not become an Agent escape hatch.

## 4. Working recommendation and its limits

**Candidate recommendation:** choose the API as BoxWeave's *long-lived internal management contract*; keep BoxWeave MCP as the Agent's stable first-class entry point; retain a separate SSH execution plane. Prefer this to:

- **MCP-only inside and out:** fastest inheritance of upstream tools, but more catalogue/text coupling and different app/gateway mutation boundaries.
- **CLI subprocess for all operations:** local-vs-remote behavior ambiguity, output parsing, subprocess privilege and weaker structured error guarantees.
- **Direct Incus:** violates Containarium authorization, tenant encryption, network/Guard ownership and host safety.
- **Full direct gRPC at day one:** possible technical strength from generated clients, but adds daemon exposure, mTLS and deployment/config complexity beyond the current default REST gateway.
- **Full upstream tool reimplementation immediately:** makes the 99%-parity target and security reviews unbounded.

This is an **architectural preference under a no-compatibility-cost assumption**, not proof that REST performs better or requires fewer total engineering hours today. In particular, rebuilding the entire Agent gateway is significantly harder than changing the app-side Provider.

## 5. Non-negotiable operational invariants

- Authenticate every entry; keep OAuth/API token for BoxWeave Agent clients distinct from outbound Containarium JWT/API authority, SSH identities and Web session cookies. Do not borrow Agent model provider credentials.
- Bind every action to a freshly observed Box **incarnation**; names and IPs can be reused. Refuse drift and caller attempts to substitute host, user, path or key.
- Claims and one-use intent must be committed before dispatch; do not hold SQLite write locks during remote HTTP/SSH calls. Every mutation path uses the same conflict guard when protected mode is promised.
- Failed-before-dispatch, upstream-acknowledged, observed-complete, observed-failed and **unknown-after-dispatch** are different states. HTTP 200/202/204 is not business completion; timeout/connection drop after possible send is **UNKNOWN**, not permission to retry.
- Never replay uncertain creates, deletes, restores, network changes or guest commands automatically. Reconcile via independent read evidence, and state exactly what remains unverified.
- Default deny deep D4 read of any Box until an exact incarnation and check/service scope is explicitly allowed by server-owned policy. Preapproval is not approval for arbitrary shell, raw logs, secrets or new Boxes.
- Keep secret files server-side, hardened by OS ownership and path checks. API restore requests can contain decrypted \`age_identity\` on the **authenticated transport**: a host file path is not proof that key material never travels on the wire. Never persist or log it; prefer an upstream-approved host-side key *reference* if one is later provided.
- Keep public HTTPS ingress opt-in; prefer loopback for Provider API; keep platform MCP outward-facing auth and untrusted tool-output handling independent.
- Trace/log capture is secondary and best-effort: failure to log must never turn a failed operation into success, overwrite business receipts or cause duplicate dispatch.
- No hidden installation, provider upgrade, public port exposure, host wrapper repair or key rotation as a side effect of an ordinary API request.

## 6. Questions intentionally left open

| Decision | What must be proved before approval |
| --- | --- |
| REST vs generated gRPC client | Exact deployed API availability, auth/TLS model, endpoint/version contracts, real latency/failure evidence, operational complexity |
| API-only gateway vs retained upstream MCP compatibility lane | Actual per-tool reachable feature matrix and owner-reviewed treatment of missing MCP-only actions |
| Standalone SSH executor vs optional upstream agent-box reuse | Real execution identity/trust, file scopes, streaming, cancellation/reconnect semantics, clean Box requirement |
| Installed-host cutover | Existing host ownership, API binding, token issuance/rotation, no secret exposure, independent rollback, zero accidental provider upgrade |
| Operation core migration | Cross-Web/SSH/HTTP mutation interception and durable UNKNOWN/claim recovery under crash/timeout |
| API diagnostics depth | Which response fields and upstream request IDs exist **on the installed daemon**, not only in upstream \`main\` source |

None of these questions should block independent read-only proofs of the typed REST adapter.

## 7. Delivery rule

A future Agent should first read the [migration plan](API_FIRST_PROVIDER_MIGRATION_PLAN.md), [diagnostic contract](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md) and actual [feature matrix](../verification/FEATURE_MATRIX.md), then record **S** (source), **T** (fixture/CI), **L** (real installed host), or **U** (unknown/unsupported) separately. Produce explicit branch/PR and review evidence per slice; do not silently change the pinned Provider version or published release.

**This document creates no implementation ticket, migration authorization, or upstream compatibility commitment.**
