# S4 network identity, peer topology, and Linux Bridge alternatives (2026-11-17 PT)

> **Research / decision preparation only; NOT an architecture decision or S4-B2 signoff.** This document consolidates the owner's network-design discussion and publicly checkable upstream mechanisms. It does not authorize code, API, installer, host, network, or milestone changes. The active network implementation remains isolated in [Draft PR #153](https://github.com/vanishs/boxweave/pull/153), and the product-landscape discussion remains in [Draft PR #155](https://github.com/vanishs/boxweave/pull/155). Both are separate from this documentation-only proposal. No new native packet tests were run for the alternatives in this document.
>
> **Source snapshot:** protected `v1` at `e3d85a65397338dc5f082cee98dfdabe9d6017e1`; PR #153 at `61451b3da11f3c8031f39a4fe0247591a34ddcee` when this note was prepared. Recheck refs and the current [product boundary](../decisions/PRODUCT_BOUNDARY.md), [network design](../design/NETWORK.md), and [S4-B plan](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/development/S4_B_SIX_STAGE_PLAN_2026-11-16.md) before implementation. Current source/evidence is more authoritative than research or a green CI badge.

## 1. Product need: arbitrary peer pairs, not an IP-based ACL editor

The motivating user requirement is deliberately small:

1. **Default isolation:** independently owned Boxes cannot directly communicate with each other unless authorized.
2. **On-demand mutual connectivity:** A↔B can be enabled and later removed, while both Boxes still have only an ordinary `eth0` and retain their separate Internet/host-port-forward behavior.
3. **Independent relationships:** A↔B and B↔C do **not** imply A↔C. A "peer pair" is a graph edge; a "peer group" is normally a clique, so these are distinct product models.
4. **Lifetime-safe identity:** an old authorization to B must not grant access to a newly created C when C inherits B's IPv4/IPv6 address, MAC or some reusable kernel number.
5. **No generic host authority for Agents:** BoxWeave operations remain scoped to verified owned UUIDs/Project, with bounded host-side effects and honest failure/UNKNOWN handling.

**Product intent is mutual communication, not directional policy as an end in itself.** The currently owner-approved permanent v1 API nevertheless specifies one **directional** `box_peer_set(source_box_id,target_box_id,allow)` primitive. Two directed permissions can express the desired mutual pair without adding a public Tool. Whether to expose a symmetric convenience action is an *unapproved* product/API decision, not implied by this research.

"Wrong authorization", "unauthorized cross-Box access" and "IP-reuse privilege inheritance" are not three unrelated user features. Unauthorized A→C is the safety failure; stale B→C address reuse is one concrete cause. Other distinct causes include incorrect broad rules, spoofed addresses, and partially applied policy updates. The fundamental invariant is **no new/old Box gains a pair it was not explicitly authorized to have**.

**Scope caveat:** "A↮C" describes direct network reachability, not a claim that a fully controlled Box B cannot intentionally relay data between A and C at the application layer. Preventing an authorized intermediary from acting as a proxy is a different threat model.

## 2. Why the existing S4-B2 investigation became complex

The current native candidate places sibling Boxes on a dedicated Incus-managed Linux Bridge, configures per-NIC Incus network ACLs plus IPv4/IPv6/MAC filters, and uses protected SQLite identity/lease claims, Incus operations, host-side inspection and restricted root actions to guard changes. It is not merely an "allow A's IP to reach B's IP" setter.

The [native B2 evidence](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_NATIVE_SECURITY_2026-11-16.md) and [B2 plan](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/development/S4_B_SIX_STAGE_PLAN_2026-11-16.md) distinguish the following actual or experimentally reproduced problems:

