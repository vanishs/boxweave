# v1 SDK native Box probe — environment gate and grouped acceptance

**Date:** 2026-11-07  
**Purpose:** Attempt real execution in a disposable Box of the [official SDK fit review](../research/OFFICIAL_SDK_FIT_2026-11-07.md) and [20-tool native map](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md). This report records what actually happened, not anticipated results.

## Environment evidence observed

- Live BoxWeave read-only inventory initially responded, reporting **47 accessible managed Boxes** on backend `dev`; this is not a test pass for their contents.
- Live `get_system_info`: host `dev`, Ubuntu, **kernel 7.0.0-42-generic**, **Incus daemon 7.5.1**, Provider/daemon **0.128.0**, storage pool `default` about **10.7% full**. An independent backend inventory snapshot reported 49 running containers; the different counts are source/scope differences, **not** an inferred problem or an exact cross-API reconciliation.
- Live `boxweave_diagnostic_run(profile=preflight)`: four checks passed (CONFIG_READ_PASSED, PROVIDER_MODE_VALID, PROVIDER_BINDING_MATCHES, REST_SELECTED), with coverage **app_owned_probe** only. This does not prove SDK guest/native behavior.
- A **non-mutating** SSH execution in existing `bw-debug-0460-lab` succeeded as its unprivileged user, showing Ubuntu/kernel, `/usr/bin/incus`, `/var/lib/incus/unix.socket` present, Git; **no nested daemon functionality or credentials were verified**. A second non-mutating SSH execution in `bw-upstream-release-1103` also succeeded; neither Box was modified.
- This environment's **Incus 7.5.1 differs from the reviewed 7.6.0 release SDK**. Tested behaviors must record both exact daemon API extensions and the compiled SDK version. Do not label a successful 7.5.1 test as Incus 7.6 end-to-end compatibility.

## Disposable Box admission (one intent only)

| Property | Actual evidence |
|---|---|
| Backend | `dev` |
| Requested logical username | `bwv1sdk-1107-a` |
| CPU / RAM / disk | 2 vCPU / 4 GB / 30 GB |
| Nesting | Enabled in the submitted request |
| Reusable immutable request key | `7b84aaf378e44b9e81c99c29381a1d02` |
| Original task ID | `0f3f3a1f9a1141b21d89e0eb08a83f24` |
| Creation response | `REST_GATE_PENDING` on submission; **`REST_CREATE_ACCEPTED`** on read-only poll |
| Guest SSH result | `TRUST_NOT_READY` at first connect — **not guest ready** |
| Latest subsequent status | **UNKNOWN:** a subsequent status read again returned `REST_CREATE_ACCEPTED` (guest SSH false); `get_container` reported `CONTAINER_STATE_ERROR` with otherwise empty/default resource fields; `debug_container` returned `UNAVAILABLE`; SSH remained `TRUST_NOT_READY`. Later repeated gateway/tool read failures prevented independently reconciling this inconsistent view. |
| Current final state | **UNKNOWN** — no final creation success, login or native outcome proven; **never resend create** |

**Important:** The apparent `CONTAINER_STATE_ERROR` from a partially populated REST get and `REST_CREATE_ACCEPTED` from the original durable creation task **must not be collapsed into an asserted final success or failure**; use exact native evidence before cleanup. After a brief tool recovery, live host info showed **50** running containers, versus 49 earlier, but that aggregate count does not identify the new Box or prove SSH readiness. The Box admission may have progressed after the last successful poll. We do not have verified readiness, no permission to infer failure, and no evidence of a guest mutation or an Incus-native SDK command actually running in the new Box. The original task must be **read**, not repeated.

## Grouped validation matrix — all remaining work unexecuted

