# Existing Containarium reuse: source change and acceptance plan

Scope: source 0.17.2, user-requested install/update/repair and minimum-platform compatibility. Reference:
`archive/0.5.1` `install.sh` used the installed Incus and Containarium
rather than replacing the Provider on ordinary reruns. Its implementation is
historical evidence, not code copied into the current release.

## Decision matrix

| Host on first BoxWeave installation | New streamed behavior |
| --- | --- |
| No Incus or Containarium | Retain pinned, source-bound managed provisioning |
| Incus already installed, with a nonempty storage pool, no Containarium | Preserve Incus/pool, install the pinned managed Provider, do not run Incus init |
| Incus + Containarium CLI + daemon | Reuse external Provider; no Provider binary download/replacement, new Provider unit, Incus init, PostgreSQL provisioning or Provider restart |
| Partial Provider binaries or an uninitialized standalone Incus | Refuse with explicit reason, rather than installing over foreign state |
| Existing managed BoxWeave install | Preserve recorded Provider mode across update/repair |
| Existing unrelated BoxWeave user, state or Provider unit | Refuse automatic adoption/migration |

External Provider defaults: local REST endpoint `http://127.0.0.1:8080`,
signing-secret **path** `/etc/containarium/jwt.secret`, and installed
`containariumd` binary. For a nonstandard setup, the streamed installer
accepts `--provider-url`, `--provider-secret-file`, and `--provider-cli`.
The external secret is inspected for root ownership and restrictive mode,
never read by the coordinator. The existing Provider's local REST API must be
reachable (200 or auth-required 401) and Incus must have at least one pool.
BoxWeave uses the release-pinned MCP client as an **application-owned snapshot**
while keeping the external daemon, packages, signing secret, pool and services
untouched. The dedicated BoxWeave bootstrap and root helper still modify
BoxWeave-owned accounts, files, timer and sudo policy; these are not a read-only
installer.

## Present limitations

- The **minimum** platform gate is Linux / CPython 3.12, x86_64; Ubuntu 24.04+ is recommended.
  The immutable release bundles SHA256-locked CPython 3.12–3.15 Linux x86_64
  wheels. Future Python versions are not guaranteed a compatible wheel until
  the distribution is rebuilt and verified. Ubuntu 26.04 real-host testing
  remains separate from CI; do not confuse a passing version gate with proven
  Provider or reboot integration.
- A Provider installed with a custom secret or API endpoint needs explicit
  operator flags. A non-REST Provider is refused; no service options are modified.
- Bootstrap pins a snapshot of the installed daemon binary. If an administrator
  separately upgrades it, rerun is refused until an explicit independently
  reviewed Provider-identity migration is implemented. Never mutate the external
  daemon or rewrite its signing authority as an implicit repair.
- No legacy BoxWeave 0.5.1 state/config migration is attempted.
- Incus `dir` on bcachefs is a shared-filesystem driver to Containarium;
  on a preconfigured pool, verify the expected `source` and the actual mount
  independently, and configure systemd mount dependencies before real workloads.

## Validation record

**S (source)**: reviewed `install.sh`, `bwmini/online.py`,
`bwmini/bootstrap.py`, and relevant archive behavior. No real Provider or
systemd action was executed from this repository-edit session.

**U (requires authorized disposable hosts)**:

1. Existing Incus + bcachefs-backed `default` pool: record `incus storage
   show default` before and after streamed installation; ensure identical
   pool config, profile and workloads, and new BoxWeave workload uses the pool.
2. Existing active Containarium: snapshot hashes, unit definitions, processes,
   Provider database, JWT secret metadata, Incus state, and network routes;
   run streamed install and verify none were changed by provider provisioning.
3. Test mismatched port, missing or non-private secret, partial installation,
   stopped/non-REST Provider, empty pool, and foreign BoxWeave state. Verify
   refusal precedes BoxWeave namespace creation.
4. Re-run identical release, upgrade to later release and repair interrupted
   BoxWeave-owned state while preserving external Provider identities and
   credentials. Separately test upstream Provider drift refusal.
5. Verify real authenticated MCP initialize/read, Web login, child container,
   Python hello world, boot readiness, and failure after mount removal. Keep
   secret content out of logs, reports and command history.
6. Run `python -m unittest tests.test_online_adoption tests.test_online`,
   `bash -n install.sh`, `python scripts/check_knowledge.py` and
   `python scripts/validate.py --browser` with a prepared non-root test
   checkout, then inspect exact GitHub CI results. CI is not host acceptance.


## 0.17.2 platform floor verification

**S (historical 0.17.2)**: the host checks then required numeric Ubuntu >= 24.04 and CPython >= 3.12. From 0.17.11, distribution-name gates are removed, while missing-package installation remains APT-dependent; the app ZIP targets CPython >= 3.12. Fresh-host Incus provisioning uses the actual Ubuntu codename and preserves the 24.04 package pin, while other accepted releases use the signed distribution-specific stable repository. The managed-provider-only local PostgreSQL install selects the distribution's default PostgreSQL package on newer Ubuntu versions; external Containarium retains its own PostgreSQL, unchanged.

**T**: CI is required to prove version-floor unit checks, shell syntax, package candidate, and actual Python 3.14 offline wheel installation after release publication. **U**: Ubuntu 26.04 standalone install, existing Provider reuse, real Web/MCP create and restart recovery; execute with before/after Incus pool, mounted bcachefs and Provider checks before claiming success.
