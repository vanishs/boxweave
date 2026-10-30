# Application installation, update and repair

This operator guide covers verified application packages on a **prepared Linux x86_64 host with CPython >= 3.12 and systemd**. It does not promise a working one-command installation from a blank server, nor install/upgrade upstream infrastructure, repair SSH/firewall, or configure public TLS. For the current host entry and exact prerequisites, read [host installation](HOST_BOOTSTRAP.md).

Choose and retain an explicit private installation location, for example `APP_PREFIX="$HOME/boxweave"`; use this exact value consistently with all installation, administrator and health commands. Existing installations may use older managed directories and service identifiers; **do not rename or move them without a reviewed migration and consistent database backup**. Prior checkpoints are in [installation history](INSTALL_HISTORY.md).

## Trust and preparation

Obtain a reviewed standalone `bwmini/distribution.py` and the expected package SHA256 through an authenticated operator channel. A digest downloaded alongside untrusted bytes is not independent authenticity evidence. The ZIP archive embeds exact public-file hashes, source commit/state, locked offline wheels, built UI and public client helpers. Deterministic ZIP metadata makes repeated builds from identical inputs reproducible. Private runtime/state, unknown client files, keys, JWTs and sign-in data are excluded. This is an operator-trusted digest workflow, not a publisher signature or automatic updater.

Build inside the assigned Linux development box, using the existing exact wheel lock and built UI:

```sh
python3 -m pip download --require-hashes --only-binary=:all: --no-deps \
  -r requirements.lock --dest /private/build/wheels
npm --prefix web ci
npm --prefix web run build
python3 bwmini/distribution.py build --root . --wheels /private/build/wheels \
  --output /private/build/boxweave.zip \
  --source-commit REVIEWED_40_CHARACTER_COMMIT --source-state clean
```

Use `modified` instead of `clean` when any packaged source differs from that commit. The caller supplies the reviewed source record; the builder does not pretend that a source string authenticates itself. Keep the printed package digest and the independently reviewed installer digest with the delivery record.

Prepare a private application account and an operator-reviewed JSON configuration with `mode=mcp`, exact provider/backend/project, authorized targets/lifecycle/access profiles, an independently authorized application launcher, absolute database path, loopback host and a nonconflicting port. Preserve each nested host's managed public key separately. Credential files stay outside releases; never copy a Windows/WSL transport private key or read a Provider JWT into setup output. The application account must already have a usable systemd user bus (`XDG_RUNTIME_DIR`); the installer does not enable linger or change account/boot policy.

The listener stays127.0.0.1. For a local HTTP test, use an exact loopback HTTP origin and `secure_cookie=false`. For a deployed HTTPS frontend, supply the exact HTTPS origin and `secure_cookie=true`, and configure a separately reviewed TLS reverse proxy to the loopback listener. The installer does not silently create a public Caddy listener, obtain certificates or install a trust root. Public exposure requires its own approval. Existing HTTPS configurations and provider services remain untouched.

## Install and first administrator

```sh
APP_PREFIX="$HOME/boxweave"
python3 /trusted/distribution.py install --archive /trusted/boxweave.zip \
  --sha256 REVIEWED_ARCHIVE_SHA256 --prefix "$APP_PREFIX" \
  --config /private/application.json --activate
```

Run as the application user, not root. The private prefix is dedicated to this install. Configuration is saved byte-for-byte as `config.json`; generated `runtime.json` changes only the managed UI path. Database, credentials and unknown operator settings stay outside immutable releases. Named API/worker user units are derived from the prefix and replace only units whose previous managed contents are verified. No `/usr/local/bin/boxweave` command or daily Agent configuration is replaced.

A new database requires explicit local administrator setup. The first install may leave `ADMIN_SETUP_REQUIRED` and its marker after publishing the verified application. Initialize using the verified release, entering the password only at the local terminal prompts:

```sh
cd "$APP_PREFIX/current"
.venv/bin/python -m bwmini init-admin --config ../runtime.json
```

Then repeat the same install command to finish verification. For a later intentional password reset, run `reset-admin` with the same config and type `RESET` at its confirmation prompt. It revokes all browser sessions and clears login attempt state in one transaction without launching MCP. No password command-line argument or plaintext report is generated.

