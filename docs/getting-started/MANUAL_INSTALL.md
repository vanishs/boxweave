# Manual installation — BoxWeave v1 RC.3 (general host guide)

This is a **human-reviewed, fresh-install** guide, not a one-shot provisioning script. No machine-specific host hashes, UID, bridge, Release digests or public Tunnel hostnames are reusable defaults. See the [glossary](../GLOSSARY.md).

**Legend:** **Required** = necessary for this stage; **Optional** = only if you want this feature; **Review/write** = changes the host/Incus and must be explicitly approved; **Extra acceptance** = testing beyond installation.

## A. Check the dedicated host — Required, read-only

- Initially tested on Ubuntu LTS Linux; published Release assets target Linux **amd64/arm64**. Incus **7.5+ with the required native extensions** is needed; Incus **7.6** was tested on a real host. Prefer an existing compatible daemon; don't silently upgrade one with other users.
- Inspect Incus state and the storage driver **before** changing anything. BoxWeave accepts any native supported **Incus storage driver** with a ready pool (`btrfs`, `zfs`, `lvm`, `dir`, etc.). It does **not** enforce or certify filesystem quotas or remaining disk/Exec spool capacity.
- Do not co-install with legacy BoxWeave `main`, reuse an unknown `/var/lib/boxweave`, or adopt unrelated Incus instances.

```sh
uname -m
cat /etc/os-release
incus version
sudo incus storage list
sudo incus network list
sudo incus list --all-projects
sudo test -S /var/lib/incus/unix.socket.user && echo "Incus multi-user socket available"
```

