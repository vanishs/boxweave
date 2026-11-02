# HTTP MCP quick-start: default loopback service and explicit public HTTPS

Status: implementation approved by owner 2026-11-01; target source v0.27.0. Parent: [remote MCP master](../design/REMOTE_MCP_MASTER.md); acceptance [REMOTE_MCP_TEST_PLAN](../verification/REMOTE_MCP_TEST_PLAN.md). Scope follows REQ/AC-021..024 and retains single trusted-operator security.

## Decisions and requirements

- QH-01: On every activated managed application install/update, provide an enabled, restartable user-systemd `remote-mcp` unit bound **only** to `127.0.0.1:18844`. If the operator explicitly sets `remote_mcp.enabled=false`, do not start it. A port conflict or failed service fails activation rather than being ignored. Existing SSH/stdio, Web and worker remain separate and are never exposed through this port.
- QH-02: `public_url` is optional only for loopback mode; with no public origin, canonical MCP resource and OAuth issuer are `http://127.0.0.1:18844`. Keep OAuth, API bearer and credential-path verification; never expose unauthenticated tools. Accept **only** HTTPS origins for a configured public URL; retain strict POST origin and host checks and browser referrer/CSP scoping. The loopback origin is not a usable hosted ChatGPT URL.
- QH-03: An explicit `public_url` switches discovery/issuer to that HTTPS origin while HTTP transport still binds loopback. An operator-configured public URL does not create an external route or launch Cloudflare. `allowed_origins` remains an explicit list, never allow `null` or wildcard. Changing public URL invalidates old resource-bound OAuth grants until the client reconnects.
- QH-04: Default port and enablement are normalized in the **managed runtime**, preserving existing `config.json`, user-supplied `remote_mcp` and unrelated keys. The installed service has an owned systemd unit, a verified shutdown/restart lifecycle, readiness metadata and a rollback/repair boundary. Do not run a second process on an occupied port.
- QH-05: `boxweave http status|doctor|configure --public-url HTTPS|restart` operates via the existing root-owned host CLI, delegating private reads/writes to the fixed non-root application identity, never importing app code as root. An explicit `configure` validates the HTTPS origin, updates both private configuration files with a safe atomic backup/rollback, and restarts the owned unit. Never print credentials or raw application config.
- QH-06: The installer prints loopback MCP URL, how to check status, and these quick-start steps. For temporary **operator-launched** Cloudflare Quick Tunnel, document `cloudflared tunnel --url http://127.0.0.1:18844`, copying the issued HTTPS origin into `boxweave http configure --public-url ...`, then entering `https://.../mcp` in ChatGPT and approving OAuth. Tunnel hostname changes require reconfigure. Quick Tunnel is temporary and not a managed production service; cloudflared is neither installed nor launched by BoxWeave.
- QH-07: Present local TCP/HTTP discovery, authenticated MCP tool read, external HTTPS and live user-client evidence as **separate** levels. Never mark a public URL verified because a local socket is open.

## Implementation slices

1. QH-A: loopback/public runtime split, strict origin/host validation, negative OAuth and URL tests.
2. QH-B: installed service and preflight/update migration, socket-conflict and explicit-disable tests; installer banner.
3. QH-C: safe operator CLI, configuration transaction, status/doctor/restart and tests.
4. QH-D: docs, compatibility matrix, CI and release.

## Acceptance matrix

| Mode | Check |
| --- | --- |
| Fresh activated install | User-systemd serves Web, worker, remote-mcp; local discovery metadata and authenticated read; no public listener |
| Upgrade from 0.26.x without remote config | Existing private config/operator keys preserved; new loopback endpoint starts |
| Upgrade with remote HTTPS config | Existing URL, credentials and Box records preserved; remote unit takes over after prior manual process stops |
| Explicitly disabled | No remote unit started; no public exposure |
| Hosted HTTPS | Operator configures URL and independently starts proxy/Tunnel; OAuth browser consent and authenticated tools/list work |
| Negative security | Missing/wrong credentials 401; Origin null/wrong 403; DNS rebinding Host 403; malformed URL fail-closed |
| Failure/recovery | Port conflict visible, no replacement of foreign units, failed upgrade marker retained, retry repairs |
| Existing functionality | SSH MCP, Web UI, Provider and agent-box fixture tests remain green |

## Release gates and unresolved live evidence

Run `python scripts/check_knowledge.py`, `python -m unittest discover -s tests -v`, frontend build, seven fixture browser flows, packaged wheel/ZIP checks, and GitHub Actions PR/main publish. Record exact commit and actual pass/fail. A successful CI build does not prove the v0.27.0 managed unit started on a real Linux host or Cloudflare callback was retested. Public port/route/host changes require separate scoped authorization. Publish immutable release only after CI; promote `PUBLISHED_RELEASE` and `install.sh` together only after release tag/assets exist.

Return to [remote master](../design/REMOTE_MCP_MASTER.md), [verification](../verification/REMOTE_MCP_TEST_PLAN.md), [readme](../../README.md).
