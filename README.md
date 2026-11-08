# BoxWeave v1

BoxWeave is an agent-oriented, single-host Incus management controller. Its sole public **management** interface is **Streamable HTTP MCP**. A local maintenance CLI supports installation, status, upgrades, recovery, and initial administrator setup.

> **Status: design baseline only.** The `v1` branch starts with an empty product source tree and no inherited v0.x application, migrations, CLI, Web UI, Containarium integration, or release assets. `VERSION=1.0.0` identifies this new specification baseline. No runtime, installation, security guarantee, or released 1.0.0 binary is claimed.

Core promises:
- Manage only resources this layer of BoxWeave owns.
- Run the same BoxWeave stack at any nesting level supported by Incus and the environment; each instance manages only its own directly owned Boxes, not a nested hierarchy.
- Use native Incus identities and **agentless guest management**. Guest Exec runs as root *inside the owned container* by default; never expose an unrestricted host shell through MCP.
- Treat every mutation as a durable, reconcilable operation, with a stable Tracking ID and Go request context propagated throughout each layer.
- Isolate private Box-to-Box networking by default while allowing outbound internet and access to public services.
- Enable a **complete P0 Agent development loop** inside clean, agentless Boxes: create and verify readiness; list/read and safely apply create/replace/patch/single-file-delete; run typed-argv async guest Exec; wait/query or request evidence-based cancel; retrieve original output/exit and diagnostics by Tracking ID; iterate and independently verify.
- Enable testing and developing new BoxWeave installations inside ordinary managed Boxes, while preserving a future path to environment subleasing without implementing cross-layer management now.

Start with [documentation index](docs/INDEX.md), [P0 development loop](docs/product/DEVELOPMENT_LOOP.md), [product scope](docs/product/SCOPE.md), [requirements](docs/product/REQUIREMENTS.md), [architecture](docs/design/ARCHITECTURE.md), [Tracking/Context](docs/design/TRACKING_CONTEXT.md), [dependency lifecycle](docs/process/DEPENDENCY_LIFECYCLE.md), [security boundaries](docs/design/SECURITY.md), [approved defaults](docs/decisions/DEFAULTS.md), and [Agent rules](AGENTS.md).

**Confirmed implementation stack:** Go, the official MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`), and the official Incus Go client (`github.com/lxc/incus/v7/client` and its typed API models). Containarium is not a dependency. Incus's native SDK transport is internal; BoxWeave exposes **no public REST management API**. SDK versions and feature support must be validated before implementation. See [technology stack](docs/design/TECH_STACK.md).

Installation and operator-invoked upgrades record and reconcile the required/observed Incus, Caddy, Go SDK/build and other managed dependency versions, installing or upgrading approved dependencies when needed; shared-host service upgrades require their own explicit safety authorization.

Version policy: `1.X.Y`; `X` increases for a completed feature increment and resets `Y` to zero; `Y` increases for a patch to existing behavior. After a successful verified **new-version** build on `v1`, CI automatically publishes exact source assets; no automatic installed upgrades. See [versioning](docs/process/VERSIONING.md).

MCP runs over authenticated **loopback-only HTTP** by default. Public MCP HTTPS is separately configured by the operator, for example with Cloudflare Tunnel. **P0 Box application sites are also HTTP-only**, through a dedicated Caddy with no automatic website TLS, certificate issuance, redirect or port 443 management; application-site HTTPS is deferred. Direct public HTTP is unencrypted.

P0 mutation requests require a caller-generated reusable idempotency key; token revocation blocks new and provably undispatched tasks, without implying already-running work stopped. Network isolation is fail-closed; installation never automatically reboots the whole host.
