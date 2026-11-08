# v1 open technical risks and explicit exclusions

**Status:** design-only; these are **validation tasks**, not permission to expand the product. Current 20-tool [scope](../product/SCOPE.md) and [permanent boundary](../decisions/PRODUCT_BOUNDARY.md) supersede the previous broad Caddy/ACL/RBAC/P1 risk register.

| ID | State | Remaining investigation or scoped limitation |
|---|---|---|
| O-001 | **DECIDED** | Go + official MCP Go/Incus Go SDKs; pin actual supported versions/API Extensions through build/native tests. |
| O-002 | **VALIDATE** | Exact Incus UUID/instance incarnation behavior on create, clone, delete/recreate; ownership cannot rely on name/IP. |
| O-003 | **VALIDATE** | Bridge/OVN NIC actual same-bridge IPv4/IPv6 deny and per-pair directional native enforcement, including spoofing and lifecycle/IP reuse. No general ACL editor. |
| O-004 | **VALIDATE** | Restricted Incus credentials/project and setup privilege boundary; no root-equivalent native admin socket in network-facing MCP runtime. |
| O-005 | **VALIDATE** | Prefer **instance-bound native proxy device** for TCP/UDP P0 host listener. Prove bind/collision, supported bridge/OVN, hotplug/restart and nesting; an independent network-forward backend is **not** a P0 fallback. No custom proxy/Caddy. |
| O-006 | **VALIDATE** | Native IPv6 safe owned disable or fail-closed refusal where private isolation cannot be proven; no global host/foreign modifications. |
| O-007 | **VALIDATE** | Full inside-Box install/nested Incus user namespaces/idmaps/storage/bridge permissions through L1/L2 and parity under actual resources. |
| O-008 | **DESIGN** | Minimal SQLite durable intent/context/Tracking journal, exact idempotency and UNKNOWN reconciliation after native result disappears. No tracing backend platform. |
| O-009 | **DESIGN** | Local bootstrap, `admin`/`viewer` token issuance/revocation and role checks; viewer reads are potentially sensitive. |
| O-010 | **VALIDATE** | Safe guest file create/conditional replace/delete via native Incus files, symlink/hardlink/concurrent writer attacks and honest atomicity limits. No guest patch engine. |
| O-011 | **VALIDATE** | Native Guest Exec long-running output/exit and attempts to cancel active process, including disconnected client, worker restart and UNKNOWN state. |
| O-012 | **VALIDATE** | Operation output retention bounds, `operation_get(wait_ms)` concurrency, request/Tracking ID propagation and safe redacted inline diagnostic summary. |
| O-013 | **VALIDATE** | Existing Incus daemon/version detection, compatible upgrade when explicitly authorized, shared consumer protection, no foreign adoption, native postflight. |
| O-014 | **VALIDATE** | Ubuntu LTS guest/host first; any new Linux guest image must pass actual Exec/file/network/nesting support matrix before advertised. |
| O-015 | **VALIDATE** | Operator-managed Cloudflare Tunnel/reverse proxy compatibility with loopback Streamable HTTP MCP and external native forwarded guest services. BoxWeave does not manage proxy. |
| O-016 | **DESIGN** | Durable owned port/peer identity after daemon restart, native IP change, explicit public bind and readback; never reattach a stale claim to a recreated Box. |
| O-017 | **RESOLVED — OWNER CHOSE A** | Existing Incus adoption means reuse, compatibility management, and explicitly authorized daemon upgrades **only**. Existing/foreign Incus instances are never automatically imported, reassigned or controlled by BoxWeave. No further product decision is needed. |
| O-018 | **VALIDATE** | Exact instance proxy-device removal and peer cleanup before/with Box delete, correct resource locks/SQLite tombstone after partial failure. A separate network-forward backend and orphan lifecycle apply **only if explicitly introduced later**. |
| O-019 | **VALIDATE** | Native host LAN binding, firewall/routing, guest listener and independent same-LAN probe across both physical L0 and explicitly chained nested L0/L1 forwarding. No arbitrary host firewall configuration. |
| O-020 | **DESIGN / VERIFY** | Exact safe `operation_get` diagnostic envelope for partial deletion: phase, native error category, observed Box/forward states, missing evidence, build SHA and authorized local escalation without an MCP host-log tool. |
| O-021 | **VERIFY** | Inner loopback MCP via guest-local test client, secure bootstrap/secret redaction and independent L1/L2 test environments; verify parent cannot directly address inner `127.0.0.1` across the bridge. |
| O-022 | **RELEASE GATE** | Trustworthy candidate-build provenance, binary SHA-bound independent native/nested regression and verified CI release gating before immutable auto-publication. A green unit suite or fabricated acceptance report does not satisfy the gate. |
| O-023 | **VALIDATE** | Native guest service persistence using guest OS facilities and app listen address; do not add a BoxWeave guest process supervisor to satisfy the Hello World scenario. |
| O-024 | **CONDITIONAL** | Orphan **network-forward object** cleanup is applicable only if a future explicitly chosen network-forward backend is implemented. For P0 instance-owned proxy devices, instead verify no stale BoxWeave claim remains after instance deletion and no foreign device is touched. |

## Current standalone native evidence and blocker (2026-11-08)

[Minimal batched Incus 7.6 experiment](SDK_MINIMAL_BATCH_2026-11-08.md): official Go SDKs compile; standalone MCP per-Tool token identity/admin-viewer/revocation test PASSED; restricted `incus`-only user Project, denied foreign Project settings/admin Unix socket and native SDK scoped `UseProject` + typed 404 PASSED **for these narrow checks**. This does **not** close broad O-003/O-004/O-010/O-011 or certify a v1 installation.

