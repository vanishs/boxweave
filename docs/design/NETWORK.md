# Minimal native networking — no managed proxy

**Permanent scope:** BoxWeave delegates networking to typed Incus native APIs and controls only its owned Box resources. It **does not ship, install, upgrade or configure Caddy**, HTTP site routers, TLS certificates or a custom forwarding daemon.

## Do not confuse native SDK method families

The SDK's `GetInstancePortForwardConn` opens a connection to an instance, **not** a lasting host listener. The SDK's `CreateNetworkPeer` peers networks, **not** two Boxes. Use an instance-owned `proxy` device through `UpdateInstance` for P0 `port_forward_*` and NIC-attached native NetworkACLs for `box_peer_set`. See [20-tool native SDK map](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md).

## Native networking fit

Incus `proxy` devices provide host TCP/UDP listeners owned by the target instance. Native `network-forward` objects belong to an Incus network and may survive instance deletion. For v1 P0's narrow host-port publishing, **one proxy-device backend** avoids an unnecessary second ownership/reconciliation system. See [Incus proxy docs](https://linuxcontainers.org/incus/docs/main/reference/devices_proxy/), [network forwards](https://linuxcontainers.org/incus/docs/main/howto/network_forwards/) and [SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md).

Incus's official bridge ACL documentation also states that network-level bridge ACLs do not enforce intra-bridge isolation; NIC-level attachment is a distinct implementation with limitations. Validate the exact NIC policy and same-bridge IPv4/IPv6 packet denial before first start.

## Private Box isolation and simple test peer access

- Direct private Box-to-Box traffic is **denied by default**. Enforce both IPv4 and enabled IPv6, including same-bridge paths, after real native compatibility tests. A restricted Incus Project is not evidence of packet isolation.
- `box_peer_set(source_box_id,target_box_id,allow)` alone can allow or revoke **directional** private access between two verified currently owned Boxes. A one-way A -> B grant supports client-initiated requests and normal replies but does not authorize B to initiate new independent connections to A. Native behavior must be tested.
- Do not accept arbitrary IP/CIDR/port/protocol rules or edit foreign ACLs, profiles, host interfaces or firewall. No separate `acl_rule_*` management or other network tool. Expose current effective peer grants in `box_get`. Bind grants to native instance UUID/incarnation and reconcile stale addresses/identity without transferring rights on IP/name reuse.
- If IPv6 filtering cannot be proven, safely disable only BoxWeave-controlled IPv6 and reverify, otherwise refuse Box creation/start; never disable unowned host IPv6. Internet egress defaults to allowed, with a simple per-Box block option.
- A Box can still reach a service intentionally exposed on a mapped host port where underlying routing/hairpin behavior permits; this is **not** a substitute for guaranteed private peer connectivity and not a claim of all-path isolation.

## Observed permission split: native Bridge ACL works; confined user cannot edit rules

The [2026-11-08 real L1/L2 Incus 7.6 SDK batch](../verification/SDK_NATIVE_BATCH_RESULTS_2026-11-08.md) demonstrated NIC-attached **Bridge** IPv4/IPv6 private deny, host-local **admin** updates allowing A→B only, B→A new-flow denial and successful post-revoke new-flow deny. TCP/UDP packet exchanges also succeeded. But the `incus`-only confined runtime account could **attach an existing ACL** to its own instance NIC while **creation/editing of the inherited `default` Project ACL was denied**. In a fresh auto-confined Project, trying `features.networks=true` before any instance failed with **`OVN is required for projects with "features.networks" enabled`**.

**Implementation gate, not authorization to change scope:** before `box_peer_set` is implemented, prove a minimal native credential path to edit only the required **owned** ACL claims; evaluate restricted TLS against the exact resource first. Otherwise bring a narrowly scoped ACL-only host-local privilege boundary to the owner for approval. Do not convert this test's admin-only ACL calls into a root-powered MCP runtime, claim generic Bridge packet safety by Project name, automatically add OVN, or implement a custom firewall. Verified A→B reply and B→A initiation behavior must survive dynamic address and IPv6 checks before full P0 PASS.

**Separately**, the same native runtime user initially received `Proxy devices are forbidden`; an explicitly administered **`restricted.devices.proxy=allow` in only its owned test Project** enabled the official instance proxy for tested **loopback TCP HTTP 200** and **UDP echoed payload**. The test also showed that the guest service bind address and native proxy's `connect` target matter: pointing to the verified guest IP succeeded. Do not widen to arbitrary host binds or raw targets. Test-only proxy devices and the temporary grant were independently removed.

