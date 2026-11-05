# BoxWeave v0.47.0 reacceptance engine — executable acceptance and release gates

**Status:** **v0.47.0 RELEASED and scoped installed-engine acceptance verified** (2026-11-05). This document distinguishes executable engine checks from full feature/action reacceptance; untested MCP/CLI/Web actions are not PASS. Product's regular operation and operator-host policies are unchanged. **Canonical owner trigger remains** `Run full BoxWeave reacceptance`.

**Authorities:** [full runbook](../process/FULL_PROJECT_REACCEPTANCE.md), [architecture and staged milestones](../design/REACCEPTANCE_EXECUTION_ORCHESTRATION_PLAN.md), [upstream v0.128.0 reference](UPSTREAM_CONTAINARIUM_REACCEPTANCE_GUIDE.md), [evidence template](REACCEPTANCE_RUN_TEMPLATE.md), [function baseline](REACCEPTANCE_FUNCTIONAL_COVERAGE_BASELINE.md).

## 1. Delivered source components and explicit boundaries

| RACC item | Actual executable source | Supported scope | Boundaries / evidence |
| --- | --- | --- | --- |
| AUTO-01 runtime/surface/drift | `bwmini/reacceptance.py inventory`, `docs/verification/REACCEPTANCE_DEPENDENCIES.json` | AST host early dispatch and nested actions, 36 app modes, Web endpoints, **actual installed stdio tools/list**, CLI help and provenance, optional pinned upstream proto declarations, changed-path feature impact and previous inventory diff | Actual SSH/HTTP/Web positive needs authorized real clients; proto route declaration is source-only. An unrecognized action remains mapped only to `UNMAPPED`, not PASS |
| AUTO-02 guarded case execution | `inventory`, `begin`, `run-safe`, `record` | Strict allowlisted non-root host version/doctor and actual stdio MCP read calls, import **independently executed** positive/invalid/denied with original result and expected oracle; fails closed for unexpected command vectors | No generic shell from untrusted JSON; all other mutators remain Agent-orchestrated through authorized BoxWeave MCP. Catalog-only or config-error only cannot PASS |
| AUTO-03 concurrency and failure | `bwmini/reacceptance_live.py` | Explicit test-owned nested Provider host, immutable outer identity/host machine-id/expiry manifest, first selftest actually in-flight, second refused with no child, original cleanup and fresh after-contention success; preserve UNKNOWN | No owner-host R-04/reboot or arbitrary parallel mutation; scenario is **only** installed selftest, not all state machines |
| AUTO-04 isolation | `verify_lab` and exact child review, outer BoxWeave owned Box allocation contract | Deny missing/expired/wrong-host lab; prevent rerun on unknown or existing selftest child | General-purpose Box create/cleanup provisioner for all profiles is **not shipped**; use existing outer authorized MCP and independent readback |
| AUTO-05 candidate evidence | `candidate-attest`, `candidate-verify` + existing CI package build and stable manifest | Verify source SHA, SHA256 exact package bytes, required case statuses when configured, compare released manifest source/digest | Candidate attestation is **not by itself** installed black-box gate; current CI still publishes automatically after passing main verify; independent installed candidate/release gate is Coordinator-controlled |
| AUTO-06 upstream contract | `inventory --upstream-tree` and source/installed maps | Read typed native protobuf HTTP route declarations vs actual stdio MCP catalog and help, source/installed differences | Direct native error parity probes and live Web HTTP authenticated requests require separate authorized case execution; recording via `record` is supported |
| AUTO-07 proof, claims and reports | `proof-review`, `bwmini/reacceptance_proof.py`, `self-test`, `seal`, `verify`, `report`; dependency JSON | Reviewed root-in-child Proof Pack adapter, private immutable per-attempt receipts, stored digest checks, seeded false PASS/log/oracle control, complete per-action and **49 named feature** table, issue handoff JSON | This is tamper detection **inside the retained local evidence set**, not an external cryptographic signature or independent Agent. Upstream generated pack is never considered a black-box feature PASS |

