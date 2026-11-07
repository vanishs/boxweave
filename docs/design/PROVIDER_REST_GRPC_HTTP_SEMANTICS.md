# Containarium REST → canonical gRPC/HTTP error adapter

**Status:** Source implementation and isolated validation in progress; not an
installed-system acceptance claim. Scope: the **eight fixed** Containarium
routes already used by BoxWeave (3 reads and 5 lifecycle writes), as observed
in Containarium **v0.128.0**. Source-bound version comparison and runtime
readbacks must be repeated when upgrading Containarium.

## Contract

```text
BoxWeave Web / CLI / SSH-MCP / HTTP-MCP
  -> app Provider RESTClient / Agent REST gateway
  -> bwmini/rest_wire.py (one fixed-loopback transport, existing audit)
  -> bwmini/rest_status.py (one centralized semantic adapter)
  -> Containarium v0.128.0 REST (gRPC-gateway)
```

The adapter returns **only standard gRPC statuses** (the published numeric
codes 0..16) and grpc-gateway's standard HTTP code mapping. It does not create
a BoxWeave-specific Provider error enum such as `RESOURCE_NOT_FOUND`, nor
translate an unclassified 500 into a guessed 404. A `NormalizedError` carries:

- `grpc_code`: a standard `GRPCCode` value, or `None` when HTTP alone
  cannot distinguish gRPC codes (e.g. 409 is both ALREADY_EXISTS and ABORTED);
- `http_status`: the standard mapped HTTP status, or the original HTTP
  status for an unclassified reply;
- `upstream_http_status`: the exact original HTTP status for traceability;
- `origin`: `grpc` (validated typed envelope), `compatibility` (exact
  proven upstream defect), `http` (unambiguous HTTP-only mapping), or
  `unclassified`.

The app RESTClient returns a normal domain Fault with the standard gRPC name
and mapped HTTP status when one exists; it preserves the upstream HTTP status
as metadata. The Agent gateway returns the same standard gRPC name for
recognized Provider read failures. Internal `API_*` symbols remain
**transport/audit diagnostics and indeterminate fallback classifications**,
not the Provider's application-facing semantic protocol.

An actual HTTP 404 with **no typed gRPC envelope** might mean the API route
rather than the target does not exist. Consequently an irreversible create
preflight admits absence only with a validated typed gRPC NOT_FOUND response
or an exact known-version compatibility signature; a bare 404 is insufficient.

## v0.128.0 source + installed read audit

| Fixed route | Observed/upstream error behavior | Normalization / caller policy |
| --- | --- | --- |
| `GET /v1/containers` (list) | Native manager `fmt.Errorf("failed to list containers")` yields gRPC Unknown / HTTP 500. Successful lists include `unreachableBackends`; isolated live read confirmed HTTP 200 with zero unreachable entries. | Never synthesize empty inventory on errors or partial peer responses. Unclassified 500 stays failed. |
| `GET /v1/containers/{name}` (get) | **Live confirmed:** truly absent instance => HTTP 500 `Unknown`, exact double `failed to get container` + Incus `Instance not found`; existing Box => HTTP 2xx. Peer list errors are suppressed in upstream source. | The *exact* target/project-bound legacy HTTP 500 body maps to gRPC NOT_FOUND / HTTP 404. Generic body substring matching is forbidden. Future typed NOT_FOUND is honored directly. |
| `GET /v1/containers/{name}/debug` (debug) | **Live confirmed:** absent instance => HTTP 200, `containerState="missing"`; existing => HTTP 200, `running`. Peer unreachable can produce a valid 200 status of `unknown (backend unreachable)`. | A successful response remains a successful response. Diagnostic state fields are **not** rewritten into errors. |
| `POST /v1/containers` (async create) | Some pre-dispatch conditions use typed InvalidArgument/FailedPrecondition/Unavailable; backend absent, async already-in-progress and lower-level errors can use plain `fmt.Errorf` => HTTP 500 Unknown. | Decode typed gRPC codes only; unknown HTTP 500 stays unknown. No retry after write dispatch, even if a failure response is received. |
| `POST /v1/containers/{name}/start` | v0.128.0 `lifecycleError` maps typed Incus missing to NOT_FOUND and unreachable peers to UNAVAILABLE; other code paths retain plain errors. | Preserve typed code and HTTP mapping; all uncertain write effects remain UNKNOWN to task journal until independently verified. |
| `POST /v1/containers/{name}/stop` | Same partially typed behavior as start; other paths use plain errors. | Same as start. |
| `DELETE /v1/containers/{name}` | Local manager and peer-forwarding failures typically return plain `fmt.Errorf`, including unreachable peers; may have partial external effects. | No string-to-NOT_FOUND inference. Failed write outcome remains UNKNOWN; do not silently mark deleted or replay. |
| `PUT /v1/containers/{name}/resize` | Local manager/peer failures and post-resize readback can be plain errors. A request may already have modified resources. | No string-to-NOT_FOUND inference. A 5xx never authorizes a retry or releases a durable coordination claim. |

