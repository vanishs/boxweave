# BoxWeave management CLI

The management CLI uses Python 3's standard library and is independent of the MCP
transport. It performs read-only observations: no installation, repairs, service
starts, instance lifecycle actions, or package upgrades.

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
| `version` | BoxWeave's own VERSION file, currently an unreleased development version |
| `version --components` | Installed Incus client and Containarium CLI version strings; no MCP backend launch or server version claim |
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

Exit code 0 means the requested observation completed; it is not a security
certification or proof that every service is ready. Exit code 1 indicates a
reported failed service, requested API probe failure, or failed instance query.
Exit code 2 indicates incomplete observations or invalid command arguments.
Inactive and starting services are reported as such rather than called corrupt.

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
