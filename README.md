# BoxWeave v1 — a small Agent-first Incus Box controller

> **Design only:** This independent `v1` branch currently contains no Go runtime, installer, tests, functional release, or inherited v0.x code. `VERSION=1.0.0` is a design baseline, not an installed product claim.

BoxWeave uses Go, the official MCP Go SDK and official Incus Go SDK. Its only public management protocol is **authenticated, loopback-only HTTP Streamable MCP** (optional operator-managed HTTPS/Tunnel). A local CLI handles install/upgrade, host status, token bootstrap and recovery.

**Only core features:**
- Manage exact **BoxWeave-owned Linux containers** via Incus UUID: create/start/get/list, typed safe resize/update, stop/delete; run async guest-root Exec with bounded original output and cancellation evidence.
- Minimal agentless guest files: list/read and one safe `box_file_apply(create|replace|delete)`. Complex code editing and Git run **inside the Box** through guest Exec, not a custom file patch engine.
- Track and recover every mutation using caller-provided reusable idempotency keys, server Tracking ID and durable async Operation ID; retain UNKNOWN rather than replaying ambiguous work.
- Native Incus **TCP/UDP port forwarding** from an explicitly authorized host port to an owned Box. Private Box-to-Box traffic denied by default; simple directional `box_peer_set` allows/revokes a peer pair for integration testing. No generic ACL editor.
- Full independently installed **nested BoxWeave** runtime when Incus/kernel permits; every layer manages only its directly owned Boxes.
- Version-aware local installation/upgrade, **including reuse and authorized version maintenance of existing Incus**. Do not silently import preexisting foreign containers or disrupt shared consumers. Never reboot the host automatically.
- Admin/viewer revocable tokens only; one physical/nested host per controller; Ubuntu LTS officially supported first, expand Linux guest images by real validation.

**Not BoxWeave responsibilities:** Caddy, website/domain/HTTPS management, a general firewall/ACL/RBAC platform, tenant billing, snapshots/backup, automatic idle sleep/wake, background process supervisor, OAuth login, bulk file transfer or built-in patch engine, VM/cluster management, or cross-layer controller orchestration. These are **excluded**, not P1 promises.

The developer/acceptance workflow remains comprehensive and black-box-first, but harnesses live in the repository/CI/disposable Boxes, not the standard runtime.

Start with [permanent product boundary](docs/decisions/PRODUCT_BOUNDARY.md), [scope](docs/product/SCOPE.md), [20 MCP tools](docs/design/MCP_TOOLS.md), [Agent development loop](docs/product/DEVELOPMENT_LOOP.md), [networking](docs/design/NETWORK.md), [dependency lifecycle](docs/process/DEPENDENCY_LIFECYCLE.md), [security](docs/design/SECURITY.md) and [Agent rules](AGENTS.md).

Version scheme: `1.X.Y` (X feature increment, Y patch); genuine verified, new-version source builds on `v1` are published through CI automatically, never automatically deployed to running hosts. `main` remains the independent v0.x line.
