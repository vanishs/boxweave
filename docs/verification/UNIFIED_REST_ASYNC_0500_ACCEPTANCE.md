# Unified existing REST control paths and asynchronous creation — v0.50.0

**2026-11-07 installed isolation update:** Official v0.50.0 was installed successfully and the actual coordinated Gateway was exercised, but **real REST async create and R-04 complete acceptance FAILED**: missing-target REST preflight received HTTP 500; a competing resize under an active lease returned a generic error and persisted a blocking UNKNOWN receipt. See the [version-scoped installed live report](UNIFIED_REST_ASYNC_0500_INSTALLED_2026-11-07.md) and [issues #104](https://github.com/vanishs/boxweave/issues/104), [#105](https://github.com/vanishs/boxweave/issues/105). No full independent multi-Agent PASS, external HTTP/OAuth proof or stable-pointer promotion is claimed.

**Evidence state:** implementation/source fixtures in candidate branch; installed host / independent multi-Agent reacceptance **NOT YET PASS**. This file is not a native installation certificate.

## Scope and actual paths

The *eight* existing Containarium REST capabilities are:
1. `list_containers` — `GET /v1/containers`
2. `get_container` — `GET /v1/containers/{username}`
3. `create_container` — `POST /v1/containers` with `async: true`
4. `start_container` — `POST /v1/containers/{username}/start`
5. `stop_container` — `POST /v1/containers/{username}/stop`
6. `delete_container` — `DELETE /v1/containers/{username}`
7. `resize_container` — `PUT /v1/containers/{username}/resize`
8. `debug_container` — `GET /v1/containers/{username}/debug`

Web/CLI application and explicitly selected SSH MCP / authenticated HTTP MCP now share `rest_wire` and `rest_routes`; for Agent writes, `gateway_create` and `rest_mutations` own durable one-shot receipts and R-04 SharedClaims. Unselected specialized upstream tools retain MCP. Source code does **not** promise 69 identical tools when operator opts into *additional* read-only task status tools.

## Safety behavior / limits

- Existing Web/CLI environment plan/confirm and its preexisting receipt table are preserved. Agent create uses `remote_create_tasks`; other Agent mutations use `remote_rest_tasks`; each original Agent task ID is readable through authenticated Web `GET /api/provider-rest-tasks/<task_id>`, app CLI `provider-rest-receipt --operation ID`, and the selected MCP status tools. They are **not a single physical task table**. The shared Provider REST transport, R-04 claim namespace, identity and no-replay rules are common.
- Containarium's `CREATING` reply is only **accepted**. The application worker validates eventual same-name/same-time incarnation and authenticated Guest SSH identity before `REST_GUEST_READY`. Native nesting apply is a separate exact UUID/createdAt postcondition with durable intent *before* the write; a restart after intent cannot re-apply.
- REST timeout / transport loss / daemon restart before a certain original reply yields UNKNOWN; no blind retries, auto-delete, adoption or force-unlock. Original historical UNKNOWN #88/#92 stays untouched.
- For start/stop/delete/resize, REST acknowledgment is **not** the final observed state. The independent worker observes the unchanged instance creation identity and requested state/resource values before releasing R-04 shared claims.
- The Gateway requires exact administrator-managed root policy and authenticated private app database/config plus shared coordination; it **does not automatically enable writes on upgrade**. Missing policy preserves legacy MCP behavior. A present but unsafe policy denies access. Root Gateway policy cannot be changed by MCP calls.
- No arbitrary REST endpoint/tool, public REST proxy, root execution of app-owned code, exposure of token body, logger with freeform upstream sensitive content, automatic retry or stable installer pointer change.
- Currently unsupported for native completion: upstream async daemon restart can erase in-memory pending task state; absent original result after restart remains UNKNOWN. Real external multi-actor F30–F32 and v0.50 installed slow-create #88/#92 tests remain pending, not source-CI PASS.

## Deployment: explicit trusted owner opt-in, not automatic

**Do not enable on owner production host as part of source publication.** On an independent test host, operator first provisions the existing R-04 private claims database and root-owned `coordination-policy.json` using the reviewed R-04 procedure in `docs/design/PHASE2_PROTECTED_GATEWAY_POLICY.md` and `docs/verification/R04_UNKNOWN_RECOVERY_0450_2026-11-05.md`; the *application* config must contain `"coordination":{"enabled":true}`. Confirm the operator's app/Agent Provider identity matches the current root-published trusted scope and that a fresh app worker is running. App/Agent share one BoxWeave effective UNIX principal, claim DB, and **the same** app-credential Provider REST endpoint. A missing/unsafe component fails closed.

The optional **root-owned** `/var/lib/boxweave-default/public/lifecycle-rest.json` schema is strictly:
```json
{
  "version": 1,
  "enabled": true,
  "mode": "coordinated",
  "url": "http://127.0.0.1:ACTUAL_PROVIDER_PORT",
  "token_file": "EXACT_PRIVATE_APPLICATION_TOKEN_PATH",
  "state_db": "EXACT_APP_STATE_SQLITE_PATH",
  "app_config": "/home/boxweave/boxweave/config.json"
}
```
Use actual verified local paths from the installed private app config; the JSON above is a **template**, not a runnable config. File must be a non-symlink, regular, root-owned, non-group/other-writable file under a root-owned non-writable parent. Credentials are **only referenced by path** and never copied into the JSON. Mode `read` uses only `version,enabled,mode,url,token_file` and exposes *no new writes*.

Before enabling write mode test the original claim DB with a real low-risk cross-entrypoint contention fixture, exact-match create/guest/nesting and a deliberately lost REST reply. Do not adopt old test Boxes or reset old UNKNOWN records. Disable mode on an error, but do not convert or retry a previously sent write. Source-CI success alone does not authorize production activation.

## Named verification matrix

| Capability | Source fixtures | Installed independent evidence |
|---|---|---|
| 8 fixed REST routes and strict argument mapping | `tests/test_unified_rest_contract.py` | Pending |
| no redirects/proxy/credential leakage/disk shrink | `tests/test_provider_rest.py`, `test_unified_rest_contract.py` | Pending |
| HTTP MCP/SSH MCP read-only REST selection and catalog protection | `tests/test_debug_rest.py`, `test_gateway_protection.py` | Pending |
| async POST returns accepted; same key/name dispatches once | `tests/test_gateway_rest.py`, `test_remote_create.py` | Pending |
| lost reply, restart and native nesting uncertain state stay UNKNOWN | `tests/test_gateway_rest.py`, `test_async_create_observer.py` | Pending |
| R-04 Web vs Agent name claim and concurrent single dispatch | `tests/test_gateway_rest.py`, `test_lifecycle_shared_claims.py` | Pending |
| start/stop/delete/resize original instance and desired state | `tests/test_rest_mutations.py` | Pending |
| read-only original Agent receipt from Web and app CLI | `tests/test_rest_task_status.py` | Pending |
| >120-second real create, guest readiness, exact nesting and no replay | Pending isolated installed Blackbox | Pending |
| R-04 F30–F32 three-ingress real crash/UNKNOWN recovery | Pending independent blackbox | Pending |

**Release contract:** PR source CI `PASS` permits code release under the ordinary main-branch GitHub Actions workflow only after code review and exact version metadata. It does not certify installed owner host or change its current version/stable pointer.
