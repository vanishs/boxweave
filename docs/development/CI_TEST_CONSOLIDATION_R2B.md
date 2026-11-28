# R2B: shared native CI package identity checks

Parent f3959dc34dff555d0bf04aba03a399bbfe6490c1, PR #201. Reused Box 73d2ecae-3932-46e6-b4c6-3feecf7fba15, exact clean parent.

## Changes

N2, S25 outer and Forward already use pr191_ci_apt_sources.py for optional source selection. Extend that existing helper with read-only signing-key and Incus 7.6 version lookup. Preserve the same first fingerprint and first matching repository package criteria; require producer command success as well as matching output. No additional APT operations, trust changes, package installation, or runtime host mutation. Original no-argument source-selection behavior and its disposable-runner guard remain; new modes do not write system sources. The original shell key downloads, package lists, repository configurations, update flags, bootstrap ordering, fixture boundaries, independent jobs and evidence checks stay unchanged.

Forward workflow previously omitted the shared helper from path filters. Add the helper/test to both existing event path lists and run its offline regression before the native script; no event or job removed. N2 and S25 already cover scripts/** and run this test. S25 inner remains standalone because its source and hash-bound transfer, proxy and nested package refresh differ; do not expand staged files or alter isolation to share a few lines. Keep distinct parent-directory checks and native bootstrap sequencing rather than creating a privileged generic preparation framework.

## Evidence

- Inventory operation c7773273d718b9e294f8f3129015a5be exit 0 identified three active outer consumers and separate nested refresh.
- Focused test operation 106d990a1595cb40fa25e4b9dce9fd59 exit 0, full stdout/stderr: 106 tests PASS (88 prior focused, 4 candidate, 4 APT, 5 S25 setup and 5 S25 refresh). APT tests include existing optional source checks and new first-key/first-package positive, wrong/absent values and producer failure controls. No real key download, package update or native bootstrap performed in the development Box.
- Shell syntax, six-file format, diff checks PASS. Staged code-only tree 75f5c2f41202e8c1aeceb5198cf8d25ef592756c, export 2de6946d4cd8b61b5bf1a3de902b9b60 exit 0, preserve file modes.

## R2 disposition

The two slices consolidate duplicated build/SQLite oracles across RC/stable and package identity checks across three native outer entrypoints. Existing transcript check and witness-map helpers are already shared; do not change producer receipt interfaces. R2A removes 49 workflow lines; R2B adds 5 to cover helper changes and calibrate Forward. Cumulative workflow count 30, lines 7692 (RC15 10241). Script count 215 (two new R2A files); R2B net +74 code/test/workflow lines, R2 cumulative +187 excluding docs. Savings are duplicate maintenance logic, not measured runner minutes or total lines in R2.

R2 bounded implementation complete, CI acceptance pending. Remaining exact native/arm64/stable proof is not local PASS; stable publication not invoked. R3 next: resolve five groups in R1 closeout by reviewing actual current evidence and minimally converting applicable missing witnesses before removing more old wrappers. R4 handles event and source-test duplication. No merge, release, host or runtime modification.
