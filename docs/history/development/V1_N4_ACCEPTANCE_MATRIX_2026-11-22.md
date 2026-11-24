# BoxWeave v1 - N4 acceptance execution matrix (2026-11-22 PT)

**Status: N4_PASS (scoped technical acceptance, owner-approved 1024 capacity substitute; real guest scale deferred).** Exact starting PR #172: `98ffae90ea16199a3986f4d90514962a459d1721`, which has N1/N2/N3 scoped functional sign-off and five green original PR checks. This document is the N4 **execution** record; it does not change N3 acceptance or authorize merge.

## Gates

| Gate | Evidence or method | Current result |
|---|---|---|
| N4-A: complete 1024 SQLite capacity and root/peer audit workload | New opt-in `TestN4ProjectCapacityAndNetworkAudit1024LocalBudget`, 1024 simultaneously OWNED seeded identities, PINNED dual-stack bindings and RESERVED fences in one private transaction; three repeated read-only measurements per API, 15-second local per-read refusal; no native mutations | **PASS, approved as capacity substitute** in 2-vCPU/2-GiB developer Box (not 1024 real guests) |
| N4-B: 1024/1025 native object bounds and unstable inventory | N3 `TestN3NativeProject1024ObjectsAnd1025FailureNoPartial` plus dual-read lease/network failure matrix, executed on an independent GitHub runner for exact PR head | **PASS** hosted 1024-capacity/model run 45123143747 |
| N4-C: authenticated installed native Peer/PRS grant/revoke and ACK-loss | Reuse the existing dedicated disposable native Incus 7.6 S4-B3 workflow with **only** N4 `installed_mcp` and `grant_ack_lost` scenarios; root/SDK/MCP operation journal, packet and native recovery checks, fresh signed dependency runner | **PASS** native PRS installed MCP and ACK-lost run 45123143701 |
| N4-D: installed native Forward data plane | Reuse dedicated disposable S4-B4 workflow: nonroot SDK and authenticated MCP, real IPv4/IPv6 TCP/UDP forwarded packets, exposure denials, crash and dependent Delete cleanup | **PASS** complete native Forward run 45124310035 |
| N4-E: exact-head general regression and distribution | Existing 5 required/specialized GitHub checks at PR N3 head all PASS; after N4 workflow/test changes, rerun and inspect exact new head | **PASS** five out of five on exact-head 86c20a861c9fa296cee02279929b873128262077 |
| N4-F: **1024 real Incus instances**, full root/kernel/PRS scale resource and latency, restart/fault/concurrency | Must use an independent large-capacity native runner; don't confuse fixture objects, 1024 SQL rows or 2-3 real data-plane guests with 1024 real instances | **DEFERRED by owner; not tested or guaranteed at real scale** |
| N4-G: independent acceptance | Evidence matrix, exact-head CI and resource budget review, clean private lab, no unsafe API/Host shortcuts | **N4_PASS**, technical/owner-scoped only (no second human code review) |

## Measurements - N4-A

Isolated developer Box: 2 vCPU / 2 GiB cgroup memory limit. Test operation `5f4eeea3585c2a855522e341469e46ce` exit 0. Exact 1024-row SQLite fixture, 3 independent read-only runs after setup:

- **Capacity:** 0.255 / 0.461 / 0.132 ms, max heap allocations per invocation **1,576 bytes**.
- **Project-native SQLite inventory:** 86.815 / 88.652 / 86.581 ms, max allocated bytes **5,914,536**.
- **Complete Peer/Root SQLite ledger:** 10.540 / 10.135 / 10.569 ms, max allocated bytes **2,823,624**.

These numbers are **not** native Incus API I/O, kernel nftables, PRS journal throughput, live packet throughput, 1024 daemon processes, or cumulative resident memory. The test intentionally has no original Create operations; it must not certify an unjournaled legacy/synthetic Box as production-proven. A real-guest scale gate remains essential.

## Constraints and safety

Current disposable nested native lab Box `b8eafebd-c0ad-4ef5-bf0a-dcb2f00613e3` also has 2 vCPU / 2 GiB memory. Do not attempt 1024 live native guests in this restricted environment or weaken Incus quotas. Reuse disposable GitHub-hosted native runners for real data-plane regression; do not modify the physical L0 Host or existing shared/native N3 guest. Any failure or missing original/native receipt must preserve UNKNOWN and static address claims. No arbitrary shell, third-party native adoption or ACL/root bypass.

This initial N4_PARTIAL checkpoint is superseded by the scoped N4_PASS decision below. Do not merge or release without the owner's separate explicit instruction.

## N4 phase-1 local exact-source and real one-guest read baseline

