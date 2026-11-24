# PR178 N3 prerequisite checkpoint

Status: partial implementation, NOT installed Peer acceptance. Keep SCOPED_PEER_N3_NOT_ENABLED and PR178 Draft.

Baseline: 551e71b7602c4cdf96ed3575c08f1b15f7862167 (N2 CI accepted). Fresh installation only; initial guard digest changes. No migration of older installations.

## Implemented and tested

- Exact nft bridge original/reply direction matching, established replies only, IPv6 on-link neighbor discovery (hop limit 255), default deny including mixed ULA/link-local pairs.
- Whole-original durable four-step journal: exclusive UNKNOWN intent before each dispatch, repeated prefix witness, verified postimage, hash-linked records, complete receipt only after full inventory readback. No replay. Private callbacks are not installed Root authority.
- Scoped SQLite retired-history audit checks exact v2 Create/Delete lineage, claims, events, UUID, ACL and RELEASED address fences. Addresses remain non-reusable.

## Box evidence

Development Box ac585d2c-37ae-4879-a2e6-3cecc338d4d7, work_ref pr178-n3. Existing nested instances untouched.

- e2eb4334aefc8432df3407e1a75f18a0: three real isolated netns runs, IPv4/IPv6 TCP/UDP; AB revoked, existing BA/CD tracked connections survive. Guard-only fixture; no Incus ACL write or conntrack purge.
- 54a5a55d229d7618cdc21ffcc43ba7d5: three race-enabled journal fault-matrix runs.
- 38486426881f23eb1e5af12bde2b6db1: three retired-history runs, real Store transactions and seven negative cases.
- b92af43aac875d2c365ada18f47e45d9: read-only actual N2 lab history, four deleted Boxes and eight released fences.

CI adds a mandatory guard prerequisite marker to the scoped N2 matrix, not a final N3 acceptance marker.

## Remaining N3 blockers

1. Bind v2 staged Peer to trusted MCP and Root, independently verify actor, claims, native inventory and all stage prefixes. Atomically finalize only the complete verified Root receipt.
2. Prove directional conntrack ownership, expiry and reclassification. Pair-wide purge can disrupt independent reverse grants. Guard-only proof does not establish ACL or conntrack correctness.
3. Verify retired Create/Delete Root receipts and native absence. Support Create/Delete with existing grants without discarding address holds.
4. Persist verified policy at boot, fail closed on unresolved effects, never replay or restore uncertain grants. Preserve Box boot preferences and unrelated ACLs, sessions and Forward.
5. Run fresh installed MCP lifecycle and failure tests in nested Boxes, then add exact-head N3 final CI gate. N2 regressions must pass.

N4 expands traffic/fault coverage. N5 is release acceptance.
