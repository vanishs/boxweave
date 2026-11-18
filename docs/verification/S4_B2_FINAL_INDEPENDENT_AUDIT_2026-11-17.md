# S4-B2 final independent audit: all-sibling root isolation and signoff gate

Date: 2026-11-17 PT / 2026-11-18 UTC. Sole Draft PR #153, branch `feat/v1-s4-runtime-integration-20261116`. Status: **S4_B2_PARTIAL / B3_NOT_STARTED** until all final-commit gates pass. No merge, RC, physical host installation, release, or global L0 ACL/Scriptlet mutation is authorized.

## Recovered exact-source evidence

Pre-fix source SHA: `d8bffc0a7a0444e83b537c77c05ad500ee9f2af3`. The installed controller-to-root Incus 7.6 jobs from [run 44245665880](https://github.com/vanishs/boxweave/actions/runs/44245665880) are both successful: normal `134603886570` and actual lost root ACK `134603884910`. Their **raw terminal logs** contain `S4_B2_INSTALLED_NONROOT_ROOT_SOCKET_OPERATOR_SHA_ORIGINAL_NFT_SQLITE_REVOKE=PASS`, `S4_B2_NATIVE_INTEGRATED_ROOT_NFT_DUALSTACK_PER_NIC=PASS`, `S4_B2_INSTALLED_FOREIGN_PROJECT_ESTABLISHED_TCP_UDP_DUALSTACK_SURVIVED_REVOKE=PASS`, root systemd SIGKILL automatic restart, restricted credentials, binary upgrade/rollback without replay, and `S4_B2_INSTALLED_ROOT_ACK_LOST_COMPLETED_JOURNAL_READONLY_RECOVERED_NO_REPLAY=PASS`. These are actual pre-fix installed/native receipts, not merely standalone design tests.

Earlier-source Scriptlet installed native run [44241705654](https://github.com/vanishs/boxweave/actions/runs/44241705654), job `134590925074`, passed old restricted TLS client Project/Profile/Files/Exec/ACL parity, real Incus restart, failed-update rights retention, journaled exact rollback, and post-rollback restart. The Scriptlet source had not changed between that successful run and pre-fix `d8bffc0`; nevertheless a later code commit is a new candidate needing its own acceptance policy.

The pre-fix S4-A installed regression [44245665986](https://github.com/vanishs/boxweave/actions/runs/44245665986) originally failed job `134605651686` at `scripts/s2_p1_operation_list_boundary.sh` after discovering unique compiled privateReadReply CALL `0xc7153a`, without reaching its PASS marker. Exact-source rerun job `134610666447` **SUCCEEDED**; raw logs explicitly show `S4_A_OPERATION_LIST_COMPILED_CALL_TWO_INDEPENDENT_AUTH_CUTS=PASS` and `S4_A_NATIVE_STAGE=S2_P1_OPERATION_LIST_BOUNDARY_PASS`, using the same compiled PC. This is concrete rerun acceptance, not a proven root-cause classification of the first GDB fixture failure.

## Independent P0 source audit and scoped repair

The installed root quarantine stops **all** owned NICs on the bridge, not merely the two UUIDs in the Peer intent. Before this change `StageRootPeerRevocation` and root `ReadRootPeerRevocation` expected exactly three claims (`acl:...`, `box:source`, `box:target`), whereas ordinary Box lifecycle/Exec operations may hold name-based claims. Third-party sibling Box C could therefore overlap the root all-NIC stop/start sequence even when native snapshots and route checks remain fail-closed. This is a real missing *mutual exclusion* property; native drift snapshots alone do not close it.

This candidate preserves the existing canonical three resource claims, journal format, original operation, approved root SHA, and no-replay behavior. It adds a **Project-level exclusive SQLite admission barrier**, not a list of arbitrary new user-supplied NIC/Box claims:
- An admitted `box_peer_set` requires no other unresolved operation in the same Project; all later non-Peer admissions in that Project reject while the Peer operation remains unresolved, including unknown UUID new Box Create.
- A privileged Peer UNKNOWN blocks all same-Project admissions regardless of any forged/legacy generic `recovery_audit.RELEASED` record. Native root-specific independent reconciliation remains the only completed-journal release boundary.
- Staging and the independent host-root read-only original proof independently recheck for existing sibling operations in older records. All preflight checks remain fail-closed. Foreign Projects are not serialized by the barrier.

The added tests `TestS4B2RootAllSiblingProjectAdmissionBarrier` and `TestS4B2RootStageAndReadRecheckOldSiblingOperations` check before/during root admission, Box Exec/Start/Stop/Create, cross-Project noninterference, UNKNOWN, forged generic release, and pre-existing sibling records. Existing partial/lost-ACK Root UNKNOWN tests remain required.

## Source validation and remaining acceptance

Disposable L0 development Box UUID `8ae9ca23-c1d5-4842-8c91-e1b52b9a6a1b` was created solely for this audit, using the L0 MCP `box_create`, `operation_get`, `box_exec`, `box_file_apply`, and `operation_output_get`. It cloned exact `d8bffc0`; Go toolchain 1.26.9; no L0 host/network/Scriptlet policy changes. Initial post-edit focused state tests and full `make verify` (including all-package race, vet, format, module verification, Python witness 7/7) passed at BW Exec `67b48563bd37ee357986dd4e9280f294`. A final verify after the forged generic-release negative is required before committing; the test result, source digest, CI evidence, and disposable cleanup receipts belong in the PR handoff.

**Do not sign B2 from the previous `d8bffc0` runs after this fix**. Remaining mandatory P0 gates are (1) final committed SHA `make verify` and same-SHA quality, race and two-architecture builds; (2) new installed controller-root normal/lost ACK, actual per-NIC nft/foreign-Project dualstack TCP/UDP witness plus partial ACL/nft failure UNKNOWN containment under the new sibling barrier; (3) exact-SHA S4-A installed GDB regression; (4) Scriptlet source-equivalence review or same-SHA revalidation, independent raw-log verifier, and exact cleanup proof. A completed root journal is eligible only for separate read-only reconciliation; a partially mutated `DISPATCH_UNKNOWN` journal remains frozen, with no automatic re-arm or original redispatch.

Any missing or failed required gate means **S4_B2_PARTIAL**, and S4-B3 remains blocked. This document is a checkpoint, never a self-certified verifier PASS.
