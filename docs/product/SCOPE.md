# v1 scope and exclusions

## Fixed requirements

- Implementation in Go using the official MCP Go SDK and official Incus Go client/module.
- Single host per BoxWeave instance, no distributed controller or cluster scheduling.
- No dependency on Containarium, no v0.x compatibility requirement, no automatic old state migration.
- Public management protocol: authenticated Streamable HTTP MCP only. No public REST management endpoints and no Web console. The only interactive initial experience is local super-administrator credential bootstrap.
- Incus controls Boxes; Caddy manages BoxWeave-owned HTTP publishing; native Incus networking supports scoped TCP/UDP forwarding and private connectivity rules.
- Per-Box ownership and isolation; no user-facing Tenant concept.
- Linux containers first. Virtual machines, complex orchestration, and clusters are not part of the P0 promise.
- Guest commands use native Incus Exec, not a required guest SSH service.
- Every accepted mutation is represented by a durable local operation and can be reconciled after failure.
- Recursively complete installations: the same BoxWeave behavior applies to every supported nesting layer.
- Keep the underlying Incus storage driver unconstrained; capability-dependent operations report unsupported accurately.

## P0

Box lifecycle, typed updates, guest Exec, host health; managed HTTP routes and TCP/UDP port forwarding; Box-owned private ACL rules and optional internet denial; single-administrator authentication; durable async operations; strict host/resource boundaries; nesting; install/update/status/recovery CLI; adversarial and real nested acceptance.

See [tool inventory](../design/MCP_TOOLS.md).

## P1 candidates (not part of P0)

Box file read/write via Incus guest file facilities; Box snapshot list/create/restore/delete where the selected storage driver supports them. These features must obey all P0 trust and operation guarantees when implemented.

## Deferred / excluded

- Whole-Box backup/restore and automatic backup systems.
- Full isolation against public-route or port-forward hairpin paths: direct **private** Box-to-Box isolation is the agreed boundary; published public websites remain reachable to other Boxes.
- Public administrative REST or Web interface, OAuth multi-tenant control plane, agent SSH management, unrelated host file/service management, unmanaged Caddy routing, multi-host orchestration.
- Imposed storage-driver selection, broad infrastructure upgrades, implicit parent-level network changes.
