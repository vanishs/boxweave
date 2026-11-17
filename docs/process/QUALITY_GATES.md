# v1 repository standards, commit gates and merge protection

**Scope:** The v1 development branch (and later v1 release PRs), not the existing legacy `main` workflows. These controls are intentionally separate from the real-native stage acceptance gate.

## What docs-only PRs should (and should not) trigger

**Required GitHub status remains `quality-gate` on every PR into `v1`.**
The active ruleset still enforces that exact job, strict freshness, normal PR
review and no bypass. **Do not add `paths-ignore` to the required PR workflow:**
an absent required check can leave the PR waiting forever.

For a PR changing **only Markdown/MDX/reStructuredText files**, the required
job only checks that the change really is documentation-only and runs a small
classifier regression suite; it **does not assess document meaning**, perform
repo-wide Markdown lint, run Go builds, Go race tests, module operations or
native tests. The independent S1 source-check and two-arch candidate PR
workflows use path exclusions to avoid starting at all on these PRs.

The classifier is fail-closed. A mixed diff (e.g., `.go`, `.yml`, workflow
config, source/test scripts, images or binary assets), a missing merge base
or an unrecognized event **uses the full original checks**, never the docs
shortcut. Release-candidate `release/v1.*-rc.*` pushes always receive the
full original source/build/native/provenance pipeline. A docs-only push on
`v1` or `feat/v1-*` may still create one minimal candidate-workflow
change-scope job; its expensive source and dual-arch jobs are skipped.

This is a **CI trigger/cost policy**, not permission to merge incorrect docs
or remove required human review. Local hooks remain independently in force:
pre-commit still checks staged UTF-8/LF/whitespace and the pre-push hook still
runs `make verify` until separately changed. Source-changing PRs retain all
original full-source checks. No release or S4/S5 native signoff is inferred.

## Deterministic local checks

| Change | Check | Reject condition |
|---|---|---|
| Go `cmd/`, `internal/` and future tracked `*.go` | `gofmt`, `go vet ./...`, `go mod tidy -diff`, `go mod verify`, `go test -race -count=1 ./...`, `go build ./...` | Any mismatch/failure; disabled CGO builds explicitly rejected by S1 smoke |
| All tracked text files | `python3 scripts/check_format.py`, `.editorconfig`, `.gitattributes` | Not UTF-8/LF, missing final newline, disallowed tabs/trailing whitespace or invalid JSON; `go run scripts/check_yaml.go` additionally rejects invalid YAML syntax and duplicate mapping keys |
| Changed lines on a PR | `git diff --check <merge-base> HEAD` | Newly added whitespace errors or extra blank EOF lines, including historical Markdown hard-break exceptions |
| Shell files | `sh -n` on versioned hooks and tracked `*.sh` | Shell syntax failure |
| Git index before commit | `.githooks/pre-commit`, installed with `make hooks` | Staged blob gofmt mismatch, staged YAML syntax or duplicate-key error, or `git diff --cached --check` failure; an unstaged worktree fix does not bypass |
| Before push | `.githooks/pre-push` and `make verify` | Any source check failure |
| GitHub PR | `.github/workflows/quality-gate.yml`: stable **job `quality-gate`**, lightweight for docs-only PRs | Any GitHub status other than SUCCESS for the candidate SHA; source-changing PRs run full gates, while docs-only PRs have scope-only checks |

**Enforcement scope:** `make hooks` is repository-local opt-in. It does not protect against `git -c core.hooksPath=/dev/null commit`, `--no-verify`, GitHub API commits, alternate Git clients or direct pushes to unprotected branches. The server-side required-check rule on `v1` **now blocks merge/update of the protected branch**, not the creation of an invalid commit on another branch.

## Server-side v1 merge rule — ACTIVE and independently verified (2026-11-09)

**[Ruleset #26842628 — v1 required quality](https://github.com/vanishs/boxweave/rules/26842628)** was saved by the repository owner and independently read back through GitHub's ruleset API. The observed effective configuration is:

- **`enforcement=active`**, `target=branch`, exact included ref **`refs/heads/v1`**; no excluded branches.
- Required **pull request**: `required_approving_review_count=0` (single maintainer), `required_review_thread_resolution=true`. Allowed merge methods: merge, squash and rebase.
- Required check **`quality-gate`** from GitHub Actions (integration ID `15368`); **strict/up-to-date=true**, `do_not_enforce_on_create=false`. The check must pass on the candidate SHA.
- **No bypass actors** (`bypass_actors=[]`, `current_user_can_bypass=never`), no force-push (`non_fast_forward`) and no branch deletion (`deletion`).
- The unrelated historical ruleset [#26251692 — Basic protection](https://github.com/vanishs/boxweave/rules/26251692) remains `disabled`, targets `~ALL` when enabled, and **was not modified**. The new active rule protects only `v1`, not every repository branch.

### Actual negative merge-gate witness

A **non-Draft** [temporary PR #118](https://github.com/vanishs/boxweave/pull/118) targeting `v1` used a previously known bad-format code commit `a86eca8b306783bce00ecf375649ccf729759e88`. Its *fresh* GitHub Actions [run #42597345601](https://github.com/vanishs/boxweave/actions/runs/42597345601) produced **`quality-gate=failure`**, and GitHub reported **`mergeable_state=blocked`** for the non-Draft PR. **No merge was attempted.** The test PR was closed after confirming rejection; branch `test/v1-quality-gate-negative-20261109` remains a cleanup-only test ref because the available GitHub connector has no branch-delete operation.

**[Issue #117](https://github.com/vanishs/boxweave/issues/117): server-side rule accepted** on its original negative evidence. S1 later independently passed the [real native stage exit](../verification/S1_FINAL_EXIT_AUDIT_2026-11-09.md) and [PR #115](https://github.com/vanishs/boxweave/pull/115) was merged into protected `v1` as commit `d29be2561b233ff250630719fcf7244217e0b384`, without bypassing required quality checks. **No GitHub Release or production installation was authorized.**

A configuration reference is stored at [`v1-required-quality.example.json`](../../.github/rulesets/v1-required-quality.example.json). This reference is not an executable GitHub settings change and its name/review-toggle defaults may differ from the owner's live rule. GitHub's live rule is authoritative.

## Phase / publication gate beyond syntax

- S1 merge **satisfied**: exact source identity, installed native candidate, root-confined capability, signed native daemon transition/rollback, unchanged UNKNOWN journal without blind replay, independent real selected virtual block device acceptance, final stage audit, and append-only execution ledger. [Merge PR #115](https://github.com/vanishs/boxweave/pull/115), `d29be2561b233ff250630719fcf7244217e0b384`.
- `S1=PARTIAL`, `BLOCKED`, missing evidence, open required issue or uncertain native outcome **blocks promotion**, even if source-format CI is green. S2–S5 likewise require their own phase exit.
- Releases require an immutable source/tag/artifact SHA, candidate-native independent verifier and all public MCP/CLI acceptance. `quality-gate` alone is never a release proof.

**One-line developer entry:** `make hooks && make verify`. Review [CONTRIBUTING](../../CONTRIBUTING.md), [S1 plan](../development/phases/01_FOUNDATION.md), and [final S1 acceptance matrix](../verification/S1_FINAL_EXIT_AUDIT_2026-11-09.md).
