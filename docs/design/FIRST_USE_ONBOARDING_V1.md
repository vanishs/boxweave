# BoxWeave First-Use Onboarding v1 — Source Audit and Development Design

**Status:** DESIGN ONLY — not implemented, not tested, not deployed. This document does not authorize host changes, Box writes, new public listeners, token access, installations, release publication or source feature implementation.
**Review date:** 2026-11-01. **Source baseline:** BoxWeave `main` v0.27.1, immediately before this documentation commit. The public default installer selects v0.27.1.
**Owner direction:** Improve installation and first authenticated MCP connection without altering BoxWeave's existing architecture. Boxes must remain independent, clean and free of mandatory in-Box agents or services. Git worktrees are not part of this proposal.
**Scope:** Human onboarding, noninteractive setup, first-client discovery, accurate readiness and safe repair guidance; retain separate, existing SSH/stdio and HTTP/OAuth MCP paths.

Related:
- [Existing first-use README](../../README.md), [install](../INSTALL.md), [installed CLI](../CLI.md), [client setup](../../clients/README.md), [remote HTTP MCP](../REMOTE_MCP.md).
- [HTTP MCP quick-start](HTTP_MCP_QUICKSTART.md), [remote onboarding R4](remote-mcp/R4_WEB_CLIENT_EXPERIENCE.md), [remote MCP master](REMOTE_MCP_MASTER.md).
- [Current verification queue](../verification/REMEDIATION_TEST_QUEUE.md), [onboarding historical acceptance](../verification/INSTALL_CLI_ONBOARDING_ACCEPTANCE.md), [agent-led tests](../process/AGENT_TEST_WORKFLOW.md).

## 1. Executive conclusion

The important missing product feature is **not another installer**. BoxWeave already has a source-bound streamed host installer, ordinary-user CLI, cross-platform generated SSH client scripts, independent Web console, protected loopback HTTP MCP, OAuth, and several diagnostic commands. The user's first connection remains more complex than necessary because these parts are presented as disconnected choices, and intermediate health checks can be mistaken for a client actually using BoxWeave.

**Design objective:** From successful verified installation to an actual client-visible, authenticated `list_containers` call through one selected route, with a short and truthful next action whenever that outcome cannot be established.

This is a **first-use orchestrator for explanations and existing operations**, not a new background daemon, second transport, alternate Provider controller, installer rewrite or host privilege system.

## 2. Code-first inventory and exact evidence boundaries

| Surface | Actual present implementation | First-use limitation / opportunity |
| --- | --- | --- |
| `install.sh` + `bwmini/online.py` | Privileged streamed host install/update; GitHub Release manifest/tag/helper digests; existing Provider reuse; clear `ready` marker and postinstall banner | Banner prioritizes `boxweave client` even when a hosted ChatGPT user should select HTTPS/OAuth. Machine JSON and human prose coexist; user must choose their own path. A failed app phase requires exact error/phase, never a generic success message. |
| Release integrity and v0.27.1 | Current `VERSION`, README, stable installer pin = 0.27.1. This release fixes a package-relative import error that broke the v0.27.0 standalone application installer and improves private-log error classification | **Unverified on a real updated host at review time:** v0.27.1 install/repair, running managed remote-MCP user service, external OAuth callback. Source fix and published release are not live acceptance. Do not describe a host as upgraded based on source alone. |
| `bwmini/host_cli.py` | Normal-user `boxweave`, `version`, `status`, `doctor`, `client`; elevated `doctor --mcp`; `http status/doctor/configure/restart` | Too many entry points for first use. `status` / local HTTP reachability is not an authenticated client tool call. |
| `bwmini/client_setup.py` / `client_profiles.py` | Unified self-contained Python 3.11+ client script; nine alphabetically sorted Apps; Linux/macOS/Windows native SSH, Windows WSL SSH; deliberate public-key enrollment, strict host pin, actual initialize/tools/list/Box read probe before client registration | Cross-machine script transfer, Python/OpenSSH setup, edit-me host/admin details, separate sudo approval and real client restart remain. A probe run outside an IDE does not prove that IDE loaded or granted its new MCP tools. |
| `web/src/ClientSetup.tsx` | Legacy Web client form supports only four Windows Codex/ZCode profiles while the CLI supports nine Apps across OS profiles | Discoverability mismatch; should link to or reuse the same public generator contract instead of maintaining inconsistent setup stories. |
| `bwmini/remote_mcp.py`, `bwmini/http_manage.py`, HTTP service setup | Authenticated loopback HTTP MCP `127.0.0.1:18844`; opt-in `https://<actual-public-origin>/mcp`; OAuth with explicit operator consent, API token and credential-path alternatives; managed loopback user service in v0.27.x source | Hosted clients cannot use localhost. The operator must supply a real public HTTPS origin and configure a tunnel/proxy externally. A configured public URL, local listener or tunnel process is not proof of external reachability/authentication. |
| `bwmini/gate_diagnostics.py`, D3/D4 | Bounded classified gateway events and separate preauthorized project diagnostics | These diagnose failures, not broad install acceptance; avoid adding secret-bearing logs or creating an unrestricted guest repair tool. |
| Historical releases/tests | Selected prior real SSH MCP, app-side health, nested tests and older ChatGPT OAuth read have evidence | Client/OS modes, updated v0.27.1 on the actual host, public HTTPS on a production deployment and current real Agent tool injection remain separately unverified. |

