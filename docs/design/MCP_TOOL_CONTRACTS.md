# Minimal P0 MCP tool contracts

**Status:** design, not frozen wire schemas. See [20-tool catalog](MCP_TOOLS.md) and [permanent scope](../decisions/PRODUCT_BOUNDARY.md). Freeze exact JSON Schema after official SDK/native tests. Every mutation requires a nonempty, caller-generated, reusable idempotency key.

## `box_file_apply`

Input: exact owned `box_id`, safe guest path, `mode=create|replace|delete`. **No patch mode**.

**Actual official Incus v7.6.0 SDK test:** [final P0 contract evidence](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) proved SFTP `O_CREATE|O_EXCL` rejects existing files, and directly demonstrated the dangerous `read ALPHA → other writer BETA → ordinary overwrite GAMMA` and `read GAMMA → other writer DELTA → ordinary delete` sequences. Neither `CreateInstanceFile` nor `DeleteInstanceFile` accepts a conditional revision/ETag. **Retain the wire modes**, but `replace/delete` must fail `UNSUPPORTED` when native atomic same-file condition cannot be enforced; the current tested Incus API has no such guarantee. Never fake CAS by pre-reading a hash. Concurrent exclusive-create/symlink/hardlink/race tests are still required at implementation. Do not add a guest patch daemon or custom Incus REST wrapper.

- `create`: fail if path already exists; create only a verified regular file within size limit. Official SDK SFTP `GetInstanceFileSFTP` + `O_CREATE|O_EXCL` **passed one real existing-file refusal**, preserving original bytes. Plain `CreateInstanceFile` overwrite mode uses `O_CREATE|O_TRUNC` and is **forbidden** for exclusive create. Guest path, inode type, symlink/hardlink and simultaneous-create isolation must still pass negative tests before v1 feature acceptance.
- `replace`: accept complete new content with required observed original hash/revision. Check the exact native target and compare immediately before the write; never claim atomic compare-and-swap if guest concurrent writers can defeat it. Fail UNSUPPORTED when safe point-of-use guarantees are unavailable.
- `delete`: delete exactly one verified regular guest file only with required original revision/hash. No recursive delete, special files, symlink/hardlink/host-path access, or uncontrolled race.

Report native post-state, observed SHA/hash coverage and errors; do not invent success for a partial write. File list/read are bounded/paginated and must not claim a verified full-file SHA from an inconsistent partial read. Complex code patching is done via an authorized guest Git or shell invocation under `box_exec`, **not** an independent in-controller patch engine.

A staging-file rename by itself only guarantees whole-file visibility, **not** compare-and-swap against a concurrent guest writer. Do not weaken the existing conditional `replace/delete` contract based on an upstream example; return `UNSUPPORTED` when native point-of-use safety cannot be proven. An Incus file response identifying a symlink or directory is not ordinary regular-file bytes.

## Additional S2 correlation and recovery results, same 20 Tools

[Agent Operation Continuity](AGENT_OPERATION_CONTINUITY.md) adds optional bounded `work_ref`/`agent_ref` from SDK-native `_meta` or supported existing Tool arguments, never a new Agent API. Authenticated actor remains independent. Original labels are immutable on same idempotency key, and the same work label doesn't grant permissions. Existing `operation_list` gains authorization-first indexed actor/work/agent/state/time filters; `operation_get` and output retain original permissions/IDs. Typed `next_action` hints contain only safe reads/host-local review, never speculative mutations. [TC-001–TC-010](../verification/CONTINUITY_ACCEPTANCE.md) are additive to original S2 tests.

## `box_exec` and operation tools

`box_exec(box_id,argv,cwd?,guest_uid?,guest_gid?,env?,timeout_ms?,output_limit?,idempotency_key)` uses nonempty typed argv, guest-root UID 0 by default, bounded guest cwd/environment/time/output and no implicit host shell. A caller may explicitly invoke a shell **inside the owned Box**. It is a durable async mutation, with original exit/status/output obtained via `operation_get` and paginated `operation_output_get`. Nonzero observed exit is a completed failed command, not native UNKNOWN.

`operation_get(operation_id,wait_ms=0)` supports a capped 20000-ms read-only long poll; cancellation or timeout of the wait does not cancel the operation. Include a **small redacted diagnostic summary** (phase, failure classification, coverage/post-state evidence, original Tracking ID) in this result; no `diagnostic_get` tool or general host-log reader.

`operation_cancel` is a separately journaled and authorized cancellation request. Prove queued/undispatched cancellation transactionally; for active Exec attempt native stop only when supported and confirmed. Do not treat attempted cancellation as rollback or resolve UNKNOWN without evidence.

For `box_exec`, report separately: observed zero exit, observed nonzero exit, and unproven/missing exit metadata. Never default absent exit data to 0, retry arbitrary commands from PID-error text, or mask a native/transport failure behind an output-cap warning. Continued native execution/output/exit observation must survive an MCP caller disconnect; `operation_get` is a read of the durable original, not a new execution.

