# Product proposal for review

Phase 2, 2026-10-28 America/Los_Angeles. **Draft for developer decision; product scope is not approved.** [Requirements](REQUIREMENTS.md) and [acceptance criteria](ACCEPTANCE.md) own behavioral definitions. [Scope](SCOPE.md) owns feature dispositions and the five pending decisions. Source IDs resolve in the [register](../knowledge/SOURCE_REGISTER.md).

## Value and evidence

Proposed promise: a developer can connect a supported coding agent to an intended remote project environment, perform explicitly authorized work there, and understand what actually happened when connection or execution fails. Trustworthy outcomes and recoverability are more important than reproducing the old console's page list.

Evidence: SRC-001/003 describe a personal agent-and-machine entry point; SRC-004/014 show enrollment and diagnosis pain; SRC-009/011 show the importance of identity, deadlines and ambiguous outcomes. These establish legacy context, not validated market demand or a user-approved v2 direction. The user has authorized this analysis (SRC-023), not its proposed feature choices. No interviews, adoption metrics, willingness-to-pay data or fresh client acceptance exist.

## Proposed users and jobs

- Primary: a developer using their own or explicitly entrusted Ubuntu development server. Job: get from an agent on a client computer to the correct project environment without manual identity guesswork or opaque failure.
- Supporting: the server operator, potentially the same person. Job: install/update without losing configuration/data, see readiness and recover a failed operation with known limits.
- Conditional: an administrator managing shared resources. Their authorization and isolation needs are materially broader; this is a separate scope decision, not implied by the personal workflow.

These roles are hypotheses drawn from sources. A personal deployment can still contain valuable data and privileged provider credentials; single-operator scope does not waive trust, safe targeting or explicit destructive-action authorization.

## Core scenarios and traceability

| Scenario | Desired outcome | Requirement / acceptance references |
| --- | --- | --- |
| First connection | User selects a supported client profile, reviews server identity, enrolls through an authorized channel and gets honest connection/registration evidence | REQ-001/002/003; AC-001/002/003 |
| Start project work | User selects an existing environment or authorizes a new one; actual identity is checked before a harmless project command | REQ-004/005/006; AC-004/005/006 |
| Connection fails | User can distinguish missing prerequisites, pin mismatch, denied provider access and unsupported configuration without invoking enrollment/repair | REQ-002/003; AC-002/003 |
| Create response disappears | User sees unknown/submitted rather than false failure; observation of the original target precedes any new create | REQ-005/006; AC-005/006 |
| Work outlives a connection | User finds the same receipt, distinguishes job query from command exit, and does not accidentally replay work | REQ-007; AC-007 |
| Update fails | Operator retains unrelated state, sees incomplete status and a bounded repair path rather than a false success banner | REQ-008/009; AC-008/009 |
| Choose a project again | Optional binding presents a verifiable target, not a stale IP or assumed cloned UUID identity | REQ-010; AC-010 |
| Share, expose or organize environments | Conditional console, Tenant access, routes and graph workflows retain evidence/authorization boundaries if included | REQ-011..015; AC-011..015 |

## Proposed success standards

For the selected release profiles, complete the reviewed end-to-end path on real clients and an assigned isolated environment; demonstrate negative/failure paths without duplicate mutations, silent configuration loss or fabricated success. Every supported claim must name the environment/version and verification level. REQ-017 and AC-017 define the release evidence gate; no timing SLA or platform coverage is promised without measurement.

Completion of this document is not achievement of those standards. The current phase performs documentation checks only. Product observations, authenticated provider operation, workload execution, actual packet reachability and effective confinement are different claims.

## Experience principles and boundaries

Proposed defaults: make the intended target and action understandable before a write; preserve user choices across generation and execution; show a durable next diagnostic/reconciliation step after uncertainty; separate observation from repair. Exact UI, command names, state storage, framework, protocol implementation, package layout and service design are phase 3 matters.

No first-release assumption of a hosted cloud, billing, arbitrary workflow scheduler, automatic Prompt Graph execution, universal agent/OS support, transparent provider interchangeability or security certification. These are proposed scope exclusions, pending the decisions in SCOPE. No old implementation is scheduled for removal by this document.

## Short review summary

Recommended first release: personal development connection/execution, safe environment targeting, independent diagnosis, honest asynchronous receipts and installation recovery. Add explicit project binding only after confirming that job is central. Make Web optional; defer Tenant writes, exposure management, model-provider configuration and Prompt Graph from the first release unless the developer identifies them as essential. The tradeoff is a smaller initial product with less console feature parity. All five DEC choices in SCOPE remain pending; no recommendation has become an approved requirement.
