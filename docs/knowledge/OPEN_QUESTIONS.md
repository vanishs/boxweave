# Open questions and risks

No item below is silently resolved by the phase 1 document commit. Sources: [register](SOURCE_REGISTER.md). Priority P0 = product-scope decision before design; P1 = acceptance/safety concern before relevant implementation/release; P2 = later operational/maintenance evidence.

| ID | Priority/status | Unknown or risk and impact | Source | How to resolve / owner and gate |
| --- | --- | --- | --- | --- |
| RISK-001 | resolved scope / SRC-026 | Primary v2 user/outcome: personal agent entry, console, multi-user orchestration, or recursive infrastructure? Scope changes requirements materially | SRC-001/014/016 | Resolved by SRC-026; personal remote work with binding; see current SCOPE |
| RISK-002 | resolved scope / SRC-026 | Which legacy behaviors to keep/change/remove, and whether Web remains mandatory | F-001..015 | Resolved by SRC-026; basic requested features included, recursive deployment deferred; see SCOPE |
| RISK-003 | P1 / open | Asset download trust has contradictory legacy prose; pinning commits is not independent asset verification | SRC-002/003/004 | Trace generator/installer trust chain; choose and approve security acceptance in product/design phases |
| RISK-004 | P1 / inferred, not reproduced | Dirty client replacement answer is discarded; users may believe replacement selected when generation ignores it | SRC-020 | Separately authorized old-project fix or future fixture comparing interactive choice to rendered invocation; do not alter WIP in this phase |
| RISK-005 | P1 / unverified | Fresh Windows native/WSL/macOS enrollment, real agent GUI recognition and instruction loading lack universal acceptance | SRC-004/011/014 | Explicit per-runtime/client test matrix in later phases; no fixture-only compatibility claim |
| RISK-006 | P1 / unverified | Current installed provider capability/version/credential scope and actual Tenant TCP isolation | SRC-007/010/018/019/021 | Separately reviewed target/identity/readonly capability probe, then explicitly scoped policy/packet tests; not prerequisite for this static phase |
| RISK-007 | P1 / open | Broad admin provider credentials and local console authorization may not suit shared use | SRC-008/010 | Developer defines trust model; real positive/negative authorization tests before shared deployment |
| RISK-008 | P1 / unverified | Effective nested device filtering, parent policy and mount propagation are not certified | SRC-011/016/017 | If nesting/confinement is in scope, design safe evidence gathering; no security relaxation or dangerous-device probes |
| RISK-009 | P1 / open | Update publication is per file and mixed failure can require manual recovery; backup/software compatibility matters | SRC-002/006 | Decide recovery objectives; isolated failure/recovery validation before release |
| RISK-010 | P2 / unverified | Current full suite, CI automation, clean bare-metal install and boot.autostart | SRC-011/013/014/015 | Later assigned-LXC non-root tests, separate clean-platform/reboot checks; no CI assumed |
| RISK-011 | P2 / open | Ignored research is mutable, incomplete and not retained by new Git history | SRC-016..020 | Preserve originals outside commits; use recorded hashes and curated summaries; review additional narrative only if needed |
| RISK-012 | P1 / open | Job/protocol timeout does not prove remote cancellation or no delayed side effects | SRC-009/014 | Specify cancellation/unknown receipts and resource reconciliation acceptance; bounded fixtures and approved live tests |
| RISK-013 | P2 / open | Additional provider interface and standalone Linux client are proposals, not proven old behavior | SRC-001/003 | Phase 2 decide actual need; no speculative abstraction inherited |

## Historical phase 2 draft disposition, superseded by SRC-026

No RISK is closed by drafting a proposal. RISK-001/002 now have concrete alternatives DEC-001/003/004 in [scope](../product/SCOPE.md). RISK-003 maps to proposed REQ-016/AC-016; RISK-004 to REQ-001/AC-001; RISK-005 to DEC-002 and REQ-017/AC-017; RISK-006/007 to DEC-004/005 and REQ-012/018; RISK-008 to deferred recursive/confinement claims under DEC-004; RISK-009 to REQ-008/AC-008; RISK-010 to REQ-009/017; RISK-011 remains an evidence-preservation issue; RISK-012 to REQ-005/006/007; RISK-013 remains proposed out of initial scope. These references add decision/acceptance paths, not implementation evidence.

The actual client launch matrix, personal-versus-team audience, console timing, advanced features and demand for project binding remain unknown. DEC-001..005 are the authority for the five developer choices; do not duplicate their changing answers here. No market/user-demand claim is invented from source code. Product requirements remain proposed until the developer accepts/revises the scope and key behaviors.

Knowledge gaps do not prevent document handoff. They prevent unsupported runtime/support claims and automatic phase advancement. Any later live experiment must have target resources, changes, deadline, success criteria and cleanup agreed first.


## Current phase 3 risk disposition

Developer scope confirmation SRC-026 closes only RISK-001/002 as product choices. Basic Web/Tenant/ports/Graph/model wizard and project binding are included; native Windows/WSL SSH with Codex and ZCode(GLM), single trusted operator. Earlier deferral recommendations are historical, not current scope.

RISK-003: ADR-003 selects trusted descriptor + payload digests; bootstrap/tamper verification still untested (TASK-003/012). RISK-004 remains a legacy-only inferred defect; new adapter must preserve replacement intent. RISK-005 now gates the four selected Windows-hosted Agent profiles; macOS/WSL-hosted Agent CLIs are deferred. RISK-006 remains high for included Tenant writes: TASK-009 must prove actual authenticated capability/atomic behavior; no provider upgrade workaround. RISK-007: shared-team product excluded, but broad upstream privilege still disclosed and tested within trusted-operator boundary. RISK-008 remains unverified; recursive deployment is deferred and no confinement claim is made. RISK-009 maps to staged packaging/recovery TASK-012. RISK-010 maps to TASK-013; no bare-metal/autostart claim without evidence. RISK-011 remains mutable historical evidence. RISK-012 maps to ADR-002 crash/unknown/deadline behavior. RISK-013 is deferred extension scope.

| ID | Priority/status | New design risk | Resolution path |
| --- | --- | --- | --- |
| RISK-014 | P1 / unverified | ZCode GLM model wizard lacks a proven installed-version automation contract; old GLM integration was Claude-specific | SRC-027; ADR-004 and TASK-008: client-owned native settings/sign-in handoff, only automate verified subset; MCP success never means model ready |
| RISK-015 | P1 / unverified | WSL SSH distro/argv/process lifetime can differ from native and false cleanup claims can hide remote work | ADR-003; EXP-002, TASK-007, actual four-profile evidence |
| RISK-016 | P1 / unverified | Single SQLite/worker intent lease cannot make a remote write exactly-once; crash before/after send remains ambiguous | ADR-002; TASK-002/006 fault injection and no replay; provider policy atomicity must be native |

All experiments in TEST_STRATEGY are plans, not executed results or automatic resource authorization. Architecture is complete only as design; release and phase-4 feasibility remain unverified.
