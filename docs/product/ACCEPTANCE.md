# Acceptance criteria — behavioral authority

Phase6 current disposition:0.9.0's broad completion claim missed AC003/009 empty-target checking, AC010 binding lifecycle and the selected Web setup entry. All are fixed in0.9.1 with focused regressions, actual browser/client execution and independent-copy migration. User SRC-086 resolves the Web scope decision by requiring implementation. Normative criteria below are retained; [comparison and evidence](../verification/COMPARISON_REPORT.md), not historical pass-count prose, determine acceptance. No known reviewed selected P0/P1 remains open; deployment/platform limits remain explicit.

All criteria are the selected behavioral baseline within the developer-approved basic scope (SRC-026). Current TASK-012/0130.9.0 dispositions for all18 criteria are in [the release report](../verification/TASK_012_013_RELEASE.md), with actual/fixture/human evidence and explicit unclaimed deployment extensions separated. Phase5 selected implementation/evidence is ready for developer acceptance, not universal platform certification or phase6 authorization. Each AC maps to [REQUIREMENTS](REQUIREMENTS.md); sources and priorities are maintained there. A conditional AC is not waived as passed: it is out of selected scope only after an explicit decision. Supported/unsupported states must remain distinguishable.

## AC-001 — Enrollment preserves intent (REQ-001)

Given a selected supported client profile, trusted public connection descriptor and existing authorized bootstrap access: enrollment registers only the selected agent and preserves unrelated configuration. Repeating the same setup is safe. With a conflicting entry and replacement declined, preserve it and report conflict; with replacement explicitly selected at generation time, execution uses that selection, makes recovery available and reports the effective profile. Missing input/noninteractive ambiguity fails before mutation. A Unicode/space-containing path must not alter the selected runtime or arguments. Verify real registration separately from agent GUI loading. Fixture plus real selected-profile run required later.

## AC-002 — Trust is a gate (REQ-002)

Given a pinned server and a selected workload: matching identities permit the relevant connection; missing or changed pin produces a narrow trust error without enrolling a new key, editing trust automatically or executing a workload command. A trusted managed reconciliation may resolve a reused workload IP only after rechecking the resource; it cannot replace client server trust. No private key/JWT text appears in output/export. Test host and workload mismatches independently; unavailable reconciliation ends as unresolved, never insecure success.

## AC-003 — Diagnose independently (REQ-003)

Given already-enrolled configuration, diagnosis does not enroll, install packages, regenerate keys, repair configuration or mutate remote resources. Verify a valid initialize/initialized exchange, bounded tool discovery and one actual successful read-only tool result. Malformed/empty result object, denied call, repeated pagination cursor, unsupported protocol or deadline produces a precise failed/incomplete category; a valid empty inventory is accepted. Report registration mismatch even when the connection passes. Unsupported registration inspection and manual GUI loading are explicit; required incomplete checks prevent overall passed. Compare before/after state; actual selected client runtime required for a support claim.

## AC-004 — Execute in the intended environment (REQ-004)

Given two environments with similar names or inherited UUIDs, an explicit target selection resolves the intended provider/backend/project/resource and verifies expected execution identity before a fixed harmless command. Changed identity, unresolved target or unauthorized override stops execution without falling back to another box. Read-only selection of an existing resource must not create/delete it. Later evidence includes exact selected identity, sanitized stdout and actual exit code; an internal IP alone cannot satisfy this criterion.

## AC-005 — Ambiguous create is not replayed (REQ-005)

Given an authorized absent target and a create whose response is lost after submission: retain operation intent, reconcile its original logical identity and issue no duplicate create, rename, force-recreate or deletion on timeout/restart. If a later observation finds the same created target, resume readiness observation. If a transient query says absent or the query fails, remain unknown rather than auto-resubmitting; replacement detection stops scope expansion. Prove the write count and persisted intent with a delayed-response fixture, then a separately scoped real provider run. Explicit new intent after operator reconciliation is a new authorized operation, not a retry hidden by the system.

## AC-006 — Readiness, deadlines and exits (REQ-006)

