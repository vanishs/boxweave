# RC8 — Private Exec archive and Activity issue clarity: isolated implementation (2026-11-19 PT)

**Scope:** Draft PR #153 source only, branch `feat/v1-s4-runtime-integration-20261116`. No installed L0 binary upgrade, merge, RC release, physical quota verification, or installed-native S5 signoff is claimed. Related original [ChatGPT Agent observation](V1_RC8_REAL_CHATGPT_AGENT_ACTIVITY_RECOVERY_OBSERVATIONS_2026-11-19.md).

## Implementation

- Immutable operation IDs, events, exit results and native recorded-output paths stay in SQLite. New verified native Exec streams are copied **before the original claim is released** to a private controller archive at `<sqlite-db-path>.exec-archive`. The directory is service-owned 0700; each exact operation-ID/stream file is 0600, never from caller-selected paths, without symlink/hardlink or overwrite acceptance.
- Limits: first **1 MiB per stream**, **256 MiB total** and 4096 files per controller DB; bounded `statfs` spare-space preflight of 64 MiB. Truncation and archive failures must not forge native operation success/failure. This preflight is **not** evidence of host Btrfs quota enforcement.
- `operation_output_get` authenticates the original scoped operation and current TokenID, checks the seven-day TTL anchored to the durable final Exec witness, then prefers the private archive even when the Box has been deleted. Missing archive falls back to exact native output only for an existing owned Box; otherwise it preserves known exit details with structured `unavailable` and `BOX_REMOVED_OUTPUT_NOT_ARCHIVED` (or `EXACT_NATIVE_BOX_NOT_AVAILABLE`). Expired output is `availability=expired`, reason `RETENTION_WINDOW_ELAPSED`, with no returned bytes.
- Every 15 minutes the existing maintenance loop also removes expired private files, at most 64 per sweep, without replaying Guest/Incus operations or deleting SQLite evidence. Crash-orphan captures use file age. The original native Exec output GC and its no-replay/host-review semantics remain separate.
- `box_activity_get(view=issues)` continues to enumerate current SQLite-visible blockers and dependencies; each gets a `disposition` such as `HOST_REVIEW_REQUIRED`, `IN_PROGRESS_OR_STALLED` or `ACTIVE_DEPENDENCY_NOT_ERROR`. A separate bounded `history` sample (maximum five) classifies FAILED terminal operations and ordinary UNKNOWN whose resource claim was reviewed and RELEASED; this is **not** proof of resolved native effects. Special Root Peer UNKNOWN remains an active barrier.
- `coverage_details` explicitly marks authorized-project-only, SQLite-only, no live-native check, page truncation, independent pagination snapshots and `attention_complete=false`. A blank active list is never a global all-clear.

## Recorded tests

- Isolated owned L0 development Box `cdda8d43-73f5-44da-81b9-c34ce656c5c9`, work ref `rc8-activity-output-issues-fix-20261119`. Initial source `3934d573ca0302cb57dd6d93b2aeb6107f64363c`, Go 1.26.9 and pinned MCP/Incus SDKs.
- Focused Box Exec `bf3ee30aa18bc18d13df5f0eefa7d450`: **PASS / exit 0** for real localhost MCP archived stdout read without an owned Box; separate exit, seven-day expiry and filesystem GC; >1 MiB truncation; symlink refusal; and current vs released UNKNOWN vs FAILED History categorization.
- Original full Go package test attempt failed **only** on existing `strict_tool_test` expecting `next_offset` in Tool description. Pagination text was restored; original failure is not hidden.
- Further full Go, race, formatting and CI gates require their own exact results. There is no new candidate installed in the shared physical Host, so no true native Box Delete/readback or qgroup acceptance may be claimed.

## Known limitations and release gates

- Archived output is a **bounded 1 MiB prefix**, not the native 16 MiB page-reading ceiling. It is best effort under storage pressure; the API must report missing/expired rather than invent bytes.
- Already-deleted historical Boxes are not retroactively recoverable. File content remains private and is never inserted into Activity, general logs or immutable Operation events.
- Scoped uninstall/controller replacement must explicitly handle adjacent private archive files, preserving ownership and requiring authorization for destructive cleanup.
- Before release: verify installed exact-source Create/Exec/Delete/output, free-space/aggregate-cap exhaustion, Crash/GC, OAuth revocation, and physical independent Host quota/qgroup witness; root-network and S5 multi-Agent gates remain separate.

## S4-A original native output regression — 2026-11-19 PT

**Root cause confirmed:** [S4-A installed-native run #44674010605](https://github.com/vanishs/boxweave/actions/runs/44674010605) at source `cf8c080` failed the baseline original recorded-output probe with `real recorded stdout breached 16MiB public read cap or hid truncation`, then did not reach the independent native spool `output-witness.ready` rendezvous. The new controller-private 1 MiB archive was preferred even while the exact original Box still existed, incorrectly shortening the original native 16 MiB pagination contract. The second failure is downstream, not an independent absent-native-output finding.

**Repair:** `operation_output_get` now checks the authorized original and TTL, then reads the **original native recorded stream first** if the exact owned native Box still exists, honoring the original 16 MiB offset and truncation semantics. Only after a missing/unavailable original native stream or Box deletion does it use the private bounded 1 MiB archive. It never replays Exec, widens the allowed native path, exposes a foreign Layer, or mistakes successful transport for Guest exit 0. This retains both 16 MiB live-Box original evidence and seven-day bounded post-Delete recovery.

**Local isolated acceptance:** Dedicated new L0 Box `217c7b3b-c80b-4848-a8a0-fd7bd72a2150` at `work_ref=rc8-s4a-native-output-regression-20261119`; focused state/control/native tests PASS and full `make verify` PASS with CGO Go Race, lint, scripted MCP and unittest regressions. This is **not** a substitute for rerunning exact-source S4-A native/Incus 7.6 CI; no installed shared L0 host was changed.
