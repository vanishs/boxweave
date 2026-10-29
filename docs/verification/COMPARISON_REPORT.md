# Phase6 comparison and validation — TASK-014, 0.9.1

Date2026-10-29, America/Los_Angeles. User authorization SRC-084 and explicit Web setup decision SRC-086. New starting revision f602acf0996b46688ad2b1322769414fea894fe6 (0.9.0); legacy reference d8a09dd180139159e035963502e7793a37e92bae, unchanged from phase5. Work occurs only in rewrite/v2 and its assigned LXC. The old checkout/untracked archive is preserved; copied tracked legacy sources exist only under ignored isolated test state, not product implementation.

## Entry audit and result

The phase5 matrix and real evidence cover the selected architecture/profiles/features; there is no broad missing foundation that prevents comparison. The audit nevertheless found three narrow but real acceptance gaps. Prior phase5 completion prose was too broad for these paths and is superseded by this report; old test results remain historical facts rather than being relabeled failures or erased. All three defects are fixed and specifically regressed in0.9.1. The original requirements are retained, not weakened to improve test results.

The stage is complete within the declared selected scope. There is no open reviewed P0/P1 new-version defect or pending product difference. Source-level phase7 entry is **eligible**, but phase7 is not authorized or executed. Production rollout, fresh public TLS, reboot/autostart, effective confinement and inference are not newly certified. The retained primary0.9.0 test installation was deliberately not upgraded; patched0.9.1 was installed and tested on an independent consistent database copy. See [OPEN_RISKS](OPEN_RISKS.md) before using retained installations.

## What actually ran

| Lane | Actual execution and result | Evidence boundary |
| --- | --- | --- |
| New suite | Non-root assigned Ubuntu24.04 LXC, CPython3.12.3:163 tests passed63.162s; TypeScript/Vite build0.9.1 passed | Test duration is an observation, not a performance benchmark or coverage percentage |
| Browser | Seven source-browser workflows: original hello, history, Graph, access, model, lifecycle and new binding/client setup; Chromium141.0.7390.37, all exit0 | Real HTTP/SQLite/browser; infrastructure explicitly fixture. No new network route/container created by these tests |
| Web-generated client setup | Chromium generated/downloaded4 descriptors +4 PowerShell scripts. All hashes/ASTs and real Windows enrollment against existing isolated Codex/native, Codex/Ubuntu WSL, ZCode/native, ZCode/Ubuntu WSL passed; configuration bytes unchanged | Actual selected client runtimes and authenticated existing MCP. Web did not execute the scripts; the explicit local acceptance runner did. No model inference/global config/key enrollment |
| Live empty-target diagnosis | Corrected CLI completed initialize/discovery109/list_containers against existing nested Provider0.120.1 despite targets=[] | Original installed database bytes/schema6 unchanged, services not restarted. Registration/GUI/workload success are separate checks |
| Real isolated patch installation | Candidate58feeb18 installed0.9.1 with real authenticated MCP/read. A consistent0.9.0 database copy migrated6→7; backup read back6. Installed browser unbound a project while retaining it and generated a client profile | Original application database bytes and service PIDs unchanged. Test-prefix user services stopped/disabled. No infrastructure workload mutation |
| Legacy subset | Exact tracked d8a09dd export, non-root isolated directory:6 job tests and9 hostkey tests passed | Jobs run local subprocesses; hostkeys use mocks. No old live backend/network/confinement certification |
| Legacy readiness | Test module failed collection with ValueError: substring not found while extracting an obsolete shell fragment from current install.sh | No readiness test assertions ran. The extractor/expectation was not edited to get a green result; isolated lane exit1 retained |
| Same-input receipt comparison | Both implementations executed local Python print/exit0 and print/exit7. Success stdout/state/exit agree; failure state/exit7 agree, v2 withholds unexpected output | Actual local subprocesses, old job files and new SQLite; fixture identity. Not old/new remote execution parity |
| Historical0.9.0 counterexample | With identical targets=[] and /usr/bin/false launcher, actual0.9.0 diagnose returned0 without a provider check; corrected CLI returns2/MCP_STREAM_CLOSED | Reproduced using the checksum-verified0.9.0 archive. Actual0.9.0 Store also refuses a new schema7 database |

Public records: [runtime and comparison](phase6-runtime-evidence.json), [Windows generated setup](phase6-windows-web-profiles.json), [immutable legacy source hashes](phase6-legacy-hashes.json), [exact changed source hashes](phase6-source-hashes.json). Raw logs and private test state remain ignored; their digests are retained. Previous actual four-profile hello and native Tenant/TCP/UDP evidence are reused only for unchanged implementations, explicitly as prior evidence, not fresh phase6 packet tests.

## Difference classification and resolution

Categories follow the supplied instructions: new-version defect; confirmed intentional improvement; obsolete legacy error; pending decision; unverifiable. Equal behavior is reported separately, not forced into a defect category.

