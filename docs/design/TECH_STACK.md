# Confirmed Go runtime and official SDKs

**Decision:** The owner explicitly selected **Go + the official MCP Go SDK + the official Incus Go SDK** for the v1 rewrite. This supersedes the earlier Python implementation idea. These are imported Go modules, not additional runtime services or layers of MCP-to-MCP proxying.

## Mandatory dependencies

| Responsibility | Module / interface | Role |
|---|---|---|
| Public Model Context Protocol | `github.com/modelcontextprotocol/go-sdk` | Registers typed BoxWeave tools, handles MCP JSON-RPC and Streamable HTTP transport; default listener is loopback HTTP, optionally externally proxied by the operator |
| Native Incus management | `github.com/lxc/incus/v7/client` and matching `github.com/lxc/incus/v7/shared/api` | Instance lifecycle, Exec/WebSocket, native Operation, native TCP/UDP forwarding, minimal peer isolation and guest file APIs |
| Durable state | SQLite | Ownership records, Tracking ID/async operation journal, bounded output references, resource locks, owned native port forwards and simple private peer grants |
| Host-local lifecycle | Go binary + Linux service management | Version-manifest-driven install/status/upgrade, dependency reconciliation, diagnosis, administrator credential bootstrap/reset and safe recovery |

The `v7` component above identifies the currently reviewed Incus Go module path, **not** a mandatory hardcoded maximum daemon version. Select and pin an exact compatible Go module version during implementation; check required `api_extensions` and actual behavior on the target daemon.

## Explicit architecture constraints

- **External control:** authenticated Streamable HTTP MCP only; it binds `127.0.0.1` HTTP by default and can be exposed via optional operator-managed HTTPS proxy/Cloudflare Tunnel. There is no Caddy dependency or managed website route; native Incus TCP/UDP forwards can expose owned guest services to operator-managed proxies. An Incus SDK request over a local Unix socket is an *internal* API call, not a BoxWeave public REST endpoint.
- **No Containarium:** do not retain an upstream controller, external MCP proxy, SSH hop or wrapper CLI for lifecycle and Exec.
- **SDKs are libraries:** BoxWeave links both SDKs into its Go program. Domain services handle policy, idempotency, ownership and post-state verification, not the MCP SDK.
- **P0 Tracking and development:** Go `context.Context` and server-generated Tracking ID propagate through request/domain/journal/worker/SDK adapters/diagnostics. Native Incus guest file APIs and async Exec power the complete Agent editing/build/test/retest loop without a guest-installed Agent.
- **No raw client pass-through:** exposed MCP schemas are narrow BoxWeave business operations. Do not publish arbitrary Incus API operations, reverse proxy settings, host path editing or host command execution.
- **Permission boundary:** the Go MCP process is not trusted with unrestricted native root-equivalent host access merely because it imports an SDK. The privileged helper/restricted Incus credentials must enforce scope independently.
- **Capability-first:** check actual API Extensions, network isolation, Incus Exec stream/exit behavior, storage support and recursive nesting; unsupported dependencies result in accurate structured errors rather than invented success.
- **No extra management surface:** the host-local CLI exists only for host bootstrap and maintenance, and is not reachable as a public RPC endpoint.

## Upstream source baseline (reviewed; not yet implemented)

The 2026-11-07 [SDK fit review](../research/OFFICIAL_SDK_FIT_2026-11-07.md) inspected released **Incus v7.6.0** (`github.com/lxc/incus/v7`) and **MCP Go SDK v1.8.0**. These are an engineering review baseline, **not** compiled v1 dependency pins. Use tagged modules and build against their supported Go toolchain before freezing release manifests.

Incus `Operation.WaitContext` may request native cancellation when its context expires; its ordinary instance methods are name-addressed and many lack context parameters; native guest file writes do not provide hash-based atomic compare-and-swap. The local Incus admin socket is not a restricted Project identity. MCP 2026-07-28 requires SDK Stateless Streamable HTTP, while BoxWeave durable operations remain in SQLite independently of MCP sessions.

## Tagged Go requirements and reusable upstream primitives

Incus **v7.6.0** declares `go 1.26.9` in its released `go.mod`; MCP Go SDK **v1.8.0** declares `go 1.25.0`. When using the reviewed Incus release, build with a compatible Go 1.26.9+ toolchain; users installing signed prebuilt BoxWeave binaries should not need a Go compiler.

Use MCP `auth.RequireBearerToken`, typed `mcp.AddTool`, built-in localhost Host/DNS-rebinding protection and default 4 MiB body limit rather than reimplementing them. Use Incus native confined `incus`-group service-user credentials rather than assuming a privileged helper. Match native HTTP errors with `api.StatusErrorMatch`. The [20/20 mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) documents exact SDK methods and mismatched name traps.

## Verification gates

1. Pin reviewed compatible SDK module revisions and Go toolchain versions with reproducible dependency checks.
2. Build and exercise a real authenticated MCP handshake and one typed read tool.
3. Call a real Incus daemon using the official Go SDK and record the native version/extensions and tested operations.
4. Demonstrate agentless Linux container guest-root Exec stdout/stderr/exit, canceled/unknown native Operation, typed peer policy/forward update and ownership denial.
5. Repeat the integration at nested L1/L2 without relying on an outer privileged socket.

Record exact Go toolchain, official MCP/Incus SDK revisions and observed Incus daemon versions in the source-bound release/dependency manifest; explicitly invoked local install/upgrade automatically reconciles managed dependency versions. See [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md).

Review upstream [Incus Go client](https://github.com/lxc/incus/tree/main/client) and [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). The exact SDK versions and supporting Linux/Incus baseline are engineering choices to validate, not existing v1 implementation facts.
