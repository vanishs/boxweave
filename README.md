# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave connects AI coding agents to self-hosted, isolated development environments through an infrastructure provider and an MCP-compatible interface. The first implementation uses **Containarium + Incus/LXC** on Ubuntu, with **Windows and macOS** client setup for **Codex, Cursor and Claude Code** over restricted SSH.

> **Project status: preview.** Ubuntu 24.04 LXC installation and a nested MCP create/connect/Python workflow have been tested, including a first-install fix discovered during that test. A clean bare-metal Ubuntu installation and fresh Windows client enrollment have not been validated together end to end. See the [actual test results and limitations](docs/TEST-BOXWEAVE-LXC.md); review privileged scripts before use.

## Install on an Ubuntu server

The published installer may be run directly, but review the code and prefer a pinned, audited release in production:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

BoxWeave detects an existing Containarium installation, or invokes its official installer if it is missing (`https://containarium.dev/install.sh`) with APT-compatible file permissions. It repairs verified Zabbly APT keyring permissions from an interrupted prior attempt and initializes an empty Incus instance that has no storage pool. It installs the Containarium MCP backend, a protected launcher, short-lived JWT rotation, and a verified container SSH host-key synchronization timer. It does **not** delete existing LXC instances.

At completion, run **`boxweave client`** on the server. Its interactive wizard selects Windows/macOS, agent and Windows native/WSL execution mode, then prints a command for the client computer. Use `boxweave client --help` for parameter-based generation.

Setup pins the server identity, enrolls a public transport key using an existing SSH login, verifies authenticated MCP operation, and configures the agent. macOS/WSL CLI setup requires Python 3.11+. Cursor global User Rules require the displayed manual Settings step. See [client setup and validation limits](docs/CLIENTS.md).

**No private key is printed by the Ubuntu installer or transferred from Ubuntu to Windows.** Server installation does not require any client-specific Windows username.

**First-boot readiness:** Containarium's core containers may take a while to initialize. BoxWeave now waits up to 15 minutes for the REST API, displaying progress, and fails with targeted diagnostics rather than printing repeated connection errors. See [Troubleshooting](docs/TROUBLESHOOTING.md). This is not a guarantee that Containarium supports every newer Ubuntu release.

For more reliable error detection over SSH, download the installer and run it explicitly instead of using a pipeline:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh -o /tmp/boxweave-install.sh
sudo bash /tmp/boxweave-install.sh
```

## Independent web console

The revised installer also installs the Web UI by default: Boxes, Network,
Monitoring (verified state/configuration) and a persistent Prompt Graph. It uses
its own local administrator, provider credential, SQLite database, systemd
services and Caddy Local CA. Existing MCP transport and credentials remain
independent. Default HTTPS port is 18443; initialize locally with
`sudo boxweave web init-admin`. Never expose an unreviewed installation publicly.
See [installation, CA trust, HTTP mode and recovery](docs/WEB-UI.md),
[provider capability gate](docs/WEB-CAPABILITIES.md) and
[test evidence and verification limits](docs/WEB-TESTING.md).

## Operating expectations: timeouts, startup and nesting

These rules come from real installation and nested MCP tests. Recorded times
and versions describe those runs; they are not requirements or timing guarantees.

| Observation | What to do |
| --- | --- |
| `create_container` times out | Query `list_containers` and `get_container` using the original logical box name. Creation can succeed after the request times out. Reuse the resulting instance; do not submit a duplicate create, force-delete it or change its name to bypass the timeout. |
| A box is Running but SSH is not ready | Provisioning can still be installing packages and SSH keys. Wait and retry `connect` within the operation's deadline. Verify `whoami && hostname` before running the workload. |
| Containarium is active but its API is unavailable | Core containers may still be downloading, starting or initializing PostgreSQL. Inspect progress and wait; service activity alone does not establish API readiness. Do not restart the daemon or delete core containers simply to accelerate startup. |
| API returns HTTP 200, 401 or 403 | The endpoint is reachable. Authentication and execution still require actual MCP tool calls; HTTP 401 is not proof of authenticated health. |
| A oneshot service is inactive after finishing | Check `Result` and `ExecMainStatus`. Inactive/dead with success and exit 0 is normal for a completed oneshot; its timer should be active/waiting and enabled. |
| `status` or `doctor` exits 0 | Updated diagnostics report local installation_state=ready: required timers, successful oneshots and pending-update state are checked. This does not prove authenticated MCP operation or confinement; inspect the installer result and requested end-to-end test too. |
| SSH reports a host-key mismatch after recreation | Allow the trusted host-key synchronization timer to reconcile verified keys, then retry. Never disable host-key verification or manually remove trusted entries as a shortcut. Persistent authentication errors need diagnosis. |

For unattended tests, use a recorded deadline: allow up to **10 minutes per
provisioning, installation or readiness phase**, unless the user specifies a
different limit. Poll about every 15-30 seconds; do not reset the deadline on
every retry. Save the installer log and actual exit code so an MCP request timeout
or SSH disconnect is not mistaken for completion. A known error should be
diagnosed rather than blindly waited out. For a ten-minute API wait, use
`--api-ready-timeout 600`; that flag bounds only the API wait, so an automation
runner must separately enforce any total installation deadline. The installer's
default API wait remains 900 seconds.

To inspect an installed server:

```bash
sudo boxweave status
sudo boxweave doctor --api
sudo systemctl show boxweave-token.service boxweave-hostkeys.service \
  -p Result -p ExecMainStatus -p ActiveState
