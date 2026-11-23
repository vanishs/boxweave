# RC13 PRS availability and one-original helper lifecycle — remediation contract

**Status: proposal / development specification, not implemented.** Read [source RCA](V1_RC13_PEER_ROOT_RCA_2026-11-23.md) before editing. Owner asked for detailed repair documents and an independent Draft PR; this is **not** permission to change the installed L0 Host or relax native policy.

## 0. Updated owner decision: THIS OVERRIDES EARLIER CONSERVATIVE RC13 RECOMMENDATIONS

**Read the [new authoritative acceptance/dependency contract](V1_PEER_MCP_AUTOSTART_NONDISRUPTIVE_OWNER_DECISION_2026-11-23.md) before planning or coding.** Previous notes below about keeping `boot.autostart=false`, preserving a full-bridge STOP during successful normal operation, or manual Host intervention per Peer original are **historical safety requirements of the immutable RC13 implementation, not acceptable final product behavior**. Owner now requires:

- New Box defaults to `boot.autostart=true` and successful Peer changes preserve actual user-intended autostart state.
- Routine authorized Peer grant/revoke and safe supported recoveries are fully MCP-driven after trusted initial installation. No per-operation manual Host commands; bounded least-privilege mediator, no generic root service.
- No stopping/restarting or dropping running network services on unrelated Boxes for normal A→B changes; fail closed if non-disruptive native policy cannot yet be independently proven.
- New unified Doctor/Recover development in PR #177 **waits until #178 has produced and integrated verified runtime semantics**, so the diagnostic contract is not generated from obsolete RC13 behavior. Existing #177 uninstall recovery stays intact.

**Delivery priority changes:** first architectural audit + safety experiments → default autostart + preservation → MCP bounded automation → non-disruptive native policy; all must be demonstrated before #178 claims full completion. An impact warning is useful but **not a substitute** for no unrelated-Box downtime.

## 1. Engineering objective and explicit non-goals

Make Peer root operations **understandable, auditable and safely repeatable across separately approved originals** without compromising the current complete-bridge stop/ACL/nft/conntrack safety proof. The current run completed the intended authorization and chained Forward. Do not call working Forward a defect or sacrifice fail-closed safety to avoid workload restart.

