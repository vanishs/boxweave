# Native Filesystem R0 lab — execution record (2026-11-28)

Scope: independently owned new nested Incus and MCP SDK labs, plus a synthetic local staging fixture. No changes to outer RC15, existing development Boxes, parent daemon, host policies, or release. This is an in-progress feasibility checkpoint, not R0 acceptance.

## Exact identities

- PR #202, branch feat/v1-native-filesystem, base v1 at 82f0d4fdb0be1bd0e0681f39df70f6bcc13a4f3b (merged #201).
- Outer authenticated RC15, commit c0ad27a11fd1644e7e9755900c2c9b5421f84c9f, schema 7, native Incus 7.6, Project user-1001. Host status tracking b4ee801d8a109a79acf6506efaf7e8d6. Host storage quota not independently verified; do not stress physical capacity.
- New Box native-fs-r0-incus77, UUID/incarnation 0784a712-72fa-442e-8a04-188174990203, CPU 4, memory 4096 MiB, disk requested 30 GiB, nesting=true, autostart=false. Original create 4e4b5e2327afd3c6b269ff328fea642c; native ab2f9608-7774-48b8-8e3f-9f0cad235397; SUCCEEDED.
- Preflight Exec dee5b2826d385c574fd5b1a2dc766eac: exit 0; Ubuntu 24.04.5 LTS noble, x86_64, curl available, no Go or Incus initially. Guest-reported /proc/self/attr/current was unconfined; no profile change was requested or made. This is not evidence of enforcing ancestor AppArmor or production nesting safety. No privileged or host policy changes.
- Zabbly stable package source signed with verified documented fingerprint 4EFC590696CB15B87C73A3AD82CC8797C838DCFD. Apt reported candidate 1:7.7-ubuntu24.04-202611270613 for incus, incus-base and incus-client. Initial install Exec bede86d17cc7184f7b1cefb3e0161aaa completed with guest exit 1 after successful prerequisites/repository setup: a local version check omitted Debian epoch 1:. Incus was not installed by that command.
- Corrected exact-version install original fb96d83570f3666a42d29fb4d3fea217, native e65feb74-51d6-418e-8b85-ff50dfd64f85, Tracking 7aa4946c73bb60e17c07ceb68a37048a. Continuation verified SUCCEEDED, guest exit 0, exact dpkg packages incus/incus-base/incus-client 1:7.7-ubuntu24.04-202611270613, Client and Server 7.7. Both original output streams read.
- [Probe source](native-filesystem-r0/probe.go.txt), commit 25970f3e7c196aed3ef773f93f78c123f1408576. Exact source uploaded to /root/native-fs-r0-probe.go using exclusive file apply original 98d63a102eeb080bba778d93761608e8; SUCCEEDED with native readback, no native Operation ID for SFTP.
- Build attempt refused before admission with RESOURCE_BUSY (corrected installation original still active). No build operation, no SDK execution or new native result. Do not describe this as a compiler failure.

## Initial bounded research plan (historical; continuation below supersedes execution status)

1. Inspect corrected install original, read stdout/stderr and guest exit; verify dpkg exact version and local daemon version. Inspect fresh inner daemon inventory before initializing. Use only fresh inner fixtures and Unix connection; no remote listener. If normal nested initialization or launch fails, preserve precise error and do not weaken confinement.
2. Prepare isolated Go module in /root/native-fs-r0-sdk with exact Incus SDK v7.7.0, pkg/sftp v1.13.13 and Go 1.26.10. Copy retained probe to main.go, use go mod tidy/build, retain go version -m and source/binary digest. Compilation alone is not native acceptance.
3. Initialize fresh inner daemon with minimal dir storage and no public listener; create one never-started inner Linux container named fs-r0-fixture. Verify original native UUID, then run probe with its name, UUID, Stopped and never-started label. If launch is permitted without changes to policy, repeat on Running and stopped-again with separate probe paths. Do not reuse earlier probe output directories or replay uncertain writes.
4. Probe records SFTP advertised extensions, exclusive collision, >1 MiB observed streaming read/write, digest, offset read, attributes, two independent SFTP handles, ordinary/Posix rename outcomes and native state. No proof of cross-Box copying, CAS, snapshot consistency, hostile path races, crash durability, restricted Project or full lifecycle coordination from this program. Follow with distinct bounded native experiments for those claims only as justified.
5. Preserve exact output and actual failures in this record. Record downstream R0 risks and fallback decisions; do not turn unsupported operations into PASS. Keep lab for continuation.

