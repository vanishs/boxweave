# MCP integration implementation checkpoint and developer handoff

**2026-10-31 status: PARTIALLY IMPLEMENTED.** This is the authoritative follow-up record for [MCP capability master](../design/MCP_CAPABILITY_INTEGRATION_MASTER.md), [source gap audit](../verification/MCP_CAPABILITY_SOURCE_AUDIT.md), [M0–M5 delivery](../design/MCP_INTEGRATION_DELIVERY_PLAN.md) and [acceptance](../verification/MCP_INTEGRATION_TEST_PLAN.md).

## Baseline reconciliation

Documentation PR #6 was originally written at BoxWeave 0.21.4. At implementation start, main aaa8362 was already version **0.22.0** and diagnostics **D0–D2 were implemented** (SRC-108). PR #6 was corrected for this change and merged into main as commit 29e5dd1. D3 direct-Agent diagnostic lookup remains deferred. Environment E0–E4 remain unimplemented. The original Platform tool count 109 and agent-box 14 tools plus two Resources are **source inventories**, not installed, scoped, runnable tool counts.

## Implemented code in PR #7

- **M0 (partial, source implementation):** bwmini/capabilities.py reads the configured *application* Platform MCP via initialize/tools/list and hashes advertised schemas. It never uses tools/call. It distinguishes existing typed app adapters from merely upstream-advertised tools, and does not claim permissions or direct-Agent equivalence. The new CLI command is python -m bwmini capabilities --config FILE; Web endpoint is authenticated GET /api/mcp-capabilities.
- **M2 (partial, source implementation):** bwmini/box_debug.py reads upstream debug_container after the existing Service.observe_target verifies exact allowed, *running* Box identity. Authenticated GET /api/boxes/LOGICAL/debug-summary returns only approved state/boolean fields, withholding sshd journal lines, shell paths, free-form diagnoses and remediation strings. It does not prove SSH connectivity.
- Tests: tests/test_capabilities.py and tests/test_box_debug.py cover no tool execution, invalid catalogue/schema, fixture behavior, authenticated Web, target restrictions and redaction.
- Version: 0.23.0 **source candidate** is synchronized in VERSION, pyproject.toml, web package metadata, README and install.sh. **This does not prove a published release**.

## Verification actually performed

- Documentation PR #6: [GitHub CI 41187777954](https://github.com/vanishs/boxweave/actions/runs/41187777954) successful, including source/knowledge and fixture validation.
- Implementation PR #7 source candidate: [GitHub CI 41188227613](https://github.com/vanishs/boxweave/actions/runs/41188227613) successful: 373 Python 3.12 unit/contract tests, 57 Python 3.14 tests and offline wheel smoke, shell checks, frontend build, fixture browser workflow and knowledge validation.
- This handoff/merge commit must receive its **own** GitHub CI result before asserting final source validation.
- **Not tested:** any installed Containarium MCP tools/list and JWT scope, live debug_container, host-side Agent gateway, actual agent-box MCP, test Box network/SSH, mutating action, real application deployment or release install. CI fixtures are not production acceptance.

## Phase status and unresolved work

| Phase | Status | Remaining acceptance |
| --- | --- | --- |
| M0 | Partial source implementation | Actual installed upstream versions and scopes, agent-box discovery/resources; direct gateway is a separate principal |
| M1 | NOT IMPLEMENTED | Server-enforced per-tool policy with backwards-compatible direct-Agent trust and actor mapping |
| M2 | Limited read-only debug SOURCE IMPLEMENTED; auto-sleep NOT IMPLEMENTED | Live debug test, stopped-target policy and lifecycle/auto-resume plan/confirmation |
| M3 | NOT IMPLEMENTED | Per-Box pinned SSH/agent-box transport, immutable identity, workspace-limited reads and Resources |
| M4 | NOT IMPLEMENTED | Execution grant, process/file/Compose writes, durable single-dispatch receipt and unknown-result safety |
| M5 | Two authenticated read-only APIs SOURCE IMPLEMENTED; rest NOT IMPLEMENTED | Selected Web views, Agent test/approval workflows, run-level permissions and end-to-end acceptance |
| D0–D2 | Already implemented on main at 0.22.0 | Existing documented live checkpoint limitations |
| E0–E4 | NOT IMPLEMENTED | Snapshot/restore/clone requires separate authenticated Provider support, not database backup |

## Decisions and required engineering order

**Next: M1 trust/authorization.** Current direct Agent mcp_gate.py broadly forwards Platform tools under its own upstream identity. The app-side capability catalogue cannot protect that path. Decide a server-enforced tool policy and a staged, testable migration so existing trusted Agent clients are not silently broken. The default is not to advertise a permission system that is not enforced.

