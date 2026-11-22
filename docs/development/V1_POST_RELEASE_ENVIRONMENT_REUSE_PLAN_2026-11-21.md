# BoxWeave v1 post-release development — unified Box environment reuse and recovery

Date: 2026-11-21 PT. Owner-selected future feature track, **after v1.0.0 formal publication and verified release closeout**. Not part of S5 or a v1.0.0 prerequisite. See [formal publication](V1_STABLE_PUBLICATION_PLAN_2026-11-21.md). This plan approves the feature family for later design; detailed contracts and native safety require E0 verification and review before implementation. No tool is registered or effect enabled by this document. Do not bump VERSION for planning. Per [versioning](../process/VERSIONING.md), a coherent accepted feature increment after 1.0.0 would normally be 1.1.0, not a version promise.

## 1. Product experience

Design five capabilities as one lifecycle, rather than five subsystems.

| Capability | Intent | Contract direction |
| --- | --- | --- |
| Snapshot | Save a development checkpoint | Exact native snapshot of supported guest state, bound to a Box incarnation |
| Restore | Revert an environment | Explicit destructive return to a checkpoint; not a rewind of controller history |
| Clone | Independently reuse an environment | New owned Box from stopped Box or exact checkpoint; fresh identity and isolated network |
| Backup export | Preserve or transport an environment | Verified portable artifact with versioned manifest and root filesystem coverage |
| Backup import | Recover an environment on this or another Host | New owned stopped Box verified from an approved artifact |

Target Agent workflow: create and configure Box → stop/checkpoint → list checkpoint → clone experiments → start/wait/test through existing tools → restore failed experiment with reviewed intent → export success → explicit operator transport → import and start a new isolated Box. Keep a small number of calls, no shell packaging chore, consistent defaults and original Operation navigation. Use existing read/Exec/Git/lifecycle/Activity capabilities for ordinary work. Snapshot is not independent disaster backup; clone is not a copy of controller history. Installer backup of binary/unit/ledger is separate.

## 2. Boundaries and supported initial scope

- Exact owned Box UUID/incarnation, Layer, restricted Project, official Incus/MCP Go SDKs, current authenticated management Actor and Token. No host admin socket, foreign native adoption, arbitrary Host path or general service/Project administration.
- Initial proposal: stopped, stateless Linux containers, owned root filesystem only, explicitly verified storage driver and native permissions. No live/stateful memory snapshot, VM, external/custom/shared volume or unknown device dependency. No auto-stop/start. Verify effective devices, capacity, quota and storage features; unsupported configurations refuse.
- Post-stable compatibility is mandatory. Preserve supported 1.0.0 API/CLI, customer SQLite data, credentials, original receipts and refused/UNKNOWN effects. If schema evolves, prove versioned retained-data migration, atomic failure/recovery, backup and safe older-binary refusal; no pre-release disposable reset model.
- No silent native permission widening, new privilege, new scheduler, global storage service, or another management plane. Use native copy/snapshot/backup capabilities only after source/API and real native assessment.

## 3. Unified resources and evidence

| Record | Proposed role |
| --- | --- |
| Source ref | Exact Layer/Project/Box incarnation plus current state or immutable checkpoint ID |
| Checkpoint | Opaque ID, owned source lineage, independently verified native identity, creation Operation, time, consistency/storage coverage and availability |
| Backup artifact | Opaque ID, format/version, source lineage, manifest/archive digests, byte count, compatibility and original export Operation |
| Destination reservation | Exact intended name, storage/NIC reservations, new Box incarnation/UUID binding and original copy/import Operation |
| Operation | Existing Actor, key, Tracking ID, original/native ID, intent, work_ref/agent_ref, source/destination/checkpoint/artifact refs, phase and verified poststate |

Names/aliases/paths alone are not identity proof. E0 must learn what Incus can independently prove about snapshot names/identity/reuse. Reject ambiguous stale IDs, not infer from timestamps. Apply bounded paging and auth recheck before private reads. No full artifact contents, guest secrets, sensitive paths or credential URLs in Activity/logs. Reuse actual Operation and Activity models rather than introduce separate jobs or tracing.

## 4. Identity, restore and independent reuse

