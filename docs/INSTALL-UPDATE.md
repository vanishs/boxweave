# Install, update and repair

The revised `install.sh` uses the same workflow for a fresh installation and a
repeat run. No separate `--update` or `--fix` flag is needed.

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh -o /tmp/boxweave-install.sh
sudo bash /tmp/boxweave-install.sh
```

Prefer a reviewed pinned ref for production. A copy of the installer downloaded
before these changes still contains its old behavior. Fetch the revised script
to obtain the fixes; an old local file cannot upgrade its own logic. This script
does not automatically execute a newer copy of itself.

## Settings and update scope

Precedence is explicit command-line flags, then explicitly supplied environment
variables, then saved `/etc/boxweave/config.json` settings, then automatic defaults.
The existing user wins over a different invoking sudo user. The installer retains
the API origin, secret-file and daemon paths, approved CIDRs, saved SSH endpoint,
sync interval, readiness timeout, source repository/ref and MCP release choice.
Unknown configuration keys are preserved. Older configurations without a saved
sync interval inherit it from the existing timer when available. Endpoint values
not recorded by an older installer must be supplied once if autodetection is
inappropriate. The chosen container public-key path is saved on this run and
reused thereafter; an incomplete saved pair is reported rather than regenerated.

Examples of deliberate overrides:

```bash
sudo bash /tmp/boxweave-install.sh --sync-interval 30 --ssh-host server.example.net
sudo bash /tmp/boxweave-install.sh --repo vanishs/boxweave --ref main
sudo env CONTAINARIUM_VERSION=v0.111.0 bash /tmp/boxweave-install.sh
sudo env CONTAINARIUM_VERSION=latest bash /tmp/boxweave-install.sh
```

`latest` is resolved on each run; an explicitly chosen release tag remains pinned
on later runs. This selects the MCP backend, not the Containarium daemon package.
All components use the selected source ref; a pinned old ref stays pinned until
explicitly changed. A source ref without the required files fails during staging.

Managed BoxWeave files are refreshed, missing files recreated, failed oneshot
services retried, and managed timers enabled and restarted to apply the current
configuration. Existing provider/Incus installations are reused. Provider package
upgrades and Incus upgrades remain separate operations. Daemon/CLI binary replacement
is available only through the explicit `--upgrade-provider` option below.
Container data, port routes and client enrollment keys are not replaced. A local
modification to a managed BoxWeave file is backed up and overwritten; systemd
drop-ins remain in place and can still affect service behavior.

## Failure and recovery behavior

- A host-local lock rejects concurrent runs of the revised installer.
- Legacy timer conflicts and invalid saved configuration are rejected early.
  Unsupported configuration schemas are not silently downgraded.
- All required Python components, the version file and the checksum-verified MCP
  backend are staged before installed BoxWeave files are replaced. A download,
  syntax or checksum failure leaves those installed files intact.
- Before publishing, the installer backs up its managed files under a fresh
  root-only `/var/backups/boxweave/update-*` directory. The `files/` tree mirrors
  existing absolute paths; `previously-absent.txt` records newly introduced paths.
  No token, private key, signing-secret contents or container data is copied.
- Only BoxWeave's own timers and oneshot services are quiesced for publication.
  Files are installed through same-directory temporary files and atomic rename.
  An already running MCP process retains its old executable; reconnect to use
  the new one. Managed-path symlinks and intentional unit masks require operator
  review instead of being overwritten.
- After publication, daemon reload, CLI execution, token generation and host-key
  synchronization must succeed before managed timers are restarted and checked.
  An error requests both timers to stop and retains
  `/etc/boxweave/update-in-progress` with the backup location. Fix the reported
  cause and rerun the command. If power is lost or the installer is killed, check
  unit state and rerun; the lock is released when its owning process exits.

Atomic publication is per file, not a transaction across the entire host. A
failure during publication may leave some new files installed. There is no
automatic global rollback: package operations, SSH/provider service enablement,
new JWT issuance and verified known-host updates are not undone. The backup is
for manual inspection/recovery; a repeat run repairs the managed file set and
checks services again. Existing corruption that cannot be inferred safely
(invalid JSON, missing signing secret, incomplete key pair or intentional mask)
is reported with an error rather than guessed away.

Reenrollment is normally unnecessary for an unchanged endpoint/account. Changing
them means editing the generated script's EDIT ME variables (or the managed
`Host boxweave` block in the client's SSH config) and rerunning it; agent
registrations use `ssh boxweave` and keep working. Reconnect the coding agent's
MCP session after a backend update.

## Verification

`tests/test_installer_rerun.py` runs a rewritten copy of the complete installer in
a temporary filesystem as a non-root Linux user. Fake commands stand in for
network downloads, Incus, APT, SSH and systemd. Tests cover repeated installation,
saved settings and identity selection, explicit upgrades, timer changes, missing
files, invalid configuration/source/checksum, service failure recovery, locking,
legacy interval migration and intentional masks. A real running executable is
also replaced atomically in the sandbox. These tests do not replace a live
Ubuntu installation/update integration test.

On 2026-10-22 the complete Linux regression suite passed all 50 tests in WSL
Ubuntu (exit 0). After moving the installer lock into a root-only runtime
directory, all 10 installer sandbox tests were rerun and passed (exit 0).
`bash -n install.sh` and `git diff --check` also passed. (Those counts are the
2026-10-22 snapshot; as of 0.3.0 the sandbox suite in
`tests/test_installer_rerun.py` has 15 tests and the repository suite has 219
test functions.) The revised installer
has not been executed against the live Ubuntu host's installed services.

## Client handoff in 0.2.0

Server installation retains the workflow above. It additionally stages the client generator, publishes `/etc/boxweave/client.json` atomically at 0644 (descriptor schema 2), and leaves privileged `config.json` at 0600. Client assets are not checksummed: generated scripts fetch enrollment logic from the pinned source commit at run time.

After successful completion, the final message prompts the human or agent to run `boxweave client` or `boxweave client --help`. It does not print client commands or start a wizard in an installer pipeline. Generated scripts fetch enrollment logic from the installed source commit at run time. That source ref must contain the new client files before this installer is deployed. See [client setup](CLIENTS.md).

## Mandatory Web UI staging

The Web UI is installed by default. All Web sources share the resolved source
commit. Its locked npm production build, SHA256-locked Python environment and
separate Caddy binary/config validation are staged before managed-file
publication. Web settings use flags, then environment, then saved web.json, then
defaults (HTTPS, port 18443); use --web-host when LAN detection is ambiguous.
Web publication happens after MCP timers resume and never stops those timers on
Web failure. The pending-update marker remains until the complete installation
passes. SQLite and Caddy CA data live outside replaceable releases and survive
reruns. Consistent database and managed-file backups support diagnosis, not an
automatic whole-host rollback. See [Web update/recovery](WEB-UI.md).

For a reviewed, unpublished local checkout, use `--source-dir "$PWD"`. The normal
remote source behavior remains the default; this explicit override is recorded
as `source.local_tree_staged=true` with component hashes and is not retained on
rerun. The recorded upstream commit is only a reference in this mode. Publish a
matching immutable ref before distributing modified client enrollment assets.

## Tenant metadata migration

Web schema 2 adds Tenant display metadata atomically, preserving administrator,
sessions, graphs, plans, jobs, settings and audit records. Repeated migration is
idempotent. The installer includes the new Python/TypeScript assets and records
schema 2 in provenance. Existing pre-migration SQLite backup and service-user
migration apply; a failure keeps old table data and the pending update marker.
Rollback requires matching software plus a private pre-migration backup; preserve
new writes before restoration. No Provider/Incus update or resource relabeling is
required. See [detailed recovery and limitations](TENANTS.md).

## Upstream native Tenant networking support

A/B Web adapters recognize the authenticated `native-network-tenant/v1` contract
merged in official Containarium main by PR #3165. The reviewed upstream source
pin and build instructions are in [the source profile](../patches/containarium/README.md).
The official [v0.120.1 release](https://github.com/FootprintAI/Containarium/releases/tag/v0.120.1)
contains the reviewed merge commit. The installer still
selects official MCP releases, retains existing Provider/Incus installations and
does not automatically replace the daemon with a source build. Update compatible
daemon/CLI/MCP components together through a separately approved Provider upgrade
from a containing official release, then reconnect MCP.

The independent Web token rotation recognizes the authenticated atomic capability
before adding network-policy read/write scopes. Web never borrows MCP credentials.
Unknown, old or incomplete capabilities retain the read-only gate. Account tenancy
is default off; exact account/shared identities remain separate. Existing Boxes
are not relabeled. Revision-checked ingress rollback preserves unrelated writes;
software/data rollback still needs compatible backups and reconciliation.

Historical v0.114.2 patches and fork test reports are not applied to current
upstream. Source adaptation and contract fixtures do not establish live TCP
isolation or a production deployment.

Codex client transport updates preserve existing `mcp_servers.boxweave.tools.*`
approval tables, including when replacing the BoxWeave transport registration.
Inline MCP tables and other nested BoxWeave tables still require explicit
migration; the setup does not remove them automatically.

## Explicit official Provider upgrade (0.5.0)

A normal rerun updates BoxWeave and its MCP backend but preserves the installed
Provider daemon. An old or downstream daemon can therefore still return
`boxweave.tenant-network/v1`; changing the MCP binary alone cannot change that
response. The Web adapter requires the official `native-network-tenant/v1`.

To align the Provider and MCP with the latest official release:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh -o /tmp/boxweave-install.sh
sudo bash /tmp/boxweave-install.sh --upgrade-provider
```

