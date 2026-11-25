## Change and phase

- Related issue / decision:
- Current S1–S5 phase and scope:
- Source commit SHA and independently tested binary SHA-256:

## What was tested?

- [ ] `python3 scripts/check_format.py`, `go vet ./...`, `go mod tidy -diff`
- [ ] `go test -race -count=1 ./...` and GitHub `quality-gate` PASS on this SHA
- [ ] Exact candidate installed/native black-box or explicit **NOT REQUIRED / BLOCKED** with reason
- [ ] Negative security/concurrency/failure tests and independent native readback recorded
- [ ] Unknown tasks, old identities, backup/rollback/clean-up documented
- [ ] Relevant specs and append-only execution log updated

## Development Harness (all tasks; lightweight answers allowed)

- Work ref / workflow profile / environment (Box-first or reason for fallback):
- Task or checkpoint link (or N/A for a trivial task):
- Relevant acceptance, exact candidate and remaining independent gates:
- Learning disposition (no-new-asset with reason / candidate link / verified improvement with evidence):
- Next safe action and unresolved blocker, if any:

See [BDH entry](../dev/README.md). Existing quality policy determines applicable checks for docs-only vs source/native work; mark N/A with a reason rather than claiming an unrun test. Do not use this checklist to lower mandatory gates.

## Merge decision

- Stage exit: **PASS / PARTIAL / BLOCKED / UNKNOWN**
- Required reviewer/owner decisions:
- Reasons not to merge yet:

> A passing Go build or CI is not a substitute for installed-native evidence. Keep Draft and do not merge a PARTIAL mandatory phase. Repository-wide quality rules: [QUALITY_GATES](../docs/process/QUALITY_GATES.md).
