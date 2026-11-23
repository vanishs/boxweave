# BoxWeave v1  N4 acceptance execution matrix (2026-11-22 PT)

**Status: N4_IN_PROGRESS / NOT SIGNED.** Exact starting PR #172: `98ffae90ea16199a3986f4d90514962a459d1721`, which has N1/N2/N3 scoped functional sign-off and five green original PR checks. This document is the N4 **execution** record; it does not change N3 acceptance or authorize merge.

## Gates

| Gate | Evidence or method | Current result |
|---|---|---|
| N4-A: complete 1024 SQLite capacity and root/peer audit workload | New opt-in `TestN4ProjectCapacityAndNetworkAudit1024LocalBudget`, 1024 simultaneously OWNED seeded identities, PINNED dual-stack bindings and RESERVED fences in one private transaction; three repeated read-only measurements per API, 15-second local per-read refusal; no native mutations | **PASS, SQLite-only** in 2-vCPU/2-GiB developer Box |
| N4-B: 1024/1025 native object bounds and unstable inventory | N3 `TestN3NativeProject1024ObjectsAnd1025FailureNoPartial` plus dual-read lease/network failure matrix, executed on an independent GitHub runner for exact PR head | local previous PASS; independent CI **PENDING** |
| N4-C: authenticated installed native Peer/PRS grant/revoke and ACK-loss | Reuse the existing dedicated disposable native Incus 7.6 S4-B3 workflow with **only** N4 `installed_mcp` and `grant_ack_lost` scenarios; root/SDK/MCP operation journal, packet and native recovery checks, fresh signed dependency runner | independent CI **PENDING** |
| N4-D: installed native Forward data plane | Reuse dedicated disposable S4-B4 workflow: nonroot SDK and authenticated MCP, real IPv4/IPv6 TCP/UDP forwarded packets, exposure denials, crash and dependent Delete cleanup | independent CI **PENDING** |
| N4-E: exact-head general regression and distribution | Existing 5 required/specialized GitHub checks at PR N3 head all PASS; after N4 workflow/test changes, rerun and inspect exact new head | new-head CI **PENDING** |
| N4-F: **1024 real Incus instances**, full root/kernel/PRS scale resource and latency, restart/fault/concurrency | Must use an independent large-capacity native runner; don't confuse fixture objects, 1024 SQL rows or 23 real data-plane guests with 1024 real instances | **NOT RUN / OPEN** |
| N4-G: independent acceptance | Evidence matrix, exact-head CI and resource budget review, clean private lab, no unsafe API/Host shortcuts | **NOT SIGNED** |

## Measurements  N4-A

Isolated developer Box: 2 vCPU / 2 GiB cgroup memory limit. Test operation `5f4eeea3585c2a855522e341469e46ce` exit 0. Exact 1024-row SQLite fixture, 3 independent read-only runs after setup:

- **Capacity:** 0.255 / 0.461 / 0.132 ms, max heap allocations per invocation **1,576 bytes**.
- **Project-native SQLite inventory:** 86.815 / 88.652 / 86.581 ms, max allocated bytes **5,914,536**.
- **Complete Peer/Root SQLite ledger:** 10.540 / 10.135 / 10.569 ms, max allocated bytes **2,823,624**.

These numbers are **not** native Incus API I/O, kernel nftables, PRS journal throughput, live packet throughput, 1024 daemon processes, or cumulative resident memory. The test intentionally has no original Create operations; it must not certify an unjournaled legacy/synthetic Box as production-proven. A real-guest scale gate remains essential.

## Constraints and safety

Current disposable nested native lab Box `b8eafebd-c0ad-4ef5-bf0a-dcb2f00613e3` also has 2 vCPU / 2 GiB memory. Do not attempt 1024 live native guests in this restricted environment or weaken Incus quotas. Reuse disposable GitHub-hosted native runners for real data-plane regression; do not modify the physical L0 Host or existing shared/native N3 guest. Any failure or missing original/native receipt must preserve UNKNOWN and static address claims. No arbitrary shell, third-party native adoption or ACL/root bypass.

N4 remains **PARTIAL** until the fresh CI native jobs and honest 1024-real-instance scale limits are resolved. Do not merge PR #172 or sign a final v1 release based solely on this matrix.

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
