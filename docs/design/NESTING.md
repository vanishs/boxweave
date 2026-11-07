# Recursive self-hosting and parity — P0

A nested BoxWeave is **not** a special or reduced mode. Every layer independently runs the same application binaries, management MCP contract, installation/upgrade behavior, local operation store, Caddy ownership boundary and native Incus management integration.

```text
Physical Linux host
  L0 BoxWeave + Incus + Caddy + SQLite
    `-- Box A (owned by L0, nesting permitted)
          L1 BoxWeave + Incus + Caddy + SQLite
            `-- Box B (owned by L1, nesting permitted)
                  L2 BoxWeave + Incus + Caddy + SQLite
                    `-- ...
```

## Invariants

1. **No BoxWeave-imposed depth limit.** Real limits are dictated by kernel namespaces, user mappings, storage/network drivers, available resources and upstream Incus support.
2. **Equivalent functionality.** The same MCP tool names, schemas, error semantics, installation and upgrade procedures, operation durability, route/forward/ACL policy and security invariants apply at every level. Features unsupported by the actual environment report a precise capability condition rather than silently disappearing.
3. **Each level owns only its descendants.** Parent and child have distinct administrator credentials, instance UUID namespaces/ownership records, SQLite DBs and Caddy configurations. Parent and sibling resources are not discoverable or mutable through child MCP.
4. **Nesting must be provisioned and verified safely.** Parent-side allowlisted creation/update can request nested Incus capability. The inner standard installer must bootstrap its own Incus/Caddy/controller where allowed and report missing prerequisites; it must never silently mutate the parent. `security.nesting=true` alone does not prove nested Incus works; check actual user namespaces/idmaps, kernel policy, network and storage without indiscriminate privilege escalation.
5. **Host means this layer's runtime environment.** Child-local installation CLI can maintain services inside its containing Box, never on the parent's physical host. No host socket mounts or parent credential transfer.
6. **Public ingress is delegated, not stolen.** Inner Caddy/routes/forwards work within connectivity assigned by the parent. End-to-end public reachability across all layers requires explicit parent-owned port or route delegation; a child cannot edit parent Caddy or firewall to achieve it. A missing parent grant is a capability/access boundary, not permission to bypass it.
7. **Resources are bounded upstream.** Inner capacity and network reachability are constrained by the resources its parent has assigned. Failure to provision deeper Boxes due to host resource or upstream limitations is reported, never hidden or made into a successful status.
8. **Unknown effects survive parent resets.** Each layer reconciles only its own original operations; a parent stopping/starting its Box must not silently erase or replay a child action.

## Test matrix

At minimum, test L0, L1 and L2 with the same source/build and full P0 tool catalog, plus L3 for critical lifecycle and status when Incus and resources permit. Include nested guest create -> start -> Exec with independently recorded exit -> ACL allow/deny -> Caddy route -> TCP/UDP forward -> stop -> restart -> delete, subject to safe owned targets.

Test a middle-layer restart and crash, parent/child credential isolation, route delegation refusal, native Incus UUID binding, and no modification of parent/sibling resources. Record `UNSUPPORTED` with the exact upstream preflight for any blocked deeper level; never call it PASS.

This requirement does **not** guarantee that all physical hosts can support unlimited nesting.
