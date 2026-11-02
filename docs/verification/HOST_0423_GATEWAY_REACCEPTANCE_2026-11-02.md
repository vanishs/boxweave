# Updated host v0.42.3 — live MCP reacceptance (2026-11-02)

[Previous v0.42.3 new-Box acceptance](REACCEPTANCE_0423_R01_2026-11-02.md) · [Feature matrix](FEATURE_MATRIX.md) · [Verification queue](REMEDIATION_TEST_QUEUE.md) · [Risk register](OPEN_RISKS.md)

**Overall: scoped live PASS for host Gateway R-01 and guarded reads; whole-project acceptance remains PARTIAL.** This check was performed **after the operator reported upgrading the real host to v0.42.3**. It is independent of the prior disposable inner-host install. Evidence L = actual connected MCP/Box result, T = executable source/CI evidence, U = not run. No host settings, trust, privilege or network policy was changed.

## Identity and boundaries

- Current repository main at check: `c1293365394f4a1b618f655acfbefdf04b1d1696` (release code: immutable [v0.42.3](https://github.com/vanishs/boxweave/releases/tag/v0.42.3), source commit `7041fe0faa67c2bedd6b59dbaaaf36a0af1f65f0`). User-reported host v0.42.3 aligns with connected `boxweave_info.source_version=0.42.3` and observed 0.42.3 error typing. **A host-local `boxweave version` or running Web/worker process version was not independently read**, so this is Gateway live behavior evidence rather than full post-upgrade installed-service attestation.
- Independent Provider query: host `dev` (Ubuntu; kernel `7.0.0-42-generic`), Containarium daemon `0.120.1`, Incus `7.5.1`. `list_containers` returned the original **seven** managed Boxes, all running; no new outer Box was required.
- Target for bounded file tests: the previously prepared development Box `bwphase2-1102`, immutable observed `createdAt=1793614975`, `CONTAINER_STATE_RUNNING`, `10.100.0.8`. A uniquely named task-owned ephemeral home directory was created solely for this read-only gateway test and removed after it. Historic Box installations and root-owned host files remained unchanged.

## Actually executed host MCP checks

| Test | Evidence | Result |
| --- | --- | --- |
| R-01 path validation: absolute `/etc/passwd`, `../etc/passwd`, and list traversal `../etc` | L | **PASS**: all three requests returned `BOX_PATH_INVALID`, not the previously misleading generic trust error; no private bytes returned |
| R-01 positive existing checkout read and directory listing | L | **PASS**: `audit-main-20261102-r2/VERSION` returned base64 of `0.42.3\n`, size 7; existing directory enumeration and `next_offset` pagination succeeded |
| R-01 exact binary window/EOF/hashes | L | **PASS**: task-owned 30-byte file `BW-R01-HOST-0423\\0binary-finish`; two windows (10 and 20 bytes) recombined to exact original, their SHA-256 values were independently recomputed, EOF false then true, offset 30 returned an empty EOF=true chunk |
| R-01 guest symlinks and intermediate symlinks | L | **PASS safety**: direct file symlink to `/etc/passwd`, symlink directory and a file via symlink parent were all refused with `BOX_READ_UNVERIFIED`. Directory entries classify links `other`; paging returned expected kinds and next offset. No raw guest failure stderr was returned |
| Exact-target SSH guard | L | **PASS**: conflicting `connect(host=1.2.3.4)` returned `TARGET_OVERRIDE_REFUSED` without executing supplied guest command |
| Older existing Box read compatibility | L | **PASS scoped**: home directory lists succeeded on `boxweave-test-ser`, `boxweave-test-ser2`, `bwaudit-1101` without reinstallation or mutation. This does not establish old-to-new host data migration |
| Provider observation, nesting inspection | L | **PASS scoped**: `get_system_info`, exact `get_container`, `get_metrics` and read-only `boxweave_nesting(action=inspect)` returned structured state. No claim of effective nested child execution from inspect alone |
| D3 classified diagnostics | L | **PASS read path, accuracy gap**: `boxweave_diagnose(source=gateway,recent=...)` returned bounded event entries; D3 app reader on a deliberately missing trace returned `not_verified` / `trace_not_found` without exposing private content |
| D4 default-deny project diagnostics | L | **PASS negative, positive U**: unallowlisted `boxweave_project_diagnose(check=processes)` returned `PROJECT_DIAG_NOT_AUTHORIZED`; no authorized scoped-service read executed |
| R-02 unprovisioned gateway tasks | L | **PASS negative, positive U**: `boxweave_task_list` returned `TASK_TRACKING_UNAVAILABLE`; an invalid `task_get` identifier returned `TASK_SELECTOR_INVALID`. No root-owned task store was provisioned, so recovery, deduplication and restart persistence are **U** |
| Cleanup | L | **PASS**: inside original development Box, verified the exact owned four entries and expected file bytes, unlinked only these owned entries, removed the task directory, confirmed it no longer exists. Seven preexisting managed Boxes retained |

## Residual issues and honest acceptance limits

1. **D3 diagnostic classification friction (P3, open; not an access bypass):** For multiple deliberately rejected R-01 and exact-target requests, the Gateway journal recorded `outcome=unknown` and `code=UPSTREAM_JSONRPC_ERROR` even though the public MCP error was a deterministic `BOX_PATH_INVALID`, `BOX_READ_UNVERIFIED` or `TARGET_OVERRIDE_REFUSED`. Current source `bwmini/gate_diagnostics.py:classify` maps *every* JSON-RPC `error` into that generic bucket. Consider differentiating reviewed fixed local validation/refusal categories from genuine uncertain upstream writes, without storing payloads, dropping safeguards or ever auto-retrying an UNKNOWN mutation. Not changed in this reacceptance.
2. **Agent client catalog refresh (U):** The current preloaded connector tool descriptions still contain the **previous** R-01 path wording, whereas the connected self-guide reports source v0.42.3, live error handling is corrected, and the prior fresh v0.42.3 installed `tools/list` returned the revised home-relative descriptions. A fresh client/plugin reconnect followed by an independent new `tools/list` should confirm the current catalog. Current session metadata alone cannot establish a running Gateway regression.
3. **Blocked/U**: host-local Web/worker version and old-host migration; real external SSH/HTTP/OAuth desktop clients and cross-transport concurrency; protected R-04 global shared policy/lease/guarded CAS races; R-02 provisioned storage and reconnect deduplication; D4 allowlisted positive checks; E2–E4 snapshot/restore/clone unsupported Provider contract; full worker-crash/UNKNOWN recovery; live model inference. None is promoted to PASS by this read-only host check.
4. **Version boundaries**: this live host test did not update the host, default stable installer, any historical Box's installed software or public TLS. `PUBLISHED_RELEASE` remains v0.42.1 until independently promoted.

**Disposition:** post-upgrade Gateway R-01 positive/negative read, old-Box read compatibility, exact-target trust refusal, D3 bounded reads and D4/R-02 default-deny **L PASS in tested scope**; D3 error-classification quality **P3 open**, all listed positive/high-impact scenarios **U/BLOCKED**, overall **PARTIAL**.
