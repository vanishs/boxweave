# agent-box MCP — Box-scoped candidate mapping

**Status: RESEARCH / PROPOSAL ONLY — NOT APPROVED FOR IMPLEMENTATION; NOT DEPLOYED.**

**Evidence:** static inspection (S) of BoxWeave `main@a55926fcea55fdd3e2219900c787b90e49cb5e19` (VERSION 0.21.4) and Containarium `main@f37358e631282ad90bd534796667af8d0a56b72c` on 2026-10-31. Prior runtime documents (D) are historical, not retested. No live MCP, Box, client, Provider or CI execution was performed for this research.


## Status, naming and routing

Source: [agent-box registry](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/agentbox/server.go), [entrypoint](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/cmd/agent-box/main.go), [file root policy](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/agentbox/sandbox.go).

All listed tools exist **in upstream source**, not in the reviewed BoxWeave `bwmini` application adapter. Candidate BoxWeave names below are **illustrative**, not deployed tool names or permission grants. The logical per-Box route needs exact provider/backend/project/logical/immutable instance identity before a verified transport is opened. **Never select target by caller-controlled hostname or SSH command**.

| agent-box tool | Candidate BoxWeave name | Actual upstream purpose | Proposed treatment |
| --- | --- | --- | --- |
| `shell_exec` | `boxweave_box_shell_exec` | Run a bounded command in a Box; stdout/stderr/exit status | Execution; M4 |
| `read_file` | `boxweave_box_read_file` | Read an in-Box file window | Read; M3 |
| `write_file` | `boxweave_box_write_file` | Atomically write a file in a Box | Write; M4 |
| `list_directory` | `boxweave_box_list_directory` | List directory entries in a Box | Read; M3 |
| `move_file` | `boxweave_box_move_file` | Move/rename a file or directory | Write; M4 |
| `delete_file` | `boxweave_box_delete_file` | Delete a file, not a directory | Destructive write; M4 |
| `tail_log` | `boxweave_box_tail_log` | Read a bounded log-tail window | Read, potentially sensitive; M3 |
| `process_start` | `boxweave_box_process_start` | Start a long-running in-Box process | Execution; M4 |
| `process_list` | `boxweave_box_process_list` | List managed in-Box processes | Read, potentially sensitive; M3 |
| `process_kill` | `boxweave_box_process_kill` | Stop a managed process | Execution/destructive; M4 |
| `compose_discover` | `boxweave_box_compose_discover` | Find in-Box Compose stacks | Read; M3/M5 |
| `compose_autostart_enable` | `boxweave_box_compose_autostart_enable` | Create/enable in-Box user autostart | Service mutation; M4/M5 |
| `compose_autostart_disable` | `boxweave_box_compose_autostart_disable` | Disable in-Box user autostart | Service mutation; M4/M5 |
| `compose_autostart_status` | `boxweave_box_compose_autostart_status` | Read per-stack autostart state | Read; M3/M5 |

### MCP Resources: separate resource protocol

| Upstream resource | Meaning | BoxWeave design |
| --- | --- | --- |
| `containarium://ci-context` | CI PR/commit/test context or unavailable stub | Scoped `resources/list`/`resources/read` by exact selected Box; do not invent a successful CI record |
| `containarium://ci-prompt` | Static CI debugging playbook | Treat as read-only *resource*, not a tool or human approval |

A direct proxy cannot simply reuse the same unqualified `containarium://...` URI across multiple Box sessions: represent the selected Box safely in the BoxWeave layer's resource namespace, and retain a mapping to the original URI. Do not rewrite URIs and report support before client compatibility is tested.

### Distinct third server: internal approvals

`agent-box approval-mcp` starts a **run-internal** MCP server named `containarium-approval` and publishes a single `approve` tool. It supports signed approval for `code run --approvals ask`; it is **not** a permanent cross-Box Agent-facing MCP. BoxWeave may observe task approval status through the approved Platform run interfaces (e.g. `code_pending`, `code_answer`), preserving run-bound signatures and upstream authorization. Do not install or proxy `approve` as a general-purpose BoxWeave tool. Source: [approval.go](https://github.com/FootprintAI/Containarium/blob/f37358e631282ad90bd534796667af8d0a56b72c/internal/agentbox/approval.go).

### Required trust/compatibility gates

1. Only an **already registered exact Box**; verify immutable identity immediately before connection and before each mutation. Check running/readiness state separately.
2. Authenticate with an independently approved per-Box transport and strict host key; do not reuse direct Agent transport private keys as workload credentials or reveal private key text.
3. Probe existence/version of `agent-box` **without installing or executing untrusted software**; validate MCP initialize, tool schemas, supported resources and result envelopes on actual intended Box.
4. File operation root restrictions apply only to file ops; `AGENTBOX_ROOT` and client MCP Roots do not sandbox `shell_exec`. When both roots are absent, upstream file ops may also be unrestricted. BoxWeave must enforce its own access policy and use the OS/container boundary for execution.
5. Explicitly distinguish `permission denied`, `tool unsupported`, `agent-box not installed`, `Box stopped`, `trust mismatch`, `target changed` and `unknown result`. Never silently fall back to a privileged host shell.
6. Limit output, command duration, log exposure and concurrent processes; never assume a background process completed merely because `process_start` acknowledged it.

**Backlinks:** [master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [security](MCP_INTEGRATION_SECURITY.md), [M3](mcp-integration/M3_AGENT_BOX_READONLY.md), [M4](mcp-integration/M4_AGENT_BOX_EXECUTION.md), [test plan](../verification/MCP_INTEGRATION_TEST_PLAN.md). Existing [diagnostics](DIAGNOSTICS_PROPOSAL.md) and [environment](BOX_ENVIRONMENT_PROPOSAL.md) remain independent **NOT IMPLEMENTED** proposals.
