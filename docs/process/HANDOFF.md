# Phase 4 handoff — retained partial minimum system

Date: 2026-10-28 America/Los_Angeles. **Result: partial implementation; real minimum-system gate not passed.** User SRC-030 approved the exact execution scope; after the missing independent application launcher was reported, SRC-032 confirmed no path is available and requested preservation. No new credential was minted or borrowed.

Legacy: refs/heads/main@4318da6ea4e4e45c55cf43b64acbdd2ede65fc43; last knowledge sync same SHA. Branch/worktree: rewrite/v2 at C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2. Phase 4 parent: 40a600a56a7fd3a08a0c71b6763bbd7d79f7f621. Exact implementation commit is reported after committing; locate with `git log --format=oneline --grep='feat: retain phase four project binding and receipt slice' rewrite/v2`. Historical phase 3 handoff is archived separately.

## Delivered implementation

- New Python 0.1.0 minimum package with typed target/binding contracts, private SQLite state, revisions, one-use action plans, separate worker lock/intent/receipt and unknown/no-replay recovery.
- Real authenticated Flask/Waitress API and React/TypeScript interface for project binding, exact hello confirmation and receipt observation. No arbitrary Web command. Fixtures are explicitly labeled and cannot silently replace missing live configuration.
- Bounded POSIX MCP stdio adapter with lifecycle, pagination, read-only observation, exact identity verification and fixed hello parsing. Real independent application credential/installed-provider compatibility is unverified.
- Native Windows connection-only PowerShell probe and three no-network rejection tests; no enrollment or registration modification. It never calls overall acceptance passed from connection success.
- Python runtime/test wheel hash locks, npm lock, runnable fixture/live-placeholder examples, README, reproducible commands and desktop/mobile fixture screenshots.

All product implementation was authored inside `/home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a` in the approved non-root Ubuntu LXC, then synchronized with 28 matching source SHA256 values. Old remote worktree at 7818668 and its existing 32 dirty/untracked entries were not modified. Local main/client WIP/archive remain unchanged.

## Actual validation

29 Linux tests passed (exit 0); same 29 passed in a clean offline/hash-locked virtual environment (exit 0). TypeScript/Vite production build and Python compilation passed. Real Chromium passed login/binding/preview/confirm/receipt/API+worker restart/no-replay/mobile-no-overflow with synthetic infrastructure; exactly one hello fixture call. Windows PowerShell 5.1.26100.5651 AST and three no-network preflight cases passed, exit 0. Dependencies checked without inconsistency.

Direct Platform MCP identity/hello returned the approved username, hostname and `hello world`, exit 0. This is **separate** from the fixture Web result and does not establish real Web->worker->MCP E2E. The live prerequisite fails explicitly as MCP_LAUNCHER_UNAVAILABLE. Full commands, stdout, exits, REQ/AC/TASK mappings and limits are in [VERTICAL_SLICE](../verification/VERTICAL_SLICE.md) and [FEATURE_MATRIX](../verification/FEATURE_MATRIX.md).

## Design adjustments and incomplete work

[ADR-005](../design/ADR/0005-phase-four-boundary.md) records the approved loopback fixture HTTP boundary, consolidated minimal binding record and preservation after missing credentials. The architecture's required production identity/HTTPS and real-client/live acceptance are unchanged obligations, not silently waived.

TASK-001 minimum contracts/package and TASK-004 minimum fixture UI are implemented. TASK-002 and TASK-003 remain partial (full reconciliation/DB-full coverage/enrollment/payload trust/real client still absent). TASK-005 is not satisfied. Current fixed operation budget is 60 seconds bounded at 600; no provisioning/create workflow is implemented. No full first-release AC is claimed passed merely from component tests.

The remaining prerequisite is a separately reviewed independent protected application MCP launcher/identity; provide/configure its supported path later without sharing secret text. Then complete the real application/native-client chain and ask the developer to accept architecture and experience. No phase 5 feature work can start before that gate. Tenant/ports/Graph/model wizard and four-profile acceptance remain selected future scope, not completed functionality.

## Cleanup and preservation

Owned temporary API/worker/browser processes were stopped; `ss` showed no listener on 18640. No private-address/public listener, route, policy, container lifecycle, persistent service, provider/Incus upgrade, model request, global trust or real client registration change occurred. Source, dependency artifacts, sanitized test logs and private synthetic state remain in the isolated remote directory for review. No secrets/database/raw old logs entered the commit.

Final source/link/ID/whitespace and repository preservation checks are performed before commit; their actual results are reported below. Stage 4 remains partial regardless of documentation-check success. No phase 5 work was started.

Final document/source validation passed: 35 Markdown files, 71 valid local links, source/risk/ADR references defined, supplied phase 4 prompt byte-identical and archived phase 3 handoff equal to the prior Git blob. All 28 synchronized source files match the remote SHA256 manifest. Tracked whitespace check passed. Private-key/JWT payload pattern scan found none; this narrow scan is not a universal secret detector. Final staging/commit checks cover newly added files as well. These checks do not change the partial/live-gate status above.
