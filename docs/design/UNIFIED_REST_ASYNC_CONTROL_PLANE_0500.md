# Unified existing REST routes and asynchronous Box creation (v0.50.0 development contract)

**Status:** approved implementation direction, not an installed acceptance certificate.  
**Owner:** BoxWeave. **Baseline:** `v0.49.6` / Containarium `v0.128.0`.  
**Scope:** reuse **already implemented** Containarium REST operations through *every* BoxWeave ingress; retain upstream MCP for other capabilities. Do **not** reopen the abandoned bulk 8+21 MCP-to-REST study.

## v0.50.3 implementation update — REST mandatory for migrated operations

**Current source contract (not installed acceptance):** The eight fixed
REST-migrated Box operations in Section 2 now have only one outbound
Containarium transport from both the application and Agent Gateway:
authenticated REST through the app-owned loopback Provider binding.
`mcp_gate.py` no longer sends the eight methods to upstream MCP or
reads their upstream MCP schemas. `box_debug.summary()` likewise has
no MCP fallback. There is no functional root
`lifecycle-rest.json`/`debug-rest.json` routing switch and no
`debug_rest_enabled` bypass. The installer enables the shared
R-04 app coordination config and provisions the claim database before
activation; unsafe/incomplete binding fails closed. The upstream
MCP launcher continues only for unmigrated tools and guest connect/exec.

The original requirements for immutable identity, one dispatch,
`CREATING` vs guest-ready, UNKNOWN non-replay, source-bound
release and independent installed verification are unchanged.
The historical pre-v0.50.3 discussion below describes the migration
motivation; it is not the current runtime path.

## 1. Verified baseline and mismatch

The app-owned `bwmini/provider_rest.py` currently implements loopback/authenticated:
`GET /v1/containers`, `GET /v1/containers/{logical}`,
`POST /v1/containers`, `POST /v1/containers/{logical}/start`,
`POST /v1/containers/{logical}/stop`,
`DELETE /v1/containers/{logical}`,
`PUT /v1/containers/{logical}/resize`,
and `GET /v1/containers/{logical}/debug`.
App Web/CLI uses seven lifecycle methods, plus conditional debug. Agent SSH/stdio and authenticated HTTP MCP still enter `mcp_gate.py`, which forwards those calls to the upstream MCP and thence REST, imposing the upstream MCP client's hard 120-second HTTP timeout.

In the single v0.49.6 owner-host #88/#92 reproduction, original HTTP task `abd9e8adfc11b626dead60e902987690` returned `UNKNOWN` after 121,324 ms even though Incus had created UUID `4d168b3d-188a-4a96-8470-88eaa14edf53`; guest user/SSH appeared later, and the requested nesting remained false. This is not authorization to replay the create.

## 2. Eight-route single-provider execution target

| BoxWeave logical method | Exact Containarium REST | Ingresses to cover |
|---|---|---|
| list_containers | GET /v1/containers | Web, app CLI, SSH/stdio MCP, HTTP MCP |
| get_container | GET /v1/containers/{logical} | same |
| create_container | POST /v1/containers | same; **one durable asynchronous task** |
| start_container | POST /v1/containers/{logical}/start | same |
| stop_container | POST /v1/containers/{logical}/stop | same |
| delete_container | DELETE /v1/containers/{logical} | same |
| resize_container | PUT /v1/containers/{logical}/resize | same |
| debug_container | GET /v1/containers/{logical}/debug | same, strictly validated/redacted |

The public **MCP** names, authorization, schema and discovery must remain compatible. The Agent gateway is not a general REST proxy and must never accept arbitrary URL/method/token input. Any other upstream tool remains MCP, including `connect`, specialized network/tenant/Guard/route, backup and runner functions. `enable_nesting` is **not** an upstream REST field and must stay an independently authorized, identity-attested post-create operation.

## 3. One durable asynchronous create contract

