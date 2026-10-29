# Final handoff — start without chat history

This is the onboarding/runbook entry, not a second status/risk register. Read [PHASE_STATUS](PHASE_STATUS.md) for current authorization, [OPEN_RISKS](../verification/OPEN_RISKS.md) before any deployment, and [VERSION](../../VERSION) for product version. [MAINTENANCE](MAINTENANCE.md) defines future change procedure.

## First fifteen minutes

1. Confirm the rewrite root and branch explicitly. On the current Windows host the rewrite is C:/Users/vanis/.codex/visualizations/2026/10/28/01a18158-8e09-7373-b64c-800d96ecb72e/boxweave-rewrite-v2; tool default cwd may instead be the read-only legacy C:/Users/vanis/Downloads/boxweave. Run git status/branch/rev-parse with explicit workdir before editing. No private conversation is needed to identify these roles.
2. Read [PRODUCT](../product/PRODUCT.md), [SCOPE](../product/SCOPE.md), [REQUIREMENTS](../product/REQUIREMENTS.md) and [ACCEPTANCE](../product/ACCEPTANCE.md). This is a personal trusted-operator Agent remote-development console, not a team-RBAC or graph-execution platform.
3. Read [ARCHITECTURE](../design/ARCHITECTURE.md) and the [ADR index](../design/ADR/INDEX.md); use [FEATURE_MATRIX](../verification/FEATURE_MATRIX.md) to reach implementation/tests/evidence for a behavior.
4. Inspect existing BoxWeave MCP inventory. Assigned development box: boxweave-tenant-dev, source /home/boxweave-tenant-dev/rewrite-v2-phase4-40a600a. Verify whoami and hostname through its dynamic connect tool before work. The remote source copy may have no Git metadata; local rewrite Git plus exact synchronized hashes record provenance.
5. Run the prepared checks below. Read the failure report before changing anything; do not install dependencies or repeat a live mutation just to get a green result.

## Resource and trust boundaries

Existing app test box is boxweave-v2-test; independent nested backend is bw-v2-access-lab. Other old boxes/core/host settings are not repair targets. Deployment state must be freshly inspected; RISK-605 explains why a source patch is not proof the retained test service was upgraded. Last known private app root: /home/boxweave-v2-test/boxweave-v2-package-lab. Inspect selected installed.json/health fields without dumping credential configuration or state. Do not advertise a private loopback listener as a user-accessible mapped URL.

Use existing MCP tools, not a general host shell. The Ubuntu transport credential is restricted to its installed MCP launcher. Never read/export private keys, JWTs or sign-in files. Do not copy an outer/client identity into a nested layer. The operator-provided management-layer Ubuntu-to-container public reference is:

```text
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICIoY7yK5+VYtbzXBFgHguKKWJFTx98DSeDtmDTfS1zA containarium-mcp
```

This is a recorded public reference, not a fresh phase7 key revalidation. Before any authorized create, verify that host/layer's full managed public key through trusted read-only discovery, supply ssh_keys explicitly, and bind host/project/name/creation identity. Never omit it to obtain private material or substitute a fingerprint/another layer's key.

Inspect/reuse resources first. A create timeout is ambiguous: query the original logical name; no duplicate, rename, force recreation or deletion to bypass it. Running is not SSH-ready. Use one original phase deadline (normally600 seconds,15–30-second polls), verify actual identity/stdout/exit, and stop on unresolved trust failures. Never disable checking/edit known_hosts as a shortcut. A successful service/HTTP response is not authenticated MCP; prove initialize/initialized, discovery and read. Explicit destructive/public/privileged scope remains required; existing same-scope authorization is not requested twice.

## Prepared development and validation

The tested server-development baseline is non-root Ubuntu24.04, CPython3.12 and Node18 with the repository locks, plus existing Chromium/system libraries. These are versioned observations, not universal platform support. In a prepared assigned source directory:

```sh
.venv/bin/python scripts/check_knowledge.py
.venv/bin/python scripts/validate.py
.venv/bin/python scripts/validate.py --browser
```

No live identity or previous .runtime state is needed by the default fixture lane. It tests real HTTP/SQLite/browser behavior with explicit synthetic infrastructure. The browser scripts actually start the same python -m bwmini serve command on their owned loopback ports and stop their own processes. Logs/results are unique under .runtime; inspect them, do not add them wholesale to Git.

If the environment is not prepared, review the locks and [README development steps](../../README.md) before non-root venv/pip --require-hashes, npm ci or browser installation. Missing OS libraries are prerequisites, not permission for root/host package repair. No clean-host installation is claimed by reusing existing dependencies.

Production/package install/update/admin reset and backup/downgrade commands belong only in [INSTALL](../INSTALL.md). An existing configured app can be started with its reviewed config:

```sh
python -m bwmini serve --config /private/reviewed-runtime.json
python -m bwmini worker --config /private/reviewed-runtime.json
python -m bwmini diagnose --config /private/reviewed-runtime.json
```

The first two are persistent workloads and need the requested deployment/service scope; they are not part of docs-only review. Diagnose performs a real read without opening/migrating the app DB. Never invent a launcher, credential, backend, TLS or account to make these examples work. Current schema/recovery limits live in code/INSTALL; older binaries must refuse unsupported state.

## Where to continue

Use [IMPLEMENTATION_PLAN](IMPLEMENTATION_PLAN.md) for task dependencies, [SOURCE_REGISTER](../knowledge/SOURCE_REGISTER.md) for evidence identities and [LEGACY_SYNC](../knowledge/LEGACY_SYNC.md) for the review baseline. Follow the maintenance pipeline for new requests. No new feature, deployment, model call, automatic legacy sync or forever-complete claim is implied by this handoff.

Live acceptance drivers are retained for reproducibility, not universal startup commands. Read their original scoped report, inspect original resources/state, obtain any genuinely missing authorization and use a fresh explicit intent. Human GUI/model access reports, fixture runs, prior live network tests and new local checks are separate evidence. Refer to [phase6 comparison](../verification/COMPARISON_REPORT.md) and [phase7 review](../verification/PHASE_07_REVIEW.md) for exact findings.
