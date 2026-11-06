# Existing reacceptance tool capability audit

**Date:** 2026-11-05, owner timezone. **Source:** main `742eaf22a28da89191a47ca6dc0628a0590a5ab3`, v0.49.1. **Scope:** source/read-only command help and optional draft fixture tests, not a full reacceptance invocation or installed acceptance.

The owner asked whether an Agent-operated find/fix/independent verify/publish command can be achieved primarily with documentation. **Most orchestration already can. Publication remains a code-level blocker because the active workflows require the former signing/environment/Runner design.** See the [reviewable simplification proposal](REACCEPTANCE_SIMPLIFICATION_REVIEW.md). Existing gates are unchanged.

| Existing capability | Working entrypoint | Actual limit |
| --- | --- | --- |
| Source/runtime catalog and action accounting | `python scripts/reacceptance.py inventory --help` | Discovery is not executed behavioral coverage; installed catalogs need actual authorized access |
| Scenario planning | `python scripts/reacceptance.py plan --help` | Includes all 20 canonical IDs; missing graph remains explicitly blocked, not absent |
| Bounded installed reads | `python scripts/reacceptance.py installed-readonly --help` | Read-only positive/invalid probes; not every mutation/denial or native effect oracle |
| Owned nested lifecycle | `python scripts/reacceptance.py lifecycle --help` | Four steps plus native state/intent; resize, exec and cross-interface cases still need actual Agent execution |
| Exact private candidate install | `python scripts/reacceptance.py private-install --help` | Requires root-owned verified stage and exact test-Box grant; not generic host permission |
| Installed identity/doctor inspection | `python scripts/reacceptance.py installed-verifier --help` | Reports PARTIAL honestly; cannot establish all functional acceptance |
| Original UNKNOWN readback | `python scripts/reacceptance.py unknown-inspect --help` | Read-only original UUID/incarnation; never replays or unlocks |
| Original CI artifact retrieval | `python -m tools.reacceptance.candidate_fetch --help` | Verifies wrapper, safe extraction and complete asset bytes; no install or behavior acceptance |
| Role handoff | Actual Agent collaboration tools, with `agents.py` as an optional record | The Python handoff function does not spawn Agents; the live Agent can delegate directly |
| Official publication | `.github/workflows/ci.yml` workflow dispatch | Active signed/environment gate remains; docs alone cannot change it |

All nine tested help forms (including root help) returned exit 0 in the local Python 3.12 test venv. The guessed combined-launcher alias `candidate-fetch --help` returned exit 2 because that alias is not wired; the direct module above is the supported working form. This distinction is recorded rather than inventing a new command. Help checks execute no installs, Box mutations, release dispatches or fixes.

The read-only connected `boxweave_info` development guide still identifies itself as static v0.47.0 guidance, not live runtime evidence. Do not infer the current installed product or Containarium version from it. Available outer MCP tools and collaboration capabilities are usable within grants; they are not an installed inner SSH/HTTP catalog.

## Documentation work versus code work

| Work | Documentation sufficient? | Required treatment |
| --- | --- | --- |
| Make the Agent drive the existing find/fix/independent-verification loop | Yes for orchestration instructions | Retain real observations; manual supported steps are valid |
| Stop treating complete automation-driver development as a prerequisite to testing | Yes | Existing manual Agent execution remains honest and scoped |
| Preserve detailed automation/signing plans as optional knowledge | Yes as a proposed/default policy disposition | Historical evidence is preserved, not relabelled PASS |
| Remove current signature/environment checks from executable release workflow | No | Explicit policy confirmation, reviewed code/workflow diff, regressions and new exact candidate |
| Prove v0.49.1 product behavior or resolve #88 | No | Real independent Box execution; no documentation or fixture substitution |

No production safety features, credentials, stable pointer, historical UNKNOWN targets or current release checks were changed by this audit. The proposed helper is a data-only draft with `release_authorized=false`; its source/tests are retained as a patch, not installed or wired into active workflows.

## Draft calibration

The data-only helper draft passed 13 fixture tests, exit 0, after independent review found and reproduced an original-versus-derived receipt consistency defect. It now rejects changed original status/decision/actor and conflicting supplied hashes even when transport hashes are coherently recomputed. No result grants release authority. Draft YAML and embedded shell syntax are checked separately; these are not installed or live release evidence.
