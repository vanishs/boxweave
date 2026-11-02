# Box Operation Interface v1 — External-First, Agentless Design

**Global plan:** [BoxWeave feature roadmap](FEATURE_ROADMAP_MASTER.md) — R-01 Box operation interfaces. This link tracks the overall selection/status; this document retains detailed scope and test obligations.

**Status:** DESIGN ONLY — implementation, live Box changes, release and acceptance are **not** authorized by this document.
**Baseline:** BoxWeave main at `16470c715c16c713af2ba31f4f6145d647967f73`, source version 0.26.9, reviewed 2026-11-01.
**Product decision:** Preserve existing BoxWeave architecture and lifecycle; a Box remains an independent, clean execution environment. External, non-invasive weaving is preferred. Do not broaden this design into management of foreign/unmanaged Boxes.
**Related:** [MCP capability master](MCP_CAPABILITY_INTEGRATION_MASTER.md), [agent-box matrix](MCP_AGENT_BOX_CAPABILITY_MATRIX.md), [M0–M5 source/acceptance](../process/MCP_M0_M5_SOURCE_PROGRESS.md), [Agent guide](../MCP_AGENT_GUIDE.md), [testing practice](../process/AGENT_TEST_WORKFLOW.md), [security boundaries](MCP_INTEGRATION_SECURITY.md).

## 1. Goal and non-goals

Give authorized coding Agents a small, dependable, **Box-scoped** operation surface for the ordinary loop:

1. Select and validate an existing managed Box.
2. Inspect project files and search code.
3. Make a bounded, conflict-aware file change.
4. Run a test/build command in the Box.
5. Read its actual exit status and relevant output.

**The interface is a façade over existing BoxWeave / Containarium / SSH capabilities, not a new Box runtime.** Workspaces are call-time authorized project roots associated with an exact Box identity; they are not another kind of Box and do not affect Box ownership, lifecycle, network membership or nesting.

Non-goals for v1:

- No mandatory in-Box Agent, DevSpace, MCP server, systemd service, tmux, Python, package installation or boot-time process.
- No rewrite of Containarium, Provider, Incus, Box creation or Box SSH trust.
- No host shell, host file access, arbitrary caller-selected SSH host/user/key, implicit SCP, or privilege escalation.
- No multi-Agent orchestration, Git worktree manager, dynamic workflow engine, full IDE or file synchronization daemon.
- No blanket per-Agent/per-Box RBAC claim; existing trusted-operator and optional host-wide tool-policy boundaries remain explicit.
- No promise of durable long-running command recovery in this slice; that is the separate MCP task-reliability design.
- No assumption that a configured `agent_box` implementation is installed or available on every Box.

## 2. Verified existing surface and precise gaps

| Source/component | Already present | Limitation to preserve or address |
| --- | --- | --- |
| `bwmini/mcp_gate.py` + upstream Platform MCP | Direct Agent `connect(exec)`, exact-target trust, list/get/start/stop/delete, optional fixed tool allowlist; SSH and HTTP routes | `connect(exec)` is a generic command string, not a typed file/project interface; the gate must not imply paths or host/user overrides are safe. |
| Upstream MCP as observed on 2026-11-01 | `connect` with optional `session` (requires tmux in Box), `code_runs`, `code_logs` (read-only run observation) | Existing run observation is not evidence of a universally supported durable launch/reattach API. Installed backend/version/scope must be checked at runtime. |
| `bwmini/agent_box.py` | Exact-Box, configured trusted launcher; read/list/write/move/delete, shell/process/log/Compose interfaces | App-side and separately configured. No default direct-MCP file tools; file paths are scoped, shell commands are not confined by file-root checks. |
| `bwmini/box_actions.py` | One-use plan/confirm, durable receipts, unknown-on-restart, no automatic replay | App/Web write path, not a direct-Agent general-purpose transaction manager; receipts intentionally omit raw command/file contents. |
| D3/D4 diagnostics | Classified gateway evidence and explicitly granted bounded guest diagnostics | Not a substitute for arbitrary project file reads or output capture. |
| Historical live M0–M5 tests | Real exact-Box `agent_box` read and path denials were verified against a specifically configured nested Box | Do not extrapolate that all operations or the current 0.26.9 installed runtime were live-accepted. |

