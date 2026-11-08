# Approved v1 operating defaults

**Owner approval:** 2026-11-07. These fourteen choices are accepted for v1, subject to security invariants and actual Incus capability checks. A different option requires another owner decision, not an Agent's assumption.

| # | Approved choice | Contract |
|---|---|---|
| 1 | **B — guest root by default** | `box_exec` executes as UID 0 **inside the owned container** by default. An explicit supported guest UID/GID may override it. This does not grant host root, unrestricted Incus privilege, host mount/device access or command execution on the physical/parent host. |
| 2 | **A — default Ubuntu LTS image** | A reviewed Ubuntu LTS Linux-container image is the default, while callers may choose other supported, validated images. The exact image reference/digest is pinned during implementation. |
| 3 | **A — configurable default resources** | Provide sensible configurable defaults for CPU, RAM and disk; a caller may override them inside tested safe limits. Native host or nested resource availability is checked before dispatch. Exact numbers are engineering defaults, not an unreviewed owner choice. |
| 4 | **A — nesting opt-in** | Ordinary Boxes are not nested-Incus-capable by default. The parent enables the narrowly supported nesting configuration only when requested for a directly owned Box. |
| 5 | **A — reuse existing Incus** | Reuse a compatible already-installed Incus daemon, create/use dedicated BoxWeave-controlled project/resources, and refuse any attempt to seize pre-existing unrelated instances or settings. An explicitly invoked local installation/upgrade may reconcile dependencies; upgrading a shared daemon requires separate target-specific local approval. |
| 6 | **SUPERSEDED — no Caddy** | No BoxWeave-managed Caddy, HTTP domain routing, website TLS or reverse proxy; retain minimal native Incus TCP/UDP forwarding and operator-owned external proxies. |
| 7 | **Updated — version-aware dependency lifecycle** | A locally invoked installer/upgrade records required and observed dependency versions, automatically installs missing or upgrades outdated/incompatible **approved managed** dependencies to pinned compatible versions, then verifies. Shared Incus/foreign services require an explicit impact plan and separate local approval before their upgrade/restart; no blanket system package upgrade. |
| 8 | **Custom — loopback HTTP MCP** | The Go MCP service listens on `127.0.0.1` only over HTTP by default. Do not automatically issue TLS certificates, open a public management listener or force a Caddy proxy for MCP. The operator may independently expose it via HTTPS proxy or Cloudflare Tunnel. BoxWeave authentication/authorization remains mandatory. |
| 9 | **SIMPLIFIED — two roles** | Local administrator bootstrap and multiple independently revocable **admin** or **viewer** tokens. No per-Box delegation or fine-grained RBAC; viewer's read access may contain secrets. |
| 10 | **SIMPLIFIED — Box peer toggle** | A single `box_peer_set` allows/revokes one directional all-protocol private Box UUID pair; no arbitrary IP/CIDR, protocol or port ACL editor. |
| 11 | **A — fail closed on state loss** | A missing, corrupt or inconsistent SQLite ownership store blocks dangerous operations. An explicit host-local, independently verified recovery procedure is required; do not auto-adopt resources from labels or names. |
| 12 | **A — explicit product upgrades with dependency reconciliation** | Operators initiate local install/upgrade explicitly; that action automatically reconciles approved BoxWeave-managed dependency versions, without a periodic updater. Shared/foreign services require separate target-specific local approval. Nested installations are independent and never auto-updated by their parent. |
| 13 | **A — initial Ubuntu LTS host support** | Ubuntu LTS is the initial officially tested installation host OS, physical or nested. Other Linux distributions are added only after actual verification. This does not prohibit creating other supported Box guest images. |
| 14 | **B — CI-driven automatic source releases** | A successful `v1` branch build/verify of an **unpublished, valid incremented VERSION** automatically creates the immutable matching tag and GitHub Release with those exact tested assets. Do not republish the same version, build a fake binary, auto-upgrade any installation or change the stable installer selector. The current design-only `1.0.0` baseline has no runnable build artifacts and **must not trigger a pretend release**. Independent installed acceptance remains separate from source-CI publication. |

## Interpretation and precedence

The original 14 owner choices are retained in the table for historical traceability, including guest-root Exec, locally bound authenticated MCP, explicit local upgrades and CI-controlled publication. Later explicit decisions superseded Caddy, generic ACL, fine-grained Token scope and the custom patch/diagnostic platform.

The **permanent product boundary** is now authoritative over any earlier P0/P1 or default wording that conflicts with it. Final P0: 20 tools, native owned TCP/UDP forwarding, simple directional owned peer toggle, no Caddy, admin/viewer tokens, and native guest list/read/create/conditional replace/delete. Strong async Tracking/idempotency/UNKNOWN, nested parity and Incus dependency management are preserved.

See [permanent product boundary](PRODUCT_BOUNDARY.md), [MCP tools](../design/MCP_TOOLS.md) and [P0 operational policies](P0_OPERATIONAL_POLICIES.md).

## Permanent narrowing amendment

The owner's subsequent instruction explicitly removes Caddy and generic ACL/RBAC/diagnostic/text-patching machinery, while **retaining native Incus TCP/UDP forwards** and a simple Box-to-Box test peer grant. The dependency installer still supports taking **operational responsibility for an existing compatible Incus daemon**, with safety authorization for shared upgrades and no automatic adoption of foreign containers. Ubuntu LTS is the first tested guest image; other distributions are added as native support is verified. See [permanent scope](PRODUCT_BOUNDARY.md). This amendment overrides superseded historical defaults and phase labels above.
