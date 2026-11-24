# PR 178 N3 — Root execution, durable policy and recovery

Status: N3 development checkpoint; **NOT final N3 acceptance**. Fresh install only. MCP scoped Peer remains disabled until installed end-to-end verification. Do not merge or release on this evidence. Prior N2 acceptance and owner decisions remain authoritative.

## Implemented

- Stage and claim an exact schema-2 original in one SQLite transaction. Proof binds actor/token, operation/resource claim layer, complete live and retired ledger, and terminal Peer history. Completion requires the exact proof digest and a complete four-step typed Root receipt; it atomically finalizes the original and its grant/revoke row. UNKNOWN recovery is read-only and does not repeat writes.
- Directional dual-stack conntrack cleanup, not blanket pair deletion. Four-step durable execution composes guarded ACL CAS, directional conntrack, and Guard publication with full native prefix witnesses; fixed Root adapter derives its original, scope and policy independently of public input.
- Root boot policy is append-only, canonical typed data bound to the installation, exact original, verified step records, previous policy and before/after digests. Incomplete/unpublished originals, missing records, tampering and unexpected histories fail closed. Boot restore recompiles verified policy; it never executes stored arbitrary nft text.
- Completed UNKNOWN readback checks complete ACL/Guard/port/global state and original again without replay. Root provenance includes current and retired lifecycle receipts and native absence for retired UUIDs/ACLs.
- Create/Delete validate terminal Peer policy rather than prohibit all historical grants. Root compares active policy against published boot policy and compiles exact per-owner ACLs including retained revoked history. A live dependency still prevents deletion. Retired owners cannot be granted new access.

## Evidence in isolated development Box

Baseline 01e95ab passed four CI workflows. Worktree: n3-01e95ab. No existing production install or nested lab instances modified, no L0 network writes.

- Real isolated Linux namespace Guard + conntrack: 68e14871f439d0b843e74966c33af344. IPv4/IPv6 TCP/UDP A->B revoked; B->A and C->D existing tracked connections preserved. This is a Guard/conntrack test, NOT proof of Incus ACL kernel convergence.
- SQLite atomic completion fault matrix (16 cases, race): c04252a11093d6e5b9119185da5f4aad.
- Composed Root stages (14 cases, race, mock SDK/sysfs/Guard/conntrack): 1528963a47be8a405fe5a9a2ee06a473.
- Boot journal and completed UNKNOWN recovery (3 race runs, mock SDK, filesystem journal, no host reboot): c51e29eb9b9556400f6965a8de54b92a.
- Updated original/stage proof and history: 9c98e9fb04fdafffc3b0b20b29d70b86.
- PR178 state/install regression after terminal history and lifecycle integration: e3895d79539bd8de5b64f01f0626b8b9.
- Create/Delete with existing grant, revoke, retired target; pending policy refused (real SQLite transactions, synthetic Root receipts): 7ee7bdda8c3629009f426f3201a76366.

## Remaining final gate (do not treat as passed)

1. Resolve scoped ACL compiler reverse-direction grant/revoke coexistence: legacy symmetric DROP semantics still reject overlapping reverse live/tombstone policy. Prove no transient or stale-flow bypass in real Incus ACL + Guard.
2. Wire exact scoped MCP admission/replay, authenticated local Root dispatch and read-only UNKNOWN recovery. Preserve fresh-only gate and reject old schema before effects. Current private Root adapter is not exposed; SCOPED_PEER_N3_NOT_ENABLED remains.
3. In a newly installed disposable nested Box, exercise MCP grant/revoke/regrant, preserve independent reverse and unrelated established flows, actual per-NIC ACL kernel convergence, lifecycle with active and retired policy, boot restore, explicit false/default true autostart, Root crash/ACK loss and concurrent requests. Prove global native/lease/route witness against real SDK and no unrelated effects. Incomplete/unpublished originals must not restore uncertain grants. Distinguish safe blocking from automated successful recovery.
4. Submit installed N3 candidate workflow with executable evidence and require exact-head final CI. N4 broader matrix and N5 remain separate; PR stays Draft and #177 remains frozen pending #178.
