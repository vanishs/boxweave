# BoxWeave v0.43.2 P0 — Containarium minimum-only compatibility

**Date:** 2026-11-03. **Status:** immutable v0.43.2 released with source/CI acceptance; a v0.43.3 runtime-version-gate follow-up is now the recommended operator candidate. The real v0.128.0 Provider is still untested. **Stable default:** 0.43.1 unchanged. **Operator host:** Containarium 0.120.1 last observed and not upgraded by this work. 

## Selected contract

- `containariumd >=0.128.0`, `mcp-server >=0.128.0`; no upper cap or release allowlist.
- SHA-pinned official v0.128.0 assets only for first-time empty-host bootstrap. Existing managed/externally managed upstream daemons are not upgraded by BoxWeave `install.sh --version`.
- App-owned snapshots may be refreshed with re-verified digests. Strict trust and signed inputs; unknown/malformed version or MCP result refuses acceptance.
- REST error 404 on exact target GET differs from unknown route; mutating requests remain conservatively UNKNOWN after uncertain transport.
- Toolchain BUILD_TOOLS prerequisite is advisory for basic lifecycle and required by advanced code execution.
- No default stable version promotion; no new E2/E3/E4 write capability.

## Evidence ledger

| Gate | Evidence class | Status | Proof needed |
| --- | --- | --- | --- |
| Upstream source/Swagger comparison | S | Reviewed | [full review](../design/CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md) |
| Python minimum-version regression | T | Pending CI | `tests/test_provider_minimum_versions.py` |
| SSH wrapper pin/permission negative tests | T | Pending CI | `tests/test_containarium_shell.py` |
| Existing Provider adoption/digest | T | Pending CI | `tests/test_online_adoption.py` and installer tests |
| Independent authenticated MCP/REST | L | Not yet executed at 0.128.0 | Disposable isolated Provider, exact source/host evidence |
| Guest SSH/task/coordination/UNKNOWN | L | Not yet executed at 0.128.0 | Approved disposable Box, exact identity and receipt |
| Published v0.43.2 | P | PASS | [PR #65](https://github.com/vanishs/boxweave/pull/65) merge `952b7b677ad53fdb4b17268f76f8fe038ef6e0a8`; PR 710/710 Python, 7/7 browser; [main CI 41782830084](https://github.com/vanishs/boxweave/actions/runs/41782830084) success; [immutable Release](https://github.com/vanishs/boxweave/releases/tag/v0.43.2) |
| Operator-host v0.128.0 smoke | L | Deferred to operator | Follow [SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md) |

The CI status will be updated only from actual workflow results. A source-only release is **not** real-host compatibility certification.
