# D2 implementation specification — offline report, nested evidence and durable selftest

Status: **implementation-ready architectural design; unimplemented and not a grant to run live tests**. Reviewed source `main@817273d98d8823fcd8489eb0d3f329a67929b91d` (BoxWeave 0.21.4). Depends on [D0](DIAGNOSTICS_D0_IMPLEMENTATION.md) (truthful health states) and [D1](DIAGNOSTICS_D1_IMPLEMENTATION.md) (versioned private trace/event storage). Parent [proposal](DIAGNOSTICS_PROPOSAL.md), [source audit](../verification/DIAGNOSTICS_SOURCE_AUDIT.md). Permission and sharing settings require maintainer approval.

## 0. End-user target and non-goals

Desired operator workflow for a failure:

```text
sudo boxweave doctor
sudo boxweave report --kind environment --operation <opaque-op-id> \
  --output /some/pre-reviewed/new/private/report-directory
# inspect manifest/omissions; review before sharing with developer or Agent
```

For unexpected behavior without an operation ID:

```text
sudo boxweave report --since 2026-10-30T08:00:00Z \
  --until 2026-10-30T09:00:00Z --output <new-private-directory>
```

These are **planned** commands, not currently supported. A report is an account-scoped, bounded, **read-only** forensic snapshot; it is not a debugger that executes arbitrary commands, a reproduction driver, or a guarantee that a missing event never happened. When MCP/Web/Worker have stopped, host-local collection must still return whatever independently available evidence the operator is authorized to inspect.

Do not add a second Debug MCP. Existing-MCP `boxweave_diagnose` remains conditional D3. Do not gather user project files by default, reconfigure nesting, restart core containers, perform new creates, or silently manipulate firewall/SSH/provider settings.

## 1. Architecture and source anchors

```text
operator -> root-owned /usr/local/bin/boxweave -> root-owned host_cli.py
                                    |
                                    +-- public trust-validated host observations
                                    +-- limited fixed Incus commands (only on explicit target)
                                    |
                                    +-- runuser -u boxweave -- <trusted app python>
                                            -m bwmini.diagnostics read-report ...
                                            |
                                            +-- existing receipt tables
                                            +-- D1 diagnostic events + trace links
                                            +-- readonly SQLite connection (no Store migration)
                                    |
                                    v
                         bounded redacted report in operator-owned new path
```

Privilege constraints:
- `host_cli.py` and `online.py` are root-owned installed code, executed under isolated `python3 -I`; never load app-modifiable modules into root.
- The application uses `/home/boxweave/boxweave` with private config, installed release and SQLite. Launch a **fixed, reviewed app-owned reader as the app user** with no arbitrary executable, path or shell arguments accepted from the user.
- In split deployments the direct MCP gate may live on another host. Its private journal (if a separately approved D1-gate collector exists) is **not available** to the app's reader automatically.
- A parent Incus host is a separate trust principal; child reports must never claim to inspect it merely because the child observes an LXC environment.

Current `bwmini/host_cli.py` already contains public trust checks (`published_descriptor`), fixed `runuser` patterns (`mcp_probe`, `job_command`, `selftest_command`), exact `box inspect/diagnose` and no arbitrary root app import. Reuse the restrictions, not the `job` arbitrary command execution path. `bwmini/nesting.py` already classifies a proc-mount signature. `bwmini/selftest.py` can run a live sequence but intermediate steps are only returned at completion.

## 2. CLI contract and exit status

Define `boxweave report` with a mutually exclusive selection:
- `--kind {hello,environment,access,job,selftest} --operation ID`; `--kind` is necessary because existing operation IDs live in separate tables; to preserve earlier proposed `--operation ID` syntax it may default to cross-table *read-only discovery* only if the result is unique, otherwise return `AMBIGUOUS_OPERATION_KIND` without exporting;
- `--trace TRACE_ID` for a D1-linked trace that passes actor/operation lookup;
- `--since RFC3339-Z --until RFC3339-Z` as a bounded period selector for missing-ID crashes, both required together; no more than 24 hours in the initial review profile.

