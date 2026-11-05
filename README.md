# BoxWeave

**Your agents. Your machines. Connected.**

BoxWeave is a self-hosted control plane for AI coding, remote development environments, and an independent Web console. Containarium and Incus manage the underlying containers.

**Architecture: MCP-first + REST-assisted hybrid (owner decision 2026-11-05).** Agents use BoxWeave's SSH/stdio or authenticated HTTP MCP; upstream Containarium MCP remains the default for new and specialist operations. The application keeps its seven previously implemented authenticated loopback REST lifecycle methods as a scoped auxiliary adapter. The proposed batch migration of 29 additional MCP tools to REST and a universal API-first architecture were abandoned; existing deployed routes were not reversed. [Architecture and safety boundaries](docs/design/ARCHITECTURE.md), [ADR-010](docs/design/ADR/0010-api-first-control-plane-candidate.md).

**Current source version: 0.46.2 (selftest journal atomic admission and complete CLI help discovery, retaining the earlier Release CDN fallback). Default stable installer: v0.43.1.** The immutable v0.43.1 release passed source CI and a separate isolated 0.43.0-to-0.43.1 installed upgrade; the owner's current host installation must be verified independently of source publication. See [release and promotion evidence](docs/verification/STABLE_PROMOTION_0431_2026-11-03.md). The immutable v0.43.0 release passed CI, an isolated Ubuntu 24.04 in-place upgrade from v0.42.12, authenticated live Provider REST/MCP reads, and a real app lifecycle-to-Agent-D2-trace check with cleanup. See [v0.43.0 stable promotion evidence](docs/verification/STABLE_PROMOTION_DC01_0430_2026-11-03.md); this does not certify the current owner host installation, remote OAuth, all Providers, or alternate platforms.

**v0.44.10 selftest operator plan handoff:** The root-owned host CLI reads only an authorized caller- or app-owned regular 8 KiB-or-smaller plan, rejects symlinks and untrusted owner/type, and pipes bytes to the unprivileged app without exposing a private home directory. The application still requires the exact confirmation and journals remote effects; typed missing/unreadable/unsafe/oversized/invalid failures occur before mutation. [Release and acceptance notes](docs/verification/SELFTEST_PLAN_HANDOFF_04410_2026-11-05.md). **Owner-host R-02 post-reboot verification remains explicitly deferred; no host restart or global R-04 activation.**

**v0.46.2 selftest concurrent-start safety:** Journal admission now checks any running selftest and durably records the next run under the same SQLite BEGIN IMMEDIATE transaction, refusing racing Agents before another create. Two concurrency regressions failed before the fix and passed afterward; three separate serial live selftests on official v0.46.1 each completed create/hello/stop/delete with confirmed absence (source candidate vs installed release acceptance are distinct). `boxweave --help` also advertises all five already callable early-dispatch families without changing their security controls. Compose issue #79 remains open/waiting upstream. [Detailed verification](docs/verification/SELFTEST_CONCURRENCY_0462_2026-11-05.md).

**v0.46.1 bounded Release-download fallback:** If GitHub Release asset CDN returns HTTP 502/503/504 for the first-stage streamed installer, BoxWeave can fetch the exact published asset from GitHub’s release asset API instead. It verifies the exact asset identity, source tag and unchanged SHA-256 before executing; invalid or tampered data still fails closed. This does not move the default stable selector. [Reacceptance evidence](docs/verification/REACCEPTANCE_0461_2026-11-05.md).

**v0.46.0 selected debug REST:** With a configured authenticated loopback Provider API, the existing Web debug-summary uses GET /v1/containers/{username}/debug after exact-target checks and withholds all journal/shell/cause text. The standalone Agent Gateway retains inbound MCP and its root-owned tool policy; selecting the same outbound REST read requires a separate root-owned debug-rest.json opt-in (no JWT copied into that file). Old Gateway installations with no opt-in keep upstream MCP; the app can independently disable the new read with debug_rest_enabled=false without reverting lifecycle REST. Rejected REST reads never silently fall back. [Implementation and evidence boundary](docs/verification/REST_DEBUG_CONTAINER_0460_2026-11-05.md). Stable installer remains v0.43.1.

