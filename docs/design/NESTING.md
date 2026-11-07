# Full nested runtime compatibility — P0

**Goal:** An ordinary Box can run a complete, independent BoxWeave installation with the same features as an installation on a physical host, as long as Incus and the Linux environment support them. This is **runtime compatibility**, not a feature for orchestrating nested BoxWeave controllers.

The principal motivations are:
- Develop, install, upgrade, test, and independently accept new BoxWeave versions **inside an existing BoxWeave-managed Box**, without changing the outer controller.
- Preserve a foundation for possible future environment subleasing: an operator may allocate one Box that happens to contain another full BoxWeave. Subleasing, cross-layer administration, tenant delegation, usage metering, or commercial billing are **not v1 scope**.

```text
Physical host: BoxWeave L0
  owns Box A (ordinary owned Box; nesting capability enabled)
    Box A runs BoxWeave L1, with its own Incus / Caddy / SQLite / MCP
      owns Box B (ordinary L1-owned Box; nesting enabled)
        Box B runs BoxWeave L2
          owns Box C ...
```

## Local-only responsibilities

1. **Each layer manages only its own directly owned Boxes.** L0 manages Box A; L1 manages Box B; L2 manages Box C. No layer discovers, inventories, coordinates, upgrades, migrates, backs up, or manages the **inner BoxWeave controller** or any of its child Boxes as a special hierarchy.
2. **A nested installation is just software in a Box.** Its containing Box is an ordinary resource of the parent layer. The parent is not required to detect whether it runs BoxWeave, nor to inspect the inner operation database, Incus instances or Caddy state.
3. **Ordinary lifecycle rules apply.** Starting, stopping, updating, and deleting a Box that contains another BoxWeave have exactly the same parent-layer authorization, identity, confirmation and operation semantics as other owned Boxes. Deleting the containing Box may destroy everything stored inside it; the parent does not walk a descendant tree, apply special recursive deletion protection, or require an inner-controller handshake.
4. **Independent scope and identity.** Every BoxWeave has its own host-local Incus, Caddy, SQLite, administrator credentials, Tracking ID namespace and MCP endpoint. Its host is the environment where it runs. The child does not receive its parent's credentials, native socket or privileged management access.
5. **Identical feature contract, no artificial depth limit.** The same program, public MCP tool schemas, local CLI and operations work at any supported nesting depth. Different native capabilities or resource limits may cause a precise `UNSUPPORTED` result; the product must not silently downgrade functionality.
6. **Ordinary network ownership.** Each layer may manage HTTP routes, forwards and ACLs only for its own Boxes. Connectivity across layers uses the normal, explicitly authorized routing/port-forward actions at each relevant layer, set up separately by the operator. No cross-layer delegation protocol, automatic parent Caddy changes or recursive publishing service is required.
7. **Supported nesting must actually work.** A parent can enable allowed nesting capabilities on a directly owned Box. `security.nesting=true` alone is not proof that an inner Incus can create a child. The guest-side installer must check its own user namespaces/idmaps, storage, networking and privileges, and report limitations without changing the parent.

## Development and acceptance use

Use an isolated parent-owned Box as a complete nested Linux host for building, installing, upgrading and testing a candidate BoxWeave. A failing inner controller must not be relied on to collect its own failure evidence; an authorized outer controller may inspect its **owned containing Box** using ordinary Box lifecycle/Exec tools. It does not get special access to the inner controller's managed resources.

Test equivalent BoxWeave functionality on L0, L1 and L2, and deeper where possible: create/start/async Exec/update/stop/delete, guest file list/read/safe write/patch, Tracking ID and Go context propagation within **each** layer, ACL, internet access, Caddy routes, forwarding, durable operation/output recovery, managed dependency reconciliation, local upgrade, and authentication boundaries. Use ordinary operators and current layer credentials for each layer. Verify that a failure in one layer neither grants authority in another nor changes foreign resources.

The acceptance target is **same capabilities under supported environments**, not one controller managing the nested hierarchy and not a promise of unlimited physical kernel capacity.
