# Archive 0.5.1 to main — source audit (R-00)

Baseline checked 2026-10-29, read-only via GitHub refs/tree and selected source:
- `archive/0.5.1@d8a09dd180139159e035963502e7793a37e92bae`
- `main@19b3b61c75c85754d6b53699f5bfcdb4e0791b04` (before remediation)
- Unrelated Git histories: compare semantics/path/behavior; no ancestry diff. No actual program or host tests were run in this audit. Evidence: **S** source, **D** decision/docs, **T** actually executed test, **U** unverified.
- Current scope: [SCOPE](../product/SCOPE.md); current evidence: [FEATURE_MATRIX](FEATURE_MATRIX.md); current risks: [OPEN_RISKS](OPEN_RISKS.md). This is a dated comparative evidence document, **not** a second project status owner.

| Legacy capability | Legacy path | Current counterpart | Disposition and evidence | Remaining acceptance |
| --- | --- | --- | --- | --- |
| F-001 integrated install, update, repair | `install.sh`, `publish.*`, `server/web_install.py` | `bwmini/distribution.py`, `docs/INSTALL.md` | **Gap** in first-host orchestration; independently packaged app exists (S); owner requests restored experience | R-02 bootstrap, new-host rerun, recovery, no provider destruction (U) |
| F-002 client onboarding | `server/clients.py`, `clients/*/setup*` | `clients/profile_client.py`, `ClientSetup.tsx`, Windows/WSL scripts | **Partly replaced**, narrowed to four Windows-hosted profiles (S/D) | Windows path spaces, hash/pin, actual Agent E2E (U) |
| F-003 Agent integration | `agents/{codex,zcode,cursor,claude-code}` | `clients/README.md`, profiles | **Replaced for Codex/ZCode; deferred** Cursor/Claude/macOS (D) | Selected four runtime/client confirmations (U) |
| F-004 GLM wizard | old Claude adapter/client UI | `bwmini/models.py`, `ModelWizard.tsx` | **Replaced**, native ZCode handoff instead of Claude (S/D) | Actual native settings/inference separately (U) |
| F-005 diagnosis | `server/cli.py`, `server/mcp_client.py` | `bwmini/deploy_check.py`, `__main__.py diagnose` | **Partly replaced**; system-facing doctor CLI missing (S) | Read-only scoped diagnostics and error contract (U) |
| F-006 target health/inspect | `server/cli.py` | `bwmini/provider.py`, `service.py`, `deploy_check.py` | **Partly replaced**; exact Incus inspect CLI absent (S) | Read-only inspect and strict naming (U) |
| F-007 durable jobs | `server/jobs.py` | `bwmini/store.py`, `service.py`, `lifecycle.py` | **Replaced in selected subset** (S/D) | Crash, unknown/no-replay and timeouts (U in this audit) |
| F-008 selftest/smoke | `server/smoke.py`, `docs/TASKS-AND-SELFTEST.md` | `scripts/validate.py` + fixture tests | **Partly replaced**; old remote selftest intentionally not inherited (S/D) | Distinguish fixture from live (U) |
| F-009 Web admin | `server/webui/app.py`, `web/src/main.tsx` | `bwmini/web.py`, `web/src/main.tsx` | **Reimplemented** (S) | Auth reset/cookie/CSRF/browser (U in this audit) |
| F-010 boxes | old Web box UI, jobs | `lifecycle.py`, `Environments.tsx` | **Reimplemented subset** with explicit confirmation; bulk/resize deferred (S/D) | Lifecycle conflict/unknown (U) |
| F-011 network/publication | old Web routes/access | `access_provider.py`, `AccessPanel.tsx` | **Reimplemented TCP/UDP subset**, no generic HTTP routing (S/D) | Actual packet/Guard proof (U) |
| F-012 monitoring | legacy Overview/Monitoring | `provider.py`, project/environment state | **Partly replaced**; full historical UI parity not established (S/U) | Browser screens, actual telemetry limits (U) |
| F-013 Tenant inventory | `server/webui/tenants.py` | `bwmini/access.py`, `access_provider.py` | **Reimplemented** (S) | Membership version/permissions (U) |
| F-014 Tenant native network | `server/webui/access.py` and retired patch | `access_provider.py`, capability guards | **Reimplemented selected upstream contract** (S/D) | Fresh Provider protocol and Guard proof (U) |
| F-015 Prompt Graph | old `web/src`, `server/webui/graph.py` | `graphs.py`, `PromptGraph.tsx` | **Reimplemented** with revised scope (S/D) | Save conflicts, export redaction, browser (U) |

## Path disposition and boundaries

- Legacy `install.sh` and `boxweave.sh`: newly approved **functional goals**, not reusable code; first-host orchestration and CLI require independent design/implementation.
- Legacy `server/hostkeys.py`, `server/tokens.py`, `server/provider_upgrade.py`: no v2 app replacements; the external host/Provider operator owns these until explicitly designed and verified. Not silently enabled by the new installer.
- Legacy `server/web_install.py`, `server/web_token.py`: partial replacement by app package + local administrator. Public TLS/reverse proxy and credential lifecycle remain out of app installer.
- Legacy POSIX/macOS and Cursor/Claude guides: archived, intentionally not new support claims.
- Legacy Playwright tests replaced by Python fixture-browser lane in `scripts/*_browser_check.py`; not proof of matching UI assertions.
- Legacy `patches/containarium/*`: obsolete compatibility evidence; do not copy or apply patches to an unverified upstream. Native capability negotiation is the v2 contract.
- Legacy `docs/{CLI,CLIENTS,INSTALL-UPDATE,WEB-UI,TROUBLESHOOTING,MIGRATION,SECURITY}.md`: subjects are being remapped to current install, client, operator and migration docs, rather than copied.
- Legacy `.gitattributes`: revisit current shell/Python newline and release packaging policy.

## Decisions already obtained

Owner confirms: restore **integrated fresh Ubuntu installation/update/repair** and a useful `boxweave.sh` CLI, keep the four-profile Windows scope, do **not** auto-migrate 0.5.1 data, no automatic Provider upgrade. Version line begins `0.10.0`; small commits to `main` explicitly authorized. Each feature and its tests belong to a separate source commit. Unsupported/untested behavior remains in [verification queue](REMEDIATION_TEST_QUEUE.md).

## Nesting knowledge (SRC-103)

Read-only origin/archive/0.5.1 patch maps EnableNesting to EnablePodman. Retain prerequisite knowledge, reject coupling for new explicit selections.0.16.0 uses independently implemented capability and parent handoff, never applies old Provider patches. [Evidence](NESTING_VERIFICATION.md).