All selectors require `--output NEW_DIRECTORY`. No arbitrary `--log-path`, `--command`, `--config`, `--tenant`, `--raw` or network upload flag. Optional `--box EXACT_NAME --project PROJECT` can be introduced only after confirming it doesn't silently require privilege or read unrelated Incus instances. Initial D2 may keep exact `box diagnose` as separate operator-requested command.

Exit contract for **report creation** (independent from recorded operation outcome):
- `0` report generated, all selected **available** required sources read, schema verified, manifest finalized;
- `2` safe partial report generated with explicit omissions (`app_reader_unavailable`, `parent_unverified`, `gate_capture_not_installed`, time window not covered). For a report that declares a specific source required, partial is not full success;
- `1` cannot safely create an export, illegal selector, unauthorized/unknown trace, unsafe output path, or export write failure. If an incomplete directory remains, mark it `invalid_partial` or remove only that freshly owned temporary directory; never silently claim final success.

**An operation in `failed` state may have an export exit 0**; export completeness and operation outcome are separate concepts. Likewise, a successful export can contain `root_cause=not_verified`: completeness means the promised evidence was collected, not that diagnosis is certain. Define both `collection_status` and `diagnostic_conclusion` in the manifest.

`--json` should print only a bounded result (state, code, report path, counts, omissions), not the report body. Existing `doctor` and `status` remain independent.

## 3. Report file contract (proposed schema 1)

Create a new directory only; no overwriting existing report, symlink, hardlink, non-private parent, or path traversal. Set 0700 directory, 0600 files, atomically finalize after writing and fsync. Existing files under the chosen parent must remain untouched. If running with sudo, the result is root-owned by default; do not automatically chown it to an untrusted caller.

```text
<new-report>/
  manifest.json
  timeline.jsonl
  checks.json
  operation.json
  layers.json
  omissions.json
  README.txt
```

Schemas:

| File | Safe content | Limits / prohibited |
| --- | --- | --- |
| `manifest.json` | schema, collection ID, request selector type, UTC start/end, installed source and release *when independently observed*, complete/partial, attribution confidence | no private config, keys, connection string |
| `timeline.jsonl` | sorted D1 allowlisted events with capture and source scope | max e.g. 1,000 events / report; bounded line length and total bytes |
| `checks.json` | D0 per-check state/scope/code and collection time | no full health subprocess output |
| `operation.json` | receipt type, ID, revision/state/phase/recognized status codes, deadline, outcome uncertainty | redact `job_runs.argv`, workspace path, policy documents, arbitrary execution output |
| `layers.json` | local installed layer identity evidence, operator-attested or independently verified parent link, app/gate/provider distinction | never infer parent UUID/config from hostname/IP alone |
| `omissions.json` | missing sources, permission denied, time skew, schema mismatch, pre-D1 trace gap, dropped capture, truncated limit | absence of data must not be reported as negative proof |
| `README.txt` | safe handling, expected uncertainty, review-before-sharing warning | no copied operator input that may contain secrets |

Suggested manifest snippet (illustrative):

```json
{
  "schema_version": 1,
  "collection_id": "opaque-random",
  "selector": {"kind": "environment", "operation_id": "opaque-id"},
  "collection_status": "partial",
  "diagnostic_conclusion": "not_verified",
  "source": {"installed_release": "0.21.4", "runtime_attestation": "not_verified"},
  "scope": ["host_public", "app_user_sqlite"],
  "missing": ["gate_host_not_available", "parent_policy_unverified"]
}
```

A report created in fixture mode must label itself `fixture`, never `live_mcp`. The report includes a source provenance distinction: Git commit of source checkout, published installation commit, and *running* process release are different claims. Record only those actually observed; a previously produced `provenance.json` is not proof the currently running Worker loaded that generation. Combine process start time/unit MainPID with release identity only if inspected and validated. Sanitized config **digest** is not an attestation of applied config until worker restart/generation is observed.