Given a resource reports Running while SSH is unavailable, report Running with connection readiness pending. One recorded phase deadline bounds all polls/resumes (default proposal 600 seconds per phase; configurable within approved limits); resume does not reset it. Within that deadline, identity and command exit 0 can establish the corresponding success; a command exit 7 is failure with exit 7. Transport interruption or missing receipt gives unknown/null command exit. Timeout reports remote completion/cancellation as unverified and requires resource reconciliation before a new mutation. Capture elapsed time and terminal observation in later tests; no numeric SLA is claimed here.

## AC-007 — Work can be found without replay (REQ-007)

Given a submitted operation with a receipt ID, disconnect the initiating client and reconnect/restart the observer: the operator can rediscover the same operation under the same authorized identity and see progress/result without re-execution. Command failure, timeout, output truncation and vanished worker have distinct observations; query success is not command success. A lost worker has unknown/null command exit. Reading another identity's protected receipt is denied. Bounded retained output and cleanup/cancellation limits are disclosed; receipt identity alone does not authorize access.

## AC-008 — Install/update is recoverable (REQ-008)

Given saved nondefault settings, identities, unrelated user config and persistent project/console data: a normal rerun preserves them unless explicitly overridden. Reject invalid config, concurrent installation, unsupported state or failed download/verification before replacing known-good managed components where staging permits. Inject interruption before and during publication: show incomplete state, known installed provenance and a documented repair/backup path; do not claim whole-host rollback. A successful repair verifies required components and removes the incomplete marker only after their checks pass. Evidence must distinguish mocked filesystem/service tests from isolated real install/update runs. No provider/Incus upgrade is implicit.

## AC-009 — Health means what it says (REQ-009)

Given reachable API returning 401, report reachability without authenticated success. Given a successful inactive oneshot with execution evidence and healthy required timers, do not flag failure solely for inactivity; missing/disabled required timer or pending update cannot be locally ready. Separately report provider handshake/read-only call, workload result, current versus historical evidence and unverified boot/confinement. Test starting, incomplete, failed and ready observations; exact new exit-code/UI encoding is deferred, but these categories must be machine-distinguishable.

## AC-010 — Explicit project binding (REQ-010)

Given a project and an explicitly selected existing environment, persist a user-approved binding with sufficient scoped identity to revalidate later. Reselecting shows intended project/environment; moving to another authorized environment requires explicit change. Stale/missing/replaced target stops execution and offers reconciliation. Removing a binding deletes neither project data nor environment. Two clients with conflicting binding edits receive a conflict instead of silent overwrite. Developer demand is confirmed in SRC-026; implementation and runtime validation remain pending.

## AC-011 — Basic console operations (REQ-011)

Within the approved basic scope, given an unauthenticated or expired session, deny protected access. A valid user can preview exact targets/actions, then confirm an authorized unchanged plan once. Stale, replayed, wrong-user or cross-origin/CSRF-invalid confirmation is rejected before provider mutation. Mixed batches retain independent failed/succeeded/unknown item results and survive observer disconnect without automatic replay. Browser responses never contain provider credentials; local credential reset invalidates old sessions. Test browser behavior plus a real authorized provider path separately; fixture HTTP success alone is not fleet acceptance.

## AC-012 — Basic Tenant authority and atomic policy (REQ-012)

Within the approved basic scope, renaming display metadata leaves provider membership and enforcement unchanged. Old/unknown capability, remote unsupported placement or stale Guard blocks a new grant/shared create; a supported authenticated contract enables only its declared operation. Preserve account/shared identities distinctly. For reviewed multi-policy edits with full revisions, either apply the complete change or none on conflict; preserve unrelated ingress/egress fields and verify readback. Interrupted submission is unknown pending reconciliation. A revoke stored during degraded enforcement remains pending, not applied. Concurrent edits, malformed lists and default-emitted fields need negative coverage. Configured, Guard-verified and packet-tested statuses remain separate. Existing-box reassignment is excluded unless separately approved.

## AC-013 — Basic port exposure (REQ-013)

