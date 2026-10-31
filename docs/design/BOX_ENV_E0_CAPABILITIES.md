# E0 implementation specification — Provider capability and boundary contract

**NOT IMPLEMENTED. DESIGN ONLY.** Parent: [Box environment proposal](BOX_ENVIRONMENT_PROPOSAL.md). Source review: `main@d72035837da1170d2fe5201ee4ea65075ea038ee`; see [diagnostics research](../verification/DIAGNOSTICS_SOURCE_AUDIT.md) for related but separate failure analysis. **E0 performs read-only capability discovery and tests, no snapshot/restore/clone writes.**

## Objective

Give all later batches one trustworthy answer to "Does this exact authenticated Provider/backend/project/actor currently support this operation, on this resource, with the required safety guarantees?" Neither Incus built-in commands nor a tool name in Containarium documentation is sufficient.

Use one typed app-side adapter (`bwmini/provider.py` extension or a narrowly named `checkpoint_provider.py`) over the **already configured** `bwmini/mcp.py` client. Do not create a generic arbitrary tool bridge, second MCP, privileged Incus fallback or "auto-fix" of an absent Containarium capability.

## Current code map

- `Provider.session` creates negotiated MCP client; `Provider.lifecycle_inspect` validates name/created identity from `get_container`, `Lifecycle.check_name` restricts operator targets to configured `allowed_names`, and `Lifecycle.scope` fingerprints Provider/launcher policy.
- `Lifecycle.preflight` validates source target before start/stop/delete; reuse its immutable identity semantics, but **do not** mislabel a newly cloned identity as previously managed.
- `host_cli.inspect` performs privilege-gated exact local Incus inspection, but is a separate root-owned host boundary. It is not permission for app Web/Agent to execute Incus.
- Current `mcp_gate.catalog` augments `create_container` with nesting; it is NOT a general snapshot/clone adapter. Installed gate and Provider host may be split.

## Suggested narrow interfaces

```python
class CheckpointProvider:
    def capabilities(self, *, actor_scope, source_target) -> CapabilityReport: ...
    def inspect_snapshot(self, *, source_target, snapshot_ref) -> SnapshotObservation: ...
    def list_snapshots(self, *, source_target) -> list[SnapshotObservation]: ...
    # Mutating methods defined but NOT implemented/exposed in E0:
    # create_snapshot / delete_snapshot / restore_snapshot / clone_instance
```

Candidate `CapabilityReport`: `source_version` (actual observed, if supplied), `protocol`, `provider_scope_digest`, `target_identity_ref`, `snapshot_create/list/delete/restore`, `clone_instance/clone_snapshot`, `rekey_registry`, `storage_coverage`, `observed_at`, `evidence`, `reason_code`. States `supported / unsupported / not_verified`. Missing/unknown schema = `not_verified`; never default to supported.

Capability binding must be **source, authorization, and version aware**: values obtained during a verified read are not global permission tokens and must be checked fresh at plan confirmation/dispatch. Verify exact tool `inputSchema`/`output` or bounded observed response, server info, relevant scopes, backend identity. Privileged clone must additionally demonstrate fresh identity, registry and SSH trust integration. A non-mutating probe cannot prove actual write success: report `contract_supported` separately from `live_verified`.

## Source/target identity and concurrency

Candidate immutable target identity tuple: `provider, backend, project, logical, native name, created timestamp` from `Target`, plus verified upstream instance UUID if independently provided. Do not trust arbitrary user-provided UUID. Refuse foreign/core entries, unbound source, pending/unknown conflicting lifecycle operations, pending original-instance create/delete/resize and source identity mismatches. Keep source `project` and `actor` scoped.

Before any future operation reserve a typed **source lock** that covers checkpoint operations and existing lifecycle operations. The current `environment_one_pending` unique index only guards `environment_operations` by logical name; it does **not** guard a new table automatically. E0 must design a shared per-instance claim/serialization mechanism, with DB lock ordering and crash-recovery tests, before E2/E3/E4 introduce writes. A stale lock must not be dropped merely because an operation is `unknown`.

## Storage and quiescence decision

Probe provider storage backend support; identify whether source root volume, custom storage volumes and external mounts are **in coverage** or `not_verified`. Never read entire mount paths/config blobs into public results; classify by reviewed enum and count. If a complete restorable snapshot cannot be guaranteed, block such a claim and expose a precise limitation. Incus root snapshot may exclude attached custom volumes. Stop/quiesce must be explicit and externally observed, not inferred from a guest Web 200 or a running container alone.

Do not assume nested Incus internal volumes live inside the outer Box root disk; validate in a sandbox. Stopping an outer Box may interrupt nested Incus/DB; still check storage consistency separately.

## Implementation order

1. Add capability/test fixture model with strict input/output validation (read-only), no extra MCP tool.
2. Test existing `list/get_container` and unsupported snapshot discovery with a fake MCP server.
3. Verify exact identity/scope validation and a source serialization strategy across existing `Lifecycle` and new feature actions.
4. Add an internal capability/introspection route only if it can use existing authenticated app domain access; no new generic host endpoint.
5. Document deployment-specific signed/reviewed upstream capabilities in a dedicated acceptance report after **separate** live approval.

## Required tests

| ID | Case | Expected |
| --- | --- | --- |
| E0-01 | Upstream MCP has no snapshot/clone tools | `unsupported`; no Incus fallback |
| E0-02 | Tool name exists with wrong schema/capabilities | `not_verified`; no write path |
| E0-03 | Provider version / current scope changed between plan and confirm | refuse stale plan |
| E0-04 | Target exists but unowned, or mismatches immutable identity | refuse |
| E0-05 | Core infrastructure target, cross-project or actor mismatch | refuse |
| E0-06 | Snapshot operation conflicts with pending/unknown lifecycle op | source claim prevents race |
| E0-07 | Custom volumes not covered or storage backend unknown | explicitly `not_verified` / block full snapshot claim |
| E0-08 | Malformed/oversized MCP response or timeout | bounded safe code; never display token/raw tool bytes |
| E0-09 | Gate host differs from Provider host | no implicit local Incus authority or fabricated identity |
| E0-10 | Read-only discovery invoked repeatedly | no host, DB schema, runtime, client registration or Provider mutation |

## Go/no-go and next batch

E0 succeeds as **read-only contract design/implementation** if selected authenticated Provider version and local deployment can be classified without side effects, while unsupported capabilities fail closed. Do not ship snapshot actions until a real Provider-side contract is independently designed and verified. Next: [E1 fingerprints](BOX_ENV_E1_FINGERPRINTS.md) (read-only), then [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md) (requires approved write contract).

Related [D0–D2 diagnostics proposal](DIAGNOSTICS_PROPOSAL.md): all **NOT IMPLEMENTED**, not an E0 prerequisite.