**v0.45.0 R-04 UNKNOWN manual recovery (candidate):** If an original remote write is genuinely uncertain, BoxWeave still blocks conflicting writes and never retries automatically. The approved simplest escape hatch is the sudo-only, two-step `boxweave task recover inspect|release|audit` with exact original task/claim identity, short-lived one-use challenge, explicit risk acknowledgement, and a durable audit-before-unlock transaction. The original receipt remains UNKNOWN, and remote work is **not** cancelled by releasing a coordination claim. Do not run against original retained failures without specific operator review. [Command and acceptance procedure](docs/verification/R04_UNKNOWN_RECOVERY_0450_2026-11-05.md).

**v0.44.9 P1 dispatch-intent and test isolation:** An in-flight or crash-interrupted coordinator claim in dispatch_intent now returns RECOVERY_REQUIRED instead of ambiguous RESOURCE_BUSY to competing writers. This is a classification-only, fail-closed change: it does not mark the original operation terminal, release claims, or replay work. Six previously ambient-policy-sensitive Gateway test suites now explicitly isolate their legacy-mode fixtures; real protected-mode tests remain separately enforced. [Execution and acceptance ledger](docs/verification/REACCEPTANCE_0449_2026-11-05.md).

**v0.44.8 R-04 deadline regression:** An acknowledged, already-dispatched lifecycle operation reaching its deadline must update the shared coordination claim to `unknown` after the app receipt becomes durably UNKNOWN; pre-dispatch expiry never creates a claim or sends a Provider write. [Reproduction and verification](docs/verification/R04_DEADLINE_CLAIM_SYNC_0448_2026-11-04.md). The new source does not clear the two retained UNKNOWN incidents, authorize replay or change the stable selector.

**v0.44.7 R-04 isolated lifecycle fix:** A clean nested test host with protected multi-Agent coordination reproduced an acknowledged create that left the Box RUNNING yet prematurely terminalized the app receipt as `TRUST_FAILED`. The application actually calls the native Provider MCP launcher; adding a gateway-only identity tool was tested and rolled back as ineffective. The minimal fix retains fail-closed SSH host-key checking, waits within the *original* creation deadline to retry only a verified created Box's identity observation, never redispatches the create, and still immediately denies host-key mismatches for existing Box starts. A real patched guest created `bwweb-0448` with `connection_ready`, verified its exact identity, then handed its released shared claim to a deduplicated Gateway task; a separate lease conflict/holder exercise passed. [Scoped evidence and blockers](docs/verification/TRUST_READINESS_R04_0447_2026-11-04.md). The outer owner host remains at v0.44.6 until independently upgraded; R-02 there is still unprovisioned.

**v0.44.1 protected-unit upstream upgrade and version management (historical; 0.44.2 recovery fix below):** The development baseline is explicitly recorded as **Containarium v0.128.0**. An exact BoxWeave installation automatically provisions or raises a trusted older Containarium daemon and MCP server to that minimum; an already compatible newer upstream version stays untouched, with **no maximum constraint**. An operator may select a *published exact* upstream release with `--upstream-version X.Y.Z` or the *latest published* with `--upstream-latest`, separately from BoxWeave's `--version 0.46.1` release selector. After installation, `sudo boxweave upstream status`, `sudo boxweave upstream update --latest` and `sudo boxweave upstream update --version X.Y.Z` manage the local upstream; downgrades and unverifiable assets fail closed. The running daemon and authenticated MCP are checked separately. Automatic upgrades can restart the Provider and affect a connected fleet: follow the [standard upgrading and rollback SOP](docs/process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md) before use. This protected-unit host-swap fix is a source/CI candidate until it has independent isolated live acceptance; see [retained failure and 0.44.1 acceptance boundary](docs/verification/UPSTREAM_EXTERNAL_PROTECTED_UNIT_0441_2026-11-04.md). the original host is not changed by publishing.

