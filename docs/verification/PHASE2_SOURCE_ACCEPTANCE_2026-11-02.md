# Phase 2 incremental source delivery — acceptance and outstanding gates (2026-11-02)

**Decision: PARTIAL / NOT FULLY ACCEPTED.** R-01 read-only, R-02 metadata-only recovery, R-03 first-use guide, and R-04 internal lease primitive are source/fixture accepted. Cross-entry mutation guard, guarded patch, optional Agent lease tool, mixed-version host migration and live SSH/HTTP/Web acceptance are UNIMPLEMENTED or UNVERIFIED.

## Exact source acceptance evidence

| Slice | Source PR | Fixture + source release | Scope |
| --- | --- | --- | --- |
| R-03 | [PR #36](https://github.com/vanishs/boxweave/pull/36) | [CI 41417034821](https://github.com/vanishs/boxweave/actions/runs/41417034821), [v0.31.0](https://github.com/vanishs/boxweave/releases/tag/v0.31.0), main CI 41417309331 PASS | Read-only host setup guide; no external OAuth or MCP result claim |
| R-01 BOI-1 | [PR #36](https://github.com/vanishs/boxweave/pull/36) | Same v0.31.0 source release | Bounded symlink-denying file and directory reads via original trusted connect; no writes or guest agent |
| R-02 MTR-1/2 source | [PR #37](https://github.com/vanishs/boxweave/pull/37) | [CI 41417633459](https://github.com/vanishs/boxweave/actions/runs/41417633459), [CI 41418231631](https://github.com/vanishs/boxweave/actions/runs/41418231631), [v0.32.0](https://github.com/vanishs/boxweave/releases/tag/v0.32.0), main CI 41419407778 PASS | Private task DB, request-key dedupe, UNKNOWN-after-loss, observed exit code; original command/output not persisted; explicit provision |
| R-04 internal lease core | [PR #38](https://github.com/vanishs/boxweave/pull/38) | [CI 41420072940](https://github.com/vanishs/boxweave/actions/runs/41420072940), [CI 41421063973](https://github.com/vanishs/boxweave/actions/runs/41421063973), [CI 41421729358](https://github.com/vanishs/boxweave/actions/runs/41421729358), [main 41422076227](https://github.com/vanishs/boxweave/actions/runs/41422076227) PASS, [v0.33.0](https://github.com/vanishs/boxweave/releases/tag/v0.33.0) | Optional hashed-token lease method, 600s renewable expiry, safe unknown claims, private shared coordinator DB **provision only**; no runtime dispatch enforcement |

## Unimplemented safety-critical Phase 2 gates

1. **P1-B2/INT-01:** an authoritative dispatch-intent claim must cover the actual Web/app operations, SSH direct MCP, isolated HTTP MCP and old Provider mutators. A separate root-owned activation/effective-UID contract, canonical Provider scope and fail-closed handling of unclassified forwarded writes are needed. A subset does not qualify as global write protection.
2. **R-04 exposure/lease:** no live box_lease gateway endpoint is advertised. It would be misleading and unsafe before all selected mutation entrypoints check the same claim/lease authority. Lease expiry never proves dispatched guest writes have ended.
3. **R-01 BOI-2/3:** conflict-aware safe file patches, stale-hash refusal, exact project-root checks and live byte-level SSH/HTTP guest reads remain pending.
4. **R-02 MTR-3/live:** manually run sudo boxweave task provision on an approved host, confirm 0700/0600 storage and same user across SSH/HTTP, validate metadata recovery with physical disconnect, OAuth, and independent task_get. No direct Web/app task receipt access is authorized.
5. **P1-C/live migration:** old/new root-helper compatibility, SSH/HTTP reconnect, two-Agent same-Box and independent-Box races, Web receipts and crash/rollback must be observed on an approved disposable Box and source-matching host. CI is not runtime evidence.

## Explicit release and deployment boundary

v0.31.0, v0.32.0 and v0.33.0 are immutable **source** Releases; their CI runs do not establish that the user's host runs those versions. PUBLISHED_RELEASE and the default stable installer still point at v0.28.0. There was no host upgrade, Provider mutation or live Box action in this staged implementation. The source-only R-04 lease implementation reports coordination_enabled=false from the opt-in provisioner.

**Next safe implementation order:** P1-B2 exact mutation/receipt coverage with original business receipts, then optional R-04 MCP lease tool, then guarded BOI patch, followed by cross-transport live negative and recovery tests. Do not mark the overall second stage complete until these gates pass. See [master roadmap](../design/FEATURE_ROADMAP_MASTER.md), [Phase 1 handoff](../process/PHASE1_UNIFIED_OPERATION_BASE.md), and [R-04 design](../design/MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md).
