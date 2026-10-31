# Agent-led BoxWeave development and test workflow

**Status: active developer working practice, not a shipped product feature.** Owner decision SRC-107 (2026-10-30). Follow [AGENTS.md](../../AGENTS.md), [maintenance](MAINTENANCE.md), [test strategy](../verification/TEST_STRATEGY.md) and [current scope](../product/SCOPE.md). This document **does not grant authorization** to modify a particular host, Box, Provider, SSH identity, tenant, network, credential or deployment.

## Goal and actual environment

Keep Agent-led tests flexible while ensuring they are fast, attributable and reproducible. The usual live topology is:

```text
Outer BoxWeave (stable management and observation)
  └─ one managed test Box
      ├─ inner BoxWeave
      ├─ Containarium + Incus
      └─ inner nested test workload Box (when needed)
```

The outer and inner BoxWeave are **separate controllers and trust domains**. If the inner stack fails, prefer preserving the failing Box and investigating through the authorized outer controller rather than reinstalling or deleting the guest. Nested `security.nesting` setting is not proof that actual nested execution works.

| Test lane | Use for | Limit |
| --- | --- | --- |
| **Reused test Box** | Fix, upgrade, config, restart and regression against existing state | Cannot prove clean installation |
| **Genuinely clean test Box** | Fresh installation, first-time bootstrap, SSH/MCP enrollment and initial nesting | Must begin with verified clean OS/installer state |
| Future cloned baseline | Consistent preinstalled environment comparison | **NOT IMPLEMENTED** in BoxWeave; not a fresh install |

Do **not** create a new Box for every small edit. Do **not** reuse an old Box to declare that a fresh installer works.

## Execution modes

| Question | Auto within authorized scope | Plan-first approval |
| --- | --- | --- |
| Preparation | State goal, exact target and expected evidence briefly; proceed only within an **existing specific grant** | Present short test plan and seek approval before environment writes |
| Diagnostic steps | Agent may adapt read-only checks and already permitted actions | Agent may adapt steps inside the approved plan |
| Unforeseen destructive/privileged write or new target | Stop and request additional specific authorization | Amend plan and obtain additional approval |
| Missing/unclear permission or mode | Read-only repository analysis or plan draft only | Plan draft only |

Neither mode authorizes unapproved Provider/Incus/storage/network/SSH/identity changes, trust bypass, parent restart, destructive cleanup, forced repair or a general host shell. Explicit session authorization **can** include exact disposable workload create/start/stop/delete, but only in the approved identity and scope. A tool's technical access is not operator authorization. Do not ask for repeated approval of the **same already authorized safe step**; scope expansion always needs approval.

## The short development loop

1. **Pin what changed.** Record actual source commit SHA (or reviewed working-tree diff), change intent, exact acceptance criteria and nearest test. Identify any existing non-committed changes by other Agents; don't overwrite them.
2. **Select the smallest honest first check.** Documentation: source and link check. Domain code: focused unit/fixture. Web/client: affected browser or client registration fixture. Installer/SSH/Provider/nesting: focused mocks/preflight first, then relevant **authorized** real integration. Don't run all suites reflexively after each edit.
3. **Select the correct lane.** Reuse an owned test Box for rapid upgrade or regression; use a genuinely clean Box only when initial-state behavior is the claim. Confirm actual outer controller/project, exact test Box identity, installed BoxWeave version, **running** app/worker version, Provider/Incus version if observed, and nested guest identity if needed. Source checkout, release package, installed manifest and running process may differ.
4. **Reproduce before rewriting.** Run one small verification. When it fails, preserve the first failing step, actual exit and observed state; classify whether source, packaging, config, startup, MCP transport, Provider or nested execution failed. Do not immediately wipe/reinstall the Box to make an error disappear.
5. **Escalate only where evidence requires it.** With separate live authorization: check actual running app/worker, authenticated MCP initialize/discovery/read, then **if required** create the approved nested Box, verify exact target identity, run Python and observe `hello world` stdout **plus exit 0**. A green CLI response, MCP tool listing or HTTP 401 does **not** prove inner workload execution. Record cleanup/retention separately.
6. **Avoid unknown-result replay.** After a timeout or transport interruption on create/delete/restore, outcome is **UNKNOWN** until the *original* target and receipt are safely inspected. Do not issue another write based on guessing. Do not auto-reset parent nesting, re-enroll SSH, change operator config, stop unrelated processes or delete failure evidence.
7. **Close a small iteration.** State what's verified and what's not, add any executable follow-up to the [remediation test queue](../verification/REMEDIATION_TEST_QUEUE.md), then commit/push only in the separately authorized repository scope. A short evidence-bearing handoff is more useful than a giant transcript.

### Existing routine checks (select based on change)

In the approved prepared development environment with appropriate dependencies:

```sh
python scripts/check_knowledge.py
.venv/bin/python scripts/validate.py
# Web changes only, when warranted:
.venv/bin/python scripts/validate.py --browser
```

They are separate choices; the normal runner exercises fixtures, not an authenticated live Containarium/Incus test. Do not invent installed CLI flags: inspect current `--help` and deployed binary first. Existing live `selftest` is **mutating** (creates, stops, deletes and may attempt cleanup); never treat it as a harmless read-only probe or retry it on an uncertain outcome.

After an **authorized** update to the inner installation, verify both release manifest and running service version. Operator config belongs in `config.json`; `runtime.json` is generated. Some changes need an explicitly authorized worker restart. Do not hand-edit generated runtime config or restart an unrelated app to claim success.

