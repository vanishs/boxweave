# v0.49.0 source-only reacceptance MVP checkpoint

**State: PARTIAL / NOT RELEASE-ELIGIBLE.** Source checkout based on main
`7a7ac62755675c37961c595fdb7a6b3fa27600c1`, before subsequent
candidate CI and code review. This record does not certify any publication.
Stable installer pin remains v0.43.1.

## Implemented within the developer-only plane

- `tools/reacceptance/installed.py` performs real installed stdio MCP
  catalogue discovery, reviewed read-only positive/invalid calls, installed
  CLI help and read-only positive/invalid probes. It never auto-dispatches a
  mutation and never marks a catalog/response alone PASS. Missing credentials,
  native version, HTTP/Web authentication or independent observation remains
  per-action BLOCKED/PARTIAL.
- `tools/reacceptance/lifecycle.py` adds bounded test-owned nested-host
  create -> native read -> stop -> read -> start -> read -> delete -> read.
  Private grant binds machine ID, hostname, outer incarnation, one unique
  test-owned child and expiry. Durable write intent precedes every external
  call. Any ambiguous response blocks further writes and retains the
  original identity for manual recovery. Native Incus is its independent
  observation; no host privilege is added to the product.
- GitHub candidate gate and publish jobs now both consume the artifact from
  one explicitly selected **successful main-push CI run for the same commit**.
  The dispatch rebuild is not used as a replacement candidate. A genuine
  separate trusted signed Verifier run and protected release environment
  are still mandatory and **not configured or proven by this code**.
- Normal customer ZIP continues to exclude `tools/reacceptance` and
  all source-only reacceptance shims/data.

## Actual source/isolated baseline evidence

- Original v0.47.0 official installer was launched on test-owned
  `bwracc-mvp-verify-container` (outer UUID
  `b95d62fe-6914-41e2-bbc7-8c6d122a844c`; createdAt `1793931912`),
  on a separate Ubuntu 24.04 Box with nesting enabled and restarted.
- The official v0.47.0 installation reached a working
  `sudo boxweave version` result `0.47.0` and
  `sudo boxweave doctor --mcp` exit 0, **10/10 checks passed**.
  Native Containarium version `0.128.0` was read. This certifies the
  **old baseline only**, not any v0.49.0 candidate or its upgrade.
- The new safe runner invoked without installed credentials reported
  169 per-action rows, zero independent oracles and
  `PARTIAL_INSTALLED_READONLY_NOT_SYSTEM_ACCEPTANCE`. This is a
  missing-transport control, **not 169 executed or successful probes**.
- The lifecycle and installed probes have targeted **synthetic unit
  fixtures** for rejection, no-replay, mock native reads and false PASS.
  No real lifecycle or independent external Verifier PASS is claimed at
  this document checkpoint.

## Must be done before a public release

1. Source regressions, official CI and **private exact v0.49.0 candidate**.
2. Separate live test-owned installed v0.48/0.49 upgrade, MCP/CLI/Web and
   high-risk lifecycle/concurrency/UNKNOWN native independent oracles.
3. Genuine Finder original attempt plus a different clean Box/actor Verifier,
   complete required case records and cryptographic signature from a
   protected trusted key.
4. Protected environment configured, workflow-dispatch candidate gate
   actual denial AND success, public asset-byte parity and official install
   smoke. Do not create a public tag or move the stable selector early.

Remaining RACC-SYS-01..09 items are PARTIAL/NOT IMPLEMENTED, especially real
auto multi-Agent dispatch, full 19 scenario runtime and 100% safe eligible
MCP/CLI black-box coverage. The normal owner trigger remains
`Run full BoxWeave reacceptance`.

## Newly reproduced acceptance-tool defect (real installed baseline)

