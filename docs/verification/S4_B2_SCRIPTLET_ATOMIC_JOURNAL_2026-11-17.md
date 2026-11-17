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
