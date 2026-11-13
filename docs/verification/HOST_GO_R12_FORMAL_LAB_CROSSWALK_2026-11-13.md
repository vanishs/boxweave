# HOST_GO Round 1 and Round 2: formal isolated LAB per-case crosswalk

**Decision at acceptance, 2026-11-13 UTC:** ROUND1_PASS, ROUND2_PASS and
LAB_PASS are signed for the owner-authorized isolated S2.5 development-LAB
scope **with explicitly accepted alternative evidence**. This is **not**
HOST_GO on a physical machine, BH-017 remains NOT_AUTHORIZED, and the
third stage may begin only at nonmutating exact-host authorization
preparation. NO v1 merge, release or physical host installation.

## Exact source, original evidence and the accepted variances

1. **Code under final acceptance:** product native/worker/runtime semantics
   are the same as the verified 07084fde2cc5173ab39419a3e0b515a90e0d9e9b
   documentation HEAD; its original source implementation is 457c6dcf8f95feba64ce4d8ae936cbcf12297b92.
   The intervening d671d0c experiment added ONLY test-client assertions,
   exposed a genuine GitHub-runner Guest unproxied IPv4 HTTPS timeout
   [run 43379314014](https://github.com/vanishs/boxweave/actions/runs/43379314014)
   and was **reverted for the final source/CI**. Its failed original
   test Exec ID 87e8ecf655fb82172c409ca030ce627f remains documented;
   do not retry it. This is not a falsely claimed integrated success.
2. **Final successfully installed candidate prior to the diagnostic:**
   source 07084fd, five complete SUCCESS Actions:
   [S1 43374582088](https://github.com/vanishs/boxweave/actions/runs/43374582088),
   [quality A 43374581994](https://github.com/vanishs/boxweave/actions/runs/43374581994),
   [S2 native 43374576501](https://github.com/vanishs/boxweave/actions/runs/43374576501),
   [L0/L1/L2 43374576510](https://github.com/vanishs/boxweave/actions/runs/43374576510),
   [quality B 43374576368](https://github.com/vanishs/boxweave/actions/runs/43374576368).
   The signed Incus runtime was 7.6, one-use Ubuntu 24.04 VM,
   nested original candidate built once per layer and independently
   verified using its SHA. The prior independently downloaded S2.5
   artifact [14012879072](https://github.com/vanishs/boxweave/actions/runs/43372809750/artifacts/14012879072)
   pins code-bearing source 457c6dc, L0/L1 installed binary SHA256
   f8d6487b607b5fc234ffd7664bae04e7dffb8d63ae93676b29136341e76feca2.
   The separate S2 native fault candidate was built with its own label/
   binary hash and independently verified for its own original faults;
   **different CI builds are not falsely asserted byte-identical**.
3. **Approved real-public-IPv4 LAB equivalent (BH-009 only):** one-use
   isolated boxweave-test-ser2 native Incus 7.6 Project bw-r12-public4
   with exact source-Go-generated v1 exclusive ACL, an independently
   native-observed Guest bw-r12-public4-a UUID
   6f51f039-6156-49b4-aeb4-b3f79c081974, source-matched rejected
   IPv4 CIDR digest d9bba25746c1923e107b93a5fd453b50fc074246d2e364c870758dc4fce2e228,
   IPv6 CIDR digest cd16a2e4a8c121ef887c9cf64b6a80f4b28ad95954a11e816481ea2bcce221dd.
   **Real unproxied Guest TLS** to https://pkgs.zabbly.com/key.asc
   gave HTTP 200 from global IPv4 45.45.148.12, deliberately denied
   with only that Guest's egress set to reject, and restored HTTP 200
   after allowing egress. An independent root-local native read-only V
   repeated public HTTPS positive, compared exact actual ACL/private
   negatives/sole gateway exception/UUID and checked no foreign changes.
   The entire temporary Project/Guest/image/network/ACL was deleted by
   original UUID; unrelated demo-1-container remained RUNNING.
   [Native original proof and executable verifier](evidence/bh009_native_public4_verify.py),
   [exact ACL](evidence/bh009-native-public4-acl-pre.json),
   [guest/native identity](evidence/bh009-native-public4-guest-pre.json),
   [independent HTTPS transcript](evidence/bh009-native-public4-positive.txt),
   [reacceptance and cleanup record](evidence/bh009-native-public4-reacceptance.txt).
   **Acceptance limitation:** current ephemeral GitHub runner's
   authenticated installed MCP Guest direct-public-IP4 attempt timed out;
   the independent native public-IP4 test is same-current-product-policy
   N+V, not a latest-binary original MCP I+N+V. The earlier owner-approved
   real Codex I+N+V model-driven direct Box operations and public IPv4
   nested original code-path remain separate evidence. We **accept this
   as the owner's expressly permitted environment-equivalent LAB proof**,
   not as strict same-candidate public IPv4 I+N+V. **Stage 3 MUST
   re-test direct, unproxied public IPv4 HTTPS on the exact host-approved
   installed candidate with the actual Agent, native N and independent
   V before physical HOST_GO.** No authority to weaken ACL or accept
   relay/stand-in as physical-host public connectivity.
4. **Previously passed real intended Agent TC-013 reused by explicit
   owner direction:** actual nonroot Codex CLI 0.167.2 independently
   authenticated, enumerated and invoked v1 MCP, read host_status,
   performed original-key Create/Start/Stop/Delete and subsequent
   terminal restart reconnect. Source 0cf474f, binary SHA256
   021127cf723bfbe96c3668965b1e60dcc82856e5e3d41e09e84982568b5c1e85,
   exact immutable Operation IDs and second Box snapshot V in
   [real Codex report](HOST_GO_CODEX_REAL_AGENT_R2_2026-11-12.md).
   Not rerun on every ordinary development change. Stage 3 actual
   installed-Agent workflow on owner-approved host is still BH-017.
5. **Other owner-approved LAB equivalents/limits:** real separate
   routed simulated IPv6 WAN/TLS at both v1 layers (not public IPv6),
   controlled dropped native ACK, actual worker SIGKILL, WAL/ENOSPC
   failure injection rather than physical power loss; no new destructive
   S2.5 dir-quota pressure and no false hard quota enforcement. Stage 3
   must report host capacity, mount/inodes/PSI, dir quota warning and
   local rollback risk before any native mutation.
6. **BH-015 exact executable scope:** within the nested installed
   L0/L1/L2 I/N/V chain, the identical verified binary bytes were used
   and independently read. The separate full native S2 fault runner
   has its own exact independently accepted binary digest and
   original SQLite/native fault V. We accept a **per-fault/per-layer
   exact SHA and same product source combined crosswalk** for LAB
   rather than pretending two different ldflags-produced CI binaries
   are the same physical file. Dedicated host will require one
   pinned reviewed binary across its own actual install/rollback/V.

## Evidence-class ledger for every BH-001–BH-016 case

I = actual installed authenticated MCP/CLI; N = real native Incus/Guest/
network/kernel/SQLite; V = independent read-only original verifier;
T = deliberate controlled fault injection. PASSED below is **LAB** only,
under the two expressly documented equivalent/reuse boundaries above.

| Case | LAB verdict | Original I/N/V/T evidence and what it proves |
| --- | --- | --- |
| BH-001 | PASS | Accepted merged S1 and S2 audits, real current 13-tool SDK tools/list, CLI doctor/install/identity, source/binary SHA, pinned Go/Incus APT, native SQLite/UNKNOWN inventories; signed S2 PR #130. |
| BH-002 | PASS (LAB) | Fresh single-use Ubuntu 24.04 L0 and independent negative native-state inventory, signed Incus 7.6, dir pool/actual backing/free space and quota warning, protected foreign resources; physical host measurements deferred exclusively to BH-017. |
| BH-003 | PASS | Exact binary staging, approved root-local 8/8 prerequisites, private user/Project/SQLite/ACL, wrong digest and foreign-Project identity denied, MCP localhost 401, independent native readbacks. |
| BH-004 | PASS | Same-ledger genuine systemd restart and actual SIGKILL, independently reconciled original Operations, bad-digest/no-backup refusal and actual offline canary/root-local rollback with original DB/native resource. |
| BH-005 | PASS (LAB) | Official SDK protocol and 13 Tools; bad Host/Origin/path/Token, Viewer/expired/revoked denied; actual Codex login/MCP tools and original lifecycle separately proven at 0cf474f; host-specific Agent remains BH-017. |
| BH-006 | PASS | Current native S2 real Box list/Get/Guest Ready/restart, lifecycle Create/Exec 0+nonzero/output/Update/Stop/Start/Delete on original UUID, nested L2 MCP+native+separate V. |
| BH-007 | PASS | Same-key original receipt/no re-dispatch, changed-intent conflict, native accepted ACK loss/delayed postread original UNKNOWN, actual dropped MCP HTTP reply only after controller returned, independent original reconciliation. |
| BH-008 | PASS | Original caller disconnect and separate Agent checkpoint, genuine distinct-PID service SIGKILL, native worker interruption, WAL fsync before/after and ext4 ENOSPC with original UNKNOWN/no replay, fresh independent recipient/native receipt V. |
| BH-009 | PASS (OWNER-APPROVED LAB EQUIVALENT) | Source-exact installed MCP I + kernel/N/Verifier v4/v6 foreign/peer denial, source/MAC spoof, real routed simulated IPv6 HTTPS/on-off, private ACL exclusive readback and parent socket deny; **real public IPv4 direct N+V with current exact ACL on a distinct isolated Box**, plus earlier official MCP public IPv4. Current GH direct-public-v4 test FAILED and is disclosed. Exact new-candidate integrated real IPv4 is a **mandatory Stage-3 host preflight**. |
| BH-010 | PASS | Exact nested binary/Incus 7.6 inner L1 independent private daemon, original L0 MCP nesting-enabled L1, original L1 MCP L2 Create/Exec/Update/Stop/Start/Delete with original UUID, distinct SQLite/ACL/net/isolation and V; DIR_QUOTA_UNVERIFIED disclosed. |
| BH-011 | PASS | Native v4/v6 sibling/foreign and forged source/MAC denial, nested parent/private socket inode distinction, no parent token/foreign Project access, wrong actor/Viewer/reused name and UUID denial on real MCP/native. |
| BH-012 | PASS | Real quiet Exec VERIFYING and bounded wait preserving original stdout, client lost reply vs native incomplete/UNKNOWN vs completed, native unavailable/timeout originals and fresh independent receipt read, no implicit Cancel/replay. |
| BH-013 | PASS (LAB) | Existing signed S2 native bounded concurrency/slow I/O/timeout/retry/SQLite/native failure and same-source S2.5 source-race+fault witnesses; owner waived new destructive dir-quota pressure. Host CPU/RAM/disk/inodes/PSI measurement is BH-017. |
| BH-014 | PASS | Latest native nested MCP Guest Git failure with real nonzero unittest/persisted output, Git repair commit and passing test, independent two-commit/clean worktree/native SQLite I/N/V at source 07084fd; same Guest Exec path unaffected by subsequent test-only direct-v4 probe. |
| BH-015 | PASS (COMPOSITE LAB I/N/V/T) | Separate independent Verifier re-reads installed L0→L1→L2 original SHA/UUID/MCP operations, security negative, peer/foreign packets, signed rollback and exact owned cleanup; separately original same-source, exact-binary native S2 ACK/UNKNOWN/kill V, with independently pinned per-job binary. No claim of single SHA across two CI builds. |
| BH-016 | PASS | Native exact original L0/L1/L2 Stop/Delete, independent native/SQLite deletion and foreign untouched, operator stop/fallback while MCP offline, historical unrelated UNKNOWN intentionally unchanged, no legacy-main coinstall/release. |

## Additional TC-011–TC-014 S2.5 matrix

| Case | LAB verdict | Original I/N/V/Agent evidence |
| --- | --- | --- |
| TC-011 | PASS | Distinct authenticated SDK Agent A/B, same work/independent Agent refs/keys, distinct Box resources and audit, conflict denied, read-only foreign work and Viewer privilege borrow refused, exact original V. |
| TC-012 | PASS | A disconnects immediately after original irreversible admission; independent B reads fsynced external checkpoint/original ID/native UUID; lost real HTTP reply after dispatch leads to exactly one original Exec, no retry, V. |
| TC-013 | PASS (OWNER-APPROVED REUSE) | Actual chosen Codex CLI original v1 MCP Tool enumeration, Viewer read, authorized 4 original native mutations, reconnect with independent native V on earlier accepted binary; no per-commit re-login needed, actual host Agent BH-017 pending. |
| TC-014 | PASS | Real stalled/lost original reply, token A revoked and HTTP 401, rotated independent actor B reads same original without permission inheritance, original UNKNOWN explicitly blocked/no native replay or physical effects. |

## Formal outcome and Stage-3 entry controls

**ROUND1_PASS=YES, ROUND2_PASS=YES, LAB_PASS=YES** for this
owner-approved LAB evidence-contract scope, with BH-009 and BH-015
equivalence explicitly noted above; signed acceptance is not S5 final
release acceptance or real-host authorization. Prior red tests and
UNKNOWN remain preserved, not changed to PASS or retried.

**ROUND3=READY_FOR_EXACT_HOST_APPROVAL, HOST_GO=NOT_AUTHORIZED.**
No physical host was selected or approved, so **no physical host
discovery, native changes, installer, service, project, source, disk,
ACL or runtime mutation is authorized yet**. The next permission must
identify the standalone physical host and OS identity/project/pool/
network/ACL/service/port and rollback/risk envelope. Even after
authorization, exact host baseline, actual latest binary installed
MCP/Agent **unproxied public IPv4** and network negatives, native V,
host capacity/quota warning and fallback are a hard BH-017 postflight,
not an automatically inherited LAB_PASS.
