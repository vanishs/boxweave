# Current phase status

Authority for authorization/gates only; product version is [VERSION](../../VERSION), risk state is [OPEN_RISKS](../verification/OPEN_RISKS.md).

Phase7 is authorized by the user's supplied instructions (SRC-089). Entry audit accepts the phase6 closure of reviewed P0/P1 items; P2 deployment/evidence bounds are unchanged. Phase7 maintenance is complete within its tested scope (SRC-091); final result and actual command/evidence details belong to [PHASE_07_REVIEW](../verification/PHASE_07_REVIEW.md).

No deployment, push, inference, destructive cleanup or host/provider change is authorized by source/document cleanup. Source completion never implies the retained test deployment was upgraded. Future work requires its own request and the [MAINTENANCE](MAINTENANCE.md) path.

Historical gate/checkpoint text is preserved in [PHASE_STATUS_HISTORY](PHASE_STATUS_HISTORY.md); it must not override this file.

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