**No new public HTTP endpoint, generic privileged MCP tool, credentials or production scheduler** was added. Operator still chooses the test Box and Agent roles. The canonical one-sentence trigger is the orchestration contract; the tools are callable primitives that make it more reliable, not a replacement for Finder/Fixer/Verifier.

### Packaged registry and one-call Agent bootstrap

The **production release ZIP now bundles** the pinned 49-feature acceptance baseline and source-to-feature dependency rules under `bwmini/reacceptance_data/`. When the runner operates from the installed application tree without a Git checkout, it reads these embedded reference bytes and derives the source SHA from the immutable ZIP `manifest.json`. Missing/invalid embedded data **fails closed**, rather than silently producing a zero-feature report. Git working trees record `clean` versus `modified`; installed packages verify the critical bundled source/data digests against `manifest.json`. Dirty or unverified worktrees cannot issue a candidate attestation.

For a newly assigned Agent, the safe first call can be a single *read-only* bootstrap:

```bash
python -m bwmini.reacceptance bootstrap-readonly --root /pinned/source-or-installed/tree   --host-argv-json /private/host.json --mcp-argv-json /private/mcp.json   --out /private/new-reacceptance-run
```

This performs discovery and the allowlisted actual read-only probes, records immutable original attempts, and leaves the run **unsealed for Finder/Verifier** to import real mutation/negative tests. It never treats bootstrap as full reacceptance; call `record`, `inspect`, `seal`, `verify`, and `report` after the subsequent cases. Existing BoxWeave MCP `connect(exec)` in a test Box is the authorized invocation channel; there is no new generic root execution endpoint.

The candidate attestation now **requires at least one explicitly named required action whose original proof is PASS at the same source SHA**; empty required-action lists and source mismatches fail closed. This attestation still proves only bytes and recorded per-action evidence, not independent installed candidate acceptance unless the separate Verifier has actually performed it.

## 2. Safe inventory and metadata collection

Run as the **test Box non-root user** from a clean, pinned BoxWeave source checkout (or as the application user within an installed v0.47.0 environment). The script wrapper is included for source checkout convenience; the release package ships the Python modules.

```bash
# From the checked-out BoxWeave repository. Use a NEW private test-owned path.
mkdir -m 700 -p .runtime/reacceptance-test
python scripts/reacceptance.py self-test

# Source inventory does not invoke any Provider write or assume MCP was tested.
python scripts/reacceptance.py inventory --root . \
  --out .runtime/reacceptance-test/catalogue.json

# Optional installed runtime probes: argv files are **reviewed JSON vectors**,
# never shell snippets. The MCP command must launch the installed gateway.
printf '%s\n' '["/usr/local/bin/boxweave"]' > .runtime/reacceptance-test/host.json
printf '%s\n' \
  '["sudo","-n","-u","boxweave","/usr/bin/python3","/usr/local/lib/boxweave/default/mcp_gate.py"]' \
  > .runtime/reacceptance-test/mcp.json

# Only use authorized installed binaries on a disposable test Box.
python scripts/reacceptance.py inventory --root . \
  --host-argv-json .runtime/reacceptance-test/host.json \
  --mcp-argv-json .runtime/reacceptance-test/mcp.json \
  --upstream-tree /reviewed/pinned/containarium-checkout \
  --out .runtime/reacceptance-test/live.json
```

The output distinguishes **source and installed runtime**. If installed provenance source differs from local Git HEAD, hidden actions are a *cross-version discrepancy to investigate*, not an unqualified defect in the source. The upstream tree is source-only until its actual version and route are probed. Local/stdio is not remote SSH/HTTP/OAuth verification.

### 2A. Running only unprivileged safe cases and recording external tests

