# E2 implementation specification — managed instance snapshots and checkpoint catalog

**NOT IMPLEMENTED. DESIGN ONLY.** Parent [Box environment proposal](BOX_ENVIRONMENT_PROPOSAL.md). Depends on [E0](BOX_ENV_E0_CAPABILITIES.md) for versioned Provider write authority and [E1](BOX_ENV_E1_FINGERPRINTS.md) for optional before/after fingerprint linkage. Snapshot **restoration** and cloning are separate [E3](BOX_ENV_E3_RESTORE.md)/[E4](BOX_ENV_E4_CLONE.md) changes.

## Scope

Support **list, inspect, create and delete snapshots for exactly one approved, managed, stopped instance**. No automatic scheduled snapshots, no running/stateful snapshot promise, no snapshot->recovered confirmation, no root shell or direct Incus fallback. Existence of a native Incus `snapshot create` command does not imply Containarium MCP has the necessary authenticated API; E0 must prove the deployed contract first.

### Proposed command/API journey

- Read-only list/inspect: exact `actor+source Target`, upstream inventory with bounded IDs and creation/source evidence. Locally stored catalog entries are **not** truth for snapshot existence.
- `plan-create`: select existing managed source, requested label, storage coverage, retained running state, size/quota estimate, full component observation, expected source UUID/creation/current state. Require stopped state; no auto-stop.
- `confirm-create`: re-fetch Target identity, state, capability, pending source claims and plan digest/expiry, require explicit exact name, durably mark dispatch then **one** Provider call.
- `observe`: independent upstream snapshot inventory reconciles exact source and returned snapshot identity. Only mark success after matching observation, otherwise `observing/unknown` with distinct cause.
- `plan-delete/confirm-delete`: exact source+snapshot identity, explicit confirmation and reference check; no clearing unknown or auto pruning snapshots underlying a pending restore/clone or preserved incident evidence.

Keep snapshots as managed *resources* with their own immutable IDs and separate source generation. Do not allow operator to pass arbitrary filesystem paths, Incus project names outside scope or snapshot reuse/overwrite flag. Snapshot naming collisions must fail closed; never use native `--reuse` in v1.

## Write contract and data model

Candidate `checkpoint_plans`, `checkpoint_operations` and `checkpoint_catalog` app-owned tables:
- `plans`: id, actor, operation, source identity, optional snapshot ID/label, expected stopped state, provider scope/config generation, storage coverage, expiry, one-use flag;
- `operations`: op ID, source identity, actor, operation kind, phase/state/deadline, ack flag, error code, immutable expected precondition, result observation;
- `catalog`: exact provider-managed source, upstream snapshot reference, known create/delete result, created and observed timestamp, storage coverage, source fingerprint ID, optional references from clone/restore, expiry and evidence.

Use additive schema migration at **next real version** rather than blindly using diagnostics' proposed v9/v10. A pending/unknown checkpoint operation must conflict with any lifecycle start/stop/delete/resize/create on same logical target. Existing `environment_one_pending` is separate: implement a shared durable guard or equivalent narrowly scoped serializable claim with explicit unit+concurrency tests. Claims spanning two tables must be acquired atomically before remote dispatch; no global lock held while remote call runs.

If remote snapshot succeeds but persistence of its result fails, the outcome remains **unknown**; requery snapshot identity, don't re-create with a different label to hide uncertainty. For deletion, missing listing may be inconclusive if inventory incomplete; do not interpret a transport error as proof of deletion.

## Storage consistency and nested safety

Incus instance snapshot includes its own instance root storage, **not necessarily separately attached custom volumes**. For BoxWeave nesting, a "stopped outer Box" can still reference external Incus storage, mounted host paths, or databases backed outside its root. Check coverage **before** accepting a complete restore-point description.

Required first release policy:
- manual snapshot of a **stopped** managed outer instance only;
- storage scope explicitly stated in plan and catalog; if external/custom mount coverage unknown, either refuse with a typed code or allow only a distinct non-restorable "instance-root-only" research operation after an additional explicit owner decision;
- record source provider/backend/storage driver and inspected capability, but no raw mount paths or root config;
- no automatic copy of host-level nesting configuration, SSH known_hosts, provider JWT or secrets outside instance;
- no automatic deletion of pre-upgrade snapshots or incident snapshots; quota/existing retention policy must be reviewed.

Do not present snapshot as independent backup: same storage pool failures can destroy both source and snapshots. Separate export/off-host backup is explicitly out of E2 scope.

## Developer work order

1. Implement strict snapshot-label/reference validation, capability gate and typed Provider contract in `bwmini/checkpoint_provider.py` (or small extension `provider.py`).
2. Implement app-owned `CheckpointService` state/plan/receipt/observe in `bwmini/checkpoints.py` plus SQL migration and index, **without** modifying existing `Lifecycle` action IDs/contracts. If locking existing lifecycle needs coordination, make a small shared serialization change with regression tests.
3. Add read-only snapshot list/inspect via authorized `__main__.py`/`web.py` domain endpoints; require actor/project checks. Do not expose arbitrary MCP calls.
4. Add plan+confirmed create and separately plan+confirmed delete; verify no unbounded retries.
5. Optionally record `E1` pre-observation capture and checkpoint relation as `pre_capture` (not point-in-time equal) after successful snapshot validation.
6. Verify release inclusion (`distribution.py`) and actual installed code/Provider schema in disposable test Box. User-facing CLI/Web docs only after test.

## Tests

| ID | Case | Required |
| --- | --- | --- |
| E2-01 | Missing/malformed upstream capability | no write, `CAPABILITY_UNAVAILABLE` |
| E2-02 | Snapshot while running | refused; no implicit stop |
| E2-03 | Exact target changed after plan | stale plan rejected |
| E2-04 | Foreign/core target or unsupported project | rejected before remote |
| E2-05 | Storage custom volumes outside snapshot coverage | false "complete backup" impossible |
| E2-06 | Simultaneous lifecycle stop/resize and snapshot plan | one target claim, no race |
| E2-07 | Provider succeeds but response lost | unknown, no re-dispatch on restart |
| E2-08 | Idempotent reconcile of a named snapshot | inspection-only; accurate source/identity |
| E2-09 | Snapshot deletion while referenced or incident preservation requested | fail closed |
| E2-10 | DB full before confirmed dispatch | no Provider write |
| E2-11 | DB full after remote effect | unknown receipt, no auto retry |
| E2-12 | Provider from supported and unsupported storage drivers | actual acceptance separately documented |
| E2-13 | Create and delete while other project's Box same name | strictly scoped, no cross-project change |

Completion: a stopped test Box snapshot is confirmed by independent inventory, bound to an exact source, with correct storage limitations. If snapshot API doesn't exist in the deployed Provider, **stop at E0**; don't bypass security to meet schedule. Next [E3 restore](BOX_ENV_E3_RESTORE.md) and [E4 clone](BOX_ENV_E4_CLONE.md).
