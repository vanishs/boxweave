# Lightweight MCP compatibility diagnosis and local restore-key guard (v0.42.11)

**Date:** 2026-11-03. **Owner decision:** implement and test together, without upgrading the pinned Containarium Provider or promoting the default stable installer.

[Master roadmap](FEATURE_ROADMAP_MASTER.md) · [Existing D3/D4 diagnostics](DIAGNOSTICS_D3_D4_MASTER.md) · [Upstream issues](CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md) · [Risks](../verification/OPEN_RISKS.md)

## Purpose and precise scope

- **P1:** When a BoxWeave Agent's Provider tool call fails after an upstream upgrade, give it an immediate **read-only, bounded, non-secret** clue whether the *critical tool names and key input schema* changed. No polling, schema database, auto-adaptation, retries or false compatibility certification.
- **P2 (same-host only):** Reuse upstream `restore_box_backup` and its existing `age_identity_file`. Keep the private key local to the **same MCP process host**; hide arbitrary host paths from Agent requests, enforce a fixed private directory and explicit operator approval of each exact restore. No new KMS, backup protocol, decryption service, key export or upstream interface change.

**Boundaries:** Containarium remains pinned at **v0.120.1** in BoxWeave's managed local installation. Upstream whole-Box **create** via MCP is still a separate P2; local restore can only act on an already existing valid whole-Box backup. No live destructive restore without an isolated authorized backup and valid key.

## P1: compatibility probe

1. Reuse `boxweave_diagnose` with `{"source":"compat"}`, no other arguments. This is a single *on-demand* tools/list read using the already initialized upstream MCP connection; at most 8 pages/256 tools. Do **not** call any state-mutating tool and do not sample on every request.
2. Check a **small stable BoxWeave-required subset**: `list_containers`, `get_container(username)`, `connect(box|username, exec)`, `create_container(username, ssh_keys)`, `debug_container(username)`. The plan is a local set of expectations, **not** an upstream stable contract. Check the important property **types** where present and report precise missing/wrong fields; accept `connect`'s historical `box` OR `username`. Respect operator tool allowlists: hidden tools are `not_checked`, not "missing".
3. Return source `compat`, `state=compatible_subset|schema_changed|tool_missing|not_verified`, static code(s), bounded per-tool changes, a SHA-256 digest of the normalized monitored schema, and `upstream_version=not_verified` when not explicitly observed. Never return raw upstream descriptions, arbitrary tool names, arguments, keys or error messages.
4. If discovery itself fails, return `not_verified/UPSTREAM_DISCOVERY_UNAVAILABLE`. Existing journal covers transport errors and unknown write outcomes. No schema check should claim actual response behavior is compatible. **Never retry a failed mutation automatically.**
5. No new SQLite tables. Gateway and HTTP clients use the same relay. Source self-info and operator documentation explain where to trigger the probe.

## P2: minimal same-host restore guard

The upstream `restore_box_backup` client **reads `age_identity_file` on the MCP launcher host**, then sends the decrypted identity to the existing Provider API for that request. Same-host deployment makes this usable, but **does not** mean the secret stays out of an existing local API request. This is a bounded integration of upstream behavior; no claim of HSM/KMS isolation.

### Operator provisioning

- Default **DENY** if no root-owned policy exists at `/etc/boxweave/restore-policy.json`. The file is **root-owned, group boxweave**, 0640 or stricter and no writable parent/symlink; the operator alone configures exact allowlisted tuples `{box, backup, as, key_id}` (up to 16). No policy file is generated automatically; no automatic approvals.
- The private directory is fixed as `/home/boxweave/boxweave/.restore-keys`, with root:boxweave 0750. Approved key files are root:boxweave 0640, regular, single-link, no symlink; the gateway verifies metadata, **never reads key bytes**, and the upstream MCP process alone reads the requested key at call time. Operators must confirm file access under the actual `boxweave` account; an inaccessible key fails before upstream dispatch where possible.
- The Agent-visible gateway `restore_box_backup` tool takes `key_id` (safe basename, no path) and `confirm` (the **exact backup ID**) instead of the upstream `age_identity_file` path. Exact `box`, `backup`, `as` are required; `force=true` is **not supported** by this initial guarded route. The gateway replaces `key_id` with the approved **fixed local absolute path** only after root-policy tuple match and exact confirmation; no caller-supplied file path is forwarded.
- Existing Containarium remains responsible for checking backup type/source, real target authorization and restore outcomes. If a dispatched restore times out, outcome is **UNKNOWN**; no replay. Operators can keep managing keys and backups using explicitly authorized upstream CLI/REST outside the BoxWeave Agent route.

**Threat limits:** This policy restricts **BoxWeave's delegated Agent gateway**. It does not override Containarium administrator APIs, a separately configured direct upstream MCP connection, or malicious code already executing as the `boxweave` service account. It does not solve host-level disaster recovery (host-local backup and key should have separately protected off-host copies).

### Policy example (documentation only)

```json
{
  "version": 1,
  "allowed_restores": [
    {
      "box": "demo",
      "backup": "demo-box-20261103T180000Z",
      "as": "demo-restored",
      "key_id": "demo.key"
    }
  ]
}
```

Agent call after administrator has installed that exact policy and key:

```json
{
  "box": "demo",
  "backup": "demo-box-20261103T180000Z",
  "as": "demo-restored",
  "key_id": "demo.key",
  "confirm": "demo-box-20261103T180000Z"
}
```

## Verification and release gates

- P1 fixtures: healthy subset, missing tool, changed property type, missing argument, old `box` / `username`, malformed/unavailable discovery, allowlisted exclusion, no upstream mutation and no secret payload in output.
- P2 fixtures: no policy, unsafe owner/mode/symlink/hardlink, traversal and foreign key path, wrong approved box/backup/target/key, missing confirmation, `force=true`, accepted fixed local key path, no private-key reads in gateway and no dispatch on failures.
- Full Python tests; build/manifest, docs, frontend and release CI; isolated published-version installed smoke if available. Do not run a destructive restore without an authorized isolated backup. Record bypasses outside the gateway and incomplete live recovery separately.
- New version **v0.42.11** only after successful CI and exact tagged immutable release; leave `PUBLISHED_RELEASE` stable selector unchanged until its own acceptance.

## Upstream version observation, not a daemon claim

The gateway can safely echo a validated semantic version captured from `initialize.serverInfo.version` during the *existing* MCP handshake. It makes **no** extra version query. This is the MCP server's advertised version, which may differ from the installed Containarium daemon version; it is not a runtime compatibility guarantee. The schema fingerprint is calculated only when requested, and no historical catalog is persisted.