```bash
python scripts/reacceptance.py begin \
  --inventory .runtime/reacceptance-test/live.json \
  --out .runtime/reacceptance-test/run-A

python scripts/reacceptance.py run-safe \
  --run .runtime/reacceptance-test/run-A \
  --host-argv-json .runtime/reacceptance-test/host.json \
  --mcp-argv-json .runtime/reacceptance-test/mcp.json

# Finder independently executes real additional action cases (via authorized
# BoxWeave outer MCP or installed clients), preserves original stdout/exit and
# independent Provider observer. Import a bounded JSON case without executing it:
python scripts/reacceptance.py record \
  --run .runtime/reacceptance-test/run-A \
  --action stdio_mcp:boxweave_info --category invalid --input case-invalid.json

python scripts/reacceptance.py inspect --run .runtime/reacceptance-test/run-A
python scripts/reacceptance.py seal --run .runtime/reacceptance-test/run-A
python scripts/reacceptance.py verify --run .runtime/reacceptance-test/run-A
python scripts/reacceptance.py report --run .runtime/reacceptance-test/run-A
```

The record file must describe the **actual** `observed` invocation result (`execution`, `exit_code` and `stdout` or `response`), independent `oracle`, `expected`, `actor` and exact immutable `target` for writes. Never reconstruct missing output. There are **no positive-pass defaults**. For mutation, a second observer must count effects and confirm cleanup; the denial path must prove zero forbidden second effects. `UNKNOWN` requires stopping related writes and preserving original identity. A sealed record refuses overwrite, detects changed or injected cases and produces `next-agent.json`. A malicious writer who can rewrite **all** files and hashes may still forge a local seal; the Verifier must independently capture/anchor evidence.

## 3. Exactly scoped real concurrency and upstream Proof Pack

### 3A. Installed selftest race in a disposable Box (specialized tool)

The owner or Coordinator first creates a **test-only nested Provider host** with the authorized outer MCP, verifies actual UUID/createdAt and machine-id, installs the exact candidate, then writes a private test-owned approval JSON with:

```json
{
  "scope": "test_owned_nested_selftest_only",
  "approved": true,
  "hostname": "<observed-test-Box-hostname>",
  "machine_id": "<observed-/etc/machine-id>",
  "outer_uuid": "<verified-outer-provider-UUID>",
  "outer_created_at": "<verified-provider-createdAt>",
  "expires_epoch": 1793919600
}
```

Expiry must be after current time and less than 24 h away; the file must be owned by the calling user and mode 0600. This **identity file is not a signed authorization token**: its trust derives from owner-approved isolated execution, BoxWeave's exact-target outer controller and a separately observed Provider identity. Do not use it on an owner host.

```bash
python -m bwmini.reacceptance_live \
  --lab /private/owned-lab.json \
  --out /private/new-selftest-race-evidence \
  --confirm I_AUTHORIZE_TEST_OWNED_NESTED_SELFTEST
```

The runner never retries UNKNOWN, never removes another actor's Box and refuses an existing nested selftest child. It records original/competing/later plan, exit, original in-flight proof, independent Incus no-second-child and final absence. R-04, owner-host reboot and other race families remain independent tasks.

### 3B. Optional Containarium root-in-child Proof Pack

The **host/server** executable `containariumd` v0.128.0 advertises `proof-pack generate`; ordinary `containarium` client does not. Upstream generated packs execute declared checkout commands **as root inside the child Box**. This is an **extra risk**, not a general substitute for a BoxWeave MCP case.

Review `.github/containarium.yml` in the exact test-owned child checkout. The adapter allows only the narrow, known-safe `python -m unittest discover -s tests` command form; shell pipelines, implicit `make test` fallback or unreviewed scripts are refused. Independently verify the nested child identity through local Incus, matching checkout commit and declaration bytes, and an approved lab manifest. The source file remains code that could be malicious: execute **only** on test-owned disposable children with explicit operator authority.

```bash
python -m bwmini.reacceptance proof-review \
  --checkout /reviewed/checkout --declaration /reviewed/.github/containarium.yml \
  --child racc-owned-child-container

# Deliberately separate, explicit opt-in specialized executable:
python -m bwmini.reacceptance_proof \
  --lab /private/approved-nested-host.json \
  --child-identity /private/exact-owned-child.json \
  --declaration /private/reviewed-declaration.yml \
  --source-commit <verified-40-char-Git-SHA> \
  --out /private/new-proof-pack-attempt \
  --confirm I_AUTHORIZE_REVIEWED_ROOT_TESTS_IN_DISPOSABLE_CHILD
```

