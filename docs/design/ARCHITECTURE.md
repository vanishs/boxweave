# Architecture — v1

**Status:** target design only. No v1 runtime has been implemented.

## Layers and responsibility

```text
External MCP client
   |
   | authenticated HTTP Streamable MCP on loopback; optional operator HTTPS proxy
   v
BoxWeave MCP adapter (schema, auth, ownership, Tracking ID, Go context)
   |
   v
Domain services / policy engine  ----> local durable operation journal
      |                                  (Tracking ID, operation context)
   |                                        |
   |                                  worker + reconciler
   |                                        |
   +------------------+---------------------+
                      |
              narrow local adapters
               /      |       \
          Incus SDK  Caddy     SQLite
          / guest    Admin API / owned records
          / bridge
   Incus daemon
      |
  owned Box(es)
      |
  nested BoxWeave (same executable and contracts, independent state)
```

Each layer is independent. A Box is an Incus Linux container managed by the current BoxWeave instance. The physical host is L0; an appropriately configured Box can run its own Incus, Caddy, operation store, MCP listener and BoxWeave as L1, and so on.

Do not build a second public management protocol, Web control plane, remotely callable local CLI, generic SSH hop, separate nested-only binary, or Containarium adapter.

## Implementation boundaries

1. **Public transport adapter:** official Go MCP SDK bound to `127.0.0.1` HTTP by default, with separately operator-configured external HTTPS transport, request/response schemas, authentication and filtering. It must not itself perform unchecked host or Incus mutations.
2. **Domain/policy layer:** Box identities, ownership, planned state, ACL rules, published routes and ports, allowed configuration updates and operation contracts.
3. **Operation store and reconciliation:** local SQLite WAL plus locking and transactional updates. Every admitted mutation, including guest Exec/file writes, is asynchronous and has a stable Tracking ID and distinct durable operation ID. Durable intents precede external effects. Workers reconstruct safe Go context metadata and retain original upstream identities where available.
4. **Incus adapter:** use the official Go Incus client and typed Incus API models. A narrow SDK adapter enables compatibility checks, native error mapping, resource ownership validation and independent native reads.
5. **Caddy adapter:** dedicated BoxWeave-owned Caddy process/configuration where feasible; only BoxWeave-managed **HTTP-only P0 application route** namespace, local-only Admin API. Explicitly prevent automatic website HTTPS, ACME, 443 or redirect. Reconcile desired SQLite records into Caddy live configuration.
6. **Privileged boundary:** service privilege separation or a constrained local helper owns necessary Incus/host setup privileges. Public MCP never accesses unrestricted Incus socket, Caddy Admin API or host shell.
7. **Host-local CLI:** bootstrap password, operation-class/exact-Box scoped token issuance/revocation, diagnostics, dependency-aware explicit install/upgrade/recovery, version/dependency provenance, credential reset; never a public management endpoint. Each tracked local operation carries a safe correlation ID.
8. **P0 Agent development path:** no guest Client or SSH. Incus performs typed-argv async Exec and guest file list/read plus `box_file_apply(create|replace|patch|delete)`, with independently observed state. Bounded `operation_get(wait_ms)` reads and separately tracked `operation_cancel` requests never falsely imply rollback. Tracking IDs, original output and diagnostics remain queryable after reconnect.

## Confirmed technology selection

- **Language:** Go. The owner's Go decision supersedes the earlier Python proposal.
- **Public MCP protocol:** official `github.com/modelcontextprotocol/go-sdk`, using Streamable HTTP; the SDK is an embedded library, not another running MCP proxy.
- **Native container integration:** official Incus Go client `github.com/lxc/incus/v7/client` and the matching typed API package. Use Incus's local Unix socket where privileges permit; do not expose Incus's native API publicly.
- **State and integrations:** local SQLite journal, Caddy Admin API, Linux/systemd; the exact implementation libraries and version compatibility are verified in the first development slice.
- **No secondary controller:** no Containarium layer, arbitrary Incus CLI subprocess wrapper for core operations, or separate public REST/gRPC management service.

See [technology stack](TECH_STACK.md), [detailed MCP tool contracts](MCP_TOOL_CONTRACTS.md), [Tracking/context](TRACKING_CONTEXT.md), [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md), [P0 development loop](../product/DEVELOPMENT_LOOP.md), [MCP binding](MCP_BINDING.md), [agentless guest management](GUEST_MANAGEMENT.md), and [approved defaults](../decisions/DEFAULTS.md).

## Security invariants

All privileged decisions derive from durable local ownership and observed native identity. A BoxWeave administrator has broad control **within this instance's ownership domain**, not arbitrary host authority. Guest Exec is confined to an owned guest; nested Incus cannot inherit or reach the parent's host socket. Unknown resources remain unmanaged.

See [security](SECURITY.md), [operations](OPERATIONS.md), [network](NETWORK.md), [nesting](NESTING.md), and [Incus compatibility](INCUS_INTEGRATION.md).
