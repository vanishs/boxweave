# Phase cards — load only the phase you are executing

**Canonical entry:** [Implementation plan](../../process/IMPLEMENTATION_PLAN.md). **Execution rules:** [Phase execution protocol](../PHASE_EXECUTION_PROTOCOL.md). **Current facts and handoff:** [Execution log](../EXECUTION_LOG.md). The cards are **not separate products or independent feature specifications**; each one defines a bounded, continuous build→test→fix→acceptance work package.

| Order | Phase card | Scope delivered inside this phase | Tool count newly activated |
|---|---|---|---|
| **S1** | [Foundation, installation, identity, journal](01_FOUNDATION.md) | Go runtime + official MCP HTTP, minimal local installer/reuse, restricted Incus, SQLite, auth, tracking, host_status | **1** |
| **S2** | [Owned Box lifecycle + durable async Exec](02_BOX_LIFECYCLE_OPERATIONS.md) | 7 Box lifecycle tools, Exec and four operation tools; crash/UNKNOWN/idempotency/readiness | **12** |
| **S2.5 BRIDGEHEAD** | [Independent early development-base / real-host gate](025_BRIDGEHEAD.md) · [BH-001–BH-017](../../verification/BRIDGEHEAD_ACCEPTANCE.md) | Actual S2-accepted v1 host/Box/Exec/SQLite/MCP, strict native L0→L1→L2 minimum, bounded load, recovery, independent real-host approval and rollback; no S4 feature claim | **0** |
| **S3** | [Guest files + complete Agent workflow](03_FILES_AGENT_LOOP.md) | file list/read/apply, SFTP exclusive create, honest unsupported conditional modes, guest Git/Exec fix/retest | **3** |
| **S4** | [Native network + nesting + delete integration](04_NETWORK_NESTING.md) | three TCP/UDP forward tools, directional box_peer_set, one shared ACL/Scriptlet, two-layer parity and dependency-aware delete | **4** |
| **S5** | [Adversarial full acceptance, candidate and release](05_RELEASE_ACCEPTANCE.md) | 20/20 installed black-box, CLI leaves, Finder/Fixer/independent Verifier, exact artifact provenance and release gate | **0** |

**Five feature phases + one inserted S2.5 acceptance/deployment checkpoint; not six feature releases. 20 public MCP tools exactly: 1 + 12 + 0 + 3 + 4 = 20.** A phase can also test and strengthen earlier tools without creating more public tools. Tool names and contracts remain authoritative in [MCP_TOOLS](../../design/MCP_TOOLS.md).

**Run instruction:** Execute one **S#** end to end following its card, including internal focused research, implementation, tests, independent acceptance and bug-fix iterations; keep changing docs with code as necessary. Log every durable action and any BLOCKED/UNKNOWN with original identifiers. Do not stop because one subcase requires an owner decision; continue unaffected work while retaining honest phase status. Do not publish unless S5 mandatory gate is genuinely PASS.

**BRIDGEHEAD is a stronger intermediate gate, not a shortcut:** S2 PASS is prerequisite, and BRIDGEHEAD LAB_PASS does not authorize physical host installation; HOST_GO needs explicit owner approval, independent real proof and rollback. S3 continues the original three-tool phase and S4/S5 retain their full contracts. If physical pilot cannot be authorized, record BLOCKED and continue safe isolated S3 without claiming early host readiness. [Acceptance matrix](../../verification/BRIDGEHEAD_ACCEPTANCE.md).

**First start:** read S1 only; do not preload cards S2–S5. **Subsequent starts:** read only the current card and preceding phase's handoff checkpoint. Any newly discovered cross-phase dependency must be entered in the shared log with an owner and explicit closure phase; adjust cards via links, not repeated large descriptions.
