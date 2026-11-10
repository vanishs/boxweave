# Decision register — v1 design baseline

Recorded from the owner's current design discussion. **Accepted** means a product requirement, not an implemented feature. **Candidate** is a recommendation awaiting a separate decision.

| ID | Status | Decision |
|---|---|---|
| D-001 | Accepted | Start a blank-source v1 branch and version it 1.0.0; 1.X.Y increments X for features, Y for patches. |
| D-002 | Accepted | Single physical host per layer; public management exclusively Streamable HTTP MCP, no public REST/Web console. |
| D-003 | Accepted | Remove Containarium, v0.x migrations, Tenant UI and mandatory SSH; use native Incus and guest Exec. |
| D-004 | Accepted | Box identity is the native Incus UUID, subject to verification; names/IPs may be reused without inheriting authority. |
| D-005 | Accepted | Default private Box-to-Box direct isolation, explicit one-way grants, default internet access, optional per-Box denial. |
| D-006 | **SUPERSEDED by permanent scope D-049–D-056** | A Box may access another Box's public Caddy website; bypass protection for that public path is not P0. |
| D-007 | **SUPERSEDED by permanent scope D-049–D-056** | HTTP routes and TCP/UDP forwards are owned, Box-targeted services; Caddy/Incus unmanaged services cannot be published via MCP. |
| D-008 | Accepted | Mutations are asynchronous durable BoxWeave operations with dedup, concurrency protection, UNKNOWN preservation and post-state verification. |
| D-009 | Accepted | Fully compatible self-hosting with no artificial depth limit; each independent layer has the same functions but manages only its directly owned Boxes, not a recursive management tree. |
| D-010 | Accepted | Strict resource ownership boundary even for superadmin MCP calls; no generic host shell, host file operations or arbitrary native API passthrough. |
| D-011 | **SUPERSEDED by permanent scope D-049–D-056** | No mandatory storage driver and whole-Box backup remains deferred. Guest file list/read/write/optimistic patch are now P0; native snapshots remain P1. |
| D-012 | Accepted | A limited host-local install/status/upgrade/credential recovery CLI exists despite no Web UI. |
| D-013 | Accepted | All project artifacts English; Agents communicate in the requester's preferred/current language. |
| D-014 | **Accepted** | Implement v1 in Go using the official Model Context Protocol Go SDK and official Incus Go client/module; Python is superseded as the v1 implementation language. |
| D-015 | **PARTIALLY SUPERSEDED** | Dedicated BoxWeave-owned Caddy rejected permanently. A narrowly privileged local Incus setup helper remains an implementation option only when required to enforce the accepted native ownership/security boundary. |
| D-016 | Accepted | Internal Incus SDK/API calls are allowed; the no-REST constraint applies to BoxWeave's public management surface. Do not build or expose a second public management API. |
| D-017 | Accepted | Nested installs use the same build and bootstrap/maintenance workflow; each layer treats its own environment as host and never silently modifies parent infrastructure. |
| D-018 | **SUPERSEDED by permanent scope D-049–D-056** | Public routing, forwarding and private ACL changes must be bound to an owned Box identity; no arbitrary host, remote service or unmanaged target may be published via MCP. |
| D-019 | Accepted | Native operation receipts must be retained separately from BoxWeave durable business-operation receipts; ambiguous results stay UNKNOWN instead of automatic replay. |
| D-020 | Accepted | Nested operation is a compatibility property for development/testing new BoxWeave versions in managed Boxes and possible future environment subleasing; it is not a v1 cross-layer administration or delegation feature. |
| D-021 | Accepted | A parent treats a Box hosting an inner BoxWeave as an ordinary owned Box; normal deletion semantics apply with no inner-controller discovery, recursive protection or extra deletion handshake. |
| D-022 | **SUPERSEDED by permanent scope D-049–D-056** | Cross-layer ingress is configured through ordinary separately authorized route/forward actions per layer; no automatic multi-layer publish/delegation protocol is in v1 scope. |
| D-023 | Accepted | All fourteen operator defaults were approved on 2026-11-07, as specified in [DEFAULTS.md](DEFAULTS.md): item 1B, item 14B, clarified item 8 (loopback-only HTTP with externally configured proxy/Tunnel), and A for the other items. |
| D-024 | Accepted | Normal Linux Boxes require no installed BoxWeave guest client, resident Agent or SSH to be managed; lifecycle/Exec uses the native Incus Go SDK. Guest root is the default Exec identity; there is no host root MCP surface. |
| D-025 | Accepted | Only a compatible, verified, unissued VERSION on a successfully built `v1` CI run triggers immutable GitHub tag/Release publication; existing installed BoxWeave updates and stable installer promotions remain separately initiated. |
| D-026 | Accepted | The core P0 user outcome is a sustainable Agent development loop: start a clean Box, safely inspect/edit guest files, run asynchronous build/test Exec, recover output and errors by ID, diagnose and iteratively verify. |
| D-027 | Accepted | Every BoxWeave request receives a server-owned Tracking ID where protocol-appropriate. Typed Go context is propagated across handler/domain/worker/Incus/SQLite/minimal operation evidence; distinct request, idempotency, journal and native operation IDs are not conflated. |
| D-028 | Accepted | All mutating tools, including file writes and guest Exec, use durable asynchronous operation receipts; reads may complete synchronously but remain tracked. Timeouts/disconnects must not silently replay an uncertain original. |
| D-029 | Accepted | Box creation starts the Box by default, with explicit `start=false`, and distinguishes native running from actual guest Exec readiness. |
| D-030 | **SUPERSEDED by permanent scope D-049–D-056** | P0 includes guest file list/read and one `box_file_apply` API with create/replace/patch/delete on exactly owned regular guest files, verified original revision and native post-state; P1 retains optional search, snapshots and durable process supervision. |
| D-031 | **SUPERSEDED by permanent scope D-049–D-056** | MCP Token rights are scoped by operation class and optional exact owned Box UUID, including read-only and network-specific rights; no external Token can widen instance ownership. |
| D-032 | **SUPERSEDED by permanent scope D-049–D-056** | P0 provides safe read-only tracking-linked diagnostics and metadata; root/host-wide arbitrary diagnostics and secrets are never available via MCP. |
| D-033 | Accepted | P0 rejects disk shrink before dispatch, even when a storage driver appears to offer resizing; safe disk growth remains capability-gated. |
| D-034 | **SUPERSEDED by permanent scope D-049–D-056** | P0 excludes auto-idle sleep, HTTP auto-wake, OAuth 2.1 browser flows and persistent background process supervision; they may be P1 or later. |
| D-035 | Accepted | Explicit local BoxWeave installation/upgrades record pinned required and observed dependency versions, automatically install or upgrade approved managed dependencies and verify the result. A shared Incus or unowned service requires separate specific local authorization before any disruptive upgrade/restart; no background product/dependency updater. |
| D-036 | **SUPERSEDED by permanent scope D-049–D-056** | P0 optimizes MCP usability without broad generic tools: merge `box_file_write`+`box_file_patch` as `box_file_apply` with four safe modes, add `operation_cancel`, retain 26 tools. |
| D-037 | Accepted | `operation_get` has a bounded read-only `wait_ms` (default 0, maximum 20000); long-poll timeout/disconnect does not cancel the original asynchronous operation. |
| D-038 | Accepted | Cancellation is a separately tracked, explicitly authorized attempt: undispatched queued work may be atomically canceled, running guest Exec may be terminated only with native evidence, and unknown/uninterruptible operations are never mislabeled as canceled or rolled back. |
| D-039 | Accepted | `box_exec` uses typed argv, safe guest cwd, bounded env/timeout/output and guest-root by default; `box_update` exposes only safe explicit CPU/memory/disk-growth/nesting/internet fields. |
| D-040 | **SUPERSEDED by permanent scope D-049–D-056** | Unrestricted guest-root `box.exec` inherently permits guest file changes, so file-tool read/write/delete permission flags cannot pretend to restrict a token that also has Exec. Tool cancel and guest file delete use separately explicit permissions. |
| D-041 | Accepted | P0 source import uses ordinary Git inside an authorized Box through native guest Exec, not a BoxWeave bulk repository upload/download tool. Source credentials, if necessary, are supplied through a separately authorized safe guest workflow, not logged or copied into the controller. |
| D-042 | Accepted | P0 guest file changes fail closed if point-of-use identity, expected hash/revision, symlink/hardlink or atomicity guarantees cannot be verified; no unlabelled best-effort read-compare-write pretending to be safe compare-and-swap. Preserve explicit UNSUPPORTED and native evidence. |
| D-043 | Accepted | P0 cancellation provides provable cancellation of queued-undispatched operations where possible. Running guest Exec cancellation is best-effort and capability-gated, requiring independently verified termination; do not promise process death, effect rollback, or that UNKNOWN has been resolved. |
| D-044 | Accepted (1A) | P0 requires real private Box-to-Box traffic isolation. Deny owned IPv4 and enabled IPv6; if IPv6 cannot be isolated, disable only the owned IPv6 path and reverify or fail closed. Never globally weaken/change foreign network settings. |
| D-045 | Accepted (2A) | Every mutating MCP call requires a caller-provided reusable idempotency key, with durable conflict-safe dedup and original Tracking/Operation ID recovery after lost transport. |
| D-046 | Accepted (3A) | Token revocation blocks new requests and provably undispatched queued effects; dispatched/uncertain operations remain reconciled and are not automatically considered stopped or rolled back. |
| D-047 | Accepted (4A) | Explicit local install/upgrade reconciles approved dependencies but never automatically reboots a whole host; report exact reboot_required and require operator reboot and post-boot verification. |
| D-048 | **SUPERSEDED by permanent scope D-049–D-056** | P0 Box application sites are HTTP-only. Managed Caddy must not auto-enable HTTPS, ACME certificates, redirects or port 443. Website HTTPS is deferred; operator external HTTPS termination is separate. The authenticated loopback MCP transport decision is unchanged. |