## Update, failure and repair

Repeat `install` with the reviewed new package/digest. Saved settings are reused when `--config` is omitted; conflicting supplied settings are refused rather than silently merged. Deliberate setting changes require reviewing/editing the protected saved config while quiesced. Different installs must use distinct state paths and ports.

The installer verifies all archive members before running package code, rejects traversal/links/duplicate members, takes an install lock, stages an immutable release, stops only its owned API/worker units, builds a fresh environment from offline hash-locked wheels, takes the worker lock and a consistent SQLite backup, validates candidate state/config, then atomically switches `current`. Old environments and backups are retained for review. During a failure, `update-in-progress.json` records the last stage, package, previous release and backup. Configuration acknowledgment is not authenticated MCP evidence: the marker clears only after required service/HTTP checks and a real initialize/discovery/list_containers call, plus initialized local administrator state.

For a known installation failure, inspect the marker, services and diagnosis; correct the specific cause and rerun the same reviewed package. Do not start a competing installer or delete locks. Changed managed payloads/foreign services, invalid configuration, a busy worker and downgrades are refused. An inactive invalid owned unit with MainPID0 has no running process to stop; unknown service state is not treated as quiesced. The first candidate's incorrect WorkingDirectory quoting was found and repaired this way, with original evidence preserved.

Before publication, the old pointer remains. After publication, a failed health check retains the new pointer and marker; there is no whole-host rollback promise. Downgrade needs a reviewed pre-upgrade consistent database backup and compatible release/environment restored while API/worker are stopped. Never copy only a live SQLite database file over active state. A corrupted managed release needs explicit quarantine/review, not an insecure trust bypass. `--quiesced` supports files-only installation after the operator stops all app writers; it explicitly reports services_not_activated and is not autostart evidence.

```sh
python3 /trusted/distribution.py health --prefix "$APP_PREFIX" --mcp
```

Machine states distinguish starting, incomplete, failed and ready. HTTP401 is reachable/unauthenticated; MCP authentication is reported separately. Enabled/running user services prove current user-manager operation, not LXC boot.autostart or a fresh-host reboot. Successful inactive oneshots and required waiting/enabled timers are assessed separately in supporting-provider evidence. Web can be stopped independently while the existing restricted SSH MCP remains usable.

Actual evidence and recovery limits: [TASK-012/013 report](verification/TASK_012_013_RELEASE.md). Do not rerun its destructive lab driver without inspecting its original state and exact authorization.

## Source 0.12.0 host entry (remediation, not yet a released host-acceptance claim)

`install.sh` is a preflight/coordinator for operator-reviewed offline artifacts. See [host bootstrap boundaries](HOST_BOOTSTRAP.md) for its `plan`/`apply` arguments and Provider gating. Existing standalone app package, backup, user services and incomplete-marker behavior above remain the implementation of app update/repair. The host entry does not yet assemble credentials/MCP launcher, configure a fresh systemd user bus or expose HTTPS; a full new-host end-to-end pass is still required.

Host coordinator 0.13.4 requires `--distribution-sha256` as an additional independent review gate for `distribution.py`. Do not treat an archive digest as verification of the Python installer code executing it.

## Optional dedicated application-account setup

