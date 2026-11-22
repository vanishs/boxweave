# v1 one-time Host Forward authorization

Owner-approved post-RC10 policy adjustment. Baseline v1 0ea1feec61829bdbb065a796c10d302119d9d15b. No new native backend, tool, schema migration, live Host deployment or stable release.

## Contract

Trusted service environment BOXWEAVE_FORWARD_EXTERNAL_ALLOW:

| Value | New listener authority |
|---|---|
| Empty (default) | Loopback only (127.0.0.1 default, ::1 explicit) |
| Existing interface/protocol/IP:port entries | Exact assigned host address and port; wildcard entries invalid |
| Single literal all | Caller-selected valid TCP/UDP ports, unique assigned up non-loopback host addresses, explicit 0.0.0.0 or :: |

An operator selects all once at service startup and must separately permit restricted Project proxy devices. The MCP caller explicitly acknowledges each external bind with confirm_external_exposure; this is not independent human authentication. Wildcards can include current/future public interfaces and privileged ports. IPv6 dual-stack behavior is native-dependent. No automatic wildcard default, arbitrary destination, generic proxy device, firewall/ACL change, TLS or connectivity guarantee. Destination remains exact owned Box guest loopback port.

Removing the policy prevents new external creates, not existing native listeners. Explicitly remove existing owned forwards before withdrawing Project proxy capability. No new Host management tool. The original idempotency, native readback and UNKNOWN rules remain.

Either wildcard conservatively reserves the whole same-protocol/port across address families against all non-revoked claims, including pending/uncertain originals. Different protocols and ports remain independent. Native Incus still checks actual foreign/kernel listener conflicts. No replay or forced cleanup of UNKNOWN.

## Evidence

- Isolated development Box 1025aeed-11d1-42bd-af7b-8e76176d06d6, work_ref v1-forward-policy-1121, exact baseline checkout 08de64727ef63c1d3bd8641c9544bce7 exit 0.
- Focused native/state/control tests 2a465acc82cb113e83ac47fd1b32aa1b / ea9b64c3-384c-4d3f-a8a7-46c858739fb0 exit 0 FORWARD_CORE_UNIT_PASS.
- Extended native harness compiled and targeted tests repeated 423b275d74048b928f68492b94889ef0 / 47e42755-4551-4cc3-a59e-08ed97a4d517 exit 0 FORWARD_NATIVE_HARNESS_COMPILED. Guarded native tests not claimed executed by compilation.
- Native acceptance requires legacy exact-mode and broad-mode real TCP/UDP packets including IPv4/IPv6 wildcard on distinct bridge and loopback addresses, arbitrary port, overlapping claims, replay, occupied native listener, cleanup and original fault/dependent Delete regressions, all on isolated disposable CI. No public Internet reachability claim.

Final scoped native and quality review: PASS on tested source; see the evidence below. No merge or stable release authorization. Existing RC10/RC11 acceptance covers previous policy.


## Final scoped functional and affected native acceptance — PASS

Reviewed exact source `154343f150bdcb76203408baa1aae5c32c71d47b`, formatted tree `1564ea4afeabbb8e207257ba1d7bfd57a2dadd6b` (remote tree identical to locally staged Box tree). Candidate binary build-info reports 1.0.0-dev.forward-all and that commit; SHA256 `77debc754eac95e3048c08ebbef4c80566da32f4a5c5c1263ac59a094941c214`, actual build operation `232b5970dc759fb83523ea31c592d4f8` / `24c9b3b5-a351-451f-be1e-50ea37a0fcb9` exit0. This is a source-bound development artifact, not a published RC or stable artifact.

### Real authenticated MCP in isolated nested Incus 7.6

Dedicated development Box s5b-peer-binding-dev-1120 (UUID 1025aeed-11d1-42bd-af7b-8e76176d06d6), no preexisting native guests/forwards, original service ledger with only one terminal FAILED historical receipt. Outer forward list empty tracking `27e9d25f0f18f62e51c17493a901bca1`. Test used a separate private SQLite ledger and temporary local loopback management service with a short-lived test credential. Prior Project proxy policy already allow; no Host policy modification. Original L1 service was temporarily stopped and restored. First local harness original `86960cca4752f30f991c189c15ebe2bf` guest exit1: Python helper parameter collision before any inner Box create or forward admission; empty lab verified, original service restored. Fixed harness in fresh ledger; no retry of uncertain native operation.

