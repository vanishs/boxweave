# bwdev: script-first development toolbox

Status: v0.2 candidate in PR #189; source-tree CLI, no runtime or service change.
Read [BDH design](DEVELOPMENT_HARNESS.md) for the approved full framework and
[developer entry](../../dev/README.md) for the currently usable subset.

## Decisions

- Box First (not Box Required): use an authorized reusable Box and isolated
  Git worktree when useful; Local/VM/CI remain supported.
- Script First: Python standard library and argparse; small Shell wrappers
  only where appropriate. Modify and test immediately, without a build or
  installation step. Evaluate a compiled tool only for a demonstrated
  performance, reliability, portability or deployment need. Product Go
  architecture is unchanged.
- Thin dispatcher and independent command modules: each module supplies a
  register/run interface. Reuse Git, existing Makefile, scripts, GitHub and
  MCP rather than duplicate them. No service, database, auto-update,
  scheduler, provider-specific model integration or third-party dependency
  is required. The checkout pins the tool version.
- Convention first, progressive disclosure, evidence first and small
  batches. Use JSON for structured output and Markdown for knowledge.
  Keep one canonical asset; discover rather than copy it. No arbitrary
  shell execution or automatic retries of a native mutation.
- External tools: experiment in authorized isolated Box, check license,
  versions, dependencies and outcomes before adoption; retain replaceable
  adapters and native alternatives. Evaluate with real tasks, not popularity.

## Engineering precedents and research

These are design inputs, not claims that their results transfer to BoxWeave.
Traditional engineering systems are long-established; agent studies and
newer tools are more recent, often benchmark-specific. No external code
or dependencies have been imported or tested in this increment.

