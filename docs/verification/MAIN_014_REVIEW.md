# Main completion — 0.14.0 candidate

Source: working tree based on `08efb06997bf60e6a1f58b74f742c4300d265d5e`, state modified. SRC-097 owns the explicit test authorization. Evidence below is current testing, not a retroactive change to old reports.

## Confirmed baseline and fixes

GitHub source 0.13.10 [run40985891994](https://github.com/vanishs/boxweave/actions/runs/40985891994) completed successfully. Assigned non-root developer box: boxweave-tenant-dev; isolated source `/home/boxweave-tenant-dev/main-completion-20261030/source`. Python3.12.3, Node18.19.1, npm9.2.0; locked dependency installation exit0. Original baseline:182 unit tests, exit0,122.097s; frontend typecheck/build exit0. Initial full validation failed the existing Graph browser mobile-fit assertion: controlled ReactFlow nodes lost measured sizes on editor renders. Preserving ephemeral measured dimensions fixes the actual viewport failure without persisting them in graph documents. The focused graph workflow then passed; the complete seven-browser-only run passed, exit0, report `.runtime/validation/a1e4b13fb4bd43bb8d3a167cde0799ba/result.json`. Chromium141.0.7390.37. These workflows are real browser/HTTP/SQLite with fixture infrastructure.

GitHub CI now includes the same seven browser workflows, selected evidence uploads and a locked-wheel application candidate build. This changed workflow has not run on GitHub until source publication; baseline CI cannot certify new working-tree code.

## Actual Windows matrix

Windows Python3.12.14; Codex0.166.0-alpha.5.1 and ZCode core0.17.0. Four new private test profile directories under the owner workspace's `.runtime/main-completion-20261030/windows`; existing dedicated restricted identities/pins/18447 endpoint reused without reading credential contents or changing daily config.

`scripts/windows_profile_check.py --descriptors <four reviewed descriptors> --result <private result>` exited0. Codex/native, Codex/Ubuntu WSL, ZCode/native and ZCode/Ubuntu WSL each performed initialize/initialized, tools/list101 and a real list_containers call. Each passed repeat-registration idempotence, diagnosis byte-preservation, wrong payload/version/pin negatives; WSL rows rejected a missing distro without fallback. Codex registration was verified through its CLI and ZCode through its actual core. No new GUI observation or model request was performed. The public sanitized matrix is retained separately from profile/private state.

## New application bootstrap lab

Dedicated outer target: `bw-main-install-1030`, exact instance `bw-main-install-1030-container`,10.206.30.12, Ubuntu24.04 x86_64,2CPU/3GB/12GB. Dynamic MCP identity check stdout `bw-main-install-1030` and `bw-main-install-1030-container`, exit0. No public route. Incus6.0.0 was installed from Ubuntu's signed repository in this dedicated LXC; Provider0.120.1/MCP binaries were verified against exact GitHub release-asset digests:

- containariumd: `7d992e5df41d7070c187e8b11d516d61e8b89b4b7be677f7a0880b9cfed5d4bd`
- mcp-server: `5b12e284b3b103c01bf9eff0cfe2f2a1e3b25476dfcc72aebe8166b4235198a2`

The dedicated Provider service listens on local18080. Its optional nested core-postgres creation failed under Incus6.0.0; policy storage and bridge ACL support are unavailable. No core container was deleted/reconfigured/restarted to force success. This lab therefore proves the recorded application/MCP/CLI paths only, not native network enforcement, all Provider features or a bare-metal/VM deployment.

The new explicit bootstrap creates `bwapp1030`, a user bus/linger, root-controlled independent launcher, mode0400 app credential with explicit non-secret scopes, and a credential refresh oneshot/timer. Actual failures retained: restrictive umask made launcher parents untraversable (fixed with explicit public traversal modes and regression); upstream MCP refused local HTTP without its explicit dev/test option (fixed only for validated127.0.0.1); first administrator returned ADMIN_SETUP_REQUIRED as designed. A temporary lab administrator was initialized from an in-process random password, never output or retained in plaintext.

Bootstrap rerun returned exit0/state ready; timer independently active/waiting/enabled. Actual independent initialize/discovery58 and list_containers read passed as the new non-root app user using its own scoped token-file launcher, Provider0.120.1. Scope filtering explains the lower tool count versus the old unrestricted endpoint.

A one-off large transfer closed the original MCP client connection. A new proper MCP session through the existing configured restricted launcher recovered initialize/initialized, tools/list and actual get_system_info. All subsequent public package transfers used small chunks and matching source/target SHA256; no host SSH/credential/trust changes were made.

## Final acceptance and delivery boundaries

Final main source regression on the corrected convention:192 unit tests63.054s, knowledge links and frontend typecheck/build all passed, exit0. The report is `.runtime/validation/6b8a9d3aa9554cf98d8a03ba26a1e365/result.json`. The earlier complete seven-browser pass remains separate; documentation/CLI-alias changes do not alter those UI paths.

Actual installed `boxweave.sh version --components`, `status` and `doctor --mcp` passed. A missing installation returned2. Exact local Incus inspect observed the existing stopped core-postgres metadata without modifying it; a nonexistent exact name returned1 and an invalid path/name returned2. App HTTP401 was recorded as unauthenticated reachability; the independent MCP initialized/discovered58 tools and performed list_containers successfully.

A deliberate test-only credential permission failure made an actual same-package install return2/MCP_READ_FAILED and retain its marker. The dedicated credential oneshot completed inactive/dead with Result=success/ExecMainStatus=0; its timer was active/waiting/enabled. After rotation the same install returned ready/exit0, retained a consistent SQLite backup, cleared the marker and passed doctor --mcp. No secret/JWT/private-key contents were read or output. This is live app/Provider integration with an injected availability failure, not a whole-host rollback or proof of all Provider capabilities.

SRC-099 narrows the public README convention to obsolete product/stage narratives. Normal archive vocabulary/options and upstream API versions are allowed. New service display descriptions use BoxWeave without v2 branding; persisted unit filenames, directory defaults, route ownership and key comments keep compatibility. The old service description is accepted only through the existing exact managed-byte/legacy migration checks.
 No new source commit, push, public release, original deployment update or model inference is implied.

The reviewed offline app-account bootstrap is functional on a separately prepared Provider. The public zero-argument installer still intentionally returns DEFAULT_INSTALL_UNAVAILABLE: a published application release and complete verified empty-host Provider/SSH/client-trust/reboot path do not exist yet. RISK-610 stays open. Existing source/version and retained primary runtime are separate facts. Test LXC, app state, backups and logs are retained; no automatic old0.5.1 migration or whole-host rollback is claimed.

## Final archive identity and installed update

Final public archive `boxweave-0.14.0-final.zip`,903956bytes, SHA256 `4fcb89219c9d5078f1c219ae5b218e2d61b84480a7b4d4124c2ea4648dbde5b4`. Its manifest records base commit08efb069 and state modified, not a fictitious clean commit. Bootstrap source SHA256 `51c62c290ab7f553800a9a750955118fe8211c4cb8354901da4a06d835966c81`; distribution source SHA256 `587d302e9259cbfabeeb7a515f55acdf829ee5cf4b9f42bb37bf0d52c1bcd178`.

The final archive was independently checked on developer box, Windows and the dedicated installation LXC. Its packaged `install.sh bootstrap --archive ...` updated the existing test app successfully, exit0/state ready. Saved configuration SHA256 stayed `73d6c1e049c6c5233fd3bdb0d264ca7fc54de7a03b946058ccb50c70f7c43744` before/after. Existing managed service filenames were retained; actual Description became `BoxWeave serve`. API/worker were active/running/enabled, credential timer active/waiting/enabled, update marker absent and final doctor --mcp authenticated read passed. A consistent database backup was retained.

After the service-display compatibility change,40 focused distribution/bootstrap/installer/README tests passed, exit0,13.075s; shell syntax and knowledge checks passed. The README rules also permit an external vendor's v2 application reference: the rule is about BoxWeave branding, not every upstream use of v2. No source commit/push/release was performed.
