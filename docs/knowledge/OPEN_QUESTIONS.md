# Open questions and risks

Phase6 current authority0.9.1/SRC-084..088: audit found and fixed three gaps in the previous completion claim: project reselect/edit/unbind, empty-target false-ready diagnosis and missing Web public client setup. User explicitly chose completing Web (SRC-086); no product decision remains pending. [OPEN_RISKS](../verification/OPEN_RISKS.md) is authoritative for closed P1 fixes and bounded open P2 items. Original test installation remains0.9.0 by design; upgrade is an explicit deployment action, not implied by the source patch. New0.9.1 independent-copy migration, actual generated Windows enrollment and all163 tests passed. Source-level phase7 entry eligible; not authorization to run it or certify untested deployment/security/model properties.

Current disposition0.9.0/SRC-079..083 supersedes the earlier implementation blockers below: selected TASK-012/013 and all18 AC dispositions are complete/evidenced in [the release report](../verification/TASK_012_013_RELEASE.md). No remaining selected phase5 implementation blocker. Retain operational limits: single trusted operator and broad upstream credential authority; unknown writes never replay/adopt; convergence delays are not immediate connectivity proof; pre-upgrade consistent backup is required for downgrade; no whole-host rollback. Fresh public TLS rollout, other server runtimes/clean bare metal, reboot/LXC boot.autostart, effective confinement and inference are unclaimed extensions, not fabricated passes. Phase6 comparison still requires authorization. Historical questions below remain traceability, not a declaration that already-evidenced features are missing.

No item below is silently resolved by the phase 1 document commit. Sources: [register](SOURCE_REGISTER.md). Priority P0 = product-scope decision before design; P1 = acceptance/safety concern before relevant implementation/release; P2 = later operational/maintenance evidence.

| ID | Priority/status | Unknown or risk and impact | Source | How to resolve / owner and gate |
| --- | --- | --- | --- | --- |
| RISK-001 | resolved scope / SRC-026 | Primary v2 user/outcome: personal agent entry, console, multi-user orchestration, or recursive infrastructure? Scope changes requirements materially | SRC-001/014/016 | Resolved by SRC-026; personal remote work with binding; see current SCOPE |
| RISK-002 | resolved scope / SRC-026 | Which legacy behaviors to keep/change/remove, and whether Web remains mandatory | F-001..015 | Resolved by SRC-026; basic requested features included, recursive deployment deferred; see SCOPE |
| RISK-003 | P1 / open | Asset download trust has contradictory legacy prose; pinning commits is not independent asset verification | SRC-002/003/004 | Trace generator/installer trust chain; choose and approve security acceptance in product/design phases |
| RISK-004 | P1 / inferred, not reproduced | Dirty client replacement answer is discarded; users may believe replacement selected when generation ignores it | SRC-020 | Separately authorized old-project fix or future fixture comparing interactive choice to rendered invocation; do not alter WIP in this phase |
| RISK-005 | P1 / unverified | Fresh Windows native/WSL/macOS enrollment, real agent GUI recognition and instruction loading lack universal acceptance | SRC-004/011/014 | Explicit per-runtime/client test matrix in later phases; no fixture-only compatibility claim |
| RISK-006 | P1 / unverified | Current installed provider capability/version/credential scope and actual Tenant TCP isolation | SRC-007/010/018/019/021 | Separately reviewed target/identity/readonly capability probe, then explicitly scoped policy/packet tests; not prerequisite for this static phase |
| RISK-007 | P1 / open | Broad admin provider credentials and local console authorization may not suit shared use | SRC-008/010 | Developer defines trust model; real positive/negative authorization tests before shared deployment |
| RISK-008 | P1 / unverified | Effective nested device filtering, parent policy and mount propagation are not certified | SRC-011/016/017 | If nesting/confinement is in scope, design safe evidence gathering; no security relaxation or dangerous-device probes |
| RISK-009 | P1 / open | Update publication is per file and mixed failure can require manual recovery; backup/software compatibility matters | SRC-002/006 | Decide recovery objectives; isolated failure/recovery validation before release |
| RISK-010 | P2 / unverified | Current full suite, CI automation, clean bare-metal install and boot.autostart | SRC-011/013/014/015 | Later assigned-LXC non-root tests, separate clean-platform/reboot checks; no CI assumed |
| RISK-011 | P2 / open | Ignored research is mutable, incomplete and not retained by new Git history | SRC-016..020 | Preserve originals outside commits; use recorded hashes and curated summaries; review additional narrative only if needed |
| RISK-012 | P1 / open | Job/protocol timeout does not prove remote cancellation or no delayed side effects | SRC-009/014 | Specify cancellation/unknown receipts and resource reconciliation acceptance; bounded fixtures and approved live tests |
| RISK-013 | P2 / open | Additional provider interface and standalone Linux client are proposals, not proven old behavior | SRC-001/003 | Phase 2 decide actual need; no speculative abstraction inherited |