**Observed source version and installed runtime must be reported separately.** A checked-in test that passes on v0.27.1 cannot establish the host's active v0.27.1 app process, service, Provider or client behavior.

## 3. External source and user-friction comparison

### DevSpace

DevSpace uses `devspace init` with **one question at a time**: ChatGPT, Coding Agents, or both. It skips inapplicable setup, supports noninteractive flags (`--yes`, `--public-url`, `--roots`), shows the right public URL/credential process, and provides a separate `doctor` command. Source: [setup](https://github.com/Waishnav/devspace/blob/main/docs/setup.md), [init implementation](https://github.com/Waishnav/devspace/blob/main/src/onboarding.ts), [CLI](https://github.com/Waishnav/devspace/blob/main/src/cli.ts).

**Transfer:** a short destination-based first-use guide, explicit copyable next commands, noninteractive machine outputs and a single success criterion. **Do not transfer:** DevSpace's local Node runtime, subagent provisioning, project-root model, worktrees or permission assumptions. BoxWeave is a Linux host/Provider control plane; privileged host provisioning is genuinely needed.

### Coder

Coder separates a short first-workspace quickstart from deeper production installation instructions. Its first-use story has an unambiguous progression: install, create/launch, connect through the user's editor. Source: [Coder quickstart](https://coder.com/docs/get-started) and [installation guide](https://coder.com/docs/install).

**Transfer:** show one concrete user outcome at a time and keep advanced host/network operations out of the shortest route. **Do not transfer:** Coder templates, workspace lifecycle, or a new Box abstraction.

### MCP transport and security

The MCP Streamable HTTP transport requires correct HTTP origin validation, local binding where appropriate and authentication; OAuth discovery/consent is distinct from tool readiness. These are preconditions to preserve, not shortcuts to bypass. Sources: [MCP transport spec](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports), [MCP authorization](https://modelcontextprotocol.io/specification/2025-06-18/basic/authorization).

## 4. Proposed product flow

### 4.1 One new guide command, not another installer

Candidate CLI: `boxweave setup` — an ordinary-user guide, delegating to **existing** commands. Do not launch root actions on behalf of an unprivileged caller.

```text
BoxWeave v0.X.Y    Server: ready / incomplete / not verified
Choose your first connection:
  1. Local coding client / IDE (SSH MCP)
  2. Hosted ChatGPT / remote MCP client (HTTPS MCP)
  3. Diagnose server or open local Web

Recommended next command: <one exact, supported command>
```

Interactive: ask only relevant questions. Noninteractive (candidate): `boxweave setup --mode ssh|https|local --check --format json`, refusing prompts when there is no TTY and not guessing credentials. This is an **illustrative interface**, not currently shipped.

The final install banner should point to `boxweave setup` **only once the command really exists**. Until then continue to display accurate existing `boxweave client` and `boxweave http` commands, not a dead command.

### 4.2 Local coding Agent / IDE: SSH route

1. Inspect installed public descriptor for actual server SSH host, port and pinned host key. Suggest a client-reachable server name only if it is observed/reviewed; never substitute `127.0.0.1` for a remote machine or silently trust a changed host key.
2. Reuse `boxweave client` for the supported **alphabetical** App catalog: Antigravity, Claude, Codex, Copilot, Cursor, OpenCode, Qoder, TRAE, ZCode. Ask App, host OS and native/WSL transport only where needed.
3. Produce the existing self-contained Python 3.11+ generator output or a protected new file. Print which machine it must run on, what prerequisites it needs and where it registers the client.
4. Clearly separate the human SSH administrator, `boxweave` service user and the Box's ordinary workload user. Only the **public** client key is enrolled by an expressly authorized administrator. Generation never requires sudo.
5. Confirm external-client SSH handshake, MCP initialize, tools/list, real `list_containers` read, and App registration. Prompt the user to restart the actual App when needed; verify that the App exposes the tools rather than assuming registration equals loading.
6. Keep legacy `--no-replace` behavior; backup and scoped replacement require reviewed user intent. No edits to other MCP entries or private key printouts.

If user cannot access a host administrator to enroll a public key, show the existing `--no-enroll` and `--already-enrolled` split; do not work around enrollment with a broad root login or guessed credential.

### 4.3 Hosted ChatGPT / remote client: HTTPS route

1. Observe whether managed HTTP MCP is running on `127.0.0.1:18844` and whether a usable public origin has been explicitly configured. If locally stopped/failed, offer the existing `boxweave http status` and `sudo boxweave http doctor`, never claim external reachability.
2. Explain that the local URL is **not** a hosted ChatGPT connector URL. An operator controls the external tunnel/proxy. The current documented Cloudflare Quick Tunnel example is optional and **not installed or started automatically**.
3. If an authorized operator starts a tunnel, capture the exact actual HTTPS **origin** and use the existing safe `sudo boxweave http configure --public-url https://ACTUAL-ORIGIN` operation. If that origin changes, explain OAuth issuer/resource rebinding and the need to reconfigure/reconnect.
4. Display the derived `https://ACTUAL-ORIGIN/mcp` as the client connection URL only after it has been explicitly configured. Do not fabricate a reachable hostname or print credential-bearing URLs/tokens in diagnostics.
5. Guide the user through existing OAuth admin login/consent (or approved client-supported API token flow). Do not silently approve, widen permissions, lower Origin restrictions or expose the loopback listener on `0.0.0.0`.
6. Distinguish local ready, external HTTPS reachable, OAuth accepted, MCP tools discovered and **actual hosted client's safe tool call**. An OAuth browser callback or HTTP 200 by itself is not full acceptance.

### 4.4 Server-only / Web route

Show the exact local Web URL, port and optional reviewed SSH port-forward command, then `boxweave status` or `doctor`. Server-local `127.0.0.1` and the client's own `127.0.0.1` are not the same network endpoint without a deliberate tunnel. Display any required first-admin password **path only**, never contents. No implicit new public Web route.

## 5. Readiness and failure-state contract

Adopt a stable small vocabulary:

| Stage | Acceptable proof | Not proof |
| --- | --- | --- |
| `release_installed` | Exact package identity and active service version | GitHub tag alone |
| `provider_ready` | Actually authorized Provider read | Installation config |
| `web_local` | Observed server-local HTTP endpoint | Public Web access |
| `http_mcp_local` | Authenticated local MCP handshake/read | Open TCP socket only |
| `ssh_mcp_authenticated` | Real client identity and tools/list/read | Generated client script |
| `https_external` | Independently observed real HTTPS route | Local listener / saved URL |
| `oauth_complete` | Real client authorization and accepted OAuth flow | Authorization page loaded |
| `client_tools_visible` | Selected real App invokes safe `list_containers` | Config file written |
| `box_execution` | Optional separately authorized exact Box command, stdout and exit code | MCP discovery only |

For each stage: `passed`, `failed`, `blocked`, `not_checked` or `not_verified`, plus `next_action` and evidence scope. Overall first MCP success requires a real permitted read; task/Box execution remains an optional separate milestone.

Provide concise reasons for: incomplete installer/app unit, port conflict, missing local MCP service, wrong SSH host pin, public-key not enrolled, client prerequisite missing, invalid public HTTPS origin, OAuth issuer/redirect mismatch, expired Quick Tunnel hostname, failed tools discovery, client App not reloaded, Provider denial, and Box SSH trust failure. Never automatically recreate Boxes or repeat uncertain mutations to turn failure into a green report.

## 6. Priorities and implementation batches

| Batch | Scope | Target value | Dependencies and limit |
| --- | --- | --- | --- |
| **ONB-0 (P0)** | Code-first baseline, define stable readiness evidence and exact v0.27.1 actual release/host checkpoint | Trustworthy initial status | No claim of successful host repair without a real separately authorized acceptance |
| **ONB-1 (P0)** | Simplify final banner, one explicit route selector / recommended next command; preserve machine JSON | Eliminate choice overload | Avoid modifying verified package installer internals beyond needed user-facing output |
| **ONB-2 (P1)** | Add `boxweave setup` thin ordinary-user guidance using existing `boxweave client` and `boxweave http` paths | One entry, conditional steps | No new generator, root escalation, tunnel launcher, OAuth service or Box runtime |
| **ONB-3 (P1)** | End-to-end check state machine and typed next-action diagnostics | Real first-read evidence, faster fault isolation | Do not confuse simulated/loopback probes with real App acceptance |
| **ONB-4 (P2)** | Web client entry consistency, CLI/README/setup documentation, noninteractive JSON and packaging regressions | Reduce duplicated/conflicting instructions | Preserve supported existing Apps and host permissions |

**Prerequisite:** owner authorizes source implementation separately. This design document alone cannot authorize a host upgrade, public listener, user enrollment or Box creation.

## 7. Acceptance matrix

| ID | Required scenario | Evidence / success criterion |
| --- | --- | --- |
| ONB-T01 | Fresh clean host install | Exact release manifest, installer exit 0, real app/worker/HTTP service versions and authorized Provider read |
| ONB-T02 | Historical v0.27.0 package-relative import failure and v0.27.1/future repair | Standalone installer actually imports correctly, original failure classified; no assertion of live fix from fixture only |
| ONB-T03 | Upgrade/reinstall | Identity, private config, data, valid SSH client keys, Provider and unrelated services preserved |
| ONB-T04 | Missing service, bus, package, occupied local port | Correct stage and actionable error; no false success |
| ONB-T05 | Interactive first-use selector | Only relevant questions; SSH/HTTPS/server-only choices all reach existing commands |
| ONB-T06 | Noninteractive JSON path | Valid JSON-only machine output, no accidental prompts or secret material |
| ONB-T07 | Linux/macOS/Windows/WSL supported client cases | Python 3.11+ generation, proper catalog ordering, safe host pin, public-only enrollment, scoped config changes |
| ONB-T08 | Real SSH client | Authenticated initialize, tools/list, list_containers, actual target App sees/calls tools |
| ONB-T09 | Loopback HTTP | Real authenticated local MCP read; public status remains unverified |
| ONB-T10 | Hosted HTTPS/OAuth real client | Actual configured origin, browser consent, MCP discovery and safe list_containers read |
| ONB-T11 | Wrong public URL, stale tunnel, OAuth failure | Exact negative codes; no Origin bypass, token leak or fabricated success |
| ONB-T12 | Existing Web client generator mismatch | User is routed to supported full profile CLI, never shown an unsupported Web promise |
| ONB-T13 | Normal-user CLI privilege boundary | `boxweave setup` / client generator without sudo, no privileged app code imported as root |
| ONB-T14 | Existing configured SSH / Web / HTTP / MCP clients | No breaking changes, no silent rewrites and no discarded settings |
| ONB-T15 | Optional first Box execution | Only in exact separately authorized Box, real exit code/stdout; never automatic Box creation |
| ONB-T16 | Broken installer/service and retry | Original marker/log/target preserved, no blind repair and no false runtime acceptance |

For all acceptance: S = source review, T = executed fixture/unit, L = real installed authenticated client/runtime test, U = unverified, BLOCKED = known prerequisite missing. Record actual commands/exits, exact source and installed versions and source-to-client evidence. An independent testable slice may proceed around a blocked check but cannot mark it passed.

## 8. Rollback, tracking and non-goals

- Only documentation and optional front-door UI/CLI changes in this design; keep existing SSH client scripts, force-command gateway, Provider credentials, HTTP listener, OAuth tokens, Web session model and Box lifecycle unchanged.
- Avoid a second configuration store, host status daemon, Cloudflare installer, automatic tunnel process, multiuser RBAC and new general guest MCP.
- If a new `setup` command breaks, disable it and keep `boxweave client`, `boxweave http`, `boxweave doctor` and `install.sh` independently functional.
- Before implementation, align normative scope/REQ/AC, document source-to-test traceability and update the actual live verification ledger as required by `AGENTS.md`. No version bump for docs only.
- **This design does not claim v0.27.1 has been installed or accepted on the user's actual host. No host/Box test is executed as part of this research.**
