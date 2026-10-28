# Open questions and risks

No item below is silently resolved by the phase 1 document commit. Sources: [register](SOURCE_REGISTER.md). Priority P0 = product-scope decision before design; P1 = acceptance/safety concern before relevant implementation/release; P2 = later operational/maintenance evidence.

| ID | Priority/status | Unknown or risk and impact | Source | How to resolve / owner and gate |
| --- | --- | --- | --- | --- |
| RISK-001 | P0 / open | Primary v2 user/outcome: personal agent entry, console, multi-user orchestration, or recursive infrastructure? Scope changes requirements materially | SRC-001/014/016 | Developer decides in phase 2; record REQ/AC and explicit non-goals |
| RISK-002 | P0 / open | Which legacy behaviors to keep/change/remove, and whether Web remains mandatory | F-001..015 | Phase 2 review by developer; no automatic feature deletion |
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

## Phase 2 disposition, 2026-10-28

No RISK is closed by drafting a proposal. RISK-001/002 now have concrete alternatives DEC-001/003/004 in [scope](../product/SCOPE.md). RISK-003 maps to proposed REQ-016/AC-016; RISK-004 to REQ-001/AC-001; RISK-005 to DEC-002 and REQ-017/AC-017; RISK-006/007 to DEC-004/005 and REQ-012/018; RISK-008 to deferred recursive/confinement claims under DEC-004; RISK-009 to REQ-008/AC-008; RISK-010 to REQ-009/017; RISK-011 remains an evidence-preservation issue; RISK-012 to REQ-005/006/007; RISK-013 remains proposed out of initial scope. These references add decision/acceptance paths, not implementation evidence.

The actual client launch matrix, personal-versus-team audience, console timing, advanced features and demand for project binding remain unknown. DEC-001..005 are the authority for the five developer choices; do not duplicate their changing answers here. No market/user-demand claim is invented from source code. Product requirements remain proposed until the developer accepts/revises the scope and key behaviors.

Knowledge gaps do not prevent document handoff. They prevent unsupported runtime/support claims and automatic phase advancement. Any later live experiment must have target resources, changes, deadline, success criteria and cleanup agreed first.
