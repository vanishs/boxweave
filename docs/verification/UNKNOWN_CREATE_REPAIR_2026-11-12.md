# v1 native Create UNKNOWN: iterative root-cause and repair evidence, 2026-11-12

**Verdict:** installed failed-start recovery and exact-source disposable
Incus 7.6 CI PASS at 172115e; independent physical-host acceptance and
test-owned residual Delete remain BLOCKED/NOT SIGNED. Separate topic branch
`fix/v1-unknown-create-reconciliation-20261112` starts from PR #133's
`7e47c109c0da03ad50da6d5fb3e3fa6a06258f1d`. This work is **not**
a HOST_GO stage invocation, LAB_PASS, permission for a physical host or
permission to rewrite historical UNKNOWN receipts.

## Verified causes and bounded fix

1. `state.Worker` previously mapped every postdispatch verification error
   to `UNKNOWN / POSTCONDITION_UNVERIFIED`, even when Incus had a final
   failed native Operation. It now permits a typed proved terminal FAILED only
   after strictly verified native outcomes; any doubt stays UNKNOWN.
2. Native Create previously omitted a durable original admission identifier
   from the instance, so a crash before `native_operation_id`/UUID persistence
   could not tie the postcrash Box to the exact original request without
   unsafe name-only adoption. New Create config includes
   `user.boxweave.create_operation` equal to the original durable Operation ID.
   The restricted host-local create-inspect/challenge/bind path checks this
   marker, Project, NIC and UUID; binding remains `UNKNOWN` and locked.
3. Native Create can create a valid STOPPED Box even though the requested
   Guest `start=true` never reaches RUNNING. Instead of treating a stable
   exact-owned STOPPED residual as a success, the candidate classifies it
   as terminal FAILED with a retained owned UUID and explicit cleanup path.
   A second Create under a different key cannot claim the name while an
   exact owned Box exists. Native postread is bounded and **never redispatches**.
4. Generic postcondition failures are now split into bounded diagnostic
   classes (native final Operation, marker/NIC, egress, unstable postread,
   UUID journal), preserving uncertainty rather than inventing a result.

## Box-first real Incus results (non-release)

- **Isolated environment:** `bwracc-1106-v2`, outer nesting=false,
  disposable nested Incus 7.6 / Project `user-1002`, nonroot
  `boxweave` service user, isolated test SQLite and separate loopback
  MCP per candidate. Original `boxweave.service` remained active/unchanged.
  All test guests had native ingress/egress reject, v4/v6/MAC filtering.
- **First candidate `f1b3eec272ba868fd9bc2fa230d419e3b3d9b491`:**
  binary SHA-256 `644738b58365fe60103452bb7932836078bd6f757fc432f2a76c278d2d23f884`.
  Original Create `bdc783ea07e589f45cbbf82c14a4cc9f`,
  original native `464546a2-28fd-4d1a-8507-c5219df3b88d`,
  native STOPPED UUID `fe24e24c-8dfa-46f5-bf40-bca4fc11e57f`.
  Marker exactly matches original, but BoxWeave status remained
  `UNKNOWN / POSTCONDITION_UNVERIFIED`. **FAIL** of proposed fix.
- **Second candidate `bc9cc634384cae46711ddd022942d7009ff178c3`:**
  binary SHA-256 `74770c36fce67128f69162d883a56e3c4a2058bd6a8ab9d07f94cd5caf5a0ef2`.
  Original Create `c59501b9b5d9a21dbbc68898f3821f7f`,
  native `9a9c986a-bed9-45aa-9eff-5ebc5b9fd99e`,
  STOPPED UUID `89f6b076-6b4a-4b5a-adfb-d5ed7c557655`.
  Again `UNKNOWN / POSTCONDITION_UNVERIFIED`, marker exact. **FAIL**.
  Subsequent independent restricted-UID read-only native SDK observation of
  that UUID passed: `Project=user-1002`, `Stopped`, `internet=false`,
  correct original identity marker/NIC/CPU/memory/disk. Incus operation
  history had already expired (`Operation not found`); do not invent its
  precise final status. The actual Guest LXC log shows cgroup BPF/proc mount
  EPERM under the no-nesting outer Box, a genuine startup failure.
