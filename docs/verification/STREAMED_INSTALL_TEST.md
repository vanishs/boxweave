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
