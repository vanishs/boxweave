# Phase 2 R-01 / BOI-1 read-only Box façade (source candidate)

**Scope:** `boxweave_box_read_file` and `boxweave_box_list_directory` are server-owned MCP tools exposed by the *existing* SSH and HTTP gateway. Both delegate to the existing `connect(exec)` Provider tool and require the same configured operator allowlist permission for `connect`, a fresh trusted registry and exact container identity. No extra guest execution authority or app-Box agent is installed.

The remote script is fixed server code, invoked using bounded base64 JSON. Paths are strictly relative to `/home/<verified guest username>` and reject parent traversal, absolute paths, NUL/control characters, and every symlink component. File reads are limited to 1 KiB per request; listing pages at most 32 entries; both return bounded typed JSON. This is **not** arbitrary shell confinement and it requires `python3` already in the Box. The original `connect(exec)` is unchanged; its effects are **untracked** until R-02/R-04.

**Security:** A denied local tool policy, missing `connect`, stale trust data, nonzero guest command, malformed result, unsupported client arg or Box replacement fails without returning untrusted file contents. No read paths or output are persisted into task receipts or the diagnostics journal. No client-chosen host/user/SSH key or root directory. Root helper `box_reads.py` is included in independently hashed public release assets.

**Testing:** `tests/test_box_reads.py` includes path/injection, symlink, byte offset, result envelope and standalone helper import fixtures. Full CI should run the existing SSH/HTTP gateway regression, site browser and source bound packaging. **Runtime proof is still U:** installed SSH/HTTP, Python availability, byte-for-byte approved Box read and mixed-version helper acceptance must be verified on a disposable authorized Box.

Other second-stage work R-02 recoverable mutating tasks, R-04 cross-entry resource arbitration, and R-01 guarded edits are independent and NOT claimed here.
