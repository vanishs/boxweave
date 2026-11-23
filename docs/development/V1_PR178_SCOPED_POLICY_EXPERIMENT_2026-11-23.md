# PR #178: isolated per-Box Peer ACL compiler and Incus 7.6 research (2026-11-23)

**Status: EXPERIMENT + TESTED PURE COMPILER; NOT APPROVED FOR PRODUCTION NATIVE DISPATCH.** Authoritative owner acceptance remains [2026-11-23 decision](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md). This is a source-level component, not an automatic MCP network grant or a replacement for the RC13 full-bridge safety barrier. Do not merge PR #178 yet; #177 remains frozen pending final architecture integration.

## New source

- `internal/install/network_peer_scoped_policy.go` exports `NativePeerScopedACLName` and `NativePeerScopedACLFromPolicySnapshot`. Deterministic ACL naming is scoped by *Project, bridge and immutable exact UUID*; a name is NOT an ownership proof and must never authorize adoption of an existing ACL.
- Inputs are an independently native-verified owner UUID and canonical IPv4/IPv6 leased identity, plus complete durable confirmed/revoked directions. The compiler first validates the **entire global snapshot** through the previous strict whole-policy validation, then takes only the owned participant's pairs. A stale or forged owner IP, unrelated invalid row, duplicate IP, reverse live grant conflicting with a revoke, or invalid boundary fails closed.
- Reuses existing bridge-reserved IPv4/IPv6 egress deny, exact directional allow rules, and symmetric revoke DROP tombstones. An A→B direction changes **A's and B's desired policies**, not the C/D desired policies; no native writer, MCP handler, installer migration, Root helper delegation or cross-ACL transition is included in this commit.
- `internal/install/network_peer_scoped_policy_test.go` proves four-owner A/B/C/D with a separate C→D grant, unchanged C and D after A→B grant/revoke, restricted other-private-address carve-outs, dual-stack DROP tombstones, explicit malformed global/owner refusals, and deterministic independently scoped ACL naming.
- The existing `nativeACLWithDeniedPeers` and shared-ACL native pipeline are deliberately unchanged. **Do not substitute the scoped compiler into the old root Apply with only a search/replace.**

## Repeatable disposable Native evidence

Environment: L0-managed **disposable** `pr178-lab-1123` (UUID `ac585d2c-37ae-4879-a2e6-3cecc338d4d7`), Go 1.26.9, signed Zabbly Incus 7.6 (`1:7.6-ubuntu24.04-202611072126`) installed in the *guest only*. Its own `incusbr0` IPv4 `10.152.98.1/24`, IPv6 `fd42:1530:6db:ee51::1/64`. Inner instances `pr178-a/b/c/d` all kept Running and `boot.autostart=true`; inner per-NIC ACLs `pr178-a-acl/b-acl/c-acl/d-acl` have single-instance `UsedBy`. The physical L0 system, installed L0 controller, other outer Boxes and their options were NOT mutated. AppArmor outer lab is `unconfined`: **enforcing nested security is not established here**.

| Fixture | Native IP | Test role |
| --- | --- | --- |
| A | `10.152.98.160` | Client, A→B ongoing HTTP/TCP |
| B | `10.152.98.190` | HTTP IPv4+IPv6 and TCP echo |
| C | `10.152.98.67` | Unrelated client and independent HTTP + persistent TCP monitor |
| D | `10.152.98.185` | Unrelated HTTP and TCP echo service |

Exact operational witnesses (each `box_exec` has a separately observed `Operation SUCCEEDED` **and Guest exit=0** unless described otherwise):

