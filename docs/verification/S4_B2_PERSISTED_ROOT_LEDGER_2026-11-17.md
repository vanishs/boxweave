# S4-B2 — persistent SQLite to Incus-root identity proof (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** No MCP Peer write or shared L0 host mutation.

## Purpose and boundary

Previously the B2 root bridge audit took a caller-assembled A/B/C inventory. This change introduces a typed, bounded, read-only SQLite snapshot and a host-root-only adapter that independently checks the actual Incus global bridge, ACL, Project, Profile, all native UUIDs, NICs, MACs and static IPv4/IPv6 addresses. A local SQLite claim is *not* native proof.

`state.Store.ReadCompleteNetworkAuditLedger` requires 1–64 owned Boxes with exactly one PINNED eth0 per original UUID/incarnation on the target bridge/ACL, canonical unique MAC/IPv4/IPv6 and matching RESERVED address fences. Missing, mismatched, foreign, duplicate, released or quarantined records fail closed without returning a partial ledger. The read is one bounded SQL transaction, including row errors.

`install.NativePeerBridgeAuditFromStore` additionally requires host root and the exact local service UID-derived Project. It never gives the MCP runtime the host socket, makes a native ACL mutation, restarts a Box or executes root shell.

## Evidence and remaining work

In L0 isolated worktree `/root/boxweave-b2-runtime-1117` (Box UUID `745d1550-97e8-4a02-9f19-4ef356a0274e`), three-Box SQLite positive and foreign-Project / missing binding / MAC format / wrong ACL / UUID-incarnation / missing, released, quarantined and reused v4/v6 fence negatives were exercised. A syntactically valid *false* MAC is deliberately **not** authenticated by SQLite; it must be refused by Incus root postread.

The disposable Incus 7.6 native fixture `TestS4B2NativeStoreLedgerBridgeAudit` now exercises real SDK identities seeded into a temporary test SQLite DB, root readback, canonical forged MAC denial, and IPv6-quarantine denial; the quiescent GitHub workflow requires non-SKIP markers. **The exact-commit native result is pending until CI runs.**

This is a **read-only prerequisite** only: production worker has not yet recorded PINNED bindings, the installed controller has no root-audit handoff, and independent per-NIC DROP/conntrack revocation and post-UNKNOWN recovery are not closed. An unprepared legacy/S1 DB must fail closed. No `S4_B2_PASS` or B3 authorization is claimed.

## First exact-head native run — failed (2026-11-17)

- [GitHub Actions #44144350444](https://github.com/vanishs/boxweave/actions/runs/44144350444), original native job `134255284885`, source `e691abadb0f9c9f5e13ad1f2ed9a88819ebfcc97`, **FAILURE**.
- On fresh Incus 7.6, `S4_B2_QUIESCENT_BASELINE_A_TO_B_DENIED=PASS` and existing exact root bridge, MAC spoof, missing-sibling and foreign-bridge negatives all PASS. Immediately afterward, the new temporary-SQLite native test returned a nonzero exit. The original workflow redirected its stdout/stderr to `s4b2-sqlite-native-root-audit.txt` without printing or uploading it on failure, so the *specific native error was not captured*. This is **not** evidence of native pass, spoof bypass or a known root cause.
- Follow-up workflow change prints the native test's original failure output before exiting and uploads the exact test output as a CI artifact. No native shared L0 state was touched. Keep B2 PARTIAL; do not infer success until a fresh exact-commit run with all required markers.

## Exact native failure and narrow fixture repair (2026-11-17)
Run [#44145170209](https://github.com/vanishs/boxweave/actions/runs/44145170209), original job `134258211023`, exact source `3f44d5f45683266fe6f4a9d9ad23dbe1fcee8ba6`, failed **before** Incus root identity comparisons. Captured original Go error: `database parent must be a private real directory` at the temporary SQLite setup in `TestS4B2NativeStoreLedgerBridgeAudit`. The fixture now explicitly makes its own one-use DB parent `0700`, preserving `state.Open` security checks unmodified. This is a test-environment fix only: latest exact-commit native outcome must still be retested, and B2 remains PARTIAL.
