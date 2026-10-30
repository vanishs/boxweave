# R-00/R-01 — Verification handoff for an authorized AGENT

**Scope of changes proposed in this handoff:**

1. Replace root `README.md`: correct current `0.9.1`/Phase 7 status and create a single entry for supported features, external dependencies, safe fixture use, independent app installation and realistic evidence limits. No application logic changes.
2. Add `docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md`: code-anchored `F-001`–`F-015` old-to-v2 matrix with per-row acceptance gaps and separate product decisions. **Neither legacy code nor legacy docs are copied or edited.**
3. Add this test handoff. It records commands/expected behavior without asserting they were executed.

Source versions inspected on 2026-10-29: `main@19b3b61c75c85754d6b53699f5bfcdb4e0791b04`, `archive/0.5.1@d8a09dd180139159e035963502e7793a37e92bae`. Before work, verify actual refs and `git status` in the **authorized v2 checkout/worktree**; do not change or clean the legacy worktree. If either ref has moved, reconcile the changed source before marking anything current. Follow [FINAL_HANDOFF](../process/FINAL_HANDOFF.md), [MAINTENANCE](../process/MAINTENANCE.md), [SCOPE](../product/SCOPE.md), [ACCEPTANCE](../product/ACCEPTANCE.md) and [OPEN_RISKS](OPEN_RISKS.md).

## A. Required immediate checks (documentation-only, no live systems)

| ID | What to run/inspect | Expected evidence and failure conditions |
| --- | --- | --- |
| **D01** | From checkout root: `python scripts/check_knowledge.py` (portable). Verify `README.md` and the two new docs appear in local-link checking. | Exit 0 and no broken local links. Capture exact stdout/exit plus source SHA. On failure, fix *new* links only; do not weaken authority/REQ coverage. The tool checks link/navigation/ID consistency, **not** the truth of README prose. |
| **D02** | Human/code-backed review of `VERSION`, `pyproject.toml`, `web/package.json`, `docs/process/PHASE_STATUS.md`, `docs/verification/PHASE_07_REVIEW.md`, `docs/product/SCOPE.md` against README. | All current version statements resolve to `0.9.1`; Phase 7 is finished **within stated maintenance scope**; only Codex/ZCode × Windows native/explicit WSL is advertised. Reject claims of a new live test based solely on these documents. |
| **D03** | Review README commands with current `bwmini/__main__.py`, `bwmini/distribution.py` argument parsing, `examples/fixture.json`, [INSTALL](../INSTALL.md). Search for `main/install.sh`, stale `Phase7 ... has not started`, unfounded macOS/Cursor/Claude support, or implied fresh deployment claims. | No executable obsolete `main/install.sh` path; each example command has an implemented entry; private MCP configuration is not fabricated; no credential in documentation. Test exact argument meanings rather than old CLI names. |
| **D04** | Recheck each F-row in [AUDIT](ARCHIVE_051_TO_MAIN_AUDIT.md) against **code at pinned refs**, especially old `server/webui/app.py` endpoints and current `bwmini/web.py`. | Each code anchor exists and its interpretation matches executable conditions. Reclassify any uncertain row to `U`, not `R`. Confirm that old docs have not been used to establish implementation. |
| **D05** | Review rendered Markdown on GitHub or a local Markdown renderer and follow the README onboarding links manually. | Table legible, commands copyable, code fences valid, historical branch distinguishable, links land on current files. A renderer check does not prove installation success. |

Recommended static review commands (adjust to approved worktree):

```sh
git status --short --branch
git rev-parse HEAD
git rev-parse refs/heads/main
git rev-parse refs/heads/archive/0.5.1
python scripts/check_knowledge.py
rg -n 'main/install\.sh|Phase7.*not started|Phase 7.*not started' README.md
```

`rg` absence of matches is only a content check, not proof of capability. If the archived ref is only available through a remote connector, read the pinned SHA and exact source path rather than assuming a local branch. Do not run `git clean`, implicit merge, force push, release publishing or host repair as part of this handoff.

## B. Runtime and legacy parity queue (DO NOT mark passed from this document)

All Linux development/installation/browser checks belong **only** in the existing assigned prepared, non-root project LXC after verifying `whoami`, `hostname`, checkout identity, config and existing resources. No new Provider write, key enrollment, route/TLS change, host command, model inference or destructive cleanup is authorized by R-00/R-01. Separate approval is required for any actual live resource experiment, Windows Agent change, or public-facing service.

