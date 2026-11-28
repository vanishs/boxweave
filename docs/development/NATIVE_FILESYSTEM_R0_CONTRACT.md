# Native Filesystem R0 — proposed implementation contract

Status: R0 decision candidate, based on [native/SDK evidence](NATIVE_FILESYSTEM_R0_LAB.md) and [design](NATIVE_FILESYSTEM_DESIGN.md). This is not a product contract change or R0 independent signoff. The current 23 Tools, Schema 7 and existing fail-closed write semantics remain authoritative until owner approval and implementation acceptance.

## 1. Minimum useful delivery

Choose Resource-native interface C with Agent-facing operations from B. One bounded Go filesystem service and official Incus SFTP, one controller/SQLite and existing Operation system. Prefer ordinary MCP core over speculative extensions. Target Incus daemon/SDK 7.7, SFTP 1.13.13, Go 1.26.10; both MCP SDK 1.8 and 1.9-pre1 support tested core. Owner-selected 1.9-series development baseline is viable; retain 1.8 stable fallback if prerelease-specific value is absent or compatibility fails. Exact production dependency selection belongs to P0; no host update or dependency commit in R0.

Prioritize file discovery, text/line read, bounded search, exclusive create and mkdir, stopped Guest access and streaming copy. These are useful without conditional overwrite, append, notifications or experimental Files uploads. No guest Agent or Exec substitution. Do not couple them to a future CAS solution.

## 2. Public API candidate (25 total Tools, owner approval required)

| Tool | Initial supported candidate modes | Later / deferred |
| --- | --- | --- |
| box_file_list | Existing list, stat, bounded tree, batch stat | Unbounded stable snapshot traversal |
| box_file_search | Filename, text, bounded RE2 search with explicit coverage | Persistent index, arbitrary command/regex engine |
| box_file_read | Existing bytes, text, line range, head/tail, bounded batch, digest, resource URI, read-only edit preview | Unbounded read, pretending observed content is a snapshot |
| box_file_apply | Existing create; exclusive mkdir; bounded ordered new-file batch | Conditional replace/delete, in-place edit, append and move until guarantees and scope are approved |
| box_file_transfer | Same/across owned Box copy, upload begin/chunk/commit/abort, optional download preparation | Experimental Files adapter; advanced incremental sync |

All tools have strict tagged schemas; reject irrelevant/unknown fields, ambiguous path/URI combinations and conflicting identifiers. Existing callers without read/list mode retain original behavior and existing result fields. Existing create semantics and error classes remain compatible. Preview belongs to a read-only Tool, not mixed mutation annotations. Transfer status is a read via existing operation_get or box_file_read transfer-status view; download bytes via read/Resources or future negotiated Files adapter. Any transfer action that reserves, writes or releases durable state is a mutation with its own original caller key/receipt. No per-chunk external polling requirement when the synchronous bounded acknowledgment can include the verified original state. If outcome is pending/ambiguous, return original receipt and next safe read action. No model-generated shell script as implementation.

Agent-friendly result common fields: tracking_id, exact box_id/incarnation, normalized path or URI, capability/limit, covered bytes/entries, ended/complete, next_cursor/offset where applicable, and concise typed errors. Distinguish FILE_NOT_FOUND vs refused without enumerating foreign resources. Preserve original UNKNOWN behavior, avoid new terminal state vocabulary. Batch is bounded and ordered; report per-item state and effects, not atomic multi-file transaction. Initially accept source/destination only within same owned layer/project.

## 3. Resources

Template candidate boxweave://boxes/{box_id}/{incarnation}/fs/{+path}; one canonical path encoding, exact UUID and incarnation; generic template listing, no full dynamic filename enumeration. Path segments must have a single accepted percent-encoding representation, decoded once and validated against existing path policy; reject decoded slash within a segment, dot/traversal, controls, invalid UTF-8, ambiguous query/fragment, trailing slash ambiguity, wrong incarnation and noncanonical roundtrip. Root descriptor is discovery metadata, not implicit permission to expose unrestricted root listing. Separate URI namespace for bounded views and private transfer references; no URI as bearer credential, no arbitrary client URL fetch. Exact RFC6570 behavior needs deterministic tests before wire freeze.

Use current per-request authenticated Actor and native owned Box resolution on resources/read, including post-read checks; static resource/template discovery only. Private zero TTL, no assumption old clients honor cache hints. Resource links in tool output carry MIME type and known size; text or binary bytes in core ResourceContents remain JSON. Cap each resource and entire response; large resources expose metadata/link plus bounded read or a supported transfer path, not silently truncate or send the entire file. Tool-only compatibility is mandatory. Defer subscriptions and notification support until actual negotiated client stream and revocation tests; no SSE/HTTP transport change just to gain keepalive.

## 4. Initial budget envelope for R1 review

Numbers below are conservative proposed starting points, not measured product or connector limits; adjust with bounded native/installed tests. Preserve old limits and semantics for legacy calls.

| Resource | Candidate default / maximum |
| --- | --- |
| List | 20 / 50 entries per result, existing 4096 native entries and 512 KiB wire cap |
| Read, resource, batch combined | 16 / 64 KiB raw result content; total encoded/JSON overhead independently bounded |
| Search | 20 / 100 matches; 8 / 32 MiB total scanned, 4096 examined entries, depth 16; bounded line length and deadlines |
| Inline mutation | existing maximum 64 KiB raw per request including batch combined |
| Internal streaming buffer | 128–256 KiB, no whole-file buffering |
| Tools-only upload chunk | 32 / 64 KiB raw initially, aggregate multiple chunks only if real request budget proves feasible |
| Initial transfer test ceiling | 64 MiB individual transfer; candidate controller aggregate reservation 256 MiB, not physical quota guarantee |
| Concurrency | bounded per Box and controller; initial one conflicting native writer; do not hold Box claims while awaiting client input |

