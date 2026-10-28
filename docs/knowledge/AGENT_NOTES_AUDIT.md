# Agent knowledge audit

Source identities/confidence: [register](SOURCE_REGISTER.md). Prior Agent notes are evidence; only the user's current phase authorization applies.

## Scan scope and limits

Tracked file inventory, root hidden-directory names, Git subjects, AGENTS.md, adapter instructions, documentation research/run reports and ignored Markdown filenames were scanned. No tracked `.github/`, `.claude/`, `.cursor/`, `.codex/` or CLAUDE.md was found in the root repository inventory. No CI workflow was found. Copied upstream projects under outputs contain their own CLAUDE.md/.github; they are third-party artifacts, not root BoxWeave governance.

Ignored outputs contains a substantial research archive: boxweave-next-stages knowledge-base/stage reports/reference bundles, provider source exports, PR review/fix/CI notes, and local fixture artifacts. Selected narrative files were read (SRC-016..019); raw logs, credentials, fixture AGENTS files, copied code and full duplicate bundles were not imported. This audit is representative, not exhaustive. Preserve ignored sources locally because their conclusions are not recoverable from H alone.

## Findings and dispositions

| Evidence | Useful knowledge retained | Superseded/conflicting claim and disposition |
| --- | --- | --- |
| Root AGENTS.md | Resource inspection, identity separation, actual exits, readiness deadlines, source-vs-live distinction | Keep operational safeguards, not every old experiment as a mandatory v2 gate; user-authorized phase 1 includes the document-only root commit |
| agents/README.md and adapter inventory, SRC-003/004 | Different configuration surfaces and manual Cursor guidance | Agent GUI loading remains independently unverified; inventory is not support certification |
| SRC-014 diagnostics research | Reproduce malformed result/pagination/deadline failures and report stage evidence | Earlier claims of no pagination/empty-result acceptance are stale for inspected current POSIX probe, which now paginates and validates content. Reject as a universal current defect; retain historical lesson |
| SRC-014 development readiness | Workspace binding, output continuation, recovery and scoped identities are candidate needs | Ranked proposals are not approved requirements; independent diagnostics already shipped, no CI assumed |
| SRC-015 research integration | Ordinary nested hello-world should remain simple; experimental idmaps/package audits do not belong in every product gate | Heading 'Future work, not implemented' contains an explicitly implemented selftest item; read item detail/code, not heading alone |
| SRC-016 ignored knowledge-base | Layer-specific identity, UUID clones, readiness races, unknown state and scope of old security evidence | Earlier 'G0 NOT EXECUTED' corrected to failed historical attempt; later reconciliation-v2 reports running/active sample. Neither is current 2026-10-28 state |
| SRC-017 boundary report | Interrupted local stream does not justify inventing an exit; preserve recovered evidence separately | Enforced AppArmor sample and absent dangerous devices do not prove complete device filtering; safety remains unverified |
| SRC-018 reviewer report -> SRC-019 fixes -> SRC-010 profile | A green old suite missed account/shared collisions, malformed-list partial requests and placement errors; preserve targeted regression knowledge | Do not carry old failures as proven present at the later pinned source. Fix report's open/unmerged PR snapshot is superseded by tracked profile's merged claim; no fresh upstream check was done |
| SRC-019 older downstream recovery | Real PostgreSQL/protocol fixture success differs from production ACL/TCP tests | Old fork/patch results do not certify current upstream or installed daemon; never apply historical patch automatically |

## Explicit contradictions to preserve

1. SRC-003 architecture says downloaded enrollment assets are checked against recorded hashes. Current README/INSTALL-UPDATE say assets are not checksummed and scripts fetch at pinned commit. Treat architecture wording as stale; v2 download-integrity policy remains RISK-003, not decided here.
2. Product VERSION is 0.4.3; Web package/adapter overview remain 0.3.0 and CLI table still mentions 0.4.0. Version labels are dated evidence, not synchronized runtime facts (SRC-013).
3. Historical inventories name boxes no longer in today's four-resource list. IP/name reuse makes old observations insufficient for selecting a target (SRC-011/016/021).
4. Dirty replacement prompt commentary claims an answer is baked in, but its return is discarded by run; mark inferred defect rather than trusting a comment (SRC-020, RISK-004).

Rejected inheritance: machine-specific fixed idmaps, old host access authorizations, recovery-copy lifecycle instructions, experiment stage names as rebuild gates, upstream README claims as installed truth, wholesale outputs import, and the old implementation's stack as a v2 decision.
