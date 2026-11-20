# BoxWeave v1 — contributing and quality requirements

This repository is deliberately a staged, security-sensitive Go implementation. Follow [AGENTS.md](AGENTS.md), the [active phase plan](docs/process/IMPLEMENTATION_PLAN.md), and [quality gates](docs/process/QUALITY_GATES.md) rather than the historical `main` branch workflow.

## Development environment

Prefer an authorized reusable development Box when it is available and useful; otherwise work in an isolated conventional clone/worktree or approved VM/CI. Use identical source gates and the appropriate native evidence tier. See the [Box-first workflow and user-confirmed cleanup](docs/development/AGENT_WORKFLOW.md#development-environment-and-box-lifecycle). A successful development Box remains intact until the user explicitly approves deletion after saved code, docs and evidence are reported. Repository-only changes do not require provisioning a Box.

## Before any commit

1. Work on an isolated topic branch; do not commit directly to `v1`, force-push or publish an unfinished stage. Avoid mixing S1 fixes with S2–S5 work.
2. Run `make hooks` **once per clone** to install this repository's versioned Git `pre-commit` and `pre-push` hooks. Existing installations of the repository-scoped `.githooks` path pick up tracked hook changes after checkout; separate clones need their own installation.
3. For Go/source/config changes, fix Go code with `gofmt -w` (or `make fmt`), then run `make fmt-check` and `make verify`; inspect `git diff --check` and `git diff --cached --check`. Pure Markdown/MDX/RST documentation commits and pushes use the fast local hook and CI path described in [quality gates](docs/process/QUALITY_GATES.md): changed-document byte format is checked, not prose meaning or Go behavior. Any mixed/uncertain changes continue through full checks.
4. Stage only intended files. The hook checks **staged** changes, not just the worktree. The fast documentation path verifies selected staged file bytes; Go/config commits retain the complete staged-source validation.
5. Use descriptive commit messages (`feat:`, `fix:`, `test:`, `docs:`, `ci:`, `chore:`) and include original issue ID or required acceptance ID for risky native changes. Never commit tokens, private keys, credential receipts, native backup artifacts or guessed PASS evidence.

## Pull requests and merges

- Open a PR targeting `v1`; keep it **Draft** until the relevant phase evidence and every applicable CI job are green.
- The stable `quality-gate` status must be green on the **exact candidate commit**. For source/config-changing PRs, also require the applicable `v1 S1 source checks (non-release)` and specialized workflows to pass. **Docs-only PRs intentionally omit those unrelated workflows**, so a skipped/not-triggered S1/build workflow is not a failure of docs-only acceptance; the required `quality-gate` itself must still pass. Red, cancelled, pending or stale **required** checks never count as passing.
- Every risky change documents exact source/test binary SHA-256, permissions, native Project and scope, positive/negative observations, failure and rollback, unknown tasks, and append-only log updates.
- An independent verifier performs the installed-native tests required by the current S1–S5 phase; green source CI alone never proves real Incus or release acceptance. No release or merge while mandatory stage gates are PARTIAL/BLOCKED.
- GitHub's server-side `v1` merge gate **is active** (Ruleset #26842628, strict required `quality-gate`; failing non-Draft PR #118 blocked) as documented in [quality gates](docs/process/QUALITY_GATES.md). Local hooks still cannot prevent GitHub API commits or `--no-verify`; the server rule blocks protected-branch merges, not invalid commits on unprotected branches.
- Avoid `git push --force`; prefer reviewed topic commits and preserve the original UNKNOWN identifiers and source history.

## Supported baseline

Go 1.26.9 / `CGO_ENABLED=1` and the pinned official MCP/Incus SDKs in `go.mod`. Use UTF-8, LF and final newline. `gofmt` is authoritative for all `*.go`. YAML uses two spaces, Markdown/JSON/Python/shell follow `.editorconfig`; keep line endings/whitespace clean. Do not rename product names, change public MCP count or widen restricted service credentials to make a check green.
