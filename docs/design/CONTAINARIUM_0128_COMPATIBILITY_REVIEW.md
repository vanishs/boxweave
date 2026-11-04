# Containarium v0.120.1 → v0.128.0 — complete upstream compatibility review

**Prepared:** 2026-11-03 (Pacific). **Scope:** BoxWeave P0 / first compatibility slice and repeatable provider upgrade support. **Current candidate:** BoxWeave 0.43.2. **Owner-selected policy:** **minimum versions, no maximum / per-version allowlist**. Do not confuse minimum compatibility with a guarantee that every future upstream API behavior is unchanged.

**Source evidence:** Containarium [release history](https://github.com/FootprintAI/Containarium/releases), [exact code comparison](https://github.com/FootprintAI/Containarium/compare/v0.120.1...v0.128.0), [v0.128.0 Swagger](https://github.com/FootprintAI/Containarium/blob/v0.128.0/api/swagger/containarium.swagger.json), [upstream upgrade / rollback runbook](https://github.com/FootprintAI/Containarium/blob/v0.128.0/docs/UPGRADE-AND-ROLLBACK.md). BoxWeave application source baseline `main` at `2a3895d872569867c1e91200fb0f263610452b1a`, stable application `0.43.1`, managed installer previously bootstrapping Containarium `0.120.1`. Actual `dev` Provider was observed running `0.120.1` with Incus `7.5.1`; no v0.128.0 host was installed in this review.

## 1. Size and quantitative contract delta

- **164 Git commits** from v0.120.1 through v0.128.0, including 12 published release tags after the baseline. Segmenting the GitHub compare API to avoid its 300-file cap yielded **671 unique changed file paths** across the entire interval. These are code paths, not 671 independently implemented product features. Some paths changed repeatedly.
- Generated Swagger/OpenAPI 2.0: **225 → 229 REST paths**, **276 → 280 HTTP operations**, **681 → 695 definitions**. No existing HTTP operation was removed in the generated definitions.
- Four new HTTP operations:
  - `POST /v1/backups/{id}/restore-box` (whole-Box backup restore);
  - `GET /v1/containers/{username}/run-user/status` (run-user sign-in state);
  - `POST /v1/containers/{username}/code-run-checkouts:sweep` (stale checkout cleanup);
  - `POST /v1/containers/{username}/skill-sets:apply` (skill-set re-apply).
- **14 new schemas**, **13 changed existing schema definitions**; representative additions include `BoxRun.model/usage`, `CrewRun.memberModels/usage`, `StartBoxRunBody.model/command/timeoutSeconds`, `RecipePort.role`, `SystemInfo.managedBackupKeyConfigured`. Schema property changes do not establish runtime semantic compatibility.

## 2. Changes by release group

| Range | Changes affecting BoxWeave | Acceptance implication |
| --- | --- | --- |
| v0.120.2–v0.121.0 | SSH key revocation/ownership, start/stop idempotence, per-run model controls | Negative SSH authorization, exact incarnation read-back and lifecycle tests |
| v0.122.0 | Better REST/gRPC error representation, run-user status, backup encryption parameters | Distinguish 404 absent target from missing route; keep UNKNOWN on ambiguous writes |
| v0.123.0–v0.123.3 | Managed BUILD_TOOLS toolchain, direct-command execution, run checkout cleanup, movement/restore robustness | Check `/opt/containarium/toolchains`; test move/execute independently |
| v0.124.0 | Origin validation of cookie-authenticated REST mutations, JWT revocation recovery | Retain bearer-token app backend and test denied/expired scopes |
| v0.125.0 | Streamed encrypted whole-Box restore, backup verification and stronger SSH host-key validation | Do not export age keys; preserve host-local identity semantics |
| v0.126.0 | `restore_box_backup` MCP, scope enforcement, signed upgrades, storage/capacity fixes | Preserve BoxWeave restore guard and exact backup/target approval |
| v0.127.0 | Recipe port roles, resource validation and log-injection defenses | Validate typed responses and bounded diagnostic logging |
| v0.128.0 | Skill-set apply API, model/token accounting, agent helpers, birth auto-sleep and HTTP wake fixes | Optional feature integration; priority regression for create/sleep/wake |

These are **upgrade-relevant grouped changes**, not a claim that every one of the 164 commits has a corresponding BoxWeave change.

## 3. Architecture and compatibility decision

Retain Agent-facing BoxWeave MCP, app-owned REST for basic Provider lifecycle and separate upstream MCP/SSH compatibility for advanced operations. This maintains the deliberately **coequal** REST and MCP adapters while the unified coordination core evolves. REST introduction does not automatically make direct upstream MCP writes participate in a common transaction: keep the known coordination coverage limitation visible and avoid blindly retrying dispatched create/delete/restore.

**Minimum-only rule (selected 2026-11-03):**

| Component | Minimum | Maximum | How evaluated |
| --- | --- | --- | --- |
| Containarium daemon (`containariumd`) | **0.128.0** | **none** | Trusted root-owned executable's bounded `version` output and authenticated runtime read |
| Containarium MCP server (`mcp-server`) | **0.128.0** | **none** | Authenticated MCP `initialize.serverInfo.version` and minimal capability/contract read |
| Containarium CLI (`containarium`) | Not independently pinned by this slice | none | Existence and existing host trust checks; feature-by-feature tests as CLI features are used |
| `agent-box` | No new installation requirement | none | Optional per-Box component, separate contract |
| Incus | Existing minimum 7.5.1 (not a Containarium version gate) | none | Current installer prerequisites |

A parse failure, untrusted executable, authenticated MCP absence or missing minimum required API capability is **UNVERIFIED/BLOCKED**, not a version downgrade or a reason to bypass policy. Newer releases remain eligible without changing BoxWeave source; passing the minimum is *not* proof of every new feature. Do not compare an MCP client build with daemon version as if they were identical implementations.

The SHA-256 of the **clean-host bootstrap release** remains pinned to approved official v0.128.0 binaries. It authenticates downloaded code and is **not** a runtime version allowlist. Existing upstream daemons are preserved (including already operator-upgraded daemon and CLI); BoxWeave may replace only its own reviewed, root-controlled MCP/Provider copies when its application is reinstalled. No automatic Containarium upgrade, unknown binary download, mutable `latest` root installer, provider service adoption, public listener or Incus mutation is introduced.

## 4. Concrete BoxWeave compatibility work

| Priority | Source paths | Required change / evidence |
| --- | --- | --- |
| P0 | `bwmini/online.py` | Minimum-only version acceptance; official verified bootstrap v0.128.0 asset digests; preserve existing higher-version trusted runtime; avoid exact hash equality as compatibility predicate; no host Provider upgrade on `--version` |
| P0 | `bwmini/bootstrap.py` | Refresh only BoxWeave-owned private MCP/Provider executable snapshots using newly verified source bytes; existing root-owned app state must still pass integrity checks |
| P0 | `bwmini/deploy_check.py`, `bwmini/mcp_compat.py` | Authentication/initialize tool discovery, require MCP >=0.128.0, verify real REST inventory and small dependent MCP schema |
| P0 | `bwmini/provider_rest.py` | Differentiate target-read 404 from endpoint route removal; keep error-body redaction, no write retry, timeout=UNKNOWN, and postcondition observation |
| P0 | `tests/test_containarium_shell.py` and provider version tests | Accept future versions, deny older/untrusted/unknown, retain pinned-wrapper source digest and deny unsafe replacements |
| P0 | Gateway, `operation_coordination.py`, restore guard | Regression: exact target identity; resource claims; bad scope; new tools not auto-authorized; same-host age key alias; no `force` restore by default |
| P0 | Installer and release metadata | New BoxWeave candidate version; stable default selector and `PUBLISHED_RELEASE` unchanged |
| P1 later | E2/E3/E4, idle-stop, wake, run usage, skill sets | Separate feature capability gates and real storage/route/SSH tests; not necessary to ship minimum compatibility |

**Critical existing limitation:** Upstream `create_backup` MCP currently denotes DB/hook backup; whole-Box backup creation has a distinct CLI path. ZFS snapshot functionality requires ZFS-backed storage, not the previously observed `dir` pool. The default-deny `restore_box_backup` handling cannot be expanded merely because a new REST endpoint exists.

## 5. Acceptance matrix

| Gate | Criteria | Candidate P0 state |
| --- | --- | --- |
| S — source contract | Exact upstream release and generated schema compared; no removed base CRUD operation | **Reviewed** |
| T1 — provider floor | `0.127.x` denied; `0.128.0`, `0.129.0`, future major admitted; unsafe/unknown denied | **CI required** |
| T2 — bootstrap | SHA-pinned minimum bootstrap, no overwrite on existing Provider; modified app-owned snapshot only | **CI required** |
| T3 — REST/MCP | API read, auth failure, route vs target 404, MCP version, tools-list and schema drift | **CI required** |
| T4 — coordination/diagnostics | Unknown response stays UNKNOWN with original claim; no secret leakage or replay | **CI required** |
| L1 — isolated upgrade | Disposable Box seeded with old version; real Containarium >=0.128.0; services, DB, SSH, MCP, REST | **Unverified** |
| L2 — original operator host | Real host upgraded by owner; exact-version BoxWeave update; doctor and read-back | **Deferred to operator** |
| P — publish | Signed/immutable release assets and matching GitHub source tag | **Pending CI and merge** |
| Stable pointer | `PUBLISHED_RELEASE=0.43.1`, embedded `install.sh` same | **Must remain unchanged** |

## 6. Explicit non-goals and residual risks

- No install/update of the operator's current `dev` Containarium v0.120.1. The new minimum means the operator must **upgrade that Provider first**, then install the candidate BoxWeave release.
- No E2 snapshot, E3 restore or E4 clone implementation, no new external OAuth/TLS claims, no automatic storage conversion, and no host-wide SSH policy rewrite.
- No assertion that an upstream `/health` check confirms credentials, schema, guest readiness or forward-compatibility. Future major versions may be admitted by version but still fail runtime contracts.
- If the upstream executable has unknown provenance, the operator must review it; do not implicitly replace it or waive host-root trust.
- The original `containarium-shell` script was unchanged between v0.120.1 and v0.128.0 at the upstream Git blob level, but this alone does not prove every future host-hop behavior or version compatibility.

## 7. Ownership and next steps

This review governs P0 compatibility only. Repeated upgrades follow the [generalized upstream Provider upgrade SOP](../process/UPSTREAM_PROVIDER_VERSION_UPGRADE_SOP.md). CI/source acceptance and operator-host acceptance must have independent receipts in [verification](../verification/CONTAINARIUM_0128_COMPAT_P0_0432_2026-11-03.md). Follow the [API-first master](API_FIRST_CONTROL_PLANE_EVOLUTION_MASTER.md), [diagnostics evolution](UNIFIED_OPERATION_TRACE_DIAGNOSTICS_EVOLUTION.md), [R-04 coordination](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md) and [old v0.126.0 E2-E4 investigation](CONTAINARIUM_0126_UPGRADE_E2_E4_HANDOFF.md) before changing their scope.
