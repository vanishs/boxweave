# Historical quality-gate witnesses (November 2026)

Historical evidence moved from [the pinned original quality policy](https://github.com/vanishs/boxweave/blob/6831a5d61a551764b8cea655806188c49d7575e4/docs/process/QUALITY_GATES.md)
for navigation only. These are preserved observations, not current test
results or a prerequisite to running a source check. Current rules are in
[QUALITY_GATES](../../process/QUALITY_GATES.md).

### Actual negative merge-gate witness

A **non-Draft** [temporary PR #118](https://github.com/vanishs/boxweave/pull/118) targeting `v1` used a previously known bad-format code commit `a86eca8b306783bce00ecf375649ccf729759e88`. Its *fresh* GitHub Actions [run #42597345601](https://github.com/vanishs/boxweave/actions/runs/42597345601) produced **`quality-gate=failure`**, and GitHub reported **`mergeable_state=blocked`** for the non-Draft PR. **No merge was attempted.** The test PR was closed after confirming rejection; branch `test/v1-quality-gate-negative-20261109` remains a cleanup-only test ref because the available GitHub connector has no branch-delete operation.

**[Issue #117](https://github.com/vanishs/boxweave/issues/117): server-side rule accepted** on its original negative evidence. S1 later independently passed the [real native stage exit](../history/verification/S1_FINAL_EXIT_AUDIT_2026-11-09.md) and [PR #115](https://github.com/vanishs/boxweave/pull/115) was merged into protected `v1` as commit `d29be2561b233ff250630719fcf7244217e0b384`, without bypassing required quality checks. **No GitHub Release or production installation was authorized.**

## Documentation-only fast-path smoke

**Observed pre-final smoke:** On a real [README-only commit `207d26fe`](https://github.com/vanishs/boxweave/commit/207d26fe60bdfef01447deb260c3ef5f139bf2f5), the required [quality-gate job 44437401349](https://github.com/vanishs/boxweave/actions/runs/44437401349) printed `CI_CHANGE_SCOPE=DOCS_ONLY full=false` and `CI_DOCS_ONLY_FAST_PATH=PASS`, with 10/10 classifier tests and formatting PASS, and Go/Race steps skipped. The [candidate run 44437401314](https://github.com/vanishs/boxweave/actions/runs/44437401314) printed `CI_DOCS_ONLY_CANDIDATE_SKIP=PASS`; all expensive source/build/manifest jobs were skipped, with no S4-A run for this README-only SHA. The first workflow-source change correctly selected full CI. The dedicated release manifest negative test forbids adding `workflow_dispatch` to the release workflow; this safeguard was retained in follow-up CI commit `e1484207` and its [quality run 44437635591](https://github.com/vanishs/boxweave/actions/runs/44437635591) and [release source/build run 44437635599](https://github.com/vanishs/boxweave/actions/runs/44437635599) revalidated. This documentation update is itself another docs-only fast-path regression.
