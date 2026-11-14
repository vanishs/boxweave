# S2.5 BH-018 — Atomic native Exec finalization and installed MCP acceptance gate (2026-11-14)

**Scope:** only Draft PR [#146](https://github.com/vanishs/boxweave/pull/146), branch `fix/v1-s25-bh018-bh019-20261114`; no merge or v1 release claim. BH-018 remains **PARTIAL** until independent installed MCP and crash/lost-ACK acceptance, BH-019 remains **BLOCKED**. The operator needs no host action and has paused SELinux research.

## 1. Root cause and targeted product fix

A real Incus 7.6 native Exec may return `Failure/126/Command not executable` or `Failure/127/Command not found` even when a real Guest process executed and changed a file. Earlier native controls proved both. Original Incus Operation terminal observations expire in about five seconds, so lost/expired original terminal evidence cannot be reconstructed by observing output file names.

The prior BoxWeave implementation committed `exec_outputs` in `Store.SaveExecOutput`, then separately committed `VERIFYING → SUCCEEDED` in `Store.Verified`. A crash between the two commits led to an unnecessary UNKNOWN and locked resource despite the durable output row. The fail-closed UNKNOWN behavior was correct for uncertain side effects but not an ideal atomic record.

**Code changes on this Draft branch:**

- `internal/state/lifecycle.go`: new `FinalizeVerifiedExec` uses **one SQLite transaction** with strict `WHERE` guards for the original `box_exec`, `VERIFYING` state, exact native Operation UUID, original Box UUID/incarnation, project/resource key, and existing matching `OWNED` registration. It validates the exact original `exec_<nativeID>.stdout/.stderr` paths and numeric Guest exit, commits `exec_outputs` + `SUCCEEDED` + audit event **together**, or rolls back everything. No status transition on failed identity/transaction.
- `internal/control/box_tools.go`: the existing exact Incus terminal/ID/path verifier invokes the atomic finalizer, not two independent commits. Only a *confirmed* finalizer commit emits the internal `ErrVerifiedExecCommitted` control result.
- `internal/state/worker.go`: the Worker recognizes this signal **only after independently re-reading** the same original `box_exec` journal state `SUCCEEDED` and matching `native_operation_id`. It does not issue a second final-state write. A forged/uncertain signal cannot unlock the Box; failure remains fail-closed UNKNOWN.
- `internal/state/worker_bh018_special_terminal_test.go` now exercises the atomic path, including original-key dedupe and new-key admission after genuinely proved completion.
- `internal/state/worker_bh018_atomic_finalization_test.go`: negative tests for SQL failure between UPDATE and output/event INSERT (complete rollback), closed/reopened SQLite receipt, replay immutability, foreign project/Box/incarnation/native ID/stream/path, and a forged internal completion signal.
- Separate additional `internal/state/worker_bh018_atomic_crash_test.go` exists on the same branch and injects an event-INSERT failure *after* the output INSERT. **Treat as parallel-added source coverage, not independently proven CI until a latest-head run succeeds.**

**Non-goal:** This does *not* make a lost Incus ACK, unobserved original final, or expired original native Operation safe to automatically retry. The Guest process itself may have run for 126/127. `SUCCEEDED` denotes verified BoxWeave native Exec **terminal completion**, while the Guest command result is `command_success=false` for 126/127.

## 2. Independent Go + real SQLite acceptance

Only in disposable L0-owned D `s25-bh019-bpf-ab-d-1114`, UUID `0d0ffd60-2fd6-4734-b40c-6eabdb561033`, with pinned Go 1.26.9 toolchain already independently SHA-256 checked in the prior matrix, checked out *exact candidate commit* `e4821af7c694572e180b507d2a27a1d760d1f436` (source tar archive, not an installed release). Original L0 BW `bd8b74c39570dcc66449d0e847bdec58`, outer native `205f4328-1db3-4207-a40c-cd4279d6d6ab`, **SUCCEEDED with Guest exit0**:

`go test -count=1 -v ./internal/state ./internal/native ./internal/control -run TestBH018`

**PASS**: `TestBH018AtomicFinalizationNoPartialReceiptOnSQLiteAbort`, `TestBH018AtomicFinalizationSurvivesStoreCloseReopen`, `TestBH018AtomicFinalizationRejectsForeignIdentityWithoutAnyRow`, `TestBH018WorkerRejectsForgedAtomicExecCompletionSignal`, original dispatch-boundary/UNKNOWN negative tests, 126/127 real-outcome source controls. The additional parallel-added event-INSERT crash test requires the exact later-head CI confirmation. These are genuine local SQLite tests; they are **not** yet the full installed HTTP MCP native blackbox.

## 3. Installed service laboratory and new original UNKNOWN — immutable boundary

D has real nested Incus 7.6. For a **test-only installed MCP service** (without replacing the existing operator L1 RC4) a new dedicated non-root user `bwbh018` was created **only in D**, restricted project `user-1001` automatically created via the official Incus user socket. Independently read back:

- User UID1001, primary `incus` GID990; only user socket access; foreign `default` Project read **denied** (original BW `f33fdb0217c6306a278b22c8d9a5122b`).
- Project's default profile scopes NIC to `incusbr-1001` and root disk to local `default` pool, `restricted=true`; no root Incus-admin socket delegated to this user.
- Admin copied official Ubuntu image only into the one-use restricted Project; it was visible to the service user. Non-root created and started exact native Guest `bh018-mcp-owned`, independently verified native UUID **`600e387b-7efe-4ce4-9b30-138188b2ba60`**, `RUNNING`, and native restricted Exec `/bin/true` worked (original BW `9dba3ccf5ab5b057d44f5756d63bfa34` / `c5316b380c4d2990f1d150c2779ff421`).
- Built exact candidate binary from commit `e4821af...` inside D: SHA256 **`3ae6fb83a3bad80ad3261b047727dda06e877e107c03ecf176a19c0ee8b91853`**; build-info version `1.0.24-bh018-atomic-lab`, recorded commit `e4821af7c694572e180b507d2a27a1d760d1f436`. Original outer BW `9f0d3b59522afddd3dc2f7af13a54051`, native `b65cd459-626e-4a7b-8006-76e8ee493552`, **SUCCEEDED, exit0**.
- The intent was to use a one-use, explicitly test-only `owned_boxes` fixture seeded after independently verifying the exact native UUID, **not** claim that BoxWeave performed native Create or accepted an arbitrary foreign guest. Before this full HTTP test ran, the original L0 `box_exec` setting up the protected service token/ledger/fixture returned **UNKNOWN / `POSTCONDITION_UNVERIFIED`**, original BW **`718b99438cfa8f1349895ade0e8dfa9a`**, outer native Incus **`345841b8-18b9-46f9-a272-54da45d985f6`**. Original `operation_get` independently reconfirmed UNKNOWN. `operation_output_get` is unavailable (`operation not found`); neither the test ledger's actual creation nor Guest effect outcome can be inferred.
- **STOP RULE TRIGGERED:** no further `box_exec`, installation, cleanup, manual claim unlock, or re-dispatch against D after this original UNKNOWN. D is **not** a valid installed MCP PASS. The same rule applies to earlier C and historical UNKNOWNs. This event is new evidence in the BH-018 failure matrix and must be independently diagnosed before any operator-local resolution.

## 4. Independent full installed MCP acceptance moved to the existing single-use Actions VM

To avoid unsafe new Guest dispatch on D, added **test-only** `scripts/s2_bh018_atomic_mcp_probe.go` (official `github.com/modelcontextprotocol/go-sdk/mcp` client) and wired it into `scripts/accept_s2_native.sh` **after** the original S2 official-SDK/native/foreign-project tests and cleanup, while the same one-use restricted project and installed nonroot BoxWeave service are available. The new stage:

1. Uses public `box_create` via official HTTP MCP to create a new, UUID-registered one-use Box, not a preseeded owned guest.
2. Sends **two distinct native Exec** commands: actual Guest shell writes uniquely identified marker and then exits 126 or 127; requires original `SUCCEEDED` BoxWeave receipt with exact native Incus ID and both original recorded stdout/stderr streams, `exit_code` 126/127 and `command_success=false`.
3. Persists a test-only checkpoint containing *original IDs, never secrets*, restarts **only this disposable BoxWeave service** on the exact same protected SQLite and restricted user socket, and verifies original ID/paths/output survived.
4. Replays **the same key and same argv** and requires the exact original durable receipt without a second native dispatch. Then issues a **fresh** readback command to independently assert the real Guest side-effect marker appears **exactly once**. Stops/deletes the owned Box and verifies empty native inventory.
5. Failures abort the **isolated GitHub Actions VM**. Never use the current L0 D unknown or the physical host to retry. These tests will count only after a green run tied to the exact current PR head.

Added Go probe and script stage commits `2923c279006403dbc254501220e82fd82627872f`, `58a0731e8a1d06c6bf46a0cd6cb5dfc652c975b4`. CI formatting initially failed for newly added probe, not the amd64/arm64 build. In a **separate disposable format-only E** Box `s25-bh018-format-only-e-1114`, UUID `0512aadc-18d0-4e55-b058-07554a2714df`, Go 1.22.2 `gofmt` independently formatted the exact new official probe (format witness SHA256 `b574b06adadb0e1acd4379add6b67dd3a583f495a6590e2470f88890542ba55d`), then updated the Draft branch in commit `dd4b7ab6ab20dc0f454b9934c0965a0b4e18c35a`. E exists and had **no observed UNKNOWN**; it is not a production host.

**Acceptance at this documentation instant:** Required quality, source and isolated native workflow were *running* at `dd4b7ab...` (`43640130962`, `43640131165`, `43640131008`). Do not mark **installed MCP PASS** before obtaining real run results, including actual native job rather than build-only/skip. Even if HTTP MCP above passes, independent worker SIGKILL during the **exact atomic final COMMIT** and lost-ack negative acceptance should be separately reviewed before declaring the entire BH-018 durable recovery matrix PASS.

## 5. BH-019 unchanged in this iteration

Ubuntu `linux (7.0.0-43.43)` verified source patch does not change the same-depth AppArmor policy-admin predicate; physical kernel audit separately proved `not policy admin` / -13 for a confined deeper child and success for unconfined parent on the same policy file. **No SELinux tests, no host policy changes, no CAP/AA bypass**, and no claim of secure deep nested PASS. See [pinned kernel evidence](../research/S25_BH018_019_PINNED_ROOT_CAUSES_AND_HOST_GATE_2026-11-14.md).

**Next signoff gates:** exact-head Actions with native I+N+V; a separate controlled crash/lost-ACK at atomic COMMIT; independent Verifier; only then update BH-018 from PARTIAL. BH-019 still requires secure per-depth AppArmor delegation and cgroup-device/network negative proofs. Neither may be signed off based on source-only tests.
