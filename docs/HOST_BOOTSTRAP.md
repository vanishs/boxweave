# Host bootstrap: prerequisites and boundary

The current app package is **not** an Incus or Containarium installer. The owner has requested a new integrated `install.sh` entry for **fresh Ubuntu installation, later update and repair**, but the old 0.5.1 installer is historical evidence only. No production host changes or migrations have been authorized. This page is the operator-facing boundary while that work is implemented.

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
bash install.sh plan --app-user boxweave --archive /reviewed/boxweave.zip \
  --sha256 EXPECTED_ARCHIVE_SHA256 --distribution-sha256 EXPECTED_DISTRIBUTION_SHA256 \
  --config /reviewed/application.json
bash install.sh apply --app-user boxweave --archive /reviewed/boxweave.zip \
  --sha256 EXPECTED_ARCHIVE_SHA256 --distribution-sha256 EXPECTED_DISTRIBUTION_SHA256 \
  --config /reviewed/application.json
```

The example SHA is deliberately non-operational; use an independently supplied lowercase 64-character digest. `apply` delegates to the existing non-root hash-gated release installer and may return `ADMIN_SETUP_REQUIRED` until interactive initialization. Repeat it after correcting only the known cause. Source code alone does not establish host compatibility.

`--provision-provider` requires an independent local root installer with an exact SHA, the literal `PROVISION_PROVIDER` acknowledgement and an empty host; **inspect that installer separately** because it can change host packages and infrastructure. Existing Incus, Provider state, foreign services and legacy installs are intentionally not upgraded. It does not synthesize an MCP launcher or safe SSH/token/known_hosts setup. That remaining integration is tracked as **not yet complete**. Next Agent must provide safe staged provisioning contracts and real Ubuntu test evidence, not call this complete one-command deployment.

**0.13.4 security patch:** The host coordinator now requires a separately reviewed lowercase SHA256 for the local `distribution.py` itself (`--distribution-sha256`), in addition to the archive SHA256. An independently reviewed source digest is a necessary operator trust boundary, not a publisher signature. The optional root Provider installer must be root-owned at the time of execution and is rehashed just before invocation; this is not a whole-host rollback or guarantee of a clean upstream release.

## Independent application bootstrap

Version 0.14.0 adds an explicit application-account boundary after separately reviewed Provider preparation. It is not the still-missing automatic empty-host installer or a public release. The ordinary `plan/apply` commands above retain their original prerequisites.

Use Ubuntu 24.04 x86_64, Python 3.12 with venv, systemd/logind, reviewed Incus/Containarium and an independently reviewed MCP binary. The operator must approve using that Provider's local signing-secret **file reference**; no secret or JWT is printed or copied from an existing user. Only the token-issuing Provider command accesses its signing file; its raw output goes directly to a new private application credential file.

From a reviewed source directory (the app package includes the same entry points):

```sh
sudo bash install.sh bootstrap-plan --bootstrap-sha256 BOOTSTRAP_SHA256 \
  --app-user boxweave --archive /reviewed/boxweave.zip --sha256 PACKAGE_SHA256 \
  --distribution /reviewed/distribution.py --distribution-sha256 DISTRIBUTION_SHA256 \
  --mcp-binary /reviewed/mcp-server --mcp-sha256 MCP_BINARY_SHA256 \
  --provider-cli /reviewed/containariumd --provider-cli-sha256 PROVIDER_BINARY_SHA256 \
  --secret-file /etc/containarium/jwt.secret --provider-url http://127.0.0.1:8080
