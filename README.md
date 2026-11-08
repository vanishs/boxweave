# BoxWeave v1 — a small Agent-first Incus Box controller

> **Design only:** This independent `v1` branch currently contains no Go runtime, installer, tests, functional release, or inherited v0.x code. `VERSION=1.0.0` is a design baseline, not an installed product claim.

BoxWeave uses Go, the official MCP Go SDK and official Incus Go SDK. Its only public management protocol is **authenticated, loopback-only HTTP Streamable MCP** (optional operator-managed HTTPS/Tunnel). A local CLI handles install/upgrade, host status, token bootstrap and recovery.

**Only core features:**
- Manage exact **BoxWeave-owned Linux containers** via Incus UUID: create/start/get/list, typed safe resize/update, stop/delete; run async guest-root Exec with bounded original output and cancellation evidence.
- Minimal agentless guest files: list/read and one safe `box_file_apply(create|replace|delete)`. Complex code editing and Git run **inside the Box** through guest Exec, not a custom file patch engine.
- Track and recover every mutation using caller-provided reusable idempotency keys, server Tracking ID and durable async Operation ID; retain UNKNOWN rather than replaying ambiguous work.
**SDK P0 research status (not shipped functionality):** [Last executed Incus/MCP Go SDK P0 verification](docs/verification/P0_SDK_LAST_GATE_RESULTS_2026-11-08.md) passed official MCP Streamable HTTP and native Incus async Exec/recorded output, limited live Exec termination, SFTP exclusive create and IPv4 NIC anti-spoof in the nested lab. **Native Incus file APIs cannot atomically conditionally replace/delete a file based on its earlier hash**: `box_file_apply(replace|delete)` must report `UNSUPPORTED` if safe point-of-use guarantees are unavailable, not discard another writer's changes. Nested Cowsql instability is recorded as an environment issue: **BoxWeave v1 will not implement Incus database patches or a Cowsql repair service**. This does **not** mean the 20-tool v1 controller has passed acceptance.

- Native Incus **TCP/UDP port forwarding** from an explicitly authorized host port to an owned Box. Private Box-to-Box traffic denied by default; simple directional `box_peer_set` allows/revokes a peer pair for integration testing. **Selected design:** one installer-precreated, BoxWeave-owned Bridge NIC ACL shared by its managed Boxes; only exact-object ACL updates via the official Incus SDK + native Scriptlet authorization. No generic ACL editor.
- Full independently installed **nested BoxWeave** runtime when Incus/kernel permits; every layer manages only its directly owned Boxes.
- Version-aware local installation/upgrade, **including reuse and authorized version maintenance of existing Incus**. Do not silently import preexisting foreign containers or disrupt shared consumers. Never reboot the host automatically.
- Admin/viewer revocable tokens only; one physical/nested host per controller; Ubuntu LTS officially supported first, expand Linux guest images by real validation.

## Important: Incus-wide authorization change

**This v1 design is not installed yet.** The selected native `box_peer_set` backend uses one **preprovisioned shared ACL** across BoxWeave-owned NICs (directional Box A → Box B, **all ports**, with IPv4/IPv6 rules), a dedicated restricted TLS identity, native Incus **Authorization Scriptlet** and Go SDK **ETag** checks. It avoids runtime ACL creation, custom firewall code, OpenFGA and an extra privileged helper.

**Host-wide security/compatibility warning:** enabling `authorization.client.tls-restricted=scriptlet` changes the authorization path for **every restricted-TLS Incus client on this daemon**, not just BoxWeave. Automatic certificate Project restrictions are replaced by the Scriptlet's policy. A faulty policy could **deny existing applications or grant them excessive access**. The operator has approved this trade-off for the v1 product direction; it is **not** permission to silently overwrite an existing host policy.

The future **host-local installer** must clearly display this effect; inspect existing Scriptlet/client routing and restricted certificates; save the original configuration, policy and rollback instructions; preserve per-certificate Project restrictions for other clients; and perform permission-positive **and** privilege-denial checks before/after switching the route. If existing behavior cannot be safely preserved, installation of the peer capability must stop rather than widen permissions. BoxWeave's MCP runtime must never obtain unrestricted Incus administrator authority, arbitrary ACL-create rights or control over foreign resources.

**Real nested Incus 7.6 evidence:** [three-Box shared-ACL + Scriptlet + TLS-client compatibility test](docs/verification/SINGLE_SHARED_ACL_SCRIPTLET_ACCEPTANCE_2026-11-08.md). A→B IPv4/IPv6 and TCP passed; B→A/C→B were denied; revoke restored denial; native loopback TCP/UDP proxy still worked; a simulated existing restricted TLS client's tested permission matrix was unchanged. **This is not an installed v1 controller/real shared-host compatibility guarantee**. Incus under nested container creation also experienced repeat Cowsql database unavailability; root cause remains open.

**Not BoxWeave responsibilities:** Caddy, website/domain/HTTPS management, a general firewall/ACL/RBAC platform, tenant billing, snapshots/backup, automatic idle sleep/wake, background process supervisor, OAuth login, bulk file transfer or built-in patch engine, VM/cluster management, or cross-layer controller orchestration. These are **excluded**, not P1 promises.

The developer/acceptance workflow remains comprehensive and black-box-first. The [fault-driven Hello World -> failed delete -> nested candidate -> release scenario](docs/verification/FAILURE_DRIVEN_RELEASE_SCENARIOS.md) is the required cross-tool smoke/regression gate; it verifies actual LAN reachability, partial-delete evidence, safe nested localhost MCP testing and exact candidate bytes **before** CI auto-publication. Harnesses live in repository/CI/disposable Boxes, not the standard runtime.

Start with [permanent product boundary](docs/decisions/PRODUCT_BOUNDARY.md), [scope](docs/product/SCOPE.md), [20 MCP tools](docs/design/MCP_TOOLS.md), [Agent development loop](docs/product/DEVELOPMENT_LOOP.md), [networking](docs/design/NETWORK.md), [dependency lifecycle](docs/process/DEPENDENCY_LIFECYCLE.md), [security](docs/design/SECURITY.md) and [Agent rules](AGENTS.md).

Version scheme: `1.X.Y` (X feature increment, Y patch); genuine verified, new-version source builds on `v1` are published through CI automatically, never automatically deployed to running hosts. `main` remains the independent v0.x line.
