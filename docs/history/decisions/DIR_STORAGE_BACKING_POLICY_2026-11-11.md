# Owner decision: Incus dir storage accepts arbitrary backing filesystems

**Date:** 2026-11-11. **Scope:** v1/S2.5, not legacy main.

The Incus **dir storage driver** is valid on any filesystem on which the
upstream Incus daemon can create and manage its directory volumes.
BoxWeave must not make ext4/XFS a general requirement for using dir.

**Separate the two properties:**
1. Directory storage availability: native SDK reports a created dir pool,
   the selected canonical directory exists without a symlink, and the host
   has acceptable currently available space. A named nondefault pool still
   needs independently distinguishable backing to avoid silently falling
   back to a missing/foreign mount.
2. Per-instance quota enforcement: filesystem capability, mount options,
   Incus daemon support and actual native volume behavior are **different**
   from the simple ability to store an instance. An ext4/XFS project-quota
   mount is only a historically tested positive *precondition*, not proof
   that every future volume is bounded. Other filesystems can be valid
   storage even when this precondition is absent.

When project-quota enforcement is not established, **doctor**, **host_status**
and the root-local pre-first-Box review must expose a warning, not fail
the entire dir driver solely for this reason. Use the stable warning code
**DIR_QUOTA_UNVERIFIED**. Preserve an explicit false
selected_storage_quota_verified; never silently display an enforced limit.
Incus 7.6 itself may only emit a daemon-local log warning when it skips a
quota and does not reliably expose this warning in the SDK operation
result; the service identity must not be granted broad daemon log access.

**Operational risk accepted, not erased:** If native quota is skipped, a Box
or its recorded Exec output can fill the entire backing filesystem,
affecting siblings or the parent daemon. Free-space preflight is not a
replacement for a native quota. Report this to operators; require native
EDQUOT / isolation evidence before advertising a guaranteed size limit,
and bound destructive quota/pressure tests to disposable labs. A dedicated
filesystem and capacity monitoring remain strongly recommended on the L0
host even though arbitrary dir backing is supported.

**Not changed:** Project and UUID isolation, private IPv4/IPv6/ACL,
native UNKNOWN no-replay, image trust, authentication, mount/source safety,
no implicit adoption, and no physical HOST_GO from configuration-only
results. Non-dir drivers still need separate storage semantics acceptance.
This decision supersedes earlier unconditional ext4/XFS failure as an
admission rule; earlier S2 ext4 native quota results remain valid evidence
for that particular accepted topology.
