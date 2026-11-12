# S2.5 exact-head Box-first and final native Actions report — 2026-11-12

**Overall: targeted expanded LAB PASS, formal S2.5 LAB_PASS = NO; HOST_GO = NOT_AUTHORIZED.** This distinction is mandatory: successful source-bound tests do not attest to an untested client, physical host, or missing fault matrix.

## Exact source and downloadable evidence

- Draft PR #133, branch feat/v1-s25-native-readiness-20261111, exact tested source 7a01000a12114eba0fe4937af0e955bd6e46ed87.
- Ubuntu 24.04 one-use GitHub-hosted native host, signed Incus 7.6 APT package 1:7.6-ubuntu24.04-202611072126; separate L0 and L1 Incus daemons, restricted users/Projects, private bridges, distinct SQLite and credentials.
- Exact installed candidate v1 SHA-256: 26a72b432c7ec6f3fd43721feb86012caa0800d388a363dcbf6b91311ecde584. Independent SDK Agent binary SHA-256: c3618f75f6e0f4e859aa02850b96ac8a5f2b284b671200b1441c4d9c0d272a58. Nested installer script SHA-256: b4cf52fb5f57fd8bc7381384e3ada9630178c7935ba86256f8c160a8714720cb.
- Native MCP LAB: https://github.com/vanishs/boxweave/actions/runs/43189054651 — SUCCESS.
- Original native S2 regression and S2.5 preflight: https://github.com/vanishs/boxweave/actions/runs/43189054514 — SUCCESS.
- Source S1/race: https://github.com/vanishs/boxweave/actions/runs/43189065574 — SUCCESS.
- Required quality gates: https://github.com/vanishs/boxweave/actions/runs/43189065348 and https://github.com/vanishs/boxweave/actions/runs/43189054955 — both SUCCESS.
- Evidence ZIP: https://github.com/vanishs/boxweave/actions/runs/43189054651/artifacts/13930306740 . SHA-256 909166e1f26e5a52d2f4348d3535c1893b3fe437c7803f6822f7324acbb038dd. Independently opened archive contains doctor.json, console.log, identity.txt, native-prereq.json, outer-checkpoint.json. Source and binary digest match exact identity.txt.
- All eight operator native prerequisite checks passed; doctor native READY. DIR_QUOTA_UNVERIFIED and DIR_POOL_HOST_ROOT_SHARED were reported honestly. Per owner decision, NO new S2.5 disk-quota pressure test and no asserted hard limit.

## Original-operation and independent verification evidence

- L0 authorized SDK MCP created nesting-enabled L1 original Create Operation dd7ffca0a23ab14d883412623399e30e, native UUID a2d420af-0bb6-4170-89c2-cee8597d745d. Agent A disconnected; separate Agent B recovered the original receipt/UUID, and replay with the identical key did not redispatch; changed intent under the same key was denied.
- Two authenticated Agent processes independently executed, and a separately owned peer Box was created and deleted with native and SQLite original IDs. Quiet Exec original 005f6c44835931f8225eb88b612b84e3 returned a real transient VERIFYING state on a 150ms bounded read, remained uncancelled, then provided its original persisted output; independent SQLite verifier passed.
- Deliberately restarted L0 v1 service; bounded real HTTP 401 startup readiness observed, all original operations and native identities remained stable, no repeated Create. Revoked bearer returned 401; a rotated Viewer could read but could not mutate. Invalid path, Host, Origin and Bearer negatives passed.
- L0 original Guest Exec bootstrap 4322b18eef49a583a1bc6e23a0213e11 installed independent signed Incus 7.6 and the exact same v1 binary in the MCP-created L1. L1 official SDK MCP created L2; original Exec exit 0/nonzero, output, same-UUID Update(cpu=2), Stop, Start, post-restart Exec and final Delete were confirmed independently from nested SQLite and native Incus. The bounded positive IPv4 AND IPv6 gateway packet witnesses passed; configured anti-spoof/ACL was read back.
- BH-014 real source repair used L0 MCP box_exec in L1: initial Git commit, original failing unittest Exec c3e78359e34fd5c0eefe73e30885c146 with recorded nonzero exit and persistent stderr AssertionError 5 != 4; repaired Git commit via original Exec 2d3d3580752d759d8005dab985274c02 passed test. Independent guest-native Git and host SQLite verifier confirmed two commits, clean tree, final pass, same original owned Box. **BH-014 I+N+V PASS on this exact candidate.**
- L1 MCP deleted L2 using original child Operation; separate V witnessed native absence and original SQLite Delete. L0 MCP stopped and deleted L1 using original Delete 5d703260d0051550f47e125b98159461; V confirmed only its exact UUID absent and durable SQLite original. All test-owned boxes cleaned; foreign native resources and three historic UNKNOWN receipts untouched.
- Prior Box-first probes on retained, isolated older-test L1 established legitimate native IPv4/IPv6 gateway and actual public HTTPS, peer/foreign denial, IPv4/IPv6 source spoof and MAC spoof denial, and post-start RA route convergence after the initial IPv6 no-route period. These are true isolated native witnesses **not exact-head MCP packet proofs**; all temporary A/B/Git/RA native test boxes were independently cleaned.

