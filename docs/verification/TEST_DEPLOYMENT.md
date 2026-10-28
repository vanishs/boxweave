# Isolated test deployment — client access verified

Date: 2026-10-28 America/Los_Angeles. User SRC-033 requested a dedicated test container for the new BoxWeave and/or browser access. The subsequent user approval SRC-036 authorizes the exact unfiltered host-port mapping. It is now created and client-verified as recorded below.

## Deployed resource and revision

- Backend dev; logical box `boxweave-v2-test`; actual name `boxweave-v2-test-container`; observed IP `10.206.30.6`; provider creation timestamp `1793176101`.
- Ubuntu 24.04; requested resources 4 CPU / 4 GB / 25 GB; Podman disabled. Original full managed Ubuntu public key explicitly supplied at creation plus a separate test-client public key. No private key/JWT was exported.
- Create completed once; get_container and `whoami && hostname` confirmed the expected account/name, exit 0. No duplicate/recreate was used.
- Deployed source commit `aa0857c8518454a1de6ee25203ab1d4f877c5220`; code archive SHA256 `f75275c07b15b7c4dfc582b4f2593f0918ca429918cafcd3a5c4892f8cc6d786` verified before extraction into `/home/boxweave-v2-test/boxweave-v2-app`.
- Private operational state/config/artifacts under `/home/boxweave-v2-test/boxweave-v2-deploy`. Old local/remote projects and unrelated containers were not modified.

## Application and actual tests

New container dependencies, hash-locked Python runtime and locked npm frontend were installed/built inside this new LXC. The long system dependency call hit an outer MCP 180-second response timeout; this was not treated as installation failure/success. A later read of its original exit file and log established exit 0. No competing installer or lock removal was used. Python install, npm install/build and Caddy package download all have explicit exit 0 records.

Caddy was extracted from the Ubuntu package without installing its default system service. Three dedicated units run as the container's non-root user: `boxweave-v2-test-web`, `boxweave-v2-test-worker`, `boxweave-v2-test-tls`. All reported active/running, enabled, Result=success and ExecMainStatus=0. API/worker were restarted during verification and returned healthy. This proves observed application service enablement, not LXC boot.autostart or a host reboot test.

- API binds only `127.0.0.1:18640` inside the container.
- HTTPS frontend binds `10.206.30.6:18446`; configured browser origin is the proposed `https://192.168.2.233:18446`.
- Caddy local CA with explicit default SNI was needed for IP-address clients. Initial TLS handshake failed without that default; after configuration correction, certificate/SAN and TLS 1.3 verified without disabling certificate checks. CA trust installation is deliberately disabled.
- Root page returned HTTP 200, protected API without login returned 401, login and secure cookie passed. Saving a project binding, confirming fixed hello and querying a succeeded receipt passed using **fixture infrastructure**. API/worker restart preserved the same receipt and audit count, with no replay. Health script exited 0; services remained active/enabled.
- Real application MCP/Agent integration remains incomplete. This deployment does not close RISK-017 or the phase 4 gate; UI remains clearly marked fixture, with a demo-environment target.

## Access status and approved mapping

Windows could not directly reach the container SSH address. A dedicated new-key strict-trust SSH tunnel attempt was rejected. Read-only debug_container identified a host account shell path `/usr/local/bin/containarium-shell` that does not exist. No host SSH/shell/firewall repair or transport-key reuse was attempted; no matching task-owned tunnel remains.

User SRC-036 approved TCP host port 18446 -> `10.206.30.6:18446`. Fresh identity/service/conflict checks preceded creation; list_passthrough_routes subsequently showed exactly this active route. Actual Windows client HTTPS/login/resources/receipt checks passed at `https://192.168.2.233:18446` (SRC-037). The route tool has no LAN-only source filter or external bind-address argument. The host address being private does not prove the forwarded service cannot be reached through other interfaces/NAT. The user explicitly confirmed this unfiltered exposure before the route was added. No firewall/host SSH change was made.

Public CA certificate SHA256: `547faa25da140aadd79be53d7123833afb58835ed7f9bc13777f85e94353908a`. A copy is retained in the ignored local test-access directory. Test login information is also in that private ignored directory, not in Git or this report; no plaintext password was sent to the container (only its generated hash for this new test administrator). No browser/global root trust was installed automatically. Windows native Python verified the actual endpoint using this explicit CA with normal certificate/SAN verification and proxies disabled; no TLS bypass was used. Browser/global root trust was not installed. The user may import the supplied public test CA for browser use.

## Retention and limits

The requested test deployment remains running for review; stopping/deleting it or cleaning its state requires a separate instruction. No other resource or provider core instance was modified; no Tenant policy/model call/real workload mutation ran. Fixture hello is not real infrastructure execution. The user-visible test endpoint is available from the current Windows client through the approved mapping. Windows browser UI acceptance remains a user review step; successful native HTTPS tests are not falsely labeled an automated browser test. Source commit remains unchanged; this document records deployment evidence only.

Official references: [Caddy local TLS](https://caddyserver.com/docs/caddyfile/directives/tls), [local HTTPS trust](https://caddyserver.com/docs/automatic-https), [default SNI](https://caddyserver.com/docs/caddyfile/options#default-sni). These explain the test TLS setup; they do not establish a production security certification.


## Actual client check (SRC-037)

Windows native Python using the public CA returned exit 0: homepage 200; unauthenticated API 401; login passed with Secure session cookie; session mode fixture; demo resource list passed; existing succeeded fixture receipt queried. The retained machine-readable report is in the ignored local test-access directory and contains no password/session token. Login information remains in LOGIN.txt there. No new project/hello action was submitted by this client check. Route filtering remains unverified/unsupported; TLS comes from the container's Caddy frontend, not the passthrough layer. No real application MCP credential was created and the phase-4 live integration gate remains unmet.
