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
upgrades, daemon binary replacement and Incus upgrades are separate operations.
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
them can require rerunning the generated client setup. Reconnect the coding
agent's MCP session after a backend update.

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

Server installation retains the workflow above. It additionally stages the client generator and source hashes, publishes `/etc/boxweave/client.json` atomically at 0644, and leaves privileged `config.json` at 0600. Client source hashes are included in provenance.

After successful completion, the final message prompts the human or agent to run `boxweave client` or `boxweave client --help`. It no longer prints Windows/Codex-specific commands or starts a wizard in an installer pipeline. Setup downloads are pinned to the installed source commit and checked against recorded SHA256. That source ref must contain the new client files before this installer is deployed. See [client setup](CLIENTS.md).

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

## Optional native Tenant networking Provider patch

A/B Web adapters and assets are staged by the normal installer, but the baseline
Provider pin and installed daemon are unchanged. Follow the explicit source
[patch/build/deployment gate](TENANT-UPSTREAM.md) to evaluate the candidate.
Never substitute a candidate binary without reviewing its exact base/patch hashes
and separately authorizing Provider replacement. Update compatible daemon/CLI/MCP
components together and reconnect MCP. Existing Providers retain the old feature
gate and original Web-token scopes.

The normal Web token rotation discovers the recognized capability contract and
only then adds network-policy read/write scopes. Web never borrows MCP credentials.
Tenant key binding reuses SQLite schema 2; it does not relabel existing Boxes.
Policy rollback requires fresh revision-checked inverse ingress edits, preserving
concurrent egress/deny changes. Software/data rollback still requires matching
private backups and reconciliation of writes since backup.

The current native Tenant candidate is pinned in [the Provider source profile](../patches/containarium/README.md). The historical v0.114.2 patch is not applied to newer forks. Candidate binaries require separate reviewed deployment; the normal installer selects official MCP releases.
