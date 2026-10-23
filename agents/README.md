# Agent adapters

BoxWeave does not assume that every AI coding agent has the same settings file or permissions model.

Currently supported:

- `codex/`: Codex Desktop on Windows, configured using `clients/windows/setup.ps1`.

Future adapters should implement the agent's MCP registration and instruction policy generation **without changing the Ubuntu transport or the Containarium provider**. Keep agent-specific paths and syntax in the adapter, not in shared server configuration. The Windows script exposes an `-Agent` selector that currently only accepts `Codex`.

## Operational knowledge for agents

Read the repository's [AGENTS.md](../AGENTS.md) for execution rules and the
[README operating expectations](../README.md#operating-expectations-timeouts-startup-and-nesting)
for the human-facing explanation. These are shared provider behaviors, not
Codex-specific exceptions:

- Reconcile a timed-out create against inventory before retrying creation.
- Wait for provisioning, SSH and provider API readiness within a recorded
  deadline. Running containers and active daemons can still be initializing.
- Verify actual MCP connect identity and workload output; API reachability and
  CLI exit codes alone do not prove end-to-end operation.
- Use the managed public identity for the MCP host at the current nesting layer.
- Preserve host-key verification and diagnose persistent errors rather than
  deleting instances, replacing keys or changing provider infrastructure.
- Separate the minimal nested smoke test from optional recovery/security audits.

Policy generators should include these rules so newly enrolled agents receive
them, not just link documents that may be absent from a remote workload checkout.
Keep deployment-specific addresses and public keys in generated policy only;
do not embed experiment identities in generic adapter instructions. Existing
global policy files do not change merely because the repository was edited.
