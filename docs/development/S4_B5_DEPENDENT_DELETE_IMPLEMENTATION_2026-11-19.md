# S4-B5 dependent Box Delete implementation — 2026-11-19

Status: implementation and development-Box regression pass; native acceptance pending. This is not S4_B5_PASS.

## Contract
- A dependent delete holds one durable parent operation and a Project-wide admission fence. Its internal children revoke confirmed instance forwards, revoke confirmed peer grants, and stop the instance if required. No new public tool or database tables are required.
- Peer revocation still requires the existing independent host-local PRS review and exact digest approval. The controller cannot grant itself root approval. Linked child claims are verified against the parent actor, token, Project, UUID and incarnation.
- Unknown, quarantined, foreign or mismatched dependencies fail closed. A failed or interrupted child prevents final deletion, and no compensating regrant/recreate is attempted.
- Native UUID, exact pinned NIC and complete proxy inventory are checked. Parent success requires native delete completion and independent absence plus atomic tombstone and dependency settlement.
- Completed deleted network bindings and addresses retain verified historical lineage. RELEASED history does not permit address reassignment.
- Dependency-free deletion retains the existing STOPPED-only behavior. A dependent deletion may stop the exact Box as a recorded child.
- Parent and child IDs are linked in operation resource claims and safe operation events. Recover reads the durable originals rather than replaying side effects.

## Evidence and boundaries
Development Box: s4b5-dependency-lab (1dc9a7f9-3f12-4d2f-b07b-f81c9edd838f).
Work ref: s4b5-dependent-delete-20261119.
- SQLite tests: parent/child atomicity and replay, linked root proof and historical DROP retention, partial cleanup/restart UNKNOWN fencing.
- Development Box full state/native/control Go package tests passed; original Exec f5607aec196ec049a41537ae7701eb1b, guest exit 0.
- Native acceptance test is a one-use restricted Incus fixture, using actual HTTP MCP forward creation, TCP/UDP echo, parent dependent deletion, native absence, packet refusal, SQLite child receipts and replay. It runs after existing B4 native checks.
- Still requires actual linked peer PRS revocation integration and parent final-delete ACK-loss/reconciliation acceptance. No live L0 host configuration changes are authorized. B6 is not started.