Before overturning an accepted item, update product requirements, the affected design and acceptance tests and explicitly record why. Do not treat archived v0.x docs or external repositories as a competing source of authority.

## Permanent product narrowing — superseding earlier accepted items

The following final decisions supersede earlier broad commitments where they conflict. Historical rows remain for traceability; **only these current rules govern implementation**.

| ID | Status | Current decision |
|---|---|---|
| D-049 | **Accepted / supersedes Caddy** | BoxWeave does not install or manage Caddy, websites, domain routes, TLS or operator reverse proxies. Native Incus TCP/UDP port forwards remain an essential permanent capability. |
| D-050 | **Accepted / supersedes generic ACL** | Default native private isolation remains fail-closed; one directional Box UUID pair allow/revoke tool `box_peer_set` serves Box interconnection tests. No generic ACL management or IP/CIDR/port-policy DSL. |
| D-051 | **Accepted / simplifies Token model** | Multiple revocable `admin` or `viewer` MCP tokens only. No per-Box Token scope, tool-class role matrix, tenant or RBAC framework. Viewer can read sensitive guest file/output data. |
| D-052 | **Accepted / simplifies files** | Guest file list/read and one `box_file_apply(create|replace|delete)` only. No built-in patch engine or file search/bulk transfer; complex edits via authorized Guest Git/Exec. Conditional changes fail closed when unsafe. |
| D-053 | **Accepted / simplifies diagnostics** | Eliminate `diagnostic_get` and separate diagnostic query store; use minimal redacted result/phase/evidence fields in `operation_get`, Tracking ID on errors, and protected host-local logs/doctor. |
| D-054 | **Accepted / preserves Incus lifecycle** | Explicit local dependency management can reuse, install and manage compatible already-running Incus daemon versions. Shared disruptive upgrades need target-specific approval, and existing unowned containers are never adopted automatically. Caddy removed from dependencies. |
| D-055 | **Accepted / evolving verified images** | Ubuntu LTS first, additional Linux guest distributions only after real native compatibility checks; not hard-locked to Ubuntu forever. |
| D-056 | **Accepted / permanent exclusions** | Removed Caddy/ACL/RBAC/OAuth/snapshots/backup/idle sleep-wake/background process platform/text patch engine/search/bulk file tools are not deferred feature promises. Reintroduction requires owner approval. |

