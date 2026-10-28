# Phase 4 concrete execution scope — approved

Prepared 2026-10-28 America/Los_Angeles after the user requested phase 4. This plan is a concrete application of TASK-001 through TASK-005 / EXP-001 and EXP-006, not authorization to start phase 5. The user confirmed this scope before container connection/development. Actual results, including missing live prerequisites, are in VERTICAL_SLICE.md.

## Resources and identity

Existing project candidate: backend dev, logical box boxweave-tenant-dev, actual name boxweave-tenant-dev-container, observed private IP 10.206.30.2, Ubuntu 24.04, 4 CPU/8 GB/40 GB. Fresh list/get succeeded; Running is not SSH readiness. Do not modify box1, box2, containarium-build or provider core containers.

After confirmation, use Platform MCP connect to verify `whoami && hostname` and actual exit code. Inspect files/Git state and resource capacity before selecting a fresh isolated directory under the verified user's home (candidate rewrite-v2-phase4-40a600a; never overwrite an existing directory). Existing legacy checkout and development files remain untouched. Use the same reviewed project LXC as the fixed harmless workload target only if its actual identity and supported MCP execution path are verified; no other workload is implicitly authorized.

## Expected changes

1. Develop new minimal source/tests and install dependencies into a project-local virtual environment/frontend directory inside this LXC. No APT/host package upgrades, system account creation or provider installation. Preserve existing repositories and WIP. Synchronize only reviewed new files back to the rewrite/v2 worktree; don't mirror the legacy dirty tree or use delete synchronization.
2. Implement typed target/project contracts, SQLite binding/revisions, intent/receipt recovery, allowlisted identity/hello execution, minimal authenticated Web/CLI and independent native-client diagnosis. Keep actual TASK/AC coverage honest if some interfaces cannot be verified.
3. Start only owned temporary test processes. Default HTTP backend binds container loopback. Browser fixture tests run inside the container. If the real Windows browser needs direct access, separately check reachability/conflicts and use a temporary private-address listener at 10.206.30.2:18640 only within this approved scope; no provider proxy/passthrough route, public bind, firewall change or persistent service. Do not claim client reachability until actually tested. Production HTTPS/Caddy deployment is not a prerequisite for loopback fixture tests and is not silently reported complete.
4. Use existing protected provider interfaces only through their supported runtime credential mechanism. Do not read/export private keys, JWTs or signing secrets, borrow the existing SSH MCP token for the Web worker, or repurpose the restricted Ubuntu transport key for shell/tunneling. Inspect prerequisite paths/permissions/version and configured launch interfaces without printing credentials. If an independent application credential/launch path is unavailable, mark the live worker step blocked and report the concrete prerequisite; don't mint/change credentials to make it pass.
5. Run real authenticated MCP initialize/initialized, tools/list and one read-only tools/call before calling the allowlisted identity/hello action. Verify expected user/hostname before fixed Python hello. Capture stdout/actual exit, operation identity and timestamps. A tool failure/timeout is not command exit 0. Test mismatch/no-dispatch, stale revision, crash/unknown/no-replay and receipt persistence with controlled fixtures plus a real path where available.

## Time limits and success

One recorded deadline per readiness, dependency-install and live verification phase, normally 600 seconds; retry connection at 15-30-second intervals within the same readiness deadline. Do not reset deadlines on retries. Development itself is not misreported as one 600-second installation. Preserve background state/log locations and actual exits; no competing installer or lock removal.

Minimum real success: authenticated Web -> observed exact target -> persisted binding in real SQLite -> real MCP identity + fixed hello with actual exit 0 -> persisted receipt -> owned-process restart/reload observes the same receipt without replay. Native Windows Codex transport/registration/UI acceptance is recorded separately; the current chat's successful inventory is not automatically that acceptance. Missing real client access/worker credential means partial completion, not a mock pass.

## Cleanup and exclusions

Stop only owned temporary application/browser/fixture processes; remove only explicitly identified task-owned temporary test files after evidence capture. Retain new source, dependency locks, sanitized verification summaries and isolated review state. Do not stop/restart/delete any container, change real Tenant policy, publish provider routes, invoke model inference, create persistent systemd/Compose workloads, edit existing client registration or global trust automatically. Keep old main at its recorded SHA and preserve server/clients.py WIP plus the untracked archive. Commit phase 4 code/docs only in rewrite/v2 after actual checks. Stop for developer acceptance of the minimum system before phase 5.

Approval status: user confirmed this concrete scope (SRC-030). When the independent application launcher was unavailable, the user requested preservation of results without adding credentials (SRC-032). Live gate remains incomplete.
