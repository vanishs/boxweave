# v0.43.0 stable default promotion — DC-01 installed live evidence (2026-11-03)

**Decision:** Promote immutable, already published [v0.43.0](https://github.com/vanishs/boxweave/releases/tag/v0.43.0) to the default streamed installer by changing **only** `PUBLISHED_RELEASE` and the embedded `install.sh` version selector. No release assets, Provider binaries, application source or service runtime are modified by this promotion. The original `dev` host is not upgraded.

## Immutable source and CI

- Reviewed source merge: [PR #59](https://github.com/vanishs/boxweave/pull/59), squash commit `f6e522d6259b3252664cc033df4a5807daa998dd`; tag `v0.43.0` points to this exact commit.
- Independent PR CI [41765599442](https://github.com/vanishs/boxweave/actions/runs/41765599442): **PASS**.
- Main verification and immutable publication CI [41768868895](https://github.com/vanishs/boxweave/actions/runs/41768868895): **verify PASS, publish PASS**. Release includes 23 digest-bearing assets; published release is immutable.
- The candidate's source tests in the task-owned Box: **699 Python tests PASS**, `check_knowledge.py` PASS, `validate.py` knowledge/unit/Web build three steps PASS, packaged read-only synthetic REST+MCP probe PASS. This is source/fixture evidence, not substituted for live evidence.

## Isolated Linux installation and real Containarium

The existing outer BoxWeave controls one newly created, task-owned Ubuntu 24.04 Box `bwdiag-0430-ci` (2 CPU/4 GB/30 GB, parent nesting configured true, restarted before installation). All root/Incus/PostgreSQL/Provider changes below occurred **inside this disposable Box only**. The independent outer host and its other Boxes remained untouched.

1. Downloaded the immutable `v0.42.12` installer script and explicitly installed `--version 0.42.12`; host installer **exit 0**. The installed `boxweave doctor --mcp` was **overall passed**, and reported `rest_authenticated_inventory_passed`, `authenticated_read_passed`.
2. Downloaded the immutable `v0.43.0` installer and performed a real in-place `--version 0.43.0` upgrade; **exit 0**, state `ready`. No explicit Containarium upgrade.
3. From the installed v0.43.0 host, `boxweave version` was **0.43.0**; `boxweave doctor --mcp` again reported **overall passed**, authenticated REST/MCP reads, and ready app/worker.
4. Launched the actual root-installed standalone Gateway `/usr/bin/python3 /usr/local/lib/boxweave/default/mcp_gate.py` as the non-root `boxweave` application principal and used the installed application MCP client to perform `initialize`, `tools/list`, and new coordinator calls. Tool catalog exposed **68 tools**, including `boxweave_diagnostic_run`.
5. `boxweave_diagnostic_run(profile=preflight)` **passed**: valid config, provider mode, persisted/runtime REST binding, explicit REST selection. `profile=rest_read` **passed**: authenticated live Containarium REST GET and independent MCP `list_containers`, without returning raw inventory or secrets.
6. A nonexistent exact `operation_trace` returned `not_verified` with typed missing receipt/timeline/REST-span codes. Two malformed tool calls (`cmd` injection and traversal-shaped operation ID) returned exact `COORDINATOR_SELECTOR_INVALID` and did not execute a host command.
7. For a positive actual D2 link, temporarily paused only the **inner** BoxWeave application Worker and used the **installed application lifecycle code** with an in-memory single-name configuration and its original database. An owned disposable inner workload `dc01-trace-f92c49` completed `create`, `stop`, and `delete` through authenticated REST. All three operations had `state=succeeded`, write API ACK and correctly linked D2 Trace; corresponding API span counts were **3, 3, 2**.
8. With the installed Agent Gateway, `operation_trace(environment, e5af876fd84a4c549bb2f0cb6e9944ad)` returned **passed** and the expected Trace `08eac73edecdeaaa8f8882b8fa582f56`, with five classified lifecycle/provider_api events and no bearer/token/path/private key material.
9. The inner test Box was independently absent after deletion, and the inner app Worker was restored: `DC01_WORKER_RESTORE=PASS`. A fresh **post-test** `boxweave doctor --mcp` was **overall passed** with ready Worker, authenticated REST and MCP.
10. Tested the **proposed default stable installer** from the promotion branch without `--version` on the same disposable host: `sudo bash install.sh` exited **0**, selected the previously published v0.43.0 asset without mutating its release tag, and retained installed version `0.43.0`. Post-repair `doctor --mcp` was **overall passed** and the installed Gateway still reported `preflight=passed`, `rest_read=passed`, `operation_trace` absent-ID `not_verified` with no secrets.

**Observed outcome:** installed-host read-only coordinator **L/PASS** on this exact Ubuntu/nested Provider topology; positive real REST/D2/D3 integration **L/PASS**; denial/no-secrets/cleanup **L/PASS**. These are actual installed-service and Containarium operations, not fixtures.

## Host and test-harness diagnostic lessons

- Calling a Python test file from `/tmp` without the installed source import path produced `ModuleNotFoundError: bwmini`. The **product probe** uses fixed `-m bwmini.diagnostic_probe` and a vetted cwd, avoiding the earlier manual-script failure.
- The root-installed standalone Gateway is launched by the actual SSH entry command **without** `python3 -I`; `-I` prevents sibling standalone imports (`nesting`) from loading. The root-owned `boxweave` host CLI separately uses its correct isolated entry path. Test the **real deployed launcher**, not a guessed variant.
- Real REST acknowledged writes and the ultimate observed state remain distinct evidence; unknown writes are not automatically retried.
- The Coordinator never grants a general host shell, exposes tokens, executes a write by itself, or bypasses the optional protected Gateway allowlist.

## Remaining explicit boundaries

- The live test did **not** exercise a separate remote ChatGPT/OAuth/HTTPS client or the protected-mode Gateway enabled on an actual host; their source/fixture and tool-policy contract tests passed.
- Actual write-timeout-after-dispatch fault injection, external Provider upgrade, alternate operating systems and production host installation are still **U**.
- The original independent `dev` host was not upgraded by this release/pin change. A source release, stable installer pin and particular running service version remain separate.
- The one nested test Box is task-owned and may be cleaned after evidence preservation; its removal must be observed separately.

## Stable promotion boundary

The v0.43.0 Release **tag and assets are unchanged**. The only `install.sh` executable diff against the exact published tag is the default version string from `0.42.1` to `0.43.0`. The CI publish safeguard compares the whole installer against the tagged source with exactly this single substitution, while checking the immutable tag ancestry. Do not claim production upgrade or a new tag due to this docs/pin promotion.

Related: [DC-01 contract](../design/DIAGNOSTIC_COORDINATOR_V1.md), [candidate source/fixture ledger](DIAGNOSTIC_COORDINATOR_0430_2026-11-03.md), [validation queue](REMEDIATION_TEST_QUEUE.md), [earlier stable selection](STABLE_PROMOTION_2026-11-02.md).
