# v1 real functional and security acceptance matrix

**Status:** tests specified only; no v1 runtime or live PASS evidence exists. This matrix covers the current **20-tool** permanent product, not the deleted Caddy/general ACL/patch/diagnostic capabilities. Do not reduce security verification merely because the product is smaller.

Evidence: **T** typed/unit simulation; **I** real installed black-box MCP/CLI; **N** real native/nested Incus; **V** independent native post-read/Verifier. Every item remains UNTESTED until actual evidence.

## SDK-contract acceptance prerequisites

These are **UNTESTED** design gates from the [official SDK review](../research/OFFICIAL_SDK_FIT_2026-11-07.md), not additional user-facing MCP tools: (1) native read-only operation wait never calls `Cancel`; (2) MCP runtime cannot exercise unrelated Incus admin operations despite project selection; (3) owned `volatile.uuid` is rechecked across name reuse; (4) no conditional guest file write/delete claims atomic CAS without point-of-use proof; (5) P0 host forwarding uses instance-owned proxy device, with actual native TCP/UDP/LAN witness and NIC-level same-bridge IPv4/IPv6 policy; (6) real SDK stateless modern `server/discover` and legacy negotiated MCP flows, auth and disconnect/recovery are tested. No native `N` or independent `V` status may be assigned for source inspection alone.

## Native SDK alignment checks (not yet executed)

Read the [20/20 native SDK mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) and verify real behavior before assigning N/V PASS. Use a dedicated non-root OS user in `incus`, not `incus-admin`; prove the native Project restricts foreign/admin calls while approved proxy/ACL/nesting function. Verify `api.StatusErrorMatch`, Exec `RecordOutput` time to first Operation ID and output reconnection, SDK `auth.RequireBearerToken` revocation/expiry, localhost Host protection and default 4 MiB request limit, plus explicit Origin/trusted-proxy and stateless/legacy protocol negotiation. None of this was executed by the research.

## Second-pass upstream SDK proof cases — ALL UNTESTED

The [tagged-source gap report](../research/SDK_GAP_CLOSURE_2026-11-07.md) identified practical gaps beyond API presence. These are **native/client validation prerequisites**, not fabricated PASSes or additional public MCP tools.

