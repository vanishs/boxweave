# Permanent v1 product boundary — keep BoxWeave small

**Owner review (confirmed option A):** Retain native Incus TCP/UDP forwarding and a minimal private peer-connection toggle, permanently drop Caddy and generic ACL editing. **Taking over an existing Incus installation means reusing and managing the daemon/version only; it does not mean importing any existing containers.** Never automatically adopt or change unrelated existing instances/projects. Other previously proposed simplifications are accepted. These changes **supersede** older P0/P1 plans rather than reschedule removed features.

## The product is

One independent, single-host, MCP-managed controller for **BoxWeave-owned Linux containers**, with the same capabilities when installed inside a nesting-capable Box. Its long-term core is a sustainable Agent workflow: provision a Box, inspect/read/basic safe file changes, run and recover asynchronous guest Exec, debug via Tracking/operation receipts, fix, test and independently verify.

The management MCP is authenticated loopback HTTP. External access (including Cloudflare Tunnel) is configured by the operator. Every mutation requires a reusable client idempotency key. Tracking ID/context and recoverable native UNKNOWN remain fundamental, not optional.

## Minimal network surface (permanent)

1. **Keep** native Incus `port_forward_list`, `port_forward_create`, `port_forward_delete` for owned Box UUID + guest TCP/UDP port. Support an explicitly authorized host listener/bind address and port; default to loopback when possible, and require explicit acknowledgment/authorization for a public bind. Verify conflicts/ownership and native capability. **No custom proxy daemon.**
2. **Keep one** `box_peer_set(source_box_id,target_box_id,allow)` mutation for directional **all-protocol** private peer access between two currently owned Boxes, with a strict default deny and readback in `box_get`. No generic ACL list/add/remove, CIDR, IP selectors, arbitrary protocol/port rules, or unmanaged network edits. Full native IPv4 and enabled IPv6 safety is required; fail closed on unsupported topology.
3. **Remove Caddy entirely** from the product and installer: no HTTP route tools, Caddy service/API, managed website domains, TLS/certificates, or general reverse proxy. A service may be reached via a mapped host TCP/UDP port; operators may independently provide DNS, HTTP/HTTPS reverse proxy or Tunnel. BoxWeave never edits those operator-managed services.
4. No special multi-layer forwarding or peering orchestration. Operators configure ordinary layer-local native forwards separately at each layer where necessary.

## Other permanent reductions

- **Auth:** one local super-admin concept; multiple revocable MCP tokens with **admin** (all scoped owned resources, Exec included) or **viewer** (read-only owned Box metadata/files and operation status/output). No per-Box token delegation, RBAC engine, role hierarchy or separate tool permission classes. Viewer can read sensitive guest content/output: document and protect those credentials. Viewer cannot Exec or mutate. Existing Box ownership restriction always applies.
- **Files:** bounded `box_file_list`/`box_file_read`, and one `box_file_apply` with **create/replace/delete** of a single regular guest file, with revision/hash safety where an existing file is mutated. **Latest official Incus SDK verification** proved `SFTP O_EXCL` exclusive create but **no native atomic compare-and-swap for conditional replace/delete** ([evidence](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md)); if safe point-of-use conditions cannot be established, these modes return **`UNSUPPORTED`**, never a blind overwrite/delete or fake hash pre-check. **No built-in text patch engine**, search, bulk repository transfer or general host file editor. Complex code edits use ordinary Git/shell tools via authorized guest Exec.
- **Diagnostics:** remove independent `diagnostic_get` MCP API and separate diagnostics product/model. `operation_get` returns a small, redacted phase/error/evidence summary for admitted operations, including Tracking ID. Pre-admission failures carry Tracking ID in the MCP response; detailed tracing is in protected local logs/CLI doctor. Do not build a tracing platform.
- **Dependencies:** keep version-aware explicit local installation/upgrade, including **reusing and, with scoped authorization, maintaining/upgrading an existing Incus daemon**. Record actual versions before/after and required capabilities. Never silently alter foreign Incus instances/projects, shared daemon consumers or foreign services. A disruptive shared upgrade requires its own local impact approval. **Caddy is not a dependency**.
- **Images:** Ubuntu LTS is the initial officially verified guest and host baseline; **additional Linux guest distributions can be certified incrementally** after native tests, with no promise of universal image compatibility.
- **Exclusions, not P1 commitments:** native snapshots, guest file search/bulk transfer, resident guest Agent, generic background process supervisor, automatic sleep/wake, OAuth browser login, Caddy site publishing/HTTPS, generalized ACL/RBAC, multi-tenant administration, backup platform, Web/REST control planes and whole-host tooling. Re-entry requires a new owner-approved product decision, not a scheduled phase.

## Native Incus environmental boundary, owner-confirmed 2026-11-08

Repeated **Cowsql leader unavailability** during nested Incus test Box creation is evidence of an unstable or limited test environment until its cause is independently isolated; preserve native errors/operation UUIDs and use normal BoxWeave UNKNOWN handling to avoid duplicate mutation. **Do not develop an Incus/Cowsql database patch, custom database consistency layer, Incus recovery daemon, upstream workaround engine or new BoxWeave v1 service to fix the test Box.** Native Incus health/stability ownership stays with Incus and the operator. This is a permanent *non-goal*, not P1 work. See [executed SDK P0 evidence](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md).

## Unchanged P0 safety and experience

Go + official MCP/Incus Go SDKs; native instance UUID/ownership; restricted native privilege; agentless guest-root Exec; native-lifecycle Box management; durable async Tracking ID/Operation/UNKNOWN with client reusable idempotency; guest result/output retention; minimal files; fail-closed private isolation; native TCP/UDP forwards and peer toggle; single-layer independently nestable installs; local dependency/upgrade management with no automatic host reboot; CI source release policy; real adversarial negative testing.

This narrows long-term maintenance, not evidence thresholds. No deleted feature may be silently resurrected under a different tool name. Implementation must prove the native forwarding and per-NIC peer-isolation feasibility before claiming this design is available.
