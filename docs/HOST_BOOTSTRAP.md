# Host bootstrap: prerequisites and boundary

The current v2 app package is **not** an Incus or Containarium installer. The owner has requested a new integrated `install.sh` entry for **fresh Ubuntu installation, later update and repair**, but the old 0.5.1 installer is historical evidence only. No production host changes or migrations have been authorized. This page is the operator-facing boundary while that work is implemented.

## Planned installation decision

1. **Fresh Ubuntu 24.04, x86_64, sudo available**: inspect virtualization/nesting, Incus installation and storage, Provider version/capabilities, service and SSH ownership. If host prerequisites are absent, use a separately verified, pinned upstream installer under explicit operator approval. Do not silently import upstream signing keys or run arbitrary remote shell.
2. **Existing trusted Incus + Provider + independently authorized MCP launcher**: preserve upstream services and credentials; use the [standalone app release installer](INSTALL.md) with a reviewed archive digest, explicit app config, non-root user manager and private loopback listener.
3. **Existing v2 app**: use the same reviewed release process for update and repair; inspect failure marker, backup, owned user units, and independent MCP status. Do not attempt a blind downgrade or erase an old database.
4. **Existing 0.5.1 deployment**: do not feed old SQLite/configuration/tokens into new app schema, run the old installer against v2 or disable old timers automatically. This is **not** an automatic migration.

The upstream [Containarium](https://github.com/footprintai/Containarium) README documents a current Ubuntu host installer and the separate `containarium` CLI/`containariumd` daemon; its install procedures and requirements change. **That does not mean we have verified an exact installation/compatibility contract for this BoxWeave commit.** Require pinned upstream source/package identity, reviewed privileges, reversible scope, and separate non-production real host acceptance before execution. Provider upgrades, routes/firewall, SSH authority, token rotation, TLS certificates and boot policy remain administrator responsibilities unless an individually accepted implementation explicitly covers them.

See [remediation test queue](verification/REMEDIATION_TEST_QUEUE.md) for unverified host scenarios.
