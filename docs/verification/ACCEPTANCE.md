# Functional and security acceptance matrix

**All listed cases are NOT RUN as of this design-only v1 baseline.** No production PASS is implied by a requirement or code review.

## Evidence levels

- **D:** design/document review only.
- **S:** static source inspection.
- **T:** focused automated test or mock.
- **I:** real isolated installed test using native Incus, Caddy and client.
- **N:** nested real test in multiple layers.
- **V:** separate Agent/test environment independently reproduces original behavior.

Every case reports PASS, FAIL, PARTIAL, BLOCKED, SKIPPED or UNKNOWN and the actual tested environment. An unperformed test is not PASS.

## P0 cases

| Case | Behavior and acceptance oracle | Required evidence when implemented |
|---|---|---|
| A-001 | Authenticated external HTTP MCP initialize, discover, read and invoke; unauthorized, expired token, invalid Origin denied | I + denied native effects |
| A-002 | Local super-admin bootstrap/rotation, host status, upgrade/install/repair; foreign host files/services unchanged | I |
| A-003 | Create/list/get/update/start/stop/delete exact owned Box; actual native UUID/state reads | I + V |
| A-004 | Duplicate create/rename/reused name/IP and stale UUID; no unintended inheritance or duplicate | I |
| A-005 | Exec in owned guest, correct stdout/stderr/exit and timeout handling; guest stays confined to its Box | I + V |
| A-006 | Every mutation returns stable journal ID; duplicate idempotency key returns original, conflicting payload denied | T + I |
| A-007 | Worker dies after dispatch, lost upstream reply, Incus timeout; UNKNOWN preserved and reconciled without replay | T + I + V |
| A-008 | Failed post-state check is not SUCCEEDED; native Incus/Caddy side effects and errors are observable | I |
| A-009 | Private Box A -> B traffic denied by default; explicit directional IP/protocol/port grant works | I + N |
| A-010 | Internet allowed by default and optionally disabled per Box; IPv4/IPv6 enforced as configured | I + N |
| A-011 | Public Caddy site on B accessible to Box A despite private direct denial, per agreed policy | I + N |
| A-012 | Caddy domain points only to owned Box UUID+port; arbitrary upstream/host daemon/foreign route refused | I + V |
| A-013 | Native TCP/UDP forwarding only to owned Box; collision, restart and teardown behave correctly | I + N |
| A-014 | ACL revision/ETag updates preserve unmanaged NIC/profile/ACL state; rebinding handles native IP change | I + V |
| A-015 | Real Caddy outage/restart repaired from owned SQLite route intent, without altering foreign configuration | I |
| A-016 | P0 MCP list and all callable tools exhibit same contracts at L0, L1, L2 (L3 where possible) | N + V |
| A-017 | Child cannot access or modify parent or sibling Incus/Caddy/files/credentials; nesting unsupported errors precise | N + V |
| A-018 | Installed v1 binary, exact version, fresh installation and upgrade verified independently, not inferred from CI | I + V |
| A-019 | Unknown/foreign Incus Instance, host file, raw API, unmanaged route/forward and privileges denied with no side effects | I + V |
| A-020 | Every installed MCP tool and local maintenance CLI leaf gets valid, invalid and relevant denial scenarios or named blocker | I + V |
| A-021 | Production runtime builds with the official MCP Go SDK and official Incus Go client; native Exec, Operation, ACL and Unix-socket behavior are tested against the observed Incus daemon rather than assumed | T + I |
| A-022 | A nested installation performs its own bootstrap without taking parent admin credentials or mutating parent services; parent permission deficits are correctly reported | N + V |
| A-023 | A Box containing another BoxWeave is managed through ordinary parent Box lifecycle only; deleting that owned Box uses the standard confirmation and does not require inner-controller discovery or a recursive orchestration API | I + N |
| A-024 | An unmodified supported Linux container needs no BoxWeave guest Client, resident Agent or SSH; guest-root Exec has correct exit/output and cannot become host root | I + N |
| A-025 | MCP only listens on 127.0.0.1 HTTP by default, no automatic public ingress; operator-configured HTTPS proxy/Tunnel works with auth and MCP transport; bad tokens still denied | I |
| A-026 | Default Ubuntu LTS image, supported alternative image, configurable resources and explicit nesting switch behave as declared and respect nested resource limits | I + N |
| A-027 | Host install reuses existing compatible Incus Project and dedicated Caddy, records dependency versions, automatically installs/upgrades authorized managed dependencies, and refuses foreign-service/port/ownership conflicts | I + V |
| A-028 | Multiple independently revocable scoped tokens enforce permissions with safe bootstrap/rotation and fail-closed storage recovery | I + V |
| A-029 | ACL selection by Box ID or verified owned Box IP correctly resolves native identity; foreign IP changes do not transfer grants | I + N |
| A-030 | Missing/corrupt SQLite ownership store halts dangerous mutations; explicit local recovery does not silently adopt unowned Incus/Caddy assets | I + V |
| A-031 | Successful new-version `v1` CI builds a genuine artifact and automatically publishes one immutable matching tag/Release; unchanged version and docs-only tree do not create false assets; installed upgrades and stable pointer remain manual | T + I |
| A-032 | A new or nested Ubuntu LTS host can complete managed dependency installation/**version upgrades** and explicit BoxWeave upgrade, preserving unrelated services | I + N |
| A-033 | Server-assigned Tracking ID persists across MCP, Go context/domain, SQLite worker, native Incus/Caddy adapter, diagnostic event and final receipt; dedup returns original Tracking/Operation IDs | T + I |
| A-034 | Read and denied MCP actions are safely correlated; client-provided tracking fields do not grant identity; request ID differs on retry while logical Tracking ID remains stable | T + I |
| A-035 | Typed-argv async guest Exec survives client disconnect where service/native worker remains alive; cwd/env/timeout/output bounds, original exit 0/nonzero, stdout/stderr and completion/unknown distinctions verified | I + V |
| A-036 | Clean Box creation defaults to start and verifies independent guest Exec readiness; explicit `start=false` leaves Box stopped without false-ready claim | I + N |
| A-037 | P0 agentless guest file list/read and four `box_file_apply` modes work in clean Box without Guest Agent/SSH; original SHA and independent post-state verify results | I + N + V |
| A-038 | Unsafe guest path, symlink/hardlink/device, stale hash, foreign Box and unauthorized `box.file.delete` Token class are denied without effect; no host mutation or unproven atomic-CAS claims | I + V |
| A-039 | Complete P0 development loop: create -> read -> edit -> deliberate failing Go test -> track and recover output after reconnect -> diagnose -> fix -> passing Go test -> independently verify | I + N + V |
| A-040 | `diagnostic_get` redacts secrets, scope-checks Tracking ID, reports omitted/unverified stages and does not permit generic host log access | T + I |
| A-041 | Known disk shrink denied before native dispatch with no side effects; supported disk growth returns native/postcondition evidence, otherwise UNSUPPORTED | I + V |
| A-042 | Explicit local upgrade detects outdated owned Incus/Caddy dependencies, records before/after, upgrades compatible pinned versions and proves running/native feature compatibility | I + N |
| A-043 | Shared pre-existing Incus with unrelated consumers requires separate local approval before upgrade/restart; decline/refusal leaves foreign services intact | I + V |
| A-044 | Interrupted dependency upgrade or SQLite migration retains exact Tracking ID, source digest, phase and recoverable error; no silent reinstall/downgrade, blanket host update or automatic background package mutation | T + I |
| A-045 | P0 has no silent idle stop/HTTP wake or OAuth-only login prerequisite; later P1 additions do not break Token-backed Agent development loop | I |
| A-046 | `operation_get(wait_ms)` has a max 20000 ms bounded wait, responds on state change and never cancels native work from read timeout/disconnect | T + I |
| A-047 | `operation_cancel` atomically cancels a proven undispatched intent; has its own Tracking/cancel operation receipt; `operation.read`-only token cannot cancel | T + I + V |
| A-048 | Active Exec cancel needs actual native termination proof; concurrent completion, unsupported native actions, UNKNOWN and prior guest side effects are not mislabeled as rolled back | I + V |
| A-049 | All four `box_file_apply` modes, including one-file delete, verify native post-state; unsafe host paths, race, symlink/hardlink/device and stale hashes refused | I + N + V |
| A-050 | Guest-root `box.exec` may modify guest files despite absent file-write/delete Token rights, but cannot access parent/foreign host files | I + V |
| A-051 | `box_update` forbids raw Incus/NIC/device/host mounts and disk shrink; only typed CPU/memory/disk-growth/nesting/internet changes with post-state verification | I + V |
| A-052 | `box_file_read` returns trustworthy size, offset, EOF, SHA/hash coverage; concurrent or partial reads without snapshot-consistency do not fabricate verified full-file hashes | I + V |

## Required adversarial combinations

- Create is in flight while concurrent delete/update arrives; enforce one allowed effect, typed conflict, verified follow-on success.
- Delete then recreate a Box with the same name/IP; ACL, routes, forwarding and old operation receipts never rebind to the new UUID.
- Lose MCP client connection after mutating dispatch (including a long guest build/test); query the *original* operation/output by Tracking/Operation ID. `operation_get(wait_ms)` times out without canceling the original, and `operation_cancel` never implies rollback.
- Incus accepts a configuration write but ignores a key; verify postcondition failure.
- Caddy route written to DB, Caddy API unavailable, later Caddy restart; reconcile without touching unowned domains.
- Parent L0 stops L1 Box while L1 has a running nested operation; verify child recovery and no accidental parent modification.
- Box A cannot privately connect to B without permission but can reach B's intentionally published website.
- Attempt to proxy arbitrary host local port or external service, edit foreign Incus network, or delete arbitrary host files; verify unchanged native state.

Test harnesses must include a real source edit/build/test/retest scenario, known-bad negative controls, and a pre/post managed-dependency version record. Keep original receipts, native side-effect oracles, actor identities and exact candidate hashes. Do not label same-Agent tests independent.

## Coverage reporting

For each public MCP tool, list actual argument boundaries, permission class, positive case, denial/no-side-effect case, concurrency/recovery case where relevant, source/test/installed evidence, and untested items by name. Run the full acceptance command only in authorized, disposable settings.
