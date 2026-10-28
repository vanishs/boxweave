# Legacy project overview

All SRC references resolve in [SOURCE_REGISTER](SOURCE_REGISTER.md). Status vocabulary: **static-confirmed** means a source/code fact was inspected; **historical-reported** means a recorded run was read without rerunning; **inferred** means an interpretation; **unverified** means missing evidence; **rejected** means a specific claim is contradicted. Static confirmation is not runtime acceptance.

## Purpose and users

Static-confirmed, SRC-001/003: BoxWeave is a personal entry point connecting coding agents, model access and isolated machines. Containarium runs boxes; BoxWeave organizes enrollment, trust, diagnostics and an independent console. Operators install/manage an Ubuntu server; developers enroll Windows native/WSL or macOS agents; console administrators manage boxes, Tenant context, network plans and prompt graphs. Multi-user least privilege is not established by this positioning.

## Actual flows and boundaries

- Agent -> local adapter/config -> `ssh boxweave` -> forced-command launcher -> upstream MCP -> provider REST/Incus -> workload. Container access and agent transport use separate SSH identities. Source: SRC-003/004/008.
- Server CLI observes local state separately from MCP. Durable jobs persist worker receipts; approved nested selftests exercise the installed inner MCP. Source: SRC-005/009.
- Browser -> independent Caddy -> Flask/Waitress session/CSRF layer -> fixed Provider API adapter. SQLite owns graphs, local Tenant display metadata, operation plans/jobs and audit. Provider membership/Guard owns network truth. Source: SRC-006/007/010.
- Console credentials, database and service lifecycle are distinct from MCP credentials/timers. Provider binaries, Incus state and physical-host security remain external authority. Source: SRC-002/008/010.

## Dependencies and installation

Static-confirmed, SRC-002/006: Ubuntu/systemd, Containarium and Incus/LXC are the initial server integration; Python supports management/enrollment; Web uses Flask/Waitress/Argon2/SQLite, React/TypeScript/XYFlow and Caddy. These describe old implementation choices only. Windows enrollment uses PowerShell/OpenSSH; POSIX CLI setup requires Python 3.11+. Client GLM/Z.ai configuration belongs to the Claude adapter, not a new inference backend.

Installer staging resolves source and validates components before publishing managed files. Config and persistent data live outside replaceable Web releases; systemd timers rotate credentials and reconcile trusted host keys. Update is atomic per file, not a host transaction; pending markers and backups support repair rather than automatic global rollback (SRC-002).

## Evolution and constraints

SRC-012 history: initialization -> first-boot/enrollment fixes -> read-only CLI -> rerun repair -> durable jobs/selftest/provenance -> client wizard -> independent diagnostics/Web -> Tenants/shared access -> GLM/ZCode -> upstream native contract -> editable enrollment scripts/stable SSH alias -> transport fixes. Current committed VERSION is 0.4.3; several prose/package labels lag (SRC-013).

Historical-reported: the 2026-10-22 nested install/create/connect/Python run succeeded after a first-install oneshot fix, but not on the pristine original installer alone. It does not certify current checkout, Windows onboarding, bare metal, effective device filtering or reboot/autostart (SRC-011).

Inferred: the most valuable inheritance is evidence discipline around asynchronous writes, identity/trust, authority boundaries and honest diagnostics. Whether each old UI/workflow belongs in v2 remains a phase 2 decision. Generic provider abstraction and standalone Linux enrollment were extension ideas, not delivered equivalents (SRC-001/003).
