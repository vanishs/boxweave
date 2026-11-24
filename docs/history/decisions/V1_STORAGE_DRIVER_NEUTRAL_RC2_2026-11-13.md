# RC.2 native Incus storage-driver neutrality (2026-11-13)

## Owner decision
BoxWeave does **not** administer or certify the host's Incus storage environment. The Incus operator selects, creates and maintains the pool and its backing storage. On the dedicated Ubuntu 26.04 + Incus 7.6 physical test host, the operator has created an empty `default` Btrfs pool on an existing Btrfs subvolume. RC.1 incorrectly rejects that legitimate pool with `SELECTED_STORAGE_QUOTA_DRIVER_UNVERIFIED`.

## RC.2 contract
- Runtime `doctor` / strict native admission, root-local `native-prereq-plan/verify` and approved `native-prepare-plan/apply` check only that the exact selected Incus pool is accessible and reports `Created`. They never select, repair, mount, reformat, migrate or upgrade a pool, and never reject a pool based on its driver, mount point or filesystem.
- The existing Project confinement, owned default root device, managed bridge + exclusive ACL, IPv4/IPv6/MAC filters, image authenticity, exact native state/host review digests and operator approvals remain mandatory.
- The existing JSON status fields for separate mount, space and quota are conservatively **unverified**, not claims of enforcement. `STORAGE_QUOTA_UNVERIFIED` is a warning, never a hard installation gate. Native create/resize failures remain genuine errors/operation receipts, never silently retried if possibly dispatched.
- Incus is the source of truth for its storage and quotas. Operational capacity, disk-exhaustion protections, snapshots and actual quota behavior remain for operator review and explicit physical-host tests. No quota or nested-volume guarantee is inferred from `Created`.
- RC.1's dir-only lab tests are historical mechanism evidence, **not** a demand that RC.2 use `dir`. RC.2 adds driver-neutral unit coverage and retains isolated native source and release tests. Dedicated physical Btrfs acceptance remains **pending**, not a pre-publication PASS.

## Publication / test boundary
An RC.2 prerelease may be published after reviewed source, protected-v1 merge and the exact-source release workflow passes. Publication is for testing on the separately approved machine; it is not S5 release readiness or HOST_GO. No automatic migration, native setup or host installation occurs on release.
