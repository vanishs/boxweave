# S1 Foundation — partial implementation and evidence, 2026-11-08

**Status: PARTIAL, not installed/native acceptance and not release eligible.** This is an execution artifact for [S1](../development/phases/01_FOUNDATION.md), not a replacement plan or relaxation of [ACCEPTANCE](ACCEPTANCE.md).

## Exact scope and candidate

- Source branch: `feat/v1-s1-foundation-20261108` from `v1` HEAD `e5c7ae26bdaa24f9243b7dec28dbd34ca561bcb2`. Candidate SHA must be read from GitHub at resumption; the branch received multiple fix commits.
- Official SDK pins: Go 1.26.9, MCP Go SDK v1.8.0, Incus Go SDK v7.6.0, SQLite go-sqlite3 v1.14.52; committed `go.mod` and `go.sum`. No v1 L0/L1 installed binary, digest, service or release is known.
- Final source/loopback verification: **PASS on exact source SHA `66b7e0eec4a94f5a29469598c975e2c0f7394fe9`** ([CI #42522526700](https://github.com/vanishs/boxweave/actions/runs/42522526700)); **overall S1 PARTIAL**.
- Source modules: `internal/control` (authenticated typed loopback MCP `host_status`), `internal/native` (confined Project readback), `internal/state` (SQLite schema v1, hashed revocable role tokens, immutable Tracking/operation intent, admission, fail-closed worker and UNKNOWN), `internal/install` (fixed-target *staging only*), `cmd/boxweave` (minimal local CLI). No Containarium or public REST management added.
- Official Go SDK HTTP integration test exercises local test server, authenticated list/call and negative token/Host/Origin; it is **not** deployed native acceptance.
- The installer intentionally stages but does **not** start systemd, repair Incus, authorize a host-level permission, or upgrade a daemon. Full rollback/upgrade/credential recovery remains **not implemented/accepted**. Do not call this a complete installer.

## Executed CI and correction trail

1. Initial source CI [run 42520224678](https://github.com/vanishs/boxweave/actions/runs/42520224678): Go1.26.9 SDK resolution and `go build ./...` **PASS**; race/unit **FAIL** in three SQLite tests because GitHub temp parent directory was more permissive than the ledger's intended security policy. The DB permission check was **not** weakened.
2. [run 42520445330](https://github.com/vanishs/boxweave/actions/runs/42520445330) after secure `chmod 0700` test fixtures: build and `go test -race -count=1 ./...` **PASS**, Go formatting **FAIL**.
3. A subsequent installer edit caused Go syntax/format parsing failure ([run 42521071684](https://github.com/vanishs/boxweave/actions/runs/42521071684)); the corrupted edit was restored from a source-pinned previous version, corrected and committed. Do not erase this prior failure.
4. One-time *feature-branch-only* CI formatting and checksum bootstrap produced [source commit d38d49d](https://github.com/vanishs/boxweave/commit/d38d49df12455a301278381eb1f00a3d78fe66f6). Temporary write-enabled workflow step was subsequently **removed**, restoring read-only CI.
5. Standalone binary direct HTTP negative test [run #42521689966](https://github.com/vanishs/boxweave/actions/runs/42521689966): status, Token creation, server launch, anonymous 401 PASSED; modern raw JSON-RPC `tools/list` 400 because the test omitted required `_meta` fields. Corrected source-neutral test request. [run #42522284849](https://github.com/vanishs/boxweave/actions/runs/42522284849): second 400 because test omitted `Mcp-Method` header; corrected and retested.
6. **Final complete source/loopback CI PASS**: [run #42522526700](https://github.com/vanishs/boxweave/actions/runs/42522526700) on exact source SHA `66b7e0eec4a94f5a29469598c975e2c0f7394fe9`. `go mod verify`, `go build ./...`, `go test -race -count=1 ./...`, `gofmt` clean, and compiled executable CLI/status/serve + anonymous 401 + modern MCP authenticated list with `host_status` + revoked Token 401 **all PASS**. These are ephemeral CI runner tests; not installed native/Incus tests.
7. Additional source-negative tests guard symlink SQLite adoption and unknown preexisting SQLite modifications; both passed in the final race/unit CI. Native/installed gates remain unchanged.

## Native/installed blocker — do not replay

A **single** authorized disposable test Box request was sent through the installed *old* BoxWeave API:
- Box name `bw-v1-s1-1108`, backend `dev`; original task `7e1cbf043d1fd4ea3d9ec0093a04371e`, request key `1d6007f5972bd21b01f89eacd42c3160`.
- Original create receipt `REST_GATE_PENDING` and then `UPSTREAM_CREATE_OUTCOME_UNKNOWN`, upstream acknowledged=false. One independent `get_container` returned NOT_FOUND; `debug_container` reported missing host/guest state in daemon 0.128.0. A later read of the **same original task** again reported UNKNOWN.
- **UNKNOWN remains**: a transient negative inventory is not authoritative evidence of no dispatch. No second create, auto-delete, new substitute Box, foreign ownership adoption or host-wide Incus change was performed.
- Consequence: no eligible disposable environment for verified installed v1 MCP/Incus, non-root native confinement, exact bootstrap, real worker crash windows or upgraded/shared daemon proof. Existing test Boxes were not reused or changed.

## Phase-specific acceptance inventory (tier and honest status)

| Acceptance IDs | Work attempted | Highest actual evidence / remainder |
|---|---|---|
| A-001, A-003, A-077 | Typed SDK MCP, role-specific request handling, HTTP auth/Host/Origin and single-tool tests | **Unit/loopback candidate only**, full modern+legacy/native V pending; future mutation viewer denial requires S2 |
| A-016, A-017, A-018 | SQLite caller key fingerprint, Tracking vs operation/native IDs, revoked queued tasks, ambiguity and restart-safe no replay | **Unit/source only**, crash/native ACK/poststate tests **BLOCKED** |
| A-002, A-032, A-033, A-071, A-072 | Local staging, pinned binary integrity, restricted-user/project design, CLI status/doctor | **PARTIAL SOURCE**, upgrade/bootstrap/rollback/installed/native **BLOCKED/NOT IMPLEMENTED** |
| A-004, A-034, A-075, A-076, A-080 | UUID ownership ledger, schema1 reject unknown schema, native exact Project checks | **PARTIAL UNIT/SOURCE**, native foreign negative, migration/corrupt/recovery and independent verifier **BLOCKED** |
| A-019, A-022 | Operation read/diagnostics contract | **NOT IMPLEMENTED** (S2 public `operation_get` surface); underlying S1 journal partially present |

**No phase-local mandatory native/installed case is claimed PASS.** Until there is a reliable authorized lab, S1 must remain PARTIAL. Continue unblocked review, unit/CI and fixes without weakening scope.

## Targeted follow-up

1. Inspect original UNKNOWN task and authoritative native target identity read-only; do not resend the original mutation or create a replacement on an unproven outcome.
2. Review actual final CI on exact SHA; fix and rerun any remaining errors. Add independent installer/negative/schema and restart verification.
3. On a separately authorized **clean** disposable lab, prove restricted OS Project and SDK rights, installed exact binary + MCP modern/legacy and host_status, real resource and dangerous-effect denials, SQLite kill/restart and safe upgrade/rollback. Record independent verifier evidence, then and only then mark S1 PASS and hand off to S2.

Links: [plan](../process/IMPLEMENTATION_PLAN.md) · [S1 card](../development/phases/01_FOUNDATION.md) · [execution log](../development/EXECUTION_LOG.md) · [UNKNOWN local review](../process/UNKNOWN_MANUAL_RECOVERY.md).
