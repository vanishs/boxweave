# Install / uninstall incident closure

Baseline v1 84d2e9a (PR #180 merged). Follow-up to [physical L0 evidence and #177 design](V1_UNIFIED_DOCTOR_RECOVER_DESIGN_2026-11-23.md), [#176](https://github.com/vanishs/boxweave/issues/176), [#177](https://github.com/vanishs/boxweave/pull/177) and [#180](https://github.com/vanishs/boxweave/pull/180). Do not repeat the completed physical RC12 retirement, change the actual L0 install, or reinterpret old acceptance as new evidence.

## Scope and staged acceptance

| Stage | Work | Acceptance / current state |
| --- | --- | --- |
| I1 | Legacy guided prerequisite checks before approval and again before Stage; typed refusal for declined approval | Missing conntrack and other fixed reader paths, unsafe owner/mode/type, permission errors, drift, no subprocess writes. Implemented; I1 Box focused tests passed; final exact-head CI required. |
| I2 | Complete guided read-only Legacy preview, installed identity, partial install phase navigation, optional OAuth completion and effective-service policy feedback | Preview performs no durable writes; explicit approval and native digest remain required; distinguish optional incomplete auth from failed service. Reuse exact Go status/Doctor authority, no secret output, no automatic apt or OAuth reset. Implemented; I2 Box regression passed; final exact-head installed preview marker required. |
| I3 | Legacy uninstall preflight and actionable recovery navigation for prior STARTED audit, native drift, protected originals, missing helper and backups | Preserve existing exact #177 resume-plan/resume contract; investigate whether drift can be rejected before stop/delete, do not add an alternate rollback engine. Implemented; Python and Go regressions plus actual read-only Legacy native witness passed; final installed drift marker required. |
| I4 | Scoped retirement architecture and acceptance decision | Scoped upgrade/rollback/uninstall remain explicitly unsupported. Study exact journal, boot dependency, ACL, guard, address and ownership teardown; no implementation of destructive scoped retirement without separately reviewed contract and owner decision. Design completed: [Scoped retirement design](V1_SCOPED_RETIREMENT_DESIGN_2026-11-24.md); actual installed read-only refusal passed. Destructive implementation deferred. |
| I5 | Integrated delivery | Iterate in isolated Boxes; exact-head quality/build and independent clean installed Legacy/Scoped CI, no regression to #177/#178/#180, doc updates and evidence. Final candidate includes mandatory new native CI markers; exact-head verification required. |

## Historical incident disposition

- RC12 uninstall original STARTED to COMPLETE, archived evidence and 9/9 physical postflight: already accepted by #177. Retain historical result, never replay. I3 adds preventive native observation and original-audit-only navigation without replay.
- Missing conntrack after interactive approval: I1/I2 close the wrapper timing gap. The Go Stage check remains authoritative and independently rejects missing or unsafe executables before its writes. The wrapper mirrors the fixed executable metadata contract because it can install older immutable candidates. Package hints are Debian/Ubuntu-specific and not automatic package installation.
- Exact terminal approval refusal: I1 labels expected refusal, without weakening approval or changing exit status.
- OAuth password configured but Origin absent; effective CLI Forward policy vs daemon policy: I2 separates the installed result and optional OAuth status, and only labels running-service Forward policy when Doctor observes it; no unauthenticated reachability claim.
- Capacity / native inventory / guest readiness: retain #177 diagnostic coverage and #178 scoped limits; no automatic bridge resize or false packet/quota claims.
- Existing scoped paths, stale reviews and scoped refusal: #180 already accepted; retain its regression suite.

## Boundaries

No new MCP Tool or broad Host repair authority. Do not weaken confinement, touch unrelated Projects, replay UNKNOWN, change original audit, erase Root journals, automatically install dependencies, or bypass checks. Keep the PR Draft until integrated acceptance and explicit merge authorization. Historical physical-host and released RC13 evidence remain distinct from current candidate testing.

## Stage evidence before final I5

- I1: 7dade7fc, tree 49112965; original operation f211cbfaf099f42530162ada3e943e11 guest exit 0; 28 Python tests, syntax and format PASS. Superseded exact-head CI was not a final gate (quality cancelled by newer revisions).
- I2: 356fe3a7, tree 223ba35c; operation e6113c3eda9d2d954644c28459fa9623 guest exit 0; 30 Python tests, syntax and format PASS. Full native Legacy preview, phase navigation, and optional OAuth readback/failure tested with coordinator fixtures.
- I3: first operation a093717d92754e9839a63a9edfce880f: 46 Python tests passed, Go fixture failed compilation due to embedded SDK ACL field; corrected at 19e81ddc, operation 9095b3c54db063b6f0d55b7fe6f791eb exit 0, Go targeted and format PASS. Actual staged Legacy Box dd6fb4ab-280b-4c51-a6f9-295e389472f0 original ac85ccc110c272aa2a3d1789a6a44ff3 exit 0: I3_NATIVE_PRECHECK_OBSERVED and I3_NATIVE_PRECHECK_NO_EFFECT; original inactive controller, binary/manifest/journal and native Project/Profile unchanged. No activation or retirement attempted in that nested lab; existing LXC effective-unit limitation remains.
- I4: 19e81ddc wrapper original 8a71ccaf95122b59b40ba7e49936e757 exit 0; all scoped previews refused, service/PID/hash unchanged. See dedicated design. First checkout attempt ba306352c79d4fa774fd6e22082e7165 failed due to single-branch remote reference; no retirement effects; exact-commit retry passed. No scoped retirement implementation.

I5 requires exact-head source, quality, dual architecture/distribution and clean installed native matrix. New Legacy markers: I2_GUIDED_LEGACY_PREVIEW_NO_EFFECT and I3_LEGACY_DRIFT_REFUSED_BEFORE_RETIREMENT. The latter changes only a benign Project description on the clean CI VM and restores it before full retirement. Previous #177/#178/#180 gates remain. Do not claim final native acceptance from the earlier Box checks. A newer wrapper paired with an older installed binary lacking the native retirement reader refuses safely; use the matching verified Release uninstaller for that build. No actual L0 changes or release.
