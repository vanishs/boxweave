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
| D-009 | Accepted | Fully recursive self-hosting with no artificial depth limit; same functions at each supported layer and independent trust domains. |
| D-010 | Accepted | Strict resource ownership boundary even for superadmin MCP calls; no generic host shell, host file operations or arbitrary native API passthrough. |
| D-011 | Accepted | No mandatory storage driver; backup deferred, guest file/snapshot tools later P1. |
| D-012 | Accepted | A limited host-local install/status/upgrade/credential recovery CLI exists despite no Web UI. |
| D-013 | Accepted | All project artifacts English; Agents communicate in the requester's preferred/current language. |
| D-014 | Candidate | Go + official Incus Go SDK + official MCP Go SDK as preferred implementation stack; previously chosen Python not explicitly reversed. |
| D-015 | Candidate | Dedicated BoxWeave-owned Caddy instance and restricted local Incus helper as the implementation means for the accepted security boundary. |

Before overturning an accepted item, update product requirements, the affected design and acceptance tests and explicitly record why. Do not treat archived v0.x docs or external repositories as a competing source of authority.