- **Third bounded-postread candidate:** source
  `ca6b3b110fb5083ad270bbe80578176291fed2d2`; focused and full
  source/race checks PASS. Attempted next installed Box service/test via
  connected execution was explicitly **blocked by tool safety check**,
  and read-only follow-up confirmed no `unknown-ca6b` checkout/SQLite or
  service had been created. Therefore this source has **NO live Box
  failed-start PASS**. Do not bypass this blocker or retry old originals.

Both new FAILED-tests-that-stayed-UNKNOWN are preserved with exact original
Operations and native objects, intentionally **not deleted, retried, adopted
or changed**; previous Round-1 two STOPPED/UNKNOWN and all six external gateway
UNKNOWN remain separately preserved. The old external gateway is a distinct
product/runtime and is not repaired by this v1 patch.

## Source/race negative tests

- Typed native final failure with double native absence -> original FAILED;
  indeterminate residual -> original UNKNOWN with lock and exact key replay.
- Failed Create with registered exact owned STOPPED UUID -> original FAILED
  and owned name cannot be re-created, but a new separately admitted exact
  Delete intent is eligible for independent checks.
- Exact original Create marker / expected Internet / UUID changes / foreign
  or mismatched identity, transient postread and unstable state simulated in
  bounded no-effect verifier tests.
- Missing native ACK followed by actual SQLite close/reopen + `Recover`:
  original UNKNOWN, no fabricated native ID, original idempotency still
  locked, local reviewed UUID binding available.
- Wrong reviewer, changed native observation, short reason, missing
  acknowledgment, forged/replayed challenge and malformed original intent:
  fail closed without native or SQLite ownership effects.
- Injected protected audit INSERT failure rolls back the ownership INSERT,
  UUID transition and event as one transaction; original UNKNOWN remains
  locked.
- Full `make verify` including all Go race tests passed on the isolated
  `boxweave-test-ser2` source worktree. Independent Actions and verified
  native post-fix acceptance remain required.

## New expired-native-Operation recovery boundary (candidate source)

A native Incus 7.6 Create Operation ID can expire after the dispatch is
accepted, while the **exact instance and original native marker** remain.
A native `GET /operations/<id>` **404 is never alone an outcome**.
The Create verifier now requires:

1. Original durable native Operation ID exists in SQLite, and Incus
   `GET` returns specifically a typed 404 (not a timeout/auth failure).
2. Twice read the restricted Project's native active Operation list,
   separated in time, and refuse if the original native ID is active,
   or if either list read fails.
3. Read the exact `user.boxweave.create_operation` original marker,
   current native UUID/Project and effective private NIC/egress;
   wait boundedly for a stable Guest Running/Stopped state and verify
   requested CPU, memory, root disk and nesting, with no Create/Start
   redispatch.
4. Re-read the original native active Operation list before a terminal
   receipt. Only verified requested postconditions can be SUCCEEDED;
   a requested Running guest still stable STOPPED may be FAILED
   **with its actual residual owned UUID recorded**. Any missing, changing
   or inconsistent proof retains UNKNOWN and the original lock.

Unit/race negatives cover active original `Running`/`Pending`,
conflicting native resource configuration, and operation-list status
classification. This is deliberately not considered proven against the
outer-nesting-disabled real failed-start case until a safety-approved
native acceptance of the new exact source exists.

## Added exact-ACK-loss native recovery acceptance probe (pending CI)

