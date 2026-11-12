# S2 2026-11-11 — fastlab acceleration, native acceptance and bounded P1 experiment

## Evidence tiers and exact provenance

| Gate | Exact scope | Result |
|---|---|---|
| Native broad 13-Tool/Incus 7.6 | [Full disposable run](https://github.com/vanishs/boxweave/actions/runs/43013736064/job/130499504254), source `6495d893040b79179a4f33b449a10ffd860bc6d8`; binary `7f18b7d349f7b89520368855623407fe6f35773af65fc4b592febfcbe58d45a0` | **PASS** |
| Real `operation_list` in-flight credentials | An unchanged installed MCP handler paused after its SQLite list query and before private response; separate credential revoked or naturally expired; original not disclosed | **PASS** |
| Real P2 negative native | Real native Guest Exec accepted and performed once; wrong-type final return produced original UNKNOWN/no replay after SQLite restart | **PASS** |
| Real stdout and timeout | Native stdout HTTP 200 held while client cancelled; new SDK session recovered identical original and bytes. Continuous guest stdout with explicit bounded timeout preserved true outcome | **PASS** |
| Selected alternative pool | New 12GiB mounted ext4 project quota; preconfigured Project default; omitted disk size became 10GiB. Independent administrator checked live expanded root pool, size and exact UUID; full teardown/restoration | **PASS** |
| Independent reproducible binary | Separate authorized Box rebuilt `6495d893040b79179a4f33b449a10ffd860bc6d8` from exact source and CI Go 1.26.9/CGO flags; SHA256 exactly matched `7f18b7d349f7b89520368855623407fe6f35773af65fc4b592febfcbe58d45a0` | **PASS** |
| Local full Go/race/vet | Existing `bwracc-1106-v2`, exact source `22f91d2`, Go1.26.9, CGO: `go vet ./...`, `go test -race -count=1 ./...` across 5 packages | **PASS (source-only)** |
| Genuine native Create UUID bound and concurrent live MCP Delete while still VERIFYING | Test-only GDB non-stop/MI approach reached source line; actual post-bind SQLite snapshot was not proven | **UNVERIFIED (tester)** |
| Physical power loss / public WAN positive | No authorized controlled physical lab or independent confirmed external Internet positive control | **BLOCKED/UNVERIFIED** |

**Important:** the separate local Box is only for efficient source/race and reproducibility; it is not the official restricted Incus 7.6 test instance. The accepted real native verifier used a new disposable GH-hosted VM and ran actual source-bound MCP/Incus lifecycle.

## GDB exact overlap attempt — three CI failures are not three product failures

The first one-use CI stopped at incompatible non-stop `gdb --batch` behavior. The second stopped at the GDB MI event format not matching a CLI breakpoint text check. The third **confirmed** the real worker.go:79 source breakpoint, but the independent SQLite probe did not see `VERIFYING` with its exact bound UUID. Debugger source line mapping is not a proven post-commit instruction boundary, so neither a safety violation nor a passed interleaving was observed. The 24-way DB/race bound-UUID lock and genuine native Create final-receipt failure/UNKNOWN and different-key Delete refusal are distinct, accepted tests.

The GDB test script and Go SDK probe remain CI-only, gated behind `BW_S2_RUN_EXPERIMENTAL_BOUND_OVERLAP=true`; default isolated native acceptance explicitly emits `UNVERIFIED_TESTER_BOUNDARY`, **never PASS**.

## New original UNKNOWN from external legacy create

A one-off dedicated fast-lab request returned `UPSTREAM_CREATE_OUTCOME_UNKNOWN` for Box `bw-v1-s2-fastlab-1111`, task ID `192a6601c54078f8eaf1437a6a3dc13b` on the separate connected legacy BoxWeave 0.128.0/Incus7.5.1 backend. Read-only status remained UNKNOWN; Box GET was NOT_FOUND and debug said missing. Neither establishes a durable no-effect proof. **Never retry, overwrite, adopt, unlock or automatically delete this original**. It is NOT one of the two preexisting historical UNKNOWNs. The fastlab was therefore bootstrapped inside an already existing authorized test Box instead.

## Gate decisions

S2 remains **PARTIAL** until remaining TC/A items and environmental blockers have honest closure. The new 12-way real overlapping MCP Delete proof stays UNVERIFIED, not promoted from Go/race or after-failure UNKNOWN. Guest Ready `true` is not established from `Running`; direct public pool selection remains unsupported and rejected. S4-owned dependent Delete is not silently accepted into S2. Draft PR #130 remains unmerged, protected `v1` unchanged.

[Open-gate register](S2_OPEN_GATES_AND_ENVIRONMENT_BLOCKERS_2026-11-10.md) · [Crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md) · [Execution log](../development/EXECUTION_LOG.md)