Default proposed cap (owner to ratify): 2 MiB complete exported report, 1,000 timeline events, 24-hour time window, 10 seconds per host-side source and max 60 seconds total; stricter per-source budget for app reader, stdout and JSON depth. Implement incremental capped JSON serialization rather than assembling a multi-gigabyte collection then truncating. A cap breach becomes explicit `truncated` in omissions with partial status.

## 4. Collection algorithm and filesystem safety

### D2-1: host-safe orchestrator

1. Parse validated selector and output path. Refuse empty/malformed IDs, non-UTC ranges, reversed times, oversized windows, and unsupported operations **before** any filesystem write.
2. Validate root-owned public descriptor and installed CLI provenance. Use read-only `status`/D0 doctor subset for host metadata, never infer authenticated health from HTTP status.
3. Use a fixed, already reviewed app-user subprocess for `bwmini.diagnostics read-report`. Do not use `boxweave job -- arbitrary command` or an interpolated shell. Pass only serialized allowlisted selector and a max-byte budget; capture bounded UTF-8 output. No stdin credentials. Do not pass user-selected file names or output paths into app code.
4. The app-owned reader obtains a **read-only SQLite connection** to the configured DB with URI `mode=ro`, `PRAGMA query_only=ON`, bounded busy timeout and schema checking. **Never instantiate `Store()` or `__main__.load()` in the reader**, because `Store.__init__` performs schema upgrades/chmod and the diagnostic read must not migrate/modify state. Fail closed on unsafe/symlinked state path or wrong owner/mode.
5. The reader first verifies the authoritative receipt **and actor**, then its D1 trace link/events. It returns only typed allowed fields, no database row dumping. For time-window requests collect only a bounded set of allowed events; default local operator scope must not expose another actor's records.
6. Host CLI validates returned JSON schema and sizes, merges safe host and app reports; separately records unavailable sources. Do not trust arbitrary strings returned from a compromised app subprocess as shell commands/paths.
7. Write an isolated new private temporary directory under the reviewed output parent; write allowlisted files with exclusive create, fsync files and directory, finalize by renaming into a **nonexistent** final directory. Do not leave misleading `complete` manifest on interrupted export; mark and handle partial safely.
8. Print a small summary to stdout, not JSONL logs or unfiltered secrets.

Because the root CLI is also used by an ordinary operator with sudo, resist symlink races, attacker-controlled output path or inherited environment. Prefer directory-fd-oriented O_NOFOLLOW / O_EXCL and restrictive umask, owner/mode checks on every path component. Ensure no app-owned code is loaded with root UID, even when the app venv is missing or an old release is incompatible.

### D2-2: fallback when primary services have failed

If Flask is dead, do **not** query Flask as the only path. If Worker is dead, use read-only SQLite. If MCP is dead, no MCP call is necessary. If the installed app-owned interpreter cannot start or the DB is unreadable, produce a valid **host-only partial report** (when output remains safe) and record `app_data_unavailable`. If the public descriptor is invalid, collect only safely verified metadata, refuse private app access and mark validation failure; do not guess app prefix/UID.

Never fall back to reading app DB as root, reading unfiltered journald, or executing remote `incus exec`. Report omissions must name unavailable evidence and stop on invalid ownership. This is why CLI recovery remains useful when MCP cannot be used.

## 5. Nested environment provenance and attribution

The topology is not necessarily a strict L0/L1/L2/L3 chain; gate and Provider may be split peers. Model each **observed node** with:
- `node_ref`: opaque report-local identifier;
- `role`: `host`, `app`, `mcp_gate`, `provider` or `workload`;
- `identity_source`: `locally_observed`, `authorized_outer_observation`, `operator_attested` or `not_verified`;
- optional exact Incus project/name/UUID/creation fingerprint **only from authorized verified observation**;
- installed version/source and observed capability; `effective_nested_execution=not_verified` unless a real child workload proved it.

