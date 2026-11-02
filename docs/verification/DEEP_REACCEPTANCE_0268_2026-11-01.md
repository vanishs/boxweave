# Whole-project code-first deep reacceptance — 0.26.8 candidate

**Run:** 2026-11-01 America/Los_Angeles (2026-11-02 UTC). **Base source:** main `154eb8b8de931f4647da37271fcd7132f9e8489a`, source candidate 0.26.8. **Scope:** 38 code/feature acceptance areas, revised from the historical [A01–A33 ledger](REACCEPTANCE_0262_TO_0264_2026-11-01.md), [A34 audit](DEEP_AUDIT_0266_2026-11-01.md), and [A35/A36 proof](SELFTEST_TRUST_RACE_0267_2026-11-01.md). This is a new set of executed observations, **not** an assertion that every implementation or external feature was accepted. Scope results below attach to their exact version.

**Notation:** S = inspected source; T = actually run local unit/fixture; L = authenticated real installation/guest command; D = older documented evidence, not reexecuted here; U = not independently verified; BLOCKED = no usable test transport/prerequisite. PASS is always limited to the named criterion, never the entire feature.

## Executed environment and concrete evidence

- **Source:** clean base cloned into historical `bwaudit-1101-container` (Ubuntu 24.04 / Python 3.12) at the pinned SHA; this task changed 0.26.8 source after the base check. `python3 scripts/check_knowledge.py` exited 0 (138 documents, 26 requirements/acceptance pairs). `.venv/bin/python scripts/validate.py` exited 0 with knowledge/unit/frontend-build steps PASS; **458 Python unittest cases PASS** after patch. The 0.26.8 candidate validation explicitly used `systemd-run --property=UMask=0002`, so the host-specific temporary-public-key fixture fault could not be hidden by a permissive default. Checked `git diff --check`.
- **Fixture UI:** on historical `bwaudit-1101`, installed Chromium + dependencies. Original browser-only attempt was BLOCKED by occupied `127.0.0.1:18640`, not a source error. Temporarily stopped only the existing test Box's Web serve unit, ran `.venv/bin/python scripts/validate.py --browser-only` (7/7 browser workflows, exit 0) and restored the unit (exit 0). Immediately after restart one doctor result was transient FAILED because Web had not yet become reachable; independent later `sudo boxweave doctor --mcp` returned all ten checks PASS. These are **fixture** browsers, not external production GUI tests.
- **Fresh live test:** created task-owned Ubuntu 24.04.5 `bwfullaudit-1101-container` (creation `1793589674`, native Tenant `bwfullaudit-1101`, 2 CPU/4 GiB/30 GiB, nesting true); stop/start performed after nesting configuration. Outer `get_container`, `boxweave_nesting inspect`, and `connect(exec)` independently verified target/identity, configuration, real Python stdout and exit 0. Initial overconstrained create parameters were refused `LOCAL_UBUNTU_NESTING_ONLY`; exact target absence was checked before creating with valid Ubuntu parameters (no blind retry of ambiguous creation).
- **Stable v0.26.7 installed live (NOT yet 0.26.8):** official streamed pinned install exited **0**, `boxweave version` 0.26.7, `sudo boxweave doctor --mcp` **10/10 PASS**, authenticated inner MCP list/read (58 tools), Web unauthenticated 401, `GET /api/login` 405. Fresh real app-owned nested selftest: invalid confirmation rejected `SELFTEST_CONFIRMATION_MISMATCH`, exit **2**, zero child; correct confirmed plan `boxweave-selftest-115305` returned `state=passed`, `create/hello/stop/delete` all `OK`, exit **0**, independently zero child instances afterwards. No host-key bypass or lifecycle write retry.
- **High-frequency combinations:** installed v0.26.7 generated **36/36** valid Python client scripts (nine Apps × four OS/SSH profiles), all with correct shebang and persistent SSH keepalive; this is *script generation*, not desktop enrollment. Operator jobs on same installed Box: command exit 0 → succeeded/host exit 0; child exit 7 → `JOB_EXITED`/host exit 1; budget 1 second vs sleep 4 → `JOB_TIMEOUT`/null child exit/host exit 1. No failed job was replayed. Compared historical installed 0.26.1 Web `GET /api/login` 500 against historical 0.26.4 and freshly installed 0.26.7 Web 405; unauthenticated `/api/projects` remained 401.
- **New A37 reproducible defect and fix:** `gate_diagnostics.project_summary` previously collapsed documented `systemctl --user is-active` service exit 3 (`inactive`/`failed`) or exit 4 (`unknown`) into a generic unknown command failure, even with a recognized state on stdout. 0.26.8 source now reports a bounded classified observed service state and true observed exit code only for the specific service-state/exit combinations; all other nonzero results, ports and unrecognized text remain fail-closed. Added focused positive/negative regressions, tested `test_gate_diagnostics.py` **11/11 PASS**. A **real server-allowlisted D4-positive gateway read remains U/BLOCKED**: fixing the parser and testing Python are not that proof.
- **New A38 fixture defect and fix:** when current shell `umask=0002`, temporary public-key files in two tests had mode 0664 and were correctly rejected as unsafe by production identity protection; unpatched full suite produced seven fixture errors. Two test fixtures now explicitly `chmod(0644)`, not weaker production validation. Under actual 0002, focused online 13/13 and workload-guidance 5/5 PASS; complete 0.26.8 candidate suite 458/458 PASS.

