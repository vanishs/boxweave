# Current phase status

Authority for authorization/gates only; product version is [VERSION](../../VERSION), risk state is [OPEN_RISKS](../verification/OPEN_RISKS.md).

Phase7 is authorized by the user's supplied instructions (SRC-089). Entry audit accepts the phase6 closure of reviewed P0/P1 items; P2 deployment/evidence bounds are unchanged. Phase7 maintenance is complete within its tested scope (SRC-091); final result and actual command/evidence details belong to [PHASE_07_REVIEW](../verification/PHASE_07_REVIEW.md).

No deployment, push, inference, destructive cleanup or host/provider change is authorized by source/document cleanup. Source completion never implies the retained test deployment was upgraded. Future work requires its own request and the [MAINTENANCE](MAINTENANCE.md) path.

Historical gate/checkpoint text is preserved in [PHASE_STATUS_HISTORY](PHASE_STATUS_HISTORY.md); it must not override this file.

## 2026-11-03 Containarium minimum-only P0 upgrade candidate

Owner approved source compatibility, documentation, immutable candidate release and independent future operator-host acceptance. Containarium runtime minimum is 0.128.0 with no upper version bound; existing Providers are never automatically upgraded by a BoxWeave release. Candidate is [v0.43.2 source work](../verification/CONTAINARIUM_0128_COMPAT_P0_0432_2026-11-03.md); real 0.128.0 host acceptance is deferred to operator, and `PUBLISHED_RELEASE` + default streamed installer remain 0.43.1. [General SOP](UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md). This source authorization does not authorize real host/Provider writes.

## 2026-11-03 v0.43.2 released; v0.43.3 running-daemon gate candidate

