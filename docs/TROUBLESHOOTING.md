# Installation troubleshooting

BoxWeave's Ubuntu installer invokes the official Containarium installer when
necessary. On a new machine, Incus packages, image downloads, and
Containarium core containers can take several minutes to initialize.

## Safer download and execution

A plain `curl URL | sudo bash` pipeline can conceal a curl download failure
unless the calling shell enables `pipefail`. Prefer downloading first:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh \
  -o /tmp/boxweave-install.sh
sudo bash /tmp/boxweave-install.sh
```

For long first-time package installations over SSH, consider running the
installer inside a terminal multiplexer such as `tmux`. Disconnecting a normal
SSH session may interrupt package setup. Never remove APT lock files while a
package manager is running.

## Zabbly APT keyring permission error

**Symptom:** `NO_PUBKEY 82CC8797C838DCFD` together with messages that
`/usr/share/keyrings/zabbly-incus.gpg` is not readable or has an unsupported
file type.

**Cause (observed):** BoxWeave previously invoked the upstream installer while
`umask 077` was active, causing root-only APT keyring and source-list files.
The corrected installer runs upstream package setup with `umask 022` and
verifies the known Zabbly key fingerprint before fixing permissions from an
interrupted previous attempt.

Do not bypass APT signature verification or configure a repository as trusted
without signature checking. If the installed key fingerprint differs from
`4EFC590696CB15B87C73A3AD82CC8797C838DCFD`, stop and investigate.

## Incus installed but no storage or network

**Symptom:** `incus version` works but `incus storage list` is empty and
`incus network list` shows no managed bridge. Containarium's installer may
skip initialization when Incus is already recent enough.

**Correction:** BoxWeave now initializes Incus automatically if there is no
storage pool **and no existing containers**. It does not reinitialize an
Incus deployment that already contains containers.

On fresh machines, `incus admin init --auto` commonly selects a `dir` storage
pool. This permits basic operation but provides weaker storage-I/O isolation
than a properly configured ZFS or btrfs pool. For production multi-tenant use,
choose a suitable storage design before creating workloads. BoxWeave does not
silently convert or delete storage pools.

## Daemon active but port 8080 unavailable

**Symptom:** `containarium.service` is active, but the REST API is not yet
listening, and the logs say `Waiting for core containers to be ready...`.

**Correction:** The installer waits up to 900 seconds by default, displaying
periodic progress messages. Increase the limit when needed:

```bash
sudo bash /tmp/boxweave-install.sh --api-ready-timeout 1800
```

If it ultimately times out, inspect these read-only diagnostics:

```bash
sudo systemctl status containarium --no-pager
sudo incus list
sudo journalctl -u containarium -n 80 --no-pager
sudo ss -ltnp | grep ':8080' || true
```

Do not repeatedly rerun or delete half-created core containers while the daemon
is still initializing. First determine whether the service is progressing.

## Interrupted APT/Incus installation

If an SSH session closes or the installer receives Ctrl+C, inspect processes
and package integrity before retrying:

```bash
ps -eo pid,ppid,stat,cmd | grep -E '[a]pt-get|[d]pkg|[b]oxweave|[c]ontainarium'
sudo dpkg --audit
sudo incus storage list
```

Avoid deleting APT locks or killing active package configuration processes.
When no installers are running and `dpkg --audit` is clean, rerun the BoxWeave
installer; existing Incus and Containarium components are reused where safe.

## Installation completed, but MCP is not available

```bash
ls -l /usr/local/bin/boxweave-mcp
sudo systemctl is-active boxweave-hostkeys.timer
sudo systemctl is-active boxweave-token.timer
sudo journalctl -u boxweave-token.service -n 40 --no-pager
```

The server installation must finish before setting up a client. The client
script still requires an existing SSH login for first-time public-key
registration. The BoxWeave SSH key itself is restricted to starting MCP.
