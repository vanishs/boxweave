# BoxWeave MCP guide for AI coding agents

This guide describes source version 0.24.5, not a live installation. Available
tools depend on the installed Containarium Provider, authenticated scope,
release version and any enabled host-wide MCP tool policy.

## Discover BoxWeave capabilities

SSH/stdio and optional authenticated Streamable HTTP MCP initialization return
BoxWeave-specific identity and agent instructions. The read-only
**boxweave_info** tool exposes architectural and development guidance without
reading current host state, private configuration, SSH keys or credentials.

| topic | Returned section |
| --- | --- |
| all (default) | All six sections |
| overview | Product purpose and components |
| architecture | MCP/Web/Provider/Box relationship |
| development | Source workflow and tests |
| capabilities | Example tools and runtime discovery |
| security | Identity, permissions and risky mutations |
| documentation | Relevant source paths |

Example: invoke **boxweave_info** with topic set to **development** when
developing BoxWeave itself. Use MCP **tools/list** for actual authorized tool
names and schemas, and **list_containers**, **get_container**,
**get_system_info** and similar read tools for current runtime information.
The guide is not a live diagnostics API or an execution tool.

## Development workflow

First read [development guidance](DEVELOPMENT.md) and
[Agent test workflow](process/AGENT_TEST_WORKFLOW.md). Prefer an existing,
authorized isolated Box; execute appropriate tests through connect with exec,
and check actual exit codes/output. If SSH fails, use debug_container rather
than recreating a Box. If a mutation times out, the result is UNKNOWN: observe
the exact state before considering a retry.

The gateway injects its own managed workload SSH public key when creating
Boxes. It rejects host/user overrides that violate exact-target SSH trust.
The optional root-owned **allowed_tools** policy also filters this new tool:
include **boxweave_info** explicitly when using an allowlist. The policy is
host-wide, not full per-Agent/per-Box RBAC. Application-side agent-box adapters
and Web action plans are separate from the direct-Agent MCP gateway.

Read [remote HTTP MCP](REMOTE_MCP.md) and
[open risks](verification/OPEN_RISKS.md) for current boundaries.
\n\n## D3/D4: failure evidence and safe disclosure\n\nAfter a failed or ambiguous call, query `boxweave_diagnose` with `{"source":"gateway","recent":5}`; for app receipts use `{"source":"app","operation_kind":"environment","operation_id":"EXACT-ID"}`. Unknown write outcome means inspect the original resource before retry, not automatic replay. `boxweave_project_diagnose` only reads predefined summaries in administrator-allowlisted exact Box incarnations; absent or denied grants are not a reason to elevate privilege or use a different unrestricted tool. Agent workflow differs by task, not by diagnostic depth: ordinary users receive explanations/private issue drafts, while authorized BoxWeave developers may investigate source. **Never send any report or Issue externally until the user personally reviews every field/attachment for sensitive data and separately approves submission.**\n