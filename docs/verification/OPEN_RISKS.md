# Current risk register — 0.14.0 (historical evidence preserved)

Owner is the project maintainer/operator unless specified. P0/P1 = blocks the selected claim until resolved or explicitly cannot accept; P2 = bounded operational/coverage limitation. Source: [comparison](COMPARISON_REPORT.md), SRC-084..088. The historical phase6 P0/P1 items are closed within their original scope. Current remediation still has open P1 work, including RISK-610; use the current rows below. Residual P2 entries do not silently certify the unverified property.

| ID | Priority / status | Impact / scope | Evidence and action / owner |
| --- | --- | --- | --- |
| RISK-601 | P1 closed, fixed0.9.1 | Missing project binding edit/reselect/unbind violated REQ010; phase5 completion prose overstated coverage | New service/UI/revision7 tests, browser and actual installed-copy unbind preserve metadata/receipts. Maintainer: retain regressions and require backup on migration |
| RISK-602 | P1 closed, fixed0.9.1 | Empty target list could claim diagnostic success without contacting MCP and mutate local state | Actual0.9.0 dead-launcher exit0 versus patched exit2, positive real109-tool read with unchanged DB6; no Store in diagnose. Maintainer: preserve empty-list/unavailable-launcher cases |
| RISK-603 | P1 closed, user decision SRC-086 implemented | Missing promised Web client setup entry | Four profile generator, downloaded hash-gated scripts/replacement intent, strict validation/auth/CSRF and actual Windows registration/config-preservation. Operator still independently reviews source/pin and runs local setup; Web never enrolls or reads credential contents |
| RISK-604 | P2 open, legacy test maintenance | Readiness test at d8a09dd fails collection because its installer text extractor is stale |15 other isolated legacy tests pass; readiness assertions unexecuted. Legacy owner may repair its harness in its own authorized task. Do not change old expectations here or infer old runtime readiness |
| RISK-605 | P2 open, deployment handoff | Primary private test app remains0.9.0; editing source does not upgrade it, and new DB7 cannot be read by old binaries |0.9.1 migration/install tested on independent consistent copy; original bytes/PIDs unchanged. Operator: use reviewed0.9.1 package and consistent backup before using patched behavior; retain older artifacts as historical evidence |
| RISK-606 | P2 open, evidence boundary | Full old/new live fleet/network equivalence and other server platforms are unverified | Local old/new receipt execution, old mocks/static inspection and new prior real Provider0.120.1 packets are different lanes. No fabricated parity. Scope any future live legacy deployment separately |
| RISK-607 | P2 accepted boundary | Reboot/LXC boot.autostart, effective confinement, new public TLS rollout and model inference/SLA not demonstrated | Selected user-manager/private-LXC/SSH/native-model-handoff evidence only. Owner: obtain specific evidence before making any additional claim; no phase6 host changes or model spending |
| RISK-608 | P2 accepted trust assumption | Operator-delivered source/pin/digests and broad upstream credential authority remain trusted; generated descriptor does not grant per-tool least privilege | Single trusted operator DEC005. Public setup never contains private key/JWT; helper/payload hashes, exact versions, local ACL/pin and registration gates remain. No publisher signature or general secret-detector completeness claimed |
| RISK-609 | P2 accepted recovery/convergence boundary | Unknown commands cannot be replayed/adopted; dispatched work cannot be cancelled by unbinding; route acknowledgment is not instant packet convergence | UI warning and retained receipts/revision gates; previous real TCP/UDP delay evidence. Reconcile original identities and inspect saved outcomes before a fresh explicit action; no whole-host rollback promise |

Phase7 maintenance is authorized by SRC-089; its completion evidence belongs in [PHASE_07_REVIEW](PHASE_07_REVIEW.md). It does not resolve these operational bounds or authorize publication/deployment. No pending product decision remains after SRC-086. A future required claim becoming unsupported must be reopened as cannot accept, not hidden under a P2 label.

## New 0.10+ remediation risks (current; not closed by the old phase6 report)