| Issue | Mechanism | What must be preserved or reconsidered |
| --- | --- | --- |
| IP reuse across Box lifetimes | A former B address can be assigned to a new C; rules keyed by old address can accidentally match C | Authorization must bind the current owned Box identity/incarnation, not only the address |
| Incus ACL partial kernel fan-out | An ACL PUT error / database rollback has been observed while some per-NIC effective nft rules still changed; a failed **grant** allowed real A→B traffic | DB/API readback alone is not proof of runtime denial; fail-closed pre-change barrier and packet evidence matter |
| Same-bridge address spoofing | Foreign/unmanaged sender spoofed an approved address and reached a target in native tests | Validate ingress attachment identity and isolate unmanaged bridge users, not just source IP |
| Revocation of existing sessions | Connection tracking can outlive an ACL update; broad conntrack deletion can affect unrelated flows with overlapping IPs | Decide explicit old-session semantics and test a narrower enforcement point; never assume "revoke" equals immediate packet stop |
| Cross-Project and host-global effects | Shared ACL scope, root conntrack, or Incus Scriptlet permission changes can affect foreign/legacy users | Keep foreign resources and existing authorizations unchanged, even if exhaustive overlap experiments are staged separately |
| Lost ACK, crash, stale leases | A failed/uncertain native operation can leave stale security state | Original-operation receipts, claims, bounded recovery and `UNKNOWN` remain necessary |