| Batch | Source-grounded test targets | Status / no-go |
|---|---|---|
| 0 — reachability | Read **original task**, verify guest SSH, isolate dedicated namespace/user, read daemon version and extensions, confirm nested Incus is initialized without changes to outer host | **BLOCKED** — gateway tool access |
| 1 — native identity and permissions | Restricted non-root `incus` group user confined to per-user Project; foreign Project and privileged device denial; approved `restricted.devices.proxy` and `restricted.containers.nesting` allowances, native ACL permissions; compare restricted TLS only if required | **UNTESTED** |
| 2 — native SDK lifecycle | Build tagged Go Incus SDK with compatible Go toolchain; instance create/get/UUID/name resolution, ETag guarded update and readback; state start/stop; `GetOperationWait` read-only timeout and explicit cancel; classify errors with `api.StatusErrorMatch` | **UNTESTED** |
| 3 — Exec and files | Native `ExecInstance` typed argv, stream/exit and `RecordOutput` return timing; bounded file list/read/ordinary write/delete; conditional write race/symlink/hardlink checks **only in disposable nested guest files** | **UNTESTED** |
| 4 — network | Native instance-owned proxy TCP/UDP on a sandbox-owned loopback port, checked runtime collisions, exact device readback; NIC ACL same-bridge deny and directional A-to-B/reply-only behavior for IPv4 and enabled IPv6 | **UNTESTED** |
| 5 — MCP Go SDK | SDK typed `mcp.AddTool`, `auth.RequireBearerToken` success/expired/revoked, tool error vs JSON-RPC error, default Host and 4 MiB request cap, explicit Origin, 2026-07-28 Stateless `server/discover`, legacy initialize and reconnect `operation_get` | **UNTESTED** |
| 6 — recovery/cleanup | Delete **only** sandbox-owned nested test instances, proxy devices, ACL, Project/temporary credentials; verify resource absence. Retain original Box create receipt without issuing a second creation | **NOT STARTED** |

### Safety boundaries

- Never change the outer `dev` Incus daemon, its host firewall/storage/Project or existing Boxes to make a test pass.
- Create disposable nested instances only after guest and nested Incus ownership are verified. Preserve unrelated resources.
- No background or automatic retry of an uncertain create/mutate request. Read original receipt and verify exact native identity.
- All test evidence must include command/result, SDK module version, daemon version/API extension, owned instance UUID and independent native readback. A code review or an SSH prompt does not count as native validation.
- If Go 1.26.9+ (Incus v7.6 SDK requirement) or native network controls are unavailable in this Box, mark exact tests BLOCKED/UNSUPPORTED rather than substituting an upstream implementation or modifying shared host.

## Additional gateway observations after transient recovery

- A read-only follow-up on original task still reported **`REST_CREATE_ACCEPTED`**, `guest_user_and_ssh_verified=false`.
- `get_container(username=bwv1sdk-1107-a)` reported `CONTAINER_STATE_ERROR`, but most fields were empty/default; this is **not** independent native state verification.
- `debug_container` returned `UNAVAILABLE`; `connect` returned `TRUST_NOT_READY`.
- The host aggregate went from **49** to **50** running containers; no exact Box identity could be inferred from aggregate reporting.
- Additional read-only SSH probes of previously accessible Boxes, create-status and gateway diagnosis again returned **internal tool errors**. The testing transport is therefore unstable for this run.

**Handling:** Do not dispatch a replacement creation, delete an unverified instance or treat these gateway errors as SDK regressions. The original accepted test Box may remain allocated; preserve original task ID for operator-native reconciliation and cleanup **only after exact identity and current status are established**.

## Result and architecture impact

**No Incus or MCP Go SDK integration tests were completed.** The initial Box inventory/host app preflight/SSH probe are **environment-only checks**. Nothing observed contradicts the upstream-trusting SDK-first v1 design; conversely, none of the P0 architecture assumptions are proven by this exercise.

**Follow-up gate:** only once read-only `boxweave_create_status(task_id=0f3f3a1f9a1141b21d89e0eb08a83f24)` and verified SSH work, use **the same existing original Box** for all six grouped test batches. Do not automatically dispatch a second create or treat a gateway failure as a provider/guest failure.