| D-057 | **Accepted — owner chose A** | “Take over existing Incus” means reuse its compatible daemon and manage its required dependency versions through the authorized local installer; do **not** import existing containers or assume their ownership. Already-running foreign instances/projects remain untouched. |

| D-058 | **Accepted — owner approved minimal UNKNOWN recovery** | A host-local operator can review exact UNKNOWN task/resource evidence, receive a short-lived challenge, confirm a second time with reason/risk acknowledgement, and durably audit release of **only the local claim**. Never auto-retry, cancel, mark success or modify the original receipt; no new MCP tool. |

| D-059 | **Accepted — owner-approved S2.5 BRIDGEHEAD plan** | Insert a zero-new-tool **independent native early development-base acceptance and reversible physical-host pilot** between fully accepted S2 and S3. Preserve all S1/S2/S3/S4/S5 security, acceptance and release rules. LAB_PASS is not HOST_GO; actual host/Project/pool/service/port modification requires a new separate explicit operator authorization, and unsafe shared Incus/network prerequisites mean NO_GO. S3 may proceed safely in isolated labs if physical pilot is blocked; no legacy OAuth, global Incus repair, new SDK transport or automatic `main` adoption. [Phase](../development/phases/025_BRIDGEHEAD.md) · [BH cases](../verification/BRIDGEHEAD_ACCEPTANCE.md). |

Canonical narrow policy: [PRODUCT_BOUNDARY.md](PRODUCT_BOUNDARY.md).
