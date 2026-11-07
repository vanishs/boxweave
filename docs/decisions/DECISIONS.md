# Decision register — v1 design baseline

Recorded from the owner's current design discussion. **Accepted** means a product requirement, not an implemented feature. **Candidate** is a recommendation awaiting a separate decision.

| ID | Status | Decision |
|---|---|---|
| D-001 | Accepted | Start a blank-source v1 branch and version it 1.0.0; 1.X.Y increments X for features, Y for patches. |
| D-002 | Accepted | Single physical host per layer; public management exclusively Streamable HTTP MCP, no public REST/Web console. |
| D-003 | Accepted | Remove Containarium, v0.x migrations, Tenant UI and mandatory SSH; use native Incus and guest Exec. |
| D-004 | Accepted | Box identity is the native Incus UUID, subject to verification; names/IPs may be reused without inheriting authority. |
| D-005 | Accepted | Default private Box-to-Box direct isolation, explicit one-way grants, default internet access, optional per-Box denial. |
| D-006 | Accepted | A Box may access another Box's public Caddy website; bypass protection for that public path is not P0. |
| D-007 | Accepted | HTTP routes and TCP/UDP forwards are owned, Box-targeted services; Caddy/Incus unmanaged services cannot be published via MCP. |
| D-008 | Accepted | Mutations are asynchronous durable BoxWeave operations with dedup, concurrency protection, UNKNOWN preservation and post-state verification. |
| D-009 | Accepted | Fully compatible self-hosting with no artificial depth limit; each independent layer has the same functions but manages only its directly owned Boxes, not a recursive management tree. |
| D-010 | Accepted | Strict resource ownership boundary even for superadmin MCP calls; no generic host shell, host file operations or arbitrary native API passthrough. |
| D-011 | Accepted | No mandatory storage driver; backup deferred, guest file/snapshot tools later P1. |
| D-012 | Accepted | A limited host-local install/status/upgrade/credential recovery CLI exists despite no Web UI. |
| D-013 | Accepted | All project artifacts English; Agents communicate in the requester's preferred/current language. |
| D-014 | **Accepted** | Implement v1 in Go using the official Model Context Protocol Go SDK and official Incus Go client/module; Python is superseded as the v1 implementation language. |
| D-015 | Candidate | Dedicated BoxWeave-owned Caddy instance and restricted local Incus helper as the implementation means for the accepted security boundary. |
| D-016 | Accepted | Internal Incus SDK/API calls are allowed; the no-REST constraint applies to BoxWeave's public management surface. Do not build or expose a second public management API. |
| D-017 | Accepted | Nested installs use the same build and bootstrap/maintenance workflow; each layer treats its own environment as host and never silently modifies parent infrastructure. |
| D-018 | Accepted | Public routing, forwarding and private ACL changes must be bound to an owned Box identity; no arbitrary host, remote service or unmanaged target may be published via MCP. |
| D-019 | Accepted | Native operation receipts must be retained separately from BoxWeave durable business-operation receipts; ambiguous results stay UNKNOWN instead of automatic replay. |
| D-020 | Accepted | Nested operation is a compatibility property for development/testing new BoxWeave versions in managed Boxes and possible future environment subleasing; it is not a v1 cross-layer administration or delegation feature. |
| D-021 | Accepted | A parent treats a Box hosting an inner BoxWeave as an ordinary owned Box; normal deletion semantics apply with no inner-controller discovery, recursive protection or extra deletion handshake. |
| D-022 | Accepted | Cross-layer ingress is configured through ordinary separately authorized route/forward actions per layer; no automatic multi-layer publish/delegation protocol is in v1 scope. |
| D-023 | Accepted | All fourteen operator defaults were approved on 2026-11-07, as specified in [DEFAULTS.md](DEFAULTS.md): item 1B, item 14B, clarified item 8 (loopback-only HTTP with externally configured proxy/Tunnel), and A for the other items. |
| D-024 | Accepted | Normal Linux Boxes require no installed BoxWeave guest client, resident Agent or SSH to be managed; lifecycle/Exec uses the native Incus Go SDK. Guest root is the default Exec identity; there is no host root MCP surface. |
| D-025 | Accepted | Only a compatible, verified, unissued VERSION on a successfully built `v1` CI run triggers immutable GitHub tag/Release publication; existing installed BoxWeave updates and stable installer promotions remain separately initiated. |

Before overturning an accepted item, update product requirements, the affected design and acceptance tests and explicitly record why. Do not treat archived v0.x docs or external repositories as a competing source of authority.
