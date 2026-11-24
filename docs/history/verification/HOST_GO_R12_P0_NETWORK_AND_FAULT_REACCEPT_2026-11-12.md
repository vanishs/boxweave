# HOST_GO Round 1/2 P0 safety and recovery reacceptance

**Targeted three P0 gate groups: LAB-SCOPED PASS with transparently
identified controlled-fault/simulated-WAN equivalents. Five source-exact
Actions on 457c6dc completed SUCCESS; formal all-BH/TC LAB_PASS and
physical-host HOST_GO are still NOT signed. Round 3 NOT_STARTED.**

## Exact provenance and negative chronology

The independent Box-first source under review is Draft PR #133.
Exact completed evidence source is
457c6dcf8f95feba64ce4d8ae936cbcf12297b92, following the
5ac7680 private-network fix, 7c80db8 root-local preflight correction
and 59bdf00 controlled native ACL ownership diagnostics.
All five Actions at this exact source are complete SUCCESS.
The one-use GitHub CI uses signed Incus 7.6 on Ubuntu 24.04,
isolated native Projects/bridges, separate L0 and nested L1 daemons,
authenticated installed v1 MCP and distinct agent identities. Each
installed binary SHA must be taken from that workflow's actual identity,
not guessed from the Git SHA.

1. At test commit 79804ff (native run 43359387426), the independent
   nested L1 foreign-Project guest actually replied to a routed IPv4 ICMP
   Echo sent by an internet-enabled owned L2 guest (10.179.98.182,
   successful two Echo replies). That is **P0 real product network
   isolation failure**, not an environment limitation or a simulation
   PASS. The original dedicated ACL was empty; internet=true changed the
   NIC egress default to allow, inadvertently allowing routed RFC1918.
2. The first proposed fix bf97ba1 added gateway allow and private reject
   but incorrectly assumed that rule order could override native reject
   priority. Incus rejects traffic matching a reject even when an allow
   appears before it; gateway connectivity regressed. This fix was
   rejected, not signed off.
3. Commit 4dfb631 subtracted exactly the selected native gateway /32
   and /128 from disjoint private IPv4 and IPv6 ULA rejected CIDR lists,
   and compared product Go / standalone Python generator SHA256 byte
   output. However, an extra fe80::/10 reject killed native IPv6
   router advertisements/default route, causing regression CI 43365572305
   to fail. It was correctly recorded as a **failed safety gate**.
4. An independent native Incus 7.6 diagnostic used a NEW temporary
   project bw-r12diag, bridge bw-r12diagbr, ACL bw-r12diagacl, Guest UUID
   dc4be29b-1c94-4b16-b26a-086c54b8b76f. Empty ACL gave real IPv4
   and IPv6 gateway Ping positives. After adding disjoint private IPv4
   and ULA rejects, both gateways remained reachable; adding only
   fe80::/10 reject removed global IPv6/default RA route and gateway
   Ping failed with Network is unreachable; removing only the fe80
   reject restored router advertisement, IPv6/IPv4 gateway Ping positives.
   Native ACL and packet readbacks were saved before cleanup:
   evidence/r12_native_diag_acl_precleanup.json SHA256
   32eb085a50a41b86163930ecb07cd7ff1b2289efd3c765241e683308d61f5ad3
   and evidence/r12_native_diag_gateway_positive.txt SHA256
   baa8d70d6a8e677a6daeb16c18286e82568e36fa52ae2d285d81cff0c621f0a9.
   Test Guest, bridge, Project and ACL were ALL deleted by exact
   identity and verified absent; unrelated demo-1-container remained
   RUNNING. Original parent L1 bw-s25-l1a and original services untouched.

The network fix at 5ac7680 preserves link-local fe80::/10 for IPv6 RA/NDP,
rejects other routed ULA and RFC1918/CGNAT destinations with gateway-only
IP carveouts, tests against broadened/empty/mutated ACL, and verifies
current installed native ACL at the pre-first-Box acceptance gate.
No explicit allow rule overrides rejects. Global-unicast foreign
Project networks require host-specific reacceptance, not blanket PASS.