The host-side command runs only **after** all guards and only on the original exact child. It records `GENERATED_NOT_SEALED`; it does not silently upload audit data, seal the pack, run unrelated project operations or promote MCP actions to PASS. If the installed upstream does not support the command or root-in-child cannot be reviewed safely, label Proof Pack **BLOCKED** and use the non-root BoxWeave case recorder.

## 4. Candidate package and release acceptance

```bash
# Source candidate/package built by existing BoxWeave distribution tooling.
python -m bwmini.reacceptance candidate-attest \
  --package /private/candidate/boxweave.zip \
  --source <candidate-source-sha40> \
  --inventory /private/candidate/required-actions.json \
  --run /private/verified-run \
  --out /private/candidate-attestation.json

python -m bwmini.reacceptance candidate-verify \
  --attestation /private/candidate-attestation.json \
  --package /private/candidate/boxweave.zip \
  --release-manifest /private/release.json
```

This verifies a source-bound byte comparison and required action results. **It cannot by itself claim an installed candidate or independent verifier**: those are separate human/Agent evidence gates and are reported as such. The main CI publishes on a successful main push; retain the existing release workflow, and stage isolated premerge package acceptance before advancing main. Keep `PUBLISHED_RELEASE` unchanged unless the owner separately orders promotion. Never infer release success from a local package check.

## 5. Built-in adversarial controls and status handling

**Additional defect exposed by the complete source-suite run:** the R-04 UNKNOWN-recovery CLI occasionally rejected a legitimate one-use challenge because upstream-style base64url may begin with `-` and be parsed as an option after `--challenge`. v0.47.0 candidate now prefixes the 43-character challenge with `r` (248 random bits remain) and adds a deterministic regression forcing the RNG to return a leading-dash token. This does **not** force-release a task, change UNKNOWN semantics or replay a remote operation. Verify the final full suite and installed CLI separately before marking the release PASS.



`python -m bwmini.reacceptance self-test` intentionally tries to pass forged success, modified original log, missing observer, incomplete positive/invalid/denial triad and unverified mutation. **All five must be rejected (status DETECTED)** before trusted report generation. The CI checks this and verifies source/top-level help discovery. Per-action results are never inferred from the number of listed MCP tools. The complete 49-feature report includes each historical function even when its present action evidence is missing; an individual action PASS is not automatically a whole-feature PASS.

### v0.47.0 official release and scope-specific verification closure — 2026-11-05

This table records **actual test evidence**, not the fictional outcome of 235 completed black-box calls. The untested feature/action ledger remains in the private test-owned verification environment; the below source, runtime and publish claims are independently observable.

