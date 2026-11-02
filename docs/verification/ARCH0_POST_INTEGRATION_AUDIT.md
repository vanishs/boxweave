# ARCH-0 follow-up: post-integration dependency and delivery audit (v0.42.1)

[ARCH-0 gate](../design/ARCHITECTURE_CONSOLIDATION_GATE.md) ·
[Master roadmap](../design/FEATURE_ROADMAP_MASTER.md) ·
[Runtime boundary inventory](ARCH0_BOUNDARY_INVENTORY.md)

**Baseline:** exact main `22641c83dd063cfb8ad2eac828a0a90d16a90e4f`
(source v0.42.0). This is an owner-requested follow-up review, not a
retroactive claim that all historical A1–A8 were rewritten.

## Audit findings

A reproducible AST import graph enumerated **50 package Python modules**
and **112 source import edges** at baseline. It found the single static
back edge `deploy_check -> __main__ -> deploy_check`, because the
deploy-check layer imported CLI `load/read_config` while the CLI
`diagnose` action independently imported `deploy_check.read_provider`.
This makes an independent health probe depend on a full command dispatcher.
The original ARCH-0A/B/C extraction and CI were real but did not cover this
later import dependency. File size alone does not require a rewrite.

The standalone Gateway helper set still exists in **three intentionally
independent deployment contexts**: `online.RELEASE_FILES`,
`scripts/build_release_manifest.py` copy/manifest, and
`online.py` root installed helper copies. Their duplication preserves
standalone/root trust assumptions but is prone to missing a newly imported
gateway sibling. Past releases added separate `box_patch.py`, confirming
this is a practical regression class, not a mandate to unify privileged
installer and application imports.

## Source remediation

- New `bwmini/app_runtime.py` owns `read_config` and `load`, the
  application composition factory for Provider, Store, Service and optional
  shared claims. It does not depend on CLI argument parsing.
- The previous `bwmini.__main__.read_config` name remains an identical
  alias. `bwmini.__main__.load` stays a compatibility wrapper and passes
  the existing CLI read-config seam so monkeypatched integration callers
  behave identically. The independent `deploy_check` imports lower-level
  `app_runtime` directly; it no longer loads the CLI module.
- Added `tests/test_arch0_boundary_guards.py`: ensure module-level
  package import graph stays acyclic, the independent deploy check has
  no CLI import, the three reviewed helper inventories agree, every
  root-installed helper exists in release, and the standalone Gateway
  never imports private Web/app state modules.
- No changes to user config schema, private DB layout, root-owned
  installer, SSH forced-command path, HTTP MCP relay, gateway transport,
  target identity, resource-claim semantics, default stable release or
  original operation receipts.

## Actually executed checks in existing authorized development Box

Box `bwphase2-1102`, Ubuntu Python 3.12, source `main 22641c8`
plus isolated `arch0-followup` modifications:

- `python -m unittest discover -s tests -p test_box_actions_shared_claims.py -q`:
  **10 PASS**, exit 0. Verifies existing `__main__.read_config` injection.
- `python -m unittest discover -s tests -p test_arch0_contracts.py -q`:
  **7 PASS**, exit 0.
- `python -m unittest discover -s tests -p test_arch0_boundary_guards.py -v`:
  **4 PASS**, exit 0. Cycle and three-way helper parity proved at source.
- `python -m unittest discover -s tests -q`:
  **639 PASS**, exit 0.
- `python scripts/check_knowledge.py`:
  exit 0, 161 documents (before this new report).
- Separate fresh Python process `import bwmini.deploy_check`:
  `bwmini.__main__ not in sys.modules`, assertion PASS, exit 0.
- `python -m bwmini.deploy_check --help` and
  `python -m bwmini --help`: both exit 0.
- `python scripts/validate.py`: **exit 2** because Node/npm is absent
  from this preexisting development Box; do not report local frontend
  validation as PASS. Exact-SHA GitHub CI is required for browser,
  packaging, Python multi-version and immutable source publication.

## Boundaries still open

1. The six logical ownership blocks are **not six physical directories**.
   Large `mcp_gate.py`, `host_cli.py` and `online.py` still exist by
   design because they are sensitive standalone/privileged entrypoints.
   Any larger split must include rooted installed-package parity.
2. Cross-UID Gateway/Web shared DB activation, protected host policy and
   external SSH/HTTP/Web race remain **L=U**, not silently solved by imports.
3. This development Box contains source tests but no configured inner
   production-like Provider or Node. Full installed previous-release
   upgrade, failing selfcheck and rollback comparison remain **BLOCKED**
   here and need a separately provisioned, authorized installed Box.
4. Historical ARCH-0A/B/C source-release acceptance remains PASS.
   This follow-up closes one real dependency edge and guards helper
   publication drift. It does not declare every roadmap feature done.

**Result:** SOURCE/T passed for the narrowly scoped follow-up.
CI/release must be evidenced at exact final SHA. Installed service-level
architecture acceptance remains U until performed on a suitable host.
