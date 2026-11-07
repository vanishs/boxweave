# v1 scope and exclusions

## Fixed requirements

- Implementation in Go using the official MCP Go SDK and official Incus Go client/module.
- Single host per BoxWeave instance, no distributed controller or cluster scheduling.
- No dependency on Containarium, no v0.x compatibility requirement, no automatic old state migration.
- Public management protocol: authenticated Streamable HTTP MCP only, bound to `127.0.0.1` over HTTP by default; public HTTPS is optional, external and operator-configured (for example through Cloudflare Tunnel). No public REST management endpoints and no Web console. The only interactive initial experience is local super-administrator credential bootstrap.
- Incus controls Boxes; Caddy manages BoxWeave-owned HTTP publishing; native Incus networking supports scoped TCP/UDP forwarding and private connectivity rules.
- Per-Box ownership and isolation; no user-facing Tenant concept.
- Linux containers first. Virtual machines, complex orchestration, and clusters are not part of the P0 promise.
- Guest commands use native Incus Exec with **guest root (UID 0) by default**, not a required guest SSH service or installed guest-side BoxWeave Agent. P0 also uses safe native Incus guest file APIs for list/read/write/optimistic patch. Host shell and host privilege remain forbidden.
- Every accepted mutation, including guest Exec and guest file writes, is represented by a durable local asynchronous operation and can be reconciled after failure. Each MCP action is correlated using a stable server-owned Tracking ID and typed Go context propagation.
- Fully functional installations at every supported nesting layer: each layer independently manages only its own directly owned Boxes. Nesting is runtime compatibility, not a recursive BoxWeave management feature.
- Keep the underlying Incus storage driver unconstrained; capability-dependent operations report unsupported accurately.
- Ubuntu LTS is the initial supported **host** installation baseline. Guest image defaults to a reviewed Ubuntu LTS image, with supported alternatives.
- Reuse an existing compatible Incus daemon in a BoxWeave-controlled restricted Project; run a dedicated BoxWeave Caddy for Box application routes. Explicit local install/upgrade **automatically installs or upgrades approved managed dependencies** against a recorded compatible version manifest. A shared Incus daemon or foreign service is never disrupted without separately scoped local approval.
- Nesting is opt-in per directly owned Box; resource defaults are configurable. An operator explicitly invokes installed upgrades; source release automation never upgrades a running host.

## P0

**Sustainable Agent development loop**: Box creation defaults to start and proves guest Exec readiness; Box-scoped guest file listing/reading/create-or-safe-write/optimistic patch; asynchronous guest-root Exec with paginated bounded stdout/stderr and observed exit retrieval; read-only Tracking ID diagnostics; authenticated per-operation-class/per-Box Token rights. Box lifecycle and safe updates (including **no disk shrink**), host health; managed HTTP routes and TCP/UDP port forwarding; Box-owned private ACL and optional internet denial; universal Tracking ID/Go context and durable async operations; strict host/resource boundaries; independent nesting; version-aware auto-reconciliation of approved dependencies during explicit local install/update/status/recovery; adversarial and real nested acceptance.

See [tool inventory](../design/MCP_TOOLS.md).

## P1 candidates (not part of P0)

Native snapshot list/create/restore/delete where storage supports it; optional guest file search convenience; durable background process start/stop/log supervision; Box idle auto-sleep and HTTP wake; OAuth 2.1 browser authorization. These features must obey all P0 trust and operation guarantees when implemented.

## Deferred / excluded

- Whole-Box backup/restore and automatic backup systems.
- Full isolation against public-route or port-forward hairpin paths: direct **private** Box-to-Box isolation is the agreed boundary; published public websites remain reachable to other Boxes.
- Public administrative REST or Web interface, OAuth multi-tenant control plane, agent SSH management, unrelated host file/service management, unmanaged Caddy routing, multi-host orchestration.
- Cross-layer BoxWeave discovery, delegation, controller management, automatic multi-level ingress propagation, special subtree deletion safeguards, subleasing accounts, metering and billing. Future subleasing is a motivation, not an implemented v1 feature.
- Imposed storage-driver selection, **unrelated or blanket system upgrades** (reviewed owned dependency reconciliation is P0), implicit parent-level network changes.
