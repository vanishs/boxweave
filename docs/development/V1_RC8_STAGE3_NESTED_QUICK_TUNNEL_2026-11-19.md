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
