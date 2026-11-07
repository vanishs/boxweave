# Incus integration and runtime decisions

## Upstream interfaces

Incus's stable foundation is its native management API. The Incus CLI and the official Go client use that API. **BoxWeave external management remains HTTP MCP only**; an internal Incus API or Unix-socket call is not a second BoxWeave public REST API.

**The owner selected Go + the official Incus Go client.** Use `github.com/lxc/incus/v7/client` and matching typed Incus API models directly behind a narrow domain adapter. Prefer local Unix socket access through a properly restricted local security boundary. Do not implement an independent generic Incus REST client or rely on the Incus CLI as the core controller. The Go SDK covers native Instance, Exec, Operation, ACL and snapshot APIs, but each BoxWeave-needed feature still requires real integration verification.

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

## Selected Go and MCP integrations

The v1 implementation language is **Go**. The HTTP MCP server uses the official `github.com/modelcontextprotocol/go-sdk` library; the native Incus adapter uses the official `github.com/lxc/incus/v7/client` library. Both SDKs are linked into BoxWeave; neither is a separate service or a public REST interface. Compile against a reviewed compatible release and validate the actual daemon's `api_extensions`; never equate a successful compile with complete feature coverage. See [technology stack](TECH_STACK.md).

## Compatibility and tests

Use official SDK typed models and API-extension negotiation; run native Incus integration tests for lifecycle, Exec, ACL, forwarding, storage capability and multi-level nested setups. A 2xx/API success is not postcondition proof. Pin exact upstream versions and independently verify native state.
