# Approved v1 operating defaults

**Owner approval:** 2026-11-07. These fourteen choices are accepted for v1, subject to security invariants and actual Incus capability checks. A different option requires another owner decision, not an Agent's assumption.

| # | Approved choice | Contract |
|---|---|---|
| 1 | **B — guest root by default** | `box_exec` executes as UID 0 **inside the owned container** by default. An explicit supported guest UID/GID may override it. This does not grant host root, unrestricted Incus privilege, host mount/device access or command execution on the physical/parent host. |
| 2 | **A — default Ubuntu LTS image** | A reviewed Ubuntu LTS Linux-container image is the default, while callers may choose other supported, validated images. The exact image reference/digest is pinned during implementation. |
| 3 | **A — configurable default resources** | Provide sensible configurable defaults for CPU, RAM and disk; a caller may override them inside tested safe limits. Native host or nested resource availability is checked before dispatch. Exact numbers are engineering defaults, not an unreviewed owner choice. |
| 4 | **A — nesting opt-in** | Ordinary Boxes are not nested-Incus-capable by default. The parent enables the narrowly supported nesting configuration only when requested for a directly owned Box. |
| 5 | **A — reuse existing Incus** | Reuse a compatible already-installed Incus daemon, create/use dedicated BoxWeave-controlled project/resources, and refuse any attempt to seize pre-existing unrelated instances or settings. Do not silently upgrade an existing daemon. |
| 6 | **A — dedicated Caddy** | BoxWeave controls its own isolated Caddy service/configuration for **Box application routes**. Do not edit another Caddy instance. Refuse port/domain conflicts instead of taking over foreign services. |
| 7 | **A — dependency installer** | A locally authorized installer checks and installs missing supported dependencies. It must not silently upgrade, overwrite or reconfigure already installed foreign services; provide a safe preflight and clear refusal when conflicts cannot be resolved. |
| 8 | **Custom — loopback HTTP MCP** | The Go MCP service listens on `127.0.0.1` only over HTTP by default. Do not automatically issue TLS certificates, open a public management listener or force a Caddy proxy for MCP. The operator may independently expose it via HTTPS proxy or Cloudflare Tunnel. BoxWeave authentication/authorization remains mandatory. |
| 9 | **A — scoped revocable tokens** | Local administrator initialization and multiple independently revocable credentials with constrained permissions. Bootstrap and token rotation/revocation require an audited local administrative path; no default anonymous MCP access. |
| 10 | **A — Box identity or verified Box IP** | ACL requests may use a Box identifier or a validated IP associated with a managed Box, with direction and optional protocol/port. Internally bind rules to stable native Box identities and re-resolve their actual IPs. Never manage foreign hosts or networks. |
| 11 | **A — fail closed on state loss** | A missing, corrupt or inconsistent SQLite ownership store blocks dangerous operations. An explicit host-local, independently verified recovery procedure is required; do not auto-adopt resources from labels or names. |
| 12 | **A — explicit upgrades** | The installed BoxWeave product and existing host dependencies are not silently upgraded. Operators initiate local upgrades explicitly; nested installations are independent and never auto-updated by their parent. |
| 13 | **A — initial Ubuntu LTS host support** | Ubuntu LTS is the initial officially tested installation host OS, physical or nested. Other Linux distributions are added only after actual verification. This does not prohibit creating other supported Box guest images. |
| 14 | **B — CI-driven automatic source releases** | A successful `v1` branch build/verify of an **unpublished, valid incremented VERSION** automatically creates the immutable matching tag and GitHub Release with those exact tested assets. Do not republish the same version, build a fake binary, auto-upgrade any installation or change the stable installer selector. The current design-only `1.0.0` baseline has no runnable build artifacts and **must not trigger a pretend release**. Independent installed acceptance remains separate from source-CI publication. |

## Interpretation

The user's selection was `1B, 14B`; item 8 was clarified to **loopback-only HTTP with external exposure configured by the operator**; all remaining choices are option A.

Local HTTP is secure only as a deliberately restricted listener, not a public plaintext service. Cloudflare Tunnel or another reverse proxy can terminate external HTTPS, but the BoxWeave MCP endpoint must still validate caller authorization. Public website routing through Caddy and **public management MCP transport are separate concerns**.

See [MCP binding](../design/MCP_BINDING.md), [guest management](../design/GUEST_MANAGEMENT.md), [versioning](../process/VERSIONING.md), [security](../design/SECURITY.md), and [open risks](../verification/OPEN_RISKS.md).