Edges `parent_of` or `depends_on` require evidence; do not infer from identical private IP, hostname, Python environment, clock proximity or configured `security.nesting`. Parent-host `security.nesting` and effective child mounts are different observations (ADR-008). Read-only `boxweave box diagnose NAME` can classify a specific proc mount denial via `nesting.diagnostic`, but this is **candidate cause**, not confirmed parent configuration.

For outer/inner diagnosis, an authorized operator should collect **separate** reports at each relevant layer and manually compare/share those reviewed files. D2 does not silently climb to a parent, use sudo on another host or auto-enroll parent identities. If offline correlation is later requested, implement as a **non-privileged** comparison tool over explicitly selected reviewed reports with schema/identity verification, not a new root `--parent-file` read of arbitrary paths.

**Split gate-host case:** RISK-620 reports that nesting helper may be absent where the gate runs. Distinguish three states in the report:
- upstream `create_container` acknowledged/succeeded;
- gate-local post-create `boxweave_nesting apply` failed/unverified due to helper host;
- exact created Box identity must be inspected before any future mutation.
Never advise repeating `create_container` merely because nesting post-config failed. `unknown` remote effect remains unknown until separately observed.

## 6. Durable selftest recording: D2-b sub-slice

D2-a should ship reliable reports first; then enable **D2-b** incremental selftest evidence. Current `selftest.py` performs live create / identity+hello / stop / delete and holds steps in memory. An unexpected process death can hide the last step. This subsection proposes persisting evidence **without enabling automatic recreation**.

Suggested `selftest_runs` row, added in a narrowly reviewed migration (likely schema v10, *after* D1 v9) if D1 events alone cannot provide sufficient durable run metadata:

```sql
CREATE TABLE selftest_runs (
  id TEXT PRIMARY KEY,
  actor TEXT NOT NULL,
  plan_digest TEXT NOT NULL,
  exact_test_box TEXT NOT NULL,
  state TEXT NOT NULL,
  created_utc TEXT NOT NULL,
  deadline_utc TEXT NOT NULL,
  last_step TEXT,
  cleanup_state TEXT NOT NULL
);
```

Do **not** store the private key, full plan or resource credential in `selftest_runs`. Link to D1 event trace by `operation_kind=selftest`. Before each mutating step persist `step_started` with run ID and exact target/intent and flush it. After returning from Provider persist observed result; if killed in between, record the step as **unknown** on next read, with no replay. Preserve a single total deadline and the existing plan/confirmation checksum. The documented caller must still confirm that create/stop/delete is an actual mutating test.

Example permissible step lifecycle:

```text
confirmed plan -> run created
create/step_started (durable) -> one provider create call
create/ack_observed or unknown
identity+hello/step_started -> actual identity/hello result
stop/step_started -> one stop call
delete/step_started -> one delete call
cleanup verified separately -> final state
```

**Crash test:** kill selftest after remote create and before reply; `selftest run` must not on restart auto-create another name, auto-delete an unverified Box, or assert the original was not created. Report original exact name/identity and a safe reconciliation prompt. If cleanup is incomplete, preserve original resource and explain no automatic retry authorization. Do not replace current best-effort cleanup behavior without a separate reviewed semantic change; any `--preserve-on-failure` option needs clear approval and tests.

Be careful that `host_cli.selftest_command` uses `runuser` with bounded stdout and currently records only returned step details. The test CLI must not issue a second `selftest-run` automatically when the subprocess times out. D2-b moves intermediate evidence to durable storage, not to a new long-running daemon.

## 7. Privacy model and hostile inputs

Data never exported by default:
- whole `state.sqlite3` or other DBs (including `sessions`, `admin`, `plans`);
- `config.json`, `runtime.json`, `.env`, private or public key material, client registration pin data, SSH argv and bearer tokens;
- `job_runs.argv` or arbitrary stdout/stderr; raw Flask request/exception messages; raw MCP tool JSON; full Incus expanded config/devices;
- journalctl output, user workspace files, user project source, model prompts, cookies, passwords;
- operator-provided arbitrary paths/text or untrusted Agent-supplied diagnostic interpretations.

