# Product definition — small by design

BoxWeave is a **single-host, Agent-first Incus Linux-container controller**. One BoxWeave installation manages only the Boxes it directly owns. An ordinary nesting-capable Box can run an independent installation of the same BoxWeave; the parent does not administer the inner controller.

The core experience is a reliable development loop: create and start an agentless Box, inspect/read/basic-safe-edit files, run guest-root commands asynchronously, obtain original output/exit/diagnostic summaries after reconnect, fix, retest and independently accept a candidate—including another BoxWeave installed inside a nested Box.

BoxWeave exposes authenticated loopback HTTP Streamable MCP and a local maintenance CLI. **RC.3 also ships optional embedded OAuth** for browser consent and issuance of the same admin/viewer privileges; public HTTPS/Tunnel is operator-managed and uses the same /mcp endpoint. It uses Go and the official MCP/Incus SDKs, exact native ownership, durable tracking/operation receipts, bounded output and fail-closed unsafe states.

**Networking stays small:** owned Boxes cannot directly communicate privately by default; a single directional Box-to-Box peer allow/revoke operation supports integration testing. Native Incus TCP/UDP port forwarding exposes an owned Box service on an explicitly authorized host port. **No Caddy, website/domain publishing or custom forwarding daemon** is part of BoxWeave. Operators can independently use a reverse proxy, Cloudflare Tunnel or DNS as they wish.

The installer detects, records, installs or upgrades compatible dependencies, and **can reuse an already running Incus daemon** in its own restricted Project, with separately authorized shared-daemon impact when needed. The installer does not automatically adopt unrelated pre-existing Incus instances, change foreign resources or reboot the host.

Ubuntu LTS is the initial verified host/guest baseline. Other guest Linux distributions can be supported progressively after real validation. General external identity/tenant platforms, backup/snapshot managers, idle wake, general process supervision, rich ACL/RBAC, native website publishing and full multi-host infrastructure are **not** product roadmap obligations.

See [permanent scope](SCOPE.md), [boundary decisions](../decisions/PRODUCT_BOUNDARY.md), and [Agent development loop](DEVELOPMENT_LOOP.md).


**RC.3 evidence:** The [manual installation guide](../getting-started/MANUAL_INSTALL.md) separates service installation from first-Box preparation. A [scoped physical-host pilot](../verification/RC3_PHYSICAL_HOST_PILOT_2026-11-13.md) proved real ChatGPT/OAuth MCP reads, not first Box Create or network/quota acceptance; neither HOST_GO nor S5 is signed.
