---
name: continuous-learning
description: Turn verified development experience, failures, user corrections, repeated operations and handoff issues into maintainable project assets without inflating the repository.
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
