# Managed dependency versions, installation and upgrades — P0

**Owner decision:** Record the observed and required versions of BoxWeave's dependencies. A locally authorized BoxWeave install or upgrade **automatically installs missing dependencies or upgrades incompatible/outdated approved dependencies**, rather than leaving the operator to guess or manually resolve ordinary compatible cases.

This does **not** authorize remote MCP commands, unrequested background updates, broad package upgrades, **automatic whole-host reboots**, arbitrary host configuration takeover, or mutation of unrelated services. Each layer installs/upgrades only its own local environment; it never touches a parent BoxWeave host.

## Versioned dependency manifest

The installer and built artifacts must carry a versioned, source-bound manifest with:

- Product `VERSION`, Git commit, Go build/toolchain version, module graph and checksums (`go.sum` or equivalent), official MCP Go SDK and Incus Go SDK revisions.
- Managed runtime prerequisites: exact Incus daemon version observed; minimum/compatible target versions or ranges and required Incus `api_extensions`; Caddy version and required capabilities; SQLite implementation/schema version and database compatibility; Ubuntu LTS release/architecture; Linux kernel and nesting prerequisites.
- For each dependency: package/source name, distributor/channel, pinned artifact or repository trust identity, allowed upgrade range, observed version, proposed target, whether the service is **BoxWeave-owned**, **shared/existing**, or **unmanaged**, changes needed, trust/checksum verification and rollback capability/limits.
- A private installation ledger records source/binary digest, dependency observations *before and after*, plan, applied steps, outcomes, migration/checkpoint IDs, tracking ID, failure or UNKNOWN stage and operator authorization. Keep credentials and unrelated host configuration out of the ledger.
- `boxweave version/status/doctor` reports BoxWeave, official SDK build info, Incus/Caddy observed versions, capabilities and drift with `supported`, `outdated`, `upgrade_planned`, `unmanaged_conflict`, `unsupported` or `not_verified`; do not fabricate a future installed version from a desired manifest.

Specific dependency version numbers are **not guessed in this design**. Pin a compatible, tested baseline during implementation after native capability and nested tests, and update the source-bound manifest for each relevant release.

## Explicit invocation, automatic dependency reconciliation

1. `boxweave install`, `boxweave upgrade` (or reviewed installer entry) are **explicit host-local operator actions**. There is no timed auto-updater and no MCP-exposed package manager. The authorized invocation includes a bounded dependency reconciliation phase.
2. **Preflight:** observe the exact running BoxWeave/Incus/Caddy and ownership, inventory protected resources and foreign consumers where evidence is available, check the installed OS/kernel, requested image/storage, available package trust, compatibility, free space, boot and service availability.
3. **Plan:** compare observed versions and required feature/API-extension contract; choose the reviewed, pinned target and a deterministic change set. If already compatible, no unnecessary upgrade; if missing, install; if outdated/incompatible, upgrade when approved ownership and service impact conditions permit. Never downgrade a newer dependency just because a string is not equal.
4. **Safety boundary:** a dedicated BoxWeave-owned Caddy and other **owned** prerequisites can be installed/upgraded automatically by this local workflow. Reusing a **shared pre-existing Incus daemon** is permitted; upgrading that daemon can affect foreign projects and applications. If it is shared or foreign consumers cannot be ruled out, do **not** silently restart or upgrade it. Stop at a typed `SHARED_DEPENDENCY_APPROVAL_REQUIRED` gate with the exact version/impact plan; an explicit local approval for that specific dependency action may resume it. The general installation request is not blanket authorization for unowned host changes.
5. Verify package signatures/hashes and repo provenance; validate configuration, hold bounded service drain/backup, install the selected version, apply only reviewed migrations, restart only services explicitly included in the approved plan, **never reboot the whole host automatically**, and confirm native version + capabilities + essential owned functions. No opaque `curl | bash` dependency installation or unpinned floating installer code.
6. **Failure handling:** preserve original service, state and installer journal where safely possible. A half-completed migration/restart, lost transport or ambiguous native effect is `UNKNOWN` until independently checked; never blindly rerun a destructive step. Do not downgrade a database schema under an old binary without proven support. Refuse migration if it would lose ownership records or modify foreign resources.
7. **Postflight:** record actual observed versions, release hashes, health/doctor results, retained state and unresolved compatibility. If a kernel/host reboot is required, record `reboot_required=true`, precise component/reason and recovery journal; return `PARTIAL`/`BLOCKED` until an **operator-performed reboot** and independent post-boot verification. Do not schedule an automatic reboot. `PASS` requires real installed behavior, not only a successful package manager exit code.
8. Repeated or interrupted local installer invocations reconcile exact recorded phases rather than starting a second conflicting update. Local installer status/repair exposes the same safe `tracking_id` concept as MCP operations, while keeping privileged recovery strictly host-local.

## Recursive/nested implications

The environment running a BoxWeave installation **is its host**. When a Box inside L0 runs BoxWeave L1, L1's installer may install/upgrade only Incus, Caddy and software **inside that Box**, subject to granted nesting/kernel capabilities. It must never invoke L0's package manager, parent Incus socket, Caddy Admin API or privileged helper. One layer's installer must not update the other automatically.

## Cases for implementation testing

- Fresh Ubuntu LTS installs required verified Incus/Caddy/BoxWeave versions; record observed versions before/after and source artifact hashes.
- Missing/outdated **owned** dependency upgrades automatically as part of explicit local install/upgrade; repeat invocation makes no unnecessary change.
- Pre-existing shared Incus daemon, unrelated Incus project or Caddy port conflict **is not modified without its own explicit local approval**; verify unchanged native state after refusal.
- A shared Incus upgrade with a specifically approved target/version is performed only after independent compatibility and rollback impact checks, and is verified post-restart against owned and foreign-observation safety conditions.
- Detect unsupported Incus API extensions and unexpected/newer incompatible versions without unsafe forced downgrade.
- Simulate crashed package manager/partial schema migration, insufficient disk, compromised artifact digest, required kernel reboot (no automatic reboot), service restart failure and nested unsupported prerequisites; preserve original recovery identity and report UNKNOWN/blocked accurately.
- Source-CI automatic GitHub Release **does not** invoke host installation, dependency reconciliation or auto-promote a stable selector.

Related: [approved defaults](../decisions/DEFAULTS.md), [installer requirements](../product/REQUIREMENTS.md), [technology stack](../design/TECH_STACK.md), [security](../design/SECURITY.md), [acceptance](../verification/ACCEPTANCE.md).
