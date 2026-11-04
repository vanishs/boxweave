# v0.43.1 release and separately promoted stable installer (2026-11-03)

**Decision:** The new v0.43.1 code repair has independently passed source CI, immutable release publication and a real installation upgrade inside one task-owned isolated Box. Promote only the **default stable selector** in `PUBLISHED_RELEASE` and `install.sh` after this gate; leave all published tag assets immutable. No operator host/provider upgrade is performed.

## Immutable source identity

- [Source repair PR #63](https://github.com/vanishs/boxweave/pull/63) merged to main commit `eadcb27db5fec7e77bcb905c0452e58c9c87a3a0`.
- [Main CI 41776913234](https://github.com/vanishs/boxweave/actions/runs/41776913234): **verify SUCCESS, publish SUCCESS**. Earlier PR 41776616591 verification SUCCESS.
- [Immutable v0.43.1 Release](https://github.com/vanishs/boxweave/releases/tag/v0.43.1): tag object is a direct commit ref pointing to exactly `eadcb27db5fec7e77bcb905c0452e58c9c87a3a0`, not an unsigned movable pointer in the acceptance evidence. Release contains 23 digested assets. Published app zip SHA-256 `dfc62ac0801bc2adae9a7c8a7bcc8ff7ba0f83366b7c3d76c824e8dda196471f`. Gateway helper digest `e495207a8379446e375aa87637da9d4da081f6491b6a638d3878c58710821603`.
- Source QA in `bwaudit-1101` using a clean main checkout plus reviewed repair: `scripts/validate.py --browser`, exit **0**, **10/10** steps (192-document knowledge checker, **703/703 Python unit tests**, Web TypeScript/Vite build, seven Chromium fixture flows) PASS. Source fixture proof is not a live Provider write claim.

## Actual isolated upgrade acceptance

- Exact test-owned outer Box `bwaudit-0430-1103-container`, `createdAt=1793766262`, Ubuntu 24.04, original outer host Containarium 0.120.1. This Box had been stopped and preserved after an earlier create-response ambiguity. Its exact original identity was read before it was started and reused for upgrade-specific comparison; other old/prod Boxes remained untouched.
- Baseline after restarting the same Box: `BoxWeave 0.43.0`, `sudo boxweave doctor --mcp` exit **0**, **10/10** required PASS. The first immediate post-restart SSH read failed `TRUST_NOT_READY`; host-hop `debug_container` showed valid shell and no sshd rejections; the original trusted path recovered without host modification. No TOFU or key rewrite.
- Downloaded tagged `v0.43.1/install.sh` and verified exact SHA-256 `af0dfde9332e35c37e582a70fa4b63fd9d291f5e94c9c2b6abc05a0b4b2d36eb` against the immutable GitHub release asset. Ran `sudo bash /tmp/boxweave-install-0431.sh --version 0.43.1`; recorded **exit 0**, original environment upgraded in place.
- Post-upgrade `boxweave version`: **0.43.1**. `sudo boxweave doctor --mcp`: **10/10** required PASS, authenticated REST inventory and authenticated upstream MCP read PASS, Web and Worker installed and available. Loopback unauthenticated HTTP MCP returned **401**, not public-client acceptance.
- Actual installed standalone Gateway under non-root `boxweave` identity: imported `mcp_gate.SERVER_INFO_VERSION=0.43.1`, helper SHA-256 exactly matched release; an error-shape fixture via installed module retained original JSON-RPC -32000 and returned `CREATE_OUTCOME_UNVERIFIED` / `unknown` / `not_authorized` for retry. A separate real gateway subprocess completed MCP initialize, discovered **68 tools**, and returned `passed` for both `boxweave_diagnostic_run(preflight)` and `rest_read`, independently proving installed authenticated REST/MCP reads.
- Real upstream create-response 500 is **not proven fixed** and was not re-triggered on this upgraded host; the regression validates BoxWeave's conservative classification and its installed helper, not the upstream root cause.

## Stable-selector change

The release/promotion follow-up modifies only `PUBLISHED_RELEASE` from `0.43.0` to `0.43.1`, the embedded default `install.sh` release-selector string, and documentation. It does **not** modify the tag `v0.43.1`, its assets, executable installer code or the upstream Containarium version. The exact-release option `--version 0.43.1` was tested above; default no-version installer selection must be rechecked after the promotion merge before claiming that last gate passed.

## Outstanding explicitly unverified surfaces

- The user's **original** host upgrade and all external client re-enrollment are intentionally deferred until the owner updates it.
- Original Containarium MCP create-after-commit HTTP 500 root cause remains open; no automatic retry, adoption, removal or unsafe fallback is authorized.
- Positive R-02/R-04 provisioning, E2 snapshots / E3 restore / E4 clone and ZFS/storage-contract gaps, public TLS/OAuth/desktop integration, true nested child workload, model inference and live mid-dispatch crash injection were not part of this release safety patch.
- The test-owned Box's exact final state should be inspected separately after default-promotion smoke, and recorded; do not infer cleanup from `install.sh` or CI.

Related: [post-dispatch regression details](CREATE_OUTCOME_0431_2026-11-03.md), [v0.43.0 prior acceptance](REACCEPTANCE_0430_2026-11-03.md), [risk register](OPEN_RISKS.md).
