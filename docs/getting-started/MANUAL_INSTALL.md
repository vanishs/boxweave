# Install BoxWeave

This is the user-facing guide for the published v1 prerelease and the current source branch. Select an exact version before making changes. As of 2026-11-24, RC13 is the latest published v1 prerelease; stable v1 is unpublished. The newer scoped Peer and Doctor/Recover implementations are merged into v1 but not released.

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

## 2. Fresh installation: published RC13 (legacy architecture)

Choose a pinned published release from [Releases](https://github.com/vanishs/boxweave/releases). The commands below use RC13, not floating branch code.

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

The guided installer derives the dedicated Project, default pool, private incusbr-UID bridge and exclusive bw-user-UID-acl. If your existing pool/network differs, specify the actual approved --pool, --network and --acl (new exclusive ACL) as supported by that installer. Review the native preparation plan and type the requested INSTALL confirmation only for that exact target. In current v1 source the guided Legacy installer checks the required root-owned ip, bridge, nft and conntrack executables before asking for approval and again before staging; missing Debian/Ubuntu packages are reported for local operator review. It never installs them automatically. An approval refusal is not a partial installation. Current v1 prints INSTALL_PHASE markers before staged mutations; on INSTALL_PHASE_UNVERIFIED preserve the original manifest/journals and inspect local install-status and Doctor rather than reinstalling. GUIDED_INSTALL_COMPLETE precedes optional OAuth setup: OAUTH_ORIGIN_MISSING or OAUTH_CONFIGURATION_DEFERRED is not installation failure; OAUTH_CONFIGURATION_UNVERIFIED (exit 2) means finish the optional setup separately. Running-service Forward policy is only asserted from observed Doctor evidence, not the CLI environment. It prepares the confined native resources, installs/activates the service, verifies configuration and offers optional OAuth setup. It does not install Incus itself, create a tunnel, or certify packet isolation or disk quota enforcement.

RC13 uses legacy Peer authorization. Some Peer work requires separate Host-local review and Root helper readiness; ordinary successful install does not prove Peer grant/revoke readiness. Inspect the original operation if approval or evidence is missing. Never treat a legacy repair as a scoped-v2 procedure.

## 3. Scoped Peer v2: current source only

This mode is not in RC13. Use an exact independently verified candidate on a fresh dedicated test host, or a future published release that explicitly contains it. The flag is --scoped-peer-fresh; its native scope is derived, not caller-selected. For a reviewed source candidate, the current wrapper supports a complete root-local read-only plan and an interactive approval. For example, set CANDIDATE to the absolute path of the independently verified binary and SHA256 to its reviewed digest:

```sh
sudo sh install.sh --plan --scoped-peer-fresh --local-binary "$CANDIDATE" --sha256 "$SHA256"
sudo sh install.sh --interactive --scoped-peer-fresh --local-binary "$CANDIDATE" --sha256 "$SHA256"
```

The plan prints the actual native scope and scoped_fresh_review_sha256 without persisting installation state. The interactive path rechecks the native review before staging. The exact reviewed explicit --apply route requires the candidate's trusted local native-scoped-fresh-plan digest and matching host/binary identities. See the [scoped installation contract and acceptance](../development/V1_PR178_N2_FRESH_SCOPED_LIFECYCLE_2026-11-23.md) and [N5 limits](../development/V1_PR178_N5_DELIVERY_2026-11-23.md).

No legacy migration/adoption, upgrade, rollback, backup commit or uninstall is supported for scoped mode. Do not delete Root journals, guard/lifecycle units, the Incus guard drop-in or state to bypass a refusal. Any surviving scoped installation evidence blocks legacy maintenance, even if the journal is absent. Unlike legacy Peer, routine scoped Peer grants/revokes are authenticated MCP operations after reviewed setup, and new Boxes default to boot.autostart=true while explicit false is preserved. The 1,024 owned-Box bound is not a physical capacity guarantee. Retained identities, addresses, directional grant history and Root policy records have separate limits.

## 4. Verify installation

```sh
sudo systemctl is-active boxweave
sudo /usr/local/bin/boxweave install-status -project "$PROJECT"
sudo runuser -u boxweave -- /usr/local/bin/boxweave doctor -project "$PROJECT"
curl -sS -o /dev/null -w 'HTTP %{http_code}\n' -X POST http://127.0.0.1:18650/mcp
```

An active service and anonymous HTTP 401 establish listener/denial, not authenticated client or workload readiness. Verify installed version and commit; a doctor configuration result does not prove packet reachability, physical storage quota or every native Root path. In current v1 source, the expanded read-only Doctor and reviewed Recover are documented in the [operator guide](../process/DOCTOR_RECOVER.md).

## Connect a remote OAuth client

Skip for local-only Bearer clients. BoxWeave's MCP listener stays on loopback. Operate your own HTTPS reverse proxy or tunnel forwarding to http://127.0.0.1:18650; BoxWeave does not manage DNS, certificates or tunnels. Use a trusted operator terminal to configure the separate OAuth approval password (at least 16 characters) and actual external HTTPS origin, without /mcp:

```sh
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

Review the displayed source, host, existing backup and target. Prior backup closure requires the displayed approval and is irreversible; the replacement backup is retained. Do not run concurrent maintenance or force incompatible schema upgrades. Use the exact installed rollback/install-commit help and [guided upgrade contract](../development/V1_GUIDED_UPGRADE_2026-11-21.md) for separately reviewed backup operations.

For deliberate legacy retirement only, obtain the matching release's self-contained uninstall.sh, verify its checksum and read the exact release's help. The source checkout wrapper requires scripts/uninstall_core.py. Guided --full reviews owned inventory and requires explicit REMOVE confirmation. This can remove owned resources and data; preserve necessary backups and original receipts first. Unresolved originals, foreign resources, unsupported schema, separately installed legacy Peer Root services or incomplete journals can block it. Use the original local review/recovery path for these rather than removing units or journals manually. Current source full retirement also checks exact native Project/Profile/ACL against the root preparation journal before approval and before effects. NATIVE_CONFIG_DRIFT or UNVERIFIED blocks it. Older installed builds lack this read-only helper; use the matching verified Release uninstaller, not the newer wrapper or a replacement binary. A prior STARTED audit directs you to the original exact archive and --resume-plan; never rerun --full. No scoped uninstall is supported. See [retirement contract](../development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md).

If installation or maintenance fails, retain the original manifest, journal and evidence. Do not blindly reinstall, erase the ledger, weaken confinement or reset unrelated native resources.

The [historical detailed RC3–RC13 guide](https://github.com/vanishs/boxweave/blob/3165f8eedfad514ee4e5c733e7807d19ddbec2ee/docs/getting-started/MANUAL_INSTALL.md) is retained as historical reference, not a current recipe. [Developer documentation](../INDEX.md).
