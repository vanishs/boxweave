# BoxWeave v1

BoxWeave is an agent-oriented, single-host Incus management controller. Its sole public **management** interface is **Streamable HTTP MCP**. A local maintenance CLI supports installation, status, upgrades, recovery, and initial administrator setup.

> **Status: design baseline only.** The `v1` branch starts with an empty product source tree and no inherited v0.x application, migrations, CLI, Web UI, Containarium integration, or release assets. `VERSION=1.0.0` identifies this new specification baseline. No runtime, installation, security guarantee, or released 1.0.0 binary is claimed.

Core promises:
- Manage only resources this layer of BoxWeave owns.
- Run the same BoxWeave stack at any nesting level supported by Incus and the environment.
- Use native Incus identities and guest Exec, never an unrestricted host shell through MCP.
- Treat every mutation as a durable, reconcilable operation.
- Isolate private Box-to-Box networking by default while allowing outbound internet and access to public services.

Start with [documentation index](docs/INDEX.md), [product scope](docs/product/SCOPE.md), [requirements](docs/product/REQUIREMENTS.md), [architecture](docs/design/ARCHITECTURE.md), [security boundaries](docs/design/SECURITY.md), and [Agent rules](AGENTS.md).

No Containarium dependency is planned. The proposed implementation stack is Go with the official Incus Go client and official MCP Go SDK; **the earlier Python choice has not been explicitly superseded by the owner**, so language selection is an open implementation decision. Irrespective of language, public management remains HTTP MCP only.

Version policy: `1.X.Y`; `X` increases for a completed feature increment and resets `Y` to zero; `Y` increases for a patch to existing behavior. See [versioning](docs/process/VERSIONING.md).
