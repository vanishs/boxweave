# Incus integration and runtime decisions

## Upstream interfaces

Incus's stable foundation is its native management API. The Incus CLI and the official Go client use that API. **BoxWeave external management remains HTTP MCP only**; an internal Incus API or Unix-socket call is not a second BoxWeave public REST API.

**The owner selected Go + the official Incus Go client.** Use `github.com/lxc/incus/v7/client` and matching typed Incus API models directly behind a narrow domain adapter. Prefer local Unix socket access through a properly restricted local security boundary. Do not implement an independent generic Incus REST client or rely on the Incus CLI as the core controller. The Go SDK covers native Instance, Exec, Operation, proxy-device/network-forward and peer-isolation APIs, but each BoxWeave-needed feature still requires real integration verification.

## Trust and division of responsibility

Incus is a mature authoritative Linux-container runtime: **trust the official daemon and typed Go SDK as the underlying infrastructure**. BoxWeave does not need a second Incus implementation, custom equivalent transaction engine, routine upstream reliability audit, or broad compatibility layer for old Containarium HTTP bugs. The necessary tests concern **our** ownership decisions, SDK argument mapping, supported API extensions, privilege boundary, and observed user workflows (e.g. LAN port publishing, safe peer policy, guest Exec state). Report unsupported SDK/runtime capabilities accurately; do not infer a flaw in Incus from an unverified BoxWeave request, network route or test fixture.

## Runtime capabilities

- Check the actual Incus daemon version and `api_extensions` during startup and before capability-sensitive actions. Version strings alone do not guarantee a feature.
- Bind Boxes to native Incus instance UUID and verify on each privileged mutation; do not treat names, IPs or guessed profiles as identity.
- Use Incus Exec rather than SSH. Guest UID 0 by default with optional explicit UID/GID; typed argv and bounded guest cwd/env/timeout/output, without implicit shell interpolation. Guest root stays inside the owned unprivileged container, never host shell. Exec is async and Tracking-ID-correlated, nonzero exit is a completed result; `operation_cancel` needs independently tested native termination semantics.
- Native Incus Operation IDs are execution evidence but not substitutes for the BoxWeave journal.
- Handle Exec websocket streams, status/exit codes, output bounds, cancellation and lost connections without fabricated outcomes.
- Use native ETag for concurrent typed Box and owned network changes where available. Validate read-back of accepted writes.
- Avoid arbitrary Incus configuration maps in public requests; explicitly allowlist CPU, memory, disk, nesting and network settings. Deny unrestricted devices, raw config, host mounts, privileged containers and host socket exposure.
- No storage driver hard constraint. Determine whether a requested resize/storage operation is supported by the actual pool and return `UNSUPPORTED` or a native semantic error instead of attempting an unsafe workaround. **P0 refuses disk shrinking before any native dispatch**; growth is observed and post-verified.
- Scope native resources to the managed project/namespaces and preserve pre-existing host/project state.

## Local privilege

Unrestricted Incus admin socket access can confer broad host powers. Use restricted Incus identities where workable, and a narrow local privileged service with an allowlist for any essential setup actions. Treat this as a real OS/process privilege boundary, not merely JSON validation in an all-powerful public HTTP process.

## Selected Go and MCP integrations

The v1 implementation language is **Go**. The HTTP MCP server uses the official `github.com/modelcontextprotocol/go-sdk` library; the native Incus adapter uses the official `github.com/lxc/incus/v7/client` library. Both SDKs are linked into BoxWeave; neither is a separate service or a public REST interface. Compile against a reviewed compatible release and validate the actual daemon's `api_extensions`; never equate a successful compile with complete feature coverage. See [technology stack](TECH_STACK.md).

## Agentless containers

Incus Linux containers require no installed BoxWeave guest Agent, daemon or SSH service for lifecycle, native Exec or **P0 native guest file list/read plus `box_file_apply(create|replace|delete)`**. The exact SDK file API and guest filesystem safety, symlink/race/atomicity guarantees must be validated; reject unsupported safe operations rather than falling back to host filesystem access. This is different from VM-focused `incus-agent` requirements and Containarium's optional in-guest `agent-box` MCP workspace extension. See [guest management](GUEST_MANAGEMENT.md).

## Context, version and capability reporting

Every native adapter call receives Go `context.Context` carrying a safe Tracking ID/actor/owned-target reference and bounded deadline. Persist original native Operation ID where available; client disconnect never justifies auto-replay of a mutating native call. Include actual Incus daemon version and API extensions in the dependency ledger, and distinguish `running` from verified guest Exec readiness. Explicit local install/upgrade reconciles compatible managed Incus versions, with shared-host upgrade approval when required. See [Tracking/context](TRACKING_CONTEXT.md) and [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md).

## Compatibility and tests

Use official SDK typed models and API-extension negotiation; run native Incus integration tests for lifecycle, Exec, minimal peer isolation, native port forwarding, storage capability and multi-level nested setups. A 2xx/API success is not postcondition proof. Pin exact upstream versions and independently verify native state.
