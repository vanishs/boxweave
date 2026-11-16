# BoxWeave v1 architecture: dated discussion snapshot

> **Record type:** historical architecture review and open discussion only. **No approved optimization work, design selection, implementation scope, priority, estimate, timeline, milestone, acceptance gate, or merge decision.** This document does not modify S4/S5 obligations or authorize implementation.
>
> **Observed on:** 2026-11-16 (America/Los_Angeles).
> **Protected `v1` baseline:** [`4a45a80c8de2fb36cde851a10e0f9c80f2f9b300`](https://github.com/vanishs/boxweave/commit/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300) (S3 merge, PR #152).
> **Separate, unmerged S4 work observed:** [Draft PR #153](https://github.com/vanishs/boxweave/pull/153), head [`3407e2a25609e885756f2fe3446394967041d37c`](https://github.com/vanishs/boxweave/commit/3407e2a25609e885756f2fe3446394967041d37c). Its source is **not** part of the `v1` baseline.
> **Source of truth:** the two exact commits above, not the state of their moving branch names when this note is read later.

## What the merged v1 code does at this snapshot

```text
MCP client (optional operator-managed HTTPS tunnel/proxy)
  -> localhost Streamable HTTP /mcp + embedded OAuth/Bearer
  -> official MCP Go SDK + typed tool/auth policy (internal/control)
  -> SQLite admission / ownership / operation journal (internal/state)
  -> in-process goroutine worker + native verification
  -> official Incus Go SDK (internal/native)
  -> restricted per-user Incus Unix socket -> incusd -> owned containers
```

- The MCP SDK and Incus SDK are **libraries linked into one Go controller process**. No intermediate BoxWeave-specific SDK-to-SDK socket service currently exists. The Unix socket in this diagram connects the Incus client to `incusd`. The host-local installation/recovery CLI belongs to the same executable, but is not a public MCP host-admin API. Sources: [CLI](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/cmd/boxweave/main.go), [control](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/control/server.go), [native](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/native/box.go).
- One controller manages its own Incus Project and recorded Box UUID/incarnation identities, with non-root OS privilege and independent native post-read checks. Nested controllers have separate Incus/SQLite/MCP identities; v1 has **no built-in cross-layer orchestration**. Sources: [native status](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/native/status.go), [nesting contract](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/docs/design/NESTING.md).
- SQLite uses WAL, FULL synchronous mode and a single Go database connection. The journal admits a mutation before its native effect, persists original identity and idempotency, and keeps ambiguous effects `UNKNOWN` rather than blind replay. Workers currently run in-process; startup recovery does **not** automatically resume a possibly dispatched native mutation. Sources: [store](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/state/store.go), [worker](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/state/worker.go).
- The merged controller registers **16 public MCP tools**, not the intended final 20. Guest file exclusive create exists; atomic conditional replace/delete is deliberately `UNSUPPORTED`. The four remaining network tools and their native security validation are S4 scope, not deployed v1 functionality. Sources: [tool registration](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/control/server.go), [files](https://github.com/vanishs/boxweave/blob/4a45a80c8de2fb36cde851a10e0f9c80f2f9b300/internal/control/file_tools.go).

## Strengths visible in this source snapshot

- Narrow, typed MCP capabilities rather than exposing host shell or arbitrary Incus management.
- Native Incus reuse, no required resident Guest agent, and relatively small per-layer deployment topology.
- Durable original operation receipts, explicit ownership/UUID verification, replay safety and careful distinction between success, verified failure and uncertainty.
- Separation of unprivileged runtime from privileged, explicitly approved host-local preparation.
- Independent nested controllers without passing parent Incus credentials down a management tree.

These are architectural observations, **not** evidence that every promised isolation, quota, deep-nesting, networking or release case has been accepted.

## Costs, limitations and open discussion areas

These are **questions to preserve**, not decisions or tasks:

1. **Module/process boundary:** The `control/state/native/install` packages are identifiable, but some MCP Tool handlers directly coordinate SQLite, worker closures and Incus verification. Would a different internal boundary be useful if new consumers ever appear? A separate core engine and Unix-socket protocol is only a hypothetical option; it introduces extra API/versioning, identity and lost-ACK problems.
2. **Transport flexibility:** The running MCP surface is loopback Streamable HTTP with optional external proxy. Is there a future reason to support stdio or other local clients, and what would remain protocol-agnostic? No alternate transport is committed here.
3. **Recovery vs safety:** In-process goroutines plus conservative startup recovery preserve no-blind-replay, at the cost of operations becoming `UNKNOWN` and requiring manual review. Could more read-only reconciliation or diagnosis eventually help without resending uncertain effects?
4. **SQLite concurrency:** Single connection, WAL/FULL and serialized durable operations simplify reasoning but may limit concurrency. No workload target or database migration is selected; performance claims require measurements.
5. **Security and networking:** Runtime restricted Project/UUID checks are strong application controls, yet network ACL editing, Scriptlet daemon-global authorization, AppArmor/BPF/UID maps, quotas and nested kernel enforcement need separate, exact-host proofs. Do not conflate safe refusals with support for all advertised features.
6. **Recursive autonomy:** Independent L0/L1/L2 state is simple and confines administrative authority, but there is no built-in descendant lifecycle discovery, subtree deletion coordination, global resource accounting or cross-layer authorization. Whether any of these should exist remains undecided.
7. **Operations and maintainability:** Native preflight/installer migrations, rollback, local recovery and testing create substantial surface area. How much should remain in one binary or be factored into reusable internal services is an open question; separation is not inherently an improvement.
8. **Source/document drift:** Architecture and Tool catalog documents may have dated phase-status headers that do not match the latest merged code. Future review must distinguish the **versioned source snapshot** from historical design language.

## Already-active changes are not proposals in this discussion

At the observation timestamp, S4-A in PR #153 was working on **OAuth grant-scoped stable actor identity**, plus **atomic Create UUID/terminal and Delete tombstone/terminal** and read-only native evidence instrumentation. The merged `v1` still scopes admission actor to token identity and has separate Create/Delete finalization steps. **PR #153 is Draft, unmerged and marked partial**; do not count those changes as baseline PASS or duplicate/redirect its work through this discussion PR.

S4's network Tools, supported native security integration, and S5's final release campaign remain governed by their own documents and gates. This record must not alter scope or block either stage.

## How future agents should interpret this record

- First inspect the **then-current** `v1` head, relevant open/merged PRs, `AGENTS.md`, active phase documents and the precise commits pinned above. Compare code and dates before repeating any observation.
- Treat each note as **dated, potentially stale evidence**, not a live specification, technical mandate or acceptance signoff. Mark later discussion with a **new date and exact commit**, and explicitly state what changed or became obsolete. Preserve this baseline rather than silently rewriting history.
- No task is approved merely because an issue is described here. **How to optimize, how much to optimize and when (if ever) to optimize are deliberately undecided.** Implementation, merge and release require separate owner decisions and their normal quality/safety gates.