Actual final outer Exec `21fe1fe50dddc99e1e1faf89c08a8034` / native `3b6c17f4-b15e-4965-aa57-6b36a5a239a0` exit0, stdout tracking `2849a3be294fcb922976397a0ad7b273`, guest Create `5b1456aa84212baeeb34b8b234012126`, native `68c87f72-9fec-4744-8c79-1f963e04509b`; bounded harmless TCP/UDP echo service setup `81a4b4494851201e4227bb00c6f0202f`. Actual original receipt and native device list verified before independent packet exchanges. All creates/deletes SUCCEEDED:

| Case | Original Create | Original Delete | Real witness |
|---|---|---|---|
| Empty policy / default loopback | ebb171849364337e5cf5c262af9cf674 | ddf363909971ecfc880d4976b40ba972 | loopback TCP yes, bridge no, wildcard refusal |
| Legacy exact host IP/port | f2d122d5279ee27a37d0d4e631e2225b | 65c2591949be3ec8b61681f466ebe5e5 | assigned IP TCP yes, loopback no, wildcard/other port refusal |
| all, arbitrary assigned address port | 6f03a7ca0da56aa397113712999fedf7 | 71c6d323ca2ceb238da479dc55d9766a | assigned address TCP yes without individual port rule |
| 0.0.0.0 TCP | 56f4625ffecd5031521e05a8dd24ed03 | 6c79b773623a84654b22b491913a99e7 | bridge + loopback packets, overlapping bind refused, post-delete packets fail |
| 0.0.0.0 UDP | 098b9ad8b5014c885d56ebe9579b933a | d62602b705281cca64fa919407967463 | same for UDP |
| :: TCP | 12a384ab723c7948c337ea8b8a5f7a6d | a86b5e55019c86e6fdc22183cdbf4261 | IPv6 bridge + loopback packets, overlapping bind refused, removed |
| :: UDP | 05680553649883118defc3f306b6427d | 788766e5b78272872d018e2535582c8c | same for UDP |
| all -> empty service restart | 9d255f3625765acfdd9c10c7fe595006 | 51767e258ad3f07cd5f2c9844025edf1 | existing listener native-verified, new external forbidden, original delete possible |

Both wildcard families refused without acknowledgment. Test guest Stop `d65182d1e591e4719029cf494e5e611e`, Delete `9cd629ddcbe6ac68ea806072cff18721`, fresh token revoked, original service active. Independent readback `4db0c8cf462733a4b5ad612b729f1f86` / `79cfc117-fc76-4372-96fb-5d52fa30d6f2` exit0, stdout `87292d96df9efb5c5cb3c402cf8994d0`: 21 SUCCEEDED original operations, 8 REVOKED forwards, 0 live owned Boxes, native list empty, Project capability unchanged, original service active. No physical L0 changes or Internet reachability claim.

### Independent hosted native CI and source quality — PASS

Exact source [native run 44970743375](https://github.com/vanishs/boxweave/actions/runs/44970743375), job [137049745428](https://github.com/vanishs/boxweave/actions/runs/44970743375/job/137049745428) SUCCESS. Actual log reports S4_B4_EXACT_SOURCE=154343f150bdcb76203408baa1aae5c32c71d47b, missing acknowledgement/foreign address refusal, arbitrary-port and all four wildcard IPv4/IPv6 TCP/UDP REAL_CREATION_REPLAY_CONFLICT_PACKET_DELETE PASS, FORWARD_ALL_NATIVE_WILDCARD_PACKET_AND_CLEANUP=PASS. Existing true exact policy tests, occupied host listener refusal and preservation, real process kill ACK/postread matrix, dependent Delete, four original ACK-loss/protected-partial cases and final native scoped packet/cleanup marker PASS. This uses authenticated Go SDK and real independent native Incus on a disposable GitHub-hosted runner, not mock-only evidence. Source PR runs 44970753662 quality, 44970753676 S1 and 44970753655 dual-architecture candidate all SUCCESS, independently listed in cleanup readback; push quality/build also succeeded.

**FORWARD_ALL_SCOPED_ACCEPTANCE_PASS.** New feature and affected native regression accepted on this source. Historical RC10/RC11 proofs remain scoped to prior behavior; no new source-wide stable signoff, published RC, physical L0 wildcard exposure, merge or deployment authorization. This follow-up evidence-only documentation commit changes no tested runtime/test code; require its final exact-head CI status before eventual merge.
