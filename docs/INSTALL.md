# Install, update and repair the application

0.9.1 / phase6: schema7 preserves project/binding history using a local unbound marker; migration6→7 and its consistent schema6 backup were tested on an independent copy.0.9.0 binaries refuse7. Project unbinding never deletes a project/environment/receipt or cancels dispatched work. Use the installer/backup boundary before downgrade. The retained primary test installation remains0.9.0 until explicitly updated; source changes alone are not deployment.

After login, **Client setup** generates the four supported public configurations and a local PowerShell enrollment script. Use an already authorized SSH identity and prepared private profile directory. Enter paths/public server pin/exact Agent version and public ZCode catalog digest; do not enter credential contents. Save both downloads under their exact names in one new private folder. The script requires explicit absolute Python/helper paths, verifies helper and descriptor SHA256 before the existing local registrar, and preserves the selected replacement flag. It never installs dependencies, authorizes a key, invokes inference or executes on the Web server. Operator review/trusted delivery remain required. **Projects** now exposes reviewed identity, Edit binding and Unbind; unbound projects can be explicitly rebound.

This is the standalone application package on main, not the legacy BoxWeave/Containarium installer. It installs neither Provider nor Incus, changes no host SSH/firewall settings, and publishes no port. Tested target: non-root Ubuntu24.04, Linux x86_64, CPython3.12 with venv/ensurepip and a working systemd user manager. Windows-hosted Agent enrollment is a separate [client workflow](../clients/README.md). Other server runtimes are unsupported by this wheel lock.

## Trust and preparation

Obtain a reviewed standalone `bwmini/distribution.py` and the expected package SHA256 through an authenticated operator channel. A digest downloaded alongside untrusted bytes is not independent authenticity evidence. The archive embeds exact public-file hashes, source commit/state, locked offline wheels, built UI and public client helpers. Deterministic ZIP metadata makes repeated builds from identical inputs reproducible. Private runtime/state, unknown client files, keys, JWTs and sign-in data are excluded. This is an operator-trusted digest workflow, not a publisher signature or automatic updater.

Build inside the assigned Linux development box, using the existing exact wheel lock and built UI:

```sh
python3 -m pip download --require-hashes --only-binary=:all: --no-deps \
  -r requirements.lock --dest /private/build/wheels
npm --prefix web ci
npm --prefix web run build
python3 bwmini/distribution.py build --root . --wheels /private/build/wheels \
  --output /private/build/boxweave-v2.zip \
  --source-commit REVIEWED_40_CHARACTER_COMMIT --source-state clean
```

Use `modified` instead of `clean` when any packaged source differs from that commit. The caller supplies the reviewed source record; the builder does not pretend that a source string authenticates itself. Keep the printed package digest and the independently reviewed installer digest with the delivery record.

Prepare a private application account and an operator-reviewed JSON configuration with `mode=mcp`, exact provider/backend/project, authorized targets/lifecycle/access profiles, an independently authorized application launcher, absolute database path, loopback host and a nonconflicting port. Preserve each nested host's managed public key separately. Credential files stay outside releases; never copy a Windows/WSL transport private key or read a Provider JWT into setup output. The application account must already have a usable systemd user bus (`XDG_RUNTIME_DIR`); the installer does not enable linger or change account/boot policy.

The listener stays127.0.0.1. For a local HTTP test, use an exact loopback HTTP origin and `secure_cookie=false`. For a deployed HTTPS frontend, supply the exact HTTPS origin and `secure_cookie=true`, and configure a separately reviewed TLS reverse proxy to the loopback listener. The installer does not silently create a public Caddy listener, obtain certificates or install a trust root. Public exposure requires its own approval. Existing HTTPS configurations and provider services remain untouched.

## Install and first administrator

```sh
python3 /trusted/distribution.py install --package /trusted/boxweave-v2.zip \
  --sha256 REVIEWED_ARCHIVE_SHA256 --prefix "$HOME/boxweave-v2" \
  --config /private/application.json --activate
```

