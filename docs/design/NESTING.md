# Full nested runtime compatibility — P0

**Goal:** An ordinary Box can run a complete, independent BoxWeave installation with the same features as an installation on a physical host, as long as Incus and the Linux environment support them. This is **runtime compatibility**, not a feature for orchestrating nested BoxWeave controllers.

The principal motivations are:
- Develop, install, upgrade, test, and independently accept new BoxWeave versions **inside an existing BoxWeave-managed Box**, without changing the outer controller.
- Preserve a foundation for possible future environment subleasing: an operator may allocate one Box that happens to contain another full BoxWeave. Subleasing, cross-layer administration, tenant delegation, usage metering, or commercial billing are **not v1 scope**.

```text
Physical host: BoxWeave L0
  owns Box A (ordinary owned Box; nesting capability enabled)
    Box A runs BoxWeave L1, with its own Incus / SQLite / MCP
      owns Box B (ordinary L1-owned Box; nesting enabled)
        Box B runs BoxWeave L2
          owns Box C ...
```

## Local-only responsibilities

1. **Each layer manages only its own directly owned Boxes.** L0 manages Box A; L1 manages Box B; L2 manages Box C. No layer discovers, inventories, coordinates, upgrades, migrates, backs up, or manages the **inner BoxWeave controller** or any of its child Boxes as a special hierarchy.
2. **A nested installation is just software in a Box.** Its containing Box is an ordinary resource of the parent layer. The parent is not required to detect whether it runs BoxWeave, nor to inspect the inner operation database, Incus instances or Caddy state.
3. **Ordinary lifecycle rules apply.** Starting, stopping, updating, and deleting a Box that contains another BoxWeave have exactly the same parent-layer authorization, identity, confirmation and operation semantics as other owned Boxes. Deleting the containing Box may destroy everything stored inside it; the parent does not walk a descendant tree, apply special recursive deletion protection, or require an inner-controller handshake.
4. **Independent scope and identity.** Every BoxWeave has its own host-local Incus, SQLite, administrator credentials, Tracking ID namespace and MCP endpoint. Its host is the environment where it runs. The child does not receive its parent's credentials, native socket or privileged management access.
5. **Identical feature contract, no artificial depth limit.** The same program, public MCP tool schemas, local CLI and operations work at any supported nesting depth. Different native capabilities or resource limits may cause a precise `UNSUPPORTED` result; the product must not silently downgrade functionality.
6. **Ordinary network ownership.** Each layer manages only native TCP/UDP forwards and optional simple private Box-to-Box peer grants for its own Boxes. To reach an inner service, the operator separately configures layer-local native forwards as necessary; there is no Caddy or cross-layer delegation/publishing protocol.
7. **Supported nesting must actually work.** A parent can enable allowed nesting capabilities on a directly owned Box. `security.nesting=true` alone is not proof that an inner Incus can create a child. The guest-side installer must check its own user namespaces/idmaps, storage, networking and privileges, and report limitations without changing the parent.

## Nested networking and native NIC anti-spoof compatibility