| ID | Difference / impact | Classification | Evidence, reproduction and decision / disposition |
| --- | --- | --- | --- |
| CMP-001 |0.9.0 could create/plan a project, but the Web could not reselect/edit/unbind; no safe unbind service existed | New-version defect, P1, fixed | REQ/AC010; tests/test_bindings.py + binding_browser_check.py and actual installed-copy browser.0.9.1 returns complete binding identity, supports revision-checked edit/local unbind/rebind, retains project/Graph/receipts, invalidates stale plans and prevents queued stale execution. RISK-601 closed |
| CMP-002 |0.9.0 application diagnose with no targets falsely returned0 even with a dead launcher and opened/initialized local state | New-version defect, P1, fixed | REQ/AC003/009; actual same-input old/new counterexample above. Diagnosis now performs handshake/discovery/read unconditionally, without Store/admin/database dependency. Real109-tool call leaves installed DB unchanged. RISK-602 closed |
| CMP-003 | SCOPE required public client setup generation on Web, while0.9.0 offered only CLI enrollment and Web model handoff | Pending decision → new-version defect, P1, fixed | User SRC-086 explicitly selected completing Web rather than deferral. Authenticated Client setup now generates four supported descriptors and a hash-gated enrollment script, preserving explicit replacement intent. No credentials/remote execution accepted. Three new tests, browser downloads, actual Windows scripts/config preservation and installed-copy endpoint pass. RISK-603 closed |
| CMP-004 | Legacy generic job output is retained/redacted; v2 fixed-check failure output is withheld while retaining actual exit | Confirmed intentional improvement | Same local print/exit7 input yields failed/7 in both, old stdout hello world, new empty/output_withheld. Approved bounded fixed-command/untrusted-output boundary SRC-026/DEC005 and REQ016; no arbitrary Web shell compatibility layer introduced |
| CMP-005 | Legacy install couples mandatory Web/provider/bootstrap settings; v2 package manages only private app releases/API/worker and reuses independent credentials | Confirmed intentional improvement | DEC003, REQ008/009, ADR001/003; legacy source/static contract versus prior and fresh v2 real isolated installs/Web-down MCP evidence. No legacy privileged installer executed for parity |
| CMP-006 | New client/runtime matrix is selected Windows-hosted Codex/ZCode native/WSL, not all legacy templates/Agents; CLI spelling/exit encoding need not match | Confirmed intentional improvement | DEC002 and SCOPE dispositions; actual generated4-profile registration. Explicit machine states replace reliance on matching legacy CLI0/1/2. No claim for unselected Agents/WSL-hosted Agents |
| CMP-007 | Initial phase1 WIP suggested interactive replacement choice was dropped | Obsolete legacy error/inference, superseded | SRC-020 was uncommitted/static, never a confirmed current runtime defect. Current d8a09dd server/clients.py assigns options[replace] after a yes choice. This is fresh source inspection, not a replay of old interactive enrollment. New script replacement flag is separately exercised; no old code copied |
| CMP-008 | Current legacy readiness test extracts text that no longer exists | Unverifiable readiness comparison; legacy harness error retained | Exact immutable export/test fails collection, no assertions run. Legacy remains read-only; RISK-604 records test-harness maintenance. This is not evidence the legacy runtime is unhealthy or the new installer is correct |
| CMP-009 | Complete old/new live Web/Tenant/packet behavior cannot be inferred from local suites or old reports | Unverifiable in this comparison lane | New native network feature has separately scoped phase5 real0.120.1 evidence; old equivalent live rollout was not executed and supporting credentials/security were not modified. Explicit confidence limit RISK-606; no parity claim |

Consistent outcomes: actual local hello success, actual nonzero exit state, lost-worker uncertainty, strict trust/identity boundaries, native membership distinct from display/account identity, no automatic replay and graph nonexecution remain aligned with the curated old intent. Local/static/fixture and actual Provider evidence are never combined into an invented live comparison.

## Reproduction and preservation

In the assigned project root, with the reviewed sources, existing locked venv/Node dependencies and original phase5 baseline archive available:

```sh
.venv/bin/python -m unittest discover -s tests -v
npm --prefix web run build
for script in browser_check history_browser_check graph_browser_check access_browser_check model_browser_check lifecycle_browser_check binding_browser_check; do
  PYTHONPATH=. .venv/bin/python scripts/$script.py
done
PYTHONPATH=. .venv/bin/python scripts/compare_local_receipts.py
```

Use a dedicated ignored legacy directory populated only from `git show d8a09dd:path` for the files in phase6-legacy-hashes.json; verify each digest. Run its three named unittest modules as an unprivileged user, retaining the readiness collection failure. No legacy working-tree import, installer run, data/config mutation or archive inspection is required. Comparison source selects actual local Python commands only; it is not a generic remote command runner.

The final concentrated new lane used one600-second deadline and exited0. Installation scripts use their declared bounded phases; no retry duplicated a remote resource. The first isolated patch browser check queried protected data before async login completed; that known pre-action test race was preserved and fixed with an explicit login completion wait in a fresh private copy. It was not counted as a product failure or hidden as a pass. Both test-prefix services are stopped/disabled. Original0.9.0 services/state, old source/main, original live hello workload, provider core and public routes are preserved. No new outer/public port, inference request, host SSH/firewall/provider change or push occurred.

Reliability evidence includes existing real process-crash/SQLite-failure/claim-race/no-replay tests plus new stale binding/session/hash gates. No throughput, p95, coverage percentage or SLA is fabricated. Schema7 is an additive local binding tombstone; old binaries reject it. Use the installer and its consistent pre-migration backup for rollback, never a blind binary downgrade or copy over active SQLite state.
