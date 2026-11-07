# BoxWeave v0.50.2 P0 scoped reacceptance — create, UNKNOWN and R-04

**Execution date:** 2026-11-07 UTC. **Source release:** [v0.50.2](https://github.com/vanishs/boxweave/releases/tag/v0.50.2), commit `586e9603f9f833ac156438129aed0c2f4b1306f4`. **Owner reported:** outer host upgraded to v0.50.2; the live `boxweave_info` description advertises `source_version=0.50.2`, but is explicitly a *static source guide*, not independent installed CLI/daemon byte attestation. **Observed provider:** Containarium 0.128.0, Incus 7.5.1, single local `dev` backend.

**Overall verdict: FAIL/PARTIAL.** Issue [#92](https://github.com/vanishs/boxweave/issues/92) partial initialization **reproduced**, issue [#88](https://github.com/vanishs/boxweave/issues/88) post-dispatch UNKNOWN while actual Box RUNNING **reproduced**. F30–F32 actual production-code *isolated simulation* **6/6 PASS**, but actual Web/SSH/HTTP protected three-ingress concurrency and test-host fault injection remain **BLOCKED/PARTIAL**. No global installed PASS or independent Verifier claim.

## Scope and safety

- This is a narrowly scoped **new reacceptance**, not the historical v0.49.4 [Draft PR #100](https://github.com/vanishs/boxweave/pull/100). No existing UNKNOWN was retried, adopted, deleted, manually unlocked or marked successful.
- A single **new** test-owned outer Box `bwp0-0502-1107-a` was created via the actual outer HTTP MCP `create_container` with 1 CPU, 1GB memory, 10GB disk, `enable_nesting=true`, `enable_podman=false`, and one original 32-hex intent. No resend or second create attempt for this name.
- Independent diagnostics used `get_container`, `debug_container`, `boxweave_nesting(action=inspect)`, `boxweave_create_status`, `connect(exec)` (read-only guest identity command), and bounded Gateway diagnostics.
- The host reports **4 physical CPUs, 86 tenant-committed CPUs (21.5x), with admission gate disabled** at observation time. This can affect timing; overload is **not** a proven root cause of these errors.
- The original UNKNOWN target remains intentionally retained. No host root shell, production policy activation, reboot, guest provisioning repair, trust bypass, forced lock release, or cleanup of uncertain effects was performed.

## Actual ingress versus outbound Provider transport attribution

The entrypoint tested was **external authenticated HTTP MCP**; that does *not* establish the Gateway-to-Containarium transport was REST. A read-only `get_container(bwp0-no-such-0502)` of a definitively nonexistent name returned the upstream **raw** Containarium `HTTP 500 / rpc error: code = Unknown ... Instance not found` text through the outer MCP, instead of the v0.50.2 centralized Adapter's canonical `NOT_FOUND / 404`. The exact same unnormalized error was observed during an early missing-target read for the new test Box. At pinned v0.50.2 source, `mcp_gate.py` selects `gateway_rest.execute_read` only when `gateway_rest.policy()` enables the separate root-owned Gateway REST policy; the `boxweave_diagnostic_run(profile=rest_read)` check only proves the **app-owned** REST route, not this Gateway selection.

**Strongly suggested route:** HTTP MCP ingress → legacy forwarded upstream MCP, **not necessarily** Gateway REST, on this updated host. The actual root-owned Gateway policy and request-level transport trace were not independently inspected. Consequently this live test **does not prove the new centralized REST adapter failed**, nor accept the untested opt-in Gateway REST write path. It does demonstrate #92/#88's real user-facing create/UNKNOWN symptom on the currently available MCP surface. Any next Agent testing the v0.50.2 REST adapter must explicitly verify the separate Gateway REST opt-in and compare exact routing before claiming REST installed acceptance.

**Source-backed nesting mechanism (not a live trace proof):** In v0.50.2 `mcp_gate.py`'s **legacy forwarded create** path, the Gateway removes `enable_nesting` before forwarding to Containarium, then invokes `nesting_plan` and `nesting_invoke(action=apply)` **only after a successful upstream create response**. A timeout/unknown result before that branch can therefore leave a real new Box at `nesting=false` even after the guest becomes ready. This precisely fits the observed symptoms, but the runtime outbound selection and individual native write/ack timing remain unverified. Upstream v0.128.0 `ContainerServer.CreateContainer` separately documents `req.Async` as returning a `CREATING` response before its background work; do not claim this live 130-second MCP result proves a REST async-create regression.

## P0 issue #92 — created Box with partial postconditions

| Assertion | Observed v0.50.2 black-box evidence | Verdict |
| --- | --- | --- |
| Exactly one original create request | HTTP MCP acknowledged original durable intent as `submitted / REST_GATE_PENDING`, task `937574372acda6ca237902e9b58f62b6` | PASS for single *client submission*; actual upstream dispatch count NOT independently proven |
| Provider side effect and stable identity | Instance `bwp0-0502-1107-a-container` reached `RUNNING`, UUID `9db2782c-9e51-4cb1-aec6-c57ace192f59`, `createdAt=1794031923`, matching CPU/memory/disk; same incarnation observed again | Observed real side effect |
| Guest account / SSH during provisioning | Initially `debug_container` reported guest account absent and `connect(exec)` rejected `TRUST_NOT_READY`. Later, **without repair**, `connect(exec='id -un; ...')` returned exact username, home-present, exit 0 | Initially PARTIAL; **eventual** guest-ready observation PASS |
| Requested `enable_nesting=true` | Native `boxweave_nesting inspect` repeatedly returned `nesting=false` on the identical UUID after guest became accessible and original receipt reached UNKNOWN | **FAIL** — requested postcondition not fulfilled |
| Overall create success | The durable receipt never became `succeeded`; it became `unknown`. A running Box is not equivalent to successful acknowledged, fully configured create | **FAIL** |

**Attribution boundary:** This run did **not** newly reproduce the historical exact HTTP 500 `already exists` error at create ingress. It instead reproduced the **same broader P0 class** of partially initialized Box plus unconfirmed create result. Initial missing guest user was transient, not proved permanent. No unsupported upstream cause is asserted.

## P0 issue #88 — timeout/result loss after actual creation

The same original task transitioned `submitted / REST_GATE_PENDING` → `unknown / UPSTREAM_CREATE_OUTCOME_UNKNOWN` while the Provider independently returned a RUNNING instance at the unchanged UUID. Gateway bounded diagnostics observed a `create_container` `UPSTREAM_JSONRPC_ERROR` with `elapsed_ms=130000`; the trace's `root_cause` was `not_verified`. The exact original task ID and subsequent status lookups were preserved; no create replay, no URI/tokens or raw private logs were published.

**Verdict: FAIL (live recurrence of result-loss / UNKNOWN with real side effect).** The deliberate fail-closed UNKNOWN classification itself is a **safety PASS**, not a successful create. Post-dispatch daemon/REST versus Gateway timeout and acknowledgement sequence remains to be traced; this observation does not prove Containarium retried internally.

A historical separate task `1038db8c847bf1d523c2541c84400ddf` for `bwrest110-1107-qa` also remains UNKNOWN despite the Provider having reported RUNNING. Its old scene was not changed.

## P0 F30–F32 — exact production coordinator simulation

The authentic v0.50.2 `application-candidate` was downloaded from the successful [main CI run 42345012490](https://github.com/vanishs/boxweave/actions/runs/42345012490); `SHA256SUMS` checks passed. `boxweave.zip` SHA-256: `8ab37c1f97db821bd2a090fda84e086fc6cc5907f08bc77ef3c26c66d97321f6`. Executed six deterministic **isolated Python/SQLite tests against the actual shipped `operation_coordination.py`**, not a mocked coordinator, on an agent-local test container (not the host or a live Box). The fake remote effect was intentionally in-memory only.

| Feature / executed test | Observed outcome | Evidence grade | Still missing |
| --- | --- | --- | --- |
| F30 lease, expiry, competing claim | Invalid lease token rejected, expired lease did **not** release dispatched claim; contender `RECOVERY_REQUIRED`, unrelated name admitted | **T PASS** | Real protected Gateway policy, actual lease and competing Provider writes |
| F31 three ordering permutations | Separate coordinator instances shared real SQLite; Web-first vs SSH, SSH-first vs HTTP, HTTP-first vs Web labels all produced one original claim, competitor `RECOVERY_REQUIRED`; fresh claim allowed only after verified original terminal event | **T PASS** (coordinator-only) | Real Web/SSH/HTTP frontends, actual independent actors, live in-flight barrier and side-effect oracle |
| F32 simulated side effect, process restart | One simulated external effect, `recover_dispatched` made original UNKNOWN, conflicting write denied, no replay | **T PASS** | Real test-only Gateway/worker SIGKILL between dispatch and acknowledgement |
| F32 manual reviewed recovery | Wrong confirmation / challenge rejected; exact reviewed release persisted one immutable audit, original receipt still UNKNOWN, no remote cancellation or retry | **T PASS** | Real privileged operator workflow and app-local reentry independently verified on separate disposable host |

**Simulation runner** SHA-256 `9b4321e380ece8164d3938d9898b5e70b7653a2d43c0c6ffe817a78fcc4579e4`. **Result JSON** SHA-256 `159b5c0ecdc439aef5f817923f647d73a35c610142fea8462f624cac119b47ca`. 6 passed / 0 failed. The runner/result are retained as private conversation artifacts; their hashes provide correlation, not proof of independent execution or of live ingress.

## Required next checks and disposition

1. For #92/#88, inspect the **original** isolated task's audited Gateway+Containarium create trace and post-create nesting observer only through approved read-only tools. Determine whether upstream `POST /v1/containers async=true` acknowledged before its side effect, whether HTTP/JSON-RPC acknowledgement was lost, and why a not-acknowledged create leaves the requested nesting unapplied. Do not reissue the original create, force-unlock or change the retained Box's identity.
2. Use a separately approved **disposable protected host** for actual F30/F31 cross-Web/SSH/HTTP concurrency and F32 fault injection/recovery. Snapshot original receipts and native Provider side effects; do not enable owner-host global protection solely for acceptance.
3. Require independent exact source/version provenance and a **different** Verifier Agent/Box before claiming installed independent PASS. A simulation PASS is not a live system PASS.
4. Keep [#92](https://github.com/vanishs/boxweave/issues/92) and [#88](https://github.com/vanishs/boxweave/issues/88) open as P0. No change to main source, immutable Release, stable installer selector or historical UNKNOWN disposition is authorized by this report.
