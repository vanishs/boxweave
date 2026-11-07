# Incus integration and runtime decisions

## Upstream interfaces

Incus's stable foundation is its native management API. The Incus CLI and the official Go client use that API. **BoxWeave external management remains HTTP MCP only**; an internal Incus API or Unix-socket call is not a second BoxWeave public REST API.

Prefer the official Go Incus client if the owner accepts Go. It provides native Instances, Exec, Operation, network ACL and file/snapshot APIs. Python wrappers (including pyincusd) are community-maintained and may not abstract websocket Exec completely. The project should not write a second generic Incus REST implementation without a demonstrable need.

## Runtime capabilities

- Check the actual Incus daemon version and `api_extensions` during startup and before capability-sensitive actions. Version strings alone do not guarantee a feature.
- Bind Boxes to native Incus instance UUID and verify on each privileged mutation; do not treat names, IPs or guessed profiles as identity.
- Use Incus Exec rather than SSH. Execute in a non-root guest user by default where feasible; escalate only through explicitly authorized guest policy, never host shell pass-through.
- Native Incus Operation IDs are execution evidence but not substitutes for the BoxWeave journal.
- Handle Exec websocket streams, status/exit codes, output bounds, cancellation and lost connections without fabricated outcomes.
- Use native ETag for concurrent updates where available. Validate read-back of accepted writes.
- Avoid arbitrary Incus configuration maps in public requests; explicitly allowlist CPU, memory, disk, nesting and network settings. Deny unrestricted devices, raw config, host mounts, privileged containers and host socket exposure.
- No storage driver hard constraint. Determine whether a requested resize/snapshot/storage operation is supported by the actual pool and return `UNSUPPORTED` or a native semantic error instead of attempting an unsafe workaround.
- Scope native resources to the managed project/namespaces and preserve pre-existing host/project state.

## Local privilege

Unrestricted Incus admin socket access can confer broad host powers. Use restricted Incus identities where workable, and a narrow local privileged service with an allowlist for any essential setup actions. Treat this as a real OS/process privilege boundary, not merely JSON validation in an all-powerful public HTTP process.

## Implementation-language decision still open

The requester initially chose Python, later requested a Go/Python comparison and was advised to prefer Go because Incus officially maintains Go client code. This is a **recommendation**, not an explicit replacement decision. Record the final runtime language and MCP SDK selection before source implementation begins. The v1 specification itself is language-neutral.

## Compatibility and tests

Use official SDK typed models and API-extension negotiation; run native Incus integration tests for lifecycle, Exec, ACL, forwarding, storage capability and multi-level nested setups. A 2xx/API success is not postcondition proof. Pin exact upstream versions and independently verify native state.