**Design rule:** Reuse existing behavior first; add a typed public façade only where it closes an actual Agent usability gap. Do not expose the Web `agent_box` API as an unguarded generic MCP tool.

## 3. Architecture: existing channels, one Box operation contract

```text
AI Agent (SSH/stdio MCP OR authorized HTTP MCP)
    -> existing BoxWeave direct gateway
       -> new small typed Box-operation dispatch + policy/target checks
          -> existing authenticated Provider connect(exec) / other verified
             Platform Box-scoped capability (when available)
          -> optional separately authorized agent-box adapter only where configured
             -> exact managed Box, ordinary non-root workload user

Web/application path -> existing Service + BoxActions plan/receipt -> existing configured BoxAgent
```

Keep direct gateway and application Service as **separate security principals**. Shared response shapes and validators may be reused without silently merging permissions. Both transports must advertise the same eligible direct-MCP surface when authenticated and the same tool policy applies; Web continues to use its reviewed action plans.

### 3.1 Box identity and project root

- Resolve a Box logical name through existing Provider registry, not caller-supplied host information.
- Record the immutable target digest (backend/project/logical/instance incarnation) for each operation; revalidate immediately before remote dispatch, especially writes.
- The project `root` must be an absolute path permitted by an explicit operator-owned project-root policy for this exact managed Box. Do not treat an arbitrary model-selected directory as authorized.
- Keep project paths out of the Box image. An operation can use an existing project directory; it must not silently create it.
- Verify path containment **at the remote point of use**, not only by string prefix; prevent `..`, symlinks/hardlink escapes, TOCTOU overwrite, device/proc pseudo-files and unsafe target types. If this cannot be enforced with the existing guest facilities, return `unsupported` rather than weaken the boundary.
- Allow a read-only observation of capabilities and configured project-root authorization before requesting any mutation.

### 3.2 Execution transport selection

Reuse `connect(exec)` where possible. For typed operations, generate fixed, reviewed remote commands internally; never interpolate untrusted paths/content as raw shell fragments. An optional configured `agent_box` can be selected **only after** an explicit capability/authorization check, never as an assumed default. Reject unsupported prerequisites honestly.

No permanent helper is installed in the Box. An optional **one-shot** guest command may be used only if the prerequisite interpreter/utilities already exist and it leaves no resident process or package; report whether any temporary file was created and clean only task-owned temporary files after confirmed completion. A remote shell alone is not a filesystem sandbox.

## 4. Proposed Agent-facing operations (names and schemas are candidates)

| Candidate operation | Priority | Input contract | Result contract | Backing strategy |
| --- | --- | --- | --- | --- |
| `boxweave_box_capabilities` | P0 | `box`, optional `root` | Supported operations, limits, prerequisites, authorization and evidence class | Observed exact Box + installed Provider capabilities and approved root policy |
| `boxweave_box_list_directory` | P0 | `box`, `root`, relative `path`, `limit`, `cursor` | Bounded names/types and next cursor | Existing approved adapter or safe non-invasive command |
| `boxweave_box_read_file` | P0 | `box`, `root`, relative `path`, `offset`, `max_bytes` | Bytes/text encoding, hash, actual offset, EOF/truncation | Existing approved adapter or safe bounded read |
| `boxweave_box_search` | P1 | `box`, `root`, bounded filename/glob/text match, result limit | File/line/offset snippets, truncation | Fixed read-only search; probe utilities before dispatch |
| `boxweave_box_exec` | P0 | `box`, approved `root`, command, timeout, output budget | Observed exit code, stdout/stderr, duration, typed execution state | Existing `connect(exec)` and exact-target trust |
| `boxweave_box_apply_patch` | P1 | `box`, `root`, relative `path`, expected-before hash, bounded patch | New hash, changed flag, conflict/denial, no silent overwrite | Explicitly authorized write, remote atomic validation + replacement |

