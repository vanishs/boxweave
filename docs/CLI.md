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
| `version` | BoxWeave's own VERSION file |
| `version --components` | Installed Incus client and Containarium CLI version strings; no MCP backend launch or server version claim |
| `version --verbose` | Recorded installation source commit, hashes, versions and installer difference indicator |
| `version --manifest PATH` | Same as `version --verbose`, reading the manifest from PATH instead of `/etc/boxweave/install-manifest.json` |
| `status` | Configuration availability and selected systemd service/timer states |
| `doctor` | Status, component versions, container detection and current AppArmor namespace observations |
| `doctor --api` | Adds a three-second unauthenticated loopback API reachability probe |
| `box inspect NAME --project PROJECT` | Exact local Incus instance, selected security/device metadata and configuration fingerprint |

`--json` is accepted before the command or after the leaf command. The
top-level `--version` flag prints the same VERSION value and exits. Status and
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

`boxweave client` generates a plain, editable enrollment script (`boxweave-setup.sh` or `boxweave-setup.ps1`) for Windows/macOS and Codex/Cursor/Claude Code/ZCode. It performs no enrollment on the server. The script header carries the connection variables (SSH host, port, user, pinned server key) in clear text for the user to review and edit; the enrollment logic is fetched from the pinned source commit at run time. Use `boxweave client --output FILE` to write the script instead of printing it. See [client setup](CLIENTS.md).

It reads public `/etc/boxweave/client.json` (schema 2), not credentials. Non-terminal input requires complete choices. Exit 0 means script generation, 2 means invalid/missing input or descriptor, and 130 means cancellation. Existing management/selftest commands retain their behavior.

Running the generated script with `--diagnose` (POSIX) or `-Diagnose` (Windows) performs a read-only diagnostic of an already enrolled client; it never enrolls or repairs. See [diagnostic stages, reports and limits](CLIENTS.md#independent-client-diagnostics).

## Independent web administrator

`boxweave web init-admin` and `boxweave web reset-password` require an interactive
local terminal and service-user/root access. Passwords are never arguments.
`boxweave web serve` runs the backend directly (it is the systemd service's
ExecStart and is not normally invoked by hand). `boxweave web status` reports the
configured origin and local backend listener, without opening provider
credentials; it exits 2 when the backend listener is unreachable. Use
`status`/`doctor` for Web systemd
observations. See [Web installation and security](WEB-UI.md).

Web status reports supported SQLite schema and Tenant capability gates without
opening Provider credentials. Doctor lists the same limits; live authenticated
Guard evidence is available from the console API, not inferred from CLI exit 0.
BoxWeave diagnostics report compiled adapter support and leave the current
Provider unprobed. The separate upstream Provider CLI adds --network-tenant and
network-policy access preview/apply; see [the upstream contract](TENANT-UPSTREAM.md)
and [Tenant operation](TENANTS.md). CLI exit 0 is not a live capability probe.

After an explicit installer `--upgrade-provider` run, `version --verbose --json`
includes install-time `tenant_network` observations: exact native contract,
local-LXC scope, Guard readiness and independent Web credential verification.
These observations do not replace a fresh authenticated probe or TCP test. See
[Provider upgrade](INSTALL-UPDATE.md#explicit-official-provider-upgrade-050).
