# S4-B2 Stage 2 wrap-up and Stage 3 acceptance handoff

Date: 2026-11-17 PT. Sole Draft PR #153, branch feat/v1-s4-runtime-integration-20261116.
STATUS: S4_B2_PARTIAL. B3 NOT_STARTED. Stage 1 freeze: docs/development/S4_B2_STAGE1_FREEZE_2026-11-17.md.

## What Stage 2 now implements

1. Canonical read-only SQLite original authority (internal/state/network_peer_root_handoff.go): host-root audit reads a real claimed DISPATCHING box_peer_set revoke, checks canonical allow=false intent, same Project exact source/target UUID and incarnation, PINNED IPv4/v6 leases, matching pending peer_grants, exactly ACL+two-Box claims and active admin token. UNKNOWN, foreign actor, malformed requests and drift are refused.
2. Controller durable staging and atomic success (network_peer_root_stage.go, network_peer_root_worker.go): transform a CONFIRMED grant into desired=0/PENDING only after checks; Claim before side effect; native isolation before root dispatch; any lost ACK, bad root outcome or failed independent verifier => UNKNOWN and no retry. Verified operation plus REVOKED grant are committed in the same SQLite transaction.
3. Bounded host-root service (native_peer_root_socket.go, cmd/boxweave/main.go, internal/control/peer_root.go): operator-started restricted Unix socket, SO_PEERCRED exact boxweave UID, bounded JSON only original operation ID and digest, host approval pinned to one exact original and one SHA, native root independently rereads all facts. No general shell, arbitrary root network parameters or public MCP Peer writer.
4. One-shot per-original root journal (network_peer_conntrack_operation.go): exclusive fsynced UNKNOWN before kernel effect, repeat/crash/ACK loss never replay; root lock shared with prior root mechanism; legacy single global journal remains an explicit stop boundary.
5. Native host route scope (network_peer_host_routes.go): exact /usr/sbin/ip IPv4/IPv6 route JSON read, refuse unknown, unmanaged overlapping routes, missing bridge ownership and kernel route changes, in addition to earlier all-Project Incus network/NIC inventory gate.

## Evidence

Disposable L0 Box 3ea5388a-1ca4-4098-8d94-38c2892818d8; original create 3d3df2232934051e453c06a7cd59cbe7; native 8cc2fc04-78a0-4822-88a0-640141beec39. Source started at 24380ac013544128d2ebe22337fe227d8e4e08c2. Tests: actual SQLite original/claims and expired/revoked token cases, Unix SO_PEERCRED wrong UID and unknown parameters, exact-root digest and original, isolated kernel route duplicates, per-original no-replay, controller quiescence/unknown and failed verifier with atomic final status.

Original L0 full make verify operation 3291f32a9f535d41918339631a13b0a1, native 8b9e6121-ae15-4a1d-8a74-77a0113d1673, exit 0. This run PRECEDED final operator exact-original pin: final source needs another full make verify. Exact native Incus 7.6 CI must be run and original logs inspected before declaring native PASS.

## Honest P0 blockers to final B2 acceptance

- The private typed isolate and verify callbacks are NOT yet product-wired to installed stop/ACL and effective per-NIC nft/foreign packet checks. Unit-test callbacks alone are NOT installed B2.
- Root socket exists as explicitly operator-started host CLI, not verified systemd/install/upgrade lifecycle with actual restricted service account, restart and cleanup.
- Per-original journal freezes UNKNOWN but does not yet provide a verified safe recovery/rearm for a fresh authorized operation after ambiguous failures, and the legacy global-journal transition is an explicit stop.
- Need independent actual live installed IPv4+IPv6 TCP/UDP sessions, foreign Project traffic unaffected, native nft readback, native Incus 7.6 full root validation, and complete installed Scriptlet old-client Files/Exec/Profile/ACL rights before/after/revert.
- Existing outside-Project IP/route checks are necessary, not a complete netns/zone or unmanaged host root guarantee.

## Stage 3 agent instructions

Read AGENTS.md, Stage1 freeze, this document, and all exact-HEAD workflow logs. Do not mark B2_PASS just because the code has tests. Start by determining whether final Stage2 P0 integration blockers above are implemented on newest PR head; if not, FINISH them IN SAME Draft PR #153, run gofmt before every commit, full make verify in disposable L0, independently run native root Incus 7.6 installed fixture and S4-A regression on SAME final SHA. Verify original operations and clean up all one-use Boxes.

Acceptance matrix: installed restricted OAuth controller -> durable original Peer revoke -> operator-approved root original ID + SHA through SO_PEERCRED -> complete stop/ACL default DROP -> per-NIC nft original readback -> exact conntrack dual-stack live TCP/UDP teardown; unrelated foreign Project live flows remain; crash/ACK loss/partial-NIC fault retain UNKNOWN without redispatch; explicit safe recovery/rearm; installed Scriptlet old restricted Files/Exec/Profile/ACL parity + rollback/restart; raw operation receipts, exact final SHA, quality/race, dual-arch, independent verifier.

If any P0 remains, report S4_B2_PARTIAL and B3_NOT_STARTED, never fabricate a signoff. Do not open PR, merge, publish RC or mutate shared L0 global bridge/ACL/Scriptlet. Never introduce public root shell, broad conntrack flush or generic agent root access.
