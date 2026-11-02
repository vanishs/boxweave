# E0/E1 Box environment observation and conservative capability discovery

**Scope:** read-only, exact-managed-target observations via the existing app Provider MCP. The owner requested Phase 3 source development and actual Box validation. This is an initial non-mutating slice; it does NOT implement snapshots, restore or clone.

`bwmini/environment_observe.py` uses the already configured Provider launcher, MCP initialize/tools/list, and exact `get_container` target identity. The authenticated Web operator can read `GET /api/boxes/<logical>/environment`; existing session, target allowlist and current-instance checks are reused. Fingerprints contain only reviewed CPU/memory/disk, network Tenant, instance state and opaque identity/scope digests. Guest stdout, IP, keys, passwords, raw environment and instance config are not copied. Missing fields remain `partial`/`not_verified` instead of false negative drift.

The E0 capability map treats absent snapshot/clone/rekey methods as `unsupported` at the exposed MCP layer and mere matching names without verified safe schemas as `not_verified`. E2/E3/E4 NEVER infer permission to mutate from a read-only report; no direct Incus fallback. E1 `diff_fingerprints` only compares the same immutable instance/scope and reports unverified fields separately. An additive application SQLite schema v13 stores bounded immutable actor-scoped captures (max 128 per operator) and preserves read-only fingerprint metadata; no guest or host state is persisted. Authenticated Web endpoints support capture, list, get and compare using the existing session, Origin and CSRF authority. Capture and compare always validate the original exact identity; unmatched Box incarnations cannot silently be compared. Database migration and old/new compatibility need independent fixture and live acceptance.

Tests: `tests/test_environment_observe.py`, including missing catalog, spoofed tool name, identity recreation, malformed resources, deterministic hashing, no IP leakage, no mutation and schema-fail closed. Run exact source unit+browser CI, then an authorized real outer Box read and versioned installed Web check before marking LIVE. Until then S/T only.

## E1 capture implementation checkpoint

`bwmini/environment_history.py` defines private user_version=13 `environment_captures`, bounded immutable capture records and actor-scoped API reads. `bwmini/store.py` migrates from schema v12 and rejects unsupported future versions. The Web API performs fresh reviewed Provider reads, not client-provided reports; POST capture needs the existing CSRF token. A report never contains arbitrary host metadata or original MCP text. Tests cover v12 migration, history immutability, cross-target mismatches, CSRF, oversized/unsafe schema, and no raw token persistence.

This is only the E0/E1 read-only environment branch. E2/E3/E4 remain blocked on **verified upstream snapshot/restore/clone/rekey and storage coverage contracts**. Do not invent direct Incus fallback or promote a source fixture to installed-Provider proof. Actual older installed host upgrades and rollback remain a separate L/U gate.


## Phase 3 real-source acceptance refinement (v0.42.0)

[Detailed current acceptance and blockers](PHASE3_SUPPORTED_ENV_ACCEPTANCE.md).
Malformed catalog/target and nonfinite timestamps fail typed; corrupted
fingerprint and stored immutable capture data fail closed. Source evidence
includes two authenticated outer Provider read inputs and an isolated
v0.28.0 schema 12 to schema 13 migration. E2/E3/E4 are still unsupported
without explicit Provider write contracts. Installed host and old-version
rollback acceptance remain separate.
