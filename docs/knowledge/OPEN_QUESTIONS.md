# Questions and proposal inbox

This file owns unresolved questions/proposals only. Current risk IDs/status belong exclusively to [OPEN_RISKS](../verification/OPEN_RISKS.md); selected decisions belong to [SCOPE](../product/SCOPE.md) and [ADR index](../design/ADR/INDEX.md). Earlier questionnaires/risk wording are preserved in [OPEN_QUESTIONS_HISTORY](OPEN_QUESTIONS_HISTORY.md) and are not current gates.

No product question remains pending from the completed phase6 decisions. A new request is not authorization to change unrelated scope.

For a new question record: source/date, affected REQ/AC, concrete ambiguity, options/tradeoffs, owner and the decision needed. On resolution add the trusted SRC/DEC/ADR reference, update the owning outcome/design/task/risk and keep the question's disposition. Do not mirror changing risk rows or phase status here.

Legacy review cadence and incremental evaluation belong to [MAINTENANCE](../process/MAINTENANCE.md) and [LEGACY_SYNC](LEGACY_SYNC.md), not an automatic merge or scheduled agent job.
## Q-DIAG-001 — Evidence-first diagnostics and optional existing-MCP read

Source: SRC-104, 2026-10-30. Affected approved contracts: REQ-002/003/005/006/007/009/017/018/020; no new product outcome has been approved. The [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md) and [proposal](../design/DIAGNOSTICS_PROPOSAL.md) recommend (a) correcting false-success diagnoses, (b) a scoped private trace/report mechanism, and only conditionally (c) a read-only tool inside the already deployed MCP gate. Decision needed from owner: approve D0 alone or D0-D2; choose retention/export boundary; determine whether any direct-Agent SSH caller may safely read a persisted trace. A generic project debugger and separate Debug MCP are not approved. Owner: product maintainer. Status: OPEN / research-only.

