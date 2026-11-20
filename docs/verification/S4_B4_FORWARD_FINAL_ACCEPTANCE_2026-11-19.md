# S4-B4 final scoped acceptance — 2026-11-19 PT

**Decision: S4_B4_PASS.** This supersedes the B4 PARTIAL decision on the initial loopback-only implementation. The accepted boundary includes independently host-authorized exact non-loopback bindings, but not physical LAN/Internet deployment. B5 and B6 remain separate gates; this does not declare S4_B_PASS, S4_B6_PASS, merge or release.

## Immutable code and evidence

- PR [#153](https://github.com/vanishs/boxweave/pull/153), exact production and test source [de5657f4](https://github.com/vanishs/boxweave/commit/de5657f43431f25b067c398fd700cd1281618dcf).
- [B4 disposable Incus 7.6 native workflow](https://github.com/vanishs/boxweave/actions/runs/44687656712), job [136125707703](https://github.com/vanishs/boxweave/actions/runs/44687656712/job/136125707703): SUCCESS.
- [Quality gate](https://github.com/vanishs/boxweave/actions/jobs/136125707207), [amd64 build](https://github.com/vanishs/boxweave/actions/jobs/136125803850), [arm64 build](https://github.com/vanishs/boxweave/actions/jobs/136125803896): SUCCESS on exact code.
- Disposable L0 Box df927bc9-957e-4eb4-b579-10aa9a57645a, work_ref s4b4-closure-20261119. Full package tests c96d8c4747f8b89e9bcb3f33ba8b5394, final targeted tests 7c8ae14ddba05d99ad05260a1c7f33a3, full make verify d4a6e9e7dd5a0340b9d5a5834dc2ceaa and exact remote worktree diff aa5b82126f9de92f85af8ec9943e8053 all guest exit 0. Box stop 179fbfe8941917c78ee5ee0824768008 and delete a7bc0c7fc70e998e32b92fabd875ad62 both SUCCEEDED.

## Native acceptance matrix

- Nonroot official Incus SDK original ETag create and real TCP/UDP payload, exact removal and packet refusal: PASS.
- Authenticated official SDK -> 21-tool HTTP MCP -> original SQLite operation/Worker -> real Incus: TCP/UDP loopback plus IPv4 and IPv6 non-loopback TCP/UDP, six cases PASS. Each checks genuine parallel competing create/delete keys, identical-key receipt replay, occupied endpoint conflicts, actual echo packet, deletion and refusal after removal.
- Missing confirmation, wrong address/port or wildcard binding: PASS (denied). Real pre-existing occupied host listener remains intact without false success: PASS.
- Actual worker-process SIGKILL at four original-operation cut points: create/delete native success before ACK, create/delete after durable ACK before finalization. Reopen the original SQLite file, recover UNKNOWN, replay original receipt without redispatch, refuse a fresh conflicting key, verify native state and real packet results. All four PASS; marker S4_B4_NATIVE_REAL_PROCESS_KILL_ACK_AND_POSTREAD_MATRIX=PASS.
- Native device and disposable guest/Project/ACL/bridge cleanup: PASS, marker S4_B4_NATIVE_SCOPED_TCP_UDP_SOURCE_PACKET_CLEANUP=PASS.
- Atomic no-dispatch settlement on queued cancellation and restart is covered by TestB4ForwardQueuedCancellationAndRestartRelease and the passing Box/full quality regression. No silent release of UNKNOWN claims.

## Scope and follow-up

The authorization contract is documented in [host-local forwarding authorization](../development/S4_B4_HOST_LOCAL_FORWARD_AUTHORIZATION_2026-11-19.md). Default deny for non-loopback; both explicit MCP confirmation and a host-local exact interface/protocol/IP/port allowlist are required. Exact interface address is checked before admission and native create. Removing approval does not automatically revoke a running listener; explicit authorized delete is required.

The native external addresses were on the disposable runner's isolated bridge. No actual physical LAN/public Internet exposure, live shared L0 configuration change, production Box adoption or general remote reachability claim is made. The controlled native test guest was test-seeded into the private ownership ledger. The crash cuts use a real worker subprocess and SQLite, not an assertion of full daemon/physical-host restart coverage.

The distinct S4-A quota/log-loss regression probe was updated for the private archive fallback, retaining the true unavailable negative. Its [fresh installed regression](https://github.com/vanishs/boxweave/actions/runs/44687656577) was still in progress at this B4 signoff; no acceptance of that separate gate is asserted. B5 dependency-aware delete/fault compensation and B6 whole-chain final fresh-source regression and independent verification remain next. Draft PR stays unmerged; no release or live installation.
