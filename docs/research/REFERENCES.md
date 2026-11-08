# Upstream research and borrowed design patterns

References are **technical evidence and inspiration**, not dependencies or instructions to copy upstream code unreviewed. No licensing conclusion about copying source is implied.

## Existing BoxWeave main

- https://github.com/vanishs/boxweave/blob/main/AGENTS.md
- https://github.com/vanishs/boxweave/blob/main/docs/process/AGENT_TEST_WORKFLOW.md
- https://github.com/vanishs/boxweave/blob/main/docs/process/FULL_PROJECT_REACCEPTANCE.md

Retain English artifacts, requester-language replies, explicit privileges, source-vs-installed evidence, UNKNOWN preservation, black-box bug hunting, separate Finder/Fixer/Verifier and concise handoff. Do **not** import Containarium-specific procedures, old code, SSH client matrices, Web UI rules, stale host/Box names, or legacy release assumptions.

**Owner's retrospective (2026-11-07):** Reliability problems appeared to intensify after MCP→REST migration. Treat this as a high-priority **anti-pattern investigation**, not as proof of REST-protocol causation or an endorsement of old MCP relaying. [Pinned two-dimensional `main` source/814-commit-history review](MAIN_HISTORY_LESSONS_2026-11-07.md#owner-retrospective-and-pinned-v0504-re-review-2026-11-07) identifies multiple receipt models, an actual resize success/UNKNOWN semantic split, REST-500/404 repair chains, permanent-payload idempotency mistakes and test-harness false evidence. Prefer **eliminating duplicated boundaries** through one native Incus Go SDK Operation path to importing the old downstream fixes.

## nikitatsym/incus-mcp

https://github.com/nikitatsym/incus-mcp

Borrow omitted-vs-null parameter semantics, schema conformance test patterns, post-terminal native readback, and real Incus integration tests. **Its write verifier checks returned key presence, not full value equality; do not call that sufficient postcondition proof.** Its in-process waiter/pending-verify maps are not durable across restarts. Keep only v1's already-approved tools and responsibilities. [Pinned source assessment](INCUS_MCP_IMPLEMENTATIONS_2026-11-07.md).

## fredmj/incus-mcp-server

https://github.com/fredmj/incus-mcp-server

Borrow typed official MCP Go SDK tool registration, small Go handler layout, injectable fake Incus client/MCP-session tests and create-before-start separation. Its Incus **v6** dependency, synchronous waits, name-only deletion checks, raw native config, default Incus CLI remote credentials and unbounded Exec buffers are **not** v1 patterns. v1 uses reviewed **Incus v7**. [Pinned source assessment](INCUS_MCP_IMPLEMENTATIONS_2026-11-07.md).

## sw2m/pyincusd

https://github.com/sw2m/pyincusd

Learn from OpenAPI generation and precise API version alignment. The Python client is community maintained and not the official Incus Go client. Generated endpoint coverage alone does not establish websocket Exec or native operation reliability.

## FootprintAI/Containarium

- https://github.com/FootprintAI/Containarium/blob/main/CLAUDE.md
- https://github.com/FootprintAI/Containarium/blob/main/CONTRIBUTING.md
- https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/op_wait.go
- https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/acl.go
- https://github.com/FootprintAI/Containarium/blob/main/internal/app/route_sync.go

Borrow bounded and typed operation timeout handling, ETag and owned ACL edits, route desired-state reconciliation, strong typed models, supply-chain review, public deployment metadata anonymization, strict PR scope and meaningful test gates. Containarium additionally has a separate in-guest `agent-box` MCP server, but its native `pkg/core/incus/client.go` performs container Exec through the Incus SDK; **do not require guest agent-box or SSH in BoxWeave P0**.

**Reject as v1 architecture:** Containarium as a runtime dependency, CLI-first platform APIs, protobuf/gRPC/REST gateway as a public management contract, separate in-Box SSH MCP, multi-host/tenant infrastructure, and PostgreSQL dependency for single-host state. Its Platform MCP handles protocol itself; its Agent Box MCP uses third-party `mark3labs/mcp-go`. V1 favors an official MCP SDK if Go is selected.

For a **fixed source, commit-by-commit, non-template assessment** (2,577 upstream commit subjects, representative diffs, 21 bounded incident lessons and test links), see [Containarium `main` and history study](CONTAINARIUM_MAIN_HISTORY_2026-11-07.md). Upstream references are **examples**, not proof of v1 behavior.

## Waishnav/devspace — external development/verification evidence

https://github.com/Waishnav/devspace

Its **current source** illustrates local MCP workspace execution, workspace-scoped identity, SQLite dispatch records, explicit cancellation/stop distinction, interrupted-workflow recovery, native Git review checkpoints, and isolated real-MCP/CLI plus record/replay tests. Its **1,260-subject commit history** supplies real counterexamples: delegated read-only authority escapes, unsafe provider extensions, cleanup followed by failed persistence and failed compensation, daemon/reply races, incompatible migrations and tests that could not fail. These are engineering evidence, **not** v1 native PASS or requirements to add an Agent service. Keep all verification orchestration in repository/CI/disposable Boxes, not the standard BoxWeave runtime. [Pinned two-dimensional source and commit assessment](DEVSPACE_MAIN_HISTORY_2026-11-07.md).

**Do not import:** DevSpace's JavaScript/Node runtime, multi-provider adapters, OAuth web authorization, local host Shell, Git worktree API, UI, workflow scripting, resident Agent daemon or best-effort activity journal as a replacement for BoxWeave's mandatory Operation/audit. Existing 20 MCP tools and product boundary remain unchanged.

## Incus / MCP originals



- https://github.com/lxc/incus/tree/main/client
- https://linuxcontainers.org/incus/docs/main/rest-api/
- https://github.com/modelcontextprotocol/go-sdk
- https://modelcontextprotocol.io/specification

Pin actual versions and API extension requirements during implementation. Upstream docs and source are compatibility references, not executable instructions.

## Agent policy reference

Containarium has `CLAUDE.md` and `CONTRIBUTING.md`. The two smaller Incus MCP repositories and pyincusd showed no repository-root `AGENTS.md` / `CLAUDE.md` in the inspected main trees; their README/test practice was evaluated instead. Keep v1 rules model-neutral in `AGENTS.md` and the detailed runbook separate.

## Permanent product narrowing (owner-approved)

Prior Containarium Caddy reconciliation, generalized ACL, Web/UI, OAuth, snapshot and resident agent patterns are **historical research only**, not features to implement or defer. The narrow product keeps only native Incus TCP/UDP forwards, a simple owned directional peer allow/revoke and the Agentless development loop. See [permanent boundary](../decisions/PRODUCT_BOUNDARY.md).

## Official Go SDK research (2026-11-07)

- [Incus and MCP Go SDK capabilities, source history and architectural fit](OFFICIAL_SDK_FIT_2026-11-07.md) — S01–S20 with the corrected local Incus privilege model.
- [Every v1 public MCP tool mapped to official Incus/MCP SDK methods](SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) — 20/20 tools, native implementations versus BoxWeave-specific policy, missing guarantees and practical test gates.

- [Second-pass exact SDK contract gaps](SDK_GAP_CLOSURE_2026-11-07.md) — MCP Tool per-request identity (distinct from HTTP context), Bridge ACL selector/anti-spoof limits, native file creation truncation and CAS, blocking Exec output modes, operation-wait transport behavior, Project features/root Profile ordering; source links and historical changes included.
