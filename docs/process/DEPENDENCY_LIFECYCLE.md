# Incus-aware dependency installation and upgrades — required

**Owner decision:** BoxWeave must support installation **on a host where Incus already runs** and explicit version-aware management of compatible pre-existing Incus, not merely install a private replacement daemon. This remains a host-local maintenance workflow, never a remote MCP package manager.

## Required and observed manifest

Pin source VERSION, exact Go toolchain/MCP SDK/Incus SDK revisions and verified hashes. Record the actual host OS/kernel, Incus daemon version, required API extensions, installation source, native features, running daemon users/projects (where safely observable), SQLite schema, installed BoxWeave binary digest and current expected upgrade target. Store safe before/after observations and a Tracking ID in a local installer journal.

**Caddy and its TLS/HTTP packages are not BoxWeave dependencies.** No independent proxy service is installed or upgraded. Exact compatible version bounds are established by engineering tests, not guessed in the design.

## Issue #116 tested v1 implementation (2026-11-09)

Approved local CLI now supports exact package/version/host-bound `native-migrate-plan`, `native-migrate-apply`, `native-migrate-status`, `native-migrate-reconcile`, `native-migrate-rollback` and `native-migrate-commit`. It **refuses every existing native instance, even stopped, and extra foreign Projects**; this does not provide a generic shared-daemon upgrade license. Dedicated disposable nested Ubuntu24.04 lab tested signed `6.0.0-1`→`6.0.0-1ubuntu0.3`, private offline native snapshot and old signed package backups, actual rollback, after-APT process crash with no blind replay, APT pre-effect simulated failure and explicit real rollback, and compatible but uninitialized package repair with exact reviewed digest. **Mid-dpkg half-configured corruption has not been auto-repaired; it remains UNKNOWN/manual on failure.** Evidence / hashes: [Issue #116 native report](../verification/S1_ISSUE116_NATIVE_MIGRATION_2026-11-09.md). GitHub merge ruleset remains an independently required owner action [#117](https://github.com/vanishs/boxweave/issues/117).

## Prefer upstream-native runtime confinement

Reviewed Incus v7.6.0 module declares **Go 1.26.9**, MCP v1.8.0 declares **Go 1.25.0**; build using the stricter selected dependency requirement. Installed binary users should not need a local compiler.

Incus natively confines a non-root service user in `incus` (not `incus-admin`) to its per-user Project. The local admin installer should first configure a `restricted=true` Project with only reviewed exceptions. `restricted.devices.proxy` and `restricted.containers.nesting` default to `block`, so negative tests and explicit operator authorization are essential. Try restricted TLS only if the native local arrangement cannot support required operations; a privileged helper is a last resort, not a normal runtime component. See [Incus authorization](https://linuxcontainers.org/incus/docs/main/authorization/) and [20-tool mapping](../research/SDK_TOOLS_NATIVE_MAPPING_2026-11-07.md).

## Pre-first-Box Project initialization gate

On both fresh and already-running Incus hosts, provision/verify the exact native service-user **confined Project before the first Box**. A new isolated Project's `default` Profile can lack its root disk; `features.*` options can become immutable after an instance is created. Preflight those decisions together with restricted proxy/nesting settings, selected pool, image access, managed network/NIC/ACL and actual kernel capabilities. Do not solve a missing root disk by mutating the shared host `default` Profile or widening credentials. Existing foreign Project data remains untouched.

Stage the installation as **observe native state → choose and configure owned Project/Profiles/networks → prove rights and effective devices → create only owned instances**. Real cross-version checks must distinguish the observed daemon (last Box host: 7.5.1) from pinned SDK source v7.6.0. See [second-pass findings](../research/SDK_GAP_CLOSURE_2026-11-07.md).

## Host-local controlled reconciliation

1. A local administrator explicitly invokes `boxweave install` or `boxweave upgrade`. Observe the existing daemon and its consumers before change; do not take over unrelated Incus instances just because they exist.
2. If Incus is absent, install a reviewed compatible version. If present and compatible, reuse it and provision only BoxWeave-controlled restricted Project/resources. If incompatible/outdated, plan a compatible upgrade and record version before and proposed after.
3. For a daemon with foreign projects/consumers, upgrading/restarting it may disrupt unrelated workloads. Require **separate exact target/version-specific local authorization** and preflight/rollback checks. Never assume a generic install command grants permission to disrupt them.
4. Verify signatures/checksums and repository provenance; do not use floating/untrusted installer URLs. Upgrade only approved managed prerequisites, preserve original storage/ownership, avoid unrelated package upgrades and avoid unreviewed downgrade/migration.
5. Apply and journal each phase; a crash/timeout with uncertain native effects stays UNKNOWN until independently verified. Do not blindly rerun destructive host operations. Never auto-reboot entire host; return `reboot_required=true` with specific component and truthful PARTIAL/BLOCKED until the operator reboots and postflight proves actual state.
6. Run `boxweave status/doctor` to report observed Incus version/extensions, own project and binary/schema compatibility. No claim of success from a package manager exit alone.
7. Nested BoxWeave treats its **own containing Box** as host. It can manage only software/resources inside that Box. It never changes the outer Incus, outer kernel, parent package manager, parent credentials or parent services.

## Incident-derived host checks

- **Usable daemon, not only reachable daemon:** Preflight the actual selected Incus storage pool, root disk/profile mapping, network and required API extensions. A fresh daemon that answers `incus info` may still lack a storage pool; a pool on another device is not the host root filesystem's available space. Never initialize/replace foreign storage/network implicitly to make a test green.
- **Persistence actually survives:** Check SQLite intent/operation/ownership/journal location, filesystem/mount durability and service permissions under the target installation, host restart and independently nested restart. A directory merely existing is not proof that the intended durable volume is mounted.
- **Rollback and reporting:** Keep exact original and candidate binary identity, observed schema/dependency version and pre/post service health. If recovery of a previous binary or service restart fails, report **rollback failed** rather than success; an interrupted install remains BLOCKED/UNKNOWN where the actual state is not established.

These are checks inside the existing local install/status/doctor/upgrade flow, not a new host backup/recovery product. See [upstream incident study](../research/CONTAINARIUM_MAIN_HISTORY_2026-11-07.md).

## Minimal test matrix

Clean Ubuntu LTS host; already-installed compatible Incus with unrelated projects; stale daemon version requiring separately authorized shared upgrade; unchanged foreign resources; native daemon capability mismatch; failed package/checksum; interrupted upgrade and recovery; nested independent installation; retained versions and no automatic host reboot.

Other Linux **guest images** may be incrementally certified with real SDK/native tests after the Ubuntu LTS baseline. The owner explicitly chose **A**: reuse/manage the existing Incus daemon only, **never import already-running containers** or assume ownership of unrelated instances/projects. See [product boundary](../decisions/PRODUCT_BOUNDARY.md).

Related [Incus integration](../design/INCUS_INTEGRATION.md), [requirements](../product/REQUIREMENTS.md), [acceptance](../verification/ACCEPTANCE.md).
