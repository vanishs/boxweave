# P1b C11 native acceptance crosswalk

Checkpoint 2026-11-26. The historical S2 evidence consumers remain unchanged:
[s2_p3_evidence_audit.py](../../scripts/s2_p3_evidence_audit.py),
[s2_p3_scoped_acceptance.json](../../scripts/s2_p3_scoped_acceptance.json),
and s2_p3_scoped_acceptance.py. Their union contains 72 distinct required
PASS markers (30 mandatory audit markers plus the grouped TC/A manifest).
This is a mapping of those inputs, not an instruction to fabricate old
markers or combine unrelated logs into a fake old acceptance run.

## Directly retained markers (48 of 72)

Exact output lines, not echoed workflow commands, observed on
d32939b877e441fb1b6b9450384792cc7852745a, N2 run 45799271822, all 25 jobs
successful. Each heading identifies the actual job; the lists are marker
names, each followed by =PASS in the original log. Earlier source evidence
does not certify a later exact-head candidate.

### 139837920546 (installed Scoped S2 SDK/native)

- S2_NATIVE_SIMULATED_WAN_NAMESPACE_ROUTE_VERIFIED_TLS
- S2_NATIVE_MCP_DISCOVERY_23_OF_23
- S2_NATIVE_MCP_TOOL_ANNOTATIONS_ACTUALLY_ADVERTISED
- S2_P1_REAL_SDK_HOSTILE_META_AND_SCHEMA_NO_NATIVE_EFFECT
- S2_NATIVE_MCP_ALL_23_CLOSED_INPUT_SCHEMAS
- S2_P1_REAL_OFFICIAL_SDK_13_TYPED_TOOL_NEGATIVES_NO_NATIVE_EFFECT
- S2_P1_REAL_AUTHENTICATED_HTTP_PRIVATE_ZERO_TTL_CACHE_CONTROL
- S2_NATIVE_INVALID_SCHEMA_CREATE_REFUSED
- S2_P1_REAL_SDK_LEGACY_TEXT_AND_STRUCTURED_RECEIPT_SAME_ID
- S2_NATIVE_QUEUED_UNDISPATCHED_CANCEL_NO_NATIVE_EFFECT
- S2_NATIVE_CONCURRENT_HISTORY_PAGING_STABLE_NEW_ROWS_VISIBLE
- S2_NATIVE_INDEXED_120_HISTORY_STATE_TIME_HMAC_SCOPE
- S2_P1_NATIVE_TWELVE_INDEPENDENT_SDK_CREATE_SAME_KEY_ONE_RECEIPT
- S2_NATIVE_SAME_KEY_CONFLICT_AND_DIFFERENT_KEY_DUPLICATE_DENIED
- S2_NATIVE_CREATE_STOPPED_UUID
- S2_NATIVE_SDK_META_FALLBACK_CONFLICT_AND_INDEXED_SEARCH
- S2_NATIVE_BOX_LIST_UUID
- S2_NATIVE_CPU_MEMORY_UPDATE_PRESERVES_DISK
- S2_NATIVE_DISK_SHRINK_DENIED
- S2_NATIVE_RUNNING_NOT_CONFUSED_WITH_GUEST_READY
- S2_NATIVE_PACKET_IPV4_IPV6_SAME_BRIDGE_DEFAULT_DENY
- S2_P1_SIMULATED_WAN_HTTPS_OFFLINE_DENIED
- S2_NATIVE_PACKET_INTERNET_TOGGLE_IPV4_IPV6_GATEWAY_EGRESS
- S2_P1_NATIVE_GUEST_ORIGIN_READY_TRUE_ACTUAL_SOCKET_AND_MCP
- S2_P1_NATIVE_GUEST_READY_RESTART_CLEARS_STALE_TRUE
- S2_NATIVE_TWO_AUTHENTICATED_ACTORS_SAME_WORK_SCOPED
- S2_NATIVE_SAME_KEY_DISTINCT_ACTOR_CANNOT_BORROW_ORIGINAL
- S2_NATIVE_EXEC_RECONNECT_EXIT_7_STDOUT_STDERR
- S2_NATIVE_DUAL_STREAM_16M_OUTPUT_READ_BUDGET
- S2_NATIVE_CANCELLED_LONG_POLL_IS_READ_ONLY
- S2_NATIVE_EXEC_UNSUPPORTED_CANCEL_TRUTHFUL
- S2_NATIVE_DELETE_VERIFIED_AND_REPLAY
- S2_NATIVE_NAME_REUSE_OLD_UUID_AND_DELETE_KEY_DENIED
- S2_NATIVE_ASYNC_PROGRESS_OPTIONAL_NO_FAKE_EVENTS