**v0.44.4–v0.44.5 disk-grow-only safety (source tested; installed v0.44.5 pending):** Containarium 0.128.0 accepted a requested 25GB→24GB disk shrink despite advertising that only growth is supported. The Agent MCP gateway, Web lifecycle plan and application REST mutator now reject shrinks and unknown disk sizes before a write. Source checks cannot prove atomic cross-client enforcement or physical disk quota behavior. [Exact reproduction, fix and live acceptance gates](docs/verification/REACCEPTANCE_0444_2026-11-04.md). [Exact v0.44.5 source CI and immutable release evidence](docs/verification/REACCEPTANCE_0445_2026-11-04.md): 731 Python tests, seven browser fixtures, 205 document checks, Python 3.14 and release package hashes PASS. Installed cross-entry shrink refusal and SSH/guest execution remain unverified. Default stable installer pin is unchanged.

**v0.44.6 independent installed verification:** [Four-gate isolated real-Box acceptance](docs/verification/FOUR_GATE_ISOLATED_ACCEPTANCE_0446_2026-11-04.md) confirms clean v0.44.5 installation, installed upgrade to v0.44.6, authenticated nested create/exec/stop/delete, explicitly provisioned R-02 metadata-only task deduplication, and installed MCP/REST/Web-shared lifecycle disk-shrink refusals. Web's authenticated Flask route passes fixture tests; a real authenticated Web HTTP request on the installed service remains unverified. This does not upgrade the owner host or provision its task database. No further patch release is required from this test cycle.

**v0.44.6 connect(exec) displayed-name compatibility:** Gateway now resolves the displayed `<username>-container` name against the trusted registry, checks the same live incarnation/IP/state, and forwards the logical username to Provider. Unknown or ambiguous aliases fail closed and lookup errors are typed. Publishing does not upgrade the outer host; installed acceptance is separate. [Exact reproduction and release evidence](docs/verification/CONNECT_TARGET_ALIAS_0446_2026-11-04.md).

**v0.44.3 first-install MCP binary resolution fix:** Live fresh-Box testing of published v0.44.2 reproduced an application-stage failure: the installation path retained the pre-bootstrap MCP absence and looked for an MCP binary not distributed with the release. The application now resolves the root-owned trusted installed MCP binary after the Provider bootstrap completes, with a focused regression test. The official v0.44.3 fresh install and failed-install repair were independently accepted on isolated Boxes: each installer exited 0, with doctor 10/10 and authenticated v0.128.0 Provider/MCP. The default stable installer remains v0.43.1.

**v0.44.2 slow-startup recovery fix:** An external Containarium upgrade on an operator host could time out after 120 seconds waiting for core containers, then incorrectly flag an otherwise-restored previous binary as unrecovered after a 60-second rollback check. The source candidate extends both authenticated readiness windows to 240 seconds, budgets the upstream installer phase at 900 seconds, and reconciles a previous `prepared` checkpoint only after verifying the exact original binary digest and running daemon. It preserves prior evidence and does not change systemd confinement. Real host upgrade acceptance remains separate. [Incident evidence and follow-up](docs/verification/UPSTREAM_SLOW_RESTART_RECOVERY_0442_2026-11-04.md).

**v0.43.3 authenticated runtime version guard:** Beyond checking the locally installed root-owned daemon binary and MCP initialize version, deployment now checks the **running** Containarium daemon's authenticated `GET /v1/system/info` `info.daemonVersion`. A stale pre-upgrade process is refused as below minimum/unknown, even if a newer binary is already on disk. See [v0.43.3 follow-up verification](docs/verification/CONTAINARIUM_0128_RUNTIME_GATE_0433_2026-11-03.md).

**v0.43.2 Containarium compatibility candidate:** BoxWeave now requires **Containarium daemon and authenticated MCP server >= v0.128.0**, with no maximum or version allowlist. Clean-host bootstrap uses SHA-verified v0.128.0 binaries only as a minimum baseline. An existing Containarium Provider is **never** upgraded by selecting a BoxWeave version. Trusted root-owned upstream binaries can be updated independently, while the app refreshes its own exact-digest-reviewed snapshots. REST/MCP capability discovery, credential scopes and exact SSH identity remain enforced. [v0.128.0 review](docs/design/CONTAINARIUM_0128_COMPATIBILITY_REVIEW.md) · [repeatable upstream upgrade SOP](docs/process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md). Source/CI acceptance is separate from real host validation.