No source dependency bump, product implementation, test acceptance, merge, or release was performed by this checkpoint.

## Continuation: parallel research lanes

### A. Incus 7.7 SDK and lifecycle (native)

Outer lab 0784a712-72fa-442e-8a04-188174990203. All following completed Exec originals had independently read terminal state, exit 0 and both output streams.

| Original | Native operation | Actual outcome |
|---|---|---|
| 3ded4ceb59cdbf535218d8446d41e0c0 | 9f158321-8374-4c9e-9c33-fe4928a31b4b | Go 1.26.10, Incus Go SDK v7.7.0 and pkg/sftp v1.13.13 compiled in /root/native-fs-r0-sdk; build module identities verified |
| ba402c5498acdcf8fbddfa3028499c74 | 9ac662fb-8fca-4af6-a15b-cd6282355b79 | Fresh minimal dir backend, default profile with internal incusbr0, fresh fs-r0-fixture UUID 22ba2396-7495-41a2-871a-ed167be08efb. Never-started Stopped SFTP probe PASS |
| 321c4e9f3e6b6d7de8b9f1f54afab7bc | a0dd23bc-4a9e-49ce-a196-b27eff73f4a7 | Same UUID Running then Stopped again, both probes PASS, final state Stopped |
| 07d8c2fce51af4a2431a6499a2537e87 | c517120e-252e-407e-bf00-f3a81957be56 | Two distinct stopped guests, controlled SFTP reconnect, 8 MiB+19 copy, hardlink collision and distinct-content ordinary Rename |
| cf04509397efaa45a926ef39c0959665 | 87974bd8-c18e-46d8-ad79-4b1d9f2a9085 | Lifecycle with open read handle and bounded directory listing |

Exact build source/binary SHA256:
- Initial probe: e9a3d58fa3813f2aa089756479624518636c81353d796662ee4b10a647da7b9c / 11bbea4c26e3bf0f0ba0dfbf207b0e60ef2731da989e40ee4878c82ea7f10ee5.
- [Cross probe](native-filesystem-r0/cross_probe.go.txt) 604eb9fa49fc0e0ebedb727affa6dc947935d12fa0cb9b417331a04555d7a3a5 / b36700dcec5eae766443ad7365a4ca8197a63aa27b3f11b85e441cc0c83670d6. Exclusive upload e3d6090cd06d1f1aa4fa8e48b174e362 SUCCEEDED.
- [Lifecycle probe](native-filesystem-r0/lifecycle_probe.go.txt) 982401a57289dbdc2dbd34661413449136e1e6595518d4d9d7aa83f673f37d6d / 064a089bbf33de27d39ba6de7eb56f77605a71c6baff1b55784a523102266a15. Exclusive upload f8c792c5a220c64c179cf561fb706235 SUCCEEDED.

Observed in all three initial states: 1,160,000-byte streamed write/read and two independent SFTP handles, SHA256 255d77adda63b9dec5ad662e209d687c64e8e8224fd5b58231fa6ad3db97e0b3; random offset read and basic attributes verified, exclusive collision refused, ordinary Rename accepted an existing target, PosixRename replaced the target. Extensions advertised: posix-rename@openssh.com v1, hardlink@openssh.com v1, statvfs@openssh.com v2; fsync@openssh.com not advertised. No assertion that StatVFS is quota enforcement.

Second inner guest fs-r0-dest UUID 370d86b8-178b-468c-9ffd-17bcc1f105be, never started. Cross-copy 8,388,627 bytes, SHA256 da73cfa063906fcc3598f89db0cafffc11163a238f2dcf007b1abf32aca6c759. Explicit 256 KiB application chunk, closed destination session after first chunk, fresh connection verified prefix and completed copy. This is controlled session reconnect after a known write, not lost acknowledgment or power loss. Hardlink creation succeeded, second link to existing final path refused (SSH_FX_FAILURE), digest preserved. Ordinary Rename over distinct old bytes succeeded, destination verified new bytes and source name missing. Hardlink leaves alias and lacks proven original identity/atomic conditional checks; do not enable as product publication until separate proof.

