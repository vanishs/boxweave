# ADR-004 — Basic feature adapters, native model credentials

Status: selected design; version-specific compatibility remains unverified. Inputs: SRC-026/027, REQ-011, REQ-012, REQ-013, REQ-014, REQ-015.

Options: port the full legacy console/model logic; implement narrow typed services against approved basic boundaries; create a universal policy/model/workflow platform. Select narrow services. This includes the requested features without importing old stack structure, API breadth or provider forks.

Tenant writes use authenticated native capability and atomic revision contract; refuse unsupported writes rather than emulate them. Port publishing is explicit TCP/UDP passthrough, never assumed TLS/LAN filtering. Prompt Graph stores composition only; no automatic execution.

For ZCode(GLM), separate MCP registration from model connection. Official MCP documentation lists user config at ~/.zcode/cli/config.json under mcp.servers; model documentation exposes native sign-in/model settings. These docs do not establish an installed-version automation API or successful GLM access. References: [MCP](https://zcode.z.ai/en/docs/mcp-services), [models](https://zcode.z.ai/en/docs/configuration), SRC-027.

Choose wizard-owned nonsecret selection + native supported sign-in/settings handoff by default. Only add direct configuration automation when a pinned supported contract and preservation fixtures are verified. Alternative server-managed model keys would violate the client credential boundary and unnecessarily add a secret/inference proxy. Manual handoff stays pending until the user's client actually verifies the choice; no Claude substitute or fake 'ready' state.

Validation: TASK-009 Tenant contract fixtures/real approved capability run; TASK-010 explicit isolated route/client test; TASK-011 prompt persistence/conflicts; TASK-008 real ZCode model setup. If automation cannot safely be supported, report its limit explicitly while retaining the native-guided basic wizard. No inference purchase or network mutation is authorized by this ADR.