**v0.43.1 post-dispatch repair:** A Provider MCP error after a lifecycle write is now recorded as UNKNOWN rather than falsely claiming it was never submitted. Direct Agent create errors retain the original upstream error and add a typed readback hint; neither path retries, adopts or changes an unverified Box. [Fix and acceptance boundary](docs/verification/CREATE_OUTCOME_0431_2026-11-03.md).

Optional multi-Agent shared coordination requires separate reviewed policy/SQLite provisioning and opt-in. Snapshot/restore/clone writes remain unsupported without verified Provider contracts. Source CI, published installer version, running service version and production-host acceptance are distinct. Historical rollout and unresolved risks are retained in [the risk register](docs/verification/OPEN_RISKS.md).


## Install (Ubuntu 24.04+ recommended)

The desired default command is:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash
```

To install or update to an exact **published** BoxWeave release, including a historical release, use:

```bash
curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash -s -- --version 0.43.1
```

**Agent-safe diagnostic coordinator (released v0.43.0; isolated installed-host acceptance passed):** `boxweave_diagnostic_run` exposes the read-only `preflight`, `rest_read`, and `operation_trace` profiles to authorized SSH/HTTP MCP clients. It checks the fixed installed app runtime, performs independently classified authenticated REST/MCP inventory reads, and summarizes actor-scoped D2 REST operation traces without returning JWT, response bodies, host paths or raw stderr. It cannot run arbitrary host commands, create/delete Boxes, rotate credentials or mark a missing trace as success. The new tool was independently tested against installed v0.43.0 on a task-owned nested Containarium host; the original dev host is not automatically upgraded. [Design and acceptance gates](docs/design/DIAGNOSTIC_COORDINATOR_V1.md).

**App Provider REST basics (source v0.42.12):** Newly provisioned or safely migrated local application configs now use authenticated loopback Containarium REST for app-owned list/get/create/start/stop/delete/resize. The existing Agent-facing MCP, specialist network/Guard and SSH `connect(exec)` remain separate; creation with independent nesting still uses the platform MCP compatibility path. API requests record redacted classified spans linked to durable environment operation traces, with timeout/unknown writes never automatically replayed. The default installer now selects v0.43.1 after separate installed-host acceptance; the earlier v0.42.12 REST tests remain useful historical evidence. [Implementation and verification scope](docs/verification/PROVIDER_REST_A1_04212_2026-11-03.md).

**MCP compatibility diagnosis and protected local restore (released v0.42.11):** After suspected Containarium MCP schema drift, call `boxweave_diagnose` with `source=compat` for a bounded, read-only comparison of the small set of BoxWeave-dependent upstream tool contracts; an unknown result is not an assurance of compatibility. Agent-facing `restore_box_backup` is default-denied: an operator must preapprove the exact source Box, existing backup ID, target and private key alias through `/etc/boxweave/restore-policy.json`. A protected key file remains on the same MCP host under `.restore-keys`; the gateway does not read private-key bytes and cannot accept an arbitrary file path. An explicit backup-ID confirmation is needed and `force=true` is refused. This adds no KMS or Provider upgrade. See the [design and operator procedure](docs/design/MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md). Live installed Containarium v0.120.1 did not advertise the restore_box_backup MCP tool; the gateway policy is shipped but a real approved whole-Box restore cannot yet run on this pinned Provider. Whole-box backup creation over MCP remains a separate upstream P2 follow-up. [Live acceptance](docs/verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md).

**Containarium host-hop SSH wrapper (v0.42.10):** BoxWeave no longer
bundles `containarium-shell` in Release assets. On a BoxWeave-managed local
Provider, when the wrapper is missing, the installer fetches the upstream
source by an immutable Containarium commit, checks source and extracted wrapper
SHA-256 digests, and installs only the reviewed wrapper. No Containarium binary
is upgraded. For externally managed Providers, the SSH configuration remains
**read-only by default**; an operator can opt into the bounded repair using
`--repair-containarium-shell`, subject to an explicit Provider version check.

If the optional wrapper cannot be fetched or safely installed, the installer
reports `containarium_shell.status=incomplete` while preserving core
application/MCP installation where independent. It never runs upstream's broad
SSH setup script, grants unrestricted Incus sudoers, or silently replaces
foreign scripts. Existing login shells remain in place. Successful installation
does not establish a verified real host-hop session. See the
[decision and test plan](docs/design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md)
and [operator runbook](docs/process/CONTAINARIUM_SHELL_RECOVERY.md).

The developer-only [Agent deployment/test workflow v1.0](docs/process/AGENT_TEST_DEPLOYMENT.md) uses existing authorized BoxWeave transports, local worktree snapshots and verified artifacts; it does not add MCP tools or treat a source snapshot as a Release installer.

The optional `--version 0.X.Y` selects a GitHub Release verified against its published metadata, source tag, and SHA-256 digests; an unknown, unpublished, malformed or mismatched release fails closed before running the downloaded helper. Omit `--version` to use the separately promoted stable installer pin. The command can include the existing `--provider-url`, `--provider-secret-file`, and `--provider-cli` options. The version selector is for the streamed installer, not `plan`, `apply` or `bootstrap` modes.


**Ubuntu 24.04 or newer is the recommended host**, not a required distribution. The installer checks Linux x86_64, CPython >= 3.12, systemd, Provider/Incus availability and needed package tooling directly. On hosts without APT, prepare reviewed dependencies and Provider yourself; BoxWeave does not invoke an alternative package manager. Other distributions require separate live acceptance. It verifies source-bound release assets, provisions missing infrastructure on a clean host, **preserves an already configured Incus storage pool**, and **reuses an existing complete Containarium** without installing or upgrading it. Existing Provider identity, configuration, network and storage remain external. Partial installations are refused. See [existing Provider instructions](docs/verification/EXISTING_PROVIDER_INSTALL.md) and [two-box MCP test](docs/verification/STREAMED_INSTALL_TEST.md). The released offline wheelhouse includes CPython 3.12–3.15 x86_64 wheels; Ubuntu 26.04 requires live host acceptance. The supported client catalog covers Linux/macOS native SSH and Windows native/WSL SSH; real App acceptance remains pending.

A successful streamed install prints the verified installed BoxWeave version and
prominently shows `boxweave client`. For a fresh host without Incus, the
managed-Provider path runs an APT dependency simulation and selects a coherent
Incus package set without the old partial `7.5.1` pins; it never upgrades
an already-installed Incus. This does not yet mean that Incus/PostgreSQL
provisioning has been moved into an independently owned Containarium installer.

Persistent MCP-over-SSH client registrations use `ServerAliveInterval=15`
and `ServerAliveCountMax=4` to mitigate silent idle connection loss.
Keepalive detects an unresponsive connection but does not automatically
restart a terminated SSH/MCP session.

The Provider API and Web application stay on loopback; the MCP client uses the existing SSH service with an explicitly enrolled client public key and a verified server pin. No public route, firewall change or SSH host-key replacement is performed. The initial Web administrator password is saved only in a private file whose location is printed; it is never printed by the installer.
The explicit `bootstrap-plan` / `bootstrap` workflow can now prepare a dedicated application account, systemd user bus, independent scoped MCP launcher and rotating credential on an already reviewed Provider. It requires independently reviewed artifact digests and explicit root-change acknowledgment; see [application bootstrap](docs/HOST_BOOTSTRAP.md#independent-application-bootstrap).

On a prepared Linux x86_64 host with Python 3.12+, reviewed Containarium/Incus, a non-root application user with a functioning systemd user bus, an authorized MCP launcher and a trusted release archive, use the existing installer:

```bash
bash install.sh plan --app-user boxweave \
  --archive /reviewed/boxweave.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 INSTALLER_SHA256 --config /reviewed/application.json

