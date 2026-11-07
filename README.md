# BoxWeave v1

BoxWeave is an agent-oriented, single-host Incus management controller. Its sole public **management** interface is **Streamable HTTP MCP**. A local maintenance CLI supports installation, status, upgrades, recovery, and initial administrator setup.

> **Status: design baseline only.** The `v1` branch starts with an empty product source tree and no inherited v0.x application, migrations, CLI, Web UI, Containarium integration, or release assets. `VERSION=1.0.0` identifies this new specification baseline. No runtime, installation, security guarantee, or released 1.0.0 binary is claimed.

Core promises:
- Manage only resources this layer of BoxWeave owns.
- Run the same BoxWeave stack at any nesting level supported by Incus and the environment; each instance manages only its own directly owned Boxes, not a nested hierarchy.
- Use native Incus identities and **agentless guest management**. Guest Exec runs as root *inside the owned container* by default; never expose an unrestricted host shell through MCP.
- Treat every mutation as a durable, reconcilable operation.
- Isolate private Box-to-Box networking by default while allowing outbound internet and access to public services.
- Enable testing and developing new BoxWeave installations inside ordinary managed Boxes, while preserving a future path to environment subleasing without implementing cross-layer management now.

Start with [documentation index](docs/INDEX.md), [product scope](docs/product/SCOPE.md), [requirements](docs/product/REQUIREMENTS.md), [architecture](docs/design/ARCHITECTURE.md), [security boundaries](docs/design/SECURITY.md), [approved defaults](docs/decisions/DEFAULTS.md), and [Agent rules](AGENTS.md).

**Confirmed implementation stack:** Go, the official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`), and the official Incus Go client (`github.com/lxc/incus/v7/client` and its typed API models). Containarium is not a dependency. Incus's native SDK transport is internal; BoxWeave exposes **no public REST management API**. SDK versions and feature support must be validated before implementation. See [technology stack](docs/design/TECH_STACK.md).

Version policy: `1.X.Y`; `X` increases for a completed feature increment and resets `Y` to zero; `Y` increases for a patch to existing behavior. After a successful verified **new-version** build on `v1`, CI automatically publishes exact source assets; no automatic installed upgrades. See [versioning](docs/process/VERSIONING.md).

MCP runs over authenticated **loopback-only HTTP** by default. Public MCP HTTPS is separately configured by the operator, for example with Cloudflare Tunnel; Caddy remains responsible for Box **application** sites, not mandatory MCP TLS termination.
