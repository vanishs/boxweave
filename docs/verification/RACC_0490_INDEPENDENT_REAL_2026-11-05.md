# v0.49.0 real independent pre-release test evidence

**State: PARTIAL / OFFICIAL RELEASE BLOCKED.**
This report documents actual isolated execution, not full RACC-SYS-01..09
completion or a trusted public release signer.

## Finder Box, in-place upgrade and real operations

Finder environment: bwracc-mvp-verify, outer UUID
b95d62fe-6914-41e2-bbc7-8c6d122a844c, createdAt 1793931912.

- Official v0.47.0 installed. Host doctor --mcp 10/10 PASS.
- A real nested selftest created a Box, executed guest hello,
  stopped and deleted the child with a native zero-residue oracle.
- A PR CI candidate full host+Gateway+app in-place upgrade from v0.47.0
  to v0.49.0 succeeded using the new digest-pinned private host installer,
  with host/application source c2e1cf3b8686e8df23fb736d10ff7aac711ecb2b
  and doctor --mcp 10/10 PASS. Source is a PR synthetic merge, not main.
- The source-only SYS-LIFE-01 limited 4-operation path passed actual
  create -> native Running -> stop -> Stopped -> start -> Running ->
  delete -> native absent on UUID 460d6572-30c8-41f4-8175-4b70a4e21670.
  **Status PARTIAL**: graph resize, guest connect and cross-entry auth
  were not tested.
- Separate test-owned bwracc-life-49b creation got no acknowledged
  response within original 75s, but native Provider had created
  UUID 7c11f1c0-f58f-49f4-bd62-a2f5d429bd30. The exact target
  remains **UNKNOWN** with no automatic retry or deletion.
- Genuine two-process selftest contention passed: first
  boxweave-selftest-f26843 succeeded, simultaneous
  boxweave-selftest-d5bd2f rejected (exit 2), subsequent
  boxweave-selftest-603486 succeeded (exit 0), residue 0.
  This is real concurrency, but is still one agent in one Box.

## Fresh separate Verifier Box

Distinct new outer Box bwracc-mvp-iv3, UUID
06a0922a-4ef9-450e-9561-7bd15c9422db, createdAt 1793939346.
A clean private candidate installation was verified using GitHub PR CI
run 42159807508, artifact ID 13475989731 and wrapper SHA-256
71f5bb1076e233440b290a68dd427d58a947607f6d919cb3ddb017afe82be266.

- Exact inner BoxWeave 0.49.0 ZIP SHA-256:
  44f0bbd0924c8d2bdc33cb6e165929fd74614ef01598deff9be38de3a3d780c2.
- Exact manifest SHA-256:
  aa1986f485e0750aeb0cc4de86c4a0f13927edbe0f5691dac4a8f9ea4b011204.
- Source SHA 839d00400d9af759b9837b133dd0ee4761a7ed9e
  is a PR synthetic merge commit, **not** an official main release.
- 23 pinned root-owned assets plus release.json were checked before
  full first-time installation. The observed Host CLI, app provenance,
  original ZIP SHA and all ten authenticated doctor checks matched.
- Independent lifecycle and selftest evidence will remain PENDING unless
  separately recorded from their actual completed commands.

The new developer-only installed_verifier.py checks exact CI candidate,
installed Host/Gateway/app identity, ten doctor checks and conservative
scenario proof completeness. It emits PARTIAL, not a release signature.

## Publication blockers, still mandatory

1. Separate real Finder and Verifier complete mandatory-case proofs:
   SYS-JOB, UNKNOWN, SELFTEST, UPGRADE, RELEASE, REPORT, plus full LIFE
   graph or separately scoped MVP cases. Missing nodes remain BLOCKED.
2. Signed external independent Verifier proof producer and protected
   trusted GitHub environment public key are still unconfigured/unproven.
3. After merge, a **new main commit** must have its exact CI ZIP
   installed, checked and signed. PR synthetic merge evidence is not
   sufficient for a public tag.
