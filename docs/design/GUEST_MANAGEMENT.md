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

Create, list, start, stop, delete, resource config and native Exec are issued by Incus on the containing host. Guest command stdout, stderr, exit status, timeout and transport interruption are captured by native Exec handling and the durable BoxWeave operation journal. A valid guest root (UID 0) is the owner's selected default; explicit supported UID/GID overrides are allowed. Guest root is not a host root command surface. Do not expose host paths, Incus sockets, devices or privileged instance configuration through guest Exec.

Native Incus file APIs can later support P1 guest file operations without requiring a guest-installed BoxWeave agent. The guest OS still needs ordinary runtime facilities needed by the specific command (e.g. an existing executable), but no BoxWeave-specific daemon.

## Contrast with Containarium

Containarium's core implementation uses the official Incus Go client for lifecycle and native `ExecInstance` operations. It separately ships `agent-box`, an in-box MCP service for an agent-oriented workspace (shell, files, logs and process management), often reached over SSH/stdio. That service is **an enhanced guest-facing MCP workload**, not a technical prerequisite for the native Incus container lifecycle or Exec.

Useful patterns to adopt: typed Incus client interface, bounded Exec output and exit handling, separate transport error from nonzero command exit, timeout semantics and independent native verification.

Patterns **not required** for BoxWeave P0: installing `agent-box` into every container, running SSH into guests to support control-plane Exec, a second guest MCP server, a resident process agent, or a special guest image with BoxWeave software baked in.

If future product requirements demand high-frequency interactive sessions or persistent in-guest control, assess an explicitly optional extension independently. Do not make such a client a universal prerequisite without a new owner decision.

## Containers versus VMs

Incus Linux **containers** support native Exec without a guest agent. Incus **virtual machines** may require the separate upstream `incus-agent` for Exec and file access. v1's P0 scope is Linux containers, not VMs. See [Incus Exec](https://linuxcontainers.org/incus/docs/main/instance-exec/) and [Containarium native client](https://github.com/FootprintAI/Containarium/blob/main/pkg/core/incus/client.go).

## Acceptance

Create a fresh base-image container with no BoxWeave client, no SSH and no extra in-guest services. From authenticated MCP, perform lifecycle and guest-root `box_exec` with actual stdout/stderr/exit recording. Verify no guest agent was installed, the container remains unprivileged under native isolation, and forged Box identities or host paths cannot target the parent host. Repeat inside an independently installed nested BoxWeave where supported.
