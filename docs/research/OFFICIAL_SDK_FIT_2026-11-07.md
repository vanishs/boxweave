# Official SDK fit review: Incus Go SDK and MCP Go SDK

**Date:** 2026-11-07  
**Target:** independent BoxWeave `v1` design branch; no implemented v1 runtime.  
**Source baselines:** Incus `v7.6.0` (`github.com/lxc/incus/v7`, released 2026-10-30); MCP Go SDK `v1.8.0` (`github.com/modelcontextprotocol/go-sdk`, released 2026-09-14). Exact Go toolchain and module pins are implementation decisions to verify by compilation, not accomplishments of this review.

## Principle

**Trust the upstream daemon and official SDKs. Shape BoxWeave v1 around their actual APIs and semantics, rather than making an artificial feature contract and working backwards.** BoxWeave owns only its narrow policy: authorized owned-resource access, durable caller idempotency, original operation receipts and business postcondition evidence. Do not reimplement an Incus engine, MCP transport/session protocol, general network platform or file-editing service.

Evidence levels: **CONFIRMED** = examined official tagged code/docs; **DESIGN** = proposed fit with upstream APIs; **UNKNOWN** = requires real native/ChatGPT-client validation. No operational integration tests were executed. Existing [product scope](../decisions/PRODUCT_BOUNDARY.md) remains authoritative unless explicitly changed.

## Cross-SDK design matrix

