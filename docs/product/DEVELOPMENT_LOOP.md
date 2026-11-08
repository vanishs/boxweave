# Permanent core: sustainable Agent development loop

BoxWeave enables a coding Agent to create and manage an owned Linux container, develop and test code, recover outputs, fix problems and rerun acceptance. This must work in independent L0/L1/L2 installations when native Incus/kernel prerequisites permit. It does not require a resident Guest Agent, SSH server, general hosted Git service or a separate controller.

## Real minimum workflow

1. **Create** a reviewed Ubuntu LTS Box using `box_create` with configured resource defaults, optional `nesting=true` and default `start=true`. Receipt is async and contains original Tracking ID/operation ID. Do not call the Box ready until native state and guest Exec readiness are observed.
2. **Fetch source** using a guest `git clone` through typed `box_exec`, installing build tools *inside the guest* as needed. Do not add repository upload or Git credential services to BoxWeave. Private repo secrets must not appear in controller logs.
3. **Read/edit basic files** with bounded `box_file_list`, `box_file_read` and `box_file_apply(mode=create|replace|delete)`. Existing-file changes require an observed revision/hash and **safe native point-of-use validation** or return UNSUPPORTED. For patches and larger refactors use guest Git/shell commands via `box_exec`; do not maintain a text-patch parser.
4. **Build/test** via guest-root async `box_exec` with typed argv, bounded guest cwd/env/timeout/stdout/stderr; no host shell. Use `operation_get(wait_ms)` for bounded read-only progress/phase/error summaries and `operation_output_get` for capped original output and exit status after reconnect.
5. **Fix/retest:** a nonzero guest test exit is a completed failed test, not a lost transport result. Correct the file and rerun. Use `operation_cancel` for proven queued cancellation or supported verified active Exec termination. Preserve UNKNOWN until native evidence resolves it; never duplicate a possibly dispatched action.
6. **Publish for testing when needed:** native `port_forward_create` maps a chosen host TCP/UDP port to an owned Box port. If test service A must privately call B, `box_peer_set` temporarily allows directional A->B; revoke when done. There is **no Caddy**, managed domain, proxy or custom network ACL editor.
7. **Verify independently:** preserve source SHA, installed binary/module versions, exact Box UUID, idempotency key, original Tracking/Operation IDs, output/exit and native post-state. Repeat inside an appropriately nested Box to validate a candidate BoxWeave without altering the outer controller.

## Explicitly not necessary

Built-in patch engine, file search/bulk file transfer, independent diagnostics MCP service, special IDE, user/tenant/RBAC, per-Box Tokens, process supervisor, idle auto-wake, website manager, Caddy and snapshots are **not scope commitments**. They are not simply waiting in P1.

## Security and acceptance

Admin tokens may Exec as guest root and therefore edit any guest file allowed by Incus confinement. Viewer tokens can read guest files and output, potentially containing secrets, but cannot Exec or mutate. Never expose host files or foreign Incus/project/port/peer resources.

Test the complete create -> clone -> file read/change -> deliberate failing guest test -> inspect original output -> correct -> successful test -> demonstrate owned port forwarding and peer allow/revoke -> independently verify and delete owned test resources. Test native race/unsupported file semantics, unsupported isolation, client disconnect, lost native receipt, token revocation, nested L1/L2 behavior and real resource cleanup.

See [minimal MCP catalog](../design/MCP_TOOLS.md), [native tool contracts](../design/MCP_TOOL_CONTRACTS.md), [operations](../design/OPERATIONS.md) and [permanent boundary](../decisions/PRODUCT_BOUNDARY.md).
