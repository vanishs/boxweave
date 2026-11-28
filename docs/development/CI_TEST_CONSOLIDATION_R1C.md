# R1 third slice: historical diagnostics and retired peer execution

Parent `1b60917fc27947c5c664dbae7a98bd226cc170f2`, PR #201. The previous slice's five exact-head GitHub runs all succeeded: PR quality [46128561691](https://github.com/vanishs/boxweave/actions/runs/46128561691), S1 [46128561706](https://github.com/vanishs/boxweave/actions/runs/46128561706), candidate [46128561702](https://github.com/vanishs/boxweave/actions/runs/46128561702), push quality 46128556611 and candidate 46128556590. This is not new-head acceptance.

## Disposition of six historical routes

| Removed workflow | Lines | Original assertion scope and current disposition |
| --- | ---: | --- |
| v1-s4-b2-ndp-native.yml | 169 | Raw shared-ACL NDP/MLD differential, explicitly diagnostic-only; IPv6 outcomes reported as REACHABLE/UNREACHABLE, not asserted positive. Old IPv4 control and exact disposable cleanup retained in original ref. Current N3 directional traffic and IPv6 live-flow checks, plus S25 separately routed IPv6 and gateway checks, remain mandatory. Do not equate this with external IPv6 acceptance. |
| v1-s4-b2-static-address-native.yml | 109 | Standalone provider experiment: duplicate simultaneous start accepted or refused, guest dual-stack assignment PASS or PARTIAL, address reuse observed or PARTIAL. It never established exclusive allocation. Current contract holds retired identities, and exact same-name reincarnation verifies distinct UUID/ACL/MAC/IPv4/IPv6, old direction replay and independent packet/Forward preservation in `pr178_n4_reincarnation.py`. Current ledger duplicate-address/retired-hold tests retained. No claim of native reuse or DHCP parity. |
| v1-s4-b2-fanout-fault-native.yml | 164 | Old shared-ACL fanout differential after one missing host link; explicitly DIAGNOSTIC_ONLY_NOT_PRODUCT_PASS, ACL reload failure optional and post-fault connectivity observed either way. Current per-Box ACL mechanism removes shared fanout. Current modeled missing-live-veth rejection, unaffected owner policy and actual N4 partial native ACK/no-replay, unrelated packet/Forward continuity remain. Exact real missing-link plus current mutation composition is not claimed verified; keep as investigation obligation. |
| v1-s4-b2-grant-fault-native.yml | 147 | Old shared-ACL failure experiment (pre-grant denied, provider fault and API state checked; actual fail-open observation could end the experiment). Explicitly DIAGNOSTIC_ONLY_NOT_PRODUCT_ACCEPTANCE. Current missing-veth guard, changed-preimage/CAS no-write and actual installed N4 partial-effect denial and independent positive unrelated direction retained. No restore/repeat of shared-ACL writes to satisfy this old diagnostic; real missing-link composition remains open. |
| v1-s4-b3-contention-upgrade-native.yml | 684 | Only installed_normal matrix active; old PRS root approval, root service and successful binary replacement/rollback no longer current product routes. Current concurrency/identity evidence in N4 concurrent original UDP (actual concurrent same-key requests, distinct other direction, untouched boot identity and retained flows), N3 regrant and S25 active/offline legacy maintenance refusal with exact original receipt readback remain. Old project contender/root journal behavior belongs to original PRS source. Generic contender/partial interleaving obligations stay open where not proven by these current scenarios. Other old B3 fault, ETag, service-cycle and foreign-flow wrappers retained. |
| v1-s25-native.yml | 42 | Old native prerequisite rehearsal through accept_s2_native.sh, historical branch-only. Old shared policy and legacy simulated-WAN comparison obsolete. Current N2 exact schema/refusal, peer, native SDK and N3 guard prerequisite plus S25 two-layer native bootstrap, stale review refusal, fresh install, independent gateway/packet, cross-layer identities and fixed ledger/ACL guard remain mandatory. Both S25 outer and inner current entry require the Scoped selector. |

Current product contract: [PRODUCT_BOUNDARY](../decisions/PRODUCT_BOUNDARY.md), [NETWORK](../design/NETWORK.md), [C01-C12 crosswalk](CONSOLIDATION_ACCEPTANCE.md). Generic behavior is not waived when an old diagnostic or mechanism is removed. Current source tests are not represented as installed native proof.

## Source and reference evidence

