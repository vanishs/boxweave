# Minimal native networking — no managed proxy

**Permanent scope:** BoxWeave delegates networking to typed Incus native APIs and controls only its owned Box resources. It **does not ship, install, upgrade or configure Caddy**, HTTP site routers, TLS certificates or a custom forwarding daemon.

## Private Box isolation and simple test peer access

- Direct private Box-to-Box traffic is **denied by default**. Enforce both IPv4 and enabled IPv6, including same-bridge paths, after real native compatibility tests. A restricted Incus Project is not evidence of packet isolation.
- `box_peer_set(source_box_id,target_box_id,allow)` alone can allow or revoke **directional** private access between two verified currently owned Boxes. A one-way A -> B grant supports client-initiated requests and normal replies but does not authorize B to initiate new independent connections to A. Native behavior must be tested.
- Do not accept arbitrary IP/CIDR/port/protocol rules or edit foreign ACLs, profiles, host interfaces or firewall. No separate `acl_rule_*` management or other network tool. Expose current effective peer grants in `box_get`. Bind grants to native instance UUID/incarnation and reconcile stale addresses/identity without transferring rights on IP/name reuse.
- If IPv6 filtering cannot be proven, safely disable only BoxWeave-controlled IPv6 and reverify, otherwise refuse Box creation/start; never disable unowned host IPv6. Internet egress defaults to allowed, with a simple per-Box block option.
- A Box can still reach a service intentionally exposed on a mapped host port where underlying routing/hairpin behavior permits; this is **not** a substitute for guaranteed private peer connectivity and not a claim of all-path isolation.

## Native Incus TCP/UDP forwarding — required

Keep `port_forward_list/create/delete` to map a host listening `protocol/address/port` to an exact verified owned `box_uuid + guest_port`. Implement through native Incus **proxy devices or network forwards**, whichever is appropriate and verified on the supported host network; do **not** build a forwarding service.

- A listener on host `127.0.0.1` supports operator-local consumers/proxies. Public/wildcard binds require explicit caller choice, administrative authority and a port-conflict/exposure check; never bind every interface silently.
- Reject arbitrary target IPs, host service targets, host mount or proxy configuration, listener collisions and unsupported native network type. Forward identity is durable and verified on create/delete and after native restarts; no foreign forwards edited.
- TCP and UDP are supported **only when the underlying Incus network, listener and guest capabilities actually support them**. Return UNSUPPORTED rather than silently falling back to a custom proxy.
- For nested L1/L2 BoxWeave, all forwards remain **layer-local**. Reaching an inner service outside its parent may require a separately authorized mapping at each relevant layer. No recursive configuration automation.

Operator-installed reverse proxies/Tunnels and DNS can target a host-published TCP port; BoxWeave neither creates nor administers those services. Direct HTTP exposed on a mapped port is unencrypted unless the app or operator supplies TLS.

## HTTP/TCP service publishing and real LAN reachability

The `port_forward_create` default **loopback host bind** is intentionally only host-local. To let another machine on the host's physical LAN reach a service, an `admin` must select the **actual reachable host LAN IP** and host port, explicitly acknowledge that exposure, and verify that native forwarding can bind it; wildcard bind is never inferred. Native proxy devices and network forwards have different address/ownership semantics and cannot be treated as interchangeable without capability tests. Incus network-forward objects may be attached to a network rather than an instance and must be tracked/deleted separately if used.

The guest app must listen on a guest-reachable interface, not just guest `127.0.0.1`, and remain alive independently of an MCP Exec request. For a supported Ubuntu LTS Box, the normal approach is a **guest-managed service** (e.g. systemd unit via `box_exec`), not a BoxWeave process supervisor. Verify in order: guest app response -> native forward readback -> L0 host bind -> **independent LAN machine** HTTP/TCP probe. A successful guest or host-local curl is *not* proof of LAN reachability. Host firewall, IP selection, physical routing, VLAN and operator network policy may prevent LAN traffic; report the blocking layer. BoxWeave must never modify unrelated host firewall/interface settings to make a test pass.

When serving from a **nested L1 Box**, layer-local forwarding does not automatically publish to the physical LAN. An operator separately configures e.g. `L1 host (Dev Box D):18080 -> L1 Box B:3000` and `L0 host LAN:28080 -> Dev Box D:18080`. Use authorized binds in each layer and independent receipts; then check from the real LAN witness. This is ordinary composition, **not** cross-layer administration. HTTP on a native forwarded port is plaintext unless the application/operator supplies TLS.

## Required verification

Same-bridge IPv4/IPv6 private denial and explicit A->B allow/revoke; source/destination IP changes; spoof/foreign resource refusal; internet egress toggle; owned loopback **and explicitly selected host LAN IP** TCP/UDP binds; independent same-LAN witness; wrong guest listener/host firewall classification; port collision; route availability after restart; associated forward cleanup on Box delete; outer/inner manually composed forwarding and external probe; no Caddy process/dependency or unmanaged network edits.

Sources: [Incus native proxy device](https://linuxcontainers.org/incus/docs/main/reference/devices_proxy/), [Incus network forwards](https://linuxcontainers.org/incus/docs/main/howto/network_forwards/), [Incus ACL native limitations](https://linuxcontainers.org/incus/docs/main/howto/network_acls/). See [boundary](../decisions/PRODUCT_BOUNDARY.md).