Incus [bridged NIC security documentation](https://github.com/lxc/incus/blob/v7.6.0/doc/explanation/security.md) says native `security.mac_filtering` and `security.ipv4_filtering`/`security.ipv6_filtering` can prevent source-address spoofing, but also restrict inner bridged/macvlan devices that send traffic with a different MAC. This is an **upstream-defined behavior**, not a reason to disable native filtering across all Boxes.

Preferred **test candidate**, not yet an acceptance result: give the outer owned Box one anti-spoof-protected ordinary NIC; run the inner Incus's **own managed bridge + NAT** so its child traffic uses the containing Box's allowed address on the parent NIC. Validate inner guest egress, default isolation, native forwarding, IPv4/IPv6 behavior and no parent-host or sibling access. If the native kernel/nesting environment lacks this path, report precise `UNSUPPORTED` rather than silently enabling raw L2 bridging/macvlan, broadening parent NIC permissions, altering outer host firewall or requiring a dedicated parent agent.

The detailed native evidence and grouped probe are in [second-pass SDK review](../research/SDK_GAP_CLOSURE_2026-11-07.md) and [earlier Box probe](../verification/SDK_NATIVE_BOX_PROBE_2026-11-07.md).

## Local MCP access inside nested development Boxes

Every inner controller's MCP endpoint listens only at its own guest **`127.0.0.1`**; an outer Agent **cannot** call that endpoint directly through the inner Box's bridge/LAN IP, nor does an L0 port forward to an L1 guest IP magically expose L1 localhost MCP. For testing, an authorized outer `box_exec` may run an **ordinary temporary MCP client in the owned Dev Box**, which talks to that independently installed inner controller's localhost endpoint and retains original L1 operation receipts. The inner credentials are initialized through its own guest-local bootstrap; keep them in restricted guest-local storage and never print them in Exec argv, stdout or shared logs. This is normal guest software, **not a parent-to-child management API**.

L1 service publishing to the physical LAN similarly requires two **separate, explicit, owned native forwards**, one at L1 and another at L0, where supported. Testing inside the Dev Box is **not** proof of reachability from a different machine on L0's physical LAN. A candidate L1 may create an L2 nested installation for the full original bug reproduction, but unavailable native nesting capabilities must be reported honestly. Do not silently weaken isolation or grant the inner Incus access to L0's socket.

## S2.5 nested storage and actual L1 runtime compatibility — priority blocker

The S2 merged controller's `internal/native/status.go` currently rejects any selected pool driver other than `dir` with **verified ext4/XFS project quota**. `internal/native/storage_filesystem.go` checks effective mount information and the backing filesystem; a generic nested `dir` on the L1 container root is **not** sufficient evidence of quota enforcement. Even if `security.nesting=true`, an unprivileged container may not be able to create the required dedicated mount. The outer root-disk quota is not automatically proof that every **inner** L2 root/Exec-spool quota is enforced.

For BRIDGEHEAD, test actual unprivileged L1 on the selected physical topology: `doctor`/host_status, exact inner `dir` pool mount and project-quota evidence, original bound Guest Exec spool limit + true EDQUOT, inner Project restricted rights/root Profile/image alias, userns/idmap, managed bridge/NAT behind filtered outer NIC, and independently read-back L2 child lifecycle. A verified safe alternative storage backend may be designed only with real quota/retention/negative evidence and a full reacceptance of existing S2 safety gates; **never bypass a quota guard merely to mark nesting PASS**. If the native environment cannot satisfy it, mark `NESTED_STORAGE_BLOCKED`; an L0-only success cannot be relabeled as L0→L1→L2. [Pinned source/upstream audit](../research/S25_PREDEPLOY_REALITY_AUDIT_2026-11-11.md).

S2.5's physical L0 is **dedicated and never cohosts legacy `main`**. Its existing fixed single-controller service identity, localhost port and installation paths remain valid; no special coexistence or cross-layer parent management API is required.

## Earlier S2.5 BRIDGEHEAD gate versus full S4 nested parity

[BRIDGEHEAD](../development/phases/025_BRIDGEHEAD.md) is a deliberately limited **intermediate native development-base gate**: after all S2 acceptance, an explicitly authorized exact v1 L0 binary creates one L1 development Box with nesting; an independently installed, credential-isolated v1 L1 controller uses a **guest-local** authenticated client to create/Exec/delete one L2 native child Box. It requires independently witnessed private IPv4/IPv6 safe default, NAT/anti-spoof, exact UUID, SQLite/worker restart/UNKNOWN and operator recovery. The inner v1 endpoint remains loopback; no outer controller gets inner credentials or native socket. **Not** a full v1 L2 controller, not public/LAN peer/forward publishing, not S4's global native TLS Scriptlet change and **not** A-035/A-036 complete parity. [Exact BH matrix](../verification/BRIDGEHEAD_ACCEPTANCE.md).

If this minimal chain cannot work without relaxing the approved network/ownership boundary, classify NO_GO and continue S3 in an isolated lab rather than installing a weakened candidate on a production/shared host. S4 still verifies all native networking, new public tools, complete L0/L1/L2 controller and LAN acceptance on its exact binary, then S5 repeats release acceptance.

## Development and acceptance use

Use an isolated parent-owned Box as a complete nested Linux host for building, installing, upgrading and testing a candidate BoxWeave. The first-version L0 remains unchanged until an explicitly authorized local host upgrade; a candidate tested in L1 **does not** repair any earlier failed L0 operation. A failing inner controller must not be relied on to collect its own failure evidence; an authorized outer controller may inspect its **owned containing Box** using ordinary Box lifecycle/Exec tools. It does not get special access to the inner controller's managed resources.

Test equivalent BoxWeave functionality on L0, L1 and L2, and deeper where possible: create/start/async Exec/update/stop/delete, guest file list/read/create-or-safe-replace, Tracking ID and Go context propagation within **each** layer, simple peer interconnection, internet access, native TCP/UDP forwarding, durable operation/output recovery, managed dependency reconciliation, local upgrade, and authentication boundaries. Use ordinary operators and current layer credentials for each layer. Verify that a failure in one layer neither grants authority in another nor changes foreign resources.

The acceptance target is **same capabilities under supported environments**, not one controller managing the nested hierarchy and not a promise of unlimited physical kernel capacity.