- Exact tracked checkout in the existing ordinary development Box `73d2ecae-3932-46e6-b4c6-3feecf7fba15`. Exec `7c1b5b70a6d1543771d935e9ced4b8f9` completed exit 0; output sampled and further precise audit `e1650c95d2b3bc6debcc9246305d1aa9` completed exit 0 with complete output. Read full old diagnostic bodies and current CAS, kernel ports, reincarnation, concurrency and partial installed scripts. No native diagnostics were executed in the development Box.
- Exact six filename search excluding prose and old inventory, plus prior full tree dynamic consumer scan, found only historical self-trigger filters. Exec `70a058fc2e30aeb717fa87691915459b` exit 0; preserved original branch workflow blobs:
  - [S4 historical ref f1c7986](https://github.com/vanishs/boxweave/tree/f1c7986c070bf0c4d15a9cbaded6b73caccc0e58/.github/workflows): NDP f7433be222f06eaaf5c08617102400484b658c87; static 97dc5b68c72e6255a8f9d3aef449ff5883a9cf92; fanout 5d9f33c4587b32b5691dd267ce689d979c376898; grant-fault f738f78c61b7a71cfad16c109c4027ade7d6a360; contention-upgrade b36ea2da16303ac2a90afe6212c9a92aed9a0616.
  - [S25 historical ref d2b4b2e](https://github.com/vanishs/boxweave/tree/d2b4b2e9e8b4ddadc09f7bbaf6ca48f5fbe92d3b/.github/workflows): native prerequisite blob 3d8cfe9bbc6a9b9f7e2c731d3a9620d615fa731f.
- Current source anchors: `TestPR178ExactKernelPortsRejectsForeignRawAndDrift` (missing-live-veth/foreign/identity cases, modeled sysfs); `TestPR178ScopedNativeACLETagCASRefusesUntrustedPreimage` and `TestPR178ScopedNativeACLETagCASACKLossAndFaultsNeverRetry` (modeled native driver); `TestPR178ScopedAuditLedgerSQLNeverAdmitsDuplicateBridgeAddress`; `TestP1ScopedAddressHoldsRetainRetiredIdentity`. N2 main actual installed witnesses `PR178_N3_INSTALLED_LIVE_FLOWS`, `PR178_N4_CONCURRENT_ORIGINAL_UDP`, `PR178_N4_REINCARNATION`, `PR178_N4_PARTIAL_PACKET_SAFETY`, `PR178_N4_PARTIAL_FORWARD_CONTINUITY`, `PR178_N4_PARTIAL_NO_RECOVERY_REPLAY`. Prior RC15 [N2 main lane](https://github.com/vanishs/boxweave/actions/runs/46124999390/job/140917531110) and [S25](https://github.com/vanishs/boxweave/actions/runs/46124999468/job/140917530982) success, not new-head evidence.

## Remaining 20 historical-only routes: explicit hold list

These wrappers are not needed for current trigger identity, but cannot be responsibly deleted as a blanket cleanup. Hold until their remaining generic obligations are classified or converted in R1/R3.

| Group / exact workflow names (without .yml) | Unresolved disposition |
| --- | --- |
| v1-s25-bh018-installed-candidate, v1-s25-bh018-native | Exact independent 126/127 and original Exec side-effect/terminal WAL and kernel compatibility mapping; old labels not equivalent just by name. |
| v1-s3-file-installed, v1-s3-file-native | Exclusive SFTP create race, hostile types, bounded listing, independent source/Git winner evidence. |
| v1-s4-a-installed | Independent raw native event/SQLite and OAuth terminal correlation. |
| v1-s4-b1-native | Dual-stack new-flow and native Forward collision/removal matrix; old address-reuse scope separately obsolete. |
| v1-s4-b2-bridge-ownership-native | Actual foreign restricted-project bridge/raw-parent denials; current modeled port guard alone not equivalent. |
| v1-s4-b2-controller-root-installed | Foreign established flow continuity, root socket/PID/identity and uncertain original journal across effects. |
| v1-s4-b2-dedicated-native, v1-s4-b2-native | Exact internet=false directional IPv4/IPv6 and gateway denial under current architecture, foreign scope and reuse distinction. |
| v1-s4-b2-quiescent-native | Legacy quiescence/rotation obsolete; actual composed SQLite/native interleaving, foreign flow preservation and exact conntrack effect obligations not automatically resolved. |
| v1-s4-b3-stale-etag-native | Real approved-preimage native ETag competition; modeled CAS not sufficient to claim actual native fault. |
| v1-s4-b3-final-closure-native | Actual native service cycle, inside-ETag race and original recovery/fence. |
| v1-s4-b3-final-gates-native, v1-s4-b3-foreign-revoke-native | Actual foreign Project established TCP/UDP IPv4/IPv6 continuity during grant/revoke and root restart. |
| v1-s4-b3-grant-native | Broad fault matrix and parent Delete linked cleanup/original evidence. |
| v1-s4-b4-forward-native | Native Forward process kill/ACK/postread and dependent Delete matrix. |
| v1-s5a1-installed, v1-s5a2-installed, v1-s5b-peer-production | Original 126/127 fresh readback, CLI/MCP recovery parity, combined active Peer/Forward/pending Exec restart and existing S5 historical-presence contract. |

Keep two optional manual A071/UNKNOWN Create workflows and eight current workflows. Do not alter current required checks, historical refs, scripts, tests or runtime. No additional native-host access or host mutations.

## Verification

Exec `4ae1d4cf86dab8fbea375dc620e1f4e4`, exit 0, complete stdout/stderr. Staged six deletions, git diff --cached --check PASS. All 30 remaining workflow bytes identical to parent, scripts/runtime unchanged. Existing focused suites (CI scope 13, local hook 8, witness map 15, exact CI gate 12, S25 evidence 4, stable gate 14, release manifest 22) = 88 PASS. Full Go/race/candidate CI pending normal execution; no new native or runtime savings claim.

This slice: 36 -> 30 files, 9,051 -> 7,736 workflow lines (-1,315). Cumulative R1: 46 -> 30 files (-16, 34.8%), 10,241 -> 7,736 lines (-2,505, 24.5%). Historical-only: 36 -> 20. R1 remains partial; remaining unique behavior review is the concrete next task, not a blanket cleanup or current-coverage waiver. No merge or release. Revert the individual change to restore current-branch wrappers if necessary.
