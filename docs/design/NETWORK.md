# Networking, ACL and public exposure

## Agreed network policy

- **Default:** private direct traffic between managed Boxes is denied, including IPv4 and IPv6 where enabled.
- **Allow grants:** explicit directional Box A -> Box B, with optional IP/protocol/port or range restrictions. Public-facing input can express source/destination IP patterns when safe, but ownership is bound to stable Box UUIDs, not IP strings alone.
- **Internet:** Boxes have outbound internet access by default; selected Boxes may disable it.
- **Published services:** A Box may access another Box's public website over its published Caddy address, even while their private direct network remains blocked.
- **Deferred:** attempting to prevent private-isolation bypass via public-route/forward hairpin paths. Do not advertise all-path isolation.

## Private ACL enforcement

Implement through native Incus bridge/NIC ACL or an equivalent fail-closed, narrowly managed policy. Validate actual same-bridge packet paths, NIC inheritance, subnet assignment, IPv4/IPv6 and nested bridge forwarding before claiming isolation. Native bridge ACL capabilities must be checked in the target Incus version; do not assume configured ACL equals enforced isolation.

Maintain rules as identity-anchored desired state; resolve current IPs from native owned Instances and refresh on address changes. Use native ETag/revision coordination and preserve unmanaged NIC/profile/ACL settings. If an address or ownership mapping is ambiguous, deny grant application.

## Caddy HTTP routes

MCP accepts `box_id`, `guest_port`, and a validated domain—not arbitrary upstream URLs, local host ports, raw Caddy JSON, or destination IP. Resolve the Box UUID to the current verified guest address. Keep routes in a durable BoxWeave-owned namespace and reconcile only that namespace into a dedicated managed Caddy instance. HTTPS and TLS automation must be ownership-safe, with checks for DNS/reachability and port conflicts. Never modify a foreign web server.

## TCP/UDP forwards

MCP accepts a requested protocol/host bind port + `box_id` + `guest_port`, not a target IP. Prefer native Incus forwarding facilities where supported, selecting network forward vs proxy device only after compatibility and nested tests. Check bind-port exclusivity, identity, owner, IPv4/IPv6 and restart restoration. Removal never touches a forward that ownership cannot prove.

## Nested exposure

Every layer can create the same route and forward records in its own scope. To publish an L2 service externally, L1 and L0 must grant relevant ingress explicitly; neither child nor its token may alter the parent. Automatic delegated publish flows, if added, must use scoped authorization, journaled mutations and verifiable ownership at **each** intervening layer.

## Safety checks

Test private denial, explicitly granted peer access, public-route allowance, blocked internet egress, IPv6, IP reuse, Caddy outage/restart, foreign route preservation, arbitrary-upstream rejection, nested reachability and no unintended parent modifications.
