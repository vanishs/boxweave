# Phase 4 feature/evidence matrix

All rows describe partial minimum-system progress, not first-release completion. Sources: SRC-030..032; details in [VERTICAL_SLICE](VERTICAL_SLICE.md). No deferred phase-5 feature is silently dropped.

| Requirement | Criterion | Tasks | Current evidence/status |
| --- | --- | --- | --- |
| REQ-001 | AC-001 | TASK-003 | Native connection probe only; full enrollment/replacement integrity path not implemented |
| REQ-002 | AC-002 | TASK-003 | Windows pin preflight and synthetic MCP mismatch; live trust chain unverified |
| REQ-003 | AC-003 | TASK-003 | Bounded POSIX client + native connection-only probe; real registration/UI incomplete |
| REQ-004 | AC-004 | TASK-001, TASK-002, TASK-005 | Typed scope/creation identity + zero-dispatch-on-change tests; direct Platform identity separate from application live path |
| REQ-005 | AC-005 | TASK-002, TASK-006 | Intent/one-use/no-replay groundwork; real create not in slice |
| REQ-006 | AC-006 | TASK-002, TASK-005 | Persisted deadline and unknown states, fixed result validation; live overall bound/cleanup not proven |
| REQ-007 | AC-007 | TASK-002, TASK-004 | SQLite receipt query/restart and synthetic loss recovery passed |
| REQ-008 | AC-008 | TASK-012 | Unstarted installer/recovery release work |
| REQ-009 | AC-009 | TASK-012, TASK-013 | Explicit fixture/live prerequisite failure only; full installed-health contract unstarted |
| REQ-010 | AC-010 | TASK-002, TASK-004 | Binding create/update/revisions/persistence verified; unbind/live binding continuation pending |
| REQ-011 | AC-011 | TASK-004, TASK-005 | Real authenticated loopback Web/browser fixture verified; production HTTPS/live services incomplete |
| REQ-012 | AC-012 | TASK-009 | Not started; requires phase-4 developer gate before feature work |
| REQ-013 | AC-013 | TASK-010 | Not started; no routes published |
| REQ-014 | AC-014 | TASK-011 | Not started; minimal UI intentionally has no graph |
| REQ-015 | AC-015 | TASK-008 | Not started; no model requests or credentials moved |
| REQ-016 | AC-016 | TASK-003, TASK-012 | Exact Linux wheel locks/source hashes; enrollment payload trust distribution incomplete |
| REQ-017 | AC-017 | TASK-005, TASK-013 | Linux/browser fixture + Windows preflight evidence; no real selected-profile acceptance |
| REQ-018 | AC-018 | TASK-004, TASK-005 | Target/action allowlist, auth/CSRF, actor-bound receipts; upstream broad privileges/production boundary unverified |