On an independently prepared Provider, the explicit [application bootstrap](HOST_BOOTSTRAP.md#independent-application-bootstrap) can now create the separate app user/bus, launcher and scoped credential rotation. It retains the same non-root package installer, first-admin, backup and repair checks. It does not publish a release, provision the Provider or replace existing accounts/deployments.


## Fresh-host streamed entry

The README command selects the source-bound release's `online.py` coordinator. It stages/validates all artifacts, provisions only an inspected empty local host, then delegates app installation to the dedicated non-root account. Repeat the same command for managed app update/repair after inspecting a failed phase and any original package processes; existing Provider binary/version mismatches remain refused. No old data migration or arbitrary environment adoption occurs.

After a ready result, generate a client identity on the **client** machine, enroll only its public half with the printed root helper command, and run that helper's `client-config --identity ABSOLUTE_CLIENT_KEY --known-hosts ABSOLUTE_CLIENT_KNOWN_HOSTS`. Deliver the resulting public JSON through the authenticated administrative channel. Put the returned pin line in the selected known_hosts file and configure the returned `mcpServers.boxweave` entry. Start a standard MCP client and verify initialize/initialized, tools/list and an actual read before creating workloads. Supply the returned managed workload public key explicitly in create_container.ssh_keys; it belongs to this nesting layer.

Installation's app health/MCP read is separate from a later client's actual creation/connect/Python test. See [current test progress and exact outcomes](verification/STREAMED_INSTALL_TEST.md).


## Streamed installation with an existing Containarium

On **Linux x86_64 with CPython >= 3.12 and systemd**, the default streamed entry can reuse an existing Incus + Containarium deployment. The installed Provider must expose its authenticated REST API on loopback and have a root-private JWT signing-secret file. BoxWeave inspects its availability and the existing Incus pool, but never installs/upgrades/restarts the external Provider, rewrites its secret, or replaces its network/storage configuration. BoxWeave-owned application bootstrap, user/timer, trust helper and app updates are still carried out.

The ordinary zero-argument streamed command uses Provider URL `http://127.0.0.1:8080`, secret path `/etc/containarium/jwt.secret` and installed `containariumd`. For a different local Provider setup, download and review the matching release `install.sh`, then execute `sudo bash install.sh --provider-url http://127.0.0.1:PORT --provider-secret-file /reviewed/secret/path --provider-cli /absolute/path/to/containariumd`. The custom options are only for the streamed entry; the independent `plan/apply` mode is unchanged. Existing Incus **without** Containarium is also supported when its storage pool is already configured: the installer preserves the pool and provisions the fixed managed Provider.

The installer saves the chosen ownership mode on first installation. Repeating the command updates or repairs only BoxWeave-owned artifacts and verifies the same external Provider identity. Changes to the external Provider binary/secret require a separate operator-reviewed adoption process, not an implicit upgrade. Partial existing Provider installations, foreign BoxWeave accounts/state, and empty Incus pools are refused safely. This does not import historical 0.5.1 data.

**Minimum application runtime: Linux x86_64 and CPython >= 3.12, with systemd and reviewed Provider prerequisites. No Ubuntu-version or distribution-name gate remains.** If packages are missing, APT is used only when apt-get and a compatible suite are available; otherwise prepare reviewed dependencies/Provider first. Ubuntu 24.04+ is recommended, not required. The release contains hash-checked offline wheels for CPython 3.12, 3.13, 3.14 and 3.15. A later Python may pass the version gate but lack a compatible bundled wheel: it must fail safely rather than download an unreviewed dependency. Ubuntu 26.04 is accepted by source checks but still needs live host and reboot acceptance. See [implementation and test queue](verification/EXISTING_PROVIDER_INSTALL.md).


## Cross-version release wheel verification (0.17.2)

The CI build downloads the locked wheels for x86_64 CPython 3.12–3.15 using `pip download --python-version` with `--require-hashes` and `--only-binary=:all:`, then packs all exact verified wheels into one source-bound release ZIP. On Python 3.14 it also creates a real virtual environment, installs **offline only** from that ZIP using the unchanged requirements lock, and imports Flask/MarkupSafe/Waitress. The application installer never downloads packages at activation time. This CI evidence is still not a claim about a live Ubuntu 26.04 host.


## Streamed Windows Codex client setup (0.17.3)

After BoxWeave's host installer reports `ready`, run `boxweave client` for a terminal wizard or `boxweave client --agent codex --transport wsl --admin-user example-admin --output /home/example-admin/boxweave-setup.ps1` for an explicitly selected non-interactive generator. The output is a public-only PowerShell script, never a private key or Provider secret. Copy it over an authenticated administrative channel and run on Windows Codex using an existing WSL distribution. Administrator-approved public-key enrollment uses the already-installed `online.py enroll` (not direct `authorized_keys` rewriting). Only after an actual MCP read passes will it safely update the Windows ChatGPT/Codex desktop user-level `config.toml`, without running a separate Codex CLI. Administrator SSH enrollment defaults to password-only auth to avoid excess-key attempts; explicit key-only login is supported. This is distinct from the older manual `client-config` fallback. See [client setup](../clients/README.md) for review, `-NoEnroll`, `-AlreadyEnrolled` and `-Replace` modes.
