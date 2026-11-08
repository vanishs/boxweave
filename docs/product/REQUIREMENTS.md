# Requirements — v1

Status: **accepted design goals**, not implemented or tested facts.

| ID | Required outcome |
|---|---|
| R-001 | The only public management endpoint is an authenticated Streamable HTTP MCP service; no management REST/Web or host shell is exposed. |
| R-002 | A host-local CLI supports installation, status, upgrade, reset of administrator credentials and safe recovery. |
| R-003 | Box lifecycle tools use native Incus to create/list/read/update/start/stop/delete and run guest commands. |
| R-004 | Each Box is bound to an Incus native instance UUID, independently checked on every identity-sensitive mutation; reuse of a name/IP never transfers ownership. |
| R-005 | MCP may inspect or mutate only this layer's verified BoxWeave-owned resources and consequences, never arbitrary host, Incus, Caddy or third-party resources. |
| R-006 | Default private Box-to-Box direct traffic is denied. Explicit scoped grants support peer IP and protocol/port restrictions while using stable Box identities as policy owners. |
| R-007 | Internet egress is enabled by default, with a per-Box opt-out. Other Boxes may access public Caddy routes of Boxes. |
| R-008 | Managed HTTP routes and TCP/UDP forwards resolve destination from a verified Box UUID and guest port, never an arbitrary MCP-supplied target address. |
| R-009 | Accepted mutations persist their intent, caller scope, target identity and idempotency key before dispatch, return a BoxWeave operation ID and expose its state through MCP. |
| R-010 | Worker crashes, transport interruption and deadline expiration must not automatically retry an uncertain mutation; preserve UNKNOWN and reconcile native evidence. |
| R-011 | Successful mutation claims require independent post-state verification, with explicit partial/unknown state when verification is impossible. |
| R-012 | Each nested installation exposes the same feature contract as a physical-host installation if upstream capabilities permit; each controller manages only its own directly owned Boxes, without recursive orchestration or child-controller administration. |
| R-013 | A child layer cannot administer parent/sibling resources or inherit parent credentials, even if the child was given permission to enable nesting. |
| R-014 | Incus access and Caddy changes are mediated by least-privilege local components; remote MCP callers receive no raw high-privilege endpoints. |
| R-015 | State, diagnostics, logs, errors and exposed schemas have typed, bounded, English contracts; testing proves both valid and denied paths. |
| R-016 | No mandatory Incus storage backend is imposed, and optional storage features are capability-gated. |
| R-017 | Version scheme is 1.X.Y, where X is feature increments and Y patches. Release identity is immutable; a source release is not installed acceptance. |
| R-018 | Project artifacts are English; Agents converse in the requester's language, preserving technical identifiers. |
| R-019 | The runtime is Go, using the official MCP Go SDK for public Streamable HTTP MCP and the official Incus Go client for native management; no extra intermediary controller or public REST management service is introduced. |
| R-020 | The same install/upgrade/capability-check process works at each supported nesting layer without seizing parent resources; parent capabilities are configured through ordinary authorized operations on the containing Box. |
| R-021 | The Box hosting an inner BoxWeave is an ordinary parent-owned Box: it has the same start/stop/update/delete semantics, with no special discovery, subtree protection or recursive cleanup. |
| R-022 | Nested compatibility supports BoxWeave-in-BoxWeave development and acceptance, and allows future environment subleasing without implementing subleasing management, billing, user tenancy or cross-layer orchestration in v1. |
| R-023 | Linux container lifecycle and Exec require no guest-installed BoxWeave client, resident Agent or SSH daemon; default Exec UID is guest root only, never parent/host root. |
| R-024 | The HTTP MCP listener binds only to `127.0.0.1` by default; the operator may separately proxy/Tunnel to public HTTPS, without enabling anonymous or raw native access. |
| R-025 | Ubuntu LTS is the first tested host environment and default guest image; guest resources have configurable defaults and nesting is opt-in. |
| R-026 | When supported and authorized, the installer reuses an existing Incus daemon and provisions only owned Project resources and a dedicated Caddy; explicitly invoked install/upgrade reconciles approved dependency versions without unauthorized foreign-service takeover. |
| R-027 | Host upgrades are explicit, while new-version source releases from successful `v1` CI are automatic and immutable; no docs-only pseudo-release, duplicate tag, automatic deployment or stable selector change. |
| R-028 | Local bootstrap issues multiple independently revocable scoped MCP credentials; managed resource ACLs may accept verified owned Box IPs or Box IDs, with identity-safe native state updates. |
| R-029 | Missing/corrupt ownership state fails closed, with only explicit host-local verified recovery. |
| R-030 | Every authenticated MCP action, including reads and validation/denial paths where protocol permits, emits a server-assigned Tracking ID; stable across deduplicated logical retries, distinct from request, idempotency, BoxWeave operation and native Incus IDs. |
| R-031 | Go `context.Context` propagates deadlines, actor/scope and Tracking ID through transport, domain, durable journal, workers, Incus/Caddy/SQLite adapters, diagnostics and local maintenance where applicable; workers reconstruct only a safe persisted projection. |
| R-032 | All accepted mutations including guest Exec and guest file writes run as durable asynchronous operations with scoped idempotency; client timeout/disconnect never implies native cancellation or permits blind replay of UNKNOWN. |
| R-033 | Box creation starts by default (`start=false` opt-out), and distinguishes native RUNNING from verified guest Exec readiness. |
| R-034 | P0 supplies agentless native guest file list/read and a unified `box_file_apply` for create/replace/patch/delete, bounded and version-checked within exact owned Box identities, with no host path or guest management Agent. |
| R-035 | P0 supports asynchronous builds/tests via guest-root Incus Exec, bounded paginated output and observed exit/command success retrieval after client reconnect; missing evidence is UNKNOWN/not captured, not invented output. |
| R-036 | P0 provides Tracking-ID-linked, ownership-scoped, redacted read-only diagnostics, including known failure layer, observed status, omissions and evidence level; no generic host inspection API. |
| R-037 | MCP credentials can be restricted by operation class and exact owned Box UUID, with independently revocable rights; auth scope is distinct from resource ownership. |
| R-038 | Disk shrink is refused before dispatch in P0 regardless of storage driver; disk growth is capability checked and verified after mutation. |
| R-039 | Explicit local install/upgrade records source-bound required/observed Incus, Caddy, Go SDK/toolchain, OS and state-schema versions, automatically installs missing or upgrades outdated/incompatible approved **managed** dependencies, and verifies postflight. |
| R-040 | Existing shared Incus or unrelated host services must not be restarted/upgraded by dependency reconciliation without explicit, specific, host-local operator approval and impact preflight; no broad or background updater. |
| R-041 | P0 must pass the real clean-Box Agent development loop: create -> readiness -> read/list -> apply create/replace/patch -> async test/build -> bounded status wait/reconnect and recover result -> optional supported cancel -> diagnose -> fix/retest -> independent validation. |
| R-042 | P1, not P0, may add persistent guest process management, guest search convenience, idle auto-sleep and HTTP wake, OAuth 2.1 and snapshots. |
| R-043 | P0 has one `box_file_apply` guest mutation with `create/replace/patch/delete` modes, expected SHA/revision for existing files, a verified post-state, no recursive delete and explicit file deletion permission; `box_file_read` returns size/offset/EOF and trustworthy hash coverage. |
| R-044 | P0 `operation_get` supports a read-only bounded `wait_ms` of 0–20000; timeout/disconnect stops waiting but never cancels the original operation. |
| R-045 | P0 `operation_cancel` has a separately journaled and authorized cancellation receipt; proven no-dispatch permits queued cancel, and running guest Exec may be terminated only with native evidence. No cancel request is proof of rollback or permission to free UNKNOWN claims. |
| R-046 | P0 `box_exec` accepts typed `argv`, safe guest `cwd`, bounded environment, timeout and output limits without implicit shell expansion; guest root by default and no host command execution. |
| R-047 | P0 `box_update` only accepts explicitly allowlisted CPU, memory, disk growth, nesting and internet toggles, never arbitrary Incus/NIC/host-device configuration. |
| R-048 | An unrestricted guest-root `box.exec` Token can modify guest files regardless of denied `box.file.write`/`box.file.delete` scopes; the product must not claim otherwise. Native parent-host confinement and per-Box ownership still apply. |

See [acceptance cases](../verification/ACCEPTANCE.md), [P0 development loop](DEVELOPMENT_LOOP.md), [Tracking/Context](../design/TRACKING_CONTEXT.md), [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md), and detailed [security](../design/SECURITY.md), [operations](../design/OPERATIONS.md), and [nesting](../design/NESTING.md).