4. No automatic multi-Agent spawn, complete 19 scenario graph or full
   mutating MCP/CLI/HTTP/Web cross-interface black-box coverage is claimed.
5. Original test-owned UNKNOWN child remains frozen for explicitly
   reviewed original-incarnation recovery. Owner R-02 host reboot deferred.

No public Release, no stable pin change while any mandatory gate fails.


## Independent Verifier's first actual MCP create: post-dispatch UNKNOWN

The fresh separately installed Verifier Box attempted SYS-LIFE-01 against
its own unique bwracc-iv3-life-a-container, using the correct installed
Gateway as the boxweave application account. Even with a 240-second cold
create budget, the stdio process provided an initialize response only and
its create call ended in execution UNKNOWN. However independent native
Incus shows the test-only child Running, created
2026-11-06T04:40:07.511640597Z, exact UUID
95328612-e76c-418d-b70a-c9c7874ce656. A subsequent read-only
child account lookup returned UID 1001, sshd active, TCP 22 reachable;
these observations do **not** establish that the lost MCP create reply
would eventually have acknowledged success. The original Verifier call
was NOT replayed and no automatic delete or further write was performed.
The independent lifecycle is therefore **UNKNOWN, not PASS**. This is a
high-priority real installation/Provider timeout and receipt investigation,
not proof that the 4-operation lifecycle works on every fresh host.

Read-only boxweave-provider journal confirms an upstream create operation,
account setup, and SSH collector exit-code-2 messages within the observed
window. Root cause (upstream operation latency, post-create SSH collector or
Gateway stdio completion) remains **UNCONFIRMED**; investigate before
shipping a gate that treats timeout as deletion/no-side-effect.

New Verifier Box provisioning also encountered two independent upstream
quirks: bwracc-mvp-iv2 existed but its outer host SSH account was not
created; bwracc-mvp-iv3 briefly lacked the guest user, then recovered
through its normal asynchronous provisioning. The connector correctly
refused SSH with TRUST_NOT_READY. Do not bypass that protection or count
those as a passing clean Verifier until actual access is restored.

## Developer-only trusted pre-execution installer hardening

New tools/reacceptance/private_installer.py runs from an independently
reviewed root-owned developer checkout and checks an out-of-band candidate
manifest digest, source commit, root-only parent chain, exact fixed asset
list, each asset including candidate online.py, and product ZIP embedded
provenance *before* allowing candidate script execution.
Without an explicit exact test-owned grant plus confirmation it is a
preflight-only reader. Authorized apply writes durable dispatch intent before
a bounded host install, preserves post-dispatch UNKNOWN without replay,
and still requires separate installed Verifier evidence.
Four adversarial unit tests pass. An actual installation through this
new wrapper is not yet marked complete: earlier isolated clean/upgrade
installations used candidate online.py's internal verification directly.

## Confirmed source harness bug: 240-second create budget silently capped at 120

Read-only inspection of the exact Verifier attempt file proves
`timeout_seconds=240` was only the lifecycle layer's **requested** value;
actual `process.elapsed_ms=120102` and `execution=UNKNOWN`. The older
Finder small-child successful create also returned after 119574 ms, close
to the same hidden cap. The shared `tools/reacceptance/engine.py:capture`
forced `timeout=min(max(timeout,1),120)`, so `mcp_session(timeout=240)`
never received the advertised window. **This is a confirmed tester Bug.**
The new code preserves the 120-second default for ordinary probes, permits
only the reviewed MCP lane to opt into a hard 300-second ceiling, and checks
that the actual subprocess timeout for normal cold create is exactly 240.
This fixes the test budget, but does not prove the upstream Provider and
Gateway can return an acknowledgment within any bound, nor reclassify the
original uncertain 49b/iv3-a writes as SUCCESS. They remain frozen.

## 2026-11-06 independent fresh-target lifecycle after correcting test timeout

This is the **latest** Verifier result; it supersedes only the earlier
"independent lifecycle remains UNKNOWN" conclusion for **a newly owned target**.
The original bwracc-iv3-life-a UNKNOWN and Finder 49b UNKNOWN are not
reclassified, removed, replayed or silently unlocked.

