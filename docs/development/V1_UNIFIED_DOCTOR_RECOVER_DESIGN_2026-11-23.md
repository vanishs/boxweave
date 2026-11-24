# BoxWeave v1 — Unified Doctor + Recover (PR #177 design and migration lessons)

**D4 candidate update (2026-11-23 PT):** [Reviewed recovery execution and exact Box evidence](V1_PR177_D4_REVIEWED_RECOVERY_2026-11-23.md). D2/D3 independent exact-head CI all four passed. D4 code, three focused Race rounds, full verify and fresh scoped installed positive recovery passed in Box. D4 independent CI is required before D5; legacy installed D4 and scoped partial apply refusal remain CI gates. Earlier staged-status notes below are historical, not instructions to redo accepted stages. Keep Draft; no v1 merge, release or physical L0 modification.

**D3 candidate update (2026-11-23):** [Read-only recovery plan and Box evidence](V1_PR177_D3_RECOVERY_PLAN_2026-11-23.md). D2 installed JSON assertion corrected; D2/D3 independent CI still required. D4 apply is not implemented.

> **D1 accepted, D2 diagnosis candidate:** [D2 implementation, Box evidence and CI gates](V1_PR177_D2_DIAGNOSIS_2026-11-23.md). D1 exact-head CI all four passed. D2 shared diagnosis, mode-qualified retirement/capacity, bounded activity/history, Native/Root/Forward/OAuth coverage and stable issue routes delivered; review D2 exact-head CI before D3. Routes are not repair authority.

> **D0 accepted, D1 foundation delivered for CI:** [D1 implementation and Box evidence](V1_PR177_D1_FOUNDATION_2026-11-23.md). Check exact D1 HEAD CI including new installed mode/authority markers before D2. Doctor now captures a bounded private locked SQLite epoch and never calls state.Open on source; this handle is not a Root witness. Do not repeat D0.

> **Mandatory D0 extended findings:** [D1-D4 investigation and additional Box evidence](V1_PR177_D0_EXTENDED_D1_D4_INVESTIGATION_2026-11-23.md). Reuse the source/authority and leaf matrix; fix proven retirement/operation-cause/navigation classification gaps in D2, separate Root proof from restricted recovery worker, bind fresh review evidence separately from idempotency, and do not certify a moving DB/WAL copy. D0 extension is characterization only; review exact-head CI before D1.

> **Current staged execution contract:** [D0-D5 plan](V1_PR177_D0_D5_DEVELOPMENT_PLAN_2026-11-23.md) and [D0 source/Box experiments](V1_PR177_D0_EXPERIMENTS_2026-11-23.md). The v1 merge is complete at e60de8f; all four baseline CI workflows passed. D0 characterizes readers only; new feature implementation starts with D1 after D0 CI. Every stage iterates and verifies inside a Box before submitting code and docs to CI. The cold SQLite WAL/SHM observation and immutable stale-read negative are mandatory D1 constraints.

