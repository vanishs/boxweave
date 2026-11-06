# Agent-managed reacceptance and lightweight GitHub CI releases

**Current owner decision (2026-11-06 PT):** Restore the original simple
GitHub Actions publishing model: on a successful `main` push, ordinary CI
automatically publishes its exact candidate as a GitHub tag and Release.
The publishing job uses the built-in `GITHUB_TOKEN` with scoped
`contents: write`; no separate verifier workflow, signature key, special
account, protected environment, evidence uploader, or dedicated Runner.
It verifies source SHA, version, assets and hashes, and never changes the
stable installer pointer. GitHub CI publication is **not** installed acceptance.

Full multi-Agent black-box reacceptance remains an independent owner-invoked
quality process. Earlier "no release workflow" and "direct Agent publication"
instructions in this repository are superseded.

## Single owner instruction

`Run full BoxWeave reacceptance` means: discover all actual callable MCP and CLI
leaves; prioritize real L2/L3 workflows and concurrency/fault boundaries; reproduce
defects with immutable original attempts; fix and run regressions; obtain the exact
candidate; hand it to a **different Agent in a separate authorized Box** to
reproduce and independently observe; repeat until independently accepted.
Publishing a source-CI Release does not itself establish that verdict. Maintain complete
named action and feature ledgers; discovery, unit fixtures, synthetic oracles and
same-Agent rechecks cannot claim installed independent PASS. Unsupported optional
scenarios need native capability proof, not unsupported-by-assertion.

## Minimal steps and evidence contract

1. Coordinator records exact clean source SHA, version, candidate package SHA256,
   complete release asset index and permitted Box UUID/incarnation boundaries.
   Build one candidate or obtain its existing artifact. **Freeze its bytes before
   independent installation.** Never rebuild different bytes under an accepted
   digest/tag. Regular CI success is useful but not a substitute for installed proof.
2. Finder executes actual supported black-box operations (positive, invalid and
   meaningful denied cases), preserves original stdout/stderr/exit, request/receipt,
   post-state and native readback. Record every MCP/CLI action and unresolved
   blocker. Snapshot original UNKNOWN identities; no replay, adoption, forced
   unlock or implicit deletion.
3. Fixer repairs confirmed root causes, runs focused/full regressions and produces
   a fresh exact candidate when source changes. A changed candidate invalidates
   earlier installed acceptance for the changed behavior.
4. A distinct Verifier Agent on another explicitly authorized Box installs **those
   exact candidate bytes**, runs the same repro plus all applicable scenarios and
   independently records native state and raw structured case reports. The
   Coordinator retains the unchanged Finder original receipts and Verifier output
   privately; do not place credentials or sensitive logs in public repository
   artifacts. Separate context/Box must be actual, not two names in one fixture.
5. Optional **local-only** integrity check on a prepared developer checkout:

   ```sh
   python -m tools.reacceptance.local_acceptance \
     --assets PRIVATE_EXACT_RELEASE_ASSETS \
     --evidence PRIVATE_REACCEPTANCE/release-gate.json \
     --source EXACT_CLEAN_SOURCE_SHA
   ```

   Evidence directories use the pre-existing 20-scenario receipt graph and
   `cases/SCENARIO/{original-attempt,oracle,installed-report}.json` (or explicitly
   supported native capability files). The command validates source, package,
   release manifest, every release asset's SHA256, actor/Box differences, graph
   status, bounded case file identities and hashes. It never runs tests, accesses
   GitHub, authenticates external Agent claims, signs data or publishes. Its
   positive verdict is `LOCAL_EVIDENCE_CONSISTENT_NOT_RELEASE_AUTHORIZATION`.
   The verifying Agent must manually confirm evidence is genuine and complete.
6. Coordinator reviews the actual functional ledger and incident queue:
   block a **full independent installed acceptance claim** if any mandatory
   case is untested, partial, failed, UNKNOWN, false PASS or an applicable
   safety defect remains. A local checker PASS alone does **not** authorize
   an installed PASS. This evidence decision is **separate** from the ordinary
   source-CI GitHub Release. Keep other authorized work moving.
7. Record full installed acceptance separately; a source-CI GitHub Release
   does not change any incomplete, failed, or UNKNOWN independent test verdict.

## HTTP MCP async-create acceptance (incident #92)

For `create_container` through ChatGPT HTTP MCP, verify the **original
one-dispatch receipt** and exact duplicate/replay, then read-only poll the
original operation; a later RPC ID is not a reliable idempotency identity.
`boxweave_create_status` is not always advertised by capped client
tool catalogs. The v0.49.4 fallback is `create_container` with only
`username` and `status_only=true`; this must not write or reserve a name.
Test invalid upstream-typed arguments before reservation: the wrong type
must not leave a permanent tombstone for a never-dispatched Box.

While `state=submitted`, a transient STOPPED instance, missing guest user
or `TRUST_NOT_READY` is not a terminal failure. After upstream
`succeeded`, independently verify exact native Box, in-guest username/HOME,
SSH and any explicitly requested nesting. Do not equate native RUNNING or
an upstream acknowledgement with end-to-end readiness. Preserve the exact
original UNKNOWN task and Incus incarnation and never replay it.

The [real v0.49.3 scoped report](../verification/HTTP_MCP_CREATE_0493_REAL_2026-11-06.md)
is **not** proof of v0.49.4 installed behavior or of #92's full root cause.

## Automatic release (ordinary CI, no extra acceptance gate)

- `.github/workflows/ci.yml` first runs its regular `verify` job and
  generates `application-candidate`. Only a successful `main` push can
  continue to `publish`; pull requests cannot publish.
- The publish job checks that `main` has not advanced, validates the
  `VERSION`, exact source SHA, full asset names and SHA256 hashes, then uses
  the same-run candidate bytes.
- `GITHUB_TOKEN` has `contents: write` only for publication. Create
  `refs/tags/vVERSION` at that exact commit, then a draft GitHub Release.
  Do not retarget existing tags or replace published files. For a new
  version, increment `VERSION` and corresponding package versions.
- Upload all release assets, independently download draft bytes and compare
  hashes, publish, then download and recheck public bytes. Interrupted
  drafts may resume without silently replacing existing files.
- Release notes explicitly distinguish source CI from installed independent
  acceptance. The stable installation pointer remains unchanged.

## Scope and inherited safety limits

The original 20-scenario checker is conservative; a missing or genuinely
unsupported mandatory scenario blocks **full independent installed acceptance**
until there is genuine proof or a separately reviewed **scope decision**, not a
faked PASS. It does not block ordinary source-CI publication, which cannot
claim installed acceptance. A private Agent test report can describe partial installed
capabilities without claiming a public release gate passed. The former signed
workflow design remains historical knowledge; existing product release package
must not include developer-only `tools/reacceptance` code. R-02 host reboot,
production R-04, new credentials, public networking and original UNKNOWN
recovery remain outside ordinary test authorization. The canonical
[FULL_PROJECT_REACCEPTANCE](FULL_PROJECT_REACCEPTANCE.md) continues to govern
complete action/feature evidence.
