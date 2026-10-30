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
