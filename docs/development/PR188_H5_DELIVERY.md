# PR #188 H5 integrated delivery and acceptance

Status: Box acceptance complete for the tested source and fixture scope; final exact-head independent CI required. Keep PR #188 draft until CI review and explicit merge authorization. This is not a release or physical Host sign-off.

## Integrated candidate scope

H1A/H1B unified read-only observation and maintenance planning; H2A normal Scoped drain/retirement; H2B bounded Legacy offline retirement; H2C verified archive purge and supported resume; H3A staged installer and OAuth continuation; H3B Doctor/Recover; H4A typed MCP refusal and original navigation; H4B capacity budgets and separately reviewed reclamation design.

H5 changes only tests, CI and documentation, not runtime code. Candidate runtime source tested in Box A: 41293f2451a935b5d0d5b79a4ad05c836f385eb9, pinned Go 1.26.9. Locally built amd64 H5 development artifact: SHA256 d6c53973cf613584f6ca79266756e5384fc564cae1228167f43d8ba2c18086d6; build-info version 1.0.0-dev.h5 and exact source commit. This is not a published artifact.

## Box evidence

| Acceptance | L0 original / observation |
| --- | --- |
| Exact source full format, YAML, shell, module verification/tidy, vet, full Go race, s1lab installation tests, full build, opt-in capacity model | b59d2e574ba25322790e84eb71fe0041, guest exit 0. All Go packages passed; state race 166.642s. 1024 SQLite/model budget and 1025 fail-closed native model passed. First attempt 15c7b26ff9353f4730f04c0d8a6c0a44 had no origin/v1 ref; corrected to exact PR base SHA. |
| Offline Python in populated Box B | 55cfaffeeeb5c59da56fe131a7d8aab4, guest exit 0; 173 tests and nine OAuth wrapper tests passed after isolating synthetic guided-install paths. Initial all-test attempt aede35a102f77b6d5b653748962e463c: 13 Git-dependent errors (Git missing), eight clean-state fixture failures, no production bypass. |
| Remaining Git classifier tests and exact-head amd64 build, non-CGO refusal | 7c9b3c2fafcd6d11ce9dc8a8b8295e18, exit 0: 13 classifier and eight local hook tests passed. Two unmodified external installer subprocess tests require a clean host; not counted as Box pass. Final clean-runner suite runs all tests. |
| Live candidate observation / absent recovery / no effect | 682d1a366c5057d8f8476f8d7711ef50, exit 0. Three healthy Doctor reads with H4B development binary (prior recorded SHA256 5ac5ef7409a39e078f77c32607cd7ac8bcf23eb3239d9fe435775dde8043b136) against retained actual RC14 Scoped B. Maintenance blocked by three inherited effective-unit drop-ins, absent issue refused without retry, binary and manifest hashes, UUIDs, autostart and both PIDs unchanged, four guests running, sampled Forward HTTP 200. This is live observation, not a new installed candidate or continuous packet witness. |
| Final CI binding and guided mock regression | 380a3bea731af3a12b812cb79b419bc3, exit 0: four source/guided tests passed. |

## Independent evidence from the immediate H4B parent

At exact PR head 41293f2451a935b5d0d5b79a4ad05c836f385eb9, GitHub workflow run conclusions are all SUCCESS:

| Workflow | Run |
| --- | --- |
| Required quality gate | [45534306321](https://github.com/vanishs/boxweave/actions/runs/45534306321) |
| S1 source checks | [45534306328](https://github.com/vanishs/boxweave/actions/runs/45534306328) |
| amd64/arm64 distribution | [45534306285](https://github.com/vanishs/boxweave/actions/runs/45534306285) |
| Installed Forward policy | [45534306329](https://github.com/vanishs/boxweave/actions/runs/45534306329) |
| Exact-head independent installed lifecycle | [45534306247](https://github.com/vanishs/boxweave/actions/runs/45534306247) |

All four installed jobs passed: Legacy offline 138937012248, Legacy 138937012475, Scoped 138937012508, Scoped retirement 138937012510. Required markers are evaluated by each lane itself, not inferred from grep command text in logs. These lanes cover actual clean install, reviewed outage and original recovery, normal Scoped Peer/Forward and partial effects, exact owned drain and native restoration, interrupted resume/purge, historical UNKNOWN archive preservation, and Legacy-to-Scoped fresh reinstall. Source/distribution previously used the PR checkout (merge-result SHA); H5 binds quality and candidate distribution to the exact PR head; separate S1 merge-result checks remain. H4B evidence is not final H5 CI.

## H5 independent submission gate

The required quality workflow now runs all offline Python test files and the H3A wrapper, full Go race and s1lab tests, bounded capacity fixtures and builds. The native lifecycle workflow covers all H1-H4 scripts and runtime paths and prints its exact source SHA; existing required markers remain. Distribution checks/builds/manifest all use the pinned PR head, while protected release-only gates and publish authorization are unchanged. Run all five workflows on the H5 head. Review each job conclusion and exact-source evidence before requesting merge approval.

## Deliberately retained limitations and decisions

- No physical L0 changes, installed controller replacement, merge, release or issue closure. Retained Boxes A/B remain available. The nested effective-unit drop-ins are not bypassed.
- H2B is not generic offline repair: earlier interrupted native phases, unproved historical Peer Root journal or partial Scoped topology are review-blocked. H2C is ordinary reviewed file erasure, not forensic sanitization.
- H4B retains address/identity/history holds; [reclamation proposal](PR188_H4B_RECLAMATION_DESIGN.md) requires separate owner decision and proof. No cap increase or live bridge resize.
- 1024 SQLite records and native model are accepted substitutes for local scale only; 1024 physical guests, external/LAN/IPv6 physical Forward, physical storage quota and browser OAuth remain distinct target-specific acceptance. Independent S2 and nested S2.5 release-only gates still required on the eventual release commit.
- No blanket local repair authority; incomplete proof and UNKNOWN retain claims. Candidate release must preserve matching uninstaller/installer and historical evidence.

Stage reports: [H0](PR188_H0_LAB_PREFLIGHT.md), [H1A](PR188_H1A_BOX_ACCEPTANCE.md), [H1B](PR188_H1B_BOX_ACCEPTANCE.md), [H2A](PR188_H2A_BOX_ACCEPTANCE.md), [H2B](PR188_H2B_BOX_ACCEPTANCE.md), [H2C](PR188_H2C_BOX_ACCEPTANCE.md), [H2 CI repairs](PR188_H2_CI_REPAIR.md), [H3A](PR188_H3A_BOX_ACCEPTANCE.md), [H3B](PR188_H3B_BOX_ACCEPTANCE.md), [H4A](PR188_H4A_BOX_ACCEPTANCE.md), [H4B](PR188_H4B_BOX_ACCEPTANCE.md).
