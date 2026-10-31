# BoxWeave M0–M5 MCP source implementation and deferred live acceptance

**2026-10-31 status: SOURCE CANDIDATE; LIVE PROVIDER/BOX NOT TESTED.**
User authorized completing source code and fixture/CI tests without entering a real Box.
This is not a certified working agent-box deployment, installer release, or per-client RBAC.
Parent: [MCP capability master](../design/MCP_CAPABILITY_INTEGRATION_MASTER.md),
[security boundaries](../design/MCP_INTEGRATION_SECURITY.md),
[acceptance plan](../verification/MCP_INTEGRATION_TEST_PLAN.md),
[previous handoff](MCP_REUSE_IMPLEMENTATION_HANDOFF.md).

## Scope and implementation status

| Slice | Source added in PR #9 | Remaining real-world gate |
| --- | --- | --- |
| **M0** | Application MCP client now supports bounded MCP `resources/list` and `resources/read`; Platform discovery from 0.23.0 retained; per-Box agent-box tool/resource capability query | Installed Provider/tool scope, agent-box version and protocol negotiation on an authorized real Box |
| **M1** | Existing direct Agent gateway optionally reads a fixed, operator-owned `/etc/boxweave/mcp-tool-policy.json`; it filters all `tools/list` pages and rejects unlisted `tools/call` *before forwarding* | Full enrolled-client identity-to-Box/tenant RBAC is NOT implemented; missing file intentionally retains legacy broad trusted-operator passthrough |
| **M2** | Typed `toggle_auto_sleep` tool validation; action uses BoxWeave plan/confirm/Worker receipts and no automatic retry | Backend support, actual idle state and wake/recovery behavior are NOT tested; an upstream acknowledgement is not proof of successful automatic resume |
| **M3** | `BoxAgent` maps exact Box identity to a separately configured launcher; read-only tools/resources are mapped, bounded and path constrained | No automatic SSH/agent-box installation, no live host-key/root confinement or resource validation |
| **M4** | SQLite schema v11 plans/receipts, one-use confirmation, Worker single dispatch, restart recovery to unknown, no raw shell outputs persisted | OS-level confinement, process restart semantics, remote timeouts and real file/process execution untested |
| **M5** | Authenticated Web routes for capability reads, scoped file reads, resources, action plans/confirmations/receipts; MCP React panel showing selected capabilities and plan-first operations | No complete team RBAC, generalized coding-run approval bridge, automatic task scheduler, or Web forms for every upstream tool |

Existing D0–D2 diagnostics are independently implemented in v0.22.0, not reproduced here. E0–E4 environment snapshots/clones are separate, NOT IMPLEMENTED; database `create_backup` is not a snapshot.

## Compatibility and security boundaries

- **Direct-Agent gateway:** absence of the protected policy file retains the **legacy** broad upstream tool relay. This must not be called least-privilege security. To opt in, a trusted operator writes a root-owned regular policy file under a root-owned, non-writable `/etc/boxweave` directory. The policy file must be mode 0600, single hard link, <=16 KiB, with exactly `{"version":1,"allowed_tools":[...]}`. Its list must include any existing tools the operator still wants, including `boxweave_nesting` if used. Policy is loaded once at gateway process start, not hot-reloaded; rejected calls never reach the Provider. Unknown tools are blocked only when the allowlist is enabled. Never auto-enable without an operator migration decision.
- **Application:** Web login and CSRF remain the local operator boundary. The Web Provider and the direct SSH gateway are different principals; the policy above does not grant Web access, and Web settings do not restrict legacy gateway tools.
- **Box adapter:** per-Box `agent_box` configuration must point to an already reviewed, pinned-SSH or equivalent trusted launcher installed on the **app host**, not a browser-supplied address. Any command to provision SSH credentials, new users, a Box or its `agent-box` binary is outside this PR. The adapter checks the immutable `Target.digest()` against the operator-reviewed configuration at connection time and again before work.
- **Root/workspace limitation:** all supported file, log and Compose paths must lexically reside within the configured workspace. The **trusted launcher must enforce** `AGENTBOX_ROOT` within the real Box, including symlink canonicalization at the remote end. Because `shell_exec` is not confined by `AGENTBOX_ROOT`, the remote non-root OS user and LXC boundary remain the actual command sandbox. Tests without a real Box cannot prove this.
- **Writes:** action plans hold arguments in the application private SQLite state and expire in 120 seconds; confirmation consumes the plan once, queues an action, and Worker marks dispatch intent before calling upstream. A crash, timeout, lost response or abnormal tool result becomes **UNKNOWN**, never automatically replays or deletes any Box. Receipts omit shell output, secrets and file content. No generic `tools/call` HTTP endpoint exists.
- **Auto-sleep:** only the reviewed Platform `toggle_auto_sleep` operation is supported, with an explicit boolean and idle threshold from 5–1440 minutes. Automatic wake is not guaranteed by this tool. State should be observed out of band before trusting sleep/resume.
- **Release:** source is v0.23.0 (unreleased at the reviewed baseline); `PUBLISHED_RELEASE` is v0.22.0. Default streamed installs use the published release, not these unshipped source changes.