The independent one-use GitHub Actions acceptance now also verifies that
the **real lost-HTTP-ACK** native instance carries the original immutable
`user.boxweave.create_operation` marker. After Incus has no in-flight
Operations, it makes an SQLite online backup of the still-unbound UNKNOWN
journal into a separate `0700` service-owned fixture and uses a restricted
native SDK reader to verify the exact Project, original marker and UUID.
Wrong marker is explicitly denied. A one-use review challenge binds only
the **copied** original record; the original remains UNKNOWN and unbound,
the original key never redispatches, different-key Create remains blocked,
and the copied UNKNOWN is not relabeled as success. The existing CI
administrator then safely cleans the one-use native fixture only under
the original S2 acceptance contract. This new I+N+SQLite test is not a
protected root-local installed CLI acceptance, nor a success until Actions
reports the exact check's PASS.

## Exact Actions red results (fixed test harness, native follow-up pending)

- `5092f668` / [run 43324672518](https://github.com/vanishs/boxweave/actions/runs/43324672518):
  all source/race checks passed; native acceptance progressed through real
  create/exec/delete/ack-loss and many safety tests but was blocked by
  the old test-only overlap-overlay generator expecting the previous
  `worker.go` import layout. The protective fail-closed check itself
  worked, but its pinned source expectation needed updating.
- `7264d910` / [run 43325581720](https://github.com/vanishs/boxweave/actions/runs/43325581720):
  all source/race checks passed and the actual Incus one-use
  lost-ACK Create original native identity marker independently matched
  **PASS**. The new copied SQLite bind probe couldn't read the runner-owned
  private `ack-original.json` as the restricted service UID and failed
  before binding, correctly not bypassing filesystem permissions.
- Corrected harness explicitly copies just that checkpoint as `0600`
  owned by the verifier service account, ensures the copied ledger is
  private `0600`, and preserves strict original source-only import
  matching for the overlay. Until a **new exact-head** Actions succeeds,
  the original two runs remain FAILED and native create-binding is not
  accepted.

**Remaining gate:** obtain a safety-approved disposable real Incus test for
the bounded-postread source and prove positive/negative official SDK/SQLite/
native result plus independent verifier, then run same-head Actions. Until
then `UNKNOWN_CREATE_FIX=NATIVE_BLOCKED`, `HOST_GO=NOT_AUTHORIZED`;
do not promote this branch as release-ready.

## 2026-11-12 continuation: real Incus terminal failed-start fixed

- Source 381121b failed on isolated bwracc-1106-v2: one original
  Create 8701e9e2fdba924894891c5679b173fd, native operation
  20e97752-3183-481e-be1f-00835355114b, STOPPED native UUID
  f8194b2a-ad2c-4fa8-b800-b36284b8a40d. SQLite remained
  UNKNOWN/NATIVE_CREATE_FAILED_STATE_UNVERIFIED despite the exact
  original create marker. This original is preserved and never retried.
- Root cause: Incus may briefly report Running for a guest that then
  stops after a terminal failed Create; the verifier accepted the
  instantaneous Running state, then rejected terminal native failure.
  Fix: on confirmed native failed Create require two consistent, spaced
  STOPPED readbacks of the same exact UUID, original admission marker
  and safe native configuration; reject any UUID drift. A guest that
  stays Running or has conflicting evidence remains UNKNOWN.
  Positive and negative unit tests and full make verify/race passed.
- Fixed exact source 172115eab8379c391a9b5fc0ac6d7d347f41fba6,
  binary SHA256
  768337e0d9e8a92a818716724d403378d237aa96da28f4f85c6d80cd9c4edccd.
  One original SDK MCP Create f36b99af9e748eff20de243d9dd086ff,
  native operation de006402-7abe-400a-9e6e-f6013eac659e,
  created STOPPED instance bw-unk-1721-start-1112 native UUID
  f76b7655-97ae-412a-8e94-2b5d07a84e30. Native original
  marker matched; final SQLite result was
  FAILED/NATIVE_CREATE_FAILED_OWNED_STOPPED with the original
  native operation ID and exact UUID atomically registered OWNED.
  Event journal recorded UUID confirmation and terminal failure.
- Separate-process official Go MCP SDK operation_get and box_get
  passed for this failed-start receipt and STOPPED UUID. Same original
  idempotency key replay returned duplicate=true and the unchanged
  FAILED receipt, with one original operation and one native instance.
  Restart of only the isolated test service preserved both answers;
  an initial immediate connect before listener startup was transient.
  Main boxweave.service stayed active and unmodified.
- This is REAL SOURCE+INSTALLED MCP+NATIVE scoped PASS for terminal
  failed-start only, with a same-host independent reader, not a
  distinct-host verifier. The outer no-nesting EPERM remains a genuine
  environment failure, not modified. The older eight UNKNOWN
  (six external gateway + two earlier native) and this additional
  381121b UNKNOWN are retained, not replayed/adopted/deleted.
  Exact-head broad Actions / lost-ACK proof and formal HOST_GO
  remain pending; PR #133 stays Draft.

## Independent Actions, binary reproducibility, and retained cleanup gate

- Exact source 172115e Actions run 43329153954 SUCCESS. Real one-use
  Incus 7.6 MCP blackbox PASS, including original lost-ACK Create
  marker/UUID, copied SQLite UNKNOWN binding with no retry, immutable
  original journal and lock, injected ACK loss restart, native teardown,
  source format/vet and full race. Log markers:
  S2_NATIVE_CREATE_REAL_LOST_ACK_MARKER_UUID_COPIED_SQLITE_BIND_NO_RETRY=PASS,
  S2_NATIVE_CREATE_REAL_LOST_ACK_BIND_ORIGINAL_REPLAY_AND_LOCK=PASS,
  S2_NATIVE_SDK_MCP_BLACKBOX_REAL_INCUS=PASS.
  https://github.com/vanishs/boxweave/actions/runs/43329153954
- Historical 381121b independent Actions run 43328408896 also SUCCESS
  for true lost-ACK recovery. That source had the native failed-start
  bug reproduced locally, so its green Actions do NOT replace the
  later 172115e installed failed-start proof.
- Independent second Box regular clone with Go 1.26.9, identical
  compile flags and source 172115e reproduced executable SHA256
  768337e0d9e8a92a818716724d403378d237aa96da28f4f85c6d80cd9c4edccd
  exactly. An initial Git worktree build differed because the Go
  linker did not automatically stamp vcs.revision metadata into
  that worktree; it was not claimed binary-identical. The ordinary
  clone positive control proved same-byte reproducibility.
- A one-use normal v1 box_delete for the known FAILED/owned
  bw-unk-1721-start-1112 UUID was proposed but the connected
  tool safety check explicitly blocked the full command BEFORE
  dispatch. No cleanup intent file or new Delete Operation exists.
  Independent read-only follow-up confirmed the original FAILED
  journal, UUID and STOPPED native guest unchanged; the original
  controller service remained active. No alternate-path bypass,
  manual native delete or original Create retry was attempted.
  The older native UNKNOWN and six external gateway UNKNOWN
  remain locked and untouched.
- The isolated temporary test controller services for earlier
  bc9cc and 381121b candidates were stopped without touching
  their separate protected ledgers or native residuals.
  The latest 172115e temporary service remains a separate
  non-production controller for original receipt readback.
- **Result classification:** FAILED-START_UNKNOWN_REDUCTION=PASS,
  REAL_LOST_ACK_ORIGINAL_BOUND_COPY=PASS,
  SAME_SOURCE_CI=PASS, SAME_BINARY_TWO_BOX=PASS,
  REAL_RESIDUAL_DELETE=TOOL_SAFETY_BLOCKED,
  HOST_LOCAL_MANUAL_RECOVERY_INSTALLED=UNVERIFIED,
  HOST_GO=NOT_AUTHORIZED. Original UNKNOWN receipts are never
  automatically rewritten or replayed, and v1 code does not
  retroactively repair older external-gateway UNKNOWNs.
