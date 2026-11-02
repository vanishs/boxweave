# D3 deterministic denial classification — v0.42.4 (2026-11-02)

[Updated-host discovery](HOST_0423_GATEWAY_REACCEPTANCE_2026-11-02.md) · [D3/D4 design](../design/DIAGNOSTICS_D3_D4_MASTER.md) · [Verification queue](REMEDIATION_TEST_QUEUE.md) · [Risk register](OPEN_RISKS.md)

**Source scope:** Fix the upgraded v0.42.3 Gateway journal's misleading `UPSTREAM_JSONRPC_ERROR / unknown` for deterministically denied requests. No grant, retry, client response, storage schema, D4 privilege, R-02 tracked mutation or R-04 protected policy changes.

## Implementation and safety

- `bwmini/gate_diagnostics.py` now accepts an **internal-only** `local_denial` classification hint. Only fixed tool-specific, reviewed denial codes for read-only and pre-dispatch operations can become `outcome=failed` and a matching safe code. The raw provider `error.message` is never accepted as proof or persisted. Other JSON-RPC errors remain `UPSTREAM_JSONRPC_ERROR / unknown`.
- `bwmini/mcp_gate.py` provides the hint only in its own caught-exception path, after normal public fixed-code sanitization. It does not alter the original MCP error response or introduce a retry.
- Read-only Box file/directory path/argument/symlink failures and exact-target `connect` override refusal have deterministic labels. Read-only task-store unavailable and selector errors are similarly categorized. Mutation routes (`boxweave_task_submit`, `boxweave_nesting`, arbitrary Provider mutators) are excluded from the deterministic list; their post-dispatch uncertainty remains `unknown`.
- Private journal keeps bounded `tool`, `trace_id`, `utc`, `outcome`, fixed `code`, `elapsed_ms`; no arguments, command, stdout, stderr, credentials, or raw exception. Public codes still use the existing gateway allowlist. No migration necessary.

## Executed source checks (T)

- Prepared existing development Box `bwphase2-1102`, clean base `2898a6414769616fbebff1ed0e44ce1ceee6e24b`, Python 3.12 with pinned project dependencies. Source tree changed only by this task; no old Box installation or host configuration modified.
- Before source fix: new deterministic-denial tests **failed as expected** because `record()` did not accept a local hint. After fix: focused D3/R-01 suite **21/21 PASS, exit 0**; full Python suite **644/644 PASS, exit 0**; architecture **4/4 PASS, exit 0**; documentation knowledge check **166 documents, 26 requirements/acceptance entries, exit 0**; `bash -n install.sh` and `git diff --check` exit 0.
- Regression includes a real `mcp_gate.main` loop with simulated empty Provider that proves no Provider dispatch, the original error string `BOX_PATH_INVALID`, and persisted journal classification `failed / BOX_PATH_INVALID`. Explicit checks retain `unknown / UPSTREAM_JSONRPC_ERROR` for untrusted upstream error payload, unreviewed code, wrong tool-code pair and potentially mutating task/nesting routes. No secret fixture marker appears in the journal.

## Release and installed checkpoint

- Source candidate: `v0.42.4`. Version files and installed Gateway serverInfo aligned. Default `PUBLISHED_RELEASE` and `install.sh` default remain **v0.42.1**, by deliberate separate stable-pin policy.
- **At source test time:** exact GitHub CI, immutable v0.42.4 tag/release, new-Box pinned install, installed Gateway `boxweave_diagnose`, installed original-client SSH/HTTP behavior, D4 authorized positive and real unknown mutating dispatch are **L/U**. Record observed outcomes in a separate post-release addendum; do not infer them from source/CI.
- Existing v0.42.3 journal rows retain their historical code. No retroactive database rewrite; only requests processed by the updated Gateway can yield revised classifications.

**Status:** D3 exact source/fixture T PASS; installed acceptance **pending** at this checkpoint. Original issue scope and remaining external-client, R-02/R-04 and Provider gaps stay independently unaccepted.