**Then: M3 first read-only, M4 later.** Determine whether a preapproved test Box actually has agent-box installed and a strict host-key-pinned, operator-owned transport. Bind immutable Box and caller identities before opening file/process/resource tools. Never accept caller-selected SSH hosts/credentials, fall back to host/Incus root Shell, auto-install agent-box or assume file-root enforcement confines shell_exec. In-run containarium-approval.approve is not a generic BoxWeave permission API.

**M2 auto-sleep and M5 workflows are optional feature decisions.** Verify support, idle/stop/resume state semantics, plans, durable receipts and no-replay recovery. Platform database backup is not an instance snapshot. Preserve the existing diagnostics and environment plan owners.

## Release / installation blocker

At this checkpoint the last observed GitHub release is **v0.22.0**, with 13 published assets. The source candidate version is 0.23.0, and the streamed installer references candidate 0.23.0. **Until matching reviewed v0.23.0 release assets are published, the streamed installer must fail closed; do not represent it as a verified fresh-host install.** Publishing release assets, tags, managing a live host or setting Provider/Box permissions needs separately reviewed execution and evidence.

## Follow-on Agent checklist

Read AGENTS.md, the current source main, this handoff, the MCP master, policy and acceptance plans. Pin source/installed/running versions and all actor/Provider/Box identities. First run focused fixtures, then only specifically authorized real tests. Record normal and negative cases, tool discovery and scope, original unknown operations, command/exit/evidence classes (S/D/T/U), unresolved blockers and exact handoff before considering each phase complete.

**Do not mark M0–M5 as fully delivered based on this partial source implementation.**

## Post-merge installer safety correction

After code PR #7 merge (main 56bb425), the source candidate remained 0.23.0, while GitHub published stable v0.22.0 was confirmed with 13 assets and a commit tag. The streamed default installer previously targeted absent 0.23.0 assets, a prospective fresh-install regression. The follow-on source fix introduces PUBLISHED_RELEASE = 0.22.0, preserves the explicit published pin in install.sh, and tests its relation to VERSION. It **does not** make the new 0.23.0 MCP features available to a user who installs the stable v0.22.0 release. Only a separately reviewed and published 0.23.0 release plus an explicit stable-pin update can do that. Never silently fall back from a failed signature/digest check; a valid pin must be changed in reviewed source.

## Final repository publication record (2026-10-31)

- **Merged:** [docs PR #6](https://github.com/vanishs/boxweave/pull/6) at main commit 29e5dd1567c722780a7df30bf46505ad98a17de0; reconciled D0–D2 source implementation status to 0.22.0.
- **Merged:** [implementation PR #7](https://github.com/vanishs/boxweave/pull/7) at main commit 56bb425ca221614d677d7692fbf35ba1fa4167f3; only partial M0 app tool-discovery and M2 redacted debug summary, not full M0–M5. GitHub Actions final PR verification [41188468493](https://github.com/vanishs/boxweave/actions/runs/41188468493) succeeded.
- **Merged:** [installer safety PR #8](https://github.com/vanishs/boxweave/pull/8) at main commit 69282160a61f1be67b904c0a9a246851985ca46b; default streamed installs remain pinned to verified published v0.22.0 via PUBLISHED_RELEASE, while source VERSION is unreleased 0.23.0. GitHub Actions [41188709106](https://github.com/vanishs/boxweave/actions/runs/41188709106) succeeded.
- **Production/deployment:** not performed. No installed Containarium Platform or agent-box MCP, target Box, SSH, route or operator privilege was modified or live tested. New source APIs do not appear in the published v0.22.0 release.
- **Remaining status:** M1 trust-policy and router NOT IMPLEMENTED; M2 auto-sleep NOT IMPLEMENTED; M3 per-Box agent-box read NOT IMPLEMENTED; M4 controlled writes/exec NOT IMPLEMENTED; M5 selected workflows/GUI still NOT IMPLEMENTED. M0 runtime capability verification incomplete. Environment E0–E4 remain unimplemented. Full owner request is therefore **NOT COMPLETED**, despite the reviewed partial source merge.

**Future developer decision:** The immediate gating work is authenticated per-tool policy for the separate direct Agent gateway (M1), followed by an approved actual installed Provider/agent-box capability probe (M0/M3). Do not widen tool access before actor-to-Box scope and negative authorization tests exist. A separately reviewed 0.23.0 release/publication is required to distribute these merged source features; update the stable install pin only after verifying its published assets. All commits above were repository-only.
