# Containarium v0.126.0 upgrade and E2/E3/E4 integration — future Agent handoff

**Status:** DESIGN / NOT EXECUTED. **Target:** upstream Containarium `v0.126.0` annotated Git tag `3b37f1ac544c4b0c1094113f024595043b599cf0` -> source commit `a0bb0c682be52d7b20d7c38bf3fbae9c9b1bdd3a`. The inspected pre-fix BoxWeave source baseline was `v0.42.7`; this handoff accompanies the v0.42.8 D4 corrective candidate, with stable default installer pin `v0.42.1`. Neither target version is a claim that the operator host has been upgraded.

[Roadmap master](FEATURE_ROADMAP_MASTER.md) · [E0 provider capability](BOX_ENV_E0_CAPABILITIES.md) · [E1 fingerprint](BOX_ENV_E1_FINGERPRINTS.md) · [E2 snapshots](BOX_ENV_E2_SNAPSHOTS.md) · [E3 restore](BOX_ENV_E3_RESTORE.md) · [E4 clone](BOX_ENV_E4_CLONE.md) · [Phase 3 acceptance](PHASE3_SUPPORTED_ENV_ACCEPTANCE.md) · [Verification queue](../verification/REMEDIATION_TEST_QUEUE.md) · [Residual reacceptance](../verification/REACCEPTANCE_FULL_RETEST_0427_2026-11-02.md)

## What was verified in upstream source (S), not installed