```

Use sudo when the protected configuration is not readable by the normal user.
An incomplete update leaves `/etc/boxweave/update-in-progress`; fix the reported
cause and rerun the revised installer. Fresh oneshot units may be unloaded even
after `daemon-reload`: the installer must only call `reset-failed` for units that
are actually failed. See [update and repair behavior](docs/INSTALL-UPDATE.md).

### Minimal nested smoke test

1. Create an ordinary Ubuntu LXC and verify MCP `connect` identity.
2. Install BoxWeave inside it and wait for API and managed timer readiness.
   Reuse compatible installed Incus; nesting alone does not require an upgrade
   or a specially prepared image.
3. Start the installed inner `/usr/local/bin/boxweave-mcp` as its configured
   non-root user, using an MCP client with a real initialize/initialized exchange
   and tool discovery. Its managed launcher supplies authentication; do not read
   or print JWT/private-key contents or add a fixed MCP registration per box.
4. Use the **inner** MCP to list resources and create a disposable workload LXC.
   Supply the verified full public key from that nested host's managed
   Ubuntu-to-container identity. An outer host's public key or a Windows/WSL
   transport key is not an interchangeable identity.
5. Use inner MCP `connect` to verify identity, then run
   `python3 -c "print('hello world')"`. Record stdout and the returned exit code.
   Direct Incus execution does not establish that this MCP execution chain works.

This chain was verified on 2026-10-22. The first installation attempt reached a
systemd first-run error after about eight minutes; after the documented fix, a
rerun succeeded in about eight seconds. Inner MCP creation took about a minute
and Python returned `hello world`, exit 0. See the [test report](docs/TEST-BOXWEAVE-LXC.md).
No cold-backup rehearsal, PCI configuration, fixed UID map or upgrade audit is
required for this basic test. Those remain optional, separate investigations.
Nested AppArmor labels and internal bridge addresses do not by themselves prove
host confinement or LAN reachability.

Agent instructions are in [AGENTS.md](AGENTS.md); adapter authors should also read
[agents/README.md](agents/README.md).

## Advanced server options

**Update/fix:** rerun the revised installer to refresh BoxWeave files and repair
managed services. Saved settings are retained; explicit flags/environment override
them. Downloads are staged before replacement, managed files are backed up, and
timers are restarted after validation. Existing Containarium/Incus packages are
reused. Old locally saved installer copies need to be downloaded again to obtain
these fixes. See [installation, updates and recovery](docs/INSTALL-UPDATE.md).

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh \
  | sudo bash -s -- --user dev --ssh-host dev.example.net --ssh-port 2222
```