If Incus is missing, use the [Incus official installation guide](https://linuxcontainers.org/incus/docs/main/installing/) or [Zabbly's signed package instructions](https://github.com/zabbly/incus); verify that the **server**, not just the client, is compatible. Ubuntu's default `apt install incus` may select an older major release. On an **uninitialized, empty** daemon only, the administrator may run `sudo incus admin init` and choose a native storage pool and managed bridge. A directory used as a storage-pool source does **not** imply the Incus `dir` driver. Never recreate or reinitialize a live pool to follow an example.

## B. Confined nonroot Incus identity — Required, Review/write

Create one dedicated service user with only the non-admin `incus` group; **never** add it to `incus-admin`, `sudo`, `docker` or `lxd`. If this account already exists, inspect it instead of recreating.

```sh
getent group incus
getent passwd boxweave || sudo useradd --create-home --gid incus \
  --shell /usr/sbin/nologin boxweave
id boxweave
PROJECT="user-$(id -u boxweave)"
echo "$PROJECT"
```

Incus **multi-user per-user Projects** normally create an isolated `user-<UID>` Project and private bridge on the user's first use of the `incus` socket. Check it through the actual service identity:

```sh
sudo runuser -u boxweave -- env \
  -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  incus project list

sudo incus project show "$PROJECT"
sudo incus profile show default --project "$PROJECT"
sudo incus network list
```

If these are absent, fix the native [Incus multi-user confinement setup](https://linuxcontainers.org/incus/docs/main/explanation/projects/) **before** proceeding. Do **not** switch BoxWeave to the privileged `/var/lib/incus/unix.socket` or manually invent a permissive replacement Project. Record the **actual** private network name and **existing** pool. Changing the global `default` Profile is not necessary.

## C. Verify a pinned Release and install the service — Required, Review/write

The following `v1.0.0-rc.3` is an **example exact published prerelease**, not a floating upgrade instruction. The source's `VERSION=1.0.0` and `V1_PUBLISHED_RELEASE=UNPUBLISHED` are **not** stable publication. Review the [Release page](https://github.com/vanishs/boxweave/releases/tag/v1.0.0-rc.3) and use the exact selected tag:

```sh
RELEASE=v1.0.0-rc.3
mkdir -p "$HOME/boxweave-install"
cd "$HOME/boxweave-install"
curl -fL "https://github.com/vanishs/boxweave/releases/download/$RELEASE/install.sh" -o install.sh
curl -fL "https://github.com/vanishs/boxweave/releases/download/$RELEASE/SHA256SUMS" -o SHA256SUMS
grep '  install.sh$' SHA256SUMS | sha256sum -c -
sh ./install.sh --plan --version "$RELEASE" --project "$PROJECT"
```

Check the pinned tag, arch, **source commit**, binary digest, project, nonadmin socket and `host_machine_id_sha256` from **this host's plan**; inspect the downloaded installer. A checksum in the same release binds assets but is not an independent signature. The installer itself verifies tag, GitHub asset digest, manifest, binary SHA256 and size. Do not pipe downloads into `sudo sh`.

Only when the exact plan is approved:

```sh
HOST_SHA256="PASTE_host_machine_id_sha256_FROM_PLAN"
sudo sh ./install.sh --apply --version "$RELEASE" \
  --project "$PROJECT" --confirm-host-sha256 "$HOST_SHA256"
```

**Install ≠ first Box preparation.** RC.3 activates the single Go service using restricted actor, compatible Incus API and SQLite checks **without** preloading a Guest image, setting the Profile/ACL or modifying Incus. It will not auto-start a Tunnel or authorize OAuth. Do not blindly rerun `--apply` if it partly failed; inspect the original manifest/recovery state.

## D. Confirm the loopback service — Required, read-only

```sh
sudo systemctl is-active boxweave
sudo /usr/local/bin/boxweave install-status -project "$PROJECT"
curl -sS -o /dev/null -w 'HTTP %{http_code}\n' \
  -X POST http://127.0.0.1:18650/mcp
```

Expect `active` and **HTTP 401** for anonymous `/mcp`: reachable loopback listener plus correct denial, **not** an authenticated OAuth test or guest readiness.

## E. Prepare native Project/Profile/ACL for the first Box — Required before creation, Review/write

Use the **existing** pool and private **managed bridge** selected above. Choose a **new, exclusive, `bw-`-prefixed** ACL name. The native prepare code only changes an **empty, unmodified, confined** per-user Project/Profile; it refuses an existing ACL, used Profile, foreign network, or unreviewed devices.

```sh
POOL=default                 # example; replace with selected pool
BRIDGE=incusbr-EXAMPLE       # replace with actual private bridge name
ACL=bw-boxweave-acl          # example; must be unused and unique
sudo /usr/local/bin/boxweave native-prepare-plan \
  -project "$PROJECT" -pool "$POOL" -network "$BRIDGE" -acl "$ACL"
```

Review the reported `target`, `actions`, `warnings`, host/native digests and `review_sha256`. Then approve **that exact snapshot**:

```sh
PREPARE_SHA256="PASTE_review_sha256_FROM_native-prepare-plan"
sudo /usr/local/bin/boxweave native-prepare-apply \
  -project "$PROJECT" -pool "$POOL" -network "$BRIDGE" -acl "$ACL" \
  -native-prepare-sha256 "$PREPARE_SHA256" -approve-local
```

Expect `status: "APPLIED"` and readback. **On any error/timeout/uncertain state, never repeat apply blindly.** Read `sudo /usr/local/bin/boxweave native-prepare-status` and reconcile the durable native journal; root-local rollback has separate identity/host/safety gates. Actual **IPv4/IPv6 packet isolation must still be tested**, not inferred from ACL JSON.

## F. Independently verify admission readiness — Required before creation

```sh
sudo /usr/local/bin/boxweave native-prereq-plan -project "$PROJECT"
```

Check the **eight** `checks[].ok` booleans, `config_ready=true` and `review_sha256`. Eight checks cover restricted service identity, Incus extensions, Project nesting, exact safe disk/NIC, ready pool, managed bridge, exclusive secure ACL and **image source policy** (not image download). Confirm the digest of this **new plan**, not the prior prepare digest:

```sh
PREREQ_SHA256="PASTE_review_sha256_FROM_native-prereq-plan"
sudo /usr/local/bin/boxweave native-prereq-verify \
  -project "$PROJECT" -native-prereq-sha256 "$PREREQ_SHA256" -approve-local

sudo runuser -u boxweave -- env \
  -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave doctor -project "$PROJECT"
```

Expected `mcp_ready=true`, `native.service_ready=true`, `native.ready=true`, `ready=true`. **`default_image_local_ready=false` is valid.** RC.3 resolves Guest image **inside the first original durable `box_create`**: local `boxweave-default`, legacy alias, or Incus official `images:ubuntu/24.04`. A caller can choose an owned local alias or `images:<catalog alias>`, but not arbitrary remote URLs. Actual image download, Guest boot, disk and packet enforcement are separate acceptance work.

## G. Optional: ChatGPT / remote OAuth through operator HTTPS — Optional, Review/write

Skip entirely for local-only MCP clients. BoxWeave serves **one** loopback-only `http://127.0.0.1:18650/mcp` endpoint. Optional in-process OAuth **shares `/mcp`** with direct Bearer auth; a public endpoint requires an **operator-managed** HTTPS tunnel/proxy.

First configure the **separate OAuth consent password** (at least 16 characters). The original non-echoing TTY flow remains supported (no password in arguments, chat, or logs):

```sh
sudo runuser -u boxweave -- env \
  -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave oauth-setup -project "$PROJECT"
```

**RC.4 noninteractive per-layer provisioning:** For automated hosts, use exactly one of `oauth-setup --generate --output-file ABSOLUTE_NEW_PATH` or `oauth-setup --password-file ABSOLUTE_EXISTING_PATH`, as the **nonroot service user**. The parent directory must already exist, be owned by `boxweave`, and be mode **0700**; the secret must be a regular, single-link **0600** file owned by that user. Relative paths, world/group-accessible files, symlinks and overwriting existing output files are refused. Generation uses 32 bytes of OS cryptographic randomness, writes one 43-character URL-safe password (plus a newline), synchronizes it to disk and stores only an Argon2id hash in SQLite. The CLI prints **only the secret-file path**, never its contents. There is no new network administration endpoint or cross-layer orchestration.

```sh
# One-time preparation of the private local secret directory per controller:
sudo install -d -m 0700 -o boxweave -g "$(id -gn boxweave)" /home/boxweave/.config/boxweave

# The following can run in a noninteractive local provisioning workflow:
SECRET_FILE=/home/boxweave/.config/boxweave/oauth-approval.secret
sudo runuser -u boxweave -- env \
  -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave oauth-setup -project "$PROJECT" \
  --generate --output-file "$SECRET_FILE"

# Optional: load an existing protected secret on a fresh controller instead:
# sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-setup \
#   -project "$PROJECT" --password-file "$SECRET_FILE"

# Display the password ONLY in your own trusted terminal, not in CI/Agent logs:
sudo runuser -u boxweave -- cat "$SECRET_FILE"
```

The **host operator** can see/copy that one-time password from their own terminal for the ChatGPT OAuth approval page; an Agent should report the `$SECRET_FILE` path and the HTTPS URL, **not echo the plaintext password in its tool stdout**. `--generate` refuses to overwrite an existing file; deliberate credential rotation requires a new file path and review of existing OAuth grants. Every nested controller needs its **own independently generated secret**; no parent token or approval password propagates to a child. Setting the password alone does not publish the service or approve a ChatGPT OAuth grant.

Install `cloudflared` from [Cloudflare's verified installation instructions](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/). For **temporary testing only**, in a second terminal:

```sh
cloudflared tunnel --url http://127.0.0.1:18650
```

Copy the actual generated external **`https://random.trycloudflare.com` origin, without `/mcp`**. From the host terminal:

```sh
PUBLIC_ORIGIN="https://REPLACE-WITH-ACTUAL.trycloudflare.com"
sudo runuser -u boxweave -- env \
  -u INCUS_CONF -u INCUS_CONFIG_DIR -u XDG_CONFIG_HOME \
  HOME=/home/boxweave XDG_CONFIG_HOME=/home/boxweave/.config \
  /usr/local/bin/boxweave oauth-set-origin -project "$PROJECT" \
  -public-origin "$PUBLIC_ORIGIN"

sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-status
```

Expect `Origin` to match **exactly** and `Enabled=true`. The URL entered into an OAuth-capable ChatGPT/custom MCP connection is **`$PUBLIC_ORIGIN/mcp`**, not the bare origin or `/mcp/oauth`. Review OAuth client, redirect URI and granted `admin`/`viewer` role; keep credentials private. ChatGPT connection UI depends on account/workspace access. Verify genuine `host_status` and `box_list` reads **before** writes; OAuth browser approval alone is not end-to-end proof.

Quick Tunnel URLs change between sessions. Update `oauth-set-origin` when that happens. To finish public testing, stop the tunnel and run `sudo runuser -u boxweave -- /usr/local/bin/boxweave oauth-disable`. Long-lived public access needs separate review of HTTPS, identity, exposure, revocation and negative tests.

## H. First Box and host acceptance — Extra acceptance (not installation)

Through an **authorized** MCP client, create **one** test Box with an original `idempotency_key`. Retain the returned Operation ID; inspect `operation_get` and native UUID readback before Start/Exec/Stop/Delete. A default image may be fetched from the official catalog; ensure sufficient capacity and public HTTPS outbound access. If an operation is `UNKNOWN` or network-disconnected, **do not generate a new key and retry** before reconciling the original.

For broader HOST_GO evidence, separately test real IPv4/IPv6 isolation and anti-spoof packets, egress toggling, driver-specific disk quota/Exec spool limits, faults and recovery, exact-owned cleanup and independent Verifier/rollback. A green `doctor`, 8/8 prerequisite check or remote MCP read **does not** establish these. See [host pilot evidence](../verification/RC3_PHYSICAL_HOST_PILOT_2026-11-13.md) and [formal gates](../development/V1_PILOT_DEPLOY_AND_RELEASE_CONTRACT.md).

## Common failure boundaries

| Observation | Safe interpretation |
| --- | --- |
| No `user-UID` Project or `unix.socket.user` | Fix Incus multi-user confinement; never upgrade runtime privilege as workaround. |
| Existing state detected by installer | Inspect previous manifest, unit, state and recovery; do not implicitly upgrade/adopt legacy `main`. |
| `native-prepare-plan` refuses existing device/ACL | Review actual use; first-Box auto preparation is intentionally limited to untouched empty owned objects. |
| `native-prepare-apply` ambiguous | Read original native journal with `native-prepare-status`; do not blindly rerun. |
| `service_ready` true but `native.ready` false | Service is healthy, first-Box Project/Profile/ACL is not yet prepared. |
| `STORAGE_QUOTA_UNVERIFIED` | Warning: test Incus driver quota and host spool/disk independently; not a readiness gate. |
| Local `/mcp` returns HTTP 401 | Correct unauthenticated rejection, not proof of completed OAuth login. |
| No Guest image locally | Acceptable in RC.3; default image selection occurs during original Create. |
| Public client requests `/` instead of `/mcp` | Fix the complete MCP URL; root-path 405 is not evidence of a broken `/mcp`. |