Lifecycle: start native 85c5bab1-e5cd-410a-bce2-0164b6d21cc0 remained pending at 2s with stopped-rootfs read handle held, then completed after closing handle. Fresh running SFTP read passed; stop native 830afbac-da44-4096-a290-2ebe2679ab1c completed, both guests Stopped at final readback. This single observation is consistent with native lifecycle coordination, not proof of all races. Product must serialize affected operations, bound read lifetimes, close handles before state transitions, revalidate identity and state; do not reuse pre-transition SFTP clients. Directory: full 1024 entries returned, 32768-byte wire cap caused connection lost after 100 partial entries; discard partial entries on enumeration error. Do not claim server-side paging, full memory cap, or complete listing.

Native scope: fresh default Project, dir backend, root local Unix client in nested disposable lab, no profile relaxation, no parent daemon changes, no public listener configured by these commands. Guest AppArmor preflight reported unconfined; no assertion of enforcing ancestor, restricted Project or production nesting safety. No CAS, hostile parent rename, snapshot, native power-loss durability, explicit storage unmount proof, cross-layer access or host quota acceptance.

### B. MCP SDK (independent lab)

Created native-fs-r0-mcp-sdk UUID/incarnation 45457608-4b9d-4ee9-a81b-f3a388c3b1ef, 2 CPU, 2048 MiB, requested disk 12 GiB, autostart=false. Create 5a2d37aea7d861c167c69072aaf6178e, native b5adc0d8-46d1-406e-8e13-ef58829b664b, SUCCEEDED. Go module prerequisites 63016d92fbc5a04f65c36d2cf5b0453a, native 73448898-fe61-499b-a20e-fc19fbb4db78, exit 0. Initial upload refused RESOURCE_BUSY before admission while prerequisites active; subsequent original 1ec8e7389c2cae41fdbe9038e9c8af0b SUCCEEDED. Same lab resource claims serialize writes; separate lab avoids waiting on native research.

- [Resource probe](native-filesystem-r0/mcp_probe.go.txt) Exec e0101c2b28a680948c38f5e881b60947, native 3ac89895-5af9-46b4-af7b-6868e754ce70, exit 0; Go 1.26.10, SDK 1.9.0-pre.1, source SHA256 b241c26b6c6c2de4c00dc79442ffe1868eea1bd8c1981e8726ef2a5f1c61918e, binary dbb48dc591a6ddbbff6a3d12b596b3794c547ec8b1e07bdd93f48236e4591c15. Synthetic loopback official HTTP SDK, stateless JSON: template discovery, typed ResourceLink, text read, cacheScope=private/ttlMs=0, repeated read hits handler, synthetic foreign Actor refused, noncanonical URI refused, revoked token refused. Not actual product auth or external Agent witness.
- [Budget/protocol probe](native-filesystem-r0/mcp_budget_probe.go.txt) upload f532a62000c4136fb536251d4f2ff35e SUCCEEDED, Exec 6440ad0e29ce060f8e28656997bca2fd native bccad58c-f1f1-45b2-9cce-dd04c79b604b, exit 0. Source SHA256 4689e18b622e3b8212e1e72ba8e9c1b2cd9b9ea3d9d48845506ddc60268e4e8b, prerelease binary 2de9fe770e28d5854a755ff437b8f79066f66d84646bbbff4372dd2553f25eca. Negotiated 2026-07-28 and 2025-11-25, 1 MiB binary Resource round trip, exact default JSON ceiling. Both probes compiled and passed again against SDK 1.8.0 in separate module /root/native-fs-r0-mcp-stable.

| Raw upload data | Wire request current / legacy | HTTP, both SDK versions |
|---|---|---|
| 2,097,152 bytes | 2,796,520 / 2,796,300 | 200, one handler invocation |
| 3,145,728 bytes | 4,194,620 / 4,194,400 | 413, no second handler invocation |

