# PR #178 — Exact root kernel-port and nft JSON readback checkpoint (2026-11-23)

**STATUS: READ-ONLY KERNEL FENCES TESTED IN A DISPOSABLE BOX. NOT an MCP-to-Root grant/revoke path, kernel writer, boot installer, or final no-downtime acceptance.** This continues [the owner-approved requirements](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md), the [scoped policy experiment](V1_PR178_SCOPED_POLICY_EXPERIMENT_2026-11-23.md), [bridge guard proof](V1_PR178_SCOPED_KERNEL_GUARD_PROTOTYPE_2026-11-23.md) and [native ownership preflight](V1_PR178_ROOT_GUARD_READONLY_INVENTORY_2026-11-23.md). PR #178 remains Draft; #177 Doctor must remain unchanged.

## New source and threat boundaries

- `internal/install/network_peer_scoped_kernel_ports.go`: root-collected **read-only** sysfs veth ownership check that allows unaffected C/D to remain running. Uses independently observed Incus `volatile.eth0.host_name` and immutable UUID for each RUNNING/READY Box, and twice enumerates symlink entries of `/sys/class/net/<bridge>/brif`. Fails closed for an extra raw/foreign veth, missing native port, inconsistent Box/UUID/name, duplicate/invalid host name, missing sysfs, unsupported lifecycle status or concurrent port changes. A STOPPED Box cannot have a live port. This is not an independent verification that supplied native structs came from Root; the production collector must first re-read protected SQLite and full root Incus objects.
- `internal/install/network_peer_guard_kernel_readback.go`: pure `NativePeerGuardKernelVerifyJSON` checks the actual **nftables JSON v1 kernel ABI** against the exact expected policy: one `bridge` table, one `protect` forward chain priority -250, no missing/foreign/shadow rules, exact ordered IPv4+IPv6 MAC/IP/conntrack/link-local expressions. Only rule handles and numeric packet/byte counters are volatile. Rejects hidden allow, forged source/destination, changed chain hook/policy, extra attributes, malformed/trailing JSON, unsupported ABI, duplicate handles and output above 1 MiB.
- Root-only `NativePeerGuardKernelReadback` calls fixed absolute `/usr/sbin/nft -j -a list table bridge <derived-scoped-table>` through direct argv (no shell, no Host write, no MCP dispatch). Non-root callers are rejected. Unknown/missing kernel outputs refuse to assert success.
- `internal/install/network_peer_scoped_guard.go` now compiles independent `ExpectedKernelJSON` expression vectors alongside bounded nft scripts. `NativePeerGuardPlan` digest is updated to **v2**, binding both script texts and semantic expectation. The earlier plan has never been deployed in the production Root service.
- `internal/install/network_peer_scoped_kernel_ports_test.go`: positive 2-runner/one-stopped tests, cancellation, 12+ negative cases including foreign raw veth, same host_name, missing or fake sysfs link, UUID drift and incomplete inventory.
- `internal/install/network_peer_guard_kernel_readback_test.go`: positive exact dual-stack JSON with changing counters, 18 negative cases for malformed/extra rules, shadow allow, hook priority, bridge, MAC, IPv6, reverse `ct`, DROP to ALLOW, metadata, unsafe table input, extra attributes and corrupted digests.

**LIMIT:** These are required *preconditions*, not proof of an approved original, actor, complete Host kernel firewall state, previously installed guard ownership, recovery journal, global foreign-route isolation or a safe policy mutation. A privileged out-of-band Host root can still race the read-only checks; production must repeat them under durable locks before and after every effect.

## Real nested Incus 7.6 observations

All experiments were inside the L0-owned isolated dev guest `pr178-lab-1123` (UUID `ac585d2c-37ae-4879-a2e6-3cecc338d4d7`), not on L0 Host's network or controller. Four inner guests `pr178-a/b/c/d` Running with `boot.autostart=true` throughout; unrelated C→D Python HTTP + long-lived TCP echo monitor enabled.