## Example operator-managed application configuration (illustrative only)

```json
{
  "agent_box": {
    "my-box": {
      "target_digest": "<digest copied from a separately verified BoxWeave Target>",
      "workspace": "/home/my-box/project",
      "launcher": ["/absolute/path/to/operator-reviewed-pinned-ssh-agentbox-launcher"],
      "allowed_tools": ["read_file", "list_directory", "tail_log",
                        "process_list", "compose_discover",
                        "compose_autostart_status"]
    }
  }
}
```

This fragment is added to an existing, separately approved `mode=mcp` application config; it is **not** a complete config, install command, authorization, key material or real executable path. Start with read-only tools. Later write/exec tools must be explicitly added to `allowed_tools` and remain subject to one-use plans. The launcher must be a protected regular file owned by root or the application user and not writable by group/others. Do not put passwords, JWTs, private SSH keys or arbitrary hosts in caller-supplied parameters.

## Web/API behavior

- `GET /api/mcp-capabilities`: app-side Platform catalogue (existing implementation).
- `GET /api/boxes/LOGICAL/debug-summary`: redacted, typed Platform debug evidence (existing implementation).
- `GET /api/boxes/LOGICAL/agent-box`: exact Box capabilities and resource identifiers.
- `POST /api/boxes/LOGICAL/agent-box/read`: only explicitly approved read tools (body: `tool`, `arguments`).
- `GET /api/boxes/LOGICAL/agent-box/resource?uri=containarium://ci-context`: exact allowed Box/resource.
- `POST /api/box-action-plans`: action and arguments (no tool executed).
- `POST /api/box-action-plans/PLAN/confirm`: exact logical name and `confirm: true`; queues once.
- `GET /api/box-action-receipts` and `GET /api/box-action-receipts/ID`: no raw sensitive outputs.
- The **MCP integrations** Web tab is optional for Agent clients; it never changes the original BoxWeave direct MCP enrollment.

## Validation and decisions

The accepted validation boundary for this task is **source tests + GitHub fixture CI only**; no live Box tests were authorized. A future integration Agent must record exact versions, caller/token/target trust, real `tools/list` and `resources/list`, test no-permission/no-Box/changed-host-key conditions and prove real bounded reads and writes. A fixture green result is not a live pass. See [acceptance plan](../verification/MCP_INTEGRATION_TEST_PLAN.md).

**Open:** whether to make allowlist mandatory for future restricted clients while retaining a legacy operator profile; separate actor-specific RBAC; agent-box launcher format/install; verified OS sandbox and root enforcement; real auto-sleep recovery, long-running process persistence and approvals; any official release/publication. None is silently granted here.

**Repository changes only:** this work does not modify hosts, Provider, Box credentials, SSH keys, routes or live services.
