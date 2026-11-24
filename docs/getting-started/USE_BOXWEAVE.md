# Connect and use BoxWeave

Start with the [installation guide](MANUAL_INSTALL.md). This guide is version-aware; see [current status](../STATUS.md). Discover the installed tools and build identity first.

## Connect an MCP client

The local Streamable HTTP endpoint is http://127.0.0.1:18650/mcp. Set PROJECT to the actual restricted Project (normally user-UID for the boxweave service user).

For a trusted local client supporting Bearer authentication, the operator can issue a separate credential from a private terminal:

```sh
sudo runuser -u boxweave -- /usr/local/bin/boxweave token-create -project "$PROJECT"
```

Use the returned token as an Authorization Bearer credential in the client. Keep it private; never put it in a prompt, repository or log. Revoke it by exact token ID with token-revoke under the same service identity (check installed command help).

For remote OAuth, configure an approval password and an operator-managed HTTPS origin as described in [installation](MANUAL_INSTALL.md#connect-a-remote-oauth-client). Use the exact HTTPS origin followed by /mcp. OAuth grants and Bearer tokens are independently revocable full-management credentials, not per-Box roles or application permission scopes. Each nested controller needs its own credentials.

After authorization, discover tools and call host_status and box_list. Anonymous HTTP 401 confirms denial, not successful OAuth.

## Work with Boxes

The [MCP catalog](../design/MCP_TOOLS.md) describes the 23-tool current source interface; the installed client's tools/list and schemas govern actual calls.

| Task | Tools |
| --- | --- |
| Create and inspect | box_create, box_list, box_get, box_wait_ready |
| Manage lifecycle | box_update, box_start, box_stop, box_delete |
| Build/test | box_exec, box_file_list, box_file_read, box_file_apply |
| Connect workloads | box_peer_set, port_forward_list, port_forward_create, port_forward_delete |
| Follow work | operation_get, operation_list, operation_output_get, operation_cancel, box_activity_get, operation_recover |
| Check controller | host_status |

For each mutation keep its client idempotency key and original operation ID. Inspect operation_get until terminal; use the verified owned Box UUID, not just its name. Check readiness before Exec. Guest exit code must be read independently of the operation state.

Use box_file_apply only for supported exclusive create. Use guest Git/Exec for ordinary code changes. For cleanup remove unwanted forwards, stop the Box, then delete the exact owned Box after reviewing dependent resources. Retained output is bounded and may expire.

## Connect workloads

Grant only the required directional Peer relationship; reverse access is not implied. Legacy Peer operations can require Host-local review and may affect other consumers during maintenance. Current v1's explicit fresh scoped architecture supports ordinary authenticated MCP grant/revoke without repeated Host approval and preserves unrelated flows under its verified native conditions. RC13 is not scoped.

A TCP/UDP forward maps a host listener to an owned Box port. Default host bind is loopback; external binds require explicit acknowledgement and operator-approved native/service policy. Native device verification is not a guest listener, LAN or public IPv6 test. Test the actual application from the intended client.

## If work is interrupted

Recover original IDs via operation_list or box_activity_get (overview/issues/evidence). Read operation_get and, for Exec, operation_output_get. Do not submit the original action under a new key merely because a request timed out.

Use operation_recover inspect for an UNKNOWN original; use reconcile only for a supported authorized settlement with its own key. Retain evidence when blocked. On builds containing #177, the local [Doctor and Recover guide](../process/DOCTOR_RECOVER.md) describes additional evidence-bound diagnosis and reviewed recovery, not a general remote Host repair API.

See [README](../../README.md) and [developer documentation](../INDEX.md).
