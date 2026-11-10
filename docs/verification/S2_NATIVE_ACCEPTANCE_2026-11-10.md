# S2 installed-candidate Incus 7.6 black-box — 2026-11-10

**Result: S2 CORE NATIVE SUBSET PASS; S2 PHASE NOT YET PASS.** This is a
single-use GitHub-hosted Ubuntu 24.04 VM run on an isolated daemon and never
a deployed BoxWeave v1 instance, shared Incus host, or product release.

- **Exact tested commit:** `d50371ac9cf83299bf8a1b18854ad114d10e43d0`
- **Original run:** [S2 native job](https://github.com/vanishs/boxweave/actions/runs/42796072254/job/129761466288)
- **Independent source checks:** [build-test](https://github.com/vanishs/boxweave/actions/runs/42796081380/job/129761502898) and [quality-gate](https://github.com/vanishs/boxweave/actions/runs/42796081112/job/129761497473), both SUCCESS.
- **Actual backend:** signed Zabbly `1:7.6-ubuntu24.04-202611072126`; upstream package signing key fingerprint `4EFC590696CB15B87C73A3AD82CC8797C838DCFD` independently asserted by the CI script. BoxWeave links `github.com/lxc/incus/v7` v7.6.0 and `github.com/modelcontextprotocol/go-sdk` v1.8.0.
- **Native isolation:** newly initialized daemon on one-use runner, non-root dedicated `incus`-only account and restricted `user-1002` Project, distinct mounted loop/ext4 backing storage pool, single dedicated pre-created NetworkACL, Profile NIC with filtering, Ubuntu 24.04 `noble` image imported into the **target** Project. CI refuses an existing Incus data dir/service identity.
- **Probe mechanism:** official MCP Go SDK client talks to actual compiled candidate binary's loopback HTTP MCP listener, then real native Incus operation readback. Final inventory is independently checked by administrator-only `incus list --all-projects` and must be empty.

## Follow-up exact-candidate reacceptance (later on 2026-11-10)

The original `d50371ac` proof below is preserved as historical evidence.
A verified earlier expansion is `facd9a0b03894db64d81577b5b0978508a69ade2`:
[full native black-box](https://github.com/vanishs/boxweave/actions/runs/42803767377/job/129787650466),
[build/race](https://github.com/vanishs/boxweave/actions/runs/42803776642/job/129787688238)
and [quality](https://github.com/vanishs/boxweave/actions/runs/42803767463/job/129787651269)
all SUCCESS on that exact new candidate. Recorded stderr/stdout after a fresh
client, guest exit 7, 70KB stdout paging, quiet and delayed exit 0, CPU/RAM
and root growth, native postread after delete and source auth remain green.

Additional independent native witnesses:

- `S2_NATIVE_UNSAFE_INGRESS_DOCTOR_DENIED=PASS`: on a **disposable owned Project**, changing native NIC default inbound to `allow` made BoxWeave `doctor` refuse readiness. Restoring `reject` returned native readiness. The *same* safety rule now gates runtime admissions. This is configuration fail-closed, **not yet packet-level isolation**.
- `S2_NATIVE_SAME_KEY_CONFLICT_AND_DIFFERENT_KEY_DUPLICATE_DENIED=PASS`: two original create intents cannot use one actor key, and a new key cannot induce a second native same-name effect.
- `S2_NATIVE_CANCELLED_LONG_POLL_IS_READ_ONLY=PASS`: official MCP client canceled a 20-second operation reader while real guest Exec ran; the original command later completed with its recorded content and no native cancellation.
- `S2_NATIVE_STOP_ALREADY_STOPPED_NOOP=PASS` and `S2_NATIVE_START_ALREADY_RUNNING_NOOP=PASS`: no invented native operation IDs for verified already-desired states.
- `S2_NATIVE_STOP_START_STOP_DISTINCT_KEYS=PASS`: an earlier Stop key returned its original immutable receipt; a new Start and later new Stop performed a second genuine native cycle.
- `S2_NATIVE_CRASH_DISPATCHED_ORIGINAL_RECEIPT=PASS`,
  `S2_NATIVE_EXACT_SERVICE_PROCESS_KILLED=PASS`,
  `S2_NATIVE_PROCESS_KILL_UNKNOWN_REPLAY_DENIED=PASS`,
  `S2_NATIVE_CRASH_ORIGINAL_GUEST_EFFECT_EXACTLY_ONCE=PASS`: original
  persisted Exec native Operation ID, actual runtime SIGKILL during guest command,
  fresh instance reopening **same SQLite** and protected UNKNOWN, same-key
  replay/no re-dispatch, and independent guest file line count **1**. The
  single-use fixture was removed by its test-host admin after evidence;
  this is not MCP force-unlock and did not touch either old UNKNOWN create.
- `S2_NATIVE_INDEPENDENT_INCUS_POSTREAD_EMPTY=PASS` verified no residual
  native instance in the single-use daemon after all phases.

A newer **exact installed candidate** `74577cbe2ddda8e228fce76111db68279ba3b5f4`
also passed [real native black-box](https://github.com/vanishs/boxweave/actions/runs/42804739893/job/129790881090),
[Go build/race](https://github.com/vanishs/boxweave/actions/runs/42804750212/job/129790917094)
and [quality gate](https://github.com/vanishs/boxweave/actions/runs/42804740092/job/129790886009).
The actual compiled candidate digest printed in the native job is
`1008c96d9eaa8ae546519cde327faab1f42016dbdf90e518cd4b713f601f9efd`.
Additional native PASS:
`S2_NATIVE_RESTRICTED_USER_FOREIGN_PROJECT_DENIED` and
`S2_NATIVE_NAME_REUSE_OLD_UUID_AND_DELETE_KEY_DENIED`. A deleted name was
recreated with a different native UUID, the old Box ID was denied, and replay
of the original deletion receipt while the new Box still existed did not
affect it. The second Box was then explicitly deleted and a separate
administrator inventory showed no remaining native instance.

The latest **per-case** progress and residual blockers are in
[S2 acceptance crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md).
Neither this run nor the earlier one establishes S2 exit, first-start
packet-level IPv4/IPv6 isolation, running recorded Exec cancellation,
internet toggle, interrupted SQLite commit, or S4 dependent deletion.

## 2026-11-10 extended installed packet/isolation and continuity reacceptance

**Exact accepted binary source:** `b679248d2ec1d112996a1da0e3ac2136cd7b5a1a`,
**compiled SHA256:** `df89109f15d134c433c1539e4b5b28ff56cf2c41b27999463cb3e004fb316f2b`.
[Native job](https://github.com/vanishs/boxweave/actions/runs/42845735407/job/129932568109)
SUCCESS; [Go build/race](https://github.com/vanishs/boxweave/actions/runs/42845751449/job/129932625888)
and [quality](https://github.com/vanishs/boxweave/actions/runs/42845735396/job/129932570441)
SUCCESS on the exact candidate. This is an additional installed-binary/real
Incus 7.6 evidence slice **NOT** an S2 phase or physical-host release PASS.

**New real packet and identity proof:**

- `S2_NATIVE_PACKET_IPV4_IPV6_SAME_BRIDGE_DEFAULT_DENY=PASS`: two original UUID-registered Boxes using the same managed NIC bridge/shared ACL each passed guest-to-bridge-gateway IPv4 Ping, while A→B and B→A native guest IPv4 and IPv6 Ping returned the true no-reply exit code 1. The probe checked actual `ping` binary availability and separately waited for assigned guest IPs, so missing client binaries or uninitialized guest routing could not be counted as isolation.
- `S2_NATIVE_PACKET_INTERNET_TOGGLE_GATEWAY_EGRESS=PASS`: Box A was **already running**; the native `internet=false` update blocked an earlier working IPv4 gateway Ping, then `internet=true` restored real packets. Per-instance ETag updates preserved actual native inbound `reject` ACL attachment, IP/IPv6/MAC filtering and CPU/disk. **Caveat:** testing gateway packets is not proof of successful public WAN/HTTPS connection, IPv6 Internet toggling, full peer-grant/revoke, host LAN or S4 routing.
- `S2_NATIVE_PACKET_PEER_EXACT_CLEANUP=PASS`: test peer was stopped/deleted by its own original MCP operations, then the earlier core and crash fixture were also removed. Independent host administrator's final all-Project inventory was empty.
- `S2_NATIVE_CREATE_DEFAULT_INTERNET_ALLOW=PASS` and real `start=true` guest creation now survive repeated source runs. A **real defect** was caught: `GetExactBox` and `ObserveCreated` broke out of a Go `api.DevicesMap` range when visiting root disk; random map ordering skipped the NIC's `internet` field, returned false even though independent native expanded NIC showed egress `allow`, and marked otherwise successful create UNKNOWN. Commit `de9224e4` removed this order dependency via one full traversal. A 256-run pure regression and the new native/SDK full lifecycle verify both root size and Internet value together.
- `S2_NATIVE_COMPLETED_OUTPUT_AFTER_SQLITE_RESTART=PASS`: a completed guest Exec's real stdout, stderr and **exit 7** were recovered after a separate active Exec's BoxWeave service SIGKILL and reopening the **same original SQLite**. The killed original native Exec remains UNKNOWN/no re-dispatch; independent guest marker was observed exactly once.
- The exact accepted binary also re-proved custom official MCP SDK `_meta` attribution + conflict, work/agent joint search, scoped HMAC cursor paging/tamper denial, true opt-in nesting, same-key conflict and name-reuse anti-adoption, truthful late cancel, read-only long-poll cancellation and 70KB paginated output.

**Unclosed security limits:** Public Internet routing beyond the owned bridge,
IPv6 egress-toggle policy, heavy output native storage exhaustion/retention,
native response loss before Operation ID persistence, SQLite interrupted commit,
unverified update readback and full TC/A phase gate remain PARTIAL or BLOCKED
as specified in the [updated case crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md).
A **later exact installed candidate** `97df5d63005eea33d35155cb6862556027ea70c1`
[real native run](https://github.com/vanishs/boxweave/actions/runs/42847588015/job/129939158833),
[build/race](https://github.com/vanishs/boxweave/actions/runs/42847607027/job/129939222907)
and [quality](https://github.com/vanishs/boxweave/actions/runs/42847587965/job/129939159097)
all SUCCESS; compiled binary SHA256
`46faf46df444133b7024a0b6271def19fc1537be31520f3bd3e282b0138f5235`.
New `S2_NATIVE_DUAL_STREAM_16M_OUTPUT_READ_BUDGET=PASS`: a guest produced
16MiB+4 stdout **and independently** 16MiB+4 stderr. Each native output
read from offset 16MiB-16 returned exactly 16 bytes, `ended=true`,
`truncated=true`, `next_offset=16777216`; an offset beyond 16MiB
was rejected. No claim about underlying native output file storage
truncation or garbage collection is made. First fixture use of
`tee /dev/stderr` failed the stderr proof, so the final probe writes
stdout and actual guest fd 2 independently, retaining the original
unweakened byte/flag assertions. The complete native packet/identity,
SIGKILL/recording-restart and empty-inventory matrix reran successfully.
**Installed foreign/corrupt SQLite refusal now PASS:** exact source
`21db2e9bdd76f6453cb0c149b1957f8b1efee158`,
[full native CI](https://github.com/vanishs/boxweave/actions/runs/42848419436/job/129942104632)
and [build/race](https://github.com/vanishs/boxweave/actions/runs/42848434191/job/129942159145)
both succeeded. Native marker `S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED=PASS`
means the compiled `boxweave doctor` refused both an independently constructed
foreign schema and invalid SQLite file **without changing either input SHA256**.
The earlier complete packet/dual-stream/worker SIGKILL/guest exactly-once
matrix reran on that exact installed candidate. **Torn transactions and
native lost-ACK failures are not accepted**.

## Latest exact installed reacceptance — `cd4ac36` (2026-11-10)

**Actual installed-source SHA:** `cd4ac36d15f689ed1c054e9991c4d9958c94dbd4`;
**independent test-built binary SHA256:**
`c4f27f4c44f0d04af06cdfbf17f3e21a9fecff0f401ffb16857f980e575613d1`.
[Real MCP/Incus 7.6/guest/SQLite black-box](https://github.com/vanishs/boxweave/actions/runs/42857519652/job/129974501751),
[Go build/race](https://github.com/vanishs/boxweave/actions/runs/42857537255/job/129974563073),
and [quality](https://github.com/vanishs/boxweave/actions/runs/42857519569/job/129974501188)
**all SUCCESS**. This is a **core-native-subset PASS**, not overall S2
release/phase acceptance or proof of physical/shared-host deployment.

New actual native markers on this exact installed candidate:

- `S2_NATIVE_PACKET_IPV6_GATEWAY_BASELINE=PASS` and
  `S2_NATIVE_PACKET_INTERNET_TOGGLE_IPV4_IPV6_GATEWAY_EGRESS=PASS`:
  after first-start packet isolation, an owned running Box reached its real
  native IPv4 and IPv6 bridge gateways; `internet=false` blocked both and
  `internet=true` restored both. Link-local IPv6 gateway requests explicitly
  bound `eth0`; absent interface scope must not count as a firewall denial.
- `S2_NATIVE_PACKET_INTERNET_ON_DOES_NOT_UNBLOCK_PEERS=PASS`:
  IPv4/IPv6 peer traffic remained denied even after egress was restored.
  The guest had actual IPv4/IPv6 addresses and a positive bridge route,
  and the true guest Ping exit status was checked.
- `S2_NATIVE_SQLITE_OLD_MULTIPLE_VERSION_AND_FOREIGN_TABLES_DENIED=PASS`:
  real compiled `boxweave doctor` refused an old `schema_version`, duplicate
  version rows and version 1 with unexpected foreign table; original SQLite
  file SHA256 remained identical. This supplements foreign/unparseable DB
  refusal, not a migration or interrupted-commit fault-injection PASS.
- `S2_NATIVE_MCP_TOOL_ANNOTATIONS_ACTUALLY_ADVERTISED=PASS`:
  genuine official Go SDK `ListTools` over HTTP returned 13 exact tools
  with correct read-only/mutation/idempotent/open-world/destructive hints.
- All earlier core real lifecycle/Exec tests also reran: optional nesting,
  CPU/memory/disk, original UUID/name/key checks, 70KB paging and >16MiB
  dual-stream truncation, canceled long-poll without canceling native effect,
  real BoxWeave process SIGKILL, completed stdout+stderr/exit after reopening
  original SQLite, original in-flight UNKNOWN/no replay, independent guest
  effect exactly once, native inventory empty.

**Important WAN distinction:** `ec22a5e` printed
`S2_NATIVE_EXTERNAL_IPV4_WAN_BASELINE=UNVERIFIED`, because the disposable
runner's guest could not demonstrate its initial public HTTPS control.
This is neither an external WAN PASS nor evidence of a BoxWeave bug; a
separate Internet-enabled environment with independent positive routing
is still needed to prove public HTTPS on/off/on. IPv4/IPv6 **bridge
gateway** reachability is not equivalent to public Internet.

Still open: installed fault injection of lost native ACK before ledger
persistence, interrupted SQLite commit, stale ETag/ambiguous update postread,
on-disk Incus Exec log storage bounds/retention, and remaining TC/A-case
negative tests. Running detached recorded Exec cancellation stays
truthfully unsupported. **Do not merge protected `v1` or remove the two
unrelated historical UNKNOWN creates.**

## Native proof matrix (all observed PASS in this run)

| Proof | Assertion |
|---|---|
| S2_NATIVE_SIGNED_PACKAGE_KEY_FINGERPRINT_VERIFIED | Signed trusted source, not unsigned installer |
| S2_NATIVE_OWNED_UBUNTU_ALIAS_VERIFIED | Exact private image imported into owned Project, version inspected |
| S2_NATIVE_PRECONDITION_EXACT_NONROOT_BLOCK_POOL | Actual restricted user/Project, native storage and mounted block pool |
| S2_NATIVE_MCP_DISCOVERY_13_OF_13 | 12 S2 tools plus S1 `host_status`, no ghost tool |
| S2_NATIVE_VIEWER_CREATE_REFUSED | Viewer token cannot mutate |
| S2_NATIVE_CREATE_IDEMPOTENCY | Same actor/key returns the original journal Operation, not a second native create |
| S2_NATIVE_CREATE_STOPPED_UUID | Start=false created one real stopped Incus instance and persisted verified native UUID |
| S2_NATIVE_OPERATION_WORK_SCOPE | A second token cannot borrow the original actor's operation using `work_ref` |
| S2_NATIVE_BOX_LIST_UUID | `box_list` contains exactly the original owned UUID |
| S2_NATIVE_CPU_MEMORY_UPDATE_PRESERVES_DISK | Typed update changed CPU and RAM; omitted disk remains unchanged |
| S2_NATIVE_DISK_GROWTH_PRESERVES_CPU | Root disk growth persisted while preserving CPU |
| S2_NATIVE_DISK_SHRINK_DENIED | Separate shrink request terminally refused; effective disk remains grown |
| S2_NATIVE_START | Exact original UUID started and native Running observed |
| S2_NATIVE_EXEC_RECONNECT_EXIT_7_STDOUT_STDERR | Actual guest nonzero exit 7; fresh MCP session reads recorded stdout/stderr and `command_success:false` |
| S2_NATIVE_EXEC_UNSUPPORTED_CANCEL_TRUTHFUL | Late cancel is independently journaled as FAILED and never rewrites original succeeded Exec |
| S2_NATIVE_STOP | Original instance stopped and native Stopped observed |
| S2_NATIVE_DELETE_VERIFIED_AND_REPLAY | Original UUID deleted and readback verified; same delete key replays original receipt after tombstone |
| S2_NATIVE_DELETED_BOX_NOT_LISTED | Tombstoned UUID excluded from owned inventory |
| S2_NATIVE_INDEPENDENT_INCUS_POSTREAD_EMPTY | Separate host-admin inventory confirms no test instance left |
| S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS | Full installed candidate/protocol/native lifecycle proof completed |

## Important observed corrections

1. Ubuntu 24.04 image metadata in Incus simplestreams uses `release: noble`, not always `24.04`. The typed adapter accepts those two exact Ubuntu release representations, not any OS.
2. The CLI `incus image copy --project` refers to **source**, so provisioning to the new restricted Project needs `--target-project`.
3. For official Incus SDK `DoHTTP`, a relative recorded-output path does not automatically acquire the authenticated native server base URL or Project query; both must be built from the SDK connection and the verified restricted Project. Never accept a caller-supplied URL.
4. Two retry classes are distinct: same-key creation before/after native completion returns the original receipt; same-key deletion **after native instance disappeared** must first replay the journal, rather than failing ownership readback or issuing another delete.
5. The test runner's inherited `XDG_CONFIG_HOME` must not point the non-root native actor into the GitHub runner's private configuration directory.

## Still open; not silently promoted to S2 PASS

- True native SIGKILL **after the native Exec ACK was persisted**, original UNKNOWN and exactly-once guest side effect are now independently proved on later `facd9a0b`. **Still missing:** pre-ACK lost-response native create/resize, interrupted SQLite commit, positive nesting, live name reuse/foreign Project on the later candidate and hostile readback-failure injection. Earlier S1 tests alone are not a substitute.
- Real **70KB noisy, quiet and delayed** Exec plus 16MiB+4 native stdout AND stderr each return accurate 16MiB API truncation PASS, as does completed output after a server SIGKILL/restart. **Native log storage retention, garbage collection and exhaustion still have no accepted bound**; the API read cap is not an Incus on-disk cap.
- **Running Exec cancellation:** tagged Incus 7.6 explicitly rejects `RecordOutput:true` combined with `WaitForWS:true`, and a detached recorded Exec does not guarantee a working `Operation.Cancel()`. This S2 implementation truthfully supports cancellation before dispatch and refuses detached/late cancellation. A separately bounded live-control WebSocket execution mode, if developed, needs its own native termination/return-code acceptance; it may not be mislabeled as durable recorded mode.
- The **earlier** single-Box run did not test IPv4/IPv6 peer packets, per-Box Internet or nesting. All now have positive **restricted-host** proof at later `b679248d` as above; **public WAN/IPv6 egress-toggle**, foreign NIC attachment and S4 peer/forward/dependent delete are still unaccepted. The prior Internet-UNSUPPORTED source state was superseded by the later independently native-verified typed NIC egress implementation; this does not imply an S4 packet policy PASS.
- Complete `TC-001–TC-010` and phase acceptance IDs require a separate explicit evidence crosswalk, including test types (unit/mocked/native/fault injected) and failures. These are NOT covered by the one successful lifecycle run.
- No installed v1 production database exists; initial schema can still change. Do not migrate or adopt old S1 disposable DBs.
- Two older, **distinct** originally UNKNOWN create tasks described in the S1 development handoff are not part of this test and were neither redispatched nor cleaned.

**Disposition:** Keep [Draft PR #130](https://github.com/vanishs/boxweave/pull/130)
open and keep protected `v1` untouched until all mandatory S2 blockers have
independent evidence, documented unsupported design decisions, and an explicit
phase-exit determination. Do not reinterpret this result as S2.5, S4 or release
acceptance.
