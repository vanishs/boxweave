# TASK-011 — basic Prompt Graph, 0.2.0

Date: 2026-10-28 America/Los_Angeles. Developer gate SRC-051 explicitly accepted the verified minimum architecture/slice and selected this one phase-5 iteration. Covers REQ-011/014/016 and AC-011/014/016 only within graph scope; this is not complete release/client acceptance. Actual evidence SRC-052.

## Delivered behavior

Authenticated Prompt Graph view with React/TypeScript + exact @xyflow/react 12.12.0: named graphs, graph prompt, editable/draggable cards, editable comments, undirected links, textual project/environment/Tenant context, explicit saves, reload, deletion, Markdown download and browser clipboard copy. Node positions/context/comments/relations persist. Changed graph or viewport fits the canvas after layout; mobile controls remain available.

All graph operations use local Graphs + Store only, without a provider dependency. Context labels are text, not membership claims or execution references. No relation runs a command/agent, creates a workload, changes policy or publishes a port. Console login/project bootstrap may independently observe its configured provider; graph API actions themselves do not do so.

Explicit saves use whole-document optimistic revisions. A stale editor gets REVISION_CONFLICT without overwriting server data or its local draft. Reload explicitly discards local changes after confirmation and refreshes current list metadata. Dirty graphs block view switching; new unsaved drafts can be discarded explicitly, and a before-unload warning handler is registered (browser display is browser-controlled). Unsaved drafts are not crash-durable. Deleting a graph requires its current revision and confirmation; only that graph and its copy reports are removed, never environments/projects/policies.

## Data/API contract

Schema 2 adds graphs(id, revision, document) and graph_copies(id, graph, revision, at, status). Copy reports cascade on graph deletion; retain the latest 1000 reports globally. Schema 0/1 -> 2 creation/migration runs within BEGIN IMMEDIATE, rechecking version under the writer lock. Existing administrator/project/session/operation data remains intact. Unsupported versions fail closed; failed migration rolls back. Older 0.1.x binaries refuse schema 2. Before updating a persistent deployment, stop its API/worker and make an operator-private consistent SQLite backup; use that compatible backup for a downgrade. This task does not deploy or perform whole-host rollback.

Document keys: name, prompt, cards, comments, links. Each card has id/title/body/x/y/context; context has project/environment/tenant strings. Comments have id/text/card (nullable card ID); links have id/source/target. IDs ASCII letters/digits/underscore/hyphen, 1..64 characters. Reject unknown fields, duplicate IDs, orphan comments/links, self links, duplicate undirected pairs, nonfinite coordinates and boolean revisions. Bounds: 100 saved graphs; 24 cards/comments/links each; name 80, prompt 2000, title 100, body 2000, context 120, comment 1000 characters; coordinates +/-10000; encoded document <=12000 UTF-8 bytes, existing HTTP request cap 16384 bytes.

| Endpoint | Behavior |
| --- | --- |
| GET/POST /api/graphs | list / create with {document} |
| GET/PUT /api/graphs/{id} | read / save with {document, revision} |
| DELETE /api/graphs/{id} | delete with {revision} |
| GET /api/graphs/{id}/markdown | {markdown, revision}, no copy report |
| POST /api/graphs/{id}/copies | {revision}; store reported_success, not independent proof of physical clipboard state |

Existing session/Origin/CSRF/request-size boundaries protect these APIs. Copy/export require saved content and matching revision. Client only POSTs a success report after await navigator.clipboard.writeText succeeds. Clipboard rejection creates no report. A failed report after successful copying says copied locally but report not recorded; no false server success.

Markdown escapes syntax/HTML-bearing punctuation and redacts recognized private-key blocks, JWT-shaped tokens and api-key/token/password/secret assignments in prompts, cards, comments and context. Original private graph text is not automatically rewritten. This is a composition tool, not a secret vault: pattern recognition is incomplete; review content before sharing. Normal errors contain codes, not prompt bodies or credential-bearing diagnostics.

## Actual validation

All product implementation/dependency installation/tests authored and executed as non-root boxweave-tenant-dev inside boxweave-tenant-dev-container, in /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a. Identity check exit 0. Existing legacy remote checkout was not modified. Local synchronization is exact-byte checked using [13-file manifest](phase5-graph-source-hashes.json).

| Command | Actual outcome / evidence |
| --- | --- |
| .venv/bin/python -m unittest discover -s tests -v | 41 tests OK, exit 0, 11.173 seconds; 11 graph/migration/API tests + existing 30 |
| npm install --save-exact @xyflow/react@12.12.0 --ignore-scripts --no-audit --no-fund | exit 0; exact dependency and transitive integrity in npm lock |
| cd web && npm run build | TypeScript + Vite 6.4.5 passed, exit 0; final JS 388.69 kB / gzip 123.67 kB |
| PYTHONPATH=. .venv/bin/python scripts/graph_browser_check.py | final pass, exit 0; actual Chromium 141.0.7390.37, HTTP/React/SQLite, fixture infrastructure, zero execution operations |
| PYTHONPATH=. .venv/bin/python scripts/browser_check.py | original project binding/hello/restart/no-replay/mobile fixture regression passed, exit 0, one hello fixture call |
| .venv/bin/python -m compileall -q bwmini tests scripts | exit 0; syntax only |

Graph browser exercised card edit/drag, comment editing/context/link persistence, full reload, two-editor conflict preserving draft, explicit draft discard, actual Chromium clipboard API, injected clipboard rejection with no extra report, equal escaped/redacted Markdown download, graph deletion preserving the existing project, mobile canvas fit and no horizontal overflow. No page script errors. The graph API test replaces provider observe/hello with forbidden calls and passes all graph operations; browser database has zero operations and fixture hello counter is empty. This is not a live MCP workload test and does not claim one is required for pure graph composition.

Browser evidence: /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a/.runtime/graphs/70173ffb74452f30. Sanitized [browser result](phase5-graph-browser-result.json); ignored local screenshots .runtime/phase5-graph/graph-desktop.png and graph-mobile.png. Screenshot test text is deliberately synthetic. No database, key, password, raw user data or dependency tree is committed.

Early browser checks exposed ambiguous control labels and stale graph-list metadata; explicit accessible labels and reload refresh fixed them. Mobile viewport timing was tightened and responsive canvas fitting added; final geometry check verifies both nodes fit. A test helper initially used a CSP-incompatible dynamic wait expression; it now polls ordinary browser evaluation without changing product CSP. Vite emits a non-fatal warning that XYFlow's use-client directive is ignored for this client-only build; build/browser checks passed. No CSP relaxation was added.

## Retention, provenance and next iteration

Version and Python/UI package metadata are 0.2.0 (significant basic feature). The MCP client metadata version is aligned; transport behavior is unchanged. Python 3.12.3 / Node 18.19.1 / npm 9.2.0 / React 19.1.1 / XYFlow 12.12.0 / Playwright 1.56.0. Temporary graph and regression servers are stopped by harness finally blocks. Source, locks and private synthetic evidence remain in the project LXC. No persistent service, deployment, host setting, public route, new box or model request was changed/created by TASK-011.

Legacy main remains d8a09dd180139159e035963502e7793a37e92bae at this phase boundary; no new committed knowledge delta since SRC-048, no legacy implementation merge or archive read. User acceptance SRC-051 opens this iteration only; actual Windows/WSL Agent registration and full selected-feature release evidence remain obligations. TASK-011 is complete in its stated basic scope; stop this iteration. Suggested next work: finish the minimum actual client enrollment/recognition evidence in TASK-003/005, then enable dependent TASK-006 lifecycle work. No automatic dependency waiver is inferred from graph success.