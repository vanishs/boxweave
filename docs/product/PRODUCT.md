# Product definition — small by design

BoxWeave is a **single-host, Agent-first Incus Linux-container controller**. One BoxWeave installation manages only the Boxes it directly owns. An ordinary nesting-capable Box can run an independent installation of the same BoxWeave; the parent does not administer the inner controller.

The core experience is a reliable development loop: create and start an agentless Box, inspect/read/basic-safe-edit files, run guest-root commands asynchronously, obtain original output/exit/diagnostic summaries after reconnect, fix, retest and independently accept a candidate—including another BoxWeave installed inside a nested Box.

BoxWeave offers authenticated loopback Streamable HTTP MCP plus local maintenance CLI. Current v1 and RC13 use scope-free OAuth and independently revocable full-management credentials per Actor. The Go implementation uses official MCP/Incus SDKs, exact native ownership, durable original Operation receipts, bounded output and fail-closed unsafe states. External HTTPS/Tunnel stays operator-managed. See [current release and source status](../INDEX.md).

**Networking stays small:** owned Boxes cannot directly communicate privately by default; a single directional Box-to-Box peer allow/revoke operation supports integration testing. Native Incus TCP/UDP port forwarding exposes an owned Box service on an explicitly authorized host port. **No Caddy, website/domain publishing or custom forwarding daemon** is part of BoxWeave. Operators can independently use a reverse proxy, Cloudflare Tunnel or DNS as they wish.

The installer verifies and uses an already configured compatible Incus daemon, with a restricted Project and reviewed native preparation. It does not provision a shared daemon, adopt unrelated resources, change foreign resources or reboot the host. The draft #191 controller is fresh-only Scoped v2: no product upgrade, rollback or adoption. Reviewed full retirement and separate completed-data purge are supported within the exact installed contract; historical Legacy upgrades apply only to their original builds.

Ubuntu LTS is the initial verified host/guest baseline. Other guest Linux distributions can be supported progressively after real validation. General external identity/tenant platforms, backup/snapshot managers, idle wake, general process supervision, rich ACL/RBAC, native website publishing and full multi-host infrastructure are **not** product roadmap obligations.

See [permanent scope](SCOPE.md), [boundary decisions](../decisions/PRODUCT_BOUNDARY.md), and [Agent development loop](DEVELOPMENT_LOOP.md).



See [current status](../STATUS.md), [installation](../getting-started/MANUAL_INSTALL.md) and [acceptance](../verification/ACCEPTANCE.md).
