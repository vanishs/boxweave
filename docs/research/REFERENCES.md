# Upstream research and borrowed design patterns

References are **technical evidence and inspiration**, not dependencies or instructions to copy upstream code unreviewed. No licensing conclusion about copying source is implied.

## Existing BoxWeave main

- https://github.com/vanishs/boxweave/blob/main/AGENTS.md
- https://github.com/vanishs/boxweave/blob/main/docs/process/AGENT_TEST_WORKFLOW.md
- https://github.com/vanishs/boxweave/blob/main/docs/process/FULL_PROJECT_REACCEPTANCE.md

Retain English artifacts, requester-language replies, explicit privileges, source-vs-installed evidence, UNKNOWN preservation, black-box bug hunting, separate Finder/Fixer/Verifier and concise handoff. Do **not** import Containarium-specific procedures, old code, SSH client matrices, Web UI rules, stale host/Box names, or legacy release assumptions.

## nikitatsym/incus-mcp

https://github.com/nikitatsym/incus-mcp

Borrow write-after-read validation, explicit omitted-vs-null parameter semantics, non-blocking native operation waiting, API schema checks and real Incus integration tests. Its **in-process** waiter registry is not a substitute for v1 durable cross-restart operation journaling.

## fredmj/incus-mcp-server

https://github.com/fredmj/incus-mcp-server

Borrow scoped destructive confirmation, narrow MCP tool schemas, read/write safety separation and official Incus Go SDK usage. It uses `github.com/modelcontextprotocol/go-sdk`. Do not adopt synchronous mutation waiting or assume the same version/features of Incus SDK support v1 untouched.

## sw2m/pyincusd

https://github.com/sw2m/pyincusd

Learn from OpenAPI generation and precise API version alignment. The Python client is community maintained and not the official Incus Go client. Generated endpoint coverage alone does not establish websocket Exec or native operation reliability.

## FootprintAI/Containarium

- https://github.com/FootprintAI/Containarium/blob/main/CLAUDE.md
- https://github.com/FootprintAI/Containarium/blob/main/CONTRIBUTING.md
- https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/op_wait.go
- https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/acl.go
- https://github.com/FootprintAI/Containarium/blob/main/internal/app/route_sync.go

Borrow bounded and typed operation timeout handling, ETag and owned ACL edits, route desired-state reconciliation, strong typed models, supply-chain review, public deployment metadata anonymization, strict PR scope and meaningful test gates.

**Reject as v1 architecture:** Containarium as a runtime dependency, CLI-first platform APIs, protobuf/gRPC/REST gateway as a public management contract, separate in-Box SSH MCP, multi-host/tenant infrastructure, and PostgreSQL dependency for single-host state. Its Platform MCP handles protocol itself; its Agent Box MCP uses third-party `mark3labs/mcp-go`. V1 favors an official MCP SDK if Go is selected.

## Incus / MCP originals

- https://github.com/lxc/incus/tree/main/client
- https://linuxcontainers.org/incus/docs/main/rest-api/
- https://github.com/modelcontextprotocol/go-sdk
- https://modelcontextprotocol.io/specification

Pin actual versions and API extension requirements during implementation. Upstream docs and source are compatibility references, not executable instructions.

## Agent policy reference

Containarium has `CLAUDE.md` and `CONTRIBUTING.md`. The two smaller Incus MCP repositories and pyincusd showed no repository-root `AGENTS.md` / `CLAUDE.md` in the inspected main trees; their README/test practice was evaluated instead. Keep v1 rules model-neutral in `AGENTS.md` and the detailed runbook separate.
