# Development asset index

Bootstrap registry, read directly by bwdev context. These paths are canonical references, not copies. Start with only the matching row; follow links when needed. Asset lifecycle and future metadata generation: [design](../docs/development/DEVELOPMENT_HARNESS.md). Status "initial" means usable guidance derived from current contracts, not experimentally proven Agent efficacy.

## Short task routes

- **Tiny question:** read the matching knowledge contract; no CLI required.
- **Run an existing source check:** see [registered verification](README.md#universal-entry).
  In an authorized Python checkout, `python3 dev/bwdev --json verify plan`
  lists the available checks. Prefer the matching registered `verify run`
  and `verify inspect` with a fresh evidence directory. If not applicable
  or unavailable, use a documented manual command and state why.
  Read the verification Skill and applicable quality/acceptance rules;
  do not load historical test reports merely to run a check.
- **Review improvement opportunities:** see [learning inbox](LEARNING_INBOX.md#deferred-opportunity-review-current-193-plan).
  Use its existing compact review command where available, or inspect exact
  records through GitHub without a checkout. No automatic selection.
- **Other material work:** use the [toolbox](README.md) and matching rows;
  read only the relevant contracts. Preserve source, authority and evidence.

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
| skill.harness-improver | Evolve Loop, deferred candidate, feedback and bounded improvement | [improver](skills/harness-improver/SKILL.md) | Initial |
| skill.continuous-learning | Failure, correction, stage end, reusable improvement | [continuous-learning](skills/continuous-learning/SKILL.md) | Initial |
| module.install | Edit installer internals | [local guide](../internal/install/AGENTS.md) | Initial |
| tool.transfer-plan | Bounded guest file transfer, MCP chunk upload/copy, payload assembly, SHA256, receipt journal, interruption recovery | [offline planner](transfer_plan.py), [Box skill](skills/box-development/SKILL.md) | Candidate; independent utility pending |
| tool.bwdev | Universal work entry, start/resume/status/finish, context, task, verify, learn, CI | [CLI](bwdev), [contract](../docs/development/BWDEV_DESIGN.md) | Initial |
| knowledge.profiles | Task-specific environment prerequisites | [profiles](profiles.json), [box skill](skills/box-development/SKILL.md) | Initial |
| tool.context-domains | Deterministic path associations | [domains](context_domains.json) | Initial |
| verification.eval | Harness behavior and comparative evaluation plan | [eval guide](evals/README.md), [cases](evals/cases.json) | Initial |
| tool.make | Format, Go/source checks | [Makefile](../Makefile) | Existing |
| tool.mcp-testkit | Candidate HTTP MCP tests | [test kit](../scripts/mcp_testkit.py) | Existing |
| tool.installed-lifecycle | Native lifecycle, only authorized clean test fixture | [script](../scripts/n2_installed_lifecycle.sh), [test scenarios](../scripts/n2_installed_lifecycle.py) | Existing |
| tool.ci-scope | Source vs docs CI classification | [classifier](../scripts/ci_change_scope.py), [quality policy](../docs/process/QUALITY_GATES.md) | Existing |
| verification.acceptance | Mandatory product tests | [acceptance](../docs/verification/ACCEPTANCE.md), [continuity](../docs/verification/CONTINUITY_ACCEPTANCE.md) | Existing |
| knowledge.reliable-evolve | Material feedback and candidate review | [v0.6 plan](../docs/development/BDH_V06_RELIABLE_EVOLVE_PLAN.md) | Candidate |
| knowledge.proactive-improvement | Evolve Loop residual friction, improvement execution and independent utility evaluation | [v0.7 plan](../docs/development/BDH_V07_PROACTIVE_IMPROVEMENT_PLAN.md), [investigation](../docs/development/BDH_V07_IMPROVEMENT_RESEARCH.md) | Candidate |
| knowledge.consolidation | Pre-release consolidation PR, runtime retirement and CI conversion | [plan](../docs/development/CONSOLIDATION_PLAN.md), [progress](../docs/development/CONSOLIDATION_PROGRESS.md) | Initial |
| evidence.h5 | Historical test environment and SHA pitfalls | [H5 delivery](../docs/development/PR188_H5_DELIVERY.md) | Historical |
| template.task | Start or replan task | [task card](templates/TASK.md) | Initial |
| template.checkpoint | Durable handoff | [checkpoint](templates/CHECKPOINT.md) | Initial |
| learning.inbox | Candidate improvements, disposition | [inbox](LEARNING_INBOX.md) | Initial |

For original GitHub CI checks, use the actual PR head and associated run/job metadata, not copied historical links. For BoxWeave MCP use actual tools/list and the [catalog](../docs/design/MCP_TOOLS.md); installed versions may differ. Registered script references do not authorize execution; inspect applicable prerequisites and effects. For unknown tasks, use the source and ordinary search. Extend the index only for assets with a useful discovery path. The CLI discovers additional local skill headers as Candidate; later generate the full registry from distributed metadata.
