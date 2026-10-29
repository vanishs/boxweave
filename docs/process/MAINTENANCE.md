# Maintenance and knowledge workflow

This file owns change procedure. [DOCUMENT_AUTHORITY.json](DOCUMENT_AUTHORITY.json) assigns one owner per information type; [INDEX](../INDEX.md) is navigation, [PHASE_STATUS](PHASE_STATUS.md) owns gates, and [OPEN_RISKS](../verification/OPEN_RISKS.md) owns risk status. Do not copy their changing conclusions into other documents.

## Daily change path

1. Confirm explicit worktree/branch, clean or understood Git state, requested scope and current risks. Follow FINAL_HANDOFF's resource/credential boundaries.
2. Find the source of the request. Record a new SRC entry for a new decision or observation; identify existing REQ/AC and the affected selected boundary. Historical code is evidence, not authority.
3. Update REQ/AC or SCOPE only when the requested outcome changes. An architecture/product reversal requires developer disposition; ordinary implementation details do not need repeated permission.
4. Identify/update a TASK, dependencies and acceptance cases. Update the relevant ADR when a decision changes, including why and when to reconsider; mark replaced records historical rather than deleting them.
5. Make the smallest justified change in the assigned LXC. Prefer typed application services over adapter duplication. Do not consolidate different trust-boundary checks merely because they look similar.
6. Run the appropriate actual checks below. Record source/environment/commands/exit and evidence class. A timeout is unknown; inspect original state and never rerun a live driver blindly.
7. Update FEATURE_MATRIX for changed REQ/AC paths, SOURCE_REGISTER for evidence identity, OPEN_RISKS for risk state, and only the owning design/process document. Proposals belong in OPEN_QUESTIONS until decided, not in a duplicate risk register.
8. Check knowledge links/authority/IDs, review staged changes and secret boundaries, commit only when authorized, then write a concise HANDOFF with links to the owners. No push/deployment is implied.

## Executable checks

Portable, read-only navigation/ID/link check:

```sh
python scripts/check_knowledge.py
```

Prepared non-root Linux application checks (no dependency installation):

```sh
.venv/bin/python scripts/validate.py
.venv/bin/python scripts/validate.py --browser
```

The runner owns the routine lane: knowledge checks, unittest discovery, frontend build, optionally seven loopback **fixture** browser scripts. It creates a unique private .runtime/validation/<run>/ directory with logs/result and one600-second deadline for the whole invocation. Run one browser lane at a time: existing fixture scripts use owned fixed loopback ports; inspect conflicts rather than stopping unrelated listeners. A failed step stops the lane with its actual exit. Timeout leaves exit unknown and descendant cleanup unverified; inspect only this run's processes/ports before a fresh intent. It neither repairs dependencies nor calls live MCP, enrolls clients, mutates boxes/routes, deploys or runs inference.

The browser profile data defaults to [a committed nonexecuting fixture](../../examples/client-profiles.public.json), not private chat/state. Only a separately scoped acceptance may supply --profile-inputs to binding_browser_check.py. Fixture-generated scripts are not real enrollment packages. Every run's public downloads stay in that run directory; never overwrite older evidence archives.

For affected behavior use the relevant existing tests in FEATURE_MATRIX; passing default checks does not replace a required real integration test. Installer sandbox tests remain non-root. PowerShell setup changes require parse-only inspection before any separately authorized actual unprivileged enrollment. Existing live_* / package / phase6 / Windows drivers retain scoped regression/evidence value; they are deliberately excluded from the routine lane and may require original ignored state or external identities. Do not delete or mass-run them as cleanup.

No performance claim follows from test duration. Measure a named bottleneck before caching Guard/trust/identity observations or replacing the stack. Never trade fresh scope/authorization checks for fewer network calls without a reviewed contract and negative tests.

## Legacy knowledge review — manual, not an automation

Owner: maintainer. Review before each phase, before changing affected behavior, and weekly during active development; there is no scheduler, pull or automatic merge. LEGACY_SYNC alone owns the last reviewed SHA. Read-only commands include:

```sh
git -C <legacy-checkout> status --short
git -C <legacy-checkout> rev-parse HEAD
git -C <legacy-checkout> merge-base --is-ancestor <last-reviewed> <current>
git -C <legacy-checkout> log --oneline <last-reviewed>..<current> -- <affected-public-paths>
```

Exit0 from merge-base means linear ancestry;1 means inspect divergence, other nonzero means command failure. Do not describe an error as divergence. Do not fetch/pull/reset/clean the old checkout as a review shortcut. Compare relevant public rules/tests/docs, preserve WIP and ignored archives unread, then record source SHA/files, impact and accepted/rejected/deferred knowledge in LEGACY_SYNC/SOURCE_REGISTER. Translate accepted behavior into independent v2 requirements/regressions; never merge/cherry-pick old implementation automatically.

## Evidence, authority and history

- Current version/schema: VERSION/code. Current deployment: freshly inspected installed manifest/health, not a source commit or old report.
- Current behavior/scope: REQUIREMENTS, ACCEPTANCE, SCOPE. Design: ARCHITECTURE and accepted ADRs. Current risks: OPEN_RISKS. Current gate: PHASE_STATUS.
- Historical reports, source hashes and human observations retain their original revision/time/class. Do not rehash them to make current bytes look like the old run.
- Before replacing a long current document, retain a clearly labeled snapshot in the same parent directory so old relative links still work. Record its original digest and the new owner; check all links.
- check_knowledge validates local links, single-owner routing, normative IDs/matrix coverage and cold-start reachability. It does not prove semantics, remote availability, historical hash validity or security properties. Human review and actual runtime evidence still matter.

A cold-start review follows AGENTS + INDEX only, locates goal/scope/run/design/tests/risks/legacy/status within three link hops, and checks the answers in their authoritative files. Test the routine commands in a public copy without old .runtime state where feasible. State whether this was a scripted/manual simulation or an independent human/Agent review; never fabricate a new reviewer.
