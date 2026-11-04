# v0.43.3 — Containarium running-daemon minimum-version guard

**Date:** 2026-11-03 (Pacific). **Status:** source candidate; actual v0.128.0 operator-host compatibility still unverified. **Stable pointer:** `PUBLISHED_RELEASE=0.43.1`, no default streamed installer update.

## Why a follow-up to v0.43.2?

The previous [P0 compatibility slice](CONTAINARIUM_0128_COMPAT_P0_0432_2026-11-03.md) checks the trusted on-disk `containariumd version >= 0.128.0` and authenticated MCP server version >=0.128.0. That alone could miss a still-running v0.120.1 daemon after an operator swaps binaries without restarting the service. A successful old API read is not evidence the executing daemon satisfies the selected minimum.

## Source change and contract

- `bwmini/deploy_check.py` now makes an **authenticated** read-only `GET /v1/system/info` and requires `info.daemonVersion >= 0.128.0` in addition to checking the MCP server's `initialize.serverInfo.version >= 0.128.0`. These are running processes, not binaries on disk.
- The request uses the existing configured loopback REST client, rotated app token, HTTP no-redirect/no-proxy policy and bounded response; it issues no writes.
- Versions greater than minimum, including future minor/major releases, remain allowed **without a maximum or allowlist**. Missing/malformed data and older running daemon receive `PROVIDER_RUNTIME_BELOW_MINIMUM_OR_UNKNOWN` and fail the authenticated deployment gate.
- `tests/test_provider_minimum_versions.py` tests future and minimum daemon versions, denial of older/unknown running daemon, and existing MCP checks. A failed gate never silently downgrades the selected Provider.
- The v0.43.2 binary/download/source evidence is not invalidated or rewritten; this is a new immutable BoxWeave patch release, not a stable promotion or Containarium change.

## Acceptance ledger

| Check | Evidence | State |
| --- | --- | --- |
| Authenticated running daemon read and minimum-only comparison | Source + regression | CI pending |
| Old daemon process after new binary installed | Synthetic fixture | CI pending |
| General Python/Web/browser/package regressions | PR CI | Pending |
| Exact published v0.43.3 release | main CI/publisher | Pending |
| Real `dev` host upgraded to Containarium >=0.128.0 | Operator | Not run |
| Stable selector remains 0.43.1 | Source guard | Must remain unchanged |

For real host testing follow the [standard upgrade SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md). The operator upgrades Containarium first; then installs the **exact v0.43.3** BoxWeave release. Do not treat a version read, health check, or simulated test as live SSH, data recovery or network acceptance.
