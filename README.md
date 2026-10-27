# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave is a personal control plane for AI coding: it connects your coding agents
(**Codex, Cursor, Claude Code, ZCode**), model access (such as **GLM / Z.ai**) and
self-hosted, isolated development environments through an MCP-compatible interface.
The first built-in infrastructure provider is **Containarium + Incus/LXC** on Ubuntu,
with a client wizard for **Windows and macOS** over restricted SSH.

## Positioning: BoxWeave and the provider

BoxWeave and Containarium are complementary roles, not a wrapper relationship:

- **Containarium** is an open-source agent runtime platform: box lifecycle, SSH
  isolation, eBPF egress policy, a model gateway, multi-tenancy and multiple
  backends (self-hostable, with an optional hosted cloud). It is the side that
  **runs the boxes**.
- **BoxWeave** is your personal entry point and console: multi-agent enrollment
  (Windows native/WSL, macOS), a restricted SSH transport, an independent web
  console, tenant and network orchestration, and recorded installation provenance
  and self-tests. It is the side that **organizes your agents, models and
  machines**.

The provider is designed to be **replaceable**: Containarium is the first
implementation, and a future provider only needs to implement the same stable
concept interface (a design placeholder, not yet implemented — see
[Architecture](docs/ARCHITECTURE.md)).

**Explicit non-goals:** BoxWeave does not rename or wrap the provider —
Containarium's MCP tool names, metadata and Incus SSH behavior stay as upstream.
The web console does not modify the provider binary, the restricted MCP launcher
or the token timers. Monitoring shows verified state/configuration only. The
Prompt Graph is an orchestration graph without execution semantics. Tenant
metadata never overrides real provider membership or "proves" network isolation.