## Bridge NIC ACLs are not UUID-based policies

Incus v7.6.0 [native ACL documentation](https://github.com/lxc/incus/blob/v7.6.0/doc/howto/network_acls.md) confirms that Bridge **network-level** ACLs cannot block same-bridge peer traffic and that **Bridge ACL group/network selectors are not supported**. The selected P0 path must use a **NIC-attached** ACL, with rules anchored to verified current guest IPv4/IPv6 and checked MAC/NIC identity where available; the Box UUID is a BoxWeave ownership key, **not a native network ACL selector**. Native `security.mac_filtering`, `security.ipv4_filtering`, `security.ipv6_filtering` are possible anti-spoof features; they are not enabled by merely assigning an ACL.

Before granting A→B, prove that both current native address bindings are trustworthy. On DHCP/IPv6 change, Box deletion/recreation, ownership conflict, unsupported anti-spoof or rule update failure, fail closed: withhold/revoke the affected peer grant or block start rather than transferring access to a new occupant of an old IP. Bridge ACLs also have baseline service rules that precede ACL rules and may rewrite `reject` to `drop`; assert actual traffic behavior, not exact textual rule interpretation. Test permitted initiated traffic **and replies** versus newly initiated reverse traffic, as well as old established flows after revocation.

Native IP/MAC filtering can restrict nested L2 forwarding with another MAC. For `nesting=true`, prefer an independent inner managed bridge using guest NAT through the parent's permitted NIC; verify this works without relaxing anti-spoof globally. Details: [second-pass SDK review](../research/SDK_GAP_CLOSURE_2026-11-07.md), [nested topology](NESTING.md). Do not create a generic IP/CIDR ACL service.

## Guard at first start and claim only applied security

For the supported native topology, create the new owned instance **stopped**; install and verify its required IPv4 and enabled IPv6 private-isolation rules on the correct effective NIC **before any first start or autostart**. Recheck before a later start if topology, NIC or applied protection could have changed. A failed/unsupported guard blocks start and reports the actual instance identity/cleanup status; never silently start the Box reachable and then try to repair its policy. Do not copy a generic Network Tenant service.

On an inherited Profile NIC, an instance-local override may be required for `security.acls`: base it on the complete effective device, preserve unrelated fields and ACLs, update with native ETag and verify the *actual* effective attachment. A desired database rule, an ACL name attached to a NIC, and packets proven isolated are **three different evidence levels**. `box_get`/operation receipts must not claim a peer grant/revocation is effective without native readback; positive acceptance additionally requires real same-bridge IPv4/IPv6 packet tests. A directional A->B grant must permit A-initiated flows and their responses without permitting B-initiated flows.

For P0, native `proxy` devices belong to the exact instance. Journal each named owned device and verify native removal with instance deletion; reconcile stale journal claims without transferring ownership on name/IP reuse. A separately adopted network-forward backend would require its own lifetime and cleanup policy, but is **not** part of P0. Recovery must never resurrect stale exposure or remove a foreign listener. Probe guest, host-local and independent LAN reachability separately; do not add a custom firewall/proxy/reconciler platform to emulate an unsupported topology.

See [incident evidence and implementation cautions](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Native Incus TCP/UDP forwarding — required

Keep `port_forward_list/create/delete` to map a host listening `protocol/address/port` to an exact verified owned `box_uuid + guest_port`. Implement the P0 host-port capability through **instance-owned Incus proxy devices** (`type=proxy`, `bind=host`, typed `listen`/`connect`) and guarded instance update/readback; do **not** build a forwarding service. Network-forward objects are network-scoped IP-forwarding resources, not an interchangeable host-port backend. Support them only if a separately established need and native topology justify another explicitly designed lifecycle.

- A listener on host `127.0.0.1` supports operator-local consumers/proxies. Public/wildcard binds require explicit caller choice, administrative authority and a port-conflict/exposure check; never bind every interface silently.
- Reject arbitrary target IPs, host service targets, host mount or proxy configuration, listener collisions and unsupported native network type. Forward identity is durable and verified on create/delete and after native restarts; no foreign forwards edited. For the P0 instance-owned `proxy` backend, a deleted Box should not have a surviving native proxy device: verify any stale local record against the original native identity/tombstone and do not adopt a same-name or foreign forward. If an independent **network-forward object** backend is explicitly added later, separately design exact-identity orphan cleanup then; do not implement it as a P0 fallback.
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
