
# Proposal: Box checkpoints, cloning and environment fingerprints

**Status: NOT IMPLEMENTED / DESIGN ONLY.** The feature is **not part of the currently approved product scope**, is not an accepted ADR, and does not authorize host/Provider/storage/network changes. Research source baseline: BoxWeave §main@d72035837da1170d2fe5201ee4ea65075ea038ee§ (version 0.21.4). The owner uses **both reused test Boxes and fresh test Boxes**; a common development topology is outer BoxWeave -> one Box containing an inner BoxWeave, Containarium and Incus -> nested test Box. This is a priority use case, **not** a restriction to developer-only functionality.

**Related independent research, both NOT IMPLEMENTED:** [diagnostics source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md) (S/D observations) and [diagnostics implementation proposal](DIAGNOSTICS_PROPOSAL.md) (D0/D1/D2 proposed code work). These documents do not already deliver snapshots, clones, fingerprints, D0, D1 or D2. The checkpoint features must operate without any diagnostic pipeline installed; links to diagnostic evidence are additive.

## 1. Product intent

A user should safely:
1. Capture an allowlisted **environment fingerprint** and compare two captures, even if the inner BoxWeave/Provider is unhealthy.
2. Make, inspect and delete a **checkpoint/snapshot** of an exact managed Box without collecting secrets or misleading the user about external volumes.
3. Restore a selected checkpoint **only after an explicitly confirmed plan**, preserving evidence and avoiding automatic retries of an unknown remote result.
4. Clone an existing Box or one of its checkpoints into **a new, independently managed Box**, with fresh identity and correct Containarium registration/tenant/network/SSH boundaries.

Checkpoint, clone and fingerprint are three sibling capabilities under a proposed "Box checkpoints & environments" product area. A **snapshot** is a point-in-time representation of a *source* instance; **restore** modifies that source; **clone** creates a distinct target; a **fingerprint** is a small observation record, neither an image, backup nor authorization credential.

The ordinary-user feature and the developer workflow should be the same underlying system. Do not introduce a developer-only root shell, hidden MCP gateway or standalone Debug MCP to implement these features.

## 2. Actual source boundaries and upstream facts

| Question | Reviewed source / fact | Implementation constraint |
| --- | --- | --- |
| Who owns business state? | §bwmini/store.py§ private SQLite, currently user_version=8; §lifecycle.py§ uses plan/confirm, actor binding, durable one-time dispatch, managed target mapping and reconciliation | Do not create an independent truth store or reuse an unknown/expired write |
| How are boxes controlled? | §bwmini/provider.py§ §lifecycle_mutate§ handles create/start/stop/delete/resize and checks discovered MCP input schemas | **No reviewed BoxWeave snapshot/restore/clone Provider adapter currently exists** |
| Where is the MCP boundary? | §bwmini/mcp.py§ application-owned stdio; §mcp_gate.py§ restricted direct-Agent SSH bridge with nesting overlay | Neither user MCP access nor local root access automatically grants snapshot authority |
| What can outer CLI see? | §host_cli.py§ §status§/§provenance§ and exact Incus §box inspect§; §online.py§ installs root-owned helpers separate from app code | Reuse narrow read-only observations; do not run app-controlled code as root |
| What is a managed identity? | §domain.Target§ and §Lifecycle.managed§ / §preflight§ use exact provider/backend/project/logical/name/created identity; allowed_names are explicitly configured | Do not treat a matching name alone as authorization; cloning needs a **new** verified managed identity |
| What does nested-capable mean? | §nesting.py§ observes Incus config and signature; §docs/NESTING.md§ distinguishes parent config from verified nested execution | A cloned outer Box may require independent parent nesting configuration and restart/effective test |
| Can Incus do it? | Official Incus commands support instance snapshot create/restore and §incus copy SOURCE[/SNAPSHOT] NEW_NAME§ | Underlying **Incus capability is not proof the deployed Containarium MCP exposes a safe contract** |
| Is snapshot a complete backup? | Official Incus documentation says snapshots live in the same pool and exclude separate custom storage volumes | Quiescence, data location and storage-driver suitability are explicit preflight inputs |

