# RACC-SYS-01..09 source implementation checkpoint — v0.48.0 candidate

**Recorded:** 2026-11-05 PT / 2026-11-06 UTC. **Result:** PARTIAL, NOT RELEASE-ELIGIBLE.
**Baseline source:** `49c43917850e87570c0b4fe97d458af10b5df744` (v0.47.0).
**Candidate:** uncommitted isolated developer worktree at this checkpoint; CI and final candidate SHA are independent later gates.
**Isolated test owner:** `bwraccsys-1105-dev`, `bwraccsys-1105-dev-container`, outer createdAt `1793927416`, disposable Ubuntu 24.04, 2 CPU / 4 GB / 35 GB, no host privilege or owner-host reboot.
**Stable installer:** `PUBLISHED_RELEASE=0.43.1`, unchanged. No public tag/release authorized by this checkpoint.

## What actually ran and what it proves

- Original v0.47.0 baseline: `python3 -m unittest tests.test_reacceptance -q` → 35 PASS in the isolated checkout.
- After source-only extraction, a targeted 83-test suite of `test_reacceptance_system`, `test_reacceptance`, `test_distribution` passed; later 86-test targeted set including `test_mcp_info` passed.
- Full regression `.venv/bin/python -m unittest discover -s tests -p 'test_*.py' -q`: first pass 832 cases with **one version metadata failure** (v0.48.0 versus `SERVER_INFO_VERSION=0.47.0`), corrected in this source. Second pass **832 cases PASS**, exit **0**, from `/tmp/bwraccsys-full-python2.log` in the test Box.
- `python3 scripts/check_knowledge.py`: PASS, 230 documents checked at the initial candidate checkpoint; rerun after final docs edits.
- `python3 -m tools.reacceptance.cli inventory --root .` and `plan`: 19 real seed scenario IDs, risk-ordered; missing dynamic actual MCP calls are **BLOCKED**, not PASS.
- `python3 -m tools.reacceptance.cli breadth-safe --root .`: 167 source-derived named rows, all **BLOCKED** because no installed inner MCP/CLI credentials or runtime were provided. These are **not** 167 black-box tests.
- `python3 -m tools.reacceptance.upstream observe`: daemon/CLI/MCP binaries **UNAVAILABLE** in this plain developer Box. Mocked split daemon v0.131.0 / CLI v0.130.0 tags were pinned independently and offline/tampered cache/invalid path were checked in unit tests. No real live upstream source-to-installed route parity has been established.
- The release gate exercised temporary **fixture** verifier signing keys and an artificial candidate ZIP; it rejects changed bytes, forged attestation, same actor/Box, missing scenario/oracle, clean-install unproven, wrong source and missing candidate. This **does not** establish any real independent installed Verifier acceptance or GitHub publish success.

## Work item / verified scope / outstanding done gates

