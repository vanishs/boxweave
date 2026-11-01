# R1 — Streamable HTTP MCP server

Parent: [remote MCP master](../REMOTE_MCP_MASTER.md). Status: **DESIGN ONLY / NOT IMPLEMENTED**. Dependency: [R0](R0_PROTOCOL_AND_DISPATCH.md). Next: [R2](R2_AUTH_AND_CREDENTIALS.md).

## Goal

Provide a standard, independently monitored **inbound** MCP server on `/mcp` over loopback HTTP. No public exposure or OAuth credentials are provisioned in R1.

## Proposed behavior

- Support POST JSON-RPC requests, notifications, responses; honor required `Accept: application/json, text/event-stream`; respond with JSON or SSE as appropriate. Support GET SSE or explicitly return `405` where unsupported; handle DELETE/session semantics per spec.
- Negotiate `initialize` and `notifications/initialized`, protocol version, capabilities, `MCP-Protocol-Version`; support `tools/list`, `tools/call`, and supported resources. Bound pagination, request body size, output, concurrent work and upstream timeouts.
- Choose stateless behavior if it passes target-client compatibility; otherwise implement secure `Mcp-Session-Id` lifecycle and binding to the authenticated caller (R2). Never use the MCP session ID itself as authentication.
- Validate `Origin` and reject invalid supplied Origins with `403`; prevent DNS rebinding. Do not disable Origin checks just because a request comes through a reverse proxy.
- Respond clearly to invalid media types, malformed JSON-RPC, invalid/unsupported version, stale session, missing `Accept`, lost backend, idle SSE, proxy disconnect and interrupted mutating operations.
- The application Web session cookie is **not** accepted for MCP API authentication. Until R2 is complete, HTTP listener is loopback-only, disabled in the ordinary install path, accessible only to owned fixture clients.
- Preserve observability without logging sensitive tool input/results, tokens or complete credential URLs; no separate arbitrary Flask `tools/call` route.

## Suggested code seams

A separate `bwmini/mcp_http.py` adapter is preferable to adding an unrestricted route to `bwmini/web.py`. Reuse R0 dispatcher, independently start/stop listener, and use existing non-root managed service boundaries. Evaluate SDK/framework choice against real protocol compliance rather than assuming Flask automatically supports durable SSE streaming. The external exposure process must be independent of a broken Web dashboard where feasible.

## Acceptance

- Raw JSON-RPC and an independent MCP Inspector/client complete initialize, list and a read-only call, with actual stdout/results and errors distinguished.
- Concurrent requests, mixed clients and long-lived response paths cannot leak messages across clients.
- POST/GET/DELETE, content negotiation, 202 and 405, Origin rejection, session/version rules, stream disconnect, timeout, upstream loss all have precise fixture tests.
- Source/fixture completion is not a public listener or ChatGPT compatibility claim.
- No R1 installation opens a new public port; legacy SSH/stdio remains functional.

Follow [remote MCP acceptance matrix](../../verification/REMOTE_MCP_TEST_PLAN.md), R1/N1. Reference [MCP transports](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports).
