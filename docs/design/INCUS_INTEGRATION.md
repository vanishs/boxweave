# Incus integration and runtime decisions

## Upstream interfaces

Incus's stable foundation is its native management API. The Incus CLI and the official Go client use that API. **BoxWeave external management remains HTTP MCP only**; an internal Incus API or Unix-socket call is not a second BoxWeave public REST API.

**The owner selected Go + the official Incus Go client.** Use `github.com/lxc/incus/v7/client` and matching typed Incus API models directly behind a narrow domain adapter. Incus Unix socket privileges vary by OS identity: root/`incus-admin` has full rights; a non-root user **only in `incus`** is natively confined to its per-user Project. `UseProject` is only a selector. Start with a dedicated `incus`-only service account and local-admin-provisioned Project restrictions; test native proxy/ACL/nesting and denial of foreign/admin calls. Restricted TLS identity is fallback and a helper last resort. Do not implement an independent generic Incus REST client or rely on the Incus CLI as the core controller. The Go SDK covers native Instance, Exec, Operation, proxy-device/network-forward and peer-isolation APIs, but each BoxWeave-needed feature still requires real integration verification.

## Trust and division of responsibility

Incus is a mature authoritative Linux-container runtime: **trust the official daemon and typed Go SDK as the underlying infrastructure**. BoxWeave does not need a second Incus implementation, custom equivalent transaction engine, routine upstream reliability audit, or broad compatibility layer for old Containarium HTTP bugs. The necessary tests concern **our** ownership decisions, SDK argument mapping, supported API extensions, privilege boundary, and observed user workflows (e.g. LAN port publishing, safe peer policy, guest Exec state). Report unsupported SDK/runtime capabilities accurately; do not infer a flaw in Incus from an unverified BoxWeave request, network route or test fixture.

## Runtime capabilities

- Check the actual Incus daemon version and `api_extensions` during startup and before capability-sensitive actions. Version strings alone do not guarantee a feature.
- Bind Boxes to native instance `Config["volatile.uuid"]` and verify the owned Project before privileged effects. The SDK's native instance methods are typically **name-addressed**, not UUID-addressed: resolve UUID to current project/name and compare the live `volatile.uuid` immediately before mutation. Do not grant authority on name/IP reuse or conflate `volatile.uuid.generation` with instance identity.
- Use Incus Exec rather than SSH. Guest UID 0 by default with optional explicit UID/GID; typed argv and bounded guest cwd/env/timeout/output, without implicit shell interpolation. Guest root stays inside the owned unprivileged container, never host shell. Exec is async and Tracking-ID-correlated, nonzero exit is a completed result; `operation_cancel` needs independently tested native termination semantics.
- Native Incus Operation IDs are execution evidence but not substitutes for the BoxWeave journal. `Operation.WaitContext(ctx)` can call native `Cancel()` when its context expires on the event-listener path: never use it as a read-only status poll. Prefer `GetOperationWait` or bounded `GetOperation` observation, reserving native cancellation for explicit `operation_cancel`.
- Handle Exec websocket streams, status/exit codes, output bounds, cancellation and lost connections without fabricated outcomes.
- Use native ETag for concurrent typed Box and owned network changes where available. Validate read-back of accepted writes.
- Avoid arbitrary Incus configuration maps in public requests; explicitly allowlist CPU, memory, disk, nesting and network settings. Deny unrestricted devices, raw config, host mounts, privileged containers and host socket exposure.
- No storage driver hard constraint. Determine whether a requested resize/storage operation is supported by the actual pool and return `UNSUPPORTED` or a native semantic error instead of attempting an unsafe workaround. **P0 refuses disk shrinking before any native dispatch**; growth is observed and post-verified.
- Scope native resources to the managed project/namespaces and preserve pre-existing host/project state.

## Native Project bootstrap is a prerequisite, not a repair after create