Do not advertise unlimited or 2 MiB client chunks based on loopback results. Measure actual external connector payload/reply limits and model-visible bytes separately; no model token metric invented. If search scan cannot finish, explicitly report incomplete coverage, not zero-match proof. UTF-8/CRLF handling preserves bytes, not newline normalization; distinguish byte offsets from lines. Exact SHA256 of bytes observed, not a mutable source snapshot. Size/time/mtime are evidence, not revision CAS.

## 5. Transfer lifecycle and durable identity

Separate long-lived staged-transfer record from short-lived admitted operations. Avoid holding the existing Box-level mutation claim through a client's multi-request upload. Begin reserves bounded controller-owned storage and records immutable intent with exact original receipt; each bounded chunk persists its original operation identity, range, length and digest before acknowledgment; commit revalidates source/destination and obtains existing affected Box claims. Copy uses one admitted native job over source and target claims, ordered deterministically. No new public job system; transfer records refer to existing operations. Same-key identical request returns original receipt; mismatched data/range/action or actor rejected. A verified staged prefix may resume only with original intent, exact stage provenance and matching bytes. No automatic replay of ambiguous destination writes.

Current source internal/state/store.go has SchemaVersion=7 and strict exact DDL, existing operations, operation_events, operation_resource_claims and exec_output_gc. It has no durable transfer record. Propose minimal transfer and range/progress metadata tables referencing original operations, with one private generated staging filename per transfer and bounded manifests. No file payload in SQLite/event logs. No new generic journal engine; do not reuse unrelated exec_output_gc semantics for transfers. Implement via owner-approved schema decision, likely next fresh-only schema version under current prerelease policy. Existing Schema 7 DB must be preserved/refused, not altered or reset. Migration/retirement compatibility for a supported persistent installation requires distinct explicit design/approval. Installer, service user directory, startup fences, Doctor/recovery and retirement/uninstall/cleanup must agree before transfer persistence ships.

Stage local files exclusively in controller-private quota-bounded location, never arbitrary host paths. Reserve bytes transactionally; avoid SQLite locks during I/O. Sync staged bytes before durable acknowledgment; validate actual range and hash on recovery. Guest SFTP does not advertise fsync in native witness: separate verified bytes from power-loss durability. Publish using only proven native contract; ordinary/Posix rename overwrite. Hardlink no-clobber was observed on one backend; no production enablement until alias, provenance, path checks, recovery, cleanup and supported backend proof. If unavailable, keep exclusive create with explicit partial/UNKNOWN semantics; no promised atomic publish. Abort/cancel/expiry cleanup touches only proven owned stages, never ambiguous final files. Revocation rechecked before native dispatch and returning private data; does not undo dispatched effects.

## 6. Explicit deferrals and nonblocking risks

- Strict conditional replace/delete and in-place patch stay UNSUPPORTED: no native CAS proof. Agent can use ordinary authorized Guest Exec for build/test/work, but the file Tool must not silently substitute it. A later owner decision on alternative explicit write semantics is required; not a weaker implementation of the old contract.
- Append deferred: exact native 7.7 / pkg/sftp 1.13.13 O_APPEND sequential fixture accepted write but replaced prefix; independent CLI bytes confirm, no length increase. Do not use seek-to-end as a claimed atomic append or retry-safe substitute.
- Restricted nonroot Project probe upload was blocked by platform check, not a native denial result. No alternate route to that refused operation. Existing product scope remains mandatory; no native promotion beyond default-Project lab. Actual restricted installed witness is a hard acceptance prerequisite before enabling new runtime access.
- Experimental Files/Filesystems extensions and subscriptions remain nonblocking optional paths; defer when client support or draft stability is unavailable.
- True crash/power loss, disk-full, retention/cancel, quota and native recovery require explicit integrated evidence. No storage pressure on unverified shared host quota.
- Full source/race/native CI and external agent matched efficacy still separate from R0 standalone feasibility.

## 7. Stage gates

P0 R0/R1: owner settles catalog, dependency and schema strategy; exact go.mod/go.sum changes only, full applicable source/CI and installed dependency regression; no silent host upgrade. P1: bounded read-only and exclusive-create core with lifecycle ownership, stopped state, strict path and aggregate limits. P2: native Resources and Tools fallback through one read core, real protocol/Actor negatives. P3: search/text/batch and proven create/metadata modes, preview only for unsupported mutation; do not block on CAS. P4: native transfer streaming and staged upload/download, original receipt/range recovery and budget; no false atomicity. P5: opt-in experimental adapter only with evidence. P6: combined actual Agent tasks, independent review, native and CI.

For each P use R0-R5 only where material; reuse consolidated existing CI, keep deterministic focused tests in existing packages, retain research sources as .go.txt outside product compilation. Probe completion does not require creating permanent workflow or expanding scope.

R0 decision candidate: positive feasibility for the selected minimum; restricted Project and actual external client are documented evidence gaps, not permission to weaken scope. Await owner scope decision before modifying the approved tool catalog or persisted schema; no merge or release.