| Evidence | Exact durable Operation (guest exit 0) | Result |
| --- | --- | --- |
| Incus host veth → sysfs `brif` inventory | `aba768f41926a11c07daec42044ce078` | Four exact veth names matched A/B/C/D |
| Real kernel positive port audit | `637acba3bb51f444b59fd6a3258bd47e` | `PR178_LIVE_KERNEL_EXACT_FOUR_OWNED_PORTS=PASS` |
| Foreign raw veth negative | `0f35b537d32705cdae9e2711f1cac93a` | Temporary raw `pr178raw0` yielded 5 bridge ports vs 4 owners; correctly denied. C/D 174 HTTP+TCP success, 0 failures |
| Actual nft JSON format captured | `5e53a52262b3046e52a694e90d35a75f` | v1 `metainfo`, bounded bridge table/chain/rules and exact kernel expressions. `nft --check --echo -j` produced no kernel readback |
| Root native nft positive & tamper | `73922f0bd8a74060cfc1599b6ecabc3f` | Exact installed guard JSON accepted; extra manually inserted `accept` rejected. C/D 2585/2585 HTTP/TCP success, 0 failures |
| Cleanup and steady-state | `d72e981b258b64a22e338dc5082d40a7` | C/D final `http_ok=2932, tcp_ok=2932, http_fail=0, tcp_fail=0, reconnects=1`. Temp raw veth and nft table removed; 4 bridge ports, 4 Running Boxes. Test-only privileged Go probes removed from candidate source |
| Full clean source regression | `846272014e7588a47a7caad8f37ca3ca` | `make verify` SUCCEEDED, **guest exit 0**: format (837 tracked), Go mod/vet/race all packages, Python MCP/uninstall/stable/raw-witness suites |

These do NOT establish Root helper boot persistence, Host restart safety, complete forward traffic, C/D IPv6 liveness, unknown TCP revoke before exact conntrack cleanup, ACK-loss resilience or automatic MCP Peer operations. Nested outer AppArmor remains unconfined and is not an enforcing isolation proof.

## Remaining implementation / integration blockers

1. One-time trusted least-privilege Host setup, bounded continuously validated `SO_PEERCRED` mediation, exact original SQLite actor/authorization/idempotency and durable Root intent journal. The current one-original/one-SHA helper still needs per-operation Host staging.
2. Host root must independently collect entire SQLite lease/UUID/ACL claim, all-project native Incus and raw kernel bridge inventory and existing guard rules; **wire** `NativePeerGuardValidateScopedInventory`, this exact live port check and strict nft readback into a single repeated original pre/post verifier.
3. Build per-Box ACL provisioning/migration with scoped UsedBy and initial guard installed *before* any ACL edits. Grant B, A and then guard allow; revoke guard DROP first, precisely purge all A/B dual-stack conntrack scopes (including established TCP), then independent native B/A ACL DROP. No C/D STOP, autostart mutation or unrelated connection loss. A native ACK-lost or kernel mismatch must remain UNKNOWN with held original claims and never blind replay.
4. Prove boot ordering/guard rehydration, saved autostart true/false, Host crash/restart, original journal recovery, foreign NIC and IP/MAC reuse, concurrent grants, C/D long-lived IPv6/UDP/Forward traffic and faulty native filter propagation. C/D IPv6 HTTP baseline in the previous test fixture timed out even without guard: **NOT_TESTED** rather than sign-off.
5. Independently pass installed BoxWeave candidate on real nested Incus, exact-head CI and Owner review. Only then update #177 Doctor and consider v1 merge.

**Decision:** `EXACT_ROOT_KERNEL_READ_ONLY_WITNESSES=TESTED`; `MCP_AUTOMATIC_PEER=NOT_IMPLEMENTED`, `FULL_NONDISRUPTIVE_NETWORK_MUTATION=BLOCKED`. Keep Draft.