bash install.sh apply --app-user boxweave \
  --archive /reviewed/boxweave.zip --sha256 ARCHIVE_SHA256 \
  --distribution-sha256 INSTALLER_SHA256 --config /reviewed/application.json
```

The paths and SHA256 values are placeholders, not ready-made downloads. The first installation may require local administrator initialization and a rerun. See [installation and repair](docs/INSTALL.md) and [host prerequisites](docs/HOST_BOOTSTRAP.md).

### Common installer parameters

| Option | Purpose |
| --- | --- |
| `plan` | Read-only prerequisites and artifact verification |
| `apply` | Install, update or repair from the reviewed package |
| `--app-user NAME` | Existing non-root application account |
| `--archive FILE` / `--sha256 HEX` | Application ZIP archive and trusted digest; `--package` is a compatible alias |
| `--distribution-sha256 HEX` | Trusted digest of the application installer code |
| `--config FILE` | Approved application/MCP configuration |
| `--prefix PATH` | Optional installation location in the application user's home |
| `--provision-provider` | Explicit empty-host provisioning only; requires a separately verified upstream installer, SHA256 and root-change acknowledgment |

Run `bash install.sh --help` for all parameters.

Streamed install/update/repair automatically preserves an already-installed complete Containarium and Incus. It does **not** adopt an unrelated BoxWeave installation or silently repair/upgrade a third-party Provider. For a preinstalled `dir` storage pool on bcachefs, first verify its source and mount dependencies; the installer will not reconfigure the storage pool.

## After installation: finish the setup

The installer prints a `next_steps` record and prominently shows the **first
action: configure your client**. As your ordinary Linux administrator login
(**no sudo**), run:

```sh
boxweave client
boxweave client --help  # More options
```

**Optional commands** for viewing the Web address and diagnosing the host (not
required before generating client configuration):

```sh
boxweave                # Host overview and connection instructions
boxweave --help         # Full host command reference
boxweave status         # Public Web reachability, not an authenticated MCP proof
boxweave doctor         # Safe host/service observations
```

The Web server is bound to **127.0.0.1** for safety, so its URL is
accessible on the server host only. To open it in your own browser, make an
authenticated tunnel from your local computer (use the actual port printed by
`boxweave`):

```sh
ssh -N -L 18640:127.0.0.1:18640 YOUR_HOST_ADMIN@YOUR_LINUX_HOST
```

Then browse to `http://127.0.0.1:18640` locally (replace 18640 when
the installed port differs). The initial Web administrator password is stored
privately at `/home/boxweave/boxweave/initial-admin-password`: an authorized
administrator can read it locally; the installer never prints the password.
The service account `boxweave` is deliberately different from your SSH
administrator and must **not** be used as an everyday login.

