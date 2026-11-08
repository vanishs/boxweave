# Networking, ACL and public exposure

## Agreed network policy

- **Default:** private direct traffic between managed Boxes is denied. IPv4 enforcement is required; when IPv6 is enabled its private paths must also be denied. If the native network cannot enforce IPv6 safely, disable only the *owned/controlled* IPv6 path and reverify. If no safe configuration exists, refuse Box creation/start rather than silently falling back to an unisolated network.
- **Allow grants:** explicit directional Box A -> Box B, with optional IP/protocol/port or range restrictions. Public-facing input can express source/destination IP patterns when safe, but ownership is bound to stable Box UUIDs, not IP strings alone.
- **Internet:** Boxes have outbound internet access by default; selected Boxes may disable it.
- **Published services:** A Box may access another Box's public website over its published Caddy address, even while their private direct network remains blocked.
- **Deferred:** attempting to prevent private-isolation bypass via public-route/forward hairpin paths. Do not advertise all-path isolation.

## Private ACL enforcement

Implement through native Incus bridge/NIC ACL or an equivalent fail-closed, narrowly managed policy. Validate actual same-bridge packet paths, NIC inheritance, subnet assignment, IPv4/IPv6 and nested bridge forwarding before claiming isolation. Native bridge ACL capabilities must be checked in the target Incus version; do not assume a restricted Project or configured ACL equals packet-level enforced isolation. Never change unrelated host/foreign IPv6 settings in order to mask a failure; incompatible networks return a fail-closed result.

Maintain rules as identity-anchored desired state; resolve current IPs from native owned Instances and refresh on address changes. Use native ETag/revision coordination and preserve unmanaged NIC/profile/ACL settings. If an address or ownership mapping is ambiguous, deny grant application.

## Caddy HTTP routes

The BoxWeave-owned Caddy instance publishes **HTTP-only Box application websites in P0**, not the MCP management endpoint. By default MCP listens on loopback HTTP only; its optional Cloudflare Tunnel/HTTPS proxy is external operator configuration and is not a managed `http_route_create` target.

MCP accepts `box_id`, `guest_port`, and a validated domain—not arbitrary upstream URLs, local host ports, raw Caddy JSON, or destination IP. Resolve the Box UUID to the current verified guest address. Keep routes in a durable BoxWeave-owned namespace and reconcile only that namespace into a dedicated managed Caddy instance. **P0 does not provision site HTTPS, certificates, ACME, HTTP-to-HTTPS redirects or port 443**. Use explicitly HTTP-only managed Caddy site addresses/config so automatic HTTPS cannot activate by accident. Validate the owned HTTP port/domain and conflicts. Direct public HTTP is unencrypted; operators needing secure public transport must separately configure their own external HTTPS termination. Never modify a foreign web server.

## TCP/UDP forwards

MCP accepts a requested protocol/host bind port + `box_id` + `guest_port`, not a target IP. Prefer native Incus forwarding facilities where supported, selecting network forward vs proxy device only after compatibility and nested tests. Check bind-port exclusivity, identity, owner, IPv4/IPv6 and restart restoration. Removal never touches a forward that ownership cannot prove.

## Nested networking is ordinary networking

Every layer may create its own route and forward records targeting **only its directly owned Boxes**. A service inside an L2 Box can be reached through parent layers only if operators independently configure ordinary owned-Box routes/forwards and network permissions at each layer. BoxWeave neither discovers the full ancestry nor automatically traverses or programs parent Caddy/Incus configurations. A child cannot edit its parent's routes, firewall, or credentials. This is the same networking contract at every level, not a special multi-layer publishing API.

## Async and Tracking coverage

Every accepted route/forward/ACL change journals an immutable intent, receives the same Tracking ID through MCP, Go domain/worker context, Caddy/Incus adapter and final verified native effect. A request timeout or stale ownership check does not imply that Caddy/network effects were absent. Per-port and per-route claims prevent conflicting mutations; query the original operation rather than retrying it blindly.

## Safety checks

Test real IPv4 denial, safe owned IPv6 disable/denial or refusal when unsupported, explicitly granted peer access, public HTTP-route allowance, internet opt-out, IP reuse, Caddy outage/restart, absence of automatic HTTPS/port 443/redirect/ACME, foreign route preservation, arbitrary-upstream rejection, nested reachability and no unintended parent modifications. See [P0 operational policies](../decisions/P0_OPERATIONAL_POLICIES.md).
