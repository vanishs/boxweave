# Architecture Consolidation Gate — source audit and insertion before Phase 2

**Status:** DESIGN / source audit only; owner requested an intermediate
architecture-review stage. No refactor, install, service, Provider or Box write
has been performed. **Source baseline:** `main` `0a3c0b524cfb6ec33a3d03d81d6f29f1549cecac`,
v0.29.0. No claim of installed v0.29.0 validation.

**Global plan:** [Feature roadmap](FEATURE_ROADMAP_MASTER.md), especially
INT-01..INT-06; [Phase 1 work ledger](../process/PHASE1_UNIFIED_OPERATION_BASE.md);
[Architecture authority](ARCHITECTURE.md); [Multi-Agent design](MULTI_AGENT_BOX_LEASE_CONFLICT_V1.md);
[MCP task recovery](MCP_TASK_RELIABILITY_R1.md);
[Agent development/testing](../process/AGENT_TEST_WORKFLOW.md).

## Decision and placement

**Insert ARCH-0 (architecture consolidation gate) AFTER P1-A/P1-B1
source foundation and BEFORE P1-B2 broad Web/SSH/HTTP mutator integration,
Phase 2 tracked task paths and optional multi-Agent leases.**

A deliberate modularity pass is justified; a repository-wide rewrite,
mass file move, new web framework, new state database or replacement
process manager is not. The stage's first deliverable is an explicit
dependency/entrypoint/authorization/packaging contract. Implementation
extractions require separate small, reviewed slices and acceptance.

## 1. Reproducible source inventory

At the pinned revision the repository tree has 323 tracked blobs; `bwmini/`
contains 40 flat Python files, ~523 KB of source, including `__init__.py`.
There are 48 tracked files directly in `tests/`, and 9 files in
`web/src/` (eight TSX files and one CSS). These counts are **static
GitHub tree observations**, not test execution, cyclomatic complexity,
runtime hotspots or reliability benchmarks.

Larger examples, by source size: `online.py` 47.8 KB (758 lines),
`host_cli.py` 45.5 KB (874), `client_profiles.py` 38.5 KB (720),
`mcp_gate.py` 33.2 KB (561), `lifecycle.py` 25.8 KB (411),
`access.py` 24.9 KB (307), `distribution.py` 23.7 KB (314).
The first four alone account for approximately one-third of the
`bwmini` bytes. File length is an investigation priority, not an
automatic defect.

## 2. Observed architecture and concrete coupling

| Finding / priority | Source evidence | Risk and appropriate direction |
| --- | --- | --- |
| A1 **P0: gateway delivery boundary differs from package** | `online.py:19` release-helper allowlist, `online.py:608-617` copies root-owned flat helpers into `/usr/local/lib/boxweave/default`; forced-command `mcp_gate.py` imports sibling `nesting`/`gate_diagnostics` with fallback. `remote_mcp.py:38-77` executes the same helper with `-E -s`, not `-I`, because relative sibling import matters. | Newly added `operation_coordination.py` and `coordination_runtime.py` live only inside the package in v0.29.0. Do **not** make gateway call a package-only module without an independently reviewed and tested deployment/import strategy. Keep existing SSH command, `python3 -I` root helpers, and HTTP relay contract stable. |
| A2 **P0: independent execution roots** | `mcp_gate.py:366-555` forwards upstream tools after local checks; `remote_mcp.py` delegates to that gateway per request; `service.py:210-288` dispatches app jobs; `box_actions.py`, `lifecycle.py`, `access.py` each persist plans/receipts. | There is no single controller-wide mutation interception today. Build the reviewed mutation classification/dispatch boundary once and adapt each existing route. Do not rewrite all receipts into one database or claim generic legacy `connect(exec)` deduplication. |
| A3 **P0: canonical identity scope to verify** | `domain.py:14-34` Target includes provider, backend and project; `coordination_runtime.py:20-47` hashes only logical, instance, created and user. `operation_coordination.py:57-75` has distinct box versus scope-based name/Tenant/port keys. | Across multiple backends/projects, the core Box key does not directly encode the full controller scope. Decide canonical backend/instance semantics and prove no collision/false conflict, including old Provider/createdAt representations and recreation. No external identity acceptance claimed. |
| A4 **P0: private DB access contract** | `coordination_runtime.py:50-81` demands DB + immediate parent owned by effective UID with mode 0600/0700, opens `mode=rw` and does not provision. `store.py:59-121` owns a different app SQLite schema. | Verify actual Unix identities of SSH forced-command, HTTP relay, worker and Web; separately classify logical trust principals even if they share a Unix account. Test all ancestor path safety, permissions, schema downgrade, DB corruption, WAL sidecars and conflicting open handles. No gateway access to app-private receipts/keys as a substitute. |
| A5 **P1: import responsibility inversion** | `provider.py:44-57` imports Access validation and Lifecycle while `access_provider.py` imports `provider.text`; `access.py` imports `access_provider`. `Store.__init__` dynamically imports feature schemas from Lifecycle, Access, Models, Jobs, BoxActions and RemoteAuth. | These deferred edges reduce obvious import failures but make dependency direction hard to audit. Extract only pure DTO/validation/protocol contracts into import-stable modules, keep one authoritative migration coordinator, and preserve existing `Store` schema/user_version. |
| A6 **P1: route and host orchestration mixed with helpers** | `web.py` creates auth/CSRF middleware plus project, Box, network, OAuth, read and receipt routes in one factory. `mcp_gate.py` contains catalog/trust/nesting/diagnostic/forwarding logic. `online.py` handles downloads/Provider preflight/install/SSH enrollment/CLI publishing. `host_cli.py` handles doctor/report/onboarding/job/selftest. | Split responsibilities behind existing public entrypoints only where a regression suite can prove equivalence. Root/install helpers are the **last** physical-move candidate, not the first. |
| A7 **P1: apparently duplicated validation may be intentional** | `distribution.py:85-116` carries standalone remote MCP config validation while `remote_mcp.py:348-372` validates at runtime. `bootstrap.py`, `online.py`, `distribution.py` and `hostkeys.py` each implement atomic-write/path checks with different privilege assumptions. | These are distinct trust contexts; do not blindly deduplicate. Maintain parity/property tests for the common config contract, with independent privileged file ownership checks. |
| A8 **P2: client generation vs control plane** | `client_profiles.py` contains large generated Python code templates; `client_setup.py` handles selector and publication. `web/src` has separate feature panels; `main.tsx` composes them. | Keep generated shebang, user-data preservation, script SHA/grammar, existing app catalog and backups stable. Extract templates later only with byte-level fixtures; no need to block Phase 2 on a UI rewrite. |

