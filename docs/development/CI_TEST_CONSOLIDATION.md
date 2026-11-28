# CI and test consolidation

## Objective

Reduce the maintenance burden of historical phase workflows and test scripts on the current Schema 7 v1 branch without reducing applicable behavioral coverage or independent release evidence. Prefer fewer stable, product-behavior-oriented entry points and shared fixtures over PR-number-specific workflows. This task does not change product behavior, branch protection, release policy or installed hosts.

## R0 baseline (2026-11-28)

Exact source: `c0ad27a11fd1644e7e9755900c2c9b5421f84c9f` (RC15). Counts from the complete recursive Git tree (not test execution counts):

| Inventory | Count |
| --- | ---: |
| Workflow YAML files under `.github/workflows/` | 46 |
| Tracked files under `scripts/` | 213 |
| Script-directory files named `test_*` or `*_test.go` | 54 |
| Repository-wide `*_test.go` files | 296 |

The [P4 disposition](P4_WORKFLOW_DISPOSITION.md) already identifies historical and optional workflow routes; reconcile its dated baseline with the current tree. The former stable candidate workflow is not present in this tree; investigate its current replacement rather than assuming a missing gate or restoring a retired route. File counts are not measures of unique coverage, CI minutes or maintenance savings.

The current core candidates for preservation are quality-gate, S1, Forward policy, N2, S2, Scoped S2.5, RC release and the existing stable release workflow. Review A071 and UNKNOWN Create separately. Historical branch names and markers remain useful provenance, not automatic retirement decisions. `make verify`, source-specific probes, nonempty Python discovery, native consumer dependencies and release manifest checks require a caller inventory.

## Delivery sequence

| Round | Work | Exit evidence |
| --- | --- | --- |
| R0 | Complete current producer/consumer, workflow trigger, test assertion and helper inventory. Measure recent comparable CI jobs, elapsed runner minutes, duplication and LOC. | Exact baseline, classification and prioritized small changes; unknowns explicit. |
| R1 | Remove from active v1 only proven historical-only workflows with no live consumers; retain immutable Git locations and old run links. | Per-file disposition, required-check and trigger review, negative route checks. |
| R2 | Consolidate duplicate setup, fixture, build and invocation code into existing helpers. Replace phase/PR-specific entry names with behavior-oriented names only where useful. | Focused regressions, unchanged assertions, current callers and marker provenance mapped. |
| R3 | Reconcile obsolete Legacy-only positive cases and duplicated tests with current Scoped contract. | Each removed case mapped to equivalent or stronger passing witness or explicit obsolete requirement; never rely on name similarity. |
| R4 | Simplify CI orchestration and redundant source checks where equivalence is measured. Preserve independent runners and exact source/asset identity. | Full quality, affected native suites, release workflow contract and negative trigger/provenance tests. |
| R5 | Independent final review and before/after comparison. | Actual file/LOC, job and runner-minute deltas; applicable coverage, exact CI results, residual gaps and rollback plan. |

## Guardrails and scope

- Keep required `quality-gate` identity and fail-closed docs/mixed/release classification; preserve applicable full source, CGO/SQLite, race, dual-architecture build, installed native, S2/S2.5, manifest and stable publication gates. Do not count skipped tests or historical PASS as current proof.
- Bind each removed producer or assertion to its exact baseline path, consumer/acceptance ID, disposition (retained, equivalent replacement, historical-only, obsolete contract, or blocked), current witness and result. Preserve the original evidence in Git rather than copying large historical script trees into an active archive directory. Check dynamic references, workflow_dispatch, Makefile, hooks, embedded script generation, and release crosswalks before removal.
- No blanket deletion based on prefix, age or count. No coverage waiver, test oracle weakening, skipped-test substitution, reimplementation of the harness, or artificial percentage target. Consolidate only where measured maintenance or execution benefits justify the change. Maintain a small active test surface and clear scenario ownership.
- Preserve RC15 and v1 branch state; work in this PR. No host mutation, release, merge or ruleset change as part of the initial inventory. The unrelated RC14 purge receipt size defect is outside this task.

Initial status: inventory baseline recorded, detailed classifications and implementation pending. No workflow or test has been changed; no CI efficiency gain or coverage equivalence is claimed yet.