| Flag / environment variable | Default | Purpose |
| --- | --- | --- |
| `--user` / `BOXWEAVE_LINUX_USER` | Invoking sudo user | Unprivileged user running the remote MCP process |
| `--ssh-host` / `BOXWEAVE_SSH_HOST` | Server's detected primary address | Reachable address for client SSH |
| `--ssh-port` / `BOXWEAVE_SSH_PORT` | Detected OpenSSH port, then `22` | Remote SSH port |
| `--allow-cidr` / `BOXWEAVE_ALLOWED_CIDRS` | Discovered private Incus managed bridge subnets | Restrict container IPs eligible for host-key synchronization |
| `--api-url` / `BOXWEAVE_API_URL` | `http://127.0.0.1:8080` | Local Containarium REST API; plaintext HTTP only on loopback |
| `--sync-interval` / `BOXWEAVE_SYNC_INTERVAL` | `15` | Host-key sync interval in seconds |
| `--api-ready-timeout` / `BOXWEAVE_API_READY_TIMEOUT` | `900` | Maximum seconds to await Containarium first-boot API readiness (30–3600) |
| `--repo` / `BOXWEAVE_REPO` | `vanishs/boxweave` | Component source repository |
| `--ref` / `BOXWEAVE_REF` | `main` | Source branch or tag; pin a reviewed tag for production |
| `CONTAINARIUM_VERSION` | `latest` | Release tag of the upstream Containarium MCP binary |
| `--jwt-secret-file` / `BOXWEAVE_JWT_SECRET_FILE` | `/etc/containarium/jwt.secret` | Provider JWT signing secret path |
| `--containariumd-binary` / `BOXWEAVE_CONTAINARIUMD_BINARY` | Detected by `command -v` | Provider daemon executable path |

There are intentionally **no hard-coded usernames, machine addresses, WSL distribution names, or LXC subnets**. Product-wide paths such as `/etc/boxweave` are stable installation conventions rather than machine-dependent configuration.

## Management commands

The source checkout provides read-only diagnostics and explicit job/test commands:

```bash
bash boxweave.sh version
bash boxweave.sh version --components --json
sudo bash boxweave.sh doctor --api
sudo bash boxweave.sh box inspect example-container --project default
```

The updated installer installs this as `boxweave`, for example `boxweave version`.
BoxWeave's development version is separate from Containarium and Incus versions.
See [CLI usage and result interpretation](docs/CLI.md) and
[research integration and future work](docs/RESEARCH-INTEGRATION.md).

The updated CLI reports `ready / starting / incomplete / failed` installation
states. `boxweave version --verbose --json` shows recorded source commits and
component hashes. `boxweave job start/status` provides durable task receipts;
`boxweave selftest plan/run/status` (or `scripts/nested_smoke.py`) provides a
reviewable nested MCP smoke test. Test execution requires confirmation of the
generated plan. See [task, readiness and provenance details](docs/TASKS-AND-SELFTEST.md).

## Platform support

| Role | Current support | Planned extension |
| --- | --- | --- |
| Server | Ubuntu + systemd + Containarium + Incus/LXC | Additional infrastructure providers |
| Client OS | Windows native / WSL SSH bridge / WSL CLI; macOS native SSH | Standalone Linux clients |
| Coding agent | Codex, Cursor, Claude Code adapters (see client validation limits) | Other MCP-capable agents |
| Protocol | MCP over restricted SSH | Additional authenticated transports |
| UI language | English (`en-US` initial convention) | Localized messages via future locale packs |

**A planned integration is not a working integration.** Standalone Linux and additional providers remain extension points; see [Architecture](docs/ARCHITECTURE.md) and [Localization](docs/LOCALIZATION.md).

