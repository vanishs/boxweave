# BoxWeave v1 — scoped `uninstall.sh` implementation / isolated acceptance

**Date:** 2026-11-19 PT. **Status: SCRIPT + ISOLATED SIMULATION + EXACT-SOURCE QUALITY/BUILD PASS; REAL PHYSICAL HOST APPLY NOT_TESTED.**

**Exact tested source HEAD (before docs-only handoff):** `057db89fba22309000b25d309164085fb36ff251`. One sole [Draft PR #153](https://github.com/vanishs/boxweave/pull/153) on `feat/v1-s4-runtime-integration-20261116`; not merged, not published as a release/RC, never activated on the shared physical L0.

## Source delivered

- `uninstall.sh`: `sh` entrypoint; **no arguments = help** (no filesystem/service/Incus mutation).
- `scripts/uninstall_core.py`: Python standard-library-only, fixed installer-root paths with three explicit `--scope service|controller|full`. Exact read-only `--plan` produces a versioned JSON summary with `ready`, precise `blockers`, `host_machine_id_sha256`, `review_sha256`, original DB/Native inventory and proposed actions. An `--apply` must reproduce the exact plan, both digests, Project and complete Box/native allowances under host-local root.
- `scripts/test_uninstall_core.py`: **17** offline deterministic tests exercising the real Python module/CLI with a fully isolated, synthetic filesystem/SQLite/systemd/Incus fixture.
- `make test-uninstall`; integrated into `make verify` and the required **full** GitHub `quality-gate`. Docs-only CI correctly skips repeated heavyweight tests.
- [Operator usage, constraints and scope table](../development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md); [original physical RC.7→RC8 cleanup lessons](../development/V1_HOST_DECOMMISSION_REINSTALL_AND_UNINSTALL_DESIGN_2026-11-19.md) now explicitly mark the earlier *unimplemented* design as a historical snapshot.

## Test evidence

| Gate | Result | Scope / exact proof |
| --- | --- | --- |
| No-argument help | **PASS** | `sh ./uninstall.sh` printed help, performed no host action |
| Plan immutability | **PASS (synthetic)** | Deterministic host+review SHA256; inspect reported expected state without mutating test fixture |
| Approval failure | **PASS** | Actual `main(--apply)` rejects wrong/reused digest, modified service state and nonroot identity **before** writing its audit or deleting paths |
| `service` removal | **PASS (synthetic)** | Stops/disables exact unit, verifies inode/SHA/owner and preserves SQLite/credentials/native Boxes |
| `controller` removal | **PASS (synthetic)** | Removes exact authorized app state only after terminal original/empty-Box checks; refuses unknown/unexpected files and installer backups |
| `full` exact inventory | **PASS (synthetic)** | Complete explicit native immutable UUID allowlist and exact DB ownership; all Boxes must be Stopped; only explicit `local:` Incus remote and no force |
| Native preparation | **PASS (synthetic)** | Active journal blocks smaller scopes; `full` uses exactly its current `ReviewSHA256`, then executes rollback → archive **before** removing executable |
| Negative scopes | **PASS** | UNKNOWN original native claim, foreign manifest, symlink, altered unit, nonempty Root service, active global Scriptlet, foreign/modified root journals and native running Box all refused |
| Durable audit + staged scope | **PASS (synthetic)** | Original STARTED record stops blind replay; COMPLETE record preserves audit history and allows a **freshly reviewed**, larger subsequent scope |
| Python source / shell parse | **PASS** | `python3 -m py_compile`, `sh -n uninstall.sh` and actual no-args entrypoint |
| Real uninstalled Box fail-closed check | **PASS** | Running `sh uninstall.sh --plan --scope full --project user-1001` **inside an uninstalled disposable Box** exited nonzero and created **no** audit; never pretended a synthetic fixture was a production install |
| Source CI | **PASS** | [full required formatting, Go vet, Python uninstall tests and Go Race #44639986684](https://github.com/vanishs/boxweave/actions/runs/44639986684), exact code source `057db89f` |
| Cross-architecture build | **PASS** | [Linux amd64+arm64 and manifest candidate build #44639986708](https://github.com/vanishs/boxweave/actions/runs/44639986708), exact source `057db89f` (not RC published) |

**Owned test Box cleanup:** L0 one-use Box `c0221b86-d03d-4a8b-afd1-fd505fef8cc7`; its `box_stop` original `b8de069635c0eacfbd9584ddb91c9b6b` and `box_delete` original `ba78ebbfbb0c8e8f3cd79bbd219cc21b` both reached **SUCCEEDED**; subsequent authenticated `box_list` returned **`boxes: []`**. No actual Host `uninstall.sh --apply`, native ACL rollback, global Incus resource change, physical controller service stop, OAuth/ChatGPT connector removal or shared storage destruction was performed.

## Remaining acceptance / limitations

1. **NOT_TESTED:** `--apply` against any real installed host or independently staged installed-controller/native Project. The positive removals above are *synthetic fixtures*; they prove program control flow but not real systemd/Incus kernel, ACL poststate, Btrfs/dir/ZFS behavior, remaining `used_by` references or managed rollback recovery. The historical real RC.7→RC8 reset used different manually executed, operator-approved actions and is **not** evidence for this script's actual apply.
2. **Manual staged recovery:** an interrupted STARTED audit is durable, protects the original approval identity and refuses automatic second destructive apply. It does **not** perform a blind automated resume; inspect native/host poststate, preserve original IDs and plan separate operator reconciliation before authorizing another action. Interrupted Native rollback and exact in-flight service state change require independent destructive fault acceptance.
3. **Retained on purpose:** root-only journal/closed archives, separately owned native Root service or global Scriptlet (they block if active), shared Incus daemon/pool/bridge, external image cache, OS service user/group, Cloudflare DNS/Tunnel, ChatGPT registrations and other Projects. `full` means **maximal reviewed BoxWeave-owned** cleanup, never a generic machine reset.
4. **Plan purity with active WAL:** SQLite opens `mode=ro` with `query_only=ON`; a standalone physical-host test should verify that no observable WAL/`-shm` side effects occur while the service actively writes. No plan result should be presented as an exhaustive native effect inventory without independent Incus/Root verification.

**Do not run `--apply` on currently connected L0 based only on this report.** It is known to have protected historical Controller/Root/ChatGPT identity and real workloads. Obtain a distinct fresh disposable installed Host/VM and explicit human approval for a destructive real-host acceptance before promoting this script into the published installer contract.
