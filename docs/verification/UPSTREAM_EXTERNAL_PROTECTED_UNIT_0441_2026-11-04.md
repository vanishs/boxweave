# Containarium protected-unit upgrade incident and candidate 0.44.1

**Status:** source candidate; host acceptance U, release not yet published at preparation. The incident host must remain untouched for operator-controlled verification.

## Incident (UTC 2026-11-04)

The operator's Containarium 0.120.1 daemon accepted a BoxWeave 0.44.0 native upgrade request targeting v0.128.0:

- Original job: `upg-1793777286401216718`
- Upstream result: `failed`
- Exact error: `download: open /usr/local/bin/containariumd.new: read-only file system`
- The service remained at 0.120.1, no watchdog log was created, and the currently accessible backend/Boxes were not reported lost.
- Root cause: the official `containarium.service` on this release runs with `ProtectSystem=strict` and excludes `/usr/local/bin` from writable paths. The daemon's internal GitHub upgrade cannot create its staging binary there. The installer originally polled only the running version, so it incorrectly waited 180 seconds after an already terminal `failed` job.

## Candidate design

1. The privileged BoxWeave installer, outside Containarium's systemd mount namespace, takes ownership of the explicitly requested host binary transaction. It **does not** change `ProtectSystem`, `ReadWritePaths`, or open arbitrary host writes to the daemon.
2. Require the canonical, active `containarium.service` running exactly `/usr/local/bin/containariumd` with `ProtectSystem=strict`. Refuse custom units/locations rather than guessing what to restart.
3. Before a mutation, verify old daemon version with an authenticated REST read and verify container listing; abort if version drifts, native-upgrade `.new/.old` residue exists, or an earlier unresolved host-swap checkpoint exists.
4. Stage the official target binaries under a root-owned private directory, validate their SHA-256, save an immutable digest-named prior daemon executable and a private transaction checkpoint.
5. Atomically replace the daemon binary as installer root, restart only the exact service, and check authenticated running version and container listing. Preserve the prior executable. If the new runtime cannot be verified, attempt to restore the previous executable only if the staged executable remains byte-identical, then verify the old running version; otherwise require operator recovery.
6. The existing original native-upgrade helper now records the upgrade ID and polls `GET /v1/upgrades/{upgrade_id}`, reporting `failed` or `noop` instead of waiting to timeout. It never resends an uncertain POST.
7. After the daemon is verified, separately refresh and probe client/MCP binaries; do not confuse the daemon release with the v0.123.0+ host BUILD_TOOLS toolchain.

## Acceptance and limitations

- **S:** source reviewed against Containarium v0.120.1/v0.128.0 unit, native upgrade and official `docs/UPGRADE-AND-ROLLBACK.md`. That upstream runbook labels its own upgrade drills as unperformed.
- **T pending:** `python -m unittest tests.test_upstream_version_manager tests.test_provider_minimum_versions tests.test_online_adoption tests.test_arch0_boundary_guards -v`, knowledge checks, full validation, shell syntax, and source-bound release CI.
- **L unverified:** no operator host write, no service restart, no endpoint permission change, no real `0.120.1 -> 0.128.0` test, no actual rollback drill. A test-Box exercise and independent operator-host smoke are required before claiming the installed path stable.
- A binary rollback is not a database/schema rollback. Tenant data and snapshots must be separately protected. The upstream BUILD_TOOLS managed toolchain is not installed by daemon-only upgrades.

## Operator hold

Do not rerun `install.sh --version 0.44.0`, manually relax the daemon unit, delete `/var/lib/boxweave-default`, retry the original POST, or alter the retained incident host until a new **published, immutable, digest-verified** BoxWeave candidate and review have been confirmed. Keep the original log, installed Provider, service unit and original Boxes in place.
