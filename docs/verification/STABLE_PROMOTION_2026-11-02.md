# Stable installer promotion — exact real Box gates (2026-11-02)

[Master roadmap](../design/FEATURE_ROADMAP_MASTER.md) ·
[Phase status](../process/PHASE_STATUS.md) ·
[Current risks](OPEN_RISKS.md) ·
[Agent real-Box workflow](../process/AGENT_TEST_DEPLOYMENT.md)

**Decision:** Promote the **already immutable and actually installed** source
release v0.42.1 as the separately chosen default stable installer by changing
the root `PUBLISHED_RELEASE` file and the reviewed `install.sh` default.
v0.42.2 is an installer-documentation/regression patch; it is not a claim that
the entire optional roadmap E2–E4 or R-04 globally enforced mode is complete.
No production host/client settings were changed by this test.

## Target, artifact and authority

- Release: [v0.42.1](https://github.com/vanishs/boxweave/releases/tag/v0.42.1),
  source commit `1f67394166bb802ee2ef9bc20664d6d7e694fbdc`,
  immutable release `boxweave.zip` sha256
  `cbf637fa08c22cbc74b23ffcd594f21046569fdfb291ca021c68bf444c1b387d`.
  Release metadata, helper payload and source tag are verified by the
  streamed installer before executing any downloaded installer.
- Outer authorized controller: existing BoxWeave/Containarium dev backend.
  Two *task-created*, fresh Ubuntu 24.04 managed Boxes on backend `dev`:
  `bwstable-1102` and `bwstable-upgrade-1102`, isolated in the existing
  test Tenant, each 2 CPU / 4GB / 30GB, explicit parent nesting flag true,
  stopped/started once **before** installation to apply the nesting flag.
  No external host root, public listener, credentials or historical Box was
  modified. Source development tests remain on reused `bwphase2-1102`.
- Installer `install.sh` from the selected immutable v0.42.1 tag was
  downloaded with HTTPS in the test Box, inspected, and executed as the
  test Box administrator using `--version 0.42.1`.
  The old-release baseline used tagged v0.28.0 `install.sh`.
  Test-owned installation changes within these disposable Boxes only.

## Fresh install live evidence

On `bwstable-1102`:

1. Explicit version-pinned streamed install: exit **0**, source-qualified
   runtime provenance `v0.42.1`, correct application user and root
   standalone Gateway helper publication.
2. `sudo boxweave status`: v0.42.1, Web loopback HTTP 401 without
   authentication. `sudo boxweave doctor --mcp`: overall **passed**,
   app/worker user units, credential/trust timers, and **authenticated**
   Provider MCP initialize/tools/read (`list_containers`), **58 tools**.
3. Loopback anonymous POST `/mcp` returned HTTP **401**, not unauthorized
   tool execution. HTTP OAuth and independent HTTPS clients are NOT tested.
4. Original selftest plan with deliberately wrong confirmation returned
   `SELFTEST_CONFIRMATION_MISMATCH`, exit **2**, zero child Boxes.
5. Correct confirmed selftest `boxweave-selftest-c24b68`:
   `create`/authenticated `hello world`/stop/delete all **succeeded**,
   CLI exit **0**, exact inner child list empty after deletion.
   This is a real nested Provider + guest execution, not a fixture.

## Actual old release upgrade evidence

On independently created `bwstable-upgrade-1102`:

1. Tagged v0.28.0 streamed fresh install exit **0**. Public
   installed provenance `2fc27a5b70734bcc9664fb05ce7c72a979c68615`,
   `boxweave version` **0.28.0**, `doctor --mcp` **passed**,
   authenticated Provider list read, no preexisting child Box.
2. Using the new tagged v0.42.1 installer `--version 0.42.1`, the
   **in-place upgrade exited 0**. Installed version changed to
   **v0.42.1** at exact source commit `1f673941...`.
3. Post-upgrade `doctor --mcp` **passed**, 58 Provider tools, local
   Web/worker still healthy and `admin_initialized=True`; original
   administrator initialization was **preserved**, not re-created.
   Unauthenticated HTTP MCP POST remained HTTP 401.
4. A *fresh new* confirmed selftest `boxweave-selftest-8b4e2c`
   succeeded create, `hello world` execution, stop and delete with
   exit **0**, no residual inner Boxes.
5. An outer stop/start was subsequently performed on this exact
   test Box to test cold-start persistence. Initial outer
   `connect(exec)` returned `TRUST_NOT_READY`; outer
   `get_container` still showed the same running incarnation.
   Outer `debug_container` flagged the known missing host-hop shell
   (distinct from normally working direct Box-IP SSH).
   After the outer trust-readiness window, a new direct
   `connect(exec)` succeeded without changing SSH or host-key
   configuration: installed `sudo boxweave doctor --mcp` returned
   **overall passed**, v0.42.1 and `authenticated_read_passed`
   for Provider `list_containers`; the nested Incus guest list was
   again empty. This is a *confirmed recovery* of direct Box access,
   not proof that the host-hop shell or an external SSH client works.

## Promotion limitations and negative decision scope

- Default stable release only promises the **tested installed baseline**
  on a compatible Ubuntu/nested topology. The optional protected
  Gateway/Web shared-claim R-04 policy was **not** provisioned or
  enabled; external independent SSH/HTTP Client contention is still U.
- E0/E1 are supported read-only source behavior with bounded Box inputs;
  E2 snapshots, E3 restore, E4 clone/rekey are BLOCKED by absent verified
  Provider methods and storage-volume contracts.
- Real externally routed HTTP OAuth/TLS, native Windows/macOS client setup,
  outside-Box Provider mutation bypass, explicit multi-Agent approval,
  actual production-host migration and old-binary rollback remain U.
- Old v0.28.0 binaries must not be pointed at migrated v13 application DB;
  an isolated older-source compatibility test previously demonstrated
  `SCHEMA_UNSUPPORTED` refusal rather than reversible downgrade.
  Back up production app data before any real update.
- Root HTTP/web remains loopback, no public TLS or credentials were printed,
  and no automatic user-host update/promotion is performed.
- `v0.42.2` must pass exact-SHA Python, artifact, browser and package
  GitHub CI before its tag can be published. A new stable default must not
  infer unverified experimental features.

## Source acceptance and cleanup

Local candidate worktree and final GitHub CI must record exact exit
codes. Two newly created test Boxes are **task-owned**; after all
necessary post-restart evidence is collected, delete only these two
Boxes with separate outer readback and retain historical Boxes and
source worktrees. Any uncertain failure should preserve its scene,
not be silently retried or deleted.
