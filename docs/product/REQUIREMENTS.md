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
| R-012 | Every nested layer is a full independent controller with the same schemas, tools, privileges, safety and installation features if upstream capabilities permit. |
| R-013 | A child layer cannot administer parent/sibling resources or inherit parent credentials, even if the child was given permission to enable nesting. |
| R-014 | Incus access and Caddy changes are mediated by least-privilege local components; remote MCP callers receive no raw high-privilege endpoints. |
| R-015 | State, diagnostics, logs, errors and exposed schemas have typed, bounded, English contracts; testing proves both valid and denied paths. |
| R-016 | No mandatory Incus storage backend is imposed, and optional storage features are capability-gated. |
| R-017 | Version scheme is 1.X.Y, where X is feature increments and Y patches. Release identity is immutable; a source release is not installed acceptance. |
| R-018 | Project artifacts are English; Agents converse in the requester's language, preserving technical identifiers. |
| R-019 | The runtime is Go, using the official MCP Go SDK for public Streamable HTTP MCP and the official Incus Go client for native management; no extra intermediary controller or public REST management service is introduced. |
| R-020 | The same install/upgrade/capability-check process works at each supported nesting layer without seizing parent resources; any necessary parent-side permission is explicit and separately authorized. |

See [acceptance cases](../verification/ACCEPTANCE.md) and detailed [security](../design/SECURITY.md), [operations](../design/OPERATIONS.md), and [nesting](../design/NESTING.md).
