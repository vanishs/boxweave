# P0: sustainable Agent development and acceptance loop

**Primary product objective:** An authorized coding Agent can continuously develop, build, diagnose, test and independently verify software **inside a BoxWeave-owned, agentless Linux Box**, including installing and testing a new BoxWeave controller in an appropriately nested Box.

No separate in-guest management Agent, SSH daemon, IDE, project/tenant control plane, worktree manager or multi-layer orchestrator is required.

## Minimum complete loop

1. **Provision:** `box_create` defaults to a reviewed Ubuntu LTS image, configurable CPU/RAM/disk, **starts the Box by default**, and has optional `start=false` and `nesting=true` parameters. Creation returns an operation + tracking ID; wait for both native RUNNING and **guest Exec readiness** before promising a usable development environment.
2. **Inspect:** `box_get` and `box_file_list`/`box_file_read` expose scoped, bounded exact-Box information. Use guest file paths only; no host paths, foreign Incus resources, implicit SSH, or installation of management client. The local runtime's exact read/modify capabilities are reported honestly.
3. **Edit:** `box_file_write` creates or updates an owned guest file only with documented safe replace policy. `box_file_patch` requires an expected original hash/revision and refuses changed/symlink/unsafe targets. Native Incus guest file facilities are the preferred transport. Keep edits bounded/paginated and return verification evidence.
4. **Run:** `box_exec` runs as guest root by default (optional explicit guest identity), accepts a bounded command and work directory belonging to the exact owned guest, and returns a **durable asynchronous operation**. It is not host shell Exec; no hidden guest daemon or SSH prerequisite.
5. **Observe:** `operation_get` and `operation_output_get` retrieve observed exit status and bounded stdout/stderr pages after MCP reconnect, using the original `operation_id`/`tracking_id`. A nonzero guest exit is a completed command with `command_success=false`; no exit observed means UNKNOWN or verified termination reason, not guessed success.
6. **Diagnose:** `diagnostic_get` retrieves redacted, read-only source/phase/error/coverage evidence by Tracking ID. `host_status` and `box_get` distinguish controller, native daemon, Box state, guest readiness and actual Exec success; lacking permission or a probe means `not_verified`, not PASS.
7. **Verify:** Agent runs `go test`, `go vet`, builds and bounded real validation **inside the guest**. BoxWeave's host/controller tracks operations and evidence but does not manage the contents of the guest development repository. For a new BoxWeave candidate, install it inside an appropriate nesting-enabled Box and exercise its own independent MCP/Incus/Caddy stack, without touching the outer controller.
8. **Iterate and recover:** read and patch the original file, rerun exact tests, inspect original receipts after a disconnect/crash, and repeat. An UNKNOWN mutation is never blindly replayed; a new iteration requires verified completion/reconciliation or explicit reviewed recovery.
9. **Independent acceptance:** a separate authorized Agent/test Box can independently reproduce the original failure against the exact installed candidate bytes. No unsupported or unexecuted scenario becomes PASS.

## P0 versus deferred

**P0:** Agentless Box lifecycle; start/stop/update/delete, optional nesting; guest-root Exec with async receipts/output; bounded guest file list/read/create/write/optimistic patch; tracking/context; scoped read-only diagnostics; token scopes by operation class and exact Box UUID; default no disk shrink; complete developer-loop real/nested acceptance; dependency-aware install/upgrade.

**P1 or later:** background process manager with persistent process identity, attach/terminate/log supervision; automatic idle sleep and HTTP wake; guest file search convenience; native snapshots; OAuth 2.1 browser flow. None are prerequisites for ordinary code development because the P0 file and Exec path is complete.

## Tool design constraints

- Files are guest files: specify an owned Box UUID plus guest-relative or explicitly safe guest-absolute path. Never accept host filesystem paths or transfer Incus Admin API handles. Reject path traversal, unsafe symlink/hardlink/device targets and mismatched revisions, using point-of-use checks and native capability validation.
- File read/list supports pagination and size budgets; writes are durable mutations with idempotency and read-back. Do not claim whole-file atomic replacement for a backend that cannot prove it; `UNSUPPORTED` is preferable to a silent unsafe write.
- Exec output is separately bounded and subject to private retention. Do not store full commands, file content, environment variables or raw stderr in general diagnostics. A command that is still running after a lost MCP connection is **not** evidence that it will survive a host/controller crash.
- Token permission categories include `box.read`, `box.lifecycle`, `box.exec`, `box.file.read`, `box.file.write`, `network.read`, `network.write`, `operation.read`, and `diagnostic.read` (subject to frozen implementation schema). Some permissions may additionally be scoped to exact native Box UUIDs. A token may not widen the resource ownership boundary.
- A default clean Ubuntu Box need not contain Go, Git, or a build toolchain. Installing language/dev tools **inside the authorized guest** is part of the development workflow, not proof that BoxWeave installed a management client; avoid silently changing every default Box image.

## Minimum real acceptance

Using only the installed HTTP MCP tools (and a separate local maintenance CLI for an inner installation):

Create -> readiness verified -> list/read a file -> write/patch with checksum -> start a Go test/build via async Exec -> disconnect client -> query original output/exit -> observe a deliberate failing test -> patch -> rerun test successfully -> inspect redacted diagnostics -> delete an exact test-owned Box.

Repeat inside L1 and L2 when upstream permits. Prove foreign Box/host paths and stale file revision/identity are refused with no side effects. Verify restart/UNKNOWN and capped output. Record source SHA, installed binary hashes, Box UUID, dependencies, tracking/operation IDs and exact test outcomes.

Related: [tool catalog](../design/MCP_TOOLS.md), [tracking](../design/TRACKING_CONTEXT.md), [operations](../design/OPERATIONS.md), [agentless guest](../design/GUEST_MANAGEMENT.md).