Run `boxweave client` to choose the MCP App, client OS (Windows, Linux or
macOS) and SSH transport (native or Windows WSL). It prints a reviewed
self-contained Python 3.11+ client-side source on stdout for **every**
App, operating system and SSH transport, including Windows Codex/WSL.
An authorized administrator must enroll only the client public SSH key.
After a real authenticated MCP read, the script updates only the selected
App's MCP configuration (or prints official import JSON for Qoder/TRAE).
It never rewrites the client's AGENTS files. See
[alphabetically sorted client App support](clients/README.md).

If an installation fails, inspect its original log/phase and repair that
specific issue before retrying. Do not relax `/home/boxweave` from mode 0700
or grant the human login broad Incus privileges just to run the CLI.
[Host CLI reference](docs/CLI.md).


## Connect an AI coding agent

Use `boxweave client` (interactive) or select a supported profile:

```bash
boxweave client --app cursor --os linux --transport native --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
boxweave client --app codex --os windows --transport wsl --admin-user YOUR_HOST_ADMIN > boxweave-setup.py
```

Supported Apps (**alphabetical**): **Antigravity, Claude (Code), Codex,
Copilot (CLI), Cursor, OpenCode, Qoder (IDE), TRAE (IDE), ZCode**.
Linux and macOS use native OpenSSH. Windows supports native OpenSSH or a
Windows-hosted App using WSL OpenSSH. Qoder and TRAE use official IDE JSON
import rather than edits to undocumented user configuration paths.

