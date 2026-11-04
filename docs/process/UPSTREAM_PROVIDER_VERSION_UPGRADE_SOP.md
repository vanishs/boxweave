# Standard Containarium upstream version upgrade SOP for BoxWeave Agents

**Status:** active repeatable process, first applied to the **v0.120.1 → v0.128.0** compatibility candidate. **Default policy:** minimum-only Containarium component versions with **no maximum**; exact upstream release hashes authenticate downloads, not a static compatibility allowlist. This SOP is for **any future Containarium release**. Source, CI, isolated test, installed application, running daemon and real user acceptance are separate evidence classes.

## 0. Authorization, scope and immutable evidence

Read [AGENTS](../../AGENTS.md), [maintenance](MAINTENANCE.md), [current gates](PHASE_STATUS.md), [risks](../verification/OPEN_RISKS.md), [upstream ownership](../UPSTREAM_PROVIDER.md), the [upstream official upgrade runbook](https://github.com/FootprintAI/Containarium/blob/v0.128.0/docs/UPGRADE-AND-ROLLBACK.md), and the [migration/REST design](../design/API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md).

**Two different transactions:**
1. **Containarium Provider upgrade**, executed by the operator under explicit approval and the upstream's own upgrade/rollback process. It can affect the daemon, Incus, network, storage, SSH, JWT, databases and fleet peers. **Never performed implicitly by installing BoxWeave**.
2. **BoxWeave app update**, requested explicitly with `install.sh --version 0.X.Y`. Reuses the existing Provider; updates application-owned helpers and reviewed private snapshots only. Never changes a host's Provider/Incus automatically.

An Agent may perform read-only repository analysis and source CI inside its normal scope. Actual Provider updates, host restart, SSH trust reset, backup identity access, destructive Box tests, new routes and real storage rollback require the exact approved host and operation. If not approved, mark L-gates pending and continue source work. Never use a host's `sudo` or a successful MCP tool call as evidence of operator permission.

## 1. Resolve the target and meaningful differences

1. Record old and proposed Containarium Git tags, immutable tag->commit source SHAs, release timestamps and binary/SHA256SUMS digest(s), plus the local BoxWeave source SHA, stable selector and running app/worker versions.
2. Compare *every intervening published version* and inspect release notes, Git commit count, changed files by capped-safe segment and generated Swagger/proto diff. Classify changed routes, deleted routes, changed request/response shapes, error codes, auth scope, transport, SSH, backup, networking, storage, schema, auto-update and installation prerequisites.
3. Inventory BoxWeave code callers before recommending changes: `bwmini/online.py`, `bootstrap.py`, `provider_rest.py`, `provider.py`, `mcp_gate.py`, `mcp_compat.py`, `deploy_check.py`, `agent_box.py`, `auto_sleep.py`, coordinator, diagnostics and tests. Compare against actual feature matrix and any previous upgrade handoff; do not treat design-only features as already shipped.
4. Record breaking changes separately from new optional functionality. Do not infer that an unchanged REST route has identical effects, or that an MCP tool advertised in `tools/list` was authorized/executed.
5. Apply the **minimum-only** rule to components BoxWeave actually depends on. For the current policy daemon and authenticated MCP server must be >= `0.128.0`; no highest version, exact version pairing or growing allowlist. Unknown/unparseable is **unverified**, not automatically compatible; missing required runtime capability is **unsupported**, even if version passes.

**Version policy changes** (e.g., raising minimum due to a new required upstream feature) must include a concrete capability requirement and regression test. Ordinary upstream releases **do not** require a BoxWeave release solely to append the newer number.

## 2. Baseline and rollback evidence before modifying a host

On an authorized host, record independently, without copying secrets:
- `containariumd version`, `containarium version`, `incus --version`, app `boxweave version`, `sudo boxweave doctor --mcp`, `systemctl status` for the actual managed Provider unit, plus process/binary paths, file mode, owners and digest records.
- Provider topology: standalone versus externally managed, actual listener and bind addresses (loopback vs all interfaces), JWT scope ownership, daemon/backend versions, sentinel automatic-update configuration, storage driver, native tenant network, SSH host key, `containarium-shell` path, toolchain path, known Boxes and their original immutable identities. **Never output private token, age identity or contents of signing secrets**.
- Verified database backup and at least one separately tested restore route appropriate to the actual storage. A ZFS snapshot is not a PostgreSQL backup and a `dir` pool is not ZFS. Keep encrypted age keys host-owned; do not send them in MCP arguments.
- A tested non-destructive rollback plan for the exact binaries and service; mark data/schema downgrade **not verified** unless demonstrated in a separate disposable environment. Upstream's upgrade watchdog tests `/health` only; it cannot certify the full application.
- Current operations and uncertain writes. If a prior create/restore/delete outcome is UNKNOWN, reconcile that exact operation before any repeat or upgrade. Document any external agent session that might still mutate the same Box.

**Stop here** on unreviewed binary provenance, unknown storage state, unbacked DB migration, non-authorized target or active unresolved write. Do not invent a successful rollback checkpoint.

## 3. Source-only P0 compatibility change and fixture tests

1. On a dedicated branch, adjust code **only if the version's actual contracts require it**. Keep minimum version metadata and capability detection, not an ever-growing SHA/version compatibility table. An integrity SHA is still mandatory for binary downloads.
2. Keep managed/external Providers untouched. Fresh hosts may bootstrap the separately reviewed minimum-version assets. Existing trusted root-owned installed components may change independently; app-owned `mcp` and `provider` copies are refreshed only after source digest review. All other managed files keep exact checks.
3. Maintain REST/MCP alongside one operation/claim/receipt model. Validate authenticated GET/list and bounded write behavior; any dispatched timeout remains UNKNOWN and is not retried automatically. Retain exact Box identity (backend, user, createdAt/incarnation), worker diagnostics, TCP/UDP Guard boundary and root-owned restore policy.
4. Run focused unit tests: version minimum inclusive/future major/old rejection, untrusted/malformed version rejection, host-shell digest/foreign-file refusal, app-owned snapshot refresh, REST HTTP status classification, MCP version/schema, preserved read and denial, no accidental Provider upgrade, operator config preservation.
5. Then full `python scripts/check_knowledge.py`, `python -m unittest discover -s tests -v`, `bash -n install.sh`, Web typecheck/build and available fixture browser suite; preserve executable outputs. Run CI on exact PR/head commit; document warnings and fixture-only coverage. No CI green implies a real Provider test.
6. Bump `VERSION`, Python package, Web package and lock root consistently. **Do not change `PUBLISHED_RELEASE` or the embedded default streamed installer pin** for a candidate.

## 4. Isolated installed compatibility (before claiming full runtime acceptance)

Use a **new explicitly approved disposable Box** with nested capability rather than an old unrelated Box. Seed the specified old versions in the isolated host first; check original Incus, Provider, BoxWeave, tenant and Box identities. Ensure a recovery artifact is created before the Provider upgrade.

On this isolated host only, perform the **upstream-documented upgrade** (avoid floating root install scripts; verify exact binaries or official signed release), one backend at a time. Before applying, check any required toolchain or storage preflight. For `0.123.0+`, BUILD_TOOLS requires `/opt/containarium/toolchains/current/bin`; missing tools cause a typed refusal and must not be hidden by silently installing new host packages.

After upgrade, independently check:
- Running daemon exact reported version is at least the minimum; app/daemon/CLI version mismatch is reported, not guessed away.
- Authenticated REST `GET /v1/containers` and a specific known Box; MCP initialize `serverInfo.version` >=minimum, tools discovery and actual `list_containers` read; compare schemas and the identities returned by each transport.
- One approved disposable **create→inspect→SSH identity→resize→stop→start→delete** with per-step read-back. Include denied target/scopes, missing target, upstream 404 route, timeout/UNKNOWN reconciliation and no duplicate create.
- Managed SSH host-hop separately from direct MCP SSH; validate client host pins, known wrappers, negative key authorization. No blanket Incus privilege changes.
- If relevant to the version: toolchain BUILD_TOOLS run-user status, auto-sleep cold create/idle stop/HTTP wake, route/GUARD packet path, encrypted whole-Box restore with authorized host-local key, snapshot ZFS storage eligibility. Unavailable topology is `BLOCKED` rather than fake PASS.
- Restart service only within approved test scope and verify DB/client credentials, original immutable instance identity, live app/worker, receipts and trace privacy.
- Retain any failed/UNKNOWN test Box for diagnosis; delete **only** verified test-owned resources, on explicit permission.

## 5. Operator's separately authorized production Provider upgrade

A source/CI-only candidate and this SOP do **not** upgrade the operator host. When the operator approves, first perform Step 2 on the real host, then upgrade Containarium using the **official version-pinned** mechanism for that host's real topology. Inspect sentinel: a served binary may auto-update every backend, so avoid unexpected fleet-wide rollout. Do not run BoxWeave `install.sh` as a substitute for upgrading Containarium.

Verify actual running Provider version, authenticated REST/MCP, private credentials, storage, network, old Box identities and service status after the Provider upgrade, **before** updating BoxWeave. Failure: stop and use the prepared upstream recovery plan; do not re-run writes after a timeout or claim that `/health` is sufficient.

## 6. Exact candidate BoxWeave install and operator smoke tests

After the running Provider meets minimum and its credentials/SSH trust are intact, install the exact published BoxWeave candidate:

```sh
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.43.2
```

For later versions, substitute only the reviewed, immutable BoxWeave release number. **This does not change the upstream Provider version**. For externally managed Providers, use their previously reviewed `--provider-url`, `--provider-secret-file`, `--provider-cli` settings only if the original installation requires them; do not paste token contents into the command.

**Minimum operator smoke checklist**: `boxweave version` == exact selected release; `sudo boxweave doctor --mcp` exit 0 with authenticated REST+MCP reads; Web and Worker active; HTTP MCP unauthenticated read denied (401); no unexpected public listener; existing Box listing and exact instance identity unchanged; trusted Agent `connect(exec)` executes a bounded `whoami`/Python hello/read against one approved Box. No create/delete is necessary for this first smoke if the host is not disposable. Capture only public metadata, exit statuses and evidence.

## 7. Publish/rollout decision and evidence

Create a focused PR, wait for its exact-head CI, merge without force, verify main CI/build and immutable release tag/source/asset digest. **Publish an immutable candidate** but do **not** move `PUBLISHED_RELEASE` or default `install.sh` pin without a separate stable-promotion decision and installed evidence.

Record: upstream source refs/digests, BoxWeave commit/tag, test environment + target identity, actually executed commands and exits, PASS/FAIL/BLOCKED/UNKNOWN for each gate, host/version drift, original Box dispositions, remaining risks and the next smallest test. Update [feature matrix](../verification/FEATURE_MATRIX.md), [remediation queue](../verification/REMEDIATION_TEST_QUEUE.md), [risk register](../verification/OPEN_RISKS.md) and [current status](PHASE_STATUS.md). Do not turn an unverified host test into a source-pass claim.

## 8. Reuse rule for future Agents

For **Containarium vX → vY**, repeat steps 0–7 with the exact source/digest delta. If daemon/MCP still meet minimum and the verified schemas/runtime tests pass, **no BoxWeave code change or new release is automatically needed**. Raise the floor only when BoxWeave truly consumes a newer required capability. Preserve Agent-facing MCP and REST side-by-side until a separate approved design replaces a particular path.

The historical [v0.128.0 assessment](../design/CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md) is an example, **not** a future-version whitelist.
