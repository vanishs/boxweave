# PR #177 D4: evidence-bound reviewed recovery execution (2026-11-23 PT)

## Status and previous-stage acceptance

D3 is accepted on exact head 79596e77d09b7a1331c79aa8795ba5b927f8aae5.
All four independent CI workflows passed: quality 45349067199, source
45349067192, artifacts 45349067230 and installed lifecycle 45349067365.
Actual installed jobs 138290678662 (legacy) and 138290679044 (scoped)
were inspected: D1 mode/authority and no restart, D2 empty diagnostic
coverage, D3 absent-issue no-effect, scoped active/retired diagnosis,
degraded diagnosis and partial-original plan refusal all passed.
Scoped N4 normal Forward continuity 314 rounds and partial continuity
168 rounds passed. The D2 incorrect JSON field assertion had already
been corrected in D3; no additional D3 CI failure required repair.

D4 implementation and Box verification are complete for submission.
Independent CI on the final D4 head is required before D5 acceptance.
PR remains Draft; no v1 merge, release or physical L0 modification.

## Execution contract

- One binary, the existing doctor and recover CLI entries, unchanged
  MCP tool count/schema and existing recovery finalizers.
- Plan version 2 adds local apply availability only for a verified,
  supported REVIEW_READY plan obtained by root. A plan is not approval.
  Old version-1 plans cannot authorize execution.
- Explicit local approval, reviewer, independent durable recovery key,
  exact scope, issue ID and reviewed SHA-256 are required. Input is the
  exact reviewed JSON on stdin, not a caller-selected native command.
- Reobserve current installed identity and evidence; match the reviewed
  original, ledger, witness, leaf, mode and time window. A restricted
  worker with fixed scope/database/socket, root peer credential handoff,
  and independent live native evidence performs the existing recovery
  transaction. Revalidate inside the transaction. No generic Root
  dispatch or native replay.
- Commit receipt, original resolution and recovery audit atomically.
  Same exact reviewed key returns the committed receipt even after
  expiry; a different key with stale evidence is refused. Refusals
  create no recovery receipt. Do not create a missing source DB.
- Postcheck independently reads the receipt, audit, original and
  leaf-specific durable state. Native postcheck is explicitly
  NOT_REPROBED_AFTER_LEDGER_COMMIT; current packet proof is a separate
  observation. Binding an identity does not settle UNKNOWN.
- Partial/unproven native state, unsupported scoped Create/Delete,
  stale or altered review, changed history/claim/ledger, revoked
  authority and wrong scope remain blocked. No blanket unlock,
  compaction, forced retry, global repair, or legacy fallback.

## Local operator workflow

Use the actual stable issue ID and reviewed digest from the output.
Retain the exact plan bytes and use a distinct recovery key; do not
reuse an approval for a changed plan.

```sh
sudo boxweave doctor -project user-1001 > doctor.json
sudo boxweave recover --plan --issue "$issue" -project user-1001 > plan.json
sudo boxweave recover --apply --issue "$issue" -project user-1001 \
  --approve-local --reviewer "$reviewer" --recovery-key "$key" \
  --confirm-review-sha256 "$review_sha" < plan.json
```

Inspect the returned receipt/postcheck and run doctor again. A refusal
does not authorize a different repair. Existing authorized MCP recovery
remains available through its original narrow contract.

## Exact Box evidence

Source parent: 79596e77d09b7a1331c79aa8795ba5b927f8aae5.
Code candidate tree before documentation: a726c1b1d2d02c4c1974cf5bd05f5aaba9d49b26.
Development Box ac585d2c-37ae-4879-a2e6-3cecc338d4d7.