| ID | Priority / status | Impact / scope | Evidence and action / owner |
| --- | --- | --- | --- |
| RISK-610 | P1 open / wider AC-019 acceptance | Source0.15.1 published streamed bootstrap/reinstall and separate peer MCP create/identity/Python passed in a dedicated nested Ubuntu LXC. This does not certify every fresh host, confinement or platform. New0.16.0 helper privilege acceptance is tracked separately in RISK-613 | [Streamed evidence](STREAMED_INSTALL_TEST.md), [nesting limits](NESTING_VERIFICATION.md); preserve existing upstream and require exact-version tests |
| RISK-611 | P1 closed within selected CLI acceptance, 0.14.0 | Actual installed wrapper/status/doctor, exact local Incus reads and missing/invalid targets, independent scoped MCP58-tool read, credential rotation and failed-install marker/repair passed in the dedicated LXC.192 unit tests/build/seven browser workflows and current four-profile Windows reads passed; final package identity remains separately recorded | [CI and next-Agent queue](REMEDIATION_TEST_QUEUE.md). Owner: retain these regressions; broader Provider/network/reboot acceptance remains in RISK-610/606/607, not implied by CLI checks |
| RISK-612 | P2 source CI gate verified at 0.13.4 / continuing | GitHub Actions run 40985891994 on 08efb069 succeeded for source 0.13.10; 0.14.0 working-tree changes require their own pipeline after publication. Fresh CI must still be checked for every later commit | [Exact source CI and handoff](REMEDIATION_TEST_QUEUE.md); no host, Provider, Windows, browser or security acceptance follows from a green source pipeline |

Current remediation evidence: [0.14.0 review](MAIN_014_REVIEW.md). This separate test application does not upgrade the retained primary deployment in RISK-605.

Streamed source0.15.0 is now being implemented/tested under SRC-100. RISK-610 remains open until the exact streamed install and separate client creation/connect/Python path pass; [progress](STREAMED_INSTALL_TEST.md). Standalone Provider optional/native-policy/boot boundaries remain explicit.

Current SRC-100 experiment: source0.15.1 streamed reinstall and separate peer create/identity/Python passed. This resolves the specific experiment, not wider RISK-610 platform acceptance. [Measured follow-up](NESTING_VERIFICATION.md).

| RISK-613 | P1 open, new0.16.0 privilege path | New fixed helper/sudo rule and MCP extension need real installed-copy acceptance; old0.15.1 evidence does not certify this surface | [Validation and next checks](NESTING_VERIFICATION.md); no implicit restart or parent access |


## 0.17.0 existing Provider adoption risk

**RISK-614 (P1 open, existing-host acceptance):** Streamed installer source recognizes reviewed existing Containarium/Incus and retains Provider ownership across BoxWeave repair/update. The existing Provider's API must be reachable at the configured loopback endpoint and its signer readable only by the external Provider/token issuer. No real preexisting-host adoption, storage-source preservation, reboot or independent update/repair test is claimed. The current published artifact remains Ubuntu 24.04 / CPython 3.12; Ubuntu 26.04 compatibility is a separate package and acceptance task. [Test matrix](EXISTING_PROVIDER_INSTALL.md).


## 0.17.2 cross-version installation acceptance

**RISK-615 (P1 open, host verification):** Minimum host version is Ubuntu >= 24.04 / CPython >= 3.12, but the release bundles wheels only for CPython 3.12–3.15 and exact CI builds do not certify all future Python versions. The owner-provided Ubuntu 26.04 + bcachefs + existing Incus/Containarium/PostgreSQL must still pass a full real-host source-bound install, update/repair, container creation, and post-reboot mount/provider persistence check. **S**: source review; **U**: real-host acceptance. Do not bypass storage mount guards or silently replace external Provider state.


## 0.17.3 client setup live acceptance

**RISK-616 (P1 open, live client verification):** New Windows Codex + WSL SSH enrollment script generated by `boxweave client` must be exercised on an actual Windows Codex client and Ubuntu administrator SSH connection. The CI static PowerShell parse, fixtures and non-mutating source inspection do not certify interactive WSL password/sudo enrollment, forced-command SSH, MCP read or Codex app UI recognition. Required refusal tests: changed host pin, missing bootstrap sudo rights, mismatched WSL distro, keypair conflict, MCP unavailable, and existing Codex MCP entry without `-Replace`. Never bypass SSH host-key verification or auto-edit Incus/Containarium during client setup.


## 0.17.4 desktop MCP configuration regression

