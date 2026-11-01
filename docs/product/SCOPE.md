# Approved scope and basic feature boundary

Client setup decision SRC-086: the developer explicitly chose completing the Web public client setup entry rather than deferring it to CLI-only operation.0.9.1 supplies authenticated generation/download for the four selected profiles and a hash-gated local enrollment script with preserved replacement intent. Only public metadata/paths are inputs; no key contents, server-side enrollment/network connection or automatic execution. Existing authorized SSH identity/private client directory and independently reviewed source/pin remain prerequisites. This clarifies DEC003/REQ001/011/016 within the approved architecture.

Developer decisions recorded 2026-10-28 in SRC-026 supersede the earlier proposal's deferrals. Behavioral authority remains [REQUIREMENTS](REQUIREMENTS.md) and [ACCEPTANCE](ACCEPTANCE.md). Detailed boundaries below operationalize the approved basic design choices; they do not authorize live resource writes now.

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

Implementation scope has been selected, but runtime claims require independent current test evidence and authorization.

## 2026-10-29 approved restoration (SRC-093)

Owner approved restoring the **user-visible first-host `install.sh` and `boxweave.sh` management workflows** in independently reviewed current application code; priority is fresh Ubuntu provisioning, application update and repair, not automatic 0.5.1 data migration. The first-host path may coordinate approved Incus/Containarium prerequisites but must not silently upgrade or mutate an existing upstream, SSH/firewall, credentials, TLS, or release trust. Stage the work with verifiable preflight, explicit privilege boundary, artifact pinning and dedicated lab acceptance. The already approved Windows Codex/ZCode Native/WSL profiles remain unchanged; macOS, Cursor, Claude Code and other Agents are deferred. This scope decision is **not** implementation or deployment evidence. [Audit](../verification/ARCHIVE_051_TO_MAIN_AUDIT.md), [host boundary](../HOST_BOOTSTRAP.md).

## Independent nesting (SRC-103)

Include separate nesting choice in reviewed creation and exact managed-instance inspect/plan/apply through parent BoxWeave MCP. Preserve omitted legacy behavior; explicit unsupported selection fails before dispatch. Guest installer never changes parent policy. No automatic restart or privileged-container fallback. Configuration and effective child execution remain distinct. [Decision](../design/ADR/0008-independent-nesting.md).

## Remote MCP HTTP(S)/OAuth extension — source implemented, live scope limited (2026-10-31)

The owner selected a **single-operator** Streamable HTTP MCP addition compatible with existing SSH/stdio, with optional direct public HTTPS, optional self-managed HTTPS, user-operated reverse proxies and externally configured tunnels. No specific tunnel provider (including OpenAI or Cloudflare) becomes a BoxWeave dependency. OAuth uses the existing local admin login for human consent, alongside API Bearer Token and a separately verified opaque credential-bearing connection URL. The latter is a credential, **not anonymous access**; OAuth access tokens must never be placed in URL query parameters.

**Do not build** team/multiuser support, extra per-Agent/per-Box tool privilege tiers, separate URL lifetime tiers or a new unified SSH+HTTP identity/policy system in this slice. Those rights changes, if approved later, will be designed jointly for SSH and HTTP. Preserve current trusted-operator authority, explicitly disclose its breadth, implement mandatory transport authentication/credential hygiene, and keep all public exposure opt-in. The owner imposes no special public-vs-private network restrictions for explicitly enabled, properly secured HTTPS service.

Normative [REQ-021–024](REQUIREMENTS.md) and [AC-021–024](ACCEPTANCE.md); architecture/planned packages [REMOTE_MCP_MASTER](../design/REMOTE_MCP_MASTER.md), acceptance [REMOTE_MCP_TEST_PLAN](../verification/REMOTE_MCP_TEST_PLAN.md). **Status: v0.24.0 source implemented in PR #10; a historical v0.24.0 loopback HTTP/OAuth, local TLS proxy and nested-Box read were subsequently tested. This is not a current v0.25.0 installed-runtime, all-client or real public TLS acceptance.** See [code-first acceptance](../verification/CODE_FIRST_ACCEPTANCE_2026-11-01.md).


## D3/D4 agent-oriented diagnostics — owner approval (2026-11-01)

Include D3.1 app-receipt lookup, D3.2 durable classified direct-gateway MCP evidence and D4 narrowly bounded read-only guest project diagnosis using the **existing** SSH/HTTP gateway. Owner chose shared trusted-operator diagnostic depth, 7-day configurable gateway retention and 50 MiB default budget, best-effort automatic Agent diagnosis guidance and no server-driven repair/replay. Deep D4 access is **DENIED for all Boxes** until the server administrator explicitly authorizes an exact managed Box incarnation, selected check and service. Do not build per-Agent RBAC, an arbitrary guest command/filesystem debugger, cross-Box log search, independent Debug MCP or automatic external Issue publication. External diagnostics disclosure requires first a private draft and the user's personal sensitive-data review, then separate explicit final upload/submission approval. See [normative REQ-025/026](REQUIREMENTS.md), [AC-025/026](ACCEPTANCE.md) and [master](../design/DIAGNOSTICS_D3_D4_MASTER.md). Source implementation does not authorize real Provider/host mutation.
