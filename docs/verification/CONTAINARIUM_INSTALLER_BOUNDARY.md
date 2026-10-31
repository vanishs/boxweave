# Containarium upstream installer: BoxWeave ownership and delegation review

**Date:** 2026-10-30. **Status:** research only; automatic upstream execution
NOT implemented or approved. This is a source review (**S**), not a successful
live install or proof of all current upstream behavior.

## Recommendation

Prefer an existing, fully configured Containarium and never mutate its
Incus/storage/secret/service. On a genuinely empty host, delegate the
Containarium-owned prerequisites to the upstream installer **only when** the
installer provides a verified, non-public-by-default installation contract
and an immutable source/release identity. BoxWeave should then install its
own application and SSH/MCP trust integration. No BoxWeave release needs to
pin a **fixed** Containarium version, but it must record and verify the
**actual selected version**, required capabilities and compatible MCP assets.
A moving `main` script executed as root without verification is not an
acceptable replacement.

## Source-backed findings

1. [Upstream manual installer](https://github.com/FootprintAI/Containarium/blob/main/hacks/install.sh)
   installs APT prerequisites, Incus from Zabbly, optional ZFS/modules,
   `containariumd`, JWT secret, service, UFW rules and an admin token.
   Incus initialization is conditional on missing storage. This is a
   substantive host installer, not just a binary downloader.
2. If installed Incus is below its required version, `install_incus()`
   executes `apt-get remove` on Incus packages and `apt-get autoremove`,
   configures package priorities and reinstalls Incus. This must **never**
   be used on an existing operator-managed Incus host without a separate,
   scoped, explicit authorization and recovery plan.
3. The upstream [service template](https://github.com/FootprintAI/Containarium/blob/main/internal/cmd/service.go)
   starts `containariumd daemon --rest --jwt-secret-file ...` without
   an `--address` override. The upstream
   [daemon default](https://github.com/FootprintAI/Containarium/blob/main/internal/cmd/daemon.go)
   is `--address 0.0.0.0 --http-port 8080`, and the upstream installer
   calls `ufw allow 8080/tcp` if UFW is present. BoxWeave currently
   explicitly requires a loopback-only Provider URL and Web port.
   Checking only `http://127.0.0.1:8080` would **not** detect exposure
   on other interfaces.
4. The upstream installer downloads the daemon binary, but not the
   separately shipped `mcp-server` binary required by BoxWeave's launcher
   and bootstrap. Same-release MCP artifact retrieval, digest/signature
   verification and a real initialize/tools/list/read test remain
   BoxWeave integration responsibilities.
5. The upstream installer resolves `CONTAINARIUM_VERSION=latest`
   dynamically. At review time GitHub's latest non-prerelease was
   `v0.122.0`; BoxWeave's previous standalone assets were sourced
   from `v0.120.1`. This drift requires **capability verification**,
   not a permanent pinned Containarium version.
6. Upstream `--quickstart` creates a workload; BoxWeave must not invoke it
   during infrastructure setup. Upstream `service install` owns its own
   systemd unit and JWT secrets; a future BoxWeave flow must not rewrite
   these and must preserve existing operator-owned instances.
7. Upstream daemon startup discovers its own core PostgreSQL containers.
   This is not the same lifecycle as BoxWeave's current `standalone`
   path, which creates a local PostgreSQL database, a separate
   `boxweave-provider.service`, and a loopback `18080` listener.
   A migration requires changing the provider mode and bootstrap
   descriptor, not merely replacing the APT command.

## Conditions for safe future implementation

- **Strict empty-host branch only:** no existing Containarium, Incus,
  provider database, services or foreign state. Existing complete
  installations are read-only reused; partial installations stop
  for human review. Do not run upstream installer merely to repair.
- **Upstream options or an equivalent reviewed interface:** bind REST
  explicitly to `127.0.0.1` **before starting the service**, and disable
  UFW/public firewall changes; no unreviewed Incus removal on a preexisting
  host. Do not patch upstream's managed unit after the service has already
  been publicly started.
- **Dynamic but verified:** resolve a stable release at install time, record
  its immutable commit/tag, authenticate the exact reviewed installer bytes,
  and fetch `containariumd`/`mcp-server` from that same release with
  verified digests. Fail closed when compatibility or evidence is missing;
  never silently fall back to an older fixed release or execute mutable
  `main` with root privileges.
- **Clear ownership:** upstream owns Incus, its storage, PostgreSQL/core
  services, JWT signing secret and daemon. BoxWeave owns only its own user,
  credential-scoped MCP launcher, Web app, gateway, SSH enrollment and
  app-specific systemd units. Preserve upstream metadata across updates.
- **Acceptance evidence:** an isolated real Ubuntu 24.04 clean install,
  a second no-change rerun, and an existing-Provider install. Check effective
  REST sockets on **all interfaces**, UFW changes, actual Incus storage,
  the running daemon and token scopes, same-release MCP initialize/list/read,
  fresh box create/connect/Python stdout and exit code, update/rollback,
  and preservation of unrelated resources.

## Current decision

The official upstream `hacks/install.sh` as inspected does **not**
satisfy the required safe unattended delegation contract. Therefore
`install.sh` in BoxWeave must **not** invoke it automatically.
A future implementation can avoid a **fixed** Containarium version,
while still enforcing immutable release provenance and runtime
compatibility. The old draft PR #4 GA failure was caused by BoxWeave's
Incus package selection; fixing that separately does not imply that
upstream delegation is ready.