**Native Exec cancellation/output findings:** With `RecordOutput:true` and **no** stdout/stderr writers, the official SDK returned an Operation ID before a delayed command finished, recorded native stdout/stderr paths and preserved nonzero exit metadata. A **new SDK client connection** obtained original output via native `/logs/exec-output` using SDK `DoHTTP` (**HTTP 200**). For a detached Exec, `op.Cancel()` returned **not cancellable** and the command still ran; when a live SDK `Control` WebSocket was retained, `InstanceExecControl{Command:"signal",Signal:15}` terminated the guest command with exit **143**. Thus active Exec cancel is supported **only when the owned control connection exists and termination is independently observed**; otherwise report unsupported/not confirmed, not cancellation success. The caller's HTTP disconnect and process restart are separate from native command lifecycle. See [real probe](../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md).

## Delete lifecycle and remote failure visibility

`box_delete` confirms deletion of the **exact owned Box UUID** and all its **owned** dependent native TCP/UDP forwards and private peer grants. The backend must lock these claims against concurrent edits, journal each native cleanup phase, verify owned dependent removal and native Box absence before committing a terminal tombstone. It must never silently delete foreign network state, free uncertain UNKNOWN locks, or claim all-or-nothing atomicity across SQLite and Incus. On partial failure, keep the original operation record and expose verified remaining Box/forward state through existing read tools. Known incomplete operations finish as `FAILED` with `partial_effects=true`; ambiguous native completion remains `UNKNOWN`. For P0 the forward is an **instance-owned `proxy` device** and cannot survive as a separate native network-forward object after instance deletion; retire only verified stale local ownership claims. An orphan *network-forward object* deletion capability is conditional on a separately approved future backend; do not require it for P0.

`operation_get` returns a small, structured and redacted causal summary: current/failed step, safe native error category, observed Box/owned forward status, missing evidence and safe recovery hint, alongside version/build and original IDs. It is not arbitrary host diagnostics. Local operator doctor/journal remains an out-of-band escalation if the error cannot be safely localized from MCP. A duplicate `box_delete` with the same idempotency key **only queries the original receipt**, never re-dispatches another destructive call.

`host_status` identifies installed controller VERSION/build SHA and observed Incus version/API features. `box_get` and `port_forward_list` reflect observed native state and known stale/partial records honestly; a mere HTTP or SQLite success is not a verified live-service state.

## `box_peer_set` and native forwards

- `box_peer_set(source_box_id,target_box_id,allow,idempotency_key)` changes **only** the directional private access state of two verified owned Box UUID/incarnations. On Bridge, native ACL group selectors are unsupported, so test NIC-attached rules against **current IP bindings plus anti-spoof**; address change, reuse, absent IPv6 proof or partial rule change must fail closed. It has no CIDR/IP/port/protocol expression parameters. Native bridge/OVN/NIC safety must be proven; current effective grants are read in `box_get`. Deny foreign identities, address reuse, unsupported network enforcement and host-global ACL manipulation.
- `port_forward_create(box_id,guest_port,protocol,host_port,host_bind?,idempotency_key)` selects only TCP/UDP and verified Box targets. **LAN reachability requires explicit host LAN-IP binding and a probe from another LAN machine; host loopback defaults cannot provide it.** Host listener default is loopback when suitable; explicit public/wildcard bind requires administrator action and collision check. Use an **instance-owned Incus proxy device** as the P0 host-port backend. Network-forward objects are separate network-scoped resources, not a silent fallback; adding them requires a new, explicit topology and orphan-cleanup decision. `port_forward_list/delete` show/remove owned native claims only; never provide custom reverse proxy/URL forwarding or arbitrary target IP.
- Native private Box isolation remains default fail-closed. Native port forwarding creates an **explicitly exposed** service and must not silently broaden rights to other Boxes.

## Auth and source visibility

Two independently revocable role types: `admin` may manage BoxWeave-owned resources and Exec inside owned guests; `viewer` may inspect owned metadata and guest files/output, but must never mutate or Exec. No per-Box grants, tool-class ACL matrix, tenant roles or inherited parent credentials. Viewer read access may expose confidential guest data; do not describe it as harmless.

Pre-dispatch Token revocation blocks new/queued-undispatched mutations, not proved-dispatched work. On unsafe/incapable native operation, fail closed with Tracking ID and classified error; logs remain local. No Caddy, HTTP-route, generic ACL or dedicated diagnostic API.

## Tagged-SDK execution nuances

- An authenticated typed MCP handler receives the actor in **`req.Extra.TokenInfo`**, while `auth.TokenInfoFromContext` is for HTTP handlers; authorize each invocation and reject missing/revoked metadata.
- For an immediate async `box_exec` receipt, avoid `RecordOutput=true` combined with non-nil output writers until proved safe: Incus's official SDK can synchronously wait and delete native recorded logs in that path. Prefer SDK stream + bounded worker retention.
- Default read-only `operation_get(wait_ms)` to bounded native `GetOperation` polling until `GetOperationWait`'s shared HTTP transport timeout side effect is tested.
- These choices follow [source-level second-pass review](../research/SDK_GAP_CLOSURE_2026-11-07.md), not a private Incus REST implementation.

## Required native black-box checks

Verify whole Agent create/read/create-or-replace/file-delete/async test/recover/fix/retest cycle; file race/symlink/hardlink denial; viewer forbidden Exec and write; Box peer A->B allow/revoke without changing B->A initiation; host loopback/public TCP+UDP mapping; foreign resource denial; repeated idempotency/UNKNOWN; nested L1/L2 matching behavior. No P0 PASS without real Incus evidence.
