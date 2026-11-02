# Phase 1 — Unified operation base: development and acceptance ledger

[Master roadmap](../design/FEATURE_ROADMAP_MASTER.md) ·
[R-02](../design/MCP_TASK_RELIABILITY_R1.md) ·
[R-04](../design/MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) ·
[Agent test rules](AGENT_TEST_WORKFLOW.md)

**Owner authorization:** Implement Phase 1 in independently tested and committed slices.
**Source baseline:** main 1480b0671cab543d4664b57b0079f1d8962981fc (v0.28.0). Phase 1 source bumped to v0.29.0 after merged P1-A/P1-B1; installed stable pin remains v0.28.0.
**Scope:** Shared internal operation claims, then adapter migration and compatibility.
**Not authorized by this document:** production installation, Provider changes,
public exposure, upgrades or unreviewed resource destruction.

## Slice P1-A — Minimal shared resource-claim contract

- One SQLite coordination namespace for canonical exact Box incarnation, logical
  name, Tenant and published protocol/port resources. This is NOT the application
  SQLite, a guest process manager, or a second task-result database.
- Typed claim acquisition is atomic and fail-closed. A unique operation ID
  correlates the original receipt; commands, source, keys and results are not
  stored in the coordinator. All remote calls occur OUTSIDE short DB transactions.
- A durable dispatch intent MUST precede any remote write. Loss of the caller
  changes intent to UNKNOWN and retains the resource claim. Neither crash,
  lease timeout nor disconnect permits replay. Only observed original receipt
  evidence can retire a dispatched claim. Pre-dispatch cancellation is separate.
- The core has no active production caller in this slice. It does not change
  the existing single-operator authority, existing app receipts or MCP behavior.
  There is no claim of SSH/HTTP/Web parity or on-host permission readiness yet.

**Executed fixture evidence (2026-11-01 local):** GitHub Actions run 41402675634 at commit `05a95aeabd633896d11aba05079a3acbaf6baba9` ran the full verify workflow successfully (including Python unittest discovery, documentation inventory, fixture browser and packaging checks). No test used a live Provider or installed coordinator. The subsequent evidence-only commit must receive independent CI.

**Tests:** tests/test_operation_coordination.py — exact typed keys, duplicate
dispatch intent, UNKNOWN persistence, receipt mismatch, multi-resource
atomicity, independent Boxes, concurrent SQLite connections, unavailable
storage and invalid data. CI result must be recorded against the exact SHA.

## Interstitial gate ARCH-0 — architecture consolidation BEFORE P1-B2

[Architecture source audit and proposed slices](../design/ARCHITECTURE_CONSOLIDATION_GATE.md).
Owner added this gate after P1-A and P1-B1, before broad Web/SSH/HTTP
mutation integration. Source audit found standalone Gateway packaging, Box
identity scope and shared-state ownership as primary integration blockers.
Six logical code blocks are recommended, **not** a repository-wide
mass-move or new framework. ARCH-0A maps dependency and mutation entrypoints;
ARCH-0B settles runtime packaging/storage/identity contracts; ARCH-0C
extracts only verified pure interfaces behind unchanged public entrypoints.
**Status: designed / source reviewed (S), no refactor implementation or
real-host acceptance (T/L=U).** Existing P1-B2 and P1-C remain pending.

## Slice P1-B — Security/provisioning and adapter contract (IN PROGRESS)

**P1-B1 candidate:** `coordination_runtime.py` adds canonical Box incarnation
fingerprints for separately verified Provider/app/gateway views and an opt-in
private SQLite opener. Refuses missing/unowned/shared-mode/symlink DBs and
unsafe parents; does not create or authorize the DB. Ten negative/positive
fixture tests are in `test_coordination_runtime.py`. Source/fixture acceptance at `8ee12fd` passed [CI 41403521130](https://github.com/vanishs/boxweave/actions/runs/41403521130). The installer still does not provision a shared DB,
and no app/gateway mutator uses the coordinator yet.


1. Review authenticated host/group ownership of the coordination directory,
   file and connection factory; reject unsafe symlink, owner, mode, downgrade
   and schema mismatch. No gateway access to app-wide SQLite or private outputs.
2. Connect actual Web/app and SSH/HTTP gateway paths using the same resource
   claim authority, and preserve each original receipt and its real result.
3. Complete a strict mutation-route inventory. Unintegrated forwarded mutators
   must explicitly fail closed or be advertised as not protected.
4. Cover migration, restart, mixed old/new version and rollback with separate
   fixtures and approved installed-Box checkpoints.

## Slice P1-C — Backwards-compatible migration and acceptance (PENDING)

- Preserve legacy connect(exec) and stable MCP tool names. New tracked
  behavior is opt-in and must never silently claim older calls are deduplicated.
- Audit Provider, SSH, remote HTTP, Web and app operation paths, including
  nested-create side effects and namespace/port cross-resource mutations.
- Run INT-T01..INT-T06 and exact installed-client tests; record SOURCE,
  fixture, CI and LIVE evidence independently in verification docs.
- Do not mark Phase 1 accepted until all protected mutation routes have
  shared guards, no bypass, fail-closed degradation and verified rollback.

## Release gate correction

- Both feature PRs passed full verification CI, but the `main` push publish job correctly refused to reuse already immutable `v0.28.0` for changed runnable source. Affected push runs: [P1-A 41403253971](https://github.com/vanishs/boxweave/actions/runs/41403253971) and [P1-B1 41404041307](https://github.com/vanishs/boxweave/actions/runs/41404041307).
- New source uses `0.29.0` consistently in VERSION, Python metadata, Web metadata, and MCP self-description; README distinguishes stable 0.28.0 from new source. Do **not** change `PUBLISHED_RELEASE` or default installer pin before independent release/promotion validation.
- **Source publication PASS:** [main-push CI 41404753003](https://github.com/vanishs/boxweave/actions/runs/41404753003), exact merged SHA `160a09c4d2d2defff3adf3fe538861e28128b1a3`, verify and publish succeeded. [Immutable release v0.29.0](https://github.com/vanishs/boxweave/releases/tag/v0.29.0) targets the same SHA and includes the SHA256-digested application archive. Source tag/CI does not prove any installed service or external client.
- **Installed acceptance U:** no host upgrade, shared coordinator provision, effective gateway/app integration, authenticated SSH/HTTP recovery or multi-Agent race test has been executed for v0.29.0.

## Current acceptance

| Slice | Source | Focused tests | CI | Installed/live | Decision |
| --- | --- | --- | --- | --- | --- |
| P1-A | Implemented, internal-only | 9 unit cases in full CI | [Run 41402675634](https://github.com/vanishs/boxweave/actions/runs/41402675634), commit `05a95aea`, verify succeeded | No installed caller or Provider action | **PASS for source/fixture only** |
| P1-B | Identity/storage adapter source implemented; migration pending | 10 focused new cases in full fixture suite | [CI 41403521130](https://github.com/vanishs/boxweave/actions/runs/41403521130) at `8ee12fd` PASS | No host provisioning or mutation integration | **PARTIAL (source PASS, live U)** |
| P1-C | Not implemented | Not run | Not run | Not run | Pending |

**No production behavior changes in P1-A/P1-B1.** Source version is now 0.29.0; `PUBLISHED_RELEASE` and default stable installer remain 0.28.0. Evidence is reported only after actual execution.
