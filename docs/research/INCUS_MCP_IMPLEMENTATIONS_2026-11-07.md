# Incus MCP implementation study — two small codebases

**Review date:** 2026-11-07. **Status:** source research only; no v1 code changes or native test execution. This document compares two current upstream `main` snapshots:

- [nikitatsym/incus-mcp](https://github.com/nikitatsym/incus-mcp/tree/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4), commit `1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4`: Python `httpx` REST bridge and MCP operation groups.
- [fredmj/incus-mcp-server](https://github.com/fredmj/incus-mcp-server/tree/880d1f1e71747553adeb20c406530cff891ee7f9), commit `880d1f1e71747553adeb20c406530cff891ee7f9`: Go, official Incus SDK and official Go MCP SDK.

**Decision filter:** These are independent examples, **not** a proposed runtime dependency, migration, change to the existing 20 MCP tools, or evidence of v1 PASS. The approved [product boundary](../decisions/PRODUCT_BOUNDARY.md), [SDK choice](../design/TECH_STACK.md), and [durable operation semantics](../design/OPERATIONS.md) remain authoritative. Reference only specific techniques, not project structure wholesale. All following implementation assessments are based on the pinned upstream source.

## Comparison and relevance

| Area | Python repository | Go repository | Smallest v1 lesson |
|---|---|---|---|
| MCP presentation | Five generic tools, `operation` selector plus dynamically built help/schema; operations enumerate much more of Incus | Small, independent typed `mcp.AddTool` registrations | **Keep v1 explicit 20-tool catalog** and official Go SDK schemas; no generic Incus gateway |
| Native transport | Hand-built `httpx` REST calls with Incus response-envelope handling | `github.com/lxc/incus/v6/client` typed SDK and `github.com/modelcontextprotocol/go-sdk` | Use **BoxWeave-selected Incus v7 SDK**, narrow domain adapter; neither upstream version nor REST router is copied |
| Read/write verification | Explicit sent-vs-returned field-presence verification, including deferred verification after async completion | Wait for some native operations to finish; no general full-effective-value verifier | Compare **actual intended values + preserved unaffected fields**, not just field names/API success |
| Optional input semantics | `_UNSET` distinguishes omitted key from explicit JSON `null` (clearing) | Go `omitempty` on optional JSON fields; no systematic three-state abstraction | Distinguish **omitted / explicit clear / supplied value** only where v1 wire contract permits clearing; `box_update` must preserve omitted resources |
| Asynchronous state | Polling handles and pending-write verify metadata in process, TTL-reaped | Blocking `op.Wait()` / bounded Exec `WaitContext` in MCP invocation | Keep persisted BoxWeave intent/receipts, original Incus Operation ID, UNKNOWN and post-verification **across process restarts** |
| Safety | Broad direct Incus surface; HTTP transport explicitly has no built-in auth | Opt-in write/create/delete tool sets, deletion `confirm_name`, raw Incus `config`/device maps, CLI remote credentials | Use existing **admin/viewer authorization + exact Box UUID ownership**, not name confirmation or feature flags as a substitute |
| Testing | Pinned Incus OpenAPI conformance checks, mock errors/verify, real Incus e2e gate | Typed fake Incus SDK with official in-memory MCP client/server tests | Borrow **contract drift and adapter/unit-test patterns** plus native black-box acceptance |

## Detailed implementation findings

### 1. Python: source-level interface conformance is unusually valuable

[Conformance test](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/tests/test_swagger_conformance.py) extracts each REST call's HTTP method, path, query names, body field names and recognized enum values from AST, checking them against a pinned upstream API document. Explicitly tracked exceptions fail on unexpected changes; [test suite](https://github.com/nikitatsym/incus-mcp/tree/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/tests) and [README](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/README.md) describe native e2e coverage.

**Applicability:** v1 uses typed Incus Go SDK calls, *not* hand-written REST paths, so **do not implement a Python-style AST REST scanner**. Instead pin the selected SDK/runtime versions, make a concise per-v1-tool SDK method/typed field/API-extension mapping, and test input field presence, supported enums, resource state, and meaning with fake SDK plus real Incus readback. Add negative tests for a syntactically successful request whose intended change is ignored.

### 2. Python: omitted input is not the same as `null`

The [`_UNSET` sentinel](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/registry.py) and [Pydantic model `exclude_unset`](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/server.py) distinguish *parameter not supplied* from an explicit `null` that some Incus PUT/PATCH endpoints interpret as a clearing value. The [test](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/tests/test_write_verify.py) verifies what is actually sent.

**Applicability:** v1 should normalize input before native dispatch: `omitted` means preserve existing value; `explicit clear` is accepted **only where the BoxWeave-approved typed schema expressly allows it**, otherwise reject; supplied value means validate, set, then read back. In Go, typed optional fields or explicit presence flags may be needed; a pointer alone can conflate absent and JSON `null`. Never expose raw Incus `config`/devices as an escape hatch.

### 3. Python: verify after terminal completion, but compare **values**, not just keys

[`_verify_response`](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/tools/helpers.py) recursively confirms the presence of sent keys; it **does not compare all scalar values to expected values or preserve-check every unrelated field**. The [write path](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/tools/write.py) records async verification in a process-local map; the [read/wait path](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/tools/read.py) fetches the target after successful terminal operation. The map **does not survive process restart** (source comment). A returned field can still carry an ignored/wrong value.

**Applicability:** v1's existing durable journal stores immutable intended change and native Operation ID, and verification must compare **effective resource values, unchanged protected config, correct instance incarnation and dependent network resources**. A terminal native success is not user-visible verified success if independent readback fails or remains unknown. Preserve partial-effects/UNKNOWN evidence rather than repeating native mutation.

### 4. Python: read the authentication *claim*, not merely HTTP 200

[`IncusClient.check`](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/src/incus_mcp/client.py) checks that Incus's `GET /1.0` response does not report `auth=untrusted`; some API responses may be served to untrusted clients. Its [README](https://github.com/nikitatsym/incus-mcp/blob/1729fa4a216132ca7c1b3bfbb4e92c7a3b1090a4/README.md) states that optional streamable HTTP has **no authentication** and expects an external gateway.

**Applicability:** v1's native upstream privilege check must prove effective authorization for its own scoped project; v1's own **authenticated loopback HTTP MCP remains mandatory**. Do not copy unauthenticated HTTP endpoints, token/OIDC client complexity or a publicly exposed Incus admin credential.

### 5. Go: typed, minimal official-SDK registration is the better code model

[`main.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/main.go) registers typed, separately scoped tool handlers using the official MCP Go SDK. [`internal/tools`](https://github.com/fredmj/incus-mcp-server/tree/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools) keeps individual operations short. [`internal/incusclient/client.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/incusclient/client.go) uses official Incus CLI configuration and `ParseRemote`/image alias resolution rather than implementing remote name parsing. [`tools_test.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools/tools_test.go) injects a fake `incus.InstanceServer` and drives an in-process MCP client/server protocol session.

**Applicability:** borrow typed registration, small handlers, injectable narrow Incus adapter, and in-memory MCP protocol tests. Leverage official SDK-supported image resolution patterns only **within approved source/image allowlists**; never inherit arbitrary CLI default remote, unrelated Incus credentials or an SDK v6 pin. Prefer a small purpose-defined adapter interface over exposing the entire `InstanceServer` to every business handler.

### 6. Go: useful guards but not sufficient ownership/recovery

[`create.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools/create.go) does not auto-start after create, which is good for v1's native network **guard-before-first-start**. [`delete.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools/delete.go) requires `confirm_name` and checks state, but operates by **name**, without BoxWeave's durable ownership/incarnation model. [`power.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools/power.go) calls `op.Wait()` before reporting completion, but has no persisted business operation record. [`exec.go`](https://github.com/fredmj/incus-mcp-server/blob/880d1f1e71747553adeb20c406530cff891ee7f9/internal/tools/exec.go) bounds the call to 60 seconds, uses two `bytes.Buffer` outputs and returns `exit_code=-1` if metadata is missing. The missing-code sentinel is **better than guessing 0**, but a returned result can still be mistaken for verified execution, and the output buffers are **not byte-bounded**.

**Applicability:** keep v1's stronger existing semantics: fixed owned UUID/incarnation and durable claims; admission before effects; bounded *streaming* capture; explicit observed/missing exit classification; non-replay after disconnected call; native cancellation only when actually proved. `confirm_name` is at most UX defense-in-depth, never an identity or authorization guarantee.

## Minimal actionable outcome (not new features)

| Priority | Existing area | Action for implementation/verification |
|---|---|---|
| P0 | `box_update` and typed mutation schemas | Define and test explicit **omitted / clear / set** semantics; preserve every unrelated setting, reject unsupported clears |
| P0 | Operation postconditions | Compare actual **values and effective state** after native terminal events, not mere key presence or 2xx; retain original durable receipt on interruption |
| P1 | Incus adapter tests | Add pinned SDK/native API-extension + operation/field mapping tests that catch unsupported enum/field and silent ignore; real Incus remains the final gate |
| P1 | MCP protocol tests | Test registered schema, privilege rejection, malformed arguments, SDK errors, and unknown results with an in-process MCP client + fake native adapter; do not confuse mocks with deployed acceptance |

## Deliberately rejected

- Five generic dynamic dispatch MCP tools, broad unrestricted Incus resource management, raw native device/config writes, clusters, VM platform features and arbitrary remote selection.
- Bare CLI credentials/default remote as a BoxWeave authority model; `confirm_name` or `--enable-delete` as substitutes for verified owned UUID and approved admin credentials.
- In-memory waiter/pending-verification registry as the durable operation store; per-request blocking `op.Wait()` as the async controller; fixed 60s Exec plus unbounded output `bytes.Buffer`.
- New REST service, hand-written Incus REST transport, OpenAPI-ast compiler, extended CLI command surface, extra MCP tools, guest Agent or compatibility layer.

**Verification note:** this review inspected source files and their own tests, **not** a working live deployment. The upstream projects' claims/tests are not BoxWeave v1 native evidence. Existing v1 tests remain UNTESTED until implemented and actually run.
