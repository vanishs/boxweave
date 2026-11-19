# BoxWeave Python loopback MCP test kit (2026-11-19)

**Status:** developer test tooling only, not an additional MCP Tool, native acceptance standard, OAuth browser client, real Agent session or release package. Located at `scripts/mcp_testkit.py` (importable library), `scripts/mcp_lab.py` (CLI), `scripts/test_mcp_testkit.py` (wire-level unit tests) and `scripts/examples/mcp_activity_smoke.py` (copyable Python example).

## Why this architecture

- **Zero Python dependencies**, standard-library `http.client`, pinned to **Streamable HTTP MCP 2025-11-25** (the project's Go SDK protocol). Each request is an individual JSON-RPC HTTP POST. Performs `initialize`, `notifications/initialized`, `tools/list` and `tools/call`. Handles ordinary JSONResponse, finite bounded SSE body and an optional `Mcp-Session-Id`. Does not claim full MCP SDK conformance, infinite SSE support or a modern 2026 transport.
- **Loopback-only by construction:** `127.0.0.1`, `[::1]` or `localhost` dialed as numeric IPv4; explicit port and `/mcp`; refuses remote hosts, URL credentials, redirects, query strings, other protocols, ambient proxies and unbounded bodies. Always authenticates with a Bearer token read from `BOXWEAVE_MCP_TOKEN` or an owner-only file. Credentials never appear in CLI arguments or JSON reports.
- **Read-only default:** the nine known BoxWeave mutation Tools and unknown/unannotated Tools are blocked before any `tools/call`. An explicitly isolated fixture can opt in via library `allow_mutations=True`; CLI requires *both* `--allow-mutations` and `--ack-disposable`. The library never automatically retries lost replies, replays UNKNOWN or decides a native effect succeeded.
- Built-in **typed thin helpers**: `host_status`, `operation_list`, `operation_get`, `operation_output`, `activity(overview/issues/evidence)`, bounded `walk_operations`, `walk_activity`, read-only `poll_operation`, and `recover_work(work_ref)` (same-Layer originals, unresolved issues and UNKNOWN limitations). `FaultInjectingMCP` can **drop one processed read response** to test reconnection, but deliberately cannot inject a mutation ACK loss.
- `ProbeReport` stores test IDs, PASS/FAIL counts, duration and explicit evidence limits. It does **not** serialize tool arguments, reply bodies, access tokens, raw guest output or root journals. The JSON result is suitable for lightweight CI handoff, not proof of a native state change.

This favors a small reproducible transport against the pinned BoxWeave implementation; retain the project's independent official Go SDK + native runner and OAuth/ChatGPT acceptance. The Python SDK also offers a full-featured Streamable HTTP client, but its v2 transport now differs from the pinned 2025 contract, so this harness does not silently depend on or auto-negotiate future protocol versions. References: [MCP 2025-11-25 HTTP specification](https://github.com/modelcontextprotocol/modelcontextprotocol/blob/main/docs/specification/2025-11-25/basic/transports.mdx) and [official Python SDK transport guide](https://github.com/modelcontextprotocol/python-sdk/blob/main/docs/client/transports.md).

## Quick start — temporary local controller ONLY

From the repository root, after starting a **separate temporary Schema v5 controller** on `127.0.0.1:18650` using a fresh isolated SQLite directory and obtaining its test Bearer token:

```sh
export BOXWEAVE_MCP_TOKEN="$(cat /path/to/disposable/private-token)"
python3 scripts/mcp_lab.py --url http://127.0.0.1:18650/mcp inspect
python3 scripts/mcp_lab.py --url http://127.0.0.1:18650/mcp smoke --expected-tools 18 --expected-schema 5 --require-activity
python3 scripts/mcp_lab.py --url http://127.0.0.1:18650/mcp recover --work-ref test/activity-001 --max-items 100
python3 scripts/mcp_lab.py --url http://127.0.0.1:18650/mcp --report /tmp/mcp-smoke-report.json smoke --expected-tools 18
python3 scripts/examples/mcp_activity_smoke.py
```

Or use an owner-only token file, without exporting secrets:

```sh
chmod 600 /path/to/disposable/private-token
python3 scripts/mcp_lab.py --token-file /path/to/disposable/private-token inspect
```

**Never run `token-create`, restart, reset, migrate or replace the currently connected L0 Schema v4 controller just to run these examples.** The example does not provision your server. Do not paste tokens into shell history, logs, prompts, or checked-in files.

## Write a custom Python test

```python
import os, sys
sys.path.insert(0, "scripts")
from mcp_testkit import LoopbackMCP, ProbeReport, token_from_env

report = ProbeReport("my-private-smoke")
with LoopbackMCP("http://127.0.0.1:18650/mcp", token=token_from_env()) as mcp:
    report.check("required_tool", "box_activity_get" in mcp.all_tools())
    issues = list(mcp.walk_activity("issues", max_pages=10, work_ref="test/activity-001"))
    manifest = mcp.recover_work("test/activity-001")
    report.check("no_replay", manifest["safe_to_retry_unknown"] is False)
    for original in manifest["unknown_originals"]:
        receipt = mcp.operation_get(original)
        report.check("unknown_requires_host_review", receipt["next_action"]["kind"] == "HOST_LOCAL_REVIEW")
report.write("/tmp/my-private-smoke.json")
```

Tests can use `mcp.call("box_get", {"box_id": ...}).structured`, or `mcp.operation_output(...)` where appropriate. The library intentionally returns raw in-memory data to the calling Python test; your own script must not print or upload secrets, command output, file bytes, or private evidence. `mcp_lab.py call ...` prints *only* tool name, returned keys and tracking ID by default (no raw tool result).

For isolated **read-response loss** tests:

```python
from mcp_testkit import FaultInjectingMCP, MCPTransportError
with FaultInjectingMCP("http://127.0.0.1:18650/mcp", token=token_from_env()) as mcp:
    mcp.drop_next_read_reply = True
    try:
        mcp.activity("issues", work_ref="test/activity-001")
    except MCPTransportError:
        pass
    # Reconnect and query the original; do not issue a new mutation key.
```

## Developer test and scope

```sh
python3 -m unittest discover -s scripts -p 'test_mcp_testkit.py' -v
python3 -m py_compile scripts/mcp_testkit.py scripts/mcp_lab.py scripts/examples/mcp_activity_smoke.py
```

The unittest suite starts a disposable threaded loopback mock server and tests protocol handshake, strict IDs/content types, JSON/SSE, catalog pagination, access token handling, mutation refusal, recovery, read-response fault injection, report redaction, refusal of redirects and remote URLs, and CLI safety. A separate installed temporary BoxWeave v5 controller test (with a real Go binary and fresh local SQLite) is needed to prove actual HTTP compatibility; mock-only tests must never be presented as that proof.

### Explicitly outside scope

Actual OAuth DCR/browser refresh/revoke, independently initiated model/Agent planning, a real native Incus or Root operation with lost ACK and independent poststate, external HTTP/WAN TLS and production SQLite migration are **NOT_TESTED**. `attention_complete=false` and `UNKNOWN` remain explicit boundaries, not excuses to make an extra mutation. This library is not a host privilege or authorization bypass.