| Source | Adopt or evaluate |
| --- | --- |
| [Chromium depot_tools](https://chromium.googlesource.com/chromium/tools/depot_tools) | Script toolbox, independent commands, Git wrappers; do not copy automatic updates |
| [Mozilla Mach](https://firefox-source-docs.mozilla.org/mach/index.html) | Lightweight Python dispatcher, command modules and shared conventions |
| [Android Repo](https://source.android.com/docs/setup/download/source-control-tools) | Reuse Git rather than replace it |
| [LLVM lit](https://llvm.org/docs/CommandGuide/lit.html) | Discover tests, focused reruns, retain mandatory full gates |
| [Bazel test encyclopedia](https://bazel.build/reference/test-encyclopedia) | Explicit prerequisites, isolation and reproducibility |
| [Google release engineering](https://sre.google/sre-book/release-engineering/) | Candidate identity, traceable evidence and controlled releases |
| [DORA small batches](https://dora.dev/capabilities/working-in-small-batches/) | Deliver each useful increment immediately |
| [pre-commit](https://pre-commit.com/), [Nox](https://nox.thea.codes/) | Register existing tools and composable test sessions; evaluate before adding dependencies |
| [Spec Kit](https://github.com/github/spec-kit), [OpenSpec](https://github.com/Fission-AI/OpenSpec), [Beads](https://github.com/gastownhall/beads) | Evaluate specifications, task dependencies and persistence only where existing Git/PR records are insufficient |
| [CLI Guidelines](https://clig.dev/) | Help, explicit failure codes, noninteractive and machine-readable results |
| [Parnas, modular decomposition (1972)](https://doi.org/10.1145/361598.361623) | Hide independent implementation decisions behind small modules |
| [Rothermel and Harrold, safe regression test selection (1997)](https://doi.org/10.1145/248233.248262) | Conservative selection; full independent acceptance remains authoritative |
| [SWE-agent](https://arxiv.org/abs/2405.15793), [Agentless](https://arxiv.org/abs/2407.01489) | Useful tool interfaces and bounded deterministic workflows can improve task execution |
| [SWE-bench](https://www.swebench.com/) | Frozen historical task inputs, independent tests and no answer leakage |
| [SkillsBench](https://arxiv.org/abs/2602.12670), [context-file evaluation](https://arxiv.org/abs/2602.11988) | Evaluate skill/context benefit rather than accumulate indiscriminately |
| [OpenAI harness engineering](https://openai.com/index/harness-engineering/), [Anthropic long-running harness](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents) | Repository navigation, explicit checkpoints, continuous improvement |

## v0.1 contract

Run from a normal source checkout: `python3 dev/bwdev --help` or executable
`dev/bwdev` where supported. Python 3.10+ and Git for source observation;
no Python packages, Make or Go needed merely to use the CLI. The dispatcher
resolves its checkout from the script location, not the current directory.
The global `--json` flag precedes the command. Exit 0 means only that the
stated observation/capture succeeded; 1 means blocked or review required;
2 means invalid input or observation error (argparse syntax errors use its
normal stderr usage output). No exit code is product acceptance.

- `doctor [--require git|go|make|base]`: actual tool availability/version,
  Git commit, branch, worktree, origin/v1 availability; optional prerequisites
  become blockers only when required. No installs, native checks or network
  fetch; native readiness remains UNKNOWN. Go presence is not proof of the
  pinned toolchain or test readiness.
- `context --path internal/install --query recovery`: read the existing
  asset index, discover local skill headers, find nearby AGENTS.md and their
  linked assets, rank by explicit path conventions and literal tokens.
  Default limit eight recommendations; `--all` enumerates registry entries.
  Unindexed skills are Candidate, stale/deprecated/rejected entries are
  not recommended, duplicate or missing references in parsed registry rows and invalid skill headers fail closed.
  This is a small deterministic index, not full semantic retrieval or a
  substitute for reading applicable rules. Skill headers currently accept
  the simple name/description format used by this repo; richer YAML is
  deferred until justified.
- `checkpoint capture FILE ...` / `checkpoint check FILE`: explicit
  version-1 JSON, compatible field meanings with the existing client-owned
  checkpoint contract. Capture uses exclusive file creation (no overwrite);
  check validates basic fields and compares observed Git SHA and clean/dirty
  state. Box identity, original operations, dirty/unknown state and changed
  source require separate review. Never auto-resume, replay, claim ownership,
  validate evidence authenticity or mark acceptance. A crash during capture
  can leave a partial file; it must fail validation. Write to a fresh path
  and persist an approved copy outside the Box. A filename or successful
  check is not a lock, remote sync or proof of clean state at a later time.

The primary entry, task templates, manual learning inbox and verification
skill remain usable without this CLI. Existing CI discovers
`scripts/test_bwdev.py` with the offline Python suite; `make test-bwdev`
is a convenience for environments with Make. Direct Python invocation is
the equivalent focused test. No runtime/native test result is inferred.

## Rolling capability increments

| Increment | Usable scope and gate |
| --- | --- |
| v0.1 (this PR) | Dispatcher, doctor, deterministic context, initial checkpoint capture/check; offline regressions, Box command demo, separate-process handoff simulation |
| v0.2 (implemented local subset) | Task DAG, ready work, immutable local revisions and checkpoint references; remote reconciliation and distributed claims remain manual |
| v0.3 | Verification registry/plan and exact-source evidence, focused reruns; never weaken full CI/native gates |
| v0.4 | Structured learning candidates and asset lifecycle validation, reuse existing Markdown/manual records |
| v0.5+ | Environment recipes, external adapters, controlled Agent evals based on demonstrated need |

Each increment is useful on normal product work immediately. Version labels
are capability milestones, not a release schedule. Fix small tool problems
in the task that discovers them; larger changes receive separate PRs.
Keep compatibility where practical, update help and tests together,
and record learning dispositions at task boundaries. Do not wait for the
entire roadmap or claim an Agent productivity gain without comparative evals.

## v0.2 task continuity

Use the optional task subcommands with explicit JSON spec and local history directory. IDs and dependencies must form a bounded acyclic graph. A dependent task becomes ready only after its prerequisites are RECORDED. This state means a caller recorded a result, not that evidence has been independently verified. Reopen dependencies only after downstream active/recorded states are reconciled. Each transition requires the expected revision and a note; optional checkpoint references use the existing checkpoint contract. New revisions use a flushed temporary file and exclusive hard-link publication on a supporting local filesystem. A conflict rejects without overwriting. Local revision protection is not a distributed lock, power-loss durability proof or authorization. If initialization is interrupted, inspect the incomplete directory rather than overwrite or retry blindly. Save histories outside the Box via authorized Git/PR; compare divergent copies manually.