Per [Incus's official tagged Project docs](https://github.com/lxc/incus/blob/v7.6.0/doc/howto/projects_create.md), a newly isolated Project typically has `features.profiles=true` and its own `default` Profile **may not have a root-disk device**. The [Project explanation](https://github.com/lxc/incus/blob/v7.6.0/doc/explanation/projects.md) says `features.*` choices become locked when the Project contains instances. Therefore the host-local installer must preconfigure the **native confined service-user Project** before creating the first Box: intended `features.images/profiles/networks/storage`, approved restricted devices/permissions, the exact storage pool and effective root disk, usable managed NIC/network and ACL availability. Never silently import or edit the host `default` Profile or an unrelated network after a failed create.

The exact project feature/ACL/proxy rights of a non-root `incus`-only user remain **native-untested**. Local group confinement is source-confirmed but not a claim that every approved P0 device will be permitted; only consider restricted TLS or a privileged helper after proving a specific native capability gap. See [second-pass gaps](../research/SDK_GAP_CLOSURE_2026-11-07.md).

## Native preflight, effective configuration and safe updates

Use the smallest typed Incus adapter. Before dispatch: validate resource units and bounds (reject NaN/Inf, overflow, negative/unsupported sizes), actual API extensions, selected storage pool, guest image, intended network topology and exact owned target. A running Incus daemon alone does not mean an initialized usable storage pool/network exists. If space/capability is unknown or insufficient for a requested operation, report a truthful precondition/`UNSUPPORTED` result; do not add generalized capacity scheduling.

- **Create uses effective devices:** Verify the effective root disk pool even when disk size is omitted; Profile inheritance can otherwise silently substitute the default pool. Read native `ExpandedDevices` and instance-local devices to understand an inherited NIC; if an owned NIC must be shadowed to set an owned ACL, carry its other settings forward instead of replacing it with a partial device. Do not edit foreign Profiles or globally change bridge settings.
- **Update only named fields:** `box_update(cpu=...)` must not erase unchanged memory, disk, networking or security settings. Use native read -> typed merge -> ETag guarded write where supported -> complete effective readback, including untouched axes. No broad `raw.lxc`/device map from MCP.
- **Exact identity and honest source:** Resolve Incus native instance UUID/incarnation/revision and owned project at point of use; name/IP reuse never transfers authority. Distinguish what was requested from what Incus actually reports. An ETag or success ACK alone does not prove the BoxWeave postcondition.
- **Native transient errors:** Match supported typed SDK/HTTP semantics where possible. A bounded read-only recheck may resolve a transient state-read failure. A failed/uncertain mutating SDK call or Exec **must not** be automatically reissued from an upstream error substring.

These constraints come from [profile/NIC, root-pool, resize, input and Exec failure history](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md), not from adopting Containarium's generic Backend interface.

## Local privilege — upstream-first

The preferred runtime service is a dedicated non-root OS user belonging to `incus` but **not** `incus-admin`. Incus's documented per-user Project confinement is a native security boundary; the host-local installer may configure its `restricted=true` Project. `restricted.devices.proxy` and `restricted.containers.nesting` default to `block` and need explicit, reviewed allowances if proven necessary. A scoped Project is not by itself a grant to reach foreign Boxes, and BoxWeave still checks UUID ownership. Restricted TLS certificates are the next candidate if local confinement cannot support P0; do not create a separate privileged helper by default. [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/).

## Correct native API families and error semantics

- `GetInstancePortForwardConn` connects to a guest port; it **does not** install a persistent host listener. Use a named, owned instance `proxy` device via `UpdateInstance`.
- `CreateNetworkPeer` manages **network-to-network** peering, not directional Box-to-Box permission. Use NIC-attached `NetworkACL` operations for `box_peer_set`.
- Classify upstream HTTP errors with `api.StatusErrorMatch` / `api.StatusErrorCheck`, not English substrings or invented numeric error values.
- `InstancePut.Config` and `Devices` are whole native maps; preserve unrelated effective settings under ETag guarded updates.
- See [20-tool SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md).

## Selected Go and MCP integrations

The v1 implementation language is **Go**. The HTTP MCP server uses the official `github.com/modelcontextprotocol/go-sdk` library; the native Incus adapter uses the official `github.com/lxc/incus/v7/client` library. Both SDKs are linked into BoxWeave; neither is a separate service or a public REST interface. Compile against a reviewed compatible release and validate the actual daemon's `api_extensions`; never equate a successful compile with complete feature coverage. See [technology stack](TECH_STACK.md).

## Agentless containers

Incus Linux containers require no installed BoxWeave guest Agent, daemon or SSH service for lifecycle, native Exec or **P0 native guest file list/read plus `box_file_apply(create|replace|delete)`**. The exact SDK file API and guest filesystem safety, symlink/race/atomicity guarantees must be validated; reject unsupported safe operations rather than falling back to host filesystem access. This is different from VM-focused `incus-agent` requirements and Containarium's optional in-guest `agent-box` MCP workspace extension. See [guest management](GUEST_MANAGEMENT.md).

## Context, version and capability reporting

Every BoxWeave native adapter call accepts Go `context.Context` for safe Tracking ID and work budgets; however **many Incus `InstanceServer` methods do not accept a context argument themselves**. Native cancellation is only promised for tested SDK paths. Persist original native Operation ID where available; client disconnect never justifies auto-replay of a mutating native call. Include actual Incus daemon version and API extensions in the dependency ledger, and distinguish `running` from verified guest Exec readiness. Explicit local install/upgrade reconciles compatible managed Incus versions, with shared-host upgrade approval when required. See [Tracking/context](TRACKING_CONTEXT.md) and [dependency lifecycle](../process/DEPENDENCY_LIFECYCLE.md).

## Current upstream capability gates

Reviewed Incus v7.6.0 `GetInstanceFile`/`CreateInstanceFile`/`DeleteInstanceFile` provide native guest files, but **not a caller-hash/ETag atomic compare-and-swap** for conditional replace/delete. A pre-read SHA is a conflict hint, not atomic safety; preserve fail-closed behavior until verified.

For host-port forwarding, Incus instance `proxy` devices provide host `listen`/`connect`. Incus network forwards are distinct network-scoped IP-forward objects; **select proxy devices as the P0 implementation candidate** instead of silently supporting two different lifecycle models. On native bridge ACLs, network-level attachment does not isolate same-bridge peers; validate NIC-level rules and packet behavior. See [SDK fit review](../research/OFFICIAL_SDK_FIT_2026-11-07.md).

## Narrow SDK contract and postcondition checks

The two small [Incus MCP implementation case studies](../research/INCUS_MCP_IMPLEMENTATIONS_2026-11-07.md) reinforce three requirements **without changing the official Go SDK choice**:

- **Input presence is semantic:** distinguish an omitted field (preserve), an explicit clear (permitted only when a named BoxWeave typed field allows it), and a provided value (validate before dispatch). Go optional pointers/JSON `omitempty` alone must not silently turn explicit null into omission. A CPU-only update cannot erase memory or NIC settings.
- **Verification must examine values:** a successful typed SDK call, present config key, HTTP 2xx or completed Incus Operation does not establish the intended effective value or preservation of unrelated config. Read back expanded/effective device and resource state with original UUID/incarnation and compare intended values; missing proof remains honestly unresolved.
- **Contract drift is tested, not patched with a generic REST gateway:** build SDK method/typed fields/API-extension coverage tests for each v1 feature; include wrong/ignored fields and unsupported enums. In-memory MCP client + injected narrow Incus fake exercises schemas, permissions and error translation, but real native Incus post-read and negative tests remain the acceptance gate. Pin the actual SDK/runtime compatibility; upstream example versions are not substitutes.

## Compatibility and tests

Use official SDK typed models and API-extension negotiation; run native Incus integration tests for lifecycle, Exec, minimal peer isolation, native port forwarding, storage capability and multi-level nested setups. A 2xx/API success is not postcondition proof. Pin exact upstream versions and independently verify native state.