| Gate | Required negative/positive evidence | Required evidence level |
|---|---|---|
| SDK-G01 | With one typed SDK MCP server, two distinct Bearer credentials concurrently invoke a Tool. The handler sees the **current** identity in `req.Extra.TokenInfo` (not another request's context). Viewer mutation fails without native effects; admin succeeds on only its owned Box; revoked/expired credentials fail for new HTTP calls. Test modern Stateless plus legacy initialize. | T + I |
| SDK-G02 | Non-root `incus`-only user and prepared restricted Project may act on its exact owned instances, ACL and proxy, but may **not** access foreign project/host admin/resources. New Project's `features.*`, isolated default Profile **root disk** and approved network/storage are correctly prepared **before first instance**. | N + V |
| SDK-G03 | Bridge NIC-attached ACL defaults to deny between owned Boxes; native spoof filters cover source MAC/IPv4/enabled IPv6. Pair A→B allows A-initiated flows and replies but denies B-initiated flows. After A/B IP change or deleted/recreated target, old grant does not transfer; test already-established flows during revoke. Default DNS/DHCP and authorized Internet egress remain viable. | N + V packet witness |
| SDK-G04 | `box_file_apply(create)` refuses an existing regular file **without truncating it** under concurrency, including final symlink/hardlink cases. SDK native upload `O_TRUNC` path is **not** acceptable proof. Test SFTP `OpenFile(O_EXCL)` candidate. Conditional replace/delete prove point-of-use CAS or deliberately return `UNSUPPORTED` with owner-visible P0 blocker. | N + V |
| SDK-G05 | `box_exec` first response includes original durable operation identity **before long command finishes**, while SDK worker captures bounded output and actual native exit after MCP disconnect. Test `RecordOutput` with writers versus selected streaming mode; do not mistake ordinary instance logs for private native exec-output logs. | I + N + V |
| SDK-G06 | `operation_get(wait_ms)` times out without native cancellation; concurrent read polls do not alter operation execution or shared transport settings. Prefer bounded `GetOperation` polling until `GetOperationWait` transport behavior is proven. | T + N + V |
| SDK-G07 | With parent NIC spoof filtering active, independently installed nested Incus works through its own bridge/NAT without granting new parent-interface MAC or disabling host security; show accurate refusal where impossible. | N + V |
| SDK-G08 | A native feature is advertised only after actual daemon extension/permission check; verify `container_exec_recording` and `container_exec_user_group_cwd` behavior for requested Exec variants, including unsupported UID/GID/cwd. | N + V |

Results belong to the **exact test version and environment**, not generic upstream stability. The previous Box SDK probe ended in guest/gateway UNKNOWN and cannot close these gates.

## Required functional cases

| ID | Exact requirement to test | Evidence |
|---|---|---|
| A-001 | Authenticated loopback HTTP MCP `server/discover` under negotiated 2026-07-28 **and legacy initialize/list/call** where applicable; explicit Origin/Host validation, missing/revoked admin and viewer denied; optional operator Tunnel not managed by BoxWeave | I |
| A-002 | Local install/status/upgrade/bootstrap/doctor and protected credential recovery; no remote host admin route | I |
| A-003 | Viewer role may read owned Box metadata/files/output but cannot create/update/delete/Exec/peer/forward/cancel; admin only manages owned resources | I + V |
| A-004 | Exact native Box UUID/incarnation owned identity for list/get/create/read; no foreign host/Incus file or Box disclosure | I + V |
| A-005 | Create reviewed Ubuntu guest with defaults, start=true verified ready; start=false leaves stopped; opt-in nesting | I + N |
| A-006 | CPU/memory/safe disk growth, nesting/internet typed update, rejection of shrink/raw Incus/host mount/device | I + V |
| A-007 | Start, stop, delete with explicit destructive confirmation and actual native after-state readback | I + V |
| A-008 | Reused Box name/IP or new incarnation cannot inherit former ownership/peer/forward claims | I + V |
| A-009 | Agentless guest-root Exec typed argv/cwd/env/timeout/outcome observed, with no host shell or installed management client | I + N |
| A-010 | Exec client disconnect/reconnect recovers same original operation/outcome, bounded stdout/stderr and observed nonzero exit as completed failed command | I + V |
| A-011 | Guest file list/read bounded, meaningful hash/coverage, no host/foreign special-file access | I + V |
| A-012 | `box_file_apply(create|replace|delete)` only; safe revision and native post-state, no internal patch engine or recursive deletion | I + V |
| A-013 | Concurrent guest writer, unsafe path/symlink/hardlink/device and stale hash reject unsafe file changes; no fake atomic CAS | I + V |
| A-014 | Guest Git/Exec can clone and edit a normal code repository; no required BoxWeave Git/upload/patch service or leaked private credentials | I |
| A-015 | Full create -> clone/read -> edit -> failing Go test -> inspect original output -> fix -> passing test -> independent native readback | I + N + V |
| A-016 | Every mutation requires caller reusable idempotency key, rejects missing/conflict, returns original Tracking+Operation on same retry without duplicated effect | T + I |
| A-017 | Go context/Tracking ID available throughout MCP/domain/SQLite/worker/native adapters with separate request/idempotency/Incus IDs | T + I |
| A-018 | Crash/lost native result/timeout preserves UNKNOWN and original intent; no blind redispatch | T + I + V |
| A-019 | `operation_get(wait_ms)` read wait is bounded, returns changed state and never cancels original native work on timeout/disconnect | I |
| A-020 | `operation_list` and `operation_output_get` scope/retention/truncation protect original output and ownership | I + V |
| A-021 | `operation_cancel` proves queued-undispatched cancellation; active Exec stop is only claimed with native termination proof; unsupported and UNKNOWN are honest | I + V |
| A-022 | Redacted phase/error/verification summary in `operation_get`, early failure with Tracking ID and local CLI logs; no `diagnostic_get` endpoint | I |
| A-023 | Private direct Box A->B IPv4 and enabled IPv6 denied by native evidence under supported Bridge/OVN NIC topology | I + N |
| A-024 | Unsafe network topology or IPv6 bypass fails closed; any controlled IPv6 disable leaves foreign host/network unchanged | I + N + V |
| A-025 | `box_peer_set(A,B,true)` allows directional private A-initiated connections/replies; `false` revokes; B independently initiating A remains denied unless granted | I + N + V |
| A-026 | Native Box IP change, identifier spoof, unmanaged instance and foreign NIC/ACL cannot acquire or lose another Box's peer rights | I + N + V |
| A-027 | Internet egress defaults allowed; per-Box deny toggle works without changing unrelated host networking | I + N |
| A-028 | `port_forward_create` maps owned Box TCP service to explicit host port (loopback by default when supported), external bind needs explicit admin action | I + N + V |
| A-029 | UDP forward uses verified native Incus facilities, host port collision/unsupported network denied, no arbitrary target IP/native proxy takeover | I + N + V |
| A-030 | `port_forward_list/delete` show/remove only owned forwards, survive/reconcile service restart and never disturb foreign forward | I + N + V |
| A-031 | No Caddy daemon/config/installer dependency or published HTTP route tools; operator-managed proxy/Tunnel can separately consume native host-published port | I + V |
| A-032 | Existing compatible Incus reused with restricted BoxWeave Project, actual version/extensions recorded, foreign projects and instances untouched | I + V |
| A-033 | Missing/old approved Incus installed/upgraded by explicit local workflow; shared disruptive update requires separate approval, truthful failure/rollback and no automatic host reboot | I + V |
| A-034 | Corrupt/lost ownership SQLite blocks dangerous mutations, independently reviewed local recovery preserves foreign resources | I + V |
| A-035 | L0/L1/L2 use same installed BoxWeave code and 20 MCP tools if supported; each manages only its own Boxes and native ports, no parent token/socket inheritance | I + N + V |
| A-036 | An inner BoxWeave candidate can be installed/upgraded and run complete Agent loop without changing the outer layer or Caddy (absent) | I + N + V |
| A-037 | All installed MCP tools and local CLI leaves receive positive/negative tests, native after-state/oracle and explicit SKIP/BLOCKED/UNKNOWN outcomes | I + V |
| A-038 | Exact verified Go SDK/native package versions, binary hashes and immutable GitHub new-version release with no automatic live host deployment | T + I |
| A-039 | Initial Ubuntu LTS guest works; any additional guest distro is separately validated before added to supported images; unsupported cases honest | I + N |
| A-040 | All excluded product features are absent: no generic ACL editor, per-Box RBAC, Caddy/domain/HTTPS, native snapshot tools, automatic wake or guest text patch engine | I |
| A-041 | A Hello World guest service is installed via ordinary guest Exec/service manager and remains reachable after the initiating Exec is finished; no BoxWeave in-guest management Agent/supervisor | I + N |
| A-042 | Explicit L0 LAN interface bind (not default loopback) allows real HTTP access from **another machine on the same LAN**, with matching response and native forward identity; host firewall/routing failures correctly classified | I + N + V |
| A-043 | `box_delete` with owned forward and peer dependencies discloses cleanup scope, locks resource identities, removes/verifies only owned dependencies and then native Box; no leftover exposure on success | I + N + V |
| A-044 | Inject forward cleanup success followed by instance-delete failure: Box remains but mapping absent, Operation records partial substeps/cause, `box_get` and `port_forward_list` agree with observed native state | I + N + V |
| A-045 | Inject native delete success with lost MCP reply/SQLite commit: duplicate caller key returns original operation, no repeated native delete or stale forward claim; unresolved state stays UNKNOWN until independent verification | T + I + V |
| A-046 | Delete vs concurrent forward/peer mutation uses exact UUID/incarnation locks and correct dependency/tombstone order; another Box and foreign Incus resources stay unchanged | T + I + V |
| A-047 | `operation_get` provides sanitized phase/native error classification/observed dependent state/evidence gap/recovery hint/build SHA for failed delete, with no arbitrary host logs, raw trace or credentials | T + I |
| A-048 | Inner candidate MCP is contacted through guest-local authorized localhost client via parent guest Exec, with independent inner credentials never exposed in argv/stdout; parent cannot directly reach inner loopback via guest bridge IP | I + N + V |
| A-049 | L0/L1 independently authorized native forwards yield actual nested Hello World HTTP from separate **physical LAN witness**; failure inside one layer stays attributed there and no automatic parent orchestration occurs | I + N + V |
| A-050 | Old first-version installed bug is reproducible in controlled lab; corrected candidate installable **before** formal Release and verified with exact binary/source digest against original delete regression + negative/UNKNOWN cases in nested L1/L2 | I + N + V |
| A-051 | Formal second-version CI publication occurs only after independent exact-artifact native/nested gate; missing evidence or artifact digest mismatch blocks GitHub Release, tag is immutable and published bytes match tested candidate | T + I + N + V |
| A-052 | After formal Release, original L0 remains unchanged with its original failed Box until explicit host-local upgrade and safe reconciliation; no candidate test or GitHub tag silently cleans the old incident | I + V |
| A-053 | **CONDITIONAL (not P0 proxy-only):** only if a separate network-forward backend is explicitly implemented, native Box gone but owned network-forward persists; safely remove by exact native identity/tombstone, never by foreign/stale name. For P0 proxy-device backend, independently prove no stale owned mapping or foreign change after Box delete. | I + N + V (conditional) |
| A-054 | Host-local manual UNKNOWN review binds a short-lived single-use challenge to exact original Operation, immutable target and claimed resource snapshot; wrong, expired or changed review refuses with zero side effects | T + I + V |
| A-055 | Explicit second operator risk confirmation atomically audits and releases only a selected UNKNOWN claim; audit-write failure retains claim; original receipt/key remain UNKNOWN and do not re-dispatch; no remote MCP recovery route exists | T + I + V |

## Incident-derived acceptance extensions

| ID | Existing tool/CLI behavior to test | Evidence |
|---|---|---|
| A-056 | Create an owned instance stopped with inherited NIC; verify private IPv4/enabled IPv6 isolation installed and effective **before first start/autostart**, otherwise refusal with exact residual identity | I + N + V |
| A-057 | `box_get` and `box_list` show consistent separate native RUNNING vs unverified Guest Ready/create progress; a failed/interrupted create cannot be reused as ready | I + N + V |
| A-058 | Delete during async create or delayed create completion preserves original intent, exact UUID and correct cleanup; neither false create failure nor duplicate dispatch | T + I + V |
| A-059 | ACL desired/in-DB, attached-to-NIC and actually enforced are distinguished; same-bridge IPv4/IPv6 packets and A->B replies/B->A new initiations prove peer semantics | I + N + V |
| A-060 | Requesting a nondefault root storage pool with omitted disk size uses that *effective* pool; no silent Profile substitution; unsupported/missing pool refused before native effects | I + N + V |
| A-061 | `box_update(cpu-only)` preserves memory/disk/network limits; memory-only preserves CPU; ETag conflict and complete effective readback validated | T + I + V |
| A-062 | Missing/malformed Exec `return` metadata is never exit 0; observed 0/nonzero, empty stdout, stderr-only and transport failure remain distinct | T + I + N + V |
| A-063 | Disconnect caller during quiet and continuous-output Exec; original output/exit/status survive or honestly remain UNKNOWN; no duplicated command or stream-induced termination | I + N + V |
| A-064 | At/over byte limit for both Exec streams and file reads: bounded memory/partial coverage, native/transport error not hidden by truncation, no false full-file hash | T + I + V |
| A-065 | Validated safe guest file create/replace/delete rejects symlink, hardlink, special file, path escape, stale hash and concurrent writer; rename alone is not accepted as conditional CAS proof | T + I + N + V |
| A-066 | A delayed read-side `Instance not found` is boundedly rechecked without a second mutation; genuine deletion and same-name replacement cannot inherit old ownership | T + I + V |
| A-067 | Already-running Start/already-stopped Stop yield typed target-state result with original key semantics; uncertain dispatch must not become a no-op proof | T + I + V |
| A-068 | CPU NaN/Inf/oversized or overflowed quantities, negative/unsafe memory/disk and absent capabilities fail before any effect; effective config verified | T + I + V |
| A-069 | Delete when owned forwarding metadata temporarily unavailable, then restart/recovery: old tombstoned forward never reappears and no foreign forward changes | T + I + N + V |
| A-070 | Forward directly after create only with verified native target address/required state; guest, host and LAN witnesses distinguish listen address/hairpin/firewall failures | I + N + V |
| A-071 | Fresh Incus daemon missing pool/network, selected pool on separate disk, incorrect claimed free space and unmounted data directory: installer/doctor report true limitations without modifying foreign resources | I + N + V |
| A-072 | Interrupted install/upgrade rollback of binary/service is independently read back; rollback failure is reported as failure, not successful recovery; SQLite history survives restart | T + I + V |
| A-073 | Release workflow rejects unsafe version/tag inputs and verifies tag, source commit, tested binary digest and embedded version; no release from docs-only baseline | T + I + V |

The A-056–A-073 cases are **additional test cases for existing 20 tools/local CLI, not new user-facing capabilities**. All remain **UNTESTED** until actual appropriate evidence; historical Containarium commits are not v1 native PASS. [Evidence index](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Small-implementation-derived contract tests

| ID | Existing v1 behavior to test | Evidence |
|---|---|---|
| A-074 | `box_update` distinguishes omitted vs explicitly cleared vs supplied typed values; unsupported clear rejected, omitted CPU/memory/disk/NIC preserved; no raw Incus config surface | T + I + V |
| A-075 | Every supported Incus SDK method/field/enum and actual required API extension is mapped and exercised; unknown/ignored values and version drift cannot yield a false mutation success | T + I + V |
| A-076 | Native terminal ACK and returned key presence are insufficient: readback compares actual requested **values**, effective inherited devices, untouched limits and exact owned UUID; ambiguous postcheck retains original Operation evidence | T + I + V |
| A-077 | Official MCP Go SDK in-process client + injected narrow Incus fake verifies declared schemas, missing/unknown args, authorization and native errors; no mock alone is counted as a native PASS | T + I |

These four cases are **UNTESTED refinements to existing v1 interfaces**, not new tools or expansion. Evidence: [two Incus MCP code reviews](../research/INCUS_MCP_IMPLEMENTATIONS_2026-11-07.md).

## DevSpace-history-derived negative controls (existing behavior only)

| ID | Existing v1 behavior / external acceptance method to test | Evidence |
|---|---|---|
| A-078 | Inject native success or partial owned forward/Box deletion followed by SQLite commit failure **and compensation failure**: preserve both failures, original native/Tracking/Operation identities, observed partial/tombstone state and UNKNOWN lock where unproven; no successful rollback claim or automatic mutation replay | T + I + N + V |
| A-079 | For a selected current installed MCP/CLI contract, record a sanitized failing request/output, replay it through the built candidate and fake/native adapter with precise lost-reply/cancel/fault timing, and prove a **known-bad or injected negative control makes the test fail**. Record simulation vs installed/real native evidence distinctly; independent Verifier checks same exact candidate digest and original failure | T + I + N + V |
| A-080 | Unknown/newer or same-version-different-name SQLite migration history, concurrently starting/upgrading authorized local instances and interrupted migration: fail closed without clobbering original configuration/Operation/audit history; installed service does not claim readiness until real recovery succeeds | T + I + V |

These are **UNTESTED test and harness refinements**, not additional MCP tools, agents, databases or runtime services. Record/replay and fake-native calls do not count as N or V by themselves. Historical source: [DevSpace two-dimensional audit](../research/DEVSPACE_MAIN_HISTORY_2026-11-07.md). The A-001–A-077 contracts remain authoritative, with no duplicate implementation of already-required UNKNOWN/cancel semantics.

## Main migration-history regression controls (existing behavior only)

| ID | Existing v1 behavior / negative regression to test | Evidence |
|---|---|---|
| A-081 | On one unchanged, owned native Box, deliberate `stop → start → stop` with **three distinct idempotency keys** performs and verifies both stops; an exact duplicate of any original key returns only its original Operation with zero new native dispatch, and same key/different payload is rejected. Pending/UNKNOWN action prevents conflicting mutations without permanently banning later legitimate actions | T + I + N + V |
| A-082 | Inject an accepted resize/update and then remove trustworthy native effective-resource readback: `operation_get`, the MCP mutation response and host-local operator inspection must **not** claim verified success while the original claim is UNKNOWN/held. A second mutator is blocked; after exact UUID and requested values are independently proven, public state and lock disposition converge without replay. Verify missing/partial readback is never silently treated as desired-state proof | T + I + N + V |

**Both are UNTESTED negative controls for existing 20 tools/local CLI, not new operations, tables or user-facing APIs.** The source-confirmed counterexamples are [main v0.50.4 lifecycle/claims](../research/MAIN_HISTORY_LESSONS_2026-11-07.md#owner-retrospective-and-pinned-v0504-re-review-2026-11-07); the older A-016/A-018/A-076 requirements remain binding.

## Adversarial combinations

Test lost MCP reply after create/Exec/forward, concurrent Box delete vs peer/forward edits, token revocation just before and after dispatch, stale guest hashes and symlink races, fixed network ACL vs peer grants, IPv6 bypass, host listener and port collisions, simulated parent restart during nested work and existing shared Incus upgrade. No native effect outside the owned set is ever permitted.

The independent Finder/Fixer/Verifier harness belongs in repository/CI/disposable Boxes, not customer runtime. Record tool name, original Tracking/Operation IDs, exact source SHA, installed bytes, native Incus version, outcome and evidence. No false PASS from a mock.

For step-by-step injector variants V-01–V-15, expected partial-state observations and test/release gate, see [fault-driven end-to-end scenario](FAILURE_DRIVEN_RELEASE_SCENARIOS.md).

See [scope](../product/SCOPE.md), [tools](../design/MCP_TOOLS.md), [network](../design/NETWORK.md), [permanent boundary](../decisions/PRODUCT_BOUNDARY.md).
