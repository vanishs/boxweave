# Current phase status

Authority for authorization/gates only; product version is [VERSION](../../VERSION), risk state is [OPEN_RISKS](../verification/OPEN_RISKS.md).

Phase7 is authorized by the user's supplied instructions (SRC-089). Entry audit accepts the phase6 closure of reviewed P0/P1 items; P2 deployment/evidence bounds are unchanged. Phase7 maintenance is complete within its tested scope (SRC-091); final result and actual command/evidence details belong to [PHASE_07_REVIEW](../verification/PHASE_07_REVIEW.md).

No deployment, push, inference, destructive cleanup or host/provider change is authorized by source/document cleanup. Source completion never implies the retained test deployment was upgraded. Future work requires its own request and the [MAINTENANCE](MAINTENANCE.md) path.

Historical gate/checkpoint text is preserved in [PHASE_STATUS_HISTORY](PHASE_STATUS_HISTORY.md); it must not override this file.

## Source branch promotion

SRC-092 separately authorizes publishing v2 as main and preserving the old implementation as legacy/pre-v2. This overrides the earlier no-push phase scope only for this branch operation; no application deployment, host change or runtime test scope is added. Completion/verification belongs in [BRANCH_PROMOTION](BRANCH_PROMOTION.md).
