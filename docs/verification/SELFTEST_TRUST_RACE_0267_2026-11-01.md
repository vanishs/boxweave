# Selftest trust-readiness and typed-failure remediation — v0.26.7

**Date:** 2026-11-01 local / 2026-11-02 UTC. **Owner-approved test environment:** isolated Box `bwdeep-1101` on outer host v0.26.5, current source v0.26.6 released, proposed v0.26.7. **Status:** Source remedy under validation; original failed run and cleanup independently observed. Refer to [cross-layer OAuth/security audit](DEEP_AUDIT_0266_2026-11-01.md) and [risk register](OPEN_RISKS.md).

## New proof: a previously unobserved selftest race

The installed v0.26.5 full selftest originally returned `state=passed`, with four successful `create/hello/stop/delete` steps and no residual inner instances. On the *same* nesting-enabled test Box, the official v0.26.5→v0.26.6 upgrade completed successfully, with exact provenance source `16ba7c22b21432908d58a436b1970b45872c5fd7` and `doctor --mcp` all **10 checks passed**. However a new confirmed app-owned selftest run failed:

- The host wrapper `sudo boxweave selftest run` returned exit **2** with only `SELFTEST_FAILED_INSPECT_STEPS`. The original app database showed run `st-255bb2d8e68baf5f`, exact inner Box `boxweave-selftest-91f12a`, `state=failed`, `last_step=delete:OK`, `cleanup_state=performed`. Independent Incus list showed zero inner instances.
- A **different new confirmed plan** was executed directly via the installed application runner under the non-root `boxweave` account so the app's structured output was available even on failure. The exact inner Box `boxweave-selftest-249ff8` had `create:OK`, **`hello:TRUST_FAILED`**, `stop:OK`, `delete:OK`; run state `failed`, cleanup `performed`, service exit **2**, independently zero residual instances.
- The failing *hello* is not equivalent to a failed create/cleanup. An outer Box can be Running before the independent pinned SSH host-key reconcile job is ready. No host key was disabled, accepted on first use, overwritten, read or exported. No mutating operation was blindly replayed; only distinct explicitly planned selftests were executed after confirming prior cleanup.

This failure mode was not exposed by previous full-run fixture success or the healthy generic doctor; host CLI erased the typed app output when the app process exited nonzero.

## Patch v0.26.7 — strict boundaries

**`bwmini/selftest.py`:** Within the *existing single* confirmed test run, retry only the fixed, side-effect-free `provider.hello` identity/Python check when it raises exactly `TRUST_FAILED`, `TRUST_NOT_READY` or `NOT_CONNECTION_READY`. Wait in bounded steps, capped by the lesser of 90 seconds, one-third of the test budget and a reserved 15-second cleanup margin. The Provider continues using strict trusted host keys and checks the exact authorized account and hostname. Do **not** retry arbitrary commands or any create/start/stop/delete. Unknown errors, identity mismatches and confirmed nonzero command execution fail immediately and proceed to original-resource cleanup.

**`bwmini/host_cli.py`:** For `selftest run` only, use the pre-existing bounded reader that can safely accept app exit **2** and its JSON stdout. Surface a `state=failed` step report with exit **1** (known failed test) and an allowlisted typed preflight `code` with exit **2** (refusal). Unknown responses/unsafe fields still map to a generic bounded error, not arbitrary process output. `selftest plan` remains unchanged.

**Fixtures required:** transient SSH trust failure then eventual success with exactly one create/stop/delete; persistent trust failure remains FAIL with cleanup; confirmed identity mismatch is never retried; precise known CLI error and failed-step report survive nonzero process exit; non-allowlisted error content is withheld.

## Acceptance matrix

| Scope | Status / actual evidence |
| --- | --- |
| v0.26.5 normal selftest create/hello/stop/delete | **L PASS**, original positive run `f014ec`, zero orphan children |
| v0.26.6 actual selftest immediately after upgrade | **L FAIL**, two independent plan runs, one exact classified `hello:TRUST_FAILED`, both independently cleaned |
| v0.26.6 installed version, provenance, doctor | **L PASS**, `0.26.6` at source `16ba7c22...`, 10/10 |
| v0.26.7 trust retry + CLI typed error | **S patch** pending CI and repeat on isolated real Box |
| External host-hop SSH or D4 positive allowlist | **U/BLOCKED** unchanged |
| Local OAuth admin denial rate fix from v0.26.6 | **T PASS** source/CI, public OAuth client over TLS **U** |

## Completion and cleanup requirements

Before a full v0.26.7 PASS, confirm source CI, immutable Release, install the exact new package in the *same authorized Box*, verify provenance and doctor, run a new confirmed selftest, inspect per-step output/exit and the absence of orphan inner children. Also verify an intentionally invalid confirmation produces a typed refusal without creating a Box. If these fail, preserve only useful new evidence and mark FAIL/PARTIAL; **do not silently retry an unknown mutation**. On confirmed success, stop and delete the test-owned `bwdeep-1101` and independently confirm inventory absence. Historical `bwaudit-1101` remains retained untouched.