## Historical phase 2 draft disposition, superseded by SRC-026

No RISK is closed by drafting a proposal. RISK-001/002 now have concrete alternatives DEC-001/003/004 in [scope](../product/SCOPE.md). RISK-003 maps to proposed REQ-016/AC-016; RISK-004 to REQ-001/AC-001; RISK-005 to DEC-002 and REQ-017/AC-017; RISK-006/007 to DEC-004/005 and REQ-012/018; RISK-008 to deferred recursive/confinement claims under DEC-004; RISK-009 to REQ-008/AC-008; RISK-010 to REQ-009/017; RISK-011 remains an evidence-preservation issue; RISK-012 to REQ-005/006/007; RISK-013 remains proposed out of initial scope. These references add decision/acceptance paths, not implementation evidence.

The actual client launch matrix, personal-versus-team audience, console timing, advanced features and demand for project binding remain unknown. DEC-001..005 are the authority for the five developer choices; do not duplicate their changing answers here. No market/user-demand claim is invented from source code. Product requirements remain proposed until the developer accepts/revises the scope and key behaviors.

Knowledge gaps do not prevent document handoff. They prevent unsupported runtime/support claims and automatic phase advancement. Any later live experiment must have target resources, changes, deadline, success criteria and cleanup agreed first.


## Current phase 3 risk disposition

Developer scope confirmation SRC-026 closes only RISK-001/002 as product choices. Basic Web/Tenant/ports/Graph/model wizard and project binding are included; native Windows/WSL SSH with Codex and ZCode(GLM), single trusted operator. Earlier deferral recommendations are historical, not current scope.

RISK-003: ADR-003 selects trusted descriptor + payload digests; bootstrap/tamper verification still untested (TASK-003/012). RISK-004 remains a legacy-only inferred defect; new adapter must preserve replacement intent. RISK-005 now gates the four selected Windows-hosted Agent profiles; macOS/WSL-hosted Agent CLIs are deferred. RISK-006 remains high for included Tenant writes: TASK-009 must prove actual authenticated capability/atomic behavior; no provider upgrade workaround. RISK-007: shared-team product excluded, but broad upstream privilege still disclosed and tested within trusted-operator boundary. RISK-008 remains unverified; recursive deployment is deferred and no confinement claim is made. RISK-009 maps to staged packaging/recovery TASK-012. RISK-010 maps to TASK-013; no bare-metal/autostart claim without evidence. RISK-011 remains mutable historical evidence. RISK-012 maps to ADR-002 crash/unknown/deadline behavior. RISK-013 is deferred extension scope.

| ID | Priority/status | New design risk | Resolution path |
| --- | --- | --- | --- |
| RISK-014 | P1 / unverified | ZCode GLM model wizard lacks a proven installed-version automation contract; old GLM integration was Claude-specific | SRC-027; ADR-004 and TASK-008: client-owned native settings/sign-in handoff, only automate verified subset; MCP success never means model ready |
| RISK-015 | P1 / unverified | WSL SSH distro/argv/process lifetime can differ from native and false cleanup claims can hide remote work | ADR-003; EXP-002, TASK-007, actual four-profile evidence |
| RISK-016 | P1 / unverified | Single SQLite/worker intent lease cannot make a remote write exactly-once; crash before/after send remains ambiguous | ADR-002; TASK-002/006 fault injection and no replay; provider policy atomicity must be native |

All experiments in TEST_STRATEGY are plans, not executed results or automatic resource authorization. Architecture is complete only as design; release and phase-4 feasibility remain unverified.

## Phase 4 current result — partial, preservation requested

RISK-016 has component evidence for real SQLite intent/receipt persistence, state-boundary loss and no replay; real provider exactly-once/cancellation remains unverified. RISK-005 now includes actual Windows PowerShell preflight only; real native SSH/registration/UI acceptance and WSL remain unverified. RISK-003 has dependency/source hashes but no completed setup-payload distribution; no trust claim is closed. Other release risks remain open; no Tenant/port/model/production installation work ran.

