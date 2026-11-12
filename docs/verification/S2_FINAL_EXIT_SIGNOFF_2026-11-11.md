# S2 formally scoped phase-exit audit — independently accepted for protected v1 merge

**Decision date:** 2026-11-11 America/Los_Angeles (2026-11-12 UTC CI).

**Owner authorization:** In this conversation, explicitly authorized two P1 native acceptance, corrections, formal S2 P0 signoff and merge to `v1` **only once no outstanding mandatory scoped acceptance failure exists**. This does not authorize shared physical/native Incus installation, modifying historical operations, S2.5 HOST_GO, S4 feature completion or S5 release.

## Source, identity, and independent witnesses

- **Known fully accepted product+test source:** `f7828104c725480c016613073db591bf95c5bd9a`, branch `feat/v1-s2-operations-20261110`, base protected `v1` (S1 accepted).
- **Unmodified installed candidate SHA256:** `f5aee125f1e10029b07cc4f4ebe74e14046026c91be3a990ad8db70ee27e7e57` measured in [one-use genuine Incus 7.6/official MCP SDK CI](https://github.com/vanishs/boxweave/actions/runs/43131566410/job/130896688166).
- **Independent rebuild:** existing authorized isolated `bwracc-1106-v2`, Go 1.26.9 + CGO, exact source `f782810` and `-trimpath -ldflags '-X main.version=1.0.24-s2-native-ci -X main.commit=f7828104c725480c016613073db591bf95c5bd9a'`, SHA256 exact byte-for-byte match. Full `go vet ./...`, `go test -race -count=1 ./...` PASS; both P3 negative verifier suites **7+10 PASS**, no worktree pollution.
- **CI 5/5:** quality-gate (both independent runners), build-test, source-format-snapshot, and complete `s2-native-blackbox` all SUCCESS on the same exact commit.
- **Independently checked executable and exit:** original `S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS`; P3 independently reads exact native original candidate source and binary hash, 19 scoped acceptance groups and rejects spoofed/missing evidence. `S2_P3_ALL_NINETEEN_GROUPED_TC_A_EVIDENCE_SCOPED_REVIEW=PASS`, `S2_P3_SCOPED_NATIVE_GROUPS_VERIFIED=18`, `S2_P3_SUPPORTED_SCOPE_WITH_EXCEPTIONS=ACCEPTED_NOT_FINAL_RELEASE`.

## Two newly finished P1 gates (genuine native, not unit-only)

| Gate | Exact native marker | Evidence and meaning |
|---|---|---|
| P1 Guest Ready positive | `S2_P1_NATIVE_GUEST_ORIGIN_READY_TRUE_ACTUAL_SOCKET_AND_MCP=PASS` | Real owned Guest used official authenticated MCP Exec to PATCH `/dev/incus/sock` to Incus Ready; owned UUID public `box_get` returned `guest_ready=true`. Mere native Running and untrusted user config never assert Ready |
| P1 Guest Ready lifecycle negative | `S2_P1_NATIVE_GUEST_READY_RESTART_CLEARS_STALE_TRUE=PASS` | Stop gives `false`, restart before guest re-declaration gives `null`; public running state remains compatible |
| P1 WAN emulation — positive/TLS | `S2_NATIVE_SIMULATED_WAN_NAMESPACE_ROUTE_VERIFIED_TLS=PASS`, `S2_P1_SIMULATED_WAN_HTTPS_POSITIVE_TLS_BEFORE=PASS` | Own distinct veth/routed namespace `198.18.0.2`, verified private test CA and DNS SAN; host separately verified server and Guest HTTPS online success |
| P1 WAN emulation — off and recovery | `S2_P1_SIMULATED_WAN_HTTPS_OFFLINE_DENIED=PASS`, `S2_P1_SIMULATED_WAN_HTTPS_ON_OFF_ON_GUEST=PASS`, `S2_NATIVE_SIMULATED_WAN_SERVER_INDEPENDENT_ON_OFF_ON_TWO_HITS=PASS` | Same Guest and HTTPS target: internet ON permits, OFF denies, ON restores. Server independently fsynced **exactly two** successful GETs (zero during OFF) and namespace/rules cleaned up |

## S2 signed support limits and deferred work

1. **18/18 supported grouped S2 evidence scopes accepted**: 10 TC-001–010 + 8 S2 A-case groups, against named **authentic** native witness markers; the original [full-scope historical crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md) still retains PARTIAL annotations for aspirational, unsupported and future cross-phase cases.
2. **1/19 grouped A-case `A-043–047` remains DEFERRED TO S4** for dependent forward/peer-aware Box Delete. S2 verifies exact no-dependent UUID-bound Delete, not complete network topology support.
3. **Actual external public Internet HTTPS is still `UNVERIFIED_ENVIRONMENT_EXCEPTION`**. Independent namespace HTTPS is specifically labelled **SIMULATED WAN PASS**, never genuine public WAN positive. Original restrictive runner could not establish a public Guest HTTPS online baseline; do not claim it passed.
4. **Power-outage surrogate** approved by owner for S2: actual process SIGKILL immediately before/after SQLite WAL fsync, original native effect and same-Operation UNKNOWN/no replay, hard ext4 ENOSPC. **No physical power cut** test claim. Remote disk/controller failure remains outside accepted S2 scope.
5. **Contract limits:** Guest Ready is guest-declared Incus readiness, not full application health; stopped false, unprobed running null. Incus 7.6's detached recorded Exec Cancel unsupported, no client arbitrary root pool privilege, no broadened auth model or new public tools.
6. **No production use:** The accepted exact binary proves a lab candidate, not a deployed physical host or S2.5 LAB_PASS/HOST_GO. S2.5 BRIDGEHEAD must install and test an exact S2 accepted binary independently in a safe L0→L1→L2 nested chain and verify its own BH-001–017 gates before any future specific-host approval.
7. **Original immutable unrelated UNKNOWN tasks**: `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`; no retry/adoption/unlock/delete.

## Formal phase outcome and merger contract

**S2 = PASS for the expressly approved development support scope**, after independent 5/5 real-native suite and per-case negative proof. The four *physical/higher-stage* exceptions above are explicit contractual boundaries, **not** fake PASS values. [Exact source audited evidence](S2_SCOPED_EXIT_AND_FAULT_SURROGATE_2026-11-11.md) and [all TC/A scope marker policy](../../scripts/s2_p3_scoped_acceptance.json) remain machine-checkable.

The signed documentation commit changes the source revision after `f782810`. **No merge solely on this historical PASS.** The final PR head MUST independently pass all five checks including new exact-head Incus 7.6/native P3; enforce protected `v1` rules, no bypass. Then mark PR #130 ready, merge via expected-head SHA, and read GitHub's merged PR and `v1` HEAD as proof. Keep the merge's generated SHA in the PR as the authoritative merge record, not a fictitious pre-merge document hash.

**Post-merge only:** S2.5 BRIDGEHEAD `PLANNED→READY_FOR_ISOLATED_LAB`; never infer `LAB_PASS`, physical `HOST_GO` or S5 release.