- Public v0.126.0 release includes `restore_box_backup` MCP and `containarium backup restore --box`; v0.125.0 introduced streamed whole-box restore and scratch verification. These are **not** same-Box ZFS snapshot rollback. Upstream `internal/mcp/backup_restore_box_tool.go` requires `box`, `backup`, and **MCP-host-local `age_identity_file`**; targets a **new stopped** Box by default. A destination that already exists requires `force` and must be stopped. Never put an age private key in MCP arguments or output.
- The upstream `internal/mcp/backup_tools.go:create_backup` schema **still documents database/hook backup**, not an obvious `scope=box` field. Whole-box creation is documented as CLI `containarium backup create <box> --box`. A later Agent must verify an authenticated typed whole-box-create API / CLI bridge instead of assuming this MCP tool can produce a whole-box backup.
- `ContainerService` already provides typed authenticated `CreateContainerSnapshot`, `ListContainerSnapshots`, `DeleteContainerSnapshot`, `RollbackContainerSnapshot`, exposed through REST and `containarium snapshot` CLI. The architecture document marks this shipped but **`CloneContainerFromSnapshot` still proposed**. These are ZFS dataset snapshots: check actual host storage pool/dataset and encryption/key-custody constraints; a directory-backed Incus pool is not automatically eligible.
- `internal/autosleep/manager.go` includes an opt-in idle-stop ticker; `internal/autosleep/decide.go` defers an idle stop for **2x the configured threshold after last start** and can lack a reliable traffic signal. `internal/wake/proxy.go` and `internal/server/dual_server.go` wire HTTP wake **only when route hosting, a reachable host address, and gateway/HTTP integration are configured**. A `toggle_auto_sleep` acknowledgement alone cannot prove actual stop/wake.
- Previous installed upstream daemon **0.120.1**, 58 authenticated Provider MCP tools, had no snapshot/clone/box-restore methods in that MCP catalog; absence from that catalog alone did **not** establish absence from the daemon's authenticated REST/gRPC or CLI. The new version's source features do **not** grant them in BoxWeave automatically.
- Sources: [upstream release](https://github.com/FootprintAI/Containarium/releases/tag/v0.126.0), [snapshot ownership/contracts](https://github.com/FootprintAI/Containarium/blob/v0.126.0/docs/architecture/container-snapshots.md), [snapshot CLI](https://github.com/FootprintAI/Containarium/blob/v0.126.0/internal/cmd/snapshot.go), [box restore MCP](https://github.com/FootprintAI/Containarium/blob/v0.126.0/internal/mcp/backup_restore_box_tool.go), [auto-sleep manager](https://github.com/FootprintAI/Containarium/blob/v0.126.0/internal/autosleep/manager.go).

## Authorization and hard stop conditions

The **current request authorizes writing this plan**, not upgrading the owner's host or publishing an E2/E3/E4 integration. The future Agent must obtain specific authorization for the exact test host before root/Provider upgrades, ZFS snapshot manipulation, backup key use, or any destructive rollback. Start with a newly created disposable test Box running nested Provider; do not use an old historical Box for snapshot/rollback/clone. If it has no ZFS-capable storage or no authenticated Provider contract, mark **BLOCKED** with evidence; do not invoke raw `incus` or `zfs` as an unreviewed substitute for a tenant-authorized API. Read-only Incus/Provider inspection is allowed only inside the authorized lab.

Never overwrite a running Box, forcibly replace an existing backup destination, turn off auth, export keys, silently enable a public HTTP endpoint, change tenant isolation, modify the real root-owned policy, or retry an uncertain non-idempotent operation after timeout. Record original operation/identity/backup ID and reconcile the original outcome first.

## Implementation and acceptance sequence (each batch has its own commit and evidence)

### U0. Preflight, exact upstream pin, upgrade/rollback proof

1. Read `AGENTS.md`, current master/scope and installed provider diagnostics; obtain fresh exact host/target authorization. Record BoxWeave source SHA, installed app and worker versions, original Provider/Incus versions, `ZFS`/storage pool/dataset, native network Tenant, service units, auth scopes, signed package manifest and original Box registry. `v0.126.0` must resolve to the annotated tag's source SHA above, not a mutable `main` ref.
2. Create a new Ubuntu 24.04 nested-capable **disposable** outer Box; use a version-pinned, reviewed upstream package path and hashes. Prove clean bootstrap and `doctor --mcp` with original Provider; then upgrade this test-only Provider to v0.126.0. Treat installer returning early, `INSTALL_ALREADY_RUNNING`, unknown post-dispatch or a missing credential as **UNKNOWN / interrupted**, not permission to rerun. Preserve any original process/phase and inspect before resuming.
3. Verify actual *running* daemon and MCP server versions separately; authenticate, discover tools with full schemas, and make one `get_container`/list read. Restart service once, confirm state/target/keys and DB/service health, and independently compare baseline. Ensure rollback artifact and state backup are created **before** upgrade, but do not claim direct downgrade of a newer database/schema works. Test failed-upgrade recovery or documented side-by-side restore only in this lab.
4. **Gate U0 PASS:** clean installed v0.126.0, authenticated Provider identity and read, same trusted target registry, correct user/credential owner, stable app worker, service restart and evidence receipts. Otherwise U0 BLOCKED/ISSUE with no E2/E3/E4 mutation.

### U1. E0 capability matrix + safe adapter decision

1. Compare authenticated MCP schema against typed Provider REST/gRPC/CLI operations. Classify each target operation: `MCP available`, `typed REST/gRPC only`, `CLI only`, `absent`, or `storage unsupported`. Do not equate `create_backup` database-only MCP with whole-Box backup.
2. If a required method is REST/gRPC/CLI-only, **design a minimum authenticated Provider adapter** in BoxWeave or request an upstream MCP tool. Reuse existing root-owned credentials and authorization, not a direct Incus escape hatch; use exact user+createdAt+instance identity and typed bounded responses.
3. Commit an E0 capability test matrix with positive/negative and unsupported cases, no state mutation. Gate new Web/Agent controls by **actual deployed method and storage eligibility**, not merely daemon version.

### U2. E2 snapshots: safe, stopped Box only

1. Provision a task-owned stopped Box on verified supported ZFS-backed dataset. Capture E1 before state and exact incarnation; run authenticated `ListContainerSnapshots` first.
2. Implement list/create/delete with strict input validation, unique caller-specified snapshot names, retained used/referenced bytes and snapshot ownership, quota/space bounds, D3/D4 bounded diagnostics, and R-04 shared protected claims. No raw credentials, path traversal or shell interpolation.
3. Create one snapshot of a stopped Box, enumerate it by exact name, reject a duplicate and unauthorized/changed target, then delete only that test-owned snapshot. Verify no stale file/record. Record unsupported response on non-ZFS storage as a **capability gate**, not a success.
4. Gate E2 only after installed new-Box positives, denied cross-tenant, wrong name, mid-request disconnect/UNKNOWN reconciliation and actual lifecycle cleanup.

### U3. E3: distinguish destructive same-Box rollback vs whole-box restore

1. **Same-Box snapshot rollback** uses typed `RollbackContainerSnapshot` and is destructive to writes since snapshot; require an exact stopped Box, existing snapshot, explicit plan showing overwritten file/fingerprint and newer-snapshot destruction, and one-shot confirmation. Test planned refusal against running/stale/changed Box; never silently `force` a real target.
2. On a task-owned stopped test Box only: create a marker before snapshot, change it, prove pre-rollback state, invoke exact rollback with explicit confirmation, verify recovered marker and preserved identity, report covered/uncovered volumes, verify R-04 claims and no auto-retry on uncertain transport.
3. **Whole-Box backup restore** is a *different workflow*. On v0.126.0 verify `backup create --box` encrypted artifact, ID, manifest/digest, off-host/host storage, available age identity custody (never expose secret), `verify_backup` and `restore_box_backup` into a **fresh stopped target**. On missing identity, wrong backup, existing running target or wrong tenant, fail closed. Never label this exact-current-Box rollback.
4. Gate E3 only with live installed Provider + real data comparison and failure recovery, not a source fixture or success exit alone.

### U4. E4 clone-like restore and identity separation

1. Do **not** claim `CloneContainerFromSnapshot`: upstream marks it proposed. Investigate supported `RestoreBoxBackup` to a distinct stopped target as a potential *clone-like* flow, with an explicit compatibility status `restored_copy`, not a snapshot/instant clone.
2. With a task-only source and target: verify new ID/createdAt, independent storage, network Tenant, SSH host keys, authorized user, credentials/secrets isolation, no duplicated runtime identity or active services, DNS/routing conflicts, and original target remains unchanged. Ensure imported/restored Box is quarantined and stopped until reviewed rekey/provision.
3. If upstream cannot provide trusted rekey/isolation or deletion safety, mark E4 **BLOCKED** (do not advertise clone), but keep E2/E3 independently eligible. Test same-name, cross-tenant, stopped-only, insufficient-space, wrong-key and uncertain-receipt cases.

### U5. P-01 idle-stop and HTTP wake

1. On a disposable Box, enable `toggle_auto_sleep` with smallest supported threshold; observe actual runtime and `LastStartedAt`; stop touching the guest and wait **more than 2x threshold + one ticker interval**, explicitly verifying container State=Stopped with independent authenticated Provider/Incus observations. Preserve any idle network signal/SSH session caveats. Disable the toggle and recover after the test.
2. HTTP wake requires a test-owned *real route*, reachable proxy/Caddy+gateway and non-production hostname. If route support is unavailable, mark wake **BLOCKED by routing topology**, not by lack of code. If available, test a request causing exactly one wake, preserved authorization and route, a request while wake is in progress, and a denied untrusted source. Never create a public route without separate scoped permission.
3. Test idle-stop behavior around a real running task and no false sleep during active traffic; capture exit/code, timestamps and after-state.

### U6. Release and evidence

- Gate one independently testable feature at a time; version bump/commit/publish only after source + isolated **installed** acceptance on its exact target. Fresh app install, older app upgrade and rollback-repair are separate tests.
- Update [feature matrix](../verification/FEATURE_MATRIX.md), [remediation queue](../verification/REMEDIATION_TEST_QUEUE.md), [risks](../verification/OPEN_RISKS.md) and this master. Record version+commit, actual tool schema, exact Box/Provider, commands, exit codes, expected/observed behavior, negative tests, remaining BLOCKED, test-only cleanup and any preserved UNKNOWN scenes.
- Do not modify `PUBLISHED_RELEASE` without a distinct stable promotion decision. Do not mark E2/E3/E4 `implemented` just because an upstream CLI/REST exists. No real host change is authorized by this document.

**Stop/skip escalation:** Ask the owner only when a decision changes product semantics (e.g. whether destructive same-Box rollback or new-copy restore is primary), required storage/credential trust, global HTTP exposure or protected default policy. Otherwise continue independent safe tests and accurately record missing prerequisites.

[Back to roadmap master](FEATURE_ROADMAP_MASTER.md)