The failed-grant live-packet witness is [run #44034201810](https://github.com/vanishs/boxweave/actions/runs/44034201810). The matching "SUCCESS" CI result denotes **successful reproduction of a P0 failure**, not a passing product acceptance. The B2 gate at this research snapshot was `S4_B2_PARTIAL`; this note does not change it.

A simpler network implementation should remove avoidable IP-identity and per-NIC ACL fan-out complexity **without silently deleting the actual isolation guarantee**. It may be reasonable to simplify the breadth of proof for a dedicated-host deployment, but it is not acceptable to let unauthorized traffic pass.

## 3. What Incus, Linux Bridge, OVS and OVN actually do

- **Incus:** owns container/VM lifecycle and NIC configuration, including a guest `eth0` and a host-side veth on bridged containers. It can read the instance UUID, its devices and the host-side interface. Incus supports a custom `host_name` for a bridged NIC and has managed network ACL, anti-spoof, port isolation and VLAN options.
- **Linux Bridge:** the kernel's ordinary Layer-2 switch, often provisioned as an Incus managed Bridge. It forwards based on MAC/FDB and VLAN membership and knows ingress/egress ports, **not Box UUIDs**.
- **OVS (Open vSwitch):** a separate software-switching project, an alternative switch implementation to Linux Bridge.
- **OVN (Open Virtual Network):** a separate logical-network control plane on OVS. Incus can drive OVN networks; boxes normally **do not** install an OVN driver internally. OVN requires OVS/OVN services and databases on the relevant host/controller infrastructure, increasing operational and version-maintenance cost.

An Incus-managed Linux Bridge and an Incus-managed OVN network are alternative network backends, not two simultaneous meanings of "Incus Bridge". When a Box is created on an OVN NIC, its normal guest view is still an ordinary NIC.

References: [Incus NIC configuration](https://linuxcontainers.org/incus/docs/main/reference/devices_nic/), [Incus Bridge networks](https://linuxcontainers.org/incus/docs/main/reference/network_bridge/), [Incus ACL behavior](https://linuxcontainers.org/incus/docs/main/howto/network_acls/), [Linux Bridge kernel reference](https://docs.kernel.org/networking/bridge.html).

## 4. Identity: a permanent control-plane ID is not automatically a packet-level ID

| Identifier | Stable across Box deletion/recreation? | Usable for packet enforcement directly? | Judgment |
| --- | --- | --- | --- |
| BoxWeave-owned Box UUID + incarnation | New resource should receive a new identity | No; requires trusted mapping into native network objects | **Authoritative authorization subject** |
| IPv4/IPv6 address | No; lease/static address can be reassigned | Yes, but reuse/spoof must be handled | Addressing/routing only, **not** authorization identity |
| MAC address | No absolute uniqueness; can be copied or reused | Yes, but subject to filtering/spoof checks | Supporting anti-spoof check only |
| Linux interface index (`ifindex`) | No; kernel reuses integers | Yes (`iif`/`oif`) | **Not** an enduring authorization ID |
| Linux interface name (`host_name`) | Name can be reused unless owner deliberately prevents it | Yes (`iifname`/`oifname`) | Promising **controlled surrogate**, not intrinsic UUID |
| OVN Logical Switch Port name/UUID | Port UUID is unique per native object; Incus derives its instance-port *name* using instance UUID | Yes (`inport`/`outport`) in OVN ACL | Strong logical port approach, but more infrastructure |
| eBPF/Cilium identity | Workload/cgroup/label maps can be bound to runtime attachments | Only with a correct maintained enforcement program | High control-plane and compatibility burden |
| SPIFFE workload identity | Cryptographic identity can be stable per workload policy | Not directly at a Linux Bridge forwarding hook | More suitable for application/service authentication |

Important nuance: **no packet header contains the Box UUID**. Even "IP-free authorization" means the controller authorizes by UUID, then securely binds that UUID to a current interface/logical port. The new object must never inherit an old pair merely because an address, numeric ID or attachment has been recycled.

For ordinary Linux interfaces, the kernel has no universally permanent non-reused per-Box port identity. Assigning a non-reused **name** is a controller policy, not a kernel guarantee; a destroyed interface can later be recreated under the same name. A 15-byte interface-name limit means a full UUID does not fit; truncating a UUID without collision handling is not a proof of uniqueness.

## 5. Native alternatives and their limits

| Candidate | Arbitrary overlapping peer pairs (A↔B, B↔C, A↮C) with one ordinary guest `eth0`? | Uses IP to select an authorized peer? | Host / controller tradeoff |
| --- | --- | --- | --- |
| Existing shared Incus Bridge + dynamic NIC ACL | Designed for it, but S4-B2 still has P0 edge cases | Yes, in the present implementation | Reuses Incus; IP reuse, ACL fan-out, Scriptlet, conntrack |
| One Bridge per pair + extra NICs | Yes | No identity need for the pair membership | Bridges/NICs multiply; guest networking and routes get complicated |
| One Bridge per communication group | Only for a clique, not arbitrary overlapping pairs | No | Simpler groups, but group overlap reintroduces NIC/routing complexity |
| One Linux Bridge + VLAN access ports | **No**, not generally for overlapping peer edges with one untagged guest NIC | VLAN membership is port-based | Light native grouping, but wrong graph semantics |
| Linux Bridge `security.port_isolation` | **No**; binary isolated/non-isolated ports, not arbitrary pair ACL | No | Excellent default isolation building block, not selective grant by itself |
| Linux Bridge FDB / locked-port / MAB | **No**; MAC learning/attachment authorization does not express pairwise source→destination policy | May use MAC | Not a substitute for peer ACL |
| OVS + OVN Logical Ports / Port Groups | Yes via native port-based ACL rules | **Can avoid IP** in an exact `inport`/`outport` policy | Extra host services and direct OVN NB integration/ownership issues |
| Linux Bridge + per-veth eBPF | Potentially yes | Could avoid IP with UUID-to-attachment mapping | BPF programs, kernel support, attach lifecycle and debugging complexity |
| **Linux Bridge + fixed host veth name + nftables** | **Potentially yes** | **Can avoid IP** for the peer decision | No new OVN service; name allocation, ACL interaction, bypass and recovery remain unproven |

**Linux Bridge does not provide a built-in arbitrary port-to-port permission graph.** Native port isolation, VLAN, FDB and locked/MAB ports are useful but have different semantics. Therefore, under the constraint "single Bridge + single ordinary eth0 + arbitrary graph", some separate filtering/forwarding policy engine is required.

### 5.1 The VLAN question: reuse was not the real blocker

IEEE 802.1Q VLAN ID is 12 bits; ordinary IDs **1–4094** are usable, not 65,536. Reusing VLAN 100 **after all old members have been removed** is not inherently an IP-style identity vulnerability. The real risks are stale membership and cross-VLAN forwarding via a router, including the host.

Each ordinary access port can have only one default untagged VLAN/PVID. A guest NIC using only untagged traffic cannot simultaneously behave as two independent default access ports. Incus supports `vlan` and `vlan.tagged`, but tagged VLANs normally require the guest to handle tags and VLAN subinterfaces. Example: A↔B in VLAN 100 and B↔C in VLAN 200 would require B to address both contexts, rather than simply setting a shared pair permission. One-way tricks with multiple egress VLAN memberships do not provide normal symmetric bidirectional connectivity.

VLANs also affect network service design: DNS, DHCP, IPv4/IPv6 gateways, NAT, and inter-VLAN routing are separate concerns on an Incus-managed Bridge. Keeping the original Internet toggle and a simple eth0 may be substantially harder than creating an isolated VLAN.

### 5.2 Why port isolation cannot be "overridden by accept"

Linux Bridge port isolation prevents isolated ports from directly communicating with each other. It does **not** implement a pair whitelist; the kernel decision is not automatically bypassed by adding a separate nftables `accept` rule. An initial quarantine using port isolation cannot be assumed to coexist with dynamic pair release on the same path. The lifecycle would need a different, tested design.

### 5.3 OVN is technically attractive but operationally costly

OVN ACL expressions support logical `inport`/`outport` matching and Port Groups; Incus 7.6 generates logical port names from the network prefix, instance `volatile.uuid` and NIC name. This is closer to a true identity-derived logical port than IP or ifindex.

However, **Incus public ACL "source is another ACL group" is not automatically a logical-port match**: Incus 7.6's OVN ACL translator also generates IPv4/IPv6 address-set matches. Direct OVN Northbound rule management may be needed to guarantee port-only matching, which raises a shared-control-plane contract question: will Incus preserve independently written ACLs through network/NIC updates? That is not verified here.

Incus upstream [issue #3018](https://github.com/lxc/incus/issues/3018) documented stale OVN Port Group UUID membership; [PR #3140](https://github.com/lxc/incus/pull/3140) added cleanup in April 2026. A merged upstream fix does not prove that a given installed Incus 7.6 build includes it; verify the exact release and native behavior before depending on it.

Sources: [Incus 7.6 OVN port name implementation](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/driver_ovn.go), [Incus 7.6 OVN ACL translation](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/acl/acl_ovn.go), [OVN architecture/man pages](https://www.ovn.org/support/dist-docs/ovn-nb.5.html).

## 6. Public open-source precedents, not ready-made BoxWeave solutions

The adjacent projects in [PR #155](https://github.com/vanishs/boxweave/pull/155) and later source study offer implementation ideas:

| Project | Confirmed useful upstream pattern | Limit of the analogy |
| --- | --- | --- |
| [Containarium: network isolation design](https://github.com/FootprintAI/Containarium/blob/main/docs/security/NETWORK-ISOLATION-DESIGN.md) | Actual Incus/host veth experiments: per-container **tc ingress** sees sender traffic; bridge/veth tc egress were insufficient in those tests | Its eBPF policy still uses CIDR/domain/IP components; not a complete Box UUID→UUID peer API |
| [AgentBox](https://github.com/leciric/agentbox/blob/main/docs/architecture.md) | Incus agent-box lifecycle and a separate controller/API | No equivalent non-IP pairwise port-ACL implementation established |
| [Pixels](https://github.com/deevus/pixels) | Incus sandbox lifecycle and nftables-based egress policies | Egress policy primarily runs in the guest and uses domain/IP allowlists, not host-veth pair permissions |
| [OpenStack Neutron OVN PVLAN](https://github.com/openstack/neutron/blob/master/neutron/services/pvlan/drivers/ovn/driver.py) | Real open-source Port Group, isolation/drop, community grouping, OVSDB transaction and cleanup code | Its cross-chassis fallback also uses IP sets; PVLAN communities model groups more naturally than arbitrary graph edges |
| [OpenStack Neutron ML2/OVN](https://github.com/openstack/neutron/blob/master/neutron/plugins/ml2/drivers/ovn/mech_driver/ovsdb/ovn_client.py) | Security Group→OVN Port Group membership and lifecycle management | Security Group selectors can still use address sets; not an IP-free guarantee |
| [Kube-OVN Go client](https://github.com/kubeovn/kube-ovn/blob/master/pkg/ovs/ovn-nb-port_group.go) | `PortGroupAddPorts` / `PortGroupRemovePorts` / `PortGroupSetPorts` and transactional OVSDB operations | Pulling in Kubernetes controllers would overgrow BoxWeave; IP-based NetworkPolicy cases remain |
| [OVN-Kubernetes](https://github.com/ovn-kubernetes/ovn-kubernetes) | Real fleet reconciliation of logical ports, Port Groups and ACLs | Kubernetes networking/runtime assumptions differ from single-host Incus |

These projects prove there are maintained primitives and practical patterns, **not** that their full security, performance or version compatibility has been independently accepted in BoxWeave. Do not implement their entire controllers just to gain a few port-to-port rules.

## 7. Candidate to investigate last: Linux Bridge + fixed identity host veth + nftables

**Status: candidate only. Not an approved replacement for B2.** This is the most promising *low-additional-host-services* avenue under the owner's constraints:

- Keep one dedicated Incus Linux Bridge (illustratively `bwbr0`) and each Box's sole ordinary `eth0`. No OVN/OVS daemon, eBPF program or guest VLAN configuration.
- Use Incus bridged NIC `host_name` to allocate a controlled host veth name per **UUID + creation incarnation**. Keep authoritative identity and peer graph in protected SQLite; never use IP, MAC, a Linux ifindex or Box name as the peer authorization subject.
- Use an **owned** nftables `bridge forward` policy to match the **actual incoming and outgoing host veth names** as an ordered pair. A↔B generates A→B and B→A entries; A↔B, B↔C does *not* add A↔C.
- Continue preserving Incus MAC/IPv4/IPv6 spoof filters and a tight Project/Bridge attachment boundary; don't accidentally treat an address as security identity simply because it still provides routing and DHCP.

### 7.1 Identity allocator and lifetime contract

1. Reserve a short, host-wide **BoxWeave-only** veth namespace (Linux interface names have a short length limit; full UUIDs will not fit). Use a protected, non-reused allocation sequence/namespace and retained tombstones or equivalent; do not recycle names on ordinary deletion. Detect collisions with *all* existing host links and currently attached Incus devices, including foreign roots.
2. On every start/restart/recovery, compare the exact Incus instance UUID, NIC device, assigned `host_name`, native host link and saved incarnation; an unexpected name/UUID/link association must fail closed. Never dynamically substitute an arbitrary current name for a previously approved UUID.
3. Treat DB restore, duplicated installs, rollback, counter exhaustion and operator-created same-name links as **explicit collisions**. A monotonic allocator by itself is not proof if a stale backup is restored; the host's current state and old policy must be reconciled before any Box is released.
4. On B stop/delete, retire B's *runtime attachment* and withhold pair permissions to stale endpoints, even though the durable grant intent may need an original-operation receipt. A new C never adopts B's UUID/incarnation or reserved veth name.
5. The host is a trusted operator boundary: a host root user can create/rename links and bypass policy. The design does not claim cryptographic identity enforcement against an untrusted host admin.

This addresses the **specific** risk "C receives B's old IP" provided name assignment and attachment checks are actually enforced; it does not claim that names are intrinsically non-reusable.

### 7.2 Conceptual packet policy, not ready-to-run nft code

The expected decisions are:

```text
trusted BoxWeave Project + dedicated bridge + current verified host veth mapping
    |
    +-- forwarded bridge packet from veth-A to veth-B
    |       if (A,B) is in approved ordered pairs -> eligible to pass
    |       else -> drop
    |
    +-- forwarded bridge packet from veth-B to veth-A
    |       if (B,A) approved -> eligible to pass
    |       else -> drop
    |
    +-- packet toward host gateway / routed through host / port-forward
            -> different hook/path; separately preserve S1 + S4 security
```

Potential nft mechanism: `meta iifname` + `meta oifname` matching an owned set of concatenated interface-name pairs in a `table bridge` / `hook forward` chain. **Syntax, hook priority, bridge scoping (for example an ingress bridge name match), FDB/flood behavior and kernel support require `nft -c` and real packet testing on the exact host.** No broad `flush ruleset`, no edit to Incus-owned tables, and no generic "run nft" MCP surface.

A dedicated nft *table* is not by itself a reliable fail-closed baseline. If the entire table disappears while the Bridge continues to forward packets, the kernel can revert to permissive switching. **The decisive design question is how the host keeps both endpoints fenced before, during and after install/reload/recovery even when the controller or its nft chain is absent.** Merely restarting a daemon quickly is not a proof of no open window.

### 7.3 Five integration traps that can invalidate the simplification

1. **Existing Incus NIC ACL conflict.** The current [`internal/native/box.go`](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/internal/native/box.go) requires one NIC, a bound ACL, default ingress `reject`, MAC/IPv4/IPv6 anti-spoof, and default egress `reject` for `internet=false`. An nft `accept` in one base chain cannot override a later Incus ACL `drop` in another chain. Either safely redesign the static Incus baseline around **one authoritative peer-enforcement point**, or retain two policy engines (which may be *more* complex than B2). The new scheme must remain deny-by-default when any component is missing.
2. **Host-routing bypass.** `bridge forward` applies to bridged switching; Box→host gateway→Box or other L3 hairpin paths can traverse `inet`/`ip`/`ip6` forwarding hooks instead. A router/port forward may also involve a different path. Exact negative packet tests must cover both families and protocol directions without breaking legitimate internet/DNS/DHCP. IP can be used for *routing checks*, not as the permanent identity authorizing a peer.
3. **Revocation and stateful fast paths.** Ordinary `bridge forward` checking every packet could stop both new and established TCP/UDP data immediately without global conntrack deletion, *if* no earlier stateful verdict or flowtable/offload bypass exists and rules truly see all relevant packets. This is a **test hypothesis**, not an accepted property; inspect flowtable/offload, packet counters and old sessions before removing existing B2 conntrack safeguards.
4. **Cross-system atomicity.** nftables supports atomic rule-set replacement (`nft -f`) and set updates, but a transaction is **not atomic across SQLite + Incus NIC lifecycle + nft + kernel packets**. Restart, missing replies, stale ETag/instance state and race with Box deletion still need original IDs, serialization, readback and `UNKNOWN` handling.
5. **Privilege and foreign scope.** Programming nftables generally needs privileged host network authority. It must be an exact bounded, separately approved helper/action, not arbitrary root shell exposed to an Agent. The policy must not match unrelated Bridges, foreign Projects, host routing traffic or Incus's tables. Restrict attach/use of the dedicated managed Bridge to verified owned Boxes.

References: [Incus `host_name` and bridged NIC](https://linuxcontainers.org/incus/docs/main/reference/devices_nic/), [Incus 7.6 `nic_bridged.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/device/nic_bridged.go), [nftables Bridge filtering](https://wiki.nftables.org/wiki-nftables/index.php/Bridge_filtering), [interface-name matching](https://wiki.nftables.org/wiki-nftables/index.php/Matching_packet_metainformation), [concatenated keys](https://wiki.nftables.org/wiki-nftables/index.php/Concatenations), [atomic rules](https://wiki.nftables.org/wiki-nftables/index.php/Atomic_rule_replacement), [accept is not final in multiple base chains](https://wiki.nftables.org/wiki-nftables/index.php/Configuring_chains), [kernel flowtable/offload](https://www.kernel.org/doc/html/latest/networking/nf_flowtable.html).

### 7.4 Prepared native experiment matrix (NOT EXECUTED)

**Venue:** a fresh, independently disposable **single-host Incus 7.6** environment, not the existing shared L0 host unless a separate exact-target authorization is obtained. Three owned sibling Boxes A/B/C, one guest `eth0` each, dedicated `bwbr0` and controlled host `host_name` mappings. Capture exact Incus/host/kernel versions and operation identifiers; leave the host clean.

| Order | Experiment | Blocking evidence / expected result |
| --- | --- | --- |
| P0-1 | Baseline inventory: live Incus NIC UUID/`host_name`, original ACL/nft hooks, IP/IPv6 state, host routes, offload | Verify true current ownership and *all* relevant packet paths before any policy edit |
| P0-2 | Start with current Incus ACLs; add an isolated nft pair allow; observe native packets and effective rules | Establish precisely why an Incus `reject` remains decisive; no invented bypass or fake PASS |
| P0-3 | In a disposable isolated setup, test a proposed single authoritative default-deny peer gate with A/B/C | A↔B works over TCP/UDP/ICMP/ARP as applicable; C denied; both IPv4 and enabled IPv6; normal guest services preserved |
| P0-4 | Extend to A↔B and B↔C with no A↔C direct access | Prove graph semantics rather than an accidentally transitive "community" |
| P0-5 | Stop/delete B; reassign B's former v4/v6 addresses to a fresh C UUID; force stale pair/name and DB-restoration cases | C never inherits authorization; name collision blocks rather than silently rebinds |
| P0-6 | Revoke A↔B during existing TCP/UDP streams, with/without conntrack and flowtables | Old and new traffic behavior explicitly measured; retain scoped conntrack safeguards if needed |
| P0-7 | Route A→host gateway→C; test v4/v6, port forwarding, spoof attempts, non-Box bridge attachments | No L3 bypass or accidental foreign access, with Internet toggle and host listener semantics intact |
| P0-8 | Faults: failed nft batch, table flushed, restart/reload, Incus NIC update partial failure, loss of ACK, DB restore | **No unauthorised open window**, or classify proposal **BLOCKED** and do not promote |
| P0-9 | Check foreign Projects/Bridges, installation and clean rollback; compare operational steps | No foreign changes; no unapproved root entrypoint; exact ownership and original evidence retained |

**Early stop conditions:** inability to keep the kernel fail-closed if the BoxWeave-owned nft policy disappears; inability to reconcile static Incus ACL denial with dynamic exact peer permission; any reproducible host-routing/flow-offload bypass; or inability to bind a new NIC safely to the correct UUID after restart. Do not spend B2 migration effort before these are resolved.

### 7.5 Decision rule before altering PR #153

A migration proposal should provide actual measurements comparing current B2 with this candidate: number of privileged components, mutable native objects, recovery states, Box guest requirements, failure-injection outcomes, host install/upgrade burden and verified packet isolation. It should **not** assume "fewer lines of code" equals greater safety.

**Research preference, not approval:** test **Linux Bridge + controlled `host_name` + nftables** first, because it keeps Incus/Linux Bridge, one ordinary `eth0` per Box and no OVN/eBPF services; retain **OVN logical ports** as a technically sound but more operationally expensive fallback. VLAN/independent Bridges remain useful for simpler group semantics, not arbitrary overlapping pair permissions with the stated guest-NIC constraint. Existing S4-B2 protections and acceptance requirements stay in force until an owner-approved design switch is backed by real native evidence.

## 8. Source and status hygiene

- **Authoritative scope:** [permanent v1 boundary](../decisions/PRODUCT_BOUNDARY.md), [S4-B existing development PR](https://github.com/vanishs/boxweave/pull/153).
- **Prior landscape and optional topology-read interfaces:** [PR #155](https://github.com/vanishs/boxweave/pull/155). Proposed new topology tools there are research only, unrelated to authorizing a network-backend change here.
- **Native B2 evidence:** [B2 security report](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_NATIVE_SECURITY_2026-11-16.md) and [B2 staged plan](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/development/S4_B_SIX_STAGE_PLAN_2026-11-16.md).
- **Upstream mechanism sources:** [Linux Bridge kernel](https://docs.kernel.org/networking/bridge.html), [Incus NIC reference](https://linuxcontainers.org/incus/docs/main/reference/devices_nic/), [Incus 7.6 bridged NIC source](https://github.com/lxc/incus/blob/v7.6.0/internal/server/device/nic_bridged.go), [nftables Bridge and atomic update](https://wiki.nftables.org/wiki-nftables/index.php/Bridge_filtering), [nftables multiple base chains](https://wiki.nftables.org/wiki-nftables/index.php/Configuring_chains).
- **Evidence classification:** verified upstream source capability ≠ installed BoxWeave native acceptance; a cited issue/workaround ≠ universal security claim; a hypothetical nft match expression ≠ a tested deployable ruleset.

**Bottom line:** the desired product contract is **default-isolated, exact-identity, arbitrary mutual Box peers**. No plain Linux Bridge setting natively supplies arbitrary pairwise authorization; a narrowly scoped nftables Bridge policy with lifetime-safe, Incus-assigned host veth names is a **credible research candidate**, not a proven B2 replacement. The first decisive gates are Incus ACL coexistence and reliable fail-closed behavior when the nft policy is missing.
