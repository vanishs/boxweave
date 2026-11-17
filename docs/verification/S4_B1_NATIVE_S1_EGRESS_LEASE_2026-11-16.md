# S4-B1 exact S1 egress / peer grant and IPv4 address-reuse native experiments — 2026-11-16 PT

**Status: S4_B1_PARTIAL.** One disposable Incus 7.6 VM run succeeded for source-exact S1 ingress-vs-egress conflicts and actual IPv4 reallocation, but old-grant access to the replacement UUID was **not confirmed** in that initial run; a follow-up ARP-neighbor diagnostic is being executed. This report records real observations rather than assuming a particular firewall or DHCP cause.

## Authorized topology, precise source, and boundary

- Work uses only GitHub-hosted ephemeral Ubuntu 24.04 Incus 7.6 VM, nonprivileged **sibling** A/B/C native instances. No inner Incus and **no change to the user's shared L0/L1**. The CI runner has disposable host-local admin solely for this test; `authorization.client.tls-restricted` and Scriptlet were not changed.
- Source 2026-11-16 PT [PR #153](https://github.com/vanishs/boxweave/pull/153) commit `4e50d20a2413c9397f5b32433ee2fc2445b11edc`; [run #44008944313](https://github.com/vanishs/boxweave/actions/runs/44008944313), original native job `133790978744` **SUCCESS**, exact `S4_B1_SOURCE` marker independently printed. Installation validates signed Zabbly key and exact Incus7.6, Go SDK v7.6.0 from `go.mod`; Go test and source are independently compiled in job.
- Repeatable [native CI workflow](../../.github/workflows/v1-s4-b1-native.yml) uses same-package **actual existing** `nativePrepareACL` from [S1 source](../../internal/install/native_prereq_prepare.go) through the opt-in [native fixture test](../../internal/install/native_s4_b1_egress_lease_test.go). The resulting egress ACL object is GET/readback compared byte-for-byte in typed rule fields; fixture is not a retyped approximation of S1 policy.
- A = `10.244.85.2` / `fd42:6c0b:5078:e07e:1266:6aff:fe1f:da9a`, native UUID `c4494041-8e10-4668-a7f9-dabf21904b06`. B = `10.244.85.3` / `fd42:6c0b:5078:e07e:1266:6aff:fe57:68f7`, native UUID `75e3d725-df99-488d-8278-60656f50f3cd`. C initially `10.244.85.4` / `fd42:6c0b:5078:e07e:1266:6aff:fe3d:821e`. Native bridge = `10.244.85.1/24` and `fd42:6c0b:5078:e07e::1/64`. Each NIC had one ACL with ingress default reject, IPv4/IPv6/MAC filtering true and a **per-Box** egress override of `allow`, corresponding to BoxWeave `internet=true`. Initial default Profile egress property was explicitly **unset**, corresponding to actual S1 implicit reject.

## Result A: S1 egress policy blocks naive ingress-only Peer authorization

| Experiment on current S1-generated native ACL | IPv4 | IPv6 | Original evidence |
| --- | --- | --- | --- |
| A→bridge gateway, with A `internet=true` | Replies PASS | Replies PASS | `S4_B1_S1_NATIVE_EGRESS_INTERNET_TRUE_GATEWAY_V4V6=PASS` |
| A→B, no shared ACL grant | Denied | Denied | ICMP negative packet probe |
| Add exactly two SDK ingress allow rules, A→B current IPv4/IPv6 | **Still denied** | **Still denied** | `S4_B1_S1_INGRESS_GRANT_WITH_PRIVATE_EGRESS_REJECT_IPV4=DENIED` and corresponding IPv6 marker, while SDK `...GRANT_READBACK=PASS` |
| **Test-only** subtract precisely managed bridge /24 and /64 from S1 private-destination egress deny CIDRs; preserve other deny ranges | A→B replies PASS | A→B replies PASS | `S4_B1_NATIVE_EXPERIMENT_CARVE_BRIDGE_SUBNET_TEST_ONLY=PASS`, `S4_B1_EXPERIMENT_CARVED_S1_PEER_ALLOW_IPV4_IPV6=PASS` |
| With test-only carve, B→A, C→B, A→C | Denied | Denied | `S4_B1_EXPERIMENT_CARVED_S1_REVERSE_THIRD_PARTY_DENY=PASS` |
| Change A's only effective NIC egress default to `reject` (BoxWeave-like `internet=false`), leave peer ingress grant | A→B denied | Not separately tested under switch | `S4_B1_INTERNET_FALSE_PEER_EGRESS_BLOCKS=OBSERVED` |

**Confirmed design bug/risk:** B3 cannot implement peer access as only two destination ingress ACL allows while leaving **S1 egress private CIDR denies** unchanged. S4-B2 must first design an explicit, proven egress policy that retains isolation from all undesired destinations. The bridge-prefix carve was **only a diagnostic mutation on this single-use VM**: although reverse/third-party owned NICs stayed denied, **foreign/unmanaged NICs on the same bridge and cross-Project separation have NOT been proved**. This test-only carve is not approved as a product fix or a shared-L0 host modification.

A further **product semantics decision** is required: should `internet=false` mean no outbound packets including explicitly granted private peers, or no public Internet while approved peers remain reachable? The current S1 default egress reject behaves like the former in this experiment. Do not silently choose an interpretation in B2 code.

## Result B: former destination IPv4 reassigned to a different native instance

- After A→B had worked with test-only carved S1 egress, the runner stopped and **deleted B**. The original A→B ingress rules were deliberately left intact solely to investigate the failure mode.
- The runner then stopped C, set C's exact native NIC `ipv4.address` to B's old `10.244.85.3`, restarted C, and observed the guest receiving **exactly `10.244.85.3`**: `S4_B1_NATIVE_OLD_IPV4_ALLOCATED_TO_DIFFERENT_UUID=PASS`. B's original UUID was known to differ from C's by creation, while the exact C UUID was not emitted in the first run.
- Despite this IP allocation, A→new C ping to the old B IP got 0/2 replies. The log labels the result **`S4_B1_STALE_IPV4_PEER_GRANT_REACHES_NEW_UUID=NOT_CONFIRMED`**, not `SAFE` or `PASS`. A had pinged B just seconds before deletion: cached ARP old-MAC state is a plausible confounder, **not yet verified**.
- Native removal of the two stale ingress rules succeeded with ACL readback; subsequent A→same IP test denied new packets: `S4_B1_NATIVE_STALE_GRANT_EXPLICIT_REVOKE_READBACK=PASS` / `S4_B1_IPV4_REVOKE_NEW_UUID_PACKET_DENY=PASS`. However, if packets were not working before revoke, this alone cannot prove the revoke caused the denial.

**Follow-up:** [fresh Action run #44009432605](https://github.com/vanishs/boxweave/actions/runs/44009432605) at code commit `d5c85b24fec75133fd8dc1c77974e2576303a791` adds guest ARP-neighbor before/after evidence, independent exact C UUID/NIC readback, and old-neighbor flush (or expiration) before probing the stale grant. **Do not upgrade Result B to confirmed unauthorized reachability or confirmed safety before reading that job's original results.**

## Safety, cleanup, and next steps

- Run #44008944313 also reexecuted the earlier native SDK 7.6 ETag 412, two-family ingress grant/revoke, TCP/UDP proxy, collision and actual cleanup regressions. Those **PASS** results apply to the deliberately egress-open **first** temporary fixture, not the exact S1 fixture in the second job step.
- Final exact native cleanup of the second fixture independently verified both surviving native A/C deleted, default Profile ACL detached, dedicated ACL removed, zero instances: `S4_B1_S1_EGRESS_LEASE_EXPERIMENT_EXACT_CLEANUP=PASS`. GitHub-hosted disposable VM is destroyed after the job.
- None of this grants authority to modify shared L0 ACL or Scriptlet. No S4 network public Tool was added; S4-B2–B6 have not started. Safe B2 coding can proceed on claim/schema/locking and refusal logic, but real peer enablement remains gated on **source egress resolution**, foreign topology tests, IP lease reclaim protocol, kernel reload/fault and Scriptlet rights.
- `S4_B1_PARTIAL` remains; report limitations separately rather than relying on the job's green result as security-product PASS.