**Status:** Owner-approved design direction, **not yet implemented**. Recorded 2026-11-23. Implement on Draft PR [#177](https://github.com/vanishs/boxweave/pull/177), base `v1`, while preserving its existing guided-uninstall recovery. **Do not merge, promote, change the installed L0 or edit global Incus without explicit owner instruction.**

## 0. Current owner update — #178 merged; source integration and diagnostic correctness gate

**The #178 dependency hold is lifted.** #178 merged into v1 at `ba29c74292747242b211771c30b4d9d8f6f83ed9` after all four exact-head CI workflows passed. Before feature code, merge current v1 into #177, resolve conflicts preserving both PRs and verify actual runtime source. Read the **mandatory [post-#178 diagnostic contract](V1_DOCTOR_RECOVER_POST_PR178_CONTRACT_2026-11-23.md)**; it supersedes RC13-only diagnosis and repair assumptions below. The new design is not yet implemented. Historical RC13 cases remain regression fixtures for proven legacy mode only; actual deployment mode must be proven, not inferred from version. Preserve guided-uninstall acceptance and physical Host evidence; do not deploy to L0 or merge #177.

## 1. Decision and product goal

The operator approved **one binary, two public CLI entry points**:

- `boxweave doctor`: a comprehensive, explicitly read-only diagnostic entry point, with human/JSON output, declared authority and coverage, actionable findings and durable evidence references. Proposed new `--full` and `--json` options below are **design**, not currently supported RC13 flags.
- `boxweave recover`: the single repair entry point, retaining existing guarded per-original recovery; add reviewed `--plan --issue <ISSUE_ID>` navigation to exact existing recovery mechanisms. Mutations only through independently authorized, narrowly scoped, evidence-checked actions. Not a generic `--fix-all`, shell, privilege escalator or system repair daemon.

Keep existing `status`, `native-*`, install/uninstall compatibility; reuse their verified logic internally rather than delete them or create a new third CLI. Prefer **MCP repair** for ordinary owned Box/Operation/Network issues while the controller is functional; Host CLI remains the repair boundary for actual service/installer/Root/native prerequisites. A broken/uninstalled CLI requires the separately pinned minimal installer/uninstaller rescue path; do not imply the running CLI can repair its own absence.

**Important distinction:** `doctor` may discover many things but does not guarantee that every issue is repairable. State `BLOCKED` or `EXTERNAL_OPERATOR_REQUIRED` when independent evidence or authorized permissions are missing.

## 2. Source audit, exact published RC13

Code reviewed at release tag `v1.0.0-rc.13`, source commit `6c995687ae38b5cd0e23c464dee84f40a33a01ef`. These observations must be rechecked on current PR HEAD before implementation.

| Source / current entry point | Existing facts | Gap for unified interface |
| --- | --- | --- |
| `cmd/boxweave/main.go` `status` / `doctor` | Both call the same `svc.Status(ctx)`, emitting JSON. Only `doctor` additionally exits nonzero when `!status.Ready`. | No complete inventory, per-issue cause, recovery plan, scope/coverage or service-effective policy check. |
| `internal/control/server.go` `Service.Status` | Reads Schema, native readiness, 1024 logical Project capacity, address retirement and best-effort IPv4 capacity. | Readiness is not proof of real packet/quota/PRS, systemd, installed binary, OAuth/Tunnel or all ledger/native issues. |
| `internal/native/status.go` `Reader.Read/Strict` | Checks dedicated non-root identity, restricted socket/Project, config, extensions, pool/network; emits typed `error_class` and warnings. | CLI invocation may not reproduce the running service's `BOXWEAVE_FORWARD_EXTERNAL_ALLOW`; report source, not a fabricated effective daemon policy. |
| `internal/state/store.go` `Open` | Uses `os.MkdirAll`, `O_RDWR` or `O_CREATE|O_EXCL`, `state.init` and `PRAGMA journal_mode=WAL`. | **Critical: the existing doctor code path is not a strictly side-effect-free filesystem/DB observer.** Design a separate validated RO open; no create/init/WAL-changing pragmas, journal mutation, auto-recovery, snapshots, log rotation or repairs. |
| `internal/control/activity_tools.go`, `internal/control/network_native_activity.go`, `internal/control/recovery_attention.go` | Existing MCP `box_activity_get` overview/issues/evidence/native_network supports bounded, actor-scoped diagnosis; network reads compare SQLite+native, report non-atomicity and incomplete coverage honestly. | Share *internal read-only services*, not remote MCP authentication context; CLI root-only evidence must never be exposed through MCP. Design bounded Project scan/pagination and exact incident links. |
| `internal/recovery/service.go` `Inspect/Reconcile` | Original-ID evidence and a real bounded recovery result with `disposition`, `reason`, `proposed_action`, `native_observation`; `Reconcile` commits an audited state transition with key, no native effect replay. | Current `recover -action=diagnose` needs a known original ID; need discovery/report-to-original navigation and explicit per-issue plan. |
| `cmd/boxweave/recover.go` | Protected Host-only `recover` requires exact operation, reviewer, local root approval, manifest verification; spawns a restricted worker via kernel credential on a root-authenticated Unix socket. Has diagnose/reconcile, inspect/challenge/release and specialized Create/Delete/GC leaves. | **Preserve** all root-only guards, challenges, exact identity, UNKNOWN barriers and action semantics. No unapproved generic repair shortcut. |
| `internal/install/native_prereq.go`, `native_prereq_prepare.go` | Root-readable Native prerequisites and digest-bound plan/apply/rollback/status with exact Project/Profile/ACL. | Unified Host-class diagnosis and safe navigation, without modifying shared Incus resources or assuming native journals can be replayed. |
| `internal/install/local.go`, `health.go` | Manifest/unit/binary and retirement helper checks. `Stage` invokes `verifyRetirementExecutables` before its own writes, requiring fixed root-owned `/usr/sbin/ip`, `/usr/sbin/bridge`, `/usr/sbin/nft`, `/usr/sbin/conntrack`. | Detect missing prerequisites in installer *read-only preview*, before operator approval; show installable OS package/action only, do not silently install it. |
| `install.sh`, `scripts/uninstall_core.py`, `docs/development/V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md` | Installer and #177 guided uninstall, actual bound `--resume-plan` / `--resume`. | Reuse exact audit/manifest/review digests; do not build a second unwinding engine, relax digest checks or override a STARTED audit. |
| `oauth-status`, `oauth-setup`, `oauth-set-origin` | Host-local OAuth password and HTTPS Origin settings; `Enabled` requires both Origin and password hash. | Doctor must distinguish **not configured** from error; never display approval password, hash, access/refresh tokens or OAuth grant secrets. |

Related reading: [S5A1 diagnostic work](S5A1_DIAGNOSTICS_2026-11-20.md), [S5A2 recovery](S5A2_RECOVERY_2026-11-20.md), [UNKNOWN manual recovery](../process/UNKNOWN_MANUAL_RECOVERY.md), [N3/N4 network reliability and capacity](V1_NETWORK_RELIABILITY_AND_CAPACITY_2026-11-22.md), [N3 scoped acceptance](V1_N3_SCOPED_ACCEPTANCE_2026-11-22.md), [Product boundary](../decisions/PRODUCT_BOUNDARY.md), [uninstall operator contract](V1_UNINSTALL_SH_OPERATOR_CONTRACT_2026-11-19.md).

## 3. Actual physical L0 lessons: RC12 retired, published RC13 installed

These are **historical evidence and precise regression requirements**, not current open incidents. Source of truth: [Issue #176](https://github.com/vanishs/boxweave/issues/176) and #177 original commit/CI. **Do not rerun the completed RC12 removal or rewrite the old evidence.**

1. **Interrupted full guided removal (RC12)**: native rollback stopped after controller service was stopped and private archive + original `STARTED` audit persisted. Root cause was a non-owned test override `restricted.devices.proxy=allow` differing from the original Native before/after journal. Operator separately checked and reverted only that key. PR #177 introduced exact read-only `--resume-plan` then Host-tty-approved `--resume` bound to original archive, original review/audit and verified Native `APPLIED`; does **not** rearchive, replay Box mutation or create a new uninstall audit. Owner's Host actual resume returned `outcome: COMPLETE`; separate read-only postflight passed 9/9. Preserve all claims, source digests, archives and evidence. Doctor should show `NATIVE_CONFIG_DRIFT` and `UNINSTALL_AUDIT_STARTED`, original protected phase, bounded next safe action, but never auto-apply a rollback.
2. **RC13 interactive install first denied**: `installation not approved`, exit 1 because exact terminal string `INSTALL user-1001` was missing. This is an expected safety refusal, not service corruption. Show `OPERATOR_APPROVAL_REQUIRED`, not a failed or partial install.
3. **RC13 installation blocked by missing `conntrack`**: after approval, `install-stage` exited `required retirement reader unavailable: /usr/sbin/conntrack`. Source confirms `Stage.verifyRetirementExecutables` precedes its first BoxWeave filesystem write; the operator installed the Ubuntu/Debian OS `conntrack` package locally and verified `/usr/sbin/conntrack`. Retried source-pinned installer; do not reinstall repeatedly for a missing prerequisite. **Actionable development fix:** preview and doctor check fixed executables/owner/mode, report missing packages clearly **before** approval; do not auto-run apt.
4. **RC13 fresh success**: published binary SHA256 `38307cc8a6b9e39830e489873c31b1e0b99a67e4a3d8f06ee3be58038d6da68e`, Schema 6, project `user-1001`, Incus 7.6, `GUIDED_INSTALL_COMPLETE`, `install_exit_code=0`, doctor `ready/mcp_ready/native.ready=true`, zero owned Boxes/Operations/Forwards. Native prerequisites 8/8 OK. Distinct **pending** packet/PRS and real storage-quota proof; never mark them PASS because readiness passed.
5. **OAuth fresh configure**: initial `oauth-status` `{Origin:"",Enabled:false}`; operator entered new approval password on a trusted Host terminal, still showed disabled because Origin remained empty; adding the current Cloudflare Quick Tunnel HTTPS Origin gave `Enabled:true`. Doctor should show `OAUTH_ORIGIN_MISSING` (configuration incomplete) independently of password presence *without exposing password status unnecessarily*. A quick tunnel URL is ephemeral. Diagnose separately service-local endpoint, HTTPS Origin and remote reachability; never create/reset passwords or register remote clients automatically.
6. **Real authenticated ChatGPT L0 MCP**: direct `host_status`, `box_list`, `operation_list`, `port_forward_list` passed; `box_activity_get(view=issues)` returned no listed issues but **bounded/incomplete coverage**; `box_activity_get(view=native_network)` reported complete Project inventory scan, zero native/owned/orphan bindings/fences/unsettled operations, yet cross-source scan **non-atomic**. Do not equate absence of listed issues with a complete trusted all-clear.
7. **Critical diagnostic discrepancy**: locally run `doctor` emitted `FORWARD_EXTERNAL_POLICY_DENIED` and `native.external_forward_policy=disabled` (in its own environment); authenticated *running daemon* `host_status.native.external_forward_policy=all`, because fresh installed service unit sets an environment variable. The local CLI does not inherit the service's environment. Doctor must report `observed_cli_policy`, `running_service_effective_policy` (with trusted evidence) and `source` independently, or `UNVERIFIED`. Do **not** call it a service fault until effective-service evidence confirms mismatch. Explicit external Forward can expose public/future interfaces; default bind remains loopback.
8. **Capacity is not availability**: model limit is 1024 Boxes while current observed IPv4 bridge `10.249.1.1/24` offered upper bound 253 addresses. Doctor should separate logical Project limit, safe native-address upper bound, occupied/pending claims and storage headroom; report IPv4 constraint honestly. No automatic bridge resize on a configured Host.
9. **Platform approval**: `host_status.attention.approval_capability=TRUSTED_CLIENT_APPROVAL_UNAVAILABLE` signals missing separately trusted interactive approval, **not** proof that existing authenticated read-only MCP is broken. Follow current [PRODUCT_BOUNDARY](../decisions/PRODUCT_BOUNDARY.md) on OAuth/user confirmation and unsafe recovery; do not open a root bypass or enable unsupported compensation.

Current PR #177 uninstall CI and real Host postflight were already verified and recorded on #176. **Treat them as retained acceptance, don't redo unrelated tests**; add focused tests only for changed code and run required affected regressions before new code is accepted.

## 4. Proposed product contract (new arguments not yet available)

```sh
boxweave doctor                            # existing small status-compatible view remains
boxweave doctor --full --json              # NEW: bounded, complete-with-coverage diagnostic report
boxweave doctor --full --json --issue X    # optional NEW: focus exact discovered issue
boxweave recover --plan --issue X          # NEW: no mutation; recommended path + exact evidence
boxweave recover --apply --issue X --confirm-review-sha256 SHA   # NEW: ONLY for explicitly supported Host-local leaf, with existing operator authorization
```

These are illustrative UX, not permission to bypass the existing `recover -action=...` CLI. Avoid proliferating incompatible aliases. Root-only checks should be explicit `--scope host` / `--full` with independent root boundary, **not** a silent elevation from ordinary doctor. It is valid for an unprivileged doctor to return `PARTIAL` with `HOST_SCOPE_UNAVAILABLE` and suggested Host-local command rather than claim PASS.

### 4.1 Required doctor coverage and source map

| Domain | Inputs | Evidence/coverage boundary | Repair routing |
| --- | --- | --- | --- |
| Release/installation | exact manifest, binary/unit SHA, service active, restart/drop-ins, native helper units, installer/audit journals | Local installed identity and root-controlled paths; no arbitrary global systemd scan | Host-only validated installation/upgrade/uninstall/resume tool |
| Database/durability | SQLite schema, RO integrity, journal/lock presence, unknown/pending/failed original summaries, old original receipts, archive visibility | Private SQLite **read-only**; no create/init/recovery. Bounded pages, no secret dump | MCP original `operation_get`/`operation_recover` where safe; original guarded Host recover otherwise |
| Owned Box/network | exact owned UUID/incarnation, native Project scan, NIC/MAC/IPv4/IPv6 pinned/fences, orphan/conflict, Peer/Forward dependencies | Confinement to `user-UID`; Native+SQLite **non-atomic**; complete pagination and explicit scan errors | `box_activity_get`/`operation_recover`; Host-only PRS witness where needed |
| Native security | restricted Project/Profile/ACL, exact selected pool, proxy/nesting, bridge, package prerequisites, Root PRS/conntrack records | Project-only service actor; root-only helper/journal with separate Host read | Native reviewed plan/apply/reconcile; never arbitrary Project/foreign resource changes |
| API/auth/exposure | local service health, running service effective Forward policy vs CLI env, OAuth configured, public Origin/endpoint, optional bounded TLS reachability | Never fetch password/token or enumerate OAuth grants for export; do not require external network for base doctor | Operator-local auth/origin setup; manual tunnel config |
| Workload proofs | actual packet v4/v6, PRS revoke/grant, image download, quota, Host mount, Forward HTTP tests | Mark `UNVERIFIED` until independent test proof; running doctor does not mutate or generate synthetic load | Separate owner-approved disposable acceptance, not implicit diagnosis |

### 4.2 Machine-readable report, safe output

Proposed versioned `doctor_report.schema=1` with `product_version`, `commit`, `project`, `generated_at`, `scope`, `overall`, `coverage`, `findings` and `next_actions`. Minimum per finding:

```json
{
  "issue_id": "INSTALL_MISSING_PREREQUISITE:conntrack",
  "severity": "BLOCKER",
  "state": "CONFIRMED",
  "code": "INSTALL_MISSING_PREREQUISITE",
  "resource_ref": "host-prerequisite:conntrack",
  "evidence_source": ["EXACT_LOCAL_FILE_METADATA"],
  "evidence_complete": true,
  "scope": "HOST_LOCAL",
  "effect": "STAGE_REFUSED_NO_INSTALL_MUTATION",
  "original_operation_id": null,
  "repairability": "EXTERNAL_OPERATOR_REQUIRED",
  "next_action": "Install verified distribution conntrack package; rerun read-only preflight",
  "review_sha256": null
}
```

Example represents the **historic observed RC13 failure**, not current installed Host state. Actual `issue_id` should be stable/deterministic per origin and scope and safe to quote, not include secrets. For historical incidents, distinguish `RESOLVED` from active findings. Required distinctions:

- `PASS`: exact check was performed and passed; `FAIL`/`BLOCKER`: independently confirmed fault.
- `WARN`: nonblocking verified concern; `UNVERIFIED`/`SKIPPED`: witness unavailable, missing authority or deliberately not run, **never silently PASS**.
- `coverage`: bounded vs complete, source, scan timestamp, cursor, native snapshot, cross-source non-atomicity and failures.
- `repairability`: `MCP_SAFE`, `HOST_REVIEW_REQUIRED`, `EXTERNAL_OPERATOR_REQUIRED`, `BLOCKED_UNPROVEN`, `NOT_APPLICABLE`; do not mark an uncertain Native Delete repairable without original receipt/proven absence.
- Evidence: immutable original Operation/Tracking/native IDs as appropriate, approved digest or **references**, not raw secrets/private Host paths; no cross-Project data.
- Suggested action: copyable safe command *only if* its preconditions are met, otherwise clear explanation. No command injection from SQLite/Incus strings.
- Exit codes: define/verify as part of implementation (e.g. 0 complete no blocking findings; 2 findings or partial; 1 unsafe/unreadable report). Do not overload `ready` with full acceptance; preserve existing basic `doctor` callers.

Default human output should be concise (summary and top actionable issues), JSON opt-in for CI/agents. Do **not** generate a root-private bundle by default. If a sanitized support bundle is considered later, require exact explicit opt-in, allowlisted redaction and leak tests; no arbitrary log collection.

### 4.3 Recovery plan contract

`recover --plan --issue ISSUE_ID` is read-only and must identify which **existing** mechanism is safe:

1. Resolve issue to exact Project/Box UUID/incarnation/original Operation/manifest/Native root journal and independent native witness.
2. State `can_repair`, `blocking_preconditions`, `affected_resources`, `unsafe_effects`, `required_authority`, selected `MCP` vs `HOST` path, exact pre/post evidence and `review_sha256`.
3. **Only explicitly implemented leaves** may apply. For ordinary Box/Operation issues, prefer existing authenticated MCP `operation_recover` or typed tools with original keys. Do not add new third repair engine or expose privileged Host CLI to MCP.
4. Host-local apply must revalidate exact manifest, Project, issue, original receipt, Native inventory/locks and review digest. Require local explicit approval/attested terminal where existing code requires it. Repeated apply must have idempotent original result or safely refuse; never dispatch a second uncertain native effect.
5. On failure, retain original audits/UNKNOWN/claims and present next safe original-ID diagnostic. Never wipe DB, foreign Project, global bridge/storage, ACK-missing proof, Root journal or old retired archive to make status green.

Critically, root versus service-actor boundaries are **separate**. A privileged Host doctor wrapper may exec a separate `boxweave` service identity for Project-scoped observations and read independent root-only installation evidence, but cannot transfer root capabilities into the MCP runtime or untrusted JSON. Reuse the existing `recoverLocal` kernel-credential/Unix-peer boundary instead of relying on user-provided flags.

## 5. Engineering sequence for the next Agent

**First preserve existing PR #177 behavior**: fetch branch HEAD, review its three original changed files, #176 progress, current CI and current published RC13. No nested-host work or merge by default.

1. **Audit and test data-source boundaries**. Make a trace matrix for every proposed Doctor field: producer, caller UID, read authority, DB/native/Host evidence, redaction, boundedness, coverage, mutation risk, exact current CLI/MCP path. Pay special attention to SQLite `state.Open`, native scan and daemon-vs-CLI policy discrepancy. Capture required behavioral examples; no duplicate backend for matching checks.
2. **Implement true read-only Doctor first**, preserving existing minimal `doctor` behavior/exit code unless deliberate backwards-compatible versioning is reviewed. Separate RO SQLite opening from `state.Open`; validate no creation, WAL pragmas/mtime, lock or state mutations when service is running **and** when DB is missing/damaged. Add typed findings and bounded `--full --json`; provide partial, not counterfeit PASS, when Host-scope unavailable.
3. **Connect existing issue-to-original discovery** and Native/network evidence. Share read-only bounded services from Activity and Recover; explicit no foreign inventory, no secret output, cross-source non-atomic semantics. Add installer binary prereq + service-effective Forward checks with source labels and OAuth configuration status.
4. **Add read-only `recover --plan --issue`** and supported guarded apply routing only after evidence/authority contract is proven. Reuse existing specialized Host recover, root/prereq, and #177 uninstall `--resume-plan` / `--resume` rather than duplicate repairs. New issue categories may initially be `BLOCKED_UNPROVEN` with a human next action.
5. **Verify iteratively in a disposable Box before GitHub CI**: focused unit/race/negative tests, simulated corruption/drift/permission denial/crash, source-exact installed CLI, real confined Incus when feasible, then existing impacted PR #177 uninstall tests and CI. Only new/affected acceptance tests; honor prior completed Host RC12/RC13 receipts.
6. Update docs/CLI usage, examples and accepted capabilities, attach evidence/coverage matrix and no-merge checkpoint to PR #177. **Keep Draft; do not install development binary on physical L0** without explicit approval.

## 6. Mandatory acceptance cases

- `doctor` true-RO evidence: DB missing, corrupt schema, symlink/hardlink, read-only directory, live WAL with concurrent service writer, root-only files unavailable; stable DB inode/mtime/content/size and no installed/systemd/Incus mutations. Distinguish structural read failure from no issues.
- Real errors from this migration: `conntrack` missing reported **before** installer approval; drifted proxy Project prevents unsafe rollback; original `STARTED` uninstall detects exact #177 resume possible vs tamper/refusal; terminal approval refusal is not a partly installed failure.
- Current RC13 healthy: Schema 6, service=active, Native 8/8, optional configured OAuth origin, no invented packet/quota PASS; `doctor` CLI effective policy `disabled` and daemon effective `all` correctly distinguished from independent authoritative observations.
- 1024 model limit vs /24 capacity; on-demand image warnings; missing root quota witness; complete native-network pagination, 0-orphan normal state, synthetic orphan binding/fence, restored SQLite mismatch, foreign native redaction, signed cursor race/drift and non-atomic source warning.
- Unknown Create/Delete/Forward/Peer/PRS, lost ACK, stale original receipt, dependent Delete partials, prior historical released UNKNOWN: exact original holds, never replay uncertain side effects or auto-unlock; same-key and modified-intent refusal; invalid/stale review SHA.
- OAuth missing password vs Origin, transient Quick Tunnel, absent/tampered unit, PRS helper path missing, protected support report sanitization, error/JSON stable ordering, bounded runtime and honest coverage.
- Legacy `doctor/status`, existing `recover -action=...`, installer/uninstaller, Native preflight and current MCP tools behave unchanged; RC13 L0 remains untouched until owner approves actual Host install.
- Review acceptance in phases `DOCTOR_RO_PASS`, `RECOVER_PLAN_PASS`, `INTEGRATED_NATIVE_PASS`; GitHub CI alone is not proof of the independent physical Host integrations, and no v1 merge is implied.

## 7. Open decisions to recommend, not silently assume

- Should default `doctor` keep existing output exactly (recommended), adding rich diagnostics only via `--full`?
- Is separately authorized Host `--full` a wrapper/in-binary implementation, or a minimal pinned rescue helper for broken/uninstalled binary? Keep root-only scope constrained and reviewable.
- Which issue types are genuinely safe to route to MCP versus requiring local root/native journal or an operator's system package manager?
- What are the finite scan/time budgets, issue stable-ID algorithm, report JSON compatibility/exit-code rules, and source provenance?
- Whether PR #177 should eventually be split for review. **Owner explicitly requested continuing this work in #177 now; do not split, merge or retarget unilaterally.**

**Handoff prompt:** [V1_UNIFIED_DOCTOR_RECOVER_AGENT_HANDOFF_2026-11-23.md](V1_UNIFIED_DOCTOR_RECOVER_AGENT_HANDOFF_2026-11-23.md).