Within the approved basic scope, given conflicting routes or a reused target IP, reject or refresh the preview before confirmation. No route is published without explicit exposure authorization and current target identity. Show protocol, ports, destination, intended audience and the real source-filter/TLS capabilities before the write. Validate from an actual intended client before claiming client connectivity; passthrough alone does not prove LAN-only access or TLS. Deleting a route must not delete its workload; ambiguous route writes remain unknown until reconciled. A declared public boundary requires separate explicit public-exposure approval.

## AC-014 — Basic prompt composition (REQ-014)

Within the approved basic scope, save/edit/reload/export a graph without losing cards/comments/context; reject conflicting versions rather than overwriting the other editor. Clipboard failure does not record successful copying. Export escapes untrusted Markdown and applies documented secret handling without asserting that a pattern detector finds all secrets. Relations, Tenant context and import/export do not run agents or infrastructure writes. Deleting a graph removes its own context, not referenced environments/policies. Persistence/revision/browser cases must be verified later.

## AC-015 — Basic ZCode GLM setup (REQ-015)

Within the approved basic scope, explicit provider/model choice affects only the intended agent/provider settings. Invalid combinations and missing credentials fail before partial configuration changes. Preserve unrelated settings and provide recovery for overwritten managed values. Public generated scripts/reports do not contain credentials; secret use stays in the selected runtime's authorized protected interface. Repeated setup is consistent with the effective chosen model/provider. Test the selected real agent separately from rendering/configuration fixtures; no inference-model quality or upstream service SLA is promised.

## AC-016 — Reviewed source and secret-safe evidence (REQ-016)

Given a generated setup artifact and a reviewed expected source/trust record, identify the exact immutable source and verify executable payload against the approved trust mechanism before running it. A mismatch/unverifiable payload must stop without installation side effects. The expected verification evidence must not be silently taken from the same untrusted payload as proof of authenticity. Reports identify what was verified and do not call commit pinning independent checksum/signature validation. Seed known synthetic secrets into input/error paths: normal reports/exports exclude them while preserving actionable categories. Unknown arbitrary command output is never promised universally secret-free. Exact signing/hash/distribution mechanism is a phase 3 decision after product approval.

## AC-017 — Evidence required for support (REQ-017)

For each developer-selected release profile, record actual client OS/runtime/agent/server/provider revisions and perform authorized connection, read-only discovery, scoped target identity and harmless command execution plus mismatch/timeout/rejection cases. State real stdout/exit and what was not tested. GUI recognition/guidance loading requires its own observation; fixtures alone cannot establish it. Installation, reboot/autostart, TCP isolation and effective confinement require separate evidence if claimed. Any unexecuted/unsupported required claim blocks that profile's release acceptance rather than being counted passed. Acceptance inventory and evidence locations are designed in phase 3, not fabricated now.

## AC-018 — Authority and protected targets (REQ-018)

Given a caller authorized for target A only, attempts to select another owner/backend/project or override host/user are rejected without operating on target B. Use safe synthetic or dedicated resources for denial tests. Review privileges actually granted by the provider; a forced command or hidden UI control is not authorization. Resource deletion and exposure require explicit scope and fresh target identity; ordinary observation/diagnosis must not silently grant privileges, alter credentials/SSH/firewall or touch provider core/unrelated resources. Shared-user scope requires real positive/negative enforcement evidence before release; personal scope still documents residual provider authority.


## Selected launch matrix and model boundary

AC-001, AC-003 and AC-017 must cover Windows-hosted Codex/native SSH, Codex/WSL SSH, ZCode/native SSH and ZCode/WSL SSH. Explicit WSL distro and client configuration precedence must be tested. WSL-hosted Agent CLIs and other platforms are deferred. ZCode model access is verified separately from MCP; AC-015 includes native-settings handoff where a stable automated contract is unavailable, with pending status until actual completion. No provider credential goes through Web. The exact GLM model/endpoint is recorded per tested version. AC-011 through AC-015 are release-required features, not optional deferrals; their basic operation boundaries are in SCOPE.
