# BoxWeave installed CLI: permissions and next steps

The **installed** `/usr/local/bin/boxweave` is a root-owned shell launcher
that runs `/usr/bin/python3 -I /usr/local/lib/boxweave/default/host_cli.py`.
It never changes directory into the private `/home/boxweave` account,
and `sudo boxweave` never executes app-user-writable Python source as root.
The separate `boxweave.sh` is a source/package-local development tool,
not the installed entry point.

## Day-one commands (ordinary login)

```sh
boxweave                 # Read the installed Web URL, tunnel and next steps
boxweave --help          # Display the complete installed host CLI
boxweave version         # Published release identity only
boxweave status          # Loopback Web reachability; NOT authenticated login/MCP
boxweave doctor          # Read-only host/systemd state; no sudo
boxweave client          # First step: client PowerShell setup; prints script on stdout
boxweave client --help   # More client setup options
boxweave box             # Show inspect/diagnose help
boxweave box inspect NAME --project default
```

The host publishes a **root-owned, non-secret, non-writable**
`/usr/local/lib/boxweave/default/client-public.json` containing the
public SSH server key, managed workload **public** key, endpoint and Web
loopback port. The descriptor is not a credential or proof that a service is
healthy. All sensitive runtime config, token, private key and the initial
Web password remain in private locations. No ordinary CLI invocation
automatically invokes `sudo`.

`boxweave box inspect` and `boxweave box diagnose` issue read-only
Incus requests, but Incus access is itself privileged; a normal user
without authorization receives a bounded error. An authorized administrator
may explicitly invoke `sudo boxweave box inspect NAME`. Do **not** add
an ordinary human user to Incus administration groups merely to hide this
boundary.

`sudo boxweave doctor --mcp` is **optional**: it drops to the dedicated
`boxweave` service identity and performs a real authenticated app-side
read-only MCP probe. `boxweave doctor --mcp` without admin privileges
shows a reason and a next step instead. Neither a checked public Web endpoint
nor a running timer proves an authenticated Agent connection or reboot
readiness.

## Web access and first client

The Web listener binds `127.0.0.1`, not a LAN/public interface. Use the
actual port displayed by `boxweave`, for example:

```sh
# Run on your own computer, not inside the server's boxweave service account:
ssh -N -L 18640:127.0.0.1:18640 YOUR_HOST_ADMIN@YOUR_LINUX_HOST
```

Then open `http://127.0.0.1:18640` in the local browser. Only an authorized
host administrator may read the private first-password file
`/home/boxweave/boxweave/initial-admin-password`; it is never printed.

```sh
boxweave client --agent codex --transport wsl \
  --admin-user YOUR_HOST_ADMIN --no-replace > boxweave-setup.ps1
```

Transfer the script to Windows through an authenticated channel; review and
run it in PowerShell. Enrollment uses that **human SSH administrator**
to authorize a newly generated *public* transport key (with explicit
`sudo` at enrollment), then tests real MCP initialize/discovery/read.
The `boxweave` Linux account is the isolated application identity; it
must not be silently substituted for the host SSH administrator.
Client generation exports no private keys and does not change the Provider.

## Privilege/ownership audit

| Action or file | Boundary |
| --- | --- |
| `install.sh` streamed empty-host install | Explicit root required for APT, service units, account setup, SSH authorization and Incus/Provider changes |
| `install.sh plan` | Read-only; no root required when prerequisites are readable |
| App release, database, tokens, private SSH key | Dedicated `boxweave` account and private directories (0700/0600 or tighter) |
| Installed CLI launcher/helpers and public descriptor | Root-owned; code 0755, public descriptor 0644; never user-writable |
| `boxweave` / `status` / `doctor` / `client` | Ordinary user; no implicit elevation |
| `boxweave box inspect/diagnose` | Read-only but requires existing Incus authorization |
| `sudo boxweave doctor --mcp` | Explicit administrative operator; starts the probe as `boxweave`, not root |
| `online.py enroll` | Explicit administrator approval to update forced-command authorized public SSH keys |
| `hostkeys.py`, `nesting.py`, `bootstrap.py` | Host privilege boundaries preserved; never made generally writable or silently escalated |

Do not chmod the service home to 755, make secret files public, or
allow passwordless broad sudo/Incus access as a workaround.

## Verification scope

Static tests exercise public descriptor trust, normal-user generator, exact
Incus request names, no implicit escalation and onboarding help. An Linux
operator still needs to verify the installed release and service paths,
tunnel/browser login, Windows enrollment and real `MCP initialize/tools/read`,
plus a reboot check. See [installation](INSTALL.md) and the
[existing live test record](verification/STREAMED_INSTALL_TEST.md).