Treat logs and tool messages as **untrusted data** and do not render embedded instructions as an action plan. Reports should contain observed fixed states/codes and a neutral omissions summary; analysis by an Agent is separate and must cite which claims are direct evidence. A report's field allowlist must be reviewed and tested with synthetic malicious strings and Unicode control chars; string-length-only redaction is inadequate.

Attack tests must cover: symlink traversal, changed parent permissions during export, hardlinks, extra file names, output existing, malicious SQL text/JSONL record, injection from source stderr, a forged cross-actor operation ID, excessive number of trace events, corrupt v9 schema, oversized single row, disk full, read locks and subprocess timeout.

## 8. Implementation work breakdown

| Slice | Code files | Deliverables |
| --- | --- | --- |
| D2-a1 | `bwmini/host_cli.py`, new approved root-owned helper only if needed | read-only `report` command, strict flags, secure directory creation |
| D2-a2 | `bwmini/diagnostics.py` or separate app-owned `diagnostic_reader.py`, `distribution.py` | app-user read-only DB reader, typed receipt scoping, bounded output, package included |
| D2-a3 | `bwmini/nesting.py`, `host_cli.py` (if proven necessary) | nested evidence schema, parent unverified semantics, no parent writes |
| D2-a4 | `docs/TROUBLESHOOTING.md`, `docs/CLI.md`, `docs/verification/REMEDIATION_TEST_QUEUE.md` | actual command help, report privacy/runbook, evidence and limits |
| D2-b1 | `bwmini/selftest.py`, `bwmini/store.py`, `bwmini/__main__.py` | durable per-step selftest metadata, migration if chosen, no replay |
| D2-b2 | `tests/test_selftest.py`, `tests/test_recovery.py` | kill-before/after effects and orphan reconciliation |

Package boundary: `distribution.py` has an explicit Python module archive list and `online.py` publishes fixed helper file allowlists; update those only for newly required modules and verify signed/reviewed deployment artifacts. The installed root executable cannot simply import a newly added app-owned diagnostics module.

Avoid introducing a web diagnostics endpoint in D2 unless separately authorized: this would add session/CSRF/data disclosure concerns unrelated to offline failure recovery. Do not add direct MCP diagnostics here.

## 9. Test and acceptance matrix

Suggested new `tests/test_diagnostic_report.py`, plus `tests/test_host_cli.py`, `test_nesting.py`, `test_selftest.py`, `test_recovery.py`, `test_distribution.py`. Use immutable fixture events/DBs, temporary private dirs and synthetic host processes before separately approved live acceptance.

| ID | Setup / fault | Required output and safety |
| --- | --- | --- |
| D2-T01 | D1 v9 trace + receipt + app available | private report with valid manifest/timeline, state correctly attributed |
| D2-T02 | Receipt failed but all diagnostic sources available | export exit 0; `operation.state=failed`, independent status |
| D2-T03 | Missing service / dead MCP, valid app DB | report works read-only; records relevant unavailable services |
| D2-T04 | Missing/unreadable app binary or DB | host-only partial/exit 2; no app code as root |
| D2-T05 | Old pre-D1 receipt with no trace | existing receipt exported, trace gap explicit |
| D2-T06 | Operation ID clashes across kinds or another actor | ambiguous refused; no cross-actor data |
| D2-T07 | Output existing, symlink/hardlink, world-writable parent, race | no overwrite, no created unsafe file, fail closed |
| D2-T08 | Selected 24h boundary, malformed time, reversed range | invalid input rejected before writing |
| D2-T09 | 1,001 events/oversized row/huge number of operations | caps, partial marker, no memory spike |
| D2-T10 | Synthetic JWT/key/password/policy body in DB/transport | absent from every exported file and console |
| D2-T11 | Missing gate evidence or split deployment | `gate_capture_not_installed` / unverified, no false full-chain claim |
| D2-T12 | Inner proc-denied with no authorized parent read | classified symptom; parent policy not_verified |
| D2-T13 | Authenticated parent report attaches verified identity in offline analysis | no fabricated link; no remote privilege escalation |
| D2-T14 | Post-create nesting helper fails | original create success preserved; no repeat advice |
| D2-T15 | Selftest killed before create, during create and after remote effect | durable step boundary; unknown reconciled without automatic replay |
| D2-T16 | Selftest cleanup fails, target still exists | preserve original target; disclose incomplete cleanup |
| D2-T17 | Export interrupted after some files | no misleading complete manifest or overwrite, owned partial handled safely |
| D2-T18 | SQLite locked / disk full / invalid schema | timeout limited; omit with reason; source DB never mutated |
| D2-T19 | Installed source/app process version differs from Git/main | accurate separate observed provenance and uncertainty |
| D2-T20 | Reporting after 7-day retention/quota | report explicitly says missing historical events; not an invented conclusion |

