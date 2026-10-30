# From archive/0.5.1 to the independent application (R-08)

**No automatic migration or schema converter exists.** Old `archive/0.5.1` config/database/token/timers/services are not v2-compatible source artifacts. Do not run the old installer from the new main branch, feed old SQLite into the new schema, overwrite old keys, or claim preservation of historical operations by source upgrade. This is an operator checklist, **not** a migration test report.

## Choose an operation

- **New v2 app against an already authorized Provider**: install the reviewed release separately. No historical data is imported.
- **Parallel evaluation beside 0.5.1**: maintain distinct OS account, home/prefix, database, port, MCP launcher/token, known_hosts and service IDs; inspect both deployments before any start. Never share a live DB file or upgrade an old one using new code.
- **Legacy-only maintenance**: follow immutable archive documents in the original authorized environment, not current v2 installation scripts.

## Before a cutover

1. Record exact legacy and v2 commit and *installed* versions, active system and user services, old `boxweave-token`/`boxweave-hostkeys` timers, existing Incus/Provider release and native Guard/capability owner. Avoid changing upstream.
2. Quiesce each application's writers under its own operating procedure, and take independently verified consistent SQLite/config/service backups. Do not copy JWT, SSH private keys, signing material or credentials into project docs or new packages. Keep credential inventory only with the authorized owner.
3. Inventory old TLS/reverse proxy/ports, project/network/tenant placement, Agent-side Codex/ZCode native/WSL config and server pins. Old macOS/Cursor/Claude entries have no v2 support guarantee; preserve them untouched.
4. Prepare new independent application user, user manager, trusted archive/digest and reviewed private MCP launcher. Use a **new database and admin account**. Recreate selected bindings intentionally after exact target identity checks rather than copying unknown schema records.

## Verify and recover

Before switching clients, verify app release/backup/owned units, actual MCP initialize/discovery/read and exact host/project target, local browser auth and non-destructive receipt navigation. Security/Guard packet, public TLS, Windows GUI and reboot/boot require their own approved evidence. Treat failed or unknown operations as **not successful**; inspect the original target/receipt before any new action.

If a gate fails, stop cutover, keep the legacy service and original backups intact; no automatic host/network/DNS reversal is promised. Never use an old binary on a new DB schema. Removing legacy services/timers/users/certs/data is a separate destructive action requiring inventory, approval, retention and tested rollback. See [troubleshooting](TROUBLESHOOTING.md) and [current risks](verification/OPEN_RISKS.md).