## Source-to-acceptance ledger (A01–A38)

| ID | Implemented source/feature | New evidence and exact acceptance limit |
| --- | --- | --- |
| A01 | Streamed fresh install/managed Provider | **L PASS** fresh Ubuntu v0.26.7 install exit0; 0.26.8 live upgrade **U** pending release |
| A02 | Package, digest, wheelhouse, DB migration | **T PASS** source release/packaging fixtures; v0.26.7 signed/pinned installed **L PASS**; 0.26.8 release **U** |
| A03 | Host CLI/version/status/doctor/report | **L PASS** v0.26.7 doctor 10/10; unauthorized/missing cases only T/D |
| A04 | Nine-App Python client profiles | **L PASS** 36/36 script generation/compile/keepalive; actual client registration/GUI **U** |
| A05 | Managed public key, exact Box SSH, fixed Python | **L PASS** direct exact guest `connect(exec)` and nested selftest; external host-hop path **U** |
| A06 | MCP gateway catalog/tool policy | **L PASS** authenticated inner Provider list/read; optional outer policy comprehensive test **U** |
| A07 | Incus nesting/create helper | **L PASS** outer inspect configured and actual app nested child lifecycle on clean parent; original historical failed parent unchanged |
| A08 | Box create/start/stop/delete/readback | **L PASS** task-owned outer lifecycle and inner test lifecycle; arbitrary app plan receipt/recovery **U** |
| A09 | Auto-sleep/metrics/capabilities | **D** flag behavior, **U** actual idle/wake cycle and metrics calibration |
| A10 | Jobs, recovery, no replay | **L PASS** success/exit7/timeout receipts on 0.26.7; forced worker crash / uncertain operation no-replay **U** |
| A11 | Web auth/session/CSRF | **T PASS** 7 browser fixtures; **L PASS** unauthenticated 401 and login GET 405; real external admin/browser session **U** |
| A12 | Project binding/fixed hello | **T PASS** browser binding/receipt and **L PASS** app selftest fixed hello; specific user project→real guest binding **U** |
| A13 | Native Tenant policy/network routes | **D** prior same/cross-Tenant packet tests; broader v0.26.7 live packet policy **U** |
| A14 | Prompt Graph editing/export | **T PASS** browser fixture; installed authenticated public UI **U** |
| A15 | Model wizard | **T PASS** browser fixture; paid inference/native model execution **U** |
| A16 | Agent-box MCP read-only Web adapter | **T PASS** source contract; current installed true user-bound file read **U** |
| A17 | Remote HTTP MCP initialize/tools | **T PASS** contract, external HTTP OAuth/TLS client **U** |
| A18 | OAuth PKCE/DCR/revoke/bearer | **T PASS** synthetic protocol fixtures, public/real client **U** |
| A19 | Reverse proxy/TLS/ACME/Caddy | **T PASS** configuration fixtures; live public exposure/renewal **U/BLOCKED** |
| A20 | D0–D2 diagnostics/report | **L PASS** host doctor and **T** report checks; current full private report pipeline **U/D** |
| A21 | D3 gateway/app journal | **L PASS** classified gateway read, source T; app positive trace/restart/cap **U** |
| A22 | D4 exact Box project diagnosis | **L/D PASS** default deny; **T PASS** source checks and A37 fix; live allowlisted positive **U/BLOCKED** |
| A23 | App-owned selftest | **L PASS** installed 0.26.7 real create/hello/stop/delete, zero remnants |
| A24 | Browser + external desktop UI | **T PASS** seven real Chromium fixture flows; actual Windows/macOS/WSL App UI **U** |
| A25 | Agent source bundle | **T PASS** bundle tests; authenticated cross-machine dirty-source delivery **U** |
| A26 | MCP self-information/version | **L PASS** outer catalog static source description; 0.26.8 outer running process upgrade **U** |
| A27 | Exact missing Incus instance semantics | **T PASS/D** previous real missing-instance correction; newly repeated case **U** |
| A28 | Report CLI supported kind choices | **L PASS** v0.26.7 help reports only hello/environment/access |
| A29 | Dynamic upstream Provider tools | **L PASS** authenticated inner inventory 58; availability ≠ tested functionality |
| A30 | Agent-selected end-to-end testing/cleanup | **L PASS** new Box→installed selftest→independent zero child; outer task Box final cleanup pending |
| A31 | Incus v7.6 nesting query correction | **L PASS** outer nesting inspect on task Box, actual inner child via installed app |
| A32 | HTTP 404/405 exception classification | **L PASS** old 0.26.1 login GET500 vs v0.26.4/v0.26.7 405; unknown route unauthenticated 401 is not proof of post-auth 404 |
| A33 | Post-create nesting partial acknowledgement | **T PASS** source contracts; deliberate second-stage real failure client rendering **U** |
| A34 | OAuth denial throttling durability | **T PASS** SQLite/OAuth fixtures, actual public admin auth **U** |
| A35 | Typed selftest refusal on failure | **L PASS** wrong-confirmation typed code/exit2, no mutation |
| A36 | Fixed selftest trust-readiness retry | **L PASS** one real positive v0.26.7 selftest; forced trust-race variant **U** |
| A37 | D4 nonzero service-state bug | **S/T PASS** reproduced then fixed/11-case D4 fixture; **L positive U** |
| A38 | Synthetic SSH key fixture umask independence | **T PASS** 0002-focused and 458 entire suite; production key permission guard unchanged |

## Unresolved gates and cleanup

External public OAuth/TLS/browser authentication and real client enrollment, D4 root-owned exact-Box positive policy on the *outer gateway*, privileged host-hop SSH wrapper, paid inference, actual 30-minute idle sleep, worker-kill/restart and uncertain-write recovery, cross-machine transfer of unpublished source, and deliberate second-stage partial-create failure are **U/BLOCKED**, not quietly passed. Historical 0.26.1/0.26.4 environments were read only for comparison. 0.26.8 code is a **source candidate** until GitHub CI, source-bound immutable release and a distinct installed v0.26.7→v0.26.8 checkpoint verify it. Do not promote stable installer pin before those observations.

**Resource status at source-audit checkpoint:** new parent `bwfullaudit-1101` retained temporarily for candidate upgrade; its selftest child independently absent. Original six Boxes retained, especially older failed nesting scene `bwaudit-1101`. Record final cleanup separately after candidate verification. No secrets, private keys, public ingress or network policy were changed.
