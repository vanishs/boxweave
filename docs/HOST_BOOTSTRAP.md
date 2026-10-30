# Host bootstrap: prerequisites and boundary

The current current app package is **not** an Incus or Containarium installer. The owner has requested a new integrated `install.sh` entry for **fresh Ubuntu installation, later update and repair**, but the old 0.5.1 installer is historical evidence only. No production host changes or migrations have been authorized. This page is the operator-facing boundary while that work is implemented.

## Planned installation decision

1. **Fresh Ubuntu 24.04, x86_64, sudo available**: inspect virtualization/nesting, Incus installation and storage, Provider version/capabilities, service and SSH ownership. If host prerequisites are absent, use a separately verified, pinned upstream installer under explicit operator approval. Do not silently import upstream signing keys or run arbitrary remote shell.
2. **Existing trusted Incus + Provider + independently authorized MCP launcher**: preserve upstream services and credentials; use the [standalone app release installer](INSTALL.md) with a reviewed archive digest, explicit app config, non-root user manager and private loopback listener.
3. **Existing current app**: use the same reviewed release process for update and repair; inspect failure marker, backup, owned user units, and independent MCP status. Do not attempt a blind downgrade or erase an old database.
4. **Existing 0.5.1 deployment**: do not feed old SQLite/configuration/tokens into new app schema, run the old installer against the current application or disable old timers automatically. This is **not** an automatic migration.

The upstream [Containarium](https://github.com/footprintai/Containarium) README documents a current Ubuntu host installer and the separate `containarium` CLI/`containariumd` daemon; its install procedures and requirements change. **That does not mean we have verified an exact installation/compatibility contract for this BoxWeave commit.** Require pinned upstream source/package identity, reviewed privileges, reversible scope, and separate non-production real host acceptance before execution. Provider upgrades, routes/firewall, SSH authority, token rotation, TLS certificates and boot policy remain administrator responsibilities unless an individually accepted implementation explicitly covers them.

See [remediation test queue](verification/REMEDIATION_TEST_QUEUE.md) for unverified host scenarios.

## 0.12.0 staged host install entry

`bash install.sh --help` prints the complete local-file invocation. The tool does not download scripts or auto-upgrade Provider. Prepare independent trustworthy digests for the reviewed offline app archive, reviewed local `bwmini/distribution.py` and, only for an **empty** host with separate host-level approval, a local upstream installer. The app user must already exist with a working systemd user bus; config must already point to an authorized independent MCP launcher.

Example for **already prepared trusted Incus and Provider** (not a complete fresh-host recipe):

```sh
bash install.sh plan --app-user boxweave --package /reviewed/boxweave.zip \
  --sha256 EXPECTED_ARCHIVE_SHA256 --distribution-sha256 EXPECTED_DISTRIBUTION_SHA256 \
  --config /reviewed/application.json
bash install.sh apply --app-user boxweave --package /reviewed/boxweave.zip \
  --sha256 EXPECTED_ARCHIVE_SHA256 --distribution-sha256 EXPECTED_DISTRIBUTION_SHA256 \
  --config /reviewed/application.json
```

The example SHA is deliberately non-operational; use an independently supplied lowercase 64-character digest. `apply` delegates to the existing non-root hash-gated release installer and may return `ADMIN_SETUP_REQUIRED` until interactive initialization. Repeat it after correcting only the known cause. Source code alone does not establish host compatibility.

`--provision-provider` requires an independent local root installer with an exact SHA, the literal `PROVISION_PROVIDER` acknowledgement and an empty host; **inspect that installer separately** because it can change host packages and infrastructure. Existing Incus, Provider state, foreign services and legacy installs are intentionally not upgraded. It does not synthesize an MCP launcher or safe SSH/token/known_hosts setup. That remaining integration is tracked as **not yet complete**. Next Agent must provide safe staged provisioning contracts and real Ubuntu test evidence, not call this complete one-command deployment.

**0.13.4 security patch:** The host coordinator now requires a separately reviewed lowercase SHA256 for the local `distribution.py` itself (`--distribution-sha256`), in addition to the archive SHA256. An independently reviewed source digest is a necessary operator trust boundary, not a publisher signature. The optional root Provider installer must be root-owned at the time of execution and is rehashed just before invocation; this is not a whole-host rollback or guarantee of a clean upstream release.