Additional invariant test: byte-for-byte hash the SQLite DB before/after `report` in a controlled no-concurrent-writer fixture (and check page journal modes appropriately); verify report never instantiates `Store()`, never attempts migration and never calls a Provider mutation. A read-only SQLite connection may have external concurrent writers in real tests; hash comparisons there cannot prove who wrote data, so use isolated fixtures.

### Required validation ladder

1. Static: `python scripts/check_knowledge.py` and link check.
2. Prepared non-root fixtures: focused `unittest` modules, `.venv/bin/python scripts/validate.py` and approved browser lane only if touched.
3. Packaging: build release, check shipped reader/host helpers and help text from exact archived release.
4. Disposable authorized host: run real installed `boxweave doctor/report` with stopped Worker, dead MCP and missing app-reader scenarios **without** modifying unrelated installed services.
5. Separately authorized nested LXC: collect independent outer/inner reports, real MCP identity+Python, and proc-mount failure if safely reproducible; never alter parent security or restart unrelated instances merely to make a test pass.
6. Explicitly approved selftest mutation only: plan, review exact target, run once, inspect original target after timeout; preserve all incident artifacts.

Each test entry records commit/installed release, environment, exact command, exit/status, redacted observations, known gaps, fixture vs live, cleanup and affected REQ/AC/RISK. No live claim from fixture or CI alone.

## 10. Delivery / rollback criteria

**D2-a done:** a secure, bounded report on a live app trace and a useful host-only partial report when app/MCP is broken; no cross-layer authority expansion and no raw secrets. Reports correctly distinguish **observed**, **operator_attested**, **inferred** and **not_verified**, and report export itself cannot trigger execution.

**D2-b done:** confirmed selftest stores phase evidence durably before each effect, does not auto-replay remote mutations, and reports uncertain/failed cleanup and exact original identity. If D2-b migration v10 is needed, older binaries must refuse it; test upgrade/backup/rollback boundaries separately.

**Deferred:** automatic repair, generic user-project debug, Web project log viewer, direct MCP trace reader (D3), background collection agent, unrestricted host log export or cloud ingestion.

Rollback requires reviewed existing installer/DB backups and version consistency; do not run v8/v9 code against a migrated v10 DB. Report files are user-reviewed private outputs; never delete old incident evidence through normal app upgrade or cleanup.

## 11. Handoff for the next developer

Before editing, reconcile this proposal with latest main code, chosen D0/D1 output/schema and authoritative `REQ/AC`. Decide exact retention, byte/time limits and allowed operator scope; freeze `report` JSON and exit contracts with negative tests. D2-a should be code-reviewed separately from D2-b because the latter modifies a live mutating test runner. Update `FEATURE_MATRIX`, `SOURCE_REGISTER`, `OPEN_RISKS` and the append-only verification queue **only for actually implemented/evidenced slices**, and keep this design document labelled as proposal until acceptance.