**Restore:** only an exact stopped source Box with matching checkpoint lineage, no unresolved effects/claims or unsupported dependencies. Show checkpoint time, newer data loss, network/dependency and recovery limits, capacity. Review whether native restore can preserve exact UUID/incarnation with current controller identity; verify before and after. If not safe, choose separately approved new-Box restore design. No alias migration or automatic destructive compensation. Snapshot may contain stale native configuration. Require proof that restore cannot resurrect stale MAC/IP, peer/PRS grants, proxy devices, Forward listeners or unauthorized devices. Initially require relevant dependencies cleanly reconciled/removed through existing tools unless safe preservation is verified. Never rewind SQLite Operation history, output/Activity, tokens or external effects. UNKNOWN restore fences the source and checkpoint/dependencies.

**Clone/import:** reserve source and destination claims before native dispatch. New stopped destination has fresh verified UUID/incarnation and MAC/IPv4/IPv6 pins, separate reservations, safe effective Profile/devices and default deny. Do not copy Peer/PRS/ACL grants, Forward bindings, identity reservations, original Operation or authorization. Prove creation-time config safety, not merely a later pre-start check; do not expose a usable destination until exact native poststate and binding are committed. An uncertain result stays fenced, never adopt a box just found with the intended name.

Disk contents may duplicate machine-id, SSH host keys, static IP configuration, application credentials and nested controller/Incus state. E0 must define supported first-start re-identification and honest limits. Do not claim generic credential scanning is complete; retain intended application data, explicitly handle known sensitive identities and do not silently rewrite unrelated secrets. Initially refuse nested BoxWeave/Incus clone/import until safe inner identity/state handling is independently proved. Imported authority is always destination-local and independent; no inherited parent socket or nested credentials. Do not auto-start a clone/import.

## 5. Backup export/import and transport

One versioned, verified artifact format/manifest. Prefer a supported portable non-driver-optimized native backup if feasible; optimize only after compatibility tests. Record exact format, architecture, storage scope, native version constraints, source lineage, compression, snapshot inclusions/exclusions, complete file SHA256 and byte count. Cross-Host/driver/version portability must be actually tested; refuse unproven inputs. Cross-architecture is not promised.

Guest backups can contain application secrets. Choose an operator-configured private bounded artifact location and separately approved transfer; no arbitrary Host paths, remote URLs, automated uploads/downloads or generic file tools on MCP. If restricted controller cannot safely implement transport, initial root-local/operator CLI handoff is preferred. Agent references opaque approved artifact IDs. E0 must settle storage, exclusive staging, ownership, disk/inode and size limits, explicit retention/deletion, confidentiality and key custody if encryption is supported. Declare plaintext accurately; no key logging/management platform.

Export is complete only after durable verified write/readback, and partial artifacts stay unimportable. Import validates format, trusted manifest, length/hash, native compatibility, safe config and resource limits before any admitted native effect. Integrity hash alone is not trust. Reject malformed/untrusted/path-escaping/oversized archives and decompression/resource exhaustion risks. No arbitrary extraction into Host directories or native effects outside approved target; safe cleanup only of original verified temporary objects. Independent destination proof must include actual restored guest contents, not merely a successful Incus API response.

## 6. One durable effect model

Use existing per-Actor idempotency, Operation, worker, resource claims, Activity and supported operation_recover. Before native dispatch, persist immutable intent, source/dest refs, plan and key. Duplicate key returns original receipt, never creates another snapshot/clone/export/import. Establish canonical lock ordering across source/dest, snapshot/artifact, name, storage reservation and competing lifecycle/Exec/File/Peer/Forward/deletion. Refuse deleting snapshots referenced by admitted/UNKNOWN work; decide exact Box deletion cascade/refusal and independent backup retention. Preserve tombstones and historical lineage. No background cleanup until supported lifecycle proved.

Handle partial native create/copy/restore/stream, lost receipt/ACK and controller restart with exact original Operation/native identity and independent post-state. Native Operation absence is not proof of no effect. UNKNOWN stays original, retains claims, never blind replay, generic unlock or name-based adoption. Extend only safe domain-specific recovery proven by actual positive/negative tests; existing host-reviewed fallback for unsupported repair. Revocation/disconnect does not undo effects. Preserve existing current authorization, strict request schema, exact scope and independent evidence.

## 7. Minimal MCP design decision (E0)