| Client platform | Agent | Transport | Support |
| --- | --- | --- | --- |
| Linux | All nine supported Apps | Native SSH | Generator/fixture; live App verification pending |
| macOS | All nine supported Apps | Native SSH | Generator/fixture; live App verification pending |
| Windows | All nine supported Apps | Native OpenSSH | Generator/fixture; Python 3.11+ required |
| Windows | All nine supported Apps | WSL SSH | Unified Python generator/fixture; WSL distro required |
| WSL-hosted Agent | Any | Independent Linux runtime | Not implicitly configured by Windows generator |

The Web Client Setup form retains its four earlier approved local
descriptor profiles; the installed `boxweave client` CLI supports the
expanded catalog. Generating a script or passing its isolated MCP probe
does not prove the App has actually loaded or granted MCP tools.
The generated stdout starts exactly with `#!/usr/bin/env python3`;
all decorative separators are Python comments **after** the shebang.
Interactive prompts/summary messages are comment-prefixed on stderr, so
redirect stdout or use `--output NEW_FILE` for a clean, executable `.py`.
See [client setup](clients/README.md) for the individual App destinations
and prerequisites.

## MCP discovery and BoxWeave self-information

The 0.24.4 source adds a read-only `boxweave_info` MCP tool.
Its optional `topic` argument accepts `all` (default), `overview`,
`architecture`, `development`, `capabilities`, `security`, and
`documentation`. This source-backed guide helps Agents understand and
develop BoxWeave using isolated test Boxes, but never reads host secrets
or claims to be a live availability probe. SSH/stdio and HTTP MCP
`initialize` advertise BoxWeave identity and safe-use instructions.
The gateway also corrects inherited `create_container`, `connect`
and deletion descriptions to reflect managed SSH identity and exact-target
trust. For live capabilities use `tools/list`, and for actual Box state use
`list_containers`. An optional MCP allowlist must include `boxweave_info`
to expose it. See [MCP Agent guide](docs/MCP_AGENT_GUIDE.md).

## Web console

The Web UI includes project and environment binding, reviewed lifecycle plans and durable receipts, Tenant membership and controlled network access, TCP/UDP port publication, Prompt Graph, model setup guidance for ZCode, supported client setup downloads, an MCP integrations panel, and optional Remote MCP credential management. Graphs do not execute commands and an HTTP response is not proof of authenticated MCP or effective network isolation.

## Common management commands

From a checked-out source directory or its matching installed release:

| Command | Purpose |
| --- | --- |
| `bash boxweave.sh version` | Read the local BoxWeave version |
| `bash boxweave.sh version --components` | Also observe Incus/Containarium CLI versions |
| `bash boxweave.sh status` | Observe application and user service state |
| `bash boxweave.sh doctor` | Read-only host and service diagnostics |
| `bash boxweave.sh doctor --mcp` | Additionally run an explicitly authorized MCP read |
| `bash boxweave.sh box inspect NAME --project default` | Inspect one exact Incus instance |
| `bash boxweave.sh client` | Display supported client profile guidance; no enrollment |

These commands do not automatically install missing components or modify the host. The command set is smaller than the historical implementation. See the [CLI reference](docs/CLI.md) for exit codes.

## Documentation

[Install and update](docs/INSTALL.md) · [Host prerequisites](docs/HOST_BOOTSTRAP.md) · [Client setup](clients/README.md) · [CLI reference](docs/CLI.md) · [Troubleshooting](docs/TROUBLESHOOTING.md) · [Supported scope](docs/product/SCOPE.md) · [Development and tests](docs/DEVELOPMENT.md) · [Open risks](docs/verification/OPEN_RISKS.md)

Container-host setup: see [nesting configuration and restart guidance](docs/NESTING.md).

## HTTP-first Agent client setup (v0.28.0 source)

