# Legacy behavior baseline

Sources resolve in [SOURCE_REGISTER](SOURCE_REGISTER.md). No legacy runtime/test execution occurred in phase 1. Code/test inspection establishes intended or inferred behavior only. Historical reports are explicitly separate.

| ID | Conclusion and boundary | Status/source | Reproduction or future validation | v2 implication |
| --- | --- | --- | --- | --- |
| B-001 | Configuration precedence: flags, explicit environment, saved config, detected defaults; reruns preserve identities/extensions | Static-confirmed contract, SRC-002 | Non-root Linux `tests/test_installer_rerun.py` sandbox, then separately approved real update | Avoid accidental reset during repair |
| B-002 | Downloads stage before publication; writes atomic per file; failures retain pending marker; no full rollback | Static-confirmed, SRC-002 | Download/checksum/service-failure sandbox cases; approved isolated install needed for runtime | Specify recovery limits honestly |
| B-003 | Oneshoot inactive/dead can be successful; new unloaded units must not be reset-failed blindly | Static-confirmed and historical-reported, SRC-002/005/011 | First-install regression and systemd Result/ExecMainStatus/timer observations | State is richer than service active |
| B-004 | CLI ready/failed/starting-incomplete map to 0/1/2; reachable HTTP 200/401/403 does not prove MCP authentication | Static-confirmed, SRC-005/009 | Readiness tests; real initialize/initialized/list/call for authentication | Keep independent evidence levels |
| B-005 | Create timeout is unknown; original name must be queried without duplicate replay; Running does not prove SSH ready | Static-confirmed and historical-reported, SRC-006/009/011 | `test_ambiguous_create_is_reconciled_without_duplicate`, persisted-deadline and resume cases in test_smoke.py | Persist intent and uncertainty |
| B-006 | Host-key reconciler uses local/default Incus, approved private IPv4, authentic keys and network comparison | Static-confirmed, SRC-008 | test_hostkeys.py; approved IP-reuse test if later needed | Never replace trust failure with bypass |
| B-007 | Enrolled transport is forced-command and broad provider authority can still be dangerous | Static-confirmed, SRC-008 | Separately scoped least-privilege positive/negative integration tests | Transport restriction is not per-action authorization |
| B-008 | Client probe negotiates 2024-11-05, paginates up to 16 pages, bounds bytes, rejects empty/malformed CallToolResult | Static-confirmed POSIX code, SRC-004/014 | test_diagnostics.py and native Windows fixtures; no execution in this phase | Old research defects must be checked against current code |
| B-009 | Diagnostic connection, registration, guidance and GUI are separate; Claude registration can be unsupported | Static-confirmed docs/code, SRC-004/014 | Real enrolled profile matrix; GUI observation separate | Success cannot conceal unsupported inspection |
| B-010 | Web uses exact confirmation plans, authenticated session/Origin/CSRF, per-row receipts and unknown outcomes | Static-confirmed, SRC-006 | test_web.py and browser fixtures with fake provider, then approved real integration | Avoid equating HTTP acceptance with completed infrastructure |
| B-011 | Local Tenant name/selection cannot override native membership; Guard freshness/enforcement and backend identity matter | Static-confirmed, SRC-007 | test_tenants.py including unknown/remote/partial inventory and stale Guard | Separate metadata from enforcement authority |
| B-012 | Native shared/access writes require authenticated exact contract local-lxc capabilities; ingress edits preserve unrelated policy with revisions | Static-confirmed, SRC-007/010 | test_tenant_network.py, real provider-contract fixture and PostgreSQL lane; TCP separate | No version-string or label-based unlock |
| B-013 | Configured policy, verified Guard and actual TCP connectivity are different statuses | Static-confirmed, SRC-007/010 | Approved packet test needed; not performed here | Do not certify isolation from configuration |
| B-014 | Prompt Graph is persisted unordered context; optimistic revisions prevent silent lost edits; export escapes Markdown and rejects common secret patterns | Static-confirmed, SRC-006 | graph/store tests and browser competing revision/copy scenarios | No implied execution scheduler; secret regex is incomplete protection |
| B-015 | Lost job worker yields unknown/null exit; receipt query success is not workload exit 0; deadlines cannot guarantee remote cancellation | Static-confirmed, SRC-009 | test_jobs.py disappearance/nonzero/deadline/output cases | Preserve actual completion evidence |

## Historical runs, not fresh acceptance

- SRC-011 records patched installer rerun exit 0, authenticated inner MCP discovery/create/connect, identity exit 0 and Python stdout `hello world`, exit 0 on 2026-10-22. Original installer attempt exited 1. LXC boot autostart, host confinement and fresh Windows onboarding were not verified.
- SRC-015 records Web real-service checks and automated suites with fake-provider distinctions. Earlier downstream/fork runs are not current upstream production acceptance.
- SRC-018 records earlier native Tenant reviewer regressions despite a green existing suite; SRC-019 records subsequent fixes and PostgreSQL execution. Both are historical, different source heads. Current deployed support is unknown.
- SRC-017 records a later nested boundary sample with enforced AppArmor observations, yet no verified device filter. It does not prove complete effective policy or current host state.

## Current uncommitted observation

SRC-020: working-tree `server/clients.py:run` calls replacement `choose(...)` without assigning its return; options passed to render contain provider/model only. `render` can embed replacement when given `options['replace']`. **Inferred potential defect:** interactive replacement may not reach generated scripts. Not reproduced, not fixed; not part of committed main baseline. Track RISK-004.
