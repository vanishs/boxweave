# Owner decision: defer real public IPv6 WAN, not IPv6 security

**Approved 2026-11-12.** This changes the timing of **genuine public IPv6 Internet connectivity** acceptance for S2.5 BRIDGEHEAD / early developer-host HOST_GO. It is not permission to bypass IPv6 security, to claim public IPv6 tested, or to start a HOST_GO round.

## Early S2.5 and Round 1: real packets on simulated IPv6 WAN

- Keep **real unproxied public IPv4 HTTPS** positive/negative evidence. A package relay alone is not enough.
- For IPv6, create a **separately routed isolated upstream/WAN segment** in authorized disposable L0 and nested L1 environments, with a real HTTPS/TLS responder, and send **actual guest packets**. This is not a mock, a success sentinel, a single bridge peer, merely a route listing, or a native ACL text readback.
- The **exact installed BoxWeave MCP client** must create and operate two owned sibling guests at each tested layer, with valid authorized identities, original idempotency keys/Operations and native UUID/incarnation readback. Prove on/off/on or off/on/off per-NIC egress, default-deny, IPv4 and IPv6 peer/foreign isolation, forged IPv4/IPv6 source and MAC drops, legitimate positive control, and no parent/foreign access.
- A **separate independent Verifier** must witness the routed HTTPS response, native ACL `used_by`, effective NIC, route/firewall/packet results, original Operation and safe cleanup. Require the *same pinned candidate binary* at I (installed MCP), N (native packet), V (independent verifier).
- Record two separate evidence fields: `SIMULATED_WAN_IPV6_I_N_V` and `PUBLIC_IPV6_REAL`. Missing actual IPv6 Internet uplink in a lab or dedicated host is recorded `PUBLIC_IPV6_DEFERRED_TO_S5`, not product failure, and **does not by itself block** a complete S2.5 Round 1/LAB_PASS or independently authorized early HOST_GO. Missing simulated IPv6 I+N+V, IPv6 safety, public IPv4 or actual Agent/Verifier remains a blocker.
- On a dedicated physical host, independently check IPv6 routes/deny and network safety. Absence of a public IPv6 uplink is disclosed as a capability limitation, not passed or silently fixed. No physical mutation without exact-host approval.

## Late real-world validation

- **S4** networking phase: attempt real public IPv6 when a safely authorized independently usable IPv6 egress environment exists. Recheck actual installed MCP/ACL/peer/forward and packets, keeping S4's other security and LAN requirements.
- **S5 full release gate, mandatory:** on the exact final installed candidate and genuine IPv6-public-capable native provider, independently verify **real routed public Internet IPv6 HTTPS** (not simulated), individual IPv6 network on/off/on, safe default deny, peer/foreign, IPv4/IPv6/MAC anti-spoof and native postconditions at L0/L1 as supported. If unavailable, record `PUBLIC_IPV6_UNVERIFIED` and **do not sign full IPv6-public capability or S5 full release PASS** absent a separate, explicit owner-approved release-contract change.
- **Do not retroactively mark historical failures or partial tests PASS.** Existing Round 1 remains BLOCKED until the revised *early* I+N+V contract passes. Keep original UNKNOWN, exact digest and independent native evidence.
- This is an **acceptance contract amendment only**: it authorizes no new test action, physical-host change, release, PR merge or next round.

See [BH-009/BH-017](../verification/BRIDGEHEAD_ACCEPTANCE.md), [three-round runbook](../development/HOST_GO_THREE_STAGE_RUNBOOK.md), [S4](../development/phases/04_NETWORK_NESTING.md), [S5](../development/phases/05_RELEASE_ACCEPTANCE.md).
