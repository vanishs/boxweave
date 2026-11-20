# Phase cards — load only the phase you are executing

**2026-11-16 active state:** S3 is [formally signed and merged](../S3_MINIMAL_PASS_2026-11-16.md). [S4 Draft PR #153](https://github.com/vanishs/boxweave/pull/153) is opened for **preparation only**; next Agent begins `S4-A` in that same PR, followed by `S4-B` and `S4-C` as independently reported checkpoints. [Current S4/S5 reallocation](../S4_S5_EXECUTION_REBALANCE_2026-11-16.md), [kickoff](../S4_EXECUTION_KICKOFF_2026-11-16.md). Older no-S3-PASS/status rows below are historical. Tool total remains exactly 20 after S4, and S5 final assurance/Release remains mandatory.

**Latest owner stage allocation (2026-11-15):** [minimal S3 and explicit S4/S5 carry-forward](../S3_S4_S5_MINIMAL_REBALANCE_2026-11-15.md) takes precedence over historical phase-boundary language. Public catalog remains 1+12+0+3+4=20. **No S3/S4/S5 source acceptance is implied by this edit.**


**Canonical entry:** [Implementation plan](../../process/IMPLEMENTATION_PLAN.md). **Execution rules:** [Phase execution protocol](../PHASE_EXECUTION_PROTOCOL.md). **Current facts and handoff:** [Execution log](../EXECUTION_LOG.md). The cards are **not separate products or independent feature specifications**; each one defines a bounded, continuous build→test→fix→acceptance work package.

| Order | Phase card | Scope delivered inside this phase | Tool count newly activated |
|---|---|---|---|
| **S1** | [Foundation, installation, identity, journal](01_FOUNDATION.md) | Go runtime + official MCP HTTP, minimal local installer/reuse, restricted Incus, SQLite, auth, tracking, host_status | **1** |
| **S2** | [Owned Box lifecycle + durable async Exec](02_BOX_LIFECYCLE_OPERATIONS.md) | 7 Box lifecycle tools, Exec and four operation tools; crash/UNKNOWN/idempotency/readiness | **12** |
| **S2.5 BRIDGEHEAD** | [Independent early development-base / dedicated-host gate](025_BRIDGEHEAD.md) · [BH-001–BH-017](../../verification/BRIDGEHEAD_ACCEPTANCE.md) | Exact S2 candidate, minimum install Project/Profile/image/ACL/quota prerequisites, unprivileged L1 quota/NAT and L2 child, bounded workload, independent LAB and dedicated-host approval/rollback; no S4 feature claim | **0** |
| **S3** | [Guest files + one supervised Agent smoke](03_FILES_AGENT_LOOP.md) | 3 safe File Tools (read/list/O_EXCL create, unsupported unsafe modes), a single attended guest Git FAIL→FIX→PASS smoke, reusable evidence IDs | **3** |
| **S4** | [Native network + nesting + delete integration](04_NETWORK_NESTING.md) | #148 stable OAuth actor and X-02/X-03 native Create/Delete boundaries, 4 network Tools, shared ACL/Scriptlet, owned nesting and dependency-aware Delete | **4** |
| **S5** | [Adversarial full acceptance, candidate and release](05_RELEASE_ACCEPTANCE.md) | #151 restart/native receipt stabilization, TC-015/16 full two-Agent + TC-019/20, 20/20 installed/CLI, independent Verifier, release provenance | **0** |

**Additional stage-specific operation continuity:** S2 [TC-001–TC-010](../../verification/CONTINUITY_ACCEPTANCE.md) implements [4 MCP + 4 domain abilities](../../design/AGENT_OPERATION_CONTINUITY.md); BRIDGEHEAD TC-011–014, S3 one-Agent smoke only (not TC-015 PASS), S4 TC-017–018, S5 **full TC-015–016** with [client checkpoint](../TASK_CHECKPOINT_CONTRACT.md) and TC-019–020. This supplements, never replaces, A-/SDK-G/BH native gates. Future MCP Tasks adapter is not activated; no new Tool or task engine.

**Five feature phases + one inserted S2.5 acceptance/deployment checkpoint; not six feature releases. 20 public MCP tools exactly: 1 + 12 + 0 + 3 + 4 = 20.** A phase can also test and strengthen earlier tools without creating more public tools. Tool names and contracts remain authoritative in [MCP_TOOLS](../../design/MCP_TOOLS.md).

**Run instruction:** Execute one **S#** end to end following its card, including internal focused research, implementation, tests, independent acceptance and bug-fix iterations; keep changing docs with code as necessary. Log every durable action and any BLOCKED/UNKNOWN with original identifiers. Do not stop because one subcase requires an owner decision; continue unaffected work while retaining honest phase status. Do not publish unless S5 mandatory gate is genuinely PASS.

**BRIDGEHEAD is a stronger intermediate gate, not a shortcut:** S2 PASS is prerequisite, and BRIDGEHEAD LAB_PASS does not authorize physical host installation; HOST_GO needs explicit owner approval, independent real proof and rollback. S3 continues the original three-tool phase and S4/S5 retain their full contracts. If physical pilot cannot be authorized, record BLOCKED and continue safe isolated S3 without claiming early host readiness. [Acceptance matrix](../../verification/BRIDGEHEAD_ACCEPTANCE.md).

**Pre-deployment schema freedom:** [S2 may redesign the first installed schema and internal worker/admission flow directly](../../process/VERSIONING.md#initial-schema-freeze--no-user-data-migration-before-first-deployment), without migrating disposable S1 lab SQLite data; reverify all affected S1 safety paths and preserve unrelated native UNKNOWN. S2.5 pre-release pilots do **not** require S1/S2 SQLite/API compatibility. They do require original native journal preservation/reconciliation and explicit safe reset; formal supported release (or separately accepted retained-data contract) freezes compatibility.

**First start:** read S1 only; do not preload cards S2–S5. **Subsequent starts:** read only the current card and preceding phase's handoff checkpoint. Any newly discovered cross-phase dependency must be entered in the shared log with an owner and explicit closure phase; adjust cards via links, not repeated large descriptions.
