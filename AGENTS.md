# BoxWeave contributor and AGENT operating rules

Start at [docs/INDEX.md](docs/INDEX.md), [current scope](docs/product/SCOPE.md), [current status](docs/process/PHASE_STATUS.md), [FINAL_HANDOFF](docs/process/FINAL_HANDOFF.md), and [OPEN_RISKS](docs/verification/OPEN_RISKS.md). Read [DOCUMENT_AUTHORITY.json](docs/process/DOCUMENT_AUTHORITY.json) before updating documentation. Code, comments, documentation, command help, machine output and Git commit messages must be **English**; conversational replies to the requester may use their language.

## Authorized 0.10+ remediation

- The 2026-10-29 owner request authorizes **small, independent commits directly to main**, without a PR, for the remediation R-00–R-09; do not force-push. Confirm the current main SHA before each commit and reject unexpected changes. Never use an unreviewed working tree or wipe unrelated work. Remote repository-only edits via the approved GitHub connection are permitted; lack of a checked-out/test-capable environment must be recorded rather than invented.
- Restore useful `install.sh` host bootstrap and `boxweave.sh` management experiences via a reviewed new design. Retain the approved four Windows-hosted Codex/ZCode native/WSL profiles; no macOS/Cursor/Claude Code scope expansion. Fresh installation/update/repair come first; **no automatic 0.5.1 data migration**.
- Start the version line at `0.10.0`. Treat `A.B.C` as: increment B for a completed new feature slice and reset C to zero; increment C for fixes of existing behavior. Align `VERSION`, `pyproject.toml`, `web/package.json`, and the root metadata in `web/package-lock.json` on each version change. Do not bump for an empty promise. Use a distinct, descriptive English commit for each coherent slice.

## Trust and execution

- Historical source revisions, including `archive/0.5.1`, are read-only behavioral evidence, not a merge source. Their installers, bootstrap assumptions and test claims must not be transplanted unexamined.
- Never modify real host, Incus, Provider, SSH/firewall, token/key trust, private credentials, external network routes or public TLS without separate **specific environment authorization**. Repo source code for these operations can be drafted, but not run against a real host under the repository-write authorization. Prefer safe preflight, explicit operator confirmation, idempotent owned state, isolated boundaries and rollback/repair instructions.
- Do not read or export private keys, signing secrets, JWT values or production DBs. Preserve original evidence, ignored directories, historical SHAs, installed services, and legacy branch. No implicit provider upgrade, SSH trust-on-first-use, tenant/policy mutation or live inference.
- Only conduct Linux installation/real browser/Provider tests in a specifically approved prepared non-root Linux environment. The standard validation lane is `python scripts/check_knowledge.py`; in the prepared LXC, `.venv/bin/python scripts/validate.py` and optionally `--browser`. These are not host/provider acceptance. Check executable help and syntax before integration tests.
- Maintain `REQ/AC -> ADR/TASK -> code/tests/evidence` traceability. Follow [MAINTENANCE](docs/process/MAINTENANCE.md). Update the authoritative scope/requirements if an approved outcome changes, and keep [historical comparison](docs/verification/ARCHIVE_051_TO_MAIN_AUDIT.md), [risks](docs/verification/OPEN_RISKS.md), source register and handoff aligned. Preserve old reports as historical evidence.
- For every slice record *what changed*, *what was actually tested* (command, environment, exit code and evidence), *what remains untested*, *blockers*, and next Agent's executable checks in [remediation verification queue](docs/verification/REMEDIATION_TEST_QUEUE.md). A static code review is **S**, historical documentation **D**, genuinely run tests **T**, and untested properties **U**. Never report source code, fixture or CI as a successful real-host install.

## User-facing documentation conventions

- Describe the current product as **BoxWeave**, using its `0.X.X` release version. Keep obsolete archived-release and numbered development-stage narratives out of current user-facing prose, especially README; retain historical evidence in the historical records.
- This is a narrative/branding convention, not a ban on the words `archive`, `phase` or `v2`. Keep accurate ZIP/archive terminology, compatible CLI options, upstream protocol/API versions, technical phase fields, existing paths/service identifiers and useful developer-history links. Never rename runtime identities or remove compatibility to satisfy a word blacklist.
- Use tables for supported OS/Agent/SSH combinations, installer flags, and management commands.
- Do not claim the streamed default installer works until reviewed release artifacts and a complete host/launcher bootstrap have been accepted with real tests.