[PR #65](https://github.com/vanishs/boxweave/pull/65) merged v0.43.2 to `952b7b6`; exact PR and main CI PASS, immutable Release v0.43.2 published with default stable selector still 0.43.1. Follow-up [v0.43.3 runtime gate](../verification/CONTAINARIUM_0128_RUNTIME_GATE_0433_2026-11-03.md) requires authenticated running daemon `info.daemonVersion >=0.128.0`, not only on-disk binaries; future version maximum remains **none**. No operator Provider or real-host application update was performed.

## Source branch promotion

SRC-092 separately authorizes publishing v2 as main and preserving the old implementation as legacy/pre-v2. This overrides the earlier no-push phase scope only for this branch operation; no application deployment, host change or runtime test scope is added. Completion/verification belongs in [BRANCH_PROMOTION](BRANCH_PROMOTION.md).

## 2026-10-29 remediation authorization

SRC-093: The owner authorizes R-00–R-09 source remediation in small direct commits to main, no PR and no force push; restoration of fresh-host install/update/repair and management CLI is in scope, starting at 0.10.0. No automatic 0.5.1 data migration or additional Agent platforms. This supersedes older no-push restrictions **for repository source commits only**; it does not authorize any host, Provider, secret, network, public TLS or deployed application changes. See [remediation audit](../verification/ARCHIVE_051_TO_MAIN_AUDIT.md) and [execution queue](../verification/REMEDIATION_TEST_QUEUE.md). Current phase 7 historical results remain unchanged.

## Current main-completion work

SRC-097 separately authorizes the isolated 0.14.0 implementation and experiment scope. Results and remaining delivery boundaries are recorded in [MAIN_014_REVIEW](../verification/MAIN_014_REVIEW.md); this does not upgrade the original deployment or authorize public exposure, model spending, destructive cleanup or source publication.

## Nesting source remediation

SRC-103 authorizes implementation and commit/push. SRC-100 dedicated-test scope remains; source publication does not authorize original parent-host changes. [Current source and actual evidence](../verification/NESTING_VERIFICATION.md).

## Remote MCP R0–R4 source delivery authorization (2026-10-31)

The owner requested completion of [PR #10](https://github.com/vanishs/boxweave/pull/10) R0–R4 and merge into main **without entering a real Box**. v0.24.0 source adds loopback remote MCP, local-admin OAuth, API Bearer, opaque path URL, optional Caddy/manual HTTPS ingress, and Web credential controls. It does not authorize public listener/TLS/firewall/DNS/Provider mutations. CI fixture results are not production OAuth, client or Box acceptance. [Handoff](REMOTE_MCP_HANDOFF.md) and [test plan](../verification/REMOTE_MCP_TEST_PLAN.md).

## ARCH-0 interstitial architecture gate (source audit only)

The owner requested a code-first modularity investigation before
continuing the next broad integration stage. The
[ARCH-0 review](../design/ARCHITECTURE_CONSOLIDATION_GATE.md) was
added to the [roadmap](../design/FEATURE_ROADMAP_MASTER.md) after
P1-A/P1-B1 and before P1-B2. This does not authorize an indiscriminate
rewrite, host installation, Provider mutation or release promotion.
ARCH-0 implementation and installed acceptance remain pending.

## ARCH-0 A/B/C source delivery acceptance

Owner explicitly authorized implementing ARCH-0 in staged tested commits.
ARCH-0A [CI 41408681895](https://github.com/vanishs/boxweave/actions/runs/41408681895),
ARCH-0B [CI 41409106116](https://github.com/vanishs/boxweave/actions/runs/41409106116),
ARCH-0C [CI 41409883731](https://github.com/vanishs/boxweave/actions/runs/41409883731):
full fixture CI PASS. [ARCH-0 source acceptance](../design/ARCHITECTURE_CONSOLIDATION_GATE.md).
No installed service, Provider privilege, Box write or host migration is proved.
P1-B2 and R-02/R-04 dispatch integration remain NOT IMPLEMENTED.

## Phase 2 R-03 first-use guide (source candidate)

Read-only `boxweave setup --mode ssh|https|local [--check] [--format json]` reuses the installed root-reviewed public descriptor; no connection secret, enrollment, public HTTPS assumption, app login, or host mutation. Independent fixture/CI verification pending; live SSH/HTTPS client handshake not yet tested. R-01/R-02/R-04 remain separate feature tracks.

## Phase 2 R-02 gateway-private task receipts (staged candidate)

Metadata-only `gateway_tasks.py` uses private app-user SQLite, stable random request keys, exact observed Box incarnation digest and crash-safe dispatch-intent. `boxweave_task_submit/get/list` use the original trusted `connect(exec)` permission and do not store commands, outputs or credentials. Provisioning is separately authorized with `sudo boxweave task provision`. This is **gateway-tracked only**; legacy `connect(exec)` and Web/app work remain untracked by this store. No R-04 cross-entry coordination coverage is claimed. Fixture CI and installed acceptance remain separate.

## Phase 2 R-04 lease core (source candidate)

Adds opt-in 10-minute hashed-token Box leases inside the existing minimal SQLite operation coordinator. Expired leases do not release UNKNOWN dispatched claims; independent Boxes remain concurrent. Source/fixture CI must pass before any tool exposure or installed/global coordination claim. Actual Web/SSH/HTTP guarded mutation routes are still NOT IMPLEMENTED.

## Phase 2 staged source acceptance — 2026-11-02

[Exact CI/Release evidence and outstanding gates](../verification/PHASE2_SOURCE_ACCEPTANCE_2026-11-02.md): v0.31.0 R-01(read-only)/R-03, v0.32.0 R-02 metadata-only task receipts, v0.33.0 R-04 **internal lease core only**. S/T/source release PASS for these individual slices; overall Phase 2 **PARTIAL**, since P1-B2 Web/SSH/HTTP shared dispatch, R-04 live tool/lease enforcement, guarded patch, and installed cross-entry recovery are NOT IMPLEMENTED or LIVE U. Stable PUBLISHED_RELEASE remains v0.28.0.

## Phase 3 E0/E1 supported capability and fingerprint checkpoint (v0.36.0 candidate)

[Source implementation and boundaries](../design/BOX_ENV_E0_E1_IMPLEMENTATION.md). Authenticated Provider get_container with exact pinned Target; conservative unsupported/not_verified snapshot/clone, stable bounded fingerprint, app SQLite v13 immutable operator-scoped captures and CSRF-protected Web capture/list/compare. Fixture CI pending final source SHA; installed Web and current daemon source version must be independently proved. E2/E3/E4 await verified Provider snapshot/restore/clone/rekey and custom-volume contracts. Cross-entry R-04 protection remains NOT IMPLEMENTED; the phase is PARTIAL, not complete.

## Phase 2 tracked-exec shared claims (v0.37.0 candidate, NOT global enforcement)

[Shared-claim implementation](../design/PHASE2_SHARED_DISPATCH_INTEGRATION.md) integrates R-02 tracked SSH/HTTP exec with the private minimal operation coordinator, using the root-published Provider scope and exact Box incarnation. One concurrent same-Box tracked request is refused, other Boxes may proceed, and transport UNKNOWN keeps the claim. This does **not** cover direct legacy `connect(exec)`, other raw Provider mutators, or Web BoxActions/lifecycle/access workers. No globally safe Agent lease API is exposed. Stable installer stays v0.28.0; installed transport and cross-version tests remain L/U.

## Phase 2 Web BoxAction shared coordination (source v0.38.0 candidate)

[Web worker integration and failure boundaries](../design/PHASE2_APP_SHARED_ACTIONS.md). App configuration `coordination.enabled=true` opt-in protects Web BoxAction writes through the same private SQLite Box+name resources as tracked SSH/HTTP Gateway exec. Original app receipts remain authoritative; UNKNOWN, acknowledged running operations and missing/mismatched coordinator fail conservatively. This does not automatically protect Web lifecycle/access, old direct connect, or nested Provider mutation tools. Source Linux fixture CI and installed Web/SSH/HTTP parity are separate gates; global R-04 acceptance remains partial.

## Phase 2 opt-in protected SSH/HTTP gateway source (v0.39.0 candidate)

[Root-owned gateway policy design](../design/PHASE2_PROTECTED_GATEWAY_POLICY.md) blocks direct and unclassified Provider writes in protected mode, which requires a separately approved root-owned policy marker. The default remains legacy; no marker or public enable command is installed. Web lifecycle/access, external SSH bypass and old helper rollout are NOT protected. Source/fixture CI and installed transport tests remain distinct. Overall R-04 remains PARTIAL.

## Phase 2 opt-in Web lifecycle shared claims (v0.40.0 source candidate)

[Lifecycle shared-claim contract](../design/PHASE2_LIFECYCLE_SHARED_CLAIMS.md): staged source integration of create/start/stop/delete/resize with the original Web/app lifecycle receipt and the existing optional common Box/name/tenant claim coordinator. The original application receipt must first record observed terminal success before a claim can be retired. UNKNOWN and uncertain resize/nesting outcomes retain claims. This does not activate the root-owned SSH/HTTP protected Gateway, grant new authority, or protect app Access/port/network actions or direct external SSH. Source fixtures on [PR #47](https://github.com/vanishs/boxweave/pull/47) passed [CI 41493238649](https://github.com/vanishs/boxweave/actions/runs/41493238649) at its pre-version-bump head; reverify the final commit before merge. Installed/real-Box evidence remains U, overall R-04 PARTIAL; stable installer remains v0.28.0.


## Phase 2 v0.41.0 protected source slice — acceptance boundary

[Source and negative gates](../design/PHASE2_CROSS_ENTRY_COMPLETION.md).
Optional Web Access tenant/port claims, protected-only Box leases and tracked
small-file compare-and-swap patch are added. Exact source/fixture results belong
to CI and [Phase 2 acceptance](../verification/PHASE2_SOURCE_ACCEPTANCE_2026-11-02.md);
installed Gateway, Web, old versions and independently authenticated dual-client
results remain U. Default stable installer v0.28.0 remains unchanged.


## Phase 3 v0.42.0 bounded source and Box acceptance

[Supported Provider environment evidence](../design/PHASE3_SUPPORTED_ENV_ACCEPTANCE.md)
records authenticated real Box read inputs, exact-environment E0/E1
fingerprints, corrupt capture refusal and old v0.28.0 -> new v13 fixture
migration. New code strengthens failure behavior, no guest or host mutator.
E2/E3/E4 remain BLOCKED by unexposed Provider contracts; installed
service upgrade, client and rollback remain LIVE U. Stable installer
unchanged.


## ARCH-0 follow-up source recheck (v0.42.1 candidate)

[Post-integration dependency and release helper audit](../verification/ARCH0_POST_INTEGRATION_AUDIT.md).
A remaining CLI/deployment-check import cycle is removed behind
compatibility aliases. Gateway source/installer helper manifest parity
is guarded by executable tests. Existing ARCH-0A/B/C are historically
source-complete. Real installed host cross-UID/SSH/HTTP/Web, version
upgrade and rollback gates are still L=U. Default stable installer
v0.28.0 is unchanged.


## Stable installer promotion checkpoint — v0.42.2 source candidate

[Real two-Box baseline, old-to-new upgrade and acceptance limits](../verification/STABLE_PROMOTION_2026-11-02.md).
Default streamed release selected by `PUBLISHED_RELEASE` and `install.sh`
is promoted from historical v0.28.0 to source-bound **v0.42.1** after
successful disposable Ubuntu fresh-install and installed v0.28.0 to
v0.42.1 upgrade, authenticated Provider list read, child create/Python/
stop/delete and negative confirmation. Source patch uses version
v0.42.2 and must pass final CI and immutable release. No production
host was updated. Global R-04 guarded mode, external OAuth/TLS and
unsupported E2–E4 remain separate U/BLOCKED claims.


## 2026-11-02 P-01 candidate-only real acceptance, upstream release still gated

[Two-client wake plus cold-create live report](../verification/CONTAINARIUM_0126_AUTOSLEEP_WAKE_LIVE_2026-11-02.md) demonstrates patched Containarium v0.126.0 **T+L PASS** in an isolated disposable provider: new Box clock was stamped and later automatically stopped, Caddy routed HTTP to wake proxy, and first request returned HTTP 200 + original app body as target restarted. Security negative returned 403. Go focused tests and build exit 0; BoxWeave v0.42.8 Gateway doctor 10/10. The six-file [fork commit](https://github.com/vanishs/Containarium/commit/40b9d650584ceef66ea196ec6ee6dd9d88263f2f) is **not upstream merged or released**: PR creation was forbidden (403). Official Provider upgrade/rollback and external HTTPS remain blocked/unaccepted; BoxWeave version and default stable pin unchanged. All task resources removed, existing nine parent Boxes preserved.


## 2026-11-03 pinned Containarium-shell dependency source checkpoint

UP-SSH-01 is source implemented in v0.42.10 candidate, test 661/661 and bounded Box root sandbox PASS, but full installed acceptance remains U. [Evidence](../verification/CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md), [owner decisions](../design/CONTAINARIUM_UPSTREAM_MCP_FOLLOWUPS_2026-11-03.md). Upstream MCP contract/backup P1/P2/P2 remain follow-ups. No Containarium binary upgrade or stable promotion.

## 2026-11-03 v0.42.10 published and isolated installed acceptance

**UP-SSH-01 RELEASED / LIMITED INSTALLED PASS**: [official v0.42.10 Release](https://github.com/vanishs/boxweave/releases/tag/v0.42.10), merged `6ac4cb6986fe76a810321538e4af9fc3a1bfb602`, [main CI passed](https://github.com/vanishs/boxweave/actions/runs/41680412941). Published v0.42.9 fresh nested install and published v0.42.10 upgrade on disposable `bwshell-1103-verify-container` both ended `ready`, `sudo boxweave doctor` exit 0, 8 required checks passed; prior managed root shell hash retained; installed local Containarium stays at **0.120.1**. [Exact acceptance record](../verification/CONTAINARIUM_SHELL_PINNED_FETCH_04210_2026-11-03.md). Actual host-hop login/wake, cross-tenant denial and future Containarium upgrade are **not** accepted by these results. Stable default installer pin still **0.42.1**, not promoted. Other UP-MCP P1/P2/P2 requests remain upstream-facing.


## 2026-11-03 UP-SSH-01 live SSH forwarding checkpoint

**L PASS bounded:** In newly created and stopped test Box `bwhop-1103-e2e-container`, published BoxWeave v0.42.10 + local Containarium v0.120.1 successfully forwarded a real SSH command through `containarium-shell` into `hopprobe-container`; STOPPED SSH wake and denied unapproved key also passed. [Exact evidence](../verification/CONTAINARIUM_SHELL_REAL_SSH_HOP_2026-11-03.md). **U:** external sshpiper ingress, cross-tenant privilege denial. **Security review OPEN:** Containarium-generated per-account sudoers allowed unrestricted root `/usr/bin/incus`. No default stable release pin change or upstream Provider upgrade.

## 2026-11-03 P1 compat diagnosis / P2 local age-key guard implementation

**Source implemented; release/live gates pending:** [minimal design](../design/MCP_COMPAT_AND_LOCAL_RESTORE_GUARD_04211.md) · [execution evidence](../verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md). Reuses the D3 MCP gateway diagnosis, bounded tool-list fingerprint and root-owned exact local restore approvals; no polling/KMS/new Containarium integration. Upstream stable contract and whole-box backup create MCP remain OPEN. Default stable selector 0.42.1 unchanged.

## 2026-11-03 v0.42.11 publication and installed acceptance

PR #55 merged to 4f5ffe7f; main CI/publish SUCCESS; Release v0.42.11 published. Separate fresh isolated Box upgraded from v0.42.10 to official v0.42.11, phase=ready, doctor 8/8 required PASS, Containarium still v0.120.1. Real service-account stdio MCP source=compat matched all five monitored input schemas and unapproved restore was denied. **LIMIT:** this pinned Provider's real tools/list did **not** advertise restore_box_backup; approved actual encrypted whole-Box restoration remains OPEN. Stable selector stays v0.42.1. See [live evidence](../verification/MCP_COMPAT_RESTORE_04211_2026-11-03.md).

## DC-01 Agent-safe diagnostic coordinator — installed release v0.43.0

The owner authorized design, development, tests, live isolated acceptance and release. [PR #59](https://github.com/vanishs/boxweave/pull/59) merged read-only `boxweave_diagnostic_run` profiles (`preflight`, `rest_read`, `operation_trace`) without a host shell or mutation grant. [Exact independent CI, immutable release, v0.42.12-to-v0.43.0 real installed upgrade, Containarium REST/MCP and Trace PASS](../verification/STABLE_PROMOTION_DC01_0430_2026-11-03.md). Stable default pin is a distinct source-only selection and does not upgrade the owner's original host or authorize remote OAuth/public network changes. Protected-mode live integration and cross-layer write-timeout fault injection remain unverified.
