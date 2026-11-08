# Agentless Box management

**Decision:** A normal Box is a Linux container managed by the *host's* Incus daemon. The Box requires **no installed BoxWeave guest client, resident Agent, SSH daemon, or BoxWeave-specific in-guest service** for P0 lifecycle and execution.

## Management path

```text
Authenticated MCP caller
  -> BoxWeave (Go MCP SDK, authorization and ownership)
    -> official Incus Go SDK via restricted local management boundary
      -> host Incus daemon
        -> owned Linux container (no guest client required)
```

Create, list, start, stop, delete, resource config, P0 guest file APIs and native Exec are issued by Incus on the containing host. `box_exec` uses typed argv, optional guest cwd/env/timeout/output limits; short `operation_get(wait_ms)` waits never cancel native work. Creation starts and checks guest readiness by default; explicit `start=false` is supported. Guest command stdout, stderr, exit status, timeout and transport interruption are captured by native Exec handling and the durable BoxWeave operation journal. A valid guest root (UID 0) is the owner's selected default; explicit supported UID/GID overrides are allowed. Guest root is not a host root command surface. Do not expose host paths, Incus sockets, devices or privileged instance configuration through guest Exec.

Native Incus guest file APIs are part of **P0**, enabling bounded listing/reading with verified hash and unified `box_file_apply(create|replace|delete)` without a guest Agent. Complex patches use guest Git/Exec, not a BoxWeave patch engine. Conditional updates need a defensible native compare-and-swap/point-of-use guarantee; no fake atomicity. If a specific native API cannot prove safe symlink/race/atomicity handling, report UNSUPPORTED rather than using host filesystem fallbacks. All writes and Exec are durable async with stable Tracking ID and separate queryable output/diagnostic evidence. The guest OS still needs ordinary runtime facilities needed by the specific command (e.g. an existing executable), but no BoxWeave-specific daemon.

## Exact guest Exec and regular-file result guarantees

**Exec:** A nonzero observed guest exit is completed execution with `command_success=false`. A trustworthy observed zero is success. Missing or malformed exit metadata, an uncertain native dispatch/wait or lost websocket cannot be folded into zero/success; keep bounded output and original Operation/Tracking identity for reconciliation. A dead MCP client must neither cause an implicit replay nor be the sole owner of the command's output/exit bookkeeping. Test long-running commands with both silent and continuous stdout/stderr while the client disconnects. Cap bytes as they arrive; distinguish truncation from a genuine native/transport failure. Never auto-retry an arbitrary user command merely because an error string resembles a transient PID failure.

**Files:** Stage-and-rename can prevent *partial bytes being visible* but is **not** atomic compare-and-swap relative to a concurrent guest writer or a changed symlink. Treat guest file-API response types (regular/directory/symlink) and actual path targets explicitly; lexical cleaning alone is not a trust boundary. `create` must not replace an existing path. `replace/delete` require an original revision/hash *and* point-of-use safety; if the official native API cannot supply the required race guarantees, return `UNSUPPORTED` without claiming an atomic conditional edit. Read size limits, stream closure, partial hash evidence and post-readback must remain truthful; do not introduce a file helper daemon.

Historical concrete counterexamples and source risks are in [Containarium main/history review](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Contrast with Containarium

Containarium's core implementation uses the official Incus Go client for lifecycle and native `ExecInstance` operations. It separately ships `agent-box`, an in-box MCP service for an agent-oriented workspace (shell, files, logs and process management), often reached over SSH/stdio. That service is **an enhanced guest-facing MCP workload**, not a technical prerequisite for the native Incus container lifecycle or Exec.

Useful patterns to adopt: typed Incus client interface, bounded Exec output and exit handling, separate transport error from nonzero command exit, timeout semantics and independent native verification.

Patterns **not required** for BoxWeave P0: installing `agent-box` into every container, running SSH into guests to support control-plane Exec, a second guest MCP server, a resident process agent, or a special guest image with BoxWeave software baked in.

If future product requirements demand high-frequency interactive sessions or persistent in-guest control, assess an explicitly optional extension independently. Do not make such a client a universal prerequisite without a new owner decision.

## Containers versus VMs

Incus Linux **containers** support native Exec without a guest agent. Incus **virtual machines** may require the separate upstream `incus-agent` for Exec and file access. v1's P0 scope is Linux containers, not VMs. See [Incus Exec](https://linuxcontainers.org/incus/docs/main/instance-exec/) and [Containarium native client](https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/client.go).

## Acceptance

Create a fresh base-image container with no BoxWeave client, no SSH and no extra in-guest services. From authenticated MCP, perform lifecycle and guest-root async `box_exec` with actual stdout/stderr/exit and original Tracking ID recording. List/read and apply create/replace/delete to one guest regular file with checked SHA, run typed-argv async tests, disconnect/reconnect, inspect original output, request a supported long Exec cancel and rerun after a fix. Verify no guest agent was installed, the container remains unprivileged under native isolation, and forged Box identities or host paths cannot target the parent host. Repeat inside an independently installed nested BoxWeave where supported.
