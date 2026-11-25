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
