# P4 R2 workflow disposition

Baseline 17afed0fe6b39365e1d2aad2392b2126f98a7db3; 48 original workflow
files, 47 retained. Classification is based on checked-out trigger/job
conditions and current PR runs, not a claim of equivalent historical
assertions. The seven current workflows and stable workflows are not
retired. Historical branch references remain available in Git.

| Workflow | Declared events | Disposition |
| --- | --- | --- |
| quality-gate.yml | pull_request, push, workflow_dispatch | CURRENT |
| v1-forward-install-policy.yml | pull_request, push, workflow_dispatch | CURRENT |
| v1-n2-installed-lifecycle.yml | pull_request, push, workflow_dispatch | CURRENT |
| v1-n4-capacity-budgets.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-pr191-s25-scoped.yml | pull_request | CURRENT |
| v1-release.yml | pull_request, push | CURRENT |
| v1-s1-a071-block-disk.yml | push, workflow_dispatch | OPTIONAL / HISTORICAL REVIEW |
| v1-s1-check.yml | pull_request, push, workflow_dispatch | CURRENT |
| v1-s2-format-snapshot.yml | push | HISTORICAL BRANCH |
| v1-s2-list-boundary.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s2-native.yml | pull_request, push, workflow_dispatch | CURRENT |
| v1-s25-bh018-installed-candidate.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s25-bh018-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s25-bridgehead.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s25-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s3-file-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s3-file-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-a-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b1-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-bridge-ownership-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-controller-root-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-dedicated-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-fanout-fault-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-grant-fault-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-ndp-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-quiescent-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-scriptlet-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-scriptlet-parity-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b2-static-address-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-contention-upgrade-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-final-closure-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-final-gates-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-foreign-revoke-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-grant-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b3-stale-etag-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s4-b4-forward-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5a1-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5a2-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5b-integrated-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5b-peer-production.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5b-rc10-residual-native.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5b-rc10-specialized-recovery.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-s5b1-frozen-installed.yml | push, workflow_dispatch | HISTORICAL BRANCH |
| v1-stable-candidate.yml | workflow_dispatch | STABLE / #170 |
| v1-stable-release.yml | workflow_dispatch | STABLE / #170 |
| v1-unknown-create-repair.yml | push, workflow_dispatch | OPTIONAL / HISTORICAL REVIEW |

The sole removed workflow is v1-guided-upgrade.yml, paired with
scripts/guided_upgrade_ci.sh (sole caller). Original files at baseline
17afed0f; actual older run evidence remains in GitHub. Old positive product
upgrade, backup and downgrade requirements are obsolete in the fresh-only
candidate. Current Forward fresh, repeat-stage no-effect and incompatible
schema preservation remain in v1-forward-install-policy.yml (R1
46083563349 all three jobs PASS). Keep mixed test_guided_upgrade.py.

The current N2/S25/S2 PR paths now match, including all source, scripts,
installer/uninstaller and VERSION; docs-only remains excluded. Actual
same-repository PR number, head, merge event SHA, producer workflow/ref/job
and original run/attempt bind the crosswalk. Only historical S2 branch
non-PR events retain the old direct script route. Both historical S25
workflows restrict manual execution to their original branches. Their
old markers are not substitutes for current receipt-based proof.

Stable candidate and signed publication, protected v1 quality-gate,
RC source/dual arch/manifest/native S2 and S25 dependencies, 29 N2 lanes,
72 markers and 23 mapped producers remain unchanged. RC proof is modeled
locally, not executed. Generalized future PR scope is limited to existing
same-repository v1 native PR boundary, no fork or release dispatch.

Remaining historical rows: retain actual old branch workflows and optional
manual witnesses until per-assertion disposition is complete. A071 disk
and UNKNOWN Create manual routes need review; no deletion of their unique
coverage. S4/S5 source-specific/frozen witnesses remain historical; do not
execute them against a different source or infer current acceptance.
A04/A05/A06/A08 residual source depends on shared helpers, archive
recognition and old consumers; no further blanket deletion. H frozen,
C04 and final C01/C11/P5 gates remain.
