# S4 networking knowledge base: current path, security lessons, and future options

> **Decision snapshot (2026-11-17/18 PT): continue the owner-selected, conventional Incus Bridge + NIC ACL + nftables implementation in [Draft PR #153](https://github.com/vanishs/boxweave/pull/153). It has a credible implementation path and substantial native evidence; this research is NOT a request to replace it.** At the time of this update, PR #153 is still Draft at `6c7998d33cc6375cc387e4427f90b7096d10b36c`; its latest independent B2 report remains `S4_B2_PARTIAL`, with final-HEAD acceptance pending. Older partial/pending documents describe historical checkpoints; consult latest committed source and original test receipts before inferring a current failure.
>
> This is a **documentation-only, reusable technical knowledge base** in [Draft PR #160](https://github.com/vanishs/boxweave/pull/160). It does not authorize any new public Tool, Incus fork, host permission, Scriptlet or nftables change, network migration, altered S4 gate, merge or release. [PR #155](https://github.com/vanishs/boxweave/pull/155) remains separate product-positioning research. No new alternative-backend native tests were performed for this document.

**Reading order:** current S4 decision → layer model and examples → actual B2 lessons/status → identity and upstream/community evidence → alternatives → future experiments and decision triggers. The objective is to make the discussion reusable **without confusing an interesting alternative with a defect in the already-selected approach**.

## 1. Current decision: finish S4, do not restart the network design

**What we now know:** an Incus-managed dedicated Linux Bridge, instance-NIC ACLs enforced through Incus's nftables backend, protected BoxWeave ownership/lease state, narrowly authorized Scriptlet and scoped root conntrack operations are a conventional, conservative design. They have a real upstream-supported implementation path. The long B2 development cycle is principally the cost of proving security under native partial failures, races and installed privilege boundaries, **not evidence that the basic model has failed**.

- **S4-A:** formally accepted previously. **S4-B2:** product integration and substantial native tests exist; the final independent signoff is **still pending** on the currently checked PR #153 HEAD. The earlier installed controller→root normal/lost-ACK, foreign-Project flow survival and Scriptlet compatibility results came from earlier immutable commits; a later third-sibling Project-wide operation barrier still needs exact-final-HEAD regression proof. [Independent B2 audit](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_FINAL_INDEPENDENT_AUDIT_2026-11-17.md).
- **S4-B3–B6:** directional peer Tool, instance-owned TCP/UDP forwarding, peer/forward-aware Box Delete and complete network integration/independent verification are future stage gates, not proof of a failed ACL architecture. **S4-C:** separate inspection/readiness work later. Deep recursive AppArmor/BPF nesting was deliberately deferred to late S5 rather than conflated with networking.
- **What stays:** the currently approved directional `box_peer_set` public contract, Incus ACL/Scriptlet/root safety model, source-exact acceptance requirements, and the prohibition on granting public arbitrary root/network policy editing.
- **What research can change later:** diagnosis of a newly proved blocker, a simpler enforcement implementation **demonstrably preserving or strengthening all mandatory guarantees**, or a separate owner-approved future feature. No premature Incus fork or speculative migration.

**The critical distinction:** pairing permissions by host interfaces chiefly improves the *IP-reuse/identity* dimension and may simplify dual-stack ACL rule compilation. It does **not** inherently solve distributed host/Incus/SQLite transactions, rule disappearance, established connections, routing bypass, foreign attachments, privilege boundaries, or lost ACK. Much of the existing B2 code and hard-earned native evidence would still be needed or re-proven.

## 2. Mental model: policy, enforcement, privilege, and ownership

Consider three Boxes, A, B and C, all on a BoxWeave-exclusive Incus Bridge. The intended private policy is `A→B allowed`, `B→A newly initiated denied`, `A→C denied`. Normal return packets to an A-initiated connection can be accepted through conntrack without authorizing a *new* B-initiated session.

| Layer / concept | What it really means | Example in BoxWeave |
| --- | --- | --- |
| **Incus Project / `project_name`** | Administrative namespace, instance/permission/quota scope; **not** automatic network isolation | Instances in `user-1001` are eligible for the project's restricted Incus client; a foreign instance moved into it is *not* automatically BoxWeave-owned |
| **BoxWeave Box identity / SQLite** | Durable control-plane ownership, UUID/incarnation, operation/lease records and grants | Native Incus UUID and current NIC/Project must independently agree before BoxWeave admits an operation |
| **ACL** | Access-control *policy abstraction*, not a competing packet-processing daemon | Incus NIC ACL says whether the currently verified A addresses can initiate traffic to B |
| **Incus ACL→nftables** | Incus translates its bridge-NIC ACL into Linux kernel filtering rules; the host-side NIC name can select which interface's ACL is evaluated | Source-specific/destination-specific rules are rendered on per-NIC bridge hooks |
| **Linux Bridge** | Layer-2 switching (forwarding frames through host veth ports); it does not know Box UUID | Forwards A's ethernet frames toward B when forwarding is permitted |
| **Scriptlet** | Incus's built-in programmable **API authorization** (Starlark-style policy), not a firewall and not an API that edits ACLs itself | A verified restricted TLS identity may `view/edit` exactly the installer-owned ACL, but must not create/delete foreign ACLs |
| **conntrack** | Linux kernel's connection-state table; stateful filtering may accept `ESTABLISHED` traffic before later ACL drops | When A→B is revoked, existing TCP/UDP traffic may persist unless its still-authorized packet path is interrupted/its scoped conntrack state is invalidated |
| **Root-side narrow helper** | Privileged executor for bounded host-native checks/conntrack effects, not an MCP root shell | A nonroot controller submits one durable original operation; root independently checks UUID/Project/ledger/kernel and journals actual outcome |

**Two concrete examples:**

1. **Grant:** the Agent invokes the authorized `box_peer_set(A,B,true)` intent → BoxWeave verifies owned UUIDs/Project, current leases and peer policy → the restricted Incus client requests an exact ACL update → **Scriptlet only authorizes the API request** → Incus programs nftables on the relevant host NICs → real packet tests and readback determine whether A can talk to B. Incus/SQLite/API success is not synonymous with proven kernel convergence.
2. **Revoke:** BoxWeave must stop **new** A→B flows *and* deal with **already established** TCP/UDP connections. A simple ACL DELETE/DROP can leave old `ESTABLISHED` flows alive; B2 demonstrated this with real dual-stack packets, then proved a bounded root conntrack cleanup mechanism. The latter is not replaced automatically by changing the peer identity from IP to veth.

### 2.1 Why Scriptlet caused so much work

A rule like `if app == "BoxWeave" then allow` is inadequate: an untrusted client can claim a name, and allowing all actions for one client is not a scoped ACL writer. Incus must independently authenticate the restricted TLS identity and the Scriptlet must check *who*, *which API verb*, *which exact ACL*, and prior rights of other clients.

The switch `authorization.client.tls-restricted=scriptlet` affects **all** restricted TLS clients on that Incus daemon. Prior project-based restrictions are not necessarily automatically reproduced by the script. Therefore installer activation, existing-client GET/PUT and denial parity, daemon restart, abort, rollback and journal recovery are product safety work. That explains repeated B2 Scriptlet testing; the Starlark branch itself is not the hard part. See [Incus authorization docs](https://linuxcontainers.org/incus/docs/main/authorization/) and [installed native parity evidence](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_FINAL_INDEPENDENT_AUDIT_2026-11-17.md).

### 2.2 Why Project membership is not Box ownership

A Project defines an **upper bound on accessible Incus resources**, not the set of Boxes that BoxWeave has claimed. Moving an existing instance into `project_name` and manually inserting a SQLite row would bypass original Create/admission, immutable UUID/incarnation binding, network isolation before start, lease claims, ACL attachment, operation evidence and safe Delete/recovery. It must **not** be treated as supported. A future **Import/Adopt** feature could deliberately verify and atomically establish all those invariants, but it is out of current v1 scope and would need separate authorization/design.

Likewise, two different Incus Projects connected to the same host Bridge are not automatically packet-isolated. The project boundary is primarily administrative; same-Bridge network isolation requires real filtering and attachment controls.

### 2.3 Which part of networking is IP-based?

**IP remains necessary for ordinary guest addressing, routing, DNS, NAT, host-port forwarding and kernel conntrack tuples.** It need not be the *permanent authorization identity*. BoxWeave already stores UUID/incarnation as the trusted authority and uses validated current IPv4/IPv6 as temporary ACL selectors. An interface-pair backend could remove the latter from the *peer authorization predicate*, but would not remove IP from the data path or from all recovery/conntrack mechanisms.

## 3. Product contract: default isolation and independent peer pairs

The motivating user requirement is deliberately small:

1. **Default isolation:** independently owned Boxes cannot directly communicate with each other unless authorized.
2. **On-demand mutual connectivity:** A↔B can be enabled and later removed, while both Boxes still have only an ordinary `eth0` and retain their separate Internet/host-port-forward behavior.
3. **Independent relationships:** A↔B and B↔C do **not** imply A↔C. A "peer pair" is a graph edge; a "peer group" is normally a clique, so these are distinct product models.
4. **Lifetime-safe identity:** an old authorization to B must not grant access to a newly created C when C inherits B's IPv4/IPv6 address, MAC or some reusable kernel number.
5. **No generic host authority for Agents:** BoxWeave operations remain scoped to verified owned UUIDs/Project, with bounded host-side effects and honest failure/UNKNOWN handling.

**Product intent is mutual communication, not directional policy as an end in itself.** The currently owner-approved permanent v1 API nevertheless specifies one **directional** `box_peer_set(source_box_id,target_box_id,allow)` primitive. Two directed permissions can express the desired mutual pair without adding a public Tool. Whether to expose a symmetric convenience action is an *unapproved* product/API decision, not implied by this research.

"Wrong authorization", "unauthorized cross-Box access" and "IP-reuse privilege inheritance" are not three unrelated user features. Unauthorized A→C is the safety failure; stale B→C address reuse is one concrete cause. Other distinct causes include incorrect broad rules, spoofed addresses, and partially applied policy updates. The fundamental invariant is **no new/old Box gains a pair it was not explicitly authorized to have**.

**Scope caveat:** "A↮C" describes direct network reachability, not a claim that a fully controlled Box B cannot intentionally relay data between A and C at the application layer. Preventing an authorized intermediary from acting as a proxy is a different threat model.

## 4. Actual S4-B2 lessons: failures, mitigations, and remaining signoff

**Status vocabulary matters:** `native demonstrated` means one or more named earlier Incus 7.6 fixtures recorded real host/guest evidence; `implemented/guarded` means the proposed protection is in PR #153 code; `pending` means that a **final immutable HEAD**, installed controller, independent verifier and negative packet/fault matrix are not yet accepted. These categories must not be merged into a misleading B2 PASS.

| Real problem or acceptance constraint | Evidence / response now present | What is not made obsolete by a veth-pair design |
| --- | --- | --- |
| **S1 private egress rejects conflict with A→B grants** | Real native tests caught IPv4/IPv6 failure; ACL and egress/internet policy were reconciled in the chosen constrained topology | Gateway/internet/private-destination semantic separation and native packet tests |
| **IPv6 NDP neighbor discovery and dynamic first-packet behavior** | Source-exact NDP/IPv6 fixtures found an acceptable dedicated-Bridge candidate; original failures and a first-packet anomaly were recorded, not hidden | ARP/NDP, IPv6 forwarding and default-deny policy ordering |
| **Old B IP reused by new C** | Real stale-grant exploit reproduced; native dual-stack lease/UUID/incarnation fence, revoke-before-release and quarantine checks added and exercised | Native lifetime binding still matters; interface names can also be reused unless prevented |
| **Foreign/unmanaged NIC spoofs approved A address** | A real foreign unfiltered NIC spoof reached B in early experiments; root global bridge/Project/port/ACL audits and refusal tests added | Ownership and attachment integrity must be continuously defended |
| **Partial ACL PUT updates some NIC nft rules while SDK/DB reports error** | Real **grant-side fail-open** reproduced in [#44034201810](https://github.com/vanishs/boxweave/actions/runs/44034201810); stop/quiesce, effective-kernel inspection and UNKNOWN containment added | Atomic nft batches are **not** SQLite+Incus+all-NIC atomic transactions; missing policy can still fail open |
| **Long-lived TCP/UDP survives ACL removal or DROP** | [Native conntrack study](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_ESTABLISHED_CONNTRACK_2026-11-17.md) proved real IPv4/IPv6 session persistence and subsequent bounded-cutoff mechanics | Stateful fast paths and kernel conntrack must still be measured on whichever hook executes the new policy |
| **IP-only conntrack deletion collateral damage** | Real two-flow same-IP/different-port test proved collateral deletion; root global IP/Project/route checks and foreign-flow survival tests added | Interface-based peer grants **do not** retroactively make conntrack a UUID-indexed table |
| **Unprivileged Controller needs restricted ACL and root operations** | Exact-ACL Scriptlet path plus operation-ID/SHA-only root Unix-socket service, `SO_PEERCRED`, protected SQLite and original journaling; prior installed native normal/lost-ACK results positive | A self-managed nft backend still needs bounded host privilege, so eliminating Scriptlet is not eliminating authorization work |
| **Global Scriptlet changes old clients** | Native old restricted TLS Project/Profile/Files/Exec/ACL permission parity, restart and rollback tested on disposable Incus hosts | Keeping Incus ACL retains this gate; replacing it trades the gate for independent root nft policy management |
| **Lost ACK, crash, power-loss surrogate, ambiguous partial effect** | Durable one-shot journals, replay refusal, explicit `UNKNOWN`, read-only recovery of proved-complete root journals, service restart and upgrade/rollback tested | Same distributed-effects and identity/authorization problem survives backend replacement |
| **Concurrent unrelated same-Project sibling while root fences all NICs** | Latest B2 independent audit identified and implemented Project-wide SQLite admission barrier, rejecting overlapping sibling operations and forged generic releases | Any all-sibling stop/start or wide native policy replacement still needs mutual exclusion |
| **Foreign Project packet survival, bad route/topology, host root races** | Native foreign established TCP/UDP survival, global Incus inventory, host routes and bridge port snapshots exist; out-of-band root mutation is not universally lockable | Routed bypass, unmanaged host state and operator trust boundary remain |
| **Final exact-source acceptance** | Prior installed controller→root, Scriptlet and S4-A regression results have exact original action receipts; latest source also adds barriers/workflow changes | Current HEAD still needs matching quality, installed native security, fault matrix, independent verifier and cleanup; no older run signs a newer SHA |

**Current gate:** [Final independent audit](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_FINAL_INDEPENDENT_AUDIT_2026-11-17.md) explicitly records `S4_B2_PARTIAL / B3_NOT_STARTED` and names the remaining same-final-HEAD checks, including latest sibling barrier, installed controller/root normal/lost ACK, native per-NIC nft + foreign packets, ACL/nft partial failure/UNKNOWN, Scriptlet parity or source-equivalence and independent cleanup. Do not turn individual green native runs into global B2 acceptance.

**Practical lesson:** the hardest and most reusable parts of S4 are **identity and scope proof, effective kernel evidence, privilege separation, failure-state conservatism and exact-source acceptance**. Replacing an IP selector alone does not retire those obligations.

## 5. Packet processing and Incus-managed network mechanisms

- **Incus:** owns container/VM lifecycle and NIC configuration, including a guest `eth0` and a host-side veth on bridged containers. It can read the instance UUID, its devices and the host-side interface. Incus supports a custom `host_name` for a bridged NIC and has managed network ACL, anti-spoof, port isolation and VLAN options.
- **Linux Bridge:** the kernel's ordinary Layer-2 switch, often provisioned as an Incus managed Bridge. It forwards based on MAC/FDB and VLAN membership and knows ingress/egress ports, **not Box UUIDs**.
- **OVS (Open vSwitch):** a separate software-switching project, an alternative switch implementation to Linux Bridge.
- **OVN (Open Virtual Network):** a separate logical-network control plane on OVS. Incus can drive OVN networks; boxes normally **do not** install an OVN driver internally. OVN requires OVS/OVN services and databases on the relevant host/controller infrastructure, increasing operational and version-maintenance cost.

An Incus-managed Linux Bridge and an Incus-managed OVN network are alternative network backends, not two simultaneous meanings of "Incus Bridge". When a Box is created on an OVN NIC, its normal guest view is still an ordinary NIC.

References: [Incus NIC configuration](https://linuxcontainers.org/incus/docs/main/reference/devices_nic/), [Incus Bridge networks](https://linuxcontainers.org/incus/docs/main/reference/network_bridge/), [Incus ACL behavior](https://linuxcontainers.org/incus/docs/main/howto/network_acls/), [Linux Bridge kernel reference](https://docs.kernel.org/networking/bridge.html).

## 6. Stable ownership versus reusable network identifiers

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

## 7. Alternative enforcement topologies (not migration decisions)

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

### 7.1 The VLAN question: reuse was not the real blocker

IEEE 802.1Q VLAN ID is 12 bits; ordinary IDs **1–4094** are usable, not 65,536. Reusing VLAN 100 **after all old members have been removed** is not inherently an IP-style identity vulnerability. The real risks are stale membership and cross-VLAN forwarding via a router, including the host.

Each ordinary access port can have only one default untagged VLAN/PVID. A guest NIC using only untagged traffic cannot simultaneously behave as two independent default access ports. Incus supports `vlan` and `vlan.tagged`, but tagged VLANs normally require the guest to handle tags and VLAN subinterfaces. Example: A↔B in VLAN 100 and B↔C in VLAN 200 would require B to address both contexts, rather than simply setting a shared pair permission. One-way tricks with multiple egress VLAN memberships do not provide normal symmetric bidirectional connectivity.

VLANs also affect network service design: DNS, DHCP, IPv4/IPv6 gateways, NAT, and inter-VLAN routing are separate concerns on an Incus-managed Bridge. Keeping the original Internet toggle and a simple eth0 may be substantially harder than creating an isolated VLAN.

### 7.2 Why port isolation cannot be "overridden by accept"

Linux Bridge port isolation prevents isolated ports from directly communicating with each other. It does **not** implement a pair whitelist; the kernel decision is not automatically bypassed by adding a separate nftables `accept` rule. An initial quarantine using port isolation cannot be assumed to coexist with dynamic pair release on the same path. The lifecycle would need a different, tested design.

### 7.3 OVN is technically attractive but operationally costly

OVN ACL expressions support logical `inport`/`outport` matching and Port Groups; Incus 7.6 generates logical port names from the network prefix, instance `volatile.uuid` and NIC name. This is closer to a true identity-derived logical port than IP or ifindex.

However, **Incus public ACL "source is another ACL group" is not automatically a logical-port match**: Incus 7.6's OVN ACL translator also generates IPv4/IPv6 address-set matches. Direct OVN Northbound rule management may be needed to guarantee port-only matching, which raises a shared-control-plane contract question: will Incus preserve independently written ACLs through network/NIC updates? That is not verified here.

Incus upstream [issue #3018](https://github.com/lxc/incus/issues/3018) documented stale OVN Port Group UUID membership; [PR #3140](https://github.com/lxc/incus/pull/3140) added cleanup in April 2026. A merged upstream fix does not prove that a given installed Incus 7.6 build includes it; verify the exact release and native behavior before depending on it.

Sources: [Incus 7.6 OVN port name implementation](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/driver_ovn.go), [Incus 7.6 OVN ACL translation](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/acl/acl_ovn.go), [OVN architecture/man pages](https://www.ovn.org/support/dist-docs/ovn-nb.5.html).

### 7.4 Why existing Incus ACL implementations gravitate toward IP

**This is not proof that port-based authorization is unsolvable.** It is an implementation and scope tradeoff: Incus ACL expressions cover physical/virtual networks, subnets, external services, IPv4/IPv6, and routing across network boundaries. An A's **current** source host veth is known at its first Bridge, but after L3 routing the packet's ingress and egress ports may be gateway/uplink ports, not the original two Box veths. Address sets are a practical common selector for broad policies.

Two implementation details prevent an easy analogy:

- The [Incus 7.6 Bridge nftables driver](https://github.com/lxc/incus/blob/v7.6.0/internal/server/firewall/drivers/drivers_nftables.go) already uses `iifname`/`oifname` to bind **the ACL being applied to the local NIC**, while its `source`/`destination` operands are translated into **IP/source or destination address matching**, not arbitrary foreign-veth matching.
- Incus's [OVN ACL translator](https://github.com/lxc/incus/blob/v7.6.0/internal/server/network/acl/acl_ovn.go) also translates a referenced ACL group into **IPv4/IPv6 address-set** comparisons. OVN itself supports logical `inport`/`outport` and Port Groups, but **Incus ACL group syntax is not equivalent to an end-to-end native port-identity policy**. On ordinary Linux Bridge, ACL groups/network selectors are not available in the same way. [Incus network ACL guide](https://linuxcontainers.org/incus/docs/main/howto/network_acls/).

For the narrower product requirement **same dedicated Bridge + one ordinary guest eth0 + selected directly communicating Box pairs**, the bridge forwarding hook sees both host ports and *can in principle* match them independently of IPv4/IPv6 addresses. But this restricted design must explicitly refuse or separately handle other traffic paths rather than silently promising the universal scope of IP ACLs.

A future upstream-friendly extension would be to resolve ACL group members into current host veth names for a *Bridge-specific* source/target selector, keeping Incus as policy owner. That is **not** the same as a one-line API flag: group membership changes, dual-ended forward hooks, same-NIC rule fan-out, cross-Project exposure, atomic updates, disappearance/restart and looped/routed paths require new Incus lifecycle code and native tests.

## 8. Open-source implementations and actual Incus community discussion

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

### 8.1 Closest concrete implementations

- **Incus itself:** the 7.6 nftables Bridge driver accepts the actual host-side `hostName` and generates per-instance/nic bridge ACL chains. Upstream [PR #1587](https://github.com/lxc/incus/pull/1587) merged Bridge-NIC ACL support in January 2025, showing that extending Incus's native policy backend has precedent. The earlier [PR #1225](https://github.com/lxc/incus/pull/1225) discussed the design. This **does not** establish a merged dual-veth selector.
- **Proxmox VE firewall:** [actual nftables template](https://github.com/proxmox/proxmox-firewall/blob/master/proxmox-firewall/resources/proxmox-firewall.nft) has `iifname vmap @vm-map-out`, `oifname vmap @vm-map-in` and per-guest chains. It also has `meta ibrname . meta obrname vmap @bridge-map`, but **that is a bridge-name pair**, not a Box-A-veth × Box-B-veth authorization map. These are useful proven primitives, not a copied BoxWeave peer implementation.
- **turnip:** [rootless Podman networking project](https://github.com/matthewmazzanti/turnip) uses a declarative directed flow matrix with veths and nftables, but a routed L3 `/32` fabric rather than a shared Bridge; its documented topology disables IPv6. It illustrates declarative peer graphs and native tests, not an exact substitute for S4.

### 8.2 What users asked the Incus community

- [Incus forum: questions about network ACLs](https://discuss.linuxcontainers.org/t/questions-about-network-acls/25976): maintainers discuss the historical OVN-first ACL model and converting Bridge ACLs into managed nftables. Editing Incus-owned nft chains behind its back is not a stable supported extension interface.
- [Incus forum: exact same-Bridge subnet/container isolation](https://discuss.linuxcontainers.org/t/acls-for-very-specific-subnet-isolation/22352) and [older Bridge-container isolation request](https://discuss.linuxcontainers.org/t/bridge-acls-between-containers-impossible/19115): motivates finer host/guest isolation without deploying OVN or multiplying Bridges, but is **not** evidence of a released port-identity selector.
- [incus-compose issue #98](https://github.com/lxc/incus-compose/issues/98): describes the difficulty of rendering Bridge network relationships as CIDRs when group/network selector semantics are not available there; stale IP rules after a recreate are an explicit design problem.
- [Incus #4008](https://github.com/lxc/incus/issues/4008): reported a **fail-open bridge NIC ACL update** failure. Incus 7.6's bridged NIC and nftables paths were revised to preserve/replace existing filters more safely, but the BoxWeave **grant-side partially applied, multiple-NIC** failure is a separate acceptance concern, not proved closed just because the issue was fixed.

**Research verdict:** these discussions prove the requirement is real and upstream has relevant mechanisms. They do **not** establish a community-approved, ready-to-use `source_instancenic`/`destination_instancenic` Bridge ACL API. No upstream Incus fork/patch or alternate policy writer is approved by this document.

## 9. Optional future experiment: Incus-fixed host veth identity + nftables

**Status: candidate only. Not an approved replacement for B2.** This is one plausible *low-additional-host-services* avenue under the owner's constraints, to be considered **after S4 acceptance** or upon a separately proven blocking defect:

- Keep one dedicated Incus Linux Bridge (illustratively `bwbr0`) and each Box's sole ordinary `eth0`. No OVN/OVS daemon, eBPF program or guest VLAN configuration.
- Use Incus bridged NIC `host_name` to allocate a controlled host veth name per **UUID + creation incarnation**. Keep authoritative identity and peer graph in protected SQLite; never use IP, MAC, a Linux ifindex or Box name as the peer authorization subject.
- Use an **owned** nftables `bridge forward` policy to match the **actual incoming and outgoing host veth names** as an ordered pair. A↔B generates A→B and B→A entries; A↔B, B↔C does *not* add A↔C.
- Continue preserving Incus MAC/IPv4/IPv6 spoof filters and a tight Project/Bridge attachment boundary; don't accidentally treat an address as security identity simply because it still provides routing and DHCP.

### 9.1 Identity allocator and lifetime contract

1. Reserve a short, host-wide **BoxWeave-only** veth namespace (Linux interface names have a short length limit; full UUIDs will not fit). Use a protected, non-reused allocation sequence/namespace and retained tombstones or equivalent; do not recycle names on ordinary deletion. Detect collisions with *all* existing host links and currently attached Incus devices, including foreign roots.
2. On every start/restart/recovery, compare the exact Incus instance UUID, NIC device, assigned `host_name`, native host link and saved incarnation; an unexpected name/UUID/link association must fail closed. Never dynamically substitute an arbitrary current name for a previously approved UUID.
3. Treat DB restore, duplicated installs, rollback, counter exhaustion and operator-created same-name links as **explicit collisions**. A monotonic allocator by itself is not proof if a stale backup is restored; the host's current state and old policy must be reconciled before any Box is released.
4. On B stop/delete, retire B's *runtime attachment* and withhold pair permissions to stale endpoints, even though the durable grant intent may need an original-operation receipt. A new C never adopts B's UUID/incarnation or reserved veth name.
5. The host is a trusted operator boundary: a host root user can create/rename links and bypass policy. The design does not claim cryptographic identity enforcement against an untrusted host admin.

This addresses the **specific** risk "C receives B's old IP" provided name assignment and attachment checks are actually enforced; it does not claim that names are intrinsically non-reusable.

### 9.2 Conceptual packet policy, not ready-to-run nft code

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

### 9.3 Five integration traps that can invalidate the simplification

1. **Existing Incus NIC ACL conflict.** The current [`internal/native/box.go`](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/internal/native/box.go) requires one NIC, a bound ACL, default ingress `reject`, MAC/IPv4/IPv6 anti-spoof, and default egress `reject` for `internet=false`. An nft `accept` in one base chain cannot override a later Incus ACL `drop` in another chain. Either safely redesign the static Incus baseline around **one authoritative peer-enforcement point**, or retain two policy engines (which may be *more* complex than B2). The new scheme must remain deny-by-default when any component is missing.
2. **Host-routing bypass.** `bridge forward` applies to bridged switching; Box→host gateway→Box or other L3 hairpin paths can traverse `inet`/`ip`/`ip6` forwarding hooks instead. A router/port forward may also involve a different path. Exact negative packet tests must cover both families and protocol directions without breaking legitimate internet/DNS/DHCP. IP can be used for *routing checks*, not as the permanent identity authorizing a peer.
3. **Revocation and stateful fast paths.** Ordinary `bridge forward` checking every packet could stop both new and established TCP/UDP data immediately without global conntrack deletion, *if* no earlier stateful verdict or flowtable/offload bypass exists and rules truly see all relevant packets. This is a **test hypothesis**, not an accepted property; inspect flowtable/offload, packet counters and old sessions before removing existing B2 conntrack safeguards.
4. **Cross-system atomicity.** nftables supports atomic rule-set replacement (`nft -f`) and set updates, but a transaction is **not atomic across SQLite + Incus NIC lifecycle + nft + kernel packets**. Restart, missing replies, stale ETag/instance state and race with Box deletion still need original IDs, serialization, readback and `UNKNOWN` handling.
5. **Privilege and foreign scope.** Programming nftables generally needs privileged host network authority. It must be an exact bounded, separately approved helper/action, not arbitrary root shell exposed to an Agent. The policy must not match unrelated Bridges, foreign Projects, host routing traffic or Incus's tables. Restrict attach/use of the dedicated managed Bridge to verified owned Boxes.

References: [Incus `host_name` and bridged NIC](https://linuxcontainers.org/incus/docs/main/reference/devices_nic/), [Incus 7.6 `nic_bridged.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/device/nic_bridged.go), [nftables Bridge filtering](https://wiki.nftables.org/wiki-nftables/index.php/Bridge_filtering), [interface-name matching](https://wiki.nftables.org/wiki-nftables/index.php/Matching_packet_metainformation), [concatenated keys](https://wiki.nftables.org/wiki-nftables/index.php/Concatenations), [atomic rules](https://wiki.nftables.org/wiki-nftables/index.php/Atomic_rule_replacement), [accept is not final in multiple base chains](https://wiki.nftables.org/wiki-nftables/index.php/Configuring_chains), [kernel flowtable/offload](https://www.kernel.org/doc/html/latest/networking/nf_flowtable.html).

### 9.4 Prepared native experiment matrix (NOT EXECUTED)

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

### 9.5 Decision rule before altering PR #153

A migration proposal should provide actual measurements comparing current B2 with this candidate: number of privileged components, mutable native objects, recovery states, Box guest requirements, failure-injection outcomes, host install/upgrade burden and verified packet isolation. It should **not** assume "fewer lines of code" equals greater safety.

**Research ordering, not approval:** if backend reassessment becomes warranted, test **Linux Bridge + controlled `host_name` + nftables** first, because it keeps Incus/Linux Bridge, one ordinary `eth0` per Box and no OVN/eBPF services; retain **OVN logical ports** as a technically sound but more operationally expensive fallback. VLAN/independent Bridges remain useful for simpler group semantics, not arbitrary overlapping pair permissions with the stated guest-NIC constraint. Existing S4-B2 protections and acceptance requirements stay in force until an owner-approved design switch is backed by real native evidence.

## 10. When to revisit, and how to avoid losing the knowledge

**Default is no redesign.** First finish B2's **same-final-HEAD** independent signoff, then B3–B6 using the authorized S4 plan. Only open a separate architecture decision after one of these triggers:

1. A **reproducible, unavoidable P0** in the chosen Incus ACL path despite a bounded safe workaround (for example an uncontainable fail-open update or irreconcilable global Scriptlet compatibility).
2. A future supported requirement (for example more general identity-based peer graphs, migration across hosts or new host network topologies) makes current IP/lease fencing materially inadequate.
3. Independently tested evidence shows an alternative **removes enough privileged services, native mutation fan-out, permanent reconciliation states or host burden** to outweigh rewriting already validated S4 code. Measure both source complexity and failure behavior; don't count only lines of code.

Before proposing a backend change, produce a **side-by-side proof** on the same supported Incus/kernel stack: A→B/B→A/C denied, v4/v6 and NDP, new/established TCP+UDP revoke, stale IP/name/UUID collision, foreign Project attachment and spoof refusal, host-routing/hairpin and port-forward, nft-policy deletion/missing daemon, concurrent sibling operations, lost ACK/rollback, installed authority and cleanup. Any missing default-deny witness means `BLOCKED`, not a hidden assumption.

### Problem → reusable knowledge index

| Future question | Reusable evidence / starting point |
| --- | --- |
| Could a Box inherit permission when an address is recycled? | `6` identity table, `4` B1/B2 reuse experiment and current SQLite lease/UUID fencing; `9.1` host-veth-name allocator alternative |
| Why can a network rule update succeed in the DB but fail in packets? | `4` partial native ACL grant fan-out; `9.3` atomicity and missing-policy fail-open |
| Why did an established TCP connection survive a revoke? | `2` conntrack explanation; `4` real dual-stack TCP/UDP revocation results; `9.4` fast-path experiment |
| Why all the Scriptlet tests if a permission script looks simple? | `2.1` global restricted-client behavior and old-client parity; `4` prior installed acceptance |
| Can an external Incus instance be "adopted" by putting it in `project_name`? | `2.2` Project vs BoxWeave ownership; no manual SQLite import |
| Can we skip OVN and still authorize arbitrary A↔B and B↔C? | `3` graph model; `7` VLAN/port-isolation limitations; `9` direct host-veth policy hypothesis |
| Is there already an Incus/Proxmox implementation to copy? | `8` original upstream code and community requests; don't equate per-interface dispatch with Box-pair policy |

**Provenance and reading discipline:** [product scope](../decisions/PRODUCT_BOUNDARY.md), [selected current network design](../design/NETWORK.md), [sole S4 implementation PR #153](https://github.com/vanishs/boxweave/pull/153), [source-exact B2 audit](https://github.com/vanishs/boxweave/blob/feat/v1-s4-runtime-integration-20261116/docs/verification/S4_B2_FINAL_INDEPENDENT_AUDIT_2026-11-17.md), separate [PR #155](https://github.com/vanishs/boxweave/pull/155); native mechanism sources in `5–9`. A vendor source proves its own code path, not acceptance of BoxWeave's future design.

**Bottom line:** the current S4 design is a credible conventional path and has already solved or guarded many real challenges. Keep the knowledge from alternatives available, but do not turn a narrower IP-reuse improvement into an unjustified wholesale rewrite. The next concrete milestone is final B2 independent acceptance, not an experimental replacement.