### Which checks provide what proof?

| Change | First proof | Required extra proof before claiming actual deployment |
| --- | --- | --- |
| Docs | Knowledge/links and current source | Usually none |
| Pure state/domain/schema | Targeted tests; crash/unknown fixture | Real upgrade/migration if claimed |
| Web/client setup | Fixture, generated-script syntax, preservation | Actual client transport/registration if claimed |
| Host install/update | Package/installer no-side-effect preflight | Clean install/update on separate approved Box |
| Installed CLI/diagnosis contracts | Focused fixtures incl. negative and leak tests | Live reused-Box checkpoint (see next section) |
| MCP/lifecycle/SSH | Contract/error tests | Authenticated external operation on approved target |
| Nested Incus | Observed parent policy and scoped preflight | Actual nested guest create + verified Python result |

Do not confuse fixture passing, historical evidence, source-only review or CI completion with a current real host pass.

### Installed-surface slices: a live Box checkpoint is part of Done

**Owner decision, 2026-10-31:** a slice that changes an *installed surface* must pass a real-Box
checkpoint before it is reported Done, even when every fixture and unit test is green. Fixture
evidence is necessary but not sufficient for these slices, because their dominant failure mode is
integration drift (unit names, runuser/`XDG_RUNTIME_DIR` transactions, isolated-interpreter entry
paths, installed file layouts) that only exists on a real host.

A slice is **installed-surface** when any of the following holds:

- it changes the installed host CLI or a published host helper (`host_cli.py`, `online.py`,
  `client_setup.py`/`client_profiles.py`, `mcp_gate.py`, `nesting.py`) or its printed/exit contract;
- it changes app-side diagnosis or health claims (`distribution.health`, `deploy_check`, the
  generated client `--diagnose`) that operators rely on to decide host state;
- its acceptance criteria assert behavior of a real service boundary: systemd units, the
  root/app-user transaction, an authenticated MCP read, or the isolated `python3 -I` entry;
- it changes installer/publisher file or packaging module allowlists.

The checkpoint is deliberately **minimal** and reuses the same iteration-closing rules as any other
lane: one authorized reused test Box, one upgrade/install of a built package, then read-only
verification that exercises at least (a) the changed command's promised pass path and (b) one
definite-failure or explicit-incomplete path, with actual commands, exit codes and observed states
recorded in the [remediation test queue](../verification/REMEDIATION_TEST_QUEUE.md) under
PASS/FAIL/PARTIAL/UNKNOWN. The upgrade lane's own backup/rollback behavior applies. Do not claim
fresh-install, nested or cross-version behavior the checkpoint did not exercise; those remain
separately authorized lanes. If the owner defers the checkpoint, the slice stays explicitly
labelled "installed behavior unverified" and later slices built on its contract inherit that
unverified label.

## Minimal test session note

No new database, Web feature or formal test manager is required. Add a compact note in the existing verification queue, issue or handoff:

```text
Goal / success criterion:
Execution mode: auto-within-scope | plan-first
Authorization: scope reference, allowed exact targets/actions (no secrets)
Source: commit SHA or reviewed diff; installed version; running version
Lane: reused | genuinely-clean
Target: outer controller / owned test Box / inner app+Provider / nested child if relevant
Checks: named steps, actual commands, exit/result, deadline
Evidence class: S source | D historical documentation | T executed | U untested
Outcome: PASS | FAIL | PARTIAL | UNKNOWN, limited to supported claim
Resources: original retained | authorized cleanup independently verified | unknown
Next step / blocker:
```

- **PASS** only for the observed exact criterion, including exit/output/identity where relevant.
- **FAIL** for a confirmed failed criterion; **PARTIAL** for a subset completed; **UNKNOWN** for ambiguous remote effects, missing observation or interrupted execution.
- Sanitize outputs; don't paste credentials, private keys, JWTs, full raw SSH stderr, secret-bearing environment/config or private DB dumps into source reports.

Illustrative **non-executed** example:

```text
Goal: provider parser returns expected error classification
Mode: auto-within-scope (source unit tests only)
Source: <reviewed SHA>; deployed runtime unchanged
Lane: reused (no live Box mutation)
Checks: parser unit test exit 0; live nested hello not run
Evidence: T local unit test; U live Provider
Outcome: PARTIAL (unit behavior observed, live claim unverified)
Resources: unchanged
Next: separate approval before external MCP/Provider test
```

## Stop conditions and handoff

Stop mutations on exact-target mismatch, missing grant, a pending/unknown remote write, a new storage/SSH/network risk, unrecognized Provider capability, unverifiable parent/guest trust relationship or pressure to delete original failure evidence. Safe read-only diagnosis can continue within its separately authorized scope. Record resource identity and observed uncertainty rather than treating timeout as failure.

End each task with **changed source, exact tested revision/environment, verified evidence, remaining unknowns, original resource disposition and next smallest test**. Existing [AGENTS.md](../../AGENTS.md) source-control, secrets, maintenance and evidence conventions still govern.

**Related designs, NOT IMPLEMENTED:** [D0-D2 diagnostic reports/traces](../design/DIAGNOSTICS_PROPOSAL.md) and [E0-E4 snapshots, clones and environment fingerprints](../design/BOX_ENVIRONMENT_PROPOSAL.md). This document does not introduce a product Agent Test Session feature, snapshot API, automatic execution engine or scheduled test task.