One nested `incus init` then failed with a native event-listener timeout and Cowsql leader loss. The daemon process remained active, but database queries failed and an owned instance-storage symlink remained. Original native operation was reported failed, yet exact native resource post-state is **UNKNOWN**. Treat this as an O-007/O-008 test-environment blocker; preserve original evidence, do not repeat create or destructively clean unknown state, and do not call the error an Incus v7.6 SDK defect without independent non-nested reproduction.

## New official SDK capability checks

The [2026-11-07 Incus/MCP SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md) confirmed limitations that cannot be resolved by simply writing more BoxWeave code:

- **S01 P0:** Incus `Operation.WaitContext` may invoke native cancel when its context expires. Use `GetOperationWait`/`GetOperation` for observational waits and explicitly test non-cancelation.
- **S02 P0 corrected:** root/`incus-admin` has full local Incus authority; a non-root `incus`-only user has native per-user Project confinement. `UseProject` is not a credential. Test denied foreign/admin calls, project `restricted.*` allowances for proxy and nesting, and NIC ACL behavior; try restricted TLS before any helper.
- **S03 P0:** Instance methods are often name-addressed. Resolve stored `volatile.uuid` to live owned project/name and recheck UUID before effects.
- **S04 P0:** Incus guest file API has no atomic hash/ETag conditional write/delete. Strong `box_file_apply` modes are blocked until a safe native-compatible implementation is proven; return UNSUPPORTED instead of claiming atomicity.
- **S05/S06 P0:** Prefer instance proxy-device host listener and one tested NIC ACL topology; same-bridge packet enforcement needs proof.
- **S09/S10/S11 P1:** MCP SDK v1.8 handles modern Stateless HTTP and typed tools; explicit external HTTP auth/Origin protections and real ChatGPT/legacy compatibility tests are still necessary.

These are **SDK contract alignment** checks, not defects attributed to the official upstreams. No v1 real-environment acceptance has been run by this document change.

## Full native SDK coverage and newly discovered checks

See [20/20 tool-to-SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md). Verify SDK HTTP `auth.RequireBearerToken` expiration/revocation, localhost Host guard, default 4 MiB request cap, explicit Origin proxy compatibility, `api.StatusErrorMatch` semantics and Incus `ExecInstance` RecordOutput timing. Incus `GetInstancePortForwardConn` and `CreateNetworkPeer` are **not** implementations for the BoxWeave host forward / Box peer tools. These are validation tasks only, **not** live test PASS.

## Source-level follow-up: implement by SDK semantics

[Second-pass native source research](../research/SDK_GAP_CLOSURE_2026-11-07.md) tightens the already-listed validation risks, without opening a general platform backlog:

- **P0 / O-009:** In typed MCP Tool handlers, authenticate **`req.Extra.TokenInfo`** on every call. `auth.TokenInfoFromContext` is only the HTTP middleware accessor; prove no actor/role cross-request leakage, revocation and legacy client handling.
- **P0 / O-003:** Bridge NIC ACL group selectors are unavailable; require trustworthy current IP/IPv6 binding and native anti-spoof, fail-closed on address change. No custom ACL editor.
- **P0 / O-010:** Native file upload defaults to `O_CREATE|O_TRUNC`, not exclusive create; SFTP `O_EXCL` is **only a native test candidate**. Strong conditional replace/delete still require real atomic proof or an explicit owner product choice.
- **P0 / O-011/O-012:** `ExecInstance(RecordOutput=true)` with output writers can synchronously wait before returning; native recorded-output getters are private. Use a tested SDK-streaming mode with durable worker output. `GetOperationWait` modifies shared HTTP Transport `ResponseHeaderTimeout`; prefer bounded `GetOperation` polling until concurrent semantics are tested.
- **P1 / O-004/O-013:** Confined Project must have its `features.*`, root disk/Profile and required managed NIC/network/ACL rights configured **before the first instance**; feature choices can lock afterward.
- **P1 / O-007:** Parent NIC anti-spoof controls can block nested L2 forwarding with new MAC. Prove nested managed-bridge NAT instead of disabling native protection broadly.

These are tagged SDK facts plus honest native UNKNOWNs; no live acceptance was run in this document change.

## Historical evidence does not close native validation

[Containarium source and 2,577-subject history review](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md) supplies test counterexamples, **not** v1 PASS evidence. The existing O-002/003/005/010/011/013/018/024 risks already cover exact identity, pre-start security, native forwarding, file race safety, Exec truth, storage and deletion recovery; do not duplicate them as a new platform backlog. Record exact kernel/Incus SDK/runtime/API extension and the failing native probe before weakening an approved guarantee. A feature that native Incus cannot prove remains BLOCKED/UNSUPPORTED until an owner product decision. No scope expansion is authorized by the research.

## Not on the roadmap

Caddy and website domain/TLS controls, generic ACL/CIDR/firewall editing, per-Box Token delegation/RBAC, snapshots/backup, native guest patch/search/bulk transfer, persistent process supervisor, automatic idle sleep/HTTP wake, OAuth browser flow, multi-host/VM management, general host management, cross-layer orchestration. These are not technical blockers or P1 promises. Only a new explicit owner product decision can reopen them.

## Release rule

An unsupported or unverified native behavior may be a real BLOCKED/UNSUPPORTED result, not a fabricated PASS. Failure in mandatory core functions (including private network deny, native forwarding and file read/write) prevents claiming full P0 acceptance. Keep independent test evidence, actual upstream versions and intact original UNKNOWN/failed resources.
