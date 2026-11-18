# S4-B2 Stage 2: host root all-Project IP collision fence

2026-11-17 PT. Status: S4_B2_PARTIAL; B3 not started.
Source baseline: 7fb555c0048d25f86920bc693734b00f51dcf8ec.

## Product security change

[Stage1 native run #44201216635](https://github.com/vanishs/boxweave/actions/runs/44201216635) proved IP-only conntrack delete can destroy two records with the same original source/destination IP pair but different ports. BoxWeave cannot infer Project ownership from conntrack IP alone.

Actual private root conntrack paths now call Incus GetNetworksAllProjects plus GetInstancesAllProjects BEFORE deletion, immediately before the kernel effect and AFTER its readback. The independent snapshot rejects foreign managed IPv4/IPv6 networks overlapping either endpoint address, foreign expanded NICs reusing A/B IPs, invalid/missing host-wide inventory, changed inventory, and missing/duplicate owner bridge. Every peer address must be within the exact root bridge's two configured subnets. A stable digest catches ordinary drift. Public root-only review also rejects obvious overlaps before the one-shot UNKNOWN journal can be written.

Go tests cover foreign v4/v6 collisions, broad route CIDR, duplicate foreign NIC IP, unknown or missing inventory, owner subnet errors and result order independence. Native CI checks a real all-project SDK snapshot marker before the original stopped/root conntrack fixture.

## Limits / next work

This fence checks Incus-managed network addresses and expanded foreign NIC static IPs. It is NOT comprehensive for unmanaged host veth, policy routing, host network namespaces, NAT or a host root racing the effect. No public root apply CLI or MCP Peer mutation has been enabled. Stage2 still must implement a verified dedicated-host nonoverlap boundary or isolated conntrack zone, the operator-installed controller-to-root operation-ID/peercred path, per-original journal recovery/re-arm, installed nft and live foreign-Project controls, and installed Scriptlet/old restricted-client parity. This gate is a necessary precaution, not B2 signoff.
