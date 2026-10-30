# 0.17.9 install → ordinary-user CLI → Web → client acceptance

**Status:** Code and CI fixture checks only. This is an operator checklist, not
a claim of live acceptance. Run on an explicitly authorized Ubuntu BoxWeave
server and a separately authorized Windows/WSL client. Retain original logs
and host identity; never reset the Provider, Incus, SSH identities, signing
secrets or existing user workloads merely for this check.

## A. Host preparation and installation

1. Record Ubuntu version, exact host, shell user and existing Provider/Incus
   identities. Review the currently installed state before any reinstall.
2. Install the new verified **v0.17.9** release via the README's source-bound
   installer. Root is required for host installation/system services; *do not*
   skip the release-asset SHA256/source verification or rerun a failed privileged
   installer without checking its last durable phase.
3. Confirm the installer exits 0, emits `state=ready`, and prints a plain
   `Installation complete — next steps` section containing the detected Web
   loopback port, `boxweave`, `boxweave client` and first-admin password
   **path** (never password contents). Confirm Web login separately.

## B. Ordinary-user and permission checks

Run as an **ordinary SSH human account**, with no `sudo`:

```sh
command -v boxweave
boxweave
boxweave --help
boxweave version
boxweave status
boxweave doctor
boxweave box
boxweave client --help
```

Acceptance: each command invokes the **root-owned** installed CLI entry;
no `cd: can't cd to /home/boxweave` error, no implicit privilege prompt,
`--help` mentions `client`, and `box` without arguments displays its
subcommands. `status` must not claim authenticated MCP when it only checked
HTTP. The service account private home remains mode 0700:

```sh
sudo stat -c '%U %a %n' /home/boxweave /usr/local/bin/boxweave \
  /usr/local/lib/boxweave/default/host_cli.py \
  /usr/local/lib/boxweave/default/client-public.json
```

Expected: root-owned CLI code and 0644 public descriptor, private app home
0700. Check the installed launcher uses isolated root-owned `host_cli.py`,
never directly executes `/home/boxweave/boxweave/current/.venv/bin/python`
under root. Inspect the public descriptor's keys only; **never** paste private
keys, administrator passwords or tokens into reports.

Run `boxweave box inspect EXACT_NAME` only for an authorized, known existing
instance. A normal user lacking Incus authorization should receive a bounded
error; an authorized operator may explicitly rerun with `sudo`. Do not grant
broad Incus group membership or passwordless generic sudo to pass the test.

## C. Web access

On the Ubuntu server, `boxweave` must show
`http://127.0.0.1:PORT` with the actual detected port. That is not a LAN or
public Web URL. From a trusted client, establish an SSH tunnel:

```sh
ssh -N -L LOCAL_PORT:127.0.0.1:SERVER_PORT ADMIN@REACHABLE_UBUNTU_HOST
```

Use matching ports (normally 18640) and browse
`http://127.0.0.1:LOCAL_PORT` on the **client**. Check that Web login
succeeds with the separately retrieved initial administrator secret and that
the UI displays a valid read-only project listing. Do not include Web
credentials, cookies or bearer tokens in evidence. Direct LAN exposure must
remain absent unless separately approved.

## D. Client setup and real authenticated MCP

As the ordinary Ubuntu user, run:

```sh
boxweave client --agent codex --transport wsl \
  --admin-user AUTHORIZED_SSH_ADMIN --output "$HOME/boxweave-setup.ps1"
```

Verify the output is owned by that account and mode 0600, contains only
public SSH/managed workload identities and a reviewed PowerShell setup script.
Copy it securely to the Windows machine and run from PowerShell with the
correct WSL distro. The client's first key enrollment requires explicit
existing administrator approval; client generator must never request sudo.
Check the generated client's actual SSH trust pin, MCP initialize, tools/list,
and a real read-only list_containers call. Confirm ChatGPT/Codex configuration
references the registered `boxweave` server, not a hardcoded human login.

Optionally, **only with explicit destructive-operation authorization**, create
a named disposable Ubuntu test box with the server's full managed workload
public key, run the previously agreed Python hello-world smoke test, inspect
exact instance identity, and record exit 0 and exact `hello world` output.
Do not use the ordinary installation acceptance to automatically create,
delete or reset existing workloads.

For an administrator-only authenticated health check, use
`sudo boxweave doctor --mcp`, then verify it ran the app probe with the
dedicated `boxweave` uid (never as root) and did not print credentials.
Reboot/boot-autostart and off-host Web/TLS exposure require independent
validation; do not infer them from an immediate postinstall success.

## Report

Record sanitized command exit status, source release commit and tags, versions,
observed permissions, loopback Web reachability, client MCP result and any
failure phase. Statuses: `passed`, `failed` or `not_run` for each
section A–D. A code-only result cannot be labelled `passed` for live
install/Web/WSL/MCP acceptance.
