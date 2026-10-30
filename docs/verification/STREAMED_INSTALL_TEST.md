# Streamed installation and peer MCP test

Authorization: SRC-100; old implementation knowledge: SRC-101. This file records planned versus observed results, never an assumed pass.

## Owned targets and first observation

Outer old MCP server: dev, Containarium0.120.1, Incus7.5.1. Both new Ubuntu24.04 boxes have native Network Tenant `bw-e2e-1029`: server `bw-e2e-server-1029-container`,10.206.30.13,2CPU/4GB/20GB; client `bw-e2e-client-1029-container`,10.206.30.14,1CPU/1GB/8GB. Creation supplied the full approved outer managed Ubuntu public key. A pending client create was inspected by its original logical name and found running; it was not duplicated. Each old-MCP connect identity check returned its exact username and hostname, exit0.

Original main9f03b9f command:
`curl -fsSL https://raw.githubusercontent.com/vanishs/boxweave/main/install.sh | sudo bash`
actually returned curl0/installer2/runner2 with DEFAULT_INSTALL_UNAVAILABLE. Server evidence: `/home/bw-e2e-server-1029/boxweave-e2e-evidence/install-attempt-01.log` and matching exit file.

## Implemented iteration, acceptance pending

Source0.15.0 implements source-tagged GitHub release publication after CI; verified streamed download; empty-host signed/pinned Incus/Provider preparation; independent application bootstrap/credential timer; new-account forced-command SSH enrollment and public client configuration; trusted local Incus public-key reconciliation; and a relay rejecting implicit-key creation and stale/untrusted/mismatched connections. The old installer is read-only knowledge, not deployed source. Root/user boundaries and all prior managed app paths remain distinct.

Source validation and actual install/client/workload outcomes will be appended after execution. The Provider is standalone; native policy persistence, optional hosting, public TLS, effective parent confinement and reboot/LXC autostart are not claims of this basic smoke test. Public networking and paid inference are not part of the test. Retain the two outer boxes, any nested workload and evidence unless cleanup is explicitly requested.

## Pre-publication source checks

Corrected assigned non-root validation exited0 with Ran 199 tests in 61.019s, frontend typecheck/build and all seven Chromium fixture workflows. Report: `/home/boxweave-tenant-dev/e2e-default-1029/source/.runtime/validation/99070125dd2f4aad9256f17cc3b0e7d3/result.json`. Shell syntax, new public-key/trust/release-manifest gates and YAML parsing passed. The earlier source-only run failed an import due to an authoring transfer EOF escape; it was retained and corrected before publication. These are source/fixture checks, not yet the requested live hello-world acceptance.

## 0.15.0 live feedback

Commit dcad890 and GitHub run40995597271 verification/publication succeeded. The exact original streamed command was rerun in the same server box: curl0/installer0, source-bound release0.15.0 ready, with API/worker and credential timer verified by the app installer. The server generated the public MCP config; only the peer's newly generated public key was enrolled, and the private key stayed on the peer.

The peer actually completed initialize/initialized, tools/list58 and list_containers against the new SSH/MCP endpoint. Workload creation returned a definite FailedPrecondition: Tenant Guard refused because network-policy storage was the in-memory fallback (Postgres unavailable). Original-name get/debug/list plus local names/status proved no workload existed. A same-name diagnostic retry reproduced the exact refusal; no alternate or force-recreated workload was used. This was not a successful hello-world test.

Patch0.15.1 adds a dedicated Ubuntu PostgreSQL16 dependency in this owned service LXC. It refuses foreign PostgreSQL state, creates a minimal peer-authenticated database owner with no database password, verifies an actual SQL read, and supplies its Unix-socket DSN to the managed Provider. The stopped core container is not modified. Reinstall/update must preserve server/client keys, enrollment, signing authority and app config; actual rerun results remain pending.

## 0.15.1 actual acceptance

Release0.15.1 verify/publication and same-server exact streamed reinstall passed, curl0/installer0. PostgreSQL active; app config/host public key/managed public key/enrollment digests unchanged. After parent nesting correction and outer restart, peer new MCP created bw-e2e-hello-nesting-1029-container, verified identity and ran Python hello world, exit0/script0. Interrupted original retained. [Later comparisons and0.16.0 scope](NESTING_VERIFICATION.md); no broader platform/security claim.
