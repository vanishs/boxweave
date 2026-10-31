# E1 implementation specification — capture, storage and environment diff

**NOT IMPLEMENTED. DESIGN ONLY.** Parent: [Box environment proposal](BOX_ENVIRONMENT_PROPOSAL.md). Dependency: [E0 capability/read-only boundaries](BOX_ENV_E0_CAPABILITIES.md). Independent of the unimplemented [D0-D2 diagnostics proposal](DIAGNOSTICS_PROPOSAL.md).

## User goal and source boundaries

For one managed Box, capture a **bounded, immutable, read-only** environment observation and later compare two captures. Supports repeated upgrades on an existing test Box and comparison between a new clone and a known good baseline. Fingerprint is not a backup, stable identity, secret or proof that a system is ready.

Outer BoxWeave captures authoritative outer Provider/Incus observations. Optional guest-side BoxWeave `provenance`/`doctor`/versions may be included **only** over an existing authorized guest identity. Failure of inner BoxWeave must not prevent collection of an outer-only partial fingerprint.

Current usable sources:
- `host_cli.py` `provenance/status/box inspect`, with their separate host permissions and narrow evidence;
- `Provider.lifecycle_inspect`, existing Provider MCP discovery and verified Target identity;
- `nesting.environment/observe` has local/parent restrictions; do not infer parent from guest hostname;
- D0 (proposed) may improve checks, D1/D2 (proposed) may correlate traces, but none is required.

## Data contract

Split schema into **stable configuration** and **volatile state**, with explicit per-field evidence. Example:

```json
{
  "schema_version": 1,
  "capture_id": "opaque-uuid",
  "target_ref": "verified-immutable-target-ref",
  "scope": "outer_provider",
  "captured_utc": "2026-10-30T20:00:00Z",
  "stable": {
    "image": {"state": "observed", "value": "reviewed-image-reference"},
    "nesting_configured": {"state": "observed", "value": true},
    "containarium_version": {"state": "not_verified"}
  },
  "runtime": {
    "instance_state": {"state": "observed", "value": "stopped"},
    "guest_mcp_read": {"state": "not_checked"}
  },
  "hashes": {"schema": 1, "stable": "computed-canonical-sha256", "runtime": "computed-canonical-sha256"}
}
```

This is a proposed shape, not actual output. Allowed typed fields include: source release/protocol, verified installed BoxWeave version/commit (if guest reports), current observed running version only if independently attested, Containarium and Incus version if authorized, image id, CPU/memory/disk class, observed `security.nesting`, presence/coverage of custom mounts, Web/Worker/MCP health in evidence categories, config digest computed **at trusted source without publishing secret bytes**. A public SHA of low-entropy settings can reveal information; consider omission or keyed private-only digest rather than hashing raw secrets.

Never store full instance config, SSH host/public/private keys, JWTs, environment variables, command strings, cookies, project workspace or arbitrary guest stdout. Choose each key from a hard-coded allowlist, validate type/maximum size and source provenance; unknown fields are discarded. Running PID, dynamic IP/clock and other volatile values cannot influence `stable` hash.

Canonical hashing: versioned canonical JSON with sorted fixed field keys, UTF-8 normalization rule, schema/domain prefix and explicit `observed/absent/not_verified` states. Compare within compatible schema/normalization version; if versions differ, return `not_comparable`, not fake drift. A digest proves equality of recorded bytes under the chosen normalization, **not authenticity of their source**.

## SQLite and API design

Proposed immutable `fingerprint_captures` table keyed by capture ID with actor, immutable source Target identity, scope/schema/normalizer versions, capture UTC, typed canonical stable/runtime data, coverage summary, hashes, expiration and optional checkpoint relation. Use the **next available DB user_version** at implementation; currently v8, diagnostics D1/D2 propose v9/v10, so never reserve a conflicting fixed version. Follow `Store` private path and migration/backup rules.

- `capture` is read-only relative to Provider/guest. Persisting a local record is not an external workload mutation.
- `get/list` enforce actor and exact managed target; a capture of a deleted/changed target remains historical evidence but not authority for new mutation.
- `diff(a,b)` enforces actor authorization for **both** capture IDs; support cross-instance comparison as explicit different-identity mode with a warning.
- A source may fail halfway through; preserve observed values and `coverage=partial`, but do not label it complete. A timeout/no access is `not_verified`, not null/false.

Example diff:

```json
{
  "comparison": "compatible",
  "target_relation": "same_exact_instance",
  "changes": [
    {"field":"stable.boxweave_installed_version", "state":"changed", "before":"0.21.4", "after":"0.22.0"},
    {"field":"runtime.worker", "state":"unverifiable", "reason":"after_not_verified"}
  ],
  "root_cause": "not_verified"
}
```

Return states `same/changed/added/removed/unverifiable/not_comparable`; detect coverage downgrade and exclude lifecycle-driven identity differences from an alleged code regression. Optional D0 health conclusions must retain original scope, not be upgraded.

## Suggested files and work units

1. `bwmini/fingerprints.py` typed capture/normalize/diff functions and pure fixture tests.
2. `bwmini/store.py` additive migration and scoped queries. Optional new module must be included in `distribution.py` immutable release inventory.
3. `bwmini/provider.py` minimal approved typed read methods; do not invoke arbitrary shell/Incus. Optional guest check uses existing auth only.
4. `bwmini/__main__.py` and `bwmini/web.py` narrowly typed read-only actions behind existing app authority, no public unauthenticated raw fingerprint endpoint.
5. One operator-facing CLI interface through reviewed entry, or Web read-only comparison UI; phase the UI after correctness and access tests.

No browser-only representation can be treated as enforcement. No need for D1 event schema or standalone Debug MCP.

## Tests and completion

| ID | Case | Expected |
| --- | --- | --- |
| E1-01 | Same exact config captured twice with different PID/IP/clock | stable hashes equal; runtime differences separate |
| E1-02 | Upgrade changes installed version but runtime old | displays independent installed/running evidence or `not_verified` |
| E1-03 | Outer alive, inner guest MCP unavailable | partial capture persists; inner state not_verified |
| E1-04 | Config changes with same name but different Target creation | not silently same instance |
| E1-05 | Missing/false/absent values | distinct, safe comparison statuses |
| E1-06 | Unauthorized second capture actor or foreign project | no cross-actor compare |
| E1-07 | Synthetic JWT/private key/SSH config in MCP result | no value in DB, hashes derived only from safe fields |
| E1-08 | Corrupt JSON/schema mismatch/unknown version | `not_comparable`/safe error |
| E1-09 | Concurrent write and DB-full on capture | no Provider mutation or partial promoted to complete |
| E1-10 | Snapshot not yet implemented | capture/diff still works standalone |

Completion gate: a real authorized read-only capture/diff of a managed outer Box yields useful stable and runtime differences without needing a healthy inner BoxWeave. Actual commands, exits, version and coverage must be recorded; fixture source tests are not live evidence. Next: [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md).
