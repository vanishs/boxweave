# Decision index

This index owns decision status/navigation, not copied decision bodies. Runtime mapping is [ARCHITECTURE](../ARCHITECTURE.md); outcome boundaries are [SCOPE](../../product/SCOPE.md).

| ADR | Status / retained decision | Revisit when |
| --- | --- | --- |
| [001 modular app](0001-modular-application.md) | Accepted, implemented: one app/shared services, independent worker and Web/MCP failure planes | Measured coupling/concurrency or newly approved workload requires change |
| [002 intent/recovery](0002-intent-and-reconciliation.md) | Accepted, implemented: transactional intent/unknown/revisions; additive unbind metadata | Recovery/transaction contract changes, never to hide uncertain outcomes |
| [003 trust/transport](0003-transport-and-trust.md) | Accepted, implemented: upstream SSH/MCP, separate credentials, operator-reviewed artifacts | New identity/exposure/runtime/signing contract; direct REST proposal was superseded by actual MCP adapter |
| [004 bounded features/models](0004-basic-features-and-models.md) | Accepted, implemented selected scope: native gated access, passthrough, public/native model handoff | New native API/catalog or selected feature boundary changes |
| [005 phase4 boundary](0005-phase-four-boundary.md) | Historical milestone, superseded as current scope/gate by ARCHITECTURE/FEATURE_MATRIX | Consult only to explain original fixture/live limitations, not to declare current blockers |
| [007 host install boundary](0007-host-install-boundary.md) | Accepted direction, partial implementation: pinned upstream root boundary and safe app-user release | Complete controlled bootstrapping and real host verification |
| [006 prompt composition](0006-prompt-composition.md) | Accepted, implemented: revisioned context, not execution | Explicitly approved workflow semantics or storage needs |

Keep rationale/consequences in each ADR. A changed decision needs its source, old decision link, new choice, implications and regression plan; mark superseded records rather than silently deleting them.

[ADR-008 independent nesting](0008-independent-nesting.md): accepted SRC-103, implemented source0.16.0; privileged live acceptance pending.

[ADR-009 remote MCP HTTP(S), authentication and no tunnel-vendor coupling](0009-remote-mcp-transports-and-auth.md): **accepted, v0.24.0 source implemented; live acceptance pending** (2026-10-31). Keeps ADR-003 SSH runtime intact; HTTPS/OAuth/API Token/opaque URL credential are new future transport plans, and unified SSH+HTTP identity/RBAC remains deferred. [Master](../REMOTE_MCP_MASTER.md).

[ADR-010 MCP-first + REST-assisted hybrid Provider policy](0010-api-first-control-plane-candidate.md): **OWNER-SELECTED POLICY (2026-11-05); implementation acceptance remains feature-specific.** New/specialist Containarium operations prefer upstream MCP; the seven app-side lifecycle REST methods remain; other REST use needs a separate demonstrated reason and acceptance. The historical universal API-first proposal and batch **8+21=29** MCP-to-REST migration are **ABANDONED** (PR #77 merged and then reverted). Coequal ADR-003 trust/transport remains effective; [architecture master](../API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md), [archived migration research](../API_FIRST_PROVIDER_MIGRATION_PLAN.md), [protocol-neutral diagnostics](../UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md).
