# ADR-008: Independent nesting and parent-owned transitions

Accepted SRC-103; extends [ADR-007](0007-host-install-boundary.md). TASK-017 implements the source slice.

Pinned Provider couples nesting to Podman and lacks an independent MCP input. Dedicated LXC tests show different nested-start results before/after outer restart with unchanged software. Configuration does not certify effective execution.

The owned relay advertises an optional independent create field and exact inspect/plan/apply tool. Omission preserves upstream behavior; explicit unsupported selection is rejected. Forward creation once, then configure only a successfully created, freshly observed incarnation. A post-create failure is unknown and must not replay creation.

A root-owned standalone helper and exact fixed-argument sudo rule accept bounded JSON, validate default-project managed Ubuntu identity and creation/UUID/config fingerprint, require confirmation, and change one boolean. Credential-authorized get_container precedes access. This is the existing single trusted operator boundary, not multi-user RBAC. The additional privileged surface needs explicit installed-copy acceptance. No automatic parent access, profile edits, privileged containers or restart.

Installer warns inside LXC without guessing parent identity. Doctor reports unknown parent/effective capability; proc denial diagnosis gives parent-host guidance. Restart and authenticated child verification remain separate actions. Revisit when upstream supplies an independent contract, authority changes, or new evidence changes guidance. [Verification](../../verification/NESTING_VERIFICATION.md).
