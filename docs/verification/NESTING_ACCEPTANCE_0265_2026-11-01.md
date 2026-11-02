# Post-host-v0.26.4 nesting and partial-create reacceptance — 2026-11-01

**Status:** Real nesting configuration and inner Linux/Python execution **PASS** on one new authorized Box. The v0.26.5 structured partial-create reporting is **source/CI verified pending runtime negative-path acceptance**. This adds evidence to [A01–A33 code-first reacceptance](REACCEPTANCE_0262_TO_0264_2026-11-01.md), [current risks](OPEN_RISKS.md) and [remediation queue](REMEDIATION_TEST_QUEUE.md). It does not certify every host or every MCP client.

## Real host v0.26.4 retest

- User reported installed outer host v0.26.4. `boxweave_info` returned **static source guide** `0.26.4` and is not alone proof of an installed app process version.
- **Outer exact-target nesting helper now works:** `boxweave_nesting(inspect,bwaudit-1101)` returned an exact current Incus UUID and config fingerprint; nesting was **false** on this earlier failure Box. `boxweave_nesting(inspect,boxweave-test-ser)` returned nesting **true**. The old failure is not solely a broken Incus helper; its parent nesting configuration differs. `plan` on old `bwaudit-1101` was read-only, with exact name/UUID/fingerprint. **Original failure Box and stopped child were not reset, and stay retained for comparison.**
- **New** `bwnestfix-1101-container` (native Tenant `bwaudit-1101`, IP at creation `10.100.0.8`, CPU2, memory4GB, disk24GB) was created by the installed v0.26.4 `create_container(enable_nesting=true,enable_podman=false)`. The create response contained a successful second-stage BoxWeave nesting receipt: `nesting=true`, `changed=true`, `effective_nested_execution=not_verified`, `restart_required_for_verified_transition=true`. Independent `boxweave_nesting(inspect)` confirmed the same instance UUID, created timestamp and `nesting=true`.
- Stopped and restarted **only the task-owned new Box**. A first direct SSH call immediately after restart returned `TRUST_NOT_READY`; observing the original Box (running, exact identity) and repeating a **read-only** hello after readiness returned exit 0. Trust issues should not prompt recreation of a running Box. Nesting remained true after restart.
- Installed v0.26.4 via the source-bound streamed installer in the new Ubuntu 24.04 Box: installer service exited successfully, `boxweave version` returned `0.26.4`, `doctor --mcp` **10/10 checks passed**, inner Incus was installed.
- **True nested execution PASS**: after installing and starting inner Incus, `incus --force-local launch images:ubuntu/24.04 bw-nest-pass` completed with `BWNEST_INNER_CREATE_EXIT=0`. Independent `incus list` showed `bw-nest-pass Running`. `incus exec bw-nest-pass -- python3 -c ...` printed **`BW_NESTED_PYTHON_OK 3.12.3`**, and the shell reported **`CHILD_EXIT=0`**. `boxweave box inspect bw-nest-pass` read the exact child as Running. This closes *one* former full nesting unverified gate on the new parent and correct nesting setting.
- The original `bwaudit-1101` had `security.nesting=false`; its old `proc`/cgroup BPF failed child is still retained. **Do not overwrite this distinct failure scene or claim old-Box automatic repair.**
- `debug_container` still found the outer host-hop login shell `/usr/local/bin/containarium-shell` absent, but direct guest-IP `connect(exec)` succeeded. This is an unaccepted separate access route, not evidence that nesting or all SSH is broken.

## Source remediation: truthful partial creation (v0.26.5)

**Prior defect A33:** when upstream `create_container` succeeded but the independent `nesting_plan` or `nesting_invoke` step failed, the BoxWeave MCP gateway replaced the successful upstream creation receipt with a JSON-RPC error. Some MCP clients only surfaced a generic error, obscuring the fact that the Box already existed. A subsequent create could collide with the original resource.

**Patch:** Preserve the upstream `tools/call` result and its exact content, prepend a bounded structured `boxweave_nesting` message, set the MCP result `isError=true`, and include:
- `state=partial` and `box_created=true`;
- `requested_nesting` and `nesting_outcome=not_verified` (not a false claim that the setting was certainly disabled or not applied);
- exact requested logical username, approved redacted static error code, and `next_action` to inspect the **original** Box without retrying the create.

**Test contract:** Additional deterministic gateway protocol tests verify no replay, original Provider receipt retention, `isError` marker, explicit unknown second-stage outcome and redaction of unknown helper details. Existing gateway success/negative contract tests still run.

**Security boundary:** No change to root nesting-helper privileges, sudoers, Incus confinement, host keys, default permission, MCP catalog or Provider credentials. No automatic retry/recreation/rollback of an uncertain create.

## Outstanding and disposition

- **Runtime negative-path:** v0.26.5 must be installed on a controlled gateway and its post-create helper failure exercised safely before the partial-outcome client behavior is marked fully accepted. A unit fixture proves the output structure but cannot prove any third-party UI displays it.
- **Host-hop login path:** missing wrapper in the parent still needs independently reviewed host maintenance if an actual sentinel/jump route is needed; direct guest ssh works and no broad upstream sudoers script was run.
- **Other product gates:** authenticated external HTTP OAuth/TLS client, Web operator session, D4 allowlisted positive, cross-platform desktop integration, private unpublished-source delivery and historical-specific repair remain U/BLOCKED until separately executed.
- **Cleanup:** new audit Box `bwnestfix-1101` is disposable; after the version upgrade checkpoint, delete it and its nested child via the authorized Box lifecycle, observe absence and update the test receipt. Original `bwaudit-1101` remains deliberately retained for its older failure case.

**Scope of PASS:** one `dev` backend, one Ubuntu 24.04 nested parent with correct config/restart, one Ubuntu inner child launched and executed Python, no claim of generic nested selftest or all container images.
