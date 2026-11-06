# HTTP MCP create regression: v0.49.2 failure → v0.49.3 scoped live pass

**Recorded 2026-11-06 PT.** Evidence levels are intentionally distinguished:
- **H/L**: direct observations through the authenticated, refreshed HTTP MCP connection
  to the owner's existing `dev` host, with independent get/list/debug/SSH reads;
- **T**: local/GitHub-CI unit and fixture tests for deduplication and recovery;
- **U**: not independently observed. The original tool response and opaque task
  selectors remain in the private conversation evidence, not as credentials
  or potentially identifying logs in this document.

## Original failure (v0.49.2, issue #92)

A single attempted ordinary `create_container` for test-owned
`bwi92-1106-0147` returned an HTTP 500 containing `already exists`.
Independent `get_container` and `list_containers` showed the original
instance running with the requested 1 CPU, 1GB RAM and 10GB disk.
The original request was not repeated and the observed Box was retained.

This is an observed symptom, **not proof** that the client, the remote
MCP transport or Containarium is the root cause. The full incident also
includes distinct failed creates, missing guest users and requested nesting
not applied; those findings must not be erased by a later positive test.

## Installed v0.49.3 retest (same owner host, new ordinary Box)

Host CLI `boxweave status` reported version `0.49.3`; HTTP local status
reported listener `127.0.0.1:18844`, `local_discovery=true` and
`managed_service=true`. `external_verified=false` was not mistaken
for a failed or successful public HTTPS transport check.

On refreshed ChatGPT HTTP MCP, a **single original** `create_container`
for `bwi92-v493-1106` (1 CPU, 1GB RAM, 10GB disk, Podman disabled,
nesting not requested) returned a non-error, durable
`state=submitted` receipt. Subsequent same-name/same-argument calls
with different JSON-RPC invocations returned the **same original**
`task_id` and `request_key`; no second independent task was issued
by this ingress.

Read-only native Provider `get_container` first showed a real
`STOPPED` instance, then `RUNNING` with IP `10.100.0.40`.
A *premature* `debug_container` reported the inner guest user missing
and `connect` temporarily refused `TRUST_NOT_READY`. Without modifying
the host or guest, provisioning progressed: the original task receipt
became `state=succeeded` / `UPSTREAM_CREATE_ACKNOWLEDGED`, a later
`debug_container` no longer reported the missing guest user, and
`connect(exec)` returned exit 0 with the intended username and `HOME_OK`.
The independent final `list_containers` contained **exactly one**
`bwi92-v493-1106-container` with requested resources.

| Case | Evidence | Verdict |
| --- | --- | --- |
| Authenticated HTTP ingress returns durable asynchronous receipt | H | PASS, one observed original |
| Exact HTTP retry returns identical receipt without another accepted task | H | PASS, same task ID |
| Native instance eventually runs, requested CPU/RAM/disk | H | PASS |
| Inner guest user, HOME and real SSH command | H | PASS, late readiness after initial transient |
| Number of *all* underlying Containarium/Incus create dispatches | U | NOT VERIFIED: gateway/request correlation absent |
| Reboot/worker crash during in-flight create, same identity preserved | T only | NOT INSTALLED VERIFIED |
| Cross-interface SSH MCP/HTTP MCP/Web/REST same-name concurrency | U | NOT VERIFIED |
| `enable_nesting=true` application with identity proof | U | NOT VERIFIED |
| Genuine client TLS/OAuth external journey from a separate actor | U | NOT VERIFIED |
| Prior distinct #92 failed Boxes and #88 UNKNOWN incarnations | Historical | RETAINED; no automatic repair/cleanup |
| Full multi-Agent black-box reacceptance across all tools | U | NOT EXECUTED by this focused retest |

**Critical interpretation:** An early `STOPPED` instance, missing guest
user or transient `TRUST_NOT_READY` *while the creation receipt is still
`submitted`* is a provisioning observation, **not** a terminal failure.
Only after a terminal receipt and bounded readiness checks should a
missing user be classified as a confirmed provisioning defect. A
`succeeded` receipt only acknowledges upstream create; it does not
by itself prove user, SSH, nesting or authorized access.

## New v0.49.4 engineering regressions

After this live incident, source fixes address two workflow pitfalls:
1. Validate the upstream-advertised JSON-Schema type/required/enum/bound
   subset **before** persisting an irreversible create intent. A known
   invalid parameter must not permanently consume an unused Box name.
   This is *not* a substitute for upstream semantic validation.
2. Allow `create_container(username, status_only=true)` as a **read-only
   lookup** of an existing receipt without dispatch, for ChatGPT clients
   whose limited MCP tool catalogue omits `boxweave_create_status`.
   The dedicated read-only tool remains available in full catalogues.

A lightweight source-CI publication regression protects the automatic
`main` publish job. These v0.49.4 changes have **T/CI evidence only**
until installed separately; do not transfer v0.49.3's host PASS to the
new code without a live check.

## Release and process boundary (issue/PR #90)

The earlier evidence-signing/reviewer workflow PR #90 was closed
**unmerged** by owner decision. The lightweight `main` CI verify →
publish path was restored by PR #96 and successfully published v0.49.2
and v0.49.3 with immutable tags and exact asset checks. It does not
constitute full independent installed acceptance or move the stable
installation pointer. See [current release contract](../process/AGENT_MANAGED_ACCEPTANCE_RELEASE.md).

**Next bounded verification:** On a deliberately authorized v0.49.4
installation, first use invalid typed arguments for a fresh unused
name (must not reserve a task), then a valid one-shot ordinary create
with HTTP status-only polling, exact name/digest duplicate, independent
guest SSH/identity readback, and explicitly record the actual upstream
create count if trace instrumentation becomes available. Preserve
all UNKNOWN identities and never replay a dispatched create.