1. Targeted compiler tests `a10fef193e3830036f225d4b6a453668`: all three new tests and negative cases PASS.
2. Native compiled A/B baseline `308ae1b126519336f4ee2d335b3c0218`: each 0 ingress + 2 reserved-address egress rejects; both ACLs independently ETag-written and read back. A→B and C→B IPv4 denied, C/D HTTP and persistent TCP healthy (`4d31e3ee910a211fcd7f0de43e2dccae`).
3. Native compiled A/B grant `20a1e5a2d0f66400e1602583957a26f0`: update B then A, each 2 dual-stack ingress allows + 4 egress rules; original IPv4 A→B HTTP 200. IPv6 A→B HTTP 200, while C→B IPv4+IPv6, A→C and B→A IPv4 remained denied (`f0291efd584267fcc02810a4fd5e9e41`). C/D monitor `512/512` HTTP/TCP successes, zero failures and one initial TCP connection.
4. Native compiled A/B **revocation DROP only** `8c736c20736906e5af1e7ab35c778a38`: B then A, 4 dual-stack ingress DROP + 6 egress rules, independently ETag-read back. **New** A→B IPv4/IPv6 attempts timed out, *but existing persistent A→B TCP did NOT immediately stop*: `tcp_ok` increased from 104 to 109 then 116 while the A→B HTTP probe failures accumulated (`634f430d3aa10675de3351aa35ff42ba`). This is an independently reproduced **fail-to-revoke-established-flow**, not a success.
5. Research-only exact conntrack test `36900dc67886f25b178ac783b5b3f7bd`: validated DROP on **both affected** A/B ACLs, exact native UUID + IPv4/IPv6 and exclusive `UsedBy`, then used the EXISTING `nativePeerConntrackPurgeExact` root-only bounded implementation. Verified `ProbedScopes=4, DeletedScopes=1`, Guest exit 0. No `conntrack -F`, global flush or C/D entry deletion. Subsequent `462419beb52d82de1d14e99b05b5b5c9`: old A/B `tcp_ok` remained 149 while failures incremented, C/D HTTP/TCP successes grew from 1727 to **1786** with **zero failures, one initial connection**; all four Running. **Research-only** Go probe and the exact privileged test were copied into guest-local `/root/pr178/scoped/research` and removed from commit source. This is NOT a Root Helper production authorization proof.
6. Clean candidate source after removal of root probes: full `make verify` original `e83cc698f6fe49751513b36b5e82f562`, Guest exit **0**. Full Go race suite (including state), vet, format, YAML, module checks, Python MCP/uninstall/stable-gate/raw-witness PASS. **No candidate daemon was installed** in the nested Incus, and this is not an installed-MCP or physical-Host reboot proof.

## Safety/availability gates still BLOCKED

- **Transient ACL fail-open and atomicity:** Incus 7.6 updating A and B remains two independently applied native effects. A failed bridge filter render, async fanout, ETag ACK loss or crash may temporarily disable A/B filtering and expose unauthorized A→C/C→B flows. Normal packet tests **do not** establish a failure-proof kernel guard. A fixed root-owned fail-closed enforcement anchor, complete kernel verification and exact rollback/UNKNOWN journal are needed before executing scoped updates without a stop barrier. Reject unsupported instead of secretly invoking the old all-Box STOP.
- **Existing TCP revoke:** adding symmetric DROP and observing ACL config is insufficient; the **exact** A/B conntrack purge and post-kernel verification must be part of the operation. New connection denial must coexist with previously established session closure while C/D sessions survive. The root-only `NativePeerConntrackPurge` production wrapper currently requires **all** bridge consumers stopped; research test bypassed ONLY that wrapper's all-stop gate, *not* root or exact pair validation. Develop and independently prove an A/B-only recovery fence, not a production bypass of this precondition.
- **Native installation/migration:** current installer creates one shared ACL and one fixed Project default profile. Real per-Box ACLs need root-scoped provisioning, exclusive ACL ownership/UsedBy proof, stable deterministic resource claim, migration with explicit review and no destructive changes to existing Boxes. Lab used DHCP and permissive default egress, not immutable leased addresses + anti-spoof + `internet=false` correctness.
- **Host automation:** root helper still requires per-original Host staging and digest approval; one-time trusted least-privilege installed mediator with independent actor, original/ETag/journal checks and safe recovery is not implemented.
- **Autostart preservation:** prior commit defaults newly created Box to `true`, but old Peer barrier still overrides *all* consumers to `false`; explicit unset/true/false preservation through Peer and restart is not implemented.
- **Other untested:** IPv6 UDP/real Forward sessions, no-leak partial mid-ACL failures, reboot/restart, API ACK loss, concurrent grants, foreign bridge consumers, IP/MAC reuse, host-wide native identity witness, real installed OAuth MCP, tested candidate per-Box migration and independent final review. Mark NOT_TESTED, not PASS.

## Next safe code sequence

1. Build **immutable dual-stack per-Box policy attachment plus host-side baseline guard** in the disposable nested Incus, with a negative packet observer continuously probing unauthorized A→C/C→B across *every* partial ACL effect and rollback fault. Preserve C/D's long-running HTTP, TCP, Forward/IPv6 where available. Do not expose any new writer until a kernel-side fail-closed invariant is proven.
2. Build **A/B-only conntrack and journaled original dispatcher**, never a general root shell or implicit per-Operation Host steps. Must atomically prove the complete original outcome or retain UNKNOWN and locks.
3. Implement installer + one-time trusted Host mediation, MCP dispatch, migration and autostart-preservation; test full final head via installed Native, fault matrix, CI and explicit owner acceptance before v1 or #177 updates.