| Work item | State | Implemented, repeatable evidence | What is explicitly NOT done |
| --- | --- | --- | --- |
| RACC-SYS-01 | **PARTIAL** | `tools/reacceptance/{engine,live,proof,cli}` extracted; legacy source-only shim; packaging includes every normal `bwmini` module but excludes dev modules, registries and test data; fixture regression and CI ZIP member checks | Actual v0.48.0 built asset, clean install, legacy-to-candidate upgrade, running Web/MCP/CLI service proof and independent package inspection still required |
| RACC-SYS-02 | **PARTIAL** | Runtime component-by-component version observation, strict matching v-tag to peeled immutable commit, allowlisted content fetch with Git blob hash, offline cache and drift tests | Real installed daemon/CLI/MCP/REST versions, online observed-source sample, source-to-runtime handler parity |
| RACC-SYS-03 | **PARTIAL** | 19 named seeded scenario/invariant risk plans; changed-file relevance and graph preconditions for selected families | All family graphs with actual schema binding, automatic changed-path/issue discovery and successful executable graph compilation |
| RACC-SYS-04 | **PARTIAL** | Recursive installed CLI help and actual JSON-RPC stdio discovery when reachable; reviewed allowlisted read-only positive/invalid calls; every source-derived action gets a typed blocked/partial row | All actual installed CLI leaf, SSH/HTTP policy catalogs, safe mutating positive/invalid/denial on owned targets, independent per-action oracles, 100% runtime breadth |
| RACC-SYS-05 | **PARTIAL** | Immutable test-owned UUID/incarnation once-only reservation, durable target freezing on potential UNKNOWN, no automatic retry; negative unit fixtures | Full concurrent graph executor, fault injection, independent real state transitions and cleanup; no live two-actor scenario executed in this change |
| RACC-SYS-06 | **NOT IMPLEMENTED (actual automatic agents)** | Typed Finder/Fixer/Verifier handoff and explicit distinct actor/Box contract; emits `MANUAL_ORCHESTRATION_REQUIRED` | Automatic multi-Agent spawn/task delegation, actual separate workers, independently installed original-repro rerun |
| RACC-SYS-07 | **PARTIAL / DENY-BY-DEFAULT** | Main push no longer invokes public publish; protected workflow-dispatch candidate gate requires trusted signer, exact source/ZIP manifest, independent actor/Box and on-disk signed proofs. Fixture deny and approve tests | Real external signed Verifier artifact producer, protected environment/key setup, privately built candidate installation, green gated publication, post-public asset-byte parity |
| RACC-SYS-08 | **PARTIAL** | Scenario-first report generator, named action appendix and separate defects section; fake PASS gets downgraded | Durable issue/GitHub handoff sync and complete real executed feature/action evidence |
| RACC-SYS-09 | **PARTIAL** | Wrong-target/duplicate freeze, fake-positive/oracle, tampered cache, forged signed gate, wrong source and package leak adversarial regressions | Full load/budget campaign, cross-transport live faults, actual publication denial and success workflow runs |

**No work item above is marked fully Done.** Source tests, machine records, hashes, and CI scripts do not substitute for L2/L3 actual installed acceptance or an independently operating Agent.

## Release safety state

`main` push publishes **nothing** under the proposed workflow. Explicit `workflow_dispatch` requires `verifier_run_id`; the `candidate_gate` runs under a protected `boxweave-reacceptance-release` environment, retrieves independent run evidence, validates a configured `REACCEPTANCE_VERIFIER_PUBLIC_KEY` and verifier signature, candidate exact byte SHA-256 / source SHA / required per-scenario original + independent oracle + installed report. `publish` depends on that gate. In the absence of a configured signer or genuine Verifier run, **publication must be DENIED**. This is intentional; fixture positive tests do not override it. Do not move `PUBLISHED_RELEASE`.

Before accepting any v0.48.0 release, verify the final GitHub Actions workflow actually enforces fail-closed status, build the exact private ZIP, install/upgrade on separate disposable Box, obtain independent real Verifier original-case+Provider oracles with signed evidence, pass required system scenarios, and prove published asset byte parity. A public release created without these checks is a defect, not retroactive acceptance.

## Open safety/technical handoff

1. **Top priority:** add actual schema-safe MCP/CLI/HTTP adapters with approved per-test grants and real state/denial oracles; keep absent target capability as BLOCKED, not green. Existing 167-row source inventory is not a runtime catalog.
2. Implement a real scenario scheduler and two separate actors, including R-02/R-04/UNKNOWN without owner-host reboot/global protection; retain exact original unknown and no-replay.
3. Implement genuine separated Coordinator/Finder/Fixer/Verifier Agent execution, pinned original repro, isolated package delivery, separate verifier run artifact/signing and positive/negative GitHub CI rehearsal.
4. Finish remaining scenario families, cross-channel parity and GitHub incident/handoff durable queue, cleanup/readback/limits; run native Containarium version observations (dev Box has none).
5. Complete production ZIP source+manifest inspection, clean install and upgrade; source fixture tests are insufficient. Preserve all existing older open SUSPECT/UPSTREAM/UNKNOWN incidents, especially R-02 deferred reboot, R-04 global constraints, #75 upstream disk shrink and #79 Compose.

**Operator invocation remains unchanged:** `Run full BoxWeave reacceptance`. Use the system-first canonical runbook. If the new external runner cannot execute a required scenario, perform its authorized manual steps or record exact BLOCKED; never announce full automation or independent PASS.
