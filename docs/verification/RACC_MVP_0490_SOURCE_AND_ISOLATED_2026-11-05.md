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