| Gate | Verified result and boundaries |
| --- | --- |
| Official source and release | **PASS**: v0.47.0 immutable release tag `575c77f384b3b55b50626b356ecf190738dd80ef`, [PR #85 merged](https://github.com/vanishs/boxweave/pull/85), official [Release v0.47.0](https://github.com/vanishs/boxweave/releases/tag/v0.47.0) non-draft/non-prerelease with **26 assets** |
| PR and main CI | **PASS**: candidate PR verify job succeeded, [main verify and publish workflow 42125789245](https://github.com/vanishs/boxweave/actions/runs/42125789245) both completed successfully; full isolated source suite **814/814 tests PASS**, documentation knowledge checker **228** documents PASS |
| Artifact bytes and source | **PASS**: official GitHub `boxweave.zip` asset SHA256 **`02f043c76be1d991b59064a1072d47219d98b30cbe563065db864d5c303645e2`**, identical to privately built, isolated-installed and accepted candidate ZIP; `release.json` asset digest **`29134b74f3002efaea9e2269e1e7b1002cad5881ee7fb9f0ebd1016b10d501b3`** matches the separately built candidate manifest |
| Official v0.46.2 → v0.47.0 installer | **PASS** in disposable `bw-racc0470-verify-container`, exact outer UUID `f1e555f9-55b1-455c-bccf-3537a13d2610`, createdAt `1793914037`. v0.46.2 official baseline exit **0**; prepublication v0.47.0 application ZIP upgrade exit **0**; official v0.47.0 `install.sh --version 0.47.0` exit **0**. The downloaded installer SHA256 **`9945075e0958c93aa91bc0acfd5de31ba8fe09dccce77bd26ccf774b5fd1bcd5`** was checked before execution |
| Installed identity and health | **PASS** after official installer: **host CLI v0.47.0** and **app v0.47.0**, both source `575c77f...`; actual installed app archive SHA256 `02f043c...`; `boxweave doctor --mcp` **10/10 required checks PASS**; five hidden CLI help families remain advertised. Upstream daemon v0.128.0 |
| Installed runtime discovery | **PASS for discovery only**: 68 real installed local-stdio MCP tools, 49 named features, **235 distinct discovered action entries** (includes separate host, app, HTTP/Web source views). In the **official installed** `bootstrap-readonly` evidence run, **4 PARTIAL, 231 BLOCKED, 0 PASS, 0 FAIL** because safe reads do not satisfy full positive/invalid/denied and independent state oracles. Exact bundled data and host source SHA were verified, with `host_source_matches_checkout=true` |
| Evidence and anti-false-PASS | **PASS** for implementation tests: five seeded invalid claims rejected; installed `seal/verify` returned `VERIFIED` with evidence-root SHA256 **`0edee3a7c081b62fa2de64e4792ed09b256718dd8e2906fcef3a60f9fc97fa0b`**. This is local integrity checking, **not an external signature or independent-Agent attestation** |
| Isolated two-process selftest black-box | **PASS** in final prepublication candidate, under test-Box-only permission: original `boxweave-selftest-fa6d01` in flight, competitor `boxweave-selftest-31c792` rejected with **exit 2** and no second child, later `boxweave-selftest-633324` **exit 0** and all four steps successful; final Incus child inventory **0**. Earlier pre-hardening candidate also passed a separate 3-actor sequence. This validates the specialist probe, not every R-04 state-machine behavior |
| Optional upstream Proof Pack | **NOT EXECUTED / CONDITIONALLY BLOCKED**: only host-side `containariumd proof-pack generate --help` and implementation guard regressions were checked. Root-in-disposable-child command execution requires separate reviewed declaration, exact owned child and approval; no pack was generated/sealed or independently verified |
| Other transports and general action acceptance | **PARTIAL/BLOCKED**: positive authenticated HTTP/Web, external/native clients, upstream route/error parity, and per-MCP/CLI positive/invalid/denied mutation triads remain to be executed by Finder/Verifier during the next **full** reacceptance. No statement of all-68-tool functional PASS is justified |
| Legacy Compose #79 / missing-target #81 | **UNCHANGED / NOT FIXED HERE**. Preserve the existing upstream-attribution and error-parity issue records; the engine reports/probes these surfaces but does not silently cure their native behavior |
| Stable installer pin | **UNCHANGED**: `PUBLISHED_RELEASE` is still **0.43.1**. v0.47.0 is available explicitly by version; it is not a default/stable promotion |
| Test-environment cleanup | Pending explicit independent final Box stop/delete readback, recorded after exact target identity check |

The protected official release remains tied to immutable source `575c77f...`. Any later **documentation-only** verification update to `main` must not relabel this historical tag as the new branch head or replace release assets. The repository release workflow explicitly permits documentation-only updates while preserving the existing version's immutable source and tag.

**Next owner invocation:** keep using `Run full BoxWeave reacceptance`. The engine can do safe read-only discovery and per-action evidence validation, but a separate Finder and Verifier still must execute meaningful black-box cases with real independent postconditions. Count a missing/oracle-less context as BLOCKED/PARTIAL, **never PASS**.
