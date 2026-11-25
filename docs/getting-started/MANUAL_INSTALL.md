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

## 2. Download verified installer and select mode

Choose a pinned published release from [Releases](https://github.com/vanishs/boxweave/releases). Set RELEASE to the exact published tag you have verified. RC13 is the legacy example; use RC14 only after its Release and assets exist.

```sh
RELEASE=v1.0.0-rc.13
mkdir -p "$HOME/boxweave-install"
cd "$HOME/boxweave-install"
curl -fL "https://github.com/vanishs/boxweave/releases/download/$RELEASE/install.sh" -o install.sh
curl -fL "https://github.com/vanishs/boxweave/releases/download/$RELEASE/SHA256SUMS" -o SHA256SUMS
grep '  install.sh$' SHA256SUMS | sha256sum -c -
sh install.sh --plan --version "$RELEASE" --project "$PROJECT"
```

Review the script and the exact host/project/source/binary distribution plan. Legacy --plan is service-only; the guided fresh installer separately reviews native Project/Profile/ACL preparation before writes. With a matching current v1 source wrapper and verified candidate (not the immutable RC13 script), use a complete read-only Legacy preview first:

```sh
sudo sh install.sh --plan --guided-fresh-plan --local-binary "$CANDIDATE" --sha256 "$SHA256"
```

Set CANDIDATE to the exact verified binary path and SHA256 to its reviewed digest. The same native preparation and fixed-reader checks are repeated during --interactive; the preview is never approval. The release checksums are not an independent signature. Do not pipe a remote script directly into a root shell.

On a fresh compatible host, run the guided installation from a trusted local terminal:

```sh
sudo sh install.sh --interactive --version "$RELEASE"
```

The guided installer derives the dedicated Project, default pool, private incusbr-UID bridge and exclusive bw-user-UID-acl. If your existing pool/network differs, specify the actual approved --pool, --network and --acl (new exclusive ACL) as supported by that installer. Review the native preparation plan and type the requested INSTALL confirmation only for that exact target. In current v1 source the guided Legacy installer checks the required root-owned ip, bridge, nft and conntrack executables before asking for approval and again before staging; missing Debian/Ubuntu packages are reported for local operator review. It never installs them automatically. An approval refusal is not a partial installation. Current v1 prints INSTALL_PHASE markers before staged mutations; on INSTALL_PHASE_UNVERIFIED preserve the original manifest/journals and inspect local install-status and Doctor rather than reinstalling. GUIDED_INSTALL_COMPLETE precedes optional OAuth setup: OAUTH_ORIGIN_MISSING or OAUTH_CONFIGURATION_DEFERRED is not installation failure; OAuth configuration errors (exit 2) mean the controller is installed; use the printed oauth-status and targeted continuation, not a fresh installation. H3A also verifies the installed artifact hash, scope and active service before GUIDED_INSTALL_COMPLETE; CONTROLLER_DOCTOR is a separate phase. Running-service Forward policy is only asserted from observed Doctor evidence, not the CLI environment. It prepares the confined native resources, installs/activates the service, verifies configuration and offers optional OAuth setup. It does not install Incus itself, create a tunnel, or certify packet isolation or disk quota enforcement.

RC13 uses legacy Peer authorization. Some Peer work requires separate Host-local review and Root helper readiness; ordinary successful install does not prove Peer grant/revoke readiness. Inspect the original operation if approval or evidence is missing. Never treat a legacy repair as a scoped-v2 procedure.

## 3. Scoped Peer v2: explicit fresh installation

This mode is not in RC13. Use an exact published release containing #178 and #180, or an independently verified source candidate in an authorized test environment. The flag is --scoped-peer-fresh; its native scope is derived, not caller-selected. The matching wrapper supports a complete root-local read-only plan and an interactive approval. For a published release containing scoped mode, download and verify its installer as above and use:

    sudo sh install.sh --plan --scoped-peer-fresh --version "$RELEASE"
    sudo sh install.sh --interactive --scoped-peer-fresh --version "$RELEASE"

For an independently reviewed local source candidate, set CANDIDATE to its absolute verified binary path and SHA256 to its reviewed digest:

```sh
sudo sh install.sh --plan --scoped-peer-fresh --local-binary "$CANDIDATE" --sha256 "$SHA256"
sudo sh install.sh --interactive --scoped-peer-fresh --local-binary "$CANDIDATE" --sha256 "$SHA256"
```

The plan prints the actual native scope and scoped_fresh_review_sha256 without persisting installation state. The #188 candidate also displays the reviewed IPv4 prefix and nominal capacity before approval; this is not a physical or dual-stack allocation guarantee. The interactive path rechecks the native review before staging. The exact reviewed explicit --apply route requires the candidate's trusted local native-scoped-fresh-plan digest and matching host/binary identities. See the [scoped installation contract and acceptance](../history/development/V1_PR178_N2_FRESH_SCOPED_LIFECYCLE_2026-11-23.md) and [N5 limits](../history/development/V1_PR178_N5_DELIVERY_2026-11-23.md).

No legacy migration/adoption, upgrade, rollback or backup commit is supported for scoped mode. PR #182 implements a separate full-only retirement for fresh installations that recorded the original native preimage; use it only with an installed build and verified wrapper containing #182. Do not delete Root journals, guard/lifecycle units, the Incus guard drop-in or state to bypass a refusal. Any surviving scoped installation evidence blocks legacy maintenance, even if the journal is absent. Unlike legacy Peer, routine scoped Peer grants/revokes are authenticated MCP operations after reviewed setup, and new Boxes default to boot.autostart=true while explicit false is preserved. The 1,024 owned-Box bound is not a physical capacity guarantee. Retained identities, addresses, directional grant history and Root policy records have separate limits.

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

## Existing installations: legacy maintenance only

Do not use the fresh installer on an existing controller. Obtain the exact target release's verified installer as above. The guided legacy upgrade requires a compatible verified installed ledger and a private backup review.

```sh
sudo sh install.sh --plan --upgrade --version "$RELEASE"
sudo sh install.sh --upgrade --version "$RELEASE"
```

Review the displayed source, host, existing backup and target. Prior backup closure requires the displayed approval and is irreversible; the replacement backup is retained. Do not run concurrent maintenance or force incompatible schema upgrades. Use the exact installed rollback/install-commit help and [guided upgrade contract](../history/development/V1_GUIDED_UPGRADE_2026-11-21.md) for separately reviewed backup operations.

For deliberate legacy retirement only, first settle owned Peer/Forward dependencies and outstanding originals. Obtain the matching release's self-contained uninstall.sh, verify its checksum and read the exact release's help. The source checkout wrapper requires scripts/uninstall_core.py. Guided --full reviews owned inventory and requires explicit REMOVE confirmation. This can remove owned resources and data; preserve necessary backups and original receipts first. Unresolved originals, foreign resources, unsupported schema, separately installed legacy Peer Root services or incomplete journals can block it. Use the original local review/recovery path for these rather than removing units or journals manually. Current source full retirement also checks exact native Project/Profile/ACL against the root preparation journal before approval and before effects. NATIVE_CONFIG_DRIFT or UNVERIFIED blocks it. Older installed builds lack this read-only helper; use the matching verified Release uninstaller, not the newer wrapper or a replacement binary. A prior STARTED audit directs you to the original exact archive and --resume-plan; never rerun --full. Do not use this legacy flow for scoped installs. See [retirement contract](../history/development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md).

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

An explicitly supported stopped Legacy installation with exact cold
ledger/native provenance may offer the separate --offline-plan and
--offline-full routes. They archive and preserve the original UNKNOWN
and claims without settling or replaying it. The executable leaf is
limited to the verified supported originals; ambiguous historical Peer
Root journals, unproved native state and partial Scoped installs remain
blocked. Use --offline-resume-plan and --offline-resume only for supported
post-native phases of the exact interrupted original. Do not substitute
a newer candidate binary for the original installed artifact or bypass
refusal by deleting evidence. After verified complete retirement, review
purge separately. See [H5 delivery](../development/PR188_H5_DELIVERY.md).

### Nested lifecycle compatibility (candidate)

A working nested Incus and healthy Scoped controller do not by themselves
prove a complete uninstall/reinstall cycle. The installer now reports an
advisory maintenance_preflight before a reviewed fresh install. Type-wide
systemd service drop-ins may only become effective after a unit is
installed; absence of a detected file is not proof of compatibility.
In particular, Ubuntu LXC's inherited service.d policy has blocked both
live and empty Scoped retirement, even after normal MCP cleanup. Preserve
that policy and review the actual effective units with Doctor maintenance.
Do not remove or mask inherited drop-ins or relax confinement to force
retirement. The separate clean-VM roundtrip CI proves same-mode purge and
fresh workload, not safe recursive confinement or nested retirement.
See [roundtrip evidence](../development/PR188_NESTED_ROUNDTRIP.md).
