# v1 permanent scope and exclusions

**Current v1 scope:** Embedded OAuth is part of the same /mcp service, with no application permission Scope or admin/viewer role classes. Every authorized Actor receives explicit full-management consent bounded to owned Layer/Box resources, with refresh/revocation and original Actor-idempotency protection. See [current status](../INDEX.md). OAuth does not imply a Web console, generalized RBAC, Cloudflare management or a second transport.

**Status:** This is the intentionally small product boundary, not merely a P0/P1 rollout order. Supersedes superseded older broad design assumptions; see [boundary decision](../decisions/PRODUCT_BOUNDARY.md).

## Required core

- One BoxWeave controller per host/layer, implemented in Go with official MCP and Incus Go SDKs. No Containarium, Web console, public REST management, guest management Agent, SSH dependency or generic host shell.
- HTTP MCP listens on loopback by default; external HTTPS/Tunnel remains operator-owned. Local CLI is limited to authorized fresh install/retirement/status/bootstrap/token recovery/Doctor/Recover and reviewed Incus dependency lifecycle; independently revocable credentials grant **one** approved, Layer/Box-bounded management capability. No admin/viewer split, permission Scopes or delegated per-Box RBAC.
- Managed Incus Linux Boxes with native UUID-verified ownership: create/start (default ready), list/get, typed resource update, stop/delete, nesting opt-in and guest-root asynchronous Exec. Ubuntu LTS verified first; other Linux guest images added with explicit compatibility tests.
- Guest file list/read and exclusive `box_file_apply(mode=create)`; unsafe conditional replace/delete are refused. Complex edits, repository Git and build/test commands run inside the authorized Box via `box_exec`; no built-in patch engine.
- Mutations are journaled and async; caller-provided reusable idempotency key, server Tracking ID, distinct durable operation IDs, bounded/retained output, honest UNKNOWN and independent native post-state reads. `operation_get` includes redacted diagnostic summary; no standalone diagnostic MCP tool.
- Safe network policy: private Box-to-Box deny by default, one directional `box_peer_set` between exact owned Box identities for inter-Box testing, verified native IPv4/enabled IPv6 enforcement. Internet outbound by default with per-Box deny switch. Fail closed if isolation cannot be proven.
- **Keep** only native Incus `port_forward_list/create/delete` for TCP/UDP exposure of exact owned Box UUID + guest port, checking bind collisions and external exposure. No managed domain/website proxy.
- Reuse existing compatible Incus daemon where possible; manage approved dependency version installation/upgrades from an explicit **host-local** operator action. Separately approve disruptive upgrades of shared Incus; no foreign project/resource adoption. Never auto-reboot host.
- Same functional contract for L0/L1/L2 nesting when native capabilities permit; no parent-to-child controller orchestration, shared tokens or cross-layer automatic routing. Controlled RC publication and independently signed stable release have distinct exact-source gates and owner authorization; live host changes remain explicit.
- Full negative/ownership and real create/edit/build/fix/retest acceptance; keep Finder/Fixer/Verifier tools in repository/CI/test Boxes, not production runtime.

## Permanently outside the product unless separately reapproved

- Dedicated Caddy instance, Caddy API, BoxWeave-managed website hostnames/routes, TLS/certificate management, domain proxy, custom port-forwarding service.
- General Incus ACL/CIDR/IP/protocol/port editor, free-form firewall controls, multi-resource network controller, whole-host service configuration.
- Multi-user tenants, per-Box Token delegation, role hierarchy, general identity-provider integrations and billing/meters.
- Native snapshots and backup platform, bulk file upload, text-patch engine, guest file search, persistent background process supervision, automatic idle sleep/HTTP wake.
- Virtual machine management, multi-host orchestration, arbitrary privileged Incus/host operations and automatic adoption of foreign containers.
- Automatic parent/child coordination, unsafe partial/UNKNOWN replay or unverified host-namespace privilege.

**New functionality requires a new owner-approved product-scope decision, not merely shifting an excluded item to P1.**
