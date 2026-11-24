# S3 Phase 1 — source implementation of three native Guest File tools (2026-11-15)

**Scope:** Existing [Draft PR #152](https://github.com/vanishs/boxweave/pull/152), based on scoped S2.5 protected v1. Phase 1 is a **source+basic test milestone**; it is **NOT S3_MINIMAL_PASS**, has not been installed on the physical L0, and is **not** evidence for the phase-2 malicious native File matrix or phase-3 Agent Git loop.

**Implementation source checkpoint:** `a9c47821a3ede92ce57e2c5a2efe9e63274378aa` (before this documentation commit). The exact-head format/quality and both architecture candidate builds from this source passed: [quality](https://github.com/vanishs/boxweave/actions/runs/43798024377), [amd64/arm64 candidates](https://github.com/vanishs/boxweave/actions/runs/43798024417). S1/race and fresh native positive file evidence must be checked separately; their status is not inferred from a successful build.

## Added functional surfaces (20 public MCP tools not yet activated)

- `box_file_list(box_id,path,cursor?,limit?)`: registered typed strict tool and Reader through official Incus SFTP in the exact restricted Project + current UUID; canonical guest paths, reject symlink ancestors, stable lexical reply order with `next_cursor`, up to 50 metadata results and explicit `consistent_snapshot=false`. SFTP v1.13.12 `ReadDirContext` buffers the directory before application count is enforced (4096 limit): therefore the memory-size bound of a hostile enormous directory remains an **open phase-2 hardening case**, NOT proven memory-hard bounded here.
- `box_file_read(box_id,path,offset?,length?)`: registered typed role-private official native SFTP read; regular files and symlink ancestors only; 64KiB max per call, base64 original bytes, offset/total size, `page_sha256`, `whole_file` only for an observed full-page read and readback unchanged size+mtime; no made-up global immutable digest/snapshot guarantee under concurrent malicious Guest writers.
- `box_file_apply(box_id,path,mode,content_base64?,idempotency_key,work_ref?,agent_ref?)`: `mode=create` accepts only at most 64KiB decoded bytes and uses verified native SFTP `O_WRONLY|O_CREATE|O_EXCL`, after target-parent Lstat and exact UUID Project checks. Reopens exact original Guest bytes and compares full SHA before returning an observed success. Existing target is a verified no-effect `FAILED` and releases lock; ambiguous write or lost reply remains `UNKNOWN` and locks original. The public `replace/delete` modes are retained but intentionally return `UNSUPPORTED` without mutation: Incus v7.6 native SFTP provides no atomic same-file CAS precondition.
- SFTP has **no Incus async native Operation ID**. `internal/state/file_create.go` uses the existing SQLite ADMITTED→DISPATCHING Claim and one guarded final SUCCEEDED + audit event commit, with **no invented native_operation_id**. Raw base64/source content is not stored in `safe_intent`, only path/size/digest. Same token+key replay returns original ID; different key cannot bypass original UNKNOWN resource lock. No host or guest shell is introduced.

## Specific code and basic tests

- New source: `internal/control/file_tools.go`, `internal/native/file.go`, `internal/state/file_create.go`; existing `internal/control/server.go` wires Tools, and `go.mod` promotes already-pinned `pkg/sftp` to a direct dependency.
- New basic tests: `internal/control/file_tools_test.go` actual official MCP catalog/viewer denial/malformed input; `internal/native/file_test.go` canonical/magic path and type negatives; `internal/state/file_create_test.go` original-success single terminal event, no native Operation ID, replay, error/UNKNOWN locks, and true existing-target no-effect. Legacy 13-tool inventory source tests were updated to 16 tools; remaining S4 four tools still not implemented.
- Original format checker, immutable RC6 assets, shared physical host/L0–L5 and unreconciled UNKNOWN were not changed. Tests use disposable fixture DBs/CI only.

## Explicit next work (NOT yet signed)

**S3 Phase 2 native negative:** real Incus 7.6 installed official MCP File list/read/O_EXCL with actual Guest bytes/sha and dual-actor viewer, symlink/hardlink/special path, concurrent exclusive-create and no-effect negative native readback; prove or fail-close directory page memory budget, race of parent-directory rename vs SFTP nofollow (SFTP v3 lacks `openat2`), and client disconnect/operation original semantics. If stronger hostile-guest guarantee cannot be provided, reject that capability explicitly rather than claim it passed.

**S3 Phase 3 Agent:** exactly one supervised installed authenticated Guest Git → deliberately failing test → original operation output → corrective Guest Exec/Git → passing test → independent native/file postread; full TC-015/016 multi-Agent stays S5.

**S3 Phase 4 signoff:** same-head quality, race, dualarch, actual installed MCP and independently-authored real Incus verifier plus affected S1/S2/S2.5 regressions, update current acceptance matrix and only then consider normal merge of the same PR #152 into protected v1. No new PR needed.