| ID | Changed assumption / what needs testing | Safe suggested path, proof and pass criteria | Boundary |
| --- | --- | --- | --- |
| **V01** | README says current package installer supports reviewed standalone application install/repair, not host bootstrap | First `tests/test_distribution.py`; separately scoped existing isolated app-copy package check using [INSTALL](../INSTALL.md). Record source/zip SHA, actual config ownership, admin setup, backup/schema, marker after failed postpublication health, repair exit. | Never run old `install.sh` or update existing primary 0.9.0 service without approval. |
| **V02** | Windows Codex/ZCode native/explicit WSL client profiles, digest/pin/registration conflict | Run `tests/test_profiles.py`, `tests/test_enrollment.py`, `tests/test_public_profiles.py`; later authorized Windows native + explicit WSL registration, version/distro/known_hosts/replace and actual Agent UI checks. Capture per-profile real connection vs registration vs GUI. | No implicit key generation, credential read, trust bypass or profile modification without approval. |
| **V03** | ZCode GLM is native selection/handoff, not a Claude adapter | Run `tests/test_models.py` and fixture `scripts/model_browser_check.py` in prepared LXC. A separate authorized existing ZCode real-client check must verify provider/version/access; model inference is **not** covered by these tests. | No API keys in public UI/reports, no paid inference. |
| **V04** | v2 diagnosis is read-only app/MCP, not old host `doctor` | Run non-mutating `python -m bwmini diagnose --config /reviewed/config.json` only with existing approved MCP. Test zero selected targets, invalid launcher, unsupported protocol, actual `initialize`/`tools/list`/`list_containers` response and exit; measure DB bytes before/after if approved. | External host/systemd/token/Incus health belongs to a separate administrator; do not repair it. |
| **V05** | Fixed hello receipt/recovery replaces generic old subprocess job | `tests/test_slice.py`, `tests/test_history.py`, `tests/test_recovery.py`; confirm lost worker/timeout/nonzero/unknown and no replay; optional `scripts/browser_check.py` fixture. | No arbitrary shell/agent execution inferred. |
| **V06** | Current environment lifecycle and old nested selftest are not identical | `tests/test_lifecycle.py`, `scripts/lifecycle_browser_check.py` fixture; only a separately authorized real project target may test create/start/stop/delete and delayed uncertain response. | No duplicate create, force delete, fresh provider create without explicit authorization. |
| **V07** | New console security, project binding, receipts and monitoring claims | `tests/test_bindings.py`, `scripts/binding_browser_check.py`, other fixture browser lanes; examine login/CSRF/unbind-preserves-graph/receipts, old/new visible field map, mobile UI and JS console. | Fixture HTTP/SQLite is not production or network evidence. |
| **V08** | Tenant membership/ingress and TCP/UDP routes are bounded selected features | `tests/test_access.py`, `scripts/access_browser_check.py`; check capability/Guard/revision/conflict/degraded revoke/route approval. Any real packet/network acceptance is an independently authorized experiment. | A stored route ≠ packet reachability, TLS or enforced isolation. |
| **V09** | Tenant metadata CRUD, selector/filter and old Monitoring were not proven equal | Code/API field comparison plus two read-only UI fixture walkthroughs on separate approved sources; list old workflow, current equivalent, error/permission behavior and screenshot. | Not authorized to reintroduce old CRUD, dashboard or changed user scope merely because an old endpoint exists. |
| **V10** | Prompt Graph persistence, conflicts and safe export | `tests/test_graphs.py`, `scripts/graph_browser_check.py`; save/reopen/export/copy-failure, no Provider writes. | No automatic graph execution or universal secret redaction claim. |

If the prepared environment is available and authorized, the existing no-live-driver lane is:

```sh
.venv/bin/python scripts/validate.py
.venv/bin/python scripts/validate.py --browser
```

`--browser` includes seven **fixture-only** Chromium workflows. The validator requires non-root Linux, installed locks, npm and optionally Playwright/Chromium; it does not install missing dependencies. Failure due to missing prerequisites must remain **not run / blocked**, not converted to a pass or permission for root package installation.

## C. Test-result template for AGENT to fill in

For each `Dxx` or `Vxx`, record:

```text
Check ID:
Target refs and working-copy status:
Environment and authority (hostname, whoami; no private keys/tokens):
Exact command or manual steps:
Actual stdout summary / log location (sanitized):
Actual process exit code:
Evidence kind: static | fixture | current-live | historical-live | human | blocked
Result: passed | failed | incomplete | not run
What was actually demonstrated:
What remains unverified:
Risk/REQ/AC and source record updated (if needed):
```

**Current results for this handoff:** `D01`–`D05` and `V01`–`V10` have **not been executed in the authorized project worktree** by this author. Static GitHub source comparison was completed; this is not a green test report. A later AGENT must not overwrite historical evidence hashes, weaken assertions, rewrite [FEATURE_MATRIX](FEATURE_MATRIX.md) as if it ran new tests or mark [PHASE_STATUS](../process/PHASE_STATUS.md) as a new release without actual authorization and evidence. Preserve all unrelated changes and stop at the requested R-00/R-01 boundary.