```

Replace `bootstrap-plan` with `bootstrap` and add `--ack-root-changes INITIALIZE_BOXWEAVE` to apply the reviewed changes. All SHA placeholders must be independently reviewed lowercase 64-character values. `--port` chooses an unused application loopback port (default 18640). The shell snapshots and hashes the bootstrap code before executing it. The Python coordinator snapshots and verifies all supplied artifact bytes before passing app installation to the dedicated non-root user.

A new account is required; unrelated existing users are refused. The default private app prefix is `/home/USER/boxweave`. It creates a root-controlled launcher/binaries under `/usr/local/lib/boxweave/USER`, an app-readable mode0400 credential, a root-private state directory under `/var/lib/boxweave-bootstrap/USER`, and dedicated root credential oneshot/timer units. It explicitly enables that application's user manager/linger. Existing Provider binaries/services, host SSH settings, firewall, public routes and old deployments are not changed.

The token has explicit container/SSH/route/security-read/native-policy scopes, no secret/KMS/backup scope and no wildcard. The admin role is still powerful within these allowed scopes; this is a single trusted operator, not per-tool or per-workload least privilege. A persistent timer refreshes its 48-hour token twice daily by atomic file replacement. The launcher clears inherited environment and uses the token-file reference. Its explicit upstream plaintext option applies only to the validated local `127.0.0.1` Provider endpoint; external HTTP addresses are rejected.

First administrator creation remains a deliberate local action: inspect the private bootstrap application-install log, then run the [administrator command](INSTALL.md#install-and-first-administrator) as the named app account with its selected prefix. The bootstrap reports `application_setup_incomplete` until the app's administrator and actual authenticated MCP read are verified. Repeat the same reviewed bootstrap invocation after setup. Saved application settings and SQLite data are retained; a changed Provider/MCP binary digest is refused as an unapproved infrastructure upgrade.

Inspect the original state on interrupted/unknown operations before retrying. A failure before the coordinator records its managed state can require explicit review of a partially created account; it is not automatically adopted. Do not delete the state, recreate the user or overwrite foreign units to force a retry. Postpublication app failure retains the regular install marker/backups and follows the existing [repair contract](INSTALL.md#update-failure-and-repair).

Actual evidence and remaining boundaries: [0.14.0 review](verification/MAIN_014_REVIEW.md).


## Streamed fresh-host workflow

The zero-argument install entry now downloads only a matching source-bound release. GitHub HTTPS publisher identity, release asset SHA256 metadata, manifest member digests and an exact tag-to-source match are required before executing the downloaded root coordinator. This is an explicitly publisher-trusted workflow, not an independent publisher signature; keep the existing independently supplied digest workflow when that stronger operator channel is required.

Only an inspected empty Linux x86_64/systemd host is provisioned. Ubuntu 24.04+ is recommended, not required; missing dependencies can only be installed automatically when APT and a compatible repository suite are available. Exact signed Zabbly Incus7.5.1 packages and official Containarium0.120.1 asset digests are pinned. A managed install state and exact file digests govern rerun/update/repair; an unrelated preexisting Incus/Provider/account/unit is refused. No APT locks are removed and no unrelated host or core container is repaired/deleted. The selected Provider runs standalone with optional scanners disabled and a dedicated local PostgreSQL16 policy store. Database setup uses Unix peer authentication, refuses foreign PostgreSQL state, and verifies an actual SQL read before Provider startup; this does not certify packet enforcement or all optional hosting features.

The app credential and API stay local. A dedicated app account may receive explicitly enrolled SSH keys restricted to the root-controlled MCP relay; it does not gain sudo or Incus-admin membership. Use the printed root helper enrollment command with the **client's public key only**. The helper's `client-config` action emits public MCP SSH arguments, the server's full public host-key pin and this server's distinct managed workload public key. It never exports a client private key, JWT or signing secret.

The root trust timer reads public host keys through the exact local default Incus project and publishes an atomic observed target registry. The MCP relay requires explicit valid creation keys and checks each connect target's current name/creation time/IP/username against that fresh trusted registry. Unknown/stale/reused targets are rejected until reconciliation; no host-key checking is disabled or replaced with accept-new. Existing SSH daemon configuration and host identity remain unchanged.

The initial local Web administrator is generated inside the non-root app process and its password is stored in a private app-owned file. Do not include that file in reports or packages. Code handling credentials during ordinary installation is distinct from an Agent reading/exporting their contents. Actual acceptance belongs to [the streamed test](verification/STREAMED_INSTALL_TEST.md).

## Running inside LXC

Parent must allow nesting. Inspect the exact outer instance from its parent management connection; never infer its name from guest hostname. Tested mount-capability transitions required outer restart. Installer reports parent policy unverified and never changes it. [Operator guide](NESTING.md).
