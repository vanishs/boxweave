# v0.46.0 selected debug_container outbound REST migration

**Scope:** one read-only transport selection, not a return to the reverted
29-tool migration plan. Source baseline: v0.45.0 plus the separate R-04
verification record. Upstream REST mapping is Containarium v0.128.0
GET /v1/containers/{username}/debug. Inbound Agent interfaces remain MCP.

## Behavior and security

- App/Web: when provider_api is explicitly configured, the existing
  /api/boxes/<logical>/debug-summary calls the typed REST GET, after the
  existing Service.observe_target incarnation/authorization check. Missing
  API configuration keeps the old MCP call. To roll back this read alone,
  set debug_rest_enabled to false in the existing application config; this
  leaves the lifecycle REST configuration unchanged. Raw host shell, sshd journal,
  likelyCause, free-form actions and other sensitive fields never enter
  the public Web summary; no SSH/guest command is run.
- Agent SSH/stdio and HTTP MCP: the root-installed standalone Gateway keeps
  tools/list and tools/call shape and checks the host's tool allowlist
  first. A **separate trusted root-owned** policy at
  /var/lib/boxweave-default/public/debug-rest.json selects this *one*
  outbound tool for REST. The exact JSON keys are version (integer 1),
  enabled (boolean true), url (authenticated loopback Provider origin
  http://127.0.0.1:PORT), and token_file (absolute private path
  /usr/local/lib/boxweave/boxweave/application.token).
  Its parent must be root-owned, and the regular file must be root-owned,
  non-world-writable, and non-symlinked. The file does **not** contain a
  credential; the Gateway reads the same independently rotated 0400 token
  as the existing application under its own service UID. The enabled policy
  is not installed or changed automatically by this release.
- In the protected Gateway mode, debug_container becomes visible/allowed
  only when that verified REST policy selects it. Without the policy,
  protected mode retains its prior deny behavior; legacy mode retains the
  upstream MCP path. A malformed policy fails closed. This is existing
  host-wide operator authority, **not per-Agent RBAC**.
- REST request constraints: exact local loopback origin; fixed GET route and
  canonical username; no proxy, redirect, retry, arbitrary URL/query or
  JSON duplicate keys. Re-read a private token on each request. Bound
  responses to 32 KiB. HTTP/auth/shape failures stay explicit; **no silent
  fallback after an attempted REST read**. No raw debug content, path,
  token or upstream exception is written to Gateway diagnostic journals.

## Acceptance matrix / evidence

| Check | Minimum requirement | Evidence |
| --- | --- | --- |
| Source mapping (S) | Upstream v0.128.0 proto GET route and current call-chain review | Reviewed 2026-11-05 |
| App REST (T) | Existing exact target, withheld sensitive fields, no MCP fallback on error | test_debug_rest.py, test_box_debug.py |
| Gateway REST (T) | Existing MCP schema, allowlist first, protected-mode gate, zero upstream MCP calls on REST selection | test_debug_rest.py, test_gateway_protection.py |
| Token/HTTP safety (T) | Rotation, loopback, unsafe/symlink token, bad JSON, wrong types, 404/auth/no fallback | test_debug_rest.py |
| Packaging (T) | Standalone stdlib source included in both source/Release/installed lists | test_arch0_boundary_guards.py and CI build |
| Live installed REST vs MCP (L) | Exact source release installed in authorized disposable host, authenticated same-Box parity + app Web + SSH/HTTP gateway with approved policy | **PENDING independent real installed evidence** |
| Owner-host deployment (L) | Never inferred from CI or release | **NOT AUTHORIZED / NOT TESTED** |

**Rollback:** remove or disable the Gateway's root-owned opt-in policy to
restore future debug calls to the previous MCP path in legacy mode. For the
app, set debug_rest_enabled to false in the existing private app config
and restart the application under the normal reviewed procedure. This
does not disable other REST lifecycle operations.
Do not remove shared secrets, modify the Provider or assume an HTTP 2xx
proves external SSH reachability.

## Executed source/fixture validation (T)

- Disposable development Box: bwmig1-1105-dev; source worktree based on
  e19e60bb9ca4b927c30f225e490cdbc7a9410883 (v0.45.0), with this
  selected migration applied. No inherited dirty 29-tool migration worktree
  changes were used.
- Python 3.12.3, locked Python packages, Node 20.19.5, locked Web packages,
  locally installed Chromium browser and its isolated-host dependencies.
- Actual command: PYTHONPATH=tests:. python scripts/validate.py --browser;
  report .runtime/validation/b5d42424303341d8b6e0465d53c427c9/result.json.
  Exit **0**: knowledge 0, Python **773 tests PASS**, frontend build 0,
  all seven fixture browser workflows exit 0. This is **T**, not installed
  Provider/API authentication or host/network acceptance.
- Focused Python tests for test_debug_rest, test_box_debug,
  test_provider_rest, test_gateway_protection and
  test_arch0_boundary_guards all PASS; test_debug_rest exercises a *real*
  loopback HTTP listener with Bearer auth and 401 negative path, plus a
  rotated 0400 token, no proxy/redirect, malformed JSON and secret
  projection. The mock REST payload is not a real Containarium observation.
- An independently running outer Containarium v0.128.0 MCP debug read
  returned the documented camelCase shape for a task-owned test Box.
  This confirms the sampled legacy shape only; it does **not** prove the
  new compiled REST path is installed on that host.
- No production-host service restart, listener/firewall/Provider change,
  global R-04 policy activation, public route or stable installer selector
  movement. Keep separate L evidence gates above until actually executed.

**Non-goals:** no debug write operation, no Provider version bump, no
R-04 policy activation, no Agent MCP-to-REST inbound conversion, no changes
to connect(exec), network routes or stable release selector.