| ID | Priority/status | New concrete blocker | Next safe action |
| --- | --- | --- | --- |
| RISK-017 | P0 for phase 4 / unavailable | No reviewed independent application MCP launcher/credential path in the assigned LXC; managed launcher/backend paths absent and user confirms no path available | Preserve current results (SRC-032); later establish an independent protected runtime identity through separately approved provisioning. Do not request secret text, borrow Agent tokens or modify host security. Then run real application/native-client acceptance |

Fixture browser success and direct Platform MCP hello are independent and cannot be combined into a live Web->worker->MCP pass. Phase 4 gate is unmet; phase 5 must not start. Runtime code and examples remain a reviewable minimum implementation with explicit fixture labeling, not a production deployment.

| ID | Priority/status | Live lab provisioning issue | Next safe action |
| --- | --- | --- | --- |
| RISK-018 | Deferred by developer / observed prerequisite rejection | Capability advertises old boxweave.tenant-network/v1 and true shared-create flag, but create rejects native Tenant readiness. No lab resource exists in subsequent observations | SRC-039/040; attest actual MCP/CLI/daemon pair read-only, then separately approve compatible host repair. No gate bypass/duplicate/old-box reassignment; see SHARED_TENANT_LAB |


SRC-041 defers the shared-Tenant/live MCP test and further repair attempts for now. RISK-017/018 remain unresolved; no test passes or version compatibility claims are inferred. This deferral permits a readiness assessment only, not phase 5 execution or an automatic whole-stage acceptance.

## Latest same-Tenant resource result (SRC-045)

New bw-v2-mcp-test-container (logical bw-v2-mcp-test, createdAt 1793202811, 10.206.30.7, dev, 2 CPU / 4 GB / 25 GB) was created once in networkTenant boxweave-v2-test, alongside the existing v2 application. Exact metadata, identity exit 0, same attached Guard ACL and app-to-new-box TCP22 connectivity verified. Earlier shared-create contract rejection no longer reproduces in this fresh native-network-tenant/v1 observation; no host repair was performed by this task.

Supporting backend bootstrap reached its 600-second total deadline, install.exit=124, while mandatory Web dependencies were still preparing. APT/dpkg children remained; do not rerun until original state is reconciled. Managed MCP launcher absent; live MCP and prepared loopback HTTP integration harness not executed. Existing demo services active, homepage 200. RISK-017 remains open; RISK-018 has current successful shared-create evidence but no full compatibility certification. New box and dedicated lab artifacts retained, no new public route, no phase 5 implementation. Details in SHARED_TENANT_LAB.

RISK-019 — Live nested execution blocked: supporting backend installed and authenticated read-only MCP passed, but inner core PostgreSQL fails to start (BPF load denied; fatal proc mount Operation not permitted). Policy store unavailable; one MCP workload create rejected, name reconciled absent. Full parent policy not attested. Requires a supported nested runtime/host-management resolution within separately reviewed scope; no core/host/security bypass performed. RISK-017 launcher absence is resolved for this dedicated lab only; live application receipt remains open.

## Current live-chain evidence (SRC-049/050)

RISK-017: independent launcher/authentication/execution now established in the explicitly approved new lab; original project/client registration and full release boundary still incomplete. RISK-018: current native shared creation and matching Guard attachment succeeded; historical old-contract failures remain observations, not a current universal failure. RISK-019: mitigated for this lab by a separately approved new Podman/nesting-enabled box; old non-nesting box is unchanged. Do not claim full effective host confinement from this successful test.

RISK-020 — Observed provider flag mismatch: inner MCP create supplied enable_podman=false, but metadata readback reports podmanEnabled=true. Exact cause not attested. Preserve requested versus observed values, constrain test to approved harmless commands, and resolve before promising exact workload isolation/feature flag behavior. No relabeling, provider upgrade or silent flag repair occurred.

The real HTTP run also exposed a worker time-sampling race (false deadline/clock rollback for newly queued work). Fixed in 0.1.1 with a failing-before/passing-after regression; actual HTTP and browser receipt paths now passed. Missing real Windows-native enrollment/registration/UI and developer architecture acceptance remain explicit. See LIVE_MCP_CHAIN; no phase5 feature has started.

## Phase 5 iteration 1 complete — TASK-011 (SRC-051/052)

