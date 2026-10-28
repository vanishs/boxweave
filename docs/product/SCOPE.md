# Approved scope and basic feature boundary

Developer decisions recorded 2026-10-28 in SRC-026 supersede the earlier proposal's deferrals. Behavioral authority remains [REQUIREMENTS](REQUIREMENTS.md) and [ACCEPTANCE](ACCEPTANCE.md). Detailed boundaries below operationalize 'basic' as phase 3 design choices; they do not authorize live resource writes now.

| Decision | Confirmed scope |
| --- | --- |
| DEC-001 | Personal Agent remote development, including project/environment binding |
| DEC-002 | Windows native SSH and WSL SSH with Codex and ZCode(GLM); other platforms/Agents later. 'WSL SSL' interpreted as SSH; WSL-hosted Agents are not assumed |
| DEC-003 | Basic Web included; independently enabled component, part of first-release delivery, not a prerequisite for MCP connectivity |
| DEC-004 | Basic Tenant network writes, port publication, Prompt Graph and model wizard included; recursive deployment deferred |
| DEC-005 | Single trusted operator; stop on trust failure, preserve unknown results without automatic retry, state recovery limits |

## Legacy dispositions after approval

| Legacy | Disposition and first-release boundary | Requirements |
| --- | --- | --- |
| F-001 | Improve installation/repair; deliver Web independently so its failure cannot interrupt a working MCP transport | REQ-008, REQ-009, REQ-016 |
| F-002 | Improve reviewed enrollment and payload integrity for selected profiles | REQ-001, REQ-002, REQ-016 |
| F-003 | Retain Codex and ZCode adapter outcomes; defer other Agents | REQ-001, REQ-003, REQ-017 |
| F-004 | Change model wizard to basic ZCode GLM configuration/guidance; no Claude adapter required | REQ-015 |
| F-005 | Retain independent diagnosis with honest supported/unsupported observations | REQ-003, REQ-017 |
| F-006 | Retain exact-target and health observations; command spelling not inherited | REQ-004, REQ-009 |
| F-007 | Improve durable receipt discovery/reconciliation, no automatic replay | REQ-007 |
| F-008 | Retain harmless direct-workload verification; defer recursive deployment/testing product | REQ-017 |
| F-009 | Include one local administrator, session/login/logout/reset and CSRF/Origin protection | REQ-011, REQ-018 |
| F-010 | Include list/detail/create/select/bind and confirmed start/stop/delete for one box per plan; defer resize and bulk workflows | REQ-004, REQ-005, REQ-006, REQ-011, REQ-018 |
| F-011 | Include TCP/UDP port passthrough list/add/remove; defer HTTP domain proxy editor, advanced routing and automatic public publication | REQ-013 |
| F-012 | Retain observed state/readiness; no fabricated telemetry product | REQ-006, REQ-009 |
| F-013 | Include Tenant metadata and provider-attributed membership/readiness views | REQ-012 |
| F-014 | Include native shared-Tenant creation and revision-checked directional ingress grant/revoke; unsupported provider stays gated, feature not dropped | REQ-012, REQ-018 |
| F-015 | Include persistent cards/comments/undirected links, copy and Markdown export with revision conflicts; no execution scheduler | REQ-014 |

## Basic boundaries and exclusions

Tenant writes: native shared membership at new-box creation and Tenant-wide ingress allow/revoke only; existing-box reassignment, custom egress policy editor, per-port Tenant ACL authoring and account-team tenancy are excluded. A degraded Guard can leave revoke pending; never label it enforced. Provider upgrade is separate approval; lack of capability blocks the write and its acceptance, not the rest of the interface.

Port publication: exact box/protocol/internal/external port and intended audience are reviewed before mutation. Passthrough does not supply source filtering/TLS; do not promise LAN-only security from an internal address. Each actual publication/public exposure requires its own authorization; this product approval creates no route.

Model wizard: select Z.ai/BigModel or supported endpoint/model for ZCode, guide native sign-in or validated configuration, preserve unrelated settings and never move API keys through server Web/API. If a pinned ZCode version lacks a stable automated setup contract, provide an explicit native-settings handoff with completion marked pending until verified. Do not claim automated model setup from MCP registration alone; never fall back to Claude. Exact model ID is versioned input, not a permanently embedded 'latest' value.

Web basics: project/binding overview, resource/Tenant/route views, exact single-resource action plans/receipts, prompt canvas and public setup-profile generation. Local administrator scope only. No cloud accounts, billing, shared-team RBAC, graph execution, generic provider framework, full-host rollback or implicit provider/Incus upgrade. Do not copy old code or delete old data.

Phase 2 gate is satisfied for the developer-confirmed direction and feature set. Phase 3 defines implementation boundaries/alternatives; all runtime validation remains pending. Phase 4 requires a new execution request; after its minimum system, developer confirmation is required before phase 5.
