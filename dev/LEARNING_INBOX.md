# Learning inbox

Manual bootstrap queue and conventions. An active PR/issue may hold the canonical candidate instead of editing this file; add a link here only when useful for discovery after the task. Do not duplicate the full observation. At each material failure, correction, handoff or completion, check for reusable value. Record one disposition even if no asset is needed.

Candidate fields: stable ID, work ref and source/evidence link, bounded observation, cause (or hypothesis), affected versions/environment, proposed asset or existing asset to update, validation plan, risk/change class, status and next step. Dispositions: candidate, verified, active, deferred, stale, deprecated, rejected, no-new-asset. A verified historical fact is not a verified new procedure. Larger unrelated improvements are separate tasks; no need to block product delivery for them. Use [learning skill](skills/continuous-learning/SKILL.md).

## Initial seed from actual prior work

| ID | Observation / source | Disposition and next step |
| --- | --- | --- |
| BDH-L001 | [#188 H5](../docs/development/PR188_H5_DELIVERY.md): missing Git caused 13 Git-dependent errors; missing origin/v1 ref caused an earlier attempt to fail. | Candidate: environment preflight distinguishes Git/base-ref needs per test. Initial environment skill includes these checks; automation and fresh regression not yet verified. |
| BDH-L002 | Same H5 evidence: populated Box caused clean-state fixture failures; two external installer tests required a clean host. | Candidate: fixture capability tags and separate clean CI lane. Initial verification skill prevents treating fixture limits as PASS. Do not alter production admission. |
| BDH-L003 | Same H5 evidence: prior CI was bound to H4B parent; H5 required exact-head candidate and independent CI, with merge-result distinct. | Candidate: deterministic SHA/evidence binding check. Initial verification skill records the distinction; automation/eval pending. |

The original historical report owns actual commands, IDs and results. These observations are not claims about today's installed environment or final CI. Initial seed is intentionally small; mine additional history when a current task needs it.

## First bwdev implementation learning (PR #189)

| ID | Observation and evidence | Disposition |
| --- | --- | --- |
| BDH-L004 | Dedicated Box create dab2374c02b77d1d2e75deab58581a05 failed SAFE_NATIVE_PREFLIGHT_DENIED; the retained authorized lab supported a new isolated clone (a3859c60a98595358707d0d2ca517d3f). | Preserve refusal; recover/reuse a separate worktree or conventional fallback. No safety bypass. |
| BDH-L005 | A large argv source transfer was rejected at admission; exclusive per-file creation in the isolated clone worked. | Candidate: bound transfer size, read original receipt after uncertainty, prefer verified exclusive file transfer for new files. Do not infer an exact size limit from one refusal. |
| BDH-L006 | make invocation 5aaed2e8a6e98b6089779ebf1366bd9e exited 127. The follow-up 46197b71acc1db5a5e3b50b550ddcc49 observed absent Make, Go and origin/v1 while Python/Git were available. | Verified for that Box: optional capability-specific doctor blockers; use direct Python tests without installing tools. Base-ref regression included. |
| BDH-L007 | Earlier installed-lifecycle CI run 45547041460, Legacy job 138979037296: connection refused after test-induced service restart despite PID/UNKNOWN checks. | Candidate fixture timing diagnosis. Add bounded listener readiness observation, preserve original state/identity/authentication gates, rerun independent native CI. Offline regression is not installed acceptance. |