Developer acceptance of the minimum architecture/slice is explicitly recorded in SRC-051, after verified real Web/MCP evidence. TASK-011 basic Prompt Graph is implemented in 0.2.0: bounded cards, editable comments/context, undirected links, revision-checked persistence, conflict-preserved drafts, Markdown/clipboard handling and graph-only deletion. REQ-011/014/016 and AC-011/014/016 graph subset are evidenced; other selected-feature/client obligations remain incomplete.

Actual non-root project-LXC checks: 41 tests OK (11.173s), TypeScript/Vite build, final Chromium graph workflow including mobile fit and clipboard failure, original project hello/restart fixture regression; all exit0. Graph API forbids provider access in tests; browser has zero queued/executed operations. Thirteen changed source/lock files match remote SHA256. See PHASE_05_TASK_011.md for commands and precise evidence. No new box, infrastructure mutation, deployment, persistent listener, public route or push. Stop this iteration.

Next suggestion: finish TASK-003/005 minimum actual Windows-native enrollment/recognition evidence, then dependent TASK-006. Developer gate acceptance does not fabricate those missing client checks or waive per-task prerequisites. Phase 5 overall is not complete.

## Iteration 2 complete — TASK-003a, version 0.2.1 (SRC-053/054)

Scoped existing-identity Windows-native Codex registration now has trusted pre-execution payload verification, strict live connection gating, targeted config preservation, explicit replacement, byte-idempotence and backups. Actual isolated Windows/CLI/native MCP and failure-path tests passed; 41 project Linux tests and the three prior native preflight checks passed. No global/personal registration, new key, model request, mapping, box or deployment changed.

TASK-003a is complete in its stated existing-identity scope; parent TASK-003/005 remain partial. Actual GUI/TUI loading, bootstrap identity provisioning and other client/runtime profiles are unverified. Named-mutex/hash/backup recovery does not replace quiescing non-cooperating editors. See TASK_003A_NATIVE_REGISTRATION for commands, limitations and cooperative next steps. Stop this iteration; suggest TASK-003b, do not waive dependencies for TASK-006.

## Current iteration 3 — TASK-003b1, version 0.2.2 (SRC-055..058)

TASK-003b is split into TASK-003b1 (explicit Windows-native executable selection plus recorded cooperative TUI/read-only acceptance, complete) and TASK-003b2 (product bootstrap identity provisioning, incomplete). Parent TASK-003 and full TASK-005 remain partial; do not waive TASK-006 dependencies. Developer architecture gate SRC-051 remains accepted.

The native bootstrap/registration accepts an operator-selected absolute -CodexExecutable and retains exact-version/payload/trust checks. Real Windows run-06 passes with Codex absent from PATH, plus missing/relative runtime rejection and existing preservation/integrity regressions; personal config digest unchanged. Project-LXC 41 tests pass (11.586s); native AST/3 rejection checks pass. SRC-057 records the user's actual TUI connected(2 tools) and list_containers result v2-live-hello-container against the dedicated backend. This is separate from desktop GUI, instruction/model readiness, WSL/ZCode and release evidence. See [TASK-003b1 evidence](../verification/TASK_003B1_NATIVE_CLIENT.md).

The dedicated test key/18447 route were separately approved in SRC-055/056, and retained; no new route/resource/policy/model operation is part of this implementation iteration. Existing public Web demo remains fixture. No automatic key provisioning or whole-phase completion claim. Next task is TASK-003b2; stop this one slice after its code/test/docs commit, no push.

RISK-005 narrowed: native Codex TUI loading and one real read-only tool call now have user-observed evidence SRC-057. Desktop GUI/instruction loading and the other three selected transport/agent profiles remain unverified. Manual dedicated lab key authorization is not reproducible product bootstrap. RISK-020 remains unresolved. The test-only source mapping is not LAN-only filtering; no public credential, key contents or login data belong in evidence.

## Gate interpretation correction (SRC-059/060)

The phase5 startup checklist is satisfied; see [audit](../verification/PHASE_05_READINESS.md). Earlier prose conflated planned automatic identity provisioning and full release/profile coverage with entry permission. Existing-authorized-identity enrollment is permitted by REQ-001/AC-001; the correction adds no requirement and removes no feature obligation. Keep RISK-005/020 and incomplete product tasks open within phase5. No fresh user acceptance or infrastructure setup is needed solely to re-open the already accepted gate.

