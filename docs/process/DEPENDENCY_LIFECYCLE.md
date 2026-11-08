# Incus-aware dependency installation and upgrades — required

**Owner decision:** BoxWeave must support installation **on a host where Incus already runs** and explicit version-aware management of compatible pre-existing Incus, not merely install a private replacement daemon. This remains a host-local maintenance workflow, never a remote MCP package manager.

## Required and observed manifest

Pin source VERSION, exact Go toolchain/MCP SDK/Incus SDK revisions and verified hashes. Record the actual host OS/kernel, Incus daemon version, required API extensions, installation source, native features, running daemon users/projects (where safely observable), SQLite schema, installed BoxWeave binary digest and current expected upgrade target. Store safe before/after observations and a Tracking ID in a local installer journal.

**Caddy and its TLS/HTTP packages are not BoxWeave dependencies.** No independent proxy service is installed or upgraded. Exact compatible version bounds are established by engineering tests, not guessed in the design.

## Host-local controlled reconciliation

1. A local administrator explicitly invokes `boxweave install` or `boxweave upgrade`. Observe the existing daemon and its consumers before change; do not take over unrelated Incus instances just because they exist.
2. If Incus is absent, install a reviewed compatible version. If present and compatible, reuse it and provision only BoxWeave-controlled restricted Project/resources. If incompatible/outdated, plan a compatible upgrade and record version before and proposed after.
3. For a daemon with foreign projects/consumers, upgrading/restarting it may disrupt unrelated workloads. Require **separate exact target/version-specific local authorization** and preflight/rollback checks. Never assume a generic install command grants permission to disrupt them.
4. Verify signatures/checksums and repository provenance; do not use floating/untrusted installer URLs. Upgrade only approved managed prerequisites, preserve original storage/ownership, avoid unrelated package upgrades and avoid unreviewed downgrade/migration.
5. Apply and journal each phase; a crash/timeout with uncertain native effects stays UNKNOWN until independently verified. Do not blindly rerun destructive host operations. Never auto-reboot entire host; return `reboot_required=true` with specific component and truthful PARTIAL/BLOCKED until the operator reboots and postflight proves actual state.
6. Run `boxweave status/doctor` to report observed Incus version/extensions, own project and binary/schema compatibility. No claim of success from a package manager exit alone.
7. Nested BoxWeave treats its **own containing Box** as host. It can manage only software/resources inside that Box. It never changes the outer Incus, outer kernel, parent package manager, parent credentials or parent services.

## Minimal test matrix

Clean Ubuntu LTS host; already-installed compatible Incus with unrelated projects; stale daemon version requiring separately authorized shared upgrade; unchanged foreign resources; native daemon capability mismatch; failed package/checksum; interrupted upgrade and recovery; nested independent installation; retained versions and no automatic host reboot.

Other Linux **guest images** may be incrementally certified with real SDK/native tests after the Ubuntu LTS baseline. No automatic import of pre-existing foreign instances is included—see [product boundary](../decisions/PRODUCT_BOUNDARY.md) and the outstanding product clarification.

Related [Incus integration](../design/INCUS_INTEGRATION.md), [requirements](../product/REQUIREMENTS.md), [acceptance](../verification/ACCEPTANCE.md).
