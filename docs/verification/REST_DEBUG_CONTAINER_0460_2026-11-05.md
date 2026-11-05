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
| Official Release / CI (CI) | Immutable v0.46.0 tag and full publish | **PASS**: commit `3aa393e8e58fc1d79692214a2e2fd9c79d09055c`; 26 assets, non-draft release; verify and publish succeeded in CI 42015528658 |
| Installed upgrade / health (L) | Official 0.45.0 → 0.46.0 on a disposable host | **PASS**: `bw-debug-0460-lab`, exit 0, `boxweave status` 0.46.0, `doctor --mcp` overall passed / authenticated read passed |
| Same-Box Provider REST/MCP (L) | Exact isolated Box and Containarium 0.128.0, both routes authenticated | **PASS (sampled)**: native disposable Incus test Box `bwdebug0460child-container` returned identical running / hostUserExists=false / hostUserShellExists=false through actual `GET /v1/containers/bwdebug0460child/debug` and MCP `debug_container` |
| Application debug projection (L) | Exact observed incarnation, safe summary, MCP rollback | **PASS (direct installed app path)**: `Provider.observe` identity digest matched, installed `box_debug.summary` returned sensitive_details=withheld and no freeform host fields; setting `debug_rest_enabled=False` in the test process produced equivalent safe MCP summary |
| Standalone Gateway selected REST (L) | Installed helper, reviewed root-owned policy and MCP stdio | **PASS (sampled)**: Gateway v0.46.0 initialize; tools/list advertised debug_container; tools/call read real Box via selected REST with expected values; invalid `bad/target` denied as DEBUG_REST_TARGET_INVALID |
| Real Web HTTP authenticated endpoint (L) | Installed Web login + exact Box authorization + GET debug-summary | **UNVERIFIED**: direct installed app summary was tested, not an authenticated Web HTTP request |
| HTTP MCP ingress / protected Gateway (L) | Real HTTP MCP OAuth and opt-in R-04 protected-mode calls | **UNVERIFIED**: standalone stdio real, protected-mode fixture tests only; no global R-04 activation |
| Owner-host deployment (L) | Never inferred from CI, release or disposable host | **NOT AUTHORIZED / NOT TESTED** |

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

## Official v0.46.0 release and installed acceptance — 2026-11-05 UTC

- [Merged PR #78](https://github.com/vanishs/boxweave/pull/78) selected only
  `debug_container` and kept the owner's independent MCP-first / REST-assisted
  architecture amendments. The initially failed PR CI run 42012473608 was
  caused by two missing code-first inventory entries; these were added,
  the exact CI inventory checker was rerun locally (all 12 inventories
  matched), and the subsequent PR run 42015184391 **passed**. No failed
  check was waived.
- [Official Release v0.46.0](https://github.com/vanishs/boxweave/releases/tag/v0.46.0)
  published **non-draft, non-prerelease**, 26 assets and immutable tag at
  `3aa393e8e58fc1d79692214a2e2fd9c79d09055c`, published
  `2026-11-05T11:04:43Z`. [CI 42015528658](https://github.com/vanishs/boxweave/actions/runs/42015528658)
  reports both `verify` and `publish` **success**. The separate
  `PUBLISHED_RELEASE` stable selector remains **0.43.1**.
- Dedicated outer Box `bw-debug-0460-lab` was created for this test,
  restarted after its own nesting configuration, and given an official
  **v0.45.0 baseline installation** (installer exit 0, `doctor --mcp`
  overall PASS). The exact **official v0.46.0** release installer then
  completed an in-place upgrade (exit **0**). Actual installed host CLI
  `boxweave status` reported 0.46.0, `doctor --mcp` reported
  `overall=passed` and `authenticated_read_passed`; the independently
  installed root-owned helper `debug_rest.py` existed.
- A fresh **test-only Incus instance** `bwdebug0460child-container`
  launched inside this isolated lab. As the installed `boxweave` service
  user, its existing authenticated Provider client produced **matching**
  Containarium v0.128.0 results over *actual REST* and *actual upstream MCP*:
  `containerState=running`, `hostUserExists=false`,
  `hostUserShellExists=false`. These false values are expected for a
  directly provisioned test instance without managed host SSH user
  provisioning and were not misreported as an SSH success.
- The installed application Provider re-observed the exact test instance
  digest successfully; its REST safe debug summary omitted raw SSH
  journal/cause/action/shell fields, marked sensitive details withheld and
  did **not** claim SSH verified. In the same *in-memory test process*,
  `debug_rest_enabled=False` selected legacy MCP, producing the same
  safe summary. No deployed application configuration was changed.
- For the standalone Agent Gateway, a **temporary root-owned nonsecret**
  `debug-rest.json` policy selected the one REST read in this isolated
  lab. Real stdio MCP initialize returned v0.46.0, the tools catalog
  offered `debug_container`, and the tool call returned expected
  authenticated live values. A bad logical name was rejected
  `DEBUG_REST_TARGET_INVALID`. The policy was **removed** immediately
  after testing; the nested test instance was **deleted**, its absence
  verified, and `doctor --mcp` rechecked **passed**.
- **Honest boundary:** current live evidence covers the actual REST
  Provider endpoint and real Agent stdio Gateway, plus direct installed
  application safe projection. It does **not** cover an authenticated
  installed Web HTTP request, remote HTTP MCP OAuth, R-04 protected
  live policy, owner production host, alternate daemon versions, or
  cross-host SSH reachability. Those are **U**, not blanket PASS.

**Non-goals:** no debug write operation, no Provider version bump, no
R-04 policy activation, no Agent MCP-to-REST inbound conversion, no changes
to connect(exec), network routes or stable release selector.
