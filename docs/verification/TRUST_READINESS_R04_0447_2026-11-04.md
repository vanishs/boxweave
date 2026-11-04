# v0.44.7 bounded post-create SSH trust-readiness and R-04 acceptance — 2026-11-04

**Status: source and isolated manually patched candidate tested; exact published-release CI and installed v0.44.7 are separate pending gates.** Scope: only the newly acknowledged **create** lifecycle's verified host-key handshake and the opt-in cross-entry coordination paths. This does **not** claim global multi-Agent/REST/Web perfection, per-client RBAC, atomic upstream disk shrink safety, or an upgrade of the owner's host.

## Provenance and original failures

Outer owner environment, read only: BoxWeave Gateway v0.44.6 reported by `boxweave_info` and demonstrated installed alias `connect(box=bwreaccept-1104-container,exec=id -un && hostname)` exit 0; Containarium 0.128.0, Incus 7.5.1. Owner reports its v0.44.6 upgrade completed; independent full installer audit was not performed. Its R-02 `boxweave_task_list` still raises **`TASK_TRACKING_UNAVAILABLE`**; explicit root-authorized local `sudo boxweave task provision` remains needed.

Isolated live test parent: `bwverify-0446-e2e-container`, backend `dev`, 3 CPU/6GB/45GB, Ubuntu 24.04, nesting enabled, inner BoxWeave exact v0.44.6 and Containarium v0.128.0. Root-owned `coordination-policy.json` has opt-in `enabled=true` only *inside the isolated parent*, and app `coordination.enabled=true` is configured there. Parent's `gateway-tasks.sqlite3` and `claims.sqlite3` were explicitly provisioned in the previous acceptance round.

Two independently acknowledged application REST creates `bwweb-0446` (op `f654ae360e8e458485968c4b3bdabe54`) and `bwweb-0447` (op `fe71f6a4816e4fbab690a09fa83b3622`) left the actual nested LXC Boxes **RUNNING** while their original app receipts became `failed/blocked/TRUST_FAILED`. Later identical native-MCP identity reads succeeded without replacing these Boxes or relaxing host-key checks. Their shared resource claims retained **unknown** and blocked a separate Gateway task with `RECOVERY_REQUIRED`, `failed_before_dispatch` and **no guest execution**.

Critical diagnostic correction: the application Provider runtime's configured launcher directly exposes **58 upstream Containarium MCP tools**, with native `connect` but **without** a BoxWeave Gateway-scoped fixed identity probe. A first candidate added a narrow protected-Gateway probe, ran nine focused fixture checks and even passed a real Agent Gateway probe of the surviving child; nonetheless a second real application create failed the same way because this was **not the application call path**. This entire candidate was reverted from the source checkout **and** isolated installed Gateway and Provider, restoring their SHA-256 hashes before the subsequent fix. The retained v0.44.6 native MCP route was not weakened.

Evidence for asynchronous host-key publication: the dedicated test-parent `boxweave-trust.service` root-only timer publishes Incus-derived host keys at roughly 15-second intervals. Journal observations at 03:10:07/23 (+08) reported one trusted target and 03:10:39/55 two; freshly created guest SSH readiness and host-key material need not become consistent on the first observation. This supports a **bounded readiness-race hypothesis**, not proof of a particular raw-key mismatch mechanism. Raw host keys and private credentials were not exported.

## Minimal source fix

In `bwmini/lifecycle.py` only, when an already **acknowledged** `create` has a verified non-null exact target and its identity check returns `TRUST_FAILED` **before the original fixed deadline**, preserve `state=observing, phase=trust_pending, code=TRUST_READINESS_PENDING`; the regular worker observes the **same original named Box** after its existing ~15-second poll. No create/mutation is retransmitted, SSH/host-key validation is never disabled, and target incarnation, Tenant, running state and fixed deadline continue to be checked. A stable **existing** Box start/other identity trust mismatch remains a terminal `failed/TRUST_FAILED`. Persistent create trust failures expire as `unknown/DEADLINE_EXCEEDED` without release or replay of the uncertain shared claim. This is controlled observation, not blind retry of a write.