| Operation | Result |
| --- | --- |
| bb100248f572118adf3782039a60ba20 | Transaction refactor recovery tests, exit 0 |
| f4409551a1207fda3a7f0bc0fe5d2567 | CLI integration regressions, exit 0 |
| 688696c8e716e1614a73c9f75375e772 | Initial focused run, exit 1: service fixture directory permissions; product correctly refused non-0700 directory |
| 7836ce7be97915facd994b5e0d9d0341 | Corrected fixture, service tests three rounds, exit 0 |
| 31930da684b85a51e562deab0820fd4c | D1-D4 focused Race tests three rounds, formatting and script syntax, exit 0 |
| 24b1af9d3e49a1c76a244c959d210c61 | Full make verify, exit 0: format/YAML/module/vet, all Go Race packages, MCP 16, uninstall 42, stable gate 10, raw witness 7 |

The focused matrix covers positive scoped settlement, expired review,
wrong witness/leaf/mode, same-count history change, claim/ledger drift,
revocation, partial native proof, transaction abort, MCP injection,
same/different concurrent keys, lost ACK, identity binding retains
UNKNOWN, missing DB no-create, repeated live evidence, input bounds,
trailing input, altered review and non-root refusal.

Fresh nested installed Scoped Box febeb346-d924-40e0-83fc-99b4e984b16f
was created exclusively for D4. Official Go checksum, signed Incus
repository and fresh native bootstrap checked. No historical fault lab
or physical L0 configuration was changed.

| Operation | Result |
| --- | --- |
| affc4b58bbc5324e4eb98a6f44d62d41 | New isolated Box created |
| f2df24c6b7e4240fb96c312309f10fe5 | Dependencies and signed repository, exit 0 |
| 5dd942b4ed1bfe84e2a44093fccfde97 | Native bootstrap, image ready, exit 0 |
| 2a146fd4321c8b88577fb771fe5316b1 | Verified exact candidate tree, fresh scoped installed binary and units ready, exit 0 |
| 3db5d1bc668b1ae97150406b24abdd48 | Actual installed D3 positive plan and D4 reviewed recovery, exit 0 |

Real original e0da7458004eb4315f230a2a96a32b65: restricted actor
staged a Peer grant and Root executed it once; the fixture intentionally
omitted completion finalization, then restarted the controller to
produce UNKNOWN. Installed root doctor and plan obtained real live
evidence. Wrong digest, omitted approval, non-root and altered review
were refused without mutation. Apply settled the original without
native replay, returned one durable recovery receipt/audit, preserved
the original native ID and five historical claims and unchanged
journal bytes. Same reviewed key returned the same receipt; different
key was refused. Both IP families passed peer packet witness. Unrelated
Box TCP Forward remained continuous for 147 rounds; controller PID
was unchanged throughout recovery itself (the fixture restart occurred
before plan creation). Observed markers:

- PR177_D4_NATIVE_COMPLETED_ACK_LOST=PASS
- PR177_D3_INSTALLED_POSITIVE_PLAN=PASS SCOPED_V2
- PR177_D4_INSTALLED_REVIEWED_RECOVERY_NO_REPLAY=PASS SCOPED_V2 147

## Independent CI and D5 handoff

The installed lifecycle workflow now requires D3 positive planning
and D4 reviewed recovery in both legacy and scoped installations,
plus scoped partial apply refusal and existing N4 regression markers.
The legacy test stages a real native stop, preserves its ACK through
restart, verifies recovery and actual Stopped state, then removes
only dedicated test Boxes. Legacy installed D4 and the scoped partial
apply gate are independent CI acceptance items, not claimed as already
observed in the D4 Box. The scoped fresh positive test is observed
on the exact code candidate above. Physical Host parity, physical
reboot, public reachability and real 1024-instance scale remain
unproven; no scope expansion or security workaround is authorized.

Next Agent: check the exact D4 commit's four workflows and actual
installed job markers first. Fix any D4 failure in Box and resubmit
before D5. If green, perform D5 end-to-end integrated acceptance,
documentation and evidence review. Keep Draft and await Owner
authorization before merging. Do not redo accepted D0-D3 or replay
the completed native fixture.