Only change runtime behavior after source-specific unit/native tests and an approved product decision. Diagnosis belongs to the [unified `doctor` / `recover` PR #177](https://github.com/vanishs/boxweave/pull/177). This branch should not add a third CLI, generic root RPC, new Host exposure or background “repair everything” agent.

## 2. Workstream P1 — pre-approval and post-operation blast-radius contract

### 2.1 Proposed additive read-only impact plan

Extend the existing **root-only** `native-peer-root-plan` evidence (or its existing typed internal plan structure, after reading the exact repo shape) with a **versioned impact section**:

- `original_operation_id`, `operation_kind=PEER_GRANT|PEER_REVOKE`, `project`, `bridge`, `acl`, immutable reviewed digest(s), `plan_generated_at`.
- `consumer_count`, owned `running_before`/`stopped_before` counts, `will_stop_every_bridge_consumer=true`, `will_disable_autostart=true`, `can_interrupt_guest_processes=true`; exact owned Box UUID/incarnation only in *authorized Host/root scope*, not public cross-Project diagnostics.
- `autostart_prestate` as **unset vs explicit false vs true** per consumer, with ETag/source, audit fingerprint rather than untrusted raw config; and truthful uncertainty if missing.
- `active_proxy_device_count` or `owned_forward_claim_count` (with coverage and source) for impact transparency. A confirmed Forward does **not** imply live workload.
- `root_helper_unit_present`, `root_helper_unit_active`, `helper_scope_original`, `helper_readiness`; independent read-only unit check and root-only journal status, not assumptions from a previous `ADMITTED` state.
- `expected_downtime=unbounded` (unless actually bounded by implementation), `requires_operator_approval=true`, `forward_live_after_restart=UNVERIFIED`, `autostart_after_policy=RETAIN_FALSE_BY_CURRENT_DESIGN`.

Revalidate EXACT underlying native/SQLite/root plan digest immediately before mutation. If inventory or original state drifts, **refuse**. No automatic native changes in a plan and no leaking root journal contents or unrelated Projects.

For the **MCP client** `box_peer_set` and subsequent `operation_get`, distinguish `ADMITTED_AWAITING_HOST_REVIEW`, `DISPATCHING`, `VERIFYING` and terminal states with a typed, immutable original ID and safe next step. Do not promise immediate execution or claim the MCP client can itself grant root. #177 will own generalized issue reporting; avoid incompatible duplicate schema here.

### 2.2 Operator UX / Activity contract

At admission: explicitly state full bridge affected, potential temporary stop of **unrelated** owned Boxes and volatile listeners. Before applying: show approved digest and exact impacts in root-only plan. On success: persist safe event summary (`ROOT_PEER_BARRIER_APPLIED`, `AFFECTED_CONSUMERS=n`, `AUTOSTART_RETAINED_FALSE`, `RUNNING_RESUMED=n`) with immutable original, sanitized data and typed coverage. On UNKNOWN: retain full claims, guarded status and Root journal; do not mark previous running state restored. Do not mistake root service `active` for original terminal success.

**Interim development scope only:** exact observability and regression tests are useful first commits, but **do not satisfy final #178 acceptance**. It must also deliver default autostart + preservation, routine MCP-driven authorized execution, and verified no-interruption changes for unrelated Boxes. Retain old fail-closed safeguards until a tested replacement exists.

## 3. Workstream P1 — safe one-original root unit lifecycle

### 3.1 Known source facts

`internal/install/native_peer_root_service.go` generates a specific `/etc/systemd/system/boxweave-native-peer-root.service` whose `ExecStart` is pinned to one Operation and plan SHA, requires root, and is installed using `O_CREATE|O_EXCL`. The owner explicitly activated the exact scoped unit; the original reached `SUCCEEDED`. The unit was **not removed** after the trial. **A subsequent grant/revoke has not yet been performed**, so investigate reuse behavior first instead of claiming a second-operation failure.

### 3.2 First investigation and acceptance

- Under an independent disposable test Host/Box, finish original A, inspect unit `LoadState/ActiveState/ExecStart`, root-private journal, file owner/mode, exact SHA and controller original state.
- Admit a **new** original B using ordinary authenticated MCP; perform safe Host-local read-only plan/stage investigation, document whether existing unit rejects staging, requires retiring, or can service a distinct original (per actual source).
- Prove wrong original / wrong SHA / foreign Project / drifted kernel / unaudited unit / missing journal / concurrent A and B are all refused without applying a second ACL change.
- Prove the correct terminal/UNKNOWN boundary: **never retire, replace, or overwrite a unit still serving an uncertain original**. A terminal controller state alone may not prove journal completeness; require exact Root journal and read-only native poststate.
- If proper turnover is missing, design one **Host-local exact-scope reviewed retire/re-stage** path. It should (a) require original immutable ID, terminal + verified Root journal and local operator consent, (b) stop and verify socket/daemon no requests in flight, (c) archive or retain reviewed unit/audit bytes under root-owned private retention, (d) remove only the exact stale unit via exclusive path/identity compare, (e) daemon reload, (f) allow a new **separately** approved plan. No general unit delete endpoint, no overwrite-on-conflict, no cleanup on UNKNOWN.
- Recovery routing should reuse `boxweave recover` approved issue planning from #177 where appropriate; don't introduce a parallel Host repair engine. Maintain installer/uninstaller #177 exact audit compatibility.

**Explicit gating:** This is a design/test requirement; final Host unit-retirement UX and approval wording need separate review after feasibility. Do not “fix” by leaving a reusable unconstrained root daemon.

## 4. Workstream P1 — autostart lifecycle, clarify before changing code

Current source **intentionally** sets `boot.autostart=false` for every shared ACL consumer, and resume verifier tests it remains false. Candidate policies:

1. **Conservative default (recommended first patch):** keep `false` and document that successful Peer changes may leave autostart disabled. Expose impact and need for separate reviewed Host/managed config action, including optional user-facing warning when an originally autostart-enabled Box is modified. No change to proven uncertain-failure barrier.
2. **Restore original after COMPLETE (future design, NOT authorized now):** would require durable prestate snapshot with unset/true/false, terminal journal and independent kernel/native proof, tightly scoped atomic/ETag config modifications, crash-proof journal and refusal on stale native state, plus tests for reboot races/failed second write and external consumers. Existing verifier explicitly requires false; source and CI safety proof must be reworked first.
3. **Stop only Peer endpoints (NOT approved):** shared ACL fan-out affects *all* NICs; require full alternative kernel security architecture with bridge-wide negative packet checks, conntrack ACK-loss proof and independently verified rollback before discussing narrower stopping. Do not implement as a quick optimization.

A Host reboot test should distinguish Box default creation `boot.autostart` (unset) from PRS-assigned explicit false and check running availability/Forward device vs Guest listener. **Do not auto-start unrequested Boxes** merely to make `doctor` green.

## 5. Workstream P2 — application/Forward availability after Root barrier

Existing `port_forward_list.native_verified=true` is **correct** (native proxy matches exact stored claim), not a promise of live HTTP. Work with #177 to expose layers of proof:

`CLAIM_CONFIRMED` → `NATIVE_DEVICE_VERIFIED` → `GUEST_LISTENER_CHECKED?` → `HOST_LOCAL_PACKET_CHECKED?` → `REMOTE_AUTHORIZED_CLIENT_PACKET_CHECKED?`.

A read-only Doctor may provide a **recommended** Guest-local check and distinguish separately authorized packet probes, but must not invent health from a persisted receipt, silently issue network requests on behalf of every service, or restart an arbitrary Guest process. For a test HTTP service, suggest explicitly managed systemd service when survive-Guest-restart is a requirement; don't add an in-Guest daemon to BoxWeave itself.

Client IP changes and a future external interface mean `0.0.0.0` remains an exposure decision local to the exact layer. Uphold existing `confirm_external_exposure` AND trusted Host policy; test an unapproved third Box remains denied even when proxy binds wildcard.

## 6. Detailed test plan and independent gates

| Case | Required observation | Failure safety |
| --- | --- | --- |
| T1 exact active root plan | Original A, all 3 consumers prestate, count and whole-bridge blast radius, reviewed digest | Plan-only byte-for-byte no mutation of DB, unit, ACL, guest or journal |
| T2 deliberate Peer grant A→B | Before denied, after allowed TCP/IPv4 (plus UDP/IPv6 if required for affected change); C→B always denied | No temporary fail-open during native ACL switch; test root nft/conntrack |
| T3 unrelated C with transient listener and confirmed proxy | Record listener alive before; after PRS temporary process may be dead; native proxy still correct | Report interruption and no false HTTP PASS; no silent restart |
| T4 persistent guest-managed listener | Guest systemd-managed test listener survives **through restart** if configured; explicitly verify | Do not assert a daemon is up based only on `Running` |
| T5 autostart prestate matrix | Unset, explicit true, explicit false; all triaged; current accepted behavior false after PRS | No unsanctioned autostart restoration; UNKNOWN fail closed |
| T6 original fail-after-ACL / before-resume | Real stopped barrier, verified non-live ports, original UNKNOWN and preserved root journal | No reissue ACL PUT, no autostart, no claim release |
| T7 root helper A terminal → new B | Unit already exists; read-only investigation; if approved design added, exact safe turnover tested | Wrong SHA, wrong source, active original, missing journal, different Project refuse |
| T8 restart/reboot coverage | Bounded service restart and approved reboot where supported, old Forward device readback + packet | Record `NOT_TESTED` if no authorized reboot, never infer persistence |
| T9 nested L0→L1→L2 topology | Same RC13 binary/Incus, L2 HTTP response, approved Peer, unapproved C denied | No Host global ACL, OAuth, tunnel or privilege workaround |
| T10 cancellation/lock & CLI provenance | Peer ADMITTED with helper absent, cancel refused; CLI policy != daemon effective | Correct #177 diagnosis, no MCP root escalation |
| T11 existing original/recovery regression | Previously accepted PRS revoke/UNKNOWN, #177 uninstall historical, native ETag/ACK-loss | Do not rerun physical Host teardown or touch historical audits |
| T12 review/audit hygiene | Exact original IDs, SHA and source; no secrets, no other Project inventory | No machine-ID hash, raw bearer token or root journal export |

Use isolated L1 Incus/BoxWeave and disposable siblings for code/test iteration; CI only after local scoped native probes. Reuse signed published RC13 case only as a **baseline**, not as proof that a new candidate passes. Add precise **before/after** tests using actual candidate HEAD, no source-only simulated “PASS” for real packet changes. Any failure must include original Operation, native ID, guest exit and retained artifact, with UNKNOWN never blindly retried.

## 7. Implementation sequence and stop conditions

A. Read exact `v1` HEAD and this RCA, inventory existing original PRS/Native/Forward/Activity tests and conflicts with #177. Reproduce T1–T3 on a disposable Box *without changing physical L0*.

B. Implement **additive impact reporting** in the existing Host root plan and safe original activity summary; preserve old JSON clients, exact digest review and existing guardrails. Add unit tests and native integration cases for full consumer count, missing evidence and deliberate drift.

C. Investigate one-original root unit turnover T7. If absent, propose the narrowest viable separate Host-local reviewed maintenance leaf and exact test matrix; do not implement irreversible or generalized Root cleanup without owner approval. Review original journal retention.

D. Leave autostart behavior unchanged by default. Document/run T5 in disposable native tests, then request an explicit owner decision only if policy restoration is actually necessary. The fact that previous app died is not evidence that entire kernel barrier should disappear.

E. Integrate with #177 via a link and issue taxonomy rather than duplicating Doctor/recover implementation. Run affected Go/race, native CI, exact negative/ACK-loss, and review PR Draft until separate signoff. Do not merge/release/modify installed L0 without user approval.

**Owner decision resolved (2026-11-23):** default `boot.autostart=true` for new Boxes; successful Peer operations must preserve prior user-intended autostart setting and must not interrupt unrelated Box workloads. The previous `RETAIN_FALSE_BY_CURRENT_DESIGN` labels in the RC13 source study are historical observations only, **not final acceptance criteria**. For implementation, use the new authoritative requirements and independently prove safe native transition and loss/ACK handling before changing the old fail-closed logic.
