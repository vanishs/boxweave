# Agent-managed reacceptance and direct publication (no release workflow)

**Status:** Owner's controlling direction as of 2026-11-05 PT. The old required
GitHub evidence-submission workflow, signature key, protected environment and
dedicated runner are **retired** as the default release mechanism. The regular
main/PR GitHub Actions build/test job may remain as an optional source check and
candidate producer, but is not an acceptance executor or release authorizer.
No special GitHub account is required to *record or validate evidence*.
Publishing a GitHub tag/Release unavoidably needs a repository writer identity;
use existing authorized Agent/maintainer access, never create or collect extra
credentials merely to perform acceptance.

## Single owner instruction

`Run full BoxWeave reacceptance` means: discover all actual callable MCP and CLI
leaves; prioritize real L2/L3 workflows and concurrency/fault boundaries; reproduce
defects with immutable original attempts; fix and run regressions; obtain the exact
candidate; hand it to a **different Agent in a separate authorized Box** to
reproduce and independently observe; repeat until accepted; publish only when
all applicable mandatory safety/functional release cases pass. Maintain complete
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
6. Coordinator reviews the actual functional ledger and incident queue: block
   release if any mandatory case is untested, partial, failed, UNKNOWN, false PASS,
   or an applicable release-blocking defect remains. A local checker PASS alone
   does **not** grant release permission. Keep authorized non-release work moving.
7. After genuine independent acceptance **and** a version-specific release
   decision under existing owner authorization, the Agent directly creates an
   immutable tag and draft GitHub Release from the frozen candidate. No release
   workflow, evidence uploader, new account, signer, or protected environment is
   necessary. This is a manual Agent operation using existing repository write
   access, not an unattended release service.

## Direct release checklist (Agent operations, not an Actions gate)

- Ensure the clean source SHA is on the intended `main` commit and every
  `release/` file equals the accepted index. The version must not conflict with
  an existing tag/Release. Never overwrite an existing public release or
  relabel another source commit.
- For a **new** tag, use existing authorized repository credentials to create
  `refs/tags/vVERSION` at the exact accepted SHA, then create a **draft**
  GitHub Release with the exact already-accepted asset files. Prefer the
  repository's existing `gh` command or connected GitHub tool; do not provision
  a second account or pass tokens via logs.
- Download the draft assets independently and compare the full filename set and
  SHA256 index against the frozen candidate. On mismatch keep the draft blocked;
  never silently replace an immutable published asset.
- Publish the draft only after exact draft parity. Download public assets
  again, verify tag SHA/version and every asset byte. Record links and remaining
  non-release risks. Stable installer selection remains separately authorized.

If no authorized GitHub write path exists, record `PUBLICATION_BLOCKED` but
do not label the underlying real installed acceptance as failed. Do not bypass
repository permission checks or fabricate a release. This rule is about release
authentication, **not** a demand for an extra GitHub authorization step during
testing.

## Scope and inherited safety limits

The original 20-scenario checker is conservative; a missing or genuinely
unsupported mandatory scenario blocks formal full release acceptance until
there is genuine proof or a separately reviewed **scope decision**, not a
faked PASS. A private Agent test report can describe partial installed
capabilities without claiming a public release gate passed. The former signed
workflow design remains historical knowledge; existing product release package
must not include developer-only `tools/reacceptance` code. R-02 host reboot,
production R-04, new credentials, public networking and original UNKNOWN
recovery remain outside ordinary test authorization. The canonical
[FULL_PROJECT_REACCEPTANCE](FULL_PROJECT_REACCEPTANCE.md) continues to govern
complete action/feature evidence.
