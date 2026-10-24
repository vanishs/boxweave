# BoxWeave management CLI

The management CLI uses Python 3's standard library and is independent of the MCP
transport. Version/status/doctor/inspect perform read-only observations. Explicit
job and selftest commands execute approved tasks; see [tasks and selftest](TASKS-AND-SELFTEST.md).

From a source checkout on Ubuntu:

```bash
bash boxweave.sh version
python3 server/cli.py version --json
bash boxweave.sh version --components
bash boxweave.sh status
sudo bash boxweave.sh doctor --api --json
sudo bash boxweave.sh box inspect example-container --project default --json
```

The updated installer installs `/usr/local/bin/boxweave`, after which the same
commands work as `boxweave version`, `boxweave doctor`, and so on. This source
change alone does not update an already installed server. Installation requires
the CLI and VERSION files to be available at the selected repository ref.

| Command | Evidence returned |
| --- | --- |
| `version` | BoxWeave's own VERSION file, 0.2.0 |
| `version --components` | Installed Incus client and Containarium CLI version strings; no MCP backend launch or server version claim |
| `version --verbose` | Recorded installation source commit, hashes, versions and installer difference indicator |
| `status` | Configuration availability and selected systemd service/timer states |
| `doctor` | Status, component versions, container detection and current AppArmor namespace observations |
| `doctor --api` | Adds a three-second unauthenticated loopback API reachability probe |
| `box inspect NAME --project PROJECT` | Exact local Incus instance, selected security/device metadata and configuration fingerprint |

`--json` is accepted before the command or after the leaf command. Status and
doctor also accept `--config PATH`. The default configuration is
`/etc/boxweave/config.json`; its root-only permissions may require sudo. Incus
inspection requires access to the local Incus daemon. The CLI does not elevate
privileges itself.

## Interpretation

For status/doctor, exit 0 now means local installation_state=ready, exit 1 is
failed, and exit 2 is starting/incomplete. Required timers must be enabled and
active; successful inactive oneshots are accepted only with a recorded execution.
A pending-update marker prevents ready. This is not a security certification or
proof of authenticated MCP execution. For inspect, failed queries exit 1; invalid
arguments exit 2. See the [state and task result semantics](TASKS-AND-SELFTEST.md).

An HTTP 200, 401 or 403 response establishes reachability only. Authentication,
MCP execution, application health and LAN access still require their own tests.
The optional probe accepts only loopback HTTP(S) origins without credentials,
paths, queries or fragments. It uses no proxy, follows no redirects, reads no
response body, and never opens a token or signing-secret file.

Commands have time and output budgets, do not invoke a shell, and suppress raw
command error output. Diagnostics return selected fields instead of dumping
configuration, environment variables, disk sources, tokens, or raw LXC/AppArmor
configuration. Raw policy presence and hashes remain available for comparison.

Instance inspection uses `incus --force-local query` and the literal name. It
never appends `-container` and does not fall back to another name. The upstream
MCP naming behavior is unchanged. The default project is explicitly `default`;
choose another project with `--project`. A UUID can be shared by clones; the
configuration hash is comparison evidence, not an authorization mechanism.

In nested containers, an `unconfined` child AppArmor label does not establish the
parent's effective policy. The doctor reports namespace evidence and leaves
effective device filtering unverified. It does not load BPF programs, access
physical devices, change security policies or require a special LXC image.

## Client configuration

`boxweave client` starts an interactive command generator for Windows/macOS and Codex/Cursor/Claude Code. It performs no enrollment on the server. Use `boxweave client --help` for explicit selection, client-path options and JSON output. See [client setup](CLIENTS.md).

It reads public `/etc/boxweave/client.json`, not credentials. Non-terminal input requires complete choices. Exit 0 means command generation, 2 means invalid/missing input or descriptor, and 130 means cancellation. Existing management/selftest commands retain their behavior.

The client --diagnose option generates an independent diagnostic command for an
already enrolled client. It does not execute diagnosis on Ubuntu or repair a
client. Use the same OS/agent/runtime/transport selection options; Windows bridge
diagnosis also needs --wsl-distro. Replacement/bootstrap-identity options are
incompatible. See [diagnostic stages, reports and limits](CLIENTS.md#independent-client-diagnostics).

## Independent web administrator

`boxweave web init-admin` and `boxweave web reset-password` require an interactive
local terminal and service-user/root access. Passwords are never arguments.
`boxweave web status` reports the configured origin and local backend listener,
without opening provider credentials. Use `status`/`doctor` for Web systemd
observations. See [Web installation and security](WEB-UI.md).

Web status reports supported SQLite schema and Tenant capability gates without
opening Provider credentials. Doctor lists the same limits; live authenticated
Guard evidence is available from the console API, not inferred from CLI exit 0.
No new Provider Tenant-write CLI is added. See [Tenants](TENANTS.md).
