# BoxWeave v1 — Stage 2 installed/native reacceptance (scope-free authorization and three-SDK refactor)

**Status: PARTIAL / new-source native execution in progress. Not S4 final signoff.**
**Source baseline:** `2c7181aaea61f899d8bf6faebe0b6526cf61c03c` (2026-11-18 PT).
**Later focused test-only fix:** `be599aece10f908037edc5599167609ec1288d36`; changes one S2 native SDK probe's no-role PASS marker, not controller/SDK/Incus production Go source. Native results from `2c7181a` are **source-specific** and are not silently labeled as the later commit's exact-head signoff.
**Only development location:** [Draft PR #153](https://github.com/vanishs/boxweave/pull/153); never merged/deployed/published.

## Latest independent GH Actions native matrix (2026-11-19 UTC; newer than the snapshot below)

**Exact source and CI SHA:** `917c02965f1f4d2d92aecf3498fcbfdac540d37d`, still Draft PR #153. Native jobs use fresh one-use GitHub-hosted Ubuntu 24.04 / Incus 7.6; no shared BoxWeave L0 host is modified.

- **Quality gate:** [run #44480369029](https://github.com/vanishs/boxweave/actions/runs/44480369029) **PASS**; exact-head build manifest [#44480369078](https://github.com/vanishs/boxweave/actions/runs/44480369078) quality, amd64 and arm64 build jobs **PASS** (pre-release publish job skipped).
- **S4-B2 Controller→Root revoke:** [run #44480369027](https://github.com/vanishs/boxweave/actions/runs/44480369027), all three native jobs **PASS**: `normal`, `lost_ack`, `after_acl_fault`. The independent job logs contain `S4_B2_INSTALLED_ROOT_ACK_LOST_COMPLETED_JOURNAL_READONLY_RECOVERED_NO_REPLAY=PASS`, `S4_B2_INSTALLED_ROOT_REAL_ACL_PUT_THEN_UNKNOWN_JOURNAL_ALL_NICS_STOPPED=PASS`, and `S4_B2_INSTALLED_FOREIGN_PROJECT_ESTABLISHED_TCP_UDP_DUALSTACK_SURVIVED_REVOKE=PASS`. Host-root service restart, restricted socket and original-journal reuse refusal were checked. **B2 stage-2 selected native matrix = 3/3 PASS**.
- **S4-B3 Root grant/ETag:** [run #44480369108](https://github.com/vanishs/boxweave/actions/runs/44480369108): **`inside_etag_race` PASS**, real stop barrier with contender PUT rejected, UNKNOWN NIC isolation and Incus/PRS service-cycle evidence verified. **`installed_normal` IN_PROGRESS at this checkpoint**, not signed.
- **S4-A full installed S2/Operation/OAuth/native end-to-end:** [run #44479992641](https://github.com/vanishs/boxweave/actions/runs/44479992641) on source SHA `9c2a11e8484842bc3995a1c4e2d0bcdd1da843bc` **IN_PROGRESS at this checkpoint**. The earlier failed S4-A probe had a stale GDB compiled-source boundary anchor referring to the deleted `operationCredentialID(req)` function, not a demonstrated native ACL or Box effect regression. Corrected `scripts/s2_p1_operation_list_boundary.sh` to match `principal.TokenID` in `9c2a11e`; the S4-A workflow is now exercising that exact-source correction. Until it actually finishes, S4-A is **NOT PASS**. SHA differs only by the later B2/B3 CI preflight changes.
- **B2/B3 preflight on same SHA:** added source-exact Go tests of strict `RequestPrincipal` and full ACL comparison to B2/B3 workflows *before* costly native provisioning, commit `917c029`.
- **S3 native File Tools:** previously established on source `2c7181a` via runs [#44474489120](https://github.com/vanishs/boxweave/actions/runs/44474489120), [#44474489157](https://github.com/vanishs/boxweave/actions/runs/44474489157), both **PASS**. These earlier source-specific results cannot be silently relabeled exact-HEAD PASS. Later code changes from `2c7181a` through `917c029` only affect probe/workflow/doc source; re-acceptance of every S3 fixture remains an independently chosen Stage-2/Stage-3 gate.
- **Safety/limitations:** old signoffs, the exact-source S4-A job still running, the B3 normal branch still running, and actual ChatGPT scope-free OAuth connection/refresh/revoke **do not** constitute a final 100% native/E2E signoff. Other B3 matrix variants are not claimed reexecuted on `917c029`; preserve unexecuted cases as `NOT_TESTED`. No automatic v3→v4 migration or host reset is permitted.

**Current verdict: `STAGE2_NATIVE_PARTIAL`**, with **B2 3/3 PASS** at `917c029` and confirmed B3 ETag competition PASS; awaiting the distinct final outcomes of B3 normal and S4-A.

## Real fresh Incus 7.6 evidence (GitHub-hosted disposable Ubuntu 24.04)

| Gate | Source SHA | Workflow run | Observation |
| --- | --- | --- | --- |
| Required quality: full Go Race, vet, format/YAML, modules, builds | `2c7181a` | [44474489026](https://github.com/vanishs/boxweave/actions/runs/44474489026) | **PASS**, including full `go test -race -count=1 ./...` |
| Same required quality on test marker fix | `be599aec` | [44475355743](https://github.com/vanishs/boxweave/actions/runs/44475355743) | **PASS** |
| S3 native SFTP hostile-file / 12 simultaneous writers | `2c7181a` | [44474489120](https://github.com/vanishs/boxweave/actions/runs/44474489120) | **PASS**: single-winner create, symlink/hardlink/FIFO refused, bounded SFTP list/read, disposable Guest cleanup |
| S3 installed role-free MCP File Tools and Guest Git fail→fix→PASS | `2c7181a` | [44474489157](https://github.com/vanishs/boxweave/actions/runs/44474489157) | **PASS**: 17 SDK Tools, independent Actor file-create, no unsafe conditional overwrite, restart/Operation recovery, exact native Git witness and Box cleanup |
| B2 installed Controller→Root native revoke, normal | `2c7181a` | [44474489103](https://github.com/vanishs/boxweave/actions/runs/44474489103) | **PASS**: operator approval, nonroot original, actual IPv4+IPv6 nft isolated per NIC, foreign Project TCP/UDP sessions preserved, systemd restart/upgrade/rollback |
| B2 installed `lost_ack`, `after_acl_fault` | `2c7181a` | [44474489103](https://github.com/vanishs/boxweave/actions/runs/44474489103) | **RUNNING at snapshot**; no PASS claim |
| B3 actual stop-barrier/ETag race with UNKNOWN safe isolation | `2c7181a` | [44474489158](https://github.com/vanishs/boxweave/actions/runs/44474489158) | **PASS** `inside_etag_race`: live contender PUT refused, affected NICs remain isolated |
| B3 installed normal grant | `2c7181a` | [44474489158](https://github.com/vanishs/boxweave/actions/runs/44474489158) | **RUNNING at snapshot**; no PASS claim |
| S4-A installed raw native + S2/BH-018/real OAuth probe | `2c7181a` | [44474489076](https://github.com/vanishs/boxweave/actions/runs/44474489076) | **FAILED** in baseline S2 assertion after successful native Box create/exec/delete, no full S4-A PASS |
| S4-A retry after removing obsolete role-era PASS label | `be599aec` | [44475355616](https://github.com/vanishs/boxweave/actions/runs/44475355616) | **RUNNING at snapshot**; no PASS claim |

These flows run only with `GITHUB_ACTIONS=true`, nonroot controller and bounded fresh one-use Incus Projects/DBs. No live BoxWeave L0–L5 daemon, shared host Scriptlet, ACL, guest or inherited UNKNOWN ledger is changed by this effort. A disposable L0 Box was used only to inspect the repository / public Actions metadata.

## Genuine fixes found by native acceptance

1. **Hidden standalone Go API debt** (not covered by `go test ./...` because most files use `//go:build ignore`): old `Token.Role`, `IssueToken(ctx,"viewer"/"admin",ttl)`, old `GetScopedOperation(...,"admin",id)` caused compilation or execution failure. Converted test fixtures to verified Actor+Token and current APIs in commits `f7e828a`, `7d58c4c`. Added **precompile of all 28 installed native standalone Go probe binaries before expensive native Incus setup** in `v1-s4-a-installed.yml` (commit `31ebd01`), confirmed PASS on subsequent S4-A CI. This cannot be worked around by restoring the removed role model.
2. **Missing workflow invalidation** for `internal/install/native_peer_acl_compare.go` and newly changed native/MCP wrappers meant a new security refactor could skip native reacceptance. Explicit S3/B2/B3/S4-A native CI workflows now track these dependency paths and can run on the sole v1 S4 feature branch.
3. **Required quality gate** failed on two trailing spaces in the original three-SDK audit document, not on Go or native behavior. Corrected in `62f88b3`; quality then passed including full Race.
4. **S4-A baseline probe mismatched the expected no-role PASS marker**: the official MCP SDK client still emitted `...HOSTILE_META_PEER_AND_SCHEMA...`, whereas `accept_s2_native.sh` required `...HOSTILE_META_AND_SCHEMA...`. The native Box and several original Operation tests already passed; changed only the test evidence label in `be599aec`. New S4-A run is required; **previous failed workflows stay failed** as historical evidence.

## Gates not yet earned

- **No Stage 2 complete PASS:** B2 fault branches, B3 installed normal and final S4-A end-to-end must finish with independent exact-source evidence. Do not treat an in-progress runner as passed or retry UNKNOWN original native mutations with a new key.
- Real **ChatGPT** no-Scope OAuth connection/refresh/revocation has **not** been observed. Installed OAuth SDK probes do not substitute for a ChatGPT client. Independent final signoff and same-SHA complete native/CI acceptance belong to Stage 3.
- Earlier `S4_B2_PASS`/`S4_B3_PASS` refer to the exact old source and are not automatically transferred across role, Token-ID or Root ACL code changes. Do not rerun unimplemented B4–B6 as if built.
- Old Schema v3 installation cannot be reused. For this role-free v4 source, create fresh database, independently revoke/reissue tokens, reapprove OAuth clients/Grants and reconnect MCP clients. Preserve unresolved original native Operation/UNKNOWN evidence, and **never auto-delete** Incus Projects/Boxes or host ACLs during reset. [Rebuild policy](../process/VERSIONING.md).

**Current label: `STAGE2_NATIVE_PARTIAL`.** Follow the listed GitHub run IDs and fix only failures substantiated by actual logs; update this snapshot after each final run conclusion. No successful S4-A/S4-B2/S4-B3 final signoff is asserted by this document.
