# ADR-005 — Phase 4 implementation boundary and evidence

Status: implemented subset, live acceptance blocked; 2026-10-28. Inputs: SRC-030/032, TASK-001 through TASK-005.

The approved target lacked an installed managed MCP launcher and had no reviewed independent application launch/credential path. Alternatives: borrow the existing Agent token, mint a new identity, or preserve a runnable explicit-fixture slice while reporting the live gate incomplete. The approved plan forbids the first two; developer subsequently requested preservation. Choose the third. This does not change the required live acceptance or remove any selected feature.

Real implementation uses React/TypeScript, Flask/Waitress, real SQLite and a separate worker process in the existing non-root project account for fixtures. Production independent service accounts/credentials and HTTPS frontend are not installed. The approved execution plan allowed loopback fixture HTTP; no listener was exposed on the private/public network.

Minimal schema stores the project binding in the project row and uses typed named application methods rather than introducing unused interface frameworks. Plans, operations, sessions and audit remain distinct tables; revision/intent semantics remain as designed. The only action is fixed identity + hello. CLI currently observes shared state; full CLI submission/reconciliation/unbind and Agent enrollment remain unfinished, not fake stubs. Default fixed-operation budget is 60 seconds, bounded at 600; provisioning readiness is not implemented by this fixed-command slice.

Native Windows work provides an observation-only connection probe with an explicit public descriptor and no config/trust repair. It is not the full enrollment/registration diagnostic promised for release. POSIX MCP parser and Windows successful protocol path still need real-provider/client evidence.

Validation: 29 Linux tests, clean hash-locked reproduction, frontend build, actual Chromium application/DB/restart tests with fixture infrastructure, and actual Windows PowerShell AST/three no-network rejection cases. Direct Platform MCP identity/hello succeeded but is separate evidence. No combined real Web->worker->MCP pass is claimed. Phase 4 remains partial; developer acceptance and phase 5 remain blocked on the actual minimum-system gate.