## Mandatory case disposition

T = source test; I = installed v1 MCP/host-local CLI; N = actual native Incus or network; V = independent verifier. PASS below means only that the entire stated case was directly witnessed; partial witnesses are not counted as full cases.

| Case | Disposition | Exact evidence / remaining boundary |
|---|---|---|
| BH-001 | PARTIAL | Prior S1/S2 accepted, exact source and hashes, 13 SDK Tools; full host-local CLI/version/UNKNOWN inventory crosswalk still incomplete |
| BH-002 | PARTIAL | One-use host identity, signed Incus 7.6, dir pool/warnings and owned resources observed; full physical capacity/independent baseline is not approved |
| BH-003 | PARTIAL | Reviewed wrong-SHA refusal, native prepare apply, 8/8 prerequisites, 401 MCP, foreign Project denial; full independent rollback and all native pre-first-Box negatives not replayed |
| BH-004 | PARTIAL | Same SQLite restart/recovery and prior operator local rollback evidence; same-head interrupted update/compatible fallback or fail-closed exact retention not fully independently repeated |
| BH-005 | PARTIAL | Official Go SDK protocol, 13 tools, authenticated writes, wrong Host/Origin/path/bearer and revoked Viewer denials; actual intended development Agent path unverified |
| BH-006 | PARTIAL | Real nested MCP Create/Get/Exec exit 0 and nonzero, Update/Stop/Start/Delete and native UUID poststate PASS; full Box list and guest Ready coverage not separately certified here |
| BH-007 | PARTIAL | Same-key changed-intent rejection, original no-replay; extensive lost ACK/UNKNOWN native S2 suite passed exact source but full nested combined negative matrix incomplete |
| BH-008 | PARTIAL | Two processes, original checkpoint and SQLite after restart PASS; forced worker SIGKILL in nested chain and its independent V replays incomplete |
| BH-009 | BLOCKED | Native two-guest v4/v6 peer/source/MAC packet denial and NAT positive on older isolated L1; exact GitHub LAB v4/v6 gateway packets positive. Exact same-candidate direct public WAN and complete source-spoof packet matrix not established; scoped package relay is not a WAN witness |
| BH-010 | PARTIAL | Exact L0 MCP to unprivileged nested L1 with independent v1/Incus to L2 lifecycle and durable V PASS; full idmap/namespace/NAT/pressure proof incomplete. No hard quota guaranteed |
| BH-011 | PARTIAL | Project foreign denial, L0/L1 loopback isolation, viewer mutation denial; full forged UUID/reused-name and cross-layer token/socket hostile negatives incomplete |
| BH-012 | PARTIAL | Quiet-running native Exec, 150ms wait, persisted original stdout, no cancellation PASS; native-unavailable/stalled-client/timeout distinctions not all replayed nested |
| BH-013 | PARTIAL | Box CPU/memory/disk/inode/PSI baseline, source race and native S2 fault regressions passed; full bounded nested load/timeout matrix incomplete; quota stress deliberately waived |
| BH-014 | **PASS (I+N+V)** | Real L0 MCP Guest Git failing regression, persisted exact failure output, repair commit and passing test, independent native Git and SQLite attestation on same v1 candidate |
| BH-015 | PARTIAL | Independent V confirmed exact binary, SQLite UUID/ACL, L2 lifecycle, packets and both layers cleanup; same-head independent reproduction of all fault, anti-spoof, rollback and UNKNOWN cases remains incomplete |
| BH-016 | PARTIAL | Exact MCP L2/L1 native and SQLite absence/cleanup PASS; full operator fallback and all retained-state reconciliation are not separately signed |
| BH-017 | NOT_AUTHORIZED | Exact standalone physical L0 not selected/approved; no physical install, no legacy main co-installation |
| TC-011 | PARTIAL | Two actual independent authenticated SDK clients and per-Agent original receipts PASS; full contention/correlation and hostile actor filters incomplete |
| TC-012 | PARTIAL | A disconnected after admission; B recovered original and no replay after controller restart PASS; forced kill/lost ACK variants in same nested chain incomplete |
| TC-013 | BLOCKED | Official SDK test clients exercised v1, but the actual intended development Agent's own credential/transport path to v1 is not connected or tested |
| TC-014 | PARTIAL | Stall-ish bounded wait, token revoke/rotate and original receipt survives; full simultaneous blocked UNKNOWN/token race negatives not repeated nested |

**Gate accounting:** 1 of BH-001–BH-016 is individually fully signed with exact I+N+V in this scope; 15 remain PARTIAL/BLOCKED. BH-017 remains NOT_AUTHORIZED. None of TC-011–TC-014 has complete formal signoff. This does not undo the targeted 5/5 Actions SUCCESS and must not be described as LAB_PASS.

## Safe continuation

Next non-waived work is to finish the distinct BH/TC negatives in disposable native hosts, especially exact-head native WAN/sibling/foreign/anti-spoof packet tests, crash/UNKNOWN/fallback independent replays and actual intended Agent credential transport. Do not relax ingress/ACL, repeat UNKNOWN, run unsupported quota pressure, coinstall old main, install on a physical host, mark HOST_GO, merge PR #133 or release v1 until explicitly authorized phase gates pass.
