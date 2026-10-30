# F-001–F-015 cross-branch semantic parity (R-03)

Source anchors: `archive/0.5.1@d8a09dd180139159e035963502e7793a37e92bae` and pre-remediation `main@19b3b61c75c85754d6b53699f5bfcdb4e0791b04`, supplemented by current remediation source. **S** = selected source read, **D** = approved product boundary, **U** = not actually executed in this audit. The matrix below is *not* fresh runtime acceptance; [FEATURE_MATRIX](FEATURE_MATRIX.md) owns REQ/AC evidence and [OPEN_RISKS](OPEN_RISKS.md) owns live risk status.

| Feature | Old path → new path | Permission/error/recovery difference and disposition (S/D) | User impact and required regression (U) |
| --- | --- | --- | --- |
| F-001 installation | `install.sh`, `server/web_install.py` → new `install.sh`, `distribution.py` | **Gap**: new app installer stages hash-verified release/backup but a full host Provider/SSH/credential launcher is not generated | AC-019: empty Ubuntu, preexisting host refusal, user bus, admin/MCP, repair and reboot |
| F-002 client setup | `server/clients.py`, Windows/POSIX scripts → `profile_client.py`, `ClientSetup.tsx` | **Partial**: four Windows-hosted profiles only; explicit trust/replace, no macOS | AC-001/002/016: hash/pin, path/Unicode, rerun and conflicts |
| F-003 Agents | `agents/{codex,cursor,claude-code,zcode}` → selected Codex/ZCode | **Deferred**: Cursor/Claude and macOS excluded by user decision; no automatic GUI claim | AC-017: actual four-profile registration and UI independent evidence |
| F-004 model GLM | Old Claude model adapter → `models.py`, `ModelWizard.tsx` | **Replaced**: approved native ZCode handoff, no Web secrets | AC-015: preserve settings; actual access/inference separate |
| F-005 diagnosis | Old CLI/client diagnosis → `profile_client.py`, `deploy_check.py` | **Partial**: actual MCP read is independent of empty inventory; registration/UI distinct | AC-003: dead launcher, malformed MCP and trust mismatch |
| F-006 management | Old `server/cli.py` → `manage.py`, `boxweave.sh` | **Partial**: safe version/status/doctor/exact Incus inspect, no auto host repair | AC-020: missing service, exact name, failed read, secret scrubbing |
| F-007 jobs | `server/jobs.py` → `store.py`, `service.py`, `lifecycle.py` | **Selected replacement**: durable intent/deadline/unknown/no replay; old generic job alias absent | AC-005/006/007: simulated crash and receipt restoration |
| F-008 selftest | Old `smoke.py` → `validate.py` and fixture browser | **Partial/deferred**: isolated synthetic hello, no recursive host selftest | AC-017: fixture versus authorized live execution |
| F-009 Web admin | Old `server/webui/*` → `web.py`, login UI | **Reimplemented**: one admin, local reset, Origin/CSRF; not team RBAC | AC-011/018: invalid session/reset/reject |
| F-010 Boxes | Old Box Web/jobs → `lifecycle.py`, `Environments.tsx` | **Selected subset**: single exact create/start/stop/delete via plan; no bulk/resize | AC-004/005/006: stale plan and unknown action |
| F-011 network routes | Old proxy/passthrough → `AccessPanel.tsx`, `access_provider.py` | **Selected subset**: exact TCP/UDP passthrough, no automatic domain HTTP proxy/TLS | AC-013: collision, identity reuse and real packets separately |
| F-012 monitoring | Old Overview/Monitoring → Projects, Environments and receipts | **Partial**: observed readiness; no invented telemetry product | AC-006/009: unknown/missing state UI |
| F-013 Tenant views | `server/webui/tenants.py` → `access.py`, AccessPanel | **Reimplemented**: provider-identified membership not proof of isolation | AC-012: real member/denial state |
| F-014 Tenant native policy | Old `access.py`/retired patch → `access_provider.py` | **Reimplemented selected native protocol**: capability gate, revisioned grant/revoke, degraded Guard not enforced | AC-012/018: unsupported capability, actual Guard and packet proof |
| F-015 Prompt Graph | Old graph/UI → `graphs.py`, `PromptGraph.tsx` | **Reimplemented**: persistent CAS cards/comments/links, redacted export, no execution | AC-014/016: conflict, clipboard failure, zero infra calls |

### Legacy-only path disposition

- `server/{hostkeys,tokens,provider_upgrade}.py`, old token/key timers, Caddy and upstream bootstrap have **no complete v2 app counterpart**. Host-side ownership and first-host end-to-end integration remain open in AC-019, not silently copied.
- `clients/{posix,macos}/*` and old `agents/{cursor,claude-code}/*` are **intentionally deferred**; do not create fake compatibility aliases.
- `web/browser/*` and legacy Playwright/model test helpers are **different suites**, not a proof of regressions or parity; map assertions using [Web workflow review](WEB_WORKFLOW_PARITY.md).
- `patches/containarium/*` is an obsolete historical compatibility candidate; do not apply to current Provider.
- Old `publish.*` and root installer are not an automatic trusted publisher or schema migration. Exact signed artifact provenance remains a separate obligation.

Approved decisions: restore primary host installer and read-only/management entry, maintain four selected Windows-hosted profiles, no automatic legacy database migration. Remaining real tests and attribution: [verification queue](REMEDIATION_TEST_QUEUE.md).