### Other exact witnesses

| Job | Marker |
| --- | --- |
| 139837920435 | S2_P1_PREVIOUS_OFFICIAL_MCP_SDK_NATIVE_CREATE_UPDATE_EXEC_DELETE |
| 139837920219 | S2_P1_REAL_INSTALLED_OPERATION_LIST_INFLIGHT_REVOCATION_AND_EXPIRY_NO_LEAK |
| 139837920219 | S2_P2_STALLED_NATIVE_OUTPUT_CANCEL_NO_NATIVE_CHANGE |
| 139837920373 | S4_A_NATIVE_UUID_PREATOMIC_INDEPENDENT_INCUS_CONFIRM |
| 139837920373 | S4_A_REAL_NATIVE_PREATOMIC_TWELVE_SDK_DELETE_REFUSED |
| 139837920373 | S4_A_REAL_NATIVE_OVERLAP_ATOMIC_CREATE_DELETE_TOMBSTONE |
| 139837920373 | S4_A_REAL_NATIVE_PREATOMIC_CONCURRENT_DELETE_FULL_LIFECYCLE |
| 139837920365 | S2_P2_GUEST_REAL_EXEC_READY_POSITIVE_CONTROL |
| 139837920098 | S2_SQLITE_ACTUAL_ISOLATED_EXT4_ENOSPC_RECOVERY_UNKNOWN_NO_REPLAY |
| 139837920190 | S2_MCP_REAL_GUEST_QUOTA_ORIGINAL_RECEIPT_OUTPUT_EXIT_ONE |
| 139837920208 | S2_NATIVE_CRASH_DISPATCHED_ORIGINAL_RECEIPT |
| 139837920208 | S2_NATIVE_COMPLETED_OUTPUT_AFTER_SQLITE_RESTART |
| 139837920208 | S2_NATIVE_PROCESS_KILL_UNKNOWN_REPLAY_DENIED |
| 139837920208 | S2_NATIVE_CRASH_ORIGINAL_GUEST_EFFECT_EXACTLY_ONCE |

The six ordered operation_list debugger witnesses from LIST_WITNESS are
also present in the same order in 139837920219: ONE_SHOT_WITNESS=revoke,
SERVICE_CALL_COUNT=revoke:1, ONE_SHOT_WITNESS=expire,
SERVICE_CALL_COUNT=expire:1, BREAKPOINT_STDOUT_COUNT=2,
BREAKPOINT_FILE_ORDERED_HITS=revoke,expire (all use S4_A_OPERATION_LIST_
prefix). Independent compiled-call cut and exact fixed-ledger absence proof
are retained. These are additional to the 72-marker union.

## Changed markers (24 of 72)

Grouped rows explicitly list every nonidentical old marker. CI_VERIFIED
means historical exact source/job evidence, not final current-head closure.
BOX_VERIFIED means source/fixture tests only.