External upstream references (public, subject to installed-version checks):
- [Incus instance backups / snapshot semantics](https://linuxcontainers.org/incus/docs/main/howto/instances_backup/)
- [Incus copy command and snapshot source](https://linuxcontainers.org/incus/docs/main/reference/manpages/incus/copy/)
- [Incus custom volume backups](https://linuxcontainers.org/incus/docs/main/howto/storage_backup_volume/)
- [Containarium platform and MCP overview](https://github.com/FootprintAI/Containarium)
- Existing BoxWeave [upstream ownership boundary](../UPSTREAM_PROVIDER.md) and [nesting verification](../verification/NESTING_VERIFICATION.md).

No live Containarium §tools/list§, specific deployed source/version or Incus storage pool was inspected in this design session. Therefore all provider snapshot/clone support is §not_verified§ until a separately authorized capability+version test is recorded.

## 3. Control plane and failure independence

§§§text
External operator / Agent
     |
     +--> Outer BoxWeave (trusted controller and metadata store)
              |
              +--> Existing app Provider adapter -> authorized Containarium MCP
              |                                     -> Incus managing OUTER test Box
              |
              +--> Checkpoint/fingerprint metadata (outer app-owned private store)
              |
              +--> OUTER test Box [inner BoxWeave + Containarium + Incus]
                           |
                           +--> Inner work/test Box

Source instance owner = OUTER Provider, never implicitly the inner one.
§§§

A snapshot of the **outer test Box** is requested by the **outer** BoxWeave through its authorized Provider. Its metadata must remain outside the snapshot. If inner BoxWeave fails, the outer layer can still inspect/restore the exact outer instance, subject to actual external MCP/Provider availability. If the outer Provider fails, a stored metadata record is **not** proof that a snapshot exists. No automatic traversal to an Incus parent or unrelated host.

The direct-Agent MCP gate and app Provider adapter are separately authorized paths; the first release must not add tools to the unrestricted direct-Agent gateway. If a future gateway tool is considered, require a new scope-specific review (as with diagnostics D3). No generic §incus exec§, §incus config set§, shell, arbitrary REST URL or path from the user/API.

## 4. Capability discovery and mandatory preflight

Introduce a **bounded typed capability probe** over the existing application Provider session:

§§§json
{
  "feature": "box_checkpoint_v1",
  "provider_source": "observed_current_session",
  "state": "unsupported",
  "operations": {
    "snapshot_create": "not_verified",
    "snapshot_list": "not_verified",
    "snapshot_delete": "not_verified",
    "snapshot_restore": "not_verified",
    "clone_from_instance": "not_verified",
    "clone_from_snapshot": "not_verified",
    "identity_reseed": "not_verified"
  },
  "storage_consistency": "not_verified"
}
§§§

**Illustrative schema, not current output.** Each operation needs its own verified tool schema, output contract and authorization. Presence of a tool **name** alone is insufficient: verify the actual installed upstream release, input schema, credential scope, backend support, state restrictions and exact returned identity. If not supported, return §CAPABILITY_UNAVAILABLE§ and leave the source untouched. Do **not** install/upgrade/patch Containarium or fall back to direct host Incus/REST without a separate approved adapter, authority and test plan.

Minimum preflight:
- source is an application-managed target by actor+provider+backend+project+exact immutable identity; no core/provider infrastructure target, foreign resource or simply any visible Box;
- selected action is allowed by current actor/policy; target and snapshot names meet strict independent validators;
- operation has no competing pending/unknown lifecycle/checkpoint action for the same source or destination; no concurrent start/stop/delete/resize during restore;
- provider/storage backend and storage pool can safely perform requested operation, with explicit custom volume/host mount enumeration and unknown handling;
- instance state and **application quiescence** are observed; for initial snapshot/restore/clone recommend **stopped outer Box only**, with explicit consent to downtime;
- exact parent/nesting capabilities and target identity are rechecked immediately before dispatch, not inferred from remembered fingerprint.

A stopped outer Box is an operational consistency improvement, not a guarantee that all independent custom volumes/database backends are snapshotted. Unverified external volumes/mounts block any command that claims a **complete, restorable** checkpoint. A limited instance-only capture may be considered only if clearly scoped and approved, never branded a full environment recovery point.

## 5. Unified operations and unknown-result safety

Adopt the lifecycle pattern: **plan -> explicit confirmation -> durable one-time dispatch -> observe/reconcile**, with explicit operation kind and subject. No side-effectful Provider call in a SQLite transaction. Persist exact source identity and, for restore, selected immutable snapshot identity plus observed generation/config fingerprint at plan and pre-dispatch.

### Proposed state vocabulary

§§§text
planned -> queued -> dispatching -> observing -> succeeded
                                      |            ^
                                      v            |
                                    unknown --(authorized observation only)
                                      |
                                     failed (ONLY when definite)
§§§

§unknown§ does **not** authorize automatic re-dispatch. A lost response after an Incus snapshot/clone/restore may mean the remote operation succeeded. Observe snapshot inventory/source state and exact destination before deciding; no name-based duplicate create, restore or deletion. For restore, a failed network response with unknown completion must never suggest "just restore again" because data may already have changed.

Candidate operations §snapshot_create§, §snapshot_delete§, §snapshot_restore§, §clone_instance§ and §clone_snapshot§ are **not** added to §Lifecycle§'s current allowed action list without a separate reviewed feature module. Reuse shared actor/receipt patterns but avoid breaking existing lifecycle plans or schema. Query reads (§snapshot_list§, §fingerprint_capture§, §fingerprint_diff§) remain distinct and read-only.

### Atomicity limits

A fingerprint capture before snapshot and a storage snapshot are not generally atomic; link them with §relation=pre_capture§ and §relation=post_observation§, **not** §identical_point_in_time§. Snapshot reported successful by Provider is separate from an independently inspected snapshot inventory and a later test restore. A fingerprint hash is not a signed attestation, and a configuration match is not proof of runtime readiness.

## 6. Metadata design (proposed; not SQL frozen)

Possible tables in the **application-owned** private SQLite:
- §checkpoint_operations§: id, actor, kind, source target identity fingerprint, optional destination identity, snapshot identity, plan scope/revision, phase, code, deadline, acknowledged, status;
- §checkpoint_catalog§: source immutable identity, upstream snapshot identifier, creation/inspection evidence, recorded storage coverage, linked fingerprint capture, retention/expiry and deletion state;
- §fingerprint_captures§: immutable capture ID, actor, target immutable identity, schema, capture timestamp, source scopes, canonical config hash, runtime hash, provenance/coverage and allowlisted observed data;
- §checkpoint_links§: checkpoint ID, capture ID, relation (§pre_capture§, §post_observation§, §restored_from§, §clone_source§), evidence quality and operation ID.

Migrations must be **additive** and coordinated with diagnostics D1/D2: current v8, diagnostics *proposes* v9 and potentially v10, so **do not hard-code v9/v10 for checkpoint tables**. Assign the next available version from the *actual* implementation branch only after checking schema progression; older releases must reject unsupported future schemas. Preserve app-state backup/upgrade behavior and old receipt histories.

Fingerprint data schema v1, two independently hashed categories:

§§§json
{
  "schema_version": 1,
  "capture_id": "opaque-id",
  "target": {"provider": "observed", "project": "observed", "identity_ref": "opaque-target-ref"},
  "sources": {
    "outer_provider": "observed",
    "guest_boxweave": "not_verified"
  },
  "stable": {
    "image_ref": {"state": "observed", "value": "reviewed-image-ref"},
    "security_nesting": {"state": "observed", "value": true},
    "boxweave_installed_version": {"state": "not_verified"},
    "containarium_version": {"state": "not_verified"}
  },
  "runtime": {
    "instance_state": {"state": "observed", "value": "stopped"},
    "mcp_read": {"state": "not_checked"}
  },
  "hashes": {"stable_sha256": "derived", "runtime_sha256": "derived"}
}
§§§

Illustrative only. Hash canonical strictly typed data including schema version, provider scope and documented normalization version; never hash/store raw config objects, public/private keys, JWT, environment variables, full Instance expanded devices/config, logs or arbitrary tool output. Canonical representation must preserve **unknown** versus **absent** versus **false**, with explicit source quality. Normalize volatile PID/IP/clock fields into a separate observed runtime category (or exclude) so trivial drift is not shown as stable-configuration changes.

Comparison returns typed §changed§, §same§, §added§, §removed§ and §unverifiable§ **per field**, plus coverage differences. A different target identity is a warning and cross-Box comparisons must not turn clone rekeying into a configuration regression. Show cause as §not_verified§ unless directly supported by an independent D0/D1/D2 event.

## 7. Restore and clone security policy

### Snapshot and restore

- Default v1 **stopped-instance** checkpoints; no transparent live/stateful snapshot promise for nested Incus or CRIU.
- All mutations require a plan that shows exact Box, snapshot, storage/mount coverage, downtime and irreversible effect. Restore requires exact explicit confirmation and a separately selected source/target fingerprint check.
- Before restore, retain a verified pre-restore capture of observed state and recommend **separate** operator-approved evidence preservation. A pre-restore snapshot may be optional only after independently confirming sufficient space/capability; do not silently create one.
- Restore target is the **same exact immutable instance**, not an automatically replaced instance. Validate UUID/creation/revision after operation; if identity changes unexpectedly, do not adopt it silently.
- No rollback guarantee for custom volumes, external services, SSH client-side state or parent nesting changes.

### Clone

- A **new Box** must have a new verified Provider-managed identity; cloning native storage bytes is insufficient if Containarium holds separate registry, Tenant, SSH trust and backend metadata.
- Clone source must remain unchanged. Destination must be a reserved, allowed name under the current actor/project with independent resource budget.
- Copying a guest can duplicate SSH host keys, users, daemon tokens, database identities, cloud-init/machine-id, Incus nested server IDs, addresses and application secrets. A clone must start **quarantined/stopped** until a reviewed identity re-seed and trust integration succeeds; never start a byte-identical clone on the same reachable network by default.
- If the Provider does not expose an **authorized clone plus identity re-seed/registration** contract, mark cloning unsupported in v1; do not bypass Containarium with direct §incus copy§ and then pretend the result is managed.
- No automatic sharing of clone credentials or private data across user scopes; copied private project data remains private to the originating authorized operator. A clone is a development environment, not a portable shareable sanitized image.

## 8. Interaction with diagnostics and recovery

| Existing/proposed diagnostic | What the new feature may use | What it must NOT assume |
| --- | --- | --- |
| Existing §boxweave provenance/status/box inspect§ | host/public source and exact external Incus observation when independently authorized | running app version or external parent trust |
| D0: truthful health (NOT IMPLEMENTED) | optional before/after Web, worker and MCP status | that a D0 "pass" proves nested execution, snapshot consistency or restore |
| D1: diagnostic trace (NOT IMPLEMENTED) | optional link from checkpoint operation ID to trace ID after schema/security review | transport timeout proves snapshot failed |
| D2: bounded report (NOT IMPLEMENTED) | optional report attachment to checkpoint/clone/fingerprint comparison | generic host logs may be exported or a report itself repairs |
| Nested environment verification | outer control-plane capture + optional *explicit, authenticated* inner guest attestation | guest can inspect or control parent host implicitly |

**No hard dependency on D0-D2.** If diagnostics are absent, checkpoint actions still return precise, honest receipt states and fingerprint capture still records sources/coverage. If the inner Box is broken, its guest-side version/health is §not_verified§, while the outer control-plane can still capture verified source instance data if reachable.

## 9. Product UX and candidate APIs (not implemented)

Suggested future CLI / Web flow (syntax illustrative):
- §boxweave box fingerprint capture NAME --project PROJECT§;
- §boxweave box fingerprint diff CAPTURE_A CAPTURE_B§;
- §boxweave box checkpoint list NAME§;
- §boxweave box checkpoint plan-create NAME --label LABEL§ then §confirm PLAN --exact NAME§;
- §boxweave box checkpoint plan-restore NAME --snapshot SNAP_ID§ then §confirm PLAN --exact NAME§;
- §boxweave box clone plan SOURCE --snapshot SNAP_ID --new-name NAME§ then §confirm PLAN --exact NAME§.

The currently installed §host_cli§ is **read-oriented root-owned tooling**, while §bwmini.__main__§ handles app-side domain operations. Do **not** paste the illustrative commands into §host_cli.py§ so that it bypasses existing app/Provider authorization. Decide actual CLI placement, actor identity and Web API under a reviewed boundary; tests must freeze commands and error shapes before public documentation. Default actions are nonmutating until explicit plan confirmation.

## 10. Non-functional gates and verification criteria

Required outcomes:
- precise target authorization, no foreign/core target manipulation, no arbitrary local Incus shell;
- no source mutation for fingerprint/list/diff/preflight;
- exact one-time external dispatch for each confirmed checkpoint/clone/restore; unknown results do not retry;
- bounded private storage and read-only comparison, retention accounting and no raw secret capture;
- instance identity and storage coverage verified before claiming success;
- clone independently registered and rekeyed **before** network exposure or regular use;
- test against supported different Incus storage drivers and external/custom volume cases;
- operator can recover diagnostic value even when inner BoxWeave is down (outer observer remains the controller).

Testing ladder: source-level negative mocks -> durable SQLite fault/crash tests -> exact package install in approved disposable environment -> stopped outer Box nested-install scenario -> snapshot/clone/restore with real storage and controlled breakages. Record exact observed version, installed Containarium/MCP schema, Incus driver, target identity, before/after inventory, result, remediation and non-tested claims. No live acceptance is implied by this proposal.

## 11. Open decisions before coding

- Whether the required deployed Containarium can provide a safe versioned snapshot/clone API, and the precise tenant/actor scope to expose; **block writes until proven**.
- Storage scope: instance root volume only versus separately controlled custom volumes. Never name an incomplete capture a full recoverable environment.
- Safe stopped-instance quiescence and restoration semantics for running nested Incus/PostgreSQL.
- Destination clone identity reset: which Containarium/SSH/guest identities must be regenerated, and what exactly authorizes adoption.
- Retention, bytes/count quotas, TTL and whether checkpoint deletion is allowed while clone references remain.
- CLI versus Web release order, and which features are available to a non-root BoxWeave app with no privileged host access.
- Feature's additive schema migration version given D1/D2 implementation order, and namespacing with existing pending lifecycle work.

Record approved outcomes in normative product scope/REQ/AC/ADR only **after** owner approval. See [OPEN_QUESTIONS](../knowledge/OPEN_QUESTIONS.md).

---

## 12. Development batches — source of implementation handoffs

Each linked document is a **design/specification only; NOT IMPLEMENTED**. All five are staged in one architecture plan so programmers can develop small, reviewable commits. Batch selection is a proposal, not permission to mutate a host.

| Batch | Deliverable | Dependency | Go/no-go |
| --- | --- | --- | --- |
| **E0** | [Provider capability contract, identity and security preflight](BOX_ENV_E0_CAPABILITIES.md) | existing architecture | No real mutation; pin upstream capability or block write |
| **E1** | [Environment fingerprint capture and deterministic diff](BOX_ENV_E1_FINGERPRINTS.md) | E0 read-only contract | Stable/volatile schema, no secrets, honest unknown |
| **E2** | [Stopped-Box snapshot/catalog/create/delete](BOX_ENV_E2_SNAPSHOTS.md) | E0+E1 | storage coverage, durable receipts and exact identity |
| **E3** | [Explicit snapshot restore and reconciliation](BOX_ENV_E3_RESTORE.md) | E2 | stop/quiescence, fresh plan, no unknown-result replay |
| **E4** | [Clone current Box / clone snapshot](BOX_ENV_E4_CLONE.md) | E2; E1 optional metadata | provider-managed new identity and rekey, no exposed duplicate |

**Backward cross-references:** [diagnostic proposal (D0-D2, NOT IMPLEMENTED)](DIAGNOSTICS_PROPOSAL.md) and [diagnostic source audit (S/D research, NOT a feature implementation)](../verification/DIAGNOSTICS_SOURCE_AUDIT.md). The linked diagnostics documents must reciprocally label this independent area as **NOT IMPLEMENTED**; no completed Phase/ADR/REQ state is asserted by these links.