## 3. Six logical blocks (ownership boundaries, not an immediate mass move)

1. **Host / Distribution:** `online`, `bootstrap`, `distribution`,
   `host_cli`, `hostkeys`, `nesting`, `manage`, `http_manage`,
   `deploy_check`, external `install.sh`. Root and standalone-script
   identity must remain explicit.
2. **Gateway / Transports:** `mcp_gate`, `gate_diagnostics`,
   `remote_mcp`, `remote_tls`; use one reviewed mutation gate for
   SSH and authenticated HTTP. OAuth grant persistence belongs to the
   app state boundary even if its endpoint is in the remote transport.
3. **Application / Workflows:** `__main__`, `web`, `service`,
   `lifecycle`, `access`, `box_actions`, `selftest`, `jobs`,
   `graphs`, `models`. Keep original receipts and worker ownership.
4. **Provider / Box Adapter:** `mcp`, `provider`,
   `access_provider`, `agent_box`, `auto_sleep`,
   `capabilities`, `box_debug`. This is the only layer that
   should understand upstream tool schema and Provider variants.
5. **State / Core Contracts:** `domain`, `store`, `diagnostics`,
   `operation_coordination`, `coordination_runtime`,
   `remote_auth` storage. This is a **logical contract block**, not
   a plan for merging app SQLite, coordination SQLite, gateway result
   storage and diagnostic journals.
6. **Client / Presentation:** `client_profiles`, `client_setup`,
   `profiles`, `web/src`; no server-side execution authority
   derived from client metadata.

The existing flat package may retain compatibility façade modules
while pure new internals move into reviewed subpackages. Avoid a new
framework, universal service locator, broad `utils.py`, event bus or
generic provider plugin system.

## 4. Proposed ARCH-0 implementation batches

| Slice | Work | Acceptance before next slice |
| --- | --- | --- |
| **ARCH-0A: boundary map** | Enumerate each actual CLI/HTTP/SSH/Web entrypoint, import path, effective OS user, server credential, DB, public/privileged helper copy, mutating upstream tool and resource key; record installed vs source artifact identity. | Complete source trace with explicit gaps and protected/unprotected mutation map; no code behavior change. |
| **ARCH-0B: contracts first** | Resolve A1–A4: exact Box/backend scope, shared minimal DB ownership/provision/open and gateway import packaging. Add contract and hostile negative tests. | Cross-entry identity parity, same-name different project/backend, replacement, storage corruption/downgrade, root/app/SSH/HTTP import tests; unknown stays blocked. |
| **ARCH-0C: narrow extraction** | Remove A5 dependency inversion by relocating **pure** validation/protocol contracts; split gateway preflight/dispatch logic behind unchanged MCP and Web façades where packaging is proven. Avoid moving the root installer and client generator in this batch. | Full existing unit/browser/packaging CI, standalone script/installed entry checks, negative authorization, failure/UNKNOWN behavior and rollback equivalence. |

**Do not make physical moves mandatory.** If an extraction cannot be
shown safe, leave its file intact and record the interface contract.
Success is a clear, testable dependency direction, not fewer files.

## 5. Mandatory acceptance / rollback gates

- Source/transport parity: historical tool names, JSON-RPC IDs/results,
  exact trust check, root `python3 -I`, HTTP `-E -s`, CLI flags,
  login/CSRF/Origin and upstream schema variants remain stable.
- One original state/receipt, one authoritative dispatch intent before
  remote side effects, no transaction held over I/O; UNKNOWN is never
  converted to success/failure or retried automatically.
- Confirm SHA-256 provenance of offline release, separately copied
  root helpers and emitted client scripts; update allowlists and
  code-first inventory **only when actually changed**.
- Run focused unit tests first, then `python scripts/check_knowledge.py`
  and `python scripts/validate.py` on a prepared checkout. Browser
  fixtures when routes change. For any installed-path refactor, use
  the required separately approved real Box upgrade/checkpoint.
- Backward-compatible SQLite v12 migration, unchanged stable installer
  pin until release acceptance, and explicit old/new/broken helper
  combinations. A new source CI PASS is not a live acceptance.
- No publication/promotion/host or Provider modification is granted
  by this design note. Stage completion requires source evidence,
  scoped runtime evidence where applicable, and updated matrix/queue.

## 6. Exit condition for Phase 2

Only start P1-B2 / tracked MCP task and collision integration after the
A1–A4 contracts are resolved with tests and an explicit coverage map.
A5/A6 extraction may be partial when safe façades and dependency
direction are documented. A7/A8 are later cleanup, not blockers.
Record deferred work instead of turning architecture cleanup into an
unbounded rewrite.
