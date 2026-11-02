# E0/E1 Box environment observation and conservative capability discovery

**Scope:** read-only, exact-managed-target observations via the existing app Provider MCP. The owner requested Phase 3 source development and actual Box validation. This is an initial non-mutating slice; it does NOT implement snapshots, restore or clone.

`bwmini/environment_observe.py` uses the already configured Provider launcher, MCP initialize/tools/list, and exact `get_container` target identity. The authenticated Web operator can read `GET /api/boxes/<logical>/environment`; existing session, target allowlist and current-instance checks are reused. Fingerprints contain only reviewed CPU/memory/disk, network Tenant, instance state and opaque identity/scope digests. Guest stdout, IP, keys, passwords, raw environment and instance config are not copied. Missing fields remain `partial`/`not_verified` instead of false negative drift.

The E0 capability map treats absent snapshot/clone/rekey methods as `unsupported` at the exposed MCP layer and mere matching names without verified safe schemas as `not_verified`. E2/E3/E4 NEVER infer permission to mutate from a read-only report; no direct Incus fallback. E1 `diff_fingerprints` only compares the same immutable instance/scope and reports unverified fields separately. Long-term immutable capture storage and Web comparison UI remain a separate unimplemented E1 work slice.

Tests: `tests/test_environment_observe.py`, including missing catalog, spoofed tool name, identity recreation, malformed resources, deterministic hashing, no IP leakage, no mutation and schema-fail closed. Run exact source unit+browser CI, then an authorized real outer Box read and versioned installed Web check before marking LIVE. Until then S/T only.
