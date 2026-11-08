# v1 implementation plan — sustainable Agent development first

**Status:** design plan, not runnable code. The owner prioritized a self-sustaining Agent edit/build/test/diagnose loop before optional platform features. Every slice must preserve ownership restrictions, native SDK use, Tracking IDs and recursive runtime compatibility.

| Order | Slice | Minimum actual proof |
|---|---|---|
| 0 | Pin official Go MCP/Incus SDKs, dependency version manifest and required native API Extensions; freeze typed tool/Tracking/context schemas and privileged boundary | Manifest and hashes; compiled Go SDK smoke tests; observed Incus/Caddy/OS versions; reviewed security failure modes |
| 1 | Minimal Go service, local CLI installer/upgrade with automatic **approved dependency** reconciliation, admin/token bootstrap, loopback HTTP MCP and restricted native access | Clean Ubuntu LTS host, missing owned dependency auto-install, outdated owned dependency upgrade, shared Incus refusal/explicit approval, real MCP initialize/authorized and denied call |
| 2 | SQLite ownership and **universal Tracking ID / Go context**; durable async journal, bounded output store, proof-based `operation_cancel`, short `operation_get(wait_ms)` and diagnostic evidence | Same Tracking ID through ingress/worker/native adapter; duplicate idempotency one effect; wait does not cancel; queued cancel verified; crash/UNKNOWN preserves original native effect |
| 3 | Box lifecycle/typed allowlisted updates, default start/readiness; async guest-root `box_exec(argv,cwd,env,timeout,output limits)` and result retrieval; no disk shrink | Native Box identity/readiness, zero/nonzero guest Exec, disconnect/recovery and evidence-backed cancel without false termination claims |
| 4 | **Complete P0 development loop:** native guest file list/read and `box_file_apply(create|replace|patch|delete)`, safe hash metadata, scopes and diagnostics | Create/edit/failing test/debug/fix/passing test/single-file delete; stale hash, race and unsafe path refused; guest-root Exec may write guest despite file-tool scope |
| 5 | Private network topology, directional ACL grants, internet allowed by default/opt-out and revision-safe writes | Real IPv4/IPv6 same-bridge denial/allow, IP reuse, native enforcement and foreign-NIC preservation |
| 6 | Box-owned Caddy HTTP routes and native TCP/UDP forwards with durable reconciliation | Domain/port conflicts, owned target only, Caddy restart, foreign route denial, original operation Tracking ID preserved |
| 7 | Nested install/upgrade compatibility and identical per-layer MCP/function coverage (no cross-layer controller) | L0/L1/L2 full P0 loop/matrix, L3 key paths if supported, own dependency versions, no parent/sibling modification |
| 8 | Adversarial Finder/Fixer/independent Verifier reacceptance, exact installed package and Go source-bound CI auto-publication | Same original bug replay against exact built bytes; complete per-tool pass/fail coverage and safe release provenance; CI source release never upgrades hosts |

## Gate rules

- **A slice is not complete until its real acceptance criteria pass.** Design review, mock/CI tests and source Release do not prove installed native behavior.
- P0 provides a genuinely usable development workflow before building optional process supervision, sleep/wake, search, OAuth or snapshot features. Do not conflate a running Box with a ready guest.
- Every mutating tool, including Exec and guest file operations, uses durable async admission. Reads still carry correlation context. No tool supports arbitrary host shell/files or unmanaged Caddy upstreams.
- Version-aware local dependency reconciliation is part of install/upgrade; upgrading a shared pre-existing Incus daemon is a separately scoped privileged action, not an automatic side effect of a generic install.
- The `v1` branch starts at design-only `1.0.0`. No CI workflow or real release exists yet. After an actual tested build and new/versioned candidate, the standing GitHub CI publish policy applies, independent of host installation.

Relevant contracts: [development loop](../product/DEVELOPMENT_LOOP.md), [tool catalog](../design/MCP_TOOLS.md), [Tracking/context](../design/TRACKING_CONTEXT.md), [dependency lifecycle](DEPENDENCY_LIFECYCLE.md), [security](../design/SECURITY.md), [acceptance](../verification/ACCEPTANCE.md).
