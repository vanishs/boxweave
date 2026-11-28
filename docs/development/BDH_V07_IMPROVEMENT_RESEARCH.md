# BDH v0.7: investigation of the improvement execution gap

Status: source-backed investigation and proposed design inputs, not implementation or acceptance.
Investigated #191 at a1e73da0e600b3e8d22c0082fb66c4d5d0b66944 and refreshed
at paused parent 3494ae8a35c50c386686c9d9d675c1ae37e1b1b7. Canonical v1 control revision
9d07665dcfc5ae6345d6bfcd0d71de7b6510b51d. The owner paused parent
development and authorized this stacked planning PR. The earlier v0.6
implementation and independent L4 PASS remain valid within their stated scope.

## Observed behavior and evidence

BDH is a repository-side task harness, not the installed RC14 product.
v0.6 demonstrated actual ordinary-task activation, discovery, local structured
feedback, durable handoff, authorized Git reconciliation and later reuse.
It did not prove unattended evolution, independent benefit of every asset,
or autonomous end-to-end implementation of a new capability on each task.
See [v0.6 plan and evidence](BDH_V06_RELIABLE_EVOLVE_PLAN.md),
[independent review](https://github.com/vanishs/boxweave/pull/192#pullrequestreview-6054605212)
and [parent integration](https://github.com/vanishs/boxweave/pull/191#issuecomment-7157928272).

The retained developer Box 9d374e18-196a-493d-bf1f-2d582589f04d
holds relevant original evidence. Read-only Operation output retrieval, not
new testing, verified the following. SUCCEEDED transport status alone is
not proof of a successful guest command.

| Ordinary work | Original evidence | Learning disposition |
| --- | --- | --- |
| P0 closeout | 3d396135549add6a65311234da755600 plan/begin, fd191f42e78c3c2230bcf13b1b2f5a00 observe, aaa4b695248989d375ba6d03c6410731 boundary; all exit 0 | L012 revision 3; five asset feedback items; HANDOFF with FEEDBACK_RECORDED |
| P1a | f28d6c41407c92c3577a43c4b928d8c7 exclusive file, 27487bec36beb5c71213f07adb730edb compile, a412c14b59bd18d884e0d247fa11a88f observe, dc538ef24b8b818774ac77a4f6e2b40e feedback; Exec exits 0 | L014 revision 10; four asset feedback items; HANDOFF with FEEDBACK_RECORDED |
| P1b source slice | 6d119c32d5954c9200112a6823259cb5 reported exclusive file 6207 bytes; a613d026dc4e98e64cfb1d37720adb2e original exit 0; c6afa902cbd1d06c4b8a95904ebd1fa1 original exit 1; 2d88e9945cde2ac0e16de275a1a6e52a original exit 0 | L014 revision 11 proactively avoids rejected inline transfer; initial exact-association review failure, then append-only explicit correction and FEEDBACK_RECORDED |

See [parent progress](CONSOLIDATION_PROGRESS.md), actual [L012 rev 3](../../dev/learning/BDH-L012/00000003.json),
[L014 rev 10](../../dev/learning/BDH-L014/00000010.json),
[L014 rev 11](../../dev/learning/BDH-L014/00000011.json),
[P0 history](CONSOLIDATION_P0_CLOSE_WORK_HISTORY/00000002.json),
[P1a history](CONSOLIDATION_P1A_WORK_HISTORY/00000002.json) and
[P1b corrected history](CONSOLIDATION_P1B_SOURCE_WORK_HISTORY/00000003.json).
P1b final Operation prints the original local revision-11 SHA256
8e15d5f92d391254620ce57c89e482411b2324d21b215286b26c9546c4c7883c
and final history SHA256
1862b959358d9d25698a125e603174ba5f0ec04ca7fd18e42d1440a229eb5744.
These are original reported hashes, not a new local-to-Git byte comparison.

L014 at revision 11 has seven observation entries, one recurred and six helped,
and a historical implementation attachment to the bounded transfer Skill.
The entries include investigation tasks as well as ordinary development;
distinct work labels alone are not independent recurrence. The newest ordinary
task avoided the known admission refusal proactively, so it is positive
evidence for the old Skill, not proof that the current selection algorithm
caused a failed task. L012 and L013 also have historical attachments but
remain CANDIDATE. The record history, not revision number, measures evidence.
The original owner-authorized deferral of a complete transfer helper and
focus on parent P0/P1 work are independent explanations for not implementing
it. Do not infer hidden Agent intentions or treat that choice as a defect.

## Source findings and hypotheses

| Finding | Evidence at paused parent | Consequence |
| --- | --- | --- |
| F1: attachment masks residual opportunity | [evolution.select](../../dev/harness/evolution.py) checks implementation before deferred state; new-observation flag is only computed for DEFERRED, with no implementation-relative evidence window | An attached candidate with later residual friction still returns review linked implementation; even a deferred attached candidate cannot express an independently actionable new gap through next_action. This is a confirmed routing limitation, not proof of causation in every task. |
| F2: selection is advisory, not a work commitment | [learning.run](../../dev/harness/learning.py) next, [work.plan](../../dev/harness/work.py), [harness-improver](../../dev/skills/harness-improver/SKILL.md) | The Agent may choose reuse, fix, flag or defer; there is no durable selected improvement objective, acceptance criterion or required reevaluation at a later boundary. Existing task/checkpoint machinery can be reused. |
| F3: usefulness and remaining toil are conflated | [evolution.validate](../../dev/harness/evolution.py) observation fields and work_count, [feedback review](../../dev/harness/feedback.py) | helped can coexist with repeated manual work or cost; a distinct work ref is not authenticated recurrence; no explicit residual gap or comparative measurement. |
| F4: completion and reconsideration are separate | [work.run](../../dev/harness/work.py), [workflow.summary](../../dev/harness/workflow.py), [feedback review](../../dev/harness/feedback.py) | FEEDBACK_RECORDED validates supplied associations, not whether a recurring opportunity was triaged or improved. Existing HANDOFF is not finish. Do not make unrelated improvements block product completion. |
| F5: feedback ceremony can itself be toil | P1b original negative and corrected operations above | Exact matching correctly prevents false claims, but duplicate manually transcribed observation text increases repair work. Prefer a safe reference-derived envelope helper or reuse existing structured fields; do not weaken matching, history or v0.6 protections. |

Historical design already intended bounded improvements and deferred
rediscovery ([U3/U4](BDH_DUAL_LOOP_DELIVERY_PLAN.md)).
[U4 evidence](BDH_DUAL_LOOP_ACCEPTANCE.md) records actual L014 rediscovery
and implementation in #189, with tests; it was a harness-development
exercise with the same Agent across distinct work refs. It is a useful
internal precedent, not a fresh independent ordinary-task autonomous
implementation witness. v0.7 should close that distinction rather than
replace working UA, Solve Loop or v0.6 feedback.

## External precedents, checked 2026-11-25

These sources describe practical workflows or empirical research; their
results cannot be transplanted as BoxWeave efficacy evidence. No external
source code, model provider or service dependency is required.

| Reference | Adaptation to BDH |
| --- | --- |
| [Warp outer improver](https://claude.com/blog/how-warp-builds-self-improving-agents-on-claude), Anthropic practitioner account, 2026-08-26; [Warp review example](https://www.warp.dev/blog/how-to-build-a-cloud-software-factory-self-improving-code-review) | Separate normal task execution and focused feedback-driven improvement, with reviewed file-based updates. Use task boundaries instead of adopting their scheduled cloud runner. This is a reported implementation, not an independent quantified causal evaluation. |
| [KCS Knowledge Domain Analysis](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_Practices_Guide/030/040/010/060) and [KDA measures](https://library.serviceinnovation.org/KCS/KCS_v6/KCS_v6_KDA_Guide/040_KDA_Measures) | Aggregate patterns across tasks and distinguish knowledge utility from removing persistent root causes; evaluate actual resolution and cost. |
| [Google SRE eliminating toil](https://sre.google/workbook/eliminating-toil/) and [postmortem culture](https://sre.google/workbook/postmortem-culture/) | Convert recurring manual effort into bounded, tracked corrective work; distinguish its value from how much documentation is written. No fixed time allocation imported. |
| [OpenAI Performance Factory](https://openai.com/index/how-openai-performance-factory-keeps-chatgpt-fast/) | Keep durable work state and next action across review, implementation and outcome verification; re-open when recovery is not demonstrated. Borrow lifecycle design, not cloud dispatch/subscription infrastructure. |
| [SkillsBench](https://arxiv.org/abs/2602.12670), [SWE-Skills-Bench](https://arxiv.org/abs/2603.15401) | More self-authored skills do not reliably improve performance; use matched baseline and independent outcome checks, not asset quotas or self-assessed success. |

## Decision and scope

Adopt an explicit improvement-execution part of the existing Evolve Loop.
Preserve existing candidate identity and append-only observations. Separate
reuse evidence, residual gap, improvement episode and actual validated
effect. A review-due signal is not execution authority. A small bounded
authorized task may be selected and completed by the same Agent; a larger
one remains an explicit durable handoff for a later authorized Agent.
No daemon, scheduler, extra required Agent, extra product MCP Tool,
database, semantic search service, automatic promotion or automatic merge.
Do not remove safety limits or weaken independent review. No claim that
every task needs a new asset; reuse/no-change is valid when supported.

The [delivery plan](BDH_V07_PROACTIVE_IMPROVEMENT_PLAN.md) owns implementation
and acceptance. Initial findings F1-F5 are not new candidate observations
fabricated on behalf of earlier Agents. Keep L014 revision 11 intact until
actual v0.7 work records a distinct new observation or reviewed disposition.
