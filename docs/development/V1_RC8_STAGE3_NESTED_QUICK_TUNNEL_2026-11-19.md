# v1 RC8 candidate — nested L0 + Cloudflare Quick Tunnel, real ChatGPT handoff

**Date:** 2026-11-19 UTC. **Source SHA:** `bc1e844703cdfb7ea3813cc614176d35e8883018` (Draft PR #153, unmerged). **Status:** `RC8_CANDIDATE_TUNNEL_MCP_READY / REAL_CHATGPT_OAUTH_PENDING / RELEASE_NOT_PUBLISHED`.

This is a user-approved **disposable development candidate**, not an immutable `v1.0.0-rc.8` tag or published release. The existing L0 controller, host Incus ACL/Scriptlet and other L0–L5 resources were **not** upgraded or modified.

## Installed scope and immutable source evidence

- L0-created, **nested-only** disposable Box `bw-rc8-live-1119` under the already owned L0 Project; no parent host daemon/ACL change.
- Fresh Ubuntu 24.04 inner environment; signed Zabbly **Incus client/server 7.6** (exact installed package `1:7.6-ubuntu24.04-202611072126`); signed Cloudflare `cloudflared 2026.11.1`.
- Verified upstream **Go 1.26.9** tarball with the official go.dev SHA256. `CGO_ENABLED=1`, GCC, and complete `go mod verify` before building.
- Candidate native binary `/usr/local/bin/boxweave-rc8` built from pinned source `bc1e844703cdfb7ea3813cc614176d35e8883018`, with `-X main.version=1.0.0-rc.8-candidate -X main.commit=<full SHA>`, **SHA256 `06759b41fbd36727d37684276f82b7b9cfb9447df2f934b61bc79cb2b81c5386`**. `build-info` returned that exact version and SHA.
- Independent, **non-root** `boxweave` identity, only restricted inner Incus `incus` group, fresh per-user inner Project `user-1001`, a separate SQLite **Schema v4** instance. Root-free Controller systemd unit `boxweave-rc8-candidate.service` listens **only** on `127.0.0.1:18650`.
- Separate one-time-generated OAuth **approval password**, mode `0600`, owned by `boxweave`, only inside the guest at `/home/boxweave/.config/boxweave/oauth-approval.secret`. **The password is deliberately not included in GitHub docs, shell logs or this chat.** It must be read only on the operator's trusted terminal. Do not copy OAuth Grants or tokens from v3.
- A separate, non-root `boxweave-rc8-quick-tunnel.service` invokes `cloudflared tunnel --protocol http2 --url http://127.0.0.1:18650 --no-autoupdate`. Cloudflare registered the outbound HTTP/2 tunnel. **The temporary public hostname is deliberately not committed here.** It was transmitted to the Owner in the conversation for ChatGPT registration; each Quick Tunnel restart can change it and requires host-local `oauth-set-origin` reconfiguration.

## Real observed tests, without inferring full ChatGPT success

- `native-prepare-plan` on the **fresh inner** Incus `user-1001` Project / `default` dir pool / `incusbr-1001` managed bridge / unique `bw-rc8-private` ACL was reviewed at exact digest `c897b14d2a105674ae0634df2957f0a154d994c7d64dec8e62f3074f3a579a5b`; `native-prepare-apply` returned **APPLIED** and independent readback succeeded. **Only this disposable inner Incus** was affected; parent L0 host settings unchanged.
- Non-root `boxweave-rc8 doctor` returned `ready=true`, `native.ready=true`, `mcp_ready=true`, Schema v4, Incus 7.6, private NIC/ACL ready. `default_image_local_ready=false` is allowed (on-demand image). **Storage quota and native pool isolation remain UNVERIFIED**, as reported by the software itself.
- Local loopback anonymous `POST /mcp`: **401**; public HTTPS OAuth authorization-server discovery: issuer/registration/authorization/PKCE S256 **PASS**; protected-resource discovery: exact public `.../mcp` resource **PASS**; public anonymous MCP: **401**.
- Public HTTPS **authenticated** real JSON-RPC MCP `initialize` at protocol `2025-11-25`: HTTP **200**; `tools/list`: HTTP **200**, **17 tools**; authenticated `tools/call host_status`: HTTP **200**, version `1.0.0-rc.8-candidate`, `ready=true`, schema 4. The short-lived *local probe* credential was not logged in stdout and had a revocation command scheduled on script exit. These tests do **not** prove ChatGPT-specific OAuth interoperability.
- GitHub Stage-2 S4-A/B2/B3 and S3 source-equivalent native regression is recorded separately in [Stage 2 acceptance](V1_STAGE2_NATIVE_REACCEPTANCE_2026-11-18.md). **No new formal RC8 release/CI signoff is implied.**

## Next user action / remaining gates

1. In ChatGPT custom MCP configuration, enter the **actual temporary HTTPS origin plus `/mcp`** supplied in the live handoff. Choose OAuth; let ChatGPT dynamically register and redirect to the approval page. Obtain the approval password **locally** from the protected guest file, not via public URL/chat/CI logs.
2. Observe **real ChatGPT** OAuth discovery, DCR, consent, authorization code, tool invocation, refresh, reauth and revocation. This is still `NOT_TESTED` until actual evidence. Diagnose and fix any failing step against the **same current exact source**; do not blindly replay native UNKNOWN operations.
3. Exercise actual **nested Guest creation / Exec / Stop / Delete** with original durable IDs, and separately verify AppArmor/UID mapping and real packet safety. `doctor.ready=true` is a preflight, **not** proof that an inner Guest was booted. Storage quotas remain unverified.
4. When all gates pass, freeze a new reviewed source SHA, build verified Linux artifacts, independent acceptance and **only then** publish immutable `v1.0.0-rc.8`. After tunnel restart reverify its current hostname and update `oauth-set-origin`; a Quick Tunnel has no uptime or stable-host guarantee.
5. To decommission only after Owner approval: stop the **inner** cloudflared/Controller units, disable/revoke OAuth and temporary credentials, preserve unresolved original/UNKNOWN and audit, then safely remove the disposable nested Box. **Never** delete shared outer Incus state by inference.

### Resolved installation friction

Initial `go mod` toolchain proxy fetched only part of Go 1.26.9 within the L0 Exec deadline; solved by official go.dev tarball plus published SHA256 check. Initial build correctly refused SQLite with `CGO_ENABLED=0`; added GCC/C headers and rebuilt with CGO enabled. An Ubuntu package mirror timed out; switched to an Ubuntu-signed accessible mirror and minimized the toolchain packages. **These were disposable environment/bootstrap issues, not production source changes.**


---

## Stage 3 independent real-ChatGPT MCP quick acceptance — 2026-11-19 PT

**Verdict: PARTIAL / RC8 NOT RELEASE-READY.** This section supersedes only the older \`REAL_CHATGPT_OAUTH_PENDING\` status above: real ChatGPT-authenticated MCP tools have now been called successfully. The full OAuth refresh/reauth/revocation campaign remains **NOT_TESTED**. Independent verification used the user-connected **\`boxweave L1 noviewer\`** tools; no host/root/CI/OAuth settings, source, service or permissions were changed.

**Source under test (from live \`host_status\` and \`operation_get\`):** \`bc1e844703cdfb7ea3813cc614176d35e8883018\` (\`1.0.0-rc.8-candidate\`); binary SHA256 \`06759b41fbd36727d37684276f82b7b9cfb9447df2f934b61bc79cb2b81c5386\`. **PR #153 HEAD before this report update:** \`8de497fa3923698fbe70cccae5612c857b201002\`; GitHub compare showed exactly one docs-only commit after candidate source, no Go code change. PR remains Draft and unmerged. \`work_ref=rc8-stage3-20261119-acceptance\`; one test Box only, \`bw-rc8-s3-1119-a\`, UUID \`3ca24ce9-bfef-4828-9244-cd42032d071e\`, project \`user-1001\`.

### Actual test matrix

| Gate / tool | Result | Actual evidence / limit |
|---|---|---|
| Real ChatGPT MCP OAuth + \`host_status\` | **PASS** | Authenticated tool worked; \`ready=true\`, \`mcp_ready=true\`, Native \`ready=true\`, SQLite schema 4, Incus 7.6, pinned candidate binary/version |
| \`box_list\` / \`operation_list\` baseline | **PASS** | Initially zero owned Boxes and zero Operations |
| \`box_create\` | **PASS** | Key \`rc8-stage3-20261119-create-a1\`; original BW Operation \`0ae6f1d601e9535bb7ef58914e4b148c\` moved ADMITTED → VERIFYING → **SUCCEEDED**; native Operation \`96d55642-c7f9-4abf-b2c6-81c4352b350e\`; exact Box UUID above |
| \`box_get\` / post-Create \`box_list\` | **PASS** | Native Box present, \`Stopped\`, \`guest_ready=false\`, CPU 1, memory 512 MiB, disk 4 GiB, internet=false, nesting=false |
| Idempotency and Operation continuity | **PASS** | Replayed exact successful Create request with **same** key/arguments and work_ref: \`duplicate=true\`, same BW/native Operation, UUID and SUCCEEDED state; \`operation_get\` + filtered \`operation_list\` recover original receipts |
| \`box_start\` | **BLOCKED** | Key \`rc8-stage3-20261119-start-a1\`; BW Operation \`3ca081f1e0dcb018d7609b1467a102bf\`; native Operation \`aa0ad291-75cc-40f9-8366-9dc8a05d09dd\`; ADMITTED → **UNKNOWN**, \`error_class=POSTCONDITION_UNVERIFIED\`, \`next_action=HOST_LOCAL_REVIEW\`. Re-read remained UNKNOWN |
| Guest \`box_exec\` and \`operation_output_get\` | **BLOCKED** | A valid owned Guest did not reach verified Running/Exec-ready state. No command was run in it. Calling output-get on non-Exec Start ID returned \`operation not found\`; this is **not** evidence about Exec-output correctness |
| \`box_file_apply\`, \`box_file_list\`, \`box_file_read\` | **BLOCKED** | No ready Guest for actual file write/readback. Read-only root directory list returned \`invalid admission\`; read of \`/etc/hosts\` returned \`required safe native capability or approved image unavailable\`. Neither establishes a specific source defect |
| \`box_stop\` | **BLOCKED** | Do not mutate the Box while original Start is UNKNOWN |
| \`box_delete\` / cleanup | **BLOCKED** | Preserved only test-owned Box for owner-side original UNKNOWN investigation; no unsafe cleanup |
| Safe negatives: unknown UUID and nonexistent Operation | **PASS (rejection)** | \`box_get\`, \`box_file_read\`, \`operation_get\`, plus \`box_start\`, \`box_exec\`, \`box_file_apply\` against non-existent UUID \`00000000-0000-4000-8000-000000000001\` did not return a successful authorized action or data. Calls surfaced \`INVALID_ARGUMENT\` with underlying \`operation not found\`; error wording is generic and root cause is unconfirmed. Negative write keys: \`rc8-stage3-20261119-neg-start-bogus1\`, \`rc8-stage3-20261119-neg-exec-bogus1\`, \`rc8-stage3-20261119-neg-file-bogus1\` |
| OAuth refresh, reauth, revocation; File success path; guest CLI output; network packet safety; stop/delete terminal path | **NOT_TESTED or BLOCKED** | No proof from an authenticated initial MCP call that refresh/revocation work; no guest-ready path; native and physical release tests require separate evidence |

**Exact final independent readback:** \`box_list\` still contains only \`bw-rc8-s3-1119-a\` as \`Stopped\` with \`guest_ready=false\`; filtered \`operation_list\` shows exactly two recorded original Operations for this work_ref: Create SUCCEEDED and Start UNKNOWN. Empty lists for rejected negative requests must not be used to infer absence of any possible native effect. No new-key retry of the UNKNOWN Start occurred.

### Root-cause classification and release decision

- **Confirmed BoxWeave functionality:** actual authenticated MCP connectivity and reported health, owned create/get/list, durable original-operation readback, same-key idempotency. **No confirmed code defect** in this campaign.
- **Known nested environment caveat:** the tested controller runs inside an L0 Box; parent AppArmor enforcement, UID/GID delegation, inner Incus networking/storage are potential restrictions. Storage quota remains **UNVERIFIED** (\`selected_storage_quota_verified=false\`); default image may be downloaded on demand. **None of these caveats is proven as the specific Start UNKNOWN cause.**
- **Unknown root cause:** Start \`POSTCONDITION_UNVERIFIED\` with native Operation ID present, original journal UNKNOWN and post-readback native Stopped. The inconsistent/generic negative-error wording is also **unattributed**. Do not label either as a definite source failure or a definite AppArmor fault.
- **Smallest next investigation for Owner, without modifying this campaign:** inspect **read-only** original inner Incus native Operation \`aa0ad291-75cc-40f9-8366-9dc8a05d09dd\`, guest/Incus daemon startup logs, AppArmor denies, UID/GID map and instance poststate; correlate with BW \`3ca081f1e0dcb018d7609b1467a102bf\` before deciding whether the fix belongs to the nested test fixture or the software. Keep current state and key untouched until authorized review. For a release gate, run a fresh clean permitted native environment test rather than loosening nested security protections merely to pass.
- **Status:** **STAGE3_PARTIAL**. Seven high-level gates have positive evidence (OAuth/tool, baseline list, Create, Get, idempotency, continuity, safe negative rejection); Start and downstream lifecycle/file gates are blocked, and full OAuth/release-specific gates remain untested. No justified 100%-completion claim. **Do not publish RC8, merge PR #153, or sign final release.**
- **No repair and no publication:** no source/config/permission/host/CI/OAuth changes, no privilege escalation, no RC tag/release, and no cleanup that would erase UNKNOWN. This documentation-only evidence append is the sole intended repository write.