Prefer a bounded read/list surface (extend box_get/Activity where appropriate) plus a compact typed environment mutation family. Evaluate one tagged mutation tool versus a few narrow mutation tools; do not automatically add one tool per verb. Candidate shape only, no approved names or registered tools:
- Read exact checkpoint/artifact lists/details, bounded pagination and coverage.
- Typed checkpoint create/delete, restore, clone, export and import with strict per-action validation, original idempotency key, reviewed destructive intent and source/destination refs.
- Reuse operation_get/list, box_activity_get, operation_recover, box_start/wait_ready and box_delete, and existing local CLI for operator-only handoff.
- Compare strict union schema/client support, clarity and minimum tool-count tradeoffs. No generic host command or hidden arbitrary capability. Existing owner/user risk confirmation model, explicit host-local approval where needed; conversation is not independent server-authenticated human approval. Mandatory backend ownership, identity, precondition and locking remain.

Deliver exact names, schemas, tool annotations, result types, SDK fit and acceptance plan for owner review before implementation.

## 8. Staged development and exit gates

One integrated design; incremental accepted slices, not five independent product designs or mandatory release trains.

| Stage | Work | Exit |
| --- | --- | --- |
| E0 | Actual stable source/SDK/DDL and native audit; snapshot/list/restore/copy/backup capabilities, identity and permission/driver matrix; choose shared schema, restore identity, minimal APIs, transport and compatibility | Native feasibility and refusal evidence; owner-approved contracts, pending issues explicit; no new effect enabled yet |
| E1 | Common refs, checkpoint/artifact/destination ledger, resource claims, idempotency/recovery/Activity and any migration | Stable-upgraded and fresh data integrity, fault/restart, concurrent admission, independent V |
| E2 | Stopped snapshot/create/list/delete and safe restore | Actual guest before/after, identity and unchanged network/ledger, conflict/ACK-loss recovery and refusal |
| E3 | Direct stopped Box and snapshot clone; new identity and safe first start | Independent source/dest content/native readback, IPv4/IPv6 pin/isolation and dependency negatives, concurrency and UNKNOWN |
| E4 | Verified export/import and operator transfer | Real independent destination, bytes/manifest and guest readback, large data/quota, failures/corruption/incompatibility, retained evidence |
| E5 | Full Agent workflow and retained stable upgrade/rollback | Exact candidate installed native + independent verifier, positive/negative full workflow, supported compatibility, dual-arch CI and reviewed immutable next feature release |

E2/E3 share foundation; direct clone from stopped source need not require creating a snapshot or in-place restore if proven. E4 feasibility can be investigated in E0. No publish simply because staged source builds.

Acceptance dimensions: actual root filesystem contents and source unchanged; exact UUID/stale/wrong lineage; ACK loss and original-key replay/controller restart; conflict/stale approval and storage admission; large data, low disk and partial cleanup; wrong/foreign Actor/Project/backup; malformed archive/transfer interruption; config/MAC/IP isolation and no copied grants/Proxies; duplicated guest identity and unsupported nested/external volumes; actual destination start/readiness; stable migration and rollback/refusal; no unauthorized effects, lost originals or leaks. Use real independent native/SQLite/MCP/guest witnesses. Mock/source or API success is not data recovery. Trusted nested functional testing does not prove hostile recursive isolation or physical portability.

## 9. Non-goals and start conditions

No scheduled backups, auto retention sweeps, replication/HA, cloud storage platform, whole-Host/Incus or controller-ledger backup, generic recovery platform, arbitrary native import/adoption, Web/REST plane, multi-tenant roles, guest daemon, live stateful restore or cross-layer automation. Old archive snapshot/clone docs are conceptual design references, not proven implementation. Multi-client configuration wizard, environment fingerprint history, generic diagnostics platform not reopened; report/selftest consolidation may be considered separately around existing doctor and Activity.

After verified stable release and recorded closeout, open an E0 design PR on topic branch from stable-compatible v1. Do not change the v1.0.0 tag/assets. Audit actual current code and execute isolated authorized native capability probes; produce a short owner decision table for restore identity and dependencies, guest re-identification, artifact transport/limits, migration/rollback and minimum MCP interface. Review before product implementation. No current runtime or release change from this plan.
