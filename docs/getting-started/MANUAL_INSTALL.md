# Install BoxWeave

Select an exact published release and verify its assets. See [current source and release status](../STATUS.md); source features are not automatically present in a release. RC13 uses Legacy; the newer scoped, Doctor/Recover and purge features require a build containing their merged PRs.

## 1. Prepare the Host

Use a dedicated or explicitly approved Linux amd64/arm64 host (Ubuntu LTS is the initial tested baseline), systemd, Python 3 and a compatible Incus server with required native extensions (7.5+, tested 7.6). Incus must already be installed and initialized with an existing storage pool and managed bridge. See the [Incus installation guide](https://linuxcontainers.org/incus/docs/main/installing/). Do not reinitialize a live daemon, pool or bridge.

Inspect the host locally:

```sh
uname -m
incus version
sudo incus storage list
sudo incus network list
sudo incus list --all-projects
sudo test -S /var/lib/incus/unix.socket.user
```

The BoxWeave service identity must be nonroot and confined to its own Incus per-user Project. If absent, create the dedicated account on an approved host:

```sh
getent group incus
getent passwd boxweave || sudo useradd --create-home --gid incus --shell /usr/sbin/nologin boxweave
id boxweave
PROJECT="user-$(id -u boxweave)"
sudo runuser -u boxweave -- incus project list
sudo incus project show "$PROJECT"
sudo incus profile show default --project "$PROJECT"
```

Do not add this user to incus-admin or sudo, or use the privileged Incus socket. If the confined Project or private bridge is missing, resolve native multi-user setup before installation. Never adopt unrelated resources or reuse an unknown BoxWeave ledger.

## 2. Download verified installer

The consolidated current source supports one fresh Scoped installation
contract, not Legacy installation, adoption, upgrade or rollback. Use an
exact matching verified published Release when available, or an independently
verified local candidate in an authorized test environment. Historical
releases require their own matching wrapper and documentation.

Download the exact tag's install.sh and SHA256SUMS from
[Releases](https://github.com/vanishs/boxweave/releases); verify the installer
digest. Release checksums are not an independent signature. Do not pipe a
remote script directly into a root shell.

## 3. Reviewed fresh installation

On the approved fresh host, use a trusted local terminal. Set RELEASE to the
verified matching tag, or CANDIDATE to the absolute verified local binary and
SHA256 to its reviewed digest.

```sh
sudo sh install.sh --plan --version "$RELEASE"
sudo sh install.sh --interactive --version "$RELEASE"
```

For a local candidate:

```sh
sudo sh install.sh --plan --local-binary "$CANDIDATE" --sha256 "$SHA256"
sudo sh install.sh --interactive --local-binary "$CANDIDATE" --sha256 "$SHA256"
```

The native scope is derived from the dedicated restricted identity and
compatible existing Incus; arbitrary pool/network/ACL selection is refused.
The retained scoped flag is an explicit compatibility spelling, not a
second mode. Review the actual candidate, host, Project, native scope and
capacity before approval. The root-local plan is read-only, never approval.
Explicit apply also requires exact reviewed host and Scoped native digests
and Project. Missing prerequisites, stale/declined or existing/mixed
installation evidence must refuse.

Missing root readers (ip, bridge, nft and conntrack) require local operator
review; the wrapper does not install those packages.

Both install paths verify the installed Scoped manifest, candidate digest,
identity, active service and restricted Doctor readiness. Interactive
installation additionally offers optional OAuth continuation and preserves
an existing password. OAuth deferral or a configuration error (exit 2)
is separate from the already completed installation. Continue with the
printed OAuth commands, not a reinstall. On INSTALL_PHASE_UNVERIFIED preserve the original
files and journals; inspect install-status and Doctor, do not reinstall
blindly. Configuration readiness is not packet, quota, remote authorization
or release acceptance.

Normal full retirement uses the matching verified uninstaller and original
native preimage. Offline Scoped retirement remains a separate acceptance
obligation during consolidation; never substitute Legacy execution.

## 4. Verify installation

```sh
sudo systemctl is-active boxweave
sudo /usr/local/bin/boxweave install-status -project "$PROJECT"
sudo runuser -u boxweave -- /usr/local/bin/boxweave doctor -project "$PROJECT"
curl -sS -o /dev/null -w 'HTTP %{http_code}\n' -X POST http://127.0.0.1:18650/mcp
```

An active service and anonymous HTTP 401 establish listener/denial, not authenticated client or workload readiness. Verify installed version and commit; a doctor configuration result does not prove packet reachability, physical storage quota or every native Root path. In current v1 source, the expanded read-only Doctor and reviewed Recover are documented in the [operator guide](../process/DOCTOR_RECOVER.md).

## Connect a remote OAuth client

Skip for local-only Bearer clients. BoxWeave's MCP listener stays on loopback. Operate your own HTTPS reverse proxy or tunnel forwarding to http://127.0.0.1:18650; BoxWeave does not manage DNS, certificates or tunnels. Use a trusted operator terminal to inspect oauth-status first. Current candidates report PasswordConfigured independently of Enabled; do not reset a configured password just to fix an origin. The guided installer accepts one trailing slash on an otherwise valid HTTPS DNS origin, rejects userinfo, ports, paths, queries and fragments, and permits bounded correction or deferral without reinstalling. Direct oauth-set-origin still requires the canonical origin without a slash or /mcp. If no password is configured, use oauth-setup once (at least 16 characters):

```sh
sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-status -project "$PROJECT"
# Only if a password is not already configured:
sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-setup -project "$PROJECT"
PUBLIC_ORIGIN="https://your-approved-host.example"
sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-set-origin -project "$PROJECT" -public-origin "$PUBLIC_ORIGIN"
sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-status
```

Connect the OAuth-capable MCP client to the actual origin followed by /mcp. Confirm consent only for the intended client; credentials grant full management of owned resources. Verify authenticated host_status and box_list. For a changed tunnel origin update oauth-set-origin; stop the public proxy/tunnel and use oauth-disable when ending access. Do not expose passwords or tokens to chat, logs or source control. Use the [usage guide](USE_BOXWEAVE.md) to create the first Box.

## Existing installations

Do not use the fresh installer on any existing or incompletely retired
controller. Preserve original manifests, receipts and journals. Historical
Legacy installations require the exact matching historical artifact and
[upgrade contract](../history/development/V1_GUIDED_UPGRADE_2026-11-21.md)
or [retirement contract](../history/development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md);
neither is a current Scoped upgrade or repair path.

If installation or maintenance fails, retain the original manifest, journal and evidence. Do not blindly reinstall, erase the ledger, weaken confinement or reset unrelated native resources.

The [historical detailed RC3–RC13 guide](https://github.com/vanishs/boxweave/blob/3165f8eedfad514ee4e5c733e7807d19ddbec2ee/docs/getting-started/MANUAL_INSTALL.md) is retained as historical reference, not a current recipe. [Developer documentation](../INDEX.md).

### Scoped retirement (requires #182)

Use the matching verified release/candidate wrapper and installed binary. Stop/delete all owned Boxes and revoke Peer/Forward claims through normal MCP operations first; settle outstanding originals using the supported recovery workflow. The original #182 retirement plan refuses live or unsettled work. The #188 candidate can instead review a complete exact owned drain (normal Peer/Forward, stop and Delete originals) before retirement; it refuses UNKNOWN, incomplete or foreign native inventory and missing original Project/Profile preimage. Review all destructive effects and preserve required backups first.

```sh
sudo sh uninstall.sh --scoped --plan
sudo sh uninstall.sh --scoped --full
```

Review the exact plan and enter the displayed confirmation only if you intend to retire this controller. The controller and its scoped Root lifecycle service stop, original evidence is archived privately, the original native Project/Profile is restored, the owned quarantine ACL and exact guard are removed, and the owned controller files are retired. Incus, its bridge and unrelated native resources are retained. Never delete the original journal or bypass the guard manually.

If interrupted, inspect the exact original audit and use `--scoped --resume-plan` then `--scoped --resume` instead of starting a new retirement. An unverifiable phase refuses and preserves the original evidence. No old scoped preimage reconstruction or migration is provided. Installed acceptance passed in [PR #182](https://github.com/vanishs/boxweave/pull/182).

### Completely erase retired controller data before a fresh install

A successful full retirement removes active controller state and owned Boxes, but deliberately leaves private retirement archives (including SQLite, historical command output and credential material). On a trusted local terminal, using the matching verified newer uninstaller containing this feature, separately review irreversible erasure:

```sh
sudo sh uninstall.sh --purge-plan
sudo sh uninstall.sh --purge
```

The erasure plan lists only recognized, reviewed completed retirement archives, closed native preparation journals and the completed audit. It refuses active or unverified native/controller artifacts, nonempty owned Project inventory, unknown archive entries, symlinks and incomplete retirements. If interrupted, repeat the read-only purge plan and approve the same original purge; installation is blocked while the durable purge receipt survives. Do not treat a purge as a way to release outstanding operations or to bypass an installation safety check. It is ordinary file removal, not a guarantee of forensic erasure of disks, snapshots or independent backups.

For RC13, obtain the matching verified RC13 uninstaller for the original legacy retirement. Do not use the RC14 purge wrapper for that initial retirement. Clean up owned Boxes, Peer and Forward via MCP and let native retirement operations settle. A separately staged legacy Peer Root unit or its journals, if present, needs exact local original review; the current generic purge intentionally refuses these residuals rather than deleting an unverified root capability. Retain evidence if the original retirement refuses. Do not substitute the new reader on an old installed binary. Run the new purge only after verified full retirement. No automatic migration, arbitrary native cleanup, removal of Incus itself, dedicated OS account, images or external tunnel is performed. The next Scoped install requires its own fresh native preview and explicit approval; the existing bridge address range is preserved and foreign native projects or customized topology may block admission. The original RC13 Host may still use a /24 bridge rather than the /21 provisioned by a new Incus bootstrap.

### #188 candidate: maintenance preview and exceptional offline retirement

The unified observation-side preview is separate from executable approval.
For the matching verified candidate wrapper and local installed scope:

```sh
sudo boxweave doctor --maintenance --summary
sudo sh uninstall.sh --maintenance-plan --project "$PROJECT" --retention retain
```

Select erase only to review that retention intent; the preview never
authorizes erasure. Use the exact maintenance-recheck digest interface
shown by the matching wrapper to check freshness. It is not a full-retirement
approval. Normal Scoped retirement uses its own full plan and confirmation.

The historical stopped Legacy offline route is not the consolidated
Scoped contract. The current offline Scoped lifecycle remains an open
acceptance obligation; do not substitute a historical procedure, newer
binary or evidence deletion for a refused original. See
[H5 delivery](../development/PR188_H5_DELIVERY.md) for historical context.

### Nested lifecycle compatibility (candidate)

For the exact reviewed Ubuntu 24.04/systemd 255 unprivileged LXC generated
service.d policy, Scoped full retirement supports empty and running owned
Box sets. Doctor reports LXC_SCOPED_RETIREMENT_REVIEW_REQUIRED; the local
uninstaller verifies the actual unit body, manager state, policy identity,
exact scope and original claims, and rechecks them during approved
retirement. Unknown or additional overrides remain blocked. No inherited
policy is removed or masked. The profile is a retirement compatibility
contract, NOT a runtime-hardening attestation or Legacy activation
exception. The existing LXC policy overrides NoNewPrivileges; that
remains a separate security observation.

Before installation, KNOWN_LXC255_POLICY_BYTES_REVIEW_REQUIRED means
only that the single recognized global policy's bytes were observed.
It is not proof of an effective unit, mapped container identity, runtime
hardening, or retirement readiness. Unknown/additional global overrides
receive the conservative warning. Local candidates must use the absolute
path to the actual regular binary and its exact digest (not a symlink).

Use the normal reviewed Scoped full plan and terminal approval; purge
remains separate. See [nested roundtrip evidence](../development/PR188_NESTED_ROUNDTRIP.md)
and [live retirement follow-up](../development/PR188_LXC_LIVE_RETIREMENT.md).
