# Alternative state-transition and failure-injection acceptance — v0.26.9

**Date:** 2026-11-01 America/Los_Angeles (test Box clock may show 2026-11-02 UTC). **Baseline:** main `d826c175bdc391f72659f6ae0f535af88954306c`, released/installed stable v0.26.8. **Approach:** invert the feature-by-feature checklist: trace an action across its durability boundaries, inject an error between writes, verify state before/after, and test single-use concurrency. Also observe parent Box stop/start, nesting, and independent gateway trust.

**Evidence legend:** S = inspected source, T = actually run synthetic/isolated tests, L = independently observed installed/real Box action, U = unverified, BLOCKED = unavailable authorization or prerequisite, UNKNOWN = mutating request outcome uncertain. Historical documents do not upgrade a new version's status.

## A39: OAuth code exchange and refresh rotation were not atomic (P1)

**S/T FAIL reproduced against v0.26.8 source and L FAIL in a separate installed v0.26.8 Box** without using any actual administrator secret or credential: `RemoteCredentials.refresh` first revoked the old refresh token in one committed SQLite transaction, then `issue_pair` created access/refresh tokens in *separate* committed transactions. Injecting a synthetic failure on the second new token returned an exception, left the old refresh token **revoked**, and persisted **one unpaired valid access token** not delivered to any client. The same `issue_pair` transaction split meant OAuth authorization-code exchange could consume its one-use code before safely minting both tokens. This contradicts expected no-partial-issuance recovery.

**v0.26.9 source fix:** `_issue_pair(db,...)` inserts both new credential rows into the *same* `BEGIN IMMEDIATE` transaction. The `refresh` path revokes its old token inside that same transaction; `exchange_code` deletes its one-time code inside the same transaction. Failures roll back ALL database writes and preserve the original valid refresh/code for a deliberate retry. No persistent tokens or secrets are logged in diagnostic reports, and existing token semantics, TTL, resource binding, and single-use behavior are unchanged.

**New actually executed T PASS** in `bwaudit-1101` on candidate source:
- Controlled failure on second refresh-pair issuance: old refresh remains valid, no new credential row exists; a later ordinary rotation succeeds, revokes old refresh, and authenticates the new pair.
- Failure on second authorization-code token: one-time authorization code is still present, credentials table remains empty, and later valid exchange succeeds once.
- Duplicate token digest triggers SQLite uniqueness error on second insertion: first token row is rolled back, zero committed credentials.
- Eight threads concurrently rotate one token: exactly one success, seven `invalid_grant` rejections; database ends with exactly one revoked predecessor and one valid new pair.
- Actual Flask `POST /token` fixture: injected mid-issuance failure does not consume previously valid refresh; after fault clearance, first refresh returns HTTP 200 and authenticated MCP JSON-RPC works, second old-token use is HTTP 400.
- The complete existing HTTP MCP/OAuth happy-path, redirect/PKCE, authorization throttling, protocol-deny and revocation suite remains part of the regression.

**Proof limits:** These are real SQLite transactions and real Flask route tests in an isolated Python fixture, **not** a public Internet OAuth client, TLS/Cloudflare callback, or administrator credential acceptance. This source fix requires a source-bound v0.26.9 CI/Release and one installed upgrade checkpoint before promotion to stable.

## Independent state-transition matrix

| Scenario | Test layer | Result / acceptance boundary |
| --- | --- | --- |
| OAuth old refresh -> revoke -> mint access -> mint refresh -> response | **S/T** fault injection and Flask | A39 original FAIL; candidate T PASS, no partial state |
| Concurrent refresh 8 clients, single old token | **T** real SQLite multi-thread | Exactly 1 rotated pair, 7 rejected; no duplicates |
| OAuth code -> exchange -> second mint fault -> retry | **T** SQLite and code verifier | Code preserved on failure; one-time on success |
| OAuth admin password denial throttling, PKCE and revoke | **T** existing fixture | Included in HTTP MCP regression, no live user credentials |
| New parent create + exact identity + nesting policy + restart + login readiness | **L** task-owned `bwstate-1101` | Outer create/inspect and early `TRUST_NOT_READY` after restart observed; direct guest IP later succeeded, no trust bypass |
| New parent v0.26.8 fresh install -> official v0.26.9 release upgrade -> restart | **L** installation | PENDING until verified source-bound release and running versions |
| Operator job timeout/nonzero followed by worker restart | **L** installed test Box | PENDING; do not replay an uncertain execution |
| Parent CPU/memory/disk resize + inner app/selftest integrity | **L** installed test Box | PENDING; test-owned resource only |
| D4 allowlisted positive on outer gateway | **U/BLOCKED** | Requires administrator-owned exact Box policy, chat approval cannot substitute |
| Public OAuth/TLS/real desktop registration | **U/BLOCKED** | No external client/credentials or public ingress in this lane |
| Original historical `bwaudit-1101` nesting failure Box | **D** retained | Must not be mistaken for a fresh configured nesting parent |

**Remaining:** live host-hop SSH wrapper, external HTTP OAuth/TLS, actual 30-minute autosleep, real desktop GUI enrollment, positive D4 root-owned allowlist, forced crash of an unowned service and unpublished cross-machine source transfer all remain **U/BLOCKED**. No stable installer default change or old Box deletion on source-only evidence.

## Completion checkpoint

Record exact v0.26.9 source commit, GitHub verify/publish run, immutable release SHA/assets, official new test Box version/provenance and `doctor --mcp`, relevant post-upgrade failure path and resource cleanup. Any attempt that loses a mutating response must first inspect the *original* resource, not blindly repeat the write.

## Pre-upgrade installed counterexample and resource transitions (L evidence)

A new owner-authorized `bwstate-1101-container` was created through the updated outer gateway, native network Tenant `bwstate-1101`, Ubuntu 24.04.5, nesting configured true, immutable `createdAt=1793592217`. The outer `get_container` and nesting inspect matched exact identity. A normal restart briefly caused gateway `TRUST_NOT_READY` for a read-only direct command; after readiness observation direct `connect(exec)` succeeded with the matching Unix account and Python stdout, no SSH trust bypass. Independently started the official v0.26.8 streamed installer: **exit 0**, `boxweave provenance` version 0.26.8/source `1ea072865222585df96875dd4ae5b2d4ff90ed71`, authenticated MCP doctor **10/10 PASS**.

**L FAIL deliberately reproduced on that installed v0.26.8** using its shipped Python module with a fresh, separate ephemeral SQLite database containing synthetic only credentials: failing the second refresh token generation returned an exception, database retained `previous_refresh_revoked=1` and `orphan_access_count=1`. This is real installed code, **not** the live operator credential database or an external OAuth token. It independently confirms the source failure mechanism and justifies an installed v0.26.9 regression check.

**L PASS parent resource composition:** After install, the independent outer MCP resized running test Box CPU **2→3** and disk **30→40GB**, keeping original creation identity/IP and no restart. `get_container` readback matched both changes. Installed BoxWeave v0.26.8 `doctor --mcp` remained **10/10 PASS**. This is resource control-plane/readback verification; resizing persistence after subsequent restart and nested selftest still require observation.
