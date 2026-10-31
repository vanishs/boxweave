# E4 implementation specification — clone Box / clone from snapshot

**NOT IMPLEMENTED. DESIGN ONLY.** Parent [Box environment proposal](BOX_ENVIRONMENT_PROPOSAL.md). Requires [E0](BOX_ENV_E0_CAPABILITIES.md) and [E2](BOX_ENV_E2_SNAPSHOTS.md) for snapshot-source cloning. [E1 fingerprints](BOX_ENV_E1_FINGERPRINTS.md) optional for before/after drift. Restore E3 is **not** required to create a new Box; clone must not modify the source.

## Product contract

Given an authorized source, create a **new, independent, fully managed Box** from either:
- the current **stopped instance**, or
- an exact **observed snapshot** of that instance.

Do not merely copy the Incus filesystem and pretend the resulting instance is integrated with Containarium. The Provider may require daemon-specific registry entries, per-tenant logical naming, service-specific SSH/trust, network membership, address assignment, new container UUID/created timestamp, and local identity reconciliation. If the deployed Provider cannot perform these operations and verify them atomically enough, the feature **remains unavailable**. A safe refusal is preferable to an unmanageable orphan clone.

## Trust and identity issues

Copying a Box can duplicate guest:
- SSH host keys and known_hosts assumptions;
- machine-id, user identity and runtime hostname;
- Containarium instance registrations, internal daemon/token/DB identity;
- nested Incus trust, storage identities and endpoint addresses;
- secrets in application directory, enrolled Agent configuration and credentials.

**First-release default:** clone is created **stopped and isolated** from normal network access. It cannot be promoted to a usable managed Box until a reviewed, Provider-supported re-identification/re-enrollment operation has completed and the exact destination identity has been independently observed. No "auto start clone" promise; no key material disclosed via status/errors or CLI.

A clone may be private to the **same trusted actor** for personal development, not a sanitized shareable image. Cross-user cloning requires separate tenancy/policy and secret-sanitization design; explicitly out of scope.

## Source/destination plan

1. Read authenticated exact source identity via current Provider and optional snapshot identity via E2. Require source stopped; disallow unknown snapshot contents/attached custom volumes unless scope is explicitly named and approved.
2. Caller selects an **allowed new logical name**, selected project/backend/tenant and reviewed resource capacity. Validate source and destination do not coincide; destination must be absent in *complete* upstream inventory and absent in local nondeleted managed table.
3. Confirm that new target can be registered as normal BoxWeave-managed instance under current `allowed_names` rules (currently capped by `Lifecycle` configuration). Cloning cannot bypass quota or protected core names.
4. Build explicit plan: source immutable fingerprint, snapshot ref if any, **destination reservation**, expected missing state, storage/network capability, rekey method and security restrictions. Expire quickly; one use.
5. On confirmation, atomically reserve destination name and **source operation claim**; avoid conflict with live source lifecycle, restore, snapshot deletion and concurrent other clones.
6. Perform **one** Provider-supported clone operation; persist acknowledgment or unknown. Inspect exact destination source identity, Provider registry, network isolation, SSH host-key status and new immutable created/UUID. Never auto-adopt a foreign or mismatched destination.
7. Perform the **reviewed re-identification** routine only if Provider exposes a separately scoped, auditable contract. If it cannot complete, leave clone stopped/quarantined and label `rekey_pending/unknown`; preserve the original target identity, do not delete the clone automatically.
8. After independent capability and SSH trust checks pass, transition to "managed_stopped_ready" and allow a separately confirmed start/hello test. Retain `source_clone_of` metadata; a clone is not the same target even if stable fingerprint matches.

## Candidate states and metadata

```text
planned -> reserved -> dispatching -> created_unverified
                                   -> quarantined_rekey_pending
                                   -> managed_stopped_ready
                                   -> unknown / failed (only if proven)
```

Use existing lifecycle one-dispatch and "unknown is not retry" invariants. A worker death after the Provider has created the clone means destination may exist. On recovery, inspect the original reserved destination; **never** pick a new name or retry create. Preserve the source. No automatic cleanup of partial clones unless a separate actor-confirmed plan inspects exact target and isolation state.

Candidate `clone_operations` fields: actor, exact immutable source/snapshot ref, new project/logical, expected absent, provider scope digest, source+destination claims, operation state, created identity reference, key-reset/registry verification stage, quarantine state, linked fingerprint. Avoid raw guest secrets or command output.

Do not graft clone actions into `Provider.lifecycle_mutate` by inventing names and passing generic input to `mcp.tool`. Add a narrowly typed Provider contract and fixed schema. If only direct Incus copy is available, do not expose general public clone in BoxWeave app; optional future host-admin-only tooling needs its **own** authorization design and does not constitute completed E4.

## Developer implementation order

1. Implement destination reservation and source/destination claim model; tests for two parallel clones and competing source restore/delete.
2. Implement exact upstream clone capability for instance and snapshot separately; mark unsupported until source-scope/identity contracts reviewed.
3. Add quarantine/provision/rekey state machine. Constrain network and restart; no public routes until identity verified.
4. Link new managed target to existing `Lifecycle.managed`/allowed target model without fabricating immutable identity or weakening actor scope. Record original clone source relation.
5. Add one-use explicit plan/confirmation and observe/reconcile; database state must remain correct after crash at any stage.
6. Add read-only display and optional E1 fingerprint comparison; ensure new instance and source have different identity refs even if software versions/config match.
7. Package source and run isolated disposable nested test; no production instance cloning under this documentation request.

## Validation matrix

| ID | Case | Expected |
| --- | --- | --- |
| E4-01 | Source stopped, clone contract and identity reset supported | separate managed stopped destination after verification |
| E4-02 | Source running or foreign/unbound | refuse, no remote copy |
| E4-03 | Unsupported clone or rekey capability | feature unavailable, never raw Incus fallback |
| E4-04 | Destination name occupied in Provider or local pending ops | reject; no overwrite |
| E4-05 | Two concurrent clone requests to same destination | one reservation, at most one dispatch |
| E4-06 | Source deleted/restored during plan window | stale plan and source claim prevent copy |
| E4-07 | Clone after remote effect but response lost | unknown, observe reserved destination, never retry |
| E4-08 | Duplicate SSH host keys, machine-id, Containarium daemon identity | remains quarantined until reviewed reset/verification |
| E4-09 | SSH trust or Provider registry verification fails | no exposed clone, no automatic delete/recreate |
| E4-10 | Snapshot ID replaced or missing | refused; original source unchanged |
| E4-11 | Nested custom volumes not in copied root | coverage explicit, no "exact copy" claim |
| E4-12 | Cross-actor/project identity/tenant mismatch | fail closed |
| E4-13 | Source fingerprint equals destination config fingerprint | identity still distinct; warning on expected rekey drift |
| E4-14 | Provisioning crash/DB-full/source lock held | no duplicate copy, retained incident/unknown receipt |

**Done** only after separate real disposable validation that a clone can be created from both source and snapshot (if both contracts supported), can be independently managed without duplicate trust identities, and does not mutate or expose the original. Otherwise explicitly ship snapshot/restore/fingerprint without clone and mark E4 **NOT IMPLEMENTED**.

Related [diagnostics implementation proposal](DIAGNOSTICS_PROPOSAL.md) and [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md): these are complementary, **NOT IMPLEMENTED** work; E4 does not depend on D0–D2.
