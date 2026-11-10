# S2 Exec recorded-output physical storage and retention gate — 2026-11-10

**Gate: OPEN / NOT S2 PASS.** This document distinguishes a verified API **read** cap from a proven **physical disk** cap and from an explicit log lifetime. This is a research-derived test plan, **not** a native disk-quota or garbage-collection PASS.

## What is proved

- BoxWeave's `operation_output_get` refuses offsets above 16MiB per stream, returns bounded 64KiB pages, and signals truncation on the public response. One real Incus guest generated **16MiB + 4 bytes in each of stdout and stderr** and the public read cap was independently verified. This does **not** limit bytes already written by Incus.
- In pinned upstream Incus **v7.6.0**, [`instance_exec.go`](https://github.com/lxc/incus/blob/v7.6.0/cmd/incusd/instance_exec.go) uses `post.RecordOutput` to create separate `exec_<operation>.stdout` and `.stderr` files with `os.OpenFile` beneath `inst.ExecOutputPath()`, then supplies their paths in the terminal native operation metadata. The writer does not set a per-stream byte cap in that path.
- In [`driver_common.go`](https://github.com/lxc/incus/blob/v7.6.0/internal/server/instance/drivers/driver_common.go), `ExecOutputPath()` is `filepath.Join(d.Path(), "exec-output")`; `RootfsPath()` is a sibling `rootfs`. Whether an Incus storage driver actually enforces a space quota across this sibling directory **must be tested**, never inferred from an instance root-disk `size` config.
- The one-use native CI uses a separate **4GiB loop-mounted ext4 filesystem** for a **dir** storage pool; this isolates that test from the runner's root filesystem but does not prove a production per-Box output quota. The selected-storage separate-mount check is a precondition, not a substitute for output size and retention testing.
- Upstream exposes [GET/DELETE recorded-output file endpoints](https://github.com/lxc/incus/blob/v7.6.0/cmd/incusd/instance_logs.go). The SDK's regular synchronous recorded-output handling may delete files once consumed; BoxWeave's detached operation uses separately persisted native log paths and must not delete them while a still-authorized reader expects them. A typical upstream expiry of ~48h is not a contractual guarantee of this candidate's exact available lifetime.

## Required independent tests before closing this gate

1. On a fresh disposable mounted/owned pool, retain an original Box and inspect **both actual Incus files on disk** while they still exist. Compare independent `stat` byte sizes with the 16MiB API boundary, filesystem used/free bytes, and the exact original native Operation ID. Do not use API truncation as the disk witness.
2. Prove storage confinement for the selected driver: occupied bytes, pool and guest quota enforcement, exhaustion handling and recovered disk availability. A bounded isolated **test device** is mandatory; never fill a production/shared mount and never silently assume `dir` honors per-Box `size`.
3. Define a finite, measurable retention budget and expiry policy. If cleanup uses the native API, it must target only fully verified, same-Project, same-UUID original log paths, with durable terminal status and an auditable decision. Original UNKNOWN/in-flight output and foreign instance logs are not disposable.
4. Independently recheck after server restart, native operation/log expiry, a disconnected reader and deliberate partial cleanup/failed deletion. The MCP response must clearly distinguish **available**, **truncated** and **expired/unavailable** bytes, rather than inventing retained output or guest success.
5. Run a negative control that would have exhausted a non-isolated host if no cap existed, but bound that experiment **strictly within the disposable 4GiB lab mount**. Require real native post-state and zero residual test instances. Do not claim a cap merely because a short producer stopped before exhaustion.

## Product constraints and current decision

Do **not** add a host file-deletion daemon, generic host shell, second native transport, guest Agent or destructive automatic cleanup to make this green. Research driver capabilities and typed official-SDK paths first; retain safe UNKNOWN/no redispatch semantics for any ambiguous native outcome. If durable detached output and a hard safe storage bound cannot both be demonstrated with the selected Incus mode, mark the specific S2 Exec storage requirement **BLOCKED**, not PASS, and continue unrelated S2 acceptance.

**References:** [S2 crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md), [phase contract](../development/phases/02_BOX_LIFECYCLE_OPERATIONS.md), [native source `instance_exec.go`](https://github.com/lxc/incus/blob/v7.6.0/cmd/incusd/instance_exec.go), [native log API](https://github.com/lxc/incus/blob/v7.6.0/cmd/incusd/instance_logs.go).
