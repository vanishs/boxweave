# Containarium upstream MCP follow-ups and host-hop SSH ownership

**Decision date:** 2026-11-03  
**Status:** OWNER-SELECTED DESIGN; documentation only. No runtime change, release promotion, host modification, or upstream commitment is authorized by this record.

[Master roadmap](FEATURE_ROADMAP_MASTER.md) · [Upstream ownership](../UPSTREAM_PROVIDER.md) · [Host-hop recovery](../process/CONTAINARIUM_SHELL_RECOVERY.md) · [Wrapper verification](../verification/CONTAINARIUM_SHELL_PACKAGE_0429.md) · [Isolated upstream v0.127.0 test](../verification/CONTAINARIUM_0127_AUTOSLEEP_WAKE_UPSTREAM_LIVE_2026-11-03.md)

## Origin and boundaries

The upstream maintainer's [Containarium issue #3500](https://github.com/FootprintAI/Containarium/issues/3500) investigated BoxWeave as an **external** MCP client. Its "Out of scope" section means that the issue does not undertake to change BoxWeave, maintain BoxWeave's vendored \`containarium-shell\`, certify version pairs, or promise compatibility. This is a scope/ownership statement, **not proof that host-hop SSH is inherently incorrect**.

Upstream [#3474](https://github.com/FootprintAI/Containarium/pull/3474), [#3490](https://github.com/FootprintAI/Containarium/pull/3490) and test-only [#3501](https://github.com/FootprintAI/Containarium/pull/3501) have merged. Containarium \`v0.127.0\` includes #3474; #3490/#3501 merged **after** that release and must not be assumed present in its published binaries. The former BoxWeave evidence is pinned to its exact test revision and environment. No original Provider upgrade or new integration acceptance follows from a GitHub merge.

## Owner-selected follow-ups

| Decision | Priority | Direction | Scope and acceptance gate |
| --- | --- | --- | --- |
| **UP-MCP-01: documented stable MCP subset** | **P1** | Ask Containarium to document the limited external-controller contract: supported tool names, required/optional arguments and null behavior, \`CONTAINER_STATE_*\` enums, capability/version discovery and expected error/unknown-result semantics. | Prefer an explicit, versioned **subset** over freezing every MCP tool. Test discovered schemas and negative cases against a specified release; keep BoxWeave's defensive discovery and fail-closed target checks even if upstream documents a contract. This is a **request**, not an upstream promise. |
| **UP-MCP-02: whole-box backup create parity** | **P2** | Complete existing \`create_backup\` MCP functionality for the existing \`containarium backup create <box> --box\` CLI capability, without reimplementing backup machinery in BoxWeave. | Authenticate and authorize exact source Box; verify created artifact type/ID and \`verify_backup\` interoperability. Encrypted backups, restore, cleanup and denial tests need separate isolated acceptance. Do not conflate database/hook backups with whole-Box backups. |
| **UP-MCP-03: host-owned restore identity** | **P2** | Keep age private-key material under **trusted Containarium/backup-host control**. An external BoxWeave controller should request a scoped restore using a nonsecret backup reference and an administrator-authorized host-side key identity/reference, not transmit private key bytes or assume a remote controller path points into the host. | First agree with upstream on authority, key lookup and audit semantics. Resolve the current \`restore_box_backup(age_identity_file)\` path locality explicitly. Deny absent/unapproved keys; require target identity, confirmation, permissions and uncertain-result reconciliation. No automatic key export or broad file-path read. |
| **UP-SSH-01: remove vendored host-hop wrapper from *future* BoxWeave releases** | **Owner-selected packaging direction** | **Do not package, ship, automatically install or independently maintain \`containarium-shell\` as a BoxWeave release asset going forward.** Treat this as a Containarium-owned host integration component; BoxWeave retains read-only diagnosis and documentation. | Implementation is a later, separately tested delivery slice. Historical immutable releases remain unchanged, and existing installed shells must **not** be automatically removed or overwritten. |

## Why the host-hop packaging decision matters

BoxWeave source \`v0.42.9\` included an upstream-source-copied, SHA-256-pinned \`bwmini/containarium-shell\`. Its installer can place it at \`/usr/local/bin/containarium-shell\` on a BoxWeave-managed local Provider and optionally repair an external Provider with \`--repair-containarium-shell\`. That **already published historical behavior remains factual**; the owner-selected future design supersedes it, not the immutable release itself.

This script is an **official Containarium host-hop SSH wrapper**, not a BoxWeave-created MCP protocol. It uses SSH login identity to map to an Incus container and calls Incus tools, including for wake-on-SSH. Vendoring it causes BoxWeave to own an additional privileged host lifecycle path, duplicate upstream fixes, and maintain pinned compatibility as Containarium evolves. An exact old-script digest proves provenance, **not future compatibility or secure operation**.

Code-level audit consideration: the reviewed upstream wrapper uses \`sudo incus start\` and \`sudo incus config set/unset\` during SSH wake, whereas the corresponding upstream setup script example grants passwordless \`incus exec/info\`. Actual per-account privileges and SSH wake behavior need explicit positive and negative tests; **never remediate by granting unrestricted Incus privileges**. Also, the wrapper's direct Incus actions require scrutiny for consistent authorization, lifecycle receipts and troubleshooting relative to Containarium API calls.

BoxWeave's authenticated, target-pinned Provider MCP \`connect(exec)\` route has separate live evidence and does **not** require this host-hop wrapper. It remains the preferred Agent operation route when available. Host-hop SSH stays an **optional upstream-managed transport**, not a prerequisite for working direct MCP execution. No claim is made that HTTP wake automatically wakes \`connect(exec)\` or resumes an interrupted Agent task.

## Future packaging migration (NOT implemented)

1. Inventory every downstream reference: \`bwmini/containarium-shell\`, release/installer manifests and digest pins, \`bwmini/online.py\`, \`install.sh\`, \`--repair-containarium-shell\`, source tests, README, operator guidance and managed-file state. Confirm what the **actually published** release contains; do not infer from current source.
2. Remove the wrapper from **new release assets and default installation**. Keep a diagnostic-only check for configured login shells; direct the operator to Containarium's official, version-matched host setup/maintenance path. Do not silently invoke the full upstream setup script or change \`sshd\`, sudoers, \`/etc/shells\` or accounts.
3. Provide an explicit migration policy for installations where BoxWeave previously marked the script as managed. Preserve the binary and existing login paths by default; never delete a shell referenced by a host account. Reconcile old managed-file checks and the legacy repair flag without silently ignoring the operator's request. Any removal or reassignment needs host-specific authorization and a rollback plan.
4. Validate packaging exclusions and source integrity; test clean installs and upgrades from a wrapper-containing release, absence/presence/foreign-wrapper branches, direct MCP \`connect(exec)\`, optional upstream-installed host-hop positive and cross-target negative cases. Verify no broken user login shell or collateral host change. Record exact version, exits and residual unverified cases.
5. Only after separate source, CI and **installed** validation may a new BoxWeave release be claimed to implement this decision. Do not modify the stable pin or existing host during documentation work.

## Follow-up and status discipline

- **Upstream coordination:** follow [issue #3500](https://github.com/FootprintAI/Containarium/issues/3500) for decisions on documented contract, backup parity and host-held age key semantics. Authoritative behavior is the exact installed upstream revision, not the issue checklist.
- **BoxWeave:** treat the three upstream requests as P1/P2/P2 respectively; keep E2/E3/E4 and whole-Box restore explicitly blocked/unverified until authenticated interfaces, storage/crypto constraints, safe target workflows and real acceptance are established.
- **Scope:** documentation-only selection today. No upstream PR/comment was posted, no current release artifact was changed, and no production/owner-host action was performed. Existing [shell runbook](../process/CONTAINARIUM_SHELL_RECOVERY.md) remains historical/operator recovery evidence, not a future recommendation to vendor the wrapper.

[Back to feature roadmap master](FEATURE_ROADMAP_MASTER.md)