This explicit flag authorizes replacement of the configured local daemon binary
and a separate CLI binary, enabling Tenant Guard, and a brief Provider service
restart. It resolves `latest` once even if an earlier run saved a pinned MCP
release. To deliberately select a reviewed release instead:

```bash
sudo env CONTAINARIUM_VERSION=v0.120.1 bash /tmp/boxweave-install.sh --upgrade-provider
```

The reviewed floor is v0.120.1. The selected release must publish checksum-listed
Linux assets for the host architecture. Staging verifies SHA256 and reported
versions before publication; it does not build or apply downstream patches. All
BoxWeave/Web downloads and the Web production build finish before an existing
Provider is stopped. Provider binary downloads share a 600-second deadline.

The supported upgrade layout uses regular binaries under `/usr/local` and a
`containarium.service` ExecStart that directly invokes the configured daemon.
The conventional `containarium -> containariumd` alias is retained and follows
the upgraded daemon; a separate regular CLI is replaced from the same release.
Package-owned binaries, unexpected symlinks, wrappers and downgrades are refused
before publication. Custom layouts require a separately reviewed Provider update.

The installer preserves the existing service command, arguments and other
configuration. Its own drop-in sets `CONTAINARIUM_TENANT_GUARD=enforce`;
effective settings are checked through the running Provider rather than inferred
from the file. It does not upgrade Incus/APT packages, enable account tenancy,
relabel Boxes, reset ingress grants or publish ports. Existing cross-Tenant
permissions remain deliberate exceptions to default isolation. Existing Boxes
with a common explicit native key retain their shared membership.

