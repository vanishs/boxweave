---
name: continuous-learning
description: Turn verified BoxWeave task experience, failures, user corrections, repeated operations and handoff issues into maintainable project assets without inflating the repository.
---

# Continuous learning

At task start discover existing assets via [index](../../INDEX.md). At material failure, correction, decision, successful setup, handoff or task completion:

1. Capture a sanitized observation, original source/evidence link and applicability. Distinguish fact, hypothesis and decision. Do not capture secrets, private dialogue or raw private reasoning.
2. Search existing docs/skills/tests/tools first. Prefer update, merge or retirement over duplication. Classify as knowledge, skill, pattern/anti-pattern, tool, test, fixture, template, decision, workflow or eval.
3. For repeatable deterministic actions, propose a small tool/test rather than longer prompting. For a bug, preserve an observable regression. For a user correction, distinguish lasting authorized policy from task-local preference.
4. Validate against current source and suitable authorized environment. A failed/cancelled task can yield a candidate; do not claim a verified fix without evidence. Record status and scope. New procedural advice may be initially useful without proven eval uplift; label it honestly.
5. Small improvements may be included in the task PR; larger ones enter [inbox](../../LEARNING_INBOX.md) or a linked issue with validation plan. Follow change class and existing CI/review gates in [design](../../../docs/development/DEVELOPMENT_HARNESS.md). Never weaken core constraints or acceptance to pass.
6. Record disposition in PR/checkpoint: no-new-asset (reason), candidate/deferred (link), or improvement (evidence). Periodically revalidate, de-duplicate or deprecate based on source drift and observed usage. No requirement to invent new assets.

No automated capture, promotion or scheduling is assumed. Use ordinary Git and GitHub today.

Optional bwdev v0.4: use learn capture, audit and review with a local history directory; explicit review references do not prove external authorization or evidence. Preserve the Markdown inbox as navigation, use one canonical record per candidate and reconcile before activation. See [CLI contract](../../../docs/development/BWDEV_DESIGN.md).

For v0.5 use learn next to distinguish unimplemented candidates from those with implementation evidence. On a bounded improvement run relevant tests and Eval, attach the implementation/evidence with the expected revision, and persist the reviewed candidate history in Git. Start discovers related candidates. For v0.6 use the structured [material feedback protocol](../../README.md#material-feedback-protocol-v06-candidate) and finish --work-history; a reason-only legacy disposition cannot establish closure. Do not auto-promote based on local tests.

All task types use the same learning judgment. Use work boundary for material work, including source development, learn observe for distinct cross-task feedback, defer with a reconsideration condition, and [improver](../harness-improver/SKILL.md) for bounded improvements. Use the manual remote entry when CLI is unavailable; mark pending if no authorized persistence.
