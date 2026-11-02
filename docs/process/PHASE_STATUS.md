# Current phase status

Authority for authorization/gates only; product version is [VERSION](../../VERSION), risk state is [OPEN_RISKS](../verification/OPEN_RISKS.md).

Phase7 is authorized by the user's supplied instructions (SRC-089). Entry audit accepts the phase6 closure of reviewed P0/P1 items; P2 deployment/evidence bounds are unchanged. Phase7 maintenance is complete within its tested scope (SRC-091); final result and actual command/evidence details belong to [PHASE_07_REVIEW](../verification/PHASE_07_REVIEW.md).

No deployment, push, inference, destructive cleanup or host/provider change is authorized by source/document cleanup. Source completion never implies the retained test deployment was upgraded. Future work requires its own request and the [MAINTENANCE](MAINTENANCE.md) path.

Historical gate/checkpoint text is preserved in [PHASE_STATUS_HISTORY](PHASE_STATUS_HISTORY.md); it must not override this file.

## Source branch promotion

SRC-092 separately authorizes publishing v2 as main and preserving the old implementation as legacy/pre-v2. This overrides the earlier no-push phase scope only for this branch operation; no application deployment, host change or runtime test scope is added. Completion/verification belongs in [BRANCH_PROMOTION](BRANCH_PROMOTION.md).

## 2026-10-29 remediation authorization

SRC-093: The owner authorizes R-00–R-09 source remediation in small direct commits to main, no PR and no force push; restoration of fresh-host install/update/repair and management CLI is in scope, starting at 0.10.0. No automatic 0.5.1 data migration or additional Agent platforms. This supersedes older no-push restrictions **for repository source commits only**; it does not authorize any host, Provider, secret, network, public TLS or deployed application changes. See [remediation audit](../verification/ARCHIVE_051_TO_MAIN_AUDIT.md) and [execution queue](../verification/REMEDIATION_TEST_QUEUE.md). Current phase 7 historical results remain unchanged.

## Current main-completion work

SRC-097 separately authorizes the isolated 0.14.0 implementation and experiment scope. Results and remaining delivery boundaries are recorded in [MAIN_014_REVIEW](../verification/MAIN_014_REVIEW.md); this does not upgrade the original deployment or authorize public exposure, model spending, destructive cleanup or source publication.

## Nesting source remediation

SRC-103 authorizes implementation and commit/push. SRC-100 dedicated-test scope remains; source publication does not authorize original parent-host changes. [Current source and actual evidence](../verification/NESTING_VERIFICATION.md).

## Remote MCP R0–R4 source delivery authorization (2026-10-31)

The owner requested completion of [PR #10](https://github.com/vanishs/boxweave/pull/10) R0–R4 and merge into main **without entering a real Box**. v0.24.0 source adds loopback remote MCP, local-admin OAuth, API Bearer, opaque path URL, optional Caddy/manual HTTPS ingress, and Web credential controls. It does not authorize public listener/TLS/firewall/DNS/Provider mutations. CI fixture results are not production OAuth, client or Box acceptance. [Handoff](REMOTE_MCP_HANDOFF.md) and [test plan](../verification/REMOTE_MCP_TEST_PLAN.md).

## ARCH-0 interstitial architecture gate (source audit only)

The owner requested a code-first modularity investigation before
continuing the next broad integration stage. The
[ARCH-0 review](../design/ARCHITECTURE_CONSOLIDATION_GATE.md) was
added to the [roadmap](../design/FEATURE_ROADMAP_MASTER.md) after
P1-A/P1-B1 and before P1-B2. This does not authorize an indiscriminate
rewrite, host installation, Provider mutation or release promotion.
ARCH-0 implementation and installed acceptance remain pending.