- Format 827 tracked selected files, parsed 47 tracked YAML without duplicate keys, N4 native model/SQLite capacity tests and \`go vet ./...\` PASS: \`5ded51dc9230e07958574240742fdc48\` exit 0, staged tree \`69e56ab756a39a1b949c959a09e21cfb53bf98f6\`.
- Full \`go test ./... -count=1\` on unchanged Go sources PASS, including State 148.745s, Control 20.132s and all other packages; targeted \`go test -race ./internal/state ./internal/native -run '^Test(N4ProjectCapacityAndNetworkAudit1024LocalBudget|N3NativeProject1024ObjectsAnd1025FailureNoPartial)$' -count=1\` PASS: \`eb7688320ffecb1c66c1a7a183bfa969\` exit 0. The N4 perf opt-in test remains excluded by default from ordinary broad suite, and explicitly executed in \`5ded51dc9230e07958574240742fdc48\`.
- Real restricted nonroot Incus 7.6 API/CLI baseline in nested native Box \`b8eafebd-c0ad-4ef5-bf0a-dcb2f00613e3\`, project \`user-1001\`: \`e147b7d51c6f34b8ab6878f7c2503b8c\` exit 0, one existing stopped guest and three native lease rows; 7 independent reads. Instance list median **31.53 ms**, maximum **70.71 ms**; bridge lease list median **32.25 ms**, maximum **35.10 ms**. Native instance count/UUID unchanged, no ACL/bridge/guest mutations. This is a **one-guest** observation, NOT a 1024-guest latency bound.
- The existing exact-source S4-B3 native PRS workflow is now optionally activated on the N4 PR branch with only \`installed_mcp\` and \`grant_ack_lost\` scenarios; the existing S4-B4 native Forward workflow is likewise activated on that branch, plus the dedicated \`v1-n4-capacity-budgets.yml\`. The old historic branch gates and freeze pin remain unchanged. These jobs use clean GitHub-hosted disposable native infrastructure, not the shared L0 Host. **Not yet proven until their exact new-head runs finish.**

## First exact-head CI and Forward fixture correction (N4 still PARTIAL)

GitHub exact N4 source \`1c541bba88d2b2743404be138e368998ff9fc64b\`, tree \`3d28250c287358185b180758d3f079e593ddeaac\`:
- **PASS** dedicated hosted 1024-row ledger/1025 model budget \`45123143747\`.
- **PASS** both native PRS scenarios \`45123143701\`: \`installed_mcp\` with real IPv4/IPv6 TCP+UDP directional allow/reverse/unrelated deny, three real native peers, original journaling/authorization and dependent-Delete; \`grant_ack_lost\` original no-replay recovery. Original PRS/native tests passed on disposable hosted Ubuntu24.04 Incus7.6, not on shared L0.
- **PASS** five usual exact-head PR checks: \`45123148287\` Quality, \`45123148263\` installed lifecycle, \`45123148242\` Forward policy, \`45123148321\` S1, and \`45123148276\` amd64/arm64 build.
- **NOT YET PASS** separate full native Forward run \`45123143699\` (first job \`137521311773\`, rerun \`137522972965\`). Each attempt passed real nonroot SDK native TCP+UDP Forward, authenticated MCP IPv4/IPv6 TCP/UDP packet create/delete and exposure refusal, four process-kill ACK/postread recovery cuts, dependent Delete and the first native Delete ACK cut. The run then failed **outside a Go test**, at fresh native \`s4b5ack\` instance setup (\`incus start\`) after it recreated the same guest name with the same static IPv4 and IPv6. Attempt 1 failed during the third cut; repeat attempt failed during the second cut. **Do not claim this is proved an Incus flake or a product failure.** Native outcome from the failed setup is not represented as an accepted recovery result.
- **N4-only CI fixture adjustment pending new exact-head run:** each fault cut now uses a different previously unused static IPv4 and IPv6 (\`.22\` through \`.25\`), preserving original guest-name operation and all fault assertions. It also records actual native \`incus info --show-log\` plus bridge leases if start fails, without retrying a potentially ambiguous native effect or touching product privilege/allocator logic. Historical unrelated branch behavior remains fixed-IP as before. N4 and complete-Forward signoff continue blocked until full native run proves all four cuts and cleanup.

This CI iteration is an independent *real* data-plane acceptance record, **not** proof that 1024 actual native Boxes can fit or complete audit/PRS work within resource budgets.


## Scoped N4 acceptance - owner-approved alternative capacity evidence

**Decision: N4_PASS for this PR's technical scope, 2026-11-22 PT.** The owner explicitly accepts the full 1024-row SQLite network and Peer/Root ledger workload and 1024/1025 bounded native-object model checks in place of a 1024-real-Incus-guest scale experiment. Real 1024-guest runtime, memory, storage, PRS/ACL kernel cost and latency trials are **DEFERRED**, not tested or represented as passed. No product capacity guards, original-operation identity, ACL or fail-closed UNKNOWN rules were relaxed.

Exact-source evidence: 1024 SQLite/model capacity and safety hosted job 45123143747 PASS; native real Peer/PRS directional IPv4/IPv6 TCP/UDP with installed MCP and ACK-lost recovery, two scenarios, run 45123143701 PASS. Complete Forward TCP/UDP IPv4/IPv6 packet, restricted installed MCP, crash, all four Delete fault-cut and native cleanup in run 45124310035 PASS. Exact code head 86c20a861c9fa296cee02279929b873128262077 (tree c47986c5ee224fb128b5cc8ad430f2c75346d793) passed all five usual PR checks: 45124316825, 45124316724, 45124316741, 45124316809 and 45124316718.

Prior native Forward run 45123143699 and its rerun failed during repeated one-use guest creation with immediately recycled IPv4/IPv6; these failures remain retained. The later exact-head run passed when each fault cut used a distinct test fixture address, without weakening the product policy or skipping any fault assertion.

**Independent scope:** GitHub disposable native runners, separate Box worktree and verified original receipts. No independent human reviewer or physical L0 Host signature is implied. The L0 host-status storage quota attestation warning remains a separate existing deployment risk. The owner-approved alternative is not a real-1024-guest throughput or resource guarantee.

**Merge rule:** PR #172 remains Draft and UNMERGED; recheck the branch, checks and any required reviewer rules on the user's explicit merge instruction. A future real native 1024-guest scale qualification remains a deferred follow-up.
