# v1 real functional and security acceptance matrix

**Status:** tests specified only; no v1 runtime or live PASS evidence exists. This matrix covers the current **20-tool** permanent product, not the deleted Caddy/general ACL/patch/diagnostic capabilities. Do not reduce security verification merely because the product is smaller.

Evidence: **T** typed/unit simulation; **I** real installed black-box MCP/CLI; **N** real native/nested Incus; **V** independent native post-read/Verifier. Every item remains UNTESTED until actual evidence.

## Required functional cases

| ID | Exact requirement to test | Evidence |
|---|---|---|
| A-001 | Authenticated loopback HTTP MCP initialize/list/call; missing/revoked admin and viewer credentials denied appropriately; optional operator Tunnel not managed by BoxWeave | I |
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
| A-053 | Native Box already deleted but a verified BoxWeave-owned network forward persists: `port_forward_list/delete` can safely remove that exact orphan by durable forward identity/Box tombstone without requiring the old Box to exist or touching any foreign forward | I + N + V |
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

## Adversarial combinations

Test lost MCP reply after create/Exec/forward, concurrent Box delete vs peer/forward edits, token revocation just before and after dispatch, stale guest hashes and symlink races, fixed network ACL vs peer grants, IPv6 bypass, host listener and port collisions, simulated parent restart during nested work and existing shared Incus upgrade. No native effect outside the owned set is ever permitted.

The independent Finder/Fixer/Verifier harness belongs in repository/CI/disposable Boxes, not customer runtime. Record tool name, original Tracking/Operation IDs, exact source SHA, installed bytes, native Incus version, outcome and evidence. No false PASS from a mock.

For step-by-step injector variants V-01–V-15, expected partial-state observations and test/release gate, see [fault-driven end-to-end scenario](FAILURE_DRIVEN_RELEASE_SCENARIOS.md).

See [scope](../product/SCOPE.md), [tools](../design/MCP_TOOLS.md), [network](../design/NETWORK.md), [permanent boundary](../decisions/PRODUCT_BOUNDARY.md).
