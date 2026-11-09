# BoxWeave v1 — contributing and quality requirements

This repository is deliberately a staged, security-sensitive Go implementation. Follow [AGENTS.md](AGENTS.md), the [active phase plan](docs/process/IMPLEMENTATION_PLAN.md), and [quality gates](docs/process/QUALITY_GATES.md) rather than the historical `main` branch workflow.

## Before any commit

1. Work on an isolated topic branch; do not commit directly to `v1`, force-push or publish an unfinished stage. Avoid mixing S1 fixes with S2–S5 work.
2. Run `make hooks` **once per clone** to install this repository's versioned Git `pre-commit` and `pre-push` hooks.
3. Fix Go code with `gofmt -w` (or `make fmt`), then run `make fmt-check` and `make verify`; inspect `git diff --check` and `git diff --cached --check`.
4. Stage only intended files. The hook checks the **staged Go source**, including any file whose worktree was fixed but index was not.
5. Use descriptive commit messages (`feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`) and include original issue ID or required acceptance ID for risky native changes. Never commit tokens, private keys, credential receipts, native backup artifacts or guessed PASS evidence.

## Pull requests and merges

- Open a PR targeting `v1`; keep it **Draft** until the relevant phase evidence and every applicable CI job are green.
- The stable `quality-gate` status must be green on the **exact candidate commit**, with `v1 S1 source checks (non-release)` and any specialized workflows green as well. Red, skipped, cancelled, pending or stale results are **not** evidence of passing.
- Every risky change documents exact source/test binary SHA-256, permissions, native Project and scope, positive/negative observations, failure and rollback, unknown tasks, and append-only log updates.
- An independent verifier performs the installed-native tests required by the current S1–S5 phase; green source CI alone never proves real Incus or release acceptance. No release or merge while mandatory stage gates are PARTIAL/BLOCKED.
- GitHub's server-side merge blockade is effective **only after the owner enables the repository ruleset** described in [quality gates](docs/process/QUALITY_GATES.md). Local hooks cannot prevent GitHub API commits or `--no-verify`; do not claim universal commit blocking.
- Avoid `git push --force`; prefer reviewed topic commits and preserve the original UNKNOWN identifiers and source history.

## Supported baseline

Go 1.26.9 / `CGO_ENABLED=1` and the pinned official MCP/Incus SDKs in `go.mod`. Use UTF-8, LF and final newline. `gofmt` is authoritative for all `*.go`. YAML uses two spaces, Markdown/JSON/Python/shell follow `.editorconfig`; keep line endings/whitespace clean. Do not rename product names, change public MCP count or widen restricted service credentials to make a check green.
