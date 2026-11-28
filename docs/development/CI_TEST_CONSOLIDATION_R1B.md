# R1 second slice: obsolete mechanism and immutable old-product workflows

Parent `4be53b6030e8b89033af80a467ae00a4c78a646d`; same PR #201 and ordinary development Box `73d2ecae-3932-46e6-b4c6-3feecf7fba15`. Current product remains Schema 7 Scoped Peer v2. No product, fixture, current CI gate, ruleset or historical branch changes.

## Disposition

Remove only current-branch copies of the following six historical wrappers. All scenarios and scripts remain in original Git history; historical workflow refs are left unchanged. This is a route disposition, not a declaration that every generic historical assertion has been retested.

| Workflow | Lines | Reason and preserved evidence |
| --- | ---: | --- |
| v1-s4-b2-scriptlet-native.yml | 184 | Positive global TLS Scriptlet routing and rollback mechanism is not a current feature. S4-B2 source at the original S4 ref below remains. Generic ownership, foreign/no-effect and ACK-loss requirements remain in Scoped ACL/guard tests and installed N2/S25. |
| v1-s4-b2-scriptlet-parity-native.yml | 410 | Historical global authorizer migration, restart parity, legacy TLS clients and frozen S5B CLI. Current product uses restricted Unix socket, not global authorizer mutation; old parity experiment stays version-bound. Historical successful run [44878782540](https://github.com/vanishs/boxweave/actions/runs/44878782540). |
| v1-s5b1-frozen-installed.yml | 68 | Fixed product source a3e02b4750edcabb80427e3311a19614d82b0029, fixed tree 19f4548315c497f611bd83d5537bf6ec5d63e9d1; compares product files before test overlay. Cannot witness current candidate. Original frozen SDK, metadata and retirement routes retained on historical S5B ref. Current N2 SDK, native, recovery/retirement and S25 evidence retained. |
| v1-s5b-integrated-native.yml | 244 | Explicit frozen product source a3e02b4750edcabb80427e3311a19614d82b0029 and binary SHA a7a6a9fb7f32886cad8a891581cdc2eb5a8938200f48ea35b3a8cee3fed3fbf6. Old overlay and assertions retained on S5B ref and [run 44872046410](https://github.com/vanishs/boxweave/actions/runs/44872046410). Current S2/S25/N2 gates not modified. Unique S3 guest/file and verifier negative scenarios are still obligations for per-assertion review, not waived or declared equivalent. |
| v1-s5b-rc10-residual-native.yml | 40 | Exact RC10 product diff gate against eb021d3985c787c517cd7aee3efaf433d38f0393; historical CLI purge and shared-ACL partial PRS path. Actual old [run 44890352977](https://github.com/vanishs/boxweave/actions/runs/44890352977) retained. Current unsupported CLI regression and N4 partial ACL/Forward/restart/no-replay gate remain. |
| v1-s5b-rc10-specialized-recovery.yml | 35 | Exact published RC10 binary f67e769dbfeade39023f655b0a182993bf2459f645d182e52d04bba85093bfa6 and old six leaves. Actual historical [run 44892606554](https://github.com/vanishs/boxweave/actions/runs/44892606554) retained. Current original receipt and no replay obligations remain; retired leaf names are not current contracts. |

For Scriptlet, current source still contains historical helper bodies and tests. No claim that they are all unreachable or removable; R3 must handle source/helper disposition separately. Product boundary explicitly excludes shared ACL and generic authorizer/foreign management; P3 caller disposition records retired root peer and Scriptlet CLI, historical frozen S4/S5 workflows, and current recovery and retirement routes. No changes to H obligations.

## Actual old-ref preservation

Guest Exec `41d8a8d248cf494c2c4600f4835f41bc`, exit 0, fetched all three refs and checked actual workflow blobs. These are independent immutable Git references, not reconstructed archives. The old-ref guards are preserved on their original branches.

- [S4 old ref f1c7986](https://github.com/vanishs/boxweave/tree/f1c7986c070bf0c4d15a9cbaded6b73caccc0e58/.github/workflows): Scriptlet native blob acd1144dcd93e8dddf12523567bdbcf5dd3aa9cc; parity 62b5afc55a5a8c54c284314ce5e2951f13265549.
- [S5B old ref a60a7d3](https://github.com/vanishs/boxweave/tree/a60a7d3429c9c1deb125246d4023e10a4c9c4efb/.github/workflows): Scriptlet parity blob 562352b1c4d7207ac1ebc00a09647ff1aa42525b; frozen installed b1c1986df1c4e7019c0743f07918649f1baa4ee3; integrated a9109d927a7254a688912c76b2e902b58f0fccfe.
- [RC10 closeout ref 3a43b73](https://github.com/vanishs/boxweave/tree/3a43b737bc2a34e5871a0e4b5cff569d4ed02df2/.github/workflows): residual 4ab704c4a8f3127cefbeccdac642f4f6152cde54; specialized 463307f27bd938eb9de792138eac6c62e7f10488.

Historical workflow-path run query was not supported by connector; branch-based run list supplied the links above. No claim that old branches will remain dispatchable after future GitHub default-branch changes or that old run artifacts never expire. Exact source remains recoverable.

## Current behavior and caller evidence

- Full tracked-tree previous R1 scan plus exact six names scan `7bf7500be94f8dfb722cd5c32a6838b4` found only the Scriptlet wrappers' self path filters among non-doc code, no active caller. No old marker usage in active crosswalk or N2/S25 gate.
- `214fac758c6002ee458f4bab3439c1d3` identified current Scoped owner/ACL CAS/preimage/foreign rejection and atomic lineage tests, including `TestPR178ScopedNativeACLETagCASGrantRevokeNeverTouchesCD`, `TestPR178ScopedNativeACLETagCASACKLossAndFaultsNeverRetry`, `TestPR178NativeRootACLPreimageRejectsForeignAcceptAndDrift`, `TestPR178NativeScopedGuardRejectsIncompleteAndForeignInventory`. No claim these replace old TLS authorizer parity behavior.
- Current N2 main installed gate requires `PR178_N3_INSTALLED_LIVE_FLOWS`, active-policy lifecycle, boot policy, actual Peer, and N4 UDP Forward continuity, native partial ACL ACK loss, packet safety, no recovery replay, boot refusal, partial Forward continuity and fenced MCP; current final-commit/pin recovery and exact original retention stay. Prior RC15 [N2 main lane](https://github.com/vanishs/boxweave/actions/runs/46124999390/job/140917531110) succeeded. Existing `cli_contract_test.go` rejects native-peer-purge-plan; `recover_groups_test.go` rejects old leaf interface. Historical evidence does not substitute for new exact-head CI.
- Current S5 history wildcard regression still has three remaining historical wrappers and passed unchanged. Do not erase that constraint to reach a file-count goal.

## Verification and remaining work

Staged six deletions in guest, Exec `eef0234a26532bc23e8fe0943e6d8327` exited 0. Verified all 36 retained workflows byte-identical, scripts/runtime unchanged, diff whitespace check and seven existing focused Python suites (13+8+15+12+4+14+22 = 88 tests) all PASS. Full Go/race and new candidate CI not run in this guest; await normal GitHub checks. No new native acceptance or measured CI runtime savings claimed.

This slice removes 981 workflow lines, 42 -> 36 files; total R1 so far 46 -> 36, 10,241 -> 9,051 workflow lines (-1,190, 11.6%). Remaining 26 historical-only routes, eight current and two optional. Keep BH018/S25 prerequisite, S3, non-Scriptlet S4 peer/forward/installed and three S5 historical routes for generic per-assertion review. In particular old global/shared-ACL positive mechanism and new scoped behavior must not be conflated, and missing current file-race, raw-terminal or fault proof is not a PASS. R1 remains partial. No merge, release or host mutation. Rollback via normal revert of the relevant PR commit.
