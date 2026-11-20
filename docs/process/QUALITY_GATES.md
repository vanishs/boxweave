# v1 repository standards, commit gates and merge protection

**Scope:** The v1 development branch (and later v1 release PRs), not the existing legacy `main` workflows. These controls are intentionally separate from the real-native stage acceptance gate.

## What docs-only PRs should (and should not) trigger

**Required GitHub status remains `quality-gate` on every PR into `v1`.**
The active ruleset still enforces that exact job, strict freshness, normal PR
review and no bypass. **Do not add `paths-ignore` to the required PR workflow:**
an absent required check can leave the PR waiting forever.

For a PR changing **only Markdown/MDX/reStructuredText files**, the required
job verifies the change is documentation-only, checks **changed document bytes**
(UTF-8, LF, final newline and accidental whitespace), and runs small scope
regression suites. It **does not assess document meaning**, perform repo-wide
Markdown content lint, run Go builds, Go race tests, module operations or
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
or remove required human review. The repository-local hooks now follow the
**same distinction**: a docs-only staged commit checks its changed staged
document bytes and `git diff --cached --check` without Go/YAML checks; a
docs-only push of an entire branch against the tracked remote `v1` base
skips redundant `make verify`. **Mixed/unknown changes, missing remote refs,
force pushes and release branches use full verification.** Local pre-push
uses an available local remote-tracking ref and is advisory; the GitHub
PR's current authoritative base remains decisive. A pushed branch may be
fully checked locally if `origin/v1` is absent or stale, by design.
Source-changing PRs retain all original full-source checks. No release or
S4/S5 native signoff is inferred.

## Deterministic local checks

| Change | Check | Reject condition |
|---|---|---|
| Go `cmd/`, `internal/` and future tracked `*.go` | `gofmt`, `go vet ./...`, `go mod tidy -diff`, `go mod verify`, `go test -race -count=1 ./...`, `go build ./...` | Any mismatch/failure; disabled CGO builds explicitly rejected by S1 smoke |
| All tracked text files | `python3 scripts/check_format.py`, `.editorconfig`, `.gitattributes` | Not UTF-8/LF, missing final newline, disallowed tabs/trailing whitespace or invalid JSON; `go run scripts/check_yaml.go` additionally rejects invalid YAML syntax and duplicate mapping keys |
| Changed lines on a PR | `git diff --check <merge-base> HEAD` | Newly added whitespace errors or extra blank EOF lines, including historical Markdown hard-break exceptions |
| Shell files | `sh -n` on versioned hooks and tracked `*.sh` | Shell syntax failure |
| Git index before commit | `.githooks/pre-commit`, installed with `make hooks` | Pure-doc staged edits: selected-doc byte format and staged whitespace; mixed/code/config: original entire staged format and YAML checks |
| Before push | `.githooks/pre-push` | Pure-doc branch against tracked remote `v1`: skips redundant Go checks; mixed/code/config/release/uncertain changes run full `make verify` |
| GitHub PR | `.github/workflows/quality-gate.yml`: stable **job `quality-gate`**, lightweight for docs-only PRs | Any GitHub status other than SUCCESS for the candidate SHA; docs-only PRs check only changed document byte-format and scope; all other changes run full gates |

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

## Documentation-only CI fast path (2026-11-18)

**The protected `v1` branch still requires the same `quality-gate` status.** To avoid repeated Go/Race, two-architecture candidate builds and real Incus installation from a pure documentation commit, the development workflows use a conservative changed-file classifier (`scripts/ci_change_scope.py`, with temporary-Git regression tests).

- On `feat/v1-*` **pushes**, compare exactly the previous push SHA to the new HEAD; on **pull requests**, compare the entire PR against the `v1` merge base rather than only its last commit. Documentation means textual Markdown/MDX/RST files outside `.github/**`; executable/configuration files under `docs/**` still require full checks. Source-to-document renames are full.
- For **docs-only** changes, the required **`quality-gate` job still runs and must succeed**, checking repository text formatting, changed-line whitespace and CI scope unit tests. Go setup, YAML Go parser, vet, module checks, Race and builds are skipped. The candidate workflow keeps a tiny scope job but skips expensive source validation, both Linux builds, candidate manifest and artifact upload. No release is produced.
- S4-A installed Incus native regression is **push-path filtered** for its workflow, application/source and installer inputs. Docs-only updates no longer start it; `workflow_dispatch` still permits an exact-source manual run. Independently scoped B3 native matrices already use source-path triggers/manual dispatch.
- **Fail closed:** unknown/missing push bases, unavailable diffs, non-documentation changes, and manual dispatch (only on workflows that permit it) use full CI. Protected `v1` and `release/*` branches always run full checks. A change to any `.github/**` workflow/config or `scripts/**` is full, not docs-only.
- The release workflow's existing non-cancelling concurrency and exact-tag publication safeguards remain unchanged; a fast docs-only run does not authorize merge, RC publishing or replacing missing stage-specific native evidence. Changing code after a scoped PASS still requires relevant native and independent revalidation.

