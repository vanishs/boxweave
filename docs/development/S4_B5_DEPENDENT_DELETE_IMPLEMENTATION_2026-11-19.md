# S4-B5 dependent Box Delete implementation — 2026-11-19

Status: **S4_B5_PARTIAL**. Development-Box regression and real TCP/UDP dependent Delete pass. Linked native PRS peer and complete dependent-delete fault recovery acceptance remain open. This is not S4_B5_PASS.

## Contract
- A dependent delete holds one durable parent operation and a Project-wide admission fence. Its internal children revoke confirmed instance forwards, revoke confirmed peer grants, and stop the instance if required. No new public tool or database tables are required.
- Peer revocation still requires the existing independent host-local PRS review and exact digest approval. The controller cannot grant itself root approval. Linked child claims are verified against the parent actor, token, Project, UUID and incarnation.
- Unknown, quarantined, foreign or mismatched dependencies fail closed. A failed or interrupted child prevents final deletion, and no compensating regrant/recreate is attempted.
- Native UUID, exact pinned NIC and complete proxy inventory are checked. Parent success requires native delete completion and independent absence plus atomic tombstone and dependency settlement.
- Completed deleted network bindings and addresses retain verified historical lineage. RELEASED history does not permit address reassignment.
- Dependency-free deletion retains the existing STOPPED-only behavior. A dependent deletion may stop the exact Box as a recorded child.
- Parent and child IDs are linked in operation resource claims and safe operation events. Recover reads the durable originals rather than replaying side effects.

## Evidence and boundaries
Development Box: s4b5-dependency-lab (1dc9a7f9-3f12-4d2f-b07b-f81c9edd838f).
Work ref: s4b5-dependent-delete-20261119.
- SQLite tests: parent/child atomicity and replay, linked root proof and historical DROP retention, partial cleanup/restart UNKNOWN fencing.
- Development Box full state/native/control Go package tests passed; original Exec f5607aec196ec049a41537ae7701eb1b, guest exit 0.
- Native acceptance test passed in a one-use restricted Incus fixture, using actual HTTP MCP forward creation, TCP/UDP echo, parent dependent deletion, native absence, packet refusal, SQLite child receipts and replay. It runs after existing B4 native checks.
- Still requires actual linked peer PRS revocation integration and parent final-delete ACK-loss/reconciliation acceptance. No live L0 host configuration changes are authorized. B6 is not started.

## Source-anchored acceptance update

- Tested implementation source: `c483a0a9d6fa7e25d66f9a532d0dc95d617bbd9d`.
- Complete `make verify` in development Box: original Exec `8b62e9ee5383a2972613ff5eee81a7d3`, guest exit 0, including Go race across all packages and Python testkit/uninstall/raw-witness suites.
- [Real restricted Incus 7.6 native CI](https://github.com/vanishs/boxweave/actions/runs/44691838835/job/136139215889): SUCCESS. `TestB5NativeMCPForwardDependentDelete` ran (not skipped), `S4_B5_NATIVE_MCP_FORWARD_DEPENDENT_DELETE=PASS`, 8.78 seconds; native TCP/UDP, B4 regressions, exact Box absence, revoked claims, original parent replay, child native receipts, and fixture cleanup verified.
- Native B3 regression jobs for this source are separate from B5 linked-peer evidence; passing B3 jobs do not sign off the B5 parent/child PRS path.
- Remaining gates: actual linked-peer PRS delete (including unrelated grants), partial native cleanup and final Delete ACK-loss/crash or expired-receipt outcomes, and independent result classification. Maintain fail-closed UNKNOWN and do not redispatch. Keep PR Draft; retain development Box for the next iteration.

## Nested-first acceptance attempt

- Separate owned Box `be1af99d-b9b1-48f9-9548-524473eef163` (`s4b5-nested-acceptance`), work_ref `s4b5-native-acceptance-20261119`, creation 8c755aa4b61e2e182fb80e87bfb3b9b5. Bootstrap exit 0, exact source db570d9b0c5d4f277c14dfa41e786d0a85652ead, Exec 0d8b6b399bb6c907d75dcd059a0157aa.
- Read-only security preflight 78bad1b799d521b395404aa0c22187bb: kernel AppArmor enabled Y; PID1 and Exec label unconfined; parent daemon sockets absent. Cannot attest nested ancestor enforcement, so do not start inner Incus or weaken the shared host. This is an acceptance-environment blocker, not proof of a B5 defect.
- Fallback: linked PRS tests in B3 installed_mcp fixture; real parent Delete through MCP, exact approved PRS child, tombstone and unrelated real IPv4/IPv6 TCP/UDP packets. Final Delete ACK cut tests in B4 disposable native fixture: native forward cleanup, actual final native delete, abrupt test worker exit before/after durable ACK, SQLite reopen, UNKNOWN, original replay and fresh key fence. Local compilation/YAML/Python checks pass. Native runs required before signoff.

## Native linked PRS and final Delete interruption acceptance update

Tested source: `7e4982022225c6ace8665d3bf89b5aa2c4b688da` (no production code changes from c483a0a9; exact native-test additions only).

- [Installed linked PRS dependent delete](https://github.com/vanishs/boxweave/actions/runs/44694417303/job/136147638041): SUCCESS. Original parent `fac0335da96d955c655d4a9265296322`; `S4_B5_LINKED_PARENT_MCP_REPLAY=PASS` and `S4_B5_NATIVE_LINKED_PRS_DELETE_TOMBSTONE_UNRELATED_PACKETS=PASS`. Actual root reviewed child revocation, final native absence, SQLite parent tombstone, two released address fences, historical revoked grant, and unrelated dual-stack TCP/UDP confirmed.
- [Final Delete native ACK interruption](https://github.com/vanishs/boxweave/actions/runs/44694417339/job/136147637970): SUCCESS. Both cuts emitted `S4_B5_FINAL_DELETE_ACK_RECOVER_UNKNOWN_NO_REPLAY=PASS`. Before ACK original `09cd151edf7baa2f66cdd48cc6417fbf` native `dc25b846-da73-427c-80c4-3a738f23c1d3`; after ACK original `c904fe3927d662b598b7c3ec6f7309d4` native `dd98ec43-b9e2-42b6-92a8-4aaae44cfd1a`. Actual native delete after confirmed proxy cleanup, abrupt worker exit, reopen same SQLite, recovered UNKNOWN, retained confirmed child removal, no false tombstone, same-key original readback and fresh-key fence. Before-ACK case cannot prove native ID from durable storage; after-ACK case retains native ID. This verifies conservative recovery, NOT a new automatic or host-reviewed UNKNOWN-to-SUCCEEDED reconciliation.
- Same source Go quality/race gate and both architecture builds: SUCCESS. Nested Box exact-source `make verify`, Exec `26f3d16c0266c0c248d4406513f5e374`, exit 0. The nested native execution was blocked by unverifiable AppArmor ancestry; native acceptance was completed on isolated CI instead, not by weakening the shared host.
- Both named targeted acceptance scenarios PASS. Keep `S4_B5_PARTIAL` for broader acceptance until remaining original-plan native partial-cleanup failure, known-terminal reconciliation/expired receipt classification, and S4-A independent regression are assessed; the previous broad S4-A run failed, latest is in progress. No B6 or release signoff.
