# S2 scope lock and one-pass exit plan — 2026-11-11

## Status and hard boundary

**Current development source:** test-only deterministic post-native-UUID-bind Go overlay on [Draft PR #130](https://github.com/vanishs/boxweave/pull/130). Do **not** call the new overlap PASS before this exact source's disposable Incus suite succeeds. The last fully passed unmodified Incus candidate preceding this change is source `3dc712413705257b9992c13a339cf48bb46c6bf8`, binary SHA256 `2cd6a632831cc3244d6aff4214370e00c8c7769f82633962b5014249dfe0fad0` ([native proof](https://github.com/vanishs/boxweave/actions/runs/43113771010/job/130836335122); all five checks SUCCESS).

The crosswalk has **10 TC rows and 9 grouped A-case rows**: TC-010 PASS, 17 grouped rows PARTIAL/BLOCKED and one A-043–047 S4 dependency. **17 is a count of incomplete grouped signoffs, not 17 broken features.** The new scope lock does not silently change their original status.

## Nine work packages and explicit exit classification

| Work package | Priority | Minimum S2 outcome | Remaining check or decision |
|---|---|---|---|
| C1 Native Create→bound VERIFYING→parallel Delete | **P1 REQUIRED** | Real official SDK + native UUID/SQLite identity; different-key/actor Deletes fail resource lock; original Create completes exactly once, then one legitimate Delete succeeds | Source-only 24-way race and native final-receipt UNKNOWN already PASS. Newly added test-only **Go build overlay** pauses immediately after actual RegisterCreatedBox committed and verify returned, with the production source and product binary unchanged; require actual hosted CI evidence, do not promote standalone shim binary into release |
| C2 Guest readiness | **P1 CONTRACT** | Do not promise `guest_ready=true` from Incus Running alone. Explicit `UNVERIFIED/UNSUPPORTED` until a first-class bounded Guest Agent probe is both implemented and live-tested | A native `incus exec /bin/true` positive witness exists; neither implies a generally true public field. Owner must approve an expanded Guest Ready capability separately if S2 requires true |
| C3 Actor/work recovery and original no-retry | **P1 REQUIRED** | Immutable ID, native UUID/ack, wrong actor/second key denial, crash/restart and safe host-local review for every supported S2 mutation | Cross-token overlapping relay and per-action post-fault review not entirely signed; use a finite table of already-passed native/Go proofs, add missing targeted cases only |
| C4 MCP SDK/identity compatibility | **P2 SUPPORT BOUNDARY** | Pinned current official SDK + separately tested v1.6.1 compatibility; two OS identities and Projects; no claim all historical releases | Additional SDK fuzz is optional hardening, not an infinite mandatory version matrix |
| C5 Guest Exec I/O negatives | **P2 REQUIRED** | Real nonzero exit, malformed return UNKNOWN, partial quota output and bounded truncation, stalled native output/caller cancel, physical log missing/retention, timeout | Existing real native cases PASS; additional arbitrary OS/transport failure patterns should be a bounded fuzz suite |
| C6 Input, auth and leakage | **P2 REQUIRED** | 13 closed Tool schemas, nested argument rejection, viewer/foreign denial, correct private cache headers, no token/guest output leak in errors | Top-level and real read revoke+expire PASS; nested/model-error fuzz and complete redaction inventory need bounded signoff |
| C7 Public WAN | **ENVIRONMENTAL P2** | Real public TLS+2xx positive before/after, same URL denied while internet off | Hosted native gateway/IPv4+IPv6 on/off/on and peer isolation PASS; external guest HTTPS remained **UNVERIFIED** across four endpoints. Never make a pass claim without positive. Choose an authorized guest-WAN-capable lab to close |
| C8 Physical power/storage-controller interruption | **ENVIRONMENTAL P2** | Independent real power cycle plus original WAL/native readback | No authorized physical lab; `SIGKILL`/fsync/ext4 ENOSPC PASS **not equivalent**; marked **BLOCKED**, do not run on shared/production |
| C9 Independent final TC/A verifier | **P3 REQUIRED** | Exact final source, executable SHA256, quality and full race, official MCP native suite, distinct verifier report mapping each of 19 grouped rows to evidence/UNSUPPORTED/BLOCKED | Cannot upgrade a PARTIAL row solely because aggregate Actions is green. S4 A-043–047 explicitly deferred |

## Acceptance run plan and stop conditions

1. **Fastlab**: on existing authorized `bwracc-1106-v2`, Go1.26.9 CGO `go vet ./...`, `go test -race -count=1 ./...`; compile all test probes and test-only overlay with no native mutation. Do not use this legacy backend as native 7.6 evidence.
2. **Single native CI build per frozen checkpoint**: build the unmodified release candidate and its exact SHA, separately compile only the overlapped worker as a disposable acceptance binary. Run normal S2 lifecycle, fault recovery, P1 overlap and optional WAN positive control. The overlay has **no released product hook** and its proven result is labeled *test-only timing injection plus unmodified source/race*.
3. **Verifier**: inspect exact native UUID/Project/SQLite journal plus all original failing and passing job logs; classify product failures independently of fixture failures. Keep root/manual UNKNOWN dual confirmation and immutable ID boundaries.
4. **Stop after two same-class harness/environment failures**, preserve the original evidence and mark that scenario `UNVERIFIED_TESTER` or `BLOCKED`; continue other tests. Never alter native quota, auth or no-retry safety merely for green CI.
5. **Exit**: close only actual S2 mandatory contracts. Unsupported affirmative Guest Ready/live detached Exec cancel/custom root pool selection stay explicit; S4 dependent deletion stays S4; blocked physical power/WAN requires written exception/authorization, not a fabricated PASS. PR #130 stays Draft and protected `v1` unmerged until a separately authorized final decision.

## Original UNKNOWNs — preserve, never reproduce

- `7e1cbf043d1fd4ea3d9ec0093a04371e` — prior S1 original Create.
- `f811afeeb778c422fe99b9881baa40f2` — prior second isolated original.
- `192a6601c54078f8eaf1437a6a3dc13b` — extra attempted fastlab original `bw-v1-s2-fastlab-1111`, still UNKNOWN.

No same/different-key retry, unlock, adoption or opportunistic cleanup of any of these original operations. All new test fixtures are separately created within disposable GitHub-hosted Incus.
