# MCP-first + REST-assisted control-plane — current policy and archived API-first research

> **CURRENT OWNER DISPOSITION (2026-11-05): MCP-first + REST-assisted hybrid SELECTED; universal API-first and 29-tool bulk REST migration ABANDONED.** The proposed **29-tool (8 P1 + 21 P2) MCP-to-REST implementation program** and universal API-first direction are cancelled; [PR #77](https://github.com/vanishs/boxweave/pull/77) closed; merge commit `020ebfe8` was reverted by `e19e60bb`. This archived research does **not** authorize new REST work. Preserve existing 7 app-owned REST lifecycle methods and long-term **REST+MCP coexistence**, with upstream MCP the default for remaining specialist operations. Any future individual REST use requires demonstrable benefit and separately accepted design/verification. Do not mistake this status for abandoning ADR-003, authenticated MCP, or R-04 shared coordination safety work.

> **Status: OWNER-SELECTED LONG-TERM COEXISTENCE DIRECTION / FEATURE-BY-FEATURE IMPLEMENTATION ACCEPTANCE PENDING.** The owner selected Option A and coequal ADR-003/ADR-010 on 2026-11-03; this is not acceptance of universal mediation, an automatic migration or a host/release change.
> Recorded: 2026-11-03. BoxWeave source baseline: `main` at `4f5ffe7fae586c79e6d75ed90560ee196e618ec3` (`VERSION=0.42.11`). Containarium source inspected: `main` at `e5deb167c973033d9a847abdcb2420e917095546`. The installer used Containarium `v0.120.1` **at this historical v0.42.11 analysis baseline**; v0.44.7 requires Containarium >= v0.128.0 and has no automatic maximum-version ceiling. Never assume upstream-main APIs are deployed.
>
> **Current transport preference (supersedes the 2026-11-03 API-first thought experiment):** **MCP-first end-to-end for Agent-facing tools and new/specialist Containarium operations; REST-assisted only for reviewed, scoped app-owned operations.** The seven already-implemented app direct REST lifecycle methods remain active with configured localhost API credentials. Containarium MCP/SSH remains supported, including specialist Tenant/Guard/network tools and guest execution. Do not infer that preferring MCP makes unsafe writes or unknown outcomes safe. No automatic MCP-to-REST migration, or REST-to-MCP rollback after a possibly dispatched write, is permitted.

> **2026-11-03 implementation checkpoint (later than the research baseline below):** BoxWeave v0.42.12 added application-owned, loopback-only authenticated REST for list/get/create/start/stop/delete/resize, and v0.43.0 became the default stable selection after isolated installed-host evidence. The Agent-facing upstream MCP relay, specialist Tenant/Guard/nesting paths and SSH execution were **not** thereby converted. The original operator host is not implied to be upgraded. See [REST slice evidence](../verification/PROVIDER_REST_A1_04212_2026-11-03.md) and [v0.43.0 stable promotion](../verification/STABLE_PROMOTION_DC01_0430_2026-11-03.md). The v0.42.11 commit IDs below remain a **historical analysis baseline**, not the latest implementation status.

> **2026-11-04 current verification checkpoint:** BoxWeave v0.44.7 source 4c1b88d has an exact published Release and independently installed isolated acceptance, including privileged MCP doctor 10/10, real app-owned create/SSH readiness, selected R-04 cross-entry conflicts and duplicate-task handling. The [installed R-04 follow-up](../verification/RELEASE_0447_POSTINSTALL_R04_2026-11-04.md) identifies retained UNKNOWN claims and missing fault-injection coverage. **Default stable selector is still 0.43.1; owner host remains unupgraded to 0.44.7.** This is not proof that all upstream MCP write tools are mediated. Upstream Containarium defects are record-only and are not fixed by architectural direction.

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

```text
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
```

This has worked for incremental delivery; the costs appear when the product needs one stable operation, error, ownership and trace model across *both* Agent and Web mutations. MCP tool-discovery schema and some text output are poor long-term substitutes for a typed machine-to-machine API. REST/gRPC are not inherently more secure or reliable; the benefit comes from **a smaller, explicit contract** and the opportunity to centralize policy, dispatch, receipts and tracing.

Do not confuse upstream's **CLI-first delivery policy** with an instruction for BoxWeave to shell out to `containarium` for every action. Upstream CLI, REST and MCP are client entry points to common underlying operations, except for host-local CLI paths. The upstream MCP server itself normally uses REST; its `connect(exec)`, `code_*` and `push/sync` have SSH or local transfer portions not representable as plain lifecycle REST calls.

## 2. Historical scope assumptions: radical redesign versus in-place migration (retired universal API-first proposal)

**Thought experiment:** all old data, CLI spellings and external interface names could be replaced, provided the user-visible functional outcome is retained to a target of **99%**. That percentage is a **design objective, not a measured claim**. It does not authorize ignoring the existing operator's active host, credentials, data, configured clients or stable release when any real deployment is attempted.

Define an acceptance denominator before committing to a rewrite:

- Count *existing user-reachable behaviors* from source and installed evidence, not aspirational roadmap rows or an unreviewed number of MCP tools.
- Distinguish **built-in BoxWeave features**, **forwarded upstream MCP tool availability**, **optional per-Box agent-box features**, **environment/host-specific capabilities** and **design-only E2–E4 behaviors**.
- Mark each feature `preserved`, `equivalent-with-observable-difference`, `intentional-retirement`, `not-yet-proven`, `unsupported-by-runtime`. Unknown is not preserved.
- Do not justify removal of a rare **security-critical invariant** (no duplicate create, no unauthorised Box execution, target-incarnation pinning, secret non-disclosure) by a numeric 99% threshold. Such invariants have 100% mandatory acceptance.
- An upstream MCP tool that BoxWeave transparently advertises must be accounted for *separately* from app-owned typed operations. Its **presence in a catalogue** is not proof of authorization, execution or user reliance; actual feature reachability must be measured.

The **in-place migration** route can preserve current Web/SQLite and change the app Provider first. The **new-core rewrite** route can consolidate more deeply but has far larger acceptance and host-cutover cost. These are separate planning options and must not be mixed in an estimate.

## 3. Historical proposed API-first target boundaries (not the current implementation mandate)

```text
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
```

### 3.1 BoxWeave owns

- One typed set of operations and schemas serving MCP, Web and local CLI, with protocol translation at *boundaries*, not duplicated business rules.
- Actor authorization and operator policy, exact managed Box identity (provider/backend/project/logical/immutable incarnation), per-resource conflict/lease/dispatch-intent, durable receipts, reconciliation, independent Agent-facing tool contracts and diagnostics.
- Bounded outbound API transport, TLS/loopback and credential-policy validation, explicit capabilities and error normalization.
- Guest execution transport and its SSH identity, stdout/stderr limits, file-root restrictions, trust pinning and task-result semantics.
- Private prompt graphs, public client setup, model handoff and its own release/host ownership boundaries.

### 3.2 Containarium still owns

- Authorization and the *actual* lifecycle, Tenant, network, route, backup, snapshot, storage and guest-source-of-truth operations.
- Its daemon, Incus integration and service-side validation. BoxWeave does not bypass a missing API with raw Incus or host-root shell.
- Daemon-side tracing **only when upstream implements and exposes it**; downstream `trace_id` is not proof that the daemon consumed a matching identifier.

### 3.3 Three internal blocks, multiple adapters

1. **Historical Provider API proposal (not universal policy):** typed JSON→domain translation, request authentication, timeout, response-shape validation, feature probing, bounded retries of *safe reads only*, upstream response IDs where available. REST maps to a Protobuf-defined contract through grpc-gateway; direct gRPC remains a possible later benchmark/ADR, not an implied requirement.
2. **Box Executor:** separate SSH execution and file/copy/sync adapters. Prefer service-owned key/certificate, inspect upstream-stamped `ssh_host` and verified incarnation, never accept caller-supplied arbitrary host, username, key file or remote root. Optional per-Box `agent-box` MCP may remain an implementation detail for its unique capabilities.
3. **Operation Core:** single intent/claims/dispatch/observed state machine; all exposed mutators (including any compatibility path) must pass it **before** a side effect. Use one task identity and trace correlation from both entry points.

The host repair CLI remains independently functional when MCP, API or Web is down, but host privilege must not become an Agent escape hatch.

### 3.4 Upstream implementation and roadmap signal (2026-11-03 review)

**Don't equate a common CLI command with a runtime hop through the CLI binary.** In Containarium, `containariumd` serves the platform REST/gRPC API; `containarium` CLI and the platform `mcp-server` are parallel clients of the platform API/shared Go operations. The platform MCP is **REST-dominant, not REST-only**:

| Operation family | Verified upstream path | BoxWeave boundary to prefer |
| --- | --- | --- |
| Box lifecycle, inventory, resource/route/network management | Platform MCP/CLI → daemon REST/gRPC API | Typed Provider REST adapter; validate runtime-specific routes, authorization and postconditions |
| `connect(exec)`, session commands | REST discovery/SSH authorization **then direct SSH to the Box** (Go SSH for MCP exec) | Separate exact-Box SSH Executor, with service-controlled identity, host-key and incarnation checks |
| `push` / `sync` | SSH and local Git/tar/manifest operations, not simple lifecycle REST | Explicit source/transfer adapter; local workspace and file permissions are separate trust boundaries |
| `code_run` / `code_attach` / `code_status` | SSH starts or communicates with the **per-Box `agent-box` MCP** | Optional capability-backed adapter only, not a mandatory preinstalled Box service |
| `sync_ssh_config`, SSH key/certificate preparation, backup restore key-file reads | API calls **plus client-local file/credential operations** | Distinguish app-host-local effects from daemon operations; enforce key custody and private logging |
| GitHub Actions runner provisioning | Daemon API **plus SSH installation and GitHub API** | Treat as an orchestration workflow with multiple independently observed stages, not one REST verb |

Code references: [Containarium platform MCP](https://github.com/FootprintAI/Containarium/tree/main/internal/mcp), [upstream connect](https://github.com/FootprintAI/Containarium/blob/main/internal/mcp/connect.go), [upstream code tools](https://github.com/FootprintAI/Containarium/blob/main/internal/mcp/code_tools.go), [upstream transfer](https://github.com/FootprintAI/Containarium/tree/main/internal/transfer), and [upstream README roadmap](https://github.com/FootprintAI/Containarium#roadmap). The README lists `agent-box` tier-2 work and a **planned** Q4 2026 OSS v1.0/stable-API milestone. These are intentions, **not** a shipped or guaranteed REST replacement for SSH/stdio; verify an immutable deployed release, not the `main` branch or roadmap alone.

**Current architectural rule (2026-11-05, superseding the historical diagram above):** prefer **upstream MCP** for new and specialist outbound Containarium capabilities; use direct REST only for the **seven already-shipped app lifecycle operations** or a separately justified and verified auxiliary feature. Preserve **inbound BoxWeave MCP** (SSH stdio and authenticated HTTP/OAuth), guest SSH, existing app receipts and exact target trust. The historical diagram is not a mandatory redesign. No protocol choice removes the need for authoritative postconditions, claims and UNKNOWN handling. No required in-Box agent and no direct Incus/root-shell bypass.

### 3.5 One operation boundary without forcing one transport

Agent MCP, Web and non-emergency CLI mutations should enter the **same BoxWeave operation/identity/claim/receipt state machine** *before* hitting REST, SSH, transfer or any transitional upstream MCP tool. Log bounded, separately classified `provider_api`, `box_executor`, `compat_mcp` and postcondition spans under an authorized operation reference. A BoxWeave `trace_id` establishes only **BoxWeave-owned** correlation; it does not demonstrate that Containarium consumed the same trace or that a guest command finished. See [D1–D4 contract](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md).

**Owner choice (2026-11-03): Option A — preserve upstream MCP write capabilities and progressively mediate each mutating tool through the shared Operation Core.** This is a **design decision**, not evidence that the present gateway already enforces it. Do not use a blanket prohibition of advanced writes as the migration strategy. Inventory the forwarded tool catalogue and, for each mutator, bind trusted actor, exact Box/resource identity, authority, operation claim, durable dispatch intent/receipt, UNKNOWN recovery and a bounded `compat_mcp` diagnostic span **before** dispatch. Classify multi-resource, host-level and unknown-schema tools explicitly; unreviewed mutations cannot be *advertised as protected*. Where protected mode requires full interception, an unmediated path must not bypass the guard: leave it visibly outside the protected guarantee until migrated, or fail that operation closed in protected mode with an explicit coverage reason; do not turn this limited safety invariant into Option B's general retirement of MCP writes. Ordinary existing behavior and published tool exposure are unchanged by this documentation-only decision. Acceptance requires same-Box conflict and different-Box concurrency tests across Web/REST, Agent MCP, and the compatibility lane.

**Owner choice (2026-11-05 update): keep ADR-003 and ADR-010 as long-lived, complementary decisions at the same design level, while preferring MCP for **new/specialist** upstream integrations and retaining the already shipped REST lifecycle subset.** [ADR-003](ADR/0003-transport-and-trust.md) retains upstream MCP/SSH client transport, identity and bootstrap trust; [ADR-010](ADR/0010-api-first-control-plane-candidate.md) describes typed REST for application-owned management and a separately guarded SSH Executor. Both can coexist for a long time. A given operation still needs **one declared owner/path** and a common claim boundary; "coequal" does not authorize duplicate side effects or parallel unmanaged dispatch. ADR-010's formal adoption/implementation evidence remains separately gated; neither record is silently superseded.

## 4. Current recommendation — MCP-first, REST-assisted; historical alternatives below

**Owner-selected direction (2026-11-05):** **upstream MCP is the default for new/specialist operations**; keep the existing seven app-owned lifecycle direct REST calls because their scope and credential/trace contracts are already implemented, and admit additional REST **only case by case** after measured functional or diagnostic value, runtime authentication/schema/identity checks and full write-safety acceptance. Agent-facing SSH/HTTP MCP and upstream MCP/SSH stay first-class. An independent SSH Executor is **optional**, not a required migration. The historical API-first alternatives below are reference comparisons, not scheduled development. Compare the current direction against:

- **MCP-only inside and out:** closest to the current default for specialist tools, but would discard already useful app REST lifecycle contracts and still require BoxWeave-specific conflict/diagnostic boundaries.
- **CLI subprocess for all operations:** local-vs-remote behavior ambiguity, output parsing, subprocess privilege and weaker structured error guarantees.
- **Direct Incus:** violates Containarium authorization, tenant encryption, network/Guard ownership and host safety.
- **Full direct gRPC at day one:** possible technical strength from generated clients, but adds daemon exposure, mTLS and deployment/config complexity beyond the current default REST gateway.
- **Full upstream tool reimplementation immediately:** makes the 99%-parity target and security reviews unbounded.

**Historical research caveat:** the abandoned universal API-first preference assumed low compatibility cost; it did not establish that REST changes less, is inherently safer, or reduces long-term maintenance. Rebuilding the Agent Gateway remains a high-risk project and is **not authorized by the current MCP-first decision**.

## 5. Non-negotiable operational invariants

- Authenticate every entry; keep OAuth/API token for BoxWeave Agent clients distinct from outbound Containarium JWT/API authority, SSH identities and Web session cookies. Do not borrow Agent model provider credentials.
- Bind every action to a freshly observed Box **incarnation**; names and IPs can be reused. Refuse drift and caller attempts to substitute host, user, path or key.
- Claims and one-use intent must be committed before dispatch; do not hold SQLite write locks during remote HTTP/SSH calls. Every mutation path uses the same conflict guard when protected mode is promised.
- Failed-before-dispatch, upstream-acknowledged, observed-complete, observed-failed and **unknown-after-dispatch** are different states. HTTP 200/202/204 is not business completion; timeout/connection drop after possible send is **UNKNOWN**, not permission to retry.
- Never replay uncertain creates, deletes, restores, network changes or guest commands automatically. Reconcile via independent read evidence, and state exactly what remains unverified.
- Default deny deep D4 read of any Box until an exact incarnation and check/service scope is explicitly allowed by server-owned policy. Preapproval is not approval for arbitrary shell, raw logs, secrets or new Boxes.
- Keep secret files server-side, hardened by OS ownership and path checks. API restore requests can contain decrypted `age_identity` on the **authenticated transport**: a host file path is not proof that key material never travels on the wire. Never persist or log it; prefer an upstream-approved host-side key *reference* if one is later provided.
- Keep public HTTPS ingress opt-in; prefer loopback for Provider API; keep platform MCP outward-facing auth and untrusted tool-output handling independent.
- Trace/log capture is secondary and best-effort: failure to log must never turn a failed operation into success, overwrite business receipts or cause duplicate dispatch.
- No hidden installation, provider upgrade, public port exposure, host wrapper repair or key rotation as a side effect of an ordinary API request.

## 6. Questions intentionally left open

| Decision | What must be proved before approval |
| --- | --- |
| REST vs generated gRPC client | Exact deployed API availability, auth/TLS model, endpoint/version contracts, real latency/failure evidence, operational complexity |
| API-only gateway vs retained upstream MCP compatibility lane | **API-only plan abandoned:** upstream MCP remains supported/default; reconsider individual REST adapters only upon a separately reviewed need |
| Forwarded MCP mutators vs shared coordination | **Option A selected:** per-tool common claim mediation, preserving writes; review inventory, unknown/host-wide tools, cross-entry real conflict tests; record interim unprotected coverage truthfully |
| Coequal ADR-003 and ADR-010 scope | **Complementary, long-lived records**; ADR-003 owns MCP/SSH trust; ADR-010 records MCP-first default and REST auxiliary app features. This is integration preference, not ADR supersession |
| Standalone SSH executor vs optional upstream agent-box reuse | Real execution identity/trust, file scopes, streaming, cancellation/reconnect semantics, clean Box requirement |
| Installed-host cutover | Existing host ownership, API binding, token issuance/rotation, no secret exposure, independent rollback, zero accidental provider upgrade |
| Operation core migration | Cross-Web/SSH/HTTP mutation interception and durable UNKNOWN/claim recovery under crash/timeout |
| API diagnostics depth | Which response fields and upstream request IDs exist **on the installed daemon**, not only in upstream `main` source |

None of these questions should block independent read-only proofs of the typed REST adapter.

## 7. Delivery rule

A future Agent should first read the [migration plan](API_FIRST_PROVIDER_MIGRATION_PLAN.md), [diagnostic contract](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md) and actual [feature matrix](../verification/FEATURE_MATRIX.md), then record **S** (source), **T** (fixture/CI), **L** (real installed host), or **U** (unknown/unsupported) separately. Produce explicit branch/PR and review evidence per slice; do not silently change the pinned Provider version or published release.

**This document records the MCP-first + REST-assisted policy but creates no implementation ticket, blanket REST migration authorization, release, host change or upstream compatibility guarantee.**
