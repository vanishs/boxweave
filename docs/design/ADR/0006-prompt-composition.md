# ADR-006 — revisioned, non-executing prompt composition

Status: implemented 0.2.0, authorized SRC-051, verified SRC-052. TASK-011 / REQ-011/014/016.

Use the architecture-selected XYFlow with exact npm lock for a bounded canvas; keep a single typed whole-document snapshot in local SQLite with optimistic revisions. Normalize undirected relationship identity during validation (pair comparison), reject malformed references and preserve local drafts on conflict. Explicit save/reload is preferable to silent last-writer-wins or graph execution semantics. Context is plain text, never a provider authorization or membership override.

An additive transactional schema-2 migration preserves schema-1 data. Old binaries refuse schema 2; operator-private consistent backup is the downgrade boundary. No provider framework, agent orchestration or schema downgrade is introduced. Graphs depends only on Store; existing Web authentication/Origin/CSRF boundaries remain unchanged.

Copy/export uses escaped Markdown and recognized-pattern redaction, with explicit manual-review limits. Clipboard success is a client report at a specific saved revision, not server attestation. Record only after the browser API succeeds; show partial outcome if copying succeeds but report storage fails. Retain the latest 1000 reports globally and delete only graph-owned reports on graph deletion.

[Actual tests, schema and limits](../../verification/PHASE_05_TASK_011.md) distinguish real HTTP/browser/SQLite from synthetic infrastructure. No release-wide acceptance, provider call or deployment is implied.