# S4-A formal scoped runtime acceptance (2026-11-16 PT)

**Independent verdict: S4_A_PASS for code `3407e2a25609e885756f2fe3446394967041d37c`.** Existing [Draft PR #153](https://github.com/vanishs/boxweave/pull/153) remains the sole development vehicle. This is S4-A acceptance, not S4-B/C, full S4 integration, physical HOST_GO or release authority. A later reporting/workflow head must complete its own checks; its final source, measured digest and run links are recorded in the PR verdict after those checks, never inferred from this earlier run.

## Exact accepted code and independent proof

- Source: `3407e2a25609e885756f2fe3446394967041d37c`.
- Actual amd64 controller SHA256: `6d8aa027926b64104ba8baa70bf69219ad5d008e6cfed15eb510fca0027a7cb7`.
- [Installed Incus 7.6 / restricted HTTP MCP / independent raw-terminal run 43947870241](https://github.com/vanishs/boxweave/actions/runs/43947870241): SUCCESS, completed 2026-11-16 20:26:27 UTC.
- Same-head [quality 43947884997](https://github.com/vanishs/boxweave/actions/runs/43947884997), [quality 43947870601](https://github.com/vanishs/boxweave/actions/runs/43947870601), [S1 43947884393](https://github.com/vanishs/boxweave/actions/runs/43947884393), [dual-arch 43947884495](https://github.com/vanishs/boxweave/actions/runs/43947884495), [dual-arch 43947870426](https://github.com/vanishs/boxweave/actions/runs/43947870426): all SUCCESS. Dual-arch means build verification, not arm64 native acceptance.
- Independent Verifier separately fetched job `133582211751` and checked source, actual native log, sixteen mandatory evidence markers, original IDs, final SQLite correlation, all same-head checks and absence of error annotations. The coordinator did not substitute its own source tests for this verdict.
- Actual candidate ran as restricted nonroot `boxweave` through `runuser`, official HTTP MCP, original SQLite WAL and project-confined Incus 7.6. The native event observer subscribed before effects, independently of the controller. This is not a new physical-host/systemd rollout or separate live LLM session.

## What was repaired in this continuation

The recovered source was `b3db44f5ff832f50a252908b2e0c5833224b163e`. Its log stopped visibly after ACK-loss inventory because the wrapper omitted downstream assertion output. Fresh controlled runs proved the native original had settled, its UUID and `user.boxweave.create_operation` matched, the SQLite copy bound safely, and the original stayed unbound UNKNOWN. No product Create-binding or Incus bug was established at that boundary.

The reproducible later failure was a test-only operation_list debugger boundary. Historical line149 moved into Tool registration after #148. Correcting it to source169 alone still failed: optimized Go inlined credential extraction and mapped the first PC to a branch skipped by valid HTTP token_id. The fix derives the unique actual privateReadReply CALL from the exact executable, checks source/callee/ELF/address uniqueness, and requires two real breakpoint hits for revoke/expiry. No product bytes or authorization checks were weakened.

Two older lifecycle harnesses also expected an impossible split UUID/VERIFYING state after S4 atomic finalization. They now pause after native verification but before the transaction, require twelve actual UUID Deletes to refuse without admissions, then verify atomic completion. A failed final-update trigger must roll back UUID, ownership and terminal event together; identity binding is exercised only on a standalone SQLite copy, with original UNKNOWN/audit unchanged.

Additional test changes cover installed OAuth negatives, twelve concurrent same-key Deletes with post-tombstone replay, individual raw Create/126/127 terminals, raw Delete and final SQL correlation. Fixed evidence prefixes stream live; private raw errors and credentials remain runner-local. Product OAuth, WAL, worker and lifecycle Go files were not refactored during this continuation.

## Actual original receipts

Main fixture: Project `user-1002`, Box UUID `3e1a412d-95a1-4f48-9027-79159dd259ab`.

| Case | Original BoxWeave ID | Original Incus ID | Independently checked result |
|---|---|---|---|
| Create | `75bf152cd551744212f643be94169baa` | `e330d18c-f8c6-485b-b84a-d0a163e1e3af` | Raw200; final SQL SUCCEEDED, exact Project/UUID/native ID |
| Exec126 | `90eb642fde9a12c35b3e0c1318de76c0` | `7623d9bf-f646-4f11-aa1b-e47e6d6519c0` | Raw400; final SQL SUCCEEDED, persisted Guest exit126 and original outputs |
| Exec127 | `5f814e3c14afb3870c6c55e46461f5d5` | `0bbad2d2-0535-47bf-81d2-7e274ba09265` | Raw400; final SQL SUCCEEDED, persisted Guest exit127 and original outputs |
| OAuth A-to-B original Exec | `444e10883758b290c91f9025b16bc341` | `483638e5-54d5-4e7f-8a3a-e73cd057522f` | Same original after refresh; independent Guest marker count1; captured raw terminal |
| Concurrent Delete original | `7fd1e95a39d99a4c0b09a793afe51c1e` | `cf3ae93d-005b-43a8-a16f-f7967e54c1be` | Raw200/error=false; one SQL original/event; DELETED tombstone by this ID; absence and same-key replay |
| Pre-fsync SIGKILL Exec | `6587c5c5df95ac112bf07728e8a23b23` | `f40e8cc9-2723-44a2-a792-836bfe69c3d8` | Real service kill, atomic reopen/result crosscheck and one Guest effect |
| Post-fsync ACK-loss Exec | `9e60b40f77723fe95772304bb1da97cf` | `a7580ac8-afa5-46f5-a7e7-3eb53d2aae32` | Reopened SUCCEEDED, same native ID, one output row/event and one Guest effect |

Raw Exec400 is the verified Incus mapping for these controlled nonzero exits, not a fabricated exit0. The early raw Create snapshot had UUID pending; the later independent SQL/native check proved the exact final UUID above. SIGKILL/WAL evidence is not physical power-loss proof.

Independent Grant same-key harmless Exec: BW `ba75dae5a51c24f889a5c2fc68c08e32`, native `68796ff5-5d11-49a7-8317-c693ce4a51a4`. Manual-token same-key harmless Exec: BW `bdc16fffa16ccb7cfd1d380f636bc750`, native `284e9d77-a4dd-43b5-9828-307ec6534491`. Both have distinct actor/credential/native IDs and leave the original marker at one. Viewer history/Exec, old access, revoked manual token and revoked grant refusals add no admissions. Admin project-history reads retain their existing permission contract.

## Atomic failure, UNKNOWN and additional test-binary evidence

Actual-candidate final-transaction failure: BW `a6050a8ab1dfe86cff00aed518d8852b`, native `19cf2a82-9fe8-4ddc-a06e-8e6bb2bc1c1d`, Box UUID `8acc5176-014b-4fac-afff-e17835f59cd8`, Project `user-1002`. Native original marker/UUID/Stopped state were independently confirmed. Original SQL remained unbound UNKNOWN with no ownership or SUCCEEDED event; different-key mutations refused. Only a consistent standalone copy accepted the exact reviewed identity; forged/replayed challenges refused and original state/audit stayed unchanged.

The real native Create ACK-loss original separately stayed UNKNOWN with empty UUID/incarnation/native ID and zero owned rows before and after restart/copy review. Original Create POST count remained one. The test-only administrator removed only the exact disposable native fixture after independent settlement/UUID/marker checks; the controller did not retry or silently resolve its UNKNOWN. Historical/shared-host UNKNOWNs were untouched.

The pretransaction overlap test uses an explicitly separate acceptance binary with one deterministic pause. Unmodified lifecycle source SHA256 `a563b0f9d4bce54021480e5cd74a32b2b107c63808a2a15ea67f188f7678b802`; overlay source SHA256 `d84462a5570bbe870ca1e1819ede2402ae8e88b86705d175ce24cef2d86f1b1e`. Create BW `96bfdfebf18bd4bbf9c39a0b3d228d73`, native `842d700d-f2c5-4f5c-a5d8-8fee40a22f5a`, Box `1f214bcc-21ec-43a2-9f4e-c7996a35a861`; twelve SDK Deletes refused before atomic binding, then Create and legitimate Delete `a85af1019d343f7fca70ce234d7f9020` completed with tombstone. This extra test-binary evidence is not mislabeled as the main controller digest; main-candidate concurrent Delete has its own independent proof above.

## Quality, regressions and remaining scope

Complete local/pre-push make verify passed: tracked UTF-8/LF/JSON/whitespace/gofmt/YAML, mod tidy -diff, vet, module verification, five-package Race and Python witness7/7. Full build and focused independent probe/overlay builds, vet and wrong-source/changed-anchor/missing-or-duplicate-CALL negative calibrations passed. Existing task L1 RC7 Box also passed full quality at the measured earlier candidate; its deeper BPF/AppArmor Start restriction is not native S4 acceptance and was not bypassed.

Installed same-source S2 regressions,16/16 Tool discovery, P0 native/SQLite crosschecks, Stop/Delete accepted202 ACK loss/restart, token and operation_list in-flight revocation/expiry, two controllers, private reads, malformed Exec and alternate-pool checks all passed. Separate S3 File/Git installed and SFTP-native workflows were not rerun on3407; the reporting head enables manual execution on this exact S4 branch so final acceptance can also record those regressions without changing their pristine-runner or privilege guards.

No unresolved mandatory S4-A runtime blocker remains at accepted code3407. Final reporting-head reacceptance and the requested extra S3 regressions must finish before reporting that later HEAD as accepted. Full #151 causal/long-run provenance, #150 strong nesting, four Network Tools, dependent network cleanup, physical LAN, real public IPv6 and release remain their S4-B/C/S5 gates.

**S4-B entry:** the owner initiates S4-B after the final S4-A report. Any daemon-global Scriptlet experiment still requires its separate exact disposable-host approval. Keep PR153 Draft; do not merge, force-push, create another PR or publish a release here.
