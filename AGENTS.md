# BoxWeave v1 — Agent instructions

**Task entry:** [dev/BOOTSTRAP.md](dev/BOOTSTRAP.md) is the canonical BDH startup for every BoxWeave task. This file owns stable authority, safety and engineering rules.

**Stable rules, not a status snapshot.** Read only what the task needs. Verify current branch/HEAD, relevant PR, actual running version and original Box/Operation IDs; do not infer today's state from old reports. Start with the [active plan](docs/process/IMPLEMENTATION_PLAN.md), [phase protocol](docs/development/PHASE_EXECUTION_PROTOCOL.md) and [current status](docs/STATUS.md) as relevant.

**Authority order:** latest explicit owner decision + [permanent product boundary](docs/decisions/PRODUCT_BOUNDARY.md) / [operational policy](docs/decisions/P0_OPERATIONAL_POLICIES.md) → [MCP catalog](docs/design/MCP_TOOLS.md) and [contracts](docs/design/MCP_TOOL_CONTRACTS.md) → active phase design → dated research. Confirm implemented capabilities from source and actual MCP discovery. **Current v1 source registers 23 Tools; verify actual installed discovery.**

## Non-negotiable boundaries

- **Owned resources only.** Official Go MCP/Incus SDKs; verify exact Box UUID/incarnation, Project, caller and native poststate. Name/IP/`UseProject` alone is insufficient. No general host shell/files, foreign Incus management or exposed host-admin socket. `box_exec` runs **inside a guest**.
- **Single management capability.** Each explicitly authorized independently revocable Actor manages owned resources. No admin/viewer roles or OAuth application-permission scopes. Check authorization for every request, including delayed reads. Token revocation or client disconnect does not undo dispatched effects.
- **Durable effects.** Persist caller idempotency key, intent, Tracking ID and separate BoxWeave/native Operation IDs before native dispatch. Preserve original receipts. Never blindly replay, adopt, erase or delete an `UNKNOWN` effect; independently reconcile it. [Operations](docs/design/OPERATIONS.md) · [manual recovery](docs/process/UNKNOWN_MANUAL_RECOVERY.md).
- **Fail-closed files and networking.** Native exclusive guest-file create is safe; unproven conditional replace/delete must return `UNSUPPORTED`, never fake compare-and-swap. Native owned peer/ACL/forward only, default deny, with IPv4/IPv6 safeguards and verified poststate. No unapproved daemon-global Scriptlet/ACL, public bind or custom proxy. [File limits](docs/decisions/PRODUCT_BOUNDARY.md) · [network](docs/design/NETWORK.md).
- **Small product.** Do not exceed the approved 23-tool catalog or add speculative subsystems, a guest-resident Agent scheduler, generic ACL/RBAC, Caddy, a Web/REST console, snapshots/backups, host automation or a second Incus transport without owner approval. Nested Incus depth and isolation depend on verified host/kernel/AppArmor/BPF capabilities.
- **Explicit real-host authority.** Tools and past pilot approval are not blanket permission. Obtain exact-target owner approval for physical/shared-host installation, public exposure, shared Incus upgrades, privileged changes or destructive actions; preserve foreign resources and existing `UNKNOWN`. Never force-push or co-install v1 with legacy `main` on its dedicated host.

## Deployment scope and evidence

Operator-managed HTTPS/Tunnel access is an existing product integration option on a suitable, independently authorized deployment. See the [installation guide](docs/getting-started/MANUAL_INSTALL.md#connect-a-remote-oauth-client). The MCP listener remains loopback-only; do not expose native administration interfaces or credentials. This product capability does not establish permission or readiness for a particular environment, or override the nested-test restriction below.

Record functional acceptance, environment capabilities, public reachability, authenticated client acceptance and isolation evidence separately. A limitation in one lane is not automatically a product failure in another. A local health check is not proof of packet reachability or remote authorization.

Host confinement characteristics and client-platform limitations are not standalone BoxWeave product-function acceptance criteria. Report the particular observed effect, the affected check, and the evidence required to resolve it; do not classify an unrelated functional check as failed merely because an environmental limitation exists. Distinguish a potential limitation from an actual refusal, failure or unavailable capability.  Continue independent authorized work while the affected check remains unresolved.

## Development Harness

For every BoxWeave task, follow the [canonical BDH startup entry](dev/BOOTSTRAP.md).
It defines task activation, proportional Solve/Evolve Loops, continuity and
learning; the [toolbox](dev/README.md) and [asset index](dev/INDEX.md) provide
the applicable capabilities. If this file was loaded via BOOTSTRAP, continue
that entry without restarting it. For an authorized task checkout, use its
[short task routes](dev/INDEX.md#short-task-routes) and actual local toolbox
in addition to canonical v1 rules. Prefer a matching existing registered
check before constructing a manual command; keep tiny questions lightweight.
Use a safe manual fallback when the matching route is unsuitable or absent,
and report the reason. Script First; no new permission, product boundary
or CI exemption is granted by BDH.

## Work, checks and handoff

1. Perform the bounded task: inspect → implement → test → authorized native/independent verification → fix and retest → evidence/handoff. A blocked safety case stays `BLOCKED`/`UNKNOWN`; unrelated safe work must continue within the authorized task. While a native Operation or CI job is pending, advance independent ready work and prepare subsequent checks without duplicating effects or competing for an active resource claim. Reconcile original receipts before dependent actions. End a multi-step task only when the requested scope is complete, no independent safe work remains, or a genuine resource/budget boundary is reached; state the remaining work and exact reason. A saved checkpoint or pending job alone is not a completion condition. For multi-slice phases apply [phase-wide continuation](dev/skills/task-resume/SKILL.md#phase-wide-continuation-candidate) after each material checkpoint; CONTINUE means execute the next safe required action. Apply the [Finish Gate](dev/skills/task-resume/SKILL.md#finish-gate-candidate) before ending; a checkpoint is not a handoff. Mock/source CI, earlier-version PASS and a green check alone are **not** installed-native or release acceptance.
2. **Pass the applicable format gate before committing; never commit unverified Go.** Run/verify `gofmt` on Go changes **before** commit, including through GitHub API (which skips local Hooks). If unable to format/verify, report that limitation instead of committing or claiming PASS. Follow [CONTRIBUTING](CONTRIBUTING.md) and [quality gates](docs/process/QUALITY_GATES.md) for docs-only fast paths, full code checks and required `quality-gate`.
3. Use a topic branch and PR for `v1`; no bypass/force-push. Preserve exact candidate SHA and applicable phase gates. Do not equate RC publication, a `VERSION` string or unit tests with S5 release approval. [Versioning](docs/process/VERSIONING.md).
4. Record material decisions, failures, original identifiers, tested binaries/commits and honest limitations **without secrets** in the [execution log](docs/development/EXECUTION_LOG.md). Code, contracts and repository documentation are in English; reply in the user's language. Prefer Box-first isolated work; preserve development Boxes and UNKNOWN evidence until explicit owner deletion approval. Nested functional PASS is not recursive isolation proof. Owner clarification (2026-11-20): an existing authorized disposable trusted nested test Box reporting AppArmor `unconfined` is an evidence limitation, not by itself a veto on benign functional testing or a supported inner Incus installation attempt. Record positive functional evidence separately; investigate environment-related failures without weakening or bypassing runtime admission, host policy, parent isolation, UID/GID constraints or safety refusals.
