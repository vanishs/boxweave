# Phase 7 — simplification and institutionalized maintenance

Status: complete within the authorized maintenance scope (SRC-089..091). Baseline cb16e6d32f6443070a141da7a67e884ea1267fb8, rewrite/v2. Runtime remains 0.9.1: no bwmini, web/src, version or dependency change. No deployment, push, public route, inference or legacy mutation.

## Entry and review findings

Phase6 closed the three selected P1 gaps; [OPEN_RISKS](OPEN_RISKS.md) retains all six bounded P2 entries. Legacy HEAD was freshly read as d8a09dd180139159e035963502e7793a37e92bae; its untracked archive was left unread/unmodified. [instructions](../process/PHASE_07_INSTRUCTIONS.md) preserve the user's phase7 file byte-for-byte.

Review covered service/lifecycle/access state flow, Store transactions, MCP/provider identity boundaries, Web/CLI adapters, pure profile/model/Graph generation, distribution and existing scoped lab drivers. The service/provider distinction is useful: intent and dispatch belong to services, authenticated identity and upstream format checks to adapters. Similar checks across these boundaries are not redundant validation. No measured performance hotspot justified caching trust, merging subsystems or replacing the stack. Large live drivers carry resource-specific acceptance evidence and remain excluded from routine checks rather than deleted.

| Demonstrated issue | Change and expected benefit | Risk / regression |
| --- | --- | --- |
| Six current documents accumulated conflicting historical gates and duplicated owners | Short current owners, labeled same-directory original snapshots and 16-role authority map | Losing historical rationale/links: original SHA/payload preservation and all local links checked |
| Architecture still described proposed REST authority; ADR005 pending gate looked current | Current MCP implementation map, ADR status/reconsideration index, explicitly historical checkpoint | Documentation drift: inspect actual provider/services, preserve old design and matrix |
| Binding browser regression required ignored personal phase6 metadata | Committed nonexecuting four-profile fixture; optional explicit public-input path only for scoped acceptance | Accidentally claiming real enrollment: reserved fixture endpoint, no execution and evidence labels; actual browser passes without old runtime |
| A fixed artifact filename overwrote earlier browser proof | Unique per-run artifact directory | Artifact loss: existing archive hash identical before/after; newly generated files separate |
| Maintainer needed chat to assemble routine checks | Bounded non-root fixture-only runner with actual exits/private logs plus maintenance/onboarding procedure | Accidental live driver execution: explicit seven-script allowlist, no dependency repair/enrollment/MCP/deployment; 10-step actual run |

Six primary documents totalled 149,013 bytes before consolidation and approximately 17.3 KB after it (before final completion wording). This measures current reading surface, not total repository size or runtime speed. All six original payloads are retained verbatim after a historical banner; [archive manifest](phase7-archive-manifest.json) records original digests. Historical source/evidence hashes were not rewritten to match the new harness.

## Actual verification

[Machine-readable execution evidence](phase7-runtime-evidence.json) records environment, exact source/log digests, real exit codes and limitations.

- Non-root boxweave-tenant-dev identity verified with whoami/hostname, exit0. Python3.12.3; existing locked environment and Node18; no installation.
- .venv/bin/python scripts/validate.py --browser: exit0, all10 steps passed. 168 tests in64.600s (163 existing plus5 knowledge regressions), frontend build and seven actual Chromium fixture workflows. Individual step logs/digests and timings are in the evidence file.
- Knowledge audit:69 Markdown files,16 authority roles,18 REQ and18 AC covered; no broken local links or missing eight cold-start routes.
- First run stopped at missing two historical screenshot assets in the remote copy. Restored the public assets, then ran the complete lane afresh; failed report is retained. No assertion or expectation was weakened.
- Existing phase6 browser archive SHA unchanged before/after. No ignored operator metadata read by default profile generation.
- Public source files in bwmini/web/src and version/locks compared against baseline: unchanged. Product code is not represented as freshly live-tested by this maintenance run.

## Cold-start rehearsal

This was a fresh-process scripted rehearsal plus manual document review by the current maintainer, **not** an independent new Agent/human review. A new public source copy excluded .runtime/.git and private identities; dependencies were explicitly reused by symlink, so it is not clean-host installation evidence. From AGENTS + INDEX, all eight questions were reachable within three links:

| Question | Located answer / authority |
| --- | --- |
| Goal | PRODUCT: personal trusted-operator Agent remote development, project/environment binding |
| Boundary | SCOPE: native/explicit WSL SSH, Codex/ZCode, basic Web/Tenant/ports/Graph/model handoff; deferred features remain explicit |
| Run | INSTALL plus FINAL_HANDOFF: prepared environment, fixture checks, separately scoped configured serve/worker |
| Design | ARCHITECTURE and ADR index: service state, MCP adapter, independent trust and unknown outcomes |
| Tests | TEST_STRATEGY/FEATURE_MATRIX and MAINTENANCE: fixture lane versus separately authorized live evidence |
| Risk | OPEN_RISKS: exact remaining P2 limitations and required future evidence |
| Legacy | LEGACY_SYNC: reviewed SHA, read-only manual phase/affected-change/weekly-active review without merge |
| State | PHASE_STATUS: current phase authorization/completion, no inferred rollout |

The fresh copy actually started the app and ran binding_browser_check.py: exit0, Chromium141.0.7390.37, retained identity/edit/unbind/rebind/conflict checks, four profiles/eight downloads, no JavaScript errors, zero Provider execution. No previous .runtime was present. Findings corrected before completion: add exact worktree/resource/run entry points, explicit dependency reuse limits, canonical risk/status owners, portable fixture inputs and missing historical image synchronization.

## Continuing maintenance and remaining bounds

Follow [MAINTENANCE](../process/MAINTENANCE.md): source → REQ/AC → decision/task → implementation/checks → owning matrix/source/risk → authorized commit/handoff. [FINAL_HANDOFF](../process/FINAL_HANDOFF.md) is the next maintainer's onboarding entry; [INDEX](../INDEX.md) is navigation only. Every selected REQ/AC retains its implementation/test/evidence row.

RISK-604..609 remain in their sole owner OPEN_RISKS: legacy readiness harness collection, retained primary app0.9.0/rollout and schema boundary, incomplete live/platform equivalence, unverified boot/confinement/public TLS/inference, operator/source/upstream authority trust, and bounded unknown/recovery/convergence. Phase7 does not close them. No fresh Windows enrollment/GUI, live network packet, reboot or model-inference test was necessary for unchanged runtime; prior evidence remains explicitly prior.

No automatic weekly job was created. No old branch/resource was removed. Future changes start from the new procedure; the old project may continue independent maintenance.