- Original independent new child: bwracc-iv3-life-b-container.
- Native incarnation: 053bc35b-565d-4707-adcf-cfda67565741, created
  2026-11-06T05:25:10.610888156Z.
- Received real MCP success receipts for create (119517 ms), stop
  (1397 ms), start (516 ms) and delete (4348 ms).
- The separate native Incus oracle saw exact UUID Running, then Stopped,
  then Running, then absent. Only the new child was deleted.
- The new execution was launched from source 23670d3e which corrects the
  hidden 120-second timeout clamp; actual subprocess ceiling is now 240
  for cold create, bounded by 300.
- Fresh independent installed Verifier report confirms Host/App source
  839d00400d9af759b9837b133dd0ee4761a7ed9e, CI ZIP
  44f0bbd0924c8d2bdc33cb6e165929fd74614ef01598deff9be38de3a3d780c2,
  doctor checks 10/10, and SYS-LIFE-01 status **PARTIAL**, since only
  four lifecycle nodes were covered, not full graph.
- SYS-JOB-01, SYS-UNKNOWN-01, SYS-UPGRADE-01, SYS-REPORT-01 and
  SYS-SELFTEST-01 remain BLOCKED; SYS-RELEASE-01 stays PARTIAL.
  The trusted external signing gate is still DENIED, not approved.
- GitHub workflow run 42168214749 for 23670d3e succeeded in its
  verify job; publishing was skipped as expected. Full new Python suite:
  **859/859 PASS**, code-first registry and 233 knowledge documents PASS.

**Important source boundary:** both genuine isolated installs were from
PR synthetic merge CI builds, not the final main commit. This is convincing
partial independent *behavior* evidence, but cannot authorize GitHub v0.49.0
Release without full mandated scenarios and signed exact-main verifier proof.

## Read-only exact-identity UNKNOWN follow-up adapter

tools/reacceptance/unknown_inspect.py, developer-only, checks a fresh
test-owned grant and immutable original lifecycle request/attempt/native
oracle/result files. It independently re-reads the native child UUID,
created_at and status without any MCP write, native delete, retry or lock
release. It writes an append-only new observer report, treating a matching
child as SAME_ORIGINAL_SIDE_EFFECT_EXISTS_STILL_UNKNOWN, changed incarnation
as INCARNATION_CHANGED_HARD_BLOCK, and absent child as
NOT_PRESENT_AT_LATER_READ_ORIGINAL_UNCERTAIN. In every case,
release_acceptance=false and cleanup_authorized=false. Five malicious
fixture tests (including forged success and foreign grant) pass.
This is evidence for Issue #88, not automatic UNKNOWN resolution.

## 2026-11-06 exact native UNKNOWN reconciliation on two isolated hosts

The new read-only unknown-inspect command was executed after a fresh,
exact test-owned grant on both original post-dispatch UNKNOWN cases.
It re-read original durable lab/intent/attempt/oracle/result files and native
Incus state; originals were NOT overwritten.

- Finder original bwracc-life-49b: recorded UUID
  7c11f1c0-f58f-49f4-bd62-a2f5d429bd30 equals current native UUID.
- Independent Verifier original bwracc-iv3-life-a: recorded UUID
  95328612-e76c-418d-b70a-c9c7874ce656 equals current native UUID.
- Both separate output packets classify
  SAME_ORIGINAL_SIDE_EFFECT_EXISTS_STILL_UNKNOWN,
  no replay performed, no cleanup authorized and no release acceptance.
- A changed native incarnation and later absence are handled as separate
  BLOCKED observations and never imply a successful originally acknowledged
  create or absence of earlier side effects.
- This is **read-only incident inspection**, not manual lock release.
  Do not delete, retry or start either original Box as part of a general
  automated reacceptance campaign. An authorized operator must inspect
  exact task/lease state and identity separately before recovery.

The actual data is on the originating Finder/Verifier Boxes in distinct
private evidence directories; this Markdown records only non-secret
observations and selected exact identity fields. Issue #88 remains open.