The installed `boxweave client` interactive wizard now recommends **HTTP MCP**, while `--method ssh` and all old `--transport` / `--admin-user` SSH script invocations remain compatible. Use `boxweave client --method http --app cursor --os linux --url https://YOUR_HOST/mcp --no-replace > boxweave-setup.py`; transfer and run the self-contained Python 3.11+ script on the *client*. It safely adds only that App's BoxWeave HTTP MCP entry (or prints reviewable JSON for Qoder/TRAE), with private backups on explicit `--replace`. The client App performs OAuth and final read verification; configuration alone is not acceptance. See [HTTP-first design](docs/design/HTTP_FIRST_CLIENT_SETUP.md) and [client instructions](clients/README.md).

## Install-success HTTP MCP onboarding (v0.27.2 source)

The final human-facing successful installation message now displays the effective loopback MCP URL, local information page, status/doctor/restart commands, and explicit Cloudflare Quick Tunnel setup and ChatGPT OAuth steps. A configured public URL is printed as an unverified endpoint (not claimed reachable). An explicitly disabled HTTP MCP shows a disabled notice rather than encouraging exposure. This complements the existing machine-readable `next_steps` JSON; no Tunnel, DNS, firewall, or public listener is created automatically. The exact v0.27.2 host-upgrade and startup/reboot checks remain separate live acceptance gates.

## Remote HTTP(S) MCP (v0.27.0 quick-start source)

The installed application now starts a **managed authenticated loopback-only HTTP MCP**
service on `http://127.0.0.1:18844/mcp`, unless `remote_mcp.enabled=false`
was explicitly configured. This is not an anonymous endpoint or a public listener.
The local `http://127.0.0.1:18844/` page explains its purpose.

```sh
boxweave http status
sudo boxweave http doctor
# Install cloudflared separately; run the temporary tunnel yourself:
cloudflared tunnel --url http://127.0.0.1:18844
# After cloudflared prints an HTTPS hostname:
sudo boxweave http configure --public-url https://YOUR-HOST.trycloudflare.com
# Paste https://YOUR-HOST.trycloudflare.com/mcp into the ChatGPT MCP client.
```

Tunnel creation never runs automatically. An unchanging hostname is recommended
for reliable ongoing OAuth connections; Quick Tunnel addresses can change.
See [HTTP MCP quick-start design](docs/design/HTTP_MCP_QUICKSTART.md),
[operator guide](docs/REMOTE_MCP.md), and [verification](docs/verification/REMOTE_MCP_TEST_PLAN.md).


The opt-in Streamable HTTP MCP endpoint supports local-admin-consented OAuth, API Bearer Tokens, and one-field credential URLs while preserving SSH/stdio. The separate HTTP listener is **loopback-only**; choose an independently configured HTTPS reverse proxy, external Tunnel, or explicitly invoked Caddy runner. No SSH profiles are automatically migrated. See [remote MCP configuration and known limits](docs/REMOTE_MCP.md). Reused nested-Box and independent loopback TLS-proxy tests are recorded. On 2026-11-01, an owner-operated Cloudflare Quick Tunnel and hosted ChatGPT OAuth reached real MCP container listings after a process-local browser-header hotfix; v0.24.2 makes that fix permanent in source; v0.24.3 additionally preserves those headers through managed Caddy and reports bounded local TCP status in Web. The newly published v0.27.0 package has not been deployed/tested on that live host; production HTTPS and other clients still require separate acceptance.


## D3/D4 agent diagnostics (0.25.0 source)

The existing SSH/stdio and optional authenticated HTTP MCP support `boxweave_diagnose` for classified gateway/app evidence, and `boxweave_project_diagnose` for restricted exact-Box project checks. Gateway recording is enabled by default (7 days and 50 MiB, configurable); original tool result bodies are unchanged. **D4 project inspection defaults DENY** until a server administrator allowlists a specific Box incarnation, checks and service names. See [operator configuration](docs/DIAGNOSTICS.md), [D3/D4 master](docs/design/DIAGNOSTICS_D3_D4_MASTER.md), and [implementation batches](docs/design/DIAGNOSTICS_D3_D4_IMPLEMENTATION.md). Never share reports or submit external issues before the user personally reviews sensitive data and separately approves the final submission. CI is not real-host acceptance.