**Proof requirement for this optimization:** after the workflow/script changes themselves receive full CI, use an additional **README-only commit**. Confirm `CI_CHANGE_SCOPE=DOCS_ONLY full=false` and `CI_DOCS_ONLY_FAST_PATH=PASS` in the required quality job, `CI_DOCS_ONLY_CANDIDATE_SKIP=PASS` with build jobs skipped, and absence of a new S4-A native run for that commit. A green workflow name without its raw step evidence does not prove the fast path worked.

**Observed pre-final smoke:** On a real [README-only commit `207d26fe`](https://github.com/vanishs/boxweave/commit/207d26fe60bdfef01447deb260c3ef5f139bf2f5), the required [quality-gate job 44437401349](https://github.com/vanishs/boxweave/actions/runs/44437401349) printed `CI_CHANGE_SCOPE=DOCS_ONLY full=false` and `CI_DOCS_ONLY_FAST_PATH=PASS`, with 10/10 classifier tests and formatting PASS, and Go/Race steps skipped. The [candidate run 44437401314](https://github.com/vanishs/boxweave/actions/runs/44437401314) printed `CI_DOCS_ONLY_CANDIDATE_SKIP=PASS`; all expensive source/build/manifest jobs were skipped, with no S4-A run for this README-only SHA. The first workflow-source change correctly selected full CI. The dedicated release manifest negative test forbids adding `workflow_dispatch` to the release workflow; this safeguard was retained in follow-up CI commit `e1484207` and its [quality run 44437635591](https://github.com/vanishs/boxweave/actions/runs/44437635591) and [release source/build run 44437635599](https://github.com/vanishs/boxweave/actions/runs/44437635599) revalidated. This documentation update is itself another docs-only fast-path regression.

## Owner directive — Box-first development and functional acceptance (updated 2026-11-20 PT)

- Prefer a task-owned reusable development Box for implementation, rapid iteration and functional acceptance, with a nested Incus lab only when actual native behavior is needed. Conventional isolated local Git/worktree development and authorized VM/CI are equally valid fallbacks when a Box is unavailable or unnecessary. Follow the [development environment and Box lifecycle](../development/AGENT_WORKFLOW.md#development-environment-and-box-lifecycle) in both cases. An existing nested environment's AppArmor label (including unconfined or complain), UID/GID mapping, storage and kernel features are evidence/limitations, not by themselves a reason to stop ordinary isolated functional testing.
- Run focused checks first and the applicable local quality gate before pushing verified changes. Reuse native scenario code across Box and CI with distinct environment safety preflights; GitHub Actions supplies required final exact-head regression and independent evidence, not a replacement for the local edit/test loop. Do not force a Box-only workflow or weaken any required gate.
- A passing test in a Box counts for the functionality actually exercised, provided exact source, real versus simulated dependencies, original Operation IDs, independently observed postconditions and guest command exits are recorded. Do not require duplicate native CI merely because a functional test ran inside a Box. A mocked test does not establish real packet or native effects.
- On failure, investigate whether source behavior or nested capabilities caused it. Keep UNKNOWN intact, record raw safe evidence, continue unrelated tests; use a fresh permitted native runner/CI when nesting prevents the required effect or classification. Do not relabel unexplained failure as environment-only or PASS.
- Keep development/acceptance Boxes on PASS, FAIL, BLOCKED and interrupted work. Save code, docs, durable evidence and checkpoint before reporting completion; name the exact Box/UUID and ask for explicit user deletion approval. Only authorized ephemeral test fixture cleanup and CI runner teardown are separate. Do not condition a functional PASS on deleting the development workspace.
- Functional acceptance is not recursive-isolation/security certification. Do not change outer host AppArmor, elevate a guest to privileged, expose parent daemon sockets, alter shared Project permissions or weaken any production security boundary to pass a test. Security-specific gates and final release/merge quality and independent evidence remain mandatory. Disclose the particular environment and untested capabilities.

## Phase / publication gate beyond syntax

- S1 merge **satisfied**: exact source identity, installed native candidate, root-confined capability, signed native daemon transition/rollback, unchanged UNKNOWN journal without blind replay, independent real selected virtual block device acceptance, final stage audit, and append-only execution ledger. [Merge PR #115](https://github.com/vanishs/boxweave/pull/115), `d29be2561b233ff250630719fcf7244217e0b384`.
- `S1=PARTIAL`, `BLOCKED`, missing evidence, open required issue or uncertain native outcome **blocks promotion**, even if source-format CI is green. S2–S5 likewise require their own phase exit.
- Releases require an immutable source/tag/artifact SHA, candidate-native independent verifier and all public MCP/CLI acceptance. `quality-gate` alone is never a release proof.

**One-line developer entry:** `make hooks && make verify`. Review [CONTRIBUTING](../../CONTRIBUTING.md), [S1 plan](../development/phases/01_FOUNDATION.md), and [final S1 acceptance matrix](../verification/S1_FINAL_EXIT_AUDIT_2026-11-09.md).
