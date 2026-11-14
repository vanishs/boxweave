# BoxWeave

**Give agents a Box. Keep control.**

BoxWeave is a **small, self-hosted MCP controller for Incus Linux containers**. It lets AI coding agents create and use isolated development environments without exposing the host's Incus administrator socket or a general host shell.

**Human → Agent → BoxWeave → Incus → Box**: a human owns the machine and approves credentials and setup; an Agent calls authenticated MCP tools; BoxWeave enforces exact-Box ownership and records asynchronous operations in SQLite; Incus runs the actual containers. The Agent normally lives **outside** its Box. Guest commands are sent through native Incus APIs, not an always-on guest Agent.

Built for developers and lab operators using their **own Linux machines** who want recoverable Agent workflows, explicit infrastructure boundaries and the option to test code inside disposable Boxes. It is **not** a VM/cloud orchestrator, host management shell, web console or general networking platform.

## Get started

- **[Manual installation (RC.3)](docs/getting-started/MANUAL_INSTALL.md)** — prerequisites, compatible Incus, reviewed host install, native Box preparation, local health checks and optional OAuth/ChatGPT access.
- **[Glossary](docs/GLOSSARY.md)** — what Box, Project, Profile, ACL, MCP, OAuth, Operation, UNKNOWN and readiness really mean.
- **[Documentation index](docs/INDEX.md)** — deeper product, design, development, acceptance and research references.

The current published prerelease is **[v1.0.0-rc.3](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.3)**. It has immutable source-bound Linux amd64/arm64 artifacts, but **is not a stable production release**. The stable-release pointer is still `UNPUBLISHED`; choose an explicit prerelease tag. The legacy `main` branch is a separate v0.x line, not a supported co-install or state adoption path.

## Understand the architecture

```text
 Human / operator
   | owns host, Incus setup, credentials and public exposure
   v
 Agent (ChatGPT / coding agent / any supported MCP client)
   | authenticated typed Streamable HTTP MCP (local or operator HTTPS)
   v
 BoxWeave — one Go executable + local maintenance CLI
   | role/ownership policy + SQLite durable operation receipts
   | official Incus Go SDK
   v
 Incus daemon — operator-managed
   | confined per-user Project + reviewed Profile/pool/private NIC/ACL
   +--- Box A (owned Linux container: code, tests, processes)
   +--- Box B (owned Linux container: independent workspace)
```

**Product relationship:** the human authorizes the Agent, not arbitrary host administration; the Agent asks BoxWeave to work only within registered Boxes; BoxWeave delegates storage, lifecycle and packet enforcement to Incus. Guest-root Exec is **not** host-root. An outer Box may host an independent nested controller where native Incus/kernel constraints allow; each layer has its **own** ownership boundary.

**Code layout:** `cmd/boxweave` is the CLI/server entry point; `internal/control` provides authenticated MCP/tool policy; `internal/state` owns the SQLite journal and OAuth credentials; `internal/native` adapts the official Incus SDK; `internal/install` handles host-local install/preflight/recovery; `install.sh` and `.github/workflows` support distribution and tests rather than running another privileged server.

**Operational model:** a mutation includes a client idempotency key and returns an original Tracking/Operation ID. After a timeout, ask for the **same original Operation** and its bounded output; never blindly replay an ambiguous `UNKNOWN`. Direct Bearer and optional embedded OAuth share the canonical `/mcp` endpoint, loopback-only by default; tunnels remain operator-managed.

See [architecture](docs/design/ARCHITECTURE.md), [Agent development loop](docs/product/DEVELOPMENT_LOOP.md), [operations](docs/design/OPERATIONS.md) and [security](docs/design/SECURITY.md).

## Features and boundaries

**Implemented in RC.3:** authenticated read/status tools, owned Box lifecycle, typed update, guest Exec, durable queryable operations and output, local maintenance and optional embedded OAuth. The [full v1 design](docs/product/SCOPE.md) includes minimal guest file work, native TCP/UDP forwarding, directional private peer controls and independent nesting; **a design goal is not an assertion that every planned tool has shipped**. Refer to the [tool catalog](docs/design/MCP_TOOLS.md) and [acceptance](docs/verification/ACCEPTANCE.md) for staged availability.

BoxWeave does **not** manage arbitrary Incus/foreign containers, DNS, website certificates, Caddy, general ACL/firewall rules, snapshots/backups, whole-host commands, tenants/billing or cross-host orchestration.

## Current evidence — do not confuse readiness with acceptance

RC.3 release CI passed exact-source quality, native/nested tests and dual-architecture publishing. A dated **physical-host pilot** reported a verified RC.3 systemd installation on Incus 7.6 and an existing Btrfs pool, 8/8 native prerequisite checks, `doctor ready=true`, and real **ChatGPT → HTTPS Tunnel → OAuth → MCP** `host_status`, `box_list` and `operation_list` reads. [See exact evidence and limits](docs/verification/RC3_PHYSICAL_HOST_PILOT_2026-11-13.md).

**Not yet signed on that host:** first on-demand Guest image/Box Create, real remote native writes, actual IPv4/IPv6 packet-isolation tests, native Btrfs quotas/Exec spool bounds, fault/rollback and independent host verifier. This is **not** formal `HOST_GO` or S5 PASS.

## Develop and contribute

[Contributing](CONTRIBUTING.md) · [quality gates](docs/process/QUALITY_GATES.md) · [implementation phases](docs/process/IMPLEMENTATION_PLAN.md) · [product boundary](docs/decisions/PRODUCT_BOUNDARY.md) · [decisions](docs/decisions/DECISIONS.md) · [development checkpoint](docs/development/EXECUTION_LOG.md).