**RISK-617 (P1 live acceptance):** Confirm generated script works with the Windows ChatGPT/Codex desktop app **without** a standalone `codex` command on PATH. Verify app reads `%USERPROFILE%\\.codex\\config.toml`, password/keyboard-interactive administrator SSH auth, explicit-key admin fallback, backup and refusal of complex preexisting MCP TOML, and restart recognition. CI grammar parsing does not establish Windows desktop runtime correctness. Never modify WindowsApps ACLs or require installing a second Codex solely for MCP registration.


**RISK-617 (P1 open, workload managed SSH identity):** Historical `archive/0.5.1` clients exposed a separate full server-to-box public key in AGENTS, but the new client generator omitted it. A real `hello-world-test-container` creation returned Running, followed by an SSH public-key-authentication failure; actual supplied `ssh_keys` and guest `authorized_keys` are not yet independently attested. Version 0.17.8 adds a fail-closed create gate and restored client guidance, but **does not** modify existing resources or establish a successful `connect`/Python receipt. Operator must compare trusted full public key fingerprints, reconcile exactly the existing box with explicit approval if needed, and prove `whoami && hostname` and actual Python stdout/exit through MCP. This remains unverified until live evidence; no duplicate create or trust bypass.


**0.18.0 RISK-617 update:** The new Linux MCP gateway adds its own validated public key to each new create request, so Codex no longer needs BoxWeave-managed AGENTS guidance. The original `hello-world-test-container` still needs authorized investigation and live SSH/Python verification; neither the old block nor guest authorized keys are changed by installing this release.


**RISK-615/610 update (0.18.2):** The Ubuntu 24.04 GA run 41141400914 failed during APT resolution because a fixed Incus 7.5.1 subset conflicted with the Incus-base 7.6 candidate. The new managed fresh-host path removes partial pins and simulates the APT transaction before installation, but the exact revision still needs a complete isolated live install. The long-term owner-requested provider-only dependency boundary remains unresolved; the upstream Containarium installer itself changes Incus and host settings and is not silently executed.


**RISK-618 (P1, upstream installer integration, 2026-10-30):** Directly
executing the mutable upstream Containarium manual installer during a fresh
BoxWeave installation would open the REST listener on `0.0.0.0:8080` and
potentially UFW 8080, uninstall older Incus packages (including `apt-get
autoremove`), and install only the daemon, not the separate `mcp-server`.
Do not silently substitute it for the current reviewed managed Provider
bootstrap. Confirm upstream loopback/no-firewall support, immutable installer
identity, complete same-release MCP assets, exact provider compatibility,
owned-state separation, and real clean-host/update acceptance first. See
[the source-backed review](CONTAINARIUM_INSTALLER_BOUNDARY.md).


**RISK-619 (MCP SSH keepalive, 0.19.1):** Legacy `archive/0.5.1`
client SSH transports used `ServerAliveInterval=15` and
`ServerAliveCountMax=4` to mitigate idle session loss. Source-level
restoration of those flags in Windows Codex/WSL and legacy native/WSL
registration paths does not prove real long-idle reliability on any
network. Existing client config needs operator-approved re-registration;
the installer does not alter running SSH sessions or server sshd config.
Keepalive detects dead transports; it cannot automatically reconnect MCP.


**0.20.0 unified client generator note:** All new installed
`boxweave client` outputs are now Python 3.11+ including Codex/WSL.
This removes the old PowerShell-only Windows path from the installed
generator but does not install Python on clients or migrate old
`.ps1` registrations. Manual Windows PowerShell helpers remain
separate, and real cross-platform Python+SSH App acceptance is open.


**RISK-620 (gate-host nesting helper dependency, 0.20.1):** The
`enable_nesting` create extension and the `boxweave_nesting` tool run
`sudo .../nesting.py` on the host that hosts the MCP gate and require
that host to carry a complete BoxWeave server state
(`/var/lib/boxweave-default/state.json`, public helpers, sudoers). On
split deployments where the gate runs away from the Provider host —
observed on the 2026-10-30 dev backend — every nesting call fails with
`NESTING_HELPER_UNAVAILABLE` after the gate fix; before the fix the
failure was masked as a trust-gate error and the create response
suggested the box had not been created. Platform-created boxes were
empirically nested-capable, so workload creation was unaffected, but
the nesting tool contract remains non-functional until the gate host is
repaired or nesting is delegated to the daemon. Operators must not
delete and recreate boxes in response to `BOXWEAVE_NESTING_NOT_APPLIED`.