4 MiB default is an SDK request-body ceiling, not a BoxWeave Tool/connector limit. Core Resources return binary through JSON, not an out-of-band stream. Neither SDK version establishes Files extension or real connector support. Keep bounded Tools fallback. Prefer 1.8 stable for core if 1.9 pre-release adds no required tested capability, subject to owner freeze decision and combined product regression. Tagged 1.9 pre1 StreamKeepAlive applies to SSE POST, not application/json; targeted notification requires open subscriptions/listen stream and appropriate protocol. Keep subscription/notification optional, not a prerequisite; no silent transport redesign.

Product compatibility experiment original 1136f2601bd1554859599ec307ae3046, native 3daf601a-54b4-49db-9797-65bdf585b496, separate temporary source checkout intended exact base 82f0d4f. No repo credential requested. Latest state VERIFYING; result not yet verified. Inspect original exit and outputs before continuing. No product dependency commit.

### C. Local private staging recovery (synthetic, independent)

[Local transfer probe](native-filesystem-r0/local_transfer_probe.go.txt), source SHA256 70ddde969e4c04d513dda93611121903e93e7966d4fe98b4e1ca6e3dcadfee2d; local Go 1.23.2 Linux amd64, disposable temp directory, exit 0. 5,242,897 bytes, 256 KiB chunks, SHA256 a69aa326153be1a5436740bf7be207288eff79a2e67e09d792d5b2e63eeefe0f. Real subprocess exit 77 after local file fsync before durable journal ack; verified exact staged range on restart path, identical range readback and changed-range comparison oracle. Full file verified. Unrelated same-byte destination refused by local link and proven distinct inode. Subprocess exit 78 after link and directory fsync, before journal ack; exact local inode plus digest reconciled. Single writer, known local staging fixture, not SQLite/product, SFTP, true power loss, concurrency or full chunk API proof. The mismatch test is an oracle, not an implemented admission/refusal API. Local inode evidence cannot be assumed available via SFTP.

## R0 evidence-driven decisions and outstanding gates

- R0 dependency: standalone exact SDK/toolchain and native package PASS; combined product temporary compatibility result pending. No upgrade of outer daemon or deployed schema.
- R0 offline: basic native Running/Stopped/never-started, cross-copy and one lifecycle coordination observation PASS; restricted nonroot Project, alternate storage and full lifecycle/identity negative cases still require integrated acceptance.
- R0 mutation: native ordinary and Posix Rename overwrite; CAS stays UNSUPPORTED. Hardlink no-clobber observed on dir fixture only. Do not promise atomic completed file publish across all storage/backends; direct exclusive create has visible partial-effect risk and retains UNKNOWN on ambiguity.
- R0 MCP: stable and prerelease synthetic HTTP Resources/auth/legacy/current and size tests PASS. Real external client, current BoxWeave auth, negotiated Files extension and subscriptions remain unverified. Defer experimental adapter and subscriptions if unavailable; Tools fallback not blocked.
- R0 transfer: controlled native reconnect and local process-exit recovery PASS in distinct scopes. Durable controller staging is feasible with sync and journal, but missing native fsync means do not equate verified guest readback with power-loss durability. Retain original intent; explicit UNKNOWN and no blind publish replay; native transfer partial failure/restart, actual SQLite linkage, disk-full/cancel/expiry and quota acceptance remain.
- R0 scan: incoming wire bound remains meaningful on new SDK/SFTP; must discard partial enumeration and report incomplete/refused, not silently page or report complete. Product aggregate limits to be measured.
- Design freeze: proposed 25-tool boundary, strict mode schemas, resource URI encoding, persistent transfer schema/retirement and supported data limits require explicit owner decision and implementation design. No production feature or R0 independent signoff claimed here.

Keep two outer lab Boxes and inner stopped fixtures for continuation. No merge/release or host changes. Use original receipts, do not rerun same names/paths without explicit fresh fixture intent.

### Restricted Project feasibility: blocked execution

[Restricted Project probe source](native-filesystem-r0/restricted_probe.go.txt) retained at commit 89d09803b8b27beb1146da76facb5fd2e77c2dba. Intended to test normal nonroot Incus per-user socket with an owned inner fixture. The subsequent tool call to upload it was refused by platform safety checks; no admitted Operation ID or native result was received. No attempt was made to change route, replay or bypass the refusal. No restricted user/project was created by this experiment. Restricted nonroot file feasibility remains unverified; preserve the existing product security requirements. Independent product compatibility build is unaffected and its original receipt remains the only current pending experiment.
