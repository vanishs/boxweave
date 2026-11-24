# BoxWeave v1 N3 - scoped acceptance (2026-11-22 PT)

**Decision: N3 functional scope accepted on independent Box evidence; exact-head GitHub CI pending. N4 remains open.** No PR merge, v1 release, root/physical Host trust claim, or real 1024-native-instance scale claim is authorized by this sign-off.

**Source:** PR #172 N2 parent `64bf8b6216e2c78182ce9bf95b32f1e38bda966b`; N3 code/test HEAD `74238c292f19cf92062108f8c6e788ac6095700f`, tree `e8bb5f7df47f5d1b70ee47a8f1c5d9760226aa8f`. No new public MCP write tools, unchecked Host shell, native identity adoption, or weakening of UNKNOWN/fences.

## Independent gate matrix

- **Create staged/ACK-lost/terminal fault and exact missing binding/fence repair: PASS.** Native original Create marker/UUID/static NIC, four original static claims and complete or independently proven terminal native effect are required. Exact separate recovery is idempotent, rejects forged/missing provenance and rolls back on DB fault. See detailed original recovery operations in `V1_NETWORK_RELIABILITY_AND_CAPACITY_2026-11-22.md`.
- **Delete tombstone, real lost ACK, dependent UNKNOWN: PASS.** `8d28ad045a4eb9c78a3823fde80ab643` original Worker two-ACK fault native E2E, `76cf2957177e133762bd6373a64df469` tombstone recovery. No fabricated native receipt or speculative address release on ambiguous Delete absence; same original remains blocked without independent proof.
- **Database restoration, native NIC drift and orphan diagnosis: PASS.** `0af38d3d3174c61a9ffc8ef26a13036e` actual NIC drift, `573d403b4ab8168bf8d2097e1beea5b9` restored corrupt private SQLite MCP, `e3bdd9c07263b2313943a60fbf2cbb1b` native complete inventory. No foreign instance names/UUID or automatic adoption.
- **Bounded Project scan and signed diagnostics: PASS.** `8b8377cd89dd30007e645881b394429a` native unowned redaction, `6b8503ac6fc30e83be98e1738039b7ce` lease/network race matrix, `c04c45c81ded5ea3a9649fee0bf8ea87` double-witness native readback; authenticated actor-bound snapshot-digest pagination. Native or SQLite observer failure never claimed all-clear; cross-source reads never claimed atomic.
- **Real MCP authentication and systemd restarts: PASS in isolated L1 Box.** `de00fca4d53dd27cdd965b2e286dcde1` three independently started MCP processes. Actual restricted systemd service `d5d5e121bf83faaf55fc61136e53166d` initial healthy, `f928af1315433f268e15dbba18cf2944` restart with copied DB MAC tamper and same token, `d7c28e5d733d306e539cbee0b6e624c6` second restart after restoration. Inherited LXC NoNewPrivileges override corrected ONLY for isolated test unit, actual kernel `NoNewPrivs:1` and MCP PASS `9ca554482c889ed993949920e476574e`; ephemeral unit and token removed, untouched native Box `f0119128eddd34317770474a5845506a`.
- **1024/1025 capacity safety: PASS as bounded fixtures only.** `e3a1f30d378a9c1e32e310aca91ee882` SQLite 1024/1025; `a2f68ef05503d41b67ab5011a5ed3ec5` model-native 1024/1025. Real 1024 guest throughput is N4, NOT proven here.
- **Independent clean source checkout: PASS.** Detached clean worktree at exact N3 HEAD, operation `3a5106bdea845eec7054fdb86e04f8cd` exit 0: `scripts/check_format.py` 823 tracked files, `go test ./internal/control ./internal/native ./internal/recovery ./internal/state -run '^TestN3' -count=1`, `go vet ./...`; previous full repo `89e4ccec1bc94043f0bd47eea6dfb32c` exit 0 with `go test ./...` and native N3 race PASS.

## Limits carried into N4 and release gating

Ambiguous native Delete with missing original receipt is **not** safe to replay/release. Irreconcilable Create provenance, restored DB without original trust, foreign Project identity, shared root/ACL/kernel changes or unavailable trusted native verifier can require Host escalation. Safe normal-Box recovery needs exact original proof, not manual SQL/IP seeding. Official N2 installed lifecycle already had independent CI acceptance `45096812279`; N3 isolated systemd proves N3 operation but is not a physical Host installer replay.

N4 must independently verify installed/root/PRS/ACL/Forward data plane, physical/native 1024-instance performance and resources, and whole-PR merge gates. PR #172 remains Draft until N4 and required checks pass.

**Sign-off:** Scoped N3 technical **Box acceptance by the verification agent**, based on retained real native receipts and independent fresh checkout. This is not a second human or owner signature. Record exact-head GitHub CI separately; do not declare a pending check green.
