# S2 native Exec-output GC: host-local review

**Scope:** Only an installed and operator-approved BoxWeave v1 host; no remote MCP unlock. Real seven-day normal cleanup is accepted, but installed privileged review and deliberate native lost DELETE acknowledgment still require independent validation.

When a prior logged Incus output DELETE has an uncertain result, BoxWeave writes REVIEW_REQUIRED and preserves its original operation, output evidence and Box coordination lock. It never retries the original native DELETE. This is separate from a missing file that can only be truthfully reported as unavailable.

An authorized operator uses the already installed root-local CLI with exact installed -db /var/lib/boxweave/ledger.sqlite, -project, -incus-socket, -service-user, -operation, -stream stdout or stderr, and a reviewer identity. Call the existing command `boxweave recover` with `-action gc-inspect` and explicit `-approve-local` first. Then use `-action gc-challenge` for a single-use five-minute reviewer/native-observation-bound challenge. After independently confirming original UUID and native log status, the separate `-action gc-release` requires `-challenge`, `-confirm-uuid`, a nonsecret `-reason` of 16–240 characters, and `-acknowledge-uncertain-native`. A wrong reviewer, changed original/native state, expired challenge, or replay is refused.

This release records REVIEWED_RELEASED and a reason digest; it **never** executes another DELETE, changes original guest exit or asserts expired. A present original file remains available; a missing file remains unavailable. The reviewed original is not silently requeued for automatic GC. The audit is append-only at the application layer, not protection against an all-powerful host root.

**Evidence:** [normal two-stream real Incus CI](https://github.com/vanishs/boxweave/actions/runs/42939755998/job/130258487415); code-and-unit negative review path accepted on the same source. The local installed root review and true ambiguous DELETE failure are **PARTIAL**, not an acceptable reason to mark all of S2 PASS. No old deployed v1 database migration is performed; this is the fresh-install schema.
