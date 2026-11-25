# H4B - Scoped network reclamation proposal (owner decision required)

Reporting and existing-bound admission reserve are implemented in H4B. Reclamation is a proposed feature, NOT an approved effect, implemented reclaim command, migration recipe, or release sign-off.

## Current limits

| Budget | Meaning | Released on ordinary Delete? |
| --- | --- | --- |
| 1024 | Owned plus unresolved Create not already counted as owned | Yes after verified Delete, never on UNKNOWN |
| IPv4 nominal | 2^(32-prefix)-3; union native devices/leases and durable holds reduces the upper bound | No in Scoped; Legacy IPv4 retirement uses separate proof |
| 8192 | Existing scoped retired audit bound; reserve a future tombstone for each owned/pending Create | No; owned moves to retired |
| 32 | Retained directional Peer rows including revoked and pending/UNKNOWN | No; revoked is a deny tombstone |
| 8192 | Root grant/revoke history; reserve one revoke per active grant | No; a new grant spends two unreserved units |

A /24 can exhaust at 253 cumulatively distinct held addresses with zero live Boxes. /21 nominal 2045 does not alter other limits or solve churn. Failed/unknown Create can leave holds without an owned identity. The reported address union is not a dual-stack, storage, compute or effect guarantee. Low means at most 20% remaining; unsettled/inconsistent Peer evidence is unverified. The new-grant history upper bound is floor(max(0,8192-completed-active)/2). Root and SQLite counts are independent observations, not a cross-source transaction.

Fresh package bootstrap may widen the exclusive empty bridge to /21 after route/lease proof, otherwise retain the verified smaller pool. Before creation the exact prefix is unknown. The subsequent scoped install review displays actual prefix and nominal capacity before approval. Existing bridges never resize automatically. Custom sizing and live resizing need a separate reviewed feature.

## Alternatives and recommendation

A. Current immutable retention remains the safe v1 policy; expose exhaustion and preserve revoke/Delete headroom. A larger subnet postpones exhaustion only.

B. Reclaim only an independently verified retired tuple, preserving immutable original IDs, idempotency receipt mappings, old incarnation and archived lineage. Address release is separate from direction removal and history checkpointing.

C. Versioned Root-authenticated checkpoint/compaction of exact terminal policy history and retired direction rows. A checkpoint must commit to prior history, not truncate or edit sequence numbers. Keep all original receipts and original key/intent digests. Root boot, recovery and uninstall must verify the new generation and archived chain.

Recommend B then C as distinct follow-up phases only after owner approval and native proof. Eventually routine verified reclaim should be automatic inside existing trusted lifecycle boundaries with MCP read-only visibility, not a Host step per Box. Unprovable cases retain resources and escalate locally.

## Mandatory proof and transaction invariants

1. Bind exact original Create/Delete, UUID/incarnation, IPv4/IPv6/MAC, bridge, ACL, project and Root lifecycle journal. Require successful verified Delete and independent absence across native inventories, stopped guests, snapshots/backups and pending operations capable of restoration. Name reuse is not evidence. Refuse unresolved Operation, output-GC, Forward, Peer, restore or dependent claims.
2. Prove exclusive route/network scope, no foreign lease/device/ACL reference, no stale DHCP/SLAAC lease, neighbor/FDB or ambiguous veth. Account for delayed events, lease expiry and boot. Prove old guard/ACL cannot authorize a new incarnation. Never expose foreign inventory to MCP.
3. Fence the exact affected identity and policy transition. Root records reviewed intent, immutable evidence digest/generation, pre-effect barrier, effect if needed, independent readback and terminal result. Selective old-flow conntrack proof must cover both directions, IPv4/IPv6, NAT/zone and exact tuple lineage. No global flush or unrelated NIC restart. Absence of a conntrack record alone is insufficient; unsupported proof blocks.
4. Only after committed independent Root proof, one service-side SQLite transaction rechecks original revision, generation and conflicts, publishes exact eligibility. Allocator atomically claims the eligible tuple for a new original, derives a new UUID/incarnation, and starts with verified deny-by-default policy. Old grant and receipt must not affect new identity.
5. Root/SQLite are not distributed atomic: durable phase barriers, fsync, monotonic generation, recovery-only inspection/reconciliation of same original. Lost ACK, crash before/after every step, interrupted restore or unknown journal retain holds and never redispatch. On boot incomplete generations fail closed for affected resources while preserving unrelated flows. Backup rollback and downgrade cannot restore stale eligibility.

Direction removal additionally requires exact retired endpoint, terminal revoke, deny and absence of stale established flows. A live endpoint's revoked row is not erased merely because it is revoked. Checkpointing requires authenticated predecessor/archive digest and sequence, current active/revoked projection, guard proof and future revoke reserve, atomically published generation. Reject missing/partial/wrong-epoch checkpoint or rolled-back state. Do not raise 32/8192 until proof and new performance bounds exist.

## Follow-up acceptance (not performed here)

Real disposable Incus VM: churn past /24 and direction/history boundaries; same-name/different-UUID; lost ACK exact replay; stale lease/route/FDB/conntrack; foreign ref; ACL drift; concurrent allocation/revoke/restore; incomplete journal; expired proof; crash cuts; reboot, backup rollback. Verify default deny on IPv4/IPv6 and established unrelated TCP/UDP/Forward, unchanged PID/NIC/autostart and no inherited grant/stale-flow bypass. Independent clean CI plus bounded synthetic scale; no synthetic substitute for packet proof. Failed evidence keeps holds.

Owner decisions: approve exact retired tuple reuse? What supported lease/flow witness is sufficient? How long to retain checkpoint archives and original receipts? What backup compatibility and scale targets? Until approved, no release of Scoped holds, journal/lineage deletion, cap increase, live resize or new MCP reclamation tool.
