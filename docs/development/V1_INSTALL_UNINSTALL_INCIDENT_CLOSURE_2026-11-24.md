# Install / uninstall incident closure

Baseline v1 84d2e9a (PR #180 merged). Follow-up to [physical L0 evidence and #177 design](V1_UNIFIED_DOCTOR_RECOVER_DESIGN_2026-11-23.md), [#176](https://github.com/vanishs/boxweave/issues/176), [#177](https://github.com/vanishs/boxweave/pull/177) and [#180](https://github.com/vanishs/boxweave/pull/180). Do not repeat the completed physical RC12 retirement, change the actual L0 install, or reinterpret old acceptance as new evidence.

## Scope and staged acceptance

| Stage | Work | Acceptance / current state |
| --- | --- | --- |
| I1 | Legacy guided prerequisite checks before approval and again before Stage; typed refusal for declined approval | Missing conntrack and other fixed reader paths, unsafe owner/mode/type, permission errors, drift, no subprocess writes. Current candidate implements these checks; final tests pending. |
| I2 | Complete guided read-only Legacy preview, installed identity, partial install phase navigation, optional OAuth completion and effective-service policy feedback | Preview performs no durable writes; explicit approval and native digest remain required; distinguish optional incomplete auth from failed service. Reuse exact Go status/Doctor authority, no secret output, no automatic apt or OAuth reset. Pending. |
| I3 | Legacy uninstall preflight and actionable recovery navigation for prior STARTED audit, native drift, protected originals, missing helper and backups | Preserve existing exact #177 resume-plan/resume contract; investigate whether drift can be rejected before stop/delete, do not add an alternate rollback engine. Pending. |
| I4 | Scoped retirement architecture and acceptance decision | Scoped upgrade/rollback/uninstall remain explicitly unsupported. Study exact journal, boot dependency, ACL, guard, address and ownership teardown; no implementation of destructive scoped retirement without separately reviewed contract and owner decision. Pending. |
| I5 | Integrated delivery | Iterate in isolated Boxes; exact-head quality/build and independent clean installed Legacy/Scoped CI, no regression to #177/#178/#180, doc updates and evidence. Pending. |

## Historical incident disposition

- RC12 uninstall original STARTED to COMPLETE, archived evidence and 9/9 physical postflight: already accepted by #177. Retain historical result, never replay. I3 improves preventative diagnosis and navigation.
- Missing conntrack after interactive approval: I1 closes the wrapper timing gap. The Go Stage check remains authoritative and independently rejects missing or unsafe executables before its writes. The wrapper mirrors the fixed executable metadata contract because it can install older immutable candidates. Package hints are Debian/Ubuntu-specific and not automatic package installation.
- Exact terminal approval refusal: I1 labels expected refusal, without weakening approval or changing exit status.
- OAuth password configured but Origin absent; effective CLI Forward policy vs daemon policy: I2, distinguish optional configuration from failures and independently verified running-service observations.
- Capacity / native inventory / guest readiness: retain #177 diagnostic coverage and #178 scoped limits; no automatic bridge resize or false packet/quota claims.
- Existing scoped paths, stale reviews and scoped refusal: #180 already accepted; retain its regression suite.

## Boundaries

No new MCP Tool or broad Host repair authority. Do not weaken confinement, touch unrelated Projects, replay UNKNOWN, change original audit, erase Root journals, automatically install dependencies, or bypass checks. Keep the PR Draft until integrated acceptance and explicit merge authorization. Historical physical-host and released RC13 evidence remain distinct from current candidate testing.
