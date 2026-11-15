# BH-019 — N×M v2 sequential evidence matrix, L0 RC5 / L1 RC4 (2026-11-14 PT)

> **Purpose:** re-run the highest-information prior experiments with better fault localization, not to report recursive AppArmor delegation success. **Scope:** AppArmor / policy namespace, UserNS/NSFS, capabilities/IDs, read-only policy surface, and safe permission negatives only. Per owner instruction, do not research distribution/version matrices, patches, BPF or VM routes. No privileged Box, physical-host AppArmor/profile changes, host sysctl writes, exposed host Incus admin socket, or historical UNKNOWN replay.

## V2 matrix — sequential N with multiple M outputs

| N (in order) | Independent experiment/control | M: measure together (separate readouts) | Expected decision gate |
| --- | --- | --- | --- |
| **N0** | Authenticated L0/L1 baseline and two **new** equivalent internet-off, nesting-enabled, nonprivileged Ubuntu test Boxes | source build, native project, exact Box UUID, Incus native ID, pre-existing inventory, network, UID/GID map, kernel and AppArmor features | Stop if creation is not provably SUCCEEDED or fixture differs in security-relevant way. |
| **N1** | Root and nonroot identity × layer; read-only baseline | labels from /proc/self and PID1, visible AA policy NS, .ns_stacked, UserNS, mapped range, effective/bounding capabilities, NNP, seccomp, effective GID | Distinguish relative "unconfined" view from actual ancestor enforcement; no assumptions beyond visible label. |
| **N2** | **E5 repair:** one individual, sandboxed, time-limited stat/list/read operation at a time across a fixed whitelist of AppArmor securityfs nodes. Flush a marker **before/after every step** and isolate each probe in a separate subprocess. | path, op, errno, elapsed, child exit/timeout; directory metadata, one bounded profile text read, mountinfo | Do **not** do all-or-nothing virtual file enumeration. Halt on single timeout; no profile load/replace/remove. |
| **N3** | Root/nonroot NSFS queries with parent/owner negative and direct readbacks | NS_GET_USERNS on pid/mnt/net/uts/ipc/cgroup, NS_GET_PARENT errno, NS_GET_OWNER_UID overflow value, AA current/ns/stacking, maps, caps | Cannot infer unexposed numeric kernel depth. |
| **N4** | Root bounded nested `unshare -Ur` stages and nonroot one-level test | at depth 0/1/2: UserNS inode, owner_uid, uid/gid map, AA NS, label, capability, subprocess exit, timeout | A new UserNS alone must never be called new AppArmor policy admin. |
| **N5** | Raw `unshare(CLONE_NEWUSER)` under UID/GID 1001, 1002, 65534; multi-layer matched-pair controls; optionally UID0 for comparison | errno, uid/gid pair, map location, CapEff, NNP, SeccompFilters, AA label/ns, sysctl | Distinguish consistent layer policy from isolated UID/mapping and seccomp numerical mode; do not claim a specific LSM without audit. |
| **N6** | Disposable local file ownership/map check (only if prior safe) | stat owner before/after temp file owned by root/UID1001/1002, mapping interpretation, permission failures, cleanup | No shared mounts or host file modifications; merely verifies how user-visible IDs behave. |
| **N7** | Exact STOP/DELETE and independent inventory postcondition | BoxWeave original+native receipts, terminal states, unmodified original inventory 4 L0/9 L1, no newly created UNKNOWN | If effects unresolved, do not replay/delete; preserve fixture, identify blocker. |

### Execution contract

- Collect **each** BoxWeave original Operation ID and original Incus native ID after per-tool `operation_get`; fetch `operation_output_get` and classify **outer operation state independently of Guest exit/command_success**. Original idempotency keys unique per distinct execution. Never execute old UNKNOWN identity.
- Phase order and isolation outrank attempt count. Do not run N3+ if N2 leaves a locked resource or ambiguous Guest. Later phases may be skipped with explicit reason.
- For N2: hard limit <2 seconds per filesystem node operation in a separately terminating child; **parent flushes marker first**, records precise timed-out path and exits. Avoid recursive/list-all-walks, profile writes and read-to-EOF on virtual files. A single abnormal child fails that cell; avoid mass retry.
- The L0 and L1 guest labels need **not** be assumed equivalent. Differences identify follow-up questions, not proof of enforcement or policy delegation.
- Preserve old inventory, no L0/L1 host Exec dispatch outside newly created owned disposable Boxes, no service/systemd mutation, no policy modifications.
- At end append observed N×M evidence, new distinctions, exact receipts and cleanup to this file, link in main BH019 hypothesis and PR conversation. Keep formal **BH-019 BLOCKED** unless genuine per-depth independent policy load/replace/remove/negative enforcement is verified.

## Execution results

Pending controlled run. Do not treat planned N cells as tested results.
