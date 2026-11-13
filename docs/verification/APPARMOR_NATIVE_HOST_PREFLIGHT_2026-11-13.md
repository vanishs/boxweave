# AppArmor native-host preflight: isolated acceptance (2026-11-13 PT)

Candidate branch: fix/v1-apparmor-nested-host-preflight-20261113, based on protected v1 source a5e7b0e9d0dd8c3243d3edeb8c335594b29f1e07.
Candidate executable: SHA256 a12aa78ceeab24f148c7a31937ab8cb271dadff5376c99c8bbd826a9ac2358e3, built in isolated bw-aa-stack-test-1113 using verified Go 1.26.9 with CGO_ENABLED=1. This developer binary has no Release build-info stamp and is NOT an installable release.

## Code and source checks

- Focused tests: TestNativeUserNamespaceMap (7) and TestNativeAppArmorNestedHostPreflightClasses (7) PASS.
- Full internal/install package tests PASS.
- Full make verify PASS: tracked format, YAML parsing, go mod tidy -diff, go vet ./..., go mod verify, CGO Go race tests all packages.
- First make verify attempt in extracted test tree stopped because git was not installed; after installing git in that disposable test Box, the complete gate passed. A transport timeout resulted in duplicate test processes; they were terminated by exact process ID and one standalone verification job completed successfully, exit 0. This was a test-harness/environment issue, not a Go defect.

## Same binary real positive and negative readbacks

- Positive L1 bw-aa-stack-test-1113: root-local native-prereq-plan -project user-1001 reports apparmor_nested_host OK=true. Overall config_ready=false because the disposable host has not been prepared with the v1-exclusive ACL/image configuration. This is NOT a Host-Go or installer acceptance.
- Negative ordinary Project L2 bw-aa-inner-stack-1113: native-plan with the exact pinned package version refuses before APT or native state writes, exit 1, NATIVE_APPARMOR_CONFINED. Process was in the inherited level-1 AppArmor namespace with an enforced Incus profile and no raw.idmap.
- Negative restricted Project L2 bw-aa-restricted-1113: same native-plan refuses, exit 1, NATIVE_APPARMOR_CONFINED under the incus-user-1001 enforced profile.
- Negative forced-unconfined L1 bw-aa-test: native-plan refuses, exit 1, NATIVE_APPARMOR_NO_STACKING (unprivileged user namespace, AppArmor namespace level 0).
- Previous actual Incus 7.6 creation/init attempts failed respectively with apparmor_parser policy-admin denied and aa-exec missing profiles. The preflight catches exactly those known-failure classes.

## Limits and disposition

- All native negative tests were read-only; no new native initialization was replayed. Earlier failed partial Incus state and all original UNKNOWN tasks remain untouched.
- No runtime MCP tool, Box Create behavior, raw LXC config, Incus Project mapping or global/physical host policy changed.
- Positive preflight is diagnostic only; further L2 multi-level AppArmor delegation remains UNSUPPORTED/UNVERIFIED for this topology.
- A real same-head GitHub quality check and a separate independent verifier still must run before merge/release. HOST_GO, physical deployment and RC install remain independent, unapproved claims.
