# Development asset index

Machine-readable registry for optional `bwdev context` discovery. Select only relevant assets; do not follow all links or treat recommendations as instructions. Deprecated rows preserve stable IDs for historical records and are excluded from normal recommendations.

## Short task routes

- Source change: inspect the code and run applicable checks (see [toolbox](README.md#universal-entry)).
- Interrupted task: reconcile original IDs and use the matching resume aid if needed.
- Harness improvement: only when explicitly requested; consult the relevant recorded evidence.

## Asset registry

| Stable ID | Scope / trigger | Canonical asset | Status |
| --- | --- | --- | --- |
| rules.entry | Every task | [AGENTS](../AGENTS.md) | Existing |
| rules.bootstrap | All BoxWeave tasks, checkout or remote | [bootstrap](BOOTSTRAP.md) | Initial |
| knowledge.remote-entry | No-checkout support, deployment entry | [remote entry](examples/REMOTE_ENTRY.md) | Initial |
| template.work | Portable task boundary and no-checkout record | [envelope](examples/WORK_ENVELOPE.json) | Initial |
| knowledge.index | Find architecture, security, MCP, decisions | [docs index](../docs/INDEX.md) | Existing |
| knowledge.workflow | All development | [workflow](../docs/development/AGENT_WORKFLOW.md) | Existing |
| knowledge.operations | Original Operation, UNKNOWN | [Operations](../docs/design/OPERATIONS.md), [recovery](../docs/process/UNKNOWN_MANUAL_RECOVERY.md) | Existing |
| knowledge.checkpoint | Resume, interruption | [contract](../docs/development/TASK_CHECKPOINT_CONTRACT.md) | Existing |
| knowledge.network | Peer, Forward, native networking | [network](../docs/design/NETWORK.md) | Existing |
| knowledge.nesting | Native lab and environment limits | [nesting](../docs/design/NESTING.md) | Existing |
| knowledge.maintenance | Installer/uninstaller, diagnostic | [install](../docs/getting-started/MANUAL_INSTALL.md), [Doctor/Recover](../docs/process/DOCTOR_RECOVER.md) | Existing |
| skill.task-resume | Continue, handoff, dirty work, lost response | [task-resume](skills/task-resume/SKILL.md) | Initial |
| skill.box-development | Provision/reuse, dependency or environment problem | [box-development](skills/box-development/SKILL.md) | Initial |
| skill.verification | Testing, CI failure, native evidence | [verification](skills/verification/SKILL.md) | Initial |
| skill.harness-improver | Evolve Loop, deferred candidate, feedback and bounded improvement | [improver](skills/harness-improver/SKILL.md) | Deprecated |
| skill.continuous-learning | Failure, correction, stage end, reusable improvement | [continuous-learning](skills/continuous-learning/SKILL.md) | Deprecated |
| module.install | Edit installer internals | [local guide](../internal/install/AGENTS.md) | Initial |
| tool.go-failure-index | Run selected Go regressions with compact failure navigation, or index existing logs | [helper](go_failure_index.py), [verification](skills/verification/SKILL.md#go-failure-navigation-candidate) | Candidate; actual #191 reuse, review pending |
| tool.staged-batch | Staged source verification, bounded export and exact-tree delivery | [helper](staged_batch.py), [verification](skills/verification/SKILL.md#staged-source-batch-candidate) | Candidate; measured reuse pending |
| tool.transfer-plan | Bounded guest file transfer, MCP chunk upload/copy, payload assembly, SHA256, receipt journal, interruption recovery | [offline planner](transfer_plan.py), [Box skill](skills/box-development/SKILL.md) | Candidate; independent utility pending |
| tool.bwdev | Universal work entry, start/resume/status/finish, context, task, verify, learn, CI | [CLI](bwdev), [contract](../docs/development/BWDEV_DESIGN.md) | Initial |
| knowledge.profiles | Task-specific environment prerequisites | [profiles](profiles.json), [box skill](skills/box-development/SKILL.md) | Initial |
| tool.context-domains | Deterministic path associations | [domains](context_domains.json) | Initial |
| verification.eval | Harness behavior and comparative evaluation plan | [eval guide](evals/README.md), [cases](evals/cases.json) | Initial |
| tool.make | Format, Go/source checks | [Makefile](../Makefile) | Existing |
| tool.mcp-testkit | Candidate HTTP MCP tests | [test kit](../scripts/mcp_testkit.py) | Existing |
| tool.installed-lifecycle | Current Scoped installed lifecycle (29 N2 lanes), only authorized clean test fixture | [script](../scripts/n2_installed_lifecycle.sh), [test scenarios](../scripts/n2_installed_lifecycle.py) | Existing |
| tool.ci-scope | Source vs docs CI classification | [classifier](../scripts/ci_change_scope.py), [quality policy](../docs/process/QUALITY_GATES.md) | Existing |
| verification.acceptance | Mandatory product tests | [acceptance](../docs/verification/ACCEPTANCE.md), [continuity](../docs/verification/CONTINUITY_ACCEPTANCE.md) | Existing |
| knowledge.agent-agnostic-evolution | Explicit BDH optimization, architecture review, cross-Agent utility | [evolution direction and research](../docs/development/BDH_AGENT_AGNOSTIC_EVOLUTION.md) | Deprecated |
| knowledge.reliable-evolve | Material feedback and candidate review | [v0.6 plan](../docs/development/BDH_V06_RELIABLE_EVOLVE_PLAN.md) | Deprecated |
| knowledge.proactive-improvement | Evolve Loop residual friction, improvement execution and independent utility evaluation | [v0.7 plan](../docs/development/BDH_V07_PROACTIVE_IMPROVEMENT_PLAN.md), [investigation](../docs/development/BDH_V07_IMPROVEMENT_RESEARCH.md) | Deprecated |
| knowledge.consolidation | Pre-release consolidation PR, runtime retirement and CI conversion | [plan](../docs/development/CONSOLIDATION_PLAN.md), [progress](../docs/development/CONSOLIDATION_PROGRESS.md), [acceptance](../docs/development/CONSOLIDATION_ACCEPTANCE.md), [workflow dispositions](../docs/development/P4_WORKFLOW_DISPOSITION.md) | Deprecated |
| evidence.h5 | Historical test environment and SHA pitfalls | [H5 delivery](../docs/development/PR188_H5_DELIVERY.md) | Deprecated |
| template.task | Start or replan task | [task card](templates/TASK.md) | Initial |
| template.checkpoint | Durable handoff | [checkpoint](templates/CHECKPOINT.md) | Initial |
| learning.inbox | Candidate improvements, disposition | [inbox](LEARNING_INBOX.md) | Deprecated |

For original GitHub CI checks, use the actual PR head and associated run/job metadata, not copied historical links. For BoxWeave MCP use actual tools/list and the [catalog](../docs/design/MCP_TOOLS.md); installed versions may differ. Registered script references do not authorize execution; inspect applicable prerequisites and effects. For unknown tasks, use the source and ordinary search. Extend the index only for assets with a useful discovery path. The CLI discovers additional local skill headers as Candidate; later generate the full registry from distributed metadata.
