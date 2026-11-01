# Remote MCP implementation handoff — Agent execution contract

**Status: v0.24.0 source implementation committed to PR #10; no public endpoint, live Box, client OAuth or TLS deployment tested.** Design owner: [master](../design/REMOTE_MCP_MASTER.md), decisions [ADR-009](../design/ADR/0009-remote-mcp-transports-and-auth.md), verification owner [remote MCP test plan](../verification/REMOTE_MCP_TEST_PLAN.md).

## Confirmed baseline

As reviewed 2026-10-31: main `9da1340466ac84c6a34ac9ebca97c16ebce3b587`, `VERSION` 0.23.0, `PUBLISHED_RELEASE` 0.22.0, SQLite v11; legacy gateway `bwmini/mcp_gate.py`, outbound app client `bwmini/mcp.py`, Web admin `bwmini/web.py`. Pin exact main and install revisions **again** before implementation; branch may advance. Source merge does not upgrade the retained installed copy.

## Execute in order

| Task | Prerequisites | Required source changes | Minimum evidence |
| --- | --- | --- | --- |
| REMOTE-R0 | existing SSH fixtures | Extract shared gateway dispatcher, preserve transforms and opt-in policy | SSH-before/after identical tools and errors, negative dispatch fixtures |
| REMOTE-R1 | R0 | New loopback Streamable HTTP server/entrypoint, independent status | Inspector handshake, tools/resources, content negotiation/SSE, Origin/session negative cases |
| REMOTE-R2 | R1 | Single-admin OAuth AS + resource metadata + API Bearer + path capability, private state migration | PKCE, redirect/SSRF/token/audience negatives; actual OAuth/Bearer/path results |
| REMOTE-R3 | R1-R2 | explicit external base URL, optional reviewed TLS automation, manual proxy support | independent TLS and proxy tests, failure/rollback, no accidental exposure |
| REMOTE-R4 | R2-R3 | authenticated Web UX, docs, connection diagnostics, no SSH regression | actual client onboarding/revocation, Web UI tests, legacy profiles intact |

Follow [R0](../design/remote-mcp/R0_PROTOCOL_AND_DISPATCH.md) → [R1](../design/remote-mcp/R1_STREAMABLE_HTTP.md) → [R2](../design/remote-mcp/R2_AUTH_AND_CREDENTIALS.md) → [R3](../design/remote-mcp/R3_HTTPS_AND_PROXY.md) → [R4](../design/remote-mcp/R4_WEB_CLIENT_EXPERIENCE.md). R3 and R4 may be split into independent code reviews after R2 but not claimed working before their own gates.

## Scope and explicit prohibitions

- Implement **single trusted operator** MCP, not multiuser role/Box RBAC; keep broad-credential risk documented and existing opt-in SSH tool allowlist semantics unchanged until an owner-approved joint SSH+HTTP policy migration.
- Preserve working SSH and Windows/WSL client setup; no forced migration or credential copying.
- No third-party tunnel vendor integration, no host Incus/Provider bypass, no arbitrary Web raw-shell route, no sharing of upstream JWT with browsers or HTTP clients.
- Never run installer/port opening/public TLS/firewall/Provider/Box modifications merely because this handoff exists. Those need specific live environment authorization.
- Do not expose `None` as anonymous network access: a capability URL itself is a bearer credential and must be checked. Never place OAuth access tokens in query parameters.
- Client capability/Plus subscription support is an observation to test, not a guaranteed requirement of BoxWeave.

## Minimal validation/evidence procedure per phase

1. Record current source SHA, branch, reviewed diff, artifact version, environment/target identities and explicit authorization.
2. Run targeted fake/fixture tests and `python scripts/check_knowledge.py`; then prepared LXC `.venv/bin/python scripts/validate.py` as applicable. Record **actual** exit codes, not planned ones.
3. For any installed-surface claim, require the separate authorized live Box checkpoint in [Agent test workflow](AGENT_TEST_WORKFLOW.md); check no-permission, incorrect Target and unsupported clients.
4. Run R1 read-only remote protocol observation first. For R2, prove failure modes before any write. For R3/4, independently observe external HTTPS/redirect/real client loads and revocation. Do not claim live result from fixture.
5. Save checks and pending items in [REMEDIATION_TEST_QUEUE](../verification/REMEDIATION_TEST_QUEUE.md), maintain normative REQ/AC traceability, risk and release records. Set result PARTIAL/UNKNOWN when a stage lacks evidence.
6. Preserve `unknown` external operations on network loss: do not auto-retry create/delete/route or repair lost host trust.
7. Each phase documents changed files, exact CI/test command, outcome and rollback boundary before phase completion. Do not merge/deploy under the authority of this planning record.

## Defer to separate work, not an implicit missing phase

- Unified **SSH + HTTP** Agent identity, tool/Box policy and per-client scopes.
- User/team RBAC, tenant accounts, billing.
- OpenAI Tunnel and Cloudflare Tunnel integration/operations.
- Additional URL credential expiry tiers or token policy UI, beyond minimum validation/revocation.
- Real public-host deployment and privileged firewall/DNS changes until specifically approved.

Upon completion, update [master status](../design/REMOTE_MCP_MASTER.md), [architecture authority](../design/ARCHITECTURE.md), [phase status](PHASE_STATUS.md), and [OPEN_RISKS](../verification/OPEN_RISKS.md) with precise tested vs untested boundaries.