One typed service shall accept a reviewed create intent from Web/CLI/SSH MCP/HTTP MCP and atomically persist:
`task_id` (opaque, immutable), optional `request_key`, canonical logical name and Provider scope, argument hash (no SSH keys/tokens), actor, state, timestamps, and exact claims. 
- `accepted`/pre-dispatch `queued` does not mean the Box exists.
- `dispatch_intent` must commit **before** any outbound POST. Dispatch **exactly once**, even with simultaneous callers or process restart.
- REST `async=true` shall be used only when exact deployed upstream contract is authenticated and schema-verified; `CREATING` is **accepted**, not `ready`. Response loss, crash between intent and response, ambiguous HTTP status, or stale Provider binding means `UNKNOWN` and **no replay**.
- Poll the original REST status and exact native Incus UUID/createdAt plus provisioning/SSH state. A mere `RUNNING` Instance is insufficient. The task is `ready` only after authenticated Guest identity and SSH verification and requested post-create settings (e.g. nesting) are independently observed. Never automatically perform a post-create write against a merely observed UNKNOWN. If upstream native state is inconclusive, remain `UNKNOWN`.
- The original task status is read-only from **all** public entrypoints. A user retry with same key or exact name+hash yields the **same** receipt; conflicting payload never dispatches.
- Tenant scope and the name resource must be reserved before creation; the UNKNOWN claim persists until a reviewed, second-confirmed, append-only audited release. Never automatically clean up a possibly initialized Box.
- Do **not** silently drop MCP options: map and verify each supported parameter (`cpu, memory, disk, image, os_type, gpu/gpus, isolation, tenant_id, network_tenant, backend_id, pool, region, monitoring, encrypted, enable_podman, ssh_keys, idle_stop_minutes, disable_auto_sleep` where supported); reject unsupported nondefaults *before* intent. Managed server-owned Ed25519 public key remains mandatory. Tokens and private keys never enter log, receipts or args.
- All callers use the same REST transport and operation semantics. REST requests must remain loopback-only, no redirects/proxy, allowlisted routes, bounded body/response, per-request safe credential rotation and typed redacted errors.

## 4. R-04 F30–F32 security

The UI/Web and root-published Gateway remain separate principals, but all mutations share the same reviewed Provider scope, identity, R-04 claim store and resource keys. **Changing transport does not change authorization.** The protected Gateway must not expose direct untracked writes. For a create, use the Provider-scope logical-name claim before an Instance exists; post-create resources bind to exact Instance UUID/createdAt. The app's opt-in shared coordination policy and root Gateway protected mode are separately operator-configured; code must never turn either on silently. Without a proven common claim namespace, a mutating path must **fail closed**, not fall back to upstream MCP. SSH operator direct Incus/Containarium commands outside BoxWeave cannot be claimed protected.

Required adversarial sequences: Agent vs Agent, Agent vs Web, Web vs Agent; same-name create; start/stop and resize races; timeout during POST; process crash before/after durable dispatch intent; stale optimistic claim; crash recovery; exact-identity forced UNKNOWN release; create succeeds yet guest is unready; nesting partially applied; read-only current-state observation cannot release unknown claim.

## 5. Incremental implementation and rollback

A. Introduce **one exact-route REST operation adapter** usable from app and trusted Gateway with a strict root-owned opt-in Provider REST authority binding (same deployed version/scope; not the unrelated debug-only permission). Review installed credential permissions and authorize operations by BoxWeave action, not caller-supplied URL. Add dual entrypoint fixtures.
B. Unified read routes first, and independent REST debug projection. Preserve MCP formatting and upstream pagination contracts; incomplete inventory is `UNKNOWN`, never an empty list.
C. Durable asynchronous create service, accepted-versus-ready receipts, no-replay UNKNOWN and native guest/nesting observation. Migrate old HTTP journal records without replay, preserve their original task IDs and evidence.
D. Remaining REST mutations with exact identity preflight, authorization parity, grow-only disk and R-04 shared claims. Existing app Web/CLI lifecycle receipt semantics must survive.
E. Isolated multi-entry black box and independent verifier evidence, then ordinary full CI, GitHub Release; never move stable installer pin without explicit instruction. Installed host upgrade and real acceptance are separate from CI.

**Rollback:** Disable newly selected REST Gateway writes via an operator-owned policy; never automatically revert an already-dispatched operation to legacy MCP. If an operation entered `UNKNOWN`, preserve claims and use the reviewed manual recovery procedure. No new network listener, broad bearer leak, privileged daemon, remote log aggregator or ordinary-user 'BoxWeave repair Box'.

## 6. Evidence checklist / release gate

- Source-level route-by-route parameter coverage matrix and one canonical REST implementation, including positive/invalid/denied cases.
- `tools/list` exactly retains 69 baseline tool names unless separately justified; no authentication downgrade.
- HTTP/SSH/Web/CLI same task identity and no-double-dispatch under concurrency; restart resume via **observation only**.
- 8 REST routes exercised with scoped test credentials, auth rejection, 404/409/timeout/redirection/oversize/symlink/corrupted policy.
- Slow (>120 s) create produces an early accepted receipt, later genuine Guest SSH and nesting readiness without misclassifying accepted as ready; controlled timeout remains UNKNOWN.
- R-04 real concurrent actors and independent native UUID/createdAt assertions; no historical UNKNOWN target adoption.
- CI / source fixture evidence `PASS` must be reported separately from **installed / independent** black-box acceptance.
