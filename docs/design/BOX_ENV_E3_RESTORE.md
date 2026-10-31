# E3 implementation specification — explicit restore of a managed checkpoint

**NOT IMPLEMENTED. DESIGN ONLY.** Parent [Box environment proposal](BOX_ENVIRONMENT_PROPOSAL.md). Requires [E0 capability](BOX_ENV_E0_CAPABILITIES.md) and [E2 snapshot catalog](BOX_ENV_E2_SNAPSHOTS.md). Optional [E1 fingerprint diff](BOX_ENV_E1_FINGERPRINTS.md) gives before/after context; [D2 diagnostics](DIAGNOSTICS_PROPOSAL.md) is NOT IMPLEMENTED and is not a prerequisite.

## What restore means

Restore overwrites the **current source instance state** with a selected checkpoint belonging to that **same exact managed instance**. It is a potentially destructive change; restoring the outer Box can discard new code, installer state, nested Incus data and incident evidence. A storage snapshot is not a complete rollback for external/custom volumes and independent provider/host identities.

E3 v1 should only support a **stopped exact source Box** and explicitly approved snapshot coverage. No automatic stop/start, no restart of outer/inner components, no general "undo all changes" promise, and no restore when another lifecycle/checkpoint operation is pending or unknown.

## Operator flow (proposed)

1. Select managed exact source and a snapshot whose provider inventory proves ownership and storage coverage; do not trust catalog-only record.
2. Run `plan-restore`: authenticated Provider read of current source UUID/creation/project/backend/status, snapshot exact identity/version/creation/source, effective capabilities and storage dependencies.
3. Render an impact preview: named source, chosen checkpoint, data since snapshot that may be lost, incomplete external-volume coverage, no automatic host setting reset, downtime requirement and evidence preservation option.
4. Record optional immutable E1 **pre_restore** fingerprint and recommend manual retained incident report. These do not constitute a guaranteed backup.
5. Operator explicitly confirms plan ID + exact source identity and checkpoint label; revalidate fresh state/identity, one-time confirmation and source claim in an atomic transaction.
6. Durably mark `dispatching`, then call one typed Provider restore operation *once*, outside the transaction.
7. Read exact source/snapshot inventory, target immutable identity, storage status and optional post_restore fingerprint. Verify meaningful success only when independently observed; return `observing/unknown` if provider response or verification is ambiguous. **Never reissue restore automatically**.
8. Leave source stopped unless a separately confirmed start action follows.

## Proposed mutation record

```json
{
  "operation_kind": "snapshot_restore",
  "state": "queued",
  "source_identity_ref": "exact-managed-target",
  "snapshot_ref": "verified-owned-snapshot",
  "expected_source_state": "stopped",
  "coverage": "instance_root_only_or_verified_supported_scope",
  "scope_digest": "current-provider-and-policy",
  "confirmation_required": true,
  "auto_restart": false
}
```

Example only. In DB maintain original source immutable identity, checked snapshot identity and independently verified generation of provider/storage/permissions at time of plan; no reusable "force" flag. Re-read before dispatch: if source UUID/creation, existing nested state, allowed names, provider auth, storage driver or snapshot identity differs, reject as `STALE_RESTORE_PLAN`. Identity change after restore is a **separate security event** and cannot be automatically adopted.

Observations may say snapshot was restored but guest boot health unknown; keep `storage_state`, `source_identity` and `guest_readiness` as separate fields. A successful Provider response is *not* proof the nested inner BoxWeave service was started or that parent `security.nesting` was preserved.

## Failure branches

| Boundary | How to handle |
| --- | --- |
| Snapshot not found under exact source | reject before dispatch, no restore |
| Source running | reject; no auto-stop |
| Current Box or snapshot provenance changed | stale plan; require new plan |
| Custom volume/mount coverage unverified | block "complete restore" contract; no silent gap |
| Provider reported known rejection before effect | fail without retry |
| Timeout/connection interrupted after submission | `unknown`, preserve exact target and evidence |
| DB crash between observed remote effect and update | retain `unknown`; on restart only inspect |
| Outer source restored but inner services fail | storage restore ≠ workload health, diagnose separately |
| Restore reverts inner guest SSH/registry state | do not weaken pin or re-enroll automatically |
| Snapshot restoration conflicts with clone/create on same source | shared source claim denies |

**No automatic pre-restore snapshot:** a backup checkpoint may have cost/volume implications and cannot be assumed consistent. If user chooses to create one, treat it as separate explicitly confirmed E2 operation and independently verify success before continuing.

## Developer implementation

- `CheckpointService.plan_restore / confirm_restore / observe_restore` in `bwmini/checkpoints.py`, with strictly typed capability/actor/target/immutable snapshot checks;
- `CheckpointProvider.restore_snapshot` only when E0 proves authenticated backend contract; do not call arbitrary Incus command inside app.
- Integrate planned restore with shared lifecycle resource claim and existing Worker recovery semantics: no remote effect inside transaction, no automatic replay of unknown, no co-scheduling with lifecycle mutations.
- If E1 installed, use pre/post capture reference types; if D1/D2 later installed, expose optional operation ID trace link without adding dependency.
- Add minimal user-facing read-only plan and consent forms, then confirmed action API; no hidden "restore as part of upgrade" automatic flow.
- Package, schema/migration backup, owned repair documentation and separately authorized real nested test.

## Tests and release gates

| ID | Case | Acceptance |
| --- | --- | --- |
| E3-01 | Exact stopped source and observed matching snapshot | succeeds only after confirmation and independent read |
| E3-02 | Source running or external custom volume uncovered | blocked; no Provider restore |
| E3-03 | Source renamed/recreated but same logical name | mismatch blocks restore |
| E3-04 | Snapshot deleted/replaced after plan | stale plan refuses |
| E3-05 | Competing lifecycle/clone/snapshot op | claims serialize; no races |
| E3-06 | Timeout after remote restore | unknown, **no second call** |
| E3-07 | Worker killed during/after Provider effect | original receipt remains investigable |
| E3-08 | Failed post-restore inner MCP/SSH | storage proof and guest readiness kept separate |
| E3-09 | Provider changed backend capabilities or scope | stale plan refused |
| E3-10 | DB full and invalid actor/snapshot identifiers | no remote effect |
| E3-11 | Preserve unrelated host/custom-volume data | explicit evidence/no false rollback claim |
| E3-12 | Valid restore + optional E1 before/after capture | relates captures with distinct times, never atomic equality |

Complete only after a separately approved real stopped-instance restore in an isolated nested dev environment with exact source inventory and before/after observations, plus all no-replay/crash tests. No restore against an original production or operator Box is authorized here. Next [E4 clone](BOX_ENV_E4_CLONE.md).
