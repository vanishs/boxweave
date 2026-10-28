# Scope and decisions for developer review

**Every disposition is a proposal; no feature removal or new release scope is approved yet.** Retain = preserve user outcome, not code; improve = change behavior/experience; remove = omit a legacy coupling or first-release feature, not delete old data/code; validate/defer = need developer demand/evidence. Normative behavior is only in [REQUIREMENTS](REQUIREMENTS.md) and [ACCEPTANCE](ACCEPTANCE.md).

## Every legacy feature

| Legacy ID | Proposed disposition | Reason, source and requirement mapping | Scope dependency |
| --- | --- | --- | --- |
| F-001 | Improve install/update; remove mandatory Web coupling | Recovery is central; console build should not be an inferred prerequisite for agent work. SRC-002; REQ-008/009/016 | DEC-001/003 |
| F-002 | Improve enrollment | Preserve reviewable endpoint/trust and selection intent; clarify integrity. SRC-003/004; REQ-001/002/016 | DEC-002 |
| F-003 | Retain supported adapter outcomes; validate launch matrix | Avoid a universal compatibility claim; start with measured profiles. SRC-004; REQ-001/003/017 | DEC-002 |
| F-004 | Validate/defer model-provider wizard from first release | Related but not essential to proving remote work. SRC-003/004; REQ-015 | DEC-004 |
| F-005 | Retain/improve independent diagnosis | Failure recovery must work without setup side effects. SRC-014; REQ-003/017 | DEC-001/002 |
| F-006 | Retain observation outcomes | Separate health evidence and exact target inspection are valuable; old command names not fixed. SRC-005; REQ-004/009 | DEC-001 |
| F-007 | Improve receipts and rediscovery | Existing durable jobs lack an approved end-user workflow; propose recoverable observation first. SRC-009/014; REQ-007 | DEC-001 |
| F-008 | Retain reviewed smoke as optional verification; defer recursive deployment product | Recorded nesting success is useful, not proof of a customer need for recursive management. SRC-009/011/016; REQ-017 | DEC-004 |
| F-009 | Validate/defer console; if retained improve safe independent access | An additional user interface is not evidence it belongs in core install. SRC-006/008; REQ-011/018 | DEC-003/005 |
| F-010 | Retain explicit environment selection/create/connect; defer console batch parity | Core work needs targeting/provisioning. Full resize/start/stop/delete batch UI is not automatically needed. SRC-006/010; REQ-004/005/006/011/018 | DEC-001/003/004 |
| F-011 | Validate/defer route/exposure management | Adds externally visible effects; no evidence first-release remote work needs it. SRC-010; REQ-013/018 | DEC-004 |
| F-012 | Retain truthful state observations; remove misleading monitoring promise | No runtime telemetry was verified; avoid an inferred observability product. SRC-001/010; REQ-006/009 | DEC-001/004 |
| F-013 | Validate/defer Tenant display/filtering | Personal project binding may serve the core job without requiring Tenant UI; cannot equate them. SRC-007; REQ-010/012 | DEC-001/004/005 |
| F-014 | Validate/defer native shared create and cross-Tenant policy writes | Significant enforcement/compatibility obligation; decide need before carrying it into a rewrite. SRC-007/010/018/019; REQ-012/018 | DEC-004/005 |
| F-015 | Validate/defer Prompt Graph from first release | Context composition is a different user job; retain as an option without pretending it executes work. SRC-006; REQ-014 | DEC-004 |

## Proposed additions and explicit non-goals

Add explicit project/environment binding (REQ-010, new hypothesis) and improve operation rediscovery (REQ-007, legacy-derived). Add measurable source-integrity and per-profile evidence requirements (REQ-016/017), replacing contradictory prose with reviewable outcomes. These are proposals, not externally researched recommendations or commitments.

Proposed first-release exclusions: hosted billing/multi-tenant SaaS; arbitrary workflow/Prompt Graph execution; transparent multi-provider framework before another provider is selected; universal OS/agent promise; existing-box network-Tenant reassignment; automatic provider/Incus upgrades; full-host rollback guarantee; automatic public exposure; security certification from nesting/configuration. No requirement mandates preserving old API/CLI spelling, local database, service layout or framework. No destructive change to the legacy project is authorized.

## Five pending product decisions

All DEC statuses: **pending developer response**. IDs are product review questions, not architectural ADRs. Recommended defaults help review; silence never approves them.

| ID | Question and alternatives | Recommendation / benefit / cost | Affected requirements and risks |
| --- | --- | --- | --- |
| DEC-001 | Primary release: A personal Agent-to-project workflow; B infrastructure console; C recursive environment platform. Include explicit project binding now, or keep manual selection initially? | A with explicit binding, subject to demand confirmation. Clear repeatable core path; loses immediate console/recursive parity and adds binding validation scope | REQ-004..010; RISK-001/002 |
| DEC-002 | Launch coverage: A Windows + Codex first (native SSH; require real WSL bridge evidence if included); B all old agents/runtimes at once; C choose another named profile | A as a hypothesis based on current workspace context, not proven demand. Smaller real acceptance matrix; other users wait. Developer must explicitly name required profiles, and can choose B/C | REQ-001/002/003/015/016/017; RISK-003/005 |
| DEC-003 | Web delivery: A optional later interface; B optional but first-release requirement; C mandatory console installed with core | A. Avoid coupling Agent onboarding/recovery to a larger UI release; reduced visual administration initially | REQ-008/011; F-001/009/010; RISK-002/009 |
| DEC-004 | Capability breadth: A defer Tenant writes/routes/Prompt Graph/model wizard/recursive deployment; B retain selected named items; C retain full legacy parity | A, unless one is central to your actual job. Less enforcement/acceptance burden; capability loss versus old product. Read-only trustworthy target/state evidence remains core. Decide full lifecycle batch controls explicitly if needed | REQ-011..015/017; F-004/008/010..015; RISK-002/006/008/013 |
| DEC-005 | Trust audience: A one trusted operator with documented provider privilege; B shared team with enforced per-user/resource scope; C publicly offered service | A for initial scope. Smaller authorization model; excludes shared untrusted access. B requires early positive/negative authorization evidence; C needs a separately expanded product/security scope. Neither A nor B permits trust bypass | REQ-011/012/018; RISK-006/007 |

## Approval gate

Developer must confirm direction, selected profiles/features and key behavioral definitions, including unknown-outcome/no-replay, trust failure, replacement intent and recovery limits. Record each DEC response with a new SRC entry; update affected REQ/AC priorities/status and omissions consistently. A changed recommendation needs revised acceptance, not an unrecorded exception. Only then mark the phase 2 gate satisfied. Authorization to analyze phase 2 is not approval to enter phase 3; no technology stack or architecture is selected here.
