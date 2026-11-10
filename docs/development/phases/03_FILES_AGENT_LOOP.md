# S3 — Guest files and the complete Agent edit/build/test loop

**State: NOT_STARTED (2026-11-08).** Run in one end-to-end pass, including research specific to missing contracts, implementation, real guest evidence, negative security tests, fix/retest and durable handoff. Do not stop after adding only the three file handlers.

**Minimal entry:** [Current S2/S2.5 BRIDGEHEAD/S1 artifact and handoff](../EXECUTION_LOG.md#current-checkpoint), [phase protocol](../PHASE_EXECUTION_PROTOCOL.md), [Agent workflow requirement](../../product/DEVELOPMENT_LOOP.md) and [guest management](../../design/GUEST_MANAGEMENT.md). Reuse installed v1 typed MCP, journal, worker and exact Box UUID from S1/S2; no new guest management process.

**Read on demand:** [MCP_TOOL_CONTRACTS — file modes](../../design/MCP_TOOL_CONTRACTS.md#box_file_apply), [official SDK P0 file/native evidence](../../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md), [tool/native SDK mapping](../../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md) for each native method, [security](../../design/SECURITY.md) for viewer/host paths, and relevant [ACCEPTANCE](../../verification/ACCEPTANCE.md) **A-009–015, A-020, A-039–040, A-063–065, A-075–077**.

**Inserted predecessor:** [S2.5 BRIDGEHEAD](025_BRIDGEHEAD.md) tests whether S1+S2 can safely operate as a real-host early development base; it does **not** replace this S3 file-tool implementation or guest-file/Agent-loop acceptance. Use the approved pinned v1 candidate after **BRIDGEHEAD HOST_GO**, or continue in independent disposable labs if real-host pilot is NO_GO/BLOCKED while preserving the unmet gate. Always rerun affected S1/S2/BRIDGEHEAD native regressions if this phase changes shared Exec/auth/SQLite or installation semantics.

## Three new public tools

- **box_file_list:** enumerate only guest-owned, safe/bounded file and directory metadata with pagination, accurate file type, reject host/foreign Box/path escape.
- **box_file_read:** bounded byte offsets/pages, safe regular files only; report whether hash/byte coverage is full or partial, and never invent a consistent full-file digest from concurrent partial reads.
- **box_file_apply:** exactly **create | replace | delete** on one regular guest file, with original idempotency and operation receipt; **no patch mode**.

## Native safety contract decided by real evidence

[2026-11-08 real SDK P0](../../verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) proved native SFTP **O_CREATE|O_EXCL** can refuse an existing file without changing bytes. Ordinary SDK overwrite and delete have **no atomic hash/ETag precondition**: an intervening writer's updated contents were actually overwritten/deleted. **Do not implement fake check-hash-then-write, staging-rename-as-CAS, unsafe automatic retry or custom upload/patch daemon.**

Implement exclusive create only after verifying safe path traversal, symlink/hardlink/special target and concurrent-create denial with actual guest poststate. Retain the public replace/delete *modes* as part of the product contract but return a **typed UNSUPPORTED** when an atomic same-file condition cannot be proven. Such refusal **is an expected safety outcome**, but does **not** count as full positive replace/delete feature acceptance. A different product guarantee requires explicit owner decision; record it and continue other work.

## Work sequence inside this same phase

1. Build file target resolution that checks exact owned Box UUID, path and guest-vs-host boundary, and remains bounded under hostile input/large files.
2. Implement native list/read/exclusive create with official Incus SDK/SFTP; classify unsupported conditions before side effects. Preserve S1 actor/Tracking/operation/UNKNOWN behavior.
3. Prove negative path cases: guest absolute/relative traversal, special files, symlink to elsewhere, hardlink aliasing, reused Box identity, concurrent create/modify, stale observed hash, malformed content size and multi-page inconsistency. Test that **forbidden files are unchanged**, not merely that the API returns an error.
4. Drive the whole sustainable Agent workflow using actual installed v1 MCP tools and ordinary **guest Exec**: create Box → inspect/read/create guest file → clone or initialize a small repo **inside guest via native Exec/Git** → deliberately failing test → fetch original bounded failure output by operation ID → modify source **via guest shell/Git** (no new controller patch feature) → retest PASS → independent native readback.
5. Run partial/error/restart/caller-disconnect scenarios: admitted mutation receipt remains durable, no accidental host-shell redirection, no false successful partial writes, test source SHA and guest Box identity recorded; the Guest needs **no BoxWeave Agent, SSH daemon or embedded Git helper API**.
6. Fix code and docs on each discrepancy; rerun original failing test, regression and all earlier S1/S2 interfaces affected by file/Exec changes.

## New TC-015–TC-016: true Agent handoff with guest Git evidence

Use the [lightweight client-owned Task Checkpoint](../TASK_CHECKPOINT_CONTRACT.md), not a BoxWeave Tool or DB. [TC-015–TC-016](../../verification/CONTINUITY_ACCEPTANCE.md) require an interrupted Agent, actual guest Git SHA/dirty source, original failing test output via original Operation, a second Agent's safe fix/retest and independent postread. Two writers to checkpoint Git/external storage must detect revision conflicts; don't expose guest secrets. Full S3 file Tool and native CAS/security tests are unchanged.

## Phase acceptance and explicit carry-over

Require at least one **real** end-to-end agent development loop with a failing and passing test plus native guest evidence, both actor roles and denied host/foreign/special-file access, safe memory/output limits, consistent original receipt after reconnect and owned-resource cleanup.

A-012/013/065 **cannot be marked full PASS for conditional replace/delete** on the tested native API; document the exact UNSUPPORTED contract, caller-visible error and remaining acceptance gap. This is not a license to add a resident guest Agent or alter the product scope. Keep focused instructions and genuine test outputs, not just a test count.

**Handoff to S4:** current working candidate digest, Box UUID/Project, exact Tool schemas, lifecycle/Exec/file receipt format, real Agent-loop evidence, unsupported file modes, pending UNKNOWN or owner decision, and which acceptance IDs still require integration. Append to [execution log](../EXECUTION_LOG.md) and link focused proofs; S4 reuses these flows to test HTTP service/port-forward/nesting.