Publication backs up the daemon, separate CLI and managed drop-in with BoxWeave's
other managed files and records `/etc/boxweave/provider-upgrade.json`. The daemon
is stopped only after backup and the pending-update marker. Files are replaced
atomically, the service is started/enabled, and the MCP credential is rotated by
the existing managed issuer. One 600-second readiness deadline covers an actual
non-root launcher initialize/initialized exchange, tools/list, capability and
Guard calls, and a read-only list_containers call. Required observations are the
exact native contract, local-LXC membership/shared-creation/atomic-policy support,
and fresh converged nftables Tenant Guard in ENFORCE mode. After Web publication,
its independent credential must also pass the Web capability, Guard and policy
read paths. Installation success never claims a TCP test.

`boxweave version --verbose --json` includes the install-time `tenant_network`
evidence and actual post-upgrade daemon version/hash. Reconnect MCP afterward;
existing agent processes retain their old backend until they reconnect. Failure
keeps the update marker and backup location. Inspect logs and the actual service
path before rerunning with the flag; the installer does not automatically roll
back binaries, Provider database migrations or already reconciled ACLs. Before
upgrading, keep an operator-managed compatible Provider database/configuration
backup in addition to BoxWeave's managed-file backup. Restoring binaries alone is
not a safe data rollback.

Acceptance after deployment: create two Boxes with the same native Tenant key
and one with a different key, verify member/Guard readback, then test fresh TCP
connections in both directions. Same-key Boxes should communicate; different-key
Boxes should be denied unless an explicit ingress grant permits them. Gateway,
proxy/public routes, egress controls and established connections are separate
paths. Live acceptance requires its own agreed resource/cleanup plan.

### Confirming the installed and running Provider versions

Containarium uses `containariumd version`, not `containariumd --version`.
BoxWeave 0.5.1 fixes the staged-release check and daemon provenance reader to use
that subcommand. The 0.5.0 staging failure occurs before Provider publication;
fetch the corrected installer and rerun the same explicit upgrade command.

```bash
containariumd version --verbose
sudo systemctl show containarium.service -p ExecStart -p MainPID
pid=$(systemctl show containarium.service -p MainPID --value)
if [ "$pid" -gt 0 ]; then
  sudo "/proc/$pid/exe" version --verbose
fi
```

The first command inspects the executable on PATH. The `/proc` command inspects
the binary inode used by the active daemon, including when a replaced executable
is still running. Compare its version with the current official release, and
inspect ExecStart if the versions differ. `containariumd version --check` can
also query GitHub for release drift. Version equality alone does not establish
native Tenant readiness; the authenticated capability/Guard checks still apply.
