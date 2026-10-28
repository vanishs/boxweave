# TASK-002a — bounded receipt history

Scope selected before implementation: REQ-007/011 and AC-007/011 discovery/read subset. Parent TASK-002 remains partial for other recovery obligations. Dependencies: existing TASK-001/002 receipt storage and TASK-004 authenticated HTTP/UI; developer gate SRC-051. This task does not depend on automatic client identity provisioning or TASK-006 infrastructure lifecycle writes.

## Acceptance and test design

- Add actor-scoped, bounded keyset receipt history ordered by creation time and ID, including deterministic ties, with an opaque validated cursor. No remote provider call, output payload or replay in listing. Existing operations/latest and CLI receipts contracts remain available.
- Expose authenticated history and selected receipt details in Web. Use explicit refresh/older-page controls; preserve existing receipt details on refresh failure. Unknown and failed states retain their real status/exit boundaries. Query is observation, never a retry/cancel action.
- Add a CLI history action using the same service and an explicit limit/cursor; preserve existing CLI receipts output.
- Test tied timestamps, more than100 receipts, malformed/bounded cursor and limit, actor isolation, empty/end pages, new inserts during paging, persistence after restart and unchanged operation/audit/provider-call counts. Test unauthenticated/expired HTTP and invalid query parameters, plus actual CLI paging.
- Run Chromium history selection/older pages/reload/API-worker restart with real HTTP/SQLite and explicit fixture infrastructure; verify unknown detail and no replay, mobile layout, and original hello/graph regressions. No new public route, persistent service or real workload mutation is required.

Version 0.2.3, code parent 9c5b52e53ba359bc75ecf307845704a0f4eb1af0. SRC-061 authorizes the new iteration; SRC-062 records actual evidence. Source was authored/tested in assigned boxweave-tenant-dev under /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a; reviewed exact source and sanitized evidence synchronize to rewrite/v2. Legacy main remains d8a09dd with only the unrelated untracked archive, untouched.

## Delivered interfaces

Service.receipt_history(actor="operator", limit=20, cursor=None) selects only that actor's stored operation summaries (id/project/state/code/created). Maximum100 items, plus one lookahead row; sort created DESC,id DESC. The cursor is bounded256 characters and contains a finite timestamp and a32-character lowercase hexadecimal operation ID. It is an opaque seek position, not a credential or signed authorization token. Invalid cursor/limit gives INVALID_CURSOR/INVALID_LIMIT. Actor scoping is applied independently to every query; details retain their existing actor check.

GET /api/operation-history?limit=20&cursor=... returns operations and nullable next_cursor under the existing authenticated session. Reject unknown or repeated query keys. Summaries exclude command output, target digest and provider credentials. The existing /api/operations/latest-style flow and /api/operations/<id> detail remain unchanged; list_receipts and CLI receipts retain their previous list contract. No schema/data migration or new persistent service is introduced.

The Web receipt section adds Refresh receipt history, Older receipts and an Open receipt button per item. A selected receipt uses the existing detail display/polling. Failed history requests preserve the previous page and selected detail; logout clears them. Refresh explicitly returns to newest history; old pages do not automatically ingest newer inserts. A cursor is not a cross-request snapshot: already-known receipts may change state and intentionally inserted older records may appear in subsequent pages. No fabricated total-count or completion claim.

CLI history returns the same paged envelope. Example:

```bash
.venv/bin/python -m bwmini history --config /private/reviewed.json --limit 20
.venv/bin/python -m bwmini history --config /private/reviewed.json --limit 20 --cursor '<returned next_cursor>'
```

Only the new history action accepts cursor/nondefault limit. The config's existing private-local-state boundary still applies; CLI does not add remote or multi-user authentication. Querying does not invoke recover/worker/provider. Receipt retention, provider reconciliation and automatic retry/cancellation are not implemented by this feature.

## Actual validation

All commands below ran as non-root boxweave-tenant-dev in the assigned LXC:

```bash
.venv/bin/python -m unittest discover -s tests -p test_history.py -v
.venv/bin/python -m unittest discover -s tests -v
cd web && npm run build
PYTHONPATH=. .venv/bin/python scripts/history_browser_check.py
PYTHONPATH=. .venv/bin/python scripts/browser_check.py
PYTHONPATH=. .venv/bin/python scripts/graph_browser_check.py
```

Run the PYTHONPATH commands from the repository root after returning from web. Five new history tests passed (1.760s); complete suite46 tests OK (12.686s), exit0. TypeScript/Vite build exit0; known client-only XYFlow use-client directive warning, no new dependency. Actual Python3.12.3, Node18.19.1, Playwright1.56.0, Chromium141.0.7390.37, SQLite local state.

Final Chromium history run exit0:20+6 pagination over26 receipts, unknown detail with no success output, failed refresh preserves evidence, selected old receipt rediscovered after API/worker restart, mobile no horizontal overflow/no browser errors. Exactly one newly confirmed fixture hello executed; the other25 terminal-unknown receipts were seeded fixtures. Final DB has26 operations and one execution audit record. This is real UI/HTTP/SQLite evidence with fixture infrastructure, not a new live-MCP execution claim. Service tests separately replace the provider with a raising object and compare the entire database dump before/after paging; zero provider access and no state changes.

History final artifacts: .runtime/history-browser/d4969ff006644592. Original hello browser regression exit0 (.runtime/browser/e9558105007c5924, one fixture hello); Graph browser regression exit0 (.runtime/graphs/8af2e8a990dc1a38, zero graph execution). Existing original/latest flow remains tested. Source hashes are in receipt-history-source-hashes.json; screenshots/state stay ignored/private, not committed wholesale.

Intermediate harness-only failure: an immediate repeat's pre-bind probe encountered port18644 TIME_WAIT after the previous server closed, with no remaining listener. The new harness now sets SO_REUSEADDR on its bind probe, still refuses an active conflicting listener, and final reruns passed. It never terminates another process to acquire a port. Visual review found a garbled separator introduced during source transport; changed it to JSX &middot; and rebuilt/reran the final history browser check successfully. Original failed attempt is not counted as passed.

## Completion and limits

TASK-002a is complete in its history-discovery scope. Parent TASK-002 still has other DB-fault/reconciliation/recovery obligations; TASK-006 writes, client bootstrap/other profiles, Tenant/ports/model/install/release remain separate. RISK-020 is unchanged. This feature needs no infrastructure capability and does not waive another task's prerequisites. No deployment to the published fixture demo, new box, public route, model call or provider write. Only task-owned loopback test processes were stopped; final ports18640/18643/18644 have no listeners. No push. See current HANDOFF/PHASE_STATUS for the containing code/test/docs commit and next-task suggestion.