| Old marker(s) | Current witness, disposition and limit |
| --- | --- |
| S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS | CI_VERIFIED d32939b 139837920546: P1B_SCOPED_S2_NATIVE_PROBE, FIXED_LEDGER, exact SDK plus independent native/actor/receipt proof; old aggregate remains until consumer conversion. |
| S2_NATIVE_SQLITE_FOREIGN_CORRUPT_FAIL_CLOSED; S2_NATIVE_SQLITE_OLD_MULTIPLE_VERSION_AND_FOREIGN_TABLES_DENIED | BOX_VERIFIED new installed Doctor five-layout refusal, expected schema reason, original bytes and no sidecars, fixed ledger/native/guard unchanged; same two old markers required in new N2 plus independent no-effect marker. Existing state entry-point tests retain old/new/multiple/foreign/missing-index originals. Native CI_PENDING. |
| S2_NATIVE_CREATE_ACK_LOSS_ORIGINAL_UNKNOWN; S2_NATIVE_CREATE_ACK_LOSS_POSTRESTART_NO_SECOND_POST | CI_VERIFIED d32939b 139837920343 P1B_SCOPED_ACK_LOSS=PASS create; exact same original, no redispatch, unknown retained, independently observed native effect. |
| S2_P1_SIMULATED_WAN_HTTPS_POSITIVE_TLS_BEFORE; S2_P1_SIMULATED_WAN_HTTPS_ON_OFF_ON_GUEST; S2_NATIVE_SIMULATED_WAN_SERVER_INDEPENDENT_ON_OFF_ON_TWO_HITS | Old reserved IPv4 range positive is incompatible with Scoped default reserved-range denial. d32939b 139837920546 reserved deny both states and independent zero hits; 333b6b5 S25 139814069701 routed IPv6 HTTPS on/off/on and independent NIC/packet/peer denial both layers. This is a scoped policy replacement, NOT public IPv4 positive. Public guest WAN remains UNVERIFIED (host TLS PASS, guest TLS UNVERIFIED). No physical WAN waiver. |
| S2_P1_TWO_CONTROLLER_INDEPENDENT_NATIVE_AND_MCP_ISOLATION; S2_P1_TWO_CONTROLLER_CROSS_TOKEN_AUTH_DENIED | BOX_VERIFIED 1fbbd07 same-name distinct native UUID and original, fixed two installations and cross-token/typed foreign requests, independent no-admission readback. S25 139846416445 failed at signed package index before target, 69ac198 retry fixture CI_PENDING. Same-daemon foreign Project and packet/socket proofs remain independent. No support for two Scoped controllers sharing one daemon. |
| S2_P1_FOUR_REAL_PRIVATE_READ_TOKEN_RACES_NATIVE_POSTREAD_EMPTY | CI_VERIFIED d32939b 139837920219: eight exact HTTP 200 native cuts (four methods revoke/expire), S2_P1_FOUR_REAL_MCP_NATIVE_PRIVATE_READ_BEARER_REVOKE_NO_LEAK and EXPIRY_NO_LEAK, independent cleanup/no-effect; operation_get and signed cursor also pass. |
| S2_P2_REAL_MALFORMED_EXEC_NATIVE_202_UNKNOWN_RESTART_CLEANUP | CI_VERIFIED d32939b 139837920365 actual malformed native final result, exactly one guest effect, fixed ledger restart retains UNKNOWN and claim. Do not admin-delete unproved original; dispose of exclusive runner, not runtime cleanup. |
| S2_P2_ALTERNATE_POOL_LIVE_EXPANDED_ROOT_MATCHES_UUID_POOL_QUOTA | BLOCKED: source-tested staged alternate-pool tree 8a14a0e10979bfca5c54b7f82632aa0f579c7277 retained in separate original worktree, publication security check blocked. Actual native test not run, no substitute. |
| S2_P0_NATIVE_202_COMBINED_WITH_SQLITE_COMMIT_FAULT_BOTH_PHASES; S2_P0_REAL_NATIVE_EFFECT_AFTER_ACTUAL_SQLITE_SYNC_KILL | OPEN: old test couples real accepted Update 202 with kill at NativeAck WAL sync before/after (not the terminal transaction). New actual Create/Delete/Exec terminal WAL cuts and Update ACK SQL-abort/transport tests cover constituent behavior, NOT exact combined Update NativeAck cut. P1b-06d11 conversion BOX_VERIFIED: two hosted modes with original restricted 202 proxy, precise ACK marker WAL cut, SDK and independent native/ledger oracle. Actual CI pending; keep old gate. Old unbound Create review component is covered by a65e5e2 139839963173 atomic rollback and reviewed completed-original reconciliation. |
| S2_GC_APPROVED_SEVEN_DAY_COMPLETE_NATIVE_MCP_CYCLE | CI_VERIFIED d32939b 139837920098 actual seven-day two native stream deletion/404/audit, real lost ACK sibling locked, reviewed exact native reobservation and release without retry, durable audit, public typed expiry versus elapsed/unproved physical expiry, independent native absence. |
| S2_P1_INSTALLED_RECEIVING_MIDDLEWARE_REDACTION_AND_TYPED_NEGATIVES | CI_VERIFIED d32939b 139837920546 P1B_SCOPED_S2_REDACTION plus real hostile meta/schema and 13 typed negatives, no native effect, exact service journal sentinel absence. |
| S2_SQLITE_WAL_FSYNC_BEFORE_REAL_KILL_UNKNOWN_NO_REPLAY; S2_SQLITE_WAL_FSYNC_AFTER_REAL_KILL_UNKNOWN_NO_REPLAY | CI_VERIFIED actual terminal commit cuts Create before/after 139837920767/139837919853, Delete 139837920350/139837920527, Exec 139837920565/139837920381. Original native and SQLite atomic proof, before may durably commit or remain UNKNOWN, after requires complete durable commit, no replay. Old synthetic ACK write WAL behavior additionally requires exact Update native-ACK conversion above; not a physical outage claim. |
| S2_P1_REAL_STOP_AND_DELETE_NATIVE_202_ACK_LOSS_NO_REPLAY | CI_VERIFIED d32939b 139837920687 actual Stop 202 dropped, stopped exact UUID, original UNKNOWN retained; 139837920264 root-assisted Delete ACK persistence fault, native absence independently observed and same original after restart. Different Scoped Root receipt boundary, no invented native IDs. |
| S2_NATIVE_STALE_ETAG_ORIGINAL_UNKNOWN_AND_LOCKED | CI_VERIFIED d32939b 139837920269 P1B_SCOPED_UPDATE_TRANSPORT=PASS etag (also ack/postread), original held and native no duplicate, unchanged pinned ACL. |
| S2_P0_GENERAL_UNKNOWN_STAGED_ROOT_CLI_DUAL_CONFIRMATION; S2_P0_GENERAL_UNKNOWN_AUDIT_DURABLE; S2_P0_GENERAL_UNKNOWN_LOCAL_RELEASE_NO_RETRY | Old generic challenge/release leaving UNKNOWN but permitting new work is obsolete for Scoped: recover.go requires reviewed route except specific GC. CI_VERIFIED installed reviewed D4 exact original settlement/no replay/audit and duplicate receipt (a65e5e2 job 139839963173); scoped lifecycle rollback completed-original recovery same job. BOX_VERIFIED new exact real Update UNKNOWN negative for old inspect/challenge/release route, no audit/claim/original/native change, installed CI_PENDING. GC-specific reviewed release is retained, not a blanket unlock. |

