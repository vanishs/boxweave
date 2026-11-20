# S4-B2 Scriptlet root original-intent hardening (2026-11-17)

Stage: S4_B2_PARTIAL. B3 not started. Only Draft PR #153.

## Recovered original L0 work

The original guest worktree at 5f6ee4b4 was preserved untouched, and a new detached clean worktree was created at remote 851bcbd8. The earlier root bridge audit and scoped conntrack files were already integrated upstream and were not overwritten. Only unmerged Scriptlet journal and locking changes were ported, with independent unit tests.

## Implementation

- Root-local apply and rollback acquire a fixed host file lock via nonblocking flock, O_NOFOLLOW and 0600 mode; fail rather than allowing concurrent global authorization updates.
- The original, pre-dispatch Scriptlet journal is created with O_EXCL and fsync, including original and intended server configuration, plan digest and DISPATCH_UNKNOWN.
- Later writes require matching full intent and monotonic status transitions; ROLLBACK_UNKNOWN never reissues an uncertain native PUT.
- Incomplete journals from interrupted persistence deny both apply and automatic rollback.
- No host shell, MCP privilege escalation, daemon ACL mutation, or Scriptlet switch was performed on L0.

## Original receipts and checks

- L0 disposable development Box: UUID 745d1550-97e8-4a02-9f19-4ef356a0274e, owned Project user-1001; Incus 7.6, Go 1.26.9.
- Inspection: BoxWeave operation fc216f1d174c85969d9a9db4da8b51d4 / native 1fd7092e-8209-48c9-bd42-aa4fee123584 SUCCEEDED; old HEAD 5f6ee4b4, clean current source 851bcbd8.
- Focused tests: original BW 1c3cb5170a722299ab41758a6bbec974 / native 614a67bc-474b-486b-9cd9-e6ae75406748 SUCCEEDED.
- Complete make verify after gofmt: original BW ce595f89411bab28dd570dddb95a8d4d / native 5e96c48f-6d8d-472e-8d15-8d7d1f4c7fd2 SUCCEEDED, including go vet, race tests, mod verify, tidy -diff, YAML, tracked formats, and seven Python witness tests.
- Negative tests cover competing first-intent writes (single writer), lock reentry, symlink/mode attack, forbidden journal rewind, uncertain rollback ACK retry, and interrupted incomplete original intent.

This is unit and isolated-guest evidence only. Root Scriptlet installed-native security, original TLS client rights, host-specific approval, S4-B2 MCP end-to-end, external NIC IP spoofing, cross-project isolation and independent final-head reacceptance are still required. Exact-head CI 851bcbd8 has quality and dual-arch SUCCESS, B2 isolated native SUCCESS, but S4-A native installed FAILURE at operation_list debugger boundary; same SHA rerun requested. No B2 PASS or B3 admission.

## 2026-11-17 PT — explicit root-only no-PUT reconciliation (B2 still PARTIAL)

A subsequent isolated B2 source change provides `boxweave native-scriptlet-reconcile` (a local operator CLI command, **not** an MCP Tool). Under the same root-only authorization lock, exact approved review SHA and host identity, the command reads the Incus Server configuration twice with matching ETag and config. It **does not issue UpdateServer**, never makes the peer writer ready, and only records:
- `DISPATCH_UNKNOWN -> ROUTE_APPLIED_PARITY_PENDING` if both reads show the exact planned applied configuration. Real restricted-client parity remains a separate gate.
- `ROLLBACK_UNKNOWN -> ROLLED_BACK` if both reads show the exact original configuration.
- All uncertain, partial, unproven-in-flight, foreign-host, unapproved or drifted cases retain the original journal and return an error. Observing the original state after an uncertain *apply* never automatically declares it rolled back.

Unit fixtures proved lost apply ACK, lost rollback ACK, second reconciliation idempotency and refusals for partial/before-dispatch and wrong host/SHA. This is a **source-only** recovery primitive; a real Incus Server crash/restart, installed authorization parity and full host-global approval have not been demonstrated. B2 stays `S4_B2_PARTIAL`; B3 remains prohibited.

**Local final quality receipt:** L0 guest Exec operation `3d904067c98819cc193fd263fd395254`, native Exec `597ca4a7-e978-4e8b-a075-a378798f5031`, independently read `SUCCEEDED`. The staged five-file worktree passed full `make verify`: Go vet, full race suite (all packages), format/YAML, module checks and 7/7 Python witness cases. No installer/live Incus Server write was attempted.

## 2026-11-17 PT — durability failure cannot promote in-memory status

The root journal workflow now distinguishes an **attempted** status transition from a **persisted** one. When the final O_EXCL/atomic journal update fails after native readback or rollback, the error report retains the last durable `DISPATCH_UNKNOWN`, `ROLLBACK_UNKNOWN` or other prior status; it must never report `ROUTE_APPLIED_PARITY_PENDING`/`ROLLED_BACK` just because an in-memory struct was updated. No native PUT is replayed.

A fault-injection unit fixture switches the 0600 journal to an unsafe mode between two root Server reads and final save. Both lost-ACK apply and rollback cases returned an error with the **original durable UNKNOWN** status; after restoring mode, independent journal readback confirmed no state advancement. This does not replace native process-crash or installed-host testing. **B2 remains PARTIAL; no B3.**
