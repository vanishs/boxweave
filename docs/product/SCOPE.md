# v1 scope and exclusions

## Fixed requirements

- Implementation in Go using the official MCP Go SDK and official Incus Go client/module.
- Single host per BoxWeave instance, no distributed controller or cluster scheduling.
- No dependency on Containarium, no v0.x compatibility requirement, no automatic old state migration.
- Public management protocol: authenticated Streamable HTTP MCP only, bound to `127.0.0.1` over HTTP by default; public HTTPS is optional, external and operator-configured (for example through Cloudflare Tunnel). No public REST management endpoints and no Web console. The only interactive initial experience is local super-administrator credential bootstrap.
- Incus controls Boxes; Caddy manages BoxWeave-owned HTTP publishing; native Incus networking supports scoped TCP/UDP forwarding and private connectivity rules.
- Per-Box ownership and isolation; no user-facing Tenant concept.
- Linux containers first. Virtual machines, complex orchestration, and clusters are not part of the P0 promise.
- Guest commands use native Incus Exec with **guest root (UID 0) by default**, not a required guest SSH service or installed guest-side BoxWeave Agent. Host shell and host privilege remain forbidden.
- Every accepted mutation is represented by a durable local operation and can be reconciled after failure.
- Fully functional installations at every supported nesting layer: each layer independently manages only its own directly owned Boxes. Nesting is runtime compatibility, not a recursive BoxWeave management feature.
- Keep the underlying Incus storage driver unconstrained; capability-dependent operations report unsupported accurately.
- Ubuntu LTS is the initial supported **host** installation baseline. Guest image defaults to a reviewed Ubuntu LTS image, with supported alternatives.
- Prefer an existing compatible Incus daemon in a BoxWeave-owned restricted Project; keep a dedicated BoxWeave Caddy instance for Box application routes. The installer may install missing dependencies but not silently upgrade unrelated or existing system services.
- Nesting is opt-in per directly owned Box; resource defaults are configurable. An operator explicitly invokes installed upgrades; source release automation never upgrades a running host.

## P0

Box lifecycle, typed updates, guest Exec, host health; managed HTTP routes and TCP/UDP port forwarding; Box-owned private ACL rules and optional internet denial; single-administrator authentication; durable async operations; strict host/resource boundaries; nesting; install/update/status/recovery CLI; adversarial and real nested acceptance.

See [tool inventory](../design/MCP_TOOLS.md).

## P1 candidates (not part of P0)

Box file read/write via Incus guest file facilities; Box snapshot list/create/restore/delete where the selected storage driver supports them. These features must obey all P0 trust and operation guarantees when implemented.

## Deferred / excluded

- Whole-Box backup/restore and automatic backup systems.
- Full isolation against public-route or port-forward hairpin paths: direct **private** Box-to-Box isolation is the agreed boundary; published public websites remain reachable to other Boxes.
- Public administrative REST or Web interface, OAuth multi-tenant control plane, agent SSH management, unrelated host file/service management, unmanaged Caddy routing, multi-host orchestration.
- Cross-layer BoxWeave discovery, delegation, controller management, automatic multi-level ingress propagation, special subtree deletion safeguards, subleasing accounts, metering and billing. Future subleasing is a motivation, not an implemented v1 feature.
- Imposed storage-driver selection, broad infrastructure upgrades, implicit parent-level network changes.