**Evidence scope:** source reviewed at
`FootprintAI/Containarium@v0.128.0` in
`internal/server/container_server.go` (`Create/List/Get/Delete/Start/Stop/Resize`),
`internal/server/debug_container.go`, and
`internal/gateway/gateway.go:ErrorHandler`. The live GET/List/Debug
observations used the already authorized nested test Provider and were
**read-only**; write error classifications above are source-level observations,
not invasive live write tests.

## Exact v0.128.0 compatibility patch

Only when all are true:

1. A `GET` targets an exact valid `/v1/containers/{logical}` route.
2. Raw upstream HTTP status is 500, the JSON body is ≤4096 bytes, has
   exactly `error`, `message`, `code`, and contains no duplicate keys.
3. Both body message fields equal the known complete `GetContainer` +
   Incus error format, including `{logical}-container` and project
   `default`. No bare `not found` substring matching.
4. The normalized outcome is **gRPC NOT_FOUND (5) / HTTP 404**, while
   `upstream_http_status` remains 500 and `origin=compatibility`.

All other HTTP 500 responses stay failures, including `not found` text from
other operations, other projects, other instances, peer discovery failures and
unrecognized JSON. **Additional create-preflight corroboration:** because
v0.128.0 GetContainer ignores peer-list failures, a `compatibility` result
also triggers a fresh authenticated `GET /v1/containers`; that list must
validate successfully, have a matching complete total count, contain an
explicit empty `unreachableBackends` array, contain no duplicate names, and
not contain the requested username. Any partial read, unreachable peer, or
occupied name prevents POST and task reservation. A typed upstream gRPC
NotFound response requires no provider-specific second-read workaround.
Existing upstream #931 shows that even Incus text
`Instance not found` can be transient; this exception is a **temporary
operator-approved compatibility compromise**, not a safe general oracle.

## Fail-closed safety invariants

- No new outbound route, bearer source, redirect, HTTP proxy, retry or
  automatic write replay is introduced.
- Only verified read errors may prove target absence at *create preflight*;
  typed write errors never prove absence of side effects.
- HTTP 400 and 409 alone do not determine which of the multiple valid gRPC
  statuses applies. Leave `grpc_code=None` unless a coherent typed envelope
  establishes the code.
- Valid Containarium typed `Unauthenticated`, `PermissionDenied`,
  `Unavailable`, etc. are translated without relying on English text.
- No HTTP error body or credential is copied into public exception messages,
  diagnostics or audit; the original numeric HTTP code remains available.
- Existing task UNKNOWN, R-04 coordination, inventory completeness,
  ownership, exact-target identity and guest-ready verification must remain
  independent of this normalization.

## Upstream handoff and exit criterion

Follow [BoxWeave #107](https://github.com/vanishs/boxweave/issues/107).
The next Agent should fix Containarium's `GetContainer` with typed Incus
`IsNotFound` **and peer-availability checks**, test other still-untyped
lifecycle errors and submit an upstream PR. Remove the compatibility signature
only after installing a fixed Containarium version in a separate test Box and
observing real GET 404 plus REST async create and R-04 negative controls.

**Source regression gates:** `tests/test_rest_status.py`,
`tests/test_gateway_rest.py`, `tests/test_provider_rest.py`,
`tests/test_unified_rest_contract.py`, complete Python regression suite,
docs/code-first inventory, source archive shipping (including the
root-owned standalone Gateway helper) and CI release-asset checks.

**Installed acceptance gate:** independent re-execution of real
`GET -> async POST -> CREATING -> guest identity/ready`, duplicate-intent
readback, existing-target denial, unavailable peer negative controls and
post-dispatch UNKNOWN no-replay. None is automatically PASS from source tests.
