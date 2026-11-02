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

## Post-report main CI repair (maintenance)

After docs-only PR #39 merged at `8e89f025`, [main CI 41422944355](https://github.com/vanishs/boxweave/actions/runs/41422944355) failed in an unrelated pre-existing random selftest fixture: `tests/test_selftest.py` mutated a SHA-256 digest using `checksum.replace("a","b")`, which changes nothing for digests containing no `a`. The only failure was `test_load_plan_accepts_only_confirmed_selftest_plans`; the release job was skipped. Maintenance v0.34.0 replaces this nondeterministic test input with a guaranteed one-digit mutation; business/runtime behavior is unchanged. **New exact-SHA CI, immutable tag and installed validation must be checked separately.** Phase 2 overall PARTIAL remains unchanged.

**Maintenance closure confirmed:** [PR #40](https://github.com/vanishs/boxweave/pull/40) merged as `4f51f44682677fc1e5e982af41ed1a600484d88f`; [PR CI 41425906154](https://github.com/vanishs/boxweave/actions/runs/41425906154) and [main verify/publish CI 41426251396](https://github.com/vanishs/boxweave/actions/runs/41426251396) both PASS. [Immutable v0.34.0 Release](https://github.com/vanishs/boxweave/releases/tag/v0.34.0) targets exactly the merged commit. This fixes only nondeterministic selftest-fixture checksum corruption and version provenance; no runtime validator change, host upgrade, live Box acceptance, or completion of P1-B2/R-04 global dispatch is implied. Stable PUBLISHED_RELEASE remains v0.28.0.

## v0.40.0 lifecycle worker integration — source checkpoint

[PR #47](https://github.com/vanishs/boxweave/pull/47) adds opt-in app lifecycle create/start/stop/delete/resize claim enforcement with the existing original app receipt. Pre-version-bump code [CI 41493238649](https://github.com/vanishs/boxweave/actions/runs/41493238649) passed fixture Python, packaging, JS and browser checks; final source version and publication require separate exact-SHA CI after this metadata commit. No live Provider, real Web/SSH/HTTP cross-transport or deployed-host acceptance is inferred. Existing Access/tenant/port actions, explicit MCP lease tool, global bypass and BOI patch remain pending or unverified; source state is **PARTIAL**, installed state **U**, stable installer v0.28.0.


## v0.41.0 completion candidate / final evidence gate

[Implementation details](../design/PHASE2_CROSS_ENTRY_COMPLETION.md).
Source now integrates Access Tenant/port claims, a protected-only optional
Box lease MCP method, and bounded tracked CAS patch, reusing the original
R-02 metadata-only task receipt. Verify final exact SHA with Linux focused
tests, complete validation, built archive and source GitHub CI before claiming
source PASS. A PASS here does **not** satisfy installed cross-entry, independent
SSH/HTTP client, timeout-after-real-dispatch, legacy upgrade/rollback or
out-of-band mutation acceptance. Overall Phase 2 is **PARTIAL / LIVE U**
until installed host gates are proved.

## v0.41.0 actual disposable development Box execution evidence

Target: existing authorized `bwphase2-1102` (not a fresh installation),
Linux Ubuntu and Python 3.12.3, isolated source worktree at
`/home/bwphase2-1102/phase2-mainwork` based on `780cdf3` plus
uncommitted candidate changes. The original pre-existing source directory
and unrelated Box workloads were preserved.

- `python scripts/check_knowledge.py`: exit 0, 160 documents.
- `python -m unittest discover -s tests -q` using previously prepared
  locked virtual environment: exit 0, **630 tests passed**.
- Local `scripts/validate.py`: exit 2, missing Node/npm in this Box.
  This is **BLOCKED**, not a passing frontend or browser verification.
  Verify full release-package/frontend/browser CI on exact GitHub SHA.
- Actual guest `box_patch.guest_command` executed inside the Box
  against `/home/bwphase2-1102/phase2-cas-live-proof.txt`:
  initial file `revision A\\n`, SHA-pinned replacement
  `revision B\\n` exit 0 with SHA-256
  `38ff9cf1075f2b5e9c11bbb9a33ad4760991d9c2204113433b3a51c7511c2ee3`;
  readback matched; repeating with stale original SHA returned exit 1
  and `BOX_PATCH_STALE`, file remained unchanged. Test-owned proof
  file deleted after verification. This is **L (guest script only)**,
  not a proof of installed `boxweave_box_patch` SSH/HTTP MCP routing.
- Target running host and independent Web/Gateway shared DB root policy
  were not modified. Real dual-transport upgraded-host acceptance remains U.

Commit, GitHub CI run and immutable tag must be appended to the next
acceptance checkpoint; historical version-specific evidence is never silently
promoted to an installed production PASS.