## Files installed on the Ubuntu host

| Path | Purpose |
| --- | --- |
| `/etc/boxweave/config.json` | Provider configuration, selected Unix user, approved CIDRs |
| `/etc/boxweave/providers/containarium.env` | Non-secret provider endpoint settings |
| `/usr/local/bin/boxweave-mcp` | Root-owned SSH forced-command launcher |
| `/usr/local/bin/boxweave` | Read-only management CLI |
| `/usr/local/libexec/boxweave/{cli.py,clients.py,VERSION}` | CLI, client generator and version |
| `/etc/boxweave/client.json` | Public endpoint/keys and pinned client asset hashes (0644) |
| `/usr/local/libexec/boxweave/{jobs,mcp_client,smoke,provenance}.py` | Task receipts, approved smoke tests and provenance helpers |
| `/etc/boxweave/install-manifest.json` | Recorded installation source, hashes and completion state |
| `/usr/local/libexec/boxweave/containarium-mcp-server` | Bundled upstream MCP backend executable |
| `/usr/local/sbin/boxweave-hostkeys` | Trusted Incus → SSH host-key reconciler |
| `/usr/local/sbin/boxweave-token` | MCP authentication-token rotation |
| `/etc/systemd/system/boxweave-hostkeys.{service,timer}` | Periodic host-key reconciliation |
| `/etc/systemd/system/boxweave-token.{service,timer}` | Daily token rotation |
| `~/.config/boxweave/containarium-mcp.jwt` | Protected JWT for the configured unprivileged Unix user |
| `~/.ssh/known_hosts` | Standard file Containarium's Go SSH client reads; managed for verified project LXC IPs only |
| `~/.ssh/authorized_keys` | Restricted client transport public-key enrollment |

Existing `/etc/containarium` configuration, LXC instance files, and Containarium's own service retain their **upstream** names. BoxWeave does not rename files owned by its provider.

## Files configured on Windows

Paths follow the current Windows user's profile by default; use `-AgentHome` to override Codex's configuration directory.

| Path | Purpose |
| --- | --- |
| `%USERPROFILE%\.ssh\boxweave` | Local dedicated SSH private key; not sent to the server |
| `%USERPROFILE%\.ssh\boxweave.pub` | Matching transport public key |
| `%USERPROFILE%\.ssh\boxweave_known_hosts` | Pinned Ubuntu host ED25519 key for Native mode |
| WSL `~/.ssh/boxweave{,.pub}` | Independent key pair used only by WSL transport |
| WSL `~/.ssh/boxweave_known_hosts` | Pinned Ubuntu host key for WSL mode |
| `%USERPROFILE%\.codex\config.toml` | Adds `[mcp_servers.boxweave]` and selects native Windows agent runtime |
| `%USERPROFILE%\.codex\AGENTS.md` | A replaceable `BOXWEAVE MANAGED` policy block; preserves other content |

Existing `config.toml` and `AGENTS.md` files are backed up before modification. The `config.toml` and `AGENTS.md` names are dictated by **Codex**, not by BoxWeave.

## Why the host-key reconciler exists

Deleting and recreating an LXC can reuse an IP address with fresh SSH server keys. Containarium's MCP Go SSH client correctly rejects mismatches in `~/.ssh/known_hosts`. BoxWeave reconciles **ED25519, ECDSA, and RSA** host keys from the privileged local Incus API against those actually offered on the network. Only a matching, private, managed project-container target may be updated. Core Containarium containers are excluded.

This runs on a timer, so the agent may need to retry a newly rebuilt container after the next sync cycle. It is **not** equivalent to disabling host-key checks.

## Verify server health

```bash
sudo /usr/local/sbin/boxweave-hostkeys               # read-only dry run
sudo systemctl status boxweave-hostkeys.timer --no-pager
sudo journalctl -u boxweave-hostkeys.service -n 30 --no-pager
sudo systemctl status boxweave-token.timer --no-pager
```