5. At 5ac7680, nested native run 43367374437 **SUCCESS**: genuine
   cross-Project IPv4 and IPv6 packet negatives with positive controls,
   independent inner/outer socket verifier, exact foreign Guest UUID
   cleanup, original MCP operations, second-Agent real lost HTTP reply,
   token revocation and real same-binary offline rollback were accepted.
   The separate original S2 real native run 43367374427 completed its
   historical native crash/ACK/UNKNOWN/SQLite/ENOSPC suite PASS and
   successfully created/read back the hardened ACL, but the *new*
   operator native prerequisite review initially marked ACL=false.
   Independent root-vs-restricted Incus reads proved the underlying
   cause: a successful restricted Project network GET masks bridge
   Config to an empty object, whereas root-local GET contains real
   ipv4.address and ipv6.address. Comparing the masked response to the
   exact gateway-carved policy is a false negative. The source-only fix
   at 7c80db8 makes root-local prerequisite inspection read both the
   real default-network bridge and exact ACL through the already
   privileged Incus SDK client; there is no change to installed MCP,
   restricted OS service, tokens, native Guest or Project permissions.
   Independent local exact-source make verify and Go race PASS.
   The 7c80db8 nested native run
   [#43368636623](https://github.com/vanishs/boxweave/actions/runs/43368636623)
   is **SUCCESS**; independently downloaded evidence artifact
   [#14011269653](https://github.com/vanishs/boxweave/actions/runs/43368636623/artifacts/14011269653)
   has ZIP SHA256
   84bd2054d4fa72f48c7f11c06c05689cafeba6764e00f4d467dca42da40cf932
   and records exact source 7c80db8, original candidate binary SHA256
   73086964157f45d5fa45d025ec6d035cad9bbb9c18220f965003cf3638099e9b,
   real cross-Project owned Guest UUID
   e4d9e447-3f21-429d-ad42-83c5b78cd38a and foreign Guest UUID
   22cd3a7e-6e44-4a91-a9ea-0a9c479542c6. Its independent L1
   v4/v6 packet and parent Socket proof, exact UUID foreign cleanup,
   lost MCP reply and client rotation, real controller SIGKILL,
   separate offline rollback and exact ledger receipts all passed.
   This is bounded LAB proof and not physical-host evidence.
6. 7c80db8 **S2 native** run
   [#43368636569](https://github.com/vanishs/boxweave/actions/runs/43368636569)
   passed the original entire Incus 7.6 MCP/SQLite/SIGKILL/lost-native-ACK,
   safe recovery/no-replay, bounded ENOSPC and legacy simulated WAN suite.
   Yet the final hardened ACL preflight remained BLOCKED, even with root
   network readback. We therefore **did not weaken** the safety gate.
7. Added a standalone root-native diagnostic at 59bdf00. The failed
   [#43372141160](https://github.com/vanishs/boxweave/actions/runs/43372141160)
   proves exact persisted ACL CIDRs, action/state and description all
   correct, while the used_by list contained *two separate Profiles*:
   /1.0/profiles/default?project=user-1002 and
   /1.0/profiles/default?project=user-1003.
   This is a genuine safety preflight FAIL: the historical S2
   dual-controller script inadvertently gave the second
   controller **the first Project's exclusive ACL**. An independent
   disposable Box-first native ACL API roundtrip and Profile
   used_by test both passed and were fully cleaned up; they did not
   bypass this observed cross-Project fixture error.
8. Source 457c6dc fixes the **test fixture**, not the production
   exclusivity policy: the second Project now creates and binds a
   distinct bw-s2-dualctrl-peer-acl and a separate root-local Verifier
   requires both exact Project used_by ownership sets to be disjoint,
   with a dedicated Actions assertion. Existing S2 controller token,
   guest UUID/operation isolation and native ingress rules are retained;
   the pre-first-Box root ACL exclusivity guard stays mandatory.
   Local Box make verify, go vet, formatting and full race suite PASS.
   The complete original S2 native test on this exact commit is now
   SUCCESS: its owned ACL root-local used_by contains only the original
   user-1002 profile, both per-controller ACL ownership assertions
   pass, and the strict prereq digest/drift checks pass without bypass.

## Final exact-head native Actions and independent archive audit

Exact source **457c6dcf8f95feba64ce4d8ae936cbcf12297b92**, five
completed SUCCESS workflows:

- S1/race/source: https://github.com/vanishs/boxweave/actions/runs/43372818101
- Two quality gates:
  https://github.com/vanishs/boxweave/actions/runs/43372818066 and
  https://github.com/vanishs/boxweave/actions/runs/43372809733
- Full original S2 real Incus 7.6 faults + S2.5 exclusive ACL preflight:
  https://github.com/vanishs/boxweave/actions/runs/43372809632
- Same-commit nested L0->L1->L2 MCP/Incus/independent Verifier:
  https://github.com/vanishs/boxweave/actions/runs/43372809750

Nested evidence artifact:
https://github.com/vanishs/boxweave/actions/runs/43372809750/artifacts/14012879072.
Downloaded and independently opened ZIP SHA256:
5a5cf57506176183b2e8a41da53cbcedc8305eb38e771c631537e3121862c2c3.
Its identity.txt declares source 457c6dc, original installed candidate
L0/L1 binary SHA256
f8d6487b607b5fc234ffd7664bae04e7dffb8d63ae93676b29136341e76feca2,
official SDK dual-Agent fixture binary SHA256
49c3c9aff20e7b39603bfc8af691afbc2e1d96912f469045cb8d27a44299ed17,
and signed Incus APT 1:7.6-ubuntu24.04-202611072126.
The independently read foreign negative fixture includes exact original
native owned UUID 6cce0b5f-4c19-477c-a033-139243260296, foreign
UUID 83d317e4-27fa-4674-b16b-383babfe4d9d, positive gateway IPv4/IPv6
and denied bidirectional cross-Project packets, blocked foreign Project
API and absent parent socket mounts; final fixture state CLEANED with
the same two original UUIDs. Installed MCP lost-reply/Agent B retrieval,
real PID SIGKILL, original SQLite/native receipts, offline exact-binary
rollback and exact-cleanup markers were independently inspected.

The full S2 native fault Action 43372809632 has the original in-flight
worker SIGKILL, native Create/Update/Stop/Delete lost ACK, native 202 +
SQLite fsync before/after, isolated ext4 ENOSPC, manual UNKNOWN
review/no-redispatch, independent SQLite/native witness and cleanup
markers all PASS. Its root-local post-suite actual ACL UsedBy was
**only /1.0/profiles/default?project=user-1002**; the peer Project's
separate ACL proof and stale digest/Project-drift refusal were PASS.
Earlier failed native Actions and P0 vulnerability remain in the
chronology above, not counted as successful runs.

The actual owner-approved Codex CLI development Agent was separately
proven at source 0cf474f in HOST_GO_CODEX_REAL_AGENT_R2_2026-11-12.md,
with model-driven native Create/Start/Stop/Delete and separate snapshot V;
this is a retained functional crosswalk, **not** a claim that Codex CLI
personally invoked the 457c6dc ephemeral CI candidate. The current
two-Agent original fault matrix uses distinct authenticated official SDK
clients. Public IPv4 was native TLS, separately routed IPv6 WAN was
SIMULATED TLS (as approved for S2.5); there is no genuine public IPv6 or
physical-host power-loss witness. DIR_QUOTA_UNVERIFIED remains disclosed.

## Distinct negative/positive packet scopes

The historical S2 real Incus fault suite uses a separate *RFC2544
198.18.0.2* routed simulated HTTPS endpoint. That address is inside the
new hardened S2.5 prohibited private-style CIDRs. Do not weaken the
product ACL to make the old simulator work. The original S2 fixture
executes all its established native/ACK/UNKNOWN regression tests
first, with its original private NIC/ACL semantics; after exact owned
native cleanup, the NEW S2.5 private-route ACL is prepared and checked
on the now-empty disposable project. The S2.5 installed native MCP
L0->L1->L2 suite separately tests actual IPv4 public HTTPS and isolated
routed simulated IPv6 TLS, and the new L1 cross-Project v4/v6 real
packets with bidirectional positives and negatives. These are not
mistaken for each other's results.

## Mandatory L0/L1 actual native packet/security witness (BH-009, BH-015)

The independent one-use L1 script creates a foreign Guest in the
disposable inner daemon default Project, separately from the v1-managed
user-1001 Project and never on shared Incus. It checks actual source
addresses and both selected gateway IPv4/IPv6 positive controls, actual
bidirectional IPv4/IPv6 cross-Project ping denial, effective dedicated
ACL reject-only CIDRs excluding exactly the gateway, owned used_by,
native parent-vs-inner Unix socket inode separation, no parent socket
mount and denied nonroot foreign Project access. Outer independent
Verifier repeats real packet tests and verifies exact UUID cleanup.
Prior same-source mounted MCP suites additionally cover public IPv4,
routed simulated IPv6 HTTPS, per-NIC toggles, sibling IPv4/v6 and
source/MAC anti-spoof.

## Fault and two-Agent evidence (BH-007/008/012/015/016, TC-011/012/014)

The one-use nested installed MCP suite drives two separately authenticated
official SDK Agent sessions with distinct Agent/work labels, independent
native resources, denied same-name conflicting create and no privilege
gain from a borrowed work_ref or Agent label. The original Agent sends
one real Box Exec; only after the installed candidate returns its HTTP
reply does the test client deliberately discard the ACK. The Agent
never retries or manufactures a receipt. Independent Agent B uses
the authenticated operation_list correlation to discover exactly one
original, reads its persisted UUID, original Exec exit/stdout and
native SQLite evidence, then re-reads that same original after revoking
A's credential (401). This establishes loss of **MCP reply**, not loss
of the upstream Incus ACK.

The same-source native S2 regression separately injects actual accepted
Incus ACK loss, operation postread failure, worker restart/SIGKILL,
SQLite WAL fsync before/after crash and bounded ext4 ENOSPC faults.
Original UNKNOWN and native operation IDs must be retained without
automatic re-dispatch. The nested lab independently tests original
two-Agent reconnect/recovery, candidate-only SIGKILL with distinct PID,
and local bad-digest/no-backup upgrade refusal and offline rollback,
original SQLite and native cleanup. Controlled fault proxies are
expressly equivalent LAB simulations and are not physical-host
power/network failure evidence. The prior actual Codex worker
in-flight SIGKILL watcher failed to hit; do not relabel it PASS.

## Targeted three-group exit register

| Targeted group | Verdict | Exact witness |
| --- | --- | --- |
| BH-009/BH-015 P0 foreign network and parent socket | **LAB-SCOPED PASS (I+N+V)** | L1 root-created foreign Project, real v4/v6 bidirectional packet denials, positive gateways, exact owned ACL, separate parent/native socket, independent verifier and UUID cleanup on 457c6dc |
| BH-007/008/012/015/016 fault/rollback submatrix | **LAB-SCOPED PASS with controlled equivalent fault triggers** | Native accepted 202/ACK-drop, forced worker SIGKILL, real SQLite WAL/ext4 ENOSPC, UNKNOWN never replayed, independent original V, offline canary/rollback |
| TC-011/012/014 authenticated two-Agent recovery submatrix | **LAB-SCOPED PASS** | Two separate official SDK authenticated sessions, original work/Agent ref, lost MCP HTTP reply only after native completion, exactly one original Exec, no resend, revoked A returns 401, B re-reads original/native V |
| Same-head final quality/native Actions | **PASS 5/5** | 457c6dc, five run links and downloaded original SHA-verified artifact above |

These three *specified gaps* have been closed under the owner's allowed
isolated-LAB equivalents. This is **not** retroactive blanket signoff for
all BH-001–016/TC-011–014 entries or BH-017, and not permission for
physical-host deployment. The aggregate first/second-round formal case
crosswalk and individual exact-host readiness must be reconciled under
the runbook before claiming overall LAB_PASS or entering Round 3.

**Fixed boundaries:** This is LAB-scoped only. Do not deploy to any
physical host, enter Round 3, release, merge v1 or change unrelated
Incus/UNKNOWN receipts. Even full P0 LAB PASS is not BH-017 physical
HOST_GO: that requires the owner's fresh exact-host authorization,
capacity/security/quota review, original Agent test and rollback.