**Integration INT-02:** The candidate `boxweave_box_exec` is not a separately dispatched shell engine. It MUST call the same reviewed `connect(exec)` adapter as the existing gateway and R-02 tracked tasks. Prefer omitting the direct alias when `boxweave_task_submit(operation=exec)` already solves recoverable execution; if retained, distinguish *untracked* ordinary calls from explicitly tracked ones and keep one permission/target-validation path. See [roadmap integration review](FEATURE_ROADMAP_MASTER.md#5-integration-review-conflicts-reuse-and-corrected-dependencies-2026-11-01).

`boxweave_box_exec` is **not** a promise of confinement to `root`: setting a working directory does not limit where shell commands may access. Treat it as a powerful existing trusted-operator capability, subject to client/operator approval and Box OS/container permissions. Do not advertise `readOnlyHint` for commands merely because the caller says they are tests. The existing `connect(exec)` remains available; do not break or replace it in v1.

Do not create `upload_file`, `download_file`, `process_start` or `process_stop` as duplicate direct tools without first proving the existing transport/capability gap. Larger file transfer and durable process control are follow-on designs, not part of this minimal implementation.

### 4.1 File write safety

- Default to **create-only** or compare-and-swap update where meaningful; never overwrite an existing file based on a stale read.
- Reject absent/mismatched expected hashes for replacement and clearly distinguish `CONFLICT` from transport failure.
- Perform file-state check and final replacement in a single suitably protected remote operation; a host-side hash check followed by a separate remote write is unsafe.
- Require safe file type, path resolution and filesystem checks. Use same-directory temporary file + atomic rename only when underlying filesystem/OS semantics are actually verified. Do not claim universal crash durability or transactional rollback.
- Bounded content/patch size; do not embed unlimited content in MCP arguments, logs or diagnostics. Capability metadata must tell the Agent the supported maximum.
- A timeout or lost reply during a mutation is `UNKNOWN`; do not transparently run it again. Re-read the exact target and content/hash to reconcile, then require a fresh intentional action if necessary.

### 4.2 Result and error vocabulary

A returned `exit_code != 0` is a **completed command** with a failing exit status, not an MCP transport failure. Every operation should distinguish:

- `completed` (including command exit code and bounded stdout/stderr);
- `denied` (root/tool/operator policy or unsafe path);
- `unsupported` (missing known guest prerequisite or installed upstream capability);
- `target_changed`, `box_stopped`, `ssh_trust_failed`;
- `conflict` (write precondition);
- `failed_before_dispatch` (definite no-side-effect error);
- `unknown` (dispatch may have occurred, outcome not confirmed).

Keep sensitive output transient and explicitly bounded. Diagnostic journals and persisted receipts should record classification, correlation IDs, hashes/byte counts where safe, not raw credentials or file contents. Any exact failure reason derived from a guest may be sensitive and must not be broadcast to unrelated actors.

## 5. Implementation batches

### BOI-0 — Contract/compatibility and existing-code audit

- Compare current installed `tools/list`, `connect` schema and results on an authorized Box, and separately inspect configured `agent_box` availability; identify supported and unsupported variants.
- Specify shared shapes, root policy and refusal semantics. Map exact direct gateway, app Service, Provider and Web trust boundaries.
- Add tests for schema filtering, version mismatch and complete non-dispatch on denial.
- **Exit:** no new mutation behavior, precise compatible contract and explicit unavailable states.

### BOI-1 — Read-only file + command result improvement

- Introduce capability read, list/read/search using safe bounded operations, optional cursor/offset, with strict target and path checks.
- Add the typed command result façade only where useful; preserve `connect(exec)` as the compatibility path.
- Use small, transparent tool descriptions so Agents know when to call these tools instead of crafting shell commands.
- **Exit:** in a real permitted Box, read an existing file byte-for-byte, search a bounded project, run a test and distinguish command exit 0 / nonzero / transport failure; no install to Box.

### BOI-2 — Conflict-aware editing

**Integration INT-01/INT-03:** Before exposing a mutating patch endpoint, pass through the same canonical Box/resource operation-claim boundary used by R-04 and the original dispatch/UNKNOWN semantics from R-02 when tracking is advertised. Do not invent an independent BOI patch receipt/lock table; without an implemented shared guard, do not claim cross-SSH/HTTP/Web collision safety. Read-only BOI-1 remains independently shippable.


- Add create-only and bounded patch with expected-before verification as a reviewed write capability.
- Reuse existing M4 confirmation/receipt machinery for the application path; do not fabricate shared per-client authorization between Web and direct gateway.
- Establish one-time remote dispatch semantics and stop on uncertain replies; no automatic retry of writes.
- **Exit:** exact-target file edit observed, concurrent/stale edit rejected, symlink/out-of-root denied, and unknown mutation not automatically replayed.

### BOI-3 — Packaging, mixed-version and real-client acceptance

- Verify SSH and HTTP MCP discover the correct eligible tool surface and optional global allowlist can remove new tools.
- Test with an older Provider that lacks proposed primitives; preserve existing `connect` and surface `unsupported` honestly.
- Validate supported client/tool schema and package-to-runtime parity. Do not assert broad client compatibility from generator or fixture tests.
- **Exit:** actual source-bound release, installed system, authenticated real client and live Box evidence separately recorded before any `DONE` designation.

## 6. Required verification matrix

| ID | Test | Required evidence |
| --- | --- | --- |
| BOI-T01 | Exact Box command `printf`, exit 0, stdout/stderr | Real authenticated output + exit code |
| BOI-T02 | Exit 7, timeout, lost SSH response | Distinct exited / unknown; no invented success |
| BOI-T03 | File list/read with UTF-8, quotes, spaces, binary and large file | Byte identity, bounds, no command injection |
| BOI-T04 | Missing interpreter/search program/agent-box | `unsupported`; no auto-install or hidden fallback |
| BOI-T05 | Path traversal, symlink escape, pseudo-file, root mismatch | Denied before unsafe access |
| BOI-T06 | Wrong Box incarnation, changed SSH pin, stopped Box | No dispatch to wrong target |
| BOI-T07 | Create-only, expected-hash edit, simultaneous edit | No overwrite of newer data; deterministic conflict |
| BOI-T08 | Lost reply after mutation and retry request | Preserve `unknown`, observe before another action |
| BOI-T09 | Gateway allowlist blocks new operation, Web plan cannot bypass scope | Negative permission evidence |
| BOI-T10 | Old Provider, installed app mismatch, missing feature | Graceful typed capability refusal |
| BOI-T11 | SSH/HTTP direct-MCP parity and client-visible JSON schemas | Real authenticated consumer evidence |
| BOI-T12 | Clean Box inspection before/after | No new permanent agent, package, service, SSH bypass or foreign files |

Use fixture/unit tests first, then authorized real disposable Box validation under [Agent test workflow](../process/AGENT_TEST_WORKFLOW.md). For every test record source SHA, exact installed/runtime/Provider versions, identity, commands, exit code, class S/T/L/U, and retained/cleaned resources. If a test is impossible, mark `BLOCKED` and continue independent cases without claiming acceptance.

## 7. Relationship to MCP task reliability and resource claims

**Current owner decision:** [R-02](MCP_TASK_RELIABILITY_R1.md) now specifies reconnect-recoverable tracked requests. Its durable executor is an opt-in wrapper over the existing `connect(exec)` path, not a second guest runtime. [R-04](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) owns optional Box leases and cross-entry mutation claims; protected BOI edits must reuse that thin claim interface once activated. The initial BOI read-only batch does not wait for R-02/R-04. Any older "until approved" wording below describes the original design baseline, not today's planning status.


The operations above MUST emit enough structured evidence (target identity, operation/correlation reference, stage and observed outcome) to support a later MCP reliability layer. **Do not implement automatic retriers, durable command adoption, an Agent scheduling loop, or generalized lifecycle state in this design.**

A separate proposal should settle the difference between:

1. Connection/session resilience (SSH keepalive, HTTP reconnect, OAuth and client state);
2. Side-effect certainty (executed vs never dispatched vs outcome unknown);
3. Long-running execution observability (stable run identity, output cursor, exit and cancellation).

Until that design is approved, preserve current behavior and never promise a durable execution handle that the installed Provider cannot actually produce.

## 8. Delivery/acceptance and rollback policy

- This document creates **no new REQ/AC approval** and authorizes **no code implementation, deployment, write to any Box or new network exposure**.
- Before implementation, reconcile this design with current normative [scope](../product/SCOPE.md), [requirements](../product/REQUIREMENTS.md), [acceptance](../product/ACCEPTANCE.md) and current [status](../process/PHASE_STATUS.md). Update them only when the owner approves a behavior-changing slice.
- Implementation must be small and separately testable; do not bump product version for a design-only document.
- On regression, disable only the newly added capability; preserve `connect(exec)`, existing app action receipts and installed Box contents.
- Historical M0–M5 and older verification material are evidence, **not** a substitute for a current exact-version live test.