## Remaining consumers and disposition

- All old S2/S2.5 producers, raw witness audit, grouped acceptance manifest,
  release gates and quality requirements remain intact. The old acceptance
  scripts expect a single source and binary, not concatenated runs. During
  P4 convert the consumer against exact new run/job evidence, new marker
  requirements, negative calibration and declared obsolete contracts.
- S2.5 original nested and foreign packet/parent socket/maintenance
  assertions are retained by v1-pr191-s25-scoped.yml; 333b6b5 job 139814069701
  passed, exact same-name dual controller and index repair still need
  native acceptance. Successful Legacy upgrade/rollback is explicitly
  OUT_OF_SCOPE_170 for fresh-only Scoped, never reported PASS.
- Source test suite and the detailed P0 producer/consumer inventory remain
  applicable; this map covers the 72 consumed S2 marker union and ordered
  list proof, not every future retired source assertion. Per-file deletion
  decisions and P4 final consumer conversion remain OPEN.
- Physical hardware power, reboot, true public WAN, LAN and stable
  maintenance are not proved by this map. No relaxation of existing
  restrictions or retirement waiver. Do not close P1b until affected native
  witnesses and remaining dispositions are independently verified.

P1b-06d11 follow-up: 69ac198 S25 job 139851764466 actual same-name distinct-native, cross-token, two fixed-ledger/ACL/guard and foreign-packet proof PASS. Combined Update NativeAck WAL test source verified, actual CI pending. No public WAN or alternate-pool waiver.

P1b-06d12: [exact checkpoint and new CI failures](CONSOLIDATION_P1B_06D12_CHECKPOINT.md). Prior schema and obsolete-route refusal independently verified; real ACK cut passed but restart failed both modes; corrective source BOX_VERIFIED, native CI required. Public WAN and alternate pool remain open. No waiver.