Updated `tests/test_lifecycle.py` covers (1) an enduring create trust mismatch with repeated observations but one Provider create and one original deadline; (2) a transient create trust failure that later succeeds with one immutable createdAt and exactly one create; (3) an existing Box start that still immediately rejects `TRUST_FAILED`. **36** focused `test_lifecycle.py` plus `test_lifecycle_shared_claims.py` tests passed (25+11). After removing the ineffective Gateway-probe experiment, **738** Python tests passed as the unmodified complete suite plus the two net new regression cases, exit 0, in the installed application Python environment. A fixture wrapper made *legacy* Gateway policy absence explicit only **inside the unit-test process**; the installed test parent's real protected-mode policy stayed active. Knowledge check before the v0.44.7 report: 207 docs, 26 REQ/26 AC, exit 0. Code diff: exactly `bwmini/lifecycle.py` and `tests/test_lifecycle.py` beyond required version/metadata/doc updates.

## Actual independent nested Box proof (manually patched source; NOT a published v0.44.7 install yet)

After backing up the original test-parent application lifecycle source and installing the exact reviewed candidate only into that isolated Box (not the owner host), the real Web-shared lifecycle created `bwweb-0448-container` (user `bwweb-0448`, IP `10.203.0.4`, 1 CPU/1GB/10GB) through plan `61f280551f4a497c95e2aca6a1a2f55f` and operation `929a39593850432790a745b33b41cb7a`. The authoritative installed worker receipt became **`succeeded/connection_ready/OK`**: same exact incarnation, guest `whoami && hostname`, exit 0; no duplicate creation was issued. This positive observed startup does not establish that a spontaneous trust error occurred in *this* particular instance; the retry branch itself is source-fixture tested.

Cross-entry R-04 evidence, opt-in protected configuration unchanged:
- While the two earlier application creates remained uncertain, a Gateway task for the same Box returned **`RECOVERY_REQUIRED`**, and its durable receipt was `failed_before_dispatch` without any guest command execution.
- For the succeeded `bwweb-0448` app operation, `coordination_operations` reached `observed_terminal` with zero retained per-operation resource keys. A fresh `boxweave_task_submit` from the *separate* Gateway process executed afterward, exit 0; duplicate same-key task returned the **same task ID `15d0bd5d3294000240acc23b6db0cef5`** and its bounded file read showed exactly one guest write, 27 bytes.
- A real protected Gateway Box lease of `bwweb-0448` was acquired without disclosing its token. A second unleased task received **`RESOURCE_BUSY`** with no side effect; a token-bearing holder task returned **completed/exit 0**, the guest file contained only `holder_authorized` (17 bytes), and the lease released successfully.
- Direct `connect(exec)` and unclassified direct forwarded mutators remained **`COORDINATION_USE_TRACKED_TASK`**/`COORDINATION_TOOL_UNCLASSIFIED`, rather than being re-exposed to fix app readiness.

The two original failed test operations remain **failed with conservative UNKNOWN claims**; they were not silently adopted or replayed. App `doctor --mcp` on the manually patched installed 0.44.6 test stack reported `incomplete` for app unit visibility and release digest checks, so it is explicitly **not** counted as a release-installed doctor PASS. A fresh exact v0.44.7 installer run and independent post-install doctor are required after publication.

## Scope, remaining work and operator boundaries

| Gate | Evidence | State |
| --- | --- | --- |
| R-04 same-target conflicting app/Gateway claims and UNKNOWN retention | actual installed 0.44.6 protected isolated host; real nested children | **L PASS** |
| R-04 lease holder vs other Agent + verified one write | protected installed Gateway + real nested child | **L PASS** |
| R-02 opt-in tracked task/dedup across Gateway/app lifecycle claims | actual installed private stores and guest effect | **L PASS** |
| R-04 post-create SSH trust-readiness candidate, no retransmission | full source fixture tests and new real nested create | **T + bounded L PASS; exact release U** |
| Owner host R-02 provisioning / global R-04 configuration | untouched by this isolated task | **U, explicitly operator-authorized host action needed** |
| Real authenticated installed Web HTTP endpoint, external OAuth/TLS clients, D4 positive, E2–E4 | no authorized real credentials/upstream contract | **U / deferred** |
| Containarium raw create after-commit ambiguity / native atomic disk shrink | upstream v0.128.0 defects and no global transaction guarantee | **OPEN upstream** |

**Follow-up:** publish an exact new v0.44.7 release after source-bound CI; reinstall it in the isolated parent from the immutable asset, confirm full application health and controlled Box cleanup, then record final source SHA, release assets and any remaining blockers. Do not move `PUBLISHED_RELEASE` or upgrade the owner's host without a separate decision.
