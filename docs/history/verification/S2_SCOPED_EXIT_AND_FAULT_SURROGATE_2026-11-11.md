# S2 scoped exit — operator-selected crash surrogate and 19-group original evidence

**Independent accepted proof (2026-11-11 PT):** real exact-source `f7828104c725480c016613073db591bf95c5bd9a` [full Incus 7.6 native CI](https://github.com/vanishs/boxweave/actions/runs/43131566410/job/130896688166) 5/5 PASS and independent byte-for-byte rebuild. 19-group scoped verifier reports 18 PASS, 1 S4 defer. Both P1 positive Ready and simulated routed TLS WAN on/off/on with server-side 2-hit witness are PASS. This does **not** change the real public guest WAN status (UNVERIFIED) or physical cut (not tested); see [signoff decision and exact exceptions](S2_FINAL_EXIT_SIGNOFF_2026-11-11.md). The pending descriptions below were written before this successful run.

**Decision, 2026-11-11:** For S2 development-phase acceptance, substitute **actual process SIGKILL at SQLite WAL fsync before AND after**, exact original native-effect proof, restart/no replay and actual ext4 ENOSPC for **physical power interruption**. The replacement has specific limits: it does not reproduce a dead disk controller, cut host power, firmware cache or electrical loss. Do **not** write "physical power loss PASS".

This is a scoped candidate gate against Draft PR #130, **not** S5 release, S2.5 full host pilot, S4 network-dependent Delete or all original A-case scopes. It does not authorize production/shared Incus, force unlocking any historical UNKNOWN, or merging `v1`.

**Additional real actor isolation requirement:** Two different authenticated admin tokens intentionally reuse the SAME untrusted idempotency key on the same owned Box, and must produce distinct immutable original Operations, while exact same-token retry returns each actor's own receipt and original native stdout. This `S2_NATIVE_SAME_KEY_DISTINCT_ACTOR_CANNOT_BORROW_ORIGINAL=PASS` marker is now mandatory for TC-004 and TC-006, not merely inferred from different-key tests. The source-only probe is accepted only after actual independent native CI.

**P1 positive readiness and routed HTTPS simulation (source-exact native reacceptance pending):** BoxWeave now recognizes ONLY guest-originated Incus Ready (actual /dev/incus PATCH, volatile.last_state.ready), never merely Running or a user config label; independent real SDK checks Ready=true, Stopped=false and after restart Running=null. An independent HTTPS endpoint lives in a dedicated CI network namespace reached through a separately routed veth on 198.18.0.2, with verified test-local TLS and a fsynced HTTP server hit log. Guest SDK drives HTTPS ON→OFF→ON and independent server confirms exactly two hits; this is **SIMULATED_WAN_PASS, NOT actual public Internet PASS**. The separate public-hosted WAN positive is still UNVERIFIED. Do not promote these outcomes until the exact-source original Incus 7.6 CI succeeds; neither should modify a shared host or any historical UNKNOWN.

## Automated evidence acceptance

- `scripts/s2_p3_scoped_acceptance.json` maps all original **TC-001–010 (10) and S2 grouped A-case rows (9)** to specific named native PASS evidence, with a written residual limit for **each**. No row may pass without a real source-exact marker.
- `scripts/s2_p3_scoped_acceptance.py` validates native candidate's **unique exact source SHA and unique original unchanged binary digest**, completed native job, per-row evidence, authorized fault surrogate, Guest Ready tri-state, and refuses forged WAN/hardware assertions. It returns **18 PASS_TESTED_S2_SCOPE** + **1 DEFERRED_S4**, not 19 universal PASS or a release tag.
- `scripts/test_s2_p3_scoped_acceptance.py` deliberately removes every row's native marker, duplicates source/digests, forges WAN/hardware, drops required crash evidence, and changes the S4 scope, confirming the verifier fails. The tests run **before expensive native provisioning**, and the read-only verifier runs after native blackbox.
- [Original unmodified S2 crosswalk](S2_EXIT_CROSSWALK_2026-11-10.md) remains linked with its source-level PARTIAL annotations. Those annotations must not be erased or auto-converted to broader feature claims.
- Current exact native source is determined **at CI runtime** and must finish all checks; a green historical job is not evidence for an untested future commit.

## Explicit S2 supported contract vs deferred risk

| Area | S2 acceptance claim | Important non-claim |
|---|---|---|
| Native exact Create/Delete overlap | Test-only Go overlay held actual UUID-bound VERIFYING. 12 distinct MCP Delete denials, original Create completion and legitimate Delete, independent UUID/SQLite and zero residual | Overlay service is not the shipped binary; unmodified release candidate is independently source/race and full native accepted |
| Recovery and power-fault class | SIGKILL at real WAL fsync both sides; native Update accepted, same original ID/UUID readback, restart UNKNOWN/no replay, independent ext4 quota ENOSPC | No physical AC power, kernel block layer or remote RAID power controller failure was tested |
| Guest Ready | Genuine current native guest-origin Incus Ready → `true`; Stopped → `false`; unprobed Running → `null`; stop/restart clears stale true | Does not assert process/application health or infer positive from Running |
| Internet | Same guest reaches independent routed namespace verified HTTPS with Internet ON, denied when OFF, restored when ON; server independently witnesses exactly 2 requests, plus native IPv4/IPv6/peer protection | **SIMULATED WAN**, not real Internet; public guest HTTPS remains UNVERIFIED environment exception |
| Guest Exec and cancellation | Native recorded stdout/stderr, 16MiB bounds, malformed metadata yields UNKNOWN, hard quota original, asynchronous lifecycle, queued predispatch cancel | Detached native Exec cancellation after control socket loss explicitly UNSUPPORTED |
| Storage | Operator-preselected restricted Project ext4 quota root pool, exact UUID/10GiB, safe rejection of absent quota/foreign pool | No public arbitrary `root_pool` parameter; no native volume migration |
| Network dependencies | S2 stopped no-dependent Box deletion and exact resource lock | Forward/peer dependent Delete A-043–047 and changed peer IP topology explicitly S4 |
| SDK compatibility | Current pinned SDK and actual older v1.6.1 native full lifecycle | Not every unknown older/newer MCP SDK version |
| Release | Exact source, binary SHA256, independent native transcript and failed-negative verifier | Not a production host install, final v1 merge, S5 release or full hardware certification |

## Required phase-exit procedure

1. Run focused syntax, Python verifier negative controls, Go vet + race in the already-authorized `bwracc-1106-v2` fastlab. It is NOT the native Incus 7.6 witness.
2. Freeze one source SHA; run fully isolated GitHub-hosted native Incus 7.6 suite. Verify real original candidate binary hash and 19 named evidence rows **after** full original suite.
3. If a named supported marker fails, refuse scoped exit, investigate as product vs harness vs environment and preserve original immutable receipt. Two consecutive same-class harness failures can be recorded and moved aside, **but cannot be silently counted PASS**.
4. A distinct reviewer checks 19-row JSON, original TC/A crosswalk, physical-fault surrogate and environmental exception, and makes the final phase signoff decision. Keep Draft PR #130 and protected v1 unchanged until explicitly authorized.
5. Preserve these three unrelated UNKNOWN originals without retry, adopt, delete or force unlock: `7e1cbf043d1fd4ea3d9ec0093a04371e`, `f811afeeb778c422fe99b9881baa40f2`, `192a6601c54078f8eaf1437a6a3dc13b`.

**Scoped acceptance is conditional, not blanket phase completion.** No physical outage, real hosted public WAN, future S4 Delete or S5 publication should be invented simply to label all original 19 rows PASS.
