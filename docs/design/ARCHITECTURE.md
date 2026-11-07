# Architecture — v1

**Status:** target design only. No v1 runtime has been implemented.

## Layers and responsibility

```text
External MCP client
   |
   | HTTPS, authenticated Streamable HTTP MCP
   v
BoxWeave MCP adapter (schema, auth, ownership, request validation)
   |
   v
Domain services / policy engine  ----> local durable operation journal
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

1. **Public transport adapter:** official MCP protocol implementation, request/response schemas, authentication and filtering. It must not itself perform unchecked host or Incus mutations.
2. **Domain/policy layer:** Box identities, ownership, planned state, ACL rules, published routes and ports, allowed configuration updates and operation contracts.
3. **Operation store and reconciliation:** local SQLite WAL plus locking and transactional updates. Durable intents precede any external effect. Persist original upstream operation identity where available.
4. **Incus adapter:** use the official Go Incus client and typed Incus API models. A narrow SDK adapter enables compatibility checks, native error mapping, resource ownership validation and independent native reads.
5. **Caddy adapter:** dedicated BoxWeave-owned Caddy process/configuration where feasible; only BoxWeave-managed route namespace, local-only Admin API. Reconcile desired SQLite records into Caddy live configuration.
6. **Privileged boundary:** service privilege separation or a constrained local helper owns necessary Incus/host setup privileges. Public MCP never accesses unrestricted Incus socket, Caddy Admin API or host shell.
7. **Host-local CLI:** bootstrap password, diagnostics, upgrades, recovery, version display, credential reset; never a public management endpoint.

## Confirmed technology selection

- **Language:** Go. The owner's Go decision supersedes the earlier Python proposal.
- **Public MCP protocol:** official `github.com/modelcontextprotocol/go-sdk`, using Streamable HTTP; the SDK is an embedded library, not another running MCP proxy.
- **Native container integration:** official Incus Go client `github.com/lxc/incus/v7/client` and the matching typed API package. Use Incus's local Unix socket where privileges permit; do not expose Incus's native API publicly.
- **State and integrations:** local SQLite journal, Caddy Admin API, Linux/systemd; the exact implementation libraries and version compatibility are verified in the first development slice.
- **No secondary controller:** no Containarium layer, arbitrary Incus CLI subprocess wrapper for core operations, or separate public REST/gRPC management service.

See [technology stack](TECH_STACK.md).

## Security invariants

All privileged decisions derive from durable local ownership and observed native identity. A BoxWeave administrator has broad control **within this instance's ownership domain**, not arbitrary host authority. Guest Exec is confined to an owned guest; nested Incus cannot inherit or reach the parent's host socket. Unknown resources remain unmanaged.

See [security](SECURITY.md), [operations](OPERATIONS.md), [network](NETWORK.md), [nesting](NESTING.md), and [Incus compatibility](INCUS_INTEGRATION.md).
