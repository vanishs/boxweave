# S4-B2 -- root conntrack write-ahead UNKNOWN journal (2026-11-17)

**S4_B2_PARTIAL; B3 NOT STARTED.** This is a root-only internal execution primitive and crash-safety test suite, **not** a new MCP Tool, root socket handoff or permission to modify the shared L0 host.

## Design

The root-only `NativePeerConntrackApplyReviewed` requires an approved SHA from a fresh host-bound `NativePeerConntrackReview`, a validated existing read-only SQLite ledger, exact project/bridge/ACL/source/target UUID scope and host root. Under an exclusive root-local lock it re-reads the real global Incus bridge/Project/Profile/ACL inventory, confirms all A/B/C stopped with autostart off, the complete protected IPv4/IPv6 MAC/lease ledger, zero live host kernel bridge ports and reverse dual-stack DROP.

Before calling the existing verified exact conntrack purge, `nativePeerConntrackDispatchOnce` creates a root-only `0600`, O_EXCL/O_NOFOLLOW journal and fsyncs the file and its directory. The original status is **DISPATCH_UNKNOWN**, even if a crash occurs before dispatch. No journal is overwritten or retried for a second attempt. When the native operation returns a fully checked report, completion may transition to **COMPLETE_VERIFIED** by an atomic synced update, but only if four directed IPv4/IPv6 scopes were probed, deletion counts are bounded, the native audit digest matches the SHA-bound review and ReadyForPeer remains false.

Any lost native ACK, canceled context, crash, partial receipt, drift, unsafe journal file, final persistence failure or prior journal retains UNKNOWN and blocks replay. A verified conntrack purge **does not** imply a peer can be granted.

## Proof and limitations

Deterministic Go race tests simulate successful single dispatch, repeated approval, concurrent dispatch, lost ACK after a partial native effect, abrupt process death after journal fsync, failed final write and incomplete/foreign native digest/scope. A new disposable Incus 7.6 CI gate runs those tests and demands explicit non-SKIP markers. It operates only on private temporary journal files; no live root deletion is attempted in this journal fixture. The existing installed root verifier/native conntrack root tests remain separate.

For safety there is deliberately **no CLI apply command**. This journal currently supports one original attempt per host file, with no recovery/archive/reconcile capability; that conservatism is intentional until independent native/recovery acceptance establishes safe reuse. An exported host-root library entrypoint alone is **not** an installed controller handoff, durable SQLite-to-root operation admission, per-NIC nft convergence or end-to-end B2 signoff.

Shared L0 global Scriptlet/ACL was not changed. The new temporary L0 Box UUID is `d1f3d8de-b66e-4fd8-9b5a-e7e41d3f9f72`, original create `f26122715bf794c158165a85d619c02f`, native `c6fb81c9-7a82-4a17-8773-9c032764099e`. B3 must not start.