After Windows setup and restarting Codex, verify that MCP server `boxweave` exposes its Containarium tools; test `list_containers`, then test `connect` against a disposable container. A create call may time out even when resource creation succeeded: query its state **before retrying**.

## Security and prerequisites

- The installer requires root on Ubuntu and an existing SSH login from the client for one-time restricted public-key enrollment. It can install OpenSSH Server but does not configure external firewalls or ensure server accessibility.
- The client SSH key is limited via `authorized_keys` **forced command** plus `restrict`; it does not grant an interactive host shell. The MCP backend's token may nevertheless have **administrator privileges over the provider**. SSH command restriction is **not** full least-privilege authorization.
- The MCP API URL defaults to loopback. BoxWeave does not secure pre-existing Containarium gRPC/REST listeners; review their exposure independently.
- The initial English text and docs do not imply localization support yet.
- Running `curl | sudo bash` executes network-supplied root-level code. Prefer inspecting and pinning an immutable release, and never treat transport TLS as proof that all downloaded code has been reviewed.
- BoxWeave has no automatic permission to modify unrelated containers or provider infrastructure. Its sync script filters eligible Incus containers by Containarium metadata and name, and confirms host identities before updates.

See [Security](docs/SECURITY.md), [Troubleshooting](docs/TROUBLESHOOTING.md), and [Legacy migration](docs/MIGRATION.md) before upgrading a previously configured machine.

## Publish or update GitHub

If your repository already exists, **do not** run the initial publication scripts: they intentionally refuse to overwrite existing repositories. To update it, apply the reviewed changes in a normal Git working copy, then commit and push. For example:

```bash
git clone https://github.com/vanishs/boxweave.git
cd boxweave
# Copy the changed files from the verified source package, or apply the included patch.
git add install.sh README.md docs/ tests/
git commit -m "fix: harden Ubuntu first-boot installation"
git push origin main
```

For a *new*, empty repository, the included publication scripts create a public repository with the owner’s GitHub CLI credentials:

```bash
bash publish.sh vanishs/boxweave
```

On Windows, from the extracted project folder:

```powershell
.\publish.ps1 -Repository 'vanishs/boxweave'
```

You need `git`, `gh`, an authenticated `gh auth login` session, and a configured Git commit author. The initial publishing scripts refuse to overwrite an existing repository or Git history. A repository update should go through ordinary Git commits or a pull request.

## Development checks

```bash
bash -n install.sh publish.sh
python3 -m compileall -q server tests
python3 -m unittest discover -s tests -v
```

GitHub Actions additionally parses the Windows PowerShell script on a Windows runner. The installation flow itself must still be tested on clean Ubuntu and Windows hosts.

## Project layout

```text
install.sh                         Ubuntu server bootstrap
server/hostkeys.py                Incus-verified SSH host-key synchronization
server/tokens.py                  Provider MCP JWT rotation
clients/windows/setup.ps1         Windows Native / WSL SSH client setup
clients/macos/README.md           Reserved macOS client interface
clients/linux/README.md           Reserved Linux client interface
agents/codex/README.md            Current Codex adapter contract
agents/README.md                  Agent-adapter extension point
providers/containarium/README.md  Current infrastructure-provider interface
docs/ARCHITECTURE.md              Interfaces and compatibility boundaries
docs/MIGRATION.md                 Moving from the previous installer
docs/TROUBLESHOOTING.md           First-boot and interrupted install recovery
docs/LOCALIZATION.md             Localization convention (planned)
tests/test_hostkeys.py            Host-key unit tests
tests/test_installer_regressions.py  Fresh Ubuntu install regression checks
publish.sh / publish.ps1          GitHub publication helpers
.github/workflows/ci.yml         Automated static checks
```

## Acknowledgment

Containarium is developed independently by [FootprintAI](https://github.com/FootprintAI/Containarium) and is installed from [containarium.dev](https://containarium.dev). BoxWeave is an integration layer, not a fork of Containarium.
