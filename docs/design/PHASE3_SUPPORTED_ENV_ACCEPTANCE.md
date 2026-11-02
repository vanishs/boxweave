# Phase 3 — supported Provider environment capability acceptance (v0.42.0)

[Feature roadmap](FEATURE_ROADMAP_MASTER.md) ·
[E0/E1 implementation](BOX_ENV_E0_E1_IMPLEMENTATION.md) ·
[E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md) ·
[E3 restore](BOX_ENV_E3_RESTORE.md) ·
[E4 clone](BOX_ENV_E4_CLONE.md)

**Result:** E0/E1 supported *read-only* environment observation, immutable
fingerprint and app DB migration accepted within source/Box-input scope;
E2/E3/E4 **BLOCKED** because the authenticated exposed Provider tool catalog
has no verified snapshot/restore/clone/rekey and storage-volume contract.
No direct Incus bypass or invented snapshots are permitted.

## Phase 3 source deliverable

- Preserve exact Provider/Target binding and bounded fields from E0/E1.
- Harden malformed/oversized Provider tool catalogs, wrong target shapes and
  invalid observation timestamps to return typed refusal without guest writes.
- Diff only two validated observations with the same exact immutable Box
  incarnation and Provider scope. Corrupt or forged fingerprints are not
  silently called unchanged.
- Validate immutable SQLite capture payload integrity on read/compare, not
  only on first write. Failed integrity check preserves the original row
  and refuses the comparison; never attempts remote restore.
- Ensure `complete` vs `partial` actually matches the observed resource
  field coverage. Do not fabricate missing memory, disk or network Tenant
  values for older Providers.

## Executed evidence (2026-11-02)

In an authorized existing test Box `bwphase2-1102` (Ubuntu, Python 3.12.3),
the v0.42.0 source worktree uses the old tested v0.41.0 main as base.

**Authenticated outer Provider read inputs:** `get_container` was called
for two existing Boxes, never modified:

| Box | State | Source-computed fingerprint | E0 capability |
| --- | --- | --- | --- |
| `bwphase2-1102` | RUNNING | `d6240dfc7228b2d7f3e4f9e16f903ce934dc6420ea7a0daf5ee59d51d050c4ca` | snapshot/clone unsupported |
| `boxweave-test-ser` | RUNNING | `fcd45b13fb04e523a1bfbd2d098468e6d27006aa96283fb075f88b0ef4c76a2b` | snapshot/clone unsupported |

The real retrieved, authenticated `get_container` responses were fed to
source `observe_exact`. Both yielded complete bounded fingerprints; one
cross-Box comparison refused with `ENVIRONMENT_TARGET_CHANGED`. This is
**L for Provider read + S/T for source transformation**, NOT a live installed
Web API or an authenticated provider `snapshot_list` response.

**Old-version schema compatibility:** in the same Box, executed a real
`v0.28.0` source `Store` in its own Python process on an isolated new
fixture database. Old schema **v12** created successfully (exit 0).
A separate new-source `Store` opened that exact DB and migrated to
schema **v13** (exit 0), preserving the existing admin metadata. Reopening
v13 using the v0.28.0 process refused with `SCHEMA_UNSUPPORTED` (exit 1);
DB remained v13 unchanged. This is a deliberately safe downgrade refusal,
not a working rollback. A real host upgrade requires a backed-up migration,
same installed package identity, service startup and user-scoped Web tests.

**Fault/recovery:** source tests cover corrupt persisted capture metadata,
missing catalog, malformed Provider field or timestamp, old missing fields,
wrong incarnation and future schema rejection; each fails without replaying
a mutation or deleting saved evidence.

## Acceptance and pending gates

- Run focused `test_environment_observe.py` and
  `test_environment_history.py`; run the complete unittest suite and
  `scripts/check_knowledge.py` in the Box. Exact-SHA GitHub workflow
  must additionally prove packaging, browser fixture and frontend build.
- E2/E3/E4 cannot be activated unless E0 reports separately authorized
  typed Provider methods, identity/rekey, persistent volume/storage coverage
  and shared R-04 claims. No direct Incus fallback.
- Actual stopped-Box snapshot/create/restore/clone failure recovery and
  host upgrades cannot be PASSED when the supported upstream methods are
  unavailable. Record **BLOCKED** and do not perform unsafe imitations.
- Production root policy, installed v0.42.0 Web/SSH/HTTP, public OAuth client,
  real rollback and older root helper interoperability are **LIVE U**.
- The default stable installer still selects v0.28.0. The source Release is
  not an instruction to silently upgrade the user's server or promote that pin.

This explicitly limits the phase-complete claim to the **supported read-only
E0/E1 path**; it does not assert that every proposed E0-E4 feature shipped.