Run as the application user, not root. The private prefix is dedicated to this install. Configuration is saved byte-for-byte as `config.json`; generated `runtime.json` changes only the managed UI path. Database, credentials and unknown operator settings stay outside immutable releases. Named API/worker user units are derived from the prefix and replace only units whose previous managed contents are verified. No `/usr/local/bin/boxweave` command or daily Agent configuration is replaced.

A new database requires explicit local administrator setup. The first install may leave `ADMIN_SETUP_REQUIRED` and its marker after publishing the verified application. Initialize using the verified release, entering the password only at the local terminal prompts:

```sh
cd "$HOME/boxweave-v2/current"
.venv/bin/python -m bwmini init-admin --config ../runtime.json
```

Then repeat the same install command to finish verification. For a later intentional password reset, run `reset-admin` with the same config and type `RESET` at its confirmation prompt. It revokes all browser sessions and clears login attempt state in one transaction without launching MCP. No password command-line argument or plaintext report is generated.

## Update, failure and repair

Repeat `install` with the reviewed new archive/digest. Saved settings are reused when `--config` is omitted; conflicting supplied settings are refused rather than silently merged. Deliberate setting changes require reviewing/editing the protected saved config while quiesced. Different installs must use distinct state paths and ports.

The installer verifies all archive members before running package code, rejects traversal/links/duplicate members, takes an install lock, stages an immutable release, stops only its owned API/worker units, builds a fresh environment from offline hash-locked wheels, takes the worker lock and a consistent SQLite backup, validates candidate state/config, then atomically switches `current`. Old environments and backups are retained for review. During a failure, `update-in-progress.json` records the last stage, archive, previous release and backup. Configuration acknowledgment is not authenticated MCP evidence: the marker clears only after required service/HTTP checks and a real initialize/discovery/list_containers call, plus initialized local administrator state.

For a known installation failure, inspect the marker, services and diagnosis; correct the specific cause and rerun the same reviewed package. Do not start a competing installer or delete locks. Changed managed payloads/foreign services, invalid configuration, a busy worker and downgrades are refused. An inactive invalid owned unit with MainPID0 has no running process to stop; unknown service state is not treated as quiesced. The first candidate's incorrect WorkingDirectory quoting was found and repaired this way, with original evidence preserved.

Before publication, the old pointer remains. After publication, a failed health check retains the new pointer and marker; there is no whole-host rollback promise. Downgrade needs a reviewed pre-upgrade consistent database backup and compatible release/environment restored while API/worker are stopped. Never copy only a live SQLite database file over active state. A corrupted managed release needs explicit quarantine/review, not an insecure trust bypass. `--quiesced` supports files-only installation after the operator stops all app writers; it explicitly reports services_not_activated and is not autostart evidence.

```sh
python3 /trusted/distribution.py health --prefix "$HOME/boxweave-v2" --mcp
```

Machine states distinguish starting, incomplete, failed and ready. HTTP401 is reachable/unauthenticated; MCP authentication is reported separately. Enabled/running user services prove current user-manager operation, not LXC boot.autostart or a fresh-host reboot. Successful inactive oneshots and required waiting/enabled timers are assessed separately in supporting-provider evidence. Web can be stopped independently while the existing restricted SSH MCP remains usable.

Actual evidence and recovery limits: [TASK-012/013 report](verification/TASK_012_013_RELEASE.md). Do not rerun its destructive lab driver without inspecting its original state and exact authorization.

## Source 0.12.0 host entry (remediation, not yet a released host-acceptance claim)

`install.sh` is a preflight/coordinator for operator-reviewed offline artifacts. See [host bootstrap boundaries](HOST_BOOTSTRAP.md) for its `plan`/`apply` arguments and Provider gating. Existing standalone app package, backup, user services and incomplete-marker behavior above remain the implementation of app update/repair. The host entry does not yet assemble credentials/MCP launcher, configure a fresh systemd user bus or expose HTTPS; a full new-host end-to-end pass is still required.

Host coordinator 0.13.4 requires `--distribution-sha256` as an additional independent review gate for `distribution.py`. Do not treat an package digest as verification of the Python installer code executing it.
