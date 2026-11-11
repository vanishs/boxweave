# S2 P1 — pinned previous SDK, two independent controllers, and bound-Create lock

## Exact fully accepted checkpoint

- **Source**: `0a372fed424112c5c54d4379e6d83ff2233e9d1f`; compiled BoxWeave candidate SHA256 `0159616c6adc1a69cfbb6dcd3f08fa45dc4318ba72a6d9900189c2a634b2a0b2`.
- **Real native + official SDK**: [Incus 7.6 disposable CI](https://github.com/vanishs/boxweave/actions/runs/42995734792/job/130439257594) **SUCCESS**; [required quality](https://github.com/vanishs/boxweave/actions/runs/42995742733/job/130439283275) **SUCCESS**; [S1 Go/race](https://github.com/vanishs/boxweave/actions/runs/42995742744/job/130439282652) **SUCCESS**.
- The native job independently logged `S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_LEGACY_PROTOCOL_13_TOOLS_NO_NATIVE_EFFECT=PASS`, `S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE=PASS`, `S2_P1_TWO_RESTRICTED_OS_USERS_DENY_FOREIGN_INCUS_PROJECT=PASS`, `S2_P1_TWO_CONTROLLER_CROSS_TOKEN_AUTH_DENIED=PASS`, `S2_P1_SECOND_CONTROLLER_INDEPENDENT_ADMIN_SAME_NAME_DISTINCT_UUID=PASS`, `S2_P1_TWO_CONTROLLER_INDEPENDENT_NATIVE_AND_MCP_ISOLATION=PASS` and `S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS`.

## Proven details and limits

| Subgate | Exact evidence | Disposition |
|---|---|---|
| Older official SDK wire compatibility | A **separately built** `github.com/modelcontextprotocol/go-sdk@v1.6.1` client from its own `GOWORK=off` temporary module negotiated the current HTTP MCP server and discovered 13/13 advertised tools. Real read requests and 10 unsupported/foreign-UUID negative Tool calls preserved an empty initial native inventory | **real installed PASS for v1.6.1**, not an exhaustive historical SDK release matrix |
| Older SDK genuine operations | The same client created its own unique stopped Ubuntu 24.04 Box, independently confirmed UUID and `Stopped`; updated CPU, started, checked `Running`, issued native guest Exec, decoded exact base64 stdout `legacy-sdk-ok` with successful guest exit, read history, stopped and deleted its owned UUID, verified empty Box list | **real installed PASS**; does not authorize arbitrary third-party SDK behavior |
| Distinct service/process and Project | Two live nonroot BoxWeave processes ran with **different real OS user identities**, per-UID restricted Incus Projects, independent 0700 SQLite directories and separate admin tokens/listeners. Both created **the same native name** with **different native UUIDs**; each saw exactly its own instance and refused the other's `box_get` and `box_delete` | **real installed PASS**; not proof of cross-host cluster replication or failure recovery |
| Token and native independent witness | Wrong-controller bearer on either local authenticated HTTP endpoint returned 401. Each restricted native user was unable to inspect the other's Project. Independent administrator matched both `volatile.uuid` values simultaneously, then witnessed every test Box removed | **real installed PASS** |
| Incomplete Create / different-key Delete | 24 concurrent SQLite admissions alternate between two actor tokens and different Delete keys while original Create has an exact bound UUID but remains in `VERIFYING`. All returned `ErrResourceLocked`; same original replay stayed unchanged, and a new Delete was admissible only after the original `SUCCEEDED` | **Go/race PASS**, native MCP Create/Delete interleaving at that tiny instant still **UNVERIFIED** |

## Failed-test history, corrections, exit discipline

- The first independent-client CI build copied a `//go:build ignore` test file unchanged into its temporary Go module; `go mod tidy` found no buildable package. The runner now removes only the test-only build tag when constructing this standalone module. A later full original native run at [source 3c3d1c8](https://github.com/vanishs/boxweave/actions/runs/42994670535/job/130435654108) was already SUCCESS before the independent second-controller matrix.
- A newly added concurrency source test and later second-controller probe initially failed **gofmt**, not product behavior. The existing candidate-only source snapshot action produced exact formatted bytes; current `0a372fed424112c5c54d4379e6d83ff2233e9d1f` passed whole-repo quality, Go race, format and the full native BlackBox at the same source.
- This does **not** close five still-unproven deliberately delayed non-`operation_get` token races; **real** mid-VERIFYING bound-UUID Create/Delete interleaving; positive Guest Ready, malformed Incus metadata proxy, arbitrary stalled I/O, custom root-pool selection contract, online public WAN control, physical host power loss, or independent final TC/A-case exit review. These remain PARTIAL/UNVERIFIED/BLOCKED as appropriate in [open-gates register](S2_OPEN_GATES_AND_ENVIRONMENT_BLOCKERS_2026-11-10.md).
- Only newly allocated disposable CI identities, Incus Projects, test native Instances and SQLite ledgers were affected. No physical/shared/production Incus, historical UNKNOWN, protected `v1`, PR merge or release was changed.

**Overall S2 PARTIAL; Draft PR #130 remains open and unmerged.**