Running the developer `installed-readonly` probe against the genuine
v0.47.0 `boxweave --help` recursively expanded help choices from a fallback
root-usage page for unknown CLI subcommands. The initial real attempt exited
2 with `CLI_SURFACE_DISCOVERY_BUDGET_EXCEEDED` before MCP breadth could run.
Source mitigation parses only the actual command's usage header, rejects
fallback help masquerading as a subcommand, and retains a named BLOCKED
record if a bounded discovery budget is exhausted. A unit regression uses
root-help fallback to prevent future exponential discovery; an exact installed
rerun is still separately required before this bug is considered verified.

## Installed MCP privilege boundary discovered in test Box

The first installed MCP stdio probe used a `sudo -u boxweave` launcher and
failed with permission denied on the root-owned 0600 official release Gateway
file. An isolated, read-only manual protocol probe under the installed
root-owned gateway launched with `sudo -n python3` returned a genuine
68-tool `tools/list` catalogue after initialize. The source-only developer
argv validator now allows this exceptional exact root-owned immutable release
path only, rejecting user-controlled Python files or symlinks; it does not
introduce a new customer endpoint or generic root execution. Installed
schema-safe per-tool case results still require a separate source rerun.

The initial root-path check also uncovered an EACCES false negative:
the verified installed release parent directories are root-owned 0700, so the
ordinary user could not traverse them. The validation now uses only fixed
read-only `sudo -n /usr/bin/stat` on the strict release-path components,
with non-following file type, root UID and non-writable mode assertions.
The original failed probe was preserved as a separate run.

## 2026-11-05 further real baseline and private candidate observations

On the test-owned \`bwracc-mvp-verify\` outer Box, the authorized,
non-production v0.47.0 \`sudo boxweave selftest\` created an actual nested
Incus workload \`boxweave-selftest-8376ef-container\`, executed the guest
\`hello world\`, stopped and deleted the child, and returned \`state=passed\`.
Subsequent native Incus list was empty. This is actual **v0.47.0**
L2 installed behavior, not v0.49.0 verification.

The GitHub Actions **PR CI** artifact \`application-candidate\` (run
\`42153644892\`, artifact ID \`13474380251\`) was downloaded to this isolated
Box and wrapper SHA-256 matched
\`c569737a4ba6cf06db361bb580590030e8ab73f0c7b431bec3985e301f31e047\`.
The internal v0.49.0 \`boxweave.zip\` SHA-256 was
\`61571f8b05e71eddb0c27af477d7616237d14432a669c3171bbf5b338f28d99c\`,
matching \`release.json\` and containing no test runner. Its **PR synthetic
merge source** was \`66e0a2173019d305beecbf1e1fb9dd9580e723dd\`,
not the eventual main release source. Exact application-only upgrade from
v0.47.0 to this candidate returned \`ready\`, authenticated MCP read and
a durable prior SQLite backup. However **host \`boxweave version\` stayed
v0.47.0**, exposing that the app distribution installer alone does not
update root-owned Host CLI/Gateway. This is not full host acceptance.

A new scoped \`online.py --private-manifest-sha256\` developer-tested
entrypoint is being added to enable the **entire host+app source-bound
candidate installation before public tag creation**. It rejects a changed
manifest, changed asset, symlinks, extra paths, unsigned directory ownership
and group-writable files. Its full real host upgrade and separate trusted
Verifier gate are still NOT ACCEPTED until exercised on a newly built exact
candidate. Do not reuse this older PR synthetic candidate as the final
main candidate.

Live read-only installed probe \`racc-readonly-0470-r5\` reached the real
68-tool installed MCP stdio catalogue, 10 MCP read-only probe attempts and
81 installed CLI help/readonly evidence entries. The earlier summary counted
all 91 evidence entries as \`actual_probe_count\`; source code now reports
read-only attempts and help observations separately. This was PARTIAL:
mutating oracles were unexecuted, and non-root CLI \`doctor\` lacked required
privileged read scope (independent \`sudo doctor --mcp\` was 10/10 PASS).
The \`boxweave_task_get\` positive attempt without an original task ID
returned \`TASK_SELECTOR_INVALID\`; fixed to BLOCKED selector-required, not
a product defect. Both errors are retained as original evidence and
classification regressions.