| ID / priority | Verified SDK reality | v1 correction or better design | Evidence |
|---|---|---|---|
| S01 **P0** | Incus `Operation.WaitContext(ctx)` on its event-listener path invokes `op.Cancel()` when `ctx.Done()` fires. | **Do not** use `WaitContext` as a read-only 20-second status wait. Use `GetOperationWait` (server-side timeout) or bounded `GetOperation` observation; reserve native cancel for explicit `operation_cancel`. | [operations.go L183-L249](https://github.com/lxc/incus/blob/v7.6.0/client/operations.go#L183-L249) — CONFIRMED |
| S02 **P0** | **Corrected after deeper reading:** Incus local Unix socket grants full privileges to root/`incus-admin`, but a non-root OS user **only in `incus`** is natively confined to its per-user Project. `UseProject` remains just a client selector. | Prefer a dedicated non-root `incus`-only service user, admin-configured `restricted=true` Project and explicit needed capability permissions. Restricted TLS is fallback; a privileged helper is last resort. Prove foreign/admin denial and proxy/ACL/nesting compatibility. | [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/), [project confinement](https://linuxcontainers.org/incus/docs/main/howto/projects_confine/) — native privilege distinction CONFIRMED; exact P0 support UNKNOWN |
| S03 **P0** | Incus `GetInstance(name)`/`UpdateInstance(name,put,ETag)`/`DeleteInstance(name)` are name-addressed. The stable native identity is the internal `volatile.uuid` in returned instance `Config`. | Expose UUID as public Box ID, but resolve it to the current owned project/name for native calls and compare the live `volatile.uuid` immediately before effects. `volatile.uuid.generation` is distinct and must not be conflated with instance identity. | [Instance interface](https://github.com/lxc/incus/blob/v7.6.0/client/interfaces.go), [Instance data type](https://github.com/lxc/incus/blob/v7.6.0/shared/api/instance.go), [UUID docs](https://linuxcontainers.org/incus/docs/main/reference/instance_options/) — CONFIRMED |
| S04 **P0** | `GetInstanceFile`, `CreateInstanceFile`, `DeleteInstanceFile` and SFTP are available; `InstanceFileArgs` has content/type/overwrite/append, **no caller-hash, ETag or atomic compare-and-swap field**. Delete takes only path. | Keep native file read/list and ordinary file APIs. A pre-read SHA does not make conditional replace/delete atomic or race-free. Verify a small SDK-supported one-shot guest-Exec technique or return **UNSUPPORTED** for unproven conditional modes; no resident file daemon or invented atomicity. | [SDK interfaces and file args](https://github.com/lxc/incus/blob/v7.6.0/client/interfaces.go#L719-L759), [file methods](https://github.com/lxc/incus/blob/v7.6.0/client/incus_instances.go#L1493-L1715) — native CAS NOT PRESENT; alternative UNKNOWN |
| S05 **P0** | Native Incus `proxy` device provides an instance-owned `listen`/`connect` and `bind=host` for TCP/UDP. A `network-forward` object is **network-scoped listen IP / port rules**, not an interchangeable host-listener backend. | **Prefer one instance-bound proxy-device backend for P0** using guarded instance-device update + ETag + readback. Do not introduce a network-forward lifecycle/orphan manager as an implicit alternative; revisit only if a tested topology requires it. | [Proxy device](https://linuxcontainers.org/incus/docs/main/reference/devices_proxy/), [network forward](https://linuxcontainers.org/incus/docs/main/howto/network_forwards/), [typed API](https://github.com/lxc/incus/blob/v7.6.0/shared/api/network_forward.go) — CONFIRMED distinction; P0 backend DESIGN |
| S06 **P0** | On managed bridges, a **network-level** ACL is not an intra-bridge firewall. NIC-attached bridge ACL policy is a separate path with limitations. | Choose one native NIC-level supported topology for same-bridge IPv4/IPv6 deny and directional `box_peer_set`, enforce before first start, and independently test packet paths. Neither Project segregation nor an attached ACL name proves traffic denial. | [Incus ACL bridge limitations](https://linuxcontainers.org/incus/docs/main/howto/network_acls/) — CONFIRMED limit; live behavior UNKNOWN |
| S07 **P1** | Many `InstanceServer` methods lack `context.Context` parameters. SDK connection helpers and `Operation.WaitContext` have context, with different semantics. | Preserve safe context propagation **inside BoxWeave**, but don't promise that every SDK call accepts context or that caller disconnect cancels the native call. Use bounded transport/worker budgets and actual SDK guarantees. | [InstanceServer](https://github.com/lxc/incus/blob/v7.6.0/client/interfaces.go), [connection.go](https://github.com/lxc/incus/blob/v7.6.0/client/connection.go) — CONFIRMED |
| S08 **P1** | Incus native Operation has ID/status/error/metadata/may_cancel; SDK has `GetOperation`/`GetOperationWait` and typed Exec streams and output recording. | Keep only the **small BoxWeave business journal** for admission, authorization, idempotency and reconnect. Do not duplicate Incus's native execution engine. Validate Exec websocket/RecordOutput timing, exit metadata and explicit cancellation independently. | [Operation model](https://github.com/lxc/incus/blob/v7.6.0/shared/api/operation.go), [Exec client](https://github.com/lxc/incus/blob/v7.6.0/client/incus_instances.go) — CONFIRMED API, run-time UNKNOWN |
| S09 **P1** | MCP `v1.7+` supports protocol `2026-07-28`. Its Streamable HTTP implementation accepts that version **only when `Stateless=true`**; older protocol compatibility remains. | Candidate default: SDK-owned **Stateless Streamable HTTP**, with durable BoxWeave operations queried after reconnect. Do not maintain a custom MCP session store, JSON-RPC dispatcher or `Last-Event-ID` replay for 2026-07-28. Verify ChatGPT and legacy clients before freezing. | [v1.7 release](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.7.0), [v1.8 source](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/streamable.go) — CONFIRMED, client compatibility UNKNOWN |
| S10 **P1** | Generic `mcp.AddTool` provides typed input/output Schema inference, JSON validation and structured output handling. | Use `mcp.NewServer` and typed `mcp.AddTool` for the 20 public tools. Keep product authorization, business constraints and safe failures in BoxWeave; do not roll another protocol/schema layer. | [SDK server guide](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/docs/server.md) — CONFIRMED |
| S11 **P1** | MCP HTTP cross-origin protection is configurable, not guaranteed by a nil default. HTTP authentication and BoxWeave role/ownership checks are not done by registering tools. | Use explicit outer Go HTTP middleware for token auth/revocation, Origin/Host checks, bounded body/rate limits, trusted proxy configuration. Validate against loopback and chosen external tunnel. | [SDK transport](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/streamable.go) — CONFIRMED |
| S12 **P2** | New protocol uses `server/discover` and per-request metadata, older protocols `initialize`. Cancellation and resumption behavior depend on negotiated version. | Test both modern discovery and legacy initialization paths, record actual negotiated version, and make `operation_get` the sole business reconnect mechanism. | [SDK protocol docs](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/docs/protocol.md), [v1.7 notes](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.7.0) — CONFIRMED |

## Second-pass SDK findings

The previous 12-item review was not exhaustive. See the complete [20/20 public-tool native SDK mapping](SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md). Additional tagged-source findings:

| ID | Upstream ability/constraint | Consequence for v1 |
|---|---|---|
| S13 P0 | Native `incus` group gives project-confined local access. In restricted Projects, `restricted.devices.proxy` and `restricted.containers.nesting` default to `block`. | Start with one confined service identity; provision exact exceptions under host-local administrative approval; prove native permission denials. |
| S14 P0 | `GetInstancePortForwardConn` is a temporary connection, not a durable host listener; `CreateNetworkPeer` peers networks, not two Boxes. | Use instance `proxy` device for port publishing; NIC-attached `NetworkACL` for `box_peer_set`. |
| S15 P1 | Incus `api.StatusErrorMatch` and `StatusErrorCheck` recognize wrapped native HTTP errors. | Avoid upstream error-substring matching or private numerical statuses. |
| S16 P1 | MCP SDK `auth.RequireBearerToken` and `auth.TokenInfoFromContext` implement Bearer HTTP middleware. | BoxWeave supplies revocation-aware verifier and admin/viewer rights, not a second HTTP bearer parser. Token expiration checked by default, unless `AllowMissingExpiration` explicitly chosen. |
| S17 P1 | MCP HTTP handler already provides localhost Host/DNS-rebinding protection and a default **4 MiB** body cap; Origin protection is configurable. | Reuse SDK defaults; configure only explicit Origin/rate/proxy policy; size file upload JSON/base64 content accordingly. |
| S18 P1 | Typed `mcp.AddTool` maps ordinary handler errors to `CallToolResult.IsError`; tool annotations are hints, not security. | Return redacted business failures as tool errors; use JSON-RPC errors only for protocol failures; enforce privileges separately. |
| S19 P1 | Incus `ExecInstance` recording + output writers can internally wait before returning; raw/native operation and output-collection lifetimes differ. | Choose one experimentally proven async Exec output strategy before freezing `box_exec`. |
| S20 P2 | Incus v7.6.0 tagged `go.mod` says **Go 1.26.9**; MCP Go SDK v1.8.0 says Go 1.25.0. | Build using a toolchain meeting the stricter dependency; end users of compiled binaries need not install Go. |

References: [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/), [restricted Project settings](https://linuxcontainers.org/incus/docs/main/reference/projects/), [option definitions](https://github.com/lxc/incus/blob/v7.6.0/doc/config_options.txt), [native HTTP status errors](https://github.com/lxc/incus/blob/v7.6.0/shared/api/error.go), [Exec implementation](https://github.com/lxc/incus/blob/v7.6.0/client/incus_instances.go), [MCP Bearer middleware](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/auth/auth.go), [MCP Streamable HTTP](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/streamable.go), [typed tool handling](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/mcp/tool.go).

## Recommended native architecture

```text
127.0.0.1 HTTP listener (operator proxy optional)
  -> explicit HTTP auth, Origin/Host and request-size/rate limits
  -> official MCP Go SDK (typed AddTool, native Stateless Streamable HTTP)
  -> narrow BoxWeave domain policy (admin/viewer, owned live UUID checks)
  -> small SQLite ledger (durable idempotency, IDs, uncertainty, bounded evidence)
  -> bounded workers / small Incus adapter
  -> official Incus Go SDK (Instance, native Operation, Exec, files, proxy, NIC ACL)
  -> Incus daemon / owned containers
```

**Trust/privilege:** a dedicated non-root `incus`-only user has native Project confinement over the Unix socket; test this first. Keep host administrator provisioning separate. SDK `UseProject` alone does not restrict credentials. Use restricted TLS only if the group model is insufficient; a helper is the last resort.

**Identity:** external IDs should remain native Incus `volatile.uuid`, but SDK calls requiring names must re-resolve project/name and confirm live UUID plus ETag as applicable. No name-based adoption.

**Operations:** SDK native operations are authoritative for native execution. BoxWeave persists business admission, idempotency and receipts only; an observation timeout does not imply native cancellation or success. Beware `WaitContext` cancellation.

**Network:** prioritize the proxy-device host listener and NIC-level peer policy; do not build two equivalent-looking but semantically different network forwarding infrastructures. One backend can simplify deletion dependency handling.

**Files:** do not make the product P0 “safe atomic replace” depend on properties the SDK does not claim. The file-apply conditional modes need a real technical proof or a deliberate narrower contract before they are declared implemented.

**MCP:** source of truth for negotiation, transport and Schema is the SDK. Stateless HTTP transport does not mean stateless BoxWeave business operations. Per-request authorization is still BoxWeave's responsibility.

## Historical changes worth learning from

- [MCP Go SDK v1.7.0](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.7.0) introduced 2026-07-28 stateless/discover transport behavior, deprecated session-era features and removed resumption for the new protocol revision.
- [MCP Go SDK v1.8.0](https://github.com/modelcontextprotocol/go-sdk/releases/tag/v1.8.0) hardened resource usage/session teardown and added supported-version controls; use the released SDK behavior, don't copy an older homemade transport.
- [MCP cancellation #1267](https://github.com/modelcontextprotocol/go-sdk/commit/3ebc6d1eb7206d50590341f409e1499676db7345) shows transport/protocol-specific cancellation replies. [Version-rejection #1345](https://github.com/modelcontextprotocol/go-sdk/commit/7c486efa8223d5f66af8cf7b2ac9bb04a1ee6b91) illustrates continuing upstream evolution; **do not assume a later commit is already in v1.8.0**.
- [Incus operation access correction](https://github.com/lxc/incus/commit/86c2a125ff8844f648721249d9fc90ef541c1c76) shows the importance of native permission scoping; BoxWeave still must enforce its own ownership before returning operation data.
- [Incus v7.6.0 release](https://github.com/lxc/incus/releases/tag/v7.6.0) is the reviewed stable release, not a blanket promise about the newer `main` tree. These changes are integration lessons, not reasons to distrust Incus.

## Proposed implementation / acceptance sequence

| Stage | Action | Gate |
|---|---|---|
| D0 — documentation | Correct wait/cancel, Unix privileges, UUID/name, file CAS, same-bridge ACL and proxy backend claims; update docs cross-links. | No known false guarantees remain as unqualified assertions. |
| D1 — **Incus SDK native spike** | In authorized disposable Box/Project, test restricted native identity; UUID resolution; typed ETag update; `GetOperationWait` versus explicit cancel; long Exec stdout/exit; file race/symlink; proxy host TCP+UDP; NIC ACL same-bridge IPv4/IPv6. | Original native state plus negative evidence; independent LAN/peer packet witness where relevant. |
| D2 — **MCP SDK spike** | Typed 20-tool skeleton, explicit HTTP middleware, Stateless=true, 2026-07-28 discovery, legacy clients, real ChatGPT auth/call/disconnect/reconnect. | Capture negotiated protocol; ensure SDK, not homemade JSON-RPC, serves requests. |
| D3 — contract freeze | Settle conditional guest file write guarantee, verified network topology and exact native credential boundary; choose one backend. | UNSUPPORTED instead of fabricated atomicity/safety. |
| D4 — implement and accept | Narrow domain service + SQLite receipts + native SDK; real cross-tool black-box tests and independent verification. | A compiled SDK example is not deployed/native acceptance. |

**No v1 code, install, live SDK exercise or native verification was performed in this study.** Entries requiring runtime evidence remain UNKNOWN. The next priority is D1, not additional abstractions.
