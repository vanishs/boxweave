# v1 repository standards, commit gates and merge protection

**Scope:** The v1 development branch (and later v1 release PRs), not the existing legacy `main` workflows. These controls are intentionally separate from the real-native stage acceptance gate.

## Deterministic local checks

| Change | Check | Reject condition |
|---|---|---|
| Go `cmd/`, `internal/` and future tracked `*.go` | `gofmt`, `go vet ./...`, `go mod tidy -diff`, `go mod verify`, `go test -race -count=1 ./...`, `go build ./...` | Any mismatch/failure; disabled CGO builds explicitly rejected by S1 smoke |
| All tracked text files | `python3 scripts/check_format.py`, `.editorconfig`, `.gitattributes` | Not UTF-8/LF, missing final newline, disallowed tabs/trailing whitespace or invalid JSON; `go run scripts/check_yaml.go` additionally rejects invalid YAML syntax and duplicate mapping keys |
| Changed lines on a PR | `git diff --check <merge-base> HEAD` | Newly added whitespace errors or extra blank EOF lines, including historical Markdown hard-break exceptions |
| Shell files | `sh -n` on versioned hooks and tracked `*.sh` | Shell syntax failure |
| Git index before commit | `.githooks/pre-commit`, installed with `make hooks` | Staged blob gofmt mismatch, staged YAML syntax or duplicate-key error, or `git diff --cached --check` failure; an unstaged worktree fix does not bypass |
| Before push | `.githooks/pre-push` and `make verify` | Any source check failure |
| GitHub PR | `.github/workflows/quality-gate.yml`: stable **job `quality-gate`** | Any GitHub status other than SUCCESS for the candidate SHA; scheduled smoke and installed verification remain separate additional gates |

**Owner-side enablement and negative merge verification are tracked in [GitHub Issue #117](https://github.com/vanishs/boxweave/issues/117). Until that issue closes with direct GitHub ruleset readback, `quality-gate` PASS is advisory for merge protection.**

**Enforcement scope:** `make hooks` is repository-local opt-in. It does not protect against `git -c core.hooksPath=/dev/null commit`, `--no-verify`, GitHub API commits, alternate Git clients or a direct push to an unprotected branch. A failing workflow cannot erase an already-written commit; it only blocks merges when a server-side required-check ruleset is active. Do not promise impossible universal commit rejection on GitHub.com.

## Server-side v1 merge rule (OWNER ACTION REQUIRED)

At the time this document was created, GitHub returned **Ruleset #26251692 `Basic protection`: `enforcement=disabled`** and this connection had **read-only ruleset actions**. Thus the following settings are a required operator task, **not a claimed accomplished protection**.

Using repository admin permissions in GitHub:

1. Open [Repository rulesets](https://github.com/vanishs/boxweave/settings/rules) → **New ruleset** → **New branch ruleset**. Name it `v1 required source quality`; target the precise branch `v1` (not every experimental topic branch).
2. Set **Enforcement = Active**. Require a pull request before merge; require status checks. Select exact required check **`quality-gate`** from workflow `v1 required quality gate`. Require branches to be up to date before merging; block force-push and deletion.
3. Allow no bypass actors unless separately approved; prefer dismissing stale approvals on new commits. If this is a single-maintainer repository, do not require an unavailable second reviewer by default. Existing `main` protections are not changed by this task.
4. **Save and independently verify** the ruleset is `active`, matches `refs/heads/v1`, and a deliberately red test PR **cannot merge**. Then inspect a green candidate PR and verify it can proceed only after S1-specific acceptance. Record screenshot or ruleset ID/version in [execution log](../development/EXECUTION_LOG.md).

An illustrative review-only ruleset payload is [`v1-required-quality.example.json`](../../.github/rulesets/v1-required-quality.example.json). It is not automatically applied to GitHub. Do not treat `Draft` or a passing check alone as a merge prohibition without an active ruleset. Do not weaken a failing check or move the PR base to bypass the gate.

## Phase / publication gate beyond syntax

- Before S1 merge: exact source identity, actual accepted installed v1 binary, root-confined native capability, tested native daemon transition and rollback, UNKNOWN journal with no blind retry, independent acceptance, updated canonical phase document and append-only execution ledger.
- `S1=PARTIAL`, `BLOCKED`, missing evidence, open required issue or uncertain native outcome **blocks promotion**, even if source-format CI is green. S2–S5 likewise require their own phase exit.
- Releases require an immutable source/tag/artifact SHA, candidate-native independent verifier and all public MCP/CLI acceptance. `quality-gate` alone is never a release proof.

**One-line developer entry:** `make hooks && make verify`. Review [CONTRIBUTING](../../CONTRIBUTING.md), [S1 plan](../development/phases/01_FOUNDATION.md), and [latest S1 verification](../verification/S1_FOUNDATION_LATEST_2026-11-09.md).
