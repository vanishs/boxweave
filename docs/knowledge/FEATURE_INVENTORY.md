# Legacy feature inventory

Status below describes source evidence, not a new requirement or a fresh pass. Sources: [register](SOURCE_REGISTER.md). Every row's implication is input to phase 2, not a keep/remove decision.

| ID | User scenario and implementation | Evidence/status | Meaning for v2 |
| --- | --- | --- | --- |
| F-001 | Install/update/repair Ubuntu MCP and mandatory Web components; settings precedence and provenance | SRC-002; static-confirmed installer/docs and sandbox tests exist | Capture recovery expectations separately from installer mechanics |
| F-002 | Generate editable, commit-pinned Windows/macOS enrollment script; stable SSH alias; native/WSL transport | SRC-003/004; static-confirmed | Clarify offline/download trust and supported profiles |
| F-003 | Enroll Codex/Cursor/Claude Code/ZCode; preserve unrelated config and explicit replacement | SRC-004; static-confirmed; dirty work excluded from committed baseline | Define conflict and replacement acceptance explicitly |
| F-004 | Optional Claude GLM/Z.ai/Zhipu settings | SRC-003/004; static-confirmed | Decide model configuration ownership in phase 2 |
| F-005 | Diagnose enrolled client trust, MCP discovery/read-only tool, registration and guidance | SRC-004/014; static-confirmed; real platform matrix incomplete | Keep distinct evidence for connection, registration and GUI |
| F-006 | CLI version/status/doctor/exact-name inspection | SRC-005; static-confirmed | Preserve truthful state semantics, review command surface |
| F-007 | Durable local command jobs with receipts, deadlines/output bounds and unknown state | SRC-009; static-confirmed | Decide whether resumable discovery/output continuation is needed |
| F-008 | Plan/approve nested inner-MCP create/connect/Python smoke | SRC-009/011; static plus historical-reported | Preserve no-duplicate-create reasoning; not automatic installation |
| F-009 | Console login/session/CSRF/local admin reset | SRC-006/008; static-confirmed | Resolve intended users and privilege model |
| F-010 | Box list/detail/create/resize/start/stop/delete via reviewed batch plans/jobs | SRC-006/010; static-confirmed | Submitted/running/ready and partial/unknown outcomes need separate acceptance |
| F-011 | Proxy and TCP/UDP passthrough route list/add/delete | SRC-006/010; static-confirmed | Exposure and reachability require explicit scope and evidence |
| F-012 | Monitoring verified resource state/config; no claimed live usage telemetry | SRC-001/010; static-confirmed | Define actual monitoring need without inheriting a dashboard label |
| F-013 | Tenant metadata, selection/filtering, native membership and Guard attribution | SRC-007; static-confirmed | Display group is not an isolation authority |
| F-014 | Shared native Tenant create and atomic cross-Tenant ingress grant/revoke | SRC-007/010; static-confirmed adapter gated by authenticated capability | Include only if product needs it; deployed support is unverified |
| F-015 | Persistent prompt cards, undirected relations, comments, copy history, Markdown export and optimistic revisions | SRC-006; static-confirmed | Graph is context/composition, no automatic agent execution |

Not implemented equivalently: provider-independent lifecycle adapter, standalone Linux client enrollment, box reassignment between native Tenants, automatic graph execution, whole-host transactional rollback, verified universal client GUI integration (SRC-001/003/010/014). These are boundaries, not v2 exclusions.

Candidate simplification questions: mandatory Web during server install; repeated enrollment logic across languages; overlapping old runbooks/source bundles; generic framework before a second provider exists. These are inferred review topics, not approved removals.