Containarium is developed independently by
[FootprintAI](https://github.com/FootprintAI/Containarium) and is installed from
[containarium.dev](https://containarium.dev). BoxWeave is a control plane around
it, not a fork.

> **Project status: preview.** Ubuntu 24.04 LXC installation and a nested MCP
> create/connect/Python workflow have been tested, including a first-install fix
> discovered during that test. A clean bare-metal Ubuntu installation and fresh
> Windows client enrollment have not been validated together end to end. Review
> privileged scripts before use; see the [actual test results and
> limitations](docs/TEST-BOXWEAVE-LXC.md).

## Install on an Ubuntu server

The published installer may be run directly, but review the code and prefer a
pinned, audited release in production:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

The installer:

- detects an existing Containarium installation, or invokes its official
  installer if it is missing (`https://containarium.dev/install.sh`) with
  APT-compatible file permissions; it repairs verified Zabbly APT keyring
  permissions from an interrupted prior attempt and initializes an empty Incus
  instance that has no storage pool;
- installs the Containarium MCP backend, a protected launcher, short-lived JWT
  rotation, and a verified container SSH host-key synchronization timer;
- **also installs the independent web console by default** (see below);
- does **not** delete existing LXC instances. Rerunning it updates/repairs
  BoxWeave; saved settings are preserved unless explicitly overridden.

**First-boot readiness:** Containarium's core containers may take a while to
initialize. BoxWeave waits up to 15 minutes for the REST API, displaying
progress, and fails with targeted diagnostics rather than printing repeated
connection errors. See [Troubleshooting](docs/TROUBLESHOOTING.md). This is not a
guarantee that Containarium supports every newer Ubuntu release.

For more reliable error detection over SSH, download the installer and run it
explicitly instead of using a pipeline:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh -o /tmp/boxweave-install.sh
sudo bash /tmp/boxweave-install.sh
```

## Connect a coding agent

At completion, run **`boxweave client`** on the server. Its interactive wizard
selects Windows/macOS, an agent (Codex / Cursor / Claude Code / ZCode) and the
Windows native/WSL execution mode, then prints a command for the client computer.
Use `boxweave client --help` for parameter-based generation.

Setup pins the server identity, enrolls a public transport key using an existing
SSH login, verifies authenticated MCP operation, and configures the agent.

- **No private key is printed or transferred**: neither by the Ubuntu installer
  nor from Ubuntu to Windows. Server installation does not require any
  client-specific Windows username.
- macOS and WSL CLI setup requires Python 3.11+.
- Cursor global User Rules require the displayed manual Settings step.
- **GLM (Z.ai) access** is enabled through the Claude Code adapter:
  `boxweave client --agent claude-code --claude-provider zai` (or `zhipu`),
  with `--glm-model` to select a model. See
  [GLM client setup](docs/CLIENTS.md#glm-through-claude-code-030).
- After enrollment, `boxweave client --diagnose` produces a structured
  diagnostic report (MCP handshake, registration checks, configuration
  guidance). See [client diagnostics](docs/CLIENT-DIAGNOSTICS-READINESS.md) and
  [CLIENTS](docs/CLIENTS.md).

## Independent web console

The installer also installs the independent web console by default:
**Overview, Tenants, Boxes, Network, Monitoring (verified state/configuration)**
and a persistent **Prompt Graph**.

- It uses its own local administrator, provider credential, SQLite database,
  systemd services and Caddy Local CA. The existing MCP transport and
  credentials remain independent.
- Default HTTPS port is **18443** (local backend 18444). Initialize the
  administrator locally with `sudo boxweave web init-admin` (password of at
  least 8 characters).
- LAN HTTP mode is available with `--web-mode http`.
- **Never expose an unreviewed installation publicly.**
- Shared native Tenant creation (A) and reviewed atomic cross-Tenant ingress
  grants/revokes (B) are implemented for the authenticated
  [pinned Provider branch contract](docs/TENANT-UPSTREAM.md); an unchanged
  provider stays capability-gated (read-only) and the candidate is not installed
  automatically.
- See [installation, CA trust, HTTP mode and recovery](docs/WEB-UI.md),
  [provider capability gate](docs/WEB-CAPABILITIES.md) and
  [test evidence and verification limits](docs/WEB-TESTING.md).

## Operating expectations: timeouts, startup and nesting

These rules come from real installation and nested MCP tests. Recorded times and
versions describe those runs; they are not requirements or timing guarantees.

| Observation | What to do |
| --- | --- |
| `create_container` times out | Query `list_containers` and `get_container` using the original logical box name. Creation can succeed after the request times out. Reuse the resulting instance; do not submit a duplicate create, force-delete it or change its name to bypass the timeout. |
| A box is Running but SSH is not ready | Provisioning can still be installing packages and SSH keys. Wait and retry `connect` within the operation's deadline. Verify `whoami && hostname` before running the workload. |
| Containarium is active but its API is unavailable | Core containers may still be downloading, starting or initializing PostgreSQL. Inspect progress and wait; service activity alone does not establish API readiness. Do not restart the daemon or delete core containers simply to accelerate startup. |
| API returns HTTP 200, 401 or 403 | The endpoint is reachable. Authentication and execution still require actual MCP tool calls; HTTP 401 is not proof of authenticated health. |
| A oneshot service is inactive after finishing | Check `Result` and `ExecMainStatus`. Inactive/dead with success and exit 0 is normal for a completed oneshot; its timer should be active/waiting and enabled. |
| `status` or `doctor` exits 0 | Diagnostics report local installation_state=ready: required timers, successful oneshots and pending-update state are checked. This does not prove authenticated MCP operation or confinement; inspect the installer result and requested end-to-end test too. |
| SSH reports a host-key mismatch after recreation | Allow the trusted host-key synchronization timer to reconcile verified keys, then retry. Never disable host-key verification or manually remove trusted entries as a shortcut. Persistent authentication errors need diagnosis. |

For unattended tests, use a recorded deadline: allow up to **10 minutes per
provisioning, installation or readiness phase**, unless the user specifies a
different limit. Poll about every 15-30 seconds; do not reset the deadline on
every retry. Save the installer log and actual exit code so an MCP request
timeout or SSH disconnect is not mistaken for completion. `--api-ready-timeout`
bounds only the API wait (default 900 seconds), so an automation runner must
separately enforce any total installation deadline.

To inspect an installed server:

```bash
sudo boxweave status
sudo boxweave doctor --api
sudo systemctl show boxweave-token.service boxweave-hostkeys.service \
  -p Result -p ExecMainStatus -p ActiveState
```

Use sudo when the protected configuration is not readable by the normal user.
An incomplete update leaves `/etc/boxweave/update-in-progress`; fix the reported
cause and rerun the installer. See
[update and repair behavior](docs/INSTALL-UPDATE.md).

The nested smoke procedure (inner MCP initialize/initialized, tools/list,
create, verified `connect`, Python hello world) and its recorded results are in
[TEST-BOXWEAVE-LXC](docs/TEST-BOXWEAVE-LXC.md) and
[AGENTS.md](AGENTS.md); a reviewable, plan-confirmed variant is available as
`boxweave selftest plan/run/status` and `scripts/nested_smoke.py`.

## Installer options and environment variables

Full flag semantics are in `install.sh --help`; the common set:

| Flag / environment variable | Default | Purpose |
| --- | --- | --- |
| `--user` / `BOXWEAVE_LINUX_USER` | Invoking sudo user | Unprivileged user running the remote MCP process |
| `--ssh-host` / `BOXWEAVE_SSH_HOST` | Server's detected primary address | Reachable address for client SSH |
| `--ssh-port` / `BOXWEAVE_SSH_PORT` | Detected OpenSSH port, then `22` | Remote SSH port |
| `--allow-cidr` / `BOXWEAVE_ALLOWED_CIDRS` | Discovered private Incus managed bridge subnets | Restrict container IPs eligible for host-key synchronization |
| `--api-url` / `BOXWEAVE_API_URL` | `http://127.0.0.1:8080` | Local Containarium REST API; plaintext HTTP only on loopback |
| `--sync-interval` / `BOXWEAVE_SYNC_INTERVAL` | `15` | Host-key sync interval in seconds |
| `--api-ready-timeout` / `BOXWEAVE_API_READY_TIMEOUT` | `900` | Maximum seconds to await Containarium first-boot API readiness (30–3600); bounds only the API wait |
| `--repo` / `BOXWEAVE_REPO` | `vanishs/boxweave` | Component source repository |
| `--ref` / `BOXWEAVE_REF` | `main` | Source branch or tag; pin a reviewed tag for production |
| `CONTAINARIUM_VERSION` | `latest` | Release tag of the upstream Containarium MCP binary |
| `--jwt-secret-file` / `BOXWEAVE_JWT_SECRET_FILE` | `/etc/containarium/jwt.secret` | Provider JWT signing secret path |
| `--containariumd-binary` / `BOXWEAVE_CONTAINARIUMD_BINARY` | Detected by `command -v` | Provider daemon executable path |
| `--web-mode` | `https` | Web console mode: `https` or `http` |
| `--web-host` | Private IPv4 | Web console bind address |
| `--web-port` | `18443` | Web console external port |
| `--web-backend-port` | `18444` | Web backend local port |
| `--source-dir` | unset | Install from a local reviewed source checkout (recorded in provenance) |

There are intentionally **no hard-coded usernames, machine addresses, WSL
distribution names, or LXC subnets**. Product-wide paths such as `/etc/boxweave`
are stable installation conventions rather than machine-dependent configuration.

## Management commands

The source checkout provides read-only diagnostics and explicit job/test
commands:

```bash
bash boxweave.sh version
bash boxweave.sh version --components --json
sudo bash boxweave.sh doctor --api
sudo bash boxweave.sh box inspect example-container --project default
sudo boxweave web init-admin   # web console administrator initialization
```

The installer installs the same CLI as `boxweave` (for example
`boxweave version`). BoxWeave's version is separate from Containarium and Incus
versions. The CLI reports `ready / starting / incomplete / failed` installation
states. `boxweave version --verbose --json` shows recorded source commits and
component hashes. `boxweave job start/status` provides durable task receipts;
`boxweave selftest plan/run/status` (or `scripts/nested_smoke.py`) provides a
reviewable nested MCP smoke test; test execution requires confirmation of the
generated plan. See [CLI usage](docs/CLI.md),
[task, readiness and provenance details](docs/TASKS-AND-SELFTEST.md) and
[research integration and future work](docs/RESEARCH-INTEGRATION.md).

To verify server-side health:

```bash
sudo /usr/local/sbin/boxweave-hostkeys               # read-only dry run
sudo systemctl status boxweave-hostkeys.timer --no-pager
sudo journalctl -u boxweave-hostkeys.service -n 30 --no-pager
sudo systemctl status boxweave-token.timer --no-pager
```

After client setup and restarting the agent, verify that the MCP server
`boxweave` exposes its provider tools: test `list_containers`, then `connect`
against a disposable container. A create call may time out even when resource
creation succeeded: query its state **before retrying**.

## Platform support

| Role | Current support | Planned extension |
| --- | --- | --- |
| Server | Ubuntu + systemd + Containarium + Incus/LXC | Additional infrastructure providers |
| Client OS | Windows native / WSL SSH bridge / WSL CLI; macOS native SSH | Standalone Linux clients |
| Coding agent | Codex, Cursor, Claude Code, ZCode adapters (see client validation limits) | Other MCP-capable agents |
| Protocol | MCP over restricted SSH | Additional authenticated transports |
| UI language | English (`en-US` initial convention) | Localized messages via future locale packs |

**A planned integration is not a working integration.** Standalone Linux and
additional providers remain extension points; see
[Architecture](docs/ARCHITECTURE.md) and [Localization](docs/LOCALIZATION.md).

## Files installed on the Ubuntu host

MCP / CLI side:

| Path | Purpose |
| --- | --- |
| `/etc/boxweave/config.json` | Provider configuration, selected Unix user, approved CIDRs |
| `/etc/boxweave/providers/containarium.env` | Non-secret provider endpoint settings |
| `/usr/local/bin/boxweave-mcp` | Root-owned SSH forced-command launcher |
| `/usr/local/bin/boxweave` | Read-only management CLI |
| `/usr/local/libexec/boxweave/{cli.py,clients.py,VERSION}` | CLI, client generator and version |
| `/usr/local/libexec/boxweave/{jobs,mcp_client,smoke,provenance}.py` | Task receipts, approved smoke tests and provenance helpers |
| `/usr/local/libexec/boxweave/containarium-mcp-server` | Bundled upstream MCP backend executable |
| `/etc/boxweave/client.json` | Public endpoint/keys and pinned client asset hashes (0644) |
| `/etc/boxweave/install-manifest.json` | Recorded installation source, hashes and completion state |
| `/usr/local/sbin/boxweave-hostkeys` | Trusted Incus → SSH host-key reconciler |
| `/usr/local/sbin/boxweave-token` | MCP authentication-token rotation |
| `/etc/systemd/system/boxweave-hostkeys.{service,timer}` | Periodic host-key reconciliation |
| `/etc/systemd/system/boxweave-token.{service,timer}` | Daily token rotation |
| `~/.config/boxweave/containarium-mcp.jwt` | Protected JWT for the configured unprivileged Unix user |
| `~/.ssh/known_hosts` | Standard file Containarium's Go SSH client reads; managed for verified project LXC IPs only |
| `~/.ssh/authorized_keys` | Restricted client transport public-key enrollment |

Web console side (installed by default):

| Path | Purpose |
| --- | --- |
| `/etc/boxweave/web.json` | Web console configuration (mode, ports, credential references) |
| `/usr/local/libexec/boxweave/{web_install,web_entry,web_token}.py` | Web installation, entry point and token rotation |
| `/etc/systemd/system/boxweave-web.service` | Web backend service |
| `/etc/systemd/system/boxweave-web-caddy.service` | Web-dedicated Caddy (Local CA) |
| `/etc/systemd/system/boxweave-web-token.{service,timer}` | Periodic web token rotation |
| `/opt/boxweave-web/releases` | Atomically published web release directories |
| `/var/lib/boxweave-web-caddy` | Caddy certificate and state directory |
| System users `boxweave-web`, `boxweave-web-caddy` | Unprivileged runtime identities |

Existing `/etc/containarium` configuration, LXC instance files, and
Containarium's own service retain their **upstream** names. BoxWeave does not
rename files owned by its provider.

Client-side transport files (per-OS SSH keys, pinned `known_hosts`, agent
configuration paths and the `-AgentHome` / `--agent-home` overrides) are
documented in [CLIENTS](docs/CLIENTS.md). The names of agent configuration files
such as Codex's `config.toml` are dictated by the agent, not by BoxWeave, and
existing files are backed up before modification.

## Why the host-key reconciler exists

Deleting and recreating an LXC can reuse an IP address with fresh SSH server
keys. Containarium's MCP Go SSH client correctly rejects mismatches in
`~/.ssh/known_hosts`. BoxWeave reconciles **ED25519, ECDSA, and RSA** host keys
from the privileged local Incus API against those actually offered on the
network. Only a matching, private, managed project-container target may be
updated; core Containarium containers are excluded.

This runs on a timer, so the agent may need to retry a newly rebuilt container
after the next sync cycle. It is **not** equivalent to disabling host-key checks.

## Security model

- **Transport**: the client key is limited via `authorized_keys` **forced
  command** plus `restrict`; it does not grant an interactive host shell. The
  MCP backend's token may nevertheless have **administrator privileges over the
  provider** — SSH command restriction is not full least-privilege
  authorization. The server identity is pinned independently in the client's
  `known_hosts`, separate from container host-key reconciliation.
- **Keys**: no private key is printed or transferred; privileged configuration
  is 0600, public enrollment metadata is 0644.
- **Authentication**: MCP uses a short-lived JWT with periodic rotation; the web
  console uses 8-hour sessions, HttpOnly + SameSite=Strict cookies, CSRF with
  exact Origin checks, login rate limiting and Argon2id password hashing.
- **Tenant authority**: the provider is the single source of truth for
  membership and isolation state; local metadata is presented only inside a
  limited freshness window (180-second Guard evidence) and never presented as a
  successful TCP connectivity test.
- **Capability gating**: every provider write class is authenticated via a
  capability handshake; older provider versions degrade to read-only.
- **Prerequisites and exposure**: the installer requires root on Ubuntu and an
  existing SSH login for one-time restricted public-key enrollment; it can
  install OpenSSH Server but does not configure external firewalls. The MCP API
  URL defaults to loopback; BoxWeave does not secure pre-existing Containarium
  gRPC/REST listeners. Running `curl | sudo bash` executes network-supplied
  root-level code — prefer inspecting and pinning an immutable release. BoxWeave
  has no automatic permission to modify unrelated containers or provider
  infrastructure.

See [Security](docs/SECURITY.md), [Tenant operation](docs/TENANTS.md),
[Troubleshooting](docs/TROUBLESHOOTING.md) and
[Legacy migration](docs/MIGRATION.md) before upgrading a previously configured
machine.

## Documentation map

| Topic | Document |
| --- | --- |
| CLI usage and exit codes | [CLI](docs/CLI.md) |
| Client setup, platforms and GLM | [CLIENTS](docs/CLIENTS.md) |
| Client diagnostics design and limits | [CLIENT-DIAGNOSTICS-READINESS](docs/CLIENT-DIAGNOSTICS-READINESS.md) |
| Install/update flow and recovery | [INSTALL-UPDATE](docs/INSTALL-UPDATE.md) |
| Web console / capability gate / testing | [WEB-UI](docs/WEB-UI.md) · [WEB-CAPABILITIES](docs/WEB-CAPABILITIES.md) · [WEB-TESTING](docs/WEB-TESTING.md) |
| Tenant design / upstream patch | [TENANTS](docs/TENANTS.md) · [TENANT-UPSTREAM](docs/TENANT-UPSTREAM.md) |
| Architecture and extension points | [ARCHITECTURE](docs/ARCHITECTURE.md) |
| Security overview | [SECURITY](docs/SECURITY.md) |
| Tasks, self-tests and provenance | [TASKS-AND-SELFTEST](docs/TASKS-AND-SELFTEST.md) |
| Troubleshooting | [TROUBLESHOOTING](docs/TROUBLESHOOTING.md) |
| Legacy timer migration | [MIGRATION](docs/MIGRATION.md) |
| Localization convention | [LOCALIZATION](docs/LOCALIZATION.md) |
| Recorded baselines (historical) | [TEST-BOXWEAVE-LXC](docs/TEST-BOXWEAVE-LXC.md) · [DEVELOPMENT-READINESS-2026-10-22](docs/DEVELOPMENT-READINESS-2026-10-22.md) · [CLIENT-SUPPORT-RESEARCH](docs/CLIENT-SUPPORT-RESEARCH.md) |

## Development and self-test

```bash
bash -n install.sh publish.sh
python3 -m compileall -q server tests
python3 -m unittest discover -s tests -v
```

`tests/` holds the local suite (installer sandbox tests must run as a non-root
Linux user with temporary paths and fake host tools). PowerShell changes are
checked by parsing, not by executing privileged setup; documentation-only edits
need link, consistency and formatting checks. The repository has **no CI
pipeline** — treat the local test suite as the gate. Repository updates go
through ordinary Git commits; the included publication scripts refuse to
overwrite an existing repository or history.

Agent instructions are in [AGENTS.md](AGENTS.md); adapter authors should also
read [agents/README.md](agents/README.md).

## License

[MIT](LICENSE)
