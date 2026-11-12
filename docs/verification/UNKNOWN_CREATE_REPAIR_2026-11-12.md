# v1 native Create UNKNOWN: iterative root-cause and repair evidence, 2026-11-12

**Verdict:** FIX IN PROGRESS / NOT NATIVE-ACCEPTED. Separate topic branch
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

**Remaining gate:** obtain a safety-approved disposable real Incus test for
the bounded-postread source and prove positive/negative official SDK/SQLite/
native result plus independent verifier, then run same-head Actions. Until
then `UNKNOWN_CREATE_FIX=NATIVE_BLOCKED`, `HOST_GO=NOT_AUTHORIZED`;
do not promote this branch as release-ready.