TASK-002a SRC-062 narrows receipt-discovery coverage: old stored receipts now queryable through paged Web/CLI summaries beyond100, actor isolation and zero provider/state effects tested; actual restart and unknown-detail UI pass. This does not resolve provider-side reconciliation, DB-full/power-loss recovery, arbitrary-command output redaction or parent TASK-002 wholesale. History is a seek view, not an immutable snapshot. RISK-020 and remaining profile/infrastructure/release obligations remain open. See ../verification/TASK_002A_RECEIPT_HISTORY.md.

TASK-002 completion (SRC-063/064), version0.3.0: actual SQLite FULL/claim/result-audit failure and real process death before send/after effect, concurrent claims and persisted deadlines tested. Command exit7 is failed with actual exit retained and unexpected output withheld. Explicit actor-bound target reconciliation records matching/changed/unavailable without resolving unknown or retrying. Schema3 additive observations preserve old data; older versions refuse3, use a consistent private backup for downgrade. Full58 tests19.511s, frontend build and history/original/Graph Chromium passed. Same-Tenant app-hosted real MCP reconcile matched v2-live-hello with0 hello calls; synthetic unknown receipt stayed unchanged. See [completion evidence](../verification/TASK_002_COMPLETION.md). TASK-006 create/lifecycle/reconciliation remains separate; arbitrary command execution is not added.

TASK-003 completion0.4.0 (SRC-065): selected native Codex path now includes fresh local private-ACL key preparation, exact non-root public-key-only restricted launcher bootstrap and independent verified-probe/registration diagnosis. Real fresh-v3 enrollment/CLI/MCP and Windows positive/negative/preservation tests pass; full64 Linux tests18.550s pass, including6 authorizer cases. Existing native TUI evidence SRC-057 remains separate; machine overall stays incomplete when GUI unverified. No general host shell, private/sign-in export, new route, model invocation or silent repair. Other profiles/packaging/release remain TASK-007/012/013. See [completion report](../verification/TASK_003_COMPLETION.md).


TASK-006 / SRC-069 / 0.5.0: scoped lifecycle plans, single dispatch, pinned managed environments, schema4 recovery, Web/CLI receipts and config-scope guards implemented. 82 non-root Linux tests and Chromium fixture passed; first actual five-action Web/MCP cycle passed. Final repeat correctly stops TRUST_FAILED after reused-IP mismatch; trusted reconciler reports failure and was not repaired or bypassed. Approved exact test resource cleanup read back absent; original hello preserved. See docs/verification/TASK_006_LIFECYCLE.md for separate success/failure evidence and limits. Continue TASK-007 under SRC-066; no full release claim.


TASK-007 / SRC-070..072 / 0.6.0: selected four-profile connection/registration/UI scope complete; see [profile evidence](../verification/TASK_007_PROFILES.md) for actual Windows matrix,95 Linux tests/build, separate human ZCode/WSL confirmation and limits. Python3.11+ setup runtime is explicit; existing PowerShell/native path retained. Model and release acceptance remain pending. Legacy main d8a09dd unchanged, no automatic implementation migration or push. Continuous remaining phase5 work is authorized; no implicit public/destructive/inference action.


TASK-008 / SRC-073..075 / 0.7.0: native-guided GLM wizard complete in selected scope; schema5 revisioned public intent, shared CLI/Web, native credentials only and operator report distinct from inference.106 tests/build/Chromium pass; user confirms Z.ai/GLM-5.3/access. See TASK_008_MODEL_WIZARD.md for independent native and fixture evidence. No model request, provider repair or public deployment. Continue SRC-072 network/ports and dependent packaging/release.
TASK-009/010 / SRC-076..078 / 0.8.0: selected basic Tenant/port scope implemented and evidenced.132 Linux tests/build/Chromium fixture passed; fresh approved0.120.1 nested backend supplied actual shared membership, baseline TCP isolation, atomic bidirectional grants/revokes, stale-CAS rejection without partial write and TCP/UDP publish/unpublish packet observations. Preserve separate failed TRUST_FAILED creation and immediate convergence probes; corrected only the fresh installation's erroneous trusted CIDR through its installer. No replay/adoption or manual trust bypass. Final actual hello exit0, worker restart no replay, all three dedicated workloads/routes absent; outer lab and two empty policies retained. See docs/verification/TASK_009_010_ACCESS.md for exact attempts, commands, source hashes and limits. TASK-012 packaging/install/update/repair and TASK-013 release acceptance remain incomplete; phase6/model inference not authorized.
