# S1 Foundation — partial implementation and evidence, 2026-11-08

> **Historical checkpoint — superseded for CURRENT S1 status.** Use [Latest S1 verification and exit matrix (2026-11-09)](S1_FOUNDATION_LATEST_2026-11-09.md) for tested source `4882adc`, real Incus bootstrap, independently verified installed v1, original UNKNOWN and remaining approved older-Incus upgrade blocker. The earlier lines below intentionally preserve what was and was not verified **at that time**; they are not a claim that today's installed tests are still unrun. S1 overall is still **PARTIAL**, no production release.


**Status: PARTIAL; selected native Incus security and actual binary/HTTP MCP cases now PASS in authorized reused lab, but complete S1 exit/native mutation/installer/independent verification are NOT accepted; not release eligible.** This is an execution artifact for [S1](../development/phases/01_FOUNDATION.md), not a replacement plan or relaxation of [ACCEPTANCE](ACCEPTANCE.md).

## Latest verified S1 subset (2026-11-08 PST, real reused lab)

| Verification | Exact evidence | Verdict |
|---|---|---|
| Latest source Go SDK + CGO race/test/format/real CLI smoke | SHA `591ddaacd882cfe941565a9229ad83b78d16b309`, [CI #42553614450](https://github.com/vanishs/boxweave/actions/runs/42553614450), local `CGO_ENABLED=1 go test -race -count=1 ./...` | **PASS** |
| Disabled-CGO SQLite stub prevention | `CGO_ENABLED=0 go build ./cmd/boxweave` fails specifically on `ERROR_BOXWEAVE_SQLITE_REQUIRES_CGO_ENABLED`; normal CGO=1 build works | **PASS** |
| Native identity and official Incus 7.6 SDK read | Dedicated `bws1lab` only `incus` group, Project `user-1004` `restricted: true`; v1 CLI `doctor` ready=true, exact root Profile and extensions | **PASS** |
| Native foreign-project permission | Exact `incus project show default` and `incus info hello-py-container --project default` denied; prior `hello-py-container` remains running and untouched | **PASS** |
| Actual binary MCP HTTP and negative security | Candidate `/home/bws1lab/s1/boxweave-s1-591ddaac` SHA256 `33384e4cb5aa5a5af6bc6b89383e1c7202cd20da518bec5ac9a315fd31e72b73`. `test_live_mcp_largebody_591ddaac.py` verifies tool list/call, roles, missing/invalid/revoked tokens, unknown arguments, 5MiB body 413, malicious Host/Origin 403 | **PASS** |
| Actual process-crash SQLite UNKNOWN | Child exits without close during DISPATCHING/VERIFYING; subsequent process `Recover` retains original IDs and prevents redispatch, with **synthetic** native ACK only | **PASS (no real Incus mutation)** |
| Unsafe/uninitialized installer refusal | Non-root `install-stage --approve-stage` denied, unstaged `install-status` rejected and binary unchanged | **PASS (negative paths only)** |
| Installer complete, real native mutating UNKNOWN/postcheck, legacy MCP, independent verifier | Not proven; no shared host/old controller replacement attempted | **OPEN — overall S1 PARTIAL** |

Final tested source is newer than the earlier `f605c35` binary. Compiled binaries are **isolated test artifacts only**, not installed production services. Older validated artifact `boxweave-s1` was preserved when its overwrite failed with `File exists`; newest artifact uses a versioned filename. The test listener was stopped and issued test credentials revoked after each suite.

The original outer create task `7e1cbf043d1fd4ea3d9ec0093a04371e` is **still UNKNOWN**, not replayed, despite safe reuse of an *independent* old test Box. The source CI is green; **do not mark S1 PASS, merge or release** until remaining mandatory gates have real independent evidence. Detailed operation and correction trail: [execution log](../development/EXECUTION_LOG.md).

## Exact scope and candidate

- Source branch: `feat/v1-s1-foundation-20261108` from `v1` HEAD `e5c7ae26bdaa24f9243b7dec28dbd34ca561bcb2`. Candidate SHA must be read from GitHub at resumption; the branch received multiple fix commits.
- Official SDK pins: Go 1.26.9, MCP Go SDK v1.8.0, Incus Go SDK v7.6.0, SQLite go-sqlite3 v1.14.52; committed `go.mod` and `go.sum`. No production v1 binary, service or release is known; an isolated test binary was later built with CGO=1 (see native-lab update below).
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
7. Additional source-negative tests guard symlink SQLite adoption and unknown preexisting SQLite modifications; both passed in the final race/unit CI. Native/installed gates were later partially exercised in the reused lab (see update below).

## 2026-11-08 continuation — reused-lab real S1 evidence (supersedes earlier environment-only blocker)

The owner expressly authorized reuse of a working Box or creation of a separate Box. The host had 49 running Boxes and a 4-core/90-vCPU tenant commitment, so no new outer Box was created. Read-only inspection found `bw-racc0470-full-1105` nested Incus **Cowsql no leader** (do not repair); `bwfix-0448-e2e` healthy but no GCC, which revealed a reproducible `CGO_ENABLED=0` SQLite stub build problem. Chosen reusable lab: **`boxweave-test-ser`**, nested Incus 7.6 and existing GCC; its preexisting `hello-py-container` was left unchanged.

- **Pinned source:** `f605c35cf42f6bcd2e8e17875698c1b7b31da462`, test binary `/home/bws1lab/s1/boxweave-s1`, sha256 **`4901742a10e28169c37c6c0a9c1355bd39d88830d0179a264a361948f91a6a1c`**, `CGO_ENABLED=1` + Go 1.26.9 (official download tarball SHA256 `42d158b4d8f7b61ac0a830567c940a86098fb7aac52e467a5ebec03ef5cc2f8d`). No production install.
- **Dedicated identity:** guest-local `bws1lab` UID 1004, only `incus` group, no sudo and no `incus-admin`. Incus `user-1004` is explicitly restricted; actual user can list only that Project. `incus project show default` and `incus info hello-py-container --project default` return **permission denied**. Notably, `incus list --project default --format csv` can return empty success, so it is **not** a valid standalone denial witness.
- **Bug and correction:** original `host_status` required nonexistent Incus API extension `restricted`, reporting false unready. Real 7.6 exposes `projects_restrictions`, and effective `Project.Config["restricted"]` must be checked separately; code fixed at [f605c35](https://github.com/vanishs/boxweave/commit/f605c35cf42f6bcd2e8e17875698c1b7b31da462). Rebuilt native `status` and `doctor` **PASS**: Incus 7.6, restricted Project, correct root disk/default Profile, API extension trio `projects`, `projects_restrictions`, `container_exec_recording`, SQLite schema1, ready=true.
- **Real running test binary:** loopback `127.0.0.1:18961`, HTTP modern 2026-07-28 MCP. Non-secret local test script `/home/boxweave-test-ser/bw-v1-s1-accept-20261108/test_live_mcp.py` passed all nine: missing token 401, invalid bearer 401, tools/list exactly `host_status` HTTP 200, tools/call native ready=true, viewer call 200, unknown tool args denied, foreign Origin 403, foreign Host 403, revoked bearer 401. Test process closed, temporary credentials revoked. This is a **real isolated guest binary with real nested Incus reads**; it does **not** prove installed service/upgrade or mutation.
- **Additional source defect:** go-sqlite3 without CGO may build but returns a stub runtime error. Added compile refusal in `internal/state/require_cgo.go` and CI negative `CGO_ENABLED=0` build check; actually reproduced both failure and corrected fail-closed compilation, `CGO_ENABLED=1` works.
- **True separate-process SQLite crash test**: `internal/state/process_crash_test.go` forks subprocess for states DISPATCHING and VERIFYING, ends without closing SQLite, opens the database in a new process, marks original work UNKNOWN, preserves original native-ID field where set, dedupes key and refuses worker redispatch. Go race focused tests **PASS**. **Native ACK here is synthetic**; it does not establish Incus native postcondition/recovery.

| S1 verification slice | Evidence tier | Status |
|---|---|---|
| Pinned CGO-enabled binary, non-root service user, actual Incus restricted Project and SDK root Profile/extension readback | N, I (read only) | **PASS** in reused lab |
| Actual candidate MCP modern HTTP, admin/viewer, malformed/missing/revoked token, strict Origin/Host, single public tool, unknown arguments | N, I (real candidate HTTP) | **PASS** at test-binary SHA above |
| Foreign Project exact read denied; native group rights; preexisting nested container untouched | N, I (read only) | **PASS** for these negative controls |
| SQLite durable crash in two separate processes, hash idempotency and no blind Worker resubmit | T, I (real process; synthetic native ACK) | **PASS**, not native mutation |
| Native effect lifecycle under uncertain ACK, concurrent auth/native negatives, legacy MCP interoperability, body-size rejection | N, V | **OPEN / PARTIAL** |
| Clean install, daemon reuse/upgrade rollback, migration/backup/operator recovery, independent verifier/release | I, V | **NOT IMPLEMENTED / BLOCKED** |

**Updated stage decision:** `S1 PARTIAL`, not PASS. Do not merge/release from this source-only and limited real-native evidence. The original outer Box create remains UNKNOWN and must not be replayed. Reused lab exists for continuation, but any disruptive change to its preexisting guest/old controller requires separate staging and review.

## Original Box blocker — do not replay (historical state; later authorized reuse below)

A **single** authorized disposable test Box request was sent through the installed *old* BoxWeave API:
- Box name `bw-v1-s1-1108`, backend `dev`; original task `7e1cbf043d1fd4ea3d9ec0093a04371e`, request key `1d6007f5972bd21b01f89eacd42c3160`.
- Original create receipt `REST_GATE_PENDING` and then `UPSTREAM_CREATE_OUTCOME_UNKNOWN`, upstream acknowledged=false. One independent `get_container` returned NOT_FOUND; `debug_container` reported missing host/guest state in daemon 0.128.0. A later read of the **same original task** again reported UNKNOWN.
- **UNKNOWN remains**: a transient negative inventory is not authoritative evidence of no dispatch. No second create, auto-delete, new substitute Box, foreign ownership adoption or host-wide Incus change was performed.
- **At that earlier checkpoint:** no eligible native test environment; no existing Box had been reused yet. **Subsequently superseded for read-only native and MCP evidence** by the owner's explicit authorization and the reused `boxweave-test-ser` laboratory below. Original UNKNOWN remains unchanged.

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
