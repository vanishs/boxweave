# Phase 1 handoff

Phase: 1 — understand legacy and seed knowledge. Date: 2026-10-28 America/Los_Angeles.

Result: phase 1 knowledge work complete. Git completion requires the parentless-root and clean-worktree checks described below; exact post-commit results are supplied with the final handoff.

Legacy: `refs/heads/main@4318da6ea4e4e45c55cf43b64acbdd2ede65fc43`. Last knowledge synchronization: same SHA.

New branch: `rewrite/v2`. Worktree: `C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2`.

Root commit locator: `git rev-list --max-parents=0 rewrite/v2`. Exact commit SHA is reported after commit in the user-facing handoff; a commit cannot contain its own final SHA without changing it. Use the locator to recover the immutable document root later.

## Findings

1. BoxWeave's personal agent/console role and Containarium's runtime role are separate; generic provider abstraction remains a proposal (SRC-001/003).
2. Enrollment, diagnostics, jobs and infrastructure writes have different execution/evidence semantics; no single exit 0 proves the entire chain (SRC-004/005/009).
3. Async create needs persisted intent, original-name reconciliation and no blind replay (B-005).
4. Tenant metadata, native identity, Guard and TCP reachability are independent evidence layers (SRC-007/010).
5. Prompt Graph persists context and copy/export behavior, without an execution engine (F-015).
6. Installer repair preserves settings and stages files, but offers no whole-host transaction/automatic rollback (SRC-002).
7. Historical nested hello-world passed with a patched installer; confinement, autostart and clean client platforms remain unverified (SRC-011/017).
8. Agent notes need chronology: old diagnostics defects are partly fixed; early nested status and pre-fix upstream regressions are not current truth (AGENT_NOTES_AUDIT).
9. Download-checksum prose and version labels drift across documents (RISK-003, SRC-013).
10. Uncommitted replacement-choice handling may be incomplete; preserved and not repaired in this phase (RISK-004, SRC-020).

## Deliverables and gaps

AGENTS.md; docs/INDEX.md; seven knowledge documents (overview, features, behavior, Agent audit, sources, legacy sync, questions); process master plan, phase-1 prompt, phase status and this handoff. Only the supplied governance originals are copied verbatim. Legacy knowledge is curated, not bulk-copied.

Coverage: representative code/test paths and major 32-commit timeline; selected ignored narrative reports. Gaps: full ignored archive audit, current application suite, installed provider capability, real client/GUI acceptance and effective isolation. These are explicit in RISK-001..013. No legacy product tests or live experiments ran, so no fresh runtime compatibility is claimed.

## Next phase

Stop after phase 1. Phase 2 needs a new authorization and developer product decisions; its output should define users/outcomes, keep/change/remove/new scope, REQ/AC and unresolved tradeoffs. Product acceptance is required before phase 3. Do not implement, choose a new stack, upgrade provider infrastructure or treat historical execution permissions as current authority.

## Verification receipt

- Read-only Platform MCP `list_containers({})`: successful result, four Running resources. No connect or mutation performed; no shell exit code applies to this MCP result.
- Initial `git status --porcelain=v1`, worktree/branch/version inventory and final legacy status/SHA/hash checks: preserved main at the recorded SHA, exactly the original two dirty/untracked entries, and identical SRC-020 SHA256. Legacy code and notes were not edited.
- Documentation scan: 13 Markdown files, 21 local links valid, all 22 referenced source IDs defined, no private-key/JWT pattern detected; command exit 0. This is a narrow pattern scan plus content review, not a universal secret detector.
- Supplied master/phase prompt copies: SHA256 equality checks passed, exit 0.
- `git rev-list --count main`: 32, exit 0. No application tests ran: this phase changes documents only and does not establish fresh legacy runtime behavior.
- Worktree creation first failed because the sandbox could not write `.git/worktrees`; read-only inspection confirmed no partial target or branch. Approved escalation then completed direct `git worktree add --orphan -b rewrite/v2 <target>`, exit 0. No fallback checkout or destructive switch was used.
- Sandbox Git initially rejected the elevated-owned worktree. Subsequent read-only checks use a command-scoped exact `safe.directory` entry; no global Git trust configuration was changed.
- Before committing, inspect staged filenames and run `git diff --cached --check`. After committing, verify `git rev-list --parents -n 1 HEAD` contains only the new SHA, `git ls-tree -r --name-only HEAD` contains only the 13 Markdown files, new status is clean, and legacy status/ref/hash remain unchanged. The final user-facing SHA is the evidence locator; no self-referential commit hash is embedded in its own root.
