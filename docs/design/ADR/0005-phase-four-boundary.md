# ADR-005 — Phase 4 implementation boundary and evidence

Status: historical phase-four checkpoint. Later real execution evidence and current architecture supersede its pending gate; see [decision index](INDEX.md).

Status: implemented subset, live acceptance blocked; 2026-10-28. Inputs: SRC-030/032, TASK-001 through TASK-005.

The approved target lacked an installed managed MCP launcher and had no reviewed independent application launch/credential path. Alternatives: borrow the existing Agent token, mint a new identity, or preserve a runnable explicit-fixture slice while reporting the live gate incomplete. The approved plan forbids the first two; developer subsequently requested preservation. Choose the third. This does not change the required live acceptance or remove any selected feature.

Real implementation uses React/TypeScript, Flask/Waitress, real SQLite and a separate worker process in the existing non-root project account for fixtures. Production independent service accounts/credentials and HTTPS frontend are not installed. The approved execution plan allowed loopback fixture HTTP; no listener was exposed on the private/public network.

Minimal schema stores the project binding in the project row and uses typed named application methods rather than introducing unused interface frameworks. Plans, operations, sessions and audit remain distinct tables; revision/intent semantics remain as designed. The only action is fixed identity + hello. CLI currently observes shared state; full CLI submission/reconciliation/unbind and Agent enrollment remain unfinished, not fake stubs. Default fixed-operation budget is 60 seconds, bounded at 600; provisioning readiness is not implemented by this fixed-command slice.

Native Windows work provides an observation-only connection probe with an explicit public descriptor and no config/trust repair. It is not the full enrollment/registration diagnostic promised for release. POSIX MCP parser and Windows successful protocol path still need real-provider/client evidence.

Validation: 29 Linux tests, clean hash-locked reproduction, frontend build, actual Chromium application/DB/restart tests with fixture infrastructure, and actual Windows PowerShell AST/three no-network rejection cases. Direct Platform MCP identity/hello succeeded but is separate evidence. No combined real Web->worker->MCP pass is claimed. Phase 4 remains partial; developer acceptance and phase 5 remain blocked on the actual minimum-system gate.

## Current result — real chain verified (SRC-049/050)

Version 0.1.1 fixes a real queue-claim timing race. Thirty Linux tests pass, including a new regression that failed before the fix. An explicitly approved nesting-enabled backend now supports the real application -> independent managed MCP -> inner workload path. HTTP and Chromium binding/confirm/hello/receipt/restart/no-replay plus live trust/identity rejection passed. See [LIVE_MCP_CHAIN](../../verification/LIVE_MCP_CHAIN.md) for exact identities, operation IDs, versions, commands and limits.

The previous missing-launcher/nested-startup blocker is resolved for this new lab only, not the old box or every LXC. TASK-005 selected Web path has live evidence; real Windows-native enrollment/Agent GUI acceptance remains incomplete. Developer required this real chain before phase 5 (SRC-048); no phase 5 feature was started and architecture/slice acceptance must still be recorded before TASK-011 or other new features. Original published demo remains fixture; isolated test servers are stopped, resources retained, no new public port or host-security change.